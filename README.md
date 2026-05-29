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
| 前端 | 待定 |

## 项目结构

```
gift/
├── server/
│   ├── main.go                  # 应用入口
│   ├── go.mod                   # Go 模块定义
│   ├── config/
│   │   ├── config.go            # 配置加载（Viper）
│   │   ├── config.yml           # 配置文件
│   │   └── db.go                # 数据库初始化（GORM）
│   ├── models/
│   │   └── user.go              # User 模型 + LoginRequest DTO
│   ├── controllers/
│   │   ├── auth.go              # 登录
│   │   └── user.go              # 用户管理
│   ├── middlewares/
│   │   └── auth.go              # JWT 鉴权 + 管理员中间件
│   ├── router/
│   │   └── router.go            # 路由定义
│   ├── utils/
│   │   └── utils.go             # bcrypt 密码工具 + JWT 工具
│   └── global/
│       └── global.go            # 全局变量（DB 实例）
├── obsidian/
│   └── step.md                  # 开发笔记
└── README.md
```

## API 接口

### 公开接口

| 方法 | 路径 | 说明 |
|------|------|------|
| POST | `/api/auth/login` | 用户登录，返回 JWT token |

### 管理员接口（需 JWT + Admin 权限）

| 方法 | 路径 | 说明 |
|------|------|------|
| POST | `/api/admin/users` | 创建用户 |

## 快速开始

### 环境要求

- Go 1.26+
- MySQL 8.0+

### 配置

编辑 `server/config/config.yml`，修改数据库连接信息：

```yaml
database:
  dsn: "用户名:密码@tcp(地址:端口)/数据库名?charset=utf8mb4&parseTime=True&loc=Local"
```

### 运行

```bash
cd server
go run main.go
```

服务默认监听 `:8080`。

## 许可证

MIT
