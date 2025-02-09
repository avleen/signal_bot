package main

import (
	"context"
	"encoding/json"
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

	_ "net/http/pprof"

	"github.com/gorilla/websocket"
	_ "github.com/mattn/go-sqlite3"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace" // Use the alias here
	semconv "go.opentelemetry.io/otel/semconv/v1.4.0"
	"go.opentelemetry.io/otel/trace"
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
	"BOTNAME":          os.Getenv("BOTNAME"),
	"CHAT_PROVIDER":    os.Getenv("CHAT_PROVIDER"),
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
	"OPENAI_CHAT_MODEL": os.Getenv("OPENAI_CHAT_MODEL"),
	"OPENAI_MODEL":      os.Getenv("OPENAI_MODEL"),
	"PPROF_PORT":        os.Getenv("PPROF_PORT"),
}

func initTracer() func() {
	// Create a new exporter to send traces over http
	ctx := context.Background()
	exporter, err := otlptracehttp.New(ctx)
	if err != nil {
		log.Fatal(err)
	}

	// Create a new tracer provider with the exporter
	tp := sdktrace.NewTracerProvider( // Use the alias here
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceNameKey.String("signal-bot"),
		)),
	)

	// Set the global tracer provider
	otel.SetTracerProvider(tp)

	// Return a function to shut down the tracer provider
	return func() {
		if err := tp.Shutdown(context.Background()); err != nil {
			log.Fatal(err)
		}
	}
}

func (ctx *AppContext) helpCommand() {
	// Send a help message to the send channel
	message := "Available commands:\n" +
		"!help - Display this help message\n" +
		"!imagine <text> - Generate an image (other options: !opine, !dream, !nightmare, !hallucinate, !trip)\n" +
		"!summary <num_msgs|12h> - Generate a summary of last N messages, or last H hours\n" +
		"!ask <question> - Ask a question\n"
	ctx.MessagePoster(message, "")
}

func (ctx *AppContext) processMessage(message string) {
	// Start a new span
	tracer := otel.Tracer("signal-bot")
	tracerCtx, span := tracer.Start(ctx.TraceContext, "processMessage", trace.WithNewRoot())
	defer span.End()
	ctx.TraceContext = tracerCtx
	// Process incoming messages from the WebSocket server
	log.Println("Received message:", message)
	// Get the message root
	container, msgStruct, err := getMessageRoot(message)
	if err != nil {
		log.Println("Failed to get message root:", err)
		return
	}

	// If there is no message (for example, this is an emoji reaction), and there are no attachments return
	_, attachmentsOk := msgStruct["attachments"]
	if msgStruct["message"] == nil && !attachmentsOk {
		return
	} else if msgStruct["message"] == nil && attachmentsOk {
		// If there are attachments, but no message, set the message to "Attachment"
		msgStruct["message"] = "Uploaded attachment"
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

	// This is handy to pull out now, we use it later
	sourceName := container["envelope"].(map[string]interface{})["sourceName"].(string)
	msgBody := msgStruct["message"].(string)
	mentions := getMentions(msgStruct)

	// If the message contains attachments, fetch and process them.
	imageData, err := getImageData(msgStruct)
	if err != nil {
		log.Println("Failed to get image data:", err)
		return
	} else if len(imageData) > 0 {
		// If there is image data, append it to the message body
		msgStruct["message"] = msgBody + "\n(Image data: " + strings.Join(imageData, "\n") + ")"
	}

	// Persist the message to the database
	ctx.saveMessage(container, msgStruct, mentions)

	// If the first word in the message starts with a !, it's a command.
	// Take the first word and call the appropriate function with the rest of the message
	// Otherwise, return
	if strings.HasPrefix(msgBody, "!") {
		words := strings.Fields(msgBody)
		switch words[0] {
		case "!help":
			ctx.helpCommand()
		case "!ping":
			msgTime := container["envelope"].(map[string]interface{})["timestamp"].(float64)
			nowTime := time.Now().UnixMilli()
			elapsedMs := nowTime - int64(msgTime)
			ctx.MessagePoster(fmt.Sprintf("Pong! Elapsed time: %d ms", elapsedMs), "")
		case "!imagine", "!opine", "!dream", "!nightmare", "!hallucinate", "!trip":
			// If words[1:] is empty, call help
			if len(words) < 2 {
				ctx.helpCommand()
				return
			} else {
				ctx.imagineCommand(sourceName, strings.Join(words[1:], " "), words[0])
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
				prompt := strings.Join(words[1:], " ")
				prompt = prompt + "\nTry to use the chat log to answer this question. If the answer is not provided in the chat log above,"
				prompt = prompt + "ignore the chat log and provide the best answer you can. "
				prompt = prompt + "Do not be overly verbose in your answers unless asked. Responses under 1000 chars are preferred."
				ctx.summaryCommand(-1, -1, sourceName, prompt)
			}
		}
	}
	// If the message is not a command, call chatCommand to handle the message
	// ctx.chatCommand(sourceName, msgBody, mentions)
}

func (ctx *AppContext) debugger() {
	// Start a debugger session
	log.Println("Starting debugger session")
	// A message template to help us test with. This isn't great, one day we should
	// do this with proper types and structures, and then marshal it to JSON.
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
					"message":%s,
					"expiresInSeconds":604800,
					"viewOnce":false,
					"attachments":[
						{
							"contentType":"image/jpeg",
							"filename":"Andromeda_realigned_tiltshift.jpg",
							"id":"r4aFDRWmi_z2dfVh5iqC.jpg",
							"size":273635,
							"width":2048,
							"height":2048,
							"caption":null,
							"uploadTimestamp":null
						}
					],
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
		// Start a new span for each message
		tracer := otel.Tracer("signal-bot")
		tracerCtx, span := tracer.Start(ctx.TraceContext, "debugger", trace.WithNewRoot())
		defer span.End()
		ctx.TraceContext = tracerCtx

		// Prompt the user for a message
		request := StringPrompt("Enter a message:")
		if request == "" {
			break
		}
		// Put the message in the template
		timeNow := time.Now().Unix() * 1000
		escapedMessage, _ := json.Marshal(request)
		message := fmt.Sprintf(tpl, timeNow, timeNow, string(escapedMessage))
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
			// Start a new span for each message
			tracer := otel.Tracer("signal-bot")
			_, span := tracer.Start(ctx.TraceContext, "websocketClient")
			defer span.End()
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

	// Initialize OpenTelemetry for stdout
	shutdown := initTracer()
	defer shutdown()
	tracer := otel.Tracer("signal-bot")

	// Accept command line arguments with flag:
	//   -mode: websocket or rest
	//   -debug: enable debug logging
	mode := flag.String("mode", "websocket", "start mode: websocket or rest")
	debugflag := flag.Bool("debug", false, "enable debug logging")
	pprofFlag := flag.Bool("pprof", false, "enable pprof")
	flag.Parse()

	// Enable pprof if -pprof was used, OR if PPROF_PORT is set
	if *pprofFlag || Config["PPROF_PORT"] != "" {
		go func() {
			log.Println(http.ListenAndServe("localhost:"+Config["PPROF_PORT"], nil))
		}()
	}
	// Enable debug logging if requested
	if *debugflag {
		log.SetFlags(log.LstdFlags | log.Lshortfile)
	}

	// Create the application context
	traceCtx, span := tracer.Start(context.Background(), *mode)
	defer span.End()

	ctx := AppContext{
		DbQueryChan:        make(chan dbQuery),
		DbReplySummaryChan: make(chan interface{}),
		DbReplyAskChan:     make(chan interface{}),
		Recipients:         []string{},
		TraceContext:       traceCtx,
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
				time.Sleep(3 * time.Second)
			}
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
