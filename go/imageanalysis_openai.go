package main

import (
	"context"
	"fmt"

	openai "github.com/sashabaranov/go-openai"
)

func imageAnalysisOpenai(attachmentId string) (string, error) {
	// Generate a chat response using OpenAI's Chat API
	setup_err := ValidateChatConfig()
	if setup_err != nil {
		return "", setup_err
	}

	modelName, model_err := getChatModelName()
	if model_err != nil {
		return "", model_err
	}

	client := openai.NewClient(Config["OPENAI_API_KEY"])

	attachment, err := downloadImage(attachmentId)
	if err != nil {
		return "", err
	}

	// Analyze the image using OpenAI's image analysis model
	req := openai.ChatCompletionRequest{
		// Model: "gpt-4-vision-preview",
		Model: modelName,
		Messages: []openai.ChatCompletionMessage{
			{
				Role: openai.ChatMessageRoleUser,
				MultiContent: []openai.ChatMessagePart{
					{
						Type: openai.ChatMessagePartTypeText,
						Text: "Please describe this image in detail.",
					},
					{
						Type: openai.ChatMessagePartTypeImageURL,
						ImageURL: &openai.ChatMessageImageURL{
							URL: "data:image/jpeg;base64," + attachment,
						},
					},
				},
			},
		},
		MaxTokens: 300,
	}

	resp, err := client.CreateChatCompletion(context.Background(), req)
	if err != nil {
		fmt.Printf("ChatCompletion error: %v\n", err)
		return "", err
	}

	assistantResponse := resp.Choices[0].Message.Content

	fmt.Printf("Image analysis response: %s\n", assistantResponse)
	return assistantResponse, nil
}
