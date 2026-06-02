# ===== 阶段1：构建 Go 二进制 =====
FROM golang:1.21-alpine AS builder

# 国内构建加速（如在海外可改为 direct）
ENV GOPROXY=https://goproxy.cn,direct \
    CGO_ENABLED=0 \
    GOOS=linux

WORKDIR /src/backend

# 先拷依赖清单，利用层缓存
COPY backend/go.mod backend/go.sum ./
RUN go mod download

# 再拷源码并编译
COPY backend/ ./
RUN go build -ldflags="-s -w" -o /out/server .

# ===== 阶段2：精简运行镜像 =====
FROM alpine:3.19

# ca-certificates: HTTPS；tzdata: 时区；su-exec: 降权；wget: 健康检查
RUN apk add --no-cache ca-certificates tzdata su-exec wget \
    && addgroup -S app && adduser -S -G app app

ENV TZ=Asia/Shanghai \
    FRONTEND_DIR=/app/frontend \
    UPLOAD_DIR=/app/uploads \
    SERVER_PORT=8080

WORKDIR /app

# 拷入二进制与前端静态资源
COPY --from=builder /out/server /app/server
COPY frontend/ /app/frontend/
COPY deploy/docker-entrypoint.sh /usr/local/bin/docker-entrypoint.sh

RUN chmod +x /app/server /usr/local/bin/docker-entrypoint.sh \
    && mkdir -p /app/uploads && chown -R app:app /app/uploads

EXPOSE 8080

HEALTHCHECK --interval=15s --timeout=5s --start-period=20s --retries=5 \
    CMD wget -qO- http://127.0.0.1:8080/health || exit 1

ENTRYPOINT ["/usr/local/bin/docker-entrypoint.sh"]
CMD ["/app/server"]
