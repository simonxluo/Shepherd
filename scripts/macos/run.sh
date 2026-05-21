#!/bin/bash
# Shepherd macOS 运行脚本
# 使用 Cobra CLI: shepherd serve [--config path] [--web] [--build] [--host addr] [--port num]
# 节点角色通过配置文件 node.role 字段设置 (hybrid/master/client)

set -e

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="$(dirname "$(dirname "$SCRIPT_DIR")")"
BUILD_DIR="${PROJECT_DIR}/build"
BINARY="${BUILD_DIR}/shepherd"

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

show_help() {
    cat << EOF
  Shepherd 运行脚本 (macOS)

用法: $0 [选项]

选项:
    -h, --help         显示此帮助信息
    -b, --build        运行前先编译
    -v, --version      显示版本信息
    --config PATH      指定配置文件路径
    --web              启动前端开发服务器
    --host ADDR        监听地址
    --port PORT        监听端口

节点角色通过配置文件 node.role 字段设置:
    hybrid             混合模式 (默认)
    master             Master 模式 - 管理多个 Client 节点
    client             Client 模式 - 作为工作节点

macOS 特定选项:
    --no-gatekeeper    跳过 Gatekeeper 验证（解决隔离问题）

示例:
    # 使用默认配置启动
    $0

    # 编译后启动，同时启动前端
    $0 --build --web

    # 使用自定义配置文件
    $0 --config config/node/server.config.yaml

    # 跳过 Gatekeeper 验证
    $0 --no-gatekeeper

EOF
}

check_binary() {
    if [ ! -f "${BINARY}" ]; then
        print_warning "二进制文件不存在: ${BINARY}"
        read -p "是否现在编译? (y/N): " -n 1 -r
        echo
        if [[ $REPLY =~ ^[Yy]$ ]]; then
            make -C "${PROJECT_DIR}" build
        else
            print_error "无法继续，请先编译项目"
            exit 1
        fi
    fi
}

fix_gatekeeper() {
    if [ -f "${BINARY}" ]; then
        print_info "修复 Gatekeeper 隔离..."
        xattr -cr "${BINARY}"
        print_success "修复完成"
    fi
}

show_version() {
    if [ -f "${BINARY}" ]; then
        "${BINARY}" version
    else
        print_error "二进制文件不存在，请先编译"
        exit 1
    fi
    exit 0
}

main() {
    local BUILD_FIRST=false
    local FIX_GATEKEEPER=false
    local SERVE_ARGS=()

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
                SERVE_ARGS+=("--config" "$2")
                shift 2
                ;;
            --web)
                SERVE_ARGS+=("--web")
                shift
                ;;
            --host)
                SERVE_ARGS+=("--host" "$2")
                shift 2
                ;;
            --port)
                SERVE_ARGS+=("--port" "$2")
                shift 2
                ;;
            --no-gatekeeper)
                FIX_GATEKEEPER=true
                shift
                ;;
            *)
                print_error "未知参数: $1"
                show_help
                exit 1
                ;;
        esac
    done

    if [ "$BUILD_FIRST" = true ]; then
        print_info "编译项目..."
        make -C "${PROJECT_DIR}" build
        print_success "编译完成"
    fi

    check_binary

    if [ "$FIX_GATEKEEPER" = true ]; then
        fix_gatekeeper
    fi

    echo ""
    echo "---"
    echo "  Shepherd"
    echo "---"
    echo ""

    cd "${PROJECT_DIR}"
    exec "${BINARY}" serve "${SERVE_ARGS[@]}"
}

main "$@"
