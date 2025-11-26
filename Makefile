SHELL := /bin/bash

APP        ?= service-courier
MIGR_DIR   ?= db/migrations
COMPOSE    ?= docker compose
GOOSE      ?= goose

PKG            ?= ./...

COVER_DIR      ?= .coverage
COVER_FILE     ?= $(COVER_DIR)/coverage.out
COVER_HTML     ?= $(COVER_DIR)/coverage.html
COVER_UNIT     ?= $(COVER_DIR)/coverage.unit.out
COVER_INTEG    ?= $(COVER_DIR)/coverage.int.out

INTEG_TAGS     ?= integration
INTEG_PKGS     ?= ./...

ifneq (,$(wildcard .env))
include .env
export PORT POSTGRES_USER POSTGRES_PASSWORD POSTGRES_DB POSTGRES_PORT POSTGRES_HOST GOOSE_DRIVER GOOSE_DBSTRING
endif

INTEG_DSN ?= $(TEST_DB_DSN)
ifeq ($(strip $(INTEG_DSN)),)
INTEG_DSN := postgres://$(POSTGRES_USER):$(POSTGRES_PASSWORD)@$(if $(POSTGRES_HOST),$(POSTGRES_HOST),127.0.0.1):$(if $(POSTGRES_PORT),$(POSTGRES_PORT),5432)/$(POSTGRES_DB)?sslmode=disable
endif

PHONY_TARGETS := \
	help db-create migrate-up migrate-down run ping logs psql \
	test test-race \
	cover-dir cover-unit cover-integration cover-merge cover-all cover-html-all \
	cover cover-html open-coverage clean-cover

.PHONY: $(PHONY_TARGETS)

help:
	@echo "  db-create           - создать БД $$POSTGRES_DB в контейнере (если нет)"
	@echo "  migrate-up          - goose up (из .env)"
	@echo "  migrate-down        - goose down"
	@echo "  run                 - go run ./cmd/$(APP)"
	@echo "  ping                - curl /ping"
	@echo "  logs                - docker compose logs -f"
	@echo "  psql                - psql в контейнере"
	@echo "  test                - go test (без кэша)"
	@echo "  test-race           - go test -race"
	@echo "  cover-all           - ОБЩЕЕ покрытие (unit + integration) → $(COVER_FILE)"
	@echo "  cover-html-all      - общий HTML отчёт (unit + integration) → $(COVER_HTML)"
	@echo "  clean-cover         - удалить файлы покрытия"
	@echo "  (alias) cover       - == cover-all"
	@echo "  (alias) cover-html  - == cover-html-all"

db-create:
	@echo "→ Проверяю наличие БД '$$POSTGRES_DB'..."
	@docker exec -i my-postgres psql -U "$$POSTGRES_USER" -d postgres -tAc "SELECT 1 FROM pg_database WHERE datname='$$POSTGRES_DB'" | grep -q 1 \
	 && echo "✓ Уже существует" \
	 || (echo "→ Создаю БД '$$POSTGRES_DB'"; docker exec -i my-postgres psql -U "$$POSTGRES_USER" -d postgres -c "CREATE DATABASE $$POSTGRES_DB")

migrate-up:
	@test -d $(MIGR_DIR) || (echo "Нет каталога $(MIGR_DIR)"; exit 1)
	$(GOOSE) -dir $(MIGR_DIR) up

migrate-down:
	$(GOOSE) -dir $(MIGR_DIR) down

run:
	@bash -c 'trap "exit 0" INT; go run ./cmd/$(APP)'

ping:
	@curl -sS -i "http://127.0.0.1:$${PORT:-8080}/ping" || true

logs:
	@bash -c 'trap "exit 0" INT; $(COMPOSE) logs -f'

psql:
	docker exec -e PGPASSWORD="$$POSTGRES_PASSWORD" -it my-postgres psql -U "$$POSTGRES_USER" -d "$$POSTGRES_DB"

test:
	@echo "→ go test $(PKG)"
	go test -count=1 $(PKG)

test-race:
	@echo "→ go test -race $(PKG)"
	go test -race -count=1 $(PKG)

cover-dir:
	@mkdir -p $(COVER_DIR)

cover-unit: cover-dir
	@echo "→ unit coverage → $(COVER_UNIT)"
	env -u PORT -u POSTGRES_HOST -u POSTGRES_PORT -u POSTGRES_USER -u POSTGRES_PASSWORD -u POSTGRES_DB \
	  go test -covermode=atomic -coverpkg=./internal/... -coverprofile=$(COVER_UNIT) -count=1 ./...
	@go tool cover -func=$(COVER_UNIT) | tail -n1

cover-integration: cover-dir
	@echo "→ integration coverage → $(COVER_INTEG)"
	@test -n "$(INTEG_DSN)" || (echo "ERROR: set POSTGRES_* in .env or INTEG_DSN/TEST_DB_DSN"; exit 1)
	TEST_DB_DSN="$(INTEG_DSN)" \
	  go test -tags=$(INTEG_TAGS) -covermode=atomic -coverpkg=./internal/... -coverprofile=$(COVER_INTEG) -count=1 $(INTEG_PKGS)
	@go tool cover -func=$(COVER_INTEG) | tail -n1

cover-merge: cover-unit cover-integration
	@echo "→ merge $(COVER_UNIT) + $(COVER_INTEG) → $(COVER_FILE)"
	@{ echo "mode: atomic"; \
	   tail -n +2 "$(COVER_UNIT)"; \
	   tail -n +2 "$(COVER_INTEG)"; } > "$(COVER_FILE)"
	@go tool cover -func=$(COVER_FILE) | tail -n1

cover-all: cover-merge

cover-html-all: cover-merge
	@echo "→ генерирую HTML: $(COVER_HTML)"
	go tool cover -html=$(COVER_FILE) -o $(COVER_HTML)
	@echo "✓ готово: $(COVER_HTML)"

cover: cover-all

cover-html: cover-html-all

open-coverage: cover-html-all
	@which xdg-open >/dev/null 2>&1 && xdg-open $(COVER_HTML) || true
	@which open     >/dev/null 2>&1 && open $(COVER_HTML)     || true

clean-cover:
	@rm -rf $(COVER_DIR) || true
