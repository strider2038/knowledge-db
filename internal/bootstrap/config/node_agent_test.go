package config_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/strider2038/knowledge-db/internal/bootstrap/config"
)

func TestValidateNodeAgent(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name    string
		cfg     config.Config
		wantErr string
	}{
		{
			name: "empty agent",
			cfg:  config.Config{},
		},
		{
			name: "cursor explicit",
			cfg: config.Config{
				NodeAgent: config.NodeAgent{Agent: "cursor"},
			},
		},
		{
			name: "cursor case insensitive",
			cfg: config.Config{
				NodeAgent: config.NodeAgent{Agent: "  CURSOR  "},
			},
		},
		{
			name: "opencode missing model",
			cfg: config.Config{
				NodeAgent: config.NodeAgent{Agent: "opencode"},
			},
			wantErr: "KB_NODE_AGENT_MODEL",
		},
		{
			name: "opencode go missing key",
			cfg: config.Config{
				NodeAgent: config.NodeAgent{
					Agent: "opencode",
					Model: "opencode-go/kimi-k3",
				},
			},
			wantErr: "OPENCODE_API_KEY",
		},
		{
			name: "opencode go with url forbidden",
			cfg: config.Config{
				NodeAgent: config.NodeAgent{
					Agent:          "opencode",
					Model:          "opencode-go/kimi-k3",
					OpenCodeAPIKey: "key",
					APIURL:         "https://api.example.com/v1",
				},
			},
			wantErr: "KB_NODE_AGENT_API_URL must not be set",
		},
		{
			name: "opencode go ok",
			cfg: config.Config{
				NodeAgent: config.NodeAgent{
					Agent:          "opencode",
					Model:          "opencode-go/kimi-k3",
					OpenCodeAPIKey: "key",
				},
			},
		},
		{
			name: "custom url missing key",
			cfg: config.Config{
				NodeAgent: config.NodeAgent{
					Agent:  "opencode",
					Model:  "gpt-4o",
					APIURL: "https://api.example.com/v1",
				},
			},
			wantErr: "KB_NODE_AGENT_API_KEY",
		},
		{
			name: "custom url ok",
			cfg: config.Config{
				NodeAgent: config.NodeAgent{
					Agent:  "opencode",
					Model:  "gpt-4o",
					APIURL: "https://api.example.com/v1",
					APIKey: "sk-test",
				},
			},
		},
		{
			name: "opencode model only ok",
			cfg: config.Config{
				NodeAgent: config.NodeAgent{
					Agent: "opencode",
					Model: "anthropic/claude",
				},
			},
		},
		{
			name: "unknown agent",
			cfg: config.Config{
				NodeAgent: config.NodeAgent{Agent: "claude-code"},
			},
			wantErr: "cursor, or opencode",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			err := tc.cfg.ValidateNodeAgent()
			if tc.wantErr == "" {
				require.NoError(t, err)

				return
			}
			require.Error(t, err)
			require.Contains(t, err.Error(), tc.wantErr)
		})
	}
}
