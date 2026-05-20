package sqlite

import (
	"crypto/sha256"
	"fmt"
	"strings"

	"github.com/gofrs/uuid/v5"
)

// testNodeID returns a deterministic UUID v7-like id for index tests.
func testNodeID(path string) string {
	sum := sha256.Sum256([]byte("kb-test-node-id:" + path))
	var b [16]byte
	copy(b[:], sum[:16])
	b[6] = (b[6] & 0x0f) | 0x70
	b[8] = (b[8] & 0x3f) | 0x80
	u := uuid.Must(uuid.FromBytes(b[:]))

	return strings.ToLower(u.String())
}

// TestNodeID is exported for tests in other packages.
func TestNodeID(path string) string {
	return testNodeID(path)
}

// MustTestNodeID panics on invalid path (for test setup only).
func MustTestNodeID(path string) string {
	id := testNodeID(path)
	if id == "" {
		panic(fmt.Sprintf("test node id for %q", path))
	}

	return id
}
