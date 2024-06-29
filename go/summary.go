package main

import (
	"fmt"
	"log"
)

func (ctx *AppContext) summaryCommand(starttime int, count int, sourceName string) {
	var summary string
	// Generate a summary of the last N messages or last H hours
	// and send it to the send channel
	fmt.Printf("Generating summary for %s: hours: %d, count: %d\n", sourceName, starttime, count)

	rows, err := ctx.fetchLogsFromDB(count, starttime)
	if err != nil {
		log.Println("Failed to fetch logs:", err)
		return
	}

	chatLog, err := compileLogs(rows)
	if err != nil {
		log.Println("Failed to compile logs:", err)
		return
	}

	switch config["SUMMARY_PROVIDER"] {
	case "google":
		summary, err = summaryGoogle(chatLog)
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
