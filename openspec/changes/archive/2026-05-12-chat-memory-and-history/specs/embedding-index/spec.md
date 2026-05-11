## MODIFIED Requirements

### Requirement: Синхронизация индекса
Система ДОЛЖНА (SHALL) предоставлять SyncWorker (реализующий `runnable.Runnable`) для синхронизации индекса с git-репозиторием. SyncWorker MUST обрабатывать события: SingleNode(path) — индексация одной ноды; GitSyncDiff — diff после git pull; FullReconcile — полная сверка; ManualRebuild — полная перестройка. SyncWorker MUST ограничивать частоту запросов к embedding API (rate limit: не более 1 batch/сек). SyncWorker MUST логировать warn при ошибках синхронизации.

#### Scenario: Триггер после перемещения ноды
- **WHEN** API handler успешно перемещает ноду из `old/path` в `new/path`
- **THEN** SyncWorker MUST получить событие SingleNode для старого пути `old/path` и удалить устаревшую запись из индекса
- **AND** SyncWorker MUST получить событие SingleNode для нового пути `new/path` и проиндексировать ноду по новому пути
