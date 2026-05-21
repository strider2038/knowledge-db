## 1. Общие OAuth-хелперы

- [x] 1.1 Вынести state, allowlist, sanitize path, redirect-to-login в `internal/oauthcommon`; поддержать query `provider` при ошибке callback
- [x] 1.2 Подключить `googleoauth` к `oauthcommon` без изменения поведения Google
- [x] 1.3 Прогнать тесты Google OAuth / state / path

## 2. Конфигурация: мульти-способ

- [x] 2.1 Добавить `KB_YANDEX_OAUTH_*`, `YandexAuthConfigured()`, `anyYandexEnvSet()`
- [x] 2.2 Реализовать `AuthMethods() []string` (порядок: `password`, `google`, `yandex`); `AuthEnabled()` := len(methods) > 0
- [x] 2.3 Переписать `ValidateAuth`: убрать взаимоисключение пароль/Google/Yandex; сохранить проверки неполных наборов по каждому способу
- [x] 2.4 Обобщить проверку `KB_PUBLIC_WEB_BASE_URL` при **любом** настроенном OAuth
- [x] 2.5 Unit-тесты config: комбинации password+google, google+yandex, все три, частичные env → ошибка

## 3. Backend: session API и handlers

- [x] 3.1 `GET /api/auth/session`: `auth_methods`; `auth_mode` = единственный способ или `"multi"`
- [x] 3.2 `Login`: разрешить при `PasswordAuthConfigured()`; убрать блокировку «use Google sign-in»
- [x] 3.3 Google handlers: проверять `GoogleAuthConfigured()`, не `AuthMode() == google`
- [x] 3.4 Реализовать `internal/yandexoauth` + `YandexOAuthStart` / `YandexOAuthCallback`; router
- [x] 3.5 API-тесты: session `auth_methods` для комбинаций; Yandex callback (mock); login при password+google; login 400 без пароля

## 4. Web UI

- [x] 4.1 `api.ts`: тип `AuthMethod`, `auth_methods` в `SessionStatus`, `startYandexOAuth`
- [x] 4.2 `AuthContext`: `authMethods`, fallback с `auth_mode` для старых ответов
- [x] 4.3 `OAuthProviderIcon` (Google, Yandex SVG)
- [x] 4.4 `LoginPage`: комбинированный layout, разделитель, иконки, ошибки по `provider`
- [x] 4.5 Обновить `LoginPage.test.tsx` и связанные тесты

## 5. Документация (обязательно, с нюансами)

- [x] 5.1 **`.env.example`**: заменить «ровно один режим» на мульти-способ; блок Yandex env; комментарии к общим `KB_OAUTH_STATE_SECRET`, `KB_AUTH_ALLOWED_EMAILS`, `KB_PUBLIC_WEB_BASE_URL`; примеры «только пароль», «Google+Yandex+пароль (dev)»
- [x] 5.2 **`README.md` — раздел «Веб-авторизация»** (переструктурировать):
  - [x] 5.2.1 Матрица: какие env включают password / google / yandex; порядок `auth_methods` в API
  - [x] 5.2.2 Общие OAuth-переменные и когда они обязательны
  - [x] 5.2.3 Google: Console, redirect `/api/auth/google/callback`, test users, CORS
  - [x] 5.2.4 Yandex: [oauth.yandex.com](https://oauth.yandex.com/), право **доступ к email**, redirect `/api/auth/yandex/callback`, отличие от Google (`default_email`, нет `email_verified`)
  - [x] 5.2.5 **Рекомендации production**: не оставлять пароль без необходимости; HTTPS; `ALLOWED_CORS_ORIGIN`; ротация `KB_OAUTH_STATE_SECRET`; минимальный allowlist
  - [x] 5.2.6 **Рекомендации dev**: пароль + OAuth на localhost; примеры redirect URI для :8080
  - [x] 5.2.7 **Безопасность**: два канала атаки при пароль+OAuth; rate limit login; OAuth только с allowlist
  - [x] 5.2.8 **API для клиентов**: `auth_methods` основной; `auth_mode` deprecated / `multi`
  - [x] 5.2.9 **Миграция** с модели «Google XOR пароль»: можно добавить пароль к существующему Google без смены OAuth env
- [x] 5.3 Проверить согласованность таблицы env в README с `.env.example`
- [x] 5.4 `openspec validate add-yandex-oauth`

## 6. Финальная проверка

- [x] 6.1 `go build ./... && go test ./... -race` (затронутые пакеты)
- [x] 6.2 `cd web && npm run build && npm run test` (при изменениях web)
- [x] 6.3 `golangci-lint run ./...` / `task lint` при наличии
