package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
)

// Claude image analysis message with image content
type ClaudeContentBlock struct {
	Type   string             `json:"type"`
	Text   string             `json:"text,omitempty"`
	Source *ClaudeImageSource `json:"source,omitempty"`
}

type ClaudeImageSource struct {
	Type      string `json:"type"`
	MediaType string `json:"media_type"`
	Data      string `json:"data"`
}

type ClaudeImageRequest struct {
	Model     string               `json:"model"`
	MaxTokens int                  `json:"max_tokens"`
	Messages  []ClaudeImageMessage `json:"messages"`
}

type ClaudeImageMessage struct {
	Role    string               `json:"role"`
	Content []ClaudeContentBlock `json:"content"`
}

type ClaudeImageResponse struct {
	Content []struct {
		Text string `json:"text"`
	} `json:"content"`
}

func getImageDataClaude(msgStruct map[string]interface{}) ([]string, error) {
	// If the message contains attachments, fetch and process them.
	var imageData []string
	if attachments, ok := msgStruct["attachments"].([]interface{}); ok {
		for _, attachment := range attachments {
			attachmentMap := attachment.(map[string]interface{})
			// If the attachment is an image, call the imageAnalysisFunction
			if attachmentMap["contentType"] == "image/jpeg" {
				imageAnalysis, err := imageAnalysisClaude(attachmentMap["id"].(string))
				if err != nil {
					log.Println("Failed to process image:", err)
				} else {
					// Append the image analysis to the message body
					imageData = append(imageData, imageAnalysis)
				}
			}
		}
	}
	return imageData, nil
}

func imageAnalysisClaude(attachmentId string) (string, error) {
	// Generate image analysis using Claude's API
	setup_err := ValidateClaudeConfig()
	if setup_err != nil {
		return "", setup_err
	}

	modelName, model_err := getClaudeModelName()
	if model_err != nil {
		return "", model_err
	}

	// Download the image
	imageBase64, err := downloadImage(attachmentId)
	if err != nil {
		return "", err
	}

	// Create the image analysis request
	req := ClaudeImageRequest{
		Model:     modelName,
		MaxTokens: 300,
		Messages: []ClaudeImageMessage{
			{
				Role: "user",
				Content: []ClaudeContentBlock{
					{
						Type: "text",
						Text: "Please describe this image in detail.",
					},
					{
						Type: "image",
						Source: &ClaudeImageSource{
							Type:      "base64",
							MediaType: "image/jpeg",
							Data:      imageBase64,
						},
					},
				},
			},
		},
	}

	// Send the request to Claude API
	assistantResponse, err := sendClaudeImageRequest(req)
	if err != nil {
		fmt.Printf("Claude API error: %v\n", err)
		return "", err
	}

	fmt.Printf("Image analysis response: %s\n", assistantResponse)
	return assistantResponse, nil
}

func sendClaudeImageRequest(req ClaudeImageRequest) (string, error) {
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
	var claudeResp ClaudeImageResponse
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
