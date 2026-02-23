SHELL := /bin/bash

APP        ?= service-courier
SECRETS_PG ?= ./secrets/postgres_password.txt
SECRETS_PG_ABS := $(abspath $(SECRETS_PG))
MIGR_DIR   ?= db/migrations
COMPOSE    ?= docker compose
GOOSE      ?= goose

PKG            ?= ./...
COVERPKG_INTERNAL ?= $(shell go list ./internal/... | grep -v '/internal/proto' | paste -sd, -)


COVER_DIR      ?= .coverage
COVER_FILE     ?= $(COVER_DIR)/coverage.out
COVER_HTML     ?= $(COVER_DIR)/coverage.html
COVER_UNIT     ?= $(COVER_DIR)/coverage.unit.out
COVER_INTEG    ?= $(COVER_DIR)/coverage.int.out

INTEG_TAGS     ?= integration
INTEG_PKGS     ?= ./...

COMPOSE_PROJECT ?= $(notdir $(CURDIR))
PGDATA_VOL      ?= $(COMPOSE_PROJECT)_pgdata

ifneq (,$(wildcard .env))
include .env
export PORT POSTGRES_USER POSTGRES_DB POSTGRES_PORT POSTGRES_HOST \
	POSTGRES_PASSWORD POSTGRES_PASSWORD_FILE \
	GOOSE_DRIVER GOOSE_DBSTRING GOOSE_DBSTRING_HOST \
	TEST_DB_DSN
endif

PHONY_TARGETS := \
	help db-create db-reset migrate-up migrate-down run ping logs psql \
	test test-race test-integration \
	cover-dir cover-unit cover-integration cover-merge cover-html-all \
	cover cover-html open-coverage clean-cover

.PHONY: $(PHONY_TARGETS)

help:
	@echo "  db-create           - создать БД $$POSTGRES_DB в контейнере (если нет)"
	@echo "  db-reset            - снести pgdata volume и поднять postgres заново (нужно после смены секрета)"
	@echo "  migrate-up          - goose up (из .env)"
	@echo "  migrate-down        - goose down"
	@echo "  run                 - go run ./cmd/$(APP)"
	@echo "  ping                - curl /ping"
	@echo "  logs                - docker compose logs -f"
	@echo "  psql                - psql в контейнере"
	@echo "  test                - go test (без кэша)"
	@echo "  test-race           - go test -race"
	@echo "  test-integration    - go test -tags=integration (с хоста, через 127.0.0.1)"
	@echo "  cover-html-all      - общий HTML отчёт (unit + integration) → $(COVER_HTML)"
	@echo "  clean-cover         - удалить файлы покрытия"
	@echo "  (alias) cover-html  - == cover-html-all"
	@echo "  swagger             - swag init (генерация swagger из комментариев)"

db-create:
	@echo "→ Проверяю наличие БД '$$POSTGRES_DB'..."
	@PW="$$( \
		if [[ -n "$$POSTGRES_PASSWORD_FILE" && -f "$$POSTGRES_PASSWORD_FILE" ]]; then cat "$$POSTGRES_PASSWORD_FILE"; else echo "$$POSTGRES_PASSWORD"; fi \
	)"; \
	if $(COMPOSE) exec -T -e PGPASSWORD="$$PW" postgres \
	  psql -h localhost -U "$$POSTGRES_USER" -d postgres -tAc "SELECT 1 FROM pg_database WHERE datname='$$POSTGRES_DB'" | grep -q 1; then \
		echo "✓ Уже существует"; \
	else \
		echo "→ Создаю БД '$$POSTGRES_DB'"; \
		$(COMPOSE) exec -T -e PGPASSWORD="$$PW" postgres \
		  psql -h localhost -U "$$POSTGRES_USER" -d postgres -c "CREATE DATABASE $$POSTGRES_DB"; \
	fi

db-reset:
	@echo "→ reset pgdata volume: $(PGDATA_VOL)"
	@$(COMPOSE) down >/dev/null 2>&1 || true
	@docker volume rm -f "$(PGDATA_VOL)" >/dev/null 2>&1 || true
	@$(COMPOSE) up -d postgres

migrate-up:
	@test -d $(MIGR_DIR) || (echo "Нет каталога $(MIGR_DIR)"; exit 1)
	@PW="$$( \
		if [[ -n "$$POSTGRES_PASSWORD_FILE" && -f "$$POSTGRES_PASSWORD_FILE" ]]; then cat "$$POSTGRES_PASSWORD_FILE"; else echo "$$POSTGRES_PASSWORD"; fi \
	)"; \
	DSN="$${GOOSE_DBSTRING_HOST:-$${GOOSE_DBSTRING:-postgres://$${POSTGRES_USER}:$${PW}@127.0.0.1:$${POSTGRES_PORT:-5432}/$${POSTGRES_DB}?sslmode=disable}}"; \
	echo "→ goose up (host)"; \
	GOOSE_DBSTRING="$$DSN" $(GOOSE) -dir $(MIGR_DIR) up

migrate-down:
	@test -d $(MIGR_DIR) || (echo "Нет каталога $(MIGR_DIR)"; exit 1)
	@PW="$$( \
		if [[ -n "$$POSTGRES_PASSWORD_FILE" && -f "$$POSTGRES_PASSWORD_FILE" ]]; then cat "$$POSTGRES_PASSWORD_FILE"; else echo "$$POSTGRES_PASSWORD"; fi \
	)"; \
	DSN="$${GOOSE_DBSTRING_HOST:-$${GOOSE_DBSTRING:-postgres://$${POSTGRES_USER}:$${PW}@127.0.0.1:$${POSTGRES_PORT:-5432}/$${POSTGRES_DB}?sslmode=disable}}"; \
	echo "→ goose down (host)"; \
	GOOSE_DBSTRING="$$DSN" $(GOOSE) -dir $(MIGR_DIR) down

run:
	@bash -c 'trap "exit 0" INT; go run ./cmd/$(APP)'

ping:
	@curl -sS -i "http://127.0.0.1:$${PORT:-8080}/ping" || true

logs:
	@bash -c 'trap "exit 0" INT; $(COMPOSE) logs -f'

psql:
	@PW="$$( \
		if [[ -n "$$POSTGRES_PASSWORD_FILE" && -f "$$POSTGRES_PASSWORD_FILE" ]]; then cat "$$POSTGRES_PASSWORD_FILE"; else echo "$$POSTGRES_PASSWORD"; fi \
	)"; \
	$(COMPOSE) exec -T -e PGPASSWORD="$$PW" postgres psql -h localhost -U "$$POSTGRES_USER" -d "$$POSTGRES_DB"

test:
	@echo "→ go test $(PKG)"
	go test -count=1 $(PKG)

test-race:
	@echo "→ go test -race $(PKG)"
	go test -race -count=10 $(PKG)

test-integration:
	@echo "→ go test -tags=$(INTEG_TAGS) $(INTEG_PKGS)"
	POSTGRES_HOST=127.0.0.1 \
	POSTGRES_PASSWORD_FILE="$(SECRETS_PG_ABS)" \
	go test -tags=$(INTEG_TAGS) -count=1 $(INTEG_PKGS)

cover-dir:
	@mkdir -p $(COVER_DIR)

cover-unit: cover-dir
	@echo "→ unit coverage → $(COVER_UNIT)"
	env -u PORT -u POSTGRES_HOST -u POSTGRES_PORT -u POSTGRES_USER -u POSTGRES_PASSWORD -u POSTGRES_DB -u POSTGRES_PASSWORD_FILE \
	  go test -covermode=atomic -coverpkg="$(COVERPKG_INTERNAL)" -coverprofile=$(COVER_UNIT) -count=1 ./...
	@go tool cover -func=$(COVER_UNIT) | tail -n1

cover-integration: cover-dir
	@echo "→ integration coverage → $(COVER_INTEG)"
	POSTGRES_HOST=127.0.0.1 \
	POSTGRES_PASSWORD_FILE="$(SECRETS_PG_ABS)" \
	go test -tags=$(INTEG_TAGS) -covermode=atomic -coverpkg="$(COVERPKG_INTERNAL)" -coverprofile=$(COVER_INTEG) -count=1 $(INTEG_PKGS)
	@go tool cover -func=$(COVER_INTEG) | tail -n1

cover-merge: cover-unit cover-integration
	@echo "→ merge $(COVER_UNIT) + $(COVER_INTEG) → $(COVER_FILE)"
	@{ echo "mode: atomic"; \
	   tail -n +2 "$(COVER_UNIT)"; \
	   tail -n +2 "$(COVER_INTEG)"; } > "$(COVER_FILE)"
	@go tool cover -func=$(COVER_FILE) | tail -n1

cover-html-all: cover-merge
	@echo "→ генерирую HTML: $(COVER_HTML)"
	go tool cover -html=$(COVER_FILE) -o $(COVER_HTML)
	@echo "✓ готово: $(COVER_HTML)"

swagger:
	swag init -g main.go -d cmd/service-courier,internal --parseInternal

cover: cover-merge
cover-html: cover-html-all

open-coverage: cover-html-all
	@{ command -v xdg-open >/dev/null 2>&1 && nohup xdg-open "$(COVER_HTML)" >/dev/null 2>&1 < /dev/null & } || true
	@{ command -v open     >/dev/null 2>&1 && nohup open     "$(COVER_HTML)" >/dev/null 2>&1 < /dev/null & } || true

clean:
	@rm -rf $(COVER_DIR) || true
