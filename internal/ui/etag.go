package ui

import (
	"fmt"
	"net/http"
	"strings"
)

// StaticETag возвращает ETag для файла в embedded static (корневой путь вроде "assets/x.js" или "sw.js"),
// либо пустую строку, если ETag задавать не нужно.
func StaticETag(relativePath string) string {
	if BuildID == "" {
		return ""
	}
	// RFC 7232: допустимые символы; путь в embed — без " и управляющих.
	relpath := strings.Trim(relativePath, "/")

	return fmt.Sprintf(`"%s-%s"`, BuildID, relpath)
}

// SetStaticETagIfSet adds an ETag when BuildID is set (If-None-Match after deploy).
func SetStaticETagIfSet(w http.ResponseWriter, relpath string) {
	if etag := StaticETag(relpath); etag != "" {
		w.Header().Set("Etag", etag)
	}
}
