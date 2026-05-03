# 日志 API

## 实时日志文本流

```
GET /api/logs/stream/text
```

Chunked Transfer Encoding 端点，流式推送原始日志文本。连接时先发送历史缓冲区（512KB 循环缓冲区），然后持续推送新日志。

### 查询参数

| 参数 | 类型 | 默认值 | 说明 |
|---|---|---|---|
| no-history | bool | false | 跳过历史日志，只接收新日志 |

### 响应

- Content-Type: `text/plain`
- Transfer-Encoding: `chunked`
- X-Accel-Buffering: `no`（防止 nginx 缓冲）
- 每 30 秒发送空行作为心跳

### 前端功能

LogPanel 组件提供：
- 自动滚动到底部（用户上滑时暂停）
- 正则表达式实时过滤
- 4 级字体大小切换（xxs/xs/small/normal）
- 文本换行切换

### 节点日志

在 master 模式下，可以通过在线节点的直接地址访问其日志流：

```
GET http://{nodeAddress}:{nodePort}/api/logs/stream/text
```

前端通过左侧节点列表选择在线节点，直接连接该节点的日志流端点。

## 日志架构

### 后端

```
Logger.log() → outputs[] → LogMonitor (io.Writer)
                              ├── circularBuffer (512KB)
                              ├── os.Stdout
                              └── subscribers[] → chan []byte
```

- **LogMonitor**：实现 `io.Writer`，拦截所有日志输出，同时写入 stdout 和循环缓冲区，并广播给订阅者
- **circularBuffer**：固定大小的环形缓冲区，自动覆盖最旧数据，O(1) 写入，O(n) 读取
- **HandleLogStreamText**：chunked 流式 handler，先发送缓冲区历史，再订阅实时数据

### 前端

```
useLogStream(url?) → fetch(text/stream) → LogPanel
                                              ├── 自动滚动
                                              ├── 正则过滤
                                              ├── 字体大小
                                              └── 换行切换
```

- **useLogStream**：React hook，使用 `fetch + ReadableStream` 消费 chunked 文本流
- **LogPanel**：纯展示组件，接收 `logData: string`
- **LogsPage**：页面组件，包含"本机"和"节点"两个 tab

## 日志格式

后端支持两种日志格式：

- **Text**（默认）：`[2026-01-15 10:00:00] [file.go:72] INFO 消息内容`
- **JSON**：`{"time":"...","level":"...","msg":"...","caller":"..."}`
