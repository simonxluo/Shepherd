#!/bin/bash
# Shepherd 发布打包脚本
# 用法: ./scripts/release.sh [version]

set -e

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

VERSION=${1:-"dev"}
PROJECT_NAME="shepherd"
BUILD_DIR="build"
RELEASE_DIR="release"

echo ""
echo -e "${GREEN}  Shepherd Release Package${NC}"
echo ""
echo ""
echo "版本: ${VERSION}"
echo ""

# 检查构建文件是否存在
if [ ! -d "${BUILD_DIR}" ]; then
    echo -e "${RED}错误: 构建目录不存在，请先运行编译脚本${NC}"
    exit 1
fi

# 清理旧的发布文件
echo -e "${YELLOW}清理旧的发布文件...${NC}"
rm -rf "${RELEASE_DIR}"
mkdir -p "${RELEASE_DIR}"

# 为每个平台创建发布包
cd "${BUILD_DIR}"

for binary in ${PROJECT_NAME}-*; do
    if [ -f "$binary" ] && [ -x "$binary" ]; then
        # 解析平台和架构
        if [[ $binary =~ ${PROJECT_NAME}-([^-]+)-([^.]+) ]]; then
            OS="${BASH_REMATCH[1]}"
            ARCH="${BASH_REMATCH[2]}"
            EXT=""

            # Windows 可执行文件扩展名
            if [ "$OS" = "windows" ]; then
                binary="${binary}.exe"
            fi

            if [ ! -f "$binary" ]; then
                continue
            fi

            PACKAGE_NAME="${PROJECT_NAME}-${VERSION}-${OS}-${ARCH}"
            PACKAGE_DIR="${PACKAGE_NAME}"
            mkdir -p "../${RELEASE_DIR}/${PACKAGE_DIR}"

            # 复制二进制文件
            cp "$binary" "../${RELEASE_DIR}/${PACKAGE_DIR}/${PROJECT_NAME}${EXT}"

            # 创建启动脚本
            if [ "$OS" = "windows" ]; then
                cat > "../${RELEASE_DIR}/${PACKAGE_DIR}/start.bat" << 'EOF'
@echo off
REM Shepherd 启动脚本 (Windows)
REM 节点角色通过配置文件 config/server.config.yaml 的 node.role 字段设置

shepherd.exe serve %*
EOF
            else
                cat > "../${RELEASE_DIR}/${PACKAGE_DIR}/start.sh" << 'EOF'
#!/bin/bash
# Shepherd 启动脚本 (Linux/macOS)
# 节点角色通过配置文件 config/server.config.yaml 的 node.role 字段设置

chmod +x shepherd
./shepherd serve "$@"
EOF
                chmod +x "../${RELEASE_DIR}/${PACKAGE_DIR}/start.sh"
            fi

            # 复制配置文件示例（提供所有模式）
            mkdir -p "../${RELEASE_DIR}/${PACKAGE_DIR}/config"

            # server.config.yaml (单机模式)
            cat > "../${RELEASE_DIR}/${PACKAGE_DIR}/config/server.config.yaml" << 'EOF'
# Shepherd 配置文件

node:
  role: hybrid
  id: auto

server:
  web_port: 9190
  anthropic_port: 9170
  ollama_port: 11434
  lmstudio_port: 1234
  host: "0.0.0.0"
  read_timeout: 60
  write_timeout: 60

model:
  paths:
    - "./models"
    - "~/.cache/huggingface/hub"
  auto_scan: true
  scan_interval: 0

download:
  directory: "./downloads"
  max_concurrent: 4
  chunk_size: 1048576
  retry_count: 3
  timeout: 300

security:
  api_key_enabled: false
  api_key: ""
  cors_enabled: true
  allowed_origins: ["*"]

log:
  level: "info"
  format: "json"
  output: "both"
  directory: "./logs"
  max_size: 100
  max_backups: 3
  max_age: 7
  compress: true
EOF

            # master.config.yaml (Master 模式)
            cat > "../${RELEASE_DIR}/${PACKAGE_DIR}/config/master.config.yaml" << 'EOF'
# Shepherd Master 模式配置

node:
  role: master
  id: auto

server:
  web_port: 9190
  anthropic_port: 9170
  ollama_port: 11434
  lmstudio_port: 1234
  host: "0.0.0.0"
  read_timeout: 60
  write_timeout: 60

model:
  paths:
    - "./models"
    - "~/.cache/huggingface/hub"
  auto_scan: true
  scan_interval: 0

master:
  auto_scan: true
  scan_interval: 300
  scan_networks:
    - "192.168.1.0/24"
    - "10.0.0.0/24"
  scheduling_policy: "least-load"
  heartbeat_timeout: 60

download:
  directory: "./downloads"
  max_concurrent: 4
  chunk_size: 1048576
  retry_count: 3
  timeout: 300

security:
  api_key_enabled: false
  api_key: ""
  cors_enabled: true
  allowed_origins: ["*"]

log:
  level: "info"
  format: "json"
  output: "both"
  directory: "./logs"
  max_size: 100
  max_backups: 3
  max_age: 7
  compress: true
EOF

            # client.config.yaml (Client 模式)
            cat > "../${RELEASE_DIR}/${PACKAGE_DIR}/config/client.config.yaml" << 'EOF'
# Shepherd Client 模式配置

node:
  role: client
  id: auto
  name: "client-1"
  tags:
    - "gpu"
    - "rocm"

master:
  address: "http://192.168.1.100:9190"
  token: ""
  heartbeat_interval: 30
  reconnect_delay: 5

model:
  paths:
    - "./models"
    - "~/.cache/huggingface/hub"
  auto_scan: false

conda:
  enabled: false
  env_name: "shepherd"

resource:
  max_models: 1
  max_memory_percent: 80

log:
  level: "info"
  format: "json"
  output: "both"
  directory: "./logs"
  max_size: 100
  max_backups: 3
  max_age: 7
  compress: true
EOF

            # 创建 README
            cat > "../${RELEASE_DIR}/${PACKAGE_DIR}/README.txt" << EOF
Shepherd v${VERSION}


Shepherd 是一个轻量级的 llama.cpp 模型管理系统，支持多 API 兼容和分布式部署。

快速开始
--------

1. 编辑配置文件:
   - 默认: config/server.config.yaml (node.role 设为 hybrid/master/client)
   - Master 示例: config/master.config.yaml
   - Client 示例: config/client.config.yaml

2. 运行 Shepherd:
EOF
            if [ "$OS" = "windows" ]; then
                echo "   start.bat" >> "../${RELEASE_DIR}/${PACKAGE_DIR}/README.txt"
            else
                echo "   chmod +x start.sh" >> "../${RELEASE_DIR}/${PACKAGE_DIR}/README.txt"
                echo "   ./start.sh" >> "../${RELEASE_DIR}/${PACKAGE_DIR}/README.txt"
            fi

            cat >> "../${RELEASE_DIR}/${PACKAGE_DIR}/README.txt" << 'EOF'

3. 访问 Web UI:
   http://localhost:9190

节点角色
--------

通过配置文件 config/server.config.yaml 的 node.role 字段设置:

  hybrid (默认)   混合模式
  master          Master 模式 - 管理多个 Client 节点
  client          Client 模式 - 作为工作节点

更多信息
--------

GitHub: https://github.com/shepherd-project/shepherd
文档: https://github.com/shepherd-project/shepherd/docs
EOF

            # 打包
            cd "../${RELEASE_DIR}"
            echo -e "${YELLOW}打包 ${PACKAGE_NAME}...${NC}"

            if [ "$OS" = "windows" ]; then
                zip -r "${PACKAGE_NAME}.zip" "${PACKAGE_DIR}" > /dev/null
                echo -e "${GREEN}✓ ${PACKAGE_NAME}.zip${NC}"
            else
                tar -czf "${PACKAGE_NAME}.tar.gz" "${PACKAGE_DIR}"
                echo -e "${GREEN}✓ ${PACKAGE_NAME}.tar.gz${NC}"
            fi

            # 清理临时目录
            rm -rf "${PACKAGE_DIR}"
        fi
    fi
done

cd "${BUILD_DIR}"

# 复制校验和文件到发布目录
if [ -f "SHA256SUMS" ]; then
    cp "SHA256SUMS" "../${RELEASE_DIR}/"
    echo -e "${GREEN}✓ SHA256SUMS 已复制${NC}"
fi

cd ..

echo ""
echo ""
echo -e "${GREEN}  打包完成${NC}"
echo ""
echo ""
echo -e "${YELLOW}发布文件:${NC}"
ls -lh "${RELEASE_DIR}/" | grep -E "\.(zip|tar\.gz|SHA256SUMS)$"
echo ""
echo -e "${YELLOW}发布目录: ${RELEASE_DIR}/${NC}"
echo ""
