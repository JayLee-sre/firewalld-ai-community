# 接口手册

接口默认前缀：`/api/v1`

Dashboard 地址默认：`http://服务器IP:9090`

## 1. 认证

### 登录

```http
POST /api/v1/auth/login
Content-Type: application/json
```

请求：

```json
{
  "username": "admin",
  "password": "your-password"
}
```

响应：

```json
{
  "token": "JWT_TOKEN"
}
```

后续接口使用：

```http
Authorization: Bearer JWT_TOKEN
```

### 修改密码

```http
POST /api/v1/auth/password
Authorization: Bearer JWT_TOKEN
Content-Type: application/json
```

请求：

```json
{
  "old_password": "old",
  "new_password": "new"
}
```

## 2. 健康检查

公开接口：

```http
GET /health
```

认证详情接口：

```http
GET /api/v1/health
```

## 3. 统计接口

### 攻击统计

```http
GET /api/v1/stats
```

可选参数：

- `since`：RFC3339 时间。

返回包含：

- `total_requests`
- `blocked_count`
- `by_severity`
- `by_source`
- `top_attack_paths`
- `top_regions`

### 时间序列统计

```http
GET /api/v1/stats/timeseries?hours=24
```

## 4. 攻击日志

### 列表

```http
GET /api/v1/logs?page=1&page_size=20
```

过滤参数：

- `client_ip`
- `severity`
- `source`

### 详情

```http
GET /api/v1/logs/{id}
```

### 导出

```http
GET /api/v1/logs/export?format=csv&since=2026-01-01T00:00:00Z&severity=high
```

参数：

- `format`：`csv` 或 `json`
- `since`：起始时间（可选）
- `severity`：严重级别过滤（可选）
- `client_ip`：来源 IP 过滤（可选）

### 标记已审核

```http
POST /api/v1/logs/{id}/reviewed
```

### 标记误报

```http
POST /api/v1/logs/{id}/false-positive
```

### WebSocket 实时流

```text
ws://服务器IP:9090/api/v1/logs/stream?token=JWT_TOKEN
```

## 5. 规则管理

```http
GET    /api/v1/rules
POST   /api/v1/rules
PUT    /api/v1/rules/{id}
DELETE /api/v1/rules/{id}
```

规则字段：

- `id`
- `name`
- `description`
- `severity`：`low` / `medium` / `high` / `critical`
- `enabled`
- `patterns`：匹配模式数组
- `match_locations`：匹配位置数组（`url`, `headers`, `body`, `query`）

### 测试规则

```http
POST /api/v1/rules/test
Content-Type: application/json
```

```json
{
  "patterns": ["union.*select"],
  "match_locations": ["url", "body"],
  "test_input": "/page?id=1 UNION SELECT * FROM users"
}
```

### 预览规则匹配

```http
POST /api/v1/rules/preview
Content-Type: application/json
```

## 6. IP 黑白名单

### 列表

```http
GET /api/v1/iplist?type=blacklist
```

参数：

- `type`：`blacklist` 或 `whitelist`

### 单个添加

```http
POST /api/v1/iplist
Content-Type: application/json
```

```json
{
  "ip_address": "1.2.3.4",
  "list_type": "blacklist",
  "note": "攻击者"
}
```

支持 CIDR 格式：`"ip_address": "10.0.0.0/8"`

### 批量添加

```http
POST /api/v1/iplist/batch
Content-Type: application/json
```

```json
{
  "list_type": "blacklist",
  "entries": "192.168.1.100 攻击者\n10.0.0.0/8 内网段\n172.16.0.1"
}
```

`entries` 字段：每行一个 IP，IP 后可跟空格和备注。

### 导出

```http
GET /api/v1/iplist/export?type=blacklist
```

返回 CSV 格式文件。

### 删除

```http
DELETE /api/v1/iplist/{id}
```

## 7. 地理围栏

```http
GET    /api/v1/geo/rules
POST   /api/v1/geo/rules
PUT    /api/v1/geo/rules/{id}
DELETE /api/v1/geo/rules/{id}
```

请求示例：

```json
{
  "country": "CN",
  "action": "block",
  "enabled": true
}
```

## 8. 站点管理（专业版）

```http
GET    /api/v1/sites
POST   /api/v1/sites
PUT    /api/v1/sites/{id}
DELETE /api/v1/sites/{id}
```

## 9. AI 配置

### 获取 AI 提供商

```http
GET /api/v1/ai/providers
```

### 更新全局 AI 配置

```http
PUT /api/v1/ai/global
Content-Type: application/json
```

```json
{
  "enabled": true,
  "provider": "openai",
  "async_timeout": 5,
  "cache_ttl": 300,
  "fail_open": true
}
```

### 更新提供商配置

```http
PUT /api/v1/ai/providers/openai
Content-Type: application/json
```

```json
{
  "api_key": "sk-...",
  "model": "gpt-4o",
  "base_url": "https://api.openai.com/v1"
}
```

### 测试 AI 连接

```http
POST /api/v1/ai/test
```

### AI 统计

```http
GET /api/v1/ai/stats
GET /api/v1/ai/usage
```

### AI 建议（专业版）

```http
GET  /api/v1/ai/suggestions
POST /api/v1/ai/suggestions/promote
POST /api/v1/ai/generate-rule
GET  /api/v1/ai/threat-profile
```

## 10. SSH 监控

### 统计

```http
GET /api/v1/ssh/stats
```

### 事件列表

```http
GET /api/v1/ssh/events?page=1&page_size=20
```

过滤参数：

- `client_ip`
- `event_type`：`failed`、`blocked`、`success`
- `username`

说明：白名单 IP 的 SSH 成功登录不会记录；失败登录仍会记录。

## 11. 威胁情报

```http
GET   /api/v1/threatintel/status
POST  /api/v1/threatintel/sync
PUT   /api/v1/threatintel/config
```

## 12. 审计日志

```http
GET /api/v1/audit/events?page=1&page_size=20
```

过滤参数：

- `actor`：操作者
- `action`：操作类型
- `since`：起始时间
- `until`：截止时间

## 13. 设置

### 获取设置

```http
GET /api/v1/settings
```

### 更新设置

```http
PUT /api/v1/settings
Content-Type: application/json
```

```json
{
  "dynamic_protect": "true",
  "log_retention_days": "30"
}
```

### 重载配置

```http
POST /api/v1/config/reload
```

## 14. 备份与恢复

### 导出备份

```http
GET /api/v1/backup/export
```

返回 JSON 文件，包含规则、IP 列表、站点、地理围栏和设置（不含敏感信息）。

### 导入备份

```http
POST /api/v1/backup/import
Content-Type: application/json
```

请求体为导出的 JSON 文件内容。返回：

```json
{
  "imported": {
    "rules": 10,
    "ip_entries": 5,
    "sites": 2,
    "geo_rules": 3,
    "settings": 8
  },
  "errors": []
}
```

## 15. 用户管理（仅管理员）

### 列出用户

```http
GET /api/v1/users
```

### 创建用户

```http
POST /api/v1/users
Content-Type: application/json
```

```json
{
  "username": "operator1",
  "password": "securepassword123",
  "role": "operator"
}
```

角色：`admin` / `operator` / `viewer`

### 删除用户

```http
DELETE /api/v1/users/{id}
```

不能删除最后一个管理员账户。

### 修改用户密码

```http
PUT /api/v1/users/{id}/password
Content-Type: application/json
```

```json
{
  "new_password": "newpassword123"
}
```

## 16. 许可证

```http
POST /api/v1/license/activate
Content-Type: application/json
```

```json
{
  "license_key": "XXXX-XXXX-XXXX-XXXX"
}
```

## Prometheus 指标

```http
GET /metrics
```

返回 Prometheus 格式的监控指标。
