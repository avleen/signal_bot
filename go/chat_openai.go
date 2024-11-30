package main

import (
	"context"
	"fmt"

	openai "github.com/sashabaranov/go-openai"
)

func (ctx *AppContext) chatOpenai(sourceName string, msgBody string) (string, error) {
	// Generate a chat response using OpenAI's Chat API

	// Validate the correct configuration is set
	if Config["OPENAI_API_KEY"] == "" {
		return "", fmt.Errorf("OPENAI_API_KEY is not set")
	}

	// Get the model name, validate that it's set
	if Config["OPENAI_CHAT_MODEL"] == "" {
		return "", fmt.Errorf("OPENAI_CHAT_MODEL is not set")
	}
	modelName := Config["OPENAI_CHAT_MODEL"]
	// Rather than trying to use reflection magic, we'll map the models we accept.
	// This mean we'll have to keep this list up to date as OpenAI add more
	// models but it's not a big lift.
	allowedModels := map[string]string{
		"GPT4o":     openai.GPT4o,
		"GPT4oMini": openai.GPT4oMini,
	}
	// If modelName is not in the allowedModels map, return an error.
	// We reuse the existing modelName variable here.
	if _, ok := allowedModels[modelName]; !ok {
		return "", fmt.Errorf("model %s is not supported", modelName)
	}
	modelName = allowedModels[modelName]

	client := openai.NewClient(Config["OPENAI_API_KEY"])

	// If ctx.MessageHistory is empty, let's start one with the user's message
	if len(ctx.MessageHistory) == 0 {
		// If we are not using the O1Mini model, we need to add a completion message
		ctx.MessageHistory = []openai.ChatCompletionMessage{
			{
				Role: openai.ChatMessageRoleSystem,
				Content: fmt.Sprintf(`Your name is %s.
You are a bot who is a member of a chat group.
You do not have to respond to every comment you see.
You should respond to questions if you have a helpful answer.
Contribute to the conversation with the other participants.
Messages start with the name of the speaker.
If you don't have a helpful reply, just answer with "None",
Remember and follow the instructions you are given in the chat.`, Config["BOTNAME"]),
			},
		}
	}
	ctx.MessageHistory = append(ctx.MessageHistory, openai.ChatCompletionMessage{
		Role:    openai.ChatMessageRoleUser,
		Content: sourceName + ": " + msgBody,
	})

	// Talk to ChatGPT to generate a summary
	req := openai.ChatCompletionRequest{
		Model:    modelName,
		Messages: ctx.MessageHistory,
	}

	resp, err := client.CreateChatCompletion(context.Background(), req)
	if err != nil {
		fmt.Printf("ChatCompletion error: %v\n", err)
		return "", err
	}

	assistantResponse := resp.Choices[0].Message.Content
	ctx.MessageHistory = append(ctx.MessageHistory, openai.ChatCompletionMessage{
		Role:    openai.ChatMessageRoleAssistant,
		Content: assistantResponse,
	})

	return assistantResponse, nil
}
