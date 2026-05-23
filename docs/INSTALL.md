# 智域 WAF 安装指南

## 系统要求

- Linux (推荐 Ubuntu 20.04+ / CentOS 7+) 或 macOS
- Go 1.21+ (源码编译)
- Node.js 18+ (前端编译)
- SQLite3 (默认) 或 MySQL 5.7+ / 8.0+

## 方式一：二进制安装

```bash
# 下载最新版本
wget https://releases.zhiyuwaf.com/zhiyu-waf-linux-amd64.tar.gz
tar xzf zhiyu-waf-linux-amd64.tar.gz
cd zhiyu-waf

# 创建数据目录
mkdir -p data

# 首次运行会启动初始化向导
sudo ./bin/zhiyu-waf -config configs/zhiyu-waf.yaml
```

## 方式二：源码编译

```bash
git clone https://github.com/zhiyuwaf/zhiyu-waf.git
cd zhiyu-waf

# 安装依赖
make deps

# 编译（包含前端）
make build

# 运行
sudo ./bin/zhiyu-waf -config configs/zhiyu-waf.yaml
```

仅编译后端（跳过前端，用于开发）：

```bash
make backend
```

前端开发模式（热更新）：

```bash
make frontend-dev
```

## 方式三：Docker

```bash
# 使用 docker-compose
docker-compose up -d

# 或直接运行
docker run -d \
  -p 8080:8080 \
  -p 9090:9090 \
  -v zhiyu-waf-data:/app/data \
  -v ./configs:/app/configs \
  zhiyuwaf/zhiyu-waf:latest
```

### docker-compose.yml 示例

```yaml
version: '3'
services:
  waf:
    image: zhiyuwaf/zhiyu-waf:latest
    ports:
      - "8080:8080"   # WAF 代理端口
      - "9090:9090"   # 管理面板
    volumes:
      - waf-data:/app/data
      - ./configs:/app/configs
    restart: unless-stopped

volumes:
  waf-data:
```

## 方式四：systemd 服务

```bash
# 复制二进制
sudo cp bin/zhiyu-waf /usr/local/bin/
sudo cp -r configs /etc/zhiyu-waf/

# 创建服务文件
sudo tee /etc/systemd/system/zhiyu-waf.service << 'EOF'
[Unit]
Description=ZhiYu WAF
After=network.target

[Service]
Type=simple
ExecStart=/usr/local/bin/zhiyu-waf -config /etc/zhiyu-waf/zhiyu-waf.yaml
Restart=always
RestartSec=5
LimitNOFILE=65535

[Install]
WantedBy=multi-user.target
EOF

sudo systemctl daemon-reload
sudo systemctl enable zhiyu-waf
sudo systemctl start zhiyu-waf
```

## 初始配置

### 1. 首次访问

启动后访问 `http://服务器IP:9090`，系统会自动进入初始化向导，引导设置管理员密码。

### 2. 修改配置

编辑 `configs/zhiyu-waf.yaml`：

```yaml
proxy:
  listen_addr: ":8080"          # WAF 监听地址
  backend_addr: "127.0.0.1:80"  # 后端服务器地址

dashboard:
  listen_addr: ":9090"          # 管理面板地址
  jwt_secret: "your-random-secret"  # 务必修改
```

### 3. 配置域名（HTTPS）

方式一 — 手动证书：

```yaml
proxy:
  tls_cert_file: "/path/to/cert.pem"
  tls_key_file: "/path/to/key.pem"
```

方式二 — ACME 自动证书：

```yaml
proxy:
  acme_enabled: true
  acme_email: "admin@example.com"
  acme_domains:
    - "waf.example.com"
```

## MySQL 配置（可选）

大流量生产环境推荐使用 MySQL：

```yaml
storage:
  type: "mysql"
  dsn: "user:password@tcp(127.0.0.1:3306)/zhiyu_waf?charset=utf8mb4&parseTime=True"
  max_open_conns: 25
  max_idle_conns: 10
  log_retention_days: 30
```

首次启动会自动创建所需的数据库表。

## 验证安装

```bash
# 检查服务状态
curl http://localhost:9090/health
# 应返回: {"status":"ok"}

# 检查代理
curl http://localhost:8080/
# 应返回后端服务器的响应
```

## 常见问题

### 端口被占用

```bash
# 查看端口占用
sudo lsof -i :8080
sudo lsof -i :9090
```

### 权限问题

WAF 需要绑定低端口（如 80/443），需要 root 权限或设置：

```bash
sudo setcap cap_net_bind_service=+ep /usr/local/bin/zhiyu-waf
```

### iptables 配置

如果启用 `iptables_enable`，需要确保系统支持 iptables：

```bash
# Ubuntu/Debian
sudo apt install iptables

# CentOS/RHEL
sudo yum install iptables
```
