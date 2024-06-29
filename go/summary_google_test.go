package main

import (
	"testing"
)

func TestSummaryGoogle(t *testing.T) {
	setupTestEnv()
	chatLog := "This is a chat log."
	expectedSummary := "This is the summary."

	summary, err := summaryGoogle(chatLog, "")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if summary != expectedSummary {
		t.Errorf("expected summary '%s', got '%s'", expectedSummary, summary)
	}
}
