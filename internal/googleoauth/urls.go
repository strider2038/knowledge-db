package googleoauth

const (
	defaultAuthURL     = "https://accounts.google.com/o/oauth2/v2/auth"
	defaultTokenURL    = "https://oauth2.googleapis.com/token"
	defaultUserInfoURL = "https://www.googleapis.com/oauth2/v3/userinfo"
)

func (e Endpoints) authURL() string {
	if e.AuthURL != "" {
		return e.AuthURL
	}

	return defaultAuthURL
}

func (e Endpoints) tokenURL() string {
	if e.TokenURL != "" {
		return e.TokenURL
	}

	return defaultTokenURL
}

func (e Endpoints) userInfoURL() string {
	if e.UserInfoURL != "" {
		return e.UserInfoURL
	}

	return defaultUserInfoURL
}
