package main

import (
	"errors"
	"fmt"
	"log"
	"os"
	"regexp"
	"strings"
	"time"

	"go.opentelemetry.io/otel"
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
	// Start a new span
	tracer := otel.Tracer("signal-bot")
	_, span := tracer.Start(ctx.TraceContext, "summaryCommand")
	defer span.End()

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
		summary, err = ctx.summaryGoogle(chatLog, prompt)
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

	// Split the summary into chunks and call ctx.MessagePoster for each chunk
	summaryChunks := splitSummary(summary)
	for _, chunk := range summaryChunks {
		ctx.MessagePoster(chunk, "")
	}
}

// Function to split the summary into chunks less than or equal to maxSummaryLength
func splitSummary(summary string) []string {
	const maxSummaryLength = 2000
	var chunks []string

	for len(summary) > maxSummaryLength {
		// Find the substring from index 0 to maxSummaryLength
		substring := summary[:maxSummaryLength]

		// Check if the substring ends with a paragraph header
		re := regexp.MustCompile(`\*\*[\w\d\s]+:\*\*`)
		matches := re.FindAllStringIndex(substring, -1)
		if len(matches) > 0 {
			// Find the index of the start of the most recent paragraph
			paragraphStart := matches[len(matches)-1][0]

			// If a paragraph start is found, split the summary at that index
			if paragraphStart > 0 {
				chunks = append(chunks, substring[:paragraphStart])
				summary = summary[paragraphStart:]
				continue
			}
		}

		// If no paragraph start is found, split the summary at maxSummaryLength
		chunks = append(chunks, substring)
		summary = summary[maxSummaryLength:]
	}

	// Add the remaining part of the summary as the last chunk
	chunks = append(chunks, summary)

	return chunks
}
