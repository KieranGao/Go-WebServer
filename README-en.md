<p align="center">
  <a href="README.md">中文</a> | <a href="README-en.md">English</a>
</p>

<h1 align="center">Go-WebServer</h1>

<p align="center">
  <strong>A lightweight HTTP web server built with Go standard library</strong>
  <br />
  <em>Static file serving · User authentication · Async logging · Middleware chain</em>
</p>

<p align="center">
  <a href="#quick-start"><img src="https://img.shields.io/badge/Quick_Start-28a745?style=for-the-badge" alt="Quick Start" /></a>
  <a href="LICENSE"><img src="https://img.shields.io/badge/License-MIT-blue?style=for-the-badge" alt="License" /></a>
</p>

<p align="center">
  <img src="https://img.shields.io/badge/Go-1.24-00ADD8?style=flat&logo=go&logoColor=white" alt="Go" />
  <img src="https://img.shields.io/badge/MySQL-8.0-4479A1?style=flat&logo=mysql&logoColor=white" alt="MySQL" />
  <img src="https://img.shields.io/badge/Nginx-Reverse_Proxy-009639?style=flat&logo=nginx&logoColor=white" alt="Nginx" />
</p>

---

## Features

- **Zero framework dependency** — Built entirely on Go's `net/http` standard library
- **User authentication** — Login and registration with MySQL backend, supports both form and JSON input
- **Async file logging** — Non-blocking log writer with configurable flush intervals and log levels
- **Middleware chain** — Panic recovery, request logging, and timeout control applied in order
- **Graceful shutdown** — Handles SIGINT/SIGTERM, drains connections before exit
- **Static file serving** — MIME type detection, short-path mapping (`/login` → `/login.html`), custom error pages
- **Nginx integration** — Included reverse proxy config with load balancing and static asset caching

## Quick Start

### Prerequisites

- Go 1.24+
- MySQL 8.0+

### 1. Initialize the database

```bash
mysql -u root -p < init.sql
```

### 2. Edit configuration

```bash
vim config.conf
```

Key settings to review:

| Key | Default | Description |
|-----|---------|-------------|
| `port` | `9999` | HTTP listen port |
| `db_host` | `127.0.0.1` | MySQL host |
| `db_user` | `root` | MySQL user |
| `db_password` | `password` | MySQL password |
| `db_name` | `webserver` | Database name |

### 3. Build and run

```bash
go run ./cmd/server/
```

Or build a binary:

```bash
go build -o webserver ./cmd/server/
./webserver -config config.conf
```

### 4. Verify

```bash
curl http://localhost:9999/health
# → healthy
```

## Usage

### CLI Flags

```bash
./webserver -config config.conf -p 8080
```

| Flag | Default | Description |
|------|---------|-------------|
| `-config` | `config.conf` | Path to configuration file |
| `-p` | `0` (use config) | Override listen port |

### API Endpoints

| Method | Path | Description |
|--------|------|-------------|
| GET | `/` | Serve `index.html` |
| GET | `/login` | Serve login page |
| POST | `/login` | Authenticate user (form or JSON) |
| GET | `/register` | Serve registration page |
| POST | `/register` | Create new user (form or JSON) |
| GET | `/health` | Health check → `200 healthy` |

### Authentication

Supports `application/x-www-form-urlencoded` and `application/json`:

```bash
# Form-based
curl -X POST http://localhost:9999/login \
  -d "username=admin&password=admin123"

# JSON-based
curl -X POST http://localhost:9999/login \
  -H "Content-Type: application/json" \
  -d '{"username":"admin","password":"admin123"}'
```

## Architecture

```
cmd/server/          Application entry point
internal/
├── config/          Configuration file parser
├── db/              MySQL connection pool and queries
├── handler/         HTTP route handlers (auth, static, health)
├── log/             Async file-based slog handler
├── middleware/       Recovery, logging, timeout middleware
└── server/          Server lifecycle and route wiring
resources/           Static assets (HTML, CSS, JS, images)
```

### Middleware Chain

```
Request → Recovery → Logging → Timeout → Handler
```

- **Recovery** — Catches panics, returns 500, logs stack trace
- **Logging** — Records method, path, status, duration, remote addr
- **Timeout** — Applies context deadline per request

## Configuration

All settings are in `config.conf` (key = value format, `#` for comments).

### Network

| Key | Default | Description |
|-----|---------|-------------|
| `port` | `9999` | Listen port |
| `opt_linger` | `false` | Enable SO_LINGER on close |
| `trigger_mode` | `4` | Epoll mode (1=LT/LT, 2=LT/ET, 3=ET/LT, 4=ET/ET) |
| `thread_num` | `64` | Worker thread count |
| `max_body_size` | `1048576` | Max request body (bytes) |
| `connection_timeout` | `60` | Read/write timeout (seconds) |

### Logging

| Key | Default | Description |
|-----|---------|-------------|
| `open_log` | `true` | Enable file logging |
| `log_file` | `log/webserver` | Log file path (`.log` appended) |
| `log_level` | `1` | 0=DEBUG, 1=INFO, 2=WARN, 3=ERROR |
| `log_queue_size` | `1024` | Async log channel buffer size |
| `log_flush_interval` | `3` | Force flush interval (seconds) |

### Database

| Key | Default | Description |
|-----|---------|-------------|
| `connection_pool_size` | `16` | Connection pool size (0 = disable DB) |
| `db_host` | `127.0.0.1` | MySQL host |
| `db_port` | `3306` | MySQL port |
| `db_user` | `root` | MySQL user |
| `db_password` | `password` | MySQL password |
| `db_name` | `webserver` | Database name |

## Tech Stack

| Layer | Technology |
|-------|------------|
| Language | Go 1.24 |
| HTTP | `net/http` (standard library) |
| Database | MySQL 8.0 + `go-sql-driver/mysql` |
| Logging | `log/slog` + async file handler |
| Reverse Proxy | Nginx (optional) |
| Frontend | Bootstrap, jQuery |

## Deployment

### Nginx Reverse Proxy

An Nginx config is included for production use with load balancing and static caching:

```bash
sudo cp nginx.conf /etc/nginx/sites-available/webserver
sudo ln -s /etc/nginx/sites-available/webserver /etc/nginx/sites-enabled/
sudo nginx -t && sudo systemctl reload nginx
```

Edit `nginx.conf` to match your server paths and upstream addresses.

## Contributing

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/my-change`)
3. Commit your changes (`git commit -m "add my change"`)
4. Push to the branch (`git push origin feature/my-change`)
5. Open a Pull Request

## License

This project is licensed under the MIT License. See [LICENSE](LICENSE) for details.
