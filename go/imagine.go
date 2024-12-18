package main

import (
	"context"
	"fmt"
	"log"

	"go.opentelemetry.io/otel"
)

func (ctx *AppContext) imagineCommand(requestor string, prompt string, flavor string) {
	// Start a new span. During testing ctx.TraceContext may be nil so we need to check for that.
	if ctx.TraceContext == nil {
		ctx.TraceContext = context.Background()
	}
	tracer := otel.Tracer("signal-bot")
	_, span := tracer.Start(ctx.TraceContext, "imagineCommand")
	defer span.End()

	var filename, revisedPrompt string
	var err error

	// Generate an image from the text and send it to the send channel
	fmt.Printf("Generating image for %s: %s\n", requestor, prompt)

	// Generate the image
	switch Config["IMAGE_PROVIDER"] {
	case "openai":
		// Generate the image using OpenAI
		filename, revisedPrompt, err = ctx.imagineOpenai(prompt, requestor, flavor)
		if err != nil {
			log.Println("Failed to generate image:", err)
			ctx.MessagePoster("Failed to generate image: "+err.Error(), "")
			return
		}
	case "google":
		// Generate the image using Google
		filename, revisedPrompt, err = imagineGoogle(prompt, requestor, flavor)
		if err != nil {
			log.Println("Failed to generate image:", err)
			ctx.MessagePoster("Failed to generate image: "+err.Error(), "")
			return
		}
	}

	ctx.MessagePoster(revisedPrompt, filename)
}
