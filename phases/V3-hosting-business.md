# V3 - Hosting Business & Resellers

_Enable hosting providers to run a full business. Reseller accounts, white-labeling, migrations, advanced backups, and business reporting._

**Depends on**: V2 complete

---

## 1. Reseller Management

### Reseller Accounts
- Admin creates reseller accounts
- Reseller is a special user role with its own resource pool
- Reseller can create their own customers (sub-users)
- Reseller customers cannot see or interact with other resellers' customers
- Hierarchical: Admin → Reseller → Customer → Domains

### Reseller Resource Allocation
- Admin allocates resources to reseller:
  - Max disk space (total across all their customers)
  - Max bandwidth (total across all their customers)
  - Max domains
  - Max customers
  - Max databases
  - Max email accounts
- Reseller distributes their allocated resources to customers via plans
- Over-allocation prevention: reseller cannot assign more than they have
- Overselling option: admin can allow resellers to oversell by X% (configurable)

### Reseller Service Plans
- Resellers create their own service plans (within their allocated resources)
- Reseller plans are independent of admin's plans
- Reseller can set pricing metadata (for display/export, panel doesn't handle billing)
- Reseller can clone/modify plans

### Reseller Dashboard
- Reseller sees: their customers, their resource usage, their domains
- Quick stats: total customers, total domains, resource utilization (used/allocated)
- Customer management: create, suspend, delete, change plan
- Cannot access server settings, other resellers, or admin functions

### Reseller Operations
- Suspend reseller → suspends all their customers and domains
- Delete reseller → must reassign or delete all their customers first
- Change reseller allocation → validates against current usage
- Reseller usage reports (how much of their allocation is used)

---

## 2. White-Label & Branding

### Panel Branding
- Custom panel name (replace "PinkPanel" everywhere)
- Custom logo upload (header logo, login page logo, favicon)
- Custom color scheme (primary color, accent color, sidebar color)
- Custom login page background
- Custom footer text
- Hide "Powered by PinkPanel" (optional)

### Per-Reseller Branding
- Each reseller can set their own branding for their customers
- Reseller's customers see the reseller's branding, not PinkPanel's
- Custom panel URL per reseller (e.g., `panel.resellerdomain.com`)
- SSL for custom panel domains

### Custom Skins
- Skin system with CSS variable overrides
- Pre-built skin templates (light, dark, blue, green, etc.)
- Skin editor in panel (live preview)
- Export/import skins

### Email Branding
- Custom "From" address for panel notification emails
- Custom email templates (welcome email, password reset, alerts)
- Per-reseller email branding

---

## 3. Multiple Interface Views

### Power User View
- Single-user focused interface
- All domains in a flat list
- Direct access to all features
- Best for: personal use, small agencies

### Service Provider View
- Multi-tenant focused interface
- Customers listed first, then their domains
- Resource allocation and plan management prominent
- Reseller hierarchy visible
- Best for: hosting companies

### User View (Customer)
- Simplified interface for end customers
- Only shows their domains and services
- No server-level settings visible
- Clean, non-technical language where possible

---

## 4. Advanced Backup

### Remote Storage Backends
- **Amazon S3** (+ S3-compatible: MinIO, Wasabi, Backblaze B2)
  - Bucket configuration
  - Region selection
  - IAM credentials
  - Storage class selection (Standard, Infrequent Access, Glacier)
- **Google Drive**
  - OAuth2 authentication
  - Folder selection
- **Dropbox**
  - OAuth2 authentication
  - Folder selection
- **Custom FTP/SFTP**
  - Host, port, credentials
  - Remote directory
- **Local directory** (different mount/drive)

### Backup Encryption
- AES-256-GCM encryption for all backups
- Encryption key management (per-server key, stored securely)
- Option: user-provided encryption passphrase
- Encrypted backups can only be restored with the key

### Backup Retention Policies
- Per-schedule retention: keep last N daily, N weekly, N monthly
- Example: 7 daily + 4 weekly + 3 monthly
- Auto-delete expired backups (local and remote)
- Minimum retention enforcement (cannot set below 1)

### Backup Reporting
- Backup success/failure history
- Backup size trends over time
- Storage usage by backup type
- Failed backup alerts (email notification)
- Backup verification (periodic integrity check)

### Backup Improvements
- Parallel backup streams (backup multiple sites simultaneously)
- Bandwidth throttling for remote uploads
- Resume interrupted remote uploads
- Backup exclusion patterns (skip `node_modules`, `.git`, cache dirs)

---

## 5. Advanced Monitoring & Grafana

### Grafana Integration
- Embedded Grafana dashboards in panel
- Pre-built dashboards:
  - Server overview (CPU, RAM, disk, network, load average)
  - Web server metrics (requests/sec, response times, status codes)
  - PHP-FPM metrics (active processes, idle, queue)
  - MySQL metrics (queries/sec, connections, slow queries, buffer usage)
  - Email metrics (sent, received, bounced, spam blocked)
  - Per-domain traffic
- Custom time range selection
- Dashboard auto-refresh

### Metrics Collection
- Lightweight metrics agent (built into PinkPanel)
- Metrics stored in time-series format (SQLite or optional InfluxDB)
- Retention: 1-minute granularity for 24h, 5-min for 7 days, 1-hour for 90 days
- Exportable metrics (Prometheus format endpoint)

### Per-Site Analytics
- Requests per day/hour
- Bandwidth per day/hour
- Top URLs by request count
- Top URLs by bandwidth
- HTTP status code distribution
- Top referrers
- Top user agents (browser/bot breakdown)
- Geographic distribution (GeoIP)

---

## 6. Reporting

### Usage Reports
- Resource usage by: reseller, customer, domain
- Date range selection (daily, weekly, monthly, custom)
- Exportable as CSV and PDF

### Report Types
- **Disk Usage Report**: breakdown by files, databases, email, backups per entity
- **Bandwidth Report**: traffic per domain, daily/monthly totals
- **Email Report**: sent/received/bounced/spam per domain
- **Security Report**: blocked attacks, banned IPs, WAF events
- **Backup Report**: backup history, success rate, storage used
- **WordPress Report**: sites needing updates, security issues

### Audit Logging
- Every admin/reseller action logged with:
  - Who (user ID, username)
  - What (action, resource type, resource ID)
  - When (timestamp)
  - Where (IP address, user agent)
  - Details (before/after values for changes)
- Audit log viewer with filtering
- Audit log export
- Retention: configurable (default 90 days)

---

## 7. Email Enhancements

### Mailing Lists
- Create mailing lists per domain
- Powered by Mailman 3 or simple alias-based lists
- Subscribe/unsubscribe management
- Moderation options
- Archive access

### Advanced Anti-Spam
- **Greylisting** support (temporary rejection of unknown senders)
- Per-domain whitelist/blacklist with UI management
- **SRS** (Sender Rewriting Scheme) for forwarded email
- SpamAssassin Bayesian learning (mark as spam/not spam from webmail)
- Spam quarantine viewer (review and release quarantined messages)

### Mail Client Auto-Configuration
- `autoconfig.domain.com` for Thunderbird auto-configuration
- `autodiscover.domain.com` for Outlook auto-configuration
- SRV records for generic auto-discovery
- Mobile device (iOS/Android) configuration profile generator

### Email Deliverability Tools
- DKIM/SPF/DMARC status checker per domain
- MX record validation
- Blacklist check (is server IP on any RBLs?)
- Test email sender (send test and check delivery)
- Email reputation monitoring

---

## 8. WordPress Toolkit (Advanced)

### Smart Updates
- Before applying update: clone site → apply update to clone → run visual regression test
- Compare screenshots of key pages before/after update
- Report: safe to update / issues detected
- Admin reviews and approves/rejects update
- If approved, apply update to production with auto-backup

### Mass Operations
- Update WordPress core across ALL sites (or selected subset)
- Update a specific plugin across ALL sites that have it
- Apply security hardening to ALL sites
- Generate security report for all WordPress instances

### Remote WordPress Management
- Add external WordPress sites (on other servers) to the toolkit
- Manage plugins/themes/updates for remote sites
- Requires installing a small plugin on the remote WordPress
- Central dashboard for all WP sites (local + remote)

### WordPress Performance
- NGINX FastCGI cache tuned for WordPress
- Cache purge on content update (via WP plugin hook)
- Object cache recommendation (Redis/Memcached)
- Performance score per site (page load time, TTFB)

---

## 9. Migration Tools

### Migration Wizard
- Step-by-step wizard UI for migrations
- Source panel detection (cPanel, DirectAdmin, Plesk, PinkPanel)
- Connection: SSH credentials or API key for source server
- Discovery: scan source server for accounts, domains, databases, email
- Selection: choose what to migrate (all or specific accounts)
- Preview: show what will be created on target
- Execute: migrate with progress bar
- Verification: post-migration checks (sites loading, email working, DNS info)

### cPanel Migration
- Parse cPanel backup files (`.tar.gz` from cPanel backup)
- Map cPanel accounts → PinkPanel users
- Migrate: domains, subdomains, databases, database users, email accounts, email data, FTP accounts, cron jobs, SSL certificates, DNS zones
- Handle cPanel-specific paths and configurations

### DirectAdmin Migration
- Parse DirectAdmin backup format
- Map DA users → PinkPanel users
- Migrate same items as cPanel

### PinkPanel-to-PinkPanel Migration
- Server-to-server migration via API
- Live migration with minimal downtime
- Sync files via rsync, databases via dump/restore
- DNS cutover guidance

### Bulk Migration
- Queue multiple accounts for migration
- Run migrations in parallel (configurable concurrency)
- Migration progress dashboard
- Per-account status: pending, in progress, completed, failed
- Retry failed migrations
- Migration log per account

---

## 10. DNSSEC

### DNSSEC Management
- Generate DNSSEC keys per domain (KSK + ZSK)
- Sign DNS zones
- DS record display (for registrar configuration)
- Key rollover support
- Unsign zones (disable DNSSEC)
- DNSSEC validation status check

---

## 11. Additional OS Support

### New Platforms
- **AlmaLinux 8/9**
- **Rocky Linux 8/9**
- Automated installer adapted for RPM-based distros
- Package mapping: apt → dnf equivalents
- SELinux policy configuration (permissive or custom policies)

---

## New API Endpoints (V3 additions)

### Resellers
- `CRUD /api/resellers` — Reseller account management
- `GET /api/resellers/:id/customers` — List reseller's customers
- `GET /api/resellers/:id/usage` — Reseller resource usage
- `PUT /api/resellers/:id/allocations` — Set resource allocations

### Branding
- `GET /api/branding` — Current branding settings
- `PUT /api/branding` — Update branding
- `POST /api/branding/logo` — Upload logo
- `GET /api/resellers/:id/branding` — Reseller branding
- `PUT /api/resellers/:id/branding` — Update reseller branding

### Advanced Backup
- `GET /api/backup/storages` — List configured storage backends
- `POST /api/backup/storages` — Add storage backend
- `PUT /api/backup/storages/:id` — Update storage backend
- `DELETE /api/backup/storages/:id` — Remove storage backend
- `GET /api/backup/schedules` — List backup schedules
- `POST /api/backup/schedules` — Create backup schedule
- `PUT /api/backup/retention` — Set retention policy

### Reports
- `GET /api/reports/usage` — Usage report (with date range, grouping)
- `GET /api/reports/bandwidth` — Bandwidth report
- `GET /api/reports/email` — Email report
- `GET /api/reports/security` — Security report
- `GET /api/reports/audit` — Audit log
- `GET /api/reports/export` — Export report (CSV/PDF)

### Migration
- `POST /api/migration/scan` — Scan source server
- `POST /api/migration/start` — Start migration
- `GET /api/migration/status` — Migration status
- `GET /api/migration/history` — Migration history
- `POST /api/migration/import` — Import backup file

### Monitoring
- `GET /api/monitoring/grafana/dashboards` — Available dashboards
- `GET /api/monitoring/analytics/:domain` — Per-domain analytics
- `GET /api/monitoring/metrics` — Raw metrics (Prometheus format)

---

## New Database Tables (V3 additions)

- `resellers` — id, user_id, allocations (JSON), oversell_percent, branding (JSON), created_at
- `reseller_plans` — id, reseller_id, name, limits (JSON), features (JSON)
- `branding` — id, entity_type (global/reseller), entity_id, name, logo_path, colors (JSON), custom_css
- `backup_storages` — id, type (s3/gdrive/dropbox/ftp), name, credentials (encrypted JSON), config (JSON)
- `backup_schedules` — id, storage_id, frequency, time, scope (full/domain), retention (JSON), last_run
- `audit_log` — id, user_id, action, resource_type, resource_id, before (JSON), after (JSON), ip, user_agent, timestamp
- `migrations` — id, source_type (cpanel/directadmin/pinkpanel), source_host, status, accounts (JSON), started_at, finished_at
- `migration_items` — id, migration_id, account_name, item_type, status, log, started_at, finished_at
- `analytics_daily` — domain_id, date, requests, bandwidth, errors_4xx, errors_5xx
- `analytics_urls` — domain_id, date, url, hits, bandwidth
- `mailing_lists` — id, domain_id, name, address, members (JSON), moderated
- `dnssec_keys` — id, domain_id, key_type (KSK/ZSK), algorithm, public_key, private_key_path, created_at, active
