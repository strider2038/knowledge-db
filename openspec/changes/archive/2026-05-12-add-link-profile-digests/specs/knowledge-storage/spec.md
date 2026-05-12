## ADDED Requirements

### Requirement: Профиль внешнего источника во frontmatter

Главный markdown-файл узла MAY содержать опциональные frontmatter-поля `source_kind` и `content_profile`. Поле `source_kind` MUST описывать природу внешнего источника и при наличии иметь одно из значений: `repository`, `documentation`, `product_service`, `online_tool`, `directory_catalog`, `learning_resource`, `article`, `news`, `social_post`, `unknown`. Поле `content_profile` MUST описывать локальную форму digest и при наличии иметь одно из значений: `repository_profile`, `product_profile`, `documentation_profile`, `online_tool_profile`, `directory_profile`, `learning_resource_profile`, `conceptual_digest`, `brief_digest`, `link_bookmark`.

#### Scenario: Узел с repository profile

- **WHEN** frontmatter содержит `type: link`, `source_kind: repository`, `content_profile: repository_profile`
- **THEN** узел проходит валидацию метаданных

#### Scenario: Концептуальная заметка по статье

- **WHEN** frontmatter содержит `type: note`, `source_kind: article`, `content_profile: conceptual_digest`
- **THEN** узел проходит валидацию метаданных

#### Scenario: Старый узел без профиля

- **WHEN** frontmatter содержит обязательные поля, но не содержит `source_kind` и `content_profile`
- **THEN** узел остаётся валидным

#### Scenario: Невалидное значение source_kind

- **WHEN** поле `source_kind` присутствует и содержит значение вне допустимого списка
- **THEN** валидация метаданных сообщает об ошибке

#### Scenario: Невалидное значение content_profile

- **WHEN** поле `content_profile` присутствует и содержит значение вне допустимого списка
- **THEN** валидация метаданных сообщает об ошибке

### Requirement: Markdown-тело для link digest

Узел `type=link` MAY содержать markdown-тело с профильным digest внешнего ресурса. Если `content_profile` присутствует и не равен `link_bookmark`, тело SHOULD содержать человекочитаемое концептуальное описание ресурса. Пустое тело для обычной закладки MUST оставаться допустимым.

#### Scenario: Link с профильным телом

- **WHEN** узел `type=link` содержит `content_profile: repository_profile` и markdown-тело
- **THEN** узел проходит валидацию и тело сохраняется как часть знания

#### Scenario: Обычная закладка без тела

- **WHEN** узел `type=link` не содержит `content_profile` и имеет пустое тело
- **THEN** узел остаётся валидным
