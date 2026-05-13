## Purpose

Pipeline добавления записей в базу: текст или URL → создание узла с файлами. LLM-оркестратор определяет тип контента и действия через function calling.

## Requirements

## MODIFIED Requirements

### Requirement: LLM-оркестратор с function calling

Система ДОЛЖНА (SHALL) использовать LLM (OpenAI-совместимый API, библиотека github.com/openai/openai-go, Responses API) для оркестрации ingestion. LLM MUST получать на вход текст пользователя и компактный контекст размещения (`placement context`), подготовленный pipeline перед первым LLM-запросом. Placement context MUST включать краткую карту базы, ранжированные candidate themes, candidate keywords и similar nodes; система MUST NOT отправлять в prompt полный неранжированный список всех keywords как основной механизм выбора.

Placement context MUST строиться локально и offline-first: при доступном локальном поисковом индексе система SHALL использовать его для подбора похожих узлов и словаря терминов; если индекс недоступен, система MUST использовать fallback-обход файлов базы и frontmatter. Candidate themes MUST учитывать похожие узлы, совпадение title/annotation/keywords/path, `source_kind`, `content_profile` и плотность темы. Candidate keywords MUST учитывать термины входного материала, keywords похожих узлов, top keywords candidate themes и частотность. Система MUST NOT вводить ручной словарь синонимов keywords в рамках placement builder.

Доступные инструменты (tools) для LLM:
- `fetch_url_content(url)` — полное извлечение контента из URL (для статей); LLM получает превью первых 2000 символов для анализа метаданных, включая Author; полный контент кешируется и сохраняется автоматически
- `fetch_url_meta(url)` — извлечение метаданных для ссылок-закладок (type=link). Ответ MUST включать поля `title`, `description`, `source` (источник метаданных: например `html_meta`, `github_api`, `content_fallback`) и опционально `content_preview` — усечённое превью текста страницы (до ~2000 символов), если мета-информация признана недостаточной и выполнен fallback через ContentFetcher. Подробности — в требовании «Метаданные для ссылок».
- `search_placement_candidates(query, source_kind, content_profile, type)` — уточняющий поиск candidate themes, candidate keywords и similar nodes по локальной базе. LLM MAY вызывать этот tool, если первичный placement context недостаточен или есть сомнение между несколькими ветками.
- `create_node(keywords, annotation, theme_path, slug, type, title, source_url, source_date, source_author, content)` — создание узла; title — обязателен, при отсутствии в источнике (заметка, пересланное сообщение) LLM MUST сгенерировать заголовок на основе контента; annotation — 2–5 предложений; source_author — опционально; для articles поле content ДОЛЖНО быть пустым — полный контент подставляется из кеша fetch_url_content; при слиянии с кешем Author из FetchResult MUST подставляться в source_author, если LLM не вернул значение; для type=link аннотация MUST опираться на факты из результата `fetch_url_meta` (включая `content_preview` при наличии) и MUST избегать шаблонных формулировок без подтверждения в источнике

#### Сценарий: LLM выбирает fetch_url_content

- **WHEN** LLM определяет, что пользователь хочет сохранить статью по URL
- **THEN** LLM вызывает fetch_url_content, получает превью контента для анализа метаданных; при вызове create_node поле content остаётся пустым, а полный контент подставляется из кеша автоматически; Author из FetchResult подставляется в source_author

#### Сценарий: LLM выбирает fetch_url_meta

- **WHEN** LLM определяет, что URL — ссылка на сервис/ресурс (не статья для копирования)
- **THEN** LLM вызывает fetch_url_meta для получения метаданных (и при необходимости content_preview), формирует аннотацию на русском по фактам из ответа инструмента

#### Сценарий: LLM создаёт заметку без URL

- **WHEN** текст не содержит URL
- **THEN** LLM формирует create_node с type=note, контентом из текста и сгенерированными метаданными

#### Сценарий: LLM использует candidate keywords

- **WHEN** placement context содержит candidate keyword "goroutines" и новый контент связан с горутинами
- **THEN** LLM MUST предпочесть candidate keyword "goroutines", а не создавать синоним, если термин подходит по смыслу

#### Сценарий: LLM использует candidate themes

- **WHEN** placement context содержит candidate theme "go/concurrency" и контент связан с конкурентностью в Go
- **THEN** LLM MUST предпочитать candidate theme, если она подходит по смыслу

#### Сценарий: Первичный placement context достаточен

- **WHEN** pipeline подготовил candidate themes и candidate keywords, которые явно покрывают входной материал
- **THEN** LLM MAY сразу вызвать create_node без дополнительного поиска placement candidates

#### Сценарий: LLM уточняет placement candidates

- **WHEN** первичный placement context не покрывает входной материал или содержит несколько близких конфликтующих веток
- **THEN** LLM MAY вызвать search_placement_candidates и использовать ответ tool при выборе theme_path и keywords

#### Сценарий: Индекс недоступен

- **WHEN** локальный поисковый индекс отсутствует или недоступен при подготовке placement context
- **THEN** система MUST построить candidate themes, candidate keywords и similar nodes через fallback-обход файлов базы

#### Сценарий: Диагностика placement context

- **WHEN** ingestion pipeline запускает LLM-оркестрацию
- **THEN** система SHALL логировать количество candidate themes, candidate keywords, similar nodes и факт использования индекса или fallback-обхода
