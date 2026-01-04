# Exchange Rate Service Makefile

.PHONY: all build run test clean proto lint docker help

# Variables
BINARY_NAME=exchange-rate-service
GO=go
GOFLAGS=-v
PROTO_DIR=../../proto
PROTO_OUT=./internal/pb

# Default target
all: build

# Build the binary
build:
	$(GO) build $(GOFLAGS) -o bin/$(BINARY_NAME) ./cmd/server

# Run the service locally
run:
	$(GO) run ./cmd/server

# Run with hot reload (requires air: go install github.com/air-verse/air@latest)
dev:
	air

# Run all tests
test:
	$(GO) test -v ./...

# Run tests with coverage
test-cover:
	$(GO) test -v -coverprofile=coverage.out ./...
	$(GO) tool cover -html=coverage.out -o coverage.html

# Run tests for specific package
test-provider:
	$(GO) test -v ./internal/provider/...

test-service:
	$(GO) test -v ./internal/service/...

test-repository:
	$(GO) test -v ./internal/repository/...

# Lint the code (requires golangci-lint)
lint:
	golangci-lint run ./...

# Format code
fmt:
	$(GO) fmt ./...
	goimports -w .

# Generate proto files (requires protoc and plugins)
proto:
	@mkdir -p $(PROTO_OUT)
	protoc --proto_path=$(PROTO_DIR) \
		--go_out=$(PROTO_OUT) --go_opt=paths=source_relative \
		--go-grpc_out=$(PROTO_OUT) --go-grpc_opt=paths=source_relative \
		$(PROTO_DIR)/common.proto $(PROTO_DIR)/exchange.proto

# Install proto dependencies
proto-deps:
	$(GO) install google.golang.org/protobuf/cmd/protoc-gen-go@latest
	$(GO) install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest

# Clean build artifacts
clean:
	rm -rf bin/
	rm -rf coverage.out coverage.html
	rm -rf $(PROTO_OUT)

# Download dependencies
deps:
	$(GO) mod download
	$(GO) mod tidy

# Update dependencies
deps-update:
	$(GO) get -u ./...
	$(GO) mod tidy

# Docker build
docker-build:
	docker build -t movra/exchange-rate-service:latest .

# Docker run
docker-run:
	docker run -p 8082:8082 -p 9092:9092 \
		-e REDIS_ADDR=host.docker.internal:6379 \
		movra/exchange-rate-service:latest

# Run with docker-compose (from project root)
compose-up:
	cd ../.. && docker-compose up -d exchange-rate-service redis

compose-down:
	cd ../.. && docker-compose down exchange-rate-service

# Health check
health:
	curl -s http://localhost:8082/health | jq .

# Test rate endpoint
test-rate:
	curl -s http://localhost:8082/api/rates/SGD/PHP | jq .

# Test quote endpoint
test-quote:
	curl -s "http://localhost:8082/api/quote?from=SGD&to=PHP&amount=100" | jq .

# Test corridors endpoint
test-corridors:
	curl -s http://localhost:8082/api/corridors | jq .

# Test lock rate
test-lock:
	curl -s -X POST http://localhost:8082/api/rates/lock \
		-H "Content-Type: application/json" \
		-d '{"sourceCurrency":"SGD","targetCurrency":"PHP","durationSeconds":60}' | jq .

# gRPC health check (requires grpcurl)
grpc-health:
	grpcurl -plaintext localhost:9092 grpc.health.v1.Health/Check

# List gRPC services (requires grpcurl)
grpc-list:
	grpcurl -plaintext localhost:9092 list

# Help
help:
	@echo "Exchange Rate Service - Available targets:"
	@echo ""
	@echo "  build          - Build the binary"
	@echo "  run            - Run the service"
	@echo "  dev            - Run with hot reload (requires air)"
	@echo "  test           - Run all tests"
	@echo "  test-cover     - Run tests with coverage"
	@echo "  lint           - Run linter"
	@echo "  fmt            - Format code"
	@echo "  proto          - Generate proto files"
	@echo "  proto-deps     - Install proto dependencies"
	@echo "  clean          - Clean build artifacts"
	@echo "  deps           - Download dependencies"
	@echo "  docker-build   - Build Docker image"
	@echo "  docker-run     - Run in Docker"
	@echo "  health         - Check service health"
	@echo "  test-rate      - Test rate endpoint"
	@echo "  test-quote     - Test quote endpoint"
	@echo "  test-corridors - Test corridors endpoint"
	@echo "  test-lock      - Test lock rate endpoint"
	@echo "  grpc-health    - gRPC health check"
	@echo "  grpc-list      - List gRPC services"
	@echo "  help           - Show this help"
