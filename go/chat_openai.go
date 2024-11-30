package main

import (
	"context"
	"fmt"
	"strings"
	"time"

	openai "github.com/sashabaranov/go-openai"
)

func (ctx *AppContext) chatOpenai(msgBody string) (string, error) {
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

	messages, err := ctx.InitChatHistory()
	if err != nil {
		return "", err
	}

	// Get the chatbot history from the database. Iterate over the rows and add them to the chat history.
	// If the sourceName is the same as the chatbot name, add the message as an assistant message.
	// Otherwise, add the message as a user message.
	chatbotHistory := ctx.fetchChatbotHistoryFromDb()
	for _, row := range chatbotHistory {
		if row["sourceName"] == Config["BOTNAME"] {
			messages = append(messages, openai.ChatCompletionMessage{
				Role:    openai.ChatMessageRoleAssistant,
				Content: row["message"],
			})
		} else {
			messages = append(messages, openai.ChatCompletionMessage{
				Role:    openai.ChatMessageRoleUser,
				Content: fmt.Sprintf("%s: %s", row["sourceName"], row["message"]),
			})
		}
	}
	// Add the user message to the chat history
	messages = append(messages, openai.ChatCompletionMessage{
		Role:    openai.ChatMessageRoleUser,
		Content: msgBody,
	})

	// Talk to ChatGPT to generate a summary
	req := openai.ChatCompletionRequest{
		Model:    modelName,
		Messages: messages,
	}
	// Print the request to the console
	fmt.Printf("ChatCompletion request: %v\n", req)

	resp, err := client.CreateChatCompletion(context.Background(), req)
	if err != nil {
		fmt.Printf("ChatCompletion error: %v\n", err)
		return "", err
	}

	assistantResponse := resp.Choices[0].Message.Content
	// If the assistant response was not "<NO_RESPONSE>", add it to the chat history and return it.
	// Otherwise return an empty string.
	// In order to save the message, we first need to construct a fake message object with the message content,
	// in the format Signal uses to pass to saveMessage()
	// The assistantResponse may have newline characters, so we need to escape them first or the JSON will be invalid.
	escapedString := strings.ReplaceAll(assistantResponse, "\n", "\\n")
	// Sometimes the bot name has double quotes around it? Don't know why, but we need to remove them.
	cleanBotName := strings.ReplaceAll(Config["BOTNAME"], "\"", "")
	if assistantResponse != "<NO_RESPONSE>" {
		ts := time.Now().UnixMilli()
		chatbotMessage := fmt.Sprintf(`{
			"envelope": {
				"sourceNumber": "%s",
				"sourceName": "%s",
				"timestamp": "%d",
				"syncMessage": {
				    "sentMessage": {
						"message": "%s",
						"groupInfo": {
							"groupId": "%s"
						}
					}
				}
			}
		}`, Config["PHONE"], cleanBotName, ts, escapedString, "group.1234567890")
		container, msgStruct, err := getMessageRoot(chatbotMessage)
		if err != nil {
			return "", err
		}
		ctx.saveMessage(container, msgStruct)
		return assistantResponse, nil
	}
	return "", nil
}
