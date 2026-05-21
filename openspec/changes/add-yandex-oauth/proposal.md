## Why

Пользователям knowledge-db нужен вход через Yandex ID наряду с Google, а на одном инстансе — гибкая комбинация способов: пароль, Google, Yandex — **только те, что реально настроены в env**. Сейчас пароль и Google взаимоисключающи, `auth_mode` один на весь сервер, на `/login` нельзя показать форму и OAuth вместе. Это мешает типичным сценариям: OAuth в production + пароль для локальной отладки; Google и Yandex для разных пользователей одной семьи/команды с общим allowlist.

Дополнительно кнопки OAuth должны иметь иконки провайдеров (узнаваемость, offline-first без CDN).

## What Changes

- **Мульти-провайдерная авторизация:** независимые флаги «настроен пароль / Google / Yandex»; `auth_enabled` при любом включённом способе; снятие запрета «пароль XOR Google».
- **API сессии:** `GET /api/auth/session` возвращает `auth_methods: string[]` (`password`, `google`, `yandex`); поле `auth_mode` **deprecated** (один способ → то же имя; несколько → `"multi"` — см. design).
- **Yandex OAuth:** `KB_YANDEX_OAUTH_*`, маршруты `GET /api/auth/yandex`, `GET /api/auth/yandex/callback`; allowlist по `default_email`.
- **Google OAuth:** поведение без изменений; handlers проверяют `GoogleAuthConfigured()`, не «режим google».
- **Пароль:** `POST /api/auth/login` доступен, если заданы `KB_LOGIN` и `KB_PASSWORD`, **независимо** от OAuth.
- **Web UI:** `/login` — форма пароля (если настроен) + кнопки Google/Yandex (если настроены), с иконками; ошибки OAuth с учётом провайдера (`?error=oauth&provider=yandex`).
- **Конфиг:** частичный набор env **по провайдеру** → отказ старта; при любом OAuth — обязательны `KB_OAUTH_STATE_SECRET`, непустой `KB_AUTH_ALLOWED_EMAILS`, `KB_PUBLIC_WEB_BASE_URL`.
- **Документация (явная задача):** `README.md`, `.env.example`, при необходимости `docs/` — матрица комбинаций env, рекомендации по безопасности, настройка oauth.yandex.com и Google Console, CORS, production vs dev.

## Capabilities

### New Capabilities

- _(отдельная новая capability не вводится)_

### Modified Capabilities

- `web-session-auth`: мульти-способ входа, Yandex OAuth, общие OAuth env, валидация без взаимоисключения пароль/OAuth.
- `rest-api`: Yandex endpoints, `auth_methods`, login при включённом пароле рядом с OAuth, ошибки callback с `provider`.
- `webapp`: комбинированный login UI, иконки провайдеров, потребление `auth_methods`.

## Impact

- **Go:** `config.Auth` (`AuthMethods()`, флаги провайдеров; `AuthMode()` — legacy/совместимость), `ValidateAuth`, `auth_handlers`, `google_oauth`/`yandex_oauth`, `oauthcommon`.
- **web:** `api.ts`, `AuthContext`, `LoginPage`, иконки, тесты.
- **Документация:** `README.md`, `.env.example` (раздел «Веб-авторизация» с нюансами).
- **Breaking (мягкий):** клиенты, завязанные только на `auth_mode`, должны перейти на `auth_methods`; `auth_mode` остаётся для обратной совместимости.
