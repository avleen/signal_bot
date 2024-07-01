package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
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

var Config = map[string]string{
	"IMAGE_PROVIDER":   os.Getenv("IMAGE_PROVIDER"),
	"IMAGEDIR":         os.Getenv("IMAGEDIR"),
	"MAX_AGE":          os.Getenv("MAX_AGE"),
	"PHONE":            os.Getenv("PHONE"),
	"REST_URL":         os.Getenv("REST_URL"),
	"STATEDB":          os.Getenv("STATEDB"),
	"SUMMARY_PROVIDER": os.Getenv("SUMMARY_PROVIDER"),
	"URL":              os.Getenv("URL"),
}

// These parameters are situational and depends on the provider requested.
var optionalConfig = map[string]string{
	"GOOGLE_PROJECT_ID": os.Getenv("GOOGLE_PROJECT_ID"),
	"GOOGLE_LOCATION":   os.Getenv("GOOGLE_LOCATION"),
	"GOOGLE_TEXT_MODEL": os.Getenv("GOOGLE_TEXT_MODEL"),
	"OPENAI_API_KEY":    os.Getenv("OPENAI_API_KEY"),
	"OPENAI_MODEL":      os.Getenv("OPENAI_MODEL"),
}

func (ctx *AppContext) helpCommand() {
	// Send a help message to the send channel
	message := "Available commands:\n" +
		"!help - Display this help message\n" +
		"!imagine <text> - Generate an image\n" +
		"!summary <num_msgs|12h> - Generate a summary of last N messages, or last H hours\n" +
		"!ask <question> - Ask a question\n"
	ctx.MessagePoster(message, "")
}

func (ctx *AppContext) processMessage(message string) {
	// Process incoming messages from the WebSocket server
	log.Println("Received message:", message)
	// Get the message root
	container, msgStruct := getMessageRoot(message)

	// If there is no message (for example, this is an emoji reaction), return
	if msgStruct["message"] == nil {
		return
	}

	// If the msgStruct does not contains the field groupInfo isn't a real message, return
	if _, ok := msgStruct["groupInfo"]; !ok {
		return
	}

	// Ensure groupInfo contains a groupId. if it does, call encodeGroupIdToBase64()
	// Empty out the existing recipients and set it to the new value.
	// Otherwise, return
	ctx.Recipients = []string{}
	if groupId, ok := msgStruct["groupInfo"].(map[string]interface{})["groupId"]; ok {
		ctx.Recipients = append(ctx.Recipients, encodeGroupIdToBase64(groupId.(string)))
	} else {
		return
	}

	// Persist the message to the database
	ctx.saveMessage(container, msgStruct)

	// This is handy to pull out now, we use it later
	sourceName := container["envelope"].(map[string]interface{})["sourceName"].(string)
	msgBody := msgStruct["message"].(string)

	// If the first word in the message starts with a !, it's a command.
	// Take the first word and call the appropriate function with the rest of the message
	// Otherwise, return
	if strings.HasPrefix(msgBody, "!") {
		words := strings.Fields(msgBody)
		switch words[0] {
		case "!help":
			ctx.helpCommand()
		case "!imagine":
			// If words[1:] is empty, call help
			if len(words) < 2 {
				ctx.helpCommand()
				return
			} else {
				ctx.imagineCommand(sourceName, strings.Join(words[1:], " "))
			}
		case "!summary":
			// If no additional arguments were given, just call for the summary.
			c := TimeCountCalculator{-1, -1}
			starttime, count, err := c.calculateStarttimeAndCount(words)
			if err != nil {
				log.Println("Error parsing hours and count:", err)
				return
			}
			ctx.summaryCommand(starttime, count, sourceName, "")
		case "!ask":
			// If words[1:] is empty, call help
			if len(words) < 2 {
				ctx.helpCommand()
				return
			} else {
				ctx.summaryCommand(-1, -1, sourceName, strings.Join(words[1:], " "))
			}
		}
	} else {
		return
	}
}

func (ctx *AppContext) removeOldMessages() {
	// Convert the value of config.MAX_AGE into an int
	// We don't check for the err because we validated this in main()
	maxAge, _ := strconv.Atoi(Config["MAX_AGE"])

	// Delete messages older than config.max_age from the database
	query := "DELETE FROM messages WHERE timestamp < ?"
	maxAgeInNs := time.Hour * time.Duration(maxAge)
	args := []interface{}{time.Now().Add(-maxAgeInNs).Unix() * 1000}
	log.Println("Removing messages older than", maxAge, "hours. Timestamp:", args[0])
	ctx.DbQueryChan <- dbQuery{query, args, nil}
}

func (ctx *AppContext) debugger() {
	// Start a debugger session
	log.Println("Starting debugger session")
	// A message template to help us test with
	tpl := `{
		"envelope":{
			"source":"+123456789",
			"sourceNumber":"+123456789",
			"sourceUuid":"019063e8-9042-72ca-9b66-30a3c83d4489",
			"sourceName":"Test User",
			"sourceDevice":1,
			"timestamp":%v,
			"syncMessage":{
				"sentMessage":{
					"destination":null,
					"destinationNumber":null,
					"destinationUuid":null,
					"timestamp":%v,
					"message":"%s",
					"expiresInSeconds":604800,
					"viewOnce":false,
					"groupInfo":{
						"groupId":"VGVzdA==",
						"type":"DELIVER"
					}
				}
			}
		},
		"account":"+123456789"
	}`
	for {
		// Prompt the user for a message
		request := StringPrompt("Enter a message:")
		if request == "" {
			break
		}
		// Put the message in the template
		timeNow := time.Now().Unix() * 1000
		message := fmt.Sprintf(tpl, timeNow, timeNow, request)
		// Process the message
		ctx.processMessage(message)
	}
}

func (ctx *AppContext) websocketClient() (bool, error) {
	// Establish a WebSocket connection.
	// Return nothing on success, or an error if the connection fails.
	conn, _, err := websocket.DefaultDialer.Dial(fmt.Sprintf("ws://%s/v1/receive/%s", Config["URL"], Config["PHONE"]), nil)
	if err != nil {
		log.Println("Failed to connect to WebSocket:", err)
		// Return an error
		return false, err
	}
	log.Println("Connected to WebSocket")
	defer conn.Close()

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

	url := fmt.Sprintf("%s/v1/receive/%s", Config["REST_URL"], Config["PHONE"])
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

func (ctx *AppContext) saveMessage(container map[string]interface{}, msgStruct map[string]interface{}) {
	// Persist the message to the database at config.statedb
	ts := container["envelope"].(map[string]interface{})["timestamp"]
	sourceNumber := container["envelope"].(map[string]interface{})["sourceNumber"]
	sourceName := container["envelope"].(map[string]interface{})["sourceName"]
	message := msgStruct["message"]
	groupId := msgStruct["groupInfo"].(map[string]interface{})["groupId"]

	query := "INSERT INTO messages (timestamp, sourceNumber, sourceName, message, groupId) VALUES (?, ?, ?, ?, ?)"
	args := []interface{}{ts, sourceNumber, sourceName, message, groupId}
	ctx.DbQueryChan <- dbQuery{query, args, nil}
}

func startupValidator() {
	// In MAX_AGE is not an int, panic
	if _, err := strconv.Atoi(Config["MAX_AGE"]); err != nil {
		log.Println("Invalid MAX_AGE:", Config["MAX_AGE"], ", defaulting to 168")
		Config["MAX_AGE"] = "168"
	}
	// For each of the required environment variables, if it is empty, panic
	for key, value := range Config {
		if value == "" {
			log.Fatalf("Missing environment variable: %s", key)
		}
	}
	// Merge Config and optionalConfig
	for key, value := range optionalConfig {
		Config[key] = value
	}
	// Ensure that the IMAGEDIR is set to a full path. Using relative paths is not secure.
	if !filepath.IsAbs(Config["IMAGEDIR"]) {
		log.Fatalf("IMAGEDIR must be an absolute path: %s", Config["IMAGEDIR"])
	}
	// If the database file doesn't exist, panic
	if _, err := os.Stat(Config["STATEDB"]); os.IsNotExist(err) {
		log.Fatalf("Database file does not exist: %s", Config["STATEDB"])
	}
}

func main() {
	// Do some start-up validation
	startupValidator()

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
		DbQueryChan:        make(chan dbQuery),
		DbReplySummaryChan: make(chan interface{}),
		DbReplyAskChan:     make(chan interface{}),
		Recipients:         []string{},
	}

	go ctx.dbWorker()

	// Start the appropriate mode
	switch *mode {
	case "websocket":
		// Start a goroutine that runs cleanup_state every hour
		go func() {
			for {
				ctx.removeOldMessages()
				time.Sleep(time.Hour)
			}
		}()

		// Set the message poster to the sendMessage function
		ctx.MessagePoster = ctx.sendMessage
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
	case "debugger":
		ctx.MessagePoster = Printer
		ctx.debugger()
	default:
		log.Fatal("Invalid mode:", *mode)
	}

}
