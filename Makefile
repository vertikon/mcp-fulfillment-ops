.PHONY: build test clean lint run deps docker

# Build the application
build:
	go build -o bin/fulfillment-ops ./cmd

# Run tests
test:
	go test -v ./...

# Run tests with coverage
test-coverage:
	go test -v -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

# Clean build artifacts
clean:
	rm -rf bin/ coverage.out coverage.html

# Run linter
lint:
	golangci-lint run

# Install dependencies
deps:
	go mod tidy
	go mod download

# Run the application
run:
	go run ./cmd/main.go

# Build Docker image
docker:
	docker build -t mcp-fulfillment-ops:latest .

# Run Docker container
docker-run:
	docker run -p 8080:8080 mcp-fulfillment-ops:latest

# Generate code
generate:
	go generate ./...

# Format code
fmt:
	go fmt ./...

# Vet code for potential issues
vet:
	go vet ./...

# Security scan
security:
	gosec ./...

# Run integration tests
test-integration:
	go test -v ./tests/integration/...

# Run load tests
test-load:
	k6 run tests/load/

# Run all checks
check: lint security test

# Prepare for release
release: clean check test-coverage build
	@echo "Ready for release"