# 安全策略

## 🛡️ 安全政策

Shepherd 项目致力于维护一个安全的开发和使用环境。感谢您帮助我们保护项目和用户。

## 📋 报告安全漏洞

如果您发现安全漏洞，**请不要公开 Issue**。请发送邮件至：

**安全邮箱**: [待定]

请尽可能包含以下信息：

- 漏洞描述
- 影响版本
- 复现步骤
- 潜在影响
- 建议的修复方案

### 报告流程

1. 发送安全漏洞报告到上述邮箱
2. 维护者会在 48 小时内确认收到
3. 我们会评估漏洞的严重程度
4. 制定修复计划并通知您
5. 修复后发布新版本
6. 在公告中致谢（如果您愿意）

### 漏洞评级标准

我们使用 [CVSS v3.1](https://www.first.org/cvss/calculator/3.1) 评级漏洞：

| 等级 | 分数范围 | 响应时间 |
|------|---------|---------|
| 严重 (Critical) | 9.0-10.0 | 48 小时 |
| 高危 (High) | 7.0-8.9 | 7 天 |
| 中危 (Medium) | 4.0-6.9 | 14 天 |
| 低危 (Low) | 0.1-3.9 | 30 天 |

## 🔐 安全最佳实践

### 部署建议

1. **不要暴露到公网** - 默认配置未做安全加固
2. **使用反向代理** - 在生产环境使用 Nginx/Caddy
3. **启用 API Key** - 在生产环境启用 API 密钥认证
4. **配置 CORS** - 限制允许的跨域来源
5. **使用 HTTPS** - 生产环境必须使用 SSL/TLS

### 配置示例

#### Nginx 反向代理

```nginx
server {
    listen 443 ssl http2;
    server_name your-domain.com;

    ssl_certificate /path/to/cert.pem;
    ssl_certificate_key /path/to/key.pem;

    location / {
        proxy_pass http://localhost:9190;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }
}
```

#### API Key 认证

```yaml
security:
  api_key_enabled: true
  api_key: "your-secure-api-key-here"
```

## 🔍 已知安全问题

### 当前版本

- [ ] 未实现请求速率限制
- [ ] 未实现请求大小限制
- [ ] 未实现完整的日志审计
- [ ] 模型加载未做权限验证

### 未来改进

- [ ] JWT Token 认证
- [ ] OAuth2 集成
- [ ] 请求速率限制
- [ ] IP 白名单/黑名单
- [ ] 完整的审计日志

## 📊 安全审计

### 第三方审计

目前未进行第三方安全审计。计划在 v1.0.0 版本前进行安全审计。

### 自检清单

- [ ] 输入验证
- [ ] 输出编码
- [ ] 认证和授权
- [ ] 会话管理
- [ ] 加密存储
- [ ] 错误处理
- [ ] 日志记录

## 🔄 依赖安全

### 依赖更新

我们定期更新依赖以修复已知的安全漏洞：

```bash
go get -u ./...
go mod tidy
```

### 漏洞扫描

发布前我们会运行静态分析工具：

```bash
go install golang.org/x/vuln/cmd/govulncheck@latest
govulncheck ./...
```

## 🔐 密钥管理

### API Key

- 不要在代码中硬编码 API Key
- 使用环境变量或配置文件
- 将 API Key 添加到 `.gitignore`

### 敏感配置

```bash
# .gitignore
config/config.yaml
*.key
*.pem
```

## 📝 安全事件响应

### 响应流程

1. **确认** - 确认安全漏洞报告
2. **评估** - 评估漏洞严重程度和影响范围
3. **修复** - 开发修复补丁
4. **测试** - 验证修复效果
5. **发布** - 发布安全更新版本
6. **公告** - 发布安全公告

### 安全公告格式

```
标题: [CVE-YYYY-XXXXX] 漏洞描述

影响版本: vX.Y.Z - vA.B.C
修复版本: vX.Y.Z+1
严重程度: 严重/高危/中危/低危
CVE 编号: CVE-YYYY-XXXXX

描述:
[详细描述漏洞]

修复:
升级到 vX.Y.Z+1

致谢:
感谢 [报告者] 报告此漏洞
```

## 📞 联系方式

- 安全漏洞报告: [待定]
- 一般问题: [GitHub Issues](https://github.com/shepherd-project/shepherd/issues)

---

**最后更新**: 2026-02-19
