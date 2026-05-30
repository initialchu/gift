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

## 5.1 核心概念

```
一个礼薄 = 一个事件（如"张三婚礼"）
    ├── 往来记录1：李四，500元，北京市，附带茶叶一盒
    ├── 往来记录2：王五，300元，上海市
    └── 往来记录3：赵六，200元

人情卡片 = 按"人名"跨所有礼薄汇总
    └── 李四：往 500 + 来 800 → 净额 +300（点击看详情）
```

## 5.2 业务规则

| 规则 | 说明 |
|------|------|
| 一个礼薄方向统一 | 一本礼薄里全是"来"或全是"往"（默认"来"） |
| 地址和附赠品 | 可选填写 |
| 权限 | 礼薄和记录的增删改全部由管理员操作，普通用户只能查看 |
| 关联 | 一本礼薄（GiftBook）下有多条记录（GiftRecord），`GiftBookID` 外键关联 |

## 5.3 数据模型（`models/gift.go`）

```go
// GiftBook 礼薄
type GiftBook struct {
    gorm.Model
    EventName string       `gorm:"not null" json:"event_name"`
    EventDate time.Time    `json:"event_date"`
    Direction string       `gorm:"type:varchar(4);default:来;not null" json:"direction"`
    CreatedBy string       `gorm:"not null" json:"created_by"`
    Records   []GiftRecord `gorm:"foreignKey:GiftBookID" json:"records,omitempty"`
}

// GiftRecord 礼薄中的单条往来记录
type GiftRecord struct {
    gorm.Model
    GiftBookID uint    `gorm:"not null;index" json:"gift_book_id"`
    PersonName string  `gorm:"type:varchar(64);not null" json:"person_name"`
    Amount     float64 `gorm:"type:decimal(10,2);not null" json:"amount"`
    Address    string  `gorm:"type:varchar(255)" json:"address,omitempty"`
    GiftNote   string  `gorm:"type:varchar(255)" json:"gift_note,omitempty"`
}
```

关系：`GiftBook` 1:N `GiftRecord`（通过 `GiftBookID` 外键）。

## 5.4 API 设计

| 方法 | 路径 | 权限 | 说明 |
|------|------|------|------|
| `GET` | `/api/gift-books` | 登录用户 | 礼薄列表 |
| `GET` | `/api/gift-books/:id` | 登录用户 | 礼薄详情（含所有记录） |
| `POST` | `/api/admin/gift-books` | 管理员 | 创建礼薄 |
| `PUT` | `/api/admin/gift-books/:id` | 管理员 | 修改礼薄 |
| `DELETE` | `/api/admin/gift-books/:id` | 管理员 | 删除礼薄（级联删除记录） |
| `POST` | `/api/admin/gift-books/:id/records` | 管理员 | 添加记录 |
| `PUT` | `/api/admin/gift-books/:id/records/:rid` | 管理员 | 修改记录 |
| `DELETE` | `/api/admin/gift-books/:id/records/:rid` | 管理员 | 删除记录 |

- 列表只返回礼薄信息 + 记录数（不查详情，性能更好）
- 详情才返回所有记录

## 5.5 需要修改的文件

| 文件 | 动作 | 内容 |
|------|------|------|
| `models/gift.go` | 新建 | GiftBook + GiftRecord 模型 |
| `controllers/gift.go` | 新建 | 8 个 CRUD 处理函数 |
| `router/router.go` | 修改 | 挂接礼薄路由 |
| `config/db.go` | 修改 | `AutoMigrate` 追加两个模型 |

---

# 6. 人情卡片模块

（待实现）

---

# 7. 开发进度

## 2026-05-28（已完成）

| 模块 | 内容 | 文件 |
|------|------|------|
| 配置系统 | Viper 加载 yml 配置 | `config/config.go` |
| 数据库 | GORM 连接 MySQL + 连接池 + AutoMigrate | `config/db.go` |
| 全局变量 | 导出 DB 实例 | `global/global.go` |
| User 模型 | 字段：Username, Password, Role（默认 user） | `models/user.go` |
| LoginRequest DTO | 接收登录请求的独立结构体 | `models/user.go` |
| 密码工具 | bcrypt 加密 + 验证 | `utils/utils.go` |
| JWT 工具 | 生成 token（HS256, 24h） + 解析验证 | `utils/utils.go` |
| 登录接口 | 绑定 → 查库 → 验密 → 返回 JWT | `controllers/auth.go` |
| 创建用户接口 | 绑定 → 查重 → 哈希 → 入库 | `controllers/user.go` |
| JWT 中间件 | 解析 token，提取 username 和 role 到 Context | `middlewares/auth.go` |

## 2026-05-29（今日进度）

### 上午：JWT 鉴权链路修复 ✅

| # | 文件 | 修改内容 |
|---|------|----------|
| 1 | `middlewares/auth.go` | 添加 `c.Next()` |
| 2 | `middlewares/auth.go` | 新增 `AdminMiddleware`，检查 role 是否为 admin |
| 3 | `utils/utils.go` | `GenerateJWT` 增加 role 参数；`ParseJWT` 返回 `(username, role, error)` |
| 4 | `controllers/auth.go` | `Login` 调用 `GenerateJWT` 时传入 `user.Role` |
| 5 | `config/config.go` | 扩展 Config 结构体；`MergeInConfig` 加载 config.local.yml；注入 JWT 配置 |
| 6 | `router/router.go` | `CreateUser` 移到 `/api/admin/create`，挂认证+授权中间件 |
| 7 | `main.go` | `fmt.Println` 移到 `r.Run` 之前；调用 `CreateAdmin()` |
| 8 | `config/db.go` | 实现 `CreateAdmin()` 种子管理员；`AutoMigrate` 提取为 `Autotable()` |
| 9 | `utils/utils.go` | 消除 JWT 硬编码，改为 `SetJWTConfig` 注入 |
| 10 | `controllers/user.go` | 创建成功 `200→201`，用户名已存在 `400→409` |

### 下午：礼薄模块开发

| # | 文件 | 内容 |
|---|------|------|
| 1 | `models/gift.go` | 创建 GiftBook + GiftRecord 模型 |
| 2 | `controllers/giftbook.go` | CreateGiftBook、GetgiftBooks、GetgiftBookbyID、DeleteGiftBook、UpdateGiftBook |
| 3 | `controllers/giftrecord.go` | AddGift（创建礼金记录，含礼薄存在性校验） |
| 4 | `config/db.go` | `Autotable()` 追加 GiftBook、GiftRecord |
| 5 | `README.md` | 更新项目结构、API 文档、配置说明 |
| 6 | Git commit | `e6ba367 feat: 完成礼薄模块基础 CRUD` |

### 礼薄控制器已实现函数

| 函数 | 文件 | 关键逻辑 |
|------|------|----------|
| `CreateGiftBook` | giftbook.go | 同名检测 + 自动填充创建者 |
| `GetgiftBooks` | giftbook.go | 查询全部礼薄 |
| `GetgiftBookbyID` | giftbook.go | Preload Records 返回详情 |
| `DeleteGiftBook` | giftbook.go | 先删记录再删礼薄（级联） |
| `UpdateGiftBook` | giftbook.go | 用 map 限定允许字段防越权 |
| `AddGift` | giftrecord.go | 校验礼薄存在后才创建记录 |

---

# 8. 创建管理员账户

创建管理员有三种方式，推荐使用种子数据（方式 2）。

## 方式 1：手动操作数据库

在 MySQL 客户端中直接 INSERT：

```sql
INSERT INTO users (username, password, role, created_at, updated_at)
VALUES ('admin', '<bcrypt哈希>', 'admin', NOW(), NOW());
```

需要先生成 bcrypt 哈希，不推荐。

## 方式 2：种子数据（Seeding）——推荐 ✅

在 `main.go` 启动时检查是否存在管理员，没有则自动创建：

```go
func seedAdmin() {
    var count int64
    global.DB.Model(&models.User{}).Where("role = ?", "admin").Count(&count)
    if count == 0 {
        hashedPwd, _ := utils.HashPassword("admin123")
        global.DB.Create(&models.User{
            Username: "admin",
            Password: hashedPwd,
            Role:     "admin",
        })
        fmt.Println("已创建默认管理员账号")
    }
}
```

在 `main.go` 中 `config.InitConfig()` 之后、`r.Run()` 之前调用。

**优点**：项目启动即有管理员；清库重建时自动生成；不依赖外部工具。

## 方式 3：改造 CreateUser 接口支持传 role

新增 `CreateUserRequest` DTO，增加 role 字段。管理员接口已受 `AdminMiddleware` 保护，只有管理员能创建用户并指定角色。属于后续功能增强。

---

# 9. 敏感信息管理

## 核心原则：敏感信息分层

```
config.yml          → 提交到 git，存模板/默认值（不含密码）
config.local.yml    → 不提交（.gitignore 已忽略 *.local.yml），存真实密码
```

Viper 的 `MergeInConfig` 支持合并多个配置文件，后读的覆盖先读的。

## 信息分类

| 信息 | 放哪里 | 原因 |
|------|--------|------|
| 应用名、端口 | `config.yml` | 不敏感 |
| 数据库连接池配置 | `config.yml` | 不敏感 |
| 数据库 DSN（含密码） | `config.local.yml` | 🔒 含密码 |
| JWT 密钥 | `config.local.yml` | 🔒 密钥泄露 = 任意伪造 token |
| 管理员初始密码 | `config.local.yml` | 🔒 初始凭据 |
| JWT 过期时间 | `config.yml` | 不敏感 |

## 实现步骤

### 1. 修改 `config.go`，让 Viper 读取 `config.local.yml`

```go
func InitConfig() {
    viper.SetConfigName("config")
    viper.SetConfigType("yml")
    viper.AddConfigPath("./config")
    if err := viper.ReadInConfig(); err != nil {
        log.Fatalf("读取配置文件失败: %v", err)
    }

    // 合并本地配置（可选，不存在也不报错）
    viper.SetConfigName("config.local")
    if err := viper.MergeInConfig(); err != nil {
        if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
            log.Printf("警告: 加载 config.local.yml 失败: %v", err)
        }
    }

    AppConfig = &Config{}
    if err := viper.Unmarshal(AppConfig); err != nil {
        log.Fatalf("映射配置文件失败: %v", err)
    }
    initDB()
}
```

`MergeInConfig` 是叠加合并，`config.local.yml` 里的值覆盖 `config.yml` 同名字段。

### 2. 瘦身 `config.yml`，新建 `config.local.yml`

**config.yml**（提交到 git，只有模板）：

```yaml
app:
  name: GiftMemo
  port: :8080

database:
  dsn: "请复制到 config.local.yml 修改"
  max_idle_conns: 10
  max_open_conns: 100
```

**config.local.yml**（不提交，被 .gitignore 保护）：

```yaml
database:
  dsn: "root:真实密码@tcp(127.0.0.1:3307)/giftmemo?charset=utf8mb4&parseTime=True&loc=Local"

admin:
  default_username: admin
  default_password: 你的管理员密码

jwt:
  secret: 你的随机密钥字符串
```

### 3. JWT 密钥从配置读取

`utils/utils.go` 中当前 `"wushijiazu"` 是硬编码的，应改为从 Viper 读取：

```go
secret := viper.GetString("jwt.secret")
```

### 4. 种子管理员从配置读取凭据

```go
username := viper.GetString("admin.default_username")
password := viper.GetString("admin.default_password")
```

## 额外安全建议

- `.gitignore` 已配置 `*.local.yml` 和 `.env`，确保不会被提交 ✅
- 团队协作：提供 `config.local.yml.example`（模板，不含真实密码），每人自行改名填写
- 生产环境：密码应走环境变量（Docker `--env`、K8s Secret），学习项目用 `.local.yml` 足够
- `config.yml` 中的 `dsn` 当前也含有明文密码 `root:root`，同样应迁移到 `config.local.yml`

---

# 10. Gin 中间件顺序

中间件的执行顺序是「先 Use 先执行」：

```go
// ✅ 正确顺序
admin.Use(middlewares.AuthMiddleware())    // ① 先认证 → c.Set("username") c.Set("role")
admin.Use(middlewares.AdminMiddleware())   // ② 后鉴权 → c.Get("role") 判断权限

// ❌ 错误顺序
admin.Use(middlewares.AdminMiddleware())   // 此时角色还没注入，永远返回 403
admin.Use(middlewares.AuthMiddleware())
```

**本质**：认证（Authentication，确认你是谁）必须在授权（Authorization，判断你能做什么）之前。

---

## 2026-05-29 晚：礼金记录函数 Code Review

对 `controllers/giftrecord.go` 中新增的三个函数进行审查。

### GetGiftRecordsbyID — 基本正确 ✅

```go
func GetGiftRecordsbyID(c *gin.Context) {
    giftBookID := c.Param("giftbook_id")
    var giftRecords []models.GiftRecord
    if err := global.DB.Where("gift_book_id = ?", giftBookID).Find(&giftRecords).Error; err != nil {
        ...
    }
}
```

逻辑清晰，查某个礼薄下的所有记录。

**建议**：加礼薄存在性校验（和 `AddGift` 保持一致），否则查不存在的礼薄返回空数组 `[]`，调用方无法区分「礼薄不存在」还是「礼薄没有记录」。

### DeleteGift — 两个隐患 ⚠️

```go
func DeleteGift(c *gin.Context) {
    giftID := c.Param("gift_id")
    if err := global.DB.Delete(&models.GiftRecord{}, giftID).Error; err != nil {
        ...
    }
}
```

**问题一：不验证记录是否存在**

GORM 的 `Delete` 即使找不到记录也不会报错（影响行数 0），永远返回「删除成功」，实际可能什么都没删。参照 `DeleteGiftBook` 的写法（先 `First` 确认存在再删），应该加存在性检查。

**问题二：没有校验记录属于哪个礼薄**

路由设计是 `/api/admin/gift-books/:id/records/:rid`，删除时应确认该记录确实属于 URL 中指定的礼薄，否则用户可以跨礼薄删除任意记录。

### UpdateGift — 隐患最大 ⚠️⚠️

```go
func UpdateGift(c *gin.Context) {
    giftID := c.Param("gift_id")
    var giftRecord models.GiftRecord
    if err := c.ShouldBindJSON(&giftRecord); err != nil { ... }
    if err := global.DB.Model(&models.GiftRecord{}).Where("id = ?", giftID).Updates(giftRecord).Error; err != nil { ... }
}
```

**问题一：用 struct 做 Updates 是不安全的**

传入整个 `giftRecord` struct 给 `Updates`，如果请求体里传了 `gift_book_id`，就能把记录转移到另一个礼薄下（越权）。而且 `ID`、`CreatedAt` 等 GORM 自动字段也可能被覆盖。

应参照 `UpdateGiftBook` 的写法，用 `map[string]interface{}` 白名单限定允许更新的字段：

```go
update := map[string]interface{}{
    "person_name": giftRecord.PersonName,
    "amount":      giftRecord.Amount,
    "address":     giftRecord.Address,
    "gift_note":   giftRecord.GiftNote,
}
```

**问题二：不验证记录是否存在**（同上）

**问题三：不验证记录属于哪个礼薄**（同上）

### 总结对比表

| 函数 | 存在性校验 | 礼薄归属校验 | 字段白名单 |
|------|:---:|:---:|:---:|
| `GetGiftRecordsbyID` | 建议加 | — | ✅ |
| `DeleteGift` | ❌ 缺失 | ❌ 缺失 | ✅ |
| `UpdateGift` | ❌ 缺失 | ❌ 缺失 | ❌ struct 不安全 |

**核心原则**：URL 里的参数（礼薄 ID、记录 ID）都要做归属校验，防止越权操作。

---

---

## 2026-05-29 深夜：修复完成 — 礼金记录 CRUD + 路由挂接

### 修复内容

| # | 文件 | 修复项 | 说明 |
|---|------|--------|------|
| 1 | `giftrecord.go` AddGift | URL 参数校验 | 比对 body 的 GiftBookID 和 URL 的 `:id`，防止跨礼薄写入 |
| 2 | `giftrecord.go` GetGiftRecordsbyID | 礼薄存在性校验 | 先查礼薄是否存在，区分「礼薄不存在」和「无记录」 |
| 3 | `giftrecord.go` DeleteGift | 存在性 + 归属校验 | 先 First 确认存在，再比对 GiftBookID 与 URL `:id` |
| 4 | `giftrecord.go` UpdateGift | 存在性 + 归属校验 + 字段白名单 | 用 map 限定可更新字段，防止越权修改 gift_book_id |
| 5 | `router/router.go` | 挂接全部路由 | 礼薄 CRUD + 礼金记录 CRUD，路由含双参数支持归属校验 |

### `strconv.ParseUint` 类型转换要点

`c.Param()` 返回 `string`，不能直接 `uint(xxx)` 强转：

```go
import "strconv"

gbID, err := strconv.ParseUint(c.Param("id"), 10, 64)  // string → uint64
if err != nil {
    // ID 格式错误
}
// 比较时：uint(gbID) 转为 uint
if record.GiftBookID != uint(gbID) { ... }
```

### 最终路由结构

```
admin 组（JWT + AdminMiddleware）：

  POST   /giftbook                        → CreateGiftBook
  GET    /giftbooks                       → GetgiftBooks
  GET    /giftbook/:id                    → GetgiftBookbyID
  POST   /giftbook/edit/:id               → UpdateGiftBook
  POST   /giftbook/:id                    → DeleteGiftBook

  POST   /giftrecord/:id/records          → AddGift
  GET    /giftrecord/:id/records          → GetGiftRecordsbyID
  POST   /giftrecord/:id/records/:rid      → DeleteGift
  POST   /giftrecord/:id/records/:rid/edit → UpdateGift
```

设计选择：修改/删除操作统一用 POST（非 RESTful 约定），参数通过 URL 路径传递，归属校验用 `c.Param` 获取。

### 安全校验清单（最终版）

| 函数 | 存在性校验 | 礼薄归属校验 | 字段白名单 |
|------|:---:|:---:|:---:|
| `AddGift` | ✅ 礼薄存在 | ✅ URL:ID vs Body | — |
| `GetGiftRecordsbyID` | ✅ 礼薄存在 | — | — |
| `DeleteGift` | ✅ 记录存在 | ✅ GiftBookID vs URL | — |
| `UpdateGift` | ✅ 记录存在 | ✅ GiftBookID vs URL | ✅ map 限定 |

### 当前路由和控制器参数对应关系

| 路由模式 | `c.Param` 取值 | 用途 |
|----------|---------------|------|
| `:id` | 礼薄 ID | 定位礼薄 |
| `:rid` | 记录 ID | 定位记录 |

所有函数中 `c.Param("id")` = 礼薄 ID，`c.Param("rid")` = 记录 ID，命名统一。

---

### 下一步

1. 公开路由挂接：`api` 组加 `GET /gift-books` 和 `GET /gift-books/:id`（普通用户查看）
2. 启动项目测试完整的礼薄 + 记录 CRUD 流程
3. 开始人情卡片模块设计
