package main

import (
	"context"
	"fmt"

	openai "github.com/sashabaranov/go-openai"
)

func summaryOpenai(chatLog string, prompt string) (string, error) {
	client := openai.NewClient(config["OPENAI_API_KEY"])

	// Use the given prompt, or read from a file if not provided
	if prompt == "" {
		prompt = getSummaryPromptFromFile() + "\n" + chatLog
	} else {
		prompt = prompt + "\n" + chatLog
	}

	// Talk to ChatGPT to generate a summary
	req := openai.ChatCompletionRequest{
		Model: openai.GPT3Dot5Turbo,
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

	fmt.Println(resp.Choices[0].Message.Content)
	return resp.Choices[0].Message.Content, nil
}
