package main

import (
	"fmt"
	"log"

	"go.opentelemetry.io/otel"
)

func (ctx *AppContext) chatCommand(sourceName string, msgBody string) {
	// Start a new span
	tracer := otel.Tracer("signal-bot")
	_, span := tracer.Start(ctx.TraceContext, "chatCommand")
	defer span.End()

	var resp string
	var err error

	// Have a chat with the bot
	switch Config["CHAT_PROVIDER"] {
	case "openai":
		resp, err = ctx.chatOpenai(msgBody)
		if err != nil {
			log.Println("Failed to converse with the bot:", err)
			ctx.MessagePoster("Failed to converse with the bot: "+err.Error(), "")
			return
		}
	case "debug":
		resp = fmt.Sprintf("DEBUG: Chatting with %s\n"+"Message: %s", sourceName, msgBody)
	}

	// Split the summary into chunks and call ctx.MessagePoster for each chunk
	chatChunks := splitLongMessage(resp)
	for _, chunk := range chatChunks {
		ctx.MessagePoster(chunk, "")
	}
}
