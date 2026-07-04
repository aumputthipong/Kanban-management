# Top-level shortcuts. Run `make` (no target) to see this list.
#
# Targets are split between things that operate on backend/, frontend/, or
# both. They're thin wrappers — open the recipe to see the underlying command.

SHELL := bash

# Default target — print help.
.DEFAULT_GOAL := help

# ─── Discoverability ──────────────────────────────────────────────────────────
.PHONY: help
help: ## List the available targets
	@awk 'BEGIN {FS = ":.*?## "; printf "\nUsage: make \033[36m<target>\033[0m\n\n"} \
	     /^[a-zA-Z_-]+:.*?## / {printf "  \033[36m%-22s\033[0m %s\n", $$1, $$2}' $(MAKEFILE_LIST)
	@echo ""

# ─── Dev environment ──────────────────────────────────────────────────────────
.PHONY: dev
dev: ## Start the full dev stack (Postgres in docker + backend + frontend)
	@echo "→ starting Postgres..."
	docker compose up -d
	@echo "→ run 'make backend' and 'make frontend' in two shells"

.PHONY: dev-up
dev-up: ## Bring full prod-shaped stack up via docker-compose.prod.yml
	docker compose -f docker-compose.prod.yml up --build

.PHONY: dev-down
dev-down: ## Stop everything started by docker compose
	docker compose down
	docker compose -f docker-compose.prod.yml down

.PHONY: backend
backend: ## Run the backend with live env (needs ./backend/.env)
	cd backend && go run ./cmd/api

.PHONY: frontend
frontend: ## Run the Next.js dev server
	cd frontend && npm run dev

.PHONY: seed
seed: ## Seed demo data (demo account + sample board). Needs DB_URL (or backend/.env)
	cd backend && go run ./cmd/seed

# ─── Verification (mirrors CI) ───────────────────────────────────────────────
.PHONY: verify
verify: vet test typecheck test-fe ## Run everything CI runs (vet + test + tsc + vitest)

.PHONY: vet
vet: ## go vet ./...
	cd backend && go vet ./...

.PHONY: test
test: ## Run backend tests with -race (skips integration tests)
	cd backend && go test -race ./...

.PHONY: test-integration
test-integration: ## Run integration tests (real Postgres via testcontainers, needs Docker)
	cd backend && go test -race -tags=integration ./...

.PHONY: test-cover
test-cover: ## Backend tests + write coverage.out
	cd backend && go test -race -coverprofile=coverage.out ./...
	cd backend && go tool cover -func=coverage.out | tail -1

.PHONY: typecheck
typecheck: ## tsc --noEmit
	cd frontend && npx tsc --noEmit

.PHONY: test-fe
test-fe: ## Frontend unit tests (vitest)
	cd frontend && npm test -- --run

.PHONY: lint-fe
lint-fe: ## ESLint
	cd frontend && npm run lint

# ─── Database / migrations ────────────────────────────────────────────────────
.PHONY: migrate-new
migrate-new: ## Stub a new migration: make migrate-new name=add_foo
	@if [ -z "$(name)" ]; then echo "usage: make migrate-new name=<short_name>"; exit 1; fi
	@next=$$(ls backend/database/migrations | sed 's/_.*//' | sort -u | tail -1 | awk '{printf "%06d\n", $$0+1}'); \
	 touch "backend/database/migrations/$${next}_$(name).up.sql"; \
	 touch "backend/database/migrations/$${next}_$(name).down.sql"; \
	 echo "created backend/database/migrations/$${next}_$(name).{up,down}.sql"

# ─── OpenAPI / sqlc ──────────────────────────────────────────────────────────
.PHONY: swag
swag: ## Regenerate the OpenAPI spec (backend/docs/)
	cd backend && go generate ./cmd/api

.PHONY: sqlc
sqlc: ## Regenerate sqlc code from queries.sql
	cd backend && sqlc generate

# ─── Build ────────────────────────────────────────────────────────────────────
.PHONY: build
build: build-backend build-frontend ## Build both Docker images

.PHONY: build-backend
build-backend: ## Build backend Docker image
	docker build -t turtask-backend:dev ./backend

.PHONY: build-frontend
build-frontend: ## Build frontend Docker image
	docker build -t turtask-frontend:dev ./frontend

# ─── Maintenance ──────────────────────────────────────────────────────────────
.PHONY: tidy
tidy: ## go mod tidy
	cd backend && go mod tidy

.PHONY: fmt
fmt: ## gofmt + npm format if configured
	cd backend && gofmt -w .

.PHONY: clean
clean: ## Remove local build artifacts
	rm -rf backend/coverage.out
	rm -rf frontend/.next frontend/coverage frontend/test-results frontend/playwright-report
