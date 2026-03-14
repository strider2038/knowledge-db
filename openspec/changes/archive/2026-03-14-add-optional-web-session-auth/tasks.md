## 1. Конфигурация и серверная auth-инфраструктура

- [x] 1.1 Добавить в конфиг `KB_LOGIN`, `KB_PASSWORD`, `KB_SESSION_TTL` (дефолт `8h`) и вычисление флага включённой авторизации.
- [x] 1.2 Реализовать in-memory session store (создание, чтение, инвалидирование, TTL-истечение) с потокобезопасным доступом.
- [x] 1.3 Реализовать auth middleware для защиты `/api/*` (включая `/api/mcp` и `/api/assets/*`) с allowlist для `/api/auth/*` и health/readiness endpoint.
- [x] 1.4 Подключить middleware в bootstrap-цепочку так, чтобы при выключенной авторизации поведение сервера оставалось обратносуместимым.

## 2. Backend API для login/logout/session

- [x] 2.1 Добавить `POST /api/auth/login` с проверкой credentials, выдачей cookie-сессии и корректными атрибутами cookie.
- [x] 2.2 Добавить `GET /api/auth/session` для возврата статуса аутентификации текущей сессии.
- [x] 2.3 Добавить `POST /api/auth/logout` для инвалидирования сессии и очистки cookie.
- [x] 2.4 Добавить базовые защитные механизмы: constant-time сравнение credentials, ограничение частоты login-попыток, проверка Origin/Referer для state-changing auth-запросов.

## 3. Web UI login flow

- [x] 3.1 Добавить страницу `/login` с формой login/password и отображением ошибок авторизации.
- [x] 3.2 Добавить session-check (`GET /api/auth/session`) и route guard для защищённых разделов приложения.
- [x] 3.3 Реализовать редирект на исходный маршрут после успешного входа и переход на login при `401` от API.
- [x] 3.4 Добавить logout в UI с вызовом `POST /api/auth/logout` и очисткой клиентского auth-состояния.

## 4. Тесты и документация

- [x] 4.1 Добавить API-тесты для auth endpoints (`login`, `session`, `logout`) и сценариев доступа к защищённым `/api/*` с/без сессии.
- [x] 4.2 Добавить тесты для `/api/mcp` и `/api/assets/*` в режиме включённой авторизации (валидная сессия / отсутствие сессии).
- [x] 4.3 Добавить frontend-тесты login flow: редирект неавторизованного пользователя, успешный вход, ошибка входа, logout.
- [x] 4.4 Обновить `README` и `.env.example`: описать `KB_LOGIN`, `KB_PASSWORD`, `KB_SESSION_TTL`, примеры запуска в open и auth режимах, а также требования к TLS при публичном доступе.
