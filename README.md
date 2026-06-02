# H5 代发系统

基于 Go + MySQL 的小红书内容代发管理平台

## 项目结构

```
代发系统_1/
├── backend/                # Go 后端
│   ├── main.go            # 入口文件
│   ├── go.mod             # 依赖管理
│   ├── config/            # 配置
│   ├── models/            # 数据模型
│   ├── handlers/          # API 处理器
│   └── middleware/        # 中间件
├── frontend/              # H5 前端
│   ├── index.html         # 登录页
│   ├── css/style.css      # 样式
│   ├── js/api.js          # API 封装
│   ├── admin/             # 总后台
│   ├── merchant/          # 商家后台
│   └── user/              # 用户端
└── database/
    └── init.sql           # 数据库初始化脚本
```

## 环境要求

- Go 1.21+
- MySQL 8.0+
- 现代浏览器

## 快速开始

### 1. 初始化数据库

```bash
# 登录 MySQL 并执行初始化脚本
Get-Content database\init.sql -Encoding UTF8 | mysql -u root -p123456 --default-character-set=utf8mb4
```

### 2. 安装 Go 依赖

```bash
cd backend
go mod tidy
```

### 3. 启动后端服务

```bash
cd backend
go run main.go
```

服务将在 http://localhost:8080 启动

### 4. 访问系统

- 登录页：http://localhost:8080
- 总后台：http://localhost:8080/admin
- 商家后台：http://localhost:8080/merchant
- 用户端：http://localhost:8080/user

## 测试账号

| 角色 | 用户名 | 密码 |
|------|--------|------|
| 管理员 | admin | 123456 |
| 商家 | merchant1 | 123456 |
| 用户 | user1 | 123456 |

## 功能模块

### 总后台 (admin)
- 创建/管理商家账号
- 设置账号有效期
- 验证用户收款码

### 商家后台 (merchant)
- 📊 数据统计
- 📸 内容上传（批量）
- 🔗 发布码生成
- 📋 任务发布
- ✅ 任务审核（链接有效性检查）
- 👥 用户管理（拉黑）
- 💰 提现管理

### 用户端 (user)
- 🏠 首页（任务列表、余额）
- 📥 内容下载（扫码一键发布）
- 🔙 链接回传
- 👤 个人中心（提现）
- 📣 裂变代发（邀请分成）

## 数据库配置

数据库使用明码配置（按需求）：
- 主机：127.0.0.1
- 端口：3306
- 用户：root
- 密码：123456
- 数据库：distribution_system

## API 文档

### 认证
- `POST /api/login` - 登录
- `POST /api/register` - 注册

### 管理员
- `GET /api/admin/stats` - 统计
- `POST /api/admin/merchants` - 创建商家
- `GET /api/admin/merchants` - 商家列表

### 商家
- `GET /api/merchant/stats` - 统计
- `POST /api/merchant/materials` - 上传素材
- `POST /api/merchant/publish-codes` - 生成发布码
- `POST /api/merchant/tasks` - 发布任务
- `GET /api/merchant/submissions` - 审核列表
- `PUT /api/merchant/submissions/:id` - 审核操作

### 用户
- `GET /api/user/home` - 首页数据
- `GET /api/user/tasks` - 任务列表
- `POST /api/user/submissions` - 提交链接
- `POST /api/user/withdraw` - 申请提现

## 技术栈

- **后端**：Go + Gin + GORM + JWT
- **前端**：HTML5 + CSS3 + JavaScript
- **数据库**：MySQL 8.0
- **UI 设计**：现代化浅色主题
