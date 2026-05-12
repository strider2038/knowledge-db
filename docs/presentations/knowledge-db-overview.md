---
title: Knowledge DB
subtitle: Персональная база знаний с AI-агентами
author: Igor Lazarev
theme: beige
revealjs-url: https://cdn.jsdelivr.net/npm/reveal.js@5/
---

# Knowledge DB

## Персональная база знаний с AI-агентами

<br/>

**Принципы:** оффлайн-first · git-first · LLM-optional

<br/>

<div style="font-size: 0.7em; color: #666;">
Igor Lazarev · 2026
</div>

---

## Концепция

```mermaid
graph LR
    subgraph Запись — онлайн
        WEB[Web UI]
        TG[Telegram]
        API[REST API]
    end
    subgraph Хранение — git
        KB[(База знаний<br/>Markdown файлы)]
    end
    subgraph Чтение — оффлайн
        LOCAL[Локальный клон<br/>доступен без интернета]
    end
    WEB --> KB
    TG --> KB
    API --> KB
    KB --> LOCAL
```

> Git как источник правды: версионирование, diff, merge

---

## Архитектура системы

```mermaid
graph TB
    subgraph Клиенты
        C1[Web]
        C2[Telegram]
        C3[API клиенты]
    end
    subgraph Сервер
        API_GW[API Gateway]
        INGEST[Ingestion<br/>+ LLM]
        CHAT[RAG Chat]
        SEARCH[Поиск]
    end
    subgraph AI-сервисы
        LLM[LLM]
        EMB[Embeddings]
        FETCH[Content Fetcher]
    end
    subgraph Хранилище
        FS[(Файлы<br/>+ Git)]
        IDX[(Векторный<br/>индекс)]
    end
    C1 & C2 & C3 --> API_GW
    API_GW --> INGEST & CHAT & SEARCH
    INGEST --> LLM & FETCH & FS
    CHAT --> SEARCH & LLM
    SEARCH --> IDX
    IDX --> EMB
```

---

## Где используется LLM

<br/>

| Компонент | Роль LLM |
|-----------|----------|
| **Ingestion** | Классификация, аннотирование, выбор темы, ключевые слова |
| **RAG Chat** | Ответы на основе контента базы знаний |
| **Query Rewrite** | Оптимизация поискового запроса |
| **Перевод** | Автоматический перевод статей |
| **Git** | Генерация осмысленных commit-сообщений |

<br/>

> Принцип: LLM опционален — без него система работает с ограничениями

---

## Ingestion Pipeline

```mermaid
flowchart LR
    INPUT[Ввод<br/>текст / URL] --> LLM[LLM с инструментами]
    LLM --> |вызов инструмента| FETCH[Загрузка<br/>контента]
    LLM --> |вызов инструмента| META[Метаданные<br/>URL]
    LLM --> |результат| NODE[Структурированный<br/>узел БЗ]
    NODE --> SAVE[Сохранение<br/>+ Git commit]
    SAVE --> |опционально| TR[Перевод]
```

---

## Ingestion — LLM с Function Calling

```mermaid
sequenceDiagram
    participant P as Pipeline
    participant LLM as LLM
    participant Tools as Инструменты

    P->>LLM: Контент + контекст БЗ
    LLM->>Tools: Загрузить URL
    Tools-->>LLM: Текст страницы
    LLM->>Tools: Получить метаданные
    Tools-->>LLM: Заголовок, описание
    LLM-->>P: Структурированный результат
    
    Note over P,LLM: LLM сам решает,<br/>какие инструменты вызвать
    Note over P,LLM: Контент инжектится напрямую —<br/>не через LLM, чтобы избежать потерь
```

---

## RAG Chat

```mermaid
flowchart TD
    Q[Вопрос пользователя] --> RW[LLM переписывает<br/>запрос для поиска]
    RW --> SEARCH[Гибридный поиск]
    SEARCH --> KW[Ключевой<br/>поиск]
    SEARCH --> VEC[Векторный<br/>поиск]
    KW & VEC --> FUSE[Слияние<br/>результатов]
    FUSE --> CTX[Контекст<br/>для LLM]
    CTX --> LLM[LLM генерирует<br/>ответ]
    LLM --> A[Ответ + источники]
```

---

## Три режима чата

<br/>

| Режим | Когда | Поведение |
|-------|-------|-----------|
| **Memory** | Управление чатом | LLM без обращения к БЗ |
| **RAG** | Поиск в базе | Только контекст из БЗ |
| **Hybrid** | По умолчанию | БЗ + знания LLM |

<br/>

**Query Rewrite** — LLM переписывает запрос пользователя в поисковые термины с учётом словаря базы знаний.

---

## Векторный индекс

```mermaid
flowchart TD
    FILES[Markdown файлы] --> SYNC[Синхронизация<br/>при изменениях]
    SYNC --> EMB[Embedding API<br/>→ векторы]
    SYNC --> FTS[Полнотекстовый<br/>индекс]
    EMB --> DB[(SQLite<br/>векторы + чанки)]
    FTS --> DB
```

<br/>

**Чанкинг:** статьи разбиваются по заголовкам — каждый фрагмент получает свой вектор для точного поиска.

---

## Гибридный поиск

```mermaid
flowchart LR
    Q[Запрос] --> KW[Ключевой<br/>поиск]
    Q --> VEC[Векторный<br/>поиск]
    KW --> RRF[Слияние<br/>ранжирований]
    VEC --> RRF
    RRF --> R[Результаты<br/>с оценкой]
```

<br/>

**Reciprocal Rank Fusion** — объединяет два подхода к поиску:
- **Ключевой** — точные совпадения слов, быстрый
- **Векторный** — семантическое сходство, находит по смыслу

Ключевой поиск имеет повышенный вес — он точнее для известных терминов.

---

## Telegram Bot

```mermaid
flowchart TD
    MSG[Сообщение] --> BUF[Буфер<br/>ожидание пары]
    FWD[Пересылка] --> BUF
    BUF --> |комментарий + пересылка| INGEST[Ingestion<br/>через LLM]
    INGEST --> REPLY[Ответ со ссылкой<br/>на Web UI]
```

<br/>

**Концепция:** пользователь может добавить комментарий к пересланному сообщению. Бот ждёт 3 секунды, чтобы получить оба сообщения вместе.

---

## Git + LLM

```mermaid
flowchart LR
    CHANGE[Изменение в БЗ] --> DIFF[Git diff]
    DIFF --> LLM[LLM генерирует<br/>commit message]
    LLM --> COMMIT[Git commit + push]
```

```mermaid
flowchart LR
    TIMER[Таймер<br/>каждые 5 мин] --> FETCH[Git fetch + merge]
    FETCH --> REINDEX[Переиндексация<br/>изменённых файлов]
```

---

## Перевод статей

```mermaid
flowchart TD
    ARTICLE[Новая статья] --> CHECK{Нужен<br/>перевод?}
    CHECK --> |да| QUEUE[Очередь]
    QUEUE --> WORKER[Фоновый воркер]
    WORKER --> LLM[LLM перевод<br/>по частям]
    LLM --> SAVE[Сохранение перевода]
    CHECK --> |нет| SKIP[Пропуск]
```

---

## Аутентификация

<br/>

```mermaid
flowchart LR
    subgraph Режимы
        OFF[Открытый<br/>без авторизации]
        PWD[Пароль]
        OAUTH[Google OAuth<br/>+ allowlist email]
    end
```

<br/>

Три взаимоисключающих режима. Сессии in-memory с TTL. Открытый режим по умолчанию — для localhost.

---

## Web UI

```mermaid
flowchart LR
    subgraph React SPA
        BROWSE[Обзор<br/>и навигация]
        VIEW[Просмотр<br/>и редактирование]
        ADD[Добавление<br/>контента]
        SEARCH[Поиск]
        CHAT[Чат<br/>с RAG]
    end
    subgraph Backend
        API[REST API]
    end
    BROWSE & VIEW & ADD & SEARCH & CHAT --> API
```

<br/>

SSE-streaming для чата, поддержка Mermaid-диаграмм, мобильная адаптация.

---

## Ключевые концептуальные решения

<br/>

<table>
<tr>
<td style="vertical-align:top; width:50%;">

**Файлы, не БД**

- Markdown + Git
- Версионирование «из коробки»
- Локальность и контроль

</td>
<td style="vertical-align:top; width:50%;">

**Graceful degradation**

- Без LLM → ручной ввод
- Без embeddings → ключевой поиск
- Без интернета → полная функциональность

</td>
</tr>
<tr>
<td style="vertical-align:top; width:50%;">

**OpenAI-compatible**

- Любой провайдер
- Локальные модели (Ollama)
- Нет привязки к вендору

</td>
<td style="vertical-align:top; width:50%;">

**AI повсюду**

- Ingestion через Function Calling
- RAG для чата и поиска
- Автоперевод и автокоммиты

</td>
</tr>
</table>

---

## Итоги

<br/>

**Knowledge DB** — персональная база знаний, где AI пронизывает все уровни:

<br/>

1. **Ingestion** — LLM с инструментами классифицирует и структурирует контент
2. **RAG Chat** — гибридный поиск + генерация ответов на основе базы
3. **Векторный индекс** — семантический поиск по embeddings
4. **Query Rewrite** — LLM оптимизирует запросы для поиска
5. **Auto Translation** — фоновый перевод через LLM
6. **Smart Commits** — осмысленные git-коммиты

<br/>

> Все AI-функции опциональны — система работает и без них

---

## Ссылки

<br/>

- **Репозиторий:** https://github.com/strider2038/knowledge-db
- **Лицензия:** MIT © 2026 Igor Lazarev
- **Спецификации:** `openspec/specs/`
- **Agent Skills:** `.cursor/skills/`
