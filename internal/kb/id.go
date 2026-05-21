package kb

import (
	"strings"

	"github.com/gofrs/uuid/v5"
	"github.com/muonsoft/errors"

	"github.com/strider2038/knowledge-db/internal/pkg/urlutil"
)

// NewNodeID generates a new UUID v7 in canonical lowercase form.
func NewNodeID() (string, error) {
	id, err := uuid.NewV7()
	if err != nil {
		return "", errors.Errorf("generate node id: %w", err)
	}

	return strings.ToLower(id.String()), nil
}

// ValidateNodeID reports whether s is a valid UUID string.
func ValidateNodeID(s string) bool {
	_, err := uuid.FromString(strings.TrimSpace(s))

	return err == nil
}

// NodeIDFromMetadata returns the id field from frontmatter metadata.
func NodeIDFromMetadata(meta map[string]any) string {
	if meta == nil {
		return ""
	}
	if s, ok := meta["id"].(string); ok {
		return strings.TrimSpace(s)
	}

	return ""
}

// EnsureNodeID sets frontmatter["id"] to a new UUID v7 when missing or invalid.
func EnsureNodeID(frontmatter map[string]any) error {
	if frontmatter == nil {
		return errors.New("frontmatter required")
	}
	if id := NodeIDFromMetadata(frontmatter); id != "" {
		if !ValidateNodeID(id) {
			return errors.Errorf("invalid node id: %q", id)
		}
		frontmatter["id"] = strings.ToLower(id)

		return nil
	}
	id, err := NewNodeID()
	if err != nil {
		return err
	}
	frontmatter["id"] = id

	return nil
}

// NormalizeSourceURLForDedup normalizes source_url for ingestion/index dedup lookup.
func NormalizeSourceURLForDedup(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}

	return strings.ToLower(urlutil.StripTrackingParamsFromURL(raw))
}
