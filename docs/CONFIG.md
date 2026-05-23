# 智域 WAF 配置参考

配置文件路径：`configs/zhiyu-waf.yaml`

所有字段均有默认值，未设置的字段会使用默认配置。

## 完整配置示例

```yaml
proxy:
  listen_addr: ":8080"
  backend_addr: "127.0.0.1:80"
  tls_cert_file: ""
  tls_key_file: ""
  acme_enabled: false
  acme_email: ""
  acme_domains: []
  dynamic_protect: false
  iptables_enable: true
  iptables_port: 80
  read_timeout: 30
  write_timeout: 30

dashboard:
  listen_addr: ":9090"
  jwt_secret: "change-me-to-random-string"
  cors_origins:
    - "http://localhost:9090"

ai:
  enabled: true
  provider: "openai"
  async_timeout: 5
  cache_ttl: 300
  max_requests_per_min: 60
  fail_open: true
  per_ip_rate: 10
  per_ip_burst: 2
  circuit_threshold: 5
  circuit_reset: 30
  high_risk_paths:
    - "/admin"
    - "/login"
    - "/upload"
  providers:
    openai:
      api_key: "sk-..."
      model: "gpt-4o"
      base_url: "https://api.openai.com/v1"
    claude:
      api_key: "sk-ant-..."
      model: "claude-sonnet-4-20250514"
      base_url: "https://api.anthropic.com"

engine:
  rules_dir: "./configs/rules"
  preset: "balanced"
  observation_mode: false
  rate_limit:
    requests_per_minute: 60
    burst_size: 10

storage:
  type: "sqlite"
  path: "./data/zhiyu-waf.db"
  # MySQL:
  # type: "mysql"
  # dsn: "user:pass@tcp(localhost:3306)/zhiyu_waf?charset=utf8mb4"
  # max_open_conns: 25
  # max_idle_conns: 10
  log_retention_days: 30

ssh:
  enabled: false
  log_path: ""
  max_fails: 5
  ban_minutes: 30

alert:
  enabled: false
  throttle_minutes: 10
  webhook_url: ""
  email:
    host: ""
    port: 587
    username: ""
    password: ""
    from: ""
    to: []
```

## proxy — 反向代理

| 字段 | 类型 | 默认值 | 说明 |
|------|------|--------|------|
| `listen_addr` | string | `:8080` | WAF 监听地址 |
| `backend_addr` | string | `127.0.0.1:80` | 后端服务器地址 |
| `tls_cert_file` | string | | TLS 证书文件路径 |
| `tls_key_file` | string | | TLS 私钥文件路径 |
| `acme_enabled` | bool | `false` | 启用 ACME 自动证书 |
| `acme_email` | string | | ACME 邮箱 |
| `acme_domains` | []string | | ACME 域名列表 |
| `dynamic_protect` | bool | `false` | 启用动态 HTML 变换（防爬虫） |
| `iptables_enable` | bool | `true` | 启用 iptables 封禁 |
| `iptables_port` | int | `80` | iptables 保护的端口 |
| `read_timeout` | int | `30` | 读取超时（秒） |
| `write_timeout` | int | `30` | 写入超时（秒） |

## dashboard — 管理面板

| 字段 | 类型 | 默认值 | 说明 |
|------|------|--------|------|
| `listen_addr` | string | `:9090` | 管理面板监听地址 |
| `jwt_secret` | string | 自动生成 | JWT 签名密钥，生产环境务必修改 |
| `cors_origins` | []string | `["http://localhost:9090"]` | CORS 允许的源 |
| `tls_cert_file` | string | | 面板 TLS 证书 |
| `tls_key_file` | string | | 面板 TLS 私钥 |

## ai — AI 分析引擎

| 字段 | 类型 | 默认值 | 说明 |
|------|------|--------|------|
| `enabled` | bool | `true` | 启用 AI 分析 |
| `provider` | string | `openai` | AI 提供商（`openai` / `claude`） |
| `async_timeout` | int | `5` | AI 分析超时（秒） |
| `cache_ttl` | int | `300` | 结果缓存时间（秒） |
| `max_requests_per_min` | int | `60` | 每分钟最大 AI 请求数 |
| `fail_open` | bool | `true` | AI 故障时放行（否则拒绝） |
| `per_ip_rate` | int | `10` | 单 IP AI 调用频率限制 |
| `per_ip_burst` | int | `2` | 单 IP 突发限制 |
| `circuit_threshold` | int | `5` | 连续失败触发熔断阈值 |
| `circuit_reset` | int | `30` | 熔断恢复探测间隔（秒） |
| `high_risk_paths` | []string | 见默认值 | 高风险路径列表，优先 AI 分析 |

## engine — 规则引擎

| 字段 | 类型 | 默认值 | 说明 |
|------|------|--------|------|
| `rules_dir` | string | `./configs/rules` | 规则文件目录 |
| `preset` | string | `balanced` | 规则预设（`strict` / `balanced` / `permissive`） |
| `observation_mode` | bool | `false` | 观察模式（仅记录不拦截） |
| `rate_limit.requests_per_minute` | int | `60` | 每 IP 每分钟请求限制 |
| `rate_limit.burst_size` | int | `10` | 突发请求限制 |

## storage — 存储

| 字段 | 类型 | 默认值 | 说明 |
|------|------|--------|------|
| `type` | string | `sqlite` | 存储类型（`sqlite` / `mysql`） |
| `path` | string | `./data/zhiyu-waf.db` | SQLite 文件路径 |
| `dsn` | string | | MySQL 连接串 |
| `max_open_conns` | int | `25` | MySQL 最大打开连接数 |
| `max_idle_conns` | int | `10` | MySQL 最大空闲连接数 |
| `log_retention_days` | int | `30` | 日志保留天数 |

### MySQL DSN 格式

```
用户名:密码@tcp(主机:端口)/数据库名?charset=utf8mb4&parseTime=True
```

## ssh — SSH 监控

| 字段 | 类型 | 默认值 | 说明 |
|------|------|--------|------|
| `enabled` | bool | `false` | 启用 SSH 暴力破解监控 |
| `log_path` | string | 系统默认 | SSH 日志路径（留空自动检测） |
| `max_fails` | int | `5` | 失败次数触发封禁 |
| `ban_minutes` | int | `30` | 封禁时长（分钟） |

## alert — 告警通知

| 字段 | 类型 | 默认值 | 说明 |
|------|------|--------|------|
| `enabled` | bool | `false` | 启用告警 |
| `throttle_minutes` | int | `10` | 同类告警去重间隔（分钟） |
| `webhook_url` | string | | Webhook URL |
| `email.*` | | | 邮件配置，见下表 |

### email 子配置

| 字段 | 类型 | 说明 |
|------|------|------|
| `host` | string | SMTP 服务器地址 |
| `port` | int | SMTP 端口（通常 587 或 465） |
| `username` | string | SMTP 用户名 |
| `password` | string | SMTP 密码 |
| `from` | string | 发件人地址 |
| `to` | []string | 收件人列表 |

### Webhook 告警格式

```json
{
  "id": "alert-uuid",
  "title": "高频攻击告警",
  "severity": "high",
  "message": "检测到 SQL 注入攻击",
  "source_ip": "1.2.3.4",
  "rule_id": "SQLI-001",
  "timestamp": "2026-01-01T12:00:00Z"
}
```
