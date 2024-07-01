package main

import (
	"bytes"
	"os"
	"testing"
)

func TestStartupValidator(t *testing.T) {
	// Set up the test environment
	dbFile, err := os.CreateTemp("", "test_*.db")
	if err != nil {
		t.Fatal("Failed to create temporary db file")
	}
	defer os.Remove(dbFile.Name())
	Config = map[string]string{
		"MAX_AGE":           "168",
		"IMAGEDIR":          "/path/to/images",
		"STATEDB":           dbFile.Name(),
		"REST_URL":          "http://localhost:8080",
		"PHONE":             "+123456789",
		"URL":               "ws://localhost:8080",
		"GOOGLE_TEXT_MODEL": "text_model",
		"IMAGE_PROVIDER":    "image_provider",
		"LOCATION":          "location",
		"OPENAI_API_KEY":    "api_key",
		"PROJECT_ID":        "project_id",
		"SUMMARY_PROVIDER":  "summary_provider",
	}

	// Call the startupValidator function
	startupValidator()

	// Assert that there are no panics or fatal errors
}
func TestHelpCommand(t *testing.T) {
	ctx := &AppContext{}
	expectedMessage := "Available commands:\n" +
		"!help - Display this help message\n" +
		"!imagine <text> - Generate an image\n" +
		"!summary <num_msgs|12h> - Generate a summary of last N messages, or last H hours\n" +
		"!ask <question> - Ask a question\n"

	// Redirect the output of the function to a buffer
	var buf bytes.Buffer
	ctx.MessagePoster = func(message, _ string) {
		buf.WriteString(message)
	}

	// Call the helpCommand function
	ctx.helpCommand()

	// Check if the output matches the expected message
	if buf.String() != expectedMessage {
		t.Errorf("Unexpected help message. Expected: %q, Got: %q", expectedMessage, buf.String())
	}
}
