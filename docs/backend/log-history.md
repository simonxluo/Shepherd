# 日志历史查看功能

## 概述

本功能为 Shepherd 项目添加了日志历史查看能力，允许用户浏览存储在磁盘上的历史日志文件，而不仅限于内存中的实时日志流。

## 功能特性

### 后端 API (Go)

#### 新增端点

1. **`GET /api/logs/files`** - 列出所有可用的日志文件
   - 响应格式：
     ```json
     {
       "files": [
         {
           "name": "shepherd-standalone-2025-02-26.log",
           "path": "/path/to/logs/shepherd-standalone-2025-02-26.log",
           "size": 12345,
           "mode": "standalone",
           "date": "2025-02-26",
           "createdAt": "2025-02-26T20:17:51Z",
           "isBackup": false
         }
       ],
       "count": 1
     }
     ```

2. **`GET /api/logs/files/:filename`** - 读取指定日志文件的内容（支持分页和过滤）
   - 查询参数：
     - `level`: 过滤日志级别（DEBUG, INFO, WARN, ERROR）
     - `search`: 搜索日志内容
     - `offset`: 跳过 N 条日志
     - `limit`: 返回最多 N 条日志
   - 响应格式：
     ```json
     {
       "entries": [
         {
           "timestamp": "2025-02-26T20:17:51Z",
           "level": "INFO",
           "message": "模型加载完成",
           "caller": "server.go:72",
           "fields": {"modelCount": "1"},
           "raw": "[2025-02-26 20:17:51] [server.go:72] INFO 模型加载完成 modelCount=1"
         }
       ],
       "count": 1
     }
     ```

3. **`GET /api/logs/files/:filename/stats`** - 获取日志文件的统计信息
   - 响应格式：
     ```json
     {
       "total": 100,
       "INFO": 50,
       "DEBUG": 20,
       "WARN": 15,
       "ERROR": 10,
       "FATAL": 5
     }
     ```

4. **`DELETE /api/logs/files/:filename`** - 删除指定的日志文件
   - 安全限制：不能删除当天的日志文件
   - 响应格式：
     ```json
     {
       "message": "日志文件已删除"
     }
     ```

### 前端功能 (TypeScript/React)

#### 新增组件和页面

1. **视图模式切换**
   - 实时流模式：查看实时日志（原有功能）
   - 历史文件模式：查看历史日志文件（新功能）

2. **历史文件选择器**
   - 显示所有可用的日志文件
   - 显示文件大小、创建时间、运行模式
   - 区分当前日志和备份日志
   - 支持删除历史日志文件（当前日志除外）

3. **分页控件**
   - 支持翻页浏览大型日志文件
   - 每页显示 500 条日志
   - 显示当前页码和总页数

4. **日志过滤增强**
   - 支持按日志级别过滤（DEBUG, INFO, WARN, ERROR）
   - 支持关键词搜索
   - 在实时流和历史模式下都可用

## 文件结构

### 后端新增文件

- `internal/logger/files.go` - 日志文件管理功能
  - `ListLogFiles()` - 列出日志文件
  - `ReadLogFile()` - 读取日志文件内容
  - `GetLogFileStats()` - 获取日志统计
  - `parseLogLineToEntry()` - 解析日志行

- `internal/logger/files_test.go` - 单元测试

### 后端修改文件

- `internal/server/server.go`
  - 新增 4 个处理函数
  - `isSafeFilename()` - 文件名安全验证

### 前端新增文件

- `web/src/lib/api/logs.ts` - 日志 API 客户端
  - `listLogFiles()` - 获取日志文件列表
  - `getFileContent()` - 获取日志文件内容
  - `getFileStats()` - 获取日志统计
  - `deleteLogFile()` - 删除日志文件
  - `formatFileSize()` - 格式化文件大小
  - `formatDate()` - 格式化日期

### 前端修改文件

- `web/src/types/logs.ts` - 新增类型定义
  - `LogFileInfo` - 日志文件信息
  - `LogFileContent` - 日志文件内容
  - `ParsedLogEntry` - 解析后的日志条目
  - `LogFileFilter` - 日志过滤器

- `web/src/pages/logs/index.tsx` - 日志页面
  - 视图模式切换（实时/历史）
  - 历史文件选择器
  - 分页控件
  - 删除文件功能

## 安全特性

1. **文件名验证**
   - 使用严格的正则表达式验证文件名格式
   - 防止路径遍历攻击（`..`, `/`, `\`）
   - 只允许 `.log` 扩展名

2. **删除保护**
   - 禁止删除当天的活跃日志文件
   - 防止误删除正在使用的日志

3. **错误处理**
   - 统一的错误响应格式
   - 详细的错误信息（开发环境）
   - 友好的错误提示（用户界面）

## 日志格式支持

### 文本格式（默认）

```
[2025-02-26 20:17:51] [server.go:72] INFO 模型加载完成 modelCount=1
```

### JSON 格式

```json
{"time":"2025-02-26T20:17:51Z","level":"INFO","msg":"模型加载完成","caller":"server.go:72","modelCount":"1"}
```

## 使用示例

### 获取日志文件列表

```bash
curl http://localhost:9190/api/logs/files
```

### 获取特定日志文件内容（带过滤）

```bash
curl "http://localhost:9190/api/logs/files/shepherd-standalone-2025-02-26.log?level=ERROR&limit=10"
```

### 获取日志统计

```bash
curl http://localhost:9190/api/logs/files/shepherd-standalone-2025-02-26.log/stats
```

### 删除日志文件

```bash
curl -X DELETE http://localhost:9190/api/logs/files/shepherd-standalone-2025-02-25.log
```

## 测试

运行后端测试：

```bash
go test ./internal/logger/... -v
```

运行前端类型检查：

```bash
cd web && npm run type-check
```

## 注意事项

1. **性能考虑**
   - 大型日志文件使用分页加载
   - 每页默认 500 条记录
   - 客户端缓存已加载的页面

2. **并发安全**
   - 文件读取使用互斥锁保护
   - 删除操作检查文件是否正在使用

3. **向后兼容**
   - 保持原有的实时日志流功能
   - 不破坏现有的 API 端点
   - 前端支持运行时切换视图模式
