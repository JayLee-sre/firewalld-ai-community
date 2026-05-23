# 项目结构手册

```text
firewalld-ai/
├── cmd/
│   └── zhiyu-waf/
│       └── main.go
├── configs/
│   ├── zhiyu-waf.yaml
│   └── rules/
├── docs/
├── internal/
│   ├── ai/
│   ├── config/
│   ├── dashboard/
│   ├── engine/
│   ├── geo/
│   ├── model/
│   ├── proxy/
│   ├── sshmon/
│   └── store/
├── web/
│   ├── public/
│   ├── src/
│   └── dist/
├── Makefile
├── go.mod
└── README.md
```

## cmd

程序入口。

- `cmd/zhiyu-waf/main.go`：加载配置、初始化数据库、规则引擎、AI、SSH 监控、Dashboard、代理监听和 iptables 接管。

## configs

配置与默认规则。

- `configs/zhiyu-waf.yaml`：主配置。
- `configs/rules/*.yaml`：默认检测规则。

## internal/ai

AI 检测模块。

- Prompt 构造。
- AI Provider 接口。
- OpenAI 兼容接口客户端。
- Claude 客户端。
- AI 缓存、超时、降级策略。

## internal/config

配置加载与热更新。

- YAML 解析。
- 默认配置。
- 文件变更监听。

## internal/dashboard

管理后台后端。

- 登录认证。
- JWT 中间件。
- 攻击统计接口。
- 攻击日志接口。
- 规则管理接口。
- IP 黑白名单接口。
- SSH 监控接口。
- 前端静态资源托管。

## internal/engine

检测引擎。

- 规则加载。
- 正则匹配。
- 黑白名单。
- 速率限制。
- AI 分析接入。
- 攻击日志生成。

## internal/geo

IP 地理位置解析。

- 查询外部 IP 地理信息。
- 格式化地区显示。

## internal/model

公共数据模型。

- 攻击日志。
- 规则。
- IP 黑白名单。

## internal/proxy

透明代理和 iptables 接管。

- TCP Listener。
- HTTP 请求解析。
- WAF 检测前置处理。
- 安全验证页。
- 后端请求转发。
- iptables NAT 链管理。

## internal/sshmon

SSH 登录监控。

- 解析 `/var/log/secure` 或 `/var/log/auth.log`。
- 记录失败、成功、封禁事件。
- 白名单成功登录跳过日志。
- 暴力破解自动加入黑名单并执行 iptables 封禁。

## internal/store

SQLite 存储层。

- 数据库迁移。
- 攻击日志。
- SSH 事件。
- 规则。
- 设置。
- IP 黑白名单。

## web

Dashboard 前端。

- `web/src/App.vue`：主布局。
- `web/src/views/`：页面。
- `web/public/logo.png`：品牌 logo。
- `web/dist/`：生产构建产物，由后端直接托管。
