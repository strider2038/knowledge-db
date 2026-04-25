package googleoauth

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/binary"
	"time"

	"github.com/muonsoft/errors"
)

// OAuth state version byte prefix in signed payload.
const stateV1 = byte(1)

// max OAuth state age (defense in depth; provider usually returns quickly).
const stateMaxAge = 20 * time.Minute

// Minimum length for state signing secret.
const minStateSecretLen = 16

// ValidateStateSecret enforces a minimum key length.
func ValidateStateSecret(s string) error {
	if len(s) < minStateSecretLen {
		return errors.New("state secret must be at least 16 bytes")
	}

	return nil
}

// SignState encodes v1|unix_ms|path|16 random bytes, then appends HMAC-SHA256.
func SignState(secret, returnPath string, now time.Time) (string, error) {
	if err := ValidateStateSecret(secret); err != nil {
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
	payload = append(payload, stateV1)
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

// VerifyState checks HMAC and age, returns the return path embedded in state.
func VerifyState(secret, state string) (string, error) {
	if err := ValidateStateSecret(secret); err != nil {
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
	if raw[0] != stateV1 {
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
	if time.Since(issued) > stateMaxAge || time.Until(issued) > stateMaxAge {
		return "", errors.New("state expired")
	}
	pln := int(binary.BigEndian.Uint16(payload[9:11]))
	rest := payload[11:]
	if pln < 0 || pln > len(rest)-16 {
		return "", errors.New("corrupt path in state")
	}
	p := string(rest[:pln])
	if pln == 0 {
		p = "/"
	} else if p[0] != '/' {
		p = "/" + p
	}

	return p, nil
}
