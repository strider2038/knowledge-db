## Purpose

Delta для capability `webapp`: кнопка и сценарии «Войти через Google» при Google-режиме на сервере.

## Requirements

## ADDED Requirements

### Requirement: Кнопка и переход к Google OAuth

При **Google-режиме** (сервер сконфигурирован на OAuth) Web UI SHALL отображать на странице входа действие «Войти через Google», ведущее на `GET /api/auth/google` (полная навигация, тот же origin/порт, что и API, либо иной явно согласованный URL из конфигурации), без передачи секретов на клиент. При **парольном** режиме эта кнопка SHALL NOT отображаться (или SHALL быть disabled с пояснением — на усмотрение UI, по умолчанию скрыть).

#### Сценарий: Показ кнопки

- **WHEN** `GET /api/auth/session` возвращает `auth_mode: "google"` (и `auth_enabled: true`)
- **THEN** UI отображает «Войти через Google»

#### Сценарий: Навигация на start OAuth

- **WHEN** пользователь нажимает «Войти через Google»
- **THEN** UI SHALL инициировать переход к `GET /api/auth/google` (например, `location.href` или ссылка)

## MODIFIED Requirements

### Requirement: Поведение login/logout flow

Web UI MUST обрабатывать потоки входа в соответствии с режимом: в **парольном** режиме — отправка credentials на `POST /api/auth/login`, после успешного ответа — `refresh` сессии и переход на `redirect` или `/`. В **Google-режиме** сессия SHALL устанавливаться сервером в ходе редиректов OAuth; Web UI SHALL после возврата в SPA (на `KB_PUBLIC_WEB_BASE` или эквивалент) вызвать `GET /api/auth/session` и при `authenticated` перенаправить на сохранённый `redirect` или на `/`. UI MUST поддерживать выход через `POST /api/auth/logout` с последующим переходом на login.

#### Сценарий: Успешный парольный вход

- **WHEN** парольный режим и пользователь вводит корректные login/password и отправляет форму
- **THEN** UI вызывает `POST /api/auth/login` и перенаправляет на исходный или дефолтный маршрут

#### Сценарий: Ошибка парольного входа

- **WHEN** парольный режим и credentials неверны
- **THEN** UI отображает сообщение об ошибке и остаётся на login

#### Сценарий: Успешный возврат из Google

- **WHEN** Google-режим, браузер возвращается на публичный base URL веба (см. `KB_PUBLIC_WEB_BASE_URL`) после callback с выставленной `kb_session`
- **THEN** UI SHALL запросить `GET /api/auth/session`, при `authenticated` SHALL перенаправить согласно `redirect` (из query или sessionStorage до OAuth) или на `/`

#### Сценарий: Отказ allowlist / ошибка OAuth

- **WHEN** Google-режим, callback отказал (ошибка в query или пустой вход)
- **THEN** UI SHALL показать понятное сообщение, не аутентифицировать, оставить на login или с приглашением повторить

#### Сценарий: Выход

- **WHEN** пользователь инициирует logout
- **THEN** UI вызывает `POST /api/auth/logout` и переводит на страницу входа
