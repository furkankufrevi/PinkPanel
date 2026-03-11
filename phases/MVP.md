# MVP - Core Foundation

_The bare minimum to manage a single server with websites. One admin user, basic domain hosting, SSL, and file management._

---

## 1. Authentication & Dashboard

### Admin Authentication
- Single admin account (created during installation)
- Email + password login
- JWT access token (15min) + refresh token (7 days)
- Secure password hashing (bcrypt)
- Session invalidation on password change
- Login rate limiting (5 attempts per minute, then lockout)

### Dashboard
- **Server Overview Widget**: CPU usage (%), RAM usage (used/total), disk usage per partition, uptime
- **Quick Stats**: total domains, total sites, total databases
- **Service Status**: NGINX (running/stopped), PHP-FPM (running/stopped), MySQL (running/stopped), FTP (running/stopped)
- **Recent Activity**: last 10 actions (domain added, backup created, etc.)
- Real-time updates via WebSocket (CPU/RAM refresh every 5s)

### Settings
- Change admin password
- Change admin email
- Panel port configuration (default: 8443)
- Panel hostname/SSL configuration
- Timezone setting

---

## 2. Domain Management

### Add Domain
- Input: domain name (validated format)
- Auto-creates: NGINX vhost, document root (`/var/www/{domain}/public`), DNS zone, FTP account
- Sets ownership to `www-data` (or per-domain system user in V1)
- Option: create `www` subdomain automatically

### Domain Operations
- **List all domains** with status indicators (active, suspended, SSL status)
- **Suspend/Activate** domain (disables NGINX vhost, shows suspended page)
- **Delete domain** (removes vhost, optionally removes files, DNS, databases)
- **Domain detail page**: shows disk usage, bandwidth, linked databases, SSL status

### Subdomain Management
- Add subdomains under any managed domain
- Separate document root per subdomain
- Independent SSL per subdomain

### Domain Aliases & Redirects
- Add alias domains pointing to a primary domain
- 301/302 redirects between domains
- Wildcard subdomain support

### Parking Page
- Default "coming soon" page for new/inactive domains
- Customizable parking page template

---

## 3. DNS Management

### Zone Editor
- Auto-create zone file on domain creation with sensible defaults
- Edit records via UI: A, AAAA, CNAME, MX, TXT, NS
- TTL configuration per record
- Validate record values before saving

### DNS Server
- **BIND9** as DNS server (or PowerDNS as alternative)
- Auto-reload zones on change
- SOA record auto-management (serial increments)

### Default Records
On domain creation, auto-add:
- `A` record → server IP
- `AAAA` record → server IPv6 (if available)
- `MX` record → server hostname (placeholder for V1 email)
- `NS` records → server nameservers
- `SOA` record with defaults

---

## 4. Web Server Management

### NGINX Configuration
- Auto-generate vhost per domain from templates
- Reverse proxy to Apache (for .htaccess compatibility) or standalone mode
- Gzip compression enabled by default
- Security headers (X-Frame-Options, X-Content-Type-Options, etc.)
- Per-domain NGINX config override (advanced editor)
- HTTP → HTTPS redirect (auto-enabled when SSL is active)

### Apache Configuration (Optional Backend)
- mod_php or PHP-FPM proxy
- .htaccess support
- Per-domain Apache config

### Operations
- Restart/reload NGINX from panel
- Restart/reload Apache from panel
- Configuration syntax validation before applying
- Rollback to previous config on failure

---

## 5. PHP Management

### PHP-FPM
- Install and manage PHP-FPM
- Per-domain PHP enable/disable
- PHP version selection (whatever is installed on server)
- Pool-per-domain configuration

### PHP Settings Editor
- Edit key php.ini directives per domain via UI:
  - `upload_max_filesize`
  - `post_max_size`
  - `max_execution_time`
  - `memory_limit`
  - `display_errors`
  - `error_reporting`
  - `date.timezone`
- Apply changes without full server restart (FPM pool reload)

### PHP Info
- View `phpinfo()` output per domain
- Show installed PHP extensions

---

## 6. SSL/TLS Management

### Let's Encrypt Integration
- One-click SSL issuance per domain (using `go-acme/lego`)
- HTTP-01 challenge (auto-configured)
- Auto-include `www` subdomain in certificate
- Wildcard certificate support via DNS-01 challenge

### Certificate Operations
- **Issue** new certificate
- **Renew** certificate (auto-renewal via cron, 30 days before expiry)
- **Revoke** certificate
- **View** certificate details (issuer, expiry, SANs)
- Upload custom certificate (key + cert + chain)

### Auto-Configuration
- On successful issuance: auto-update NGINX vhost for HTTPS
- Auto-enable HTTP → HTTPS redirect
- Auto-enable HTTP/2
- HSTS header option

---

## 7. File Manager

### Web-Based File Manager
- Browse directory tree starting from domain document root
- **Operations**: create file/folder, rename, delete, move, copy, chmod
- **Upload**: drag-and-drop file upload (chunked for large files)
- **Download**: single file or zip folder download
- **Edit**: in-browser text editor with syntax highlighting (CodeMirror)
- **Extract**: unzip/untar archives in-place

### Permissions
- Display file owner, group, permissions
- Change permissions (chmod) via UI
- Change ownership (only within allowed scope)

### FTP
- Create FTP accounts per domain
- Set FTP home directory
- FTP quota per account
- **vsftpd** or **ProFTPD** as FTP server
- FTPS (FTP over TLS) support

---

## 8. Database Management

### MySQL/MariaDB
- Create databases with auto-generated names (`domain_dbname`)
- Create database users with password generation
- Assign user permissions per database (read, write, all)
- Delete databases and users
- Show database size

### phpMyAdmin
- Integrated phpMyAdmin access
- Single sign-on from panel (no separate login)
- Per-domain access restriction

### Operations
- **List** all databases with sizes and linked domains
- **Backup** individual database (mysqldump via UI)
- **Restore** database from SQL file
- **Repair** and **optimize** tables

---

## 9. Backup & Restore

### Full Backup
- Backup entire server (all domains, databases, configs, email)
- Stored in `/usr/local/pinkpanel/data/backups/`
- Compressed (tar.gz)

### Per-Site Backup
- Backup individual domain (files + databases + config)
- One-click from domain detail page

### Restore
- Full restore from backup file
- Selective restore: choose files, databases, or config separately
- Restore preview (show what will be restored)

### Management
- List all backups with size and date
- Delete old backups
- Manual backup trigger via UI
- Backup download to local machine

---

## 10. Log Viewer

### Web Server Logs
- View NGINX access logs per domain (last N lines, with pagination)
- View NGINX error logs per domain
- View Apache access/error logs (if enabled)
- Real-time log tailing via WebSocket
- Log filtering: by date, status code, IP, URL pattern

### System Logs
- Panel application logs
- System agent logs
- Service logs (FPM, MySQL, FTP)

### Log Management
- Log rotation configuration
- Log retention settings
- Download log files

---

## 11. System Agent (Root Operations)

### Agent Capabilities
The system agent handles all privileged operations. Each operation has an explicit allowlist entry:

- **Service control**: start/stop/restart/reload NGINX, Apache, PHP-FPM, MySQL, vsftpd, BIND
- **Config management**: write NGINX vhosts, PHP pool configs, DNS zone files, FTP configs
- **User management**: create/delete system users for domains
- **File operations**: set ownership, create document roots
- **Package management**: install/remove PHP versions, check installed packages
- **System info**: read CPU, RAM, disk, network stats
- **SSL**: write certificate files, configure NGINX SSL
- **Backup**: create/restore system-level backups

### Communication Protocol
- Unix socket at `/var/run/pinkpanel/agent.sock`
- JSON-RPC protocol
- Request authentication via shared secret (file-based)
- All operations logged with timestamp and caller

---

## 12. Installation Script

### One-Line Installer
```bash
curl -fsSL https://get.pinkpanel.com | bash
```

### Installer Steps
1. Check OS compatibility (Ubuntu 22.04/24.04, Debian 11/12)
2. Check minimum requirements (1 CPU, 1GB RAM, 10GB disk)
3. Install system packages: NGINX, PHP-FPM, MariaDB, BIND9, vsftpd, Redis
4. Create `pinkpanel` system user
5. Download and place binaries
6. Run database migrations (SQLite)
7. Create admin account (interactive prompt for email + password)
8. Configure systemd services
9. Configure firewall (allow 8443, 80, 443, 21, 53)
10. Start services
11. Print access URL and credentials

---

## API Endpoints (MVP)

### Auth
- `POST /api/auth/login` — Login
- `POST /api/auth/refresh` — Refresh token
- `POST /api/auth/logout` — Logout

### Dashboard
- `GET /api/dashboard/stats` — Server stats
- `GET /api/dashboard/services` — Service statuses
- `WS /api/dashboard/live` — Real-time metrics

### Domains
- `GET /api/domains` — List domains
- `POST /api/domains` — Create domain
- `GET /api/domains/:id` — Domain detail
- `PUT /api/domains/:id` — Update domain
- `DELETE /api/domains/:id` — Delete domain
- `POST /api/domains/:id/suspend` — Suspend
- `POST /api/domains/:id/activate` — Activate

### Subdomains
- `GET /api/domains/:id/subdomains` — List subdomains
- `POST /api/domains/:id/subdomains` — Create subdomain
- `DELETE /api/subdomains/:id` — Delete subdomain

### DNS
- `GET /api/domains/:id/dns` — List DNS records
- `POST /api/domains/:id/dns` — Add record
- `PUT /api/dns/:id` — Update record
- `DELETE /api/dns/:id` — Delete record

### SSL
- `GET /api/domains/:id/ssl` — SSL status
- `POST /api/domains/:id/ssl/issue` — Issue Let's Encrypt
- `POST /api/domains/:id/ssl/upload` — Upload custom cert
- `DELETE /api/domains/:id/ssl` — Remove SSL

### PHP
- `GET /api/php/versions` — Installed PHP versions
- `GET /api/domains/:id/php` — Domain PHP settings
- `PUT /api/domains/:id/php` — Update PHP settings

### Files
- `GET /api/domains/:id/files?path=` — List directory
- `POST /api/domains/:id/files` — Create file/folder
- `PUT /api/domains/:id/files` — Rename/move
- `DELETE /api/domains/:id/files` — Delete
- `POST /api/domains/:id/files/upload` — Upload
- `GET /api/domains/:id/files/download` — Download
- `GET /api/domains/:id/files/content` — Read file content
- `PUT /api/domains/:id/files/content` — Write file content

### Databases
- `GET /api/databases` — List all databases
- `POST /api/databases` — Create database
- `DELETE /api/databases/:id` — Delete database
- `GET /api/databases/:id/users` — List DB users
- `POST /api/databases/:id/users` — Create DB user
- `DELETE /api/database-users/:id` — Delete DB user
- `POST /api/databases/:id/backup` — Backup database
- `POST /api/databases/:id/restore` — Restore database

### FTP
- `GET /api/domains/:id/ftp` — List FTP accounts
- `POST /api/domains/:id/ftp` — Create FTP account
- `DELETE /api/ftp/:id` — Delete FTP account

### Backups
- `GET /api/backups` — List backups
- `POST /api/backups` — Create backup
- `POST /api/backups/:id/restore` — Restore backup
- `DELETE /api/backups/:id` — Delete backup
- `GET /api/backups/:id/download` — Download backup

### Logs
- `GET /api/domains/:id/logs/access` — Access logs
- `GET /api/domains/:id/logs/error` — Error logs
- `WS /api/domains/:id/logs/tail` — Real-time log tail

### Settings
- `GET /api/settings` — Panel settings
- `PUT /api/settings` — Update settings
- `PUT /api/settings/password` — Change password

---

## Database Schema (MVP)

### Tables
- `admins` — id, email, password_hash, created_at, updated_at
- `domains` — id, name, document_root, status (active/suspended), php_version, created_at
- `subdomains` — id, domain_id, name, document_root, created_at
- `dns_records` — id, domain_id, type, name, value, ttl, priority
- `ssl_certificates` — id, domain_id, type (letsencrypt/custom), cert_path, key_path, expires_at
- `databases` — id, domain_id, name, type (mysql), size_bytes, created_at
- `database_users` — id, database_id, username, host, permissions
- `ftp_accounts` — id, domain_id, username, home_dir, quota_mb
- `backups` — id, domain_id (nullable for full), type, file_path, size_bytes, created_at
- `activity_log` — id, admin_id, action, target_type, target_id, details, created_at
- `settings` — key, value
