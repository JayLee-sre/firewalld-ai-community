# 代理手册

## 1. 代理链路

启用接管后，流量链路如下：

```text
公网用户 -> 服务器:80 -> iptables REDIRECT -> WAF:8080 -> 真实站点 127.0.0.1:80
```

## 2. 关键配置

```yaml
proxy:
  listen_addr: ":8080"
  backend_addr: "127.0.0.1:80"
  iptables_enable: true
  iptables_port: 80
```

字段说明：

- `listen_addr`：WAF 代理监听地址。
- `backend_addr`：真实站点地址。
- `iptables_enable`：是否接管入口端口。
- `iptables_port`：入口端口，通常是 `80`。

## 3. iptables 规则

系统启用接管后会创建 NAT 链：

```text
ZHIYU_WAF_REDIRECT
```

典型规则：

```bash
iptables -t nat -A PREROUTING -p tcp --dport 80 -j ZHIYU_WAF_REDIRECT
iptables -t nat -A ZHIYU_WAF_REDIRECT -p tcp -j REDIRECT --to-port 8080
```

查看：

```bash
iptables -t nat -S | grep ZHIYU_WAF
```

## 4. 代理行为

WAF 代理收到请求后会依次执行：

1. 解析 HTTP 请求。
2. 提取客户端 IP。
3. 检查黑白名单。
4. 执行规则引擎检测。
5. 执行 AI 检测。
6. 对首次访问非静态资源的客户端展示安全验证页。
7. 通过后转发到真实后端。

## 5. 静态资源处理

以下资源类型默认跳过安全验证页：

```text
.css .js .png .jpg .jpeg .gif .svg .ico .woff .woff2 .ttf .eot
```

## 6. HTTPS 说明

当前代理主要面向 HTTP 入口。

生产 HTTPS 推荐部署方式：

```text
公网用户 -> Nginx/SLB HTTPS -> WAF HTTP -> 后端 HTTP
```

或在云负载均衡层终止 TLS，再将 HTTP 转发给 WAF。

## 7. 常见验证命令

```bash
curl -I http://服务器IP/
curl -I http://127.0.0.1:8080/
curl -I http://127.0.0.1:80/
journalctl -u zhiyu-waf -n 100 --no-pager
```

## 8. 关闭接管

```bash
sed -i 's/iptables_enable: true/iptables_enable: false/' /opt/zhiyu-waf/configs/zhiyu-waf.yaml
systemctl restart zhiyu-waf
```

服务关闭接管后会清理 `ZHIYU_WAF_REDIRECT` 链。
