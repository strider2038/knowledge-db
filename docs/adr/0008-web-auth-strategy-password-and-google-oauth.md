# ADR 0008: Auth strategy: optional session auth + Google OAuth

- Status: accepted
- Date: 2026-04-25
- Supersedes: -
- Superseded-By: -

## Context

Проект изначально ориентирован на локальное использование, но для некоторых сценариев потребовалась защита web UI с минимальным friction.

## Decision

Аутентификация остается опциональной. Поддерживаются взаимоисключающие режимы:
- password session auth (`KB_LOGIN` + `KB_PASSWORD`);
- Google OAuth web auth (с allowlist email и серверной cookie-сессией).

По умолчанию auth может быть отключен для локального single-user сценария.

## Consequences

### Плюсы

- Гибкость между локальным и более защищенным режимом.
- Без обязательной внешней IAM-инфраструктуры в базовом сценарии.
- Совместимость с embedded web UI.

### Минусы

- Больше конфигурационных веток и проверок.
- Требуется аккуратная валидация env для Google режима.

## Alternatives

- Всегда включенная auth: отклонена как избыточная для локального default usage.
- Только password mode: отклонена из-за запроса на OAuth-сценарий.

## References

- [design.md (optional auth)](/home/strider/projects/knowledge-db/openspec/changes/archive/2026-03-14-add-optional-web-session-auth/design.md)
- [proposal.md (optional auth)](/home/strider/projects/knowledge-db/openspec/changes/archive/2026-03-14-add-optional-web-session-auth/proposal.md)
- [design.md (google oauth)](/home/strider/projects/knowledge-db/openspec/changes/archive/2026-04-25-google-oauth-web-auth/design.md)
- [proposal.md (google oauth)](/home/strider/projects/knowledge-db/openspec/changes/archive/2026-04-25-google-oauth-web-auth/proposal.md)
