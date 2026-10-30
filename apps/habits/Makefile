# Standard targets shared by every app (see the root CLAUDE.md, §4).
# Variables come from .env, created from .env.example on first use.

-include .env

IMAGE ?= ghcr.io/zazi2004/habits
TAG   ?= $(shell git rev-parse --short HEAD)
GO_DIRS = $(shell go list -f '{{.Dir}}' ./...)

.PHONY: up down dev test lint fmt build migrate

.env:
	cp .env.example .env

## up: build and start the app and its database in containers (http://localhost:$(PORT))
up: .env
	docker compose up --build -d --wait
	@echo "Habits: http://localhost:$(PORT)"

## down: stop and remove the containers (the database volume is kept)
down:
	docker compose down

## dev: database in Compose, Go server on the host (restart it after a change)
dev: .env
	docker compose up -d --wait db
	set -a; . ./.env; set +a; go run -ldflags "-X main.version=dev" ./cmd/habits

## test: Go tests, including PostgreSQL integration tests
test: .env
	docker compose up -d --wait db
	TEST_DATABASE_URL='$(DATABASE_URL)' go test ./...

## lint: gofmt check + go vet
lint:
	@unformatted="$$(gofmt -l $(GO_DIRS))"; \
		if [ -n "$$unformatted" ]; then echo "Not gofmt-formatted:"; echo "$$unformatted"; exit 1; fi
	go vet ./...

## fmt: format the Go code
fmt:
	gofmt -w $(GO_DIRS)

## build: build the Docker image, tagged with the git SHA (also reported by /version)
build:
	docker build --build-arg VERSION=$(TAG) -t $(IMAGE):$(TAG) .

## migrate: apply migrations to DATABASE_URL
migrate: .env
	docker compose up -d --wait db
	set -a; . ./.env; set +a; go run ./cmd/habits migrate
