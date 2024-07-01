package main

import (
	"errors"
	"fmt"
	"log"
	"os"
	"strings"
	"time"
)

func getSummaryPromptFromFile() string {
	// Get the prompt from the file
	prompt, err := os.ReadFile("prompt_summary.txt")
	if err != nil {
		log.Println("Failed to read prompt file:", err)
		return ""
	}
	return string(prompt)
}

func (c *TimeCountCalculator) calculateStarttimeAndCount(words []string) (int, int, error) {
	// Calculates the starttime and message count requested from `words`

	// c.StartTime is used for testing. If it's -1, set a new default start time.
	if c.StartTime == -1 {
		c.StartTime = int(time.Now().Add(-time.Duration(24)*time.Hour).Unix() * 1000)
	}

	// If no arguments are given, return the default start time and 0 count
	if len(words) < 2 {
		return c.StartTime, c.Count, nil
	}

	// See if the first argument has a number
	number, err := getNumberFromString(words[1])
	if err != nil {
		// If not, return the default start time and 0 count
		return 0, 0, errors.New("Invalid argument to summary: " + words[1])
	}

	// If the first argument ends in "h", return that many hours as starttime and a zero count
	if strings.HasSuffix(words[1], "h") {
		return int(time.Now().Add(-time.Duration(number)*time.Hour).Unix() * 1000), 0, nil
	}
	// Otherwise, return zero for the start time and the requested count
	return 0, number, nil
}

func (ctx *AppContext) summaryCommand(starttime int, count int, sourceName string, prompt string) {
	var summary string
	// Generate a summary of the last N messages or last H hours
	// and send it to the send channel
	fmt.Printf("Generating summary for %s: hours: %d, count: %d\n", sourceName, starttime, count)

	rows, err := ctx.fetchLogsFromDB(starttime, count)
	if err != nil {
		log.Println("Failed to fetch logs:", err)
		return
	}

	chatLog, err := compileLogs(rows)
	if err != nil {
		log.Println("Failed to compile logs:", err)
		return
	}

	switch Config["SUMMARY_PROVIDER"] {
	case "google":
		summary, err = summaryGoogle(chatLog, prompt)
		if err != nil {
			log.Println("Failed to generate summary:", err)
			return
		}
	case "openai":
		summary, err = summaryOpenai(chatLog, prompt)
		if err != nil {
			log.Println("Failed to generate summary:", err)
			return
		}
	case "debug":
		summary = fmt.Sprintf("DEBUG: Requested %d starttime, %d message count\n"+
			"Chat log: %s", starttime, count, chatLog)
	}
	ctx.MessagePoster(summary, "")
}
