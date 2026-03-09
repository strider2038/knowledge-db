## ADDED Requirements

### Requirement: Подкоманда dump-images

kb-cli ДОЛЖЕН (SHALL) предоставлять подкоманду dump-images для скачивания удалённых изображений из markdown-статьи и замены ссылок на локальные пути. Подробная спецификация — в `kb-cli-dump-images`.

#### Сценарий: Наличие подкоманды

- **WHEN** пользователь вызывает kb-cli dump-images --help
- **THEN** отображается справка по флагам --path, --file, --dry-run
