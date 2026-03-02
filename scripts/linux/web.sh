#!/bin/bash
# Shepherd Web 前端运行脚本

set -e

# 颜色输出
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# 获取项目根目录
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# 从 linux/ 子目录向上两级到达项目根目录
PROJECT_DIR="$(dirname "$(dirname "$SCRIPT_DIR")")"
WEB_DIR="${PROJECT_DIR}/web"

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
🐏 Shepherd Web 前端

用法: $0 [命令] [选项]

命令:
    dev         启动开发服务器 (默认)
    build       构建生产版本
    preview     预览生产构建
    install     安装/重新安装依赖
    clean       清理构建文件
    fix         修复依赖问题
    check       检查依赖状态

选项:
    -b, --build    启动前先构建 (仅适用于 dev 命令)
    -h, --help     显示此帮助信息

示例:
    $0 dev                 # 启动开发服务器
    $0 dev -b              # 构建后启动开发服务器
    $0 -b                  # 等同于 $0 dev -b
    $0 build              # 构建生产版本
    $0 preview            # 预览构建结果
    $0 install            # 安装依赖
    $0 fix                # 修复依赖问题

说明:
    端口配置请修改 config/node/web.config.yaml 或 config/example/web.config.yaml 中的 server.port

EOF
}

# 检查依赖
check_dependencies() {
    if ! command -v node &> /dev/null; then
        print_error "Node.js 未安装"
        exit 1
    fi

    if ! command -v npm &> /dev/null; then
        print_error "npm 未安装"
        exit 1
    fi

    # 检查 node_modules 是否存在或是否完整
    if [ ! -d "${WEB_DIR}/node_modules" ]; then
        print_warning "node_modules 目录不存在，正在安装依赖..."
        install_dependencies
    elif [ ! -f "${WEB_DIR}/node_modules/.package-lock.json" ] && [ ! -f "${WEB_DIR}/package-lock.json" ]; then
        print_warning "依赖可能不完整，正在重新安装..."
        install_dependencies
    else
        # 检查关键依赖是否存在
        local missing_deps=false
        for dep in "react" "react-dom" "vite"; do
            if [ ! -d "${WEB_DIR}/node_modules/${dep}" ]; then
                missing_deps=true
                break
            fi
        done

        if [ "$missing_deps" = true ]; then
            print_warning "关键依赖缺失，正在安装..."
            install_dependencies
        fi
    fi
}

# 安装依赖
install_dependencies() {
    print_info "安装 Web 前端依赖..."
    cd "$WEB_DIR"

    # 清理旧的 node_modules（如果需要）
    if [ -d "node_modules" ] && [ ! -f "package-lock.json" ]; then
        print_warning "检测到 node_modules 但缺少 package-lock.json，清理后重新安装..."
        rm -rf node_modules
    fi

    # 使用 npm ci 如果有 lock 文件，否则使用 npm install
    if [ -f "package-lock.json" ]; then
        print_info "使用 npm ci 安装依赖（快速、可靠）..."
        npm ci
    else
        print_info "使用 npm install 安装依赖..."
        npm install
    fi

    print_success "依赖安装完成"
}

# 清理构建文件
clean_build() {
    print_info "清理 Web 构建文件..."
    cd "$WEB_DIR"
    rm -rf dist node_modules/.vite
    print_success "清理完成"
}

# 修复依赖
fix_dependencies() {
    print_info "修复依赖..."
    cd "$WEB_DIR"

    # 备份当前安装
    if [ -d "node_modules" ]; then
        print_info "备份当前 node_modules..."
        mv node_modules node_modules.backup.$(date +%s)
    fi

    # 重新安装
    install_dependencies

    # 清理备份
    print_success "依赖修复完成"
}

# 检查依赖状态
check_dependencies_status() {
    print_info "检查依赖状态..."
    cd "$WEB_DIR"

    local issues=0

    # 检查 Node.js 和 npm
    if ! command -v node &> /dev/null; then
        print_error "Node.js 未安装"
        issues=$((issues + 1))
    else
        local node_version=$(node --version)
        print_success "Node.js: ${node_version}"
    fi

    if ! command -v npm &> /dev/null; then
        print_error "npm 未安装"
        issues=$((issues + 1))
    else
        local npm_version=$(npm --version)
        print_success "npm: ${npm_version}"
    fi

    # 检查 node_modules
    if [ ! -d "node_modules" ]; then
        print_warning "node_modules 目录不存在"
        issues=$((issues + 1))
    else
        local dep_count=$(ls node_modules 2>/dev/null | wc -l)
        print_success "node_modules 存在 (${dep_count} 个包)"
    fi

    # 检查 package-lock.json
    if [ -f "package-lock.json" ]; then
        print_success "package-lock.json 存在"
    else
        print_warning "package-lock.json 不存在"
    fi

    # 检查关键依赖
    print_info "检查关键依赖..."
    local critical_deps=("react" "react-dom" "vite" "react-router-dom")
    for dep in "${critical_deps[@]}"; do
        if [ -d "node_modules/${dep}" ]; then
            local version=$(cat "node_modules/${dep}/package.json" 2>/dev/null | grep '"version"' | head -1)
            print_success "  ✓ ${dep} ${version}"
        else
            print_error "  ✗ ${dep} 缺失"
            issues=$((issues + 1))
        fi
    done

    echo ""
    if [ $issues -eq 0 ]; then
        print_success "所有依赖正常"
    else
        print_warning "发现 ${issues} 个问题，运行 '$0 fix' 修复"
        return 1
    fi
}

# 启动开发服务器
run_dev() {
    print_info "启动 Web 开发服务器..."

    # 1. 同步配置文件
    print_info "同步配置文件..."
    local sync_script="$SCRIPT_DIR/sync-web-config.sh"
    if [ -f "$sync_script" ]; then
        "$sync_script"
    else
        print_warning "配置同步脚本不存在，跳过"
    fi

    # 2. 读取配置的端口
    local config_file="$WEB_DIR/public/config.yaml"
    local port=$(grep -oP 'port:\s*\K\d+' "$config_file" 2>/dev/null || echo "3000")
    print_info "配置端口: $port"

    # 3. 检查端口占用
    if lsof -Pi :$port -sTCP:LISTEN -t >/dev/null 2>&1; then
        print_warning "端口 $port 已被占用"
        print_info "正在尝试停止占用进程..."

        local killed_pids=$(lsof -ti :$port -sTCP:LISTEN 2>/dev/null)
        if [ -n "$killed_pids" ]; then
            echo "$killed_pids" | xargs -r kill -9 2>/dev/null || true
            sleep 1

            # 检查是否成功停止
            if lsof -Pi :$port -sTCP:LISTEN -t >/dev/null 2>&1; then
                print_error "无法停止占用端口的进程，请手动停止"
                print_info "运行以下命令查看占用进程:"
                echo "  lsof -i :$port"
                exit 1
            else
                print_success "已停止占用端口的进程"
            fi
        fi
    fi

    # 4. 进入 web 目录
    cd "$WEB_DIR"

    # 5. 显示访问地址
    print_info "启动开发服务器 (端口: $port)..."
    print_info "访问地址:"
    print_info "  - http://localhost:$port"
    print_info "  - http://10.0.0.193:$port"

    # 6. 启动 Vite（不带 --port 参数，从配置文件读取）
    npm run dev
}

# 构建生产版本
run_build() {
    print_info "构建 Web 生产版本..."
    cd "$WEB_DIR"
    npm run build
    print_success "构建完成，输出目录: web/dist/"
}

# 预览生产构建
run_preview() {
    print_info "预览 Web 生产构建..."
    cd "$WEB_DIR"

    # 创建临时文件来存储进程ID
    local pid_file="/tmp/shepherd-web-preview.pid"
    local log_file="/tmp/shepherd-web-preview.log"

    # 清理函数
    cleanup() {
        local exit_code=$?
        print_info "正在关闭 Web 预览服务器..."

        # 读取PID并终止进程
        if [ -f "$pid_file" ]; then
            local pid=$(cat "$pid_file")
            if kill -0 "$pid" 2>/dev/null; then
                print_info "发送 SIGTERM 到进程 $pid..."
                kill -TERM "$pid" 2>/dev/null || true

                # 等待进程优雅退出（最多5秒）
                local count=0
                while kill -0 "$pid" 2>/dev/null && [ $count -lt 50 ]; do
                    sleep 0.1
                    count=$((count + 1))
                done

                # 如果进程仍在运行，强制终止
                if kill -0 "$pid" 2>/dev/null; then
                    print_warning "进程未响应，强制终止..."
                    kill -KILL "$pid" 2>/dev/null || true
                fi
            fi
            rm -f "$pid_file"
        fi

        # 清理临时文件
        rm -f "$log_file"

        if [ $exit_code -eq 0 ]; then
            print_success "Web 预览服务器已优雅关闭"
        else
            print_warning "Web 预览服务器已关闭 (退出码: $exit_code)"
        fi
        exit $exit_code
    }

    # 捕获退出信号
    trap cleanup EXIT INT TERM HUP QUIT

    # 启动预览服务器到后台
    npm run preview > "$log_file" 2>&1 &
    local npm_pid=$!
    echo $npm_pid > "$pid_file"

    print_info "Web 预览服务器已启动 (PID: $npm_pid)"
    print_info "日志文件: $log_file"
    print_success "按 Ctrl+C 停止服务器"

    # 等待进程或信号
    wait $npm_pid 2>/dev/null
    local exit_code=$?

    # 如果进程异常退出，显示日志
    if [ $exit_code -ne 0 ] && [ $exit_code -ne 143 ]; then
        print_error "预览服务器异常退出，查看日志:"
        tail -20 "$log_file" >&2
    fi

    # 清理会由 trap 自动处理
    exit $exit_code
}

# 主函数
main() {
    local command=""
    local build_first=false

    # 解析参数
    while [[ $# -gt 0 ]]; do
        case $1 in
            dev|build|preview|install|clean|fix|check)
                command="$1"
                shift
                ;;
            -b|--build)
                build_first=true
                shift
                ;;
            -h|--help)
                show_help
                exit 0
                ;;
            *)
                print_error "未知参数: $1"
                show_help
                exit 1
                ;;
        esac
    done

    # 默认命令
    if [ -z "$command" ]; then
        command="dev"
    fi

    # 执行命令（check 命令不需要先检查依赖）
    if [ "$command" != "check" ]; then
        check_dependencies
    fi

    # 如果指定了 -b 参数，先执行构建
    if [ "$build_first" = true ]; then
        if [ "$command" = "dev" ]; then
            print_info "构建前端..."
            run_build
            print_success "构建完成"
        else
            print_warning "-b 选项仅对 dev 命令有效"
        fi
    fi

    # 执行命令
    case "$command" in
        dev)
            run_dev
            ;;
        build)
            run_build
            ;;
        preview)
            run_preview
            ;;
        install)
            install_dependencies
            ;;
        clean)
            clean_build
            ;;
        fix)
            fix_dependencies
            ;;
        check)
            check_dependencies_status
            ;;
    esac
}

# 运行主函数
main "$@"
