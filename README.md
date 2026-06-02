<p align="center">
  <a href="README.md">中文</a> | <a href="README-en.md">English</a>
</p>

<h1 align="center">Go-WebServer</h1>

<p align="center">
  <strong>基于 Go 标准库构建的轻量级 HTTP Web 服务器</strong>
  <br />
  <em>静态文件服务 · 用户认证 · 异步日志 · 中间件链</em>
</p>

<p align="center">
  <a href="#快速开始"><img src="https://img.shields.io/badge/快速开始-28a745?style=for-the-badge" alt="快速开始" /></a>
  <a href="LICENSE"><img src="https://img.shields.io/badge/许可证-MIT-blue?style=for-the-badge" alt="License" /></a>
</p>

<p align="center">
  <img src="https://img.shields.io/badge/Go-1.24-00ADD8?style=flat&logo=go&logoColor=white" alt="Go" />
  <img src="https://img.shields.io/badge/MySQL-8.0-4479A1?style=flat&logo=mysql&logoColor=white" alt="MySQL" />
  <img src="https://img.shields.io/badge/Nginx-反向代理-009639?style=flat&logo=nginx&logoColor=white" alt="Nginx" />
</p>

---

## 特性

- **零框架依赖** — 完全基于 Go `net/http` 标准库构建
- **用户认证** — 登录注册，MySQL 后端，支持表单和 JSON 两种提交方式
- **异步文件日志** — 非阻塞日志写入，可配置刷新间隔和日志级别
- **中间件链** — Panic 恢复、请求日志、超时控制，按顺序执行
- **优雅关闭** — 响应 SIGINT/SIGTERM，排空连接后退出
- **静态文件服务** — MIME 类型检测、短路径映射（`/login` → `/login.html`）、自定义错误页
- **Nginx 集成** — 附带反向代理配置，支持负载均衡和静态资源缓存

## 快速开始

### 环境要求

- Go 1.24+
- MySQL 8.0+

### 1. 初始化数据库

```bash
mysql -u root -p < init.sql
```

### 2. 修改配置

```bash
vim config.conf
```

关键配置项：

| 配置键 | 默认值 | 说明 |
|--------|--------|------|
| `port` | `9999` | HTTP 监听端口 |
| `db_host` | `127.0.0.1` | MySQL 主机 |
| `db_user` | `root` | MySQL 用户名 |
| `db_password` | `password` | MySQL 密码 |
| `db_name` | `webserver` | 数据库名 |

### 3. 编译运行

```bash
go run ./cmd/server/
```

或编译为二进制：

```bash
go build -o webserver ./cmd/server/
./webserver -config config.conf
```

### 4. 验证

```bash
curl http://localhost:9999/health
# → healthy
```

## 使用说明

### 命令行参数

```bash
./webserver -config config.conf -p 8080
```

| 参数 | 默认值 | 说明 |
|------|--------|------|
| `-config` | `config.conf` | 配置文件路径 |
| `-p` | `0`（使用配置） | 覆盖监听端口 |

### API 端点

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/` | 返回 `index.html` |
| GET | `/login` | 返回登录页面 |
| POST | `/login` | 用户认证（表单或 JSON） |
| GET | `/register` | 返回注册页面 |
| POST | `/register` | 创建新用户（表单或 JSON） |
| GET | `/health` | 健康检查 → `200 healthy` |

### 认证接口

支持 `application/x-www-form-urlencoded` 和 `application/json`：

```bash
# 表单方式
curl -X POST http://localhost:9999/login \
  -d "username=admin&password=admin123"

# JSON 方式
curl -X POST http://localhost:9999/login \
  -H "Content-Type: application/json" \
  -d '{"username":"admin","password":"admin123"}'
```

## 架构

```
cmd/server/          程序入口
internal/
├── config/          配置文件解析器
├── db/              MySQL 连接池与查询
├── handler/         HTTP 路由处理器（认证、静态文件、健康检查）
├── log/             基于 slog 的异步文件日志
├── middleware/       恢复、日志、超时中间件
└── server/          服务器生命周期与路由注册
resources/           静态资源（HTML、CSS、JS、图片）
```

### 中间件链

```
请求 → Recovery → Logging → Timeout → Handler
```

- **Recovery** — 捕获 panic，返回 500，记录堆栈
- **Logging** — 记录请求方法、路径、状态码、耗时、来源地址
- **Timeout** — 为每个请求设置 context 超时

## 配置说明

所有配置项位于 `config.conf`（`key = value` 格式，`#` 为注释）。

### 网络配置

| 配置键 | 默认值 | 说明 |
|--------|--------|------|
| `port` | `9999` | 监听端口 |
| `opt_linger` | `false` | 关闭连接时启用 SO_LINGER |
| `trigger_mode` | `4` | Epoll 模式（1=LT/LT, 2=LT/ET, 3=ET/LT, 4=ET/ET） |
| `thread_num` | `64` | 工作线程数 |
| `max_body_size` | `1048576` | 最大请求体（字节） |
| `connection_timeout` | `60` | 读写超时（秒） |

### 日志配置

| 配置键 | 默认值 | 说明 |
|--------|--------|------|
| `open_log` | `true` | 启用文件日志 |
| `log_file` | `log/webserver` | 日志文件路径（自动追加 `.log`） |
| `log_level` | `1` | 0=DEBUG, 1=INFO, 2=WARN, 3=ERROR |
| `log_queue_size` | `1024` | 异步日志通道缓冲大小 |
| `log_flush_interval` | `3` | 强制刷新间隔（秒） |

### 数据库配置

| 配置键 | 默认值 | 说明 |
|--------|--------|------|
| `connection_pool_size` | `16` | 连接池大小（0 = 禁用数据库） |
| `db_host` | `127.0.0.1` | MySQL 主机 |
| `db_port` | `3306` | MySQL 端口 |
| `db_user` | `root` | MySQL 用户名 |
| `db_password` | `password` | MySQL 密码 |
| `db_name` | `webserver` | 数据库名 |

## 技术栈

| 层级 | 技术 |
|------|------|
| 语言 | Go 1.24 |
| HTTP | `net/http`（标准库） |
| 数据库 | MySQL 8.0 + `go-sql-driver/mysql` |
| 日志 | `log/slog` + 异步文件处理器 |
| 反向代理 | Nginx（可选） |
| 前端 | Bootstrap, jQuery |

## 部署

### Nginx 反向代理

项目附带 Nginx 配置文件，适用于生产环境的负载均衡和静态资源缓存：

```bash
sudo cp nginx.conf /etc/nginx/sites-available/webserver
sudo ln -s /etc/nginx/sites-available/webserver /etc/nginx/sites-enabled/
sudo nginx -t && sudo systemctl reload nginx
```

请根据实际服务器路径和上游地址修改 `nginx.conf`。

## 贡献

1. Fork 本仓库
2. 创建功能分支（`git checkout -b feature/my-change`）
3. 提交更改（`git commit -m "add my change"`）
4. 推送分支（`git push origin feature/my-change`）
5. 提交 Pull Request

## 许可证

本项目基于 MIT 许可证开源。详见 [LICENSE](LICENSE)。
