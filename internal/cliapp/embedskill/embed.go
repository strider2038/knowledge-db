package embedskill

import _ "embed"

// KnowledgeDB is the knowledge-db agent skill template ({{DATA_PATH}} placeholder).
// Keep in sync with .agents/skills/knowledge-db/SKILL.md — see init_test.go.
//
//go:embed SKILL.md
var KnowledgeDB []byte
