## ADDED Requirements

### Requirement: Runtime-поддержка Cursor Agent в контейнерном запуске
Система SHALL в контейнерном runtime включать установленный Cursor Agent (через установочный сценарий `curl https://cursor.com/install -fsS | bash` на этапе сборки образа) для поддержки серверной нормализации узлов. Документация и конфигурация запуска MUST явно учитывать эту зависимость.

#### Scenario: Сборка docker-образа с Cursor Agent
- **WHEN** выполняется сборка официального Docker-образа приложения
- **THEN** образ содержит установленный Cursor Agent, доступный серверному процессу во время выполнения

### Requirement: Конфигурация CURSOR_API_KEY
Серверный runtime MUST поддерживать переменную окружения `CURSOR_API_KEY` для аутентификации вызовов Cursor Agent в сценарии нормализации узлов.

#### Scenario: Запуск с настроенным CURSOR_API_KEY
- **WHEN** `CURSOR_API_KEY` передан в окружение процесса
- **THEN** операции нормализации через Cursor Agent могут выполняться в штатном режиме

#### Scenario: Запуск без CURSOR_API_KEY
- **WHEN** переменная `CURSOR_API_KEY` отсутствует
- **THEN** endpoint нормализации возвращает диагностируемую ошибку конфигурации
