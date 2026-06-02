#!/usr/bin/env bash
# 代发系统 一键部署脚本（效仿 sub2api/deploy/docker-deploy.sh）
# 功能：检查依赖 -> 自动生成密钥写入 .env -> 创建数据目录 -> 构建并启动
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR"

GREEN='\033[0;32m'; YELLOW='\033[1;33m'; RED='\033[0;31m'; NC='\033[0m'
info(){ printf "${GREEN}[INFO]${NC} %s\n" "$1"; }
warn(){ printf "${YELLOW}[WARN]${NC} %s\n" "$1"; }
err(){  printf "${RED}[ERR ]${NC} %s\n" "$1" >&2; }

# ---------- 1. 依赖检查 ----------
command -v docker >/dev/null 2>&1 || { err "未找到 docker，请先安装 Docker：https://docs.docker.com/engine/install/"; exit 1; }
if docker compose version >/dev/null 2>&1; then
  DC="docker compose"
elif command -v docker-compose >/dev/null 2>&1; then
  DC="docker-compose"
else
  err "未找到 docker compose（请安装 docker compose 插件）"; exit 1
fi

# 生成 32 字节随机十六进制密钥：优先 openssl，回退 /dev/urandom
generate_secret(){
  if command -v openssl >/dev/null 2>&1; then
    openssl rand -hex 32
  else
    head -c 32 /dev/urandom | od -An -tx1 | tr -d ' \n'
  fi
}

# 跨平台 sed -i（GNU 与 BSD/macOS 兼容）
sed_inplace(){
  if sed --version >/dev/null 2>&1; then
    sed -i "$@"
  else
    sed -i '' "$@"
  fi
}

# ---------- 2. 准备 .env ----------
if [ -f .env ]; then
  warn ".env 已存在，跳过密钥生成（如需重置请删除 .env 后重跑）"
else
  cp .env.example .env
  DB_PASSWORD="$(generate_secret)"
  JWT_SECRET="$(generate_secret)"
  sed_inplace "s|^DB_PASSWORD=.*|DB_PASSWORD=${DB_PASSWORD}|" .env
  sed_inplace "s|^JWT_SECRET=.*|JWT_SECRET=${JWT_SECRET}|" .env
  chmod 600 .env
  info "已生成随机 DB_PASSWORD 与 JWT_SECRET 并写入 .env (权限 600)"
fi

# ---------- 3. 创建数据目录 ----------
mkdir -p mysql_data uploads
info "数据目录就绪：mysql_data/  uploads/"

# ---------- 4. 启动 ----------
if [ "${1:-}" = "--no-start" ]; then
  info "已完成准备（--no-start）。手动启动：$DC up -d --build"
  exit 0
fi

info "开始构建镜像并启动容器（首次构建可能需要几分钟）..."
$DC up -d --build

# ---------- 5. 输出凭证 ----------
APP_PORT="$(grep -E '^APP_PORT=' .env | cut -d= -f2)"; APP_PORT="${APP_PORT:-8080}"
echo
info "========================================"
info " 部署完成！容器已在后台启动"
info " 访问地址: http://<服务器IP>:${APP_PORT}"
info "   登录页  /        管理后台 /admin"
info "   商家后台 /merchant 用户端  /user"
info " ---- 默认账号（来自 init.sql 种子数据）----"
info "   管理员: admin     / 123456"
info "   商家:   merchant1 / 123456"
info "   用户:   user1     / 123456"
info " ---- 自动生成的密钥见 deploy/.env，请妥善保管 ----"
info "========================================"
echo
info "查看日志: $DC logs -f app"
info "停止服务: $DC down"
