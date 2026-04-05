# Room Booking Service

Сервис бронирования переговорок на Go + PostgreSQL + Docker Compose.

## Быстрый старт

**Шаг 1 — сгенерировать `go.sum` (один раз после клонирования):**

```bash
# Если установлен Go локально:
make deps

# Если Go не установлен — через Docker:
sh scripts/gen-gosum.sh
```

**Шаг 2 — запустить:**

```bash
make up       # собирает образ и поднимает PostgreSQL + сервис
make seed     # наполняет БД тестовыми данными (после make up)
```

Сервис доступен на `http://localhost:8080`.

```bash
make test         # все тесты (без БД, in-memory моки)
make test-cover   # тесты + HTML-отчёт о покрытии
make lint         # запустить golangci-lint
make down         # остановить и удалить контейнеры
```

> **Примечание:** `go.sum` не коммитится в репозиторий — он генерируется один раз командой выше.
> При сборке через Docker (`make up`) зависимости скачиваются автоматически внутри контейнера.

---

## Стек

| Компонент | Выбор | Обоснование |
|---|---|---|
| Язык | Go 1.22 | Предпочтительный по заданию |
| HTTP фреймворк | Gin | Наиболее распространён, быстрый маршрутизатор |
| БД | PostgreSQL 16 | Рекомендован по заданию |
| Драйвер | `sqlx` + `lib/pq` | Идиоматично, без ORM |
| Миграции | `golang-migrate` | Embedded SQL-файлы, без внешних CLI |
| JWT | `golang-jwt/v5` | Стандартная библиотека |

---

## Архитектурные решения

### Q: Как генерируются слоты?

**Подход: скользящее окно 14 дней + фоновый расширитель.**

Слоты не вычисляются на лету при каждом запросе — они хранятся в таблице `slots` с UUID-идентификаторами. Это необходимо, потому что бронирование идёт по `slotId`, который должен быть стабильным.

При создании расписания слоты генерируются немедленно на 14 дней вперёд (горутина, не блокирует ответ). Ежесуточно фоновая горутина (`ExtendAll`) проверяет, не нужно ли раздвинуть горизонт, и при необходимости дополняет таблицу.

**Почему не генерировать при запросе (`/slots/list`)?**
Тогда слоты не имели бы стабильного UUID до момента первого запроса, и между `GET /slots` и `POST /bookings/create` мог бы возникнуть race condition — клиент посмотрел слот, но ID ещё не существует в БД. Хранение в БД снимает эту проблему.

**Почему 14 дней?**
Задание указывает, что 99.9% запросов — в пределах ближайших 7 дней. Двойной запас делает окно комфортным и не создаёт значимой нагрузки (≤ 50 комнат × 14 дней × ~32 слота/день = ~22 400 строк максимум).

### Q: Как защищено от двойного бронирования?

Двойной барьер:
1. **Уникальный частичный индекс** в PostgreSQL:
   ```sql
   CREATE UNIQUE INDEX idx_bookings_slot_active ON bookings(slot_id) WHERE status = 'active';
   ```
   Только одна строка с `status = 'active'` на один `slot_id`. При конкурентной вставке одна транзакция получит ошибку `23505` (unique violation), которую репозиторий преобразует в `ErrSlotAlreadyBooked`.

2. **Прикладная проверка** в `BookingService.Create`: хотя основная защита в БД, сервис явно проверяет статус слота (существует ли, не в прошлом ли).

### Q: Как работает daysOfWeek?

API использует 1=Пн, 7=Вс (ISO 8601). Go использует `time.Weekday`, где 0=Sun, 1=Mon, ..., 6=Sat.

Конверсия: `goWeekday = time.Weekday(apiDay % 7)`.  
Проверка: `7 % 7 = 0 = Sunday` ✓, `1 % 7 = 1 = Monday` ✓, `6 % 7 = 6 = Saturday` ✓.

Хранится в PostgreSQL как `integer[]` с API-значениями (1–7). Индекс в БД не зависит от этой конверсии.

### Q: Как реализована отмена брони (идемпотентность)?

`POST /bookings/{bookingId}/cancel` — `POST`, а не `DELETE` (по спецификации API). Операция идемпотентна: если бронь уже отменена — возвращается `200` с текущим состоянием без ошибки. Логика в `BookingService.Cancel`:

```go
if booking.Status == model.BookingStatusCancelled {
    return booking, nil // уже отменена — возвращаем как есть
}
```

### Q: Что происходит при сбое Conference Service?

Получение ссылки на конференцию — **best-effort**: сбой не блокирует и не откатывает создание брони.

Смоделированные сценарии сбоя в `MockConferenceService`:
- **Случайная недоступность** (10%): возвращается `error`, бронь создаётся с `conferenceLink = null`.
- **Ошибка после успешного ответа** (не моделируется явно): в реальной системе потребовался бы компенсирующий запрос к Conference Service — но это вне скоупа данного задания.

Решение обосновано: бронь — первичный ресурс, ссылка — вспомогательная. Пользователь получает бронь гарантированно; ссылку — если сервис доступен.

### Q: Почему миграции embedded (embed.FS), а не через CLI?

Это позволяет запускать миграции автоматически при старте сервиса без отдельной точки входа или initContainer в Docker Compose. Конфигурация проще, развёртывание атомарнее.

### Q: Как организованы тесты?

- **Юнит-тесты** (`internal/service/*_test.go`): тестируют бизнес-логику через интерфейсы репозиториев; все зависимости — in-memory моки. Покрывают: создание/отмену брони, идемпотентность, защиту от двойного бронирования, генерацию слотов, конверсию daysOfWeek, auth/JWT.

- **E2E-тесты** (`tests/e2e_test.go`): поднимают полный HTTP-сервер через `httptest.NewServer` с in-memory репозиториями, без реальной БД. Тестируют два обязательных сценария:
  1. Создание комнаты → расписания → бронирования
  2. Отмена брони (включая повторную отмену)

---

## API

| Метод | Путь | Роль |
|---|---|---|
| POST | `/dummyLogin` | — |
| POST | `/register` | — (доп. задание) |
| POST | `/login` | — (доп. задание) |
| GET | `/_info` | — |
| POST | `/rooms/create` | admin |
| GET | `/rooms/list` | admin, user |
| POST | `/rooms/{roomId}/schedule/create` | admin |
| GET | `/rooms/{roomId}/slots/list?date=` | admin, user |
| POST | `/bookings/create` | user |
| GET | `/bookings/list` | admin |
| GET | `/bookings/my` | user |
| POST | `/bookings/{bookingId}/cancel` | user |

### dummyLogin

```bash
curl -s -X POST http://localhost:8080/dummyLogin \
  -H 'Content-Type: application/json' \
  -d '{"role":"admin"}' | jq .token
```

Фиксированные UUID:
- admin: `11111111-1111-1111-1111-111111111111`
- user:  `22222222-2222-2222-2222-222222222222`

---

## Схема БД

```
users          rooms          schedules            slots             bookings
──────         ──────         ──────────           ─────             ────────
id (PK)        id (PK)        id (PK)              id (PK)           id (PK)
email          name           room_id (FK)         room_id (FK)      user_id (FK)
password_hash  description    days_of_week[]       schedule_id (FK)  slot_id (FK)
role           capacity       start_time           start_at          status
created_at     created_at     end_time             end_at            conference_link
                              created_at           created_at        created_at
```

Ключевые индексы:
- `idx_slots_room_start ON slots(room_id, start_at)` — горячий эндпоинт списка слотов
- `idx_bookings_slot_active ON bookings(slot_id) WHERE status = 'active'` — уникальность активных броней
- `uq_schedules_room UNIQUE (room_id)` — одно расписание на комнату

---

## Конфигурация

| Переменная | По умолчанию | Описание |
|---|---|---|
| `DATABASE_URL` | `postgres://postgres:postgres@localhost:5432/booking?sslmode=disable` | DSN PostgreSQL |
| `JWT_SECRET` | `super-secret-key-change-in-production` | Секрет для подписи JWT |
| `PORT` | `8080` | Порт HTTP-сервера |
