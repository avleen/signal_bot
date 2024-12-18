package main

import (
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
)

func downloadImage(attachmentId string) (string, error) {
	url := fmt.Sprintf("http://%s/v1/attachments/%s", Config["URL"], attachmentId)
	resp, err := http.Get(url)
	if err != nil {
		return "", fmt.Errorf("failed to download attachment: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("failed to download attachment: received status code %d", resp.StatusCode)
	}

	imageData, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read attachment data: %v", err)
	}

	attachment := base64.StdEncoding.EncodeToString(imageData)
	return attachment, nil
}
