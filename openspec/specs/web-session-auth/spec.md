## Purpose

Опциональная авторизация для web-доступа к knowledge-db через login form и серверные cookie-сессии.

## Requirements

### Requirement: Опциональный режим сессионной авторизации

Сервер ДОЛЖЕН (SHALL) включать режим сессионной авторизации только при одновременном наличии `KB_LOGIN` и `KB_PASSWORD`. В этом режиме сервер MUST использовать cookie-сессии (`HttpOnly`, `Secure`, `SameSite`, `Path=/`) и MUST поддерживать `KB_SESSION_TTL` с значением по умолчанию `8h`.

#### Сценарий: Авторизация выключена по умолчанию

- **WHEN** `KB_LOGIN` и/или `KB_PASSWORD` не заданы
- **THEN** сервер работает в текущем открытом режиме без требования сессии

#### Сценарий: Авторизация включается по env

- **WHEN** `KB_LOGIN` и `KB_PASSWORD` заданы
- **THEN** сервер включает проверку сессии для защищённых маршрутов и применяет TTL сессии из `KB_SESSION_TTL` или `8h` по умолчанию

### Requirement: API входа и статуса сессии

В режиме авторизации сервер MUST предоставлять endpoints `POST /api/auth/login`, `POST /api/auth/logout`, `GET /api/auth/session`. Endpoint `POST /api/auth/login` SHALL создавать сессию только при валидных credentials, `POST /api/auth/logout` SHALL инвалидировать текущую сессию, `GET /api/auth/session` SHALL возвращать факт аутентификации.

#### Сценарий: Успешный вход

- **WHEN** клиент отправляет корректные login/password на `POST /api/auth/login`
- **THEN** сервер возвращает успешный ответ и устанавливает cookie сессии

#### Сценарий: Неверные credentials

- **WHEN** клиент отправляет неверные login/password на `POST /api/auth/login`
- **THEN** сервер возвращает ошибку авторизации и не устанавливает сессию

#### Сценарий: Проверка сессии

- **WHEN** клиент вызывает `GET /api/auth/session` с валидной cookie-сессией
- **THEN** сервер возвращает `authenticated=true`

#### Сценарий: Выход

- **WHEN** клиент вызывает `POST /api/auth/logout` с валидной сессией
- **THEN** сервер инвалидирует сессию и очищает cookie

### Requirement: Исключения для health/readiness

В режиме авторизации сервер ДОЛЖЕН (SHALL) допускать allowlist маршрутов health/readiness без проверки пользовательской сессии.

#### Сценарий: Доступ к health endpoint без сессии

- **WHEN** инфраструктурный клиент вызывает endpoint health/readiness без cookie-сессии
- **THEN** сервер обрабатывает запрос без требования аутентификации
