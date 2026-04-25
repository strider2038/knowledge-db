package googleoauth_test

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/strider2038/knowledge-db/internal/googleoauth"
)

func TestSanitizeReturnPath(t *testing.T) {
	t.Parallel()
	tests := []struct {
		in   string
		want string
	}{
		{"", "/"},
		{"/../evil", "/evil"},
		{"//evil.com", "/"},
		{"https://evil.com", "/"},
		{"/valid/path", "/valid/path"},
		{"../escape", "/"},
		{"/a/../b", "/b"},
	}
	for _, tc := range tests {
		t.Run(fmt.Sprintf("%q", tc.in), func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.want, googleoauth.SanitizeReturnPath(tc.in))
		})
	}
}
