#!/bin/bash
# Shepherd macOS 编译脚本
# 用法: ./scripts/macos/build.sh [version]

set -e

# 获取脚本所在目录并切换到项目根目录
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$(dirname "$(dirname "$SCRIPT_DIR")")"

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# 项目信息
PROJECT_NAME="shepherd"
BUILD_DIR="build"
CMD_DIR="cmd/shepherd"
VERSION=${1:-"dev"}

# 构建信息
BUILD_TIME=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
GIT_COMMIT=$(git rev-parse --short HEAD 2>/dev/null || echo "unknown")
GO_VERSION=$(go version | awk '{print $3}')

echo ""
echo -e "${BLUE}  Shepherd Build Script (macOS)${NC}"
echo ""
echo ""
echo "版本: ${VERSION}"
echo "Go 版本: ${GO_VERSION}"
echo "Git Commit: ${GIT_COMMIT}"
echo ""

# 清理旧的构建文件
echo -e "${YELLOW}清理旧的构建文件...${NC}"
rm -rf "${BUILD_DIR}"
mkdir -p "${BUILD_DIR}"

# 设置编译参数
LDFLAGS="-X main.Version=${VERSION}"
LDFLAGS="${LDFLAGS} -X main.BuildTime=${BUILD_TIME}"
LDFLAGS="${LDFLAGS} -X main.GitCommit=${GIT_COMMIT}"
LDFLAGS="${LDFLAGS} -s -w"  # 去除调试信息，减小二进制大小

# macOS 系统
TARGET_OS="darwin"

# 检测架构
ARCH=$(uname -m)
case "${ARCH}" in
    x86_64)
        TARGET_ARCH="amd64"
        ;;
    arm64)
        TARGET_ARCH="arm64"
        ;;
    *)
        echo -e "${RED}未知架构: ${ARCH}${NC}"
        exit 1
        ;;
esac

BINARY_NAME="${PROJECT_NAME}-darwin-${TARGET_ARCH}"

echo -e "${YELLOW}编译目标: ${TARGET_OS}/${TARGET_ARCH}${NC}"
echo -e "${YELLOW}输出文件: ${BUILD_DIR}/${BINARY_NAME}${NC}"
echo ""

# 编译
echo -e "${GREEN}开始编译...${NC}"
if [ "${GOPROXY}" = "" ]; then
    GOPROXY="https://goproxy.cn,direct"
fi
export GOPROXY

# 检查 Go 是否安装
if ! command -v go &> /dev/null; then
    echo -e "${RED}错误: Go 未安装或不在 PATH 中${NC}"
    echo -e "${YELLOW}请使用 Homebrew 安装: brew install go${NC}"
    exit 1
fi

# macOS 特定编译参数
# 使用 macOS SDK 版本检测
MACOS_VERSION=$(sw_vers -productVersion)
echo -e "${BLUE}macOS 版本: ${MACOS_VERSION}${NC}"

# 签名设置（可选）
if [ -n "$CODESIGN_IDENTITY" ]; then
    echo -e "${YELLOW}使用代码签名: ${CODESIGN_IDENTITY}${NC}"
    SIGNING="--sign=${CODESIGN_IDENTITY}"
else
    SIGNING="--"
fi

# 构建通用二进制（如果需要 Universal Binary）
# 注意：需要安装 x86_64 和 arm64 两个版本的 Go
if [ "${BUILD_UNIVERSAL}" = "true" ]; then
    echo -e "${YELLOW}构建 Universal Binary...${NC}"

    # 构建 ARM64 版本
    GOOS=darwin GOARCH=arm64 go build \
        -ldflags "${LDFLAGS}" \
        -o "${BUILD_DIR}/${PROJECT_NAME}-darwin-arm64" \
        ./${CMD_DIR}

    # 构建 AMD64 版本
    GOOS=darwin GOARCH=amd64 go build \
        -ldflags "${LDFLAGS}" \
        -o "${BUILD_DIR}/${PROJECT_NAME}-darwin-amd64" \
        ./${CMD_DIR}

    # 合并为 Universal Binary
    lipo -create \
        -output "${BUILD_DIR}/${PROJECT_NAME}-darwin-universal" \
        "${BUILD_DIR}/${PROJECT_NAME}-darwin-arm64" \
        "${BUILD_DIR}/${PROJECT_NAME}-darwin-amd64"

    BINARY_NAME="${PROJECT_NAME}-darwin-universal"
else
    # 单架构构建
    GOOS=darwin GOARCH=${TARGET_ARCH} go build \
        -ldflags "${LDFLAGS}" \
        -o "${BUILD_DIR}/${BINARY_NAME}" \
        ./${CMD_DIR}
fi

if [ $? -eq 0 ]; then
    echo -e "${GREEN}✓ 编译成功!${NC}"
else
    echo -e "${RED}✗ 编译失败${NC}"
    exit 1
fi

# 应用代码签名（如果指定）
if [ -n "$CODESIGN_IDENTITY" ] && [ -f "${BUILD_DIR}/${BINARY_NAME}" ]; then
    codesign --force --sign "${CODESIGN_IDENTITY}" "${BUILD_DIR}/${BINARY_NAME}"
    echo -e "${GREEN}✓ 代码签名完成${NC}"
fi

# 获取二进制大小
BINARY_SIZE=$(du -h "${BUILD_DIR}/${BINARY_NAME}" | cut -f1)
echo ""
echo ""
echo -e "${BLUE}  构建完成${NC}"
echo ""
echo "二进制: ${BUILD_DIR}/${BINARY_NAME}"
echo "大小: ${BINARY_SIZE}"
echo ""

# 运行测试（可选）
if [ "${RUN_TESTS}" = "true" ]; then
    echo -e "${YELLOW}运行测试...${NC}"
    go test ./... -v
fi

# 使用提示
echo -e "${YELLOW}使用方法:${NC}"
echo "  ./${BUILD_DIR}/${BINARY_NAME} serve                       # 启动服务 (默认 hybrid 模式)"
echo "  ./${BUILD_DIR}/${BINARY_NAME} serve --web --build         # 编译 + 前后端一键启动"
echo "  ./${BUILD_DIR}/${BINARY_NAME} serve --config config.yaml  # 使用指定配置文件"
echo "  ./${BUILD_DIR}/${BINARY_NAME} stop                        # 停止服务"
echo "  ./${BUILD_DIR}/${BINARY_NAME} version                     # 查看版本"
echo ""

# macOS 特定提示
if [ -f "${BUILD_DIR}/${BINARY_NAME}" ]; then
    # 首次运行提示
    if ! xattr -p "${BUILD_DIR}/${BINARY_NAME}" &>/dev/null; then
        echo -e "${YELLOW}提示: 如果遇到无法打开的问题，请执行:${NC}"
        echo "  xattr -cr ${BUILD_DIR}/${BINARY_NAME}"
        echo ""
    fi
fi
