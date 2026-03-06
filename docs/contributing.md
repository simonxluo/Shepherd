# 贡献指南

感谢你对 Shepherd 项目的关注！我们欢迎各种形式的贡献。

## 🤝 如何贡献

### 报告 Bug

请在 [GitHub Issues](https://github.com/shepherd-project/shepherd/issues) 报告 bug，并包含：

- 清晰的标题和描述
- 复现步骤
- 预期行为 vs 实际行为
- 环境信息（操作系统、Go 版本等）
- 相关日志或截图

### 提交功能建议

1. 先在 [GitHub Discussions](https://github.com/shepherd-project/shepherd/discussions) 讨论大方向的功能建议
2. 提交 Issue 说明新功能的用途和实现思路
3. 等待维护者反馈后再开始开发

### 提交代码

#### 开发流程

1. **Fork 仓库**
   ```bash
   # 在 GitHub 上 Fork 本仓库
   git clone https://github.com/YOUR_USERNAME/shepherd.git
   cd shepherd
   git remote add upstream https://github.com/shepherd-project/shepherd.git
   ```

2. **创建分支**
   ```bash
   git checkout -b feature/your-feature-name
   # 或
   git checkout -b fix/your-bug-fix
   ```

3. **编写代码**
   - 遵循现有代码风格
   - 添加必要的测试
   - 更新相关文档

4. **运行测试**
   ```bash
   make test
   ```

5. **提交代码**
   ```bash
   git add .
   git commit -m "feat: add your feature"
   ```

   提交信息格式：
   - `feat:` 新功能
   - `fix:` Bug 修复
   - `docs:` 文档更新
   - `style:` 代码格式调整
   - `refactor:` 重构
   - `test:` 测试相关
   - `chore:` 构建/工具链相关

6. **推送分支**
   ```bash
   git push origin feature/your-feature-name
   ```

7. **创建 Pull Request**
   - 在 GitHub 上创建 PR
   - 填写 PR 模板
   - 等待 Code Review

#### 代码规范

- 遵循 [Go Code Review Comments](https://github.com/golang/go/wiki/CodeReviewComments)
- 使用 `go fmt` 格式化代码
   ```bash
   make fmt
   ```
- 运行静态检查（如果有 golangci-lint）
   ```bash
   make lint
   ```
- 添加单元测试，保持测试覆盖率

#### 测试要求

- 所有新功能必须包含测试
- 确保所有测试通过
   ```bash
   make test
   ```
- 对于复杂功能，添加集成测试

### 文档贡献

- 修正错别字和语法错误
- 改进现有文档的清晰度
- 添加使用示例
- 翻译文档

### 审查 PR

我们欢迎社区成员参与 PR 审查：

1. 查看 [Open Pull Requests](https://github.com/shepherd-project/shepherd/pulls)
2. 仔细审查代码变更
3. 在 PR 中留下评论和建议
4. 测试代码变更

## 📋 开发环境

### 环境要求

- Go 1.25 或更高版本
- Git
- Make (可选，用于快速命令)

### 初始化项目

```bash
# 克隆仓库
git clone https://github.com/shepherd-project/shepherd.git
cd shepherd

# 安装依赖
go mod download

# 验证环境
go version
make test
```

### 开发工作流

```bash
# 创建功能分支
git checkout -b feature/my-feature

# 进行开发
# ... 编写代码 ...

# 运行测试
make test

# 提交代码
git add .
git commit -m "feat: add my feature"

# 推送到远程
git push origin feature/my-feature
```

## 🎯 优先级标签

Issue 和 PR 会标记以下优先级：

- `critical` - 关键 bug 或安全漏洞，优先处理
- `high` - 重要功能或 bug
- `medium` - 一般功能或改进
- `low` - 错误提示、文档等

## 📜 行为准则

- 尊重所有贡献者
- 使用友好和包容的语言
- 接受建设性批评
- 关注对社区最有利的事情

## 🎨 风格指南

### Go 代码

- 遵循 [Effective Go](https://go.dev/doc/effective_go)
- 使用有意义的变量和函数名
- 添加必要的注释
- 保持函数简短和专注
- 避免重复代码

### Git 提交

- 提交信息清晰描述更改内容
- 一个提交只做一件事
- 提交前运行测试确保不破坏现有功能

### 文档

- 使用清晰简洁的语言
- 提供代码示例
- 更新相关文档

## 🧪 测试指南

### 单元测试

```go
func TestMyFunction(t *testing.T) {
    // Arrange
    input := "test"

    // Act
    result := MyFunction(input)

    // Assert
    if result != "expected" {
        t.Errorf("expected 'expected', got '%s'", result)
    }
}
```

### 运行测试

```bash
# 运行所有测试
make test

# 运行特定包的测试
go test ./internal/config/...

# 查看测试覆盖率
make test-coverage
```

## 🚀 发布流程

1. 更新版本号
2. 更新 CHANGELOG
3. 创建 Git tag
4. 构建发布包
5. 创建 GitHub Release

## 📞 获取帮助

- 查看 [文档](docs/)
- 在 [Discussions](https://github.com/shepherd-project/shepherd/discussions) 提问
- 在 [Issues](https://github.com/shepherd-project/shepherd/issues) 报告问题

## ⭐ 成为维护者

活跃的贡献者可能会被邀请成为项目维护者，获得：

- 写入权限
- 参与路线图规划
- 参与重大决策

---

**感谢你的贡献！** 🎉
