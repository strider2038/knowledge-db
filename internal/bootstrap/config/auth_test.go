package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAuthMethods_PasswordAndGoogle(t *testing.T) {
	t.Parallel()
	a := Auth{
		Login:              "u",
		Password:           "p",
		GoogleClientID:     "g",
		GoogleClientSecret: "s",
		GoogleRedirectURL:    "http://localhost/cb",
		OAuthStateSecret:     "0123456789abcdef",
		AuthAllowedEmails:    "a@b.com",
	}
	require.NoError(t, a.ValidateAuth())
	assert.Equal(t, []string{"password", "google"}, a.AuthMethods())
	assert.Equal(t, AuthModeMulti, a.AuthMode())
}

func TestAuthMethods_GoogleAndYandex(t *testing.T) {
	t.Parallel()
	a := Auth{
		GoogleClientID:       "g",
		GoogleClientSecret:   "gs",
		GoogleRedirectURL:    "http://localhost/gcb",
		YandexClientID:       "y",
		YandexClientSecret:   "ys",
		YandexRedirectURL:    "http://localhost/ycb",
		OAuthStateSecret:     "0123456789abcdef",
		AuthAllowedEmails:    "a@b.com",
	}
	require.NoError(t, a.ValidateAuth())
	assert.Equal(t, []string{"google", "yandex"}, a.AuthMethods())
	assert.Equal(t, AuthModeMulti, a.AuthMode())
}

func TestAuthMethods_AllThree(t *testing.T) {
	t.Parallel()
	a := Auth{
		Login:                "u",
		Password:             "p",
		GoogleClientID:       "g",
		GoogleClientSecret:   "gs",
		GoogleRedirectURL:    "http://localhost/gcb",
		YandexClientID:       "y",
		YandexClientSecret:   "ys",
		YandexRedirectURL:    "http://localhost/ycb",
		OAuthStateSecret:     "0123456789abcdef",
		AuthAllowedEmails:    "a@b.com",
	}
	require.NoError(t, a.ValidateAuth())
	assert.Equal(t, []string{"password", "google", "yandex"}, a.AuthMethods())
}

func TestValidateAuth_PartialGoogle_ExpectError(t *testing.T) {
	t.Parallel()
	a := Auth{GoogleClientID: "only-id"}
	require.EqualError(t, a.ValidateAuth(), "auth: incomplete Google OAuth env — set KB_GOOGLE_OAUTH_CLIENT_ID, KB_GOOGLE_OAUTH_CLIENT_SECRET, KB_GOOGLE_OAUTH_REDIRECT_URL, KB_OAUTH_STATE_SECRET, and non-empty KB_AUTH_ALLOWED_EMAILS, or clear all Google-specific variables")
}

func TestValidateAuth_PartialYandex_ExpectError(t *testing.T) {
	t.Parallel()
	a := Auth{YandexClientSecret: "only-secret"}
	require.EqualError(t, a.ValidateAuth(), "auth: incomplete Yandex OAuth env — set KB_YANDEX_OAUTH_CLIENT_ID, KB_YANDEX_OAUTH_CLIENT_SECRET, KB_YANDEX_OAUTH_REDIRECT_URL, KB_OAUTH_STATE_SECRET, and non-empty KB_AUTH_ALLOWED_EMAILS, or clear all Yandex-specific variables")
}

func TestValidateAuth_OAuthCommonWithoutProvider_ExpectError(t *testing.T) {
	t.Parallel()
	a := Auth{OAuthStateSecret: "0123456789abcdef"}
	require.EqualError(t, a.ValidateAuth(), "auth: KB_OAUTH_STATE_SECRET and/or KB_AUTH_ALLOWED_EMAILS set but no full OAuth provider (Google or Yandex) configured")
}

func TestValidateWebPublicBaseForOAuth_WhenOAuthWithoutBase_ExpectError(t *testing.T) {
	t.Parallel()
	a := Auth{
		GoogleClientID:     "g",
		GoogleClientSecret: "s",
		GoogleRedirectURL:  "http://localhost/cb",
		OAuthStateSecret:   "0123456789abcdef",
		AuthAllowedEmails:  "a@b.com",
	}
	require.EqualError(t, a.ValidateWebPublicBaseForOAuth(""), "auth: KB_PUBLIC_WEB_BASE_URL is required when OAuth is enabled (post-login redirect)")
}
