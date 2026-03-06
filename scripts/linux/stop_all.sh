#!/bin/bash
# Shepherd 停止所有进程脚本
# 用于停止所有前后端相关进程

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

# PID 文件和日志文件
WEB_PID_FILE="/tmp/shepherd-web-dev.pid"
WEB_LOG_FILE="/tmp/shepherd-web-dev.log"

# 前端可能使用的端口
FRONTEND_PORTS=(3000 3030 5173 8080)

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
🛑 Shepherd 停止所有进程脚本 (Linux)

用法: $0 [选项]

选项:
    -h, --help          显示此帮助信息
    -v, --verbose       显示详细的停止信息
    --dry-run           模拟运行，显示将要停止的进程但不实际停止
    --force             强制停止所有相关进程（包括 zombie 进程）

示例:
    # 停止所有前后端进程
    $0

    # 显示详细信息
    $0 -v

    # 模拟运行（不实际停止）
    $0 --dry-run

    # 强制停止所有进程
    $0 --force

注意:
    - 此脚本会停止所有 shepherd 后端进程
    - 此脚本会停止所有前端开发服务器（vite, npm, pnpm 等）
    - 使用 --dry-run 可以预览将要停止的进程

EOF
}

# 检查进程是否存在
check_process() {
    local pid=$1
    if kill -0 "$pid" 2>/dev/null; then
        return 0
    else
        return 1
    fi
}

# 优雅停止进程
graceful_stop() {
    local pid=$1
    local name=$2
    local timeout=${3:-5}

    if ! check_process "$pid"; then
        return 0
    fi

    print_info "停止 $name (PID: $pid)..."

    # 发送 SIGTERM
    kill -TERM "$pid" 2>/dev/null || true

    # 等待进程优雅退出
    local count=0
    while check_process "$pid" && [ $count -lt $timeout ]; do
        sleep 0.1
        count=$((count + 1))
    done

    # 如果进程仍在运行，强制终止
    if check_process "$pid"; then
        print_warning "强制终止 $name (PID: $pid)..."
        kill -KILL "$pid" 2>/dev/null || true
        sleep 0.5
    fi

    # 验证进程是否已停止
    if check_process "$pid"; then
        print_error "无法停止 $name (PID: $pid)"
        return 1
    else
        print_success "$name 已停止"
        return 0
    fi
}

# 停止后端进程
stop_backend() {
    print_info "正在停止后端进程..."

    local pids=$(pgrep -f "shepherd" 2>/dev/null || true)
    local stopped=0
    local failed=0

    if [ -z "$pids" ]; then
        print_warning "未发现运行中的 shepherd 后端进程"
        return 0
    fi

    for pid in $pids; do
        if [ "$DRY_RUN" = true ]; then
            echo "  将停止 shepherd 进程 (PID: $pid)"
            stopped=$((stopped + 1))
        else
            if graceful_stop "$pid" "shepherd 后端" 10; then
                stopped=$((stopped + 1))
            else
                failed=$((failed + 1))
            fi
        fi
    done

    if [ "$DRY_RUN" != true ]; then
        print_success "后端进程: 已停止 $stopped 个，失败 $failed 个"
    fi
}

# 停止由 run.sh 启动的前端进程
stop_web_frontend() {
    if [ ! -f "$WEB_PID_FILE" ]; then
        return 0
    fi

    local web_pid=$(cat "$WEB_PID_FILE")
    print_info "停止前端开发服务器 (PID: $web_pid)..."

    if [ "$DRY_RUN" = true ]; then
        echo "  将停止前端开发服务器 (PID: $web_pid)"
        return 0
    fi

    if graceful_stop "$web_pid" "前端开发服务器" 5; then
        rm -f "$WEB_PID_FILE"
        print_success "前端开发服务器已停止"
    else
        rm -f "$WEB_PID_FILE"
        print_warning "前端进程 PID 文件已清理"
    fi
}

# 停止所有 vite 相关进程
stop_vite_processes() {
    print_info "正在停止 Vite 前端进程..."

    # 查找所有 vite 相关进程（优先匹配 shepherd 项目目录）
    # 方法1: 查找 shepherd 项目目录下的 vite 进程
    local vite_pids=$(pgrep -f "vite.*shepherd|shepherd.*vite" 2>/dev/null || true)

    # 方法2: 如果没找到，尝试查找工作目录在 shepherd 项目下的进程
    if [ -z "$vite_pids" ]; then
        # 使用 lsof 查找在 shepherd web 目录下运行的进程
        vite_pids=$(lsof -t +D "${PROJECT_DIR}/web" 2>/dev/null | grep -v "^$" || true)
    fi

    # 方法3: 查找包含 shepherd 路径的 vite 进程
    if [ -z "$vite_pids" ]; then
        vite_pids=$(pgrep -f "${PROJECT_DIR}/web.*vite|vite.*${PROJECT_DIR}/web" 2>/dev/null || true)
    fi

    if [ -z "$vite_pids" ]; then
        print_warning "未发现运行中的 Vite 前端进程"
        return 0
    fi

    local stopped=0
    local failed=0

    for pid in $vite_pids; do
        # 获取进程命令行
        local cmdline=$(ps -p "$pid" -o command= 2>/dev/null || echo "")

        if [ "$DRY_RUN" = true ]; then
            echo "  将停止前端进程 (PID: $pid) - $cmdline"
            stopped=$((stopped + 1))
        else
            if graceful_stop "$pid" "前端进程 ($cmdline)" 5; then
                stopped=$((stopped + 1))
            else
                failed=$((failed + 1))
            fi
        fi
    done

    if [ "$DRY_RUN" != true ]; then
        print_success "Vite 进程: 已停止 $stopped 个，失败 $failed 个"
    fi
}

# 通过端口停止进程
stop_by_port() {
    local port=$1
    local pids=$(lsof -ti :$port -sTCP:LISTEN 2>/dev/null || true)

    if [ -z "$pids" ]; then
        return 0
    fi

    print_info "停止占用端口 $port 的进程..."

    for pid in $pids; do
        if [ "$DRY_RUN" = true ]; then
            echo "  将停止占用端口 $port 的进程 (PID: $pid)"
        else
            graceful_stop "$pid" "端口 $port 进程" 3
        fi
    done
}

# 停止所有前端端口上的进程
stop_frontend_ports() {
    print_info "检查常见前端端口..."

    for port in "${FRONTEND_PORTS[@]}"; do
        stop_by_port "$port"
    done
}

# 显示所有运行中的进程
show_running_processes() {
    print_info "当前运行中的进程："
    echo ""

    # 后端进程
    local backend_pids=$(pgrep -f "shepherd" 2>/dev/null || true)
    if [ -n "$backend_pids" ]; then
        echo "后端进程:"
        for pid in $backend_pids; do
            local cmdline=$(ps -p "$pid" -o command= 2>/dev/null || echo "")
            echo "  PID $pid: $cmdline"
        done
        echo ""
    fi

    # 前端进程（只显示 shepherd 项目相关的）
    local vite_pids=$(pgrep -f "vite.*shepherd|shepherd.*vite" 2>/dev/null || true)
    if [ -z "$vite_pids" ]; then
        vite_pids=$(pgrep -f "${PROJECT_DIR}/web.*vite|vite.*${PROJECT_DIR}/web" 2>/dev/null || true)
    fi
    if [ -n "$vite_pids" ]; then
        echo "前端进程:"
        for pid in $vite_pids; do
            local cmdline=$(ps -p "$pid" -o command= 2>/dev/null || echo "")
            echo "  PID $pid: $cmdline"
        done
        echo ""
    fi

    # 端口占用
    echo "端口占用:"
    for port in "${FRONTEND_PORTS[@]}"; do
        local port_info=$(lsof -i :$port -sTCP:LISTEN 2>/dev/null || true)
        if [ -n "$port_info" ]; then
            echo "  端口 $port:"
            echo "$port_info" | sed 's/^/    /'
        fi
    done
}

# 清理函数
cleanup() {
    # 清理 PID 文件和日志文件
    if [ "$DRY_RUN" != true ]; then
        rm -f "$WEB_PID_FILE"
        rm -f "$WEB_LOG_FILE"
    fi
}

# 主函数
main() {
    local VERBOSE=false
    local DRY_RUN=false
    local FORCE=false

    # 解析参数
    while [[ $# -gt 0 ]]; do
        case $1 in
            -h|--help)
                show_help
                exit 0
                ;;
            -v|--verbose)
                VERBOSE=true
                shift
                ;;
            --dry-run)
                DRY_RUN=true
                shift
                ;;
            --force)
                FORCE=true
                shift
                ;;
            *)
                print_error "未知参数: $1"
                show_help
                exit 1
                ;;
        esac
    done

    # 显示标题
    echo ""
    echo "=========================================="
    echo "  🛑 停止所有 Shepherd 进程"
    echo "=========================================="
    echo ""

    if [ "$VERBOSE" = true ] || [ "$DRY_RUN" = true ]; then
        show_running_processes
        echo ""
    fi

    if [ "$DRY_RUN" = true ]; then
        print_warning "模拟运行模式 - 不会实际停止进程"
        echo ""
    fi

    # 停止后端进程
    stop_backend
    echo ""

    # 停止前端进程
    stop_web_frontend
    echo ""

    stop_vite_processes
    echo ""

    # 清理端口占用（如果使用 --force）
    if [ "$FORCE" = true ]; then
        stop_frontend_ports
        echo ""
    fi

    # 清理文件
    cleanup

    # 显示结果
    if [ "$DRY_RUN" != true ]; then
        echo ""
        print_success "所有进程已停止"

        # 验证是否还有残留进程
        local remaining_backend=$(pgrep -f "shepherd" 2>/dev/null || true)
        local remaining_frontend=$(pgrep -f "(vite|npm.*dev|pnpm.*dev)" 2>/dev/null || true)

        if [ -n "$remaining_backend" ] || [ -n "$remaining_frontend" ]; then
            echo ""
            print_warning "发现残留进程："
            [ -n "$remaining_backend" ] && echo "  后端: $remaining_backend"
            [ -n "$remaining_frontend" ] && echo "  前端: $remaining_frontend"
            echo ""
            print_info "使用 --force 参数强制停止所有进程"
        fi
    fi

    echo ""
}

# 运行主函数
main "$@"
