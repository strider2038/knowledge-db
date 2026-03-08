## 1. Backend: internal/kb

- [ ] 1.1 Добавить метод рекурсивного списка узлов с метаданными (path, title, type, created, source_url, translations); паттерн переводов `*.[a-z]{2}.md` — не включать в список
- [ ] 1.2 Реализовать фильтр по type (article, link, note)
- [ ] 1.3 Реализовать поиск q по title, keywords, annotation (case-insensitive)
- [ ] 1.4 Реализовать сортировку (sort: title, type, created, source_url; order: asc, desc) и пагинацию (limit, offset)

## 2. Backend: internal/api

- [ ] 2.1 Расширить ListNodes handler: парсинг path, recursive, q, type, limit, offset, sort, order
- [ ] 2.2 Возвращать формат { nodes: NodeListItem[], total: number }
- [ ] 2.3 API-тесты для GET /api/nodes с recursive, q, type, limit, offset, sort, order

## 3. Frontend: API и типы

- [ ] 3.1 Добавить тип NodeListItem и функцию getNodes с query-параметрами

## 4. Frontend: Страница «Обзор»

- [ ] 4.1 Рефакторинг SearchPage → OverviewPage: дерево слева, таблица справа
- [ ] 4.2 Дерево с вариантом «Вся база» и breadcrumbs
- [ ] 4.3 Toggle-кнопки фильтров по типам (article, link, note)
- [ ] 4.4 Инпут поиска (debounce, передача q в API)
- [ ] 4.5 Сортируемая таблица: колонки Название, Тип (цветом), Дата, Ссылка (target="_blank" для link/article)
- [ ] 4.6 Фильтрация дерева по типам — скрытие веток без подходящих узлов
- [ ] 4.7 Синхронизация state с URL (path, type, q, sort, order, page)
- [ ] 4.8 Пагинация таблицы

## 5. Frontend: Navbar и NodePage

- [ ] 5.1 Переименовать «Поиск» → «Обзор» в Navbar
- [ ] 5.2 NodePage: выбор языка при наличии translations (оригинал / переводы)
