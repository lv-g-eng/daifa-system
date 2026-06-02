# 代发系统 · Docker 一键部署

效仿 `sub2api` 的部署方案：**准备脚本生成密钥 + `docker compose` 启动**。
一条命令即可在 Linux 服务器上完成数据库、表结构、种子账号的全自动初始化。

## 架构

```
app (Go+Gin 单容器，内置前端静态页，:8080)
 └── depends_on → mysql:8.0  (healthcheck 通过后才启动 app)
```

- 数据落本地目录，便于 `tar` 迁移：`deploy/mysql_data/`（数据库）、`deploy/uploads/`（上传文件）
- **首次启动自举**：MySQL 把 `database/init.sql` 挂进 `/docker-entrypoint-initdb.d/`，自动建表并灌入种子账号/平台/系统配置；应用启动后 GORM `AutoMigrate` 再做一次幂等补全。
- MySQL **不对宿主机暴露端口**，仅走内部网络（安全加固）。
- 密钥（MySQL 密码、JWT）由部署脚本用 `openssl rand -hex 32` 自动生成，不再是写死的 `123456`。

## 环境要求

- Docker Engine 20+ 与 docker compose 插件
- Linux 服务器（脚本为 bash；含 `openssl`/`/dev/urandom` 之一即可生成密钥）

## 一键部署

```bash
# 1. 上传/克隆项目到服务器后，进入 deploy 目录
cd 代发系统/deploy

# 2. 赋予脚本执行权限并运行（生成密钥 -> 创建目录 -> 构建并启动）
chmod +x docker-deploy.sh docker-entrypoint.sh
./docker-deploy.sh
```

完成后访问：

| 入口 | 地址 |
|------|------|
| 登录页 | `http://<服务器IP>:8080` |
| 管理后台 | `http://<服务器IP>:8080/admin` |
| 商家后台 | `http://<服务器IP>:8080/merchant` |
| 用户端 | `http://<服务器IP>:8080/user` |

默认账号（来自 `init.sql` 种子数据）：

| 角色 | 用户名 | 密码 |
|------|--------|------|
| 管理员 | admin | 123456 |
| 商家 | merchant1 | 123456 |
| 用户 | user1 | 123456 |

> 自动生成的数据库密码与 JWT 密钥保存在 `deploy/.env`（权限 600），请妥善保管。

## 常用命令

```bash
docker compose logs -f app      # 查看应用日志
docker compose ps               # 查看容器状态
docker compose restart app      # 重启应用
docker compose down             # 停止并移除容器（数据保留在本地目录）
docker compose up -d --build    # 改动代码后重新构建并启动
```

仅生成配置、不启动：

```bash
./docker-deploy.sh --no-start
```

## 修改端口 / 配置

编辑 `deploy/.env`：

```ini
APP_PORT=8080          # 改这里换对外端口
JWT_EXPIRE_HOUR=168    # JWT 过期小时数
TZ=Asia/Shanghai
```

改完执行 `docker compose up -d` 生效。

## 迁移到新服务器

```bash
# 旧机
cd 代发系统/deploy
docker compose down
tar czf distribution-backup.tar.gz mysql_data uploads .env

# 新机：拷贝整个项目 + 解压备份到 deploy/ 后
cd 代发系统/deploy
tar xzf distribution-backup.tar.gz
docker compose up -d --build
```

## 配置说明（环境变量优先）

应用 `backend/config/config.go` 已改为**环境变量优先、本地默认值兜底**，因此：

- Docker 部署：由 `docker-compose.yml` 注入 `DB_HOST=mysql`、随机 `DB_PASSWORD`、`JWT_SECRET` 等。
- 本地开发：不设环境变量时回退到原有默认值（`127.0.0.1:3306` / `root` / `123456`），`go run main.go` 行为不变。

| 变量 | 默认值 | 说明 |
|------|--------|------|
| `DB_HOST` | `127.0.0.1` | 数据库主机（容器内为 `mysql`） |
| `DB_PORT` | `3306` | 数据库端口 |
| `DB_USER` | `root` | 数据库用户 |
| `DB_PASSWORD` | `123456` | 数据库密码（部署时自动生成） |
| `DB_NAME` | `distribution_system` | 数据库名 |
| `JWT_SECRET` | `distribution-system-secret-key-2024` | JWT 密钥（部署时自动生成） |
| `JWT_EXPIRE_HOUR` | `168` | JWT 过期小时数 |
| `SERVER_PORT` | `8080` | 服务监听端口（可写 `8080` 或 `:8080`） |
| `FRONTEND_DIR` | `../frontend` | 前端静态目录（镜像内为 `/app/frontend`） |
| `UPLOAD_DIR` | `./uploads` | 上传目录（镜像内为 `/app/uploads`） |

## 故障排查

- **app 反复重启**：`docker compose logs app`，多为数据库未就绪或密码不一致——确认 `.env` 的 `DB_PASSWORD` 没被手改成与 `mysql_data` 已初始化时不一致的值。
- **种子账号没出现**：种子仅在 MySQL **首次**初始化（`mysql_data/` 为空）时执行。若想重新灌种子，先 `docker compose down`，删除 `deploy/mysql_data/` 再重跑。
- **端口被占用**：改 `deploy/.env` 里的 `APP_PORT`。
- **海外服务器构建慢**：编辑 `deploy/Dockerfile`，把 `GOPROXY` 改为 `direct`。

## ⚠️ 安全提示

本项目用户密码为**明码存储**（`init.sql` 内注释写明"按需求"）。本次部署仅在基础设施层加固（随机 DB/JWT 密钥、DB 不公网暴露）。若需将密码改为哈希存储，属于独立的应用层改造任务。
