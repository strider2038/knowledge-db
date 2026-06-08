# Концептуальная документация

Документы в этом каталоге описывают **продуктовые и архитектурные идеи** knowledge-db — то, как система должна вести себя с точки зрения пользователя и данных, без привязки к конкретным файлам реализации.

## Назначение

- Дать общий язык для обсуждения фич (ingestion, retrieval, capture).
- Связать пользовательские сценарии с OpenSpec changes и ADR.
- Зафиксировать «почему» до или параллельно с реализацией.

## Документы

| Документ | О чём |
|----------|--------|
| [ingestion-workflows.md](ingestion-workflows.md) | Четыре режима обработки тела при ingest, оси `type` / `content_profile` / `content_mode`, каналы ввода |

## Связанные артефакты

- ADR: [0004](../adr/0004-ingestion-pipeline-llm-orchestration.md), [0010](../adr/0010-link-article-digest-for-retrieval.md), [0011](../adr/0011-ingestion-content-modes.md) (proposed)
- OpenSpec change: `openspec/changes/2026-06-08-ingestion-content-modes/`
- Ранняя концепция pipeline: `openspec/changes/archive/2026-03-07-implement-ingestion-pipeline/concept.md`

## Как добавлять

1. Новый файл `kebab-case.md` в `docs/concepts/`.
2. Ссылка в таблице выше.
3. При принятии решения — ADR в `docs/adr/` и delta spec в OpenSpec.
