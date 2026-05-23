#!/usr/bin/env bash
set -Eeuo pipefail

APP_NAME="zhiyu-waf"
INSTALL_DIR="/opt/zhiyu-waf"
SERVICE_NAME="zhiyu-waf"
CONFIG_FILE=""
BACKEND_ADDR="127.0.0.1:80"
WAF_PORT="8080"
PUBLIC_PORT="80"
DASHBOARD_PORT="9090"
IPTABLES_ENABLE="true"
LICENSE_CENTER_URL="https://license.zhiyuwaf.com"
JWT_SECRET=""
BUILD_BACKEND="auto"
BUILD_FRONTEND="auto"
OPEN_FIREWALL="true"

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

usage() {
  cat <<'USAGE'
zhiyu-waf 一键安装/更新脚本

用法：
  sudo bash scripts/install-zhiyu-waf.sh [选项]

常用选项：
  --install-dir /opt/zhiyu-waf         安装目录
  --backend 127.0.0.1:80               真实业务回源地址
  --public-port 80                     公网入口端口，会转发到 WAF
  --waf-port 8080                      WAF 代理监听端口
  --dashboard-port 9090                控制台端口
  --license-center URL      授权中心地址
  --no-iptables                        不启用端口转发接管
  --no-firewall                        不自动配置 firewalld/防火墙放行
  --build-backend true|false|auto      是否编译后端，默认 auto
  --build-frontend true|false|auto     是否编译前端，默认 auto

示例：
  sudo bash scripts/install-zhiyu-waf.sh --backend 127.0.0.1:3000 --public-port 80
USAGE
}

log() {
  printf '\033[1;34m[ZhiYu-WAF]\033[0m %s\n' "$*"
}

die() {
  printf '\033[1;31m[ZhiYu-WAF] ERROR:\033[0m %s\n' "$*" >&2
  exit 1
}

need_root() {
  if [[ "${EUID}" -ne 0 ]]; then
    die "请使用 root 执行，例如 sudo bash scripts/install-zhiyu-waf.sh"
  fi
}

parse_args() {
  while [[ $# -gt 0 ]]; do
    case "$1" in
      --install-dir) INSTALL_DIR="$2"; shift 2 ;;
      --backend) BACKEND_ADDR="$2"; shift 2 ;;
      --public-port) PUBLIC_PORT="$2"; shift 2 ;;
      --waf-port) WAF_PORT="$2"; shift 2 ;;
      --dashboard-port) DASHBOARD_PORT="$2"; shift 2 ;;
      --license-center) LICENSE_CENTER_URL="$2"; shift 2 ;;
      --jwt-secret) JWT_SECRET="$2"; shift 2 ;;
      --no-iptables) IPTABLES_ENABLE="false"; shift ;;
      --no-firewall) OPEN_FIREWALL="false"; shift ;;
      --build-backend) BUILD_BACKEND="$2"; shift 2 ;;
      --build-frontend) BUILD_FRONTEND="$2"; shift 2 ;;
      -h|--help) usage; exit 0 ;;
      *) die "未知参数：$1" ;;
    esac
  done
}

command_exists() {
  command -v "$1" >/dev/null 2>&1
}

rand_secret() {
  if command_exists openssl; then
    openssl rand -hex 32
  else
    tr -dc 'A-Za-z0-9' </dev/urandom | head -c 64
  fi
}

validate_inputs() {
  [[ "${BACKEND_ADDR}" == *:* ]] || die "--backend 必须是 host:port，例如 127.0.0.1:80"
  [[ "${WAF_PORT}" =~ ^[0-9]+$ ]] || die "--waf-port 必须是数字"
  [[ "${PUBLIC_PORT}" =~ ^[0-9]+$ ]] || die "--public-port 必须是数字"
  [[ "${DASHBOARD_PORT}" =~ ^[0-9]+$ ]] || die "--dashboard-port 必须是数字"
  if [[ -z "${JWT_SECRET}" ]]; then
    JWT_SECRET="$(rand_secret)"
  fi
}

build_backend_if_needed() {
  local target="${ROOT_DIR}/bin/${APP_NAME}"
  local should_build="${BUILD_BACKEND}"
  if [[ "${should_build}" == "auto" ]]; then
    if [[ -x "${target}" && "$(uname -m)" != "arm64" ]]; then
      should_build="false"
    else
      should_build="true"
    fi
  fi

  if [[ "${should_build}" == "true" ]]; then
    command_exists go || die "未找到 go，无法编译后端。请先安装 Go，或把已编译的 bin/zhiyu-waf 放到项目里"
    command_exists gcc || log "未检测到 gcc；如果使用 sqlite3，go-sqlite3 可能需要 gcc"
    log "编译后端..."
    (cd "${ROOT_DIR}" && CGO_ENABLED=1 go build -o "${target}" ./cmd/zhiyu-waf)
  fi

  [[ -x "${target}" ]] || die "未找到可执行文件：${target}"
}

build_frontend_if_needed() {
  local dist="${ROOT_DIR}/web/dist/index.html"
  local should_build="${BUILD_FRONTEND}"
  if [[ "${should_build}" == "auto" ]]; then
    [[ -f "${dist}" ]] && should_build="false" || should_build="true"
  fi

  if [[ "${should_build}" == "true" ]]; then
    command_exists npm || die "未找到 npm，无法编译前端。请先安装 Node.js/npm，或提前构建 web/dist"
    log "编译前端..."
    (cd "${ROOT_DIR}/web" && npm install && npm run build)
  fi

  [[ -f "${dist}" ]] || die "未找到前端构建目录：${ROOT_DIR}/web/dist"
}

backup_existing() {
  if [[ -d "${INSTALL_DIR}" ]]; then
    local ts
    ts="$(date +%Y%m%d%H%M%S)"
    local backup="${INSTALL_DIR}/backups/install.${ts}"
    log "备份当前安装到 ${backup}"
    mkdir -p "${backup}"
    [[ -f "${INSTALL_DIR}/bin/${APP_NAME}" ]] && cp -a "${INSTALL_DIR}/bin/${APP_NAME}" "${backup}/${APP_NAME}" || true
    [[ -f "${INSTALL_DIR}/configs/zhiyu-waf.yaml" ]] && cp -a "${INSTALL_DIR}/configs/zhiyu-waf.yaml" "${backup}/zhiyu-waf.yaml" || true
    [[ -d "${INSTALL_DIR}/web/dist" ]] && cp -a "${INSTALL_DIR}/web/dist" "${backup}/dist" || true
  fi
}

install_files() {
  log "安装文件到 ${INSTALL_DIR}"
  mkdir -p \
    "${INSTALL_DIR}/bin" \
    "${INSTALL_DIR}/configs/rules" \
    "${INSTALL_DIR}/data" \
    "${INSTALL_DIR}/logs" \
    "${INSTALL_DIR}/web"

  install -m 0755 "${ROOT_DIR}/bin/${APP_NAME}" "${INSTALL_DIR}/bin/${APP_NAME}"
  cp -a "${ROOT_DIR}/configs/rules/." "${INSTALL_DIR}/configs/rules/"
  rm -rf "${INSTALL_DIR}/web/dist.new"
  mkdir -p "${INSTALL_DIR}/web/dist.new"
  cp -a "${ROOT_DIR}/web/dist/." "${INSTALL_DIR}/web/dist.new/"
  rm -rf "${INSTALL_DIR}/web/dist"
  mv "${INSTALL_DIR}/web/dist.new" "${INSTALL_DIR}/web/dist"

  CONFIG_FILE="${INSTALL_DIR}/configs/zhiyu-waf.yaml"
  if [[ ! -f "${CONFIG_FILE}" ]]; then
    cp "${ROOT_DIR}/configs/zhiyu-waf.yaml" "${CONFIG_FILE}"
  fi
}

write_config() {
  log "写入配置：${CONFIG_FILE}"
  cat >"${CONFIG_FILE}" <<EOF
proxy:
  listen_addr: ":${WAF_PORT}"
  backend_addr: "${BACKEND_ADDR}"
  iptables_enable: ${IPTABLES_ENABLE}
  iptables_port: ${PUBLIC_PORT}
  read_timeout: 30
  write_timeout: 30

dashboard:
  listen_addr: ":${DASHBOARD_PORT}"
  jwt_secret: "${JWT_SECRET}"
  cors_origins:
    - "*"

license:
  center_url: "${LICENSE_CENTER_URL}"
  timeout: 8

ai:
  enabled: true
  provider: "openai"
  async_timeout: 5
  cache_ttl: 300
  max_requests_per_min: 60
  fail_open: true
  providers:
    openai:
      api_key: ""
      model: "mimo-v2-pro"
      base_url: "https://token-plan-cn.xiaomimimo.com/v1"

engine:
  rules_dir: "${INSTALL_DIR}/configs/rules"
  rate_limit:
    requests_per_minute: 60
    burst_size: 10

ssh:
  enabled: true
  log_path: ""
  max_fails: 5
  ban_minutes: 30

storage:
  path: "${INSTALL_DIR}/data/zhiyu-waf.db"
EOF
}

write_systemd() {
  log "写入 systemd 服务"
  cat >/etc/systemd/system/${SERVICE_NAME}.service <<EOF
[Unit]
Description=ZhiYu-WAF - Web Application Firewall
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
WorkingDirectory=${INSTALL_DIR}
ExecStart=${INSTALL_DIR}/bin/${APP_NAME} -config ${INSTALL_DIR}/configs/zhiyu-waf.yaml
Restart=always
RestartSec=5
Environment=PATH=/usr/local/go/bin:/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin

[Install]
WantedBy=multi-user.target
EOF
}

configure_firewall() {
  if [[ "${OPEN_FIREWALL}" != "true" ]]; then
    return
  fi

  if command_exists firewall-cmd && systemctl is-active firewalld >/dev/null 2>&1; then
    log "配置 firewalld 放行端口"
    firewall-cmd --permanent --add-port="${PUBLIC_PORT}/tcp" >/dev/null || true
    firewall-cmd --permanent --add-port="${DASHBOARD_PORT}/tcp" >/dev/null || true
    firewall-cmd --permanent --add-port="443/tcp" >/dev/null || true
    firewall-cmd --reload >/dev/null || true
  elif command_exists ufw && ufw status >/dev/null 2>&1; then
    log "配置 ufw 放行端口"
    ufw allow "${PUBLIC_PORT}/tcp" >/dev/null || true
    ufw allow "${DASHBOARD_PORT}/tcp" >/dev/null || true
    ufw allow "443/tcp" >/dev/null || true
  else
    log "未检测到已启用的 firewalld/ufw，跳过防火墙端口放行"
  fi
}

start_service() {
  log "启动服务"
  systemctl daemon-reload
  systemctl enable "${SERVICE_NAME}" >/dev/null
  systemctl restart "${SERVICE_NAME}"
  sleep 2
  systemctl is-active --quiet "${SERVICE_NAME}" || {
    systemctl status "${SERVICE_NAME}" --no-pager -l || true
    die "服务启动失败"
  }
}

verify_install() {
  log "验证服务"
  curl -fsS "http://127.0.0.1:${DASHBOARD_PORT}/health" >/dev/null || die "控制台健康检查失败"

  if [[ "${IPTABLES_ENABLE}" == "true" ]]; then
    if command_exists iptables; then
      iptables -t nat -S | grep -q "ZHIYU_WAF_REDIRECT" || log "未看到 ZHIYU_WAF_REDIRECT 链，服务可能稍后写入或当前环境不支持 iptables"
    fi
  fi

  log "安装完成"
  echo
  echo "控制台地址： http://服务器IP:${DASHBOARD_PORT}/"
  echo "代理入口：   服务器IP:${PUBLIC_PORT} -> WAF:${WAF_PORT} -> ${BACKEND_ADDR}"
  echo "配置文件：   ${CONFIG_FILE}"
  echo "服务状态：   systemctl status ${SERVICE_NAME} --no-pager -l"
}

main() {
  parse_args "$@"
  need_root
  validate_inputs
  build_backend_if_needed
  build_frontend_if_needed
  backup_existing
  install_files
  write_config
  write_systemd
  configure_firewall
  start_service
  verify_install
}

main "$@"
