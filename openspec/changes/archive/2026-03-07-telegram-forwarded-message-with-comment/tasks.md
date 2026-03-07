## 1. Расширение структуры update

- [x] 1.1 Добавить в `update.message` поля `ReplyToMessage`, `ForwardFrom`, `ForwardFromChat` (вложенные структуры по Telegram API)

## 2. Извлечение и объединение текста

- [x] 2.1 Реализовать `extractText(msg)` — извлекает text или caption из сообщения (включая вложенное `reply_to_message`)
- [x] 2.2 Реализовать `combineForwardWithComment(comment, forwardedContent string) string` — формат с метками «Инструкции пользователя: …», «Пересланное сообщение: …»

## 3. Обработка reply_to_message

- [x] 3.1 В `handleUpdate`: если `message.reply_to_message` не nil, объединять текст комментария и контент из `reply_to_message`, передавать в IngestText
- [x] 3.2 При объединении логировать на уровне info (chat_id, message_id)

## 4. Буфер пересланных сообщений

- [x] 4.1 Добавить буфер отложенных сообщений: ключ — (chat_id, message_id), TTL 3 с
- [x] 4.2 При получении пересланного сообщения (forward_from/forward_from_chat) без reply_to_message — класть в буфер, не обрабатывать сразу
- [x] 4.3 При получении reply: если `reply_to_message.message_id` есть в буфере — удалить из буфера, обработать reply (в нём уже оба фрагмента)
- [x] 4.4 По таймауту 3 с: если сообщение всё ещё в буфере — обработать как одиночное
- [x] 4.5 Ограничить размер буфера (например, 10 записей на chat)

## 5. Интеграция

- [x] 5.1 Обновить `handleUpdate`: ветка для reply с reply_to_message, ветка для пересланного в буфер, ветка для обычных сообщений
- [x] 5.2 Убедиться, что один цикл подтверждения и итогового ответа на пару (не два)
