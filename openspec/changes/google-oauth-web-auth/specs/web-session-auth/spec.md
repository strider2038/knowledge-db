## Purpose

Delta для capability `web-session-auth`: Google OAuth, allowlist email, взаимоисключающий режим с паролем.

## Requirements

## ADDED Requirements

### Requirement: Проверка allowlist email при Google OAuth

Сервер ДОЛЖЕН (SHALL) после получения подтверждённого email от Google сравнивать его с адресами из `KB_AUTH_ALLOWED_EMAILS` (список через запятую, сравнение SHALL быть без учёта регистра). Сессия MUST создаваться только если `email_verified` истинен и email входит в список.

#### Сценарий: Email разрешён

- **WHEN** Google возвращает `email_verified=true` и email присутствует в allowlist
- **THEN** сервер SHALL создать cookie-сессию и завершить OAuth flow редиректом в веб-интерфейс

#### Сценарий: Email не в allowlist

- **WHEN** `email_verified` не true или email отсутствует в allowlist
- **THEN** сервер MUST NOT выдавать сессию

### Requirement: Конфликт конфигурации парольного и Google-режима

Сервер SHALL отказываться в запуске, если обнаружена полная настройка Google OAuth и **одновременно** заданы непустые `KB_LOGIN` и `KB_PASSWORD` (ошибка в логе, ненулевой exit).

#### Сценарий: Недопустимая комбинация env

- **WHEN** настроен полный Google OAuth (включая `KB_AUTH_ALLOWED_EMAILS`) и задана пара логин/пароль
- **THEN** процесс SHALL завершаться с ошибкой конфигурации

## MODIFIED Requirements

### Requirement: Опциональный режим сессионной авторизации

Сервер ДОЛЖЕН (SHALL) включать режим сессионной авторизации, если выполняется **ровно одна** из альтернатив: (A) заданы одновременно `KB_LOGIN` и `KB_PASSWORD` и **не** задана полная конфигурация Google OAuth; (B) задана полная конфигурация Google OAuth (`KB_GOOGLE_OAUTH_CLIENT_ID`, `KB_GOOGLE_OAUTH_CLIENT_SECRET`, `KB_GOOGLE_OAUTH_REDIRECT_URL`, `KB_OAUTH_STATE_SECRET`, непустой `KB_AUTH_ALLOWED_EMAILS`) **и** оба `KB_LOGIN` и `KB_PASSWORD` **пусты**. Неполный набор Google-OAuth-переменных SHALL приводить к отказу старта. Пароль (A) и Google (B) MUST быть взаимоисключающи: при полных (A) и (B) одновременно применяется требование «Конфликт конфигурации». Во включённом режиме сервер MUST использовать cookie-сессии (`HttpOnly`, `Secure`, `SameSite`, `Path=/`) и MUST поддерживать `KB_SESSION_TTL` с значением по умолчанию `8h`.

#### Сценарий: Авторизация выключена

- **WHEN** не задана полная пара логин/пароль и не задан полный набор Google OAuth, и нет прерванного частичного ввода Google-переменных
- **THEN** сервер SHALL работать в открытом режиме без сессий

#### Сценарий: Авторизация по паролю

- **WHEN** заданы `KB_LOGIN` и `KB_PASSWORD` и нет полной конфигурации Google OAuth
- **THEN** сервер SHALL требовать сессию на защищённых маршрутах с TTL `KB_SESSION_TTL` (по умолчанию 8h)

#### Сценарий: Неполная конфигурация Google

- **WHEN** задана любая, но не вся, группа Google OAuth-переменных
- **THEN** сервер SHALL отказать в старте с явной диагностикой

#### Сценарий: Включение режима Google

- **WHEN** настроен полный Google OAuth, `KB_AUTH_ALLOWED_EMAILS` непуст, и `KB_LOGIN` пуст, и `KB_PASSWORD` пуст
- **THEN** сервер SHALL требовать сессию, вход SHALL выполняться через Google; парольный `POST /api/auth/login` SHALL NOT создавать сессию

### Requirement: API входа и статуса сессии

В **парольном** режиме сервер MUST предоставлять `POST /api/auth/login`, `POST /api/auth/logout`, `GET /api/auth/session` так же, как в базовом спеке. В **Google-режиме** сервер MUST предоставлять `GET /api/auth/google`, `GET /api/auth/google/callback`, `POST /api/auth/logout`, `GET /api/auth/session`, а `POST /api/auth/login` SHALL NOT создавать сессию. `GET /api/auth/google` SHALL инициировать OAuth, `GET /api/auth/google/callback` SHALL обменивать `code` на токены и, при допустимом email, устанавливать cookie. `GET /api/auth/session` SHALL возвращать `authenticated` и, при применимости, `auth_enabled` для web-клиента. `POST /api/auth/logout` SHALL инвалидировать сессию и очищать cookie.

#### Сценарий: Успешный парольный вход

- **WHEN** в парольном режиме клиент присылает валидные `login`/`password` на `POST /api/auth/login`
- **THEN** сервер устанавливает cookie `kb_session`

#### Сценарий: Парольный отказ

- **WHEN** в парольном режиме credentials неверны
- **THEN** сервер SHALL возвращать ошибку авторизации без сессии

#### Сценарий: Успешный OAuth callback

- **WHEN** Google-режим, callback валиден, `state` проверен, allowlist и `email_verified` допускают вход
- **THEN** сервер SHALL установить `kb_session` и выполнить редирект в веб-интерфейс

#### Сценарий: Проверка сессии

- **WHEN** клиент вызывает `GET /api/auth/session` с валидной сессионной cookie
- **THEN** ответ SHALL отражать факт аутентификации (`authenticated`)

#### Сценарий: Выход

- **WHEN** клиент вызывает `POST /api/auth/logout` с валидной сессией
- **THEN** сервер SHALL инвалидировать сессию и очищать cookie
