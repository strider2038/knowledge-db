package oauthcommon

import (
	"net/http"
	"net/url"
)

// LoginErrorQuery builds the login page query for OAuth errors (optional provider).
func LoginErrorQuery(errCode, provider string) string {
	q := url.Values{}
	q.Set("error", errCode)
	if provider != "" {
		q.Set("provider", provider)
	}

	return q.Encode()
}

// RedirectToLoginError sends the browser to the login page with ?error= and optional ?provider=.
func RedirectToLoginError(w http.ResponseWriter, r *http.Request, publicBase, errCode, provider string) {
	query := LoginErrorQuery(errCode, provider)
	if publicBase == "" {
		http.Redirect(w, r, "/login?"+query, http.StatusFound)

		return
	}
	dest, err := AppendQueryPath(publicBase, "/login", query)
	if err != nil {
		http.Error(w, "redirect error", http.StatusInternalServerError)

		return
	}
	http.Redirect(w, r, dest, http.StatusFound)
}
