#!/bin/bash
# Shepherd 跨平台编译脚本
# 一次性编译所有支持的平台

set -e

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

# 项目信息
PROJECT_NAME="shepherd"
BUILD_DIR="build"
VERSION=${1:-"dev"}

# 构建时间
BUILD_TIME=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
GIT_COMMIT=$(git rev-parse --short HEAD 2>/dev/null || echo "unknown")

# 支持的平台
PLATFORMS=(
    "linux/amd64"
    "linux/arm64"
    "darwin/amd64"
    "darwin/arm64"
    "windows/amd64"
    "windows/386"
)

echo ""
echo -e "${GREEN}  Shepherd Cross-Platform Build${NC}"
echo ""
echo ""
echo "版本: ${VERSION}"
echo "Git Commit: ${GIT_COMMIT}"
echo "构建时间: ${BUILD_TIME}"
echo ""
echo -e "${YELLOW}编译平台:${NC}"
for platform in "${PLATFORMS[@]}"; do
    echo "  - $platform"
done
echo ""

# 清理旧的构建文件
echo -e "${YELLOW}清理旧的构建文件...${NC}"
rm -rf "${BUILD_DIR}"
mkdir -p "${BUILD_DIR}"

# 设置编译参数
LDFLAGS="-X main.Version=${VERSION}"
LDFLAGS="${LDFLAGS} -X main.BuildTime=${BUILD_TIME}"
LDFLAGS="${LDFLAGS} -X main.GitCommit=${GIT_COMMIT}"
LDFLAGS="${LDFLAGS} -s -w"

# 设置 Go 代理
export GOPROXY=${GOPROXY:-"https://goproxy.cn,direct"}

# 检查 Go 是否安装
if ! command -v go &> /dev/null; then
    echo -e "${RED}错误: Go 未安装或不在 PATH 中${NC}"
    exit 1
fi

# 编译各个平台
SUCCESS_COUNT=0
FAIL_COUNT=0

for platform in "${PLATFORMS[@]}"; do
    IFS='/' read -r GOOS GOARCH <<< "$platform"

    # 确定输出文件名
    OUTPUT_NAME="${PROJECT_NAME}-${GOOS}-${GOARCH}"
    if [ "$GOOS" = "windows" ]; then
        OUTPUT_NAME="${OUTPUT_NAME}.exe"
    fi

    # 显示编译进度
    echo -e "${BLUE}编译 ${GOOS}/${GOARCH}...${NC}"

    # 编译
    env GOOS=$GOOS GOARCH=$GOARCH go build \
        -ldflags "${LDFLAGS}" \
        -o "${BUILD_DIR}/${OUTPUT_NAME}" \
        cmd/shepherd/main.go

    if [ $? -eq 0 ]; then
        echo -e "${GREEN}  ✓ ${OUTPUT_NAME}${NC}"
        SUCCESS_COUNT=$((SUCCESS_COUNT + 1))
    else
        echo -e "${RED}  ✗ ${OUTPUT_NAME} (编译失败)${NC}"
        FAIL_COUNT=$((FAIL_COUNT + 1))
    fi
done

echo ""
echo ""
echo -e "${GREEN}  构建完成${NC}"
echo ""
echo "成功: ${SUCCESS_COUNT}"
echo "失败: ${FAIL_COUNT}"
echo ""

# 创建校验和文件
echo -e "${YELLOW}生成校验和...${NC}"
cd "${BUILD_DIR}"
if command -v sha256sum &> /dev/null; then
    sha256sum ${PROJECT_NAME}-* > SHA256SUMS
    echo -e "${GREEN}✓ SHA256 校验和已生成: SHA256SUMS${NC}"
elif command -v shasum &> /dev/null; then
    shasum -a 256 ${PROJECT_NAME}-* > SHA256SUMS
    echo -e "${GREEN}✓ SHA256 校验和已生成: SHA256SUMS${NC}"
fi
cd ..

# 显示构建结果
echo -e "${YELLOW}构建文件:${NC}"
ls -lh "${BUILD_DIR}/${PROJECT_NAME}-"* 2>/dev/null || echo "  (无文件)"
echo ""

# 生成编译报告
REPORT_FILE="${BUILD_DIR}/build-report.txt"
echo "Shepherd Build Report" > "${REPORT_FILE}"
echo "=" >> "${REPORT_FILE}"
echo "" >> "${REPORT_FILE}"
echo "Version: ${VERSION}" >> "${REPORT_FILE}"
echo "Build Time: ${BUILD_TIME}" >> "${REPORT_FILE}"
echo "Git Commit: ${GIT_COMMIT}" >> "${REPORT_FILE}"
echo "" >> "${REPORT_FILE}"
echo "Build Statistics:" >> "${REPORT_FILE}"
echo "  Success: ${SUCCESS_COUNT}" >> "${REPORT_FILE}"
echo "  Failed: ${FAIL_COUNT}" >> "${REPORT_FILE}"
echo "" >> "${REPORT_FILE}"
echo "Binaries:" >> "${REPORT_FILE}"
for file in "${BUILD_DIR}/${PROJECT_NAME}-"*; do
    if [ -f "$file" ]; then
        SIZE=$(du -h "$file" | cut -f1)
        SHA=$(sha256sum "$file" 2>/dev/null | cut -d' ' -f1)
        echo "  $(basename $file) - ${SIZE}" >> "${REPORT_FILE}"
        [ -n "$SHA" ] && echo "    SHA256: ${SHA}" >> "${REPORT_FILE}"
    fi
done

echo -e "${GREEN}✓ 构建报告已生成: ${REPORT_FILE}${NC}"
echo ""
