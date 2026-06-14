package ingestion

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIngestTypeAllowsSourceURLDedup(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		nodeType string
		want     bool
	}{
		{name: "article", nodeType: "article", want: true},
		{name: "link", nodeType: "link", want: true},
		{name: "note", nodeType: "note", want: false},
		{name: "empty", nodeType: "", want: false},
		{name: "whitespace", nodeType: "  ", want: false},
		{name: "article uppercase", nodeType: "Article", want: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, ingestTypeAllowsSourceURLDedup(tt.nodeType))
		})
	}
}
