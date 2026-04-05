# booking-service

Сервис бронирования переговорок. Go + PostgreSQL + Docker Compose.

## Запуск

```bash
# Один раз после клонирования — сгенерировать go.sum
go mod tidy

# Если Go нет локально — через Docker
sh scripts/gen-gosum.sh

make up      # собрать и поднять
make seed    # залить тестовые данные (опционально)
```

Сервис на `http://localhost:8080`.

## Разработка

```bash
make test          # юнит + e2e (без БД)
make test-cover    # с отчётом покрытия
make lint          # golangci-lint
make down          # остановить
```

## Стек

- Go 1.22, [Gin](https://github.com/gin-gonic/gin), [sqlx](https://github.com/jmoiron/sqlx)
- PostgreSQL 16
- [golang-migrate](https://github.com/golang-migrate/migrate) — миграции embedded в бинарник
- [golang-jwt/v5](https://github.com/golang-jwt/jwt)

## Структура

```
cmd/server/        — точка входа
internal/
  config/          — конфиг из env
  db/              — подключение, embedded-миграции
  model/           — доменные типы
  repository/      — интерфейсы и PostgreSQL-реализации
  service/         — бизнес-логика
  handler/         — HTTP (Gin)
  middleware/      — JWT
tests/             — e2e через httptest
scripts/           — seed.sql
```

## API

| Метод | Путь | Роль |
|---|---|---|
| POST | `/dummyLogin` | — |
| POST | `/register` | — |
| POST | `/login` | — |
| GET  | `/_info` | — |
| POST | `/rooms/create` | admin |
| GET  | `/rooms/list` | admin, user |
| POST | `/rooms/{roomId}/schedule/create` | admin |
| GET  | `/rooms/{roomId}/slots/list?date=` | admin, user |
| POST | `/bookings/create` | user |
| GET  | `/bookings/list` | admin |
| GET  | `/bookings/my` | user |
| POST | `/bookings/{bookingId}/cancel` | user |

Токен получается через `/dummyLogin` с параметром `role: admin` или `role: user`.

Фиксированные UUID для тестовых пользователей:
- admin — `11111111-1111-1111-1111-111111111111`  
- user  — `22222222-2222-2222-2222-222222222222`

## Слоты

Слоты хранятся в БД с UUID — без этого нельзя бронировать по `slotId`. При создании расписания слоты генерируются сразу на 14 дней вперёд. Фоновая горутина раз в сутки продлевает горизонт. 99.9% запросов приходится на ближайшие 7 дней, 14 — с запасом.

## Защита от двойного бронирования

Частичный уникальный индекс на уровне БД:

```sql
CREATE UNIQUE INDEX idx_bookings_slot_active
  ON bookings(slot_id) WHERE status = 'active';
```

При гонке двух запросов один получает `23505`, который превращается в `409 SLOT_ALREADY_BOOKED`.

## Conference link

`createConferenceLink: true` в запросе на создание брони — опциональный параметр. Если внешний сервис недоступен, бронь создаётся без ссылки (`conferenceLink: null`), ошибка пишется в лог.

## Конфигурация

| Переменная | Значение по умолчанию |
|---|---|
| `DATABASE_URL` | `postgres://postgres:postgres@localhost:5432/booking?sslmode=disable` |
| `JWT_SECRET` | `super-secret-key-change-in-production` |
| `PORT` | `8080` |
