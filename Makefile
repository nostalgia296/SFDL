.PHONY: all build clean test run install lint fmt

# 变量
BINARY_NAME := sfdl
BUILD_DIR := build
CMD_DIR := cmd/sfdl
GO := go
LDFLAGS := -s -w

# 默认目标
all: build

# 构建
build:
	@echo "Building $(BINARY_NAME)..."
	@mkdir -p $(BUILD_DIR)
	$(GO) build -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/$(BINARY_NAME) ./$(CMD_DIR)
	@echo "Build complete: $(BUILD_DIR)/$(BINARY_NAME)"

# 开发模式构建（无优化，带调试信息）
dev:
	@echo "Building $(BINARY_NAME) (dev mode)..."
	@mkdir -p $(BUILD_DIR)
	$(GO) build -o $(BUILD_DIR)/$(BINARY_NAME) ./$(CMD_DIR)

# 清理构建产物
clean:
	@echo "Cleaning..."
	@rm -rf $(BUILD_DIR)
	@echo "Clean complete"

# 运行测试
test:
	@echo "Running tests..."
	$(GO) test -v ./...

# 直接运行
run:
	$(GO) run ./$(CMD_DIR)

# 安装到 $GOPATH/bin
install:
	@echo "Installing $(BINARY_NAME)..."
	$(GO) install ./$(CMD_DIR)
	@echo "Install complete"

# 代码格式化
fmt:
	@echo "Formatting code..."
	$(GO) fmt ./...

# 代码检查
lint:
	@echo "Linting code..."
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run ./...; \
	else \
		echo "golangci-lint not installed, running go vet..."; \
		$(GO) vet ./...; \
	fi

# 依赖管理
deps:
	@echo "Downloading dependencies..."
	$(GO) mod download
	$(GO) mod tidy
	@echo "Dependencies ready"

# 交叉编译（Linux AMD64）
build-linux:
	@echo "Building for Linux AMD64..."
	@mkdir -p $(BUILD_DIR)
	GOOS=linux GOARCH=amd64 $(GO) build -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/$(BINARY_NAME)-linux-amd64 ./$(CMD_DIR)

# 交叉编译（Windows AMD64）
build-windows:
	@echo "Building for Windows AMD64..."
	@mkdir -p $(BUILD_DIR)
	GOOS=windows GOARCH=amd64 $(GO) build -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/$(BINARY_NAME)-windows-amd64.exe ./$(CMD_DIR)

# 交叉编译（macOS AMD64）
build-darwin:
	@echo "Building for macOS AMD64..."
	@mkdir -p $(BUILD_DIR)
	GOOS=darwin GOARCH=amd64 $(GO) build -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-amd64 ./$(CMD_DIR)

# 交叉编译（macOS ARM64）
build-darwin-arm64:
	@echo "Building for macOS ARM64..."
	@mkdir -p $(BUILD_DIR)
	GOOS=darwin GOARCH=arm64 $(GO) build -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-arm64 ./$(CMD_DIR)

# 全部平台构建
build-all: build-linux build-windows build-darwin build-darwin-arm64
	@echo "All builds complete"

# 帮助信息
help:
	@echo "Available targets:"
	@echo "  make build          - Build the binary"
	@echo "  make dev            - Build in dev mode (no optimization)"
	@echo "  make clean          - Remove build artifacts"
	@echo "  make test           - Run tests"
	@echo "  make run            - Run the application"
	@echo "  make install        - Install to \$$GOPATH/bin"
	@echo "  make fmt            - Format code"
	@echo "  make lint           - Run linter"
	@echo "  make deps           - Download and tidy dependencies"
	@echo "  make build-linux    - Cross-compile for Linux"
	@echo "  make build-windows  - Cross-compile for Windows"
	@echo "  make build-darwin   - Cross-compile for macOS (Intel)"
	@echo "  make build-all      - Cross-compile for all platforms"
