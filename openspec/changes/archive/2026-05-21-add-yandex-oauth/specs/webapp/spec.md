## Purpose

Web UI: комбинированный вход по `auth_methods`, OAuth с иконками.

## ADDED Requirements

### Requirement: Кнопка и переход к Yandex OAuth

Если `yandex` ∈ `auth_methods`, страница входа SHALL показывать «Войти через Yandex» с иконкой Yandex (inline SVG / локальный asset), переход на `GET /api/auth/yandex` без секретов на клиенте. Если `yandex` ∉ `auth_methods`, кнопка SHALL NOT отображаться.

#### Сценарий: Показ кнопки Yandex

- **WHEN** `auth_methods` содержит `yandex`
- **THEN** кнопка с иконкой и текстом отображается

#### Сценарий: Навигация на старт Yandex OAuth

- **WHEN** пользователь нажимает кнопку Yandex
- **THEN** полный переход на `GET /api/auth/yandex` (с сохранением redirect в sessionStorage, как у Google)

### Requirement: Иконки провайдеров на OAuth-кнопках

Кнопки «Войти через Google» и «Войти через Yandex» SHALL содержать узнаваемую иконку слева от текста. Ресурсы MUST быть локальными (inline SVG), без загрузки с внешних CDN.

#### Сценарий: Визуальное различие провайдеров

- **WHEN** отображаются обе OAuth-кнопки
- **THEN** у каждой своя иконка; подписи однозначно называют провайдера

### Requirement: Комбинированная страница входа

Web UI SHALL на `/login` отображать **все** настроенные способы: OAuth-кнопки для каждого элемента `{ google, yandex }` ∩ `auth_methods`; форму логин/пароль, если `password` ∈ `auth_methods`. При пароле и хотя бы одном OAuth SHOULD быть визуальный разделитель («или»). Порядок: OAuth-кнопки (Google, затем Yandex), разделитель, форма пароля.

#### Сценарий: Пароль и два OAuth

- **WHEN** `auth_methods` = `password`, `google`, `yandex`
- **THEN** UI показывает две OAuth-кнопки с иконками, разделитель, форму пароля

#### Сценарий: Только OAuth

- **WHEN** `auth_methods` без `password`
- **THEN** только OAuth-кнопки, форма скрыта

#### Сценарий: Только пароль

- **WHEN** `auth_methods` = `password`
- **THEN** только форма, OAuth-кнопки скрыты

## MODIFIED Requirements

### Requirement: Кнопка и переход к Google OAuth

Если `google` ∈ `auth_methods`, UI SHALL показывать «Войти через Google» с иконкой Google и переходом на `GET /api/auth/google`. Условие **не** «единственный auth_mode», а наличие `google` в `auth_methods`. При наличии пароля кнопка Google SHALL отображаться **вместе** с формой.

#### Сценарий: Показ кнопки Google

- **WHEN** `auth_methods` содержит `google`
- **THEN** кнопка с иконкой Google

#### Сценарий: Навигация на старт OAuth

- **WHEN** нажатие «Войти через Google»
- **THEN** переход на `GET /api/auth/google`

### Requirement: Поведение login/logout flow

Web UI MUST определять поведение по `auth_methods` (с fallback на `auth_mode` только если `auth_methods` отсутствует — legacy). Парольный вход — при `password` ∈ `auth_methods`. OAuth — сессия через callback; после возврата на `KB_PUBLIC_WEB_BASE_URL` — `GET /api/auth/session`, redirect из sessionStorage/query. Ошибки OAuth: учитывать query `provider` (`google`|`yandex`) для текста; без `provider` — нейтральное сообщение.

#### Сценарий: Успешный парольный вход

- **WHEN** `password` ∈ `auth_methods`, верные credentials
- **THEN** `POST /api/auth/login`, редирект на `redirect` или `/`

#### Сценарий: Ошибка парольного входа

- **WHEN** неверные credentials
- **THEN** сообщение об ошибке, остаёмся на login

#### Сценарий: Успешный возврат из Google

- **WHEN** `google` ∈ `auth_methods`, callback установил `kb_session`
- **THEN** session check, редирект на сохранённый путь

#### Сценарий: Успешный возврат из Yandex

- **WHEN** `yandex` ∈ `auth_methods`, callback установил `kb_session`
- **THEN** session check, редирект на сохранённый путь

#### Сценарий: Отказ allowlist или ошибка OAuth Google

- **WHEN** `error` в query, `provider=google` или не указан при Google-only инстансе
- **THEN** понятное сообщение; не аутентифицировать

#### Сценарий: Отказ allowlist или ошибка OAuth Yandex

- **WHEN** `error` в query, `provider=yandex`
- **THEN** сообщение с упоминанием Yandex при `oauth`/`forbidden`

#### Сценарий: Выход

- **WHEN** logout
- **THEN** `POST /api/auth/logout`, переход на login

### Requirement: Потребление auth_methods в AuthContext

`AuthContext` (и `api.ts`) SHALL экспортировать `authMethods: ('password'|'google'|'yandex')[]` из session API; компоненты login MUST использовать его вместо единственного `authMode === 'google'`.

#### Сценарий: Обновление после session

- **WHEN** `getSession()` вернул `auth_methods`
- **THEN** LoginPage и guards используют актуальный набор способов
