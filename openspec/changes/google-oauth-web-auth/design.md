## Context

В knowledge-db уже есть сессионная авторизация: пара `KB_LOGIN` / `KB_PASSWORD`, cookie `kb_session`, маршруты `POST /api/auth/login`, `GET /api/auth/session`, `POST /api/auth/logout` (см. `internal/api/auth_handlers.go`, `web-session-auth`). Web UI — форма логин/пароль на `LoginPage`. Требуется спроектировать вход через Google с ограничением по списку email в окружении, не ломая существующую модель сессий после успешной аутентификации.

## Goals / Non-Goals

**Goals:**

- Authorization Code flow с Google (стандартный `accounts.google.com` + token endpoint) и получение **verified** email (claim `email` + `email_verified=true`).
- Allowlist: только адреса из `KB_AUTH_ALLOWED_EMAILS` получают серверную сессию; остальные получают отказ (без сессии, без утечки деталей в UI сверх «доступ запрещён» / общая ошибка).
- Защита callback от CSRF через криптостойкий `state` (и сверка redirect URL с заранее заданным в env).
- Предсказуемая матрица env: задокументированные переменные для OAuth, сессий и CORS.
- OpenSpec-артефакты (delta-спеки) для последующей реализации и тестов.

**Non-Goals:**

- Мульти-тенантность, роли, профили пользователей в БД.
- OpenID / другие IdP кроме Google.
- Service account или доступ к API Google от имени сервера сверх обмена code→tokens для входа.
- Смена хранения сессий (по-прежнему in-memory store в `session.Store` для сессий браузера).

## Decisions

1. **Режимы авторизации: пароль vs Google — взаимоисключающие.**  
   - **Rationale:** два способа входа на одной инсталляции усложняют продукт и тесты; личный инстанс обычно выбирает один механизм.  
   - **Rule:** при полной настройке Google-OAuth (см. env) пароль **не** используется, даже если `KB_*` заданы — **старт с ошибкой конфигурации** (лог + exit), чтобы владелец явно убрал лишние переменные.  
   - **Альтернатива рассмотрена:** «пароль, если нет Google» — отклонено из-за риска случайно оставить старый пароль активным.

2. **Список email: одна переменная `KB_AUTH_ALLOWED_EMAILS`.**  
   - **Формат:** разделитель — запятая (и опционально пробелы после запятой); сравнение **без учёта регистра**; **не** допускается `*`/регулярки в первой версии (только полные адреса).  
   - **Rationale:** простая проверка в .env, без внешних БД.  
   - **Альтернатива:** путь к файлу со списком — отложена.

3. **Callback URL: явный `KB_GOOGLE_OAUTH_REDIRECT_URL`.**  
   - **Rationale:** значение **должно 1:1** совпадать с зарегистрированным в Google Cloud Console; явный URL снижает ошибки с подстановкой базы.  
   - **Связь с существующим:** `KB_PUBLIC_WEB_BASE_URL` — для ссылок из Telegram/прочих сценариев; редирект к SPA после callback может строиться из него, но **redirect URI для Google** — только из `KB_GOOGLE_OAUTH_REDIRECT_URL`.

4. **State для OAuth:** HMAC/подпись или сопоставление state в store с `KB_OAUTH_STATE_SECRET` (секрет из env, рекомендуемая длина ≥ 32 байта в base64/hex) либо отдельный ключ.  
   - **Rationale:** стандартная защита от подделки callback.  
   - **Альтернатива:** только in-memory state map — слабее при рестартe процесса; предпочтительно stateless с подписью.

5. **Сессия после Google:** существующий `session.Store.Create` + те же cookie-атрибуты, тот же `GET /api/auth/session`.  
   - **Rationale:** без изменения middleware `auth.Middleware` и 401 на `/api/*`.

6. **UI:** кнопка «Войти через Google» ведёт на `GET /api/auth/google` (302 на Google). После успешного callback (`GET /api/auth/google/callback`) сервер редиректит в SPA на `KB_PUBLIC_WEB_BASE_URL` (и при необходимости сохраняет маршрут `redirect` в state) с установленной cookie.  
   - **Rationale:** избегаем CORS preflight к Google с фронта, секреты остаются на сервере.

## Переменные окружения (сводка)

| Переменная | Назначение |
|------------|------------|
| `KB_GOOGLE_OAUTH_CLIENT_ID` | Client ID из Google Cloud Console. |
| `KB_GOOGLE_OAUTH_CLIENT_SECRET` | Client secret. |
| `KB_GOOGLE_OAUTH_REDIRECT_URL` | Полный URL callback (как в консоли Google), например `https://host/api/auth/google/callback`. |
| `KB_AUTH_ALLOWED_EMAILS` | Список разрешённых email через запятую. Обязателен при режиме Google. |
| `KB_OAUTH_STATE_SECRET` | Секрет для подписи/шифрования `state` (и при необходимости PKCE, если введёте). |
| `KB_SESSION_TTL` | Уже существует — TTL сессии после OAuth (и для парольного режима). |
| `ALLOWED_CORS_ORIGIN` | Уже существует — origin SPA для `login`/`logout`/`session`. |
| `KB_PUBLIC_WEB_BASE_URL` | Публичный base URL веба — для финального редиректа в браузер после успешного callback. |

Парольный режим по-прежнему: `KB_LOGIN`, `KB_PASSWORD` (и не задан полный набор Google-OAuth, см. выше — при конфликте конфигурации старт неуспешен).

## Risks / Trade-offs

- **Секреты в env** → Mitigation: не логировать client secret; хранение в secret manager в проде.  
- **In-memory сессии** (как сейчас) — сброс сессий при рестарте → существующее trade-off, для OAuth не усугубляем.  
- **Только email allowlist** — смена email в Google-аккаунте меняет доступ — владелец обновляет список.  
- **Lockout при ошибочном .env** — взаимоисключение пароля и Google: Mitigation = явное сообщение в логе при невалидной комбинации.

## Migration Plan

1. Создать OAuth Client (Web) в Google Cloud, добавить `KB_GOOGLE_OAUTH_REDIRECT_URL` в «Authorized redirect URIs».  
2. Задать `KB_AUTH_ALLOWED_EMAILS` и секреты на сервере, убрать `KB_LOGIN`/`KB_PASSWORD` если переключаетесь с пароля.  
3. Развернуть новую версию бинарника и веб-статики, проверить `GET /api/auth/session` и happy-path.  
4. **Rollback:** откат релиза + возврат к парольному набору env (без Google-переменных).

## Open Questions

- Точные пути (имена) эндпоинтов OAuth зафиксированы в delta-спеке `rest-api`; при реализации придерживаться их, чтобы не расходилось с API-тестами.  
- Нужен ли `offline` / refresh у Google в будущем — **нет** в текущем scope (сессия только в cookie сервера).
