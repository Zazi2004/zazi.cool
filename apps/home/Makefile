# Standard targets shared by every app (see the root CLAUDE.md, §4).
# Variables come from .env, created from .env.example on first use.

-include .env

IMAGE ?= ghcr.io/zazi2004/home
TAG   ?= $(shell git rev-parse --short HEAD)

.PHONY: up down dev test lint fmt build migrate hash-password export-banners

.env:
	cp .env.example .env

## up: build and start the app and its database in containers (http://localhost:$(PORT))
up: .env
	docker compose up --build -d --wait
	@echo "Accueil: http://localhost:$(PORT)"

## down: stop and remove the containers (the database volume is kept)
down:
	docker compose down

## dev: database in Compose, Go server on the host (restart after a Go change)
dev: .env
	docker compose up -d --wait db
	set -a; . ./.env; set +a; go run -ldflags "-X main.version=dev" ./cmd/home

## test: Go tests, including PostgreSQL integration tests
test: .env
	docker compose up -d --wait db
	TEST_DATABASE_URL='$(DATABASE_URL)' go test ./...

## lint: Go formatting + go vet
lint:
	@unformatted="$$(gofmt -l .)"; \
		if [ -n "$$unformatted" ]; then echo "Not gofmt-formatted:"; echo "$$unformatted"; exit 1; fi
	go vet ./...

## fmt: format Go code
fmt:
	gofmt -w .

## build: build the Docker image, tagged with the git SHA (also reported by /version)
build:
	docker build --build-arg VERSION=$(TAG) -t $(IMAGE):$(TAG) .

## migrate: apply migrations to DATABASE_URL
migrate: .env
	docker compose up -d --wait db
	set -a; . ./.env; set +a; go run ./cmd/home migrate

## hash-password: print the ADMIN_PASSWORD_HASH of a password typed without echo
hash-password:
	@printf "Mot de passe : "; stty -echo; read -r pw; stty echo; echo; \
		printf '%s' "$$pw" | go run ./cmd/hashpw

## export-banners: render web/banners/*.svg to the embedded JPEGs (needs Chrome, Chromium or Brave)
export-banners:
	node web/scripts/export-banners.mjs
