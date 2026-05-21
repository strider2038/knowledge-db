## Context

Сейчас `config.Auth.AuthMode()` возвращает ровно одно из `off` | `password` | `google` с приоритетом Google над паролем. `ValidateAuth` запрещает одновременные пароль и Google. `Login` отклоняет пароль в Google-режиме. `LoginPage` показывает **либо** OAuth **либо** форму. Middleware уже exempt'ит весь `/api/auth/*`.

Цель change: **независимые способы входа**, активные по полноте env; одна cookie-сессия `kb_session` для всех.

## Goals / Non-Goals

**Goals:**

- Yandex OAuth (authorization code) + сохранение Google.
- `auth_methods` в session API; UI рендерит только настроенные способы.
- Пароль совместим с любым набором OAuth.
- Google + Yandex одновременно на одном инстансе.
- Иконки Google/Yandex (inline SVG).
- Документация с матрицей конфигураций, security-рекомендациями и пошаговой настройкой провайдеров.

**Non-Goals:**

- Разные allowlist на провайдера (один `KB_AUTH_ALLOWED_EMAILS` на все OAuth).
- OAuth для Telegram/MCP.
- PKCE (можно позже).
- Вход без allowlist для OAuth.
- Отдельный `KB_YANDEX_OAUTH_SCOPE` в v1 (права — в кабинете oauth.yandex.com).

## Decisions

### 1. Независимые «capabilities» вместо единого `AuthMode`

**Выбор:** методы `PasswordAuthConfigured()`, `GoogleAuthConfigured()`, `YandexAuthConfigured()`; `AuthEnabled() :=` любой из них; `AuthMethods() []string` для API.

**`auth_mode` (deprecated):** если в `auth_methods` ровно один элемент — дублировать его в `auth_mode` для старых клиентов; иначе `auth_mode: "multi"` или опустить — в tasks зафиксировать `"multi"` при ≥2 способах.

**Почему:** явный контракт для UI; не ломать полностью существующие тесты с одним способом.

### 2. Валидация env

| Правило | Действие |
|--------|----------|
| Частичный пароль (только login или только password) | Ошибка старта |
| Любая **Google-специфичная** env (`KB_GOOGLE_OAUTH_*`) без полного Google | Ошибка старта |
| Любая **Yandex-специфичная** env (`KB_YANDEX_OAUTH_*`) без полного Yandex | Ошибка старта |
| Полный Google или Yandex без `KB_OAUTH_STATE_SECRET` / пустого `KB_AUTH_ALLOWED_EMAILS` | Ошибка старта |
| `KB_OAUTH_STATE_SECRET` или `KB_AUTH_ALLOWED_EMAILS` заданы, но **ни один** OAuth-провайдер не полный | Ошибка старта (висящие общие OAuth-переменные) |
| Любой полный OAuth без `KB_PUBLIC_WEB_BASE_URL` | Ошибка старта |
| Пароль + Google + Yandex все полные | **Разрешено** |

**Порядок `auth_methods` в API (зафиксировать в коде и README):** `password`, `google`, `yandex` — в этом порядке, только включённые способы.

Убрать ошибки «password and Google mutually exclusive».

### 3. Общий `internal/oauthcommon`

State HMAC, allowlist, sanitize return path, redirect to login with `?error=&provider=`. Пакеты `googleoauth`, `yandexoauth` — только provider-specific URLs и userinfo.

**Yandex userinfo:** `default_email`; нет `email_verified` — сессия только при непустом email в allowlist.

**Google:** без изменений (`email_verified`).

### 4. OAuth handlers

- `GET /api/auth/google` — 404/400 «not configured», если Google не настроен (не «wrong mode»).
- Аналогично Yandex.
- Callback: query `error`, опционально `provider=google|yandex` для UI.

### 5. Login page layout

Вертикальный стек: OAuth-кнопки (Google, затем Yandex — порядок фиксированный), разделитель «или», форма пароля. Если только OAuth — без разделителя. Если только пароль — только форма.

### 6. Документация (структура README / .env.example)

Обязательные подразделы:

1. **Матрица способов** — что включится при каких env.
2. **Общие OAuth-переменные** — state secret, allowlist, public web URL.
3. **Google** — Console, redirect URI, test users.
4. **Yandex** — oauth.yandex.com, право «доступ к email», redirect `/api/auth/yandex/callback`.
5. **Рекомендации production** — не держать пароль на публичном инстансе; ротация секретов; HTTPS; `ALLOWED_CORS_ORIGIN`.
6. **Рекомендации dev** — пароль + OAuth на localhost; примеры redirect для Google/Yandex.
7. **Безопасность** — два канала атаки при пароль+OAuth; rate limit на login; allowlist обязателен для OAuth.
8. **Миграция** с старой модели «только google» — убрать проверки пустых login/password при добавлении пароля.

## Risks / Trade-offs

| Риск | Митигация |
|------|-----------|
| Пароль оставлен в production рядом с OAuth | Документация: явно «уберите KB_LOGIN/KB_PASSWORD, если не нужен» |
| Устаревшие клиенты смотрят только `auth_mode` | `auth_mode` + `auth_methods`; в README breaking note |
| Путаница в сообщениях об ошибках OAuth | `provider` в query; разные строки в `OAUTH_ERR` на web |
| Рефакторинг `googleoauth` | Тесты Google без изменения поведения |
| Yandex без email в токене | Отказ `forbidden`; в docs — включить право email в приложении |

## Migration Plan

1. Деплой новой версии; существующие инстансы «только Google» или «только пароль» работают без изменения env.
2. Добавление Yandex: задать `KB_YANDEX_*`, не трогая Google.
3. Добавление пароля к OAuth: задать `KB_LOGIN`/`KB_PASSWORD` — раньше было запрещено, теперь разрешено.
4. Rollback: откат бинарника; env совместим назад, если не использовали новые комбинации.

## Open Questions

- _(нет открытых — `auth_mode: "multi"` при ≥2 способах зафиксировано в tasks §3.1 и спеках.)_
