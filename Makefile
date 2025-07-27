# Kubernetes Orchestrator Makefile

# Build variables
BINARY_NAME=kube-orchestrator
VERSION?=v1.0.0
BUILD_TIME=$(shell date -u '+%Y-%m-%d_%H:%M:%S')
GIT_COMMIT=$(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")

# Go build flags
LDFLAGS=-ldflags "-X main.Version=${VERSION} -X main.BuildTime=${BUILD_TIME} -X main.GitCommit=${GIT_COMMIT}"
BUILD_FLAGS=-trimpath ${LDFLAGS}

# Directories
BUILD_DIR=build
INTERNAL_DIR=internal

.PHONY: all build clean test deps check-deps install dev help

# Default target
all: check-deps build

# Build the application
build: check-deps
	@echo "🔨 Building ${BINARY_NAME}..."
	@mkdir -p ${BUILD_DIR}
	@go build ${BUILD_FLAGS} -o ${BUILD_DIR}/${BINARY_NAME} .
	@echo "✅ Build complete: ${BUILD_DIR}/${BINARY_NAME}"

# Install dependencies and verify system requirements
deps:
	@echo "📦 Installing Go dependencies..."
	@go mod download
	@go mod tidy
	@echo "✅ Dependencies installed"

# Check system dependencies
check-deps:
	@echo "🔍 Checking system dependencies..."
	@command -v kubectl >/dev/null 2>&1 || { echo "❌ kubectl is required but not installed. See: https://kubernetes.io/docs/tasks/tools/"; exit 1; }
	@command -v git >/dev/null 2>&1 || { echo "❌ git is required but not installed. See: https://git-scm.com/downloads"; exit 1; }
	@echo "✅ System dependencies verified"

# Run tests
test:
	@echo "🧪 Running tests..."
	@go test -v ./...

# Run tests with coverage
test-coverage:
	@echo "🧪 Running tests with coverage..."
	@go test -v -coverprofile=coverage.out ./...
	@go tool cover -html=coverage.out -o coverage.html
	@echo "📊 Coverage report: coverage.html"

# Install the binary globally
install: build
	@echo "📦 Installing ${BINARY_NAME} to /usr/local/bin..."
	@sudo mv ${BUILD_DIR}/${BINARY_NAME} /usr/local/bin/
	@echo "✅ ${BINARY_NAME} installed globally"

# Development build with debug info
dev:
	@echo "🚀 Building development version..."
	@go build -race -o ${BUILD_DIR}/${BINARY_NAME}-dev .
	@echo "✅ Development build: ${BUILD_DIR}/${BINARY_NAME}-dev"

# Run the application in development mode
run: dev
	@echo "🚀 Running ${BINARY_NAME} in development mode..."
	@./${BUILD_DIR}/${BINARY_NAME}-dev

# Clean build artifacts
clean:
	@echo "🧹 Cleaning build artifacts..."
	@rm -rf ${BUILD_DIR}
	@rm -f coverage.out coverage.html
	@echo "✅ Clean complete"

# Format code
fmt:
	@echo "🎨 Formatting code..."
	@go fmt ./...
	@echo "✅ Code formatted"

# Lint code
lint:
	@echo "🔍 Linting code..."
	@command -v golangci-lint >/dev/null 2>&1 || { echo "Installing golangci-lint..."; go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest; }
	@golangci-lint run
	@echo "✅ Code linted"

# Security scan
security:
	@echo "🔒 Running security scan..."
	@command -v gosec >/dev/null 2>&1 || { echo "Installing gosec..."; go install github.com/securecodewarrior/gosec/v2/cmd/gosec@latest; }
	@gosec ./...
	@echo "✅ Security scan complete"

# Build for multiple platforms
build-all: check-deps
	@echo "🔨 Building for multiple platforms..."
	@mkdir -p ${BUILD_DIR}
	
	@echo "Building for Linux (amd64)..."
	@GOOS=linux GOARCH=amd64 go build ${BUILD_FLAGS} -o ${BUILD_DIR}/${BINARY_NAME}-linux-amd64 .
	
	@echo "Building for macOS (amd64)..."
	@GOOS=darwin GOARCH=amd64 go build ${BUILD_FLAGS} -o ${BUILD_DIR}/${BINARY_NAME}-darwin-amd64 .
	
	@echo "Building for macOS (arm64)..."
	@GOOS=darwin GOARCH=arm64 go build ${BUILD_FLAGS} -o ${BUILD_DIR}/${BINARY_NAME}-darwin-arm64 .
	
	@echo "Building for Windows (amd64)..."
	@GOOS=windows GOARCH=amd64 go build ${BUILD_FLAGS} -o ${BUILD_DIR}/${BINARY_NAME}-windows-amd64.exe .
	
	@echo "✅ Multi-platform build complete"

# Create release archives
release: build-all
	@echo "📦 Creating release archives..."
	@cd ${BUILD_DIR} && \
	tar -czf ${BINARY_NAME}-${VERSION}-linux-amd64.tar.gz ${BINARY_NAME}-linux-amd64 && \
	tar -czf ${BINARY_NAME}-${VERSION}-darwin-amd64.tar.gz ${BINARY_NAME}-darwin-amd64 && \
	tar -czf ${BINARY_NAME}-${VERSION}-darwin-arm64.tar.gz ${BINARY_NAME}-darwin-arm64 && \
	zip ${BINARY_NAME}-${VERSION}-windows-amd64.zip ${BINARY_NAME}-windows-amd64.exe
	@echo "✅ Release archives created in ${BUILD_DIR}/"

# Setup development environment
setup-dev:
	@echo "🔧 Setting up development environment..."
	@$(MAKE) deps
	@$(MAKE) check-deps
	@echo "Installing development tools..."
	@go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	@go install github.com/securecodewarrior/gosec/v2/cmd/gosec@latest
	@echo "✅ Development environment ready"

# Docker build
docker-build:
	@echo "🐳 Building Docker image..."
	@docker build -t ${BINARY_NAME}:${VERSION} .
	@docker tag ${BINARY_NAME}:${VERSION} ${BINARY_NAME}:latest
	@echo "✅ Docker image built: ${BINARY_NAME}:${VERSION}"

# Show help
help:
	@echo "🚀 Kubernetes Orchestrator Build System"
	@echo ""
	@echo "Available targets:"
	@echo "  build        - Build the application"
	@echo "  deps         - Install Go dependencies"
	@echo "  check-deps   - Verify system dependencies"
	@echo "  test         - Run tests"
	@echo "  test-coverage- Run tests with coverage report"
	@echo "  install      - Install binary globally"
	@echo "  dev          - Build development version"
	@echo "  run          - Run in development mode"
	@echo "  clean        - Clean build artifacts"
	@echo "  fmt          - Format code"
	@echo "  lint         - Lint code"
	@echo "  security     - Run security scan"
	@echo "  build-all    - Build for multiple platforms"
	@echo "  release      - Create release archives"
	@echo "  setup-dev    - Setup development environment"
	@echo "  docker-build - Build Docker image"
	@echo "  help         - Show this help"
	@echo ""
	@echo "🔧 System Requirements:"
	@echo "  • kubectl - Kubernetes command-line tool"
	@echo "  • git - Version control system"
	@echo ""
	@echo "📖 Usage:"
	@echo "  make build && ./build/kube-orchestrator"