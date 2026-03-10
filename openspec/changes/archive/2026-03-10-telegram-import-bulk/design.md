## Context

Пользователь экспортирует чат «Избранное» из Telegram в JSON (формат одного чата — корень файла является Chat с messages[]). Требуется: загрузка JSON, пошаговая обработка (принять/отклонить), индикатор прогресса, сохранение сессии на backend для восстановления после перезагрузки.

Текущий flow: POST /api/ingest принимает text, source_url, source_author, type_hint и передаёт в Ingester. Web UI (AddPage) — textarea + type hint, отправка через ingestText (без source_url/source_author).

## Goals / Non-Goals

**Goals:**

- Парсинг Telegram export JSON (формат одного чата), извлечение text, source_author, source_url.
- Backend-сессии: хранить в `{KB_UPLOADS_DIR}/telegram-import-sessions/{session_id}.json`.
- API: создание сессии, получение состояния, accept/reject с вызовом ingestion.
- UI: табы «Вручную» | «Импорт из Telegram», загрузка JSON, пошаговая обработка, прогресс.

**Non-Goals:**

- Поддержка полного экспорта (chats.list) — только один чат.
- Объединение reply+forward — каждое сообщение обрабатывается отдельно.
- Проверка дубликатов в базе.
- Обработка медиа-файлов (фото, видео) — только caption.

## Decisions

### 1. Хранение сессий

**Решение:** `{KB_UPLOADS_DIR}/telegram-import-sessions/{session_id}.json`. Путь из KB_UPLOADS_DIR.

**Альтернативы:** data/.import-sessions (смешивание с базой), SQLite (избыточно для простых JSON-файлов).

### 2. Структура файла сессии

```json
{
  "session_id": "uuid",
  "created_at": "ISO8601",
  "total": 150,
  "current_index": 42,
  "processed_ids": [15184, 15183],
  "rejected_ids": [15182],
  "items": [
    {
      "id": 15184,
      "date_unixtime": "1557861184",
      "text": "полный текст для ingest",
      "source_author": "Sergey P",
      "source_url": "https://..."
    }
  ]
}
```

`items` — полные данные для ingest. Сортировка: `date_unixtime` desc (новые первыми).

### 3. API endpoints

| Метод | Путь | Описание |
|-------|------|----------|
| POST | /api/import/telegram | Body: JSON (chat export). Создать сессию, вернуть session_id, total, current_item. |
| GET | /api/import/telegram/session/:id | Получить состояние: current_index, processed/rejected counts, current_item. |
| POST | /api/import/telegram/session/:id/accept | Body: { type_hint? }. Ingest текущей записи, advance. |
| POST | /api/import/telegram/session/:id/reject | Отклонить текущую, advance. |

### 4. Парсинг text из Message

`text` может быть string или массив `[{type, text}, ...]`. Маппинг в Markdown:

- `plain` → текст как есть
- `link` → `[url](url)` или url
- `text_link` (если есть) → `[text](url)`
- остальные → `entity.text`

`source_author`: `forwarded_from` \|\| `saved_from` \|\| `from`.

`source_url`: первая link/text_link из text_entities; fallback — из text.

### 5. Фильтрация сообщений

Включать: `type === "message"` и есть извлекаемый текст (text или caption). Пропускать: service messages, пустые.

### 6. Конфигурация KB_UPLOADS_DIR

При отсутствии KB_UPLOADS_DIR — endpoints import возвращают 503 или 400 с сообщением «import not configured».

## Risks / Trade-offs

- **[Risk]** Очень большой JSON: загрузка в память целиком. → Mitigation: ограничить размер (например, 10 MB) или использовать streaming-парсер при необходимости.
- **[Risk]** Очистка старых сессий: файлы накапливаются. → Non-Goal для v1; можно добавить TTL или ручную очистку позже.
- **[Risk]** Один и тот же JSON загружен дважды — две сессии. → Не критично; пользователь сам управляет.

## Migration Plan

Нет миграции. Новый функционал. KB_UPLOADS_DIR — новая переменная; при отсутствии импорт недоступен.
