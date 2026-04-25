# 日志历史 API

## 内存日志条目

```
GET /api/logs/entries
```

返回内存中最近的日志条目。

### 查询参数

| 参数 | 类型 | 默认值 | 说明 |
|---|---|---|---|
| limit | int | 100 | 返回的最大条目数 |

### 响应

```json
{
  "success": true,
  "data": {
    "entries": [
      {
        "timestamp": "2026-01-15T10:00:00Z",
        "level": "INFO",
        "message": "模型已加载",
        "fields": {}
      }
    ],
    "count": 1
  },
  "metadata": {
    "timestamp": "2026-01-15T10:05:00Z",
    "requestId": "req-xxx"
  }
}
```

## 日志文件列表

```
GET /api/logs/files
```

返回历史日志文件列表。

### 响应

```json
{
  "success": true,
  "data": {
    "files": [
      {
        "name": "shepherd-hybrid-2026-01-15.log",
        "path": "/path/to/logs/shepherd-hybrid-2026-01-15.log",
        "size": 1048576,
        "role": "hybrid",
        "date": "2026-01-15",
        "createdAt": "2026-01-15T23:59:59Z",
        "isBackup": false
      }
    ],
    "count": 1
  },
  "metadata": {
    "timestamp": "2026-01-15T10:05:00Z",
    "requestId": "req-xxx"
  }
}
```

### LogFileInfo 字段

| 字段 | 类型 | 说明 |
|---|---|---|
| name | string | 文件名 |
| path | string | 文件完整路径 |
| size | int64 | 文件大小（字节） |
| role | string | 节点角色（master/client/hybrid） |
| date | string | 日志日期 |
| createdAt | string | 文件创建时间（RFC3339） |
| isBackup | bool | 是否为轮转备份文件 |

## 读取日志文件内容

```
GET /api/logs/files/:filename
```

### 查询参数

| 参数 | 类型 | 默认值 | 说明 |
|---|---|---|---|
| offset | int | 0 | 跳过的条目数 |
| limit | int | 100 | 返回的最大条目数 |
| level | string | - | 日志级别过滤（DEBUG/INFO/WARN/ERROR） |
| search | string | - | 搜索关键词（匹配 message 和 caller） |

### 响应

```json
{
  "success": true,
  "data": {
    "entries": [
      {
        "timestamp": "2026-01-15T10:00:00Z",
        "level": "INFO",
        "message": "模型已加载",
        "caller": "service/model.go:42",
        "fields": { "modelId": "abc123" },
        "raw": "[2026-01-15 10:00:00] [service/model.go:42] INFO 模型已加载 modelId=abc123"
      }
    ],
    "count": 1
  },
  "metadata": {
    "timestamp": "2026-01-15T10:05:00Z",
    "requestId": "req-xxx"
  }
}
```

### ParsedLogEntry 字段

| 字段 | 类型 | 说明 |
|---|---|---|
| timestamp | string | 时间戳（RFC3339） |
| level | string | 日志级别 |
| message | string | 日志消息 |
| caller | string | 调用位置（可省略） |
| fields | object | 附加字段（可省略） |
| raw | string | 原始日志行 |

### 安全限制

- 文件名正则校验（禁止 `..`、`/`、`\` 路径遍历）
- 文件扩展名必须为 `.log`
- 文件名格式：`shepherd-{role}-{date}.log` 或 `shepherd-{role}-{date} {time}.log`

## 日志文件统计

```
GET /api/logs/files/:filename/stats
```

返回日志文件中各级别的条目计数。

### 响应

```json
{
  "success": true,
  "data": {
    "DEBUG": 1000,
    "INFO": 3000,
    "WARN": 800,
    "ERROR": 200,
    "total": 5000
  },
  "metadata": {
    "timestamp": "2026-01-15T10:05:00Z",
    "requestId": "req-xxx"
  }
}
```

`data` 为 `map[string]int` 类型，键为日志级别（大写），`total` 为总计条目数。

## 删除日志文件

```
DELETE /api/logs/files/:filename
```

### 安全措施

- 文件名正则校验（禁止路径遍历）
- 当天活跃日志文件不可删除（返回 403）

### 响应

```json
{
  "success": true,
  "data": {
    "message": "日志文件已删除"
  },
  "metadata": {
    "timestamp": "2026-01-15T10:05:00Z",
    "requestId": "req-xxx"
  }
}
```

## 实时日志流

```
GET /api/logs/stream
```

SSE（Server-Sent Events）端点，推送实时日志条目。

### 查询参数

| 参数 | 类型 | 默认值 | 说明 |
|---|---|---|---|
| fromBeginning | bool | false | 是否先发送历史日志 |
| limit | int | 100 | 历史日志最大条目数（仅 fromBeginning=true 时有效） |

### SSE 数据格式

```
data: {"timestamp":"2026-01-15T10:00:00Z","level":"INFO","message":"模型已加载"}
```

每条 SSE 数据包含三个字段：

| 字段 | 类型 | 说明 |
|---|---|---|
| timestamp | string | 时间戳（RFC3339） |
| level | string | 日志级别 |
| message | string | 日志消息 |

### 心跳

每 30 秒发送一次 keepalive 心跳：

```
event: keepalive
data: 
```

## 日志格式

后端支持解析以下两种日志格式：

- **Text**（默认）：`[2026-01-15 10:00:00] [file.go:72] INFO 消息内容`
- **JSON**：`{"time":"...","level":"...","msg":"...","caller":"..."}`
