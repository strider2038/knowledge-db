## 1. Зависимости

- [ ] 1.1 Добавить react-markdown, remark-gfm, rehype-highlight, highlight.js в web/package.json
- [ ] 1.2 Подключить стили highlight.js и импорт нужных языков (javascript, typescript, bash, json, yaml, python, go)

## 2. Компонент MarkdownContent

- [ ] 2.1 Создать компонент MarkdownContent с react-markdown, remark-gfm, rehype-highlight
- [ ] 2.2 Настроить кастомный компонент для ссылок: target="_blank", rel="noopener noreferrer"

## 3. OverviewPage: передача state при переходе

- [ ] 3.1 Добавить state={{ returnTo: location.pathname + location.search }} в Link при переходе на страницу узла

## 4. NodePage: навигация и breadcrumbs

- [ ] 4.1 Заменить кнопку «Назад»: navigate(location.state?.returnTo ?? '/')
- [ ] 4.2 Реализовать breadcrumbs: Обзор + сегменты path как ссылки на /?path=накопленный_путь

## 5. NodePage: заголовок и header

- [ ] 5.1 Заголовок h1: metadata.title с fallback на slug из path
- [ ] 5.2 Type badge (article=синий, link=зелёный, note=серый — как в Overview)
- [ ] 5.3 Created, updated — форматированные даты
- [ ] 5.4 Source URL — иконка ExternalLink с href, target="_blank", Tooltip при hover
- [ ] 5.5 Source author, source_date — для article/link
- [ ] 5.6 Keywords — чипсы (pill-style)

## 6. NodePage: контент и удаление блока метаданных

- [ ] 6.1 Блок «Аннотация» — рендеринг через MarkdownContent
- [ ] 6.2 Блок «Содержание» — рендеринг через MarkdownContent
- [ ] 6.3 Удалить блок «Метаданные» (JSON)
