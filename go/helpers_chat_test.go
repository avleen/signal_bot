package main

import (
	"testing"

	openai "github.com/sashabaranov/go-openai"
)

func TestGetChatModelName(t *testing.T) {
	tests := []struct {
		name      string
		config    map[string]string
		want      string
		expectErr bool
	}{
		{
			name: "Valid model GPT4o",
			config: map[string]string{
				"OPENAI_CHAT_MODEL": "GPT4o",
			},
			want:      openai.GPT4o,
			expectErr: false,
		},
		{
			name: "Valid model GPT4oMini",
			config: map[string]string{
				"OPENAI_CHAT_MODEL": "GPT4oMini",
			},
			want:      openai.GPT4oMini,
			expectErr: false,
		},
		{
			name: "Invalid model",
			config: map[string]string{
				"OPENAI_CHAT_MODEL": "InvalidModel",
			},
			want:      "",
			expectErr: true,
		},
		{
			name: "Empty model",
			config: map[string]string{
				"OPENAI_CHAT_MODEL": "",
			},
			want:      "",
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			Config = tt.config
			got, err := getChatModelName()
			if (err != nil) != tt.expectErr {
				t.Errorf("getChatModelName() error = %v, expectErr %v", err, tt.expectErr)
				return
			}
			if got != tt.want {
				t.Errorf("getChatModelName() = %v, want %v", got, tt.want)
			}
		})
	}
}
func TestCheckIfMentioned(t *testing.T) {
	tests := []struct {
		name     string
		mentions []map[string]string
		config   map[string]string
		want     bool
	}{
		{
			name: "Mentioned by name",
			mentions: []map[string]string{
				{"name": "BotName"},
			},
			config: map[string]string{
				"BOTNAME": "BotName",
			},
			want: true,
		},
		{
			name: "Mentioned by phone",
			mentions: []map[string]string{
				{"number": "1234567890"},
			},
			config: map[string]string{
				"PHONE": "1234567890",
			},
			want: true,
		},
		{
			name: "Not mentioned",
			mentions: []map[string]string{
				{"name": "OtherName"},
				{"number": "0987654321"},
			},
			config: map[string]string{
				"BOTNAME": "BotName",
				"PHONE":   "1234567890",
			},
			want: false,
		},
		{
			name:     "Empty mentions",
			mentions: []map[string]string{},
			config: map[string]string{
				"BOTNAME": "BotName",
				"PHONE":   "1234567890",
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			Config = tt.config
			if got := checkIfMentioned(tt.mentions); got != tt.want {
				t.Errorf("checkIfMentioned() = %v, want %v", got, tt.want)
			}
		})
	}
}
