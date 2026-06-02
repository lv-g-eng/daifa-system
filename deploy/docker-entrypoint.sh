#!/bin/sh
set -e

# 以 root 启动时：修正上传目录属主，再降权到非 root 用户运行
# （仿 sub2api：root 处理权限 -> su-exec 切换 app 用户）
if [ "$(id -u)" = "0" ]; then
  mkdir -p "${UPLOAD_DIR:-/app/uploads}"
  chown -R app:app "${UPLOAD_DIR:-/app/uploads}" 2>/dev/null || true
  exec su-exec app "$@"
fi

exec "$@"
