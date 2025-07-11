.PHONY: build test lint clean install deps help

# Variables
BINARY_NAME=sqlc-analyzer
VERSION?=dev
LDFLAGS=-ldflags "-X main.version=${VERSION}"

# Default target
all: deps lint test build

# Install dependencies
deps:
	go mod download
	go mod tidy

# Run linter
lint:
	@command -v golangci-lint >/dev/null 2>&1 || { echo "golangci-lint not installed, installing..."; go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest; }
	golangci-lint run

# Run tests
test:
	go test -v -race -coverprofile=coverage.out ./...

# Run tests with coverage report
test-coverage: test
	go tool cover -html=coverage.out -o coverage.html

# Build the binary
build:
	go build ${LDFLAGS} -o bin/${BINARY_NAME} cmd/analyzer/main.go

# Build for all platforms
build-all:
	GOOS=linux GOARCH=amd64 go build ${LDFLAGS} -o bin/${BINARY_NAME}-linux-amd64 cmd/analyzer/main.go
	GOOS=darwin GOARCH=amd64 go build ${LDFLAGS} -o bin/${BINARY_NAME}-darwin-amd64 cmd/analyzer/main.go
	GOOS=windows GOARCH=amd64 go build ${LDFLAGS} -o bin/${BINARY_NAME}-windows-amd64.exe cmd/analyzer/main.go

# Install the binary
install:
	go install ${LDFLAGS} cmd/analyzer/main.go

# Clean build artifacts
clean:
	rm -rf bin/
	rm -f coverage.out coverage.html

# Run security scan
security:
	@command -v govulncheck >/dev/null 2>&1 || { echo "govulncheck not installed, installing..."; go install golang.org/x/vuln/cmd/govulncheck@latest; }
	govulncheck ./...

# Generate mocks
generate:
	go generate ./...

# Run benchmarks
bench:
	go test -bench=. -benchmem ./...

# Run integration tests
integration-test:
	go test -tags=integration ./test/integration/...

# Docker build
docker-build:
	docker build -t ${BINARY_NAME}:${VERSION} .

# Docker run
docker-run:
	docker run --rm ${BINARY_NAME}:${VERSION}

# Help
help:
	@echo "Available targets:"
	@echo "  deps             - Install dependencies"
	@echo "  lint             - Run linter"
	@echo "  test             - Run tests"
	@echo "  test-coverage    - Run tests with coverage"
	@echo "  build            - Build binary"
	@echo "  build-all        - Build for all platforms"
	@echo "  install          - Install binary"
	@echo "  clean            - Clean build artifacts"
	@echo "  security         - Run security scan"
	@echo "  generate         - Generate mocks"
	@echo "  bench            - Run benchmarks"
	@echo "  integration-test - Run integration tests"
	@echo "  docker-build     - Build Docker image"
	@echo "  docker-run       - Run Docker container"
	@echo "  help             - Show this help"