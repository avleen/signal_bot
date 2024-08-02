package main

import (
	"fmt"
	"log"
)

func (ctx *AppContext) imagineCommand(requestor string, prompt string) {
	var filename, revisedPrompt string
	var err error

	// Generate an image from the text and send it to the send channel
	fmt.Printf("Generating image for %s: %s\n", requestor, prompt)

	// Generate the image
	switch Config["IMAGE_PROVIDER"] {
	case "openai":
		// Generate the image using OpenAI
		filename, revisedPrompt, err = imagineOpenai(prompt, requestor)
		if err != nil {
			log.Println("Failed to generate image:", err)
			return
		}
	case "google":
		// Generate the image using Google
		filename, revisedPrompt, err = imagineGoogle(prompt, requestor)
		if err != nil {
			log.Println("Failed to generate image:", err)
			ctx.MessagePoster("Failed to generate image: "+err.Error(), "")
			return
		}
	}

	ctx.MessagePoster(revisedPrompt, filename)
}
