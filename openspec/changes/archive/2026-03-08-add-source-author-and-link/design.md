# Design: Сохранение автора и ссылки на источник

## Context

Текущее состояние:
- **Telegram-бот**: передаёт в `IngestText` только текст. Поле `forward_origin` в сообщении не парсится — ссылка и автор пересланного поста теряются.
- **Ingestion pipeline**: `ProcessInput` содержит только `Text`, `ExistingThemes`, `ExistingKeywords`. Tool `create_node` не имеет параметра `source_author`. `FetchResult` уже содержит `Author`, но он не передаётся в `create_node` и не сохраняется в frontmatter.
- **knowledge-storage**: frontmatter поддерживает `source_url`, `source_date`, но не `source_author`.

Ограничения: интерфейс `Ingester` используется ботом и API; изменения должны сохранять обратную совместимость для вызовов без метаданных источника.

## Goals / Non-Goals

**Goals:**
- При пересылке поста из Telegram сохранять ссылку на оригинал и автора в узле.
- При сохранении статей (Habr и др.) сохранять автора из FetchResult в frontmatter.
- Добавить опциональное поле `source_author` в frontmatter узла.

**Non-Goals:**
- Извлечение автора из произвольного текста (только из явных источников: forward_origin, FetchResult).
- Поддержка других платформ (только Telegram и веб-статьи).

## Decisions

### 1. Расширение интерфейса Ingester

**Решение**: Ввести структуру `IngestRequest` с полями `Text`, `SourceURL`, `SourceAuthor` (все опциональные, кроме Text). Метод `IngestText(ctx, text string)` заменить на `IngestText(ctx, req IngestRequest)`.

**Альтернативы**:
- Добавить метаданные в текст (префикс «Метаданные источника: …») — отвергнуто: хрупко, возможны коллизии с пользовательским текстом.
- Functional options `IngestText(ctx, text, WithSource(url, author))` — отвергнуто: усложняет сигнатуру, менее явно.

**Миграция**: Все вызовы `IngestText(ctx, text)` заменяются на `IngestText(ctx, IngestRequest{Text: text})`. StubIngester и mock-и в тестах обновляются.

### 2. Парсинг forward_origin в Telegram

**Решение**: Добавить функцию `parseForwardOrigin(raw json.RawMessage) (sourceURL, sourceAuthor string)`, парсящую типы `channel`, `user`, `chat`, `hidden_user`:

| Тип | Ссылка | Автор |
|-----|--------|-------|
| channel | `https://t.me/c/{chatId}/{msgId}` или `https://t.me/{username}/{msgId}` | `chat.Title` или `@username`, fallback `author_signature` |
| user | — (нет публичной ссылки) | `@username` или `first_name last_name` |
| hidden_user | — | `sender_user_name` |
| chat | — | `sender_chat.Title` или `@username` |

Для каналов: `chat_id` в формате `-100xxxxxxxxxx` → в ссылке использовать `c/xxxxxxxxxx` (убрать префикс `-100`).

**Альтернатива**: Не строить ссылку для user/hidden_user — принято: Telegram API не даёт публичной ссылки на сообщение из лички.

### 3. Передача метаданных в ProcessInput и LLM

**Решение**: Добавить в `ProcessInput` опциональные поля `SourceURL`, `SourceAuthor`. Оркестратор при наличии этих полей добавляет в начало пользовательского текста блок:
```
Метаданные источника: ссылка: {url}, автор: {author}

{исходный текст}
```
LLM получает явный контекст и должен включить `source_url`, `source_author` в вызов `create_node`.

**Альтернатива**: Передавать метаданные отдельно в system prompt — отвергнуто: контекст в user message надёжнее для LLM.

### 4. Подстановка Author из FetchResult при create_node

**Решение**: При слиянии результата с кешем `fetch_url_content` (когда `result.SourceURL` совпадает с URL в кеше) дополнять `result.SourceAuthor` значением `cached.Author`, если LLM не вернул author. Аналогично для `SourceDate` — использовать `cached.SourceDate` при отсутствии.

Текущая логика уже подставляет `Content` и `Title` из кеша; расширяем её на `SourceAuthor` и `SourceDate`.

### 5. IngestURL и источник

**Решение**: При вызове `IngestURL(ctx, url)` передавать в `buildProcessInput` `sourceURL=url` и `sourceAuthor=fetchResult.Author` (если fetch успешен). Это обеспечит сохранение автора статей (Habr и др.) без изменения сигнатуры `IngestURL`.

### 6. Reply на пересланное сообщение

**Решение**: При объединении комментария и пересланного (reply на forward) метаданные источника берутся из `reply_to_message.forward_origin`, а не из основного сообщения. Текст формируется как раньше; `parseForwardOrigin` вызывается для `ReplyToMessage`.

## Risks / Trade-offs

| Риск | Митигация |
|------|-----------|
| Приватный канал без username — ссылка может быть нерабочей | Использовать формат `t.me/c/{id}/{msgId}`; для супергрупп/каналов без username ссылка может требовать прав доступа |
| LLM не всегда передаёт source_author в create_node | Подстановка из кеша FetchResult; для Telegram — явный блок в тексте повышает вероятность |
| Изменение интерфейса Ingester — breaking change | Внутренний API; обновить бот, pipeline, API handler, тесты за один коммит |

## Migration Plan

1. Добавить `IngestRequest`, обновить интерфейс `Ingester`.
2. Обновить `PipelineIngester.IngestText` и `IngestURL`, `buildProcessInput`.
3. Добавить `parseForwardOrigin` в `internal/telegram`, обновить `handleUpdate` и вызовы `processIngest`.
4. Добавить `source_author` в tool `create_node`, `ProcessResult`, `saveNode`.
5. Обновить StubIngester, API handler, все тесты.
6. Обновить delta-спеки (knowledge-storage, telegram-bot, ingestion-pipeline).

Откат: revert коммита; существующие узлы без `source_author` остаются валидными (поле опционально).

## Open Questions

- Нужно ли для `MessageOriginUser` пытаться строить ссылку (например, `t.me/user/msg_id`)? Telegram API не возвращает message_id оригинала в этом случае — оставляем без ссылки.
- Отображать ли `source_author` в web UI при просмотре узла? Вне скоупа данного change.
