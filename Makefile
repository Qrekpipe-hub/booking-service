.PHONY: up down seed test test-cover lint build deps tidy

## up: запустить сервис вместе со всеми зависимостями
up:
	docker compose up --build -d
	@echo "Service running at http://localhost:8080"

## down: остановить и удалить контейнеры и тома
down:
	docker compose down -v

## build: собрать бинарник локально (требует go и go.sum)
build:
	go build -o ./bin/server ./cmd/server

## deps: сгенерировать go.sum и скачать зависимости (нужно запустить один раз после клонирования)
deps:
	go mod tidy
	go mod download

## seed: наполнить БД тестовыми данными
seed:
	@echo "Seeding database..."
	@docker compose exec -T postgres psql -U booking -d booking -f - < scripts/seed.sql
	@echo "Seed complete. Restart app to generate slots: docker compose restart app"

## test: запустить все тесты (без БД, через in-memory моки)
test:
	go test ./... -v -count=1

## test-cover: тесты с HTML-отчётом о покрытии
test-cover:
	go test ./... -coverprofile=coverage.out -covermode=atomic
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report: coverage.html"
	@go tool cover -func=coverage.out | tail -1

## lint: запустить golangci-lint
lint:
	golangci-lint run ./...

## tidy: привести go.mod/go.sum в порядок
tidy:
	go mod tidy
