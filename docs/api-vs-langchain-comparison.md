# internal/handler vs LangChainGo 对比分析

## 架构定位对比

### `internal/handler` - OpenAI/Anthropic/Ollama 兼容层

**定位**: API 协议代理，专注于与现有 OpenAI 客户端的兼容性

| 特性 | 描述 |
|------|------|
| **核心职责** | 转发标准化的 API 请求到 llama.cpp |
| **目标用户** | 使用 OpenAI/Anthropic SDK 的开发者 |
| **协议** | OpenAI API、Anthropic API、Ollama API |
| **抽象层次** | 低（直接 HTTP 转发） |
| **功能范围** | Chat Completions、Completions、Models |

### `internal/langchain` - LangChainGo 框架集成

**定位**: LangChain 框集成，提供高层次 AI 应用开发抽象

| 特性 | 描述 |
|------|------|
| **核心职责** | 实现 LangChainGo LLM 接口，支持复杂 AI 工作流 |
| **目标用户** | 使用 LangChainGo 框架的 Go 开发者 |
| **接口** | LangChainGo `llms.Model` 接口 |
| **抽象层次** | 高（支持提示模板、链式调用、智能体） |
| **功能范围** | 简单提示、聊天完成、流式生成、统计信息 |

## 详细对比

### 1. 请求流程

#### `internal/handler/openai`
```
客户端请求
    ↓
OpenAI Handler (解析)
    ↓
查找模型端口
    ↓
转发到 llama.cpp:{port}/v1/chat/completions
    ↓
返回 OpenAI 格式响应
```

#### `internal/langchain`
```
客户端请求
    ↓
LangChainGo Handler (解析)
    ↓
Manager.GetLLM() 获取实例
    ↓
LlamaCPP.Call() 或 GenerateContent()
    ↓
HTTP 请求到 llama.cpp:{port}/v1/chat/completions
    ↓
返回 LangChainGo 格式响应
```

### 2. 功能对比

| 功能 | internal/handler | internal/langchain |
|------|--------------|-------------------|
| **简单文本生成** | ✅ (通过 Completions API) | ✅ (SimplePrompt) |
| **聊天完成** | ✅ (Chat Completions) | ✅ (ChatPrompt) |
| **流式生成** | ✅ (SSE) | ✅ (SSE) |
| **模型列表** | ✅ (仅已加载模型) | ✅ (所有模型) |
| **提示模板** | ❌ | ✅ (支持变量替换) |
| **链式调用** | ❌ | 🚧 (未实现) |
| **智能体** | ❌ | 🚧 (未实现) |
| **统计信息** | ❌ | ✅ (实例统计) |

### 3. 数据格式对比

#### 请求格式

**OpenAI 格式** (`/v1/chat/completions`):
```json
{
  "model": "Qwen3.5-0.8B-UD-Q8_K_XL",
  "messages": [
    {"role": "system", "content": "You are helpful"},
    {"role": "user", "content": "Hello"}
  ],
  "temperature": 0.7,
  "max_tokens": 200,
  "stream": false
}
```

**LangChainGo 格式** (`/api/langchain/prompt`):
```json
{
  "model_id": "Qwen3.5-0.8B-UD-Q8_K_XL",
  "prompt": "What is the capital of {country}?",
  "input": {"country": "France"},
  "options": {
    "temperature": 0.7,
    "max_tokens": 200
  }
}
```

#### 响应格式

**OpenAI 响应**:
```json
{
  "id": "chatcmpl-123",
  "object": "chat.completion",
  "created": 1677652288,
  "model": "Qwen3.5-0.8B-UD-Q8_K_XL",
  "choices": [{
    "index": 0,
    "message": {
      "role": "assistant",
      "content": "Hello! How can I help you today?"
    },
    "finish_reason": "stop"
  }],
  "usage": {
    "prompt_tokens": 9,
    "completion_tokens": 12,
    "total_tokens": 21
  }
}
```

**LangChainGo 响应**:
```json
{
  "model_id": "Qwen3.5-0.8B-UD-Q8_K_XL",
  "response": "The capital of France is Paris."
}
```

### 4. 适用场景

#### 使用 `internal/handler` 的场景
- 需要与 OpenAI SDK 兼容
- 使用现有的 OpenAI 客户端库
- 简单的聊天/补全请求
- 不需要提示模板或链式调用

#### 使用 `internal/langchain` 的场景
- 使用 LangChainGo 框架开发
- 需要提示模板功能
- 计划实现链式调用或智能体
- 需要更高层次的抽象
- 统计和管理 LLM 实例

### 5. 技术实现对比

| 方面 | internal/handler | internal/langchain |
|------|--------------|-------------------|
| **HTTP 客户端** | Go `http.Client` | Go `http.Client` |
| **消息转换** | OpenAI → llama.cpp | LangChainGo → llama.cpp |
| **流式支持** | SSE 直接转发 | SSE 转换后转发 |
| **错误处理** | OpenAI 格式错误 | 自定义格式错误 |
| **模型查找** | 模型索引 | 通过 Manager |

## 集成建议

### 1. 互补共存
- **保留** `internal/handler` 作为 OpenAI 兼容层
- **新增** `internal/langchain` 作为 LangChainGo 集成
- 两者共存，服务不同使用场景

### 2. 路由分配
```
/v1/*                          → OpenAI API (internal/handler)
/api/ollama/*                  → Ollama API (internal/handler)
/v1/messages                   → Anthropic API (internal/handler)
/api/langchain/*               → LangChainGo API (internal/langchain)
```

### 3. 共享组件
- **共享 Model Manager**: 两者都使用 `model.Manager` 查找模型
- **共享 Process Manager**: 都通过 `process.Manager` 管理 llama.cpp 进程
- **独立 LLM 实例**: LangChainGo 维护自己的 LLM 实例缓存

### 4. 未来扩展
**LangChainGo 独有功能**:
- Chains（链式调用）
- Agents（智能代理）
- Tools（工具调用）
- Vector Stores（向量存储）
- Memory（对话记忆）

## 代码示例对比

### OpenAI API 调用
```bash
curl -X POST http://localhost:8080/v1/chat/completions \
  -H "Content-Type: application/json" \
  -d '{
    "model": "Qwen3.5-0.8B-UD-Q8_K_XL",
    "messages": [{"role": "user", "content": "Hello"}]
  }'
```

### LangChainGo API 调用
```bash
curl -X POST http://localhost:8080/api/langchain/prompt \
  -H "Content-Type: application/json" \
  -d '{
    "model_id": "Qwen3.5-0.8B-UD-Q8_K_XL",
    "prompt": "Translate to {lang}: {text}",
    "input": {"lang": "Chinese", "text": "Hello"}
  }'
```

## 总结

| 维度 | internal/handler | internal/langchain |
|------|--------------|-------------------|
| **主要目标** | OpenAI 兼容 | LangChainGo 框架集成 |
| **抽象层次** | 低（协议转发） | 高（框架集成） |
| **功能丰富度** | 基础 | 可扩展 |
| **使用场景** | 标准 API 客户端 | AI 应用开发 |
| **是否共存** | ✅ 是 | ✅ 是 |

**最佳实践**:
- 需要标准 OpenAI 兼容 → 使用 `/v1/*`
- 需要高级 AI 功能 → 使用 `/api/langchain/*`
- 两者可以同时使用，互不干扰
