package api

import "strings"

// NodeAgentOptions передаётся из bootstrap в раннер OpenCode (api не импортирует config).
type NodeAgentOptions struct {
	Model    string
	APIURL   string
	APIKey   string
	StateDir string
}

// ConfigureNodeAgentHandlers выбирает раннеры нормализации и agent-edit по имени агента.
func ConfigureNodeAgentHandlers(h *Handler, agent string, opts NodeAgentOptions) {
	switch strings.ToLower(strings.TrimSpace(agent)) {
	case "opencode":
		r := newOpenCodeRunner(opts)
		h.SetNodeNormalizer(r)
		h.SetNodeAgentEditor(r)
	default:
		h.SetNodeNormalizer(NewCursorNodeNormalizer())
		h.SetNodeAgentEditor(NewCursorNodeAgentEditor())
	}
}
