# 部署手册

## 1. 系统要求

- Linux x86_64 服务器，建议 2C/2G 以上。
- Root 权限，用于监听端口、管理 systemd、接管 iptables。
- 已安装 `iptables`、`sqlite3`、`gcc`。
- 如需在服务器编译，安装 Go 1.25+。

## 2. 目录规划

生产环境推荐目录：

```text
/opt/zhiyu-waf
├── bin/zhiyu-waf
├── configs/zhiyu-waf.yaml
├── configs/rules/
├── data/zhiyu-waf.db
├── logs/
└── web/dist/
```

## 3. 构建

本地构建前端：

```bash
cd web
npm install
npm run build
```

构建后端：

```bash
go test ./...
CGO_ENABLED=1 go build -o bin/zhiyu-waf ./cmd/zhiyu-waf
```

## 4. 配置

核心配置文件：`configs/zhiyu-waf.yaml`

```yaml
proxy:
  listen_addr: ":8080"
  backend_addr: "127.0.0.1:80"
  iptables_enable: true
  iptables_port: 80

dashboard:
  listen_addr: ":9090"

engine:
  rules_dir: "/opt/zhiyu-waf/configs/rules"

storage:
  path: "/opt/zhiyu-waf/data/zhiyu-waf.db"
```

说明：

- `listen_addr`：WAF 代理监听端口。
- `backend_addr`：真实业务站点地址。
- `iptables_enable`：是否自动接管公网入口流量。
- `iptables_port`：需要被接管的业务端口，通常是 `80`。

## 5. systemd 服务

创建 `/etc/systemd/system/zhiyu-waf.service`：

```ini
[Unit]
Description=ZhiYu-WAF 专业版
After=network.target

[Service]
Type=simple
WorkingDirectory=/opt/zhiyu-waf
ExecStart=/opt/zhiyu-waf/bin/zhiyu-waf -config /opt/zhiyu-waf/configs/zhiyu-waf.yaml
Restart=on-failure
RestartSec=5
Environment=PATH=/usr/local/go/bin:/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin

[Install]
WantedBy=multi-user.target
```

启动：

```bash
systemctl daemon-reload
systemctl enable zhiyu-waf
systemctl restart zhiyu-waf
systemctl status zhiyu-waf --no-pager -l
```

## 6. 验证

```bash
curl http://127.0.0.1:9090/health
curl http://127.0.0.1:9090/
curl http://127.0.0.1:8080/
iptables -t nat -S | grep ZHIYU_WAF
```

公网验证：

```bash
curl -I http://服务器IP/
curl -I http://服务器IP:9090/
```

## 7. 回滚

部署前建议备份：

```bash
tar -czf /opt/backups/zhiyu-waf-$(date +%Y%m%d-%H%M%S).tar.gz /opt/zhiyu-waf
```

如需关闭流量接管：

```bash
sed -i 's/iptables_enable: true/iptables_enable: false/' /opt/zhiyu-waf/configs/zhiyu-waf.yaml
systemctl restart zhiyu-waf
```
