# Standard targets shared by every app (see the root CLAUDE.md, §4).
# Variables come from .env, created from .env.example on first use.

-include .env

IMAGE ?= ghcr.io/zazi2004/linktree
TAG   ?= $(shell git rev-parse --short HEAD)
# Go packages of the app (`go list` skips web/node_modules thanks to go.mod's ignore).
GO_DIRS = $(shell go list -f '{{.Dir}}' ./...)

.PHONY: up down dev test lint fmt build migrate hash-password dist-placeholder

.env:
	cp .env.example .env

web/node_modules: web/package.json web/package-lock.json
	cd web && npm ci
	@touch web/node_modules

# `go build` embeds web/dist: create it when the front has not been built yet.
dist-placeholder:
	@mkdir -p web/dist
	@[ -e web/dist/.vite/manifest.json ] || touch web/dist/.keep

## up: build and start the app and its database in containers (http://localhost:$(PORT))
up: .env
	docker compose up --build -d --wait
	@echo "Linktree: http://localhost:$(PORT)  (admin: /admin)"

## down: stop and remove the containers (the database volume is kept)
down:
	docker compose down

## dev: database in Compose, Go server on the host, Vite with hot reload
dev: .env web/node_modules dist-placeholder
	docker compose up -d --wait db
	@echo "Linktree (hot reload): http://localhost:$(PORT) — restart make dev after a Go change"
	@trap 'kill 0' EXIT; set -a; . ./.env; set +a; \
		(cd web && npm run dev) & VITE_DEV_URL=http://localhost:5173 go run -ldflags "-X main.version=dev" ./cmd/linktree

## test: Go tests (with PostgreSQL integration tests) and front tests
test: .env web/node_modules dist-placeholder
	docker compose up -d --wait db
	TEST_DATABASE_URL='$(DATABASE_URL)' go test ./...
	cd web && npm test

## lint: Go formatting + go vet, TypeScript type check + Prettier
lint: web/node_modules dist-placeholder
	@unformatted="$$(gofmt -l $(GO_DIRS))"; \
		if [ -n "$$unformatted" ]; then echo "Not gofmt-formatted:"; echo "$$unformatted"; exit 1; fi
	go vet ./...
	cd web && npm run lint

## fmt: format Go and front code
fmt: web/node_modules
	gofmt -w $(GO_DIRS)
	cd web && npm run fmt

## build: build the Docker image, tagged with the git SHA (also reported by /version)
build:
	docker build --build-arg VERSION=$(TAG) -t $(IMAGE):$(TAG) .

## migrate: apply migrations to DATABASE_URL
migrate: .env dist-placeholder
	docker compose up -d --wait db
	set -a; . ./.env; set +a; go run ./cmd/linktree migrate

## hash-password: print the ADMIN_PASSWORD_HASH of a password typed without echo
hash-password:
	@printf "Mot de passe : "; stty -echo; read -r pw; stty echo; echo; \
		printf '%s' "$$pw" | go run ./cmd/hashpw
