---
name: golang-logging
description: Контекстное логирование в Go с github.com/muonsoft/clog. Используй при добавлении логов, работе с контекстным логгером в internal/ и cmd/.
---

# Логирование в Go (muonsoft/clog)

Пакет: `github.com/muonsoft/clog` (надстройка над `log/slog`).

## Основные концепции

- Логгер привязан к `context.Context`
- Атрибуты (`slog.Attr`) добавляются к логгеру
- Ошибки из `github.com/muonsoft/errors` логируются напрямую — атрибуты подхватываются

## Привязка логгера к контексту

```go
import (
	"log/slog"
	"github.com/muonsoft/clog"
)

// В middleware или main:
logger := slog.Default().With(
	slog.String("request_method", r.Method),
	slog.String("request_url", r.URL.Path),
)
ctx := clog.NewContext(r.Context(), logger)
```

Извлечение: `logger := clog.FromContext(ctx)`

## Уровни

| Уровень | Использование |
|---------|---------------|
| Debug | Отладочная информация |
| Info | Нормальный ход работы |
| Warn | Нештатные, некритичные ситуации (fallback, опциональный шаг) |
| Error | Ошибки важных частей логики — с `clog.Errorf` и `%w` для сохранения стека |

## Логирование ошибок (Error-уровень)

**Обязательно** использовать `clog.Errorf`, а не `clog.Error` + `slog.String("error", ...)`:

```go
// Неправильно
clog.Error(ctx, "failed", slog.String("error", err.Error()))

// Правильно
clog.Errorf(ctx, "get node failed: %w", err)
```

## Важные части логики: Error, не Warn

Для ошибок **важных частей логики** (перевод, сохранение, критичные шаги pipeline) — использовать **`clog.Errorf`** с `%w`, а не `Warn`:

```go
// Неправильно — теряется стек, уровень занижен
clog.Warn(ctx, "translation failed", "error", err)

// Правильно — стек сохраняется через %w
clog.Errorf(ctx, "ingest text: translation failed: %w", err)
```

Warn — для некритичных ситуаций (fallback, опциональный шаг). Error — когда сбой влияет на результат.

## Использование логгера

**Не вызывать** `clog.FromContext(ctx)` многократно в одной функции — только для однократного извлечения, если логгер переиспользуется:

```go
// Неправильно — многократный вызов FromContext
clog.FromContext(ctx).Info("start", "x", 1)
clog.FromContext(ctx).Info("done", "x", 2)

// Правильно — сокращённый метод
clog.Info(ctx, "start", "x", 1)
clog.Info(ctx, "done", "x", 2)

// Либо — однократное извлечение при частом переиспользовании
logger := clog.FromContext(ctx)
logger.Info("start", "x", 1)
logger.Debug("step", "y", 2)
logger.Info("done", "x", 2)
```

Сокращённые методы: `clog.Info(ctx, msg, args...)`, `clog.Warn(ctx, msg, args...)`, `clog.Debug(ctx, msg, args...)`.

## Правила

1. Логгер только из контекста (`clog.FromContext` или `clog.Info(ctx, ...)` и т.п.)
2. Воркеры (Telegram bot и т.п.) — только `clog.FromContext(ctx)` или `clog.*(ctx, ...)`, не `slog.*` напрямую
3. Не логировать чувствительные данные (токены, пароли)
4. Сообщения в нотации действия: `"get node"`, `"validate base"`
5. Важные ошибки — `clog.Errorf` с `%w`, не `Warn` с атрибутом
