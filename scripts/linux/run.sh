#!/bin/bash
# Shepherd Linux 运行脚本
# 支持从配置文件的 node.role 读取运行模式
# 默认配置为 hybrid 混合模式

set -e

# 颜色输出
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# 获取脚本所在目录
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="$(dirname "$(dirname "$SCRIPT_DIR")")"
BUILD_DIR="${PROJECT_DIR}/build"
BINARY_NAME="shepherd"

# 打印带颜色的消息
print_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

print_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# 显示帮助信息
show_help() {
    cat << EOF
🐏 Shepherd 运行脚本 (Linux)

用法: $0 [选项]

通用选项:
    -h, --help     显示此帮助信息
    -b, --build    运行前先编译
    -v, --version  显示版本信息
    --config PATH  指定配置文件路径 (可选)

示例:
    # 使用默认配置 (hybrid 混合模式)
    $0

    # 使用自定义配置文件
    $0 --config config/custom.yaml

    # 运行前先编译
    $0 -b

注意:
    - 节点角色由配置文件的 node.role 字段决定
    - 可选角色: hybrid (默认), master, client
    - 建议使用 hybrid 模式获得完整功能

EOF
}

# 检查二进制文件是否存在
check_binary() {
    if [ ! -f "${BUILD_DIR}/${BINARY_NAME}" ]; then
        print_warning "二进制文件不存在: ${BUILD_DIR}/${BINARY_NAME}"
        read -p "是否现在编译? (y/N): " -n 1 -r
        echo
        if [[ $REPLY =~ ^[Yy]$ ]]; then
            (cd "${SCRIPT_DIR}" && ./build.sh)
        else
            print_error "无法继续，请先编译项目"
            exit 1
        fi
    fi
}

# 显示版本信息
show_version() {
    if [ -f "${BUILD_DIR}/${BINARY_NAME}" ]; then
        "${BUILD_DIR}/${BINARY_NAME}" --version
    else
        print_error "二进制文件不存在，请先编译"
        exit 1
    fi
    exit 0
}

# 主函数
main() {
    local BUILD_FIRST=false
    local CONFIG_PATH=""

    # 解析参数
    while [[ $# -gt 0 ]]; do
        case $1 in
            -h|--help)
                show_help
                exit 0
                ;;
            -b|--build)
                BUILD_FIRST=true
                shift
                ;;
            -v|--version)
                show_version
                ;;
            --config)
                CONFIG_PATH="$2"
                shift 2
                ;;
            *)
                print_error "未知参数: $1"
                show_help
                exit 1
                ;;
        esac
    done

    # 编译（如果需要）
    if [ "$BUILD_FIRST" = true ]; then
        print_info "编译项目..."
        (cd "${SCRIPT_DIR}" && ./build.sh)
        print_success "编译完成"
    fi

    # 检查二进制文件
    check_binary

    # 自动检测配置文件：node -> 从 example 自动拷贝 -> server.config.yaml (fallback)
    if [ -z "$CONFIG_PATH" ]; then
        local NODE_CONFIG="${PROJECT_DIR}/config/node/server.config.yaml"
        local EXAMPLE_CONFIG="${PROJECT_DIR}/config/example/server.config.yaml"

        if [ -f "$NODE_CONFIG" ]; then
            # 1. 优先使用 node 目录中的配置文件
            CONFIG_PATH="$NODE_CONFIG"
            print_info "使用 node 配置文件: ${CONFIG_PATH}"
        elif [ -f "$EXAMPLE_CONFIG" ]; then
            # 2. 如果 node 目录没有配置，但 example 目录有，则自动拷贝
            mkdir -p "$(dirname "$NODE_CONFIG")"
            cp "$EXAMPLE_CONFIG" "$NODE_CONFIG"
            CONFIG_PATH="$NODE_CONFIG"
            print_success "从 example 自动复制配置文件到 node 目录"
            print_info "配置文件: ${CONFIG_PATH}"
        else
            # 3. 都不存在则报错
            print_error "未找到配置文件"
            print_error "请确保以下文件之一存在："
            print_error "  - ${EXAMPLE_CONFIG}"
            print_error "  - ${NODE_CONFIG}"
            print_error ""
            print_error "可以从 example 复制配置文件："
            print_error "  mkdir -p config/node"
            print_error "  cp config/example/*.config.yaml config/node/"
            exit 1
        fi
    else
        # 验证自定义配置文件是否存在
        if [ ! -f "$CONFIG_PATH" ]; then
            # 尝试相对于项目目录的路径
            if [ -f "${PROJECT_DIR}/${CONFIG_PATH}" ]; then
                CONFIG_PATH="${PROJECT_DIR}/${CONFIG_PATH}"
            else
                print_error "配置文件不存在: ${CONFIG_PATH}"
                exit 1
            fi
        fi
        print_info "使用自定义配置文件: ${CONFIG_PATH}"
        fi

    # 显示启动信息
    echo ""
    echo "=========================================="
    echo "  🐏 Shepherd"
    echo "=========================================="
    echo "  配置文件: ${CONFIG_PATH}"
    echo "  节点角色: (从配置文件读取)"
    echo "=========================================="
    echo ""

    # 构建命令参数
    local ARGS=()

    # 添加配置文件路径
    if [ -n "${CONFIG_PATH}" ]; then
        ARGS+=("--config=${CONFIG_PATH}")
    fi

    # 启动程序
    cd "${PROJECT_DIR}"
    exec "${BUILD_DIR}/${BINARY_NAME}" "${ARGS[@]}"
}

# 运行主函数
main "$@"
