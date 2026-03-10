## Purpose

Подсказка типа контента при ingestion — позволяет пользователю указать тип (article, link, note) для выбора оркестратором при создании узла. При отсутствии подсказки (auto) оркестратор определяет тип автоматически по содержимому.

## Requirements

### Requirement: Подсказка типа в IngestRequest

Система ДОЛЖНА (SHALL) поддерживать опциональное поле TypeHint в IngestRequest. Допустимые значения: пустая строка или "auto" (автоопределение), "article", "link", "note". При TypeHint = "" или "auto" оркестратор MUST определять тип по содержимому текста. При TypeHint = "article", "link" или "note" оркестратор MUST использовать указанный тип при вызове create_node.

#### Сценарий: TypeHint = auto

- **WHEN** IngestText вызывается с TypeHint = "" или "auto"
- **THEN** оркестратор определяет тип (article/link/note) по тексту и контексту

#### Сценарий: TypeHint = article

- **WHEN** IngestText вызывается с TypeHint = "article"
- **THEN** оркестратор создаёт узел с type=article, используя подсказку пользователя

#### Сценарий: TypeHint = link

- **WHEN** IngestText вызывается с TypeHint = "link"
- **THEN** оркестратор создаёт узел с type=link

#### Сценарий: TypeHint = note

- **WHEN** IngestText вызывается с TypeHint = "note"
- **THEN** оркестратор создаёт узел с type=note

### Requirement: Передача type_hint через API

API ДОЛЖЕН (SHALL) принимать опциональное поле type_hint в теле POST /api/ingest. Допустимые значения: "auto", "article", "link", "note". При отсутствии или неизвестном значении MUST трактовать как "auto".

#### Сценарий: Отправка с type_hint

- **WHEN** POST /api/ingest с телом { "text": "...", "type_hint": "article" }
- **THEN** текст и type_hint передаются в Ingester, оркестратор использует подсказку

#### Сценарий: Отправка без type_hint

- **WHEN** POST /api/ingest с телом { "text": "..." }
- **THEN** type_hint трактуется как "auto", оркестратор определяет тип автоматически
