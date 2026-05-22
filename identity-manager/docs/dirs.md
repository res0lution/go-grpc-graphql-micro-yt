Пример:

identity-manager/
├── cmd/
│   └── identity-manager/
│       └── main.go
├── internal/
│   ├── handler/
│   │   ├── auth_handler.go
│   │   ├── user_handler.go
│   │   ├── session_handler.go
│   │   └── health_handler.go
│   ├── service/
│   │   ├── auth_service.go
│   │   ├── user_service.go
│   │   ├── session_service.go
│   │   └── interfaces.go
│   ├── repository/
│   │   ├── user_repository.go
│   │   ├── session_repository.go
│   │   └── interfaces.go
│   ├── model/
│   │   ├── auth.go
│   │   ├── user.go
│   │   ├── session.go
│   │   └── api_error.go
│   ├── middleware/
│   │   ├── auth.go
│   │   ├── request_id.go
│   │   └── logging.go
│   ├── client/
│   │   ├── idp_client.go
│   │   └── core_client.go
│   ├── config/
│   │   └── config.go
│   ├── logger/
│   │   └── logger.go
│   └── db/
│       ├── postgres.go
│       └── migrations/
├── docs/
├── go.mod
└── Makefile
Как разложить ответственность:

handler — HTTP, bind/validate, коды ответов.
service — бизнес-логика auth/user/session.
repository — SQL и работа с БД.
client — интеграции с IdP и core backend.
model — DTO + доменные сущности.
Чтобы не превратить в “монолит 2.0”, добавьте 3 правила:

handler общается только с service.
service не знает про gin/http и SQL.
repository не знает про внешние API (IdP/core).