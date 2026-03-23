## 1. urlutil

- [x] 1.1 Добавить `tryNormalizeURL` (или экспортируемый вариант), возвращающий признак успеха HEAD, не ломая `NormalizeURL`
- [x] 1.2 Расширить снятие query: `utm_*` и распространённые трекинг-ключи (`fbclid`, `gclid`, …); обновить тесты `normalize_test.go`

## 2. internal/kb

- [x] 2.1 Реализовать извлечение `http(s)` URL из markdown (`[x](url)`, `![x](url)`, `<url>`) и однострочных значений во frontmatter
- [x] 2.2 Реализовать `RunExpandURLs` (или аналог): нормализация уникальных URL, замены с сортировкой по длине, `--dry-run`, учёт частичных ошибок
- [x] 2.3 Unit-тесты на замену и frontmatter

## 3. kb-cli

- [x] 3.1 Файл `expand_urls.go`: cobra `expand-urls`, флаги `--path`, `--file`, `--dry-run`, разрешение путей как у `dump-images`
- [x] 3.2 Зарегистрировать команду в `main.go`

## 4. Проверки

- [x] 4.1 `go build ./... && go test ./... -race` из корня репозитория
- [x] 4.2 `golangci-lint run ./...` при наличии конфига
