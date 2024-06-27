package main

import (
	"database/sql"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/websocket"
	_ "github.com/mattn/go-sqlite3"
)

// Get the environment variables from the environment:
//   IMAGEDIR: the directory where images are stored
//   STATEDB: the path to the SQLite database file
//   HOURS: the default number of hours to generate summaries
//   PHONE: the account phone number
//   URL: the URL of the WebSocket server
//   REST_URL: the URL of the REST API server
//   MAX_AGE: the maximum age of messages to keep

var config = map[string]string{
	"IMAGEDIR": os.Getenv("IMAGEDIR"),
	"STATEDB":  os.Getenv("STATEDB"),
	"PHONE":    os.Getenv("PHONE"),
	"URL":      os.Getenv("URL"),
	"REST_URL": os.Getenv("REST_URL"),
	"MAX_AGE":  os.Getenv("MAX_AGE"),
}

type dbMessage struct {
	query  string
	values []interface{}
}

type AppContext struct {
	DbChan   chan dbMessage
	SendChan chan string
}

func (ctx *AppContext) dbWorker() {
	// Open the database connection
	db, err := sql.Open("sqlite3", config["STATEDB"])
	if err != nil {
		log.Println("Failed to open database:", err)
		return
	}
	defer db.Close()

	// Loop forever, processing messages from the channel
	for {
		func() {
			message := <-ctx.DbChan
			// Process the message
			stmt, err := db.Prepare(message.query)
			if err != nil {
				log.Println("Failed to prepare statement:", err)
				return
			}
			defer stmt.Close()
			_, err = stmt.Exec(message.values...)
			if err != nil {
				log.Println("Failed to execute statement:", err)
				return
			}
			log.Println("Processing message:", message)
		}()
	}
}

func (ctx *AppContext) helpCommand() {
	// Send a help message to the send channel
	ctx.SendChan <- "Available commands:\n" +
		"!help - Display this help message\n" +
		"!image <text> - Generate an image\n" +
		"!summary <num_msgs|12h> - Generate a summary of last N messages, or last H hours\n" +
		"!ask <question> - Ask a question\n"
}

func (ctx *AppContext) imagineCommand(args []string) {
	// Generate an image from the text and send it to the send channel
	fmt.Println("Generating image from text:", strings.Join(args, " "))
}

func (ctx *AppContext) summaryCommand(args []string) {
	// Generate a summary of the last N messages or last H hours
	// and send it to the send channel
	fmt.Println("Generating summary:", strings.Join(args, " "))
}

func (ctx *AppContext) askCommand(args []string) {
	// Ask a question and send it to the send channel
	fmt.Println("Asking question:", strings.Join(args, " "))
}

func (ctx *AppContext) processMessage(message string) {
	// Process incoming messages from the WebSocket server
	log.Println("Received message:", message)
	// Get the message root
	container := getMessageRoot(message)
	// Pull out a few fields so they're easier to reference
	msgStruct := container["msgStruct"].(map[string]interface{})

	// If the message does not contains the field groupInfo isn't a real message, return
	if _, ok := msgStruct["groupInfo"]; !ok {
		return
	}
	// Ensure groupInfo contains a groupId. if it does, call encodeGroupIdToBase64()
	// Otherwise, return
	if groupId, ok := msgStruct["groupInfo"].(map[string]interface{})["groupId"]; ok {
		msgStruct["groupInfo"].(map[string]interface{})["groupId"] = encodeGroupIdToBase64(groupId.(string))
	} else {
		return
	}

	// Persist the message to the database
	ctx.saveMessage(container)

	// If the first word in the message starts with a !, it's a command.
	// Take the first word and call the appropriate function with the rest of the message
	// Otherwise, return
	if message[0] == '!' {
		words := strings.Fields(message)
		switch words[0] {
		case "!help":
			ctx.helpCommand()
		case "!imagine":
			// If words[1:] is empty, call help
			if len(words) < 2 {
				ctx.helpCommand()
				return
			} else {
				ctx.imagineCommand(words[1:])
			}
		case "!summary":
			ctx.summaryCommand(words[1:])
		case "!ask":
			// If words[1:] is empty, call help
			if len(words) < 2 {
				ctx.helpCommand()
				return
			} else {
				ctx.askCommand(words[1:])
			}
		}
	} else {
		return
	}
}

func (ctx *AppContext) removeOldMessages() {
	// Convert the value of config.MAX_AGE into an int
	// We don't check for the err because we validated this in main()
	maxAge, _ := strconv.Atoi(config["MAX_AGE"])

	// Delete messages older than config.max_age from the database
	query := "DELETE FROM messages WHERE timestamp < ?"
	args := []interface{}{time.Now().Add(-time.Hour * time.Duration(24*maxAge))}
	ctx.DbChan <- dbMessage{query, args}
}

func (ctx *AppContext) websocketClient() (bool, error) {
	// Establish a WebSocket connection.
	// Return nothing on success, or an error if the connection fails.
	conn, _, err := websocket.DefaultDialer.Dial(fmt.Sprintf("ws://%s/v1/receive/%s", config["URL"], config["PHONE"]), nil)
	if err != nil {
		log.Println("Failed to connect to WebSocket:", err)
		// Return an error
		return false, err
	}
	log.Println("Connected to WebSocket")
	defer conn.Close()

	// Start a goroutine to handle outgoing messages
	go func() {
		for {
			message := <-ctx.SendChan
			err := conn.WriteMessage(websocket.TextMessage, []byte(message))
			if err != nil {
				log.Println("Failed to send message to WebSocket:", err)
				return
			}
		}
	}()

	// Start a goroutine to handle incoming messages
	go func() {
		for {
			_, message, err := conn.ReadMessage()
			if err != nil {
				log.Println("Failed to read message from WebSocket:", err)
				return
			}
			// For each message start a goroutine to process it
			go ctx.processMessage(string(message))
		}
	}()

	// Keep the client goroutine running
	select {}
}

func restClient() {
	// Fetch the latest messages from the REST API at
	// http://{config.rest_url}/v1/receive/{config.phone}
	// Print them to the console and then persist them to the database

	url := fmt.Sprintf("%s/v1/receive/%s", config["REST_URL"], config["PHONE"])
	resp, err := http.Get(url)
	if err != nil {
		log.Println("Failed to make HTTP GET request:", err)
		return
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Println("Failed to read response body:", err)
		return
	}

	log.Println("Response:", string(body))

	// Persist the messages to the database
	// ...
}

func (ctx *AppContext) saveMessage(container map[string]interface{}) {
	// Persist the message to the database at config.statedb
	ts := container["timestamp"]
	sourceNumber := container["sourceNumber"]
	sourceName := container["sourceName"]
	message := container["message"]
	groupId := container["groupInfo"].(map[string]interface{})["groupId"]

	query := "INSERT INTO messages (timestamp, source_number, source_name, message, group_id) VALUES (?, ?, ?, ?, ?)"
	args := []interface{}{ts, sourceNumber, sourceName, message, groupId}
	ctx.DbChan <- dbMessage{query, args}
}

func main() {
	// Do some start-up validation
	// In MAX_AGE is not an int, panic
	if _, err := strconv.Atoi(config["MAX_AGE"]); err != nil {
		log.Println("Invalid MAX_AGE:", config["MAX_AGE"], ", defaulting to 168")
		config["MAX_AGE"] = "168"
	}
	// For each of the required environment variables, if it is empty, panic
	for key, value := range config {
		if value == "" {
			log.Fatalf("Missing environment variable: %s", key)
		}
	}

	// Accept command line arguments with flag:
	//   -mode: websocket or rest
	//   -debug: enable debug logging
	mode := flag.String("mode", "websocket", "start mode: websocket or rest")
	debugflag := flag.Bool("debug", false, "enable debug logging")
	flag.Parse()

	// Enable debug logging if requested
	if *debugflag {
		log.SetFlags(log.LstdFlags | log.Lshortfile)
	}

	// Create the application context
	ctx := AppContext{
		DbChan:   make(chan dbMessage),
		SendChan: make(chan string),
	}

	go ctx.dbWorker()

	// Start the appropriate mode
	switch *mode {
	case "websocket":
		// Start the WebSocket client. Retry every 3 seconds on failure.
		for {
			_, err := ctx.websocketClient()
			if err != nil {
				log.Println("Failed to connect to WebSocket:", err)
			}
			time.Sleep(3 * time.Second)
		}
	case "rest":
		restClient()
	default:
		log.Fatal("Invalid mode:", *mode)
	}

	// Start a goroutine that runs cleanup_state every hour
	go func() {
		for {
			ctx.removeOldMessages()
			time.Sleep(time.Hour)
		}
	}()

}
