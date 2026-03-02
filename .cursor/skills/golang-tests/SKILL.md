---
name: golang-tests
description: Тестирование на Go — API-тесты (muonsoft/api-testing), unit-тесты. Arrange–Act–Assert. Используй при написании тестов для internal/ и cmd/.
---

# Тестирование на Go (knowledge-db)

Цель: покрыть тестами API-handlers, internal/kb, internal/ingestion.

## Один аспект на тест, Arrange–Act–Assert

- Один тест — один сценарий
- **Arrange** — подготовка (моки, handler)
- **Act** — одно действие (HTTP-запрос или вызов функции)
- **Assert** — проверки результата

## API-тесты (muonsoft/api-testing)

Пакеты **apitest** и **assertjson**:

```go
func TestGetNode_WhenNotFound_Expect404(t *testing.T) {
	t.Parallel()

	// Arrange
	handler := setupTestHandler() // или NewTestHandler(container)

	// Act
	resp := apitest.HandleGET(t, handler, "/api/nodes/missing/path")

	// Assert
	resp.IsNotFound()
}
```

**assertjson.Node** — вариадический синтаксис: `Node("key", 0, "nested")`, не legacy `/key/0/nested`.

## Моки

- Ручные моки в `internal/<pkg>/mocks` или `*_test.go`
- Хранилище на `map` по ключу
- Ошибки через `errors.Errorf`, не `errors.New` для анонимных

## Именование

```
Test<Entity>_<Action>_When<Condition>_Expect<Result>
```

Примеры: `TestGetNode_WhenValidPath_ExpectOK`, `TestValidate_WhenMissingContent_ExpectError`.

## testify

- `require` — обязательные проверки (останавливают тест)
- `assert` — сравнения
- `require.NoError`, `assert.ErrorIs` для ошибок

## Чек-лист

- [ ] API-эндпоинты покрыты тестами
- [ ] Arrange–Act–Assert
- [ ] Именование TestXxx_WhenYyy_ExpectZzz
