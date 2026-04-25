---
name: runnable-background-processes
description: Регистрация фоновых процессов через pior/runnable. Используй при добавлении Telegram bot и других воркеров в kb-server.
---

# Фоновые процессы (runnable)

Фоновые процессы управляются [pior/runnable](https://github.com/pior/runnable) через `runnable.Manager`.

## Интерфейс Runnable

```go
type Runnable interface {
    Run(context.Context) error
}
```

Процесс блокируется в `Run` до отмены контекста. При shutdown Manager отменяет контекст.

## Добавление процесса (например, Telegram bot)

### 1. Реализовать Run

```go
func (b *TelegramBot) Run(ctx context.Context) error {
    logger := clog.FromContext(ctx)
    logger.Info("telegram bot: started")
    defer logger.Info("telegram bot: stopped")

    for {
        select {
        case <-ctx.Done():
            return nil
        default:
            b.poll(ctx)
        }
    }
}
```

### 2. Зарегистрировать в main

```go
m.Register(
    runnable.HTTPServer(srv).ShutdownTimeout(30*time.Second),
    runnable.WithName("telegram-bot", bot),  // или обёртка с логгером в контексте
)
```

При регистрации — обеспечить, чтобы в контекст воркера передавался логгер с атрибутом `runnable=<name>` (для фильтрации в логах).

## Паузы в циклах

**Не использовать `time.Sleep`** — он не реагирует на отмену контекста. При shutdown процесс будет ждать полный интервал вместо немедленной остановки.

Использовать `select` с `ctx.Done()` и `time.After`:

```go
select {
case <-ctx.Done():
    return nil
case <-time.After(5 * time.Second):
    // продолжаем работу
}
```

## Логирование в воркерах

**Только `clog.FromContext(ctx)`.** Запрещён прямой вызов `slog.Info`, `slog.Warn` — иначе нет атрибута `runnable` в логах.

```go
// Правильно
logger := clog.FromContext(ctx)
logger.Warn("poll failed", slog.String("error", err.Error()))

// Неправильно
slog.Warn("poll failed", ...)
```

## Processes vs Services

- **Register** — HTTP-сервер, воркеры (останавливаются первыми)
- **RegisterService** — инфраструктура (останавливается после процессов)
