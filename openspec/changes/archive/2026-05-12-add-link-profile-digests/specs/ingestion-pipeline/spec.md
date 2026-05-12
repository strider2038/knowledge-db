## ADDED Requirements

### Requirement: Классификация URL перед созданием узла

Ingestion pipeline ДОЛЖЕН (SHALL) классифицировать URL перед вызовом `create_node`. Классификация MUST определять `source_kind`, `content_profile` и рекомендуемый `type` на основе URL, метаданных, README, preview или полного извлечённого контента. При пользовательском `TypeHint` система MUST учитывать подсказку, но SHOULD сохранять совместимую пару `source_kind` и `content_profile`.

#### Scenario: GitHub-репозиторий

- **WHEN** IngestText или IngestURL получает `https://github.com/pior/runnable`
- **THEN** pipeline выбирает `type=link`, `source_kind=repository`, `content_profile=repository_profile`

#### Scenario: Длинная статья без полного копирования

- **WHEN** пользователь сохраняет URL длинной статьи с намерением получить концептуальное описание
- **THEN** pipeline выбирает `type=note`, `source_kind=article`, `content_profile=conceptual_digest`

#### Scenario: Новостная публикация

- **WHEN** пользователь сохраняет URL новости о релизе AI-модели
- **THEN** pipeline выбирает `type=note`, `source_kind=news`, `content_profile=brief_digest`

### Requirement: Генерация digest-тела для внешнего источника

LLM-оркестратор ДОЛЖЕН (SHALL) генерировать markdown-тело digest для `repository_profile`, `product_profile`, `documentation_profile`, `online_tool_profile`, `directory_profile`, `learning_resource_profile`, `conceptual_digest` и `brief_digest`. Digest MUST быть на русском языке, MUST быть основан на фактах из доступного источника и MUST отбрасывать технический шум, не нужный для концептуального понимания.

#### Scenario: Repository profile из README

- **WHEN** pipeline обрабатывает репозиторий с доступным README
- **THEN** LLM получает README или его релевантное preview и создаёт markdown-тело `repository_profile`

#### Scenario: Conceptual digest из статьи

- **WHEN** pipeline обрабатывает статью в режиме концептуального digest
- **THEN** LLM получает извлечённый контент или preview и создаёт markdown-тело без полного копирования статьи

#### Scenario: Brief digest из новости

- **WHEN** pipeline обрабатывает новостную публикацию
- **THEN** LLM создаёт краткое markdown-тело с сутью новости, технической значимостью и ограничениями информации

### Requirement: Передача профиля источника в create_node

Инструмент `create_node` ДОЛЖЕН (SHALL) поддерживать поля `source_kind` и `content_profile` наряду с существующими метаданными. При создании узла pipeline MUST записывать эти поля во frontmatter, если они были определены классификацией или LLM-оркестратором. Поле `content` MUST содержать digest body для профильных link/note узлов.

#### Scenario: create_node для repository profile

- **WHEN** LLM вызывает `create_node` для репозитория
- **THEN** запрос содержит `type=link`, `source_kind=repository`, `content_profile=repository_profile` и непустой `content`

#### Scenario: create_node для conceptual digest

- **WHEN** LLM вызывает `create_node` для длинной статьи без полного копирования
- **THEN** запрос содержит `type=note`, `source_kind=article`, `content_profile=conceptual_digest` и непустой `content`

#### Scenario: Старый сценарий link bookmark

- **WHEN** LLM создаёт обычную ссылку-закладку без digest
- **THEN** pipeline MAY создать `type=link` без `source_kind`, без `content_profile` и с пустым `content`

### Requirement: Сохранение полной статьи по явному намерению

Pipeline ДОЛЖЕН (SHALL) сохранять полный текст источника как `type=article`, если пользователь явно просит локальную копию статьи или TypeHint указывает `article`. В этом случае система MUST использовать существующий flow `fetch_url_content` и не заменять статью концептуальным digest.

#### Scenario: Пользователь просит сохранить статью целиком

- **WHEN** пользователь отправляет URL и инструкцию "сохрани полную статью"
- **THEN** pipeline создаёт `type=article` с полным markdown-контентом источника

#### Scenario: Пользователь просит краткую выжимку

- **WHEN** пользователь отправляет URL статьи и инструкцию "сохрани концептуальное описание"
- **THEN** pipeline создаёт `type=note`, `source_kind=article`, `content_profile=conceptual_digest`
