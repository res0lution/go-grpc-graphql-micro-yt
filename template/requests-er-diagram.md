# ER-диаграмма Requests Module (текстовая расшифровка)

Ниже приведена расшифровка диаграммы с фото в текстовом виде.

## Таблицы и поля

### `REQUEST`

| Тип | Поле |
|---|---|
| `uuid` | `id` |
| `string` | `number` |
| `string` | `status` |
| `uuid` | `client_id` |
| `uuid` | `application_id` |
| `uuid` | `service_id` |
| `int` | `service_catalog_version` |
| `uuid` | `template_id` |
| `int` | `template_version` |
| `int` | `questionnaire_catalog_version` |
| `timestamp` | `created_at` |
| `timestamp` | `updated_at` |
| `timestamp` | `submitted_at` |
| `timestamp` | `cancelled_at` |
| `string` | `cancel_reason` |
| `timestamp` | `sla_deadline` |
| `timestamp` | `sla_paused_at` |
| `decimal` | `sla_accumulated_hours` |
| `timestamp` | `ola_l1_deadline` |
| `timestamp` | `ola_l6_deadline`* |
| `int` | `ttl_days` |
| `timestamp` | `last_edited_at` |
| `timestamp` | `archived_at` |

\* На фото это поле читается неидеально: может выглядеть как `ola_lb_deadline`.

### `CONTRACT`

| Тип | Поле |
|---|---|
| `uuid` | `id` |
| `uuid` | `request_id` |
| `string` | `service_type` |
| `string` | `provision_terms` |
| `string` | `result_description` |
| `string` | `data_snapshot` |
| `timestamp` | `submitted_at` |
| `string` | `estimated_sla` |
| `string` | `pdf_s3_key` |
| `timestamp` | `created_at` |

### `FIELD_LOCK`

| Тип | Поле |
|---|---|
| `uuid` | `id` |
| `uuid` | `application_id` |
| `uuid` | `card_field_id` |
| `uuid` | `locked_by_id` |
| `uuid` | `request_id` |
| `timestamp` | `locked_at` |

### `EXPERT_COMMENT`

| Тип | Поле |
|---|---|
| `uuid` | `id` |
| `uuid` | `request_id` |
| `uuid` | `author_id` |
| `string` | `field_code` |
| `string` | `text` |
| `string` | `type` |
| `timestamp` | `created_at` |

### `REQUEST_ASSIGNMENT`

| Тип | Поле |
|---|---|
| `uuid` | `id` |
| `uuid` | `request_id` |
| `uuid` | `expert_id` |
| `string` | `role` |
| `timestamp` | `assigned_at` |
| `timestamp` | `released_at` |

### `BUG`

На фрагменте диаграммы видна сущность `BUG`, но состав полей на фото не читается.

### Внешние сущности (без полей на фото)

- `EMPLOYEE`
- `APPLICATION`
- `IB_SERVICE`
- `QUESTIONNAIRE_TEMPLATE`

## Связи (как подписано на диаграмме)

- `EMPLOYEE` -> `REQUEST`: "подает"
- `APPLICATION` -> `REQUEST`: "контекст"
- `IB_SERVICE` -> `REQUEST`: "по услуге"
- `QUESTIONNAIRE_TEMPLATE` -> `REQUEST`: "по темплейту"
- `REQUEST` -> `CONTRACT`: "фиксирует"
- `REQUEST` -> `FIELD_LOCK`: "блокирует"
- `REQUEST` -> `EXPERT_COMMENT`: "замечания"
- `REQUEST` -> `REQUEST_ASSIGNMENT`: "назначен эксперт"
- `REQUEST` -> `BUG`: "содержит"

