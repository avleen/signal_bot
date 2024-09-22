// Helper functions

package main

import (
	"bufio"
	"bytes"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"

	"go.opentelemetry.io/otel"
)

func makeOutputDir(dir string) error {
	// Create the output directory if it doesn't exist
	err := os.MkdirAll(dir, 0755)
	if err != nil {
		return err
	}
	return nil
}

func getMessageRoot(message string) (map[string]interface{}, map[string]interface{}) {
	// Parse the message and return the root message object

	var msgStruct map[string]interface{}
	container := make(map[string]interface{})
	json.Unmarshal([]byte(message), &container)

	if dataMessage, ok := container["envelope"].(map[string]interface{})["dataMessage"]; ok {
		msgStruct = dataMessage.(map[string]interface{})
	} else if syncMessage, ok := container["envelope"].(map[string]interface{})["syncMessage"]; ok {
		if sentMessage, ok := syncMessage.(map[string]interface{})["sentMessage"]; ok {
			msgStruct = sentMessage.(map[string]interface{})
		} else {
			return nil, nil
		}
	}
	return container, msgStruct
}

func encodeGroupIdToBase64(groupId string) string {
	// Convert the groupId to base64
	groupIdBase64 := base64.StdEncoding.EncodeToString([]byte(groupId))
	return fmt.Sprintf("group.%s", groupIdBase64)
}

func (ctx *AppContext) dbWorker() {
	// Open the database connection
	db, err := sql.Open("sqlite3", Config["STATEDB"])
	if err != nil {
		log.Println("Failed to open database:", err)
		return
	}
	defer db.Close()

	// Loop forever, processing messages from the channel
	for {
		func() {
			query := <-ctx.DbQueryChan
			// Process the message
			stmt, prep_err := db.Prepare(query.query)
			if prep_err != nil {
				log.Println("Failed to prepare statement:", err)
				return
			}
			defer stmt.Close()

			// If the query is a SELECT, we'll use stmt.Query.
			// If the query is anything else, we'll use stmt.Exec.
			if query.replyChan == nil {
				_, exec_err := stmt.Exec(query.values...)
				if exec_err != nil {
					log.Println("Failed to execute statement:", err)
					return
				}
				return
			}

			rows, query_err := stmt.Query(query.values...)
			if query_err != nil {
				log.Println("Failed to execute statement:", err)
				return
			}
			// If we got rows back we need to put them on the response channel
			if rows != nil && query.replyChan != nil {
				query.replyChan <- dbReply{rows, nil}
			}
		}()
	}
}

func (ctx *AppContext) fetchLogsFromDB(starttime int, count int) (*sql.Rows, error) {
	// Fetch logs from the database.
	// If count is greater than zero, get that many logs.
	// Then if starttime is not zero, get logs starting from that time.
	// Return a map of the logs.
	// If there are no logs, return an empty map.
	var query string
	var args []interface{}
	if count > 0 {
		query = "SELECT sourceName || ': ' || message FROM messages ORDER BY timestamp DESC LIMIT ?"
		args = []interface{}{count}
	} else if starttime != 0 {
		query = "SELECT sourceName || ': ' || message FROM messages WHERE timestamp >= ? ORDER BY timestamp ASC"
		args = []interface{}{starttime}
	} else {
		return nil, errors.New("either hours or count must be provided")
	}

	// Send the query to the database and return the result
	replyChan := make(chan dbReply, 1)
	defer close(replyChan)
	ctx.DbQueryChan <- dbQuery{query, args, replyChan}
	rows := <-replyChan
	if rows.rows != nil {
		return rows.rows, nil
	} else {
		return nil, rows.err
	}
}

func compileLogs(rows *sql.Rows) (string, error) {
	// Compile the logs into a string
	var logs string
	for rows.Next() {
		var log string
		err := rows.Scan(&log)
		if err != nil {
			fmt.Println("Failed to scan log:", err)
			continue
		}
		logs += log + "\n"
	}
	if logs == "" {
		return "", errors.New("no logs found")
	}
	return logs, nil
}

func getNumberFromString(number string) (int, error) {
	// Get the number from the string using regular expressions
	re := regexp.MustCompile(`\d+`)
	match := re.FindString(number)
	if match == "" {
		return 0, errors.New("no number found in the string")
	}
	num, err := strconv.Atoi(match)
	if err != nil {
		return 0, err
	}
	return num, nil
}

func (ctx *AppContext) sendMessage(message string, attachment string) {
	// Start a new span
	tracer := otel.Tracer("signal-bot")
	_, span := tracer.Start(ctx.TraceContext, "sendMessage")
	defer span.End()

	// If attachment is not nil, it's the path to a file.
	// Check that the file exists. If it does, read it and base64 encode it.
	payload := map[string]any{
		"message":    message,
		"number":     Config["PHONE"],
		"recipients": ctx.Recipients,
	}

	if attachment != "" {
		_, err := os.Stat(attachment)
		if err != nil {
			log.Println("Failed to find attachment:", err)
			return
		}
		file, err := os.Open(attachment)
		if err != nil {
			log.Println("Failed to open attachment:", err)
			return
		}
		defer file.Close()
		data, err := io.ReadAll(file)
		if err != nil {
			log.Println("Failed to read attachment:", err)
			return
		}
		encodedFile := base64.StdEncoding.EncodeToString(data)
		payload["base64_attachments"] = []string{encodedFile}
	}
	// Send a HTTP POST to the server at $url with the message in the body
	body, err := json.Marshal(payload)
	if err != nil {
		log.Println("Failed to marshal payload:", err)
		return
	}
	request, err := http.NewRequest("POST",
		"http://"+Config["URL"]+"/v2/send",
		bytes.NewBuffer(body))
	if err != nil {
		log.Println("Failed to send message:", err)
	}
	request.Header.Add("Content-Type", "application/json")
	client := &http.Client{}
	res, err := client.Do(request)
	if err != nil {
		log.Println("Failed to send message:", err)
	}
	defer res.Body.Close()
}

func Printer(message string, attachment string) {
	fmt.Println(message)
}

// StringPrompt asks for a string value using the label
func StringPrompt(label string) string {
	var s string
	r := bufio.NewReader(os.Stdin)
	for {
		fmt.Fprint(os.Stderr, label+" ")
		s, _ = r.ReadString('\n')
		if s != "" {
			break
		}
	}
	return strings.TrimSpace(s)
}
