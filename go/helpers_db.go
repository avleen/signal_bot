package main

import (
	"log"
	"strconv"
	"time"

	"go.opentelemetry.io/otel"
)

func (ctx *AppContext) removeOldMessages() {
	// Start a new span
	tracer := otel.Tracer("signal-bot")
	_, span := tracer.Start(ctx.TraceContext, "removeOldMessages")
	defer span.End()

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
