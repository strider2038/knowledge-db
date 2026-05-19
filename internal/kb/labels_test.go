package kb_test

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/strider2038/knowledge-db/internal/kb"
)

func TestNormalizeLabels_WhenDedupeCaseInsensitive_ExpectFirstSpelling(t *testing.T) {
	t.Parallel()

	out, err := kb.NormalizeLabels([]string{"Favorite", "favorite", "review"})
	require.NoError(t, err)
	assert.Equal(t, []string{"Favorite", "review"}, out)
}

func TestNormalizeLabels_WhenCommaInLabel_ExpectError(t *testing.T) {
	t.Parallel()

	_, err := kb.NormalizeLabels([]string{"a,b"})
	require.Error(t, err)
	assert.ErrorIs(t, err, kb.ErrInvalidLabels)
}

func TestNormalizeLabels_WhenTooMany_ExpectError(t *testing.T) {
	t.Parallel()

	in := make([]string, kb.MaxLabelsPerNode+1)
	for i := range in {
		in[i] = fmt.Sprintf("tag-%d", i)
	}
	_, err := kb.NormalizeLabels(in)
	require.Error(t, err)
	assert.ErrorIs(t, err, kb.ErrInvalidLabels)
}

func TestNodeHasAllLabels_WhenAND_ExpectMatch(t *testing.T) {
	t.Parallel()

	meta := map[string]any{"labels": []string{"Favorite", "review"}}
	assert.True(t, kb.NodeHasAllLabels(meta, []string{"favorite"}))
	assert.True(t, kb.NodeHasAllLabels(meta, []string{"favorite", "review"}))
	assert.False(t, kb.NodeHasAllLabels(meta, []string{"favorite", "hotspot"}))
}
