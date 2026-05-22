# Shepherd

[![Go](https://img.shields.io/badge/Go-1.25+-00ADD8?logo=go)](https://go.dev)
[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](LICENSE)

A lightweight, high-performance distributed llama.cpp model management system.

[中文文档](docs/README_zh.md)

---

## Features

- Fast startup (<500ms), low memory footprint (~30MB)
- Single binary with no runtime dependencies
- Distributed architecture with Master-Client node deployment
- Multi-protocol API compatibility: OpenAI, Anthropic, Ollama, LM Studio
- Web UI with i18n support (English / Chinese)

## Model Management

- Auto-scan GGUF model files across multiple directories
- Load/unload models with fine-grained parameters:
  - Context size, batch size, threads, GPU layers
  - Sampling: temperature, Top-P, Top-K, repeat penalty, Min-P
  - Performance: Flash Attention, memory lock, UBatch, parallel slots
  - KV cache type (K/V), unified cache
  - Template system, GPU multi-device configuration
- Model favorites, aliases, split-file auto-detection
- Vision model (mmproj) support

## Distributed Architecture

| Role | Description |
|------|-------------|
| **Hybrid** (default) | Acts as both Master and Client |
| **Master** | Central management node |
| **Client** | GPU worker node, registers to a Master |

Nodes can switch roles at runtime. The cluster provides heartbeat monitoring (5s interval), resource reporting (CPU/GPU/memory/VRAM), and resource-aware scheduling.

---

## Quick Start

### Build from source

```bash
git clone https://github.com/simonxluo/Shepherd.git
cd Shepherd
make build
```

Pre-built binaries are available on the [Releases](https://github.com/simonxluo/Shepherd/releases) page.

### Configuration

Config files are located at `config/node/*.config.yaml`. Node role is determined by the `node.role` field:

| node.role | Description |
|-----------|-------------|
| `hybrid` | Hybrid mode (default) |
| `master` | Master node |
| `client` | Client worker node |

### Run

```bash
# Default (hybrid mode)
./build/shepherd

# With custom config
./build/shepherd serve --config config/node/server.config.yaml

# Start with frontend dev server
./build/shepherd serve --web

# Build frontend then start
./build/shepherd serve --build --web

# Show version
./build/shepherd version
```

Web UI: http://localhost:9190

---

## Cluster Deployment

```
┌─────────────────┐     ┌─────────────────┐     ┌─────────────────┐
│   Master Node   │◄────┤  Hybrid Node    │◄────┤  Client Node    │
│   (Port 9190)   │     │ (Port 9190+9191)│     │                 │
└────────┬────────┘     └────────┬────────┘     └─────────────────┘
         │                       │
         ▼                       ▼
┌─────────────────┐     ┌─────────────────┐
│  Client Node 1  │     │  Client Node 2  │
│   (GPU Server)  │     │   (GPU Server)  │
└─────────────────┘     └─────────────────┘
```

1. Start a Master or Hybrid node:
   ```bash
   ./build/shepherd serve --config config/node/server.config.yaml
   ```

2. Start Client nodes (set `node.role: client` and `node.client_role.master_address` in config):
   ```bash
   ./build/shepherd serve --config config/node/client.config.yaml
   ```

3. Check cluster status:
   ```bash
   curl http://master:9190/api/nodes
   ```

---

## API Example

```python
from openai import OpenAI

client = OpenAI(
    base_url="http://localhost:9190/v1",
    api_key="dummy"
)

response = client.chat.completions.create(
    model="llama-2-7b-chat",
    messages=[{"role": "user", "content": "Hello!"}]
)

print(response.choices[0].message.content)
```

---

## Development

### Backend (Go)

```bash
make build       # Build
make lint        # Lint
make fmt         # Format
make tidy        # Tidy modules
make build-all   # Cross-platform build
make swag        # Generate Swagger docs
```

### Frontend (React + TypeScript)

```bash
cd web
npm install      # Install dependencies
npm run dev      # Dev server (port 3000)
npm run build    # Production build
npm run lint     # ESLint
npm run test     # Unit tests
```

---

## License

Apache License 2.0. See [LICENSE](LICENSE).

## Acknowledgments

- [llama.cpp](https://github.com/ggerganov/llama.cpp)

## Links

- [Issues](https://github.com/simonxluo/Shepherd/issues)
- [Discussions](https://github.com/simonxluo/Shepherd/discussions)
- [Changelog](CHANGELOG.md)
