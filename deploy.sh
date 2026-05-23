#!/bin/bash
# ZhiYu-WAF 一键部署脚本
# 用法: bash deploy.sh <server_ip> <password>

SERVER="${1:?用法: bash deploy.sh <server_ip> <password>}"
PASS="${2:?用法: bash deploy.sh <server_ip> <password>}"
DEPLOY_TAR="/tmp/zhiyu-waf-deploy.tar.gz"
REMOTE_DIR="/opt/zhiyu-waf"

set -e

echo "==> 上传部署包..."
sshpass -p "$PASS" scp -o StrictHostKeyChecking=no "$DEPLOY_TAR" root@${SERVER}:/tmp/

echo "==> 解压并部署..."
sshpass -p "$PASS" ssh -o StrictHostKeyChecking=no root@${SERVER} bash -s <<'REMOTE'
set -e
mkdir -p /opt/zhiyu-waf
cd /opt/zhiyu-waf
tar xzf /tmp/zhiyu-waf-deploy.tar.gz
chmod +x zhiyu-waf
mkdir -p configs
rm -f /tmp/zhiyu-waf-deploy.tar.gz

# 创建 systemd 服务
cat > /etc/systemd/system/zhiyu-waf.service <<'EOF'
[Unit]
Description=ZhiYu-WAF
After=network.target

[Service]
Type=simple
WorkingDirectory=/opt/zhiyu-waf
ExecStart=/opt/zhiyu-waf/zhiyu-waf -config /opt/zhiyu-waf/configs/zhiyu-waf.yaml
Restart=always
RestartSec=3
LimitNOFILE=65535

[Install]
WantedBy=multi-user.target
EOF

systemctl daemon-reload
systemctl enable zhiyu-waf
systemctl restart zhiyu-waf
echo "==> 服务已启动"
systemctl status zhiyu-waf --no-pager -l | head -20
REMOTE

echo "==> 部署完成！"
