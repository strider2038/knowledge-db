---
name: golang-errors
description: Работа с ошибками в Go (github.com/muonsoft/errors). Sentinel-ошибки, обёртывание, атрибуты, маппинг в HTTP.
---

# Ошибки в Go (muonsoft/errors)

Пакет: `github.com/muonsoft/errors`.

## Основные функции

| Функция | Назначение |
|---------|------------|
| `errors.New(msg)` | Sentinel-ошибка (без стека) |
| `errors.Errorf("action: %w", err)` | Обёртка + стек + контекст |
| `errors.Wrap(err, options...)` | Обёртка с атрибутами |
| `errors.Is(err, target)` | Проверка sentinel |
| `errors.As[T](err)` | Извлечение типизированной ошибки |

## errors.New — только для sentinel

`errors.New` — **только** для объявления sentinel на уровне пакета. Для анонимных ошибок — всегда `errors.Errorf`.

## Sentinel-ошибки

```go
var (
	ErrNodeNotFound = errors.New("node not found")
	ErrInvalidPath  = errors.New("invalid path")
)
```

## Обёртывание при возврате

**Каждый `return ..., err`** из handler или use case — обёрнут:

```go
if err != nil {
	return errors.Errorf("get node: %w", err)
}
```

С атрибутами:

```go
return errors.Errorf("save node: %w", err,
	errors.String("path", path),
)
```

## Маппинг в HTTP

- 404: `ErrNodeNotFound` → 404 Not Found
- 400: валидация → 400 Bad Request
- 500: остальные обёрнутые ошибки

## Правила

1. Sentinel только через `errors.New`
2. Каждая возвращаемая ошибка обёрнута
3. Контекст в нотации действия: `"get node"`, `"validate base"`
4. Не подавлять ошибки
5. **Не использовать panic** — возвращать ошибки вызывающему коду
