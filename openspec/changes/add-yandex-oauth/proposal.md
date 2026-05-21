## Why

Часть пользователей knowledge-db предпочитает вход через Яндекс ID, а не Google. Сейчас в OAuth-режиме доступен только Google; для self-hosted инстансов в РФ/СНГ Yandex OAuth — естественная альтернатива с тем же allowlist email и cookie-сессиями. Дополнительно кнопки входа должны визуально отличаться провайдера (иконки Google и Yandex), чтобы быстрее узнавать способ входа.

## What Changes

- Новый **режим Yandex OAuth** (взаимоисключающий с паролем и Google OAuth, по аналогии с Google): переменные `KB_YANDEX_OAUTH_CLIENT_ID`, `KB_YANDEX_OAUTH_CLIENT_SECRET`, `KB_YANDEX_OAUTH_REDIRECT_URL`; общие `KB_OAUTH_STATE_SECRET`, `KB_AUTH_ALLOWED_EMAILS`, `KB_PUBLIC_WEB_BASE_URL`.
- REST: `GET /api/auth/yandex`, `GET /api/auth/yandex/callback`; `auth_mode: "yandex"` в `GET /api/auth/session`; исключения маршрутов из проверки сессии.
- Backend: пакет клиента Yandex OAuth (authorization code, userinfo `default_email`), переиспользование общих helper'ов state/allowlist/redirect из `googleoauth` (вынос в общий `oauth` или аналог).
- Web UI: кнопка «Войти через Yandex» в Yandex-режиме; **иконки** на кнопках «Войти через Google» и «Войти через Yandex» (inline SVG или локальные assets, без внешних CDN).
- Документация: `.env.example`, `README.md` — настройка приложения в [oauth.yandex.com](https://oauth.yandex.com/).
- Валидация старта: полный набор Yandex env или ни одной переменной; конфликт с паролем/Google; непустой allowlist.

## Capabilities

### New Capabilities

- _(отдельная новая capability не вводится)_

### Modified Capabilities

- `web-session-auth`: третий OAuth-режим (Yandex), условия включения, allowlist по `default_email`, взаимоисключение с Google и паролем.
- `rest-api`: маршруты Yandex OAuth, `auth_mode` включает `yandex`, исключения для `/api/auth/yandex*`.
- `webapp`: кнопка Yandex OAuth, иконки провайдеров на кнопках входа, обработка ошибок callback.

## Impact

- **Go:** `internal/bootstrap/config`, `internal/api` (handlers, router, middleware allowlist), новый `internal/yandexoauth` (или расширение общего oauth-пакета), тесты по образцу `google_oauth_test.go`.
- **web:** `LoginPage`, `api.ts`, `AuthContext`, тесты, компоненты/иконки OAuth.
- **Документация:** `README.md`, `.env.example`.
- **Внешние системы:** регистрация OAuth-приложения Yandex (redirect URI, право «доступ к email»).
