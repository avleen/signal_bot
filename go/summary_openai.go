package main

import (
	"context"
	"fmt"

	openai "github.com/sashabaranov/go-openai"
)

func summaryOpenai(chatLog string, prompt string) (string, error) {
	// Generate a summary using OpenAI's ChatGPT

	// Validate the correct configuration is set
	if Config["OPENAI_API_KEY"] == "" {
		return "", fmt.Errorf("OPENAI_API_KEY is not set")
	}

	// Get the model name, validate that it's set
	if Config["OPENAI_MODEL"] == "" {
		return "", fmt.Errorf("OPENAI_MODEL is not set")
	}
	modelName := Config["OPENAI_MODEL"]
	// Rather than trying to use reflection magic, we'll map the models we accept.
	// This mean we'll have to keep this list up to date as OpenAI add more
	// models but it's not a big lift.
	allowedModels := map[string]string{
		"GPT3Dot5Turbo": openai.GPT3Dot5Turbo,
		"GPT4o":         openai.GPT4o,
		"O1Mini":        openai.O1Mini,
	}
	// If modelName is not in the allowedModels map, return an error.
	// We reuse the existing modelName variable here.
	if _, ok := allowedModels[modelName]; !ok {
		return "", fmt.Errorf("model %s is not supported", modelName)
	}
	modelName = allowedModels[modelName]

	client := openai.NewClient(Config["OPENAI_API_KEY"])

	// Use the given prompt, or read from a file if not provided
	if prompt == "" {
		prompt = getSummaryPromptFromFile() + "\n" + chatLog
	} else {
		prompt = prompt + "\n" + chatLog
	}

	// Talk to ChatGPT to generate a summary
	req := openai.ChatCompletionRequest{
		Model: modelName,
		Messages: []openai.ChatCompletionMessage{
			{
				Role:    openai.ChatMessageRoleSystem,
				Content: "you are a helpful chatbot",
			},
			{
				Role:    openai.ChatMessageRoleUser,
				Content: prompt,
			},
		},
	}

	resp, err := client.CreateChatCompletion(context.Background(), req)
	if err != nil {
		fmt.Printf("ChatCompletion error: %v\n", err)
		return "", err
	}

	return resp.Choices[0].Message.Content, nil
}
