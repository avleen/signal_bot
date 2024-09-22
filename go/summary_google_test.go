package main

import (
	"testing"
)

func TestSummaryGoogle(t *testing.T) {
	setupTestEnv()
	ctx := &AppContext{}
	chatLog := "This is a chat log."

	summary, err := ctx.summaryGoogle(chatLog, "")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if summary == "" {
		t.Errorf("expected a response from Google, got '%s'", summary)
	}
}
