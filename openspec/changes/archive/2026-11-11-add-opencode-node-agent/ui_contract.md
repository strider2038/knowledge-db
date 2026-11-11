# UI Contract

## Scope

| Route / surface | Change | User goal | Roles |
|-----------------|--------|-----------|-------|
| Страница узла, модалка «Редактировать с агентом» | MODIFIED | Понять, что правка выполнится на сервере, без привязки к Cursor | Пользователь базы |

Новых маршрутов, кнопок и полей нет.

## Navigation and Entry Points

Вход прежний: Action Bar страницы узла, кнопка «Редактировать с агентом».

## Page / Flow: Модалка инструкции

### Layout and Design-System Components

`Dialog` / `DialogTitle` / `DialogDescription` / `Button` без смены композиции. Меняется только текст `DialogDescription`.

Текст пояснения: «Опишите, что нужно изменить в узле. Агент отредактирует файл на сервере; ход выполнения отобразится в панели логов.»

Слова «Cursor» в модалке нет.

### Data and Related Entities

Запрос прежний: `startNodeAgentEdit(path, instruction)`. Ответ джобы не отображает имя агента.

### Filters / Sorting / Pagination

Не применимо: модалка не список.

### Actions and Forms

Поле инструкции, «Отмена», «Запустить». «Запустить» disabled при пустой инструкции. Успех закрывает модалку и открывает панель логов. Ошибка старта остаётся в модалке.

### States

| State | Required behavior |
|-------|-------------------|
| loading | Кнопка «Запуск...» и disabled на время запроса, как сейчас |
| primary empty | Пустая инструкция, «Запустить» disabled |
| filtered empty | Не применимо |
| error + retry | Текст ошибки API в модалке, поле инструкции сохраняется |
| unauthorized / forbidden | Существующий обработчик API, без нового экрана |
| validation | Пустая инструкция не отправляется |
| success / stale / reconnect | Успешный старт закрывает модалку |

### Responsive and Accessibility

- Supported viewport/breakpoint checks: модалка на той же вёрстке, отдельной мобильной ветки нет.
- Keyboard/focus behavior: без изменений, фокус в диалоге.
- Semantics/labels/announcements: заголовок «Редактировать с агентом», пояснение в `DialogDescription`.
- Reduced-motion or contrast requirements: без новых требований.

## Explicit Non-Goals

- Селектор агента, бейдж текущего провайдера, отдельная страница настроек.
- Изменение панели логов, кроме того что строки по-прежнему приходят с API.

## Open Questions

Нет.
