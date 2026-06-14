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
		{name: "link", nodeType: "link", want: true},
		{name: "article", nodeType: "article", want: false},
		{name: "note", nodeType: "note", want: false},
		{name: "empty", nodeType: "", want: false},
		{name: "link uppercase", nodeType: "Link", want: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, ingestTypeAllowsSourceURLDedup(tt.nodeType))
		})
	}
}
