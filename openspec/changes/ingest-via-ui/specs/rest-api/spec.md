## Purpose

REST API для CRUD операций с узлами базы знаний, полнотекстового и ключевого поиска. В scaffold — минимальный набор эндпоинтов.

## Requirements

## MODIFIED Requirements

### Requirement: Ingestion

API MUST предоставлять эндпоинт POST /api/ingest для приёма текста и передачи в ingestion pipeline. Тело запроса MUST поддерживать поля: text (обязательно), source_url (опционально), source_author (опционально), type_hint (опционально). Допустимые значения type_hint: "auto", "article", "link", "note". При отсутствии или неизвестном значении type_hint MUST трактовать как "auto".

#### Сценарий: Отправка текста

- **WHEN** POST /api/ingest с телом { "text": "..." }
- **THEN** текст передаётся в Ingester, возвращается результат

#### Сценарий: Отправка с type_hint

- **WHEN** POST /api/ingest с телом { "text": "...", "type_hint": "article" }
- **THEN** текст и type_hint передаются в Ingester, возвращается результат
