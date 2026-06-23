Описание системы

Сервис предназначен для обработки заявок на проведение экспертизы.

Заявка создается клиентом через портал.

После подачи заявки она проходит несколько этапов согласования и экспертной проверки.

Архитектура

Requests Module является центральным сервисом.

Интеграции:

Questionnaire Module

Используется для:

получения шаблонов анкет
валидации полей заявки
Service Catalog Module

Используется для получения:

SLA
OLA
правил маршрутизации
условий оказания услуги
Locks Module

Используется для блокировки редактирования отдельных полей.

Auth Module

Используется для получения контекста сотрудника.

Notification Service

Используется для отправки уведомлений.

Основные сущности
REQUEST

Главная сущность заявки.

Поля:

id UUID
number STRING
status STRING

client_id UUID

application_id UUID
service_id UUID

service_catalog_version INT

template_id UUID
template_version INT

questionnaire_catalog_version INT

created_at TIMESTAMP
updated_at TIMESTAMP
submitted_at TIMESTAMP

cancelled_at TIMESTAMP
cancel_reason STRING

sla_deadline TIMESTAMP
sla_paused_at TIMESTAMP
sla_accumulated_hours DECIMAL

ola_l1_deadline TIMESTAMP
ola_ib_deadline TIMESTAMP

ttl_days INT

last_edited_at TIMESTAMP
archived_at TIMESTAMP
CONTRACT

Результат выполнения заявки.

id UUID
request_id UUID

service_type STRING
provision_terms STRING

result_description STRING

data_snapshot STRING

submitted_at TIMESTAMP

estimated_sla STRING
pdf_3_key STRING

created_at TIMESTAMP
FIELD_LOCK

Блокировка поля.

id UUID

application_id UUID
card_field_id UUID

locked_by_id UUID

request_id UUID

locked_at TIMESTAMP
EXPERT_COMMENT

Комментарии экспертов.

id UUID

request_id UUID

author_id UUID

field_code STRING

text STRING

type STRING

created_at TIMESTAMP
REQUEST_ASSIGNMENT

Назначение заявки эксперту.

id UUID

request_id UUID

expert_id UUID

role STRING

assigned_at TIMESTAMP
released_at TIMESTAMP
Дополнительные сущности
EMPLOYEE

Сотрудник, подающий заявку.

APPLICATION

Контекст приложения.

IB_SERVICE

Услуга из каталога услуг.

QUESTIONNAIRE_TEMPLATE

Шаблон анкеты.

SUB

Дополнительные материалы по заявке.

VERDICT

Экспертное заключение.

Жизненный цикл заявки

Начальное состояние:

DRAFT

Клиент создает черновик.

Подача заявки
DRAFT -> SUBMITTED

При подаче:

создается CONTRACT
запускается SLA
Автомаршрутизация
SUBMITTED -> L1_REVIEW

Запускается OLA L1.

Решение L1
Принятие
L1_REVIEW -> IB_REVIEW

Действия:

SLA продолжает идти
запускается OLA IB
Возврат на доработку
L1_REVIEW -> REWORK

Действия:

SLA ставится на паузу
OLA L1 останавливается
Отклонение
L1_REVIEW -> REJECTED

Действия:

SLA останавливается
OLA L1 останавливается
Работа ИБ эксперта
Взятие в работу
IB_REVIEW -> IB_IN_PROGRESS

Действия:

эксперт назначается на заявку
Запрос дополнительной информации
IB_IN_PROGRESS -> REWORK

Действия:

SLA на паузу
Возврат на L1
IB_IN_PROGRESS -> L1_REWORK_FROM_IB
Отклонение
IB_IN_PROGRESS -> REJECTED
Успешное завершение
IB_IN_PROGRESS -> VERDICT_FORMED

Действия:

формируется VERDICT
SLA останавливается
OLA IB останавливается
Повторное согласование
L1_REWORK_FROM_IB -> IB_REVIEW

После повторного принятия L1.

Клиентская доработка
REWORK -> L1_REVIEW

После внесения изменений клиентом.

SLA возобновляется.

Отмена

Из состояния DRAFT:

DRAFT -> CANCELLED
Архивация

Если заявка не редактируется 15 дней:

DRAFT -> ARCHIVED
Конкурентный доступ экспертов

L1 и IB эксперты видят общую очередь заявок.

Для захвата заявки используется таблица:

request_assignment

Алгоритм:

INSERT INTO request_assignment (...)
ON CONFLICT DO NOTHING

Если INSERT успешен:

заявка назначается эксперту

Если возник CONFLICT:

заявка уже взята другим экспертом

Необходимо подробно описать механизм защиты от гонок и конкурентного доступа.

SLA и OLA

Требуется подробно описать:

запуск SLA
остановку SLA
постановку SLA на паузу
возобновление SLA

Отдельно описать:

OLA L1
OLA IB

И правила их расчета.