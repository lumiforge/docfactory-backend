# Структура базы данных для SaaS сервиса генерации гарантийных талонов и инструкций

Используется Yandex YDB с поддержкой multi-tenant и ACID транзакций. Таблицы проектируются с учётом изоляции данных клиентов.

***

## Основные таблицы

### 1. Tenants (Клиенты)

| Поле           | Тип           | Описание                      |
|----------------|---------------|-------------------------------|
| tenant_id      | UUID (PK)     | Уникальный идентификатор клиента |
| name           | STRING        | Название компании             |
| subscription   | STRING        | Тарифный план                 |
| created_at     | TIMESTAMP     | Дата создания записи          |
| updated_at     | TIMESTAMP     | Дата последнего обновления    |
| is_active      | BOOL          | Активен ли клиент             |

***

### 2. Users (Пользователи)

| Поле           | Тип           | Описание                                      |
|----------------|---------------|-----------------------------------------------|
| user_id        | UUID (PK)     | Уникальный идентификатор                      |
| tenant_id      | UUID (FK)     | Связь с клиентом (Tenant)                      |
| email          | STRING        | Email пользователя                             |
| password_hash  | STRING        | Хэш пароля                                    |
| full_name      | STRING        | Полное имя                                    |
| role           | ENUM          | Роль (owner, admin, editor, viewer)           |
| created_at     | TIMESTAMP     | Дата регистрации                              |
| last_login     | TIMESTAMP     | Последний вход                                |
| is_active      | BOOL          | Статус активности                             |

***

### 3. Templates (Шаблоны документов)

| Поле            | Тип           | Описание                                 |
|-----------------|---------------|------------------------------------------|
| template_id     | UUID (PK)     | Уникальный идентификатор шаблона         |
| tenant_id       | UUID (FK)     | Связь с клиентом                          |
| name            | STRING        | Название шаблона (3-100 символов)        |
| description     | STRING        | Описание (опционально, до 500 символов)  |
| document_type   | ENUM          | Тип документа (warranty, instruction, certificate, label) |
| page_size       | ENUM          | Формат страницы (A4, A5, Letter)          |
| orientation     | ENUM          | Портрет или ландшафт                      |
| json_schema_url | STRING        | Ссылка на JSON-схему в Object Storage     |
| thumbnail_url   | STRING        | URL превью шаблона                        |
| version         | INT           | Текущая версия шаблона                    |
| created_by      | UUID (FK)     | Пользователь, создавший шаблон            |
| updated_by      | UUID (FK)     | Пользователь, обновивший шаблон           |
| created_at      | TIMESTAMP     | Дата создания                            |
| updated_at      | TIMESTAMP     | Дата обновления                          |
| deleted_at      | TIMESTAMP     | Удаление (soft delete), null если активно |
| documents_count | INT           | Количество созданных документов          |
| last_used_at    | TIMESTAMP     | Дата последнего использования             |

***

### 4. TemplateVersions (Версии шаблона)

| Поле            | Тип           | Описание                                   |
|-----------------|---------------|--------------------------------------------|
| version_id      | UUID (PK)     | Уникальный ID версии                      |
| template_id     | UUID (FK)     | Ссылка на основной шаблон                   |
| version_number  | INT           | Номер версии                               |
| change_summary  | STRING        | Краткое описание изменений (опционально)  |
| json_schema_url | STRING        | Ссылка на JSON-схему версии                 |
| created_by      | UUID (FK)     | Кто создал версию                          |
| created_at      | TIMESTAMP     | Дата создания версии                       |
| is_current      | BOOL          | Флаг текущей версии                        |

***

### 5. Documents (Сгенерированные документы)

| Поле            | Тип           | Описание                                  |
|-----------------|---------------|-------------------------------------------|
| document_id     | UUID (PK)     | Уникальный ID документа                   |
| tenant_id       | UUID (FK)     | Клиент                                   |
| template_id     | UUID (FK)     | Шаблон, по которому создан документ       |
| generated_files | JSON          | URL файлов (PDF, DOCX, HTML)               |
| metadata        | JSON          | Заполненные данные документа              |
| created_by      | UUID (FK)     | Кто создал документ                        |
| created_at      | TIMESTAMP     | Дата создания                            |

***

### 6. Assets (Изображения, Логотипы, Водяные знаки)

| Поле           | Тип           | Описание                                 |
|----------------|---------------|------------------------------------------|
| asset_id       | UUID (PK)     | Уникальный идентификатор ассета           |
| tenant_id      | UUID (FK)     | Контекст клиента                         |
| template_id    | UUID (FK)     | Принадлежность (если нужно)              |
| type           | ENUM          | Тип ассета (logo, image, watermark)      |
| file_name      | STRING        | Имя файла                               |
| storage_url    | STRING        | Ссылка в Object Storage                  |
| mime_type      | STRING        | Тип файла                               |
| size           | INT           | Размер в байтах                         |
| uploaded_by    | UUID (FK)     | Кто загрузил                           |
| uploaded_at    | TIMESTAMP     | Дата загрузки                         |

***

### 7. AuditLogs (Журналы операций)

| Поле          | Тип           | Описание                               |
|---------------|---------------|----------------------------------------|
| audit_id      | UUID (PK)     | ID записи                             |
| tenant_id     | UUID (FK)     | Клиент                               |
| user_id       | UUID (FK)     | Пользователь                         |
| entity_type   | STRING        | Тип: template, document, asset и т.д. |
| entity_id     | UUID          | ID сущности                         |
| action        | STRING        | create, update, delete и др.          |
| timestamp    | TIMESTAMP     | Время события                        |
| details      | JSON          | Дополнительная информация            |

***

## Индексы

- По tenant_id и composite index на (tenant_id, template_id, deleted_at) для быстрого поиска активных объектов
- Индексы по created_at и updated_at для сортировки
- Полнотекстовый индекс по name и description для поиска

***

## Пример запросов

```sql
-- Получить активные шаблоны для клиента с пагинацией
SELECT * FROM templates
WHERE tenant_id = @tenant_id AND deleted_at IS NULL
ORDER BY updated_at DESC
LIMIT @limit OFFSET @offset;

-- Получить историю версий для шаблона
SELECT * FROM template_versions
WHERE template_id = @template_id
ORDER BY version_number DESC;

-- Soft delete шаблона (установить deleted_at)
UPDATE templates
SET deleted_at = CURRENT_TIMESTAMP
WHERE template_id = @template_id;

-- Восстановить шаблон (удалить deleted_at)
UPDATE templates
SET deleted_at = NULL
WHERE template_id = @template_id;
```

***

## Особенности

- Все даты и время в ISO 8601 формате в UTC
- UUID для всех сущностей для уникальности и масштабируемости
- Soft delete для возможности восстановления
- Multi-tenant изоляция через tenant_id во всех таблицах и запросах

***

Эта структура обеспечит высокую производительность, масштабируемость и удобство разработки с прозрачной изоляцией данных разных клиентов.