# 后端开发参考

---

# 1. 项目概述

念礼（GiftMemo）—— 礼金来往管理应用

## 技术栈
- 后端框架：Go + Gin
- ORM：GORM
- 数据库：MySQL
- 配置管理：Viper
- 鉴权：JWT（golang-jwt/jwt/v5）

## 项目结构
```
server/
├── main.go              # 应用入口
├── config/
│   ├── config.go        # 配置加载（Viper）
│   ├── config.yml       # 配置文件
│   └── db.go            # 数据库初始化（GORM）
├── models/              # 数据模型（对应数据库表）
├── controllers/         # 控制器（处理请求）
├── router/              # 路由定义
├── utils/               # 工具函数
└── global/              # 全局变量（DB 实例等）
```

---

# 2. 用户认证模块

## 2.1 User 模型 (`models/user.go`)

```go
type User struct {
    gorm.Model
    Username string `gorm:"unique;not null" json:"username" binding:"required"`
    Password string `gorm:"not null" json:"-" binding:"required"`
}
```

要点：
- `gorm.Model`：自动提供 ID、CreatedAt、UpdatedAt、DeletedAt（软删除）
- `gorm:"unique;not null"`：数据库层面保证用户名唯一且非空
- `json:"-"`：Password 不会被序列化到 JSON 响应中，也不会从 JSON 请求中绑定

## 2.2 DTO 结构体 —— LoginRequest

User 模型的 Password 字段因 `json:"-"` 无法从请求体绑定，需要单独的 DTO：

```go
type LoginRequest struct {
    Username string `json:"username" binding:"required"`
    Password string `json:"password" binding:"required"`
}
```

User 模型负责数据库映射，LoginRequest 负责接收 API 请求，各司其职。

## 2.3 密码加密 (`utils/utils.go`)

```go
// HashPassword 使用 bcrypt 加密密码，cost=12
func HashPassword(pwd string) (string, error) {
    hash, err := bcrypt.GenerateFromPassword([]byte(pwd), 12)
    return string(hash), err
}

// CheckPassword 验证密码是否匹配
func CheckPassword(hashedPwd, plainPwd string) error {
    return bcrypt.CompareHashAndPassword([]byte(hashedPwd), []byte(plainPwd))
}
```

`bcrypt.CompareHashAndPassword` 原理：
- 从哈希值中提取 salt 和 cost
- 对明文密码用同样的 salt 和 cost 做哈希
- 比较结果 → 一致返回 nil，不一致返回 error

## 2.4 JWT 生成 (`utils/utils.go`)

```go
func GenerateJWT(username string) (string, error) {
    token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
        "username": username,
        "exp":      time.Now().Add(time.Hour * 24).Unix(),
    })
    signedToken, err := token.SignedString([]byte("your-secret-key"))
    return signedToken, err
}
```

- 签名算法：HS256
- 过期时间：24 小时
- 密钥应放到 config.yml 中统一管理，不要硬编码

## 2.5 登录流程 (`controllers/auth.go`)

```
客户端 POST /api/auth/login
    { "username": "zhangsan", "password": "123456" }
           │
           ▼
┌─────────────────────────────────┐
│ 1. ShouldBindJSON → LoginRequest│  校验必填字段
│    失败 → 400 BadRequest        │
└──────────────┬──────────────────┘
           ▼
┌─────────────────────────────────┐
│ 2. DB.Where().First(&user)      │  查数据库找用户
│    未找到 → 401 用户名或密码错误  │
└──────────────┬──────────────────┘
           ▼
┌─────────────────────────────────┐
│ 3. CheckPassword(hash, plain)   │  bcrypt 比对
│    不匹配 → 401 用户名或密码错误  │
└──────────────┬──────────────────┘
           ▼
┌─────────────────────────────────┐
│ 4. GenerateJWT(username)        │  签发 token
│    失败 → 500 生成令牌失败       │
└──────────────┬──────────────────┘
           ▼
┌─────────────────────────────────┐
│ 5. 返回 200 { token }           │
└─────────────────────────────────┘
```

安全注意事项：
- 用户不存在和密码错误返回相同的错误信息，防止用户名枚举攻击
- Password 在 User 模型上设置 `json:"-"`，永远不会泄露到响应中
- bcrypt cost=12 在当前硬件下是安全与性能的合理平衡点

---

# 3. 用户管理模块

## 3.1 创建用户的基本要点

没有注册功能，用户由管理员创建。创建用户的逻辑放在 `controllers/user.go`，不要把用户 CRUD 和登录混在 `auth.go` 里：

```
auth.go     →  "我是谁"（身份认证）
user.go     →  "管理谁"（用户 CRUD）
```

管理员是一种权限角色，不是一种资源。权限控制交给中间件，具体操作按资源划分。

## 3.2 AutoMigrate 的正确用法

`AutoMigrate` 是数据库表结构的自动迁移，用于在启动时同步模型和数据库表结构。

**错误做法**：在请求处理函数中调用 `AutoMigrate`
- 每次请求都会扫描表结构，性能浪费严重
- 传的是 DTO（如 LoginRequest）而非数据库模型，无法正确建表

**正确做法**：在 `config/db.go` 的 `initDB()` 末尾调用一次：

```go
func initDB() {
    // ... 数据库连接代码 ...
    global.DB = db

    // 自动迁移（仅此一次，启动时执行）
    db.AutoMigrate(&models.User{})
}
```

这样每次启动时自动同步表结构，且只执行一次。后续新增模型也在这里追加。

## 3.3 创建用户的常见错误

**错误 1**：`Create(&req)` —— req 是 LoginRequest（DTO），不是数据库模型，无法正确写入 users 表。

**错误 2**：密码明文入库 —— 必须先用 `utils.HashPassword()` 哈希后再存入。

**错误 3**：不检查用户名重复 —— 虽然有数据库唯一索引兜底，但应该先查询并返回友好的错误提示。

## 3.4 创建用户正确流程

```
客户端 POST /api/admin/users
    { "username": "zhangsan", "password": "123456" }
           │
           ▼
┌─────────────────────────────────┐
│ 1. ShouldBindJSON → LoginRequest│  校验必填字段
│    失败 → 400 BadRequest        │
└──────────────┬──────────────────┘
           ▼
┌─────────────────────────────────┐
│ 2. 查询用户名是否已存在          │  防止重复
│    已存在 → 409 Conflict         │
└──────────────┬──────────────────┘
           ▼
┌─────────────────────────────────┐
│ 3. HashPassword(明文) → 哈希值   │  bcrypt, cost=12
│    失败 → 500 密码加密失败       │
└──────────────┬──────────────────┘
           ▼
┌─────────────────────────────────┐
│ 4. 构造 User{Username, Hash}    │  把 LoginRequest 转为 User
│    DB.Create(&user)             │  写入数据库
│    失败 → 500 创建用户失败       │
└──────────────┬──────────────────┘
           ▼
┌─────────────────────────────────┐
│ 5. 返回 201 { username }        │
└─────────────────────────────────┘
```

## 3.5 HTTP 状态码约定

| 场景 | 状态码 | 含义 |
|------|--------|------|
| 请求参数校验失败 | `400 Bad Request` | 请求格式错误 |
| 用户名已存在 | `409 Conflict` | 资源冲突 |
| 创建成功 | `201 Created` | 资源已创建 |
| 用户名或密码错误 | `401 Unauthorized` | 认证失败 |
| 服务端错误 | `500 Internal Server Error` | 服务器内部异常 |

---

# 4. JWT 中间件

（待实现）

---

# 5. 礼薄模块

（待实现）

---

# 6. 人情卡片模块

（待实现）
