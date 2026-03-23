## ADDED Requirements

### Requirement: Подкоманда expand-urls

kb-cli ДОЛЖЕН (SHALL) предоставлять подкоманду expand-urls для раскрытия редиректных ссылок и удаления UTM/трекинговых параметров в markdown-файлах базы. Подробная спецификация — в `kb-cli-expand-urls`.

#### Сценарий: Наличие подкоманды

- **WHEN** пользователь вызывает kb-cli expand-urls --help
- **THEN** отображается справка по флагам --path, --file, --dry-run
