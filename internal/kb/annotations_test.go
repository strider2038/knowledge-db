package kb_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/strider2038/knowledge-db/internal/kb"
)

func TestAnnotationsBaseNodePath_WhenTranslation_ExpectBase(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "topic/article", kb.AnnotationsBaseNodePath("topic/article.ru"))
	assert.Equal(t, "topic/article", kb.AnnotationsBaseNodePath("topic/article"))
}

func TestListNodeAnnotations_WhenNoFile_ExpectEmpty(t *testing.T) {
	t.Parallel()
	store, base := seedMemFS(map[string]string{
		"topic/node.md": `---
keywords: [a]
created: "2024-01-01T00:00:00Z"
updated: "2024-01-01T00:00:00Z"
type: note
title: Node
---

Body`,
	})
	ctx := context.Background()

	notes, err := store.ListNodeAnnotations(ctx, base, "topic/node")

	require.NoError(t, err)
	assert.Empty(t, notes)
}

func TestCreateNodeAnnotation_WhenGeneral_ExpectStored(t *testing.T) {
	t.Parallel()
	store, base := seedMemFS(map[string]string{
		"topic/node.md": `---
keywords: [a]
created: "2024-01-01T00:00:00Z"
updated: "2024-01-01T00:00:00Z"
type: note
title: Node
---

Body`,
	})
	ctx := context.Background()

	created, err := store.CreateNodeAnnotation(ctx, base, "topic/node", kb.CreateAnnotationParams{
		Body: "My note",
	})
	require.NoError(t, err)
	assert.NotEmpty(t, created.ID)
	assert.Nil(t, created.Anchor)

	notes, err := store.ListNodeAnnotations(ctx, base, "topic/node")
	require.NoError(t, err)
	require.Len(t, notes, 1)
	assert.Equal(t, "My note", notes[0].Body)
}

func TestCreateNodeAnnotation_WhenAnchored_ExpectResolved(t *testing.T) {
	t.Parallel()
	store, base := seedMemFS(map[string]string{
		"topic/node.md": `---
keywords: [a]
created: "2024-01-01T00:00:00Z"
updated: "2024-01-01T00:00:00Z"
type: note
title: Node
---

exactly-once delivery works`,
	})
	ctx := context.Background()

	created, err := store.CreateNodeAnnotation(ctx, base, "topic/node", kb.CreateAnnotationParams{
		Body: "Check this",
		Anchor: &kb.AnnotationAnchor{
			Type:        "text_quote",
			ContentPath: "topic/node",
			Exact:       "exactly-once delivery",
		},
	})
	require.NoError(t, err)
	require.NotNil(t, created.Resolved)
	assert.True(t, *created.Resolved)
}

func TestCreateNodeAnnotation_WhenAnchorMissingInBody_ExpectUnresolved(t *testing.T) {
	t.Parallel()
	store, base := seedMemFS(map[string]string{
		"topic/node.md": `---
keywords: [a]
created: "2024-01-01T00:00:00Z"
updated: "2024-01-01T00:00:00Z"
type: note
title: Node
---

Other text`,
	})
	ctx := context.Background()

	created, err := store.CreateNodeAnnotation(ctx, base, "topic/node", kb.CreateAnnotationParams{
		Body: "Orphan",
		Anchor: &kb.AnnotationAnchor{
			ContentPath: "topic/node",
			Exact:       "missing quote",
		},
	})
	require.NoError(t, err)
	require.NotNil(t, created.Resolved)
	assert.False(t, *created.Resolved)
}

func TestListNodeAnnotations_WhenTranslationPath_ExpectSameNotes(t *testing.T) {
	t.Parallel()
	store, base := seedMemFS(map[string]string{
		"topic/node.md": `---
keywords: [a]
created: "2024-01-01T00:00:00Z"
updated: "2024-01-01T00:00:00Z"
type: note
title: Node
---

Body`,
		"topic/node.ru.md": `---
keywords: [a]
created: "2024-01-01T00:00:00Z"
updated: "2024-01-01T00:00:00Z"
type: note
title: Node RU
translation_of: node
lang: ru
---

Тело`,
	})
	ctx := context.Background()
	_, err := store.CreateNodeAnnotation(ctx, base, "topic/node", kb.CreateAnnotationParams{Body: "shared"})
	require.NoError(t, err)

	notes, err := store.ListNodeAnnotations(ctx, base, "topic/node.ru")
	require.NoError(t, err)
	require.Len(t, notes, 1)
	assert.Equal(t, "shared", notes[0].Body)
}

func TestUpdateAndDeleteNodeAnnotation_ExpectCRUD(t *testing.T) {
	t.Parallel()
	store, base := seedMemFS(map[string]string{
		"topic/node.md": `---
keywords: [a]
created: "2024-01-01T00:00:00Z"
updated: "2024-01-01T00:00:00Z"
type: note
title: Node
---

Body`,
	})
	ctx := context.Background()
	created, err := store.CreateNodeAnnotation(ctx, base, "topic/node", kb.CreateAnnotationParams{Body: "v1"})
	require.NoError(t, err)

	body := "v2"
	updated, err := store.UpdateNodeAnnotation(ctx, base, "topic/node", created.ID, kb.UpdateAnnotationParams{Body: &body})
	require.NoError(t, err)
	assert.Equal(t, "v2", updated.Body)

	err = store.DeleteNodeAnnotation(ctx, base, "topic/node", created.ID)
	require.NoError(t, err)
	notes, err := store.ListNodeAnnotations(ctx, base, "topic/node")
	require.NoError(t, err)
	assert.Empty(t, notes)
}

func TestCreateNodeAnnotation_WhenTooMany_ExpectError(t *testing.T) {
	t.Parallel()
	store, base := seedMemFS(map[string]string{
		"topic/node.md": `---
keywords: [a]
created: "2024-01-01T00:00:00Z"
updated: "2024-01-01T00:00:00Z"
type: note
title: Node
---

Body`,
	})
	ctx := context.Background()
	for range 200 {
		_, err := store.CreateNodeAnnotation(ctx, base, "topic/node", kb.CreateAnnotationParams{Body: "note"})
		require.NoError(t, err)
	}
	_, err := store.CreateNodeAnnotation(ctx, base, "topic/node", kb.CreateAnnotationParams{Body: "overflow"})
	assert.ErrorIs(t, err, kb.ErrInvalidAnnotation)
}

func TestMoveNode_WhenHasAnnotations_ExpectMoved(t *testing.T) {
	t.Parallel()
	store, base := seedMemFS(map[string]string{
		"old/node.md": `---
keywords: [a]
created: "2024-01-01T00:00:00Z"
updated: "2024-01-01T00:00:00Z"
type: note
title: Node
---

Body`,
	})
	ctx := context.Background()
	_, err := store.CreateNodeAnnotation(ctx, base, "old/node", kb.CreateAnnotationParams{Body: "keep"})
	require.NoError(t, err)

	_, err = store.MoveNode(ctx, base, "old/node", "new/node")
	require.NoError(t, err)

	notes, err := store.ListNodeAnnotations(ctx, base, "new/node")
	require.NoError(t, err)
	require.Len(t, notes, 1)
}

func TestDeleteNode_WhenHasAnnotations_ExpectRemoved(t *testing.T) {
	t.Parallel()
	store, base := seedMemFS(map[string]string{
		"topic/node.md": `---
keywords: [a]
created: "2024-01-01T00:00:00Z"
updated: "2024-01-01T00:00:00Z"
type: note
title: Node
---

Body`,
	})
	ctx := context.Background()
	_, err := store.CreateNodeAnnotation(ctx, base, "topic/node", kb.CreateAnnotationParams{Body: "gone"})
	require.NoError(t, err)

	err = store.DeleteNode(ctx, base, "topic/node")
	require.NoError(t, err)

	_, err = store.ListNodeAnnotations(ctx, base, "topic/node")
	assert.ErrorIs(t, err, kb.ErrNodeNotFound)
}
