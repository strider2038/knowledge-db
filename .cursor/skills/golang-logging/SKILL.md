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
| Warn | Нештатные, некритичные ситуации |
| Error | Ошибки, требующие внимания |

## Логирование ошибок (Error-уровень)

**Обязательно** использовать `clog.Errorf`, а не `clog.Error` + `slog.String("error", ...)`:

```go
// Неправильно
clog.Error(ctx, "failed", slog.String("error", err.Error()))

// Правильно
clog.Errorf(ctx, "get node failed: %w", err)
```

## Правила

1. Логгер только из контекста (`clog.FromContext`)
2. Воркеры (Telegram bot и т.п.) — только `clog.FromContext(ctx)`, не `slog.*` напрямую
3. Не логировать чувствительные данные (токены, пароли)
4. Сообщения в нотации действия: `"get node"`, `"validate base"`
