package kb_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/strider2038/knowledge-db/internal/kb"
)

func TestSanitizePathSegment_WhenCyrillicAndEscaped_ExpectASCII(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "vyzov-c-funktsiy-iz-go-bez-cgo", kb.SanitizePathSegment("v%D1%8B%D0%B7%D0%BE%D0%B2-c-%D1%84%D1%83%D0%BD%D0%BA%D1%86%D0%B8%D0%B9-%D0%B8%D0%B7-go-%D0%B1%D0%B5%D0%B7-cgo"))
}

func TestSanitizeNodePath_WhenMixedSymbols_ExpectStrictASCIISegments(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "programming/golang/vyzov-c-funktsiy", kb.SanitizeNodePath("programming/golang/вызов__c функций"))
}
