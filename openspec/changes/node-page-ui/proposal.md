# Proposal: UI страницы узла

## Why

Текущая страница узла (NodePage) имеет несколько проблем: кнопка «Назад» всегда ведёт на главную и теряет фильтры обзора; заголовок показывает технический path вместо title; контент и аннотация выводятся как plain text без рендеринга markdown; метаданные отображаются сырым JSON. Пользователю сложно читать контент, ориентироваться в иерархии и возвращаться к предыдущему контексту просмотра.

## What Changes

### Frontend (webapp)

- **Навигация «Назад»**: при переходе из Обзора — возврат на обзор с сохранением фильтров (path, type, q, sort, order, page); при прямом заходе — fallback на главную или browser back
- **Заголовок**: отображать `metadata.title` с fallback на slug из path
- **Breadcrumbs**: путь узла (path) раскладывать в хлебные крошки; каждый сегмент — ссылка на Обзор с автоматическим применением фильтра по выбранному поддереву
- **Верхняя панель (header)**: компактная строка с type badge, created, updated, source_url (иконка со ссылкой, popup при наведении), source_author, source_date, keywords как чипсы
- **Метаданные**: не показывать отдельным блоком — вся информация в header
- **Аннотация**: отдельный блок с рендерингом markdown
- **Содержание (content)**: рендеринг markdown с поддержкой таблиц и подсветки синтаксиса кода
- **Markdown-библиотеки**: react-markdown, remark-gfm (таблицы), rehype-highlight или prism (подсветка кода)

### Изменения в компонентах

- **OverviewPage**: при переходе на узел передавать `state: { returnTo: location.pathname + location.search }` в Link
- **NodePage**: переработать layout — breadcrumbs, header с метаданными, markdown-рендеринг, кнопка «Назад» с учётом state

## Capabilities

### Modified Capabilities

- `webapp`: страница узла — новая структура, breadcrumbs, header с метаданными, markdown-рендеринг, навигация «Назад» с учётом state

## Impact

- **web/src/pages/NodePage.tsx**: полная переработка layout и логики
- **web/src/pages/OverviewPage.tsx**: добавление state в Link при переходе на узел
- **web/package.json**: новые зависимости — react-markdown, remark-gfm, rehype-highlight (или prism-react-renderer)
- **web/src/components/**: новый компонент MarkdownContent или использование существующих; возможно Breadcrumbs компонент

## Out of Scope

- Редактирование узла
- Отдельный блок «Метаданные» в JSON
- Полнотекстовый поиск по content
