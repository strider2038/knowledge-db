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

## testify (обязательно)

Все проверки в тестах — через **testify** (`require` и `assert`):

- **`require`** — обязательные проверки, останавливают тест при провале (`require.NoError`, `require.Error`, `require.NotEmpty`)
- **`assert`** — сравнения, продолжают тест (`assert.Equal`, `assert.Empty`, `assert.True`, `assert.False`)

### Типичные паттерны

```go
// Ошибки
require.NoError(t, err)           // ожидаем успех
require.Error(t, err)             // ожидаем ошибку
assert.ErrorIs(t, err, ErrNotFound) // ошибка должна быть target

// Сравнения
assert.Equal(t, expected, actual)
assert.Empty(t, slice)             // len == 0
assert.NotEmpty(t, slice)
assert.True(t, cond)
assert.False(t, cond)
assert.NotNil(t, ptr)
assert.Contains(t, slice, element)  // slice содержит element
```

### Когда require vs assert

- `require` — когда дальнейшие проверки бессмысленны (нет данных, паника и т.п.)
- `assert` — для обычных проверок значений

### Хелперы и setup: без panic

В тестовом коде **не использовать panic** — даже в хелперах и setup. При ошибках вызывать `tb.Fatalf`:

```go
func buildTestData(tb testing.TB, input string) *Response {
    tb.Helper()
    data, err := json.Marshal(input)
    if err != nil {
        tb.Fatalf("marshal: %v", err)
    }
    // ...
}
```

- Хелперы принимают `testing.TB` (работает с `*testing.T` и `*testing.B`)
- Вызывать `tb.Helper()` в начале хелпера — корректный stack trace при падении
- `tb.Fatalf` останавливает тест с понятным сообщением

## Afero: in-memory fs в тестах

Для тестов с файловой структурой используй **afero** вместо `t.TempDir()` и `os.WriteFile`:

- **Production**: `NewStore(afero.NewOsFs())`
- **Тесты**: `NewStore(afero.NewMemMapFs())` — in-memory, без диска, быстрее

### Паттерн

1. Код принимает `afero.Fs` (через `Store` или `NewStore(fs)`).
2. В тестах создаёшь `fs := afero.NewMemMapFs()`, заполняешь через `afero.WriteFile`, `afero.MkdirAll`.
3. Вызываешь методы `Store` с этим fs.

### Хелпер

```go
func seedMemFS(files map[string]string) (*Store, string) {
    fs := afero.NewMemMapFs()
    basePath := "/"
    for path, content := range files {
        fullPath := filepath.Join(basePath, path)
        _ = fs.MkdirAll(filepath.Dir(fullPath), 0755)
        _ = afero.WriteFile(fs, fullPath, []byte(content), 0644)
    }
    return NewStore(fs), basePath
}
```

Пример: `store, base := seedMemFS(map[string]string{"topic/node1/node1.md": "---\nkeywords: []\n---\n"})`.

### Пути

Для MemMapFs используй **абсолютные** пути (`/` как base). `filepath.Join` с `/` даёт корректные пути.

## Чек-лист

- [ ] API-эндпоинты покрыты тестами
- [ ] Arrange–Act–Assert
- [ ] Именование TestXxx_WhenYyy_ExpectZzz
- [ ] Проверки через testify (require/assert), не t.Error/t.Fatal
- [ ] Хелперы без panic — использовать tb.Fatalf (testing.TB)
- [ ] Тесты с файлами — через afero MemMapFs, не t.TempDir
