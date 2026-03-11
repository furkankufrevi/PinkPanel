# V0 - Technology Stack & Project Scaffolding

## Stack Decision

A hosting panel has unique requirements: it must manage Linux system services, execute privileged commands, monitor resources in real-time, and run with minimal overhead on the server it manages. The stack must prioritize **low memory footprint**, **system-level access**, **security**, and **single-binary deployment**.

---

## Chosen Stack

### Backend: **Go**

**Why Go over alternatives:**

| Criteria | Go | Node.js | Python | Rust |
|----------|-----|---------|--------|------|
| System operations | Excellent | Poor | Moderate | Excellent |
| Memory footprint | ~10-30MB | ~50-150MB | ~80-200MB | ~5-20MB |
| Single binary deploy | Yes | No (needs runtime) | No (needs runtime) | Yes |
| Development speed | Fast | Fast | Fast | Slow |
| Concurrency model | Goroutines (native) | Event loop | asyncio/threads | async/tokio |
| Infrastructure ecosystem | Docker, K8s, Terraform | Limited | Ansible | Limited |
| Compile time | Fast | N/A | N/A | Slow |

Go wins because:
- **Single binary**: No runtime dependencies on target servers. Just copy and run.
- **System-level**: Excellent `os/exec`, `syscall`, file I/O, and networking in stdlib.
- **Low memory**: Critical since the panel runs on the same server it manages.
- **Concurrency**: Goroutines are perfect for background monitoring, log tailing, and parallel service management.
- **Proven**: Docker, Kubernetes, Terraform, Traefik, Caddy — all infrastructure tools choose Go.

**Framework: Fiber v2**
- Express-like API (familiar to most developers)
- Fastest Go HTTP framework (built on fasthttp)
- Middleware ecosystem (auth, CORS, rate limiting, etc.)
- WebSocket support for real-time monitoring

### Frontend: **React + TypeScript + Vite**

**Why React:**
- Largest ecosystem for complex dashboard UIs
- Rich component libraries (Shadcn/ui, Radix)
- Strong TypeScript support
- Huge talent pool

**UI Framework: Shadcn/ui + Tailwind CSS v4**
- Not a component library — copies components into your project (full control)
- Built on Radix primitives (accessible, composable)
- Tailwind for rapid, consistent styling
- Perfect for data-heavy admin panels

**State Management: TanStack Query (React Query)**
- Server state management (caching, refetching, polling)
- Perfect for a panel that constantly fetches server data
- Reduces boilerplate vs Redux

**Client State: Zustand**
- Lightweight global state for UI-only state (sidebar open, theme, modals)
- No boilerplate, no providers, simple API
- Complements TanStack Query (server state) cleanly

**Build Tool: Vite**
- Fast HMR for development
- Optimized production builds
- The frontend gets embedded into the Go binary for single-binary deployment

### Database: **SQLite + Redis**

**SQLite (primary data store):**
- Zero configuration, file-based
- No separate database server process (saves resources)
- Perfect for single-server panel data (users, domains, plans, settings)
- Embedded in the Go binary via `modernc.org/sqlite` (pure Go, no CGO)
- Handles thousands of concurrent reads easily with WAL mode
- Backup is just copying a file
- Busy timeout and connection pooling configured to prevent lock contention

**Redis (cache, sessions, real-time):**
- Session storage
- Real-time metrics buffer (time-series ring buffer)
- Background job queue (delayed tasks, retries)
- Pub/sub for WebSocket updates (live monitoring)
- Optional: can run without Redis in minimal mode (fallback to in-memory with `sync.Map` + goroutines)

**Why not PostgreSQL:**
- Overkill for a single-server hosting panel
- Requires running a separate database server
- SQLite handles the data volume of a hosting panel perfectly
- If multi-server is needed later (V4), can add PostgreSQL as an option

---

## System Agent Architecture

```
┌──────────────────────────────────────────────────────────┐
│                       PinkPanel                           │
│                                                           │
│  ┌───────────┐  ┌───────────┐  ┌───────────┐            │
│  │ Frontend   │  │ REST API  │  │   CLI     │            │
│  │ (React)    │◄─┤ (Fiber)   │  │ (Cobra)   │            │
│  │ embedded   │  │           │  │           │            │
│  └───────────┘  └─────┬─────┘  └─────┬─────┘            │
│                       │               │                   │
│                 ┌─────▼───────────────▼────────┐         │
│                 │       Core Engine             │         │
│                 │  (Business Logic + Services)  │         │
│                 └─────┬───────────┬────────────┘         │
│                       │           │                       │
│    ┌──────────────────┤           ├──────────────┐       │
│    │                  │           │              │       │
│  ┌─▼──────┐   ┌──────▼────┐  ┌──▼────────┐ ┌──▼─────┐ │
│  │ SQLite  │   │  Template │  │  Redis    │ │ Config │ │
│  │  (data) │   │  Engine   │  │  (cache)  │ │ Store  │ │
│  └────────┘   │ (configs) │  └───────────┘ └────────┘ │
│               └─────┬─────┘                             │
│                     │                                    │
│               ┌─────▼──────────────┐                    │
│               │   System Agent     │                    │
│               │   (root, socket)   │                    │
│               └─────┬──────────────┘                    │
│                     │                                    │
│    ┌────────────────┼───────────────────┐               │
│    │                │                   │               │
│  ┌─▼──────┐  ┌─────▼─────┐  ┌─────────▼┐              │
│  │ NGINX  │  │  PHP-FPM  │  │ MariaDB  │              │
│  │        │  │  Node.js  │  │ Postfix  │              │
│  └────────┘  └───────────┘  │ Dovecot  │              │
│                              │ BIND9    │              │
│                              │ vsftpd   │              │
│                              └──────────┘              │
└──────────────────────────────────────────────────────────┘
```

### System Agent (separate process running as root)
- The main panel runs as a non-root user (`pinkpanel`) for security
- A small privileged agent handles root operations (service restart, config writes, user creation)
- Communication via Unix socket at `/var/run/pinkpanel/agent.sock` (not network-exposed)
- JSON-RPC protocol with request authentication via shared secret (file-based, 0600 permissions)
- Strict command allowlist — agent only executes pre-defined operations, no arbitrary commands
- All operations logged with timestamp, caller, and arguments
- Minimal attack surface: <1000 lines of Go

### Agent Command Allowlist Categories
- **Service control**: start/stop/restart/reload/status for managed services
- **Config writes**: write files only to predefined paths (`/etc/nginx/sites-available/`, `/etc/php/`, etc.)
- **System users**: create/delete system users within a UID range (1000-60000)
- **File ownership**: chown/chmod only within `/home/` and `/var/www/`
- **Package operations**: install/remove specific packages from an allowlist
- **Certificate files**: write to `/etc/letsencrypt/` and NGINX SSL paths
- **DNS zones**: write to BIND zone directory
- **System info**: read-only system metrics (CPU, RAM, disk, network)
- **Firewall**: iptables/nftables rule management
- **Backup operations**: tar/gzip within backup directories

---

## Template Engine for Service Configs

A critical piece missing from many panels — PinkPanel needs a robust config generation system.

### Go `text/template` Based
- All service configs (NGINX vhosts, PHP-FPM pools, BIND zones, Postfix maps, etc.) generated from Go templates
- Templates stored in `configs/templates/` and embedded into the binary
- Template variables populated from database + system state
- Config validation before writing (e.g., `nginx -t` before applying)
- Atomic write: write to temp file → validate → rename to final path
- Rollback: keep previous config version, auto-restore on validation failure

### Template Directory
```
configs/templates/
├── nginx/
│   ├── vhost.conf.tmpl          # Standard domain vhost
│   ├── vhost-ssl.conf.tmpl      # SSL-enabled vhost
│   ├── vhost-proxy.conf.tmpl    # Reverse proxy vhost
│   ├── php-fpm-proxy.conf.tmpl  # PHP-FPM upstream block
│   └── security-headers.conf.tmpl
├── php/
│   ├── pool.conf.tmpl           # PHP-FPM pool per domain
│   └── php.ini.tmpl             # Custom php.ini overrides
├── dns/
│   ├── zone.db.tmpl             # BIND zone file
│   └── named.conf.local.tmpl    # BIND zone inclusion
├── mail/
│   ├── postfix-main.cf.tmpl
│   ├── postfix-virtual.tmpl
│   ├── dovecot.conf.tmpl
│   └── dkim-signing.conf.tmpl
├── ftp/
│   └── vsftpd-user.conf.tmpl
├── systemd/
│   ├── pinkpanel.service.tmpl
│   └── pinkpanel-agent.service.tmpl
└── ssl/
    └── openssl.cnf.tmpl
```

---

## Configuration Management

### Panel Configuration (`pinkpanel.yml`)
```yaml
server:
  host: 0.0.0.0
  port: 8443
  ssl:
    enabled: true
    cert: /usr/local/pinkpanel/ssl/panel.crt
    key: /usr/local/pinkpanel/ssl/panel.key

database:
  path: /usr/local/pinkpanel/data/pinkpanel.db
  wal_mode: true
  busy_timeout_ms: 5000
  max_open_conns: 25

redis:
  enabled: true
  address: 127.0.0.1:6379
  password: ""
  db: 0

agent:
  socket: /var/run/pinkpanel/agent.sock
  secret_file: /usr/local/pinkpanel/data/.agent-secret

logging:
  level: info          # debug, info, warn, error
  file: /usr/local/pinkpanel/logs/panel.log
  max_size_mb: 100
  max_backups: 5
  max_age_days: 30
  compress: true

security:
  jwt_secret_file: /usr/local/pinkpanel/data/.jwt-secret
  access_token_ttl: 15m
  refresh_token_ttl: 168h  # 7 days
  bcrypt_cost: 12
  rate_limit:
    login: "5/minute"
    api: "100/minute"

paths:
  web_root: /home
  backups: /usr/local/pinkpanel/data/backups
  templates: embedded    # "embedded" or path to custom templates
  temp: /tmp/pinkpanel

services:
  nginx:
    config_dir: /etc/nginx
    sites_dir: /etc/nginx/sites-available
    enabled_dir: /etc/nginx/sites-enabled
  php:
    config_dir: /etc/php
    pool_dir_pattern: /etc/php/{version}/fpm/pool.d
  mysql:
    socket: /var/run/mysqld/mysqld.sock
  dns:
    zone_dir: /etc/bind/zones
    config_file: /etc/bind/named.conf.local
  mail:
    postfix_dir: /etc/postfix
    dovecot_dir: /etc/dovecot
  ftp:
    config_dir: /etc/vsftpd
```

### Configuration Loading (Viper)
- Load from `pinkpanel.yml`
- Override with environment variables: `PINKPANEL_SERVER_PORT=9443`
- Override with CLI flags: `--server.port 9443`
- Precedence: CLI flags > env vars > config file > defaults
- Hot-reload on config file change (for non-critical settings)

---

## Logging Architecture

### Structured Logging (`zerolog`)
- JSON-structured logs for machine parsing
- Human-readable console output for development
- Log levels: debug, info, warn, error, fatal
- Request logging middleware (method, path, status, duration, IP)
- Contextual fields: request_id, user_id, domain, action

### Log Rotation (`lumberjack`)
- Max file size (default: 100MB)
- Max backup count (default: 5)
- Max age (default: 30 days)
- Compression of rotated files

### Audit Trail
- Separate audit log for security-sensitive actions
- Structured: who, what, when, where, before/after values
- Stored in SQLite `audit_log` table (queryable via API)
- Never rotated, only trimmed by configurable retention policy

---

## Error Handling Strategy

### Backend
- Custom error types with codes: `ErrDomainNotFound`, `ErrQuotaExceeded`, etc.
- All errors wrapped with context using `fmt.Errorf("creating domain %s: %w", name, err)`
- API returns consistent JSON error format:
  ```json
  {
    "error": {
      "code": "DOMAIN_NOT_FOUND",
      "message": "Domain example.com not found",
      "details": {}
    }
  }
  ```
- HTTP status codes: 400 (validation), 401 (auth), 403 (forbidden), 404 (not found), 409 (conflict), 422 (business rule), 500 (internal)
- Internal errors never leak to API responses (logged server-side, generic message to client)

### Frontend
- TanStack Query error handling with retry (3 retries for 5xx, no retry for 4xx)
- Global error boundary for unhandled React errors
- Toast notifications for user-facing errors
- Error page for fatal errors (500, network down)

---

## Testing Strategy

### Backend Testing
- **Unit tests**: core business logic (pure Go, no I/O mocking needed)
- **Integration tests**: API handlers with real SQLite (in-memory) and mock agent
- **Agent tests**: agent commands with real filesystem (in temp dir)
- Test database seeding helpers
- `go test ./...` runs all tests
- Target: 80%+ coverage on `internal/core/`

### Frontend Testing
- **Unit tests**: utility functions, hooks (Vitest)
- **Component tests**: key UI components with React Testing Library
- **E2E tests**: critical flows (login, add domain, manage files) with Playwright
- Target: E2E covers all critical user flows

### CI Pipeline
- GitHub Actions
- On PR: lint + unit tests + integration tests + frontend build
- On merge to main: full test suite + build binaries + E2E tests
- Release: build binaries for linux/amd64 and linux/arm64

---

## Development Workflow

### Prerequisites
- Go 1.22+
- Node.js 20+ (LTS)
- Make
- Docker (optional, for test environment)

### Dev Commands (Makefile)
```makefile
make dev          # Start backend (air hot-reload) + frontend (vite dev) concurrently
make build        # Build production binaries
make test         # Run all Go tests
make test-e2e     # Run Playwright E2E tests
make lint         # golangci-lint + eslint
make fmt          # gofmt + prettier
make migrate      # Run database migrations
make migrate-new  # Create new migration file
make clean        # Remove build artifacts
make install      # Build + install to /usr/local/pinkpanel (requires root)
```

### Hot Reload (Development)
- Backend: `air` (Go hot-reloader) — rebuilds on `.go` file changes
- Frontend: Vite dev server with HMR on port 5173
- In dev mode, Go serves API on :8443, Vite proxies API requests to Go

### Code Quality
- `golangci-lint` with config: `govet`, `errcheck`, `staticcheck`, `gosec`, `gocritic`
- `eslint` + `prettier` for frontend
- Pre-commit hooks via `lefthook`: lint + format on commit
- PR required for merge to `main`

---

## Database Migration System

### `golang-migrate`
- SQL-based migrations in `internal/db/migrations/`
- Naming: `000001_create_admins.up.sql` / `000001_create_admins.down.sql`
- Migrations embedded into binary via `embed`
- Auto-run on startup (migrate to latest)
- `make migrate-new NAME=add_users_table` scaffolds new migration pair
- Down migrations for rollback support
- Migration version tracked in `schema_migrations` table

---

## Project Structure

```
pinkpanel/
├── cmd/
│   ├── server/              # Main panel server entry point
│   │   └── main.go
│   ├── agent/               # Privileged system agent entry point
│   │   └── main.go
│   └── cli/                 # CLI tool entry point
│       └── main.go
├── internal/
│   ├── api/                 # HTTP layer (Fiber)
│   │   ├── router.go        # Route registration
│   │   ├── middleware/       # Auth, CORS, rate limiting, request ID, logging
│   │   ├── handlers/        # Route handlers grouped by resource
│   │   │   ├── auth.go
│   │   │   ├── domain.go
│   │   │   ├── dns.go
│   │   │   ├── ssl.go
│   │   │   ├── php.go
│   │   │   ├── file.go
│   │   │   ├── database.go
│   │   │   ├── ftp.go
│   │   │   ├── backup.go
│   │   │   ├── log.go
│   │   │   └── settings.go
│   │   └── dto/             # Request/Response structs + validation tags
│   ├── core/                # Business logic (pure Go, no framework deps)
│   │   ├── domain/          # Domain CRUD, vhost generation
│   │   ├── dns/             # DNS zone management
│   │   ├── ssl/             # Certificate issuance, renewal, ACME
│   │   ├── php/             # PHP version management, pool config
│   │   ├── webserver/       # NGINX/Apache config generation + validation
│   │   ├── filemanager/     # File operations (scoped to user dirs)
│   │   ├── database/        # MySQL DB/user management
│   │   ├── ftp/             # FTP account management
│   │   ├── backup/          # Backup creation, restore, scheduling
│   │   └── monitor/         # System metrics collection
│   ├── agent/               # System agent (privileged operations)
│   │   ├── server.go        # Unix socket JSON-RPC server
│   │   ├── client.go        # Client used by core to call agent
│   │   ├── commands.go      # Command allowlist + executors
│   │   └── validate.go      # Path/argument validation
│   ├── db/                  # Database layer
│   │   ├── sqlite.go        # SQLite connection setup (WAL, busy timeout, pool)
│   │   ├── migrations/      # SQL migration files (embedded)
│   │   ├── queries/         # SQL query functions grouped by table
│   │   └── models/          # Data models (Go structs)
│   ├── config/              # Viper configuration loading
│   │   └── config.go
│   ├── template/            # Config template rendering engine
│   │   └── render.go        # Load + execute Go templates, atomic write
│   ├── auth/                # JWT token generation, validation, refresh
│   │   └── jwt.go
│   ├── websocket/           # WebSocket hub for real-time updates
│   │   └── hub.go
│   └── logger/              # Zerolog setup, structured logging
│       └── logger.go
├── web/                     # Frontend (React + Vite)
│   ├── src/
│   │   ├── components/      # Shadcn/ui components + custom components
│   │   │   ├── ui/          # Shadcn base components
│   │   │   ├── layout/      # Shell, sidebar, header, footer
│   │   │   └── shared/      # Reusable domain-specific components
│   │   ├── pages/           # Page components (one per route)
│   │   │   ├── dashboard/
│   │   │   ├── domains/
│   │   │   ├── databases/
│   │   │   ├── files/
│   │   │   ├── backups/
│   │   │   ├── logs/
│   │   │   └── settings/
│   │   ├── hooks/           # Custom React hooks
│   │   ├── api/             # API client + TanStack Query hooks per resource
│   │   ├── stores/          # Zustand stores (UI state only)
│   │   ├── lib/             # Utility functions
│   │   └── types/           # TypeScript types (mirroring backend DTOs)
│   ├── public/              # Static assets
│   ├── index.html
│   ├── package.json
│   ├── tsconfig.json
│   ├── tailwind.config.ts
│   └── vite.config.ts
├── configs/
│   └── templates/           # Go templates for service configs (embedded)
│       ├── nginx/
│       ├── php/
│       ├── dns/
│       ├── mail/
│       ├── ftp/
│       └── systemd/
├── scripts/
│   ├── install.sh           # One-line installer script
│   ├── uninstall.sh         # Clean uninstall
│   └── dev-setup.sh         # Dev environment setup
├── test/
│   ├── e2e/                 # Playwright E2E tests
│   ├── fixtures/            # Test data fixtures
│   └── testutil/            # Shared test helpers (Go)
├── .github/
│   └── workflows/
│       ├── ci.yml           # PR checks: lint + test + build
│       └── release.yml      # Release: build + package + publish
├── .golangci.yml            # Linter config
├── .air.toml                # Hot-reload config
├── lefthook.yml             # Git hooks config
├── Makefile
├── go.mod
├── go.sum
└── README.md
```

---

## Key Dependencies

### Go
| Package | Purpose |
|---------|---------|
| `gofiber/fiber/v2` | HTTP framework |
| `gofiber/contrib/websocket` | WebSocket via Fiber |
| `modernc.org/sqlite` | SQLite driver (pure Go, no CGO) |
| `redis/go-redis/v9` | Redis client |
| `spf13/cobra` | CLI framework |
| `spf13/viper` | Configuration management |
| `golang-jwt/jwt/v5` | JWT authentication |
| `go-playground/validator/v10` | Struct validation |
| `rs/zerolog` | Structured logging |
| `natefinch/lumberjack` | Log rotation |
| `golang-migrate/migrate/v4` | Database migrations |
| `robfig/cron/v3` | Cron scheduler |
| `go-acme/lego/v4` | Let's Encrypt ACME client |
| `shirou/gopsutil/v3` | System metrics (CPU, RAM, disk) |
| `google/uuid` | UUID generation |
| `cosmtrek/air` | Hot-reload (dev only) |

### Frontend (npm)
| Package | Purpose |
|---------|---------|
| `react` + `react-dom` | UI framework |
| `typescript` | Type safety |
| `@tanstack/react-query` | Server state management |
| `react-router-dom` v6 | Routing |
| `tailwindcss` v4 | Styling |
| `shadcn/ui` | Component library (copy-paste) |
| `zustand` | Client state (UI only) |
| `recharts` | Charts for monitoring |
| `@tanstack/react-table` | Data tables (domains, users, etc.) |
| `lucide-react` | Icons |
| `axios` | HTTP client |
| `zod` | Schema validation |
| `date-fns` | Date formatting |
| `react-hot-toast` / `sonner` | Toast notifications |
| `@codemirror/view` | Code editor (file manager) |
| `react-dropzone` | File upload drag-and-drop |

---

## Build & Deployment

### Single Binary Build
```bash
# Build frontend → outputs to web/dist/
cd web && npm ci && npm run build

# Embed frontend into Go binary & compile
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w -X main.version=$(git describe --tags)" -o dist/pinkpanel ./cmd/server
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o dist/pinkpanel-agent ./cmd/agent
```

The frontend is embedded using Go's `embed` package — the final output is **two binaries**:
1. `pinkpanel` — main panel + embedded frontend (serves SPA on `/` and API on `/api`)
2. `pinkpanel-agent` — privileged system agent (runs as root)

The CLI is built into the main `pinkpanel` binary as subcommands (`pinkpanel cli domain add ...` or symlinked as `pinkpanel-cli`).

### Cross-Compilation
- `linux/amd64` — primary target (most VPS)
- `linux/arm64` — ARM servers (AWS Graviton, Oracle Ampere, Raspberry Pi)
- No CGO means easy cross-compilation from any OS

### Installation Target
```bash
/usr/local/pinkpanel/
├── bin/
│   ├── pinkpanel              # Main binary (non-root)
│   └── pinkpanel-agent        # System agent (root)
├── data/
│   ├── pinkpanel.db           # SQLite database
│   ├── .jwt-secret            # JWT signing key (0600)
│   ├── .agent-secret          # Agent auth secret (0600)
│   └── backups/               # Local backup storage
├── ssl/
│   ├── panel.crt              # Panel self-signed or LE cert
│   └── panel.key              # Panel private key
├── logs/
│   ├── panel.log              # Application log
│   ├── audit.log              # Audit trail
│   └── agent.log              # Agent log
└── pinkpanel.yml              # Panel configuration
```

### Systemd Services
```ini
# /etc/systemd/system/pinkpanel.service
[Unit]
Description=PinkPanel Web Hosting Control Panel
After=network.target redis.service
Wants=pinkpanel-agent.service

[Service]
Type=simple
User=pinkpanel
Group=pinkpanel
ExecStart=/usr/local/pinkpanel/bin/pinkpanel serve
WorkingDirectory=/usr/local/pinkpanel
Restart=on-failure
RestartSec=5
LimitNOFILE=65535

[Install]
WantedBy=multi-user.target
```

```ini
# /etc/systemd/system/pinkpanel-agent.service
[Unit]
Description=PinkPanel System Agent
After=network.target

[Service]
Type=simple
User=root
ExecStart=/usr/local/pinkpanel/bin/pinkpanel-agent
WorkingDirectory=/usr/local/pinkpanel
Restart=on-failure
RestartSec=5

[Install]
WantedBy=multi-user.target
```

---

## Security Architecture

### Process Isolation
- Panel runs as unprivileged user (`pinkpanel`)
- Agent runs as `root` — only process with elevated privileges
- Communication exclusively via Unix socket (not network-exposed)
- Socket permissions: `0660`, owned by `root:pinkpanel`

### Authentication & Authorization
- JWT-based authentication with access + refresh tokens
- Access token: 15min TTL, stored in memory (not localStorage)
- Refresh token: 7 day TTL, stored in HttpOnly secure cookie
- Bcrypt password hashing (cost 12)
- All API endpoints require authentication (except `POST /api/auth/login` and `POST /api/auth/refresh`)

### Request Security
- CSRF protection via `SameSite=Strict` cookies + custom header check
- Rate limiting: 5 login attempts/minute, 100 API requests/minute (configurable)
- Input validation on all API inputs (go-playground/validator)
- SQL parameterization everywhere (no string concatenation)
- Request ID on every request for traceability

### Response Security
- Content Security Policy (CSP) headers
- X-Content-Type-Options: nosniff
- X-Frame-Options: DENY
- Strict-Transport-Security (HSTS) when SSL enabled
- No sensitive data in error responses (logged server-side only)

### Data Security
- Secrets (JWT key, agent secret) stored as files with `0600` permissions
- Encrypted fields in SQLite where needed (AES-256-GCM)
- Database file permissions: `0640`, owned by `pinkpanel:pinkpanel`
- Backup files encrypted at rest (optional, configurable)

### Agent Security
- Strict command allowlist — no arbitrary command execution
- Path validation: agent rejects paths outside predefined directories
- Argument sanitization: no shell metacharacters allowed
- Rate limiting on agent requests (prevent abuse from compromised panel)
- All agent operations logged to separate audit log

---

## Graceful Shutdown & Signal Handling

- `SIGTERM` / `SIGINT` → graceful shutdown
  1. Stop accepting new HTTP connections
  2. Wait for active requests to complete (30s timeout)
  3. Close WebSocket connections with close frame
  4. Flush pending metrics to SQLite
  5. Close database connections
  6. Close agent socket connection
  7. Exit 0
- `SIGHUP` → reload configuration (non-critical settings only)
- `SIGUSR1` → rotate log files

---

## Health Check & Self-Diagnostics

- `GET /api/health` — public endpoint (no auth)
  - Returns: panel status, agent connectivity, database status, Redis status
  - Used by monitoring and load balancers
- `GET /api/health/detailed` — authenticated, admin only
  - Returns: all service statuses, disk space, memory, uptime, version
- Agent heartbeat: panel pings agent every 30s, alerts if unreachable
- Database integrity check on startup (`PRAGMA integrity_check`)
- Self-repair: if agent socket disappears, attempt reconnection with backoff

---

## Versioning & Updates

### Semantic Versioning
- `MAJOR.MINOR.PATCH` (e.g., `1.2.3`)
- Version embedded at compile time via `-ldflags`
- Displayed in UI footer, `GET /api/health`, CLI `pinkpanel version`

### Update Mechanism (Future)
- `pinkpanel update check` — check for new version
- `pinkpanel update apply` — download + replace binary + restart service
- Binary signature verification before applying update
- Rollback: keep previous binary as `.bak`, restore if new version fails health check

---

## Supported Operating Systems

### Primary (MVP)
- **Ubuntu 22.04 LTS**
- **Ubuntu 24.04 LTS**
- **Debian 12 (Bookworm)**

### Secondary (V1)
- **Debian 11 (Bullseye)**
- **AlmaLinux 9**
- **Rocky Linux 9**

### Why Not Others
- CentOS Stream: not stable enough for hosting
- Fedora: too fast-moving
- Windows: completely different service management, deferred indefinitely

---

## Implementation Phases

V0 is divided into 4 sub-phases to be implemented sequentially:

### V0.1 — Project Scaffolding & Build Pipeline
_Get the project compiling and a blank page rendering._

1. Initialize Go module (`go mod init github.com/pinkpanel/pinkpanel`)
2. Create `cmd/server/main.go` — Fiber server that serves a "PinkPanel is running" page
3. Create `cmd/agent/main.go` — minimal agent that listens on Unix socket
4. Scaffold React app with Vite + TypeScript + Tailwind + Shadcn/ui
5. Configure Go `embed` to serve the built frontend
6. Set up Makefile with `dev`, `build`, `test`, `lint`, `fmt` targets
7. Set up `air` for Go hot-reload
8. Set up Vite proxy to Go backend in dev mode
9. Create `.golangci.yml` linter config
10. Create `lefthook.yml` for pre-commit hooks
11. Create GitHub Actions CI workflow (lint + test + build)
12. Create `.gitignore`
13. Build and verify: `make build` produces two working binaries

**Deliverable**: Running skeleton — `make dev` starts both backend and frontend with hot reload, `make build` outputs two binaries.

### V0.2 — Database, Config, & Logging Foundation
_Core infrastructure that everything else depends on._

1. Set up Viper configuration loading (`internal/config/config.go`)
2. Create `pinkpanel.yml` default config with all sections
3. Set up SQLite connection with WAL mode, busy timeout, connection pool (`internal/db/sqlite.go`)
4. Set up `golang-migrate` migration system with embed
5. Create first migration: `000001_create_settings.up.sql` (key-value settings table)
6. Set up zerolog structured logging (`internal/logger/logger.go`)
7. Set up lumberjack log rotation
8. Add request logging middleware (method, path, status, duration, request_id)
9. Add request ID middleware (UUID per request)
10. Set up Redis connection with fallback to in-memory (`internal/cache/`)
11. Create health check endpoint (`GET /api/health`)
12. Implement graceful shutdown with signal handling
13. Write tests for config loading, database setup, and health check

**Deliverable**: Server starts, reads config, connects to SQLite, logs structured JSON, responds to health checks, shuts down gracefully.

### V0.3 — Authentication & Agent Communication
_Secure access to the panel and privileged operations._

1. Create admin migration: `000002_create_admins.up.sql`
2. Implement JWT auth (`internal/auth/jwt.go`): generate, validate, refresh
3. Implement auth middleware: extract token, validate, inject user into context
4. Create `POST /api/auth/login` — validate credentials, return access + refresh tokens
5. Create `POST /api/auth/refresh` — issue new access token from refresh token
6. Create `POST /api/auth/logout` — invalidate refresh token
7. Add rate limiting middleware for auth endpoints
8. Implement agent Unix socket server (`internal/agent/server.go`)
9. Implement agent client (`internal/agent/client.go`)
10. Implement agent command allowlist with argument validation (`internal/agent/commands.go`)
11. Create first agent command: `service_status` (returns status of NGINX, etc.)
12. Create agent heartbeat (panel pings agent every 30s)
13. Add security headers middleware (CSP, HSTS, X-Frame-Options, etc.)
14. Create admin setup flow: first-run detection → create admin account
15. Write tests for auth flow, agent communication, rate limiting

**Deliverable**: Login works, JWT tokens issued, agent responds to commands, security headers present. Initial admin can be created on first run.

### V0.4 — Frontend Shell & API Client
_The panel UI skeleton that all features plug into._

1. Set up React Router with layout routes
2. Create app shell: sidebar navigation, header (user info, logout), main content area
3. Build sidebar with navigation links (Dashboard, Domains, Databases, Files, Backups, Logs, Settings)
4. Create login page with form, validation, error handling
5. Set up Axios instance with JWT interceptor (auto-attach token, auto-refresh on 401)
6. Set up TanStack Query provider with default config (retry, refetch, stale time)
7. Set up Zustand store for UI state (sidebar collapsed, theme)
8. Create auth API hooks (`useLogin`, `useLogout`, `useRefreshToken`)
9. Create protected route wrapper (redirect to login if unauthenticated)
10. Create dashboard page placeholder (server status cards — data from `GET /api/health/detailed`)
11. Create settings page (change password, panel settings)
12. Set up toast notification system (sonner)
13. Set up error boundary
14. Create reusable data table component (TanStack Table) for domain lists, etc.
15. Create reusable form components (input, select, switch, textarea) with zod validation
16. Implement dark/light theme toggle (Tailwind dark mode)
17. Responsive layout (sidebar collapses on mobile)
18. Write Playwright E2E test: login flow

**Deliverable**: Full panel shell with working login, navigation, dashboard placeholder, settings page, dark mode. Ready for feature pages to be added in MVP.

---

## V0 Completion Criteria

After V0 is complete, the following must all be true:

- [ ] `make dev` starts backend + frontend with hot reload
- [ ] `make build` produces `pinkpanel` and `pinkpanel-agent` binaries
- [ ] `make test` passes all unit + integration tests
- [ ] `make lint` passes with zero warnings
- [ ] Panel serves on `https://localhost:8443`
- [ ] Login/logout works with JWT authentication
- [ ] Agent is reachable via Unix socket and responds to `service_status`
- [ ] SQLite database created with migrations applied
- [ ] Structured JSON logging to file with rotation
- [ ] Health check endpoint returns 200 with component statuses
- [ ] Graceful shutdown works (no dropped requests)
- [ ] CI pipeline passes (lint + test + build)
- [ ] Frontend shell renders with sidebar, header, dark mode
- [ ] E2E test for login flow passes
