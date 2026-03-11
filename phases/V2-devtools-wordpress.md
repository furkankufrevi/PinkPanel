# V2 - Developer Tools & WordPress

_Attract developers and WordPress users with Git deployment, multiple runtimes, Docker, API, and a full WordPress management toolkit._

**Depends on**: V1 complete

---

## 1. WordPress Toolkit

### One-Click Installation
- Install WordPress with a single click
- Auto-create database, DB user, and configure `wp-config.php`
- Choose WP version (latest stable by default)
- Choose language, site title, admin username/email during install
- Auto-configure NGINX rewrite rules for pretty permalinks
- Auto-set file permissions (755 dirs, 644 files)

### WordPress Dashboard
- Unified view of ALL WordPress installations across the server
- Per-site status: WP version, PHP version, SSL status, update available
- Quick actions: login to wp-admin, open site, update, backup
- Filter/search by domain, user, WP version, status
- Bulk select for mass operations

### Instance Detection
- Auto-scan document roots for existing WordPress installations
- Detect WP version, active theme, active plugins
- Import unmanaged instances into toolkit

### Plugin Management
- List installed plugins per site (name, version, status, update available)
- Activate/deactivate plugins
- Install plugins from WordPress.org repository (search + install)
- Update individual or all plugins
- Delete plugins
- **Mass operations**: update/activate/deactivate a plugin across ALL sites

### Theme Management
- List installed themes per site
- Activate theme
- Install themes from WordPress.org repository
- Update themes
- Delete unused themes
- **Mass operations**: update a theme across all sites

### Security Hardening (One-Click)
- Block directory browsing
- Block access to `wp-config.php`
- Block `xmlrpc.php` (with option to keep enabled)
- Block PHP execution in `wp-content/uploads`
- Change default admin username warning
- Enforce strong passwords
- Hide WordPress version
- Block author enumeration
- Security scan: check all hardening items and report status

### WordPress Updates
- One-click core update per site
- Mass core update across all sites
- Auto-update option: enable automatic updates for core/plugins/themes
- Update notifications in dashboard
- **Pre-update backup**: auto-create backup before any update

### WP-CLI Integration
- Execute WP-CLI commands per site from panel UI
- Common command shortcuts: cache flush, rewrite flush, search-replace
- Command history per site

---

## 2. WordPress Staging & Cloning

### Staging Environment
- Create staging copy of any WordPress site
- Staging runs on a subdomain: `staging.domain.com`
- Separate database for staging (cloned from production)
- Staging environment auto-configured with:
  - `WP_DEBUG = true`
  - Search engine noindex
  - Robots.txt blocking
- **Push to production**: selective sync (files only, DB only, or both)
- Diff view before pushing (show changed files)

### Cloning
- Clone WordPress site to a new domain/subdomain
- Auto-update URLs in database (search-replace)
- Auto-update `wp-config.php` for new database
- Option to clone with or without uploads directory

### Sync
- Selective sync between any two WordPress instances
- Sync options: files, database, plugins, themes, uploads
- Direction: staging → production, production → staging, site → site
- Dry-run mode (preview changes)

---

## 3. Git Integration

### Repository Setup
- Initialize a local Git repository for any domain
- Or link to a remote repository (GitHub, GitLab, Bitbucket, any Git URL)
- SSH key generation for remote repository authentication
- Deploy key management

### Push-to-Deploy
- Webhook URL per domain for push-triggered deployments
- Auto-deploy on push to configured branch (default: `main`)
- Deployment log (commit hash, timestamp, status, duration)

### Pull Deployment
- Manual pull from remote repository
- Branch/tag selection
- Auto-run post-deploy commands (e.g., `composer install`, `npm build`)

### Deploy Configuration
- `.pinkpanel-deploy.yml` file in repo root:
  ```yaml
  branch: main
  document_root: public/   # subdirectory as webroot
  pre_deploy:
    - composer install --no-dev
  post_deploy:
    - php artisan migrate --force
    - php artisan cache:clear
  exclude:
    - .env
    - storage/
    - node_modules/
  ```
- Environment variables management per domain
- Deploy hooks: pre-deploy and post-deploy commands
- Rollback to previous deployment

### Deployment History
- List of all deployments (commit, branch, timestamp, status)
- View deployment log output
- One-click rollback to any previous deployment

---

## 4. Additional Runtime Support

### Node.js
- Install multiple Node.js versions (via nvm or system packages)
- Per-domain Node.js version selection
- Application entry point configuration (`app.js`, `server.js`, etc.)
- **Process manager**: start/stop/restart Node.js apps
- NGINX reverse proxy auto-configuration to Node.js port
- NPM/Yarn package management from UI
- Environment variables editor
- Application logs viewer
- PM2 integration for process management

### Python
- Python version management per domain
- Virtual environment auto-creation
- WSGI support (Gunicorn) with NGINX reverse proxy
- ASGI support (Uvicorn) for async frameworks
- `requirements.txt` / `pip install` from UI
- Django/Flask detection and auto-configuration
- Environment variables editor
- Application logs viewer

### Ruby
- Ruby version management per domain
- Bundler integration
- Passenger or Puma as application server
- NGINX reverse proxy configuration
- `Gemfile` management from UI
- Rails detection and auto-configuration

---

## 5. Docker Support

### Container Management
- Deploy Docker containers linked to domains
- Container lifecycle: create, start, stop, restart, remove
- Container logs viewer (real-time tail)
- Container resource stats (CPU, RAM, network)

### Configuration
- Port mapping (container port → NGINX reverse proxy)
- Volume mounts (persist data)
- Environment variables
- Docker Compose file upload and deploy
- Container auto-restart policy

### Registry
- Pull images from Docker Hub
- Pull from private registries (credentials management)
- List local images
- Remove unused images

---

## 6. CLI Tool

### Command Structure
```
pinkpanel [resource] [action] [flags]
```

### Available Commands
```bash
# Server
pinkpanel status                    # Server overview
pinkpanel services list             # List service statuses
pinkpanel services restart nginx    # Restart a service

# Domains
pinkpanel domain list
pinkpanel domain add example.com
pinkpanel domain delete example.com
pinkpanel domain suspend example.com
pinkpanel domain info example.com

# Users
pinkpanel user list
pinkpanel user add --email user@example.com --plan basic
pinkpanel user delete user@example.com

# Databases
pinkpanel db list
pinkpanel db create --name mydb --domain example.com
pinkpanel db delete mydb

# Email
pinkpanel email list --domain example.com
pinkpanel email add user@example.com
pinkpanel email delete user@example.com

# SSL
pinkpanel ssl issue example.com
pinkpanel ssl renew example.com
pinkpanel ssl status example.com

# Backup
pinkpanel backup create --domain example.com
pinkpanel backup list
pinkpanel backup restore backup_2024_01_01.tar.gz

# WordPress
pinkpanel wp list
pinkpanel wp install --domain example.com
pinkpanel wp update --domain example.com
pinkpanel wp plugin list --domain example.com
pinkpanel wp plugin update --all --domain example.com

# PHP
pinkpanel php versions
pinkpanel php set --domain example.com --version 8.3
```

### CLI Features
- Output formats: table (default), JSON, YAML
- `--quiet` flag for scripting
- Autocomplete for bash/zsh/fish
- Config file at `~/.pinkpanel-cli.yml` (API URL, saved credentials)
- All commands talk to the REST API (same backend as web UI)

---

## 7. REST API

### API Features
- Full REST API covering all panel operations
- JWT authentication (same as web panel)
- API key authentication (for server-to-server / scripts)
- Rate limiting per API key (configurable)
- OpenAPI 3.0 specification auto-generated
- Swagger UI at `/api/docs`

### API Key Management
- Generate API keys per user
- Set permissions per API key (read-only, full access, custom scopes)
- Set IP whitelist per API key
- Key expiration dates
- Usage logging per key
- Revoke keys

### Webhook System
- Register webhooks for events:
  - Domain created/deleted/suspended
  - User created/deleted
  - Backup completed/failed
  - SSL issued/expiring
  - WordPress update available
- Webhook delivery with retry (3 attempts, exponential backoff)
- Webhook delivery log (payload, response, status)
- Webhook secret for signature verification (HMAC-SHA256)

---

## 8. File Management Enhancements

### SSH Access
- Enable/disable SSH per user (from service plan)
- SSH key management per user (add/remove public keys)
- Shell selection: bash, sh, restricted shell
- SSH connection info display (host, port, username)

### SFTP
- SFTP access using same credentials as SSH
- SFTP-only mode (no shell access)
- Connection info and quick-connect instructions

### Enhanced File Manager
- Mass upload (multiple files, drag-and-drop)
- File search within domain files (name, content)
- Syntax highlighting for 30+ languages in editor
- Archive extraction (zip, tar, tar.gz, tar.bz2)
- Create archives from selected files
- Image preview thumbnails
- File/folder size calculation

---

## 9. Database Enhancements

### PostgreSQL Support
- Install and manage PostgreSQL
- Create PostgreSQL databases per domain
- PostgreSQL user management
- phpPgAdmin integration with SSO
- Backup/restore PostgreSQL databases

### Advanced Operations
- Move database between users
- Database size monitoring with growth trends
- Slow query log viewer (MySQL)
- Table-level operations: repair, optimize, check
- Remote database access toggle (bind address management)
- Connection string generator (copy-paste ready for apps)
- Import SQL file via UI upload

---

## 10. Monitoring

### Health Monitor Dashboard
- Real-time graphs (CPU, RAM, disk I/O, network I/O)
- Historical data: 1 hour, 24 hours, 7 days, 30 days
- Per-service resource usage (NGINX, PHP-FPM, MySQL, Postfix)
- Disk usage breakdown by: domains, databases, email, backups, system

### Service Monitoring
- Service status checks every 60 seconds
- Auto-restart failed services (configurable per service)
- Service uptime tracking
- Notification on service failure

### Alerting
- Email notifications for:
  - CPU above threshold for N minutes
  - RAM above threshold
  - Disk usage above threshold
  - Service down
  - SSL certificate expiring (30, 14, 7, 1 day)
  - Backup failure
- Alert cooldown (don't spam repeated alerts)
- Alert history log

### Per-Domain Stats
- Bandwidth usage per domain (daily/monthly)
- Request count per domain
- Error rate per domain (4xx, 5xx)
- Response time averages

---

## 11. Security Enhancements

### Firewall Management
- UI for managing iptables/nftables rules
- Default deny policy with managed allowlist
- Quick rules: allow/deny IP, allow/deny port, allow/deny country
- Predefined rule sets: web server, mail server, DNS server
- Rule ordering (priority)
- Emergency lockout recovery (auto-whitelist admin IP)

### IP Management
- Global IP whitelist/blacklist
- Per-domain IP restrictions
- Country-based blocking (GeoIP)
- View all blocked IPs across Fail2ban + firewall

### Password-Protected Directories
- Set HTTP Basic Auth on any directory
- Manage users/passwords for protected areas
- Per-domain protected directories list

### Hotlink Protection
- Enable per domain
- Whitelist allowed referrers
- Custom response for blocked hotlinks (403 or redirect to image)

---

## New API Endpoints (V2 additions)

### WordPress
- `GET /api/wp/instances` — List all WP instances
- `POST /api/wp/install` — Install WordPress
- `GET /api/wp/:id` — WP instance detail
- `DELETE /api/wp/:id` — Remove WP instance
- `POST /api/wp/:id/update` — Update WordPress core
- `GET /api/wp/:id/plugins` — List plugins
- `POST /api/wp/:id/plugins/install` — Install plugin
- `PUT /api/wp/:id/plugins/:name` — Activate/deactivate/update plugin
- `GET /api/wp/:id/themes` — List themes
- `POST /api/wp/:id/staging` — Create staging
- `POST /api/wp/:id/staging/push` — Push staging to production
- `POST /api/wp/:id/clone` — Clone site
- `POST /api/wp/:id/harden` — Apply security hardening
- `POST /api/wp/:id/wpcli` — Execute WP-CLI command

### Git
- `POST /api/domains/:id/git/init` — Initialize repo
- `POST /api/domains/:id/git/remote` — Link remote repo
- `POST /api/domains/:id/git/deploy` — Trigger deployment
- `GET /api/domains/:id/git/deployments` — Deployment history
- `POST /api/domains/:id/git/rollback` — Rollback

### Docker
- `GET /api/docker/containers` — List containers
- `POST /api/docker/containers` — Deploy container
- `PUT /api/docker/containers/:id` — Start/stop/restart
- `GET /api/docker/containers/:id/logs` — Container logs
- `GET /api/docker/images` — List images

### API Keys
- `CRUD /api/keys` — API key management

### Monitoring
- `GET /api/monitoring/server` — Server metrics
- `GET /api/monitoring/services` — Service status
- `GET /api/monitoring/domains/:id/stats` — Per-domain stats
- `GET /api/monitoring/alerts` — Alert history
- `PUT /api/monitoring/alerts/config` — Alert configuration

### Firewall
- `GET /api/firewall/rules` — List rules
- `POST /api/firewall/rules` — Add rule
- `DELETE /api/firewall/rules/:id` — Remove rule

---

## New Database Tables (V2 additions)

- `wp_instances` — id, domain_id, path, version, admin_url, status, detected_at
- `wp_plugins` — id, wp_instance_id, slug, name, version, status, update_available
- `wp_themes` — id, wp_instance_id, slug, name, version, active, update_available
- `wp_staging` — id, wp_instance_id, staging_domain, staging_db, created_at
- `git_repos` — id, domain_id, remote_url, branch, deploy_key_id, auto_deploy, created_at
- `git_deployments` — id, git_repo_id, commit_hash, branch, status, log, started_at, finished_at
- `docker_containers` — id, domain_id, image, container_id, ports (JSON), volumes (JSON), env (JSON), status
- `api_keys` — id, user_id, name, key_hash, permissions (JSON), ip_whitelist, expires_at, last_used_at
- `webhooks` — id, user_id, url, events (JSON), secret, active, created_at
- `webhook_deliveries` — id, webhook_id, event, payload, response_status, response_body, delivered_at
- `monitoring_metrics` — timestamp, metric_name, value (time-series, partitioned by day)
- `alerts` — id, type, threshold, cooldown_minutes, email, enabled
- `alert_history` — id, alert_id, triggered_at, value, notified
- `firewall_rules` — id, action (allow/deny), protocol, port, source_ip, direction, priority, comment
- `node_apps` — id, domain_id, entry_point, node_version, port, status, env (JSON)
- `python_apps` — id, domain_id, entry_point, python_version, port, framework, status, env (JSON)
