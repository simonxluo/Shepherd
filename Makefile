# Shepherd Makefile
# 快速编译和开发命令

.PHONY: all build build-all test clean release run stop swag sqlc sqlc-verify

# 默认目标
all: build

# 项目信息
PROJECT_NAME := shepherd
BUILD_DIR := build
CMD_DIR := cmd/shepherd
VERSION := dev

# Go 相关
GO := $(shell which go 2>/dev/null || echo go)
GOFLAGS := -mod=mod
GOPROXY := https://goproxy.cn,direct

# 编译参数
LDFLAGS := -X main.Version=$(VERSION) -X main.BuildTime=$$(date -u +"%Y-%m-%dT%H:%M:%SZ") -X main.GitCommit=$$(git rev-parse --short HEAD 2>/dev/null || echo "unknown") -s -w

# 构建单平台二进制
build:
	@echo "编译 $(PROJECT_NAME)..."
	@mkdir -p $(BUILD_DIR)
	@$(GO) build $(GOFLAGS) -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/$(PROJECT_NAME) $(CMD_DIR)/main.go
	@echo "✓ 编译完成: $(BUILD_DIR)/$(PROJECT_NAME)"
	@./$(BUILD_DIR)/$(PROJECT_NAME) version

# 跨平台编译
build-all:
	@echo "跨平台编译..."
	@./scripts/build-all.sh $(VERSION)

# 发布打包
release: build-all
	@echo "打包发布..."
	@./scripts/release.sh $(VERSION)

# 运行测试
test:
	@echo "运行测试..."
	@$(GO) test $(GOFLAGS) -v -count=1 ./tests/... ./internal/...

# 运行测试并生成覆盖率
test-coverage:
	@echo "运行测试并生成覆盖率..."
	@$(GO) test $(GOFLAGS) -v -count=1 -coverprofile=coverage.out ./tests/... ./internal/...
	@$(GO) tool cover -html=coverage.out -o coverage.html
	@echo "✓ 覆盖率报告已生成: coverage.html"

# 代码检查
lint:
	@echo "代码检查..."
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run ./...; \
	else \
		echo "golangci-lint 未安装，跳过检查"; \
	fi

# 代码格式化
fmt:
	@echo "格式化代码..."
	@$(GO) fmt ./...

# 代码整理
tidy:
	@echo "整理依赖..."
	@$(GO) mod tidy

# 清理构建文件
clean:
	@echo "清理构建文件..."
	@rm -rf $(BUILD_DIR)
	@rm -rf release
	@rm -f coverage.out coverage.html
	@echo "✓ 清理完成"

# 编译并启动服务（含前端构建和Web界面）
run: build
	@echo "启动 $(PROJECT_NAME)..."
	@./$(BUILD_DIR)/$(PROJECT_NAME) serve --build --web

# 停止所有进程
stop:
	@./$(BUILD_DIR)/$(PROJECT_NAME) stop

# 安装到本地
install: build
	@echo "安装 $(PROJECT_NAME)..."
	@cp $(BUILD_DIR)/$(PROJECT_NAME) /usr/local/bin/
	@echo "✓ 已安装到 /usr/local/bin/$(PROJECT_NAME)"

# 卸载
uninstall:
	@echo "卸载 $(PROJECT_NAME)..."
	@rm -f /usr/local/bin/$(PROJECT_NAME)
	@echo "✓ 已卸载"

# 构建 Docker 镜像
docker-build:
	@echo "构建 Docker 镜像..."
	@docker build -t $(PROJECT_NAME):$(VERSION) .

# 运行 Docker 容器
docker-run:
	@echo "运行 Docker 容器..."
	@docker run -d --name $(PROJECT_NAME) -p 9190:9190 -v $(PWD)/config:/app/config $(PROJECT_NAME):$(VERSION)

# 停止 Docker 容器
docker-stop:
	@echo "停止 Docker 容器..."
	@docker stop $(PROJECT_NAME) || true
	@docker rm $(PROJECT_NAME) || true

# 查看版本
version:
	@echo "版本: $(VERSION)"
	@echo "Go 版本:"
	@$(GO) version

# 生成 Swagger API 文档
swag:
	@echo "生成 Swagger 文档..."
	@$(shell which swag 2>/dev/null || echo swag) init -g cmd/shepherd/main.go -o api-docs --parseDependency --parseInternal --exclude internal/comm/gpu 2>&1
	@echo "✓ Swagger 文档已生成到 api-docs/"

# 生成 sqlc 代码
sqlc:
	@echo "生成 sqlc 代码..."
	@cd internal/comm/storage && sqlc generate
	@echo "✓ sqlc 代码已生成到 internal/comm/storage/sqlcgen/"

# 验证 sqlc 代码是否最新
sqlc-verify:
	@echo "验证 sqlc 代码..."
	@cd internal/comm/storage && sqlc generate && git diff --quiet sqlcgen/
	@echo "✓ sqlc 代码已是最新"

# 帮助信息
help:
	@echo "Shepherd Makefile"
	@echo ""
	@echo "使用方法: make [target]"
	@echo ""
	@echo "可用目标:"
	@echo "  all           - 默认目标，编译项目"
	@echo "  build         - 编译当前平台"
	@echo "  build-all     - 跨平台编译所有平台"
	@echo "  release       - 打包发布版本"
	@echo "  test          - 运行测试"
	@echo "  test-coverage - 运行测试并生成覆盖率报告"
	@echo "  lint          - 代码检查"
	@echo "  fmt           - 格式化代码"
	@echo "  tidy          - 整理依赖"
	@echo "  clean         - 清理构建文件"
	@echo "  run           - 编译并启动服务（含前端构建和Web界面）"
	@echo "  stop          - 停止所有进程"
	@echo "  swag          - 生成 Swagger API 文档"
	@echo "  sqlc          - 生成 sqlc 类型安全数据库代码"
	@echo "  sqlc-verify   - 验证 sqlc 代码是否最新"
	@echo "  install       - 安装到 /usr/local/bin"
	@echo "  uninstall     - 从 /usr/local/bin 卸载"
	@echo "  docker-build  - 构建 Docker 镜像"
	@echo "  docker-run    - 运行 Docker 容器"
	@echo "  docker-stop   - 停止 Docker 容器"
	@echo "  version       - 显示版本信息"
	@echo "  help          - 显示此帮助信息"
	@echo ""
	@echo "环境变量:"
	@echo "  VERSION       - 版本号 (默认: dev)"
	@echo "  GOROOT        - Go 安装路径"
	@echo "  GOPROXY       - Go 模块代理"
