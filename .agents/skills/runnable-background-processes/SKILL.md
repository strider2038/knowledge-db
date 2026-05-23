---
name: runnable-background-processes
description: Background processes in kb-server via pior/runnable. Use when adding the Telegram bot, index sync worker, or other long-running Run(context) loops.
---

# Background processes (runnable)

kb-server uses [pior/runnable](https://github.com/pior/runnable) (`runnable.Manager`) for graceful shutdown.

## Runnable interface

```go
type Runnable interface {
    Run(context.Context) error
}
```

`Run` blocks until `ctx` is cancelled. On shutdown, the manager cancels context and waits for workers.

## Registering in main

```go
m.Register(
    runnable.HTTPServer(srv).ShutdownTimeout(30*time.Second),
    runnable.WithName("telegram-bot", bot),
)
```

Pass a logger into context for each worker (e.g. with a `runnable` attribute) so logs are filterable.

## Telegram bot example

```go
func (b *Bot) Run(ctx context.Context) error {
    clog.Info(ctx, "telegram bot: started")
    defer clog.Info(ctx, "telegram bot: stopped")

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

## Delays in loops

**Do not use `time.Sleep`** for shutdown-aware waiting — it ignores cancellation.

```go
select {
case <-ctx.Done():
    return nil
case <-time.After(interval):
    // continue
}
```

## Logging in workers

Use **`clog.FromContext(ctx)`** or `clog.Info(ctx, ...)` / `clog.Errorf(ctx, "…: %w", err)`.

Do not call `slog.Info` directly in workers — you lose the shared context attributes.

## Processes vs services

- **Register** — HTTP server, workers (stopped first)
- **RegisterService** — infrastructure torn down after processes

## Related

- Index `SyncWorker` — started from bootstrap; respects context on shutdown
- See [golang-logging](../golang-logging/SKILL.md) for error logging in loops
