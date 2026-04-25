package api

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/binary"
	"strings"
	"time"

	"github.com/muonsoft/errors"
)

// oauth state version byte prefix in signed payload.
const oauthStateV1 = byte(1)

// max OAuth state age (defense in depth; Google also returns quickly).
const oauthStateMaxAge = 20 * time.Minute

// min length for KB_OAUTH_STATE_SECRET (32 bytes of entropy recommended).
const oauthStateSecretMinLen = 16

func validateOAuthStateSecret(s string) error {
	if len(s) < oauthStateSecretMinLen {
		return errors.New("KB_OAUTH_STATE_SECRET must be at least 16 bytes")
	}

	return nil
}

// signOAuthState encodes v1|unix_ms|path|16 random bytes, then appends HMAC-SHA256.
func signOAuthState(secret, returnPath string, now time.Time) (string, error) {
	if err := validateOAuthStateSecret(secret); err != nil {
		return "", err
	}
	if returnPath == "" {
		returnPath = "/"
	}
	if returnPath[0] != '/' {
		returnPath = "/" + returnPath
	}
	rb := make([]byte, 16)
	if _, err := rand.Read(rb); err != nil {
		return "", errors.Errorf("rand: %w", err)
	}
	var ts [8]byte
	binary.BigEndian.PutUint64(ts[:], uint64(now.UnixMilli()))
	payload := make([]byte, 0, 1+8+2+len(returnPath)+16)
	payload = append(payload, oauthStateV1)
	payload = append(payload, ts[:]...)
	payload = binary.BigEndian.AppendUint16(payload, uint16(len(returnPath)))
	payload = append(payload, returnPath...)
	payload = append(payload, rb...)

	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write(payload)
	tag := mac.Sum(nil)
	out := append(append([]byte{}, payload...), tag...)

	return base64.RawURLEncoding.EncodeToString(out), nil
}

// verifyOAuthState checks HMAC, age, and returns the return path.
func verifyOAuthState(secret, state string) (string, error) {
	if err := validateOAuthStateSecret(secret); err != nil {
		return "", err
	}
	if state == "" {
		return "", errors.New("missing state")
	}
	raw, err := base64.RawURLEncoding.DecodeString(state)
	if err != nil {
		return "", errors.Errorf("invalid state encoding: %w", err)
	}
	if len(raw) < 1+8+2+sha256.Size+1 { // at least 1 + ts + 2 (len) + 1 char path
		return "", errors.New("state too short")
	}
	if raw[0] != oauthStateV1 {
		return "", errors.New("unsupported state version")
	}
	if len(raw) < 1+8+2+sha256.Size {
		return "", errors.New("invalid state length")
	}
	payloadLen := len(raw) - sha256.Size
	payload := raw[:payloadLen]
	tag := raw[payloadLen:]

	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write(payload)
	expected := mac.Sum(nil)
	if !hmac.Equal(tag, expected) {
		return "", errors.New("invalid state signature")
	}

	// parse payload: v1 | ms | u16 pathlen | path | 16 random
	if len(payload) < 1+8+2 {
		return "", errors.New("corrupt state")
	}
	ms := int64(binary.BigEndian.Uint64(payload[1:9]))
	issued := time.UnixMilli(ms)
	if time.Since(issued) > oauthStateMaxAge || time.Until(issued) > oauthStateMaxAge {
		return "", errors.New("state expired")
	}
	pln := int(binary.BigEndian.Uint16(payload[9:11]))
	rest := payload[11:]
	if pln < 0 || pln > len(rest)-16 {
		return "", errors.New("corrupt path in state")
	}
	path := string(rest[:pln])
	if pln == 0 {
		path = "/"
	} else if path[0] != '/' {
		path = "/" + path
	}

	return path, nil
}

func parseEmailAllowlist(s string) map[string]struct{} {
	out := make(map[string]struct{})
	for p := range strings.SplitSeq(s, ",") {
		e := strings.TrimSpace(strings.ToLower(p))
		if e != "" {
			out[e] = struct{}{}
		}
	}

	return out
}

func isEmailAllowed(allowlist map[string]struct{}, email string) bool {
	e := strings.TrimSpace(strings.ToLower(email))
	_, ok := allowlist[e]

	return ok
}

// appendQueryPath appends a relative path and query to baseURL.
func appendQueryPath(baseURL, relPath, query string) (string, error) {
	b := strings.TrimRight(baseURL, "/")
	if b == "" {
		return "", errors.New("empty base url")
	}
	rel := relPath
	if !strings.HasPrefix(rel, "/") {
		rel = "/" + rel
	}
	if query != "" {
		rel = rel + "?" + query
	}
	if strings.HasPrefix(b, "http://") || strings.HasPrefix(b, "https://") {
		return b + rel, nil
	}

	return b + rel, nil
}
