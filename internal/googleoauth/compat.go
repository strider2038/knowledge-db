package googleoauth

import (
	"net/http"

	"github.com/strider2038/knowledge-db/internal/oauthcommon"
)

// Re-exports for callers that still import googleoauth shared helpers.

// SignState signs OAuth state (delegates to oauthcommon).
var SignState = oauthcommon.SignState

// VerifyState verifies OAuth state (delegates to oauthcommon).
var VerifyState = oauthcommon.VerifyState

// ValidateStateSecret validates the state signing secret.
var ValidateStateSecret = oauthcommon.ValidateStateSecret

// SanitizeReturnPath normalizes return paths.
var SanitizeReturnPath = oauthcommon.SanitizeReturnPath

// AppendQueryPath builds redirect URLs.
var AppendQueryPath = oauthcommon.AppendQueryPath

// ParseEmailAllowlist parses allowlisted emails.
var ParseEmailAllowlist = oauthcommon.ParseEmailAllowlist

// IsEmailAllowed checks allowlist membership.
var IsEmailAllowed = oauthcommon.IsEmailAllowed

// RedirectToLoginError redirects to login with an error (no provider; prefer oauthcommon).
func RedirectToLoginError(w http.ResponseWriter, r *http.Request, publicBase, errCode string) {
	oauthcommon.RedirectToLoginError(w, r, publicBase, errCode, "")
}
