## Purpose

Массовая обработка экспортированного чата Telegram (формат одного чата). Парсинг JSON, backend-сессии для пошаговой обработки (принять/отклонить), восстановление после перезагрузки.

## Requirements

### Requirement: Парсинг Telegram export JSON

Система ДОЛЖНА (SHALL) парсить JSON экспорта одного чата Telegram. Корень файла — объект Chat с полями id, name, type, messages[]. Из каждого Message с type="message" MUST извлекаться: text (из text или text_entities — конкатенация с учётом link/text_link в Markdown), source_author (forwarded_from || saved_from || from), source_url (первая link/text_link из text_entities). Для медиа-сообщений MUST использоваться caption при отсутствии text. Сообщения без извлекаемого текста MUST пропускаться. Сообщения MUST сортироваться по date_unixtime по убыванию (новые первыми).

#### Сценарий: Парсинг сообщения с text-массивом

- **WHEN** Message содержит text как массив [{type:"link", text:"https://..."}, {type:"plain", text:" - комментарий"}]
- **THEN** извлекается текст в формате Markdown и source_url из link-сущности

#### Сценарий: Парсинг source_author

- **WHEN** Message содержит forwarded_from и saved_from
- **THEN** source_author = forwarded_from (приоритет над saved_from)

#### Сценарий: Сортировка по дате

- **WHEN** парсятся несколько сообщений
- **THEN** порядок — по date_unixtime убыванию (новые первыми)

### Requirement: Хранение сессий импорта

Система ДОЛЖНА (SHALL) хранить сессии импорта в директории {KB_UPLOADS_DIR}/telegram-import-sessions/. Каждая сессия — JSON-файл {session_id}.json с полями: session_id, created_at, total, current_index, processed_ids, rejected_ids, items (массив записей с id, text, source_author, source_url). KB_UPLOADS_DIR задаётся переменной окружения.

#### Сценарий: Создание сессии

- **WHEN** загружен валидный JSON экспорта
- **THEN** создаётся файл сессии в telegram-import-sessions/

#### Сценарий: Отсутствие KB_UPLOADS_DIR

- **WHEN** KB_UPLOADS_DIR не задана
- **THEN** импорт недоступен (endpoints возвращают ошибку конфигурации)

### Requirement: Завершение сессии в веб-интерфейсе

После обработки последней записи сессии импорта (ответ accept или reject без `next_item`) веб-клиент ДОЛЖЕН (SHALL) показывать промежуточный экран завершения с явным сообщением и сводкой (всего записей в выгрузке, принято, отклонено) и действием для начала нового импорта, а не сразу только форму выбора JSON-файла. Детальные формулировки UI MUST соответствовать спецификации `webapp` (страница «Добавить», вкладка «Импорт из Telegram», блок «Завершение импорта»).

#### Сценарий: Нет следующей записи после accept/reject

- **WHEN** API возвращает ответ без следующей записи для обработки
- **THEN** пользователь видит экран завершения импорта до возможности выбрать новый файл
