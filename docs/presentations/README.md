# Презентации

## Knowledge DB overview

Исходник презентации:

```bash
docs/presentations/knowledge-db-overview.md
```

Сборка HTML:

```bash
bash docs/presentations/build.sh
```

Результат:

```bash
docs/presentations/knowledge-db-overview.html
```

Экспорт PDF:

```bash
bash docs/presentations/export-pdf.sh
```

Результат:

```bash
docs/presentations/knowledge-db-overview.pdf
```

Почему не просто `pandoc ...`: стандартный pandoc-шаблон для reveal.js подключает Search и Zoom плагины с CDN. В Chromium они могут блокироваться, из-за чего презентация падает с ошибкой `RevealSearch is not defined`. Скрипт сборки убирает эти плагины из HTML.

Диаграммы сделаны встроенными SVG прямо в Markdown. Это надёжнее, чем Mermaid для reveal.js: презентация не зависит от Mermaid runtime и не ломается при рендере скрытых слайдов.

В исходнике презентации не добавляйте отдельный `# H1` после YAML-заголовка: pandoc/reveal.js воспринимает его как верхний уровень и заворачивает все `##`-слайды во вложенный vertical stack. Используйте `##` для обычных слайдов.

PDF-экспорт использует отдельный `knowledge-db-overview.print.html`: это статичная версия без reveal.js runtime, где каждый слайд становится отдельной страницей 16:9. Такой путь стабильнее, чем печатать интерактивный reveal HTML напрямую.

Для локальной проверки удобно поднять статический сервер:

```bash
cd docs/presentations
python3 -m http.server 8099
```

И открыть:

```text
http://localhost:8099/knowledge-db-overview.html
```
