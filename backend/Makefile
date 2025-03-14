.PHONY: build run test lint docker-up docker-down clean

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod
BINARY_NAME=order-engine

# Build the application
build:
	$(GOBUILD) -o $(BINARY_NAME) ./cmd/server

# Run the application
run:
	$(GOBUILD) -o $(BINARY_NAME) ./cmd/server
	./$(BINARY_NAME)

# Run tests
test:
	$(GOTEST) -v ./...

# Run tests with coverage
test-coverage:
	$(GOTEST) -v -coverprofile=coverage.out ./...
	$(GOCMD) tool cover -html=coverage.out

# Run linter
lint:
	golangci-lint run

# Start Docker services
docker-up:
	docker-compose up -d

# Stop Docker services
docker-down:
	docker-compose down

# Clean build artifacts
clean:
	$(GOCLEAN)
	rm -f $(BINARY_NAME)
	rm -f coverage.out

# Download dependencies
deps:
	$(GOMOD) download

# Tidy go.mod
tidy:
	$(GOMOD) tidy

# Generate mocks (requires mockgen)
mocks:
	mockgen -source=pkg/matching/engine.go -destination=pkg/matching/mock/engine.go
	mockgen -source=app/services/market_data.go -destination=app/services/mock/market_data.go
	mockgen -source=app/services/risk.go -destination=app/services/mock/risk.go
	mockgen -source=app/services/notification.go -destination=app/services/mock/notification.go
	mockgen -source=app/services/analytics.go -destination=app/services/mock/analytics.go

# Run all pre-commit checks
check: lint test

# Default target
all: check build 