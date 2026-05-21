## Purpose

Опциональная мульти-способная авторизация web: пароль, Google OAuth, Yandex OAuth — независимо по конфигурации env; cookie-сессии.

## ADDED Requirements

### Requirement: Yandex OAuth как независимый способ входа

Сервер SHALL поддерживать Yandex OAuth 2.0 (authorization code), если задан **полный** набор: `KB_YANDEX_OAUTH_CLIENT_ID`, `KB_YANDEX_OAUTH_CLIENT_SECRET`, `KB_YANDEX_OAUTH_REDIRECT_URL`, а также общие для OAuth `KB_OAUTH_STATE_SECRET` и непустой `KB_AUTH_ALLOWED_EMAILS`. Наличие Yandex OAuth MUST NOT запрещать одновременную настройку Google OAuth и/или пароля.

#### Сценарий: Только Yandex OAuth

- **WHEN** настроен полный Yandex OAuth, пароль и Google не настроены
- **THEN** сервер SHALL требовать сессию на защищённых маршрутах; вход через Yandex; `auth_methods` содержит `yandex`

#### Сценарий: Yandex и Google вместе

- **WHEN** настроены полные Google и Yandex OAuth (общие state/allowlist)
- **THEN** сервер SHALL принимать вход через оба провайдера; `auth_methods` содержит `google` и `yandex`

### Requirement: Проверка allowlist email при Yandex OAuth

Сервер ДОЛЖЕН (SHALL) после обмена `code` получить userinfo Yandex ID и сравнить `default_email` с `KB_AUTH_ALLOWED_EMAILS` (без учёта регистра). Сессия MUST создаваться только при непустом `default_email`, входящем в список.

#### Сценарий: Разрешённый email Yandex

- **WHEN** callback Yandex успешен, `default_email` в allowlist
- **THEN** сервер SHALL установить `kb_session` и редирект в веб-интерфейс

#### Сценарий: Email отсутствует или не в allowlist

- **WHEN** `default_email` пуст или не в allowlist
- **THEN** сервер MUST NOT выдавать сессию; редирект на login с `error=forbidden` (и `provider=yandex` где применимо)

### Requirement: Общие OAuth-переменные при любом OAuth-провайдере

Если настроен **хотя бы один** полный OAuth-провайдер (Google и/или Yandex), сервер SHALL требовать `KB_OAUTH_STATE_SECRET`, непустой `KB_AUTH_ALLOWED_EMAILS` и непустой `KB_PUBLIC_WEB_BASE_URL` (редирект после callback в SPA).

#### Сценарий: OAuth без public web URL

- **WHEN** настроен полный Google или Yandex OAuth, `KB_PUBLIC_WEB_BASE_URL` пуст
- **THEN** сервер SHALL отказать в старте с явной диагностикой

## MODIFIED Requirements

### Requirement: Опциональный режим сессионной авторизации

Сервер ДОЛЖЕН (SHALL) включать сессионную авторизацию (`auth_enabled`), если настроен **хотя бы один** способ: (A) полная пара `KB_LOGIN` + `KB_PASSWORD`; (B) полный Google OAuth; (C) полный Yandex OAuth. Способы (A), (B), (C) MUST быть **независимыми** и MAY сочетаться произвольно. Неполный набор переменных **внутри** одного способа (пароль или конкретный OAuth-провайдер) SHALL приводить к отказу старта. Если ни один способ не настроен и нет «оборванных» частичных env OAuth — режим `off` без сессий. Cookie-сессии: `HttpOnly`, `Secure`, `SameSite`, `Path=/`; `KB_SESSION_TTL` по умолчанию `8h`.

#### Сценарий: Авторизация выключена

- **WHEN** не настроен ни пароль, ни полный Google, ни полный Yandex, нет частичных OAuth-переменных
- **THEN** сервер SHALL работать без требования сессии (`auth_enabled: false`)

#### Сценарий: Только пароль

- **WHEN** заданы `KB_LOGIN` и `KB_PASSWORD`, OAuth не настроен
- **THEN** сервер SHALL требовать сессию; `auth_methods` содержит `password`

#### Сценарий: Пароль и OAuth вместе

- **WHEN** заданы полный пароль и хотя бы один полный OAuth-провайдер
- **THEN** сервер SHALL требовать сессию; `auth_methods` содержит `password` и соответствующие OAuth-провайдеры; `POST /api/auth/login` SHALL создавать сессию при верных credentials

#### Сценарий: Неполная конфигурация Google

- **WHEN** задана любая, но не вся, группа **Google-специфичных** переменных (`KB_GOOGLE_OAUTH_*`)
- **THEN** сервер SHALL отказать в старте

#### Сценарий: Общие OAuth-переменные без провайдера

- **WHEN** заданы `KB_OAUTH_STATE_SECRET` и/или `KB_AUTH_ALLOWED_EMAILS`, но ни Google, ни Yandex не настроены полностью
- **THEN** сервер SHALL отказать в старте

#### Сценарий: Неполная конфигурация Yandex

- **WHEN** задана любая, но не вся, группа Yandex-OAuth-переменных
- **THEN** сервер SHALL отказать в старте

#### Сценарий: Неполный пароль

- **WHEN** задан только `KB_LOGIN` или только `KB_PASSWORD`
- **THEN** сервер SHALL отказать в старте

### Requirement: Проверка allowlist email при Google OAuth

Сервер ДОЛЖЕН (SHALL) после получения подтверждённого email от Google сравнивать его с `KB_AUTH_ALLOWED_EMAILS` (без учёта регистра). Сессия MUST создаваться только если `email_verified` истинен и email в списке. Allowlist **общий** для Google и Yandex OAuth.

#### Сценарий: Разрешённый email

- **WHEN** Google возвращает `email_verified=true` и email в allowlist
- **THEN** сервер SHALL создать cookie-сессию и редирект в веб-интерфейс

#### Сценарий: Email не в allowlist

- **WHEN** `email_verified` не true или email вне allowlist
- **THEN** сервер MUST NOT выдавать сессию

### Requirement: API входа и статуса сессии

Включённые способы MUST отражаться в `GET /api/auth/session`: `auth_enabled: true`, массив `auth_methods` ⊆ `{ password, google, yandex }`. Поле `auth_mode` MAY присутствовать для обратной совместимости: при ровно одном способе — его имя; при нескольких — `multi`. Пароль: `POST /api/auth/login` при `password` ∈ `auth_methods`. Google: `GET /api/auth/google`, `GET /api/auth/google/callback` при `google` ∈ `auth_methods`. Yandex: `GET /api/auth/yandex`, `GET /api/auth/yandex/callback` при `yandex` ∈ `auth_methods`. `POST /api/auth/logout` — всегда при `auth_enabled`. OAuth start/callback при ненастроенном провайдере SHALL NOT создавать сессию (ошибка конфигурации/404).

#### Сценарий: Успешный парольный вход

- **WHEN** пароль настроен, credentials верны
- **THEN** сервер устанавливает `kb_session`

#### Сценарий: Парольный отказ

- **WHEN** пароль настроен, credentials неверны
- **THEN** ошибка без сессии

#### Сценарий: Успешный OAuth callback Google

- **WHEN** Google настроен, callback валиден, allowlist и `email_verified` OK
- **THEN** `kb_session` и редирект в SPA

#### Сценарий: Успешный OAuth callback Yandex

- **WHEN** Yandex настроен, callback валиден, `default_email` в allowlist
- **THEN** `kb_session` и редирект в SPA

#### Сценарий: Проверка сессии

- **WHEN** валидная cookie
- **THEN** `authenticated: true`

#### Сценарий: Выход

- **WHEN** `POST /api/auth/logout`
- **THEN** инвалидация сессии и очистка cookie

### Requirement: Исключения для health/readiness

В режиме авторизации сервер ДОЛЖЕН (SHALL) допускать allowlist маршрутов health/readiness без проверки пользовательской сессии.

#### Сценарий: Доступ к health endpoint без сессии

- **WHEN** инфраструктурный клиент вызывает endpoint health/readiness без cookie-сессии
- **THEN** сервер обрабатывает запрос без требования аутентификации

## REMOVED Requirements

### Requirement: Конфликт конфигурации парольного и Google-режима

**Reason:** Пароль и OAuth-провайдеры больше не взаимоисключающие; включение определяется независимой полнотой env каждого способа.

**Migration:** Убрать из деплоя ожидание ошибки старта при `KB_LOGIN`+`KB_PASSWORD` вместе с Google/Yandex. Для production без пароля — не задавать `KB_LOGIN`/`KB_PASSWORD`. Ранее пустые login/password при Google остаются валидными.

#### Сценарий: Ранее недопустимая комбинация env теперь разрешена

- **WHEN** настроены полный Google OAuth и пара `KB_LOGIN`/`KB_PASSWORD`
- **THEN** сервер SHALL успешно стартовать и выставить `auth_methods`, содержащий `password` и `google`
