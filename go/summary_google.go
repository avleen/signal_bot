package main

import (
	"context"
	"fmt"
	"strings"

	"cloud.google.com/go/vertexai/genai"
)

func summaryGoogle(chatLog string, prompt string) (string, error) {
	// Generate a summary of the chat log using the Google AI API
	// and send it to the send channel
	fmt.Println("Generating summary using Google AI API")

	// Validate the correct configuration is set
	for _, key := range []string{"GOOGLE_PROJECT_ID", "GOOGLE_LOCATION", "GOOGLE_TEXT_MODEL"} {
		if Config[key] == "" {
			return "", fmt.Errorf("%s is not set", key)
		}
	}

	location := Config["LOCATION"]
	projectID := Config["PROJECT_ID"]

	ctx := context.Background()
	client, err := genai.NewClient(ctx, projectID, location)
	if err != nil {
		return "", fmt.Errorf("error creating google vertex client: %w", err)
	}
	defer client.Close()
	model := client.GenerativeModel(Config["GOOGLE_TEXT_MODEL"])
	model.SafetySettings = []*genai.SafetySetting{
		{
			Category:  genai.HarmCategoryHarassment,
			Threshold: genai.HarmBlockOnlyHigh,
		},
		{
			Category:  genai.HarmCategoryDangerousContent,
			Threshold: genai.HarmBlockOnlyHigh,
		},
		{
			Category:  genai.HarmCategorySexuallyExplicit,
			Threshold: genai.HarmBlockOnlyHigh,
		},
		{
			Category:  genai.HarmCategoryHateSpeech,
			Threshold: genai.HarmBlockOnlyHigh,
		},
	}

	// Use the given prompt, or read from a file if not provided
	if prompt == "" {
		prompt = getSummaryPromptFromFile() + "\n" + chatLog
	} else {
		prompt = prompt + "\n" + chatLog
	}
	resp, err := model.GenerateContent(ctx, genai.Text(prompt))
	if err != nil {
		return "", fmt.Errorf("error generating content: %w", err)
	}

	var summary []string
	for _, cand := range resp.Candidates {
		for _, part := range cand.Content.Parts {
			summary = append(summary, fmt.Sprintf("%s", part))
		}
	}
	return strings.Join(summary, "\n"), nil
}
