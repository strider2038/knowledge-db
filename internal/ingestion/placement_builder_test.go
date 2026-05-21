package ingestion_test

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	kbindex "github.com/strider2038/knowledge-db/internal/index"
	"github.com/strider2038/knowledge-db/internal/index/sqlite"
	"github.com/strider2038/knowledge-db/internal/ingestion"
	"github.com/strider2038/knowledge-db/internal/ingestion/llm"
	"github.com/strider2038/knowledge-db/internal/kb"
)

func TestPlacementBuilder_Build_WhenOverlappingAgenticCodingThemes_ExpectSkillsThemeFirst(t *testing.T) {
	t.Parallel()

	// Arrange
	ctx := context.Background()
	store, basePath := seedPlacementBase(t, map[string]string{
		"ai/agentic-coding/overview.md": `---
title: Agentic Coding
keywords: [agentic coding, Claude Code]
created: "2026-01-01T00:00:00Z"
updated: "2026-01-01T00:00:00Z"
annotation: "Материалы про агентное программирование и LLM-инструменты."
---
Claude Code agents and coding workflows.
`,
		"ai/agentic-coding/skills/openclaw-skills.md": `---
title: OpenClaw Skills
keywords: [agent skills, Claude Code, skill]
created: "2026-01-01T00:00:00Z"
updated: "2026-01-01T00:00:00Z"
annotation: "Каталог skills для Claude Code и agent tools."
---
Reusable agent skills and tool instructions for coding agents.
`,
		"programming/ai/local-llm.md": `---
title: Local LLM
keywords: [LLM, programming]
created: "2026-01-01T00:00:00Z"
updated: "2026-01-01T00:00:00Z"
annotation: "Заметка про локальные модели для программирования."
---
Programming with AI models.
`,
	})
	builder := ingestion.NewPlacementBuilder(store, basePath, nil)

	// Act
	context, _, err := builder.Build(ctx, ingestion.PlacementBuildInput{
		Text:           "Сохрани материал про Claude Code agent skills и репозиторные tool instructions",
		SourceKind:     "repository",
		ContentProfile: "repository_profile",
		Type:           "link",
	})

	// Assert
	require.NoError(t, err)
	require.NotEmpty(t, context.CandidateThemes)
	assert.Equal(t, "ai/agentic-coding/skills", context.CandidateThemes[0].Path)
	assert.Contains(t, context.CandidateThemes[0].Reasons, "similar_node")
	assert.NotEmpty(t, context.SimilarNodes)
}

func TestPlacementBuilder_Build_WhenKeywordCandidatesOverlap_ExpectFrequencyAndThemeSignals(t *testing.T) {
	t.Parallel()

	// Arrange
	ctx := context.Background()
	store, basePath := seedPlacementBase(t, map[string]string{
		"go/concurrency/goroutines.md": `---
title: Goroutines
keywords: [goroutines, Go]
created: "2026-01-01T00:00:00Z"
updated: "2026-01-01T00:00:00Z"
annotation: "Горутины и конкурентность в Go."
---
Goroutines, channels and leaks.
`,
		"go/concurrency/leaks.md": `---
title: Goroutine Leaks
keywords: [goroutines, leak, Golang]
created: "2026-01-01T00:00:00Z"
updated: "2026-01-01T00:00:00Z"
annotation: "Диагностика утечек горутин."
---
Finding goroutine leaks.
`,
		"programming/ai/agents.md": `---
title: AI Agents
keywords: [AI, ИИ]
created: "2026-01-01T00:00:00Z"
updated: "2026-01-01T00:00:00Z"
annotation: "AI agents in programming."
---
`,
	})
	builder := ingestion.NewPlacementBuilder(store, basePath, nil)

	// Act
	context, _, err := builder.Build(ctx, ingestion.PlacementBuildInput{
		Text: "Заметка про goroutine leaks и scheduler в Go",
	})

	// Assert
	require.NoError(t, err)
	require.NotEmpty(t, context.CandidateKeywords)
	assert.Equal(t, "goroutines", context.CandidateKeywords[0].Keyword)
	assert.GreaterOrEqual(t, context.CandidateKeywords[0].Frequency, 2)
	assert.Contains(t, placementKeywordNames(context.CandidateKeywords), "Go")
	assert.Contains(t, placementKeywordNames(context.CandidateKeywords), "Golang")
}

func TestPlacementBuilder_Build_WhenIndexAvailable_ExpectIndexSignals(t *testing.T) {
	t.Parallel()

	// Arrange
	ctx := context.Background()
	store, basePath := seedPlacementBase(t, map[string]string{
		"ai/agentic-coding/skills/openclaw-skills.md": `---
title: OpenClaw Skills
keywords: [agent skills, Claude Code]
created: "2026-01-01T00:00:00Z"
updated: "2026-01-01T00:00:00Z"
annotation: "Skills for Claude Code agents."
---
`,
	})
	indexStore, err := sqlite.NewStore(context.Background(), ":memory:")
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, indexStore.Close()) })
	embID, err := indexStore.InsertEmbedding(ctx, []float32{0.1}, "test")
	require.NoError(t, err)
	require.NoError(t, indexStore.UpsertNode(ctx, sqlite.TestNodeID("ai/agentic-coding/skills/openclaw-skills"), "ai/agentic-coding/skills/openclaw-skills", "hash", "body", embID))
	require.NoError(t, indexStore.UpsertNodeSearch(ctx, kbindex.NodeSearchDocument{
		NodeID:     sqlite.TestNodeID("ai/agentic-coding/skills/openclaw-skills"),
		Path:       "ai/agentic-coding/skills/openclaw-skills",
		Title:      "OpenClaw Skills",
		Type:       "link",
		Annotation: "Skills for Claude Code agents.",
		Keywords:   []string{"agent skills", "Claude Code"},
		Body:       "Repository of Claude Code agent skills.",
	}))
	builder := ingestion.NewPlacementBuilder(store, basePath, indexStore)

	// Act
	context, _, err := builder.Build(ctx, ingestion.PlacementBuildInput{
		Text: "Claude Code agent skills repository",
	})

	// Assert
	require.NoError(t, err)
	assert.Equal(t, "index", context.Source)
	require.NotEmpty(t, context.SimilarNodes)
	assert.Contains(t, context.SimilarNodes[0].MatchReasons, "index")
	assert.Contains(t, placementKeywordNames(context.CandidateKeywords), "agent skills")
}

func seedPlacementBase(tb testing.TB, files map[string]string) (*kb.Store, string) {
	tb.Helper()

	fs := afero.NewMemMapFs()
	basePath := "/data"
	for name, content := range files {
		fullPath := filepath.Join(basePath, name)
		require.NoError(tb, fs.MkdirAll(filepath.Dir(fullPath), 0o755))
		require.NoError(tb, afero.WriteFile(fs, fullPath, []byte(content), 0o644))
	}

	return kb.NewStore(fs), basePath
}

func placementKeywordNames(keywords []llm.KeywordCandidate) []string {
	result := make([]string, 0, len(keywords))
	for _, keyword := range keywords {
		result = append(result, keyword.Keyword)
	}

	return result
}
