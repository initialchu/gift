# 念礼 (GiftMemo)

礼金来往管理应用 —— 方便记录和查询人情往来记录。

## 背景

日常生活中的人情往来记录分散在多个礼薄上，时间拉长后查询和核对变得困难。念礼致力于提供一个方便的管理工具，帮助你轻松追踪每一份人情往来。

## 核心功能

- **礼薄管理**：创建事件对应的礼薄，记录事件名称、时间、礼金、姓名、地址及附赠品
- **人情卡片**：将同一人的所有往来聚合为一张卡片，自动计算"来/往"差值，一目了然
- **卡片搜索**：通过人名快速搜索对应的人情卡片
- **管理员添加用户**：无公开注册，由管理员统一管理用户账号

## 技术栈

| 层级 | 技术 |
|------|------|
| 后端框架 | Go + Gin |
| 数据库 | MySQL |
| ORM | GORM |
| 配置管理 | Viper |
| 鉴权 | JWT（golang-jwt/jwt/v5） |
| 密码加密 | bcrypt |
| 前端 | Vue 3 + Vite + TypeScript |

## 项目结构

```
gift/
├── server/
│   ├── main.go                  # 应用入口（优雅关闭 + 种子数据初始化）
│   ├── go.mod                   # Go 模块定义
│   ├── config/
│   │   ├── config.go            # 配置加载（Viper）
│   │   ├── config.yml           # 配置文件（模板，提交到 git）
│   │   ├── config.local.yml     # 本地敏感配置（不提交，已 gitignore）
│   │   └── db.go                # 数据库初始化 + AutoMigrate + 种子管理员
│   ├── models/
│   │   ├── user.go              # User 模型 + LoginRequest DTO
│   │   └── gift.go              # GiftBook 模型 + GiftRecord 模型
│   ├── controllers/
│   │   ├── auth.go              # 登录
│   │   ├── user.go              # 用户管理
│   │   ├── giftbook.go          # 礼薄 CRUD
│   │   └── giftrecord.go        # 礼金记录 CRUD（含归属校验）
│   ├── middlewares/
│   │   └── auth.go              # AuthMiddleware（JWT 鉴权）+ AdminMiddleware（管理员授权）
│   ├── router/
│   │   └── router.go            # 路由定义
│   ├── utils/
│   │   └── utils.go             # bcrypt 密码工具 + JWT 生成/解析
│   └── global/
│       └── global.go            # 全局变量（DB 实例）
├── client/                      # 前端（Vue 3 + Vite + TypeScript）
│   ├── src/
│   │   ├── App.vue
│   │   ├── main.ts
│   │   ├── router/
│   │   │   └── index.ts
│   │   └── stores/
│   │       └── counter.ts
│   ├── index.html
│   └── vite.config.ts
├── obsidian/
│   └── step.md                  # 开发笔记
└── README.md
```

## API 接口

### 公开接口

| 方法 | 路径 | 说明 |
|------|------|------|
| POST | `/api/auth/login` | 用户登录，返回 JWT token |

### 需认证接口（JWT）

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/giftbooks` | 礼薄列表 |
| GET | `/api/giftbook/:id` | 礼薄详情（含所有记录） |
| GET | `/api/giftrecord/:id/records` | 指定礼薄的礼金记录列表 |

### 管理员接口（JWT + Admin）

| 方法 | 路径 | 说明 |
|------|------|------|
| POST | `/api/admin/create` | 创建用户 |
| POST | `/api/admin/giftbook` | 创建礼薄 |
| POST | `/api/admin/giftbook/edit/:id` | 修改礼薄 |
| POST | `/api/admin/giftbook/:id` | 删除礼薄（级联删除记录） |
| POST | `/api/admin/giftrecord/:id/records` | 添加礼金记录 |
| POST | `/api/admin/giftrecord/:id/records/:rid` | 删除礼金记录 |
| POST | `/api/admin/giftrecord/:id/records/:rid/edit` | 修改礼金记录 |

## 快速开始

### 环境要求

- Go 1.26+
- MySQL 8.0+
- Node.js 20+（前端开发）

### 后端配置

1. 复制 `server/config/config.yml` 为 `server/config/config.local.yml`
2. 编辑 `config.local.yml`，填入真实的数据库密码、JWT 密钥等敏感信息：
```yaml
database:
  dsn: "用户名:密码@tcp(地址:端口)/giftmemo?charset=utf8mb4&parseTime=True&loc=Local"

admin:
  name: "管理员用户名"
  password: "管理员密码"

jwt:
  secret: "你的JWT密钥"
  expire_hours: 24
```
3. `config.local.yml` 已加入 `.gitignore`，不会被提交到 git

### 运行

```bash
# 后端
cd server
go run main.go

# 前端（新终端）
cd client
npm install
npm run dev
```

后端端口通过 `config.yml` 中 `app.port` 配置（默认 `:8080`）。服务支持优雅关闭（Ctrl+C 后等待最多 5 秒处理完现有请求）。

## 许可证

MIT
