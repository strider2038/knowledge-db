---
name: golang-validation
description: Валидация данных в Go с github.com/muonsoft/validation. Используй при проверке входных данных, доменных моделей.
---

# Валидация (muonsoft/validation)

Пакет: `github.com/muonsoft/validation`.

Применяется:
- в доменных моделях (метод `Validate`);
- в use case-ах для проверки команд/запросов;
- в тестах через `validationtest`.

## Интерфейс Validatable

```go
type Validatable interface {
    Validate(ctx context.Context, v *validation.Validator) error
}
```

## Метод Validate

```go
func (n Node) Validate(ctx context.Context, v *validation.Validator) error {
	return v.Validate(ctx,
		validation.StringProperty("path", n.Path),
		validation.StringProperty("annotation", n.Annotation),
	)
}
```

## Типы свойств

- `StringProperty`, `NumberProperty`, `BoolProperty`
- `ValidProperty` — вложенный Validatable
- `ValidSliceProperty` — слайс

## В use case

```go
if err := u.validator.ValidateIt(ctx, cmd); err != nil {
	return err
}
```

## CreateViolation (ручная проверка)

```go
if invalid {
	return v.CreateViolation(ctx, ErrInvalidFormat, "message", validation.PropertyName("field"))
}
```

## Тестирование (validationtest)

```go
validationtest.Assert(t, err).
	IsViolation().
	WithError(validation.ErrIsBlank).
	WithPropertyPath("path")
```

## Рекомендации

- Валидация формата HTTP — responsibility handler
- Бизнес-правила — домен и use case через validation
- 422 — через validation, не через кастомные ошибки API
