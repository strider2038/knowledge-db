package googleoauth

import (
	"net/http"
	"net/url"
)

// RedirectToLoginError sends the browser to the login page with a stable ?error= code.
func RedirectToLoginError(w http.ResponseWriter, r *http.Request, publicBase, errCode string) {
	if publicBase == "" {
		http.Redirect(w, r, "/login?error="+url.QueryEscape(errCode), http.StatusFound)

		return
	}
	dest, err := AppendQueryPath(publicBase, "/login", "error="+url.QueryEscape(errCode))
	if err != nil {
		http.Error(w, "redirect error", http.StatusInternalServerError)

		return
	}
	http.Redirect(w, r, dest, http.StatusFound)
}
