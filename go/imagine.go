package main

import (
	"encoding/base64"
	"fmt"
	"io"
	"log"
	"os"
)

func (ctx *AppContext) imagineCommand(requestor string, prompt string) {
	var filename, revisedPrompt string
	var err error

	// Generate an image from the text and send it to the send channel
	fmt.Printf("Generating image for %s: %s\n", requestor, prompt)

	// Generate the image
	switch config["IMAGE_PROVIDER"] {
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
			return
		}
	}

	// Open filename and read the image data
	file, err := os.OpenFile(filename, os.O_RDONLY, 0644)
	if err != nil {
		log.Println("Failed to open file:", err)
		return
	}
	defer file.Close()
	imageData, err := io.ReadAll(file)
	if err != nil {
		log.Println("Failed to read image data:", err)
		return
	}

	// Convert image data to base64
	imageDataB64 := base64.StdEncoding.EncodeToString(imageData)

	ctx.MessagePoster(revisedPrompt, imageDataB64)
}
