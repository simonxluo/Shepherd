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

# 前端进程管理
WEB_PID_FILE="/tmp/shepherd-web-dev.pid"
WEB_LOG_FILE="/tmp/shepherd-web-dev.log"

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
    --web          同时启动前端开发服务器

示例:
    # 使用默认配置 (hybrid 混合模式)
    $0

    # 使用自定义配置文件
    $0 --config config/custom.yaml

    # 运行前先编译
    $0 -b

    # 一键启动前后端
    $0 --web -b

注意:
    - 节点角色由配置文件的 node.role 字段决定
    - 可选角色: hybrid (默认), master, client
    - 建议使用 hybrid 模式获得完整功能
    - 使用 --web 参数时会同时启动前端开发服务器

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

# 启动前端开发服务器
start_web_frontend() {
    print_info "启动前端开发服务器..."

    local web_script="${SCRIPT_DIR}/web.sh"
    if [ ! -f "$web_script" ]; then
        print_error "前端脚本不存在: $web_script"
        return 1
    fi

    # 检查端口占用
    local config_file="${PROJECT_DIR}/web/public/config.yaml"
    local port=$(grep -oP 'port:\s*\K\d+' "$config_file" 2>/dev/null || echo "3000")

    if lsof -Pi :$port -sTCP:LISTEN -t >/dev/null 2>&1; then
        print_warning "前端端口 $port 已被占用，尝试停止..."
        local killed_pids=$(lsof -ti :$port -sTCP:LISTEN 2>/dev/null)
        if [ -n "$killed_pids" ]; then
            echo "$killed_pids" | xargs -r kill -9 2>/dev/null || true
            sleep 1
        fi
    fi

    # 启动前端开发服务器（后台运行）
    "$web_script" -b > "$WEB_LOG_FILE" 2>&1 &
    local web_pid=$!
    echo $web_pid > "$WEB_PID_FILE"

    print_success "前端开发服务器已启动 (PID: $web_pid)"
    print_info "日志文件: $WEB_LOG_FILE"
    print_info "前端地址: http://localhost:$port"

    # 等待前端启动
    sleep 2

    # 检查前端是否成功启动
    if ! kill -0 $web_pid 2>/dev/null; then
        print_error "前端启动失败，查看日志: $WEB_LOG_FILE"
        cat "$WEB_LOG_FILE"
        rm -f "$WEB_PID_FILE"
        return 1
    fi

    return 0
}

# 停止前端开发服务器
stop_web_frontend() {
    if [ -f "$WEB_PID_FILE" ]; then
        local web_pid=$(cat "$WEB_PID_FILE")
        print_info "停止前端开发服务器 (PID: $web_pid)..."

        if kill -0 "$web_pid" 2>/dev/null; then
            # 发送 SIGTERM
            kill -TERM "$web_pid" 2>/dev/null || true

            # 等待进程优雅退出（最多5秒）
            local count=0
            while kill -0 "$web_pid" 2>/dev/null && [ $count -lt 50 ]; do
                sleep 0.1
                count=$((count + 1))
            done

            # 如果进程仍在运行，强制终止
            if kill -0 "$web_pid" 2>/dev/null; then
                print_warning "强制终止前端进程..."
                kill -KILL "$web_pid" 2>/dev/null || true
            fi
        fi

        rm -f "$WEB_PID_FILE"
        print_success "前端开发服务器已停止"
    fi

    # 清理日志文件
    rm -f "$WEB_LOG_FILE"
}

# 清理函数（信号处理）
cleanup() {
    local exit_code=$?
    print_info "正在关闭 Shepherd..."

    # 停止前端
    stop_web_frontend

    if [ $exit_code -eq 0 ]; then
        print_success "Shepherd 已优雅关闭"
    else
        print_warning "Shepherd 已关闭 (退出码: $exit_code)"
    fi

    exit $exit_code
}

# 主函数
main() {
    local BUILD_FIRST=false
    local CONFIG_PATH=""
    local START_WEB=false

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
            -w|--web)
                START_WEB=true
                shift
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

    # 清理残留进程（如果需要启动前端或后端）
    print_info "清理残留进程..."
    local stop_script="${SCRIPT_DIR}/stop_all.sh"
    if [ -f "$stop_script" ]; then
        "$stop_script" --force >/dev/null 2>&1 || true
        print_success "残留进程已清理"
    fi
    echo ""

    # 启动前端（如果需要）
    if [ "$START_WEB" = true ]; then
        # 注册清理函数
        trap cleanup EXIT INT TERM HUP QUIT

        if ! start_web_frontend; then
            print_error "前端启动失败"
            exit 1
        fi

        echo ""
    fi

    # 显示启动信息
    echo ""
    echo "=========================================="
    echo "  🐏 Shepherd"
    echo "=========================================="
    echo "  配置文件: ${CONFIG_PATH}"
    echo "  节点角色: (从配置文件读取)"
    if [ "$START_WEB" = true ]; then
        echo "  前端服务器: 已启动"
    fi
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
