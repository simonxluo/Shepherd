# 日志历史 API

## 日志文件列表

```
GET /api/logs/files
```

返回历史日志文件列表。

```json
{
  "success": true,
  "data": [
    {
      "name": "shepherd-2026-01-15.log",
      "size": 1048576,
      "modifiedAt": "2026-01-15T23:59:59Z"
    }
  ]
}
```

## 读取日志文件

```
GET /api/logs/files/:filename
```

### 查询参数

| 参数 | 类型 | 默认值 | 说明 |
|---|---|---|---|
| page | int | 1 | 页码 |
| pageSize | int | 500 | 每页条目数 |
| level | string | - | 日志级别过滤（DEBUG/INFO/WARN/ERROR） |
| keyword | string | - | 关键词搜索 |

### 响应

```json
{
  "success": true,
  "data": {
    "content": "日志内容...",
    "pagination": {
      "page": 1,
      "pageSize": 500,
      "total": 1500,
      "totalPages": 3
    }
  }
}
```

## 日志文件统计

```
GET /api/logs/files/:filename/stats
```

```json
{
  "success": true,
  "data": {
    "filename": "shepherd-2026-01-15.log",
    "size": 1048576,
    "lineCount": 5000,
    "levels": {
      "DEBUG": 1000,
      "INFO": 3000,
      "WARN": 800,
      "ERROR": 200
    },
    "timeRange": {
      "start": "2026-01-15T00:00:00Z",
      "end": "2026-01-15T23:59:59Z"
    }
  }
}
```

## 删除日志文件

```
DELETE /api/logs/files/:filename
```

安全措施：
- 文件名正则校验（禁止路径遍历）
- 活跃日志文件不可删除

```json
{
  "success": true,
  "data": {
    "deleted": "shepherd-2026-01-10.log"
  }
}
```

## 实时日志流

```
GET /api/logs/stream
```

SSE 端点，推送实时日志条目：

```
data: {"level":"INFO","message":"模型已加载","timestamp":"2026-01-15T10:00:00Z","source":"model"}
```

## 日志格式

- **Text**（默认）：`[LEVEL] timestamp message`
- **JSON**：`{"level":"INFO","message":"...","timestamp":"..."}`
