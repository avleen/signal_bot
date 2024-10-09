package main

import (
	"context"
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	openai "github.com/sashabaranov/go-openai"
	"go.opentelemetry.io/otel"
)

func (ctx *AppContext) imagineOpenai(prompt string, requestor string, flavor string) (string, string, error) {
	// Start a new span. During testing ctx.TraceContext may be nil so we need to check for that.
	if ctx.TraceContext == nil {
		ctx.TraceContext = context.Background()
	}
	tracer := otel.Tracer("signal-bot")
	_, span := tracer.Start(ctx.TraceContext, "summaryGoogle")
	defer span.End()
	client := openai.NewClient(Config["OPENAI_API_KEY"])

	err := makeOutputDir(Config["IMAGEDIR"])
	if err != nil {
		fmt.Println("Failed to create output directory:", err)
		return "", "", err

	}

	// Modify the prompt based on the flavor
	switch flavor {
	case "!imagine":
		// No changes needed
	case "!opine":
		prompt = "When creating this image, use a style that conveys seriousness and professionalism. " + prompt
	case "!dream":
		prompt = "When creating this image, use a style that conveys whimsy and imagination in a dream-like state. " + prompt
	case "!nightmare":
		prompt = "When creating this image, use a style that conveys fear and horror in a nightmare-like state. " + prompt
	case "!hallucinate":
		prompt = "When creating this image, use a style that conveys a hallucination-like state. " + prompt
	case "!trip":
		prompt = "When creating this image, use a style that conveys a psychedelic trip-like state. " + prompt
	}

	// Generate an image from the text and send it
	jsonResp, err := client.CreateImage(context.Background(),
		openai.ImageRequest{
			Prompt:         prompt,
			Model:          openai.CreateImageModelDallE3,
			ResponseFormat: openai.CreateImageResponseFormatB64JSON,
			N:              1,
			Size:           openai.CreateImageSize1024x1024,
			Quality:        openai.CreateImageQualityStandard,
		},
	)
	if err != nil {
		fmt.Println("Failed to generate image:", err)
		return "", "", err
	}
	revisedPrompt := jsonResp.Data[0].RevisedPrompt
	data := jsonResp.Data[0].B64JSON

	// Decode the base64 image data
	imageData, err := base64.StdEncoding.DecodeString(data)
	if err != nil {
		fmt.Println("Failed to decode image data:", err)
		return "", "", err
	}

	// Save the imageData to a file in the format:
	// <date>-<time>-<requestor>.png
	filename := fmt.Sprintf("%s-%s-%s.png", time.Now().Format("2006-01-02"), time.Now().Format("15:04:05"), requestor)
	filename, err = filepath.Abs(filepath.Join(Config["IMAGEDIR"], filename))
	if err != nil || !strings.HasPrefix(filename, Config["IMAGEDIR"]) {
		fmt.Println("Invalid file name:", err)
		return "", "", err
	}
	file, err := os.OpenFile(filename, os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		fmt.Println("Failed to open file:", err)
		return "", "", err
	}
	defer file.Close()
	// Write the image data to the file
	_, err = file.Write(imageData)
	if err != nil {
		fmt.Println("Failed to write image data to file:", err)
		return "", "", err
	}
	fmt.Printf("Image saved to %s with revised prompt: %s\n", filename, revisedPrompt)

	return filename, revisedPrompt, nil
}
