## ADDED Requirements

### Requirement: Обновление описания узла из источника

REST API ДОЛЖЕН (SHALL) предоставлять endpoint `POST /api/nodes/{path}/refresh-description` для обновления описания существующего узла на основе его `source_url`. Endpoint MUST загружать текущий узел, требовать наличие `source_url`, запускать тот же алгоритм классификации и генерации digest, что используется при ingestion внешних источников, и сохранять обновлённый markdown-файл узла. Ответ MUST содержать обновлённый объект узла. Если `source_url` отсутствует, endpoint MUST возвращать 400. Если узел не найден, endpoint MUST возвращать 404.

Endpoint MUST обновлять описательные поля: `annotation`, `keywords`, `source_kind`, `content_profile` и markdown-тело digest. Endpoint MAY обновить `type`, если классификация показывает, что текущий тип был ошибочным, например новость `type=link` должна стать `type=note` с `content_profile=brief_digest`. Endpoint MUST сохранять `created`, `source_url`, `manual_processed` и пользовательские поля, не относящиеся к описанию источника. Endpoint SHOULD сохранять существующие `source_author` и `source_date`, если новый источник не даёт более точных значений.

#### Scenario: Обновление repository link

- **WHEN** клиент вызывает `POST /api/nodes/programming/golang/packages/go-libraries-runnable-manager/refresh-description` для узла с `source_url` на GitHub-репозиторий
- **THEN** API обновляет узел как `type=link`, `source_kind=repository`, `content_profile=repository_profile` и возвращает обновлённый объект узла

#### Scenario: Обновление длинной статьи как conceptual digest

- **WHEN** клиент вызывает refresh-description для узла с `source_url` на длинную статью, которая не хранится полной копией
- **THEN** API обновляет узел как `type=note`, `source_kind=article`, `content_profile=conceptual_digest` и сохраняет markdown-тело digest

#### Scenario: Исправление новости, ошибочно сохранённой как link

- **WHEN** клиент вызывает refresh-description для узла `type=link` с `source_url` на новостную публикацию
- **THEN** API MAY изменить тип на `note`, установить `source_kind=news`, `content_profile=brief_digest` и сохранить краткое markdown-тело digest

#### Scenario: Узел без source_url

- **WHEN** клиент вызывает refresh-description для узла без `source_url`
- **THEN** API возвращает 400 с сообщением, что обновление из источника невозможно

#### Scenario: Узел не найден

- **WHEN** клиент вызывает refresh-description для неизвестного пути
- **THEN** API возвращает 404

#### Scenario: Ошибка LLM или fetch источника

- **WHEN** источник недоступен или LLM-конфигурация отсутствует
- **THEN** API возвращает ошибку 503 или 502 с диагностируемым сообщением и не изменяет markdown-файл узла

#### Scenario: Переиндексация обновлённого узла

- **WHEN** refresh-description успешно сохраняет узел
- **THEN** API инициирует переиндексацию этого узла тем же механизмом, который используется после изменения узлов
