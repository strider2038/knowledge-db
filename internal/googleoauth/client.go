package googleoauth

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/muonsoft/errors"
)

// DefaultOutboundTimeout for token and userinfo requests to Google.
const DefaultOutboundTimeout = 15 * time.Second

// Client performs Google authorization-code exchange and userinfo; state signing is separate.
// If HTTPClient is nil, ExchangeCodeForUserInfo uses &http.Client{Timeout: DefaultOutboundTimeout}.
type Client struct {
	Config     Config
	Endpoints  Endpoints
	HTTPClient *http.Client
}

// AuthorizationURL builds the Google OAuth2 authorize URL with signed state.
func (c *Client) AuthorizationURL(state string) string {
	u := c.Endpoints
	v := url.Values{}
	v.Set("client_id", c.Config.ClientID)
	v.Set("redirect_uri", c.Config.RedirectURL)
	v.Set("response_type", "code")
	v.Set("scope", "openid email profile")
	v.Set("state", state)
	v.Set("access_type", "online")

	return u.authURL() + "?" + v.Encode()
}

//nolint:tagliatelle // Google token response field names
type tokenResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	ExpiresIn   int    `json:"expires_in"`
}

//nolint:tagliatelle // Google userinfo field names
type userInfo struct {
	Email         string `json:"email"`
	EmailVerified bool   `json:"email_verified"`
}

// ExchangeCodeForUserInfo exchanges the authorization code for an access token and fetches userinfo.
func (c *Client) ExchangeCodeForUserInfo(ctx context.Context, code string) (string, bool, error) {
	u := c.Endpoints
	hc := c.HTTPClient
	if hc == nil {
		hc = &http.Client{Timeout: DefaultOutboundTimeout}
	}
	data := url.Values{}
	data.Set("code", code)
	data.Set("client_id", c.Config.ClientID)
	data.Set("client_secret", c.Config.ClientSecret)
	data.Set("redirect_uri", c.Config.RedirectURL)
	data.Set("grant_type", "authorization_code")
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, u.tokenURL(), strings.NewReader(data.Encode()))
	if err != nil {
		return "", false, errors.Errorf("token request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	res, err := hc.Do(req)
	if err != nil {
		return "", false, errors.Errorf("token http: %w", err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return "", false, errors.Errorf("token: status %d", res.StatusCode)
	}
	var tr tokenResponse
	if err := json.NewDecoder(res.Body).Decode(&tr); err != nil {
		return "", false, errors.Errorf("token json: %w", err)
	}
	if tr.AccessToken == "" {
		return "", false, errors.New("empty access token from Google")
	}
	uReq, err := http.NewRequestWithContext(ctx, http.MethodGet, u.userInfoURL(), nil)
	if err != nil {
		return "", false, errors.Errorf("userinfo request: %w", err)
	}
	uReq.Header.Set("Authorization", "Bearer "+tr.AccessToken)
	uRes, err := hc.Do(uReq)
	if err != nil {
		return "", false, errors.Errorf("userinfo http: %w", err)
	}
	defer uRes.Body.Close()
	if uRes.StatusCode != http.StatusOK {
		return "", false, errors.Errorf("userinfo: status %d", uRes.StatusCode)
	}
	var ui userInfo
	if err := json.NewDecoder(uRes.Body).Decode(&ui); err != nil {
		return "", false, errors.Errorf("userinfo json: %w", err)
	}

	return strings.TrimSpace(ui.Email), ui.EmailVerified, nil
}
