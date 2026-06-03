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

---

# 11. 前端开发计划

## 11.1 当前前端状态

前端使用 Vue 3 + Vite + TypeScript 脚手架搭建，当前状态：

- ✅ `main.ts` — 正确挂载 Vue、Pinia、Router
- ✅ `vite.config.ts` — Vue 插件 + `@` 别名已配置
- ✅ `package.json` — 依赖齐全（Vue 3.5、Router 5、Pinia 3）
- 🟡 `router/index.ts` — routes 数组为空，无任何页面路由
- 🟡 `App.vue` — 脚手架模板，无实际内容
- 🟡 `stores/counter.ts` — 脚手架示例 store，需替换
- ❌ 无 API 客户端（axios）
- ❌ 无 TypeScript 类型定义
- ❌ 无 auth store（登录态管理）
- ❌ 无任何业务页面

## 11.2 建议开发顺序

采取「自底向上」的策略，逐步堆叠：

```
阶段 1: 基础设施（2-3 个文件）
  ├── 安装 axios
  ├── 创建 src/api/client.ts（axios 实例 + 拦截器）
  ├── 创建 src/types/index.ts（TS 接口定义）
  └── 创建 src/stores/auth.ts（登录态 Pinia store）

阶段 2: 布局与路由（3-4 个文件）
  ├── 创建 src/layouts/DefaultLayout.vue（导航栏 + 内容区）
  ├── 完善 src/router/index.ts（路由表 + 守卫）
  └── 重写 src/App.vue（<RouterView />）

阶段 3: 页面开发（4 个页面组件）
  ├── src/views/LoginView.vue
  ├── src/views/GiftBookListView.vue
  ├── src/views/GiftBookDetailView.vue
  └── src/views/admin/AdminDashboard.vue（可选，后续）

阶段 4: 对接收尾
  ├── 前后端联调
  └── 错误处理、loading 状态优化
```

## 11.3 阶段一：基础设施详解

### axios 客户端 (`src/api/client.ts`)

核心职责：创建统一的 axios 实例，自动处理 baseURL、token 注入、401 拦截。

设计要点：
- `baseURL: 'http://localhost:8080'`（从环境变量读取更好，学习阶段写死也行）
- **请求拦截器**：从 Pinia auth store 获取 token，加到 `Authorization: Bearer xxx` 请求头
- **响应拦截器**：遇到 401 → 清除本地 token → 跳转登录页
- 导出这个实例，所有 API 调用都通过它

### TypeScript 类型 (`src/types/index.ts`)

把后端模型一对一映射为 TS 接口：

```ts
// 用户
interface User { id: number; username: string; role: string }
// 登录请求/响应
interface LoginRequest { username: string; password: string }
interface LoginResponse { token: string }
// 礼薄
interface GiftBook { id: number; event_name: string; event_date: string; direction: string; created_by: string; records?: GiftRecord[] }
// 礼金记录
interface GiftRecord { id: number; gift_book_id: number; person_name: string; amount: number; address?: string; gift_note?: string }
// 通用 API 响应
interface ApiResponse<T> { data: T; message?: string }
```

### Auth Store (`src/stores/auth.ts`)

Pinia store，管理登录状态。核心逻辑：

- **state**：`token: string | null`、`user: User | null`
- **getters**：`isLoggedIn`（判断 token 是否存在）、`isAdmin`（判断 user.role === 'admin'）
- **actions**：
  - `login(username, password)` → 调登录 API → 存 token 到 state + localStorage → 存 user
  - `logout()` → 清除 token 和 user
  - `checkAuth()` → 页面刷新时从 localStorage 恢复 token，调 API 验证有效性

**token 持久化关键点**：token 同时存 Pinia state（内存）和 localStorage（持久化），刷新页面时从 localStorage 恢复。security 注意：生产环境应用 httpOnly cookie 或更安全的方案。

## 11.4 阶段二：路由与布局

### 路由表设计

```
/                    → 重定向到 /giftbooks
/login               → LoginView（无需登录）
/giftbooks           → GiftBookListView（需登录）
/giftbooks/:id       → GiftBookDetailView（需登录）
/admin               → AdminDashboard（需登录 + 管理员）
```

### 路由守卫

利用 Vue Router 的 `beforeEach` 守卫：

1. 从 Pinia auth store 读取 `isLoggedIn`
2. 目标路由需要登录 (`meta.requiresAuth`) 且未登录 → 重定向到 `/login`
3. 目标路由需要管理员 (`meta.requiresAdmin`) 且非管理员 → 重定向到首页
4. 已登录用户访问 `/login` → 重定向到首页

### DefaultLayout.vue

app 外壳，包含：
- 顶部导航栏（logo、礼薄列表链接、管理员入口（仅 admin 可见）、用户名/退出）
- `<RouterView />` 插槽 — 页面内容渲染区

## 11.5 阶段三：页面设计

### 登录页 (LoginView)

- 居中卡片式表单
- 字段：用户名、密码
- 登录按钮 + loading 状态 + 错误提示
- 成功后跳转到 `/giftbooks`

### 礼薄列表页 (GiftBookListView)

- 调用 `GET /api/giftbooks`
- 展示为卡片网格或表格
- 每张卡片显示：事件名、日期、方向（来/往）、记录数
- 点击卡片跳转详情页 `/giftbooks/:id`
- 管理员可见"新建礼薄"按钮

### 礼薄详情页 (GiftBookDetailView)

- 调用 `GET /api/giftbook/:id`（含 Preload records）
- 顶部：礼薄信息（事件名、日期、方向） + 编辑/删除按钮（管理员可见）
- 下方：记录表格（人名、金额、地址、赠品）
- 管理员可添加/编辑/删除单条记录
- "返回列表"链接

### 管理后台 (AdminDashboard)

- 可以简化为一个页面，通过 Tab 切换管理礼薄和管理用户
- 或者直接在列表页和详情页通过权限控制显示管理按钮

## 11.6 前端项目建议目录结构

```
client/src/
├── api/
│   ├── client.ts        # axios 实例 + 拦截器
│   ├── auth.ts          # 登录/用户相关 API 调用
│   └── giftbook.ts      # 礼薄/记录相关 API 调用
├── types/
│   └── index.ts         # 所有 TS 接口/类型
├── stores/
│   └── auth.ts          # 登录态 Pinia store
├── layouts/
│   └── DefaultLayout.vue
├── views/
│   ├── LoginView.vue
│   ├── GiftBookListView.vue
│   ├── GiftBookDetailView.vue
│   └── admin/
│       └── AdminDashboard.vue
├── components/          # 可复用组件
│   ├── GiftBookCard.vue
│   └── GiftRecordRow.vue
├── router/
│   └── index.ts
├── App.vue
└── main.ts
```

## 11.7 关键技术决策

| 决策点 | 建议 | 原因 |
|--------|------|------|
| UI 组件库 | 先不引入，手写 CSS | 学习阶段，理解组件本质比用库更重要 |
| HTTP 客户端 | axios | 拦截器机制方便 token 注入，社区标准 |
| 状态管理 | Pinia | 已安装，Vue 3 官方推荐 |
| Token 存储 | localStorage | 简单够用，学习阶段不引入 httpOnly cookie 的复杂度 |
| 路由模式 | History | 已配置 `createWebHistory`，URL 干净 |
| 类型安全 | 全 TS | 项目已是 TS 脚手架，充分利用类型系统 |

## 11.8 更新：Element Plus 已引入

用户已安装 Element Plus 并在 `main.ts` 中全局注册，11.7 的 UI 组件库决策更新为使用 Element Plus。

---

# 12. 路由与登录守卫

## 12.1 核心概念：Tabs ≠ Router

`el-tabs` 是 UI 组件，Vue Router 是路由系统，职责不同：

| 维度 | el-tabs（UI 组件） | Vue Router（路由系统） |
|------|-------------------|----------------------|
| 切换方式 | 切换面板内容 | 切换 URL + 渲染页面组件 |
| URL 变化 | 不变 | 变化（/giftbooks、/cards） |
| 浏览器前进/后退 | 不支持 | 支持 |
| 分享链接 | 无法分享 | 每个页面有独立 URL |
| 适用场景 | 详情页内切换子面板 | 全局页面导航 |

它们不互斥 —— 点击 tab 触发路由跳转，`<RouterView />` 渲染目标页面。

## 12.2 推荐的组件层级

```
App.vue                  ← 最外层壳，只有 <RouterView />
  ├── LoginView.vue      ← 登录页（不需要导航栏，独立渲染）
  │
  └── DefaultLayout.vue  ← 带导航栏的布局壳
        ├── 导航栏（el-tabs 或 el-menu）
        └── <RouterView />  ← 子路由页面渲染在这里
              ├── HomeView.vue
              ├── CardsView.vue
              ├── GiftBookListView.vue
              ├── GiftBookDetailView.vue
              └── ...
```

**为什么这样拆？** 
- 登录页不需要导航栏，直接由 `App.vue` 的顶级 `<RouterView />` 渲染
- 需要导航栏的页面共用 `DefaultLayout`，在 `children` 里定义
- 每个页面是独立组件，职责单一

## 12.3 路由表设计

```ts
routes: [
  // ① 不需要导航栏的 —— 顶级路由
  {
    path: '/login',
    name: 'login',
    component: () => import('@/views/LoginView.vue'),
    meta: { requiresAuth: false },
  },

  // ② 需要导航栏的 —— 嵌套在 DefaultLayout 下
  {
    path: '/',
    component: () => import('@/layouts/DefaultLayout.vue'),
    redirect: '/home',
    children: [
      {
        path: 'home',
        name: 'home',
        component: () => import('@/views/HomeView.vue'),
        meta: { title: '首页', requiresAuth: true },
      },
      {
        path: 'cards',
        name: 'cards',
        component: () => import('@/views/CardsView.vue'),
        meta: { title: '人情卡片', requiresAuth: true },
      },
      {
        path: 'giftbooks',
        name: 'giftbooks',
        component: () => import('@/views/GiftBookListView.vue'),
        meta: { title: '礼薄', requiresAuth: true },
      },
      {
        path: 'giftbooks/:id',
        name: 'giftbook-detail',
        component: () => import('@/views/GiftBookDetailView.vue'),
        meta: { title: '礼薄详情', requiresAuth: true, hidden: true },
      },
    ],
  },

  // ③ 404 兜底
  {
    path: '/:pathMatch(.*)*',
    redirect: '/home',
  },
]
```

### 设计要点

| 要点 | 说明 |
|------|------|
| `children` | `DefaultLayout` 里放 `<RouterView />`，子路由页面渲染在那个位置 |
| `meta` | 存附加信息：标题、是否需要登录（`requiresAuth`）、是否在导航中隐藏（`hidden`） |
| `() => import(...)` | 懒加载：访问时才加载 JS，首屏更快 |
| `redirect` | 访问 `/` 自动跳 `/home` |

## 12.4 Tabs 和 Router 联动

`DefaultLayout.vue` 里把 `el-tabs` 和 router 绑定：

**核心逻辑：**
```ts
const tabs = [
  { name: 'home',      label: '首页',     path: '/home' },
  { name: 'cards',     label: '人情卡片', path: '/cards' },
  { name: 'giftbooks', label: '礼薄',     path: '/giftbooks' },
]

const activeTab = ref('home')

// 点击 tab → 路由跳转
const handleClick = (tab) => {
  const target = tabs.find(t => t.name === tab.paneName)
  if (target) router.push(target.path)
}

// 路由变化 → 同步 tab 高亮（处理浏览器前进/后退、直接输入 URL）
watch(() => route.path, (path) => {
  const tab = tabs.find(t => path.startsWith(t.path))
  if (tab) activeTab.value = tab.name
}, { immediate: true })
```

**为什么需要双向绑定？** 用户可能通过浏览器前进/后退改变 URL，也可能直接输入地址，`watch` 保证 tab 高亮始终和当前路由一致。

## 12.5 路由守卫：未登录跳转登录页

### 流程图

```
用户访问任何页面
      │
      ▼
┌──────────────┐
│ beforeEach   │  ← 每次导航前触发
│ 路由守卫     │
└──────┬───────┘
       │
       ├── 目标页面不需要登录？（path 在白名单里）
       │      → 直接放行 ✅
       │
       ├── 目标页面需要登录 + 已登录（有 token）？
       │      → 放行 ✅
       │
       ├── 目标页面需要登录 + 未登录（无 token）？
       │      → 跳转 /login ❌（同时记住目标路径，登录后跳回来）
       │
       └── 已登录 + 访问 /login？
              → 跳转首页 ❌（登录了还去登录页干嘛）
```

### 守卫代码逻辑

```ts
router.beforeEach((to, from, next) => {
  // ① 判断是否已登录 —— 看 token 有没有值
  const token = localStorage.getItem('token')  // 或从 auth store 读
  const isLoggedIn = !!token

  // ② 白名单路由 —— 不需要登录也能访问
  const whiteList = ['/login']

  if (whiteList.includes(to.path)) {
    // 已登录 + 去登录页 → 跳首页
    if (isLoggedIn) {
      next('/home')
      return
    }
    // 未登录 + 去登录页 → 放行
    next()
    return
  }

  // ③ 需要登录的页面 + 未登录 → 跳登录页（记住目标路径）
  if (!isLoggedIn) {
    next({
      path: '/login',
      query: { redirect: to.fullPath }  // 登录完跳回来
    })
    return
  }

  // ④ 已登录 → 正常放行
  next()
})
```

### 为什么用 localStorage 判断而不是 Pinia？

`router/index.ts` 在 `app.use(pinia)` 之前执行，此时 `useAuthStore()` 可能还没初始化。用 `localStorage.getItem('token')` 是最稳妥的，没有时机问题。

**数据流：**
```
登录成功 → 写 localStorage + 写 Pinia state
刷新页面 → 从 localStorage 恢复 token → Pinia 初始化时读 localStorage
路由守卫 → 读 localStorage 判断（零依赖，无时机问题）
```

## 12.6 axios 拦截器：token 过期的兜底

路由守卫只管 token **有没有**，不管 token **是否过期**。过期的兜底交给 axios 响应拦截器：

```
调用 API → 后端返回 401 → axios 拦截器捕获
  ├── 清空 localStorage token
  ├── 清空 Pinia user state
  └── 跳转 /login
```

这样形成双层防线：
- **路由守卫**：没 token → 连页面都不让进
- **axios 拦截器**：有 token 但过期了 → API 调不通 → 踢回登录页

## 12.7 常见坑

| 坑 | 原因 | 方案 |
|----|------|------|
| 守卫里 `useAuthStore()` 报错 | pinia 还没挂载到 app | 用 `localStorage.getItem('token')` 判断，不依赖 store |
| 刷新后 token 还在但登录态丢了 | token 只存了 Pinia state，没持久化 | login 时同时写 localStorage |
| 已过期 token 不被拦截 | 路由守卫只看 token 存不存在 | axios 拦截器兜底 401 |
| 登录后跳不回之前的页面 | 跳登录时没记录来源 | `query: { redirect: to.fullPath }` 传递 |
| Tabs 高亮和当前页面不一致 | 只绑了点击事件，没监听 URL 变化 | `watch(route.path)` 双向同步 |

## 12.8 关于 el-tabs vs el-menu

`el-tabs` 适合做**内容面板切换**（如详情页里切换"基本信息"和"记录列表"），做全局导航不太自然。

Element Plus 的正统导航组件是 **`el-menu`**：
- `mode="horizontal"` → 顶部导航
- `mode="vertical"` → 侧边栏导航
- 自带 `router` 属性，不用手动写点击联动

当前学习阶段用 `el-tabs` 理解联动原理没问题，后期可以考虑迁移到 `el-menu`。

---

# 13. axios 客户端详解

## 13.1 文件位置

`client/src/axios.ts`（用户已创建）

## 13.2 逐行解析

```ts
import axios from 'axios'
```
引入 axios 库。

```ts
const instance = axios.create({
    baseURL: 'http://localhost:8080/api',
})
```

创建 **axios 实例**（可以理解为"配置好的 axios 副本"）。

**`baseURL` 的作用：** 之后发请求只需要写路径片段，axios 自动拼接前缀：

```ts
// 写：
instance.get('/giftbooks')

// axios 实际请求：
// http://localhost:8080/api/giftbooks
```

**为什么用实例而不是直接用 `axios.get()`？** baseURL 不用每次都写，改端口也只改一处。所有 API 调用共享同一个配置。

```ts
instance.interceptors.request.use(config => {
    const token = localStorage.getItem('token')
    if (token) {
        config.headers.Authorization = `Bearer ${token}`
    }
    return config
})
```

**请求拦截器（Request Interceptor）**——每次请求发出前自动执行。

执行流程：
```
调用 instance.get('/giftbooks')
        │
        ▼
┌──────────────────────────────┐
│ 拦截器执行：                  │
│ 1. 从 localStorage 拿 token   │
│ 2. 如果有 token → 塞到请求头  │
│     Authorization: Bearer xxx │
│ 3. return config（放行请求）   │
└──────────────┬───────────────┘
        ▼
   真正的 HTTP 请求发出
```

**好处：** 每次发请求都不用手动写 `Authorization` 头，拦截器自动帮你加。

## 13.3 当前缺失

| 缺失 | 说明 |
|------|------|
| **`export`** | 没 `export default instance`，别的文件无法 `import` 它 |
| **响应拦截器** | 没有处理 401（token 过期），过期 token 不会被踢掉 |

## 13.4 响应拦截器详解

### 它做什么？

响应拦截器在**每个 HTTP 响应回来后**自动执行，在业务代码拿到数据之前先拦截处理：

```
后端响应
    │
    ▼
┌──────────────────────────────┐
│ 响应拦截器执行：               │
│ 1. 检查响应状态码              │
│ 2. 如果是 401 → 踢到登录页     │
│ 3. 如果是 2xx → 正常返回数据    │
│ 4. 其他错误 → 统一提示          │
└──────────────┬───────────────┘
    │
    ▼
业务代码拿到结果
```

### 为什么需要它？

路由守卫只管 token **有没有**，不管 token **是否过期**。如果 token 已过期（24 小时后）还留在 localStorage，路由守卫会误以为已登录。响应拦截器是第二层防线：

```
双层防线：
  ┌─ 路由守卫（进入页面前）   → 没 token → 拦下，跳登录
  └─ 响应拦截器（调 API 之后） → token 过期 → 清除后跳登录
```

### 代码实现

```ts
// 响应拦截器 —— 放在 instance 创建和请求拦截器之后
instance.interceptors.response.use(
  // 第一个参数：响应成功（2xx）时走这里
  (response) => {
    return response
  },

  // 第二个参数：响应失败（非 2xx）时走这里
  (error) => {
    if (error.response) {
      const status = error.response.status

      if (status === 401) {
        // token 无效或过期 → 清理本地数据 + 跳登录页
        localStorage.removeItem('token')
        localStorage.removeItem('user')
        // 跳登录页（用 window.location 而不是 router.push，
        // 因为这里不在 Vue 组件内，拿不到 router 实例）
        window.location.href = '/login'
      } else if (status === 403) {
        // 可选：权限不足的提示
        console.warn('权限不足')
      } else if (status >= 500) {
        // 可选：服务器错误的提示
        console.error('服务器错误')
      }
    } else {
      // 网络错误（断网、CORS 等），连 response 都没有
      console.error('网络错误')
    }

    // 继续抛出错误，让调用的业务代码也能感知
    return Promise.reject(error)
  }
)
```

### 两个参数的含义

`axios` 的响应拦截器有两个回调：

| 参数 | 触发条件 | 做什么 |
|------|---------|--------|
| 第一个 `(response)` | HTTP 状态码 2xx | 正常返回数据，通常直接 `return response` |
| 第二个 `(error)` | HTTP 状态码非 2xx 或网络错误 | 按状态码分类处理 |

### 关键细节：为什么用 `window.location.href` 而不是 `router.push`？

响应拦截器在 `axios.ts` 里，这是一个普通 JS 文件，**不在 Vue 组件内**，拿不到 Vue Router 的 `router` 实例。所以用原生的 `window.location.href` 做跳转，效果等价——都会改变 URL 并触发页面重新渲染。

如果一定要用 `router.push`，需要把 `router` 实例导入进来：
```ts
import router from '@/router'
// ...
router.push('/login')
```
但这样 `axios.ts` 就和 Vue Router 耦合了，不是必须的话不推荐。

### 完整文件结构

```ts
import axios from 'axios'

// 1. 创建实例
const instance = axios.create({
    baseURL: 'http://localhost:8080/api',
})

// 2. 请求拦截器 —— 自动加 token
instance.interceptors.request.use(config => {
    const token = localStorage.getItem('token')
    if (token) {
        config.headers.Authorization = `Bearer ${token}`
    }
    return config
})

// 3. 响应拦截器 —— 统一错误处理
instance.interceptors.response.use(
    (response) => response,
    (error) => {
        if (error.response?.status === 401) {
            localStorage.removeItem('token')
            localStorage.removeItem('user')
            window.location.href = '/login'
        }
        return Promise.reject(error)
    }
)

// 4. 导出（关键！不然别的文件 import 不了）
export default instance
```

## 13.5 拦截器执行顺序（重要）

一个请求的完整生命周期：

```
发起请求                   响应回来
──────────────→          ←─────────────
[请求拦截器] → [真实请求] → [响应拦截器] → 业务代码
    ①             ②            ③           ④

① 加 token、loading 状态
② 网络传输
③ 处理 401、统一错误提示
④ 拿到干净的 response.data
```

理解这个顺序，调试时就知道问题出在哪个环节。

---

# 14. Prettier 代码格式化配置

## 14.1 已创建的文件

| 文件 | 作用 |
|------|------|
| `client/.prettierrc` | Prettier 格式化规则 |
| `client/.prettierignore` | 忽略格式化的目录 |
| `client/.vscode/settings.json` | VSCode 保存时自动格式化 |

## 14.2 .prettierrc 配置说明

```json
{
  "semi": false,
  "singleQuote": true,
  "trailingComma": "all",
  "printWidth": 100,
  "tabWidth": 2
}
```

| 配置 | 值 | 含义 |
|------|----|------|
| `semi` | `false` | 语句结尾不加分号（Vue 社区主流风格） |
| `singleQuote` | `true` | 用单引号而非双引号 |
| `trailingComma` | `"all"` | 多行末尾加逗号，git diff 更干净 |
| `printWidth` | `100` | 每行最长 100 字符自动换行 |
| `tabWidth` | `2` | 缩进 2 空格 |

## 14.3 VSCode 自动格式化

`client/.vscode/settings.json` 新增了：

```json
"editor.formatOnSave": true,
"editor.defaultFormatter": "esbenp.prettier-vscode"
```

**前提：** 需要安装 VSCode 扩展 `esbenp.prettier-vscode`。如果没装，VSCode 右下角会提示或者去扩展商店搜索 "Prettier" 安装。

效果：每次 `Ctrl+S` 保存 → Prettier 自动格式化当前文件，不用手动操作。

## 14.4 使用方式

```bash
# 手动格式化整个 src 目录
npx prettier --write src/

# 只检查不修改（CI 用）
npx prettier --check src/
```

## 14.5 安装步骤回顾

```bash
cd client
npm install -D prettier
```

然后创建上述 3 个配置文件即可。Prettier 本身已安装完成，可以立即使用。

---

# 15. 前端开发顺序（当前阶段）

## 15.1 当前缺口分析

```
App.vue（el-menu）
  ❌ 没有 <RouterView />        → 路由匹配了页面也没地方渲染
  ❌ 菜单点击没跳转路由           → 点首页/卡片/礼薄，URL 不变

router/index.ts
  ❌ 没有 beforeEach 守卫       → 未登录也能进 /home

stores/auth.ts
  ❌ 只有 defineStore 导入      → 登录状态没管起来

components/Login.vue
  ❌ 空壳                      → 没有登录表单

axios.ts
  ✅ 已完成                     → 请求/响应拦截器就绪
```

## 15.2 开发顺序：先通登录链路，再填页面

**原则：先让数据跑通（auth → guard → router），再让 UI 好看（表单 + 页面）。** 登录是整个应用的入口，它通了，后面所有页面都能拿到登录态。

### 第一步：把线接通

```
auth.ts 写完整
  → App.vue 补 <RouterView /> + 菜单绑定路由
  → router 加 beforeEach 守卫
```

效果：未登录访问 `/home` → 自动跳到 `/login`，登录页面能渲染出来。

### 第二步：实现登录

```
Login.vue 写登录表单
  → 填用户名密码 → 调登录 API → 存 token → 跳首页
```

效果：输入账号密码 → 登录成功 → 进入首页看到导航栏 → 刷新不丢登录态。

---

# 16. Auth Store 设计

## 16.1 Pinia Store 三块结构

| 块 | 作用 | 示例 |
|----|------|------|
| `state` | 存数据 | token、user |
| `getters` | 派生计算 | isLoggedIn、isAdmin |
| `actions` | 行为方法 | login()、logout()、checkAuth() |

## 16.2 State 设计

```
token    → JWT 字符串，整个应用的"通行证"。有 token = 已登录
user     → { username, role }，当前用户信息
```

两个字段够用。token 是核心判断依据。

## 16.3 Getters 设计

| Getter | 计算方式 | 用途 |
|--------|---------|------|
| `isLoggedIn` | `!!state.token` | 路由守卫判断是否放行 |
| `isAdmin` | `state.user?.role === 'admin'` | 控制管理按钮显隐 |

getter 是派生状态，不用单独存，根据 state 实时算。

## 16.4 Actions 设计

| 方法 | 做什么 | 数据流向 |
|------|--------|----------|
| `login(username, password)` | 调登录 API → 拿 token → 存起来 | API → state + localStorage |
| `logout()` | 清空 token 和 user | state 清空 + localStorage 清空 |
| `checkAuth()` | 页面刷新后，从 localStorage 恢复登录态 | localStorage → state |

## 16.5 核心设计：token 双重存储

**为什么两个地方都要存？**

```
登录时：
  API 返回 token
    ├── 存到 Pinia state（内存）    → 当前页面立即可用，响应式
    └── 存到 localStorage（硬盘）  → 刷新页面不丢

刷新页面时：
  Pinia state 清空了（内存释放）
    └── checkAuth() 从 localStorage 读回来 → 恢复到 state
```

- 只存 state → 刷新就丢，每次都要重新登录
- 只存 localStorage → 能用，但其他组件通过 store 读不到，失去响应式
- **两个都存 → 最佳体验**

## 16.6 login 方法流程

```
login(username, password)
    │
    ▼
调用 axios.post('/auth/login', { username, password })
    │
    ├── 成功 → 拿到 { token }
    │     ├── this.token = token              // 存 Pinia
    │     ├── localStorage.setItem('token', token)  // 存硬盘
    │     ├── this.user = { username, role }        // 从 token 解码
    │     └── 返回 true（调用方知道登录成功）
    │
    └── 失败 → 返回 false（调用方显示错误提示）
```

## 16.7 logout 方法流程

```
logout()
    ├── this.token = null
    ├── this.user = null
    ├── localStorage.removeItem('token')
    └── router.push('/login')
```

## 16.8 user 信息获取方案

登录接口 `/api/auth/login` 目前只返回 `{ token }`，但 token 的 JWT payload 里已包含 username 和 role（后端 `GenerateJWT` 写入的）。

**推荐方案：前端直接从 JWT 解码**

```ts
// JWT 是三段式: header.payload.signature，base64 编码
// payload 形如: { username: "admin", role: "admin", exp: 1234567890 }

function decodeToken(token: string) {
  const payloadBase64 = token.split('.')[1]  // 取中间那段
  const payloadJson = atob(payloadBase64)     // base64 解码
  return JSON.parse(payloadJson)              // JSON 解析
  // → { username: "admin", role: "admin", exp: 1234567890 }
}
```

| 方案 | 做法 | 评价 |
|------|------|------|
| A | 后端多返回 username/role | 简单但要多改一个地方 |
| B | 前端解码 JWT payload | ⭐ 推荐，零额外请求，不改后端 |
| C | 单独 /api/user/me 接口 | 小项目杀鸡用牛刀 |

### 16.8.1 decodeToken 详解：为什么需要它？

**一句话：** 后端登录接口只返回 `{ token }`，没返回 username 和 role。但前端需要知道"谁登录了"和"是不是管理员"——这些信息就藏在 token 的 Payload 里。

#### JWT 三段式结构

JWT 长这样，三个部分用 `.` 分开：

```
Header.Payload.Signature
```

实际一段 token：

```
eyJhbGciOiJIUzI1NiJ9.eyJ1c2VybmFtZSI6ImFuZ2VsaGVhcnQiLCJyb2xlIjoiYWRtaW4iLCJleHAiOjE3NDg3MjQ4MDB9.s4X9_8kL...
│                      │                                                                  │
│  Header（算法）        │  Payload（数据载荷）                                              │  Signature（签名）
│  base64(               │  base64(                                                        │  防篡改校验
│  {"alg":"HS256"}       │  {"username":"angelheart","role":"admin","exp":1748724800"}      │
│  )                     │  )                                                              │
```

**三段 base64 JSON，不是加密，只是编码。** 所以前端可以直接解码 Payload 读出内容——不需要密钥，不需要调后端。

#### decodeToken 做了什么

```
token.split('.')[1]   →  取中间那段 base64 字符串
atob(...)             →  base64 解码为 JSON 字符串
JSON.parse(...)       →  转为 JS 对象 { username, role }
```

就是三步纯计算，零网络请求。

#### 它在 store 的两处调用

```ts
// ① login 成功后 —— 解码 token 拿到用户信息
const res = await axios.post('/auth/login', { username, password })
const payload = decodeToken(res.data.token)
if (payload) user.value = { username: payload.username, role: payload.role }

// ② 刷新页面恢复时 —— 从 localStorage 读回 token，解码恢复用户信息
function checkAuth() {
  const savedToken = localStorage.getItem('token')
  const payload = decodeToken(savedToken)
  if (payload) user.value = { username: payload.username, role: payload.role }
}
```

#### 完整实现（实际代码）

```ts
function decodeToken(tokenStr: string): { username: string; role: string } | null {
  try {
    const parts = tokenStr.split('.')
    if (!parts[1]) return null       // 防止非 JWT 格式的字符串
    const payloadJson = atob(parts[1])
    return JSON.parse(payloadJson)
  } catch {
    return null                      // 解码失败 → 当做无效 token
  }
}
```

**本质：token 就是一张"自编码的用户名片"，decodeToken 把名片展开读出姓名和身份，省得再跑一趟后台去问。**

## 16.9 Auth Store 伪代码骨架

```ts
import { defineStore } from 'pinia'
import axios from '@/axios'

export const useAuthStore = defineStore('auth', {
  state: () => ({
    token: null as string | null,
    user: null as { username: string; role: string } | null,
  }),

  getters: {
    isLoggedIn: (state) => !!state.token,
    isAdmin: (state) => state.user?.role === 'admin',
  },

  actions: {
    async login(username: string, password: string) {
      // 1. 调登录 API
      // 2. 拿到 token → 解码 user → 存 state + localStorage
      // 3. 返回成功/失败
    },

    logout() {
      // 1. 清 state
      // 2. 清 localStorage
      // 3. 跳转 /login
    },

    checkAuth() {
      // 1. 从 localStorage 读 token
      // 2. 有 → 恢复到 state
      // 3. 无 → 跳过
    },
  },
})
```

## 16.10 第一步完整清单

| 文件 | 要做什么 |
|------|----------|
| `stores/auth.ts` | 按 16.9 骨架实现 login / logout / checkAuth |
| `App.vue` | 补 `<RouterView />`；el-menu 点击时 `router.push` 跳转；el-menu 高亮同步当前路由 |
| `router/index.ts` | 加 `beforeEach` 守卫（未登录 → /login，已登录访问 /login → /home） |

第一步做完后效果：访问任意页面 → 自动跳到 `/login` → Login 组件渲染（虽然还是空壳）→ 数据链路打通。

---

# 17. 2026-05-30 前端基础设施实施

## 17.1 实施清单

| # | 文件 | 动作 | 说明 |
|---|------|------|------|
| 1 | `stores/auth.ts` | 实现 | login / logout / checkAuth + decodeToken + token 双重存储 |
| 2 | `router/index.ts` | 重写 | 路由嵌套结构 + beforeEach 守卫 |
| 3 | `views/DefaultLayout.vue` | **新建** | 导航栏 + footer + `<RouterView />`，从 App.vue 中抽出 |
| 4 | `App.vue` | 精简 | 只剩 `<RouterView />`，纯路由容器 |
| 5 | `axios.ts` | 补全 | 请求拦截器（加 token）+ 响应拦截器（401 处理）+ export |
| 6 | `.prettierrc` | **新建** | Prettier 格式化规则 |
| 7 | `.prettierignore` | **新建** | 忽略 node_modules、dist |
| 8 | `.vscode/settings.json` | 追加 | formatOnSave + defaultFormatter |
| 9 | `README.md` | 更新 | 项目结构、技术栈、开发说明 |
| 10 | `stores/counter.ts` | 删除 | 脚手架示例，不再需要 |

## 17.2 最终页面层级

```
App.vue（<RouterView />）
  ├── /login           → Login.vue（纯登录表单，无导航栏无footer）
  └── /（DefaultLayout）
        ├── /home      → Home.vue（导航栏 + 内容 + footer）
        ├── /card      → Card.vue
        └── /giftbooks → GiftBooks.vue
```

## 17.3 DefaultLayout.vue 关键逻辑

- `el-menu` 使用 `router.push({ name: key })` 做导航跳转（key 即路由 name）
- `watch(() => route.name)` 监听 URL 变化，同步菜单高亮
- 不再包含"登录"菜单项（登录页独立，不在导航里）

## 17.4 路由守卫最终逻辑

```
beforeEach(to, from, next):
  token = localStorage.getItem('token')

  ① to.meta.requiresAuth !== true
     ├── token && to.path === '/login' → next('/home')  // 已登录别再看登录页
     └── 否则 → next()                                   // 放行

  ② to.meta.requiresAuth === true && !token
     → next({ path: '/login', query: { redirect: to.fullPath } })  // 没 token → 登录

  ③ to.meta.requiresAuth === true && token
     → next()  // 有 token → 放行
```

## 17.5 Auth Store 结构

```ts
useAuthStore() {
  state:    token, user
  getters:  isLoggedIn, isAdmin
  actions:  login(), logout(), checkAuth()
  工具:     decodeToken()  // JWT payload 解码，零网络请求
}
```

**token 双重存储：** Pinia state（响应式、当前页可用）+ localStorage（刷新不丢）

## 17.6 当前缺口（下一步）

| 文件 | 状态 | 说明 |
|------|------|------|
| `axios.ts` | ✅ | 请求 + 响应拦截器 |
| `stores/auth.ts` | ✅ | login/logout/checkAuth |
| `router/index.ts` | ✅ | 嵌套路由 + 守卫 |
| `App.vue` | ✅ | 纯容器 |
| `DefaultLayout.vue` | ✅ | 导航栏布局 |
| `components/Login.vue` | ❌ | 空壳，待写登录表单 |
| `views/Home.vue` | ⚠️ | 占位，待填充 |
| `views/Card.vue` | ⚠️ | 占位，待填充 |
| `views/GiftBooks.vue` | ⚠️ | 占位，待填充 |

下一步：写 `Login.vue` 登录表单，打通登录全链路。

---

# 18. Vite 代理配置

## 18.1 为什么需要代理

当前 axios baseURL 直接指向后端：`http://localhost:8080/api`，浏览器跨域请求后端。配代理后：

```
之前：浏览器 → http://localhost:8080/api/xxx  （跨域，CORS 问题）
之后：浏览器 → http://localhost:5173/api/xxx   （同源，无跨域）
                │
                └── Vite 自动转发 → http://localhost:8080/api/xxx
```

**好处：**
- 不需要后端配 CORS
- axios baseURL 简化为 `/api`，更接近生产环境
- 开发和生产环境请求路径统一

## 18.2 vite.config.ts 配置

```ts
import { defineConfig } from 'vite'
// ...其他 import

export default defineConfig({
  plugins: [vue(), vueDevTools()],
  resolve: {
    alias: { '@': fileURLToPath(new URL('./src', import.meta.url)) },
  },
  server: {
    proxy: {
      '/api': {
        target: 'http://localhost:8080',
        changeOrigin: true,
      },
    },
  },
})
```

| 配置项 | 含义 |
|--------|------|
| `'/api'` | 匹配所有以 `/api` 开头的请求路径 |
| `target` | 转发到后端的目标地址 |
| `changeOrigin` | 修改请求头 Origin 为 target，避免后端校验 Origin 导致拒绝 |

## 18.3 配完后 axios.ts 需同步修改

```ts
// 之前（直连后端）
const instance = axios.create({
  baseURL: 'http://localhost:8080/api',
})

// 之后（走代理）
const instance = axios.create({
  baseURL: '/api',
})
```

## 18.4 请求路径对照

| 前端代码 | 浏览器发出 | Vite 转发到 |
|---------|-----------|------------|
| `axios.post('/auth/login')` | `localhost:5173/api/auth/login` | `localhost:8080/api/auth/login` |
| `axios.get('/giftbooks')` | `localhost:5173/api/giftbooks` | `localhost:8080/api/giftbooks` |

## 18.5 注意事项

- **仅开发环境生效**：`server.proxy` 只在 `npm run dev` 时工作，生产构建 `npm run build` 不包含代理
- **修改 vite.config.ts 后要重启**：Vite 不会热更新自身配置，需要 `Ctrl+C` 停掉再 `npm run dev`
- **生产环境**：部署时通常用 Nginx 做反向代理，配置逻辑类似

---

# 19. 弹窗表单模式（el-dialog + el-form）

## 19.1 场景

页面有一个按钮 → 点击后弹出带表单的对话框 → 填写 → 提交。最常用模式：`el-button` + `el-dialog` + `el-form`。

## 19.2 结构

```
页面
  ├── el-button（触发按钮）
  └── el-dialog（弹窗对话框，v-model 控制显示/隐藏）
        ├── el-form（表单内容）
        │     ├── el-form-item + el-input
        │     ├── el-form-item + el-radio-group
        │     └── el-form-item + el-date-picker
        └── <template #footer>（底部确定/取消按钮）
```

## 19.3 核心三要素

### ① 触发按钮

```html
<el-button type="primary" @click="dialogVisible = true">新建礼薄</el-button>
```

### ② 弹窗对话框

```html
<el-dialog v-model="dialogVisible" title="新建礼薄" width="500px">
  <!-- 表单内容 -->
</el-dialog>
```

`v-model="dialogVisible"` 双向绑定弹窗开关：
- `true` → 弹窗打开
- `false` → 弹窗关闭（点 X、点遮罩、按 ESC 都会自动设为 false）

### ③ 表单

```html
<el-form :model="form" label-width="80px">
  <el-form-item label="事件名称">
    <el-input v-model="form.eventName" placeholder="请输入" />
  </el-form-item>
  <el-form-item label="方向">
    <el-radio-group v-model="form.direction">
      <el-radio value="来">来</el-radio>
      <el-radio value="往">往</el-radio>
    </el-radio-group>
  </el-form-item>
  <el-form-item label="日期">
    <el-date-picker v-model="form.eventDate" type="date" />
  </el-form-item>
</el-form>
```

### 底部按钮

```html
<template #footer>
  <el-button @click="dialogVisible = false">取消</el-button>
  <el-button type="primary" @click="handleSubmit">确定</el-button>
</template>
```

`<template #footer>` 是 el-dialog 的具名插槽，放弹窗底部的操作按钮。

## 19.4 脚本逻辑

```ts
import { ref } from 'vue'

const dialogVisible = ref(false)

const form = ref({
  eventName: '',
  eventDate: '',
  direction: '来',
})

const handleSubmit = async () => {
  // 调 API 提交
  await axios.post('/admin/giftbook', form.value)
  dialogVisible.value = false
  // 成功后刷新列表
}
```

## 19.5 完整交互流程

```
点击"新建礼薄"
  → dialogVisible = true
  → 弹窗打开，表单为空
  → 用户填写
  → 点"确定"
      → handleSubmit()
      → 调 API
      → dialogVisible = false（关弹窗）
      → 刷新列表
  → 点"取消" / 按 ESC / 点遮罩
      → dialogVisible = false（自动）
      → 表单数据丢弃
```

## 19.6 常见错误：`:model` 写成 `model`

```html
<!-- ❌ 错误：传的是字符串 "form" -->
<el-form model="form">

<!-- ✅ 正确：传的是变量 form（ref 对象） -->
<el-form :model="form">
```

| 写法 | 传给组件 | 结果 |
|------|---------|------|
| `model="form"` | 字符串 `"form"` | el-form 收到 String，报类型错误 |
| `:model="form"` | ref 对象 | el-form 收到 Object，正常 |

Vue 中 `:` 是 `v-bind` 的缩写，不加 `:` 的属性值永远是字符串。任何需要传对象、数组、布尔值的 prop 都要加 `:`。

---

# 20. 2026-05-31 进度记录

## 20.1 今日完成

| # | 模块 | 内容 | 文件 |
|---|------|------|------|
| 1 | 弹窗表单 | el-dialog + el-form 新建礼薄弹窗 | GiftBooks.vue |
| 2 | 礼薄展示 | el-table 展示礼薄列表，onMounted 拉数据 | GiftBooks.vue |
| 3 | el-radio 修复 | direction 值从 "1"/"2" 改为 "来"/"往" | GiftBooks.vue |
| 4 | 笔记更新 | 弹窗表单模式（第 19 章） | step.md |

## 20.2 GiftBooks.vue 当前状态

**已完成：**
- el-dialog 新建礼薄弹窗（事件名、方向 radio、日期选择器）
- createGiftBook 函数（调 POST /admin/giftbook）
- el-table 展示礼薄列表
- onMounted 加载列表数据

**待完成（明天）：**
- 修复 el-date-picker 日期格式（后端 time.Time 需要正确格式）
- el-table 加操作列（编辑、删除按钮）
- newApi 创建礼薄成功后刷新列表
- 删除礼薄功能
- 编辑礼薄弹窗
- 点击行跳转详情页（/giftbooks/:id）

## 20.3 整体项目进度

```
后端（Go/Gin/GORM）     ████████████████████ 95%
  ├── 用户认证           ████████████████████ ✅
  ├── 礼薄 CRUD          ████████████████████ ✅
  ├── 礼金记录 CRUD      ████████████████████ ✅
  └── 优雅关闭           ████████████████████ ✅

前端（Vue 3 + Element Plus）  ████████░░░░░░░░░░ 45%
  ├── 基础设施            ████████████████████ ✅
  │   ├── axios + 代理    ████████████████████ ✅
  │   ├── auth store       ████████████████████ ✅
  │   ├── 路由 + 守卫      ████████████████████ ✅
  │   ├── App.vue 容器     ████████████████████ ✅
  │   ├── DefaultLayout    ████████████████████ ✅
  │   └── Prettier         ████████████████████ ✅
  ├── 登录页              ████████████████████ ✅
  ├── 导航栏 + 头像        ██████████░░░░░░░░░░ ⚠️ 进行中
  └── 业务页面            ████░░░░░░░░░░░░░░░░ 🚧
      ├── 礼薄列表页       ██████░░░░░░░░░░░░ ⚠️ 弹窗+表格基本完成
      ├── 礼薄详情页       ░░░░░░░░░░░░░░░░░░ ❌
      ├── 人情卡片页       ░░░░░░░░░░░░░░░░░░ ❌
      └── 首页             ░░░░░░░░░░░░░░░░░░ ❌
```

## 20.4 明日待办优先

1. 完善 GiftBooks.vue（日期格式、操作列、删除、编辑）
2. 实现 GiftBookDetailView（点击行跳转详情，展示记录列表）
3. 完成 Avatar 头像区域
4. 补充 Home.vue 首页内容

---

# 21. 2026-05-31：GiftBooks 卡片网格重构方案

## 21.1 需求变更

原计划用 `el-table` 展示礼薄列表，改为**卡片网格布局**。

**效果：** 每个礼薄一张卡片，从左往右排列，超出宽度自动换行。点击卡片跳转详情页。

## 21.2 卡片 vs 表格对比

| 维度 | el-table | 卡片网格 |
|------|----------|---------|
| 信息密度 | 高，行式排列 | 中等，每张卡片独立 |
| 视觉友好 | 偏表格，数据感强 | 偏展示，浏览感好 |
| 移动端 | 需要横向滚动 | 自动换行，适配更好 |
| 点击区域 | 一整行 | 一整张卡片 |
| 适合场景 | 多列数据对比 | 内容预览 + 点击深入 |

礼薄场景（每个礼薄是一个独立实体，不是用来逐行对比的）更适合卡片。

## 21.3 整体布局

```
┌────────────────────────────────────────────────┐
│  [+ 新建礼薄]                                    │
│                                                │
│  ┌──────────┐  ┌──────────┐  ┌──────────┐     │
│  │ 婚礼      │  │ 满月酒    │  │ 生日宴    │     │
│  │ 2026-05-31│  │ 2026-04-15│  │ 2026-03-20│     │
│  │ 来 →      │  │ 去 ←      │  │ 来 →      │     │
│  └──────────┘  └──────────┘  └──────────┘     │
│  ┌──────────┐  ┌──────────┐                   │
│  │ 升学宴    │  │ 乔迁      │                   │
│  │ ...       │  │ ...       │                   │
│  └──────────┘  └──────────┘                   │
│                                                │
└────────────────────────────────────────────────┘
```

## 21.4 模板结构

```html
<template>
  <div class="giftbooks">
    <!-- 顶部操作栏 -->
    <div class="giftbooks-header">
      <h2>礼薄</h2>
      <el-button type="primary" @click="openCreateDialog">新建礼薄</el-button>
    </div>

    <!-- 卡片网格 -->
    <div class="card-grid" v-loading="loading">
      <el-card
        v-for="book in giftBooks"
        :key="book.ID"
        class="giftbook-card"
        shadow="hover"
        @click="router.push(`/giftbooks/${book.ID}`)"
      >
        <template #header>
          <div class="card-header">
            <span class="event-name">{{ book.event_name }}</span>
            <el-tag
              :type="book.direction === '来' ? 'success' : 'warning'"
              size="small"
            >
              {{ book.direction }}
            </el-tag>
          </div>
        </template>
        <div class="card-body">
          <div class="card-date">
            <el-icon><Calendar /></el-icon>
            {{ formatDate(book.event_date) }}
          </div>
          <div class="card-creator">
            创建者：{{ book.created_by }}
          </div>
        </div>
      </el-card>

      <!-- 空状态 -->
      <el-empty v-if="!loading && giftBooks.length === 0" description="暂无礼薄" />
    </div>

    <!-- 新建/编辑弹窗（复用原有 el-dialog） -->
    <el-dialog v-model="dialogVisible" ...>
      <!-- 原有表单 -->
    </el-dialog>
  </div>
</template>
```

## 21.5 脚本逻辑

```ts
import { ref, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import axios from '../axios'
import { ElMessage } from 'element-plus'

const router = useRouter()

// --- 列表数据 ---
const giftBooks = ref([])
const loading = ref(false)

const fetchGiftBooks = async () => {
  loading.value = true
  try {
    const res = await axios.get('/giftbooks')
    giftBooks.value = res.data.giftbooks ?? []
  } catch (err: any) {
    ElMessage.error('获取礼薄列表失败')
  } finally {
    loading.value = false
  }
}

onMounted(() => {
  fetchGiftBooks()
})

// --- 日期格式化 ---
// 后端 time.Time → JSON 是 ISO 8601: "2026-05-31T00:00:00+08:00"
// 截取 T 之前的部分即可得到 YYYY-MM-DD
const formatDate = (iso: string): string => {
  if (!iso) return ''
  return iso.split('T')[0]
}

// --- 新建礼薄（成功后刷新列表） ---
const createGiftBook = async () => {
  try {
    await axios.post('/admin/giftbook', form.value)
    ElMessage.success('创建成功')
    dialogVisible.value = false
    fetchGiftBooks()  // ← 关键：创建后刷新
  } catch (err: any) {
    const msg = err.response?.data?.error || '创建失败'
    ElMessage.error(msg)
  }
}
```

## 21.6 样式设计

```css
.giftbooks {
  padding: 20px;
}

/* 顶部操作栏 */
.giftbooks-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 24px;
}

/* 卡片网格 —— 核心布局 */
.card-grid {
  display: flex;
  flex-wrap: wrap;          /* 超出宽度自动换行 */
  gap: 16px;                /* 卡片之间间距 */
}

/* 单张卡片 */
.giftbook-card {
  width: 240px;             /* 固定宽度，一排能放多个 */
  cursor: pointer;          /* 鼠标变手型，暗示可点击 */
  transition: transform 0.2s, box-shadow 0.2s;
}

.giftbook-card:hover {
  transform: translateY(-4px);   /* 悬浮上移 4px */
}

/* 卡片头部：事件名 + 方向标签 */
.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.event-name {
  font-size: 16px;
  font-weight: 600;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

/* 卡片内容 */
.card-body {
  display: flex;
  flex-direction: column;
  gap: 8px;
  color: #666;
  font-size: 14px;
}
```

## 21.7 Flexbox 换行原理

```
display: flex + flex-wrap: wrap

  一行放 5 张卡片（240px × 5 + 16px × 4 = 1264px）
  ┌────┐ ┌────┐ ┌────┐ ┌────┐ ┌────┐
  │ 1  │ │ 2  │ │ 3  │ │ 4  │ │ 5  │
  └────┘ └────┘ └────┘ └────┘ └────┘

  屏幕变窄（900px）→ 第 4 张放不下 → 自动换到下一行
  ┌────┐ ┌────┐ ┌────┐
  │ 1  │ │ 2  │ │ 3  │
  └────┘ └────┘ └────┘
  ┌────┐ ┌────┐
  │ 4  │ │ 5  │
  └────┘ └────┘
```

`gap: 16px` 同时控制横向和纵向间距，不需要手动设 margin。

## 21.8 改动清单

| # | 位置 | 改动内容 |
|---|------|---------|
| 1 | `<script>` | 新增 `giftBooks`、`loading`、`fetchGiftBooks`、`formatDate`；`onMounted` 调用 `fetchGiftBooks`；引入 `useRouter` |
| 2 | `<template>` | 去掉 `<el-header>` 和 `<el-main>`；换成 `.giftbooks-header` + `.card-grid` + `el-card` + `v-for` |
| 3 | `<style>` | 新增 `.card-grid`、`.giftbook-card`、`.card-header` 等样式 |
| 4 | `createGiftBook` | 成功后加一行 `fetchGiftBooks()` |
| 5 | 删除 | 去掉 `import GiftRecords from '../components/GiftRecords.vue'`（未使用的引用） |
| 6 | `router/index.ts` | 预留 `/giftbooks/:id` 路由（第二步做详情页时需要） |

## 21.9 设计要点总结

| 要点 | 说明 |
|------|------|
| **flex-wrap** | 核心换行机制，不用 media query 就能自适应 |
| **固定卡片宽度** | `width: 240px` 而不是百分比，保证每行卡片数量随容器宽度自然变化 |
| **hover 上移** | `translateY(-4px)` + `shadow="hover"`，视觉反馈明确 |
| **日期截取** | `iso.split('T')[0]`，不引入 dayjs 依赖，够用 |
| **创建后刷新** | `fetchGiftBooks()` 重拉列表，保证数据一致 |
| **click 跳转** | `router.push()` 到详情页，预留路由即可 |

---

# 22. 2026-06-01：礼薄卡片列表完善方案

## 22.1 当前代码状态

文件关系：

```
GiftBooks.vue（父组件）
  ├── el-dialog（新建礼薄弹窗）
  │     ├── el-input（事件名）
  │     ├── el-radio-group（方向：来/去）
  │     └── el-date-picker（日期）
  └── <Books />（子组件 — 卡片网格）

Books.vue（子组件）
  ├── CSS Grid 卡片网格（300px 最小列宽，auto-fill）
  ├── el-card × N（v-for="book in giftbooks"）
  │     ├── el-dropdown（操作菜单：编辑/删除）
  │     └── 卡片内容（图片、创建者、时间）
  └── script：fetchGiftBooks / formatDate / confirmDelete / deletebook
```

## 22.2 改动点拆解

### 改动 1：创建成功后刷新列表

**问题：** `GiftBooks.vue` 的 `createGiftBook` 成功后只关弹窗，不刷新子组件 Books 的列表。

**方案：父组件通过 ref 调用子组件暴露的方法**

```
GiftBooks.vue（父）                    Books.vue（子）
─────────────────                    ──────────────
<Books ref="booksRef" />              defineExpose({ fetchGiftBooks })

const booksRef = ref<InstanceType
  <typeof Books>>()

createGiftBook 成功后
  → booksRef.value.fetchGiftBooks()  →  重新请求 GET /giftbooks
```

原理：
- 子组件用 `defineExpose` 把内部方法暴露出去
- 父组件用 `ref` 拿到子组件实例
- 父组件调用 `childRef.value.xxx()` 触发子组件行为

**改动文件：**
- `Books.vue`：script 末尾加 `defineExpose({ fetchGiftBooks })`
- `GiftBooks.vue`：`<Books ref="booksRef" />` + 创建成功后 `booksRef.value.fetchGiftBooks()`

### 改动 2：编辑弹窗

**问题：** Books.vue 下拉菜单的「编辑」选项没有绑定事件处理。

**需要新增的内容：**

```
┌─────────────────────────────────────────┐
│  Books.vue 新增：                        │
│                                         │
│  ① 编辑弹窗（el-dialog + el-form）       │
│     - 事件名 input                      │
│     - 方向 radio-group                  │
│     - 日期 date-picker                  │
│     - 确定 / 取消 按钮                   │
│                                         │
│  ② 编辑相关的响应式数据                  │
│     - editDialogVisible: ref(false)     │
│     - editForm: ref({...})  当前编辑的卡片数据 │
│     - editingBookId: ref(null)  正在编辑的礼薄ID │
│                                         │
│  ③ 三个函数                             │
│     - openEditDialog(book)  打开弹窗+预填数据 │
│     - submitEdit()  调 POST /admin/giftbook/edit/:id │
│     - 成功后刷新列表 + 关弹窗             │
└─────────────────────────────────────────┘
```

**交互流程：**

```
点击「编辑」
  → openEditDialog(book)
    ├── editingBookId = book.ID
    ├── editForm = { event_name, direction, event_date }  // 预填当前值
    └── editDialogVisible = true

用户修改 → 点「确定」
  → submitEdit()
    → axios.post(`/admin/giftbook/edit/${editingBookId}`, editForm)
    → 成功：ElMessage.success + 关弹窗 + fetchGiftBooks()
    → 失败：ElMessage.error

点「取消」/ ESC / 点遮罩
  → editDialogVisible = false（v-model 自动）
```

**后端接口：** `POST /admin/giftbook/edit/:id`（已在 router 中定义，调 `UpdateGiftBook`）

**与新建弹窗的关系：** 编辑和新建是**两个独立的 el-dialog**，不是复用的。原因：
- 新建的 form 初始值为空；编辑需要预填当前值
- 新建提交 `POST /admin/giftbook`；编辑提交 `POST /admin/giftbook/edit/:id`
- 复用会增加复杂度（需要判断"当前是新建还是编辑"），不如分开清晰

### 改动 3：卡片点击跳转详情页

**问题：** 点卡片没有跳转到详情页。

**方案：** 卡片绑 `@click` + 路由跳转

```html
<el-card @click="goDetail(book.ID)">
  ...
  <el-dropdown @click.stop>   <!-- .stop 阻止冒泡，点三点不触发卡片跳转 -->
```

`@click.stop` 的作用：
```
用户点三个点图标
    │
    ▼
事件冒泡：dropdown 的 click → 卡片的 click
    │                              │
    └── @click.stop 阻断 ──→ ✗ 被阻止，不触发 goDetail
```

**路由变更：** 需要在 `router/index.ts` 新增详情页路由：

```ts
{
  path: 'giftbooks/:id',
  name: 'giftbook-detail',
  component: () => import('../views/GiftBookDetail.vue'),
  meta: { requiresAuth: true, hidden: true },
}
```

`hidden: true` 表示不在导航菜单中显示（详情页不是独立入口，只能从卡片点击进入）。

详情页组件 `GiftBookDetail.vue` 可以先创建为占位组件，后续再填充内容。

### 改动 4：小修小补

| 问题 | 当前代码 | 修复 |
|------|---------|------|
| `:key` 绑定对象 | `:key="book"` | `:key="book.ID"` — key 应该是唯一标识符，不是整个对象 |
| 方向值不一致 | `el-radio value="去"` | 后端默认值是 `"来"`，数据库定义 `default:来`。但目前后端没有校验，"来"/"往" 和 "来"/"去" 混用可能出问题。确认后端 direction 字段用 "来"/"往" 还是 "来"/"去" |

> **方向值约定：** 统一使用 "来"/"去"。GiftBooks.vue 的 radio 已用 "来"/"去"，与后端保持一致即可。

## 22.3 改动文件清单

| # | 文件 | 改动 |
|---|------|------|
| 1 | `Books.vue` | ① 加编辑弹窗（el-dialog + el-form）② `defineExpose({ fetchGiftBooks })` ③ `@click="goDetail(book.ID)"` + `@click.stop` ④ `:key="book.ID"` |
| 2 | `GiftBooks.vue` | ① `<Books ref="booksRef" />` ② `booksRef.value.fetchGiftBooks()` ③ 统一方向值为 "来"/"往" |
| 3 | `router/index.ts` | 新增 `/giftbooks/:id` 路由 |
| 4 | `views/GiftBookDetail.vue` | **新建** — 占位组件（第二步完善） |

## 22.4 数据流图

```
用户操作                     组件                     后端 API
─────────                  ────────                  ────────
［新建礼薄］
  填表单 → 点新建 → GiftBooks.createGiftBook() → POST /admin/giftbook
                        ↓ 成功
                   booksRef.fetchGiftBooks()  → GET /giftbooks → 刷新列表

［编辑礼薄］
  点编辑 → Books.openEditDialog()（预填数据）
  改表单 → 点确定 → Books.submitEdit() → POST /admin/giftbook/edit/:id
                        ↓ 成功
                   fetchGiftBooks() → GET /giftbooks → 刷新列表

［删除礼薄］
  点删除 → ElMessageBox.confirm（二次确认）
  点确定 → Books.deletebook() → POST /admin/giftbook/:id
                        ↓ 成功
                   fetchGiftBooks() → GET /giftbooks → 刷新列表

［查看详情］
  点卡片 → router.push('/giftbooks/:id') → 渲染 GiftBookDetail.vue
```

## 22.5 defineExpose 原理

Vue 3 的 `<script setup>` 组件默认是**封闭**的——父组件通过 `ref` 拿不到子组件内部定义的变量和方法。这是一种安全设计。

`defineExpose` 是 `<script setup>` 的编译宏，用于**显式暴露**内部成员给父组件：

```ts
// 子组件 Books.vue
const fetchGiftBooks = async () => { ... }
defineExpose({ fetchGiftBooks })  // 「我只暴露这个方法，其他都是私有的」
```

```ts
// 父组件 GiftBooks.vue
const booksRef = ref<InstanceType<typeof Books>>()
booksRef.value?.fetchGiftBooks()  // OK，这个被暴露了
booksRef.value?.deletebook()      // ❌ 没暴露，TS 报错
```

**没有 defineExpose 的世界：** 只能靠事件（emit）或全局状态（Pinia）通信。对于"父调子方法"这种场景，defineExpose 是最直接的方式。

---

# 23. 2026-06-01：礼薄卡片列表实施 + 礼金记录 CRUD Code Review

## 23.1 今日完成

### 任务 1：礼薄卡片列表完善 ✅

| # | 文件 | 改动 |
|---|------|------|
| 1 | `Books.vue` | ① 编辑弹窗（el-dialog + el-form，预填数据 + submitEdit）② `defineExpose({ fetchGiftBooks })` ③ `@click="goDetail(book.ID)"` 卡片跳转 + `@click.stop` 阻止冒泡 |
| 2 | `GiftBooks.vue` | ① `<Books ref="booksRef" />` ② 创建成功后 `booksRef.value?.fetchGiftBooks()` |
| 3 | `router/index.ts` | 新增 `/giftbooks/:id` → `GiftBookDetail.vue` |
| 4 | `GiftBookDetail.vue` | 🆕 基本信息展示 + 返回按钮 |
| 5 | `README.md` | 更新前端项目结构 |

Git commit: `c6e7144 feat: 礼薄卡片网格完善 — 编辑弹窗 + 卡片跳转详情 + 创建刷新列表`

### 任务 2：礼金记录增删功能 + 自定义金额

用户在 GiftBookDetail.vue 实现了添加记录弹窗，使用自定义金额模式：

```
el-radio-group（预设值）
  ├── 100 / 200 / 500
  └── 自定义 → v-if 显示 el-input number

submitRecord 提交时：
  selectedAmount === 0 ? customAmount : selectedAmount
```

GiftRecords.vue 实现了表格展示 + 搜索 + 删除。

## 23.2 Code Review：礼金记录发现的问题

### 🔴 关键问题

**1. 新增记录后表格不刷新**

GiftBookDetail 的 `submitRecord` 成功后关弹窗，但 GiftRecords 的 `fetchRecords` 没有被调用。缺少 ref + defineExpose 联动：

```
GiftBookDetail（父）            GiftRecords（子）
───────────────────            ──────────────
<GiftRecords ref="recordsRef" />
                                defineExpose({ fetchRecords })
submitRecord 成功后
  → recordsRef.fetchRecords()
```

**2. 表单提交后没有重置**

`dialogVisible = false` 后，`form`、`selectedAmount`、`customAmount` 都没有清空。下次打开弹窗还会看到上次的数据。

### 🟡 代码问题

**3. 死代码** — `const tableData = ref<Record[]>([]) || []`，`ref()` 永远返回对象，`|| []` 永远不会执行。

**4. 删除的 catch 混在一起** — `ElMessageBox.confirm` 的取消和 `axios.post` 的异常挤在同一个 try-catch，用 `if(err !== 'cancel')` 区分，不够清晰。

建议拆成两个 try-catch：

```ts
// 先确认
try {
  await ElMessageBox.confirm('确定要删除吗？', '提示', { ... })
} catch {
  return  // 用户取消，直接退出
}

// 再删
try {
  await axios.post(`/admin/giftrecord/${id}/records/${rid}`)
  ElMessage.success('删除成功')
  fetchRecords()
} catch (err: any) {
  ElMessage.error(err.response?.data?.error || '删除失败')
}
```

**5. 未使用的 import** — `ElTableColumn` import 了但没用到（Element Plus 全局注册后模板里直接用）。

**6. `handleEdit` 空函数** — 待实现。

## 23.3 前后端命名对应关系（知识点）

Go 的 `json` tag 决定了前端收到的 JSON key，而不是 Go 的字段名：

```go
// 后端
type GiftRecord struct {
    GiftBookID uint   `json:"gift_book_id"`  // ← json tag 决定
    PersonName string `json:"person_name"`
}
```

```
Go 字段名        json tag           前端收到的 JSON key
GiftBookID  →  "gift_book_id"  →  { gift_book_id: 1 }
PersonName  →  "person_name"   →  { person_name: "张三" }
```

**原则：后端 `json:"xxx"` 写什么，前端 TS 就写什么。**

## 23.4 明日待办

1. 修复 GiftBookDetail ↔ GiftRecords 联动刷新（ref + defineExpose）
2. 修复表单提交后重置（form 复位函数）
3. 修复 GiftRecords.vue 的死代码和 try-catch 结构
4. 实现 GiftRecords 编辑功能
5. 继续 Card.vue 或 Home.vue
