---
name: golang-logging
description: Contextual logging in Go with github.com/muonsoft/clog. Use when adding logs, binding loggers to context, or logging errors with preserved stack traces.
---

# Logging in Go (muonsoft/clog)

Package: `github.com/muonsoft/clog` (built on `log/slog`).

## Context API (knowledge-db)

| Function | Purpose |
|----------|---------|
| `clog.NewContext(ctx, logger)` | Bind `*slog.Logger` to context (middleware / main) |
| `clog.FromContext(ctx)` | Extract logger from context |
| `clog.Info(ctx, msg, args...)` | Info — shorthand when logger not stored locally |
| `clog.Warn(ctx, msg, args...)` | Warning |
| `clog.Debug(ctx, msg, args...)` | Debug |
| `clog.Error(ctx, msg, ...slog.Attr)` | Error with attributes |
| `clog.Errorf(ctx, format, ...any)` | Error with `%w` for `error` values (preserves stack) |

Prefer `clog.Info(ctx, ...)` / `clog.Errorf(ctx, ...)` over calling `FromContext` repeatedly unless you need the logger for many calls in one function.

### Binding in middleware

```go
logger := slog.Default().With(
    slog.String("request_method", r.Method),
    slog.String("request_url", r.URL.Path),
)
ctx := clog.NewContext(r.Context(), logger)
// pass ctx to handlers
```

## slog attributes

```go
slog.String("path", path)
slog.Int("count", n)
slog.Bool("ok", true)
slog.Any("detail", value)
```

Pass path, node id, job id, etc. as structured fields — not only inside the message string.

## Logging errors

**Use `clog.Errorf` with `%w`** — not `clog.Error` + `slog.String("error", err.Error())`:

```go
// Wrong — loses stack
clog.Error(ctx, "get node failed", slog.String("error", err.Error()))

// Correct
clog.Errorf(ctx, "get node failed: %w", err)
```

### Error vs Warn

| Level | When |
|-------|------|
| **Error** (`clog.Errorf`) | Failed ingestion step, git commit/push failure, DB/index errors, OAuth/session failures that matter for the operation |
| **Warn** | Optional degradation, retryable git push noted in background, non-fatal queue item |
| **Debug** | Expected client errors (404 on move/delete) when useful for support — see handlers using `clog.Debug` before 404 |

When unsure, prefer **Error** so real bugs are not hidden.

### Handler pattern (from `internal/api`)

```go
if errors.Is(err, kb.ErrNodeNotFound) {
    clog.Debug(r.Context(), "delete node: not found", "path", nodePath)
    writeError(w, http.StatusNotFound, "node not found")
    return
}
clog.Errorf(r.Context(), "delete node: %w", err)
writeError(w, http.StatusInternalServerError, err.Error())
```

## Workers and loops

```go
for task := range q.Tasks() {
    if err := worker.handle(ctx, task); err != nil {
        clog.Errorf(ctx, "handle task: %w", err)
    }
}
```

## Rules

1. Do not log passwords, tokens, or session secrets.
2. Always pass `context.Context` as the first argument.
3. Business logic in `internal/` uses **`clog` only** — do not add `*slog.Logger` fields to domain types; inject logger into context at the edge (HTTP middleware, `main`).
4. Errors: `clog.Errorf` with `%w`, or attributes via `slog.*` for non-error fields.
5. Do not duplicate attributes already on the request context logger.

## Checklist

- [ ] `github.com/muonsoft/clog` used in `internal/` and handlers
- [ ] Logger bound with `clog.NewContext` at request/worker boundary
- [ ] Failures logged with `clog.Errorf` and `%w`
- [ ] No sensitive data in log fields
- [ ] No ad-hoc `slog.Default()` in business packages
