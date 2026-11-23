.PHONY: proto sqlc build build-sync frontend lint dev test install-tools up down logs sync deploy provision

BIN     := bin/server
BIN_SYNC := bin/sync
PROTO_OUT_GO := internal/gen
PROTO_OUT_TS := frontend/src/gen

## Install code-generation tools (run once)
install-tools:
	go install github.com/bufbuild/buf/cmd/buf@latest
	go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
	go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
	go install github.com/grpc-ecosystem/grpc-gateway/v2/protoc-gen-grpc-gateway@latest
	go install github.com/grpc-ecosystem/grpc-gateway/v2/protoc-gen-openapiv2@latest
	cd proto && buf dep update
	npm install --prefix frontend

## Generate Go + TypeScript from proto files
proto:
	cd proto && buf generate

## Generate Go DB code from SQL queries
sqlc:
	sqlc generate --file db/sqlc.yaml

## Compile the Go server binary
build:
	go build -o $(BIN) ./cmd/server

## Compile the Scryfall sync CLI
build-sync:
	go build -o $(BIN_SYNC) ./cmd/sync

## Build the frontend for production
frontend:
	npm run build --prefix frontend

## Run all linters
lint:
	golangci-lint run ./...
	npm run lint --prefix frontend

## Run the backend with live reload (requires air: go install github.com/air-verse/air@latest)
dev:
	air -c .air.toml

## Run tests
test:
	go test ./...

## --- Docker --------------------------------------------------------------

COMPOSE      := docker compose
DEPLOY_HOST  ?= ubuntu@thoth.abandonedfactory.net

## Build and start nginx + api + mysql locally (http://localhost:8080)
up:
	$(COMPOSE) up -d --build

down:
	$(COMPOSE) down

logs:
	$(COMPOSE) logs -f

## Run the scryfall sync job in a container
sync:
	$(COMPOSE) run --rm sync

## Deploy (or roll back to) an already-built release. CI does this on every
## merge to main; use this for rollbacks. TAG must be a commit whose images CI
## pushed to ECR. Secrets live in /opt/thoth/.env.prod on the server.
TAG ?= $(shell git rev-parse HEAD)
deploy:
	tar -c compose.yaml compose.prod.yaml | ssh $(DEPLOY_HOST) thoth-deploy $(TAG)

## Configure the VPS (nginx + certbot + Docker) with Ansible.
## Edit deploy/ansible/inventory.ini and group_vars/all.yml first.
## Pass extra flags with ANSIBLE_ARGS, e.g. ANSIBLE_ARGS="--check --diff" or "-K".
provision:
	cd deploy/ansible && ansible-playbook site.yml $(ANSIBLE_ARGS)
