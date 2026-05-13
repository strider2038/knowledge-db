# ADR 0009: Web UI build/embed pipeline (Vite -> embedded static)

- Status: accepted
- Date: 2026-03-29
- Supersedes: -
- Superseded-By: -

## Context

Нужно было сохранить проектный layout Go и одновременно встроить web UI в `kb-server` без отдельного runtime frontend-сервиса.

## Decision

Web UI собирается Vite-пайплайном в `web/dist`, после чего статика копируется в `internal/ui/static` и встраивается через `go:embed`. PWA-артефакты (manifest/SW) включаются в тот же delivery-процесс.

## Consequences

### Плюсы

- Один серверный бинарник для API+UI.
- Контролируемый и повторяемый build pipeline.
- Совместимость с локальным/offline-friendly deployment.

### Минусы

- Сборка состоит из нескольких стадий (Node + Go).
- Требуется контроль корректной доставки статических артефактов.

## Alternatives

- Держать `main.go`/embed в корне: отклонено как нарушение выбранного project layout.
- Отдельный frontend runtime: отклонен для базового сценария локального сервера.

## References

- [design.md (scaffold)](/home/strider/projects/knowledge-db/openspec/changes/archive/2026-03-06-initial-project-scaffold/design.md)
- [proposal.md (scaffold)](/home/strider/projects/knowledge-db/openspec/changes/archive/2026-03-06-initial-project-scaffold/proposal.md)
- [design.md (pwa)](/home/strider/projects/knowledge-db/openspec/changes/archive/2026-03-29-pwa-friendly/design.md)
- [proposal.md (pwa)](/home/strider/projects/knowledge-db/openspec/changes/archive/2026-03-29-pwa-friendly/proposal.md)
