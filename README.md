# Gofermart — Накопительная система лояльности

HTTP API сервис для учёта баллов лояльности пользователей интернет-магазина «Гофермарт».

## Стек технологий

| Технология | Версия | Назначение |
|---|---|---|
| Go | 1.26 | Язык программирования |
| chi/v5 | v5.3.1 | HTTP роутер |
| pgx/v5 | v5.10.0 | Драйвер PostgreSQL |
| goose/v3 | v3.27.3 | Миграции БД |
| zap | v1.28.0 | Структурированное логирование |
| easyjson | v0.9.2 | Быстрая JSON сериализация |
| jwt/v5 | v5.2.1 | JWT токены |
| bcrypt | latest | Хеширование паролей |
| testify | v1.11.1 | Юнит-тестирование |

## Требования

- Go 1.26+
- PostgreSQL 12+

## Установка

```bash
git clone https://github.com/MartsinovichDanya/pp_gophermart.git
cd pp_gophermart
go mod download
```

## Запуск

### Через переменные окружения

```bash
RUN_ADDRESS="localhost:8080" \
DATABASE_URI="postgresql://postgres:secret@localhost:5432/gophermart?sslmode=disable" \
ACCRUAL_SYSTEM_ADDRESS="http://localhost:8081" \
go run cmd/gophermart/main.go
```

### Через флаги командной строки

```bash
go run cmd/gophermart/main.go \
  -a localhost:8080 \
  -d "postgresql://postgres:secret@localhost:5432/gophermart?sslmode=disable" \
  -r "http://localhost:8081"
```

## Конфигурация

| Переменная окружения | Флаг | Описание | По умолчанию |
|---|---|---|---|
| `RUN_ADDRESS` | `-a` | Адрес и порт HTTP сервера | `localhost:8080` |
| `DATABASE_URI` | `-d` | Строка подключения к PostgreSQL | — |
| `ACCRUAL_SYSTEM_ADDRESS` | `-r` | Адрес системы расчёта начислений | — |
| `JWT_SECRET` | — | Секретный ключ для JWT токенов | `supersecretkey` |
| `LOG_LEVEL` | — | Уровень логирования (DEBUG, INFO, WARN, ERROR) | `DEBUG` |
| `MAX_BODY_SIZE` | — | Максимальный размер тела запроса (байты) | `2048` |
| `PROCESSING_INTERVAL` | — | Интервал обработки заказов воркером | `5s` |

> **Приоритет:** переменные окружения > флаги командной строки > значения по умолчанию

## API

### POST /api/user/register — Регистрация

**Запрос:**
```http
POST /api/user/register HTTP/1.1
Content-Type: application/json

{
    "login": "username",
    "password": "password123"
}
```

**Коды ответа:**
| Код | Описание |
|---|---|
| 200 | Успешная регистрация и аутентификация |
| 400 | Неверный формат запроса |
| 409 | Логин уже занят |
| 500 | Внутренняя ошибка сервера |

---

### POST /api/user/login — Аутентификация

**Запрос:**
```http
POST /api/user/login HTTP/1.1
Content-Type: application/json

{
    "login": "username",
    "password": "password123"
}
```

**Коды ответа:**
| Код | Описание |
|---|---|
| 200 | Успешная аутентификация |
| 400 | Неверный формат запроса |
| 401 | Неверная пара логин/пароль |
| 500 | Внутренняя ошибка сервера |

---

### POST /api/user/orders — Загрузка номера заказа

**Запрос:**
```http
POST /api/user/orders HTTP/1.1
Content-Type: text/plain

12345678903
```

**Коды ответа:**
| Код | Описание |
|---|---|
| 200 | Номер уже загружен этим пользователем |
| 202 | Новый номер принят в обработку |
| 400 | Неверный формат запроса |
| 401 | Пользователь не аутентифицирован |
| 409 | Номер уже загружен другим пользователем |
| 422 | Неверный формат номера (алгоритм Луна) |
| 500 | Внутренняя ошибка сервера |

---

### GET /api/user/orders — Список заказов

**Запрос:**
```http
GET /api/user/orders HTTP/1.1
```

**Ответ (200):**
```json
[
    {
        "number": "9278923470",
        "status": "PROCESSED",
        "accrual": 500,
        "uploaded_at": "2020-12-10T15:15:45+03:00"
    },
    {
        "number": "12345678903",
        "status": "PROCESSING",
        "uploaded_at": "2020-12-10T15:12:01+03:00"
    }
]
```

**Статусы заказов:**
| Статус | Описание |
|---|---|
| NEW | Заказ загружен, но не в обработке |
| PROCESSING | Вознаграждение рассчитывается |
| INVALID | Система расчёта отказала |
| PROCESSED | Расчёт успешно получен |

**Коды ответа:**
| Код | Описание |
|---|---|
| 200 | Успешный ответ |
| 204 | Нет данных |
| 401 | Пользователь не авторизован |
| 500 | Внутренняя ошибка сервера |

---

### GET /api/user/balance — Баланс

**Запрос:**
```http
GET /api/user/balance HTTP/1.1
```

**Ответ (200):**
```json
{
    "current": 500.5,
    "withdrawn": 42
}
```

**Коды ответа:**
| Код | Описание |
|---|---|
| 200 | Успешный ответ |
| 401 | Пользователь не авторизован |
| 500 | Внутренняя ошибка сервера |

---

### POST /api/user/balance/withdraw — Списание баллов

**Запрос:**
```http
POST /api/user/balance/withdraw HTTP/1.1
Content-Type: application/json

{
    "order": "2377225624",
    "sum": 751
}
```

**Коды ответа:**
| Код | Описание |
|---|---|
| 200 | Успешное списание |
| 401 | Пользователь не авторизован |
| 402 | Недостаточно средств |
| 422 | Неверный номер заказа |
| 500 | Внутренняя ошибка сервера |

---

### GET /api/user/withdrawals — История выводов

**Запрос:**
```http
GET /api/user/withdrawals HTTP/1.1
```

**Ответ (200):**
```json
[
    {
        "order": "2377225624",
        "sum": 500,
        "processed_at": "2020-12-09T16:09:57+03:00"
    }
]
```

**Коды ответа:**
| Код | Описание |
|---|---|
| 200 | Успешный ответ |
| 204 | Нет выводов |
| 401 | Пользователь не авторизован |
| 500 | Внутренняя ошибка сервера |

## Аутентификация

JWT токен передаётся через:
- **Cookie** `token` (приоритет)
- **Заголовок** `Authorization: Bearer <token>`

После регистрации или логина сервер автоматически устанавливает cookie `token`.

## Архитектура

```
cmd/gophermart/          — точка входа
internal/
├── accrual/             — клиент системы расчёта баллов
├── auth/                — JWT аутентификация и middleware
├── config/              — конфигурация приложения
├── handler/             — HTTP обработчики
├── logger/              — логирование (zap)
├── middleware/          — gzip middleware
├── model/               — модели данных
├── server/              — инициализация сервера
├── storage/             — хранилище (PostgreSQL)
│   └── migrations/      — SQL миграции
├── utils/               — утилиты (алгоритм Луна, генерация ID)
└── worker/              — фоновый воркер обработки заказов
```

## Структура базы данных

```sql
service_data.users
├── id            UUID PRIMARY KEY
├── login         VARCHAR(255) UNIQUE
├── password_hash TEXT
├── balance       NUMERIC(10, 2)
├── withdrawn     NUMERIC(10, 2)
└── created_at    TIMESTAMP WITH TIME ZONE

service_data.orders
├── id            UUID PRIMARY KEY
├── user_id       UUID (FK → users)
├── number        VARCHAR(255) UNIQUE
├── status        VARCHAR(20)
├── accrual       NUMERIC(10, 2)
└── uploaded_at   TIMESTAMP WITH TIME ZONE

service_data.withdrawals
├── id            UUID PRIMARY KEY
├── user_id       UUID (FK → users)
├── order_number  VARCHAR(255)
├── sum           NUMERIC(10, 2)
└── processed_at  TIMESTAMP WITH TIME ZONE
```

## Тестирование

```bash
# Все тесты
go test ./... -v

# С покрытием
go test ./... -cover

# HTML отчёт
go test ./... -coverprofile=coverage.out
go tool cover -html=coverage.out -o coverage.html
```
