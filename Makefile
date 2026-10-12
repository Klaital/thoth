.PHONY: proto sqlc build build-sync frontend lint dev install-tools

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
