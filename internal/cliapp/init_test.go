package cliapp

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/strider2038/knowledge-db/internal/cliapp/embedskill"
)

func TestEmbeddedSkillMatchesCanonicalAgentsSkill(t *testing.T) {
	t.Parallel()

	repoRoot := filepath.Join("..", "..")
	canonical := filepath.Join(repoRoot, ".agents", "skills", "knowledge-db", "SKILL.md")
	data, err := os.ReadFile(canonical)
	require.NoError(t, err)
	require.Equal(t, string(data), string(embedskill.KnowledgeDB))
}

func TestInstallKnowledgeDBSkill(t *testing.T) {
	t.Parallel()

	base := t.TempDir()
	destFile, err := installKnowledgeDBSkill(base)
	require.NoError(t, err)
	require.Equal(t, filepath.Join(base, ".agents", "skills", "knowledge-db", "SKILL.md"), destFile)

	data, err := os.ReadFile(destFile)
	require.NoError(t, err)
	body := string(data)
	require.NotContains(t, body, "{{DATA_PATH}}")
	require.Contains(t, body, base)
	require.Contains(t, body, "kb validate --path")
}

func TestInitCmd_WritesGitignoreAndSkill(t *testing.T) {
	t.Parallel()

	base := t.TempDir()
	app := New()
	err := app.Run(t.Context(), []string{"kb", "init", "--path", base})
	require.NoError(t, err)

	gitignore := filepath.Join(base, ".gitignore")
	_, err = os.Stat(gitignore)
	require.NoError(t, err)

	skillPath := filepath.Join(base, ".agents", "skills", "knowledge-db", "SKILL.md")
	data, err := os.ReadFile(skillPath)
	require.NoError(t, err)
	require.Contains(t, string(data), base)
}

func TestInitCmd_WithExample(t *testing.T) {
	t.Parallel()

	base := t.TempDir()
	app := New()
	err := app.Run(t.Context(), []string{"kb", "init", "--path", base, "--example"})
	require.NoError(t, err)

	nodePath := filepath.Join(base, "example", "topic", "sample-node.md")
	_, err = os.Stat(nodePath)
	require.NoError(t, err)
}
