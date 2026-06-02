# Oxy-WebServer (Go)

基于 Go 标准库 `net/http` 重构的高性能 Web 服务器。原项目为 C++20 epoll 主从 Reactor 架构，本版本利用 Go 语言特性大幅简化实现，同时保留全部业务功能。

## 快速启动

```bash
# 编译运行
go run ./cmd/server/

# 指定端口
go run ./cmd/server/ -p 8080

# 指定配置文件
go run ./cmd/server/ -config my.conf

# 编译为二进制
go build -o server ./cmd/server/
```

访问 `http://localhost:9999` 查看首页。

---

## 项目结构

```
Oxy-WebServer/
├── cmd/server/main.go              # 程序入口
├── internal/
│   ├── config/config.go            # 配置加载（兼容 config.conf 格式）
│   ├── log/logger.go               # 异步日志系统（slog + channel + worker goroutine）
│   ├── db/mysql.go                 # 数据库层（database/sql 连接池 + DAO）
│   ├── handler/
│   │   ├── health.go               # GET /health 健康检查
│   │   ├── static.go               # 静态文件服务 + 短路径映射
│   │   └── auth.go                 # POST /login, /register 认证处理
│   ├── middleware/
│   │   ├── logging.go              # 请求日志中间件
│   │   ├── recovery.go             # panic 恢复中间件
│   │   └── timeout.go              # 请求超时中间件
│   └── server/server.go            # 服务器组装、路由注册、优雅关闭
├── resources/                      # 静态资源（HTML/CSS/JS/图片/视频）
├── config.conf                     # 运行时配置
├── init.sql                        # 数据库初始化脚本
├── nginx.conf                      # Nginx 反向代理配置
└── go.mod                          # Go 模块定义
```

---

## 整体架构

### 请求处理流程

```
客户端请求
    │
    ▼
┌─────────────────────────────────────────────────┐
│  中间件链（从外到内执行）                          │
│                                                  │
│  Recovery → Log → Timeout → Handler              │
│     │        │       │         │                  │
│  捕获panic  记录请求  设置超时   路由分发           │
│  返回500    方法/路径  context   │                  │
│             状态码/耗时          │                  │
└─────────────────────────────────┼──────────────────┘
                                  │
                    ┌─────────────┼─────────────┐
                    ▼             ▼             ▼
              /health        /login         其他路径
              返回"healthy"  GET→login.html  静态文件服务
                             POST→数据库验证  resources/
```

### 启动流程

```
main()
  │
  ├─ 1. 解析命令行参数 (-config, -p)
  │
  ├─ 2. 加载配置文件 config.conf
  │     └─ config.LoadFile() → 解析 key=value 格式
  │
  ├─ 3. 初始化日志系统
  │     ├─ open_log=true  → 异步文件日志（channel + worker goroutine）
  │     └─ open_log=false → 标准输出日志
  │
  ├─ 4. 创建服务器 server.New(cfg)
  │     ├─ 连接数据库（可选，失败则降级运行）
  │     ├─ 注册路由
  │     └─ 组装中间件链
  │
  ├─ 5. 启动 HTTP 服务 srv.Run()
  │     ├─ goroutine 中启动 ListenAndServe
  │     └─ 主 goroutine 等待 SIGINT/SIGTERM 信号
  │
  └─ 6. 收到信号 → graceful shutdown（10秒超时）
        └─ 关闭数据库连接、刷新日志
```

---

## 模块详解

### 1. 配置系统 — `internal/config/config.go`

**职责**：解析 `config.conf` 文件，提供全局配置结构体。

```go
type Config struct {
    Port              int    // 监听端口
    ConnectionTimeout int    // 连接超时（秒）
    LogLevel          int    // 0=DEBUG, 1=INFO, 2=WARN, 3=ERROR
    ConnPoolSize      int    // 数据库连接池大小
    ResourceRoot      string // 静态资源根目录
    // ... 更多配置项
}
```

**Go 语言特性体现**：
- **结构体（struct）**：用字段聚合替代 C++ 的类成员变量，零样板代码
- **`bufio.Scanner`**：逐行扫描文件，替代 C++ 手写的 `getline` + 字符串处理
- **`strings.SplitN`**：一行代码完成 key=value 解析，替代 C++ 的 `find('=')` + `substr`
- **`defer f.Close()`**：函数退出时自动关闭文件，替代 C++ 的 RAII 或手动 close
- **函数返回值 `(*Config, error)`**：Go 惯用的"返回值 + 错误"模式，替代 C++ 的异常或错误码

### 2. 日志系统 — `internal/log/logger.go`

**职责**：异步写入日志到文件，支持级别过滤和定期刷盘。

**架构**：

```
调用 slog.Info("msg", "key", "val")
    │
    ▼
asyncHandler.Handle()
    │  格式化为 "2026-06-02 21:49:21.919 [INFO] msg key=val\n"
    │
    ├─ 队列未满 → channel 发送
    └─ 队列已满 → 直接写 stdout（降级）
         │
         ▼
    worker goroutine
         │  从 channel 读取，缓冲到 strings.Builder
         │
         ├─ 缓冲 ≥ 4096 字节 → flush 到文件
         └─ 定时器到期（3秒） → flush 到文件
```

**Go 语言特性体现**：
- **`chan string`（channel）**：goroutine 间安全通信的管道，替代 C++ 的 `std::mutex` + `std::condition_variable` + `std::deque` 组合。一行 `h.ch <- msg` 完成线程安全的生产者入队，`msg := <-h.ch` 完成消费者出队
- **`select` 多路复用**：同时监听 channel 消息和定时器，替代 C++ 的 `wait_for` 超时等待
- **`time.Ticker`**：定时触发刷盘，替代 C++ 的 `std::condition_variable::wait_for`
- **`go h.worker()`（goroutine 启动）**：一行代码启动后台工作线程，替代 C++ 的 `std::thread` + 手动管理生命周期
- **`slog.Handler` 接口**：实现 `Enabled/Handle/WithAttrs/WithGroup` 四个方法即可自定义日志后端，这是 Go **接口隐式实现**的典型用法——无需声明 "implements slog.Handler"
- **`sync.Mutex`**：保护缓冲区的并发访问，与 C++ 用法类似但更简洁
- **`defer close(h.done)`**：worker 退出时自动通知调用方，配合 `<-h.done` 实现优雅关闭

### 3. 数据库层 — `internal/db/mysql.go`

**职责**：MySQL 连接池管理、用户认证（登录/注册）。

```go
type DB struct {
    *sql.DB  // 嵌入标准库连接池
}

// 使用参数化查询，防止 SQL 注入
db.QueryRow("SELECT password FROM user WHERE username = ?", username)
```

**Go 语言特性体现**：
- **结构体嵌入（embedding）**：`DB` 嵌入 `*sql.DB`，自动继承所有方法（`Query`、`Exec`、`Close` 等），无需手动转发。这是 Go 的**组合优于继承**哲学
- **`database/sql` 标准库**：内置连接池，通过 `SetMaxOpenConns`/`SetMaxIdleConns` 配置，替代 C++ 手写的 `SqlConnPool`（~150 行）+ `SqlConnRAII`（~20 行）
- **`_ "github.com/go-sql-driver/mysql"`（空白标识符导入）**：只执行驱动的 `init()` 函数注册驱动，不直接使用包内容。这是 Go 的**驱动注册模式**
- **`iota` 枚举**：`LoginSuccess = iota` 自动生成递增常量，替代 C++ 的 `enum class`
- **`regexp.MustCompile`（预编译正则）**：包级别变量，程序启动时编译一次，全局复用
- **nil 接收者方法**：`func (db *DB) Login(...)` 中检查 `db == nil`，允许数据库不可用时安全降级
- **`sql.ErrNoRows`**：区分"无结果"和"查询错误"，替代 C++ 的 `mysql_num_rows` 检查

### 4. HTTP 处理器 — `internal/handler/`

#### 4.1 静态文件服务 — `static.go`

```go
func Static(root string) http.HandlerFunc {
    fs := http.Dir(root)
    return func(w http.ResponseWriter, r *http.Request) {
        // 短路径映射: /login → /login.html
        if mapped, ok := pathMap[path]; ok {
            r.URL.Path = mapped
        }
        http.ServeContent(w, r, name, modTime, file)
    }
}
```

**Go 语言特性体现**：
- **闭包（Closure）**：`Static(root)` 返回的函数捕获了 `fs` 变量，形成闭包。替代 C++ 的类 + 成员变量模式
- **`http.Dir` + `http.ServeContent`**：标准库封装了文件读取、Content-Type 检测、Range 请求支持、Last-Modified 头等，替代 C++ 的 `open` + `mmap` + `writev`（~80 行）
- **`map[string]string`**：字面量初始化的映射表，替代 C++ 的 `std::unordered_map`
- **`os.IsNotExist(err)`**：类型断言检查错误类型，替代 C++ 的 `errno == ENOENT`

#### 4.2 认证处理 — `auth.go`

```go
type Auth struct {
    DB *db.DB
}

func (a *Auth) Login() http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        username, password, ok := a.parseCredentials(w, r)
        result, err := a.DB.Login(username, password)
        switch result {
        case db.LoginSuccess:
            http.Redirect(w, r, "/welcome.html", http.StatusFound)
        case db.LoginUserNotFound:
            a.respondError(w, r, http.StatusUnauthorized, "User not found")
        }
    }
}
```

**Go 语言特性体现**：
- **方法集与接口**：`Auth.Login()` 返回 `http.HandlerFunc`，后者实现了 `http.Handler` 接口。任何函数只要签名是 `func(http.ResponseWriter, *http.Request)` 就是一个 Handler
- **`encoding/json`**：标准库 JSON 解码，一行 `json.NewDecoder(r.Body).Decode(&body)` 替代 C++ 手写的 JSON 解析器（~50 行）
- **`r.ParseForm()` + `r.FormValue()`**：标准库表单解析，替代 C++ 手写的 URL 解码器（~30 行）
- **匿名结构体**：`var body struct { Username string \`json:"username"\` }` 在函数内部定义临时结构体，无需单独声明
- **多返回值**：`parseCredentials` 返回 `(username, password string, ok bool)`，替代 C++ 的输出参数或结构体

#### 4.3 健康检查 — `health.go`

```go
func Health() http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        w.Write([]byte("healthy"))
    }
}
```

最简 Handler 示例：函数即 Handler，闭包即配置。

### 5. 中间件 — `internal/middleware/`

Go 的中间件模式是**函数包装函数**：每个中间件接收一个 `http.Handler`，返回一个新的 `http.Handler`。

#### 5.1 Recovery — `recovery.go`

```go
func Recovery(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        defer func() {
            if err := recover(); err != nil {
                // 记录 panic 堆栈，返回 500
            }
        }()
        next.ServeHTTP(w, r)
    })
}
```

**Go 语言特性体现**：
- **`recover()`**：Go 的 panic 恢复机制，类似 try-catch 但更轻量。`defer + recover` 组合捕获当前 goroutine 的 panic
- **`debug.Stack()`**：获取完整的 goroutine 堆栈信息

#### 5.2 Logging — `logging.go`

```go
type responseWriter struct {
    http.ResponseWriter  // 嵌入接口
    status int
}

func (rw *responseWriter) WriteHeader(code int) {
    rw.status = code           // 记录状态码
    rw.ResponseWriter.WriteHeader(code)  // 调用原始实现
}
```

**Go 语言特性体现**：
- **接口嵌入 + 方法覆盖**：`responseWriter` 嵌入 `http.ResponseWriter` 接口，只覆盖 `WriteHeader` 方法以捕获状态码，其他方法（`Write`、`Header`）自动委托给原始实现。这是 Go 的**装饰器模式**
- **`time.Since(start)`**：简洁的耗时计算

#### 5.3 Timeout — `timeout.go`

```go
func Timeout(d time.Duration) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            ctx, cancel := context.WithTimeout(r.Context(), d)
            defer cancel()
            next.ServeHTTP(w, r.WithContext(ctx))
        })
    }
}
```

**Go 语言特性体现**：
- **`context.Context`**：Go 的请求上下文传播机制。超时、取消信号通过 context 在整个调用链中传递，任何下游函数都可以通过 `ctx.Done()` 检测超时
- **函数式选项模式**：`Timeout(d)` 返回一个中间件工厂函数，`d` 作为配置被捕获在闭包中
- **`defer cancel()`**：确保 context 资源被释放，即使 handler panic

### 6. 服务器组装 — `internal/server/server.go`

```go
// 中间件链组装
var root http.Handler = mux
root = middleware.Timeout(timeout)(root)  // 最内层
root = middleware.Log(root)
root = middleware.Recovery(root)          // 最外层

// 优雅关闭
go func() { s.http.ListenAndServe() }()  // goroutine 中启动
sig := <-quit                              // 阻塞等待信号
s.http.Shutdown(ctx)                       // 优雅关闭
```

**Go 语言特性体现**：
- **`http.ServeMux`**：标准库路由器，`HandleFunc` 注册路径模式
- **goroutine + channel 信号同步**：主 goroutine 通过 `signal.Notify` 等待系统信号，工作 goroutine 运行 HTTP 服务
- **`context.WithTimeout`**：为 Shutdown 设置超时，避免无限等待
- **`defer srv.Close()`**：main 函数退出时自动清理资源

---

## Go 与 C++ 的关键差异总结

| 维度 | C++ 原版 | Go 重构版 |
|------|---------|----------|
| **并发模型** | 手动 epoll + 线程池（~200 行） | goroutine + net/http（0 行，标准库） |
| **连接管理** | EPOLLONESHOT + 手动 fd 管理 | 每连接一个 goroutine，自动调度 |
| **HTTP 解析** | 手写状态机（~300 行） | net/http 标准库（0 行） |
| **缓冲区** | 手写 Buffer 类（~150 行） | bufio / io.Copy（标准库） |
| **定时器** | 手写最小堆（~200 行） | http.Server 超时参数 |
| **日志** | 阻塞队列 + 工作线程（~200 行） | slog + channel + goroutine（~130 行） |
| **数据库池** | 手写连接池 + RAII（~170 行） | database/sql 内置池（~10 行配置） |
| **中间件** | 无（硬编码在 WebServer 类中） | 函数包装链（~60 行） |
| **错误处理** | 异常 / 错误码 / errno | 多返回值 `(result, error)` |
| **资源管理** | RAII / 手动 close | `defer` 自动清理 |
| **代码总量** | ~2500 行 C++ | ~650 行 Go |

### Go 被消除的 C++ 复杂性

1. **无需线程池**：goroutine 由 Go runtime 调度，创建成本 ~8KB 栈空间，可轻松创建数十万个
2. **无需手动 epoll**：`net/http` 内部使用 epoll/kqueue，对开发者透明
3. **无需 RAII**：`defer` 语句确保资源释放，不需要析构函数
4. **无需头文件**：Go 的包系统通过首字母大小写控制可见性，无需 `.h` / `.cpp` 分离
5. **无需模板元编程**：Go 的接口是隐式实现的，无需泛型即可实现多态
6. **无需智能指针**：Go 的 GC 自动管理内存，`&Server{}` 返回指针无需 `make_unique`

---

## 配置说明

`config.conf` 格式与 C++ 版本完全兼容：

```ini
# 网络
port = 9999
connection_timeout = 60

# 日志
open_log = true
log_level = 1          # 0=DEBUG 1=INFO 2=WARN 3=ERROR
log_file = log/webserver
log_flush_interval = 3

# 数据库（可选，连接失败时降级运行）
connection_pool_size = 16
db_host = 127.0.0.1
db_port = 3306
db_user = root
db_password = password
db_name = webserver

# 资源
resource_root = resources/
```

## 路由表

| 方法 | 路径 | 处理器 | 说明 |
|------|------|--------|------|
| GET | `/health` | Health | 返回 "healthy" |
| GET | `/login` | Static | 返回 login.html |
| POST | `/login` | Auth.Login | 验证用户名密码 |
| GET | `/register` | Static | 返回 register.html |
| POST | `/register` | Auth.Register | 注册新用户 |
| GET | `/` | Static | 返回 index.html |
| GET | `/css/*`, `/js/*`, `/images/*` | Static | 静态资源 |

## 依赖

- Go 1.24+
- `github.com/go-sql-driver/mysql` — MySQL 驱动（唯一外部依赖）
- MySQL 5.7+（可选，不连接时降级运行）
