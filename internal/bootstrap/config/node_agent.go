package config

import (
	"strings"

	"github.com/muonsoft/errors"
)

// NodeAgent — агент для нормализации и правки узлов (Cursor или OpenCode).
type NodeAgent struct {
	Agent    string `env:"KB_NODE_AGENT" envDefault:""`
	Model    string `env:"KB_NODE_AGENT_MODEL" envDefault:""`
	APIURL   string `env:"KB_NODE_AGENT_API_URL" envDefault:""`
	APIKey   string `env:"KB_NODE_AGENT_API_KEY" envDefault:""`
	StateDir string `env:"KB_NODE_AGENT_STATE_DIR" envDefault:""`
	// OpenCodeAPIKey — ключ подписки OpenCode Go (OPENCODE_API_KEY), только для валидации на старте.
	OpenCodeAPIKey string `env:"OPENCODE_API_KEY" envDefault:""`
}

// ValidateNodeAgent проверяет конфигурацию агента узла перед запуском сервера.
func (c Config) ValidateNodeAgent() error {
	agent := strings.ToLower(strings.TrimSpace(c.NodeAgent.Agent))
	switch agent {
	case "", "cursor":
		return nil
	case "opencode":
		model := strings.TrimSpace(c.NodeAgent.Model)
		if model == "" {
			return errors.New("node agent: KB_NODE_AGENT_MODEL is required when KB_NODE_AGENT=opencode")
		}
		if strings.HasPrefix(model, "opencode-go/") {
			if strings.TrimSpace(c.NodeAgent.OpenCodeAPIKey) == "" {
				return errors.New("node agent: OPENCODE_API_KEY is required for opencode-go models")
			}
			if strings.TrimSpace(c.NodeAgent.APIURL) != "" {
				return errors.New("node agent: KB_NODE_AGENT_API_URL must not be set for opencode-go models")
			}

			return nil
		}
		if strings.TrimSpace(c.NodeAgent.APIURL) != "" {
			if strings.TrimSpace(c.NodeAgent.APIKey) == "" {
				return errors.New("node agent: KB_NODE_AGENT_API_KEY is required when KB_NODE_AGENT_API_URL is set")
			}

			return nil
		}

		return nil
	default:
		return errors.New("node agent: KB_NODE_AGENT must be empty, cursor, or opencode")
	}
}
