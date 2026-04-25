package kb //nolint:testpackage // pathHasHiddenSegment is internal; white-box tests

import (
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPathHasHiddenSegment(t *testing.T) {
	t.Parallel()

	cases := []struct {
		rel  string
		want bool
	}{
		{"", false},
		{".", false},
		{"a/b", false},
		{".cursor", true},
		{".agents", true},
		{"topic/.cursor", true},
		{"pub/.hidden/sub", true},
		{"a/../b", false},
	}
	for i, tc := range cases {
		t.Run(strconv.Itoa(i)+"_"+tc.rel, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.want, pathHasHiddenSegment(tc.rel), "rel=%q", tc.rel)
		})
	}
}

func TestPathHasHiddenSegment_SlashPathWithDotSegment(t *testing.T) {
	t.Parallel()
	// Input is expected after filepath.ToSlash at call sites; middle segment ".bar" is hidden.
	assert.True(t, pathHasHiddenSegment("foo/.bar/baz"))
}
