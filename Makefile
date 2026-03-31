# Netconnector Makefile

# Variables
BIN_DIR=bin
PROTO_DIR=proto
SERVER_MAIN=cmd/server/main.go
CLIENT_MAIN=cmd/client/main.go
ADMIN_MAIN=cmd/admin/main.go
DUMMY_MAIN=cmd/dummy/main.go

.DEFAULT_GOAL := help

.PHONY: all proto build build-server build-client build-admin build-dummy clean tidy setup-deps run-dummy run-server help release-client release-server

all: proto build

setup-deps: ## Install protobuf dependencies (requires go on path)
	go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
	go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
	@echo "Dependencies installed. Ensure $(go env GOPATH)/bin is in your PATH."

proto: ## Generate gRPC stubs from .proto files
	@mkdir -p $(PROTO_DIR)/pb
	@# Note: requires protoc installed on system
	protoc --go_out=. --go-grpc_out=. $(PROTO_DIR)/tunnel.proto
	@echo "Protobuf compiled successfully."

build: build-server build-client build-admin build-dummy ## Build all binaries

build-server:
	@mkdir -p $(BIN_DIR)
	go build -o $(BIN_DIR)/server $(SERVER_MAIN)

build-client:
	@mkdir -p $(BIN_DIR)
	go build -o $(BIN_DIR)/client $(CLIENT_MAIN)

build-admin:
	@mkdir -p $(BIN_DIR)
	go build -o $(BIN_DIR)/admin $(ADMIN_MAIN)

build-dummy:
	@mkdir -p $(BIN_DIR)
	go build -o $(BIN_DIR)/dummy $(DUMMY_MAIN)

tidy: ## Clean up go.mod and go.sum
	go mod tidy

clean: ## Remove binaries and generated code
	rm -rf $(BIN_DIR)
	rm -f $(PROTO_DIR)/pb/*.go

run-dummy: ## Run the local dummy echo server (Port 3000)
	go run $(DUMMY_MAIN)

setup-test: build-admin ## Register a test subdomain (app1.local -> CLIENT-123) in SQLite
	./$(BIN_DIR)/admin add app1.local CLIENT-123

run-client-local: build-client ## Run the Local Agent using configuration from .env file
	./$(BIN_DIR)/client

release-client: ## Generate a portable ZIP for the Client/Agent
	@chmod +x scripts/build-portable.sh
	./scripts/build-portable.sh client

release-server: ## Generate a portable ZIP for the Server/Bridge
	@chmod +x scripts/build-portable.sh
	./scripts/build-portable.sh server

run-server: ## Run the VPS Bridge server (Port 8080 HTTP, 50051 gRPC)
	go run $(SERVER_MAIN)

test: ## Run all tests in the internal directory
	go test -v ./internal/...

cover: ## Run tests and generate coverage report
	go test -coverprofile=coverage.out ./internal/...
	go tool cover -func=coverage.out

cover-html: ## Run tests and open coverage report in browser
	go test -coverprofile=coverage.out ./internal/...
	go tool cover -html=coverage.out

help: ## Show this help message
	@echo "Usage: make [target]"
	@echo ""
	@echo "Targets:"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'
