## 1. Конфигурация и валидация при старте

- [ ] 1.1 Расширить `internal/bootstrap/config` полями Google OAuth, `KB_AUTH_ALLOWED_EMAILS`, `KB_OAUTH_STATE_SECRET` и вспомогательными типами/методами: `AuthMode` (`off` / `password` / `google`), `GoogleAuthConfigured()`, `PasswordAuthConfigured()`, `ValidateAuth() error` (взаимоисключение, неполные OAuth, пустой allowlist при Google).
- [ ] 1.2 Обновить `bootstrap` для вызова `ValidateAuth` и выхода с ошибкой при невалидной конфигурации.

## 2. Google OAuth (Go, HTTP-обработчики)

- [ ] 2.1 Реализовать обмен `code` → tokens и запрос userinfo (email + `email_verified`) к Google (stdlib + `golang.org/x/oauth2` при явной необходимости).
- [ ] 2.2 `GET /api/auth/google`: генерация и валидация `state` (HMAC/подпись с `KB_OAUTH_STATE_SECRET`), редирект на endpoint авторизации Google с `redirect_uri=KB_GOOGLE_OAUTH_REDIRECT_URL` и `scope=openid email ...`.
- [ ] 2.3 `GET /api/auth/google/callback`: валидация `state`, обмен `code`, allowlist, создание `kb_session` через `session.Store`, редирект в SPA (base из `KB_PUBLIC_WEB_BASE_URL` + обработка query `error` от Google).
- [ ] 2.4 `POST /api/auth/login`: в Google-режиме — отвечать ошибкой без создания сессии; парольный режим без регрессий.
- [ ] 2.5 `GET /api/auth/session`: возвращать `auth_mode: password|google` (и `auth_enabled`) по текущему `cfg`.
- [ ] 2.6 Проверить, что `auth.Middleware` не требует сессию для `GET /api/auth/google` и `GET /api/auth/google/callback` (и остальные публичные auth-пути).
- [ ] 2.7 API-тесты: успех/отказ allowlist, невалидный `state`, Google-режим + `POST /api/auth/login` 4xx, session JSON с `auth_mode`.

## 3. Web UI

- [ ] 3.1 `AuthContext` / `api`: разбор `auth_mode` из `GET /api/auth/session` и ветвление UI.
- [ ] 3.2 `LoginPage`: кнопка «Войти через Google» (навигация на `/api/auth/google` с `credentials: 'include'` не нужна — полный `window.location` к тому же host или документированный URL API), скрытие/показ полей login/password.
- [ ] 3.3 Сохранение/восстановление `?redirect=` до OAuth (sessionStorage) и навигация после успешного `authenticated` при возврате из callback.
- [ ] 3.4 Сообщение об ошибке при `?error=` из callback (доступ запрещён и т.д.).

## 4. Документация и среда

- [ ] 4.1 Обновить корневой `.env.example` (или `web/.env.example`, если публично документируете только фронт) списком переменных из `design.md` и ссылку на OpenSpec change.
- [ ] 4.2 Запустить `openspec validate` для `google-oauth-web-auth` и устранить замечания.

## 5. Завершение (после согласования кода)

- [ ] 5.1 `openspec archive google-oauth-web-auth` (после merge-ревью и кода) и ручной merge delta в `openspec/specs/`.
