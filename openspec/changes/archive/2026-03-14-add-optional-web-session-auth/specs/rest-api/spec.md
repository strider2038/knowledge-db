## Purpose

Уточнить поведение REST API при включённой опциональной сессионной авторизации.

## Requirements

## ADDED Requirements

### Requirement: Опциональная защита API сессией

При включённой авторизации (`KB_LOGIN` и `KB_PASSWORD` заданы) REST API ДОЛЖЕН (SHALL) требовать валидную сессионную cookie для доступа к защищённым endpoint ` /api/* `, включая `/api/assets/*`. При отсутствии или невалидности сессии сервер MUST возвращать `401 Unauthorized`.

#### Scenario: Доступ к API без сессии при включённой авторизации

- **WHEN** клиент вызывает защищённый endpoint `/api/*` без валидной cookie-сессии
- **THEN** сервер возвращает `401 Unauthorized`

#### Scenario: Доступ к API с валидной сессией

- **WHEN** клиент вызывает защищённый endpoint `/api/*` с валидной cookie-сессией
- **THEN** сервер обрабатывает запрос по обычной бизнес-логике endpoint

### Requirement: Auth endpoints для login/logout/session

REST API MUST предоставлять endpoints `POST /api/auth/login`, `POST /api/auth/logout`, `GET /api/auth/session` для работы web-клиента с сессионной авторизацией.

#### Scenario: Login endpoint выдаёт сессию

- **WHEN** клиент отправляет корректные credentials на `POST /api/auth/login`
- **THEN** сервер создаёт сессию и устанавливает cookie в ответе

#### Scenario: Session endpoint отражает текущий статус

- **WHEN** клиент вызывает `GET /api/auth/session`
- **THEN** сервер возвращает статус аутентификации текущей сессии

#### Scenario: Logout endpoint завершает сессию

- **WHEN** клиент вызывает `POST /api/auth/logout`
- **THEN** сервер инвалидирует текущую сессию и очищает auth-cookie
