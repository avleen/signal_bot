package main

import (
	"fmt"
	"os"

	openai "github.com/sashabaranov/go-openai"
)

func ValidateChatConfig() error {
	// Validate the correct configuration is set
	if Config["OPENAI_API_KEY"] == "" {
		return fmt.Errorf("OPENAI_API_KEY is not set")
	}

	// Get the model name, validate that it's set
	if Config["OPENAI_CHAT_MODEL"] == "" {
		return fmt.Errorf("OPENAI_CHAT_MODEL is not set")
	}
	return nil
}

func getChatModelName() (string, error) {
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
	return modelName, nil
}

func (ctx *AppContext) InitChatHistory() ([]openai.ChatCompletionMessage, error) {
	// Initialize the chat history with the bot's initialization message
	messages := []openai.ChatCompletionMessage{}
	initMsg, err := os.ReadFile("chatbot_init_msg.txt")
	if err != nil {
		return messages, fmt.Errorf("failed to read initialization message: %v", err)
	}
	messages = append(messages, openai.ChatCompletionMessage{
		Role:    openai.ChatMessageRoleSystem,
		Content: fmt.Sprintf(string(initMsg), Config["BOTNAME"]),
	})
	return messages, nil
}

/* func checkIfMentioned(message string) bool {
	var container map[string]interface{}
	json.Unmarshal([]byte(message), &container)

	if dataMessage, ok := container["envelope"].(map[string]interface{})["dataMessage"]; ok {
		if msg, ok := dataMessage.(map[string]interface{})["message"]; ok {
			if strings.Contains(strings.ToLower(msg.(string)), strings.ToLower(Config["BOTNAME"])) {
				return true
			}
		}
		if mentions, ok := dataMessage.(map[string]interface{})["mentions"]; ok {
			for _, mention := range mentions.([]interface{}) {
				mentionMap := mention.(map[string]interface{})
				if strings.Contains(strings.ToLower(mentionMap["name"].(string)), strings.ToLower(Config["BOTNAME"])) ||
					mentionMap["number"].(string) == Config["PHONE"] {
					return true
				}
			}
		}
	} else if syncMessage, ok := container["envelope"].(map[string]interface{})["syncMessage"]; ok {
		if sentMessage, ok := syncMessage.(map[string]interface{})["sentMessage"]; ok {
			if msg, ok := sentMessage.(map[string]interface{})["message"]; ok {
				if strings.Contains(strings.ToLower(msg.(string)), strings.ToLower(Config["BOTNAME"])) {
					return true
				}
			}
			if mentions, ok := sentMessage.(map[string]interface{})["mentions"]; ok {
				for _, mention := range mentions.([]interface{}) {
					mentionMap := mention.(map[string]interface{})
					if strings.Contains(strings.ToLower(mentionMap["name"].(string)), strings.ToLower(Config["BOTNAME"])) ||
						mentionMap["number"].(string) == Config["PHONE"] {
						return true
					}
				}
			}
		}
	}
	return false
} */

func (ctx *AppContext) fetchChatbotHistoryFromDb() []map[string]string {
	// Hydrate the chat history from the database
	// Send the query to the database and return the result
	query := fmt.Sprintf(`WITH last_100_messages AS (
    SELECT sourceName, message, created_at FROM messages
    WHERE sourceName != '%s'
      AND NOT EXISTS (
          SELECT 1 FROM json_each(messages.mentions)
          WHERE json_each.value = '%s'
      )
      AND message NOT LIKE '%%%s%%'
    ORDER BY created_at DESC
    LIMIT 100
)
SELECT sourceName, message, created_at FROM messages WHERE sourceName = '%s'
UNION ALL
SELECT sourceName, message, created_at FROM messages
WHERE EXISTS (
    SELECT 1 FROM json_each(messages.mentions)
    WHERE json_each.value = '%s'
)
UNION ALL
SELECT sourceName, message, created_at FROM messages WHERE message LIKE '%%%s%%'
UNION ALL
SELECT sourceName, message, created_at FROM last_100_messages
ORDER BY created_at ASC;
`, Config["BOTNAME"], Config["BOTNAME"], Config["BOTNAME"], Config["BOTNAME"], Config["BOTNAME"], Config["BOTNAME"])

	replyChan := make(chan dbReply, 1)
	defer close(replyChan)
	ctx.DbQueryChan <- dbQuery{query, nil, replyChan}
	rows := <-replyChan
	// Get the results from the db. Store the results in an array of arrays as [sourceName, message]
	var chatHistory []map[string]string
	for rows.rows.Next() {
		var sourceName, message, created_at string
		err := rows.rows.Scan(&sourceName, &message, &created_at)
		if err != nil {
			fmt.Println("Failed to scan log:", err)
			continue
		}
		chatHistory = append(chatHistory, map[string]string{"sourceName": sourceName, "message": message})
	}
	return chatHistory
}
