package yandexoauth

const (
	defaultAuthURL     = "https://oauth.yandex.com/authorize"
	defaultTokenURL    = "https://oauth.yandex.com/token"
	defaultUserInfoURL = "https://login.yandex.ru/info?format=json"
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
