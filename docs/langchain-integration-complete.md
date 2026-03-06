# LangChainGo 集成完成

## 集成状态

✅ **集成完成** - LangChainGo 已成功集成到 Shepherd 主程序

## 已完成的修改

### 1. 核心代码修改

#### `cmd/shepherd/main.go`
- 添加 `langchain.Manager` 和 `langchain.Handler` 字段到 App 结构
- 在 `Initialize()` 中创建 LangChainGo 组件
- 在创建 Server 后注册 LangChainGo API 路由

#### `internal/server/server.go`
- 添加 `langchainHandler` 字段到 Server 结构
- 导入 `internal/langchain` 包
- 实现 `RegisterLangChainHandler()` 方法

### 2. API 端点

LangChainGo API 已注册到以下端点：

| 端点 | 方法 | 描述 |
|------|------|------|
| `/api/langchain/prompt` | POST | 简单文本生成（支持提示模板） |
| `/api/langchain/chat` | POST | 聊天完成（支持多轮对话） |
| `/api/langchain/stream` | POST | 流式生成（Server-Sent Events） |
| `/api/langchain/models` | GET | 列出可用模型 |
| `/api/langchain/stats` | GET | 获取统计信息 |

### 3. 启动流程

```
main.go Initialize()
    ↓
创建 Model Manager
    ↓
创建 LangChainGo Manager
    ↓
创建 LangChainGo Handler
    ↓
创建 Server
    ↓
注册 LangChainGo 路由
    ↓
启动 HTTP Server
```

## API 架构对比

### 方案 1: OpenAI API (已存在)

```
前端 → /v1/chat/completions
    ↓
OpenAI Handler
    ↓
查找模型端口
    ↓
转发到 llama.cpp
    ↓
返回 OpenAI 格式
```

**用途**: 标准 OpenAI SDK 兼容

### 方案 2: LangChainGo API (新增)

```
前端 → /api/langchain/chat
    ↓
LangChainGo Handler
    ↓
Manager.GetLLM()
    ↓
LlamaCPP.GenerateContent()
    ↓
HTTP 请求到 llama.cpp
    ↓
返回 LangChainGo 格式
```

**用途**: 高级 AI 功能（提示模板、链式调用、智能体）

## 使用示例

### 1. 简单提示（带模板）

```bash
curl -X POST http://localhost:8080/api/langchain/prompt \
  -H "Content-Type: application/json" \
  -d '{
    "model_id": "Qwen3.5-0.8B-UD-Q8_K_XL",
    "prompt": "Translate to {lang}: {text}",
    "input": {"lang": "Chinese", "text": "Hello World"},
    "options": {"temperature": 0.7, "max_tokens": 200}
  }'
```

**响应**:
```json
{
  "model_id": "Qwen3.5-0.8B-UD-Q8_K_XL",
  "response": "你好，世界"
}
```

### 2. 聊天完成

```bash
curl -X POST http://localhost:8080/api/langchain/chat \
  -H "Content-Type: application/json" \
  -d '{
    "model_id": "Qwen3.5-0.8B-UD-Q8_K_XL",
    "messages": [
      {"role": "system", "content": "You are a helpful assistant"},
      {"role": "user", "content": "What is 2+2?"}
    ],
    "options": {"temperature": 0.7}
  }'
```

**响应**:
```json
{
  "model_id": "Qwen3.5-0.8B-UD-Q8_K_XL",
  "response": {
    "choices": [{
      "content": "2+2 equals 4.",
      "stop_reason": "stop"
    }]
  }
}
```

### 3. 流式生成

```bash
curl -X POST http://localhost:8080/api/langchain/stream \
  -H "Content-Type: application/json" \
  -d '{
    "model_id": "Qwen3.5-0.8B-UD-Q8_K_XL",
    "messages": [{"role": "user", "content": "Tell me a story"}]
  }'
```

**响应** (SSE):
```
data: {"content": "Once", "stop_reason": null}
data: {"content": " upon", "stop_reason": null}
data: {"content": " a", "stop_reason": null}
event: end
data: done
```

### 4. 列出模型

```bash
curl http://localhost:8080/api/langchain/models
```

**响应**:
```json
{
  "models": ["Qwen3.5-0.8B-UD-Q8_K_XL", "llama-3-8b"],
  "total": 2
}
```

### 5. 获取统计

```bash
curl http://localhost:8080/api/langchain/stats
```

**响应**:
```json
{
  "total_instances": 2,
  "active_models": ["Qwen3.5-0.8B-UD-Q8_K_XL"],
  "cache_hits": 42,
  "cache_misses": 5
}
```

## 代码示例

### Go 程序中使用

```go
package main

import (
    "context"
    "fmt"
    "github.com/tmc/langchaingo/llms"
    "github.com/shepherd-project/shepherd/Shepherd/internal/langchain"
)

func main() {
    // 创建 LLM 实例
    llm, err := langchain.NewLlamaCPP(
        "http://localhost:8080",
        "Qwen3.5-0.8B-UD-Q8_K_XL",
        langchain.WithTemperature(0.7),
        langchain.WithMaxTokens(200),
    )
    if err != nil {
        panic(err)
    }

    // 简单文本生成
    ctx := context.Background()
    response, err := llm.Call(ctx, "What is the capital of France?")
    if err != nil {
        panic(err)
    }
    fmt.Println(response) // Paris

    // 聊天完成
    messages := []llms.MessageContent{
        {
            Role: llms.ChatMessageTypeSystem,
            Parts: []llms.ContentPart{
                llms.TextPart("You are a helpful assistant"),
            },
        },
        {
            Role: llms.ChatMessageTypeHuman,
            Parts: []llms.ContentPart{
                llms.TextPart("Hello!"),
            },
        },
    }
    response, err = llm.GenerateContent(ctx, messages)
    fmt.Println(response.Choices[0].Content)

    // 流式生成
    respChan, err := llm.GenerateContentStream(ctx, messages)
    for resp := range respChan {
        fmt.Print(resp.Choices[0].Content)
    }
}
```

## 测试

### 单元测试

```bash
# 运行所有 LangChainGo 测试
go test ./internal/langchain/... -v

# 运行特定测试
go test ./internal/langchain -run TestLlamaCPPNew -v

# 测试覆盖率
go test ./internal/langchain/... -cover
```

### 集成测试

```bash
# 启动 Shepherd
./scripts/linux/run.sh -b

# 测试简单提示
curl -X POST http://localhost:8080/api/langchain/prompt \
  -H "Content-Type: application/json" \
  -d '{
    "model_id": "Qwen3.5-0.8B-UD-Q8_K_XL",
    "prompt": "Hello, {name}!",
    "input": {"name": "World"}
  }'

# 测试聊天
curl -X POST http://localhost:8080/api/langchain/chat \
  -H "Content-Type: application/json" \
  -d '{
    "model_id": "Qwen3.5-0.8B-UD-Q8_K_XL",
    "messages": [{"role": "user", "content": "Hello"}]
  }'
```

## 文件清单

### 新增文件

| 文件 | 行数 | 描述 |
|------|------|------|
| `internal/langchain/llama.go` | 479 | LangChainGo LLM 实现 |
| `internal/langchain/manager.go` | 185 | LLM 实例管理器 |
| `internal/langchain/handler.go` | 305 | RESTful API 处理器 |
| `internal/langchain/llama_test.go` | 402 | 单元测试 |
| `internal/langchain/example_test.go` | 300 | 使用示例 |
| `docs/langchain-integration-summary.md` | 262 | 集成总结 |
| `docs/api-vs-langchain-comparison.md` | 400+ | API 对比分析 |
| `docs/langchain-integration-complete.md` | 本文档 |

### 修改文件

| 文件 | 修改内容 |
|------|---------|
| `cmd/shepherd/main.go` | 添加 LangChainGo 组件初始化和路由注册 |
| `internal/server/server.go` | 添加 RegisterLangChainHandler 方法 |
| `go.mod` | 添加 langchaingo v0.1.14 依赖 |

## 架构决策

### 为什么分开实现？

1. **保持兼容性**: OpenAI API 完全兼容 OpenAI SDK
2. **功能分离**: LangChainGo 提供增强功能
3. **渐进式迁移**: 用户可以选择使用哪个 API
4. **独立测试**: 两套 API 可以独立测试和优化

### 何时使用哪个 API？

| 场景 | 推荐使用 |
|------|---------|
| 需要完全兼容 OpenAI SDK | `/v1/*` |
| 使用现有的 OpenAI 客户端库 | `/v1/*` |
| 简单的聊天/补全请求 | `/v1/*` 或 `/api/langchain/*` |
| 需要提示模板功能 | `/api/langchain/*` |
| 计划实现链式调用 | `/api/langchain/*` |
| 需要智能体功能 | `/api/langchain/*` |
| 统一管理 LLM 实例 | `/api/langchain/*` |

## 未来扩展

### LangChainGo 独有功能

- [ ] **Chains（链式调用）**
  ```go
  chain := NewChain(
      NewLLMChain(llm, "template1"),
      NewLLMChain(llm, "template2"),
  )
  ```

- [ ] **Agents（智能代理）**
  ```go
  agent := NewReActAgent(llm, tools)
  result, err := agent.Run(ctx, "What's the weather in Beijing?")
  ```

- [ ] **Tools（工具调用）**
  ```go
  tools := []Tool{
      NewCalculatorTool(),
      NewSearchTool(),
  }
  ```

- [ ] **Memory（对话记忆）**
  ```go
  memory := NewConversationBufferWindowMemory(5)
  ```

- [ ] **Prompt 模板管理**
  ```go
  template := NewPromptTemplate("Translate {text} to {lang}")
  ```

- [ ] **Vector Stores（向量存储）**
  ```go
  store := NewChromaStore("http://localhost:8000")
  ```

## 编译和运行

```bash
# 编译
make build

# 运行（使用默认配置）
./build/shepherd

# 运行（自动编译）
./scripts/linux/run.sh -b

# 检查 LangChainGo 是否启用
curl http://localhost:8080/api/langchain/stats
```

## 日志输出

启动时会看到以下日志：

```
✓ LangChainGo 管理器已创建
✓ LangChainGo API 处理器已创建
✓ LangChainGo API 已启用
✓ LangChainGo API 路由已注册: /api/langchain/*
```

## 故障排查

### 问题：LangChainGo API 不可用

**检查**：
```bash
# 检查路由是否注册
curl http://localhost:8080/api/langchain/stats

# 检查日志
grep "LangChainGo" ./logs/shepherd-*.log
```

### 问题：模型未加载

**检查**：
```bash
# 列出已加载的模型
curl http://localhost:8080/api/models/loaded

# 加载模型
curl -X POST http://localhost:8080/api/models/{model_id}/load
```

### 问题：端口冲突

**解决**：
- LangChainGo 不使用额外端口
- 通过已有的 llama.cpp 端口通信
- 确保模型已加载且端口可用

## 性能考虑

- **缓存**: LLM 实例已缓存，避免重复创建
- **连接池**: 使用 Go http.Client，支持连接复用
- **超时控制**: 继承请求上下文的超时设置
- **流式传输**: SSE 实时转发，无缓冲延迟

## 总结

✅ **集成完成**
- LangChainGo v0.1.14 集成到 Shepherd
- 5 个 API 端点可用
- 完整的单元测试和示例
- 编译成功，无错误

🎯 **架构清晰**
- OpenAI API: 标准兼容
- LangChainGo API: 高级功能
- 两者独立共存，互不干扰

🚀 **准备就绪**
- 可以立即使用
- 支持提示模板
- 支持流式生成
- 支持统计信息

---

**集成日期**: 2026-03-05
**LangChainGo 版本**: v0.1.14
**Shepherd 版本**: v0.5.1
**测试模型**: Qwen3.5-0.8B-UD-Q8_K_XL
