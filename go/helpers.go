// Helper functions

package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
)

func getMessageRoot(message string) map[string]interface{} {
	// Parse the message and return the root message object

	var msgStruct map[string]interface{}
	container := make(map[string]interface{})
	json.Unmarshal([]byte(message), &container)

	if dataMessage, ok := container["envelope"].(map[string]interface{})["dataMessage"]; ok {
		msgStruct = dataMessage.(map[string]interface{})
	} else if syncMessage, ok := container["envelope"].(map[string]interface{})["syncMessage"]; ok {
		if sentMessage, ok := syncMessage.(map[string]interface{})["sentMessage"]; ok {
			msgStruct = sentMessage.(map[string]interface{})
		} else {
			return nil
		}
	}
	// Put msgStruct back in the envelope
	container["msgStruct"] = msgStruct
	return container
}

func encodeGroupIdToBase64(groupId string) string {
	// Convert the groupId to base64
	groupIdBase64 := base64.StdEncoding.EncodeToString([]byte(groupId))
	return fmt.Sprintf("group.%s", groupIdBase64)
}
