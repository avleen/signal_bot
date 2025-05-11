package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
)

// ClaudeMessage represents a message in the Claude API format
type ClaudeMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// ClaudeRequest represents a request to the Claude API
type ClaudeRequest struct {
	Model     string          `json:"model"`
	Messages  []ClaudeMessage `json:"messages"`
	MaxTokens int             `json:"max_tokens,omitempty"`
}

// ClaudeResponse represents a response from the Claude API
type ClaudeResponse struct {
	Content []struct {
		Text string `json:"text"`
	} `json:"content"`
}

func sendClaudeRequest(req ClaudeRequest) (string, error) {
	apiKey := Config["CLAUDE_API_KEY"]
	apiURL := "https://api.anthropic.com/v1/messages"

	// Marshal the request to JSON
	reqBody, err := json.Marshal(req)
	if err != nil {
		return "", fmt.Errorf("error marshaling request: %w", err)
	}

	// Create the HTTP request
	httpReq, err := http.NewRequest("POST", apiURL, bytes.NewBuffer(reqBody))
	if err != nil {
		return "", fmt.Errorf("error creating HTTP request: %w", err)
	}

	// Set the headers
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("x-api-key", apiKey)
	httpReq.Header.Set("anthropic-version", "2023-06-01")

	// Send the request
	client := &http.Client{}
	resp, err := client.Do(httpReq)
	if err != nil {
		return "", fmt.Errorf("error sending request to Claude API: %w", err)
	}
	defer resp.Body.Close()

	// Read the response
	respBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("error reading response body: %w", err)
	}

	// Check for errors
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("claude API error: %s", string(respBody))
	}

	// Unmarshal the response
	var claudeResp ClaudeResponse
	err = json.Unmarshal(respBody, &claudeResp)
	if err != nil {
		return "", fmt.Errorf("error unmarshaling response: %w", err)
	}

	// Return the response text
	if len(claudeResp.Content) > 0 {
		return claudeResp.Content[0].Text, nil
	}

	return "", fmt.Errorf("empty response from Claude API")
}

func ValidateClaudeConfig() error {
	// Validate the correct configuration is set
	if Config["CLAUDE_API_KEY"] == "" {
		return fmt.Errorf("CLAUDE_API_KEY is not set")
	}

	// Get the model name, validate that it's set
	if Config["CLAUDE_MODEL"] == "" {
		return fmt.Errorf("CLAUDE_MODEL is not set")
	}
	return nil
}

func getClaudeModelName() (string, error) {
	modelName := Config["CLAUDE_MODEL"]
	// Map the models we accept
	allowedModels := map[string]string{
		"Claude37Sonnet": "claude-3-7-sonnet-20250219",
		"Claude35Sonnet": "claude-3-5-sonnet-20241022",
		"Claude35Haiku":  "claude-3-5-haiku-20241022",
	}
	// If modelName is not in the allowedModels map, return an error
	if _, ok := allowedModels[modelName]; !ok {
		return "", fmt.Errorf("model %s is not supported", modelName)
	}
	modelName = allowedModels[modelName]
	return modelName, nil
}

func (ctx *AppContext) InitClaudeHistory() (string, error) {
	// Initialize the chat history with the bot's initialization message
	initMsg, err := ioutil.ReadFile("chatbot_init_msg.txt")
	if err != nil {
		return "", fmt.Errorf("failed to read initialization message: %v", err)
	}

	return fmt.Sprintf(string(initMsg), Config["BOTNAME"]), nil
}

func (ctx *AppContext) summaryClaude(chatLog string, prompt string) (string, error) {
	// Start a new span. During testing ctx.TraceContext may be nil so we need to check for that.
	if ctx.TraceContext == nil {
		ctx.TraceContext = context.Background()
	}
	// tracer := otel.Tracer("signal-bot")
	// summaryCtx, span := tracer.Start(ctx.TraceContext, "summaryClaude")
	// defer span.End()

	// Generate a summary of the chat log using the Claude API
	fmt.Println("Generating summary using Claude API")

	// Validate the correct configuration is set
	if Config["CLAUDE_API_KEY"] == "" {
		return "", fmt.Errorf("CLAUDE_API_KEY is not set")
	}

	// Get the model name, validate that it's set
	if Config["CLAUDE_MODEL"] == "" {
		return "", fmt.Errorf("CLAUDE_MODEL is not set")
	}

	modelName, err := getClaudeModelName()
	if err != nil {
		return "", err
	}

	// Use the given prompt, or read from a file if not provided
	if prompt == "" {
		prompt = getSummaryPromptFromFile() + "\n" + chatLog
	} else {
		prompt = prompt + "\n" + chatLog
	}

	// Create the request
	req := ClaudeRequest{
		Model: modelName,
		Messages: []ClaudeMessage{
			{
				Role:    "user",
				Content: prompt,
			},
		},
		MaxTokens: 4096,
	}

	// Send the request to Claude API
	summary, err := sendClaudeRequest(req)
	if err != nil {
		return "", fmt.Errorf("error generating content: %w", err)
	}

	return summary, nil
}
