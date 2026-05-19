## 1. Backend: хранение и нормализация

- [x] 1.1 Добавить `Labels []string` в `NodeListItem`, хелперы чтения/нормализации labels в `internal/kb` (trim, dedupe case-insensitive, лимиты 32×64, запрет запятой)
- [x] 1.2 Расширить `PatchNodeMetadataParams` и запись frontmatter для `labels`; пустой массив удаляет ключ
- [x] 1.3 Фильтр AND в `ListNodesWithOptions` по `Labels []string`; включить `labels` в ответ списка
- [x] 1.4 Unit-тесты store: нормализация, фильтр AND, пустые labels

## 2. Backend: API

- [x] 2.1 Query `labels` в GET /api/nodes; поле `labels` в JSON узла и списка
- [x] 2.2 PATCH `labels` в handlers; GET /api/label-suggestions
- [x] 2.3 API-тесты: list filter AND, PATCH labels, suggestions, 400 на невалидные labels

## 3. Backend: индекс

- [x] 3.1 Убедиться, что `labels` не входят в `buildNodeEmbeddingText` и `computeContentHash`
- [x] 3.2 Тест sync: смена только labels не меняет content_hash / не требует re-embed

## 4. Frontend: стили и API-клиент

- [x] 4.1 `web/src/lib/label-styles.ts` — палитра и `getLabelChipClass(label)`
- [x] 4.2 Расширить `api.ts`: types, `getNodesWithParams({ labels })`, `patchNodeLabels`, `getLabelSuggestions`

## 5. Frontend: страница узла

- [x] 5.1 Блок «Метки» с чипами и модалкой (token-input + suggestions), PATCH labels
- [x] 5.2 Тесты NodePage: отображение и сохранение labels

## 6. Frontend: обзор

- [x] 6.1 Фильтр меток в URL (`labels=`), UI выбора нескольких меток (AND)
- [x] 6.2 Колонка/чипы меток в таблице; `treeFilterPaths` при активном фильтре labels
- [x] 6.3 Тесты OverviewPage: фильтр, URL, дерево

## 7. Завершение

- [x] 7.1 `go test ./... -race`, `golangci-lint`, `cd web && npm run build && npm run test && npm run lint`
- [x] 7.2 `openspec validate add-node-labels`
