package config

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestEmbedding_Validate_WhenChatAPIURLWithoutKey_ExpectError(t *testing.T) {
	t.Parallel()

	cfg := Embedding{
		Enabled:    true,
		APIKey:     "embed-key",
		APIURL:     "https://example.com",
		ChatModel:  "gpt-4o",
		ChatAPIURL: "https://chat.example.com",
	}

	err := cfg.Validate()
	require.EqualError(t, err, "embedding: KB_CHAT_API_KEY is required when KB_CHAT_API_URL is set")
}

func TestEmbedding_Validate_WhenChatAPIURLWithKey_ExpectOK(t *testing.T) {
	t.Parallel()

	cfg := Embedding{
		Enabled:    true,
		APIKey:     "embed-key",
		APIURL:     "https://example.com",
		ChatModel:  "gpt-4o",
		ChatAPIURL: "https://chat.example.com",
		ChatAPIKey: "chat-key",
	}

	require.NoError(t, cfg.Validate())
}
