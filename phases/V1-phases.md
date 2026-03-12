# V1 - Multi-User & Email: Phased Implementation

_Each phase is self-contained, independently testable, and builds on the previous._

**Current state**: Alpha (0.1.0-alpha) — single-admin, no user scoping, no email.

---

## Phase 1: Alpha Polish (v0.2.0-alpha)

_Finish deferred MVP items before adding major new features. Clean foundation._

### 1A. Let's Encrypt ACME Integration
- Integrate `go-acme/lego` library for automatic SSL issuance
- New agent commands: `ssl_issue_letsencrypt` (HTTP-01 challenge)
- New handler: `POST /api/domains/:id/ssl/issue` — triggers ACME flow
- Auto-configure NGINX for `.well-known/acme-challenge`
- Auto-renewal: background goroutine checks certs expiring within 30 days
- Store ACME account key in `/etc/pinkpanel/acme/`

### 1B. Scheduled Backups
- New DB table: `backup_schedules` (id, domain_id nullable, frequency, time, retention_count, enabled)
- CRUD handler: `/api/backup-schedules`
- Background scheduler goroutine (checks every minute, runs due backups)
- Retention policy: auto-delete oldest when exceeding `retention_count`
- UI: schedule card on backups page

### 1C. Dashboard Real-Time Metrics
- WebSocket endpoint: `WS /api/dashboard/live`
- Push CPU, RAM, disk, load every 5 seconds via agent `system_info` command
- Frontend: live-updating gauges on dashboard

### 1D. Minor Enhancements
- **phpinfo viewer**: New endpoint `GET /api/domains/:id/php/info` → agent runs `php -i`
- **Log download**: `GET /api/domains/:id/logs/download?source=access` → streams log file
- **Domain aliases**: `aliases` JSON column on domains table, NGINX server_name includes aliases
- **Parking page**: Default index.html template placed in new domain document roots

### Tests
- [ ] `POST /api/domains/:id/ssl/issue` issues cert and configures NGINX HTTPS
- [ ] Auto-renewal picks up certs expiring in <30 days
- [ ] Backup schedule triggers at configured time
- [ ] Retention deletes oldest backup when count exceeded
- [ ] WebSocket pushes metrics every 5s
- [ ] phpinfo returns PHP configuration output
- [ ] Log download streams file with Content-Disposition
- [ ] Domain with alias responds on both names

---

## Phase 2: Multi-User Foundation (v1.0.0-beta)

_The biggest architectural change. Every resource becomes user-scoped._

### 2A. Database Schema Changes
- Rename `admins` → keep as-is but add `role` column (`super_admin`, `admin`, `user`)
- Add `admin_id` FK to: `domains`, `databases`, `ftp_accounts`, `backups`, `backup_schedules`
- Add `system_username` column to `admins` (for Linux user isolation)
- New table: `sessions` (id, admin_id, token_hash, ip, user_agent, created_at, expires_at)
- New table: `login_attempts` (id, email, ip, success, created_at)
- Migration: existing resources assigned to admin ID 1 (the original super admin)

### 2B. User CRUD
- New service: `internal/core/user/service.go`
- New handler: `internal/api/handlers/user.go`
- Endpoints: `GET/POST /api/users`, `GET/PUT/DELETE /api/users/:id`, `POST /api/users/:id/suspend`, `POST /api/users/:id/activate`
- On user create: create Linux system user via agent, set up home directory `/home/{username}/domains/`
- On user delete: remove Linux user, optionally remove all data
- Password change, email update per user

### 2C. Auth Middleware Scoping
- Expand JWT claims with `Role` field
- New middleware helper: `RequireRole("super_admin", "admin")` for admin-only endpoints
- All resource handlers check ownership: `WHERE admin_id = ?`
- Super admin bypasses ownership checks (sees everything)
- Admin role: can manage users they created + those users' resources
- User role: own resources only

### 2D. System User Isolation
- Agent commands: `user_create` (adduser, set shell, create dirs), `user_delete` (userdel -r)
- Document roots move to `/home/{username}/domains/{domain}/public`
- PHP-FPM pools run as user's system user (not www-data)
- File manager scoped to user's home directory
- FTP accounts chrooted to user's home

### 2E. Frontend: User Management UI
- New page: `/users` — list, create, edit, delete users (admin only)
- User detail page with resource summary
- Scoped sidebar: users see only their menu items
- Login page: works for all roles
- Profile page: change own password, email

### 2F. Session & Login Security
- Track login attempts (IP, success/fail, timestamp)
- Account lockout after 10 failed attempts (30 min cooldown)
- Active sessions list: `GET /api/auth/sessions`
- Revoke session: `DELETE /api/auth/sessions/:id`
- Show active sessions in settings page

### Tests
- [ ] Super admin can create/list/delete users
- [ ] User can only see own domains, databases, FTP accounts
- [ ] Admin can see resources of users they manage
- [ ] Super admin sees everything
- [ ] Creating user creates Linux system user + home directory
- [ ] Deleting user cleans up system user and optionally data
- [ ] New domain for user creates docroot under `/home/{username}/domains/`
- [ ] PHP-FPM pool runs as user's system user
- [ ] Login lockout triggers after 10 failed attempts
- [ ] Session revocation invalidates that token
- [ ] Existing single-admin setup migrates cleanly (admin_id=1 owns all)

---

## Phase 3: Service Plans & Quotas (v1.1.0-beta)

_Resource limits, usage tracking, and plan-based access control._

### 3A. Plans
- New table: `plans` (id, name, description, limits JSON, features JSON, is_default, created_at)
- Limits JSON: `{ max_domains, max_subdomains, max_databases, max_email_accounts, max_ftp_accounts, disk_quota_mb, bandwidth_gb }`
- Features JSON: `{ php, ssh, ssl, cron, dns_editor, backups, file_manager }`
- CRUD: `GET/POST /api/plans`, `GET/PUT/DELETE /api/plans/:id`
- Default plan assigned to new users
- Cannot delete plan with active users

### 3B. Subscriptions
- New table: `subscriptions` (id, admin_id, plan_id, status, started_at, expires_at)
- Add `plan_id` FK to `admins` table (or use subscriptions join)
- Endpoints: `POST /api/subscriptions`, `PUT /api/subscriptions/:id`
- Suspend subscription → suspends all user's domains
- Upgrade/downgrade: limits adjust immediately

### 3C. Resource Limit Enforcement
- Middleware/helper: `CheckLimit(adminID, "domains")` before creating resources
- Returns `403 quota_exceeded` with details when limit hit
- Checks on: domain create, database create, FTP create, email create (Phase 5)
- Disk quota: agent command `quota_set` using Linux `setquota`
- Feature checks: hide/disable UI elements based on plan features

### 3D. Usage Tracking
- New table: `usage_snapshots` (id, admin_id, disk_bytes, bandwidth_bytes, snapshot_date)
- Background job: daily disk usage scan per user (du -s)
- Bandwidth: parse NGINX access logs per domain, aggregate daily
- Endpoints: `GET /api/users/:id/usage`, `GET /api/usage/summary` (admin)
- Warnings at 80% and 95% of limits (stored as notifications)

### 3E. Frontend: Plans & Usage UI
- Admin pages: plan list, plan editor, user plan assignment
- User dashboard: usage bars (domains used/max, disk used/quota, bandwidth)
- Disable "Create" buttons when at limit with tooltip explaining why

### Tests
- [ ] Create plan with limits → assign to user
- [ ] User hits domain limit → gets 403 with clear message
- [ ] Disk quota enforced at OS level
- [ ] Usage snapshot records daily disk/bandwidth
- [ ] Upgrade plan → user can create more resources immediately
- [ ] Suspend subscription → all user domains suspended
- [ ] Feature toggle hides DNS editor for users without dns_editor permission
- [ ] Default plan auto-assigned to new users

---

## Phase 4: Security Hardening (v1.2.0-beta)

_2FA, Fail2ban, and ModSecurity._

### 4A. Two-Factor Authentication (TOTP)
- New columns on `admins`: `totp_secret`, `totp_enabled`, `recovery_codes` (JSON)
- Endpoints:
  - `POST /api/auth/2fa/setup` → generates secret, returns QR code URI
  - `POST /api/auth/2fa/verify` → verifies code, enables 2FA, returns recovery codes
  - `DELETE /api/auth/2fa` → disables 2FA (requires current password)
- Login flow: if 2FA enabled, first step returns `requires_2fa: true`, second step sends TOTP code
- Recovery codes: 10 one-time-use codes, regeneratable
- Admin can enforce 2FA for all users (plan feature or global setting)
- Admin can reset user's 2FA

### 4B. Fail2ban Integration
- Agent commands: `fail2ban_status`, `fail2ban_banned`, `fail2ban_ban`, `fail2ban_unban`, `fail2ban_jail_status`
- Auto-configure jails: sshd, pinkpanel-auth (custom filter for panel login failures), postfix, dovecot
- Custom Fail2ban filter: `/etc/fail2ban/filter.d/pinkpanel.conf` watching panel logs
- Endpoints:
  - `GET /api/security/fail2ban/status` — overall + per-jail status
  - `GET /api/security/fail2ban/banned` — currently banned IPs
  - `POST /api/security/fail2ban/ban` — manual ban
  - `POST /api/security/fail2ban/unban` — manual unban
- IP whitelist for admin IPs (never ban)

### 4C. ModSecurity WAF (Optional)
- Agent commands: `modsec_enable`, `modsec_disable`, `modsec_status`, `modsec_log`
- Install OWASP CRS rule set
- Per-domain toggle: `PUT /api/domains/:id/modsec`
- Audit log viewer: `GET /api/security/modsec/log` with filtering
- Rule exception management for false positives

### 4D. Frontend: Security UI
- Settings → Security section: 2FA setup with QR code
- New page: `/security` (admin only) — Fail2ban dashboard, banned IPs, ModSecurity logs
- Login page: 2FA code step

### Tests
- [ ] 2FA setup generates valid QR code
- [ ] Login with 2FA requires TOTP code on second step
- [ ] Invalid TOTP code rejected
- [ ] Recovery code works once, then invalidated
- [ ] Admin can reset user's 2FA
- [ ] Fail2ban status shows jail information
- [ ] Manual ban/unban works from UI
- [ ] Panel login failures trigger Fail2ban after threshold
- [ ] Whitelisted IPs never banned
- [ ] ModSecurity blocks XSS/SQLi attempts on enabled domains
- [ ] ModSecurity audit log shows blocked requests

---

## Phase 5: Email Core (v1.3.0-beta)

_Postfix + Dovecot mail server with account management._

### 5A. Mail Server Setup (Agent)
- Agent commands:
  - `mail_setup` — installs/configures Postfix + Dovecot (virtual mailbox setup)
  - `mail_create_account` — creates maildir, sets password (doveadm pw)
  - `mail_delete_account` — removes maildir
  - `mail_set_quota` — Dovecot quota plugin per account
  - `mail_set_password` — updates password
  - `mail_domain_add` — adds domain to virtual_mailbox_domains
  - `mail_domain_remove` — removes domain from virtual mail config
- Postfix config: virtual mailbox transport, TLS, submission port (587)
- Dovecot config: IMAP + POP3, maildir format, `/var/mail/vhosts/{domain}/{user}/`
- Auto-configure on domain create: add to virtual domains, create MX/SPF DNS records

### 5B. Email Account CRUD
- New table: `email_accounts` (id, domain_id, admin_id, local_part, password_hash, quota_mb, status, created_at)
- Service: `internal/core/email/service.go`
- Handler: `internal/api/handlers/email.go`
- Endpoints:
  - `GET /api/domains/:id/email/accounts` — list accounts
  - `POST /api/domains/:id/email/accounts` — create account
  - `GET /api/email/:id` — account detail (usage, quota)
  - `PUT /api/email/:id/password` — change password
  - `PUT /api/email/:id/quota` — set quota
  - `DELETE /api/email/:id` — delete account
- On create: agent `mail_create_account`, auto MX record if first email on domain
- On delete: option to keep/delete mail data

### 5C. Forwarding & Aliases
- New table: `email_forwarders` (id, domain_id, source, destination, keep_copy)
- New table: `email_aliases` (id, domain_id, alias_local, target_account_id)
- Endpoints:
  - `CRUD /api/domains/:id/email/forwarders`
  - `CRUD /api/domains/:id/email/aliases`
  - `PUT /api/domains/:id/email/catchall` — set catch-all address
- Agent writes Postfix virtual alias maps, reloads

### 5D. Frontend: Email Management UI
- New sidebar section: "Email" (under domain detail)
- Email accounts list with quota usage bars
- Create/edit/delete account dialogs
- Forwarding rules table
- Aliases table
- Catch-all toggle

### Tests
- [ ] Postfix + Dovecot install and start successfully via agent
- [ ] Create email account → can send/receive test email
- [ ] Delete account → maildir removed
- [ ] Quota enforcement: reject mail when mailbox full
- [ ] Password change → new password works for IMAP login
- [ ] MX + SPF DNS records auto-created on first email account
- [ ] Forwarding: email to source@domain arrives at destination
- [ ] Alias: email to alias@domain delivers to target account
- [ ] Catch-all: unmatched addresses route to catch-all mailbox
- [ ] User scoping: users only see their domain's email accounts

---

## Phase 6: Email Security & Webmail (v1.4.0-beta)

_DKIM/DMARC, spam filtering, antivirus, and Roundcube._

### 6A. DKIM
- Agent command: `dkim_generate` — generates 2048-bit RSA key pair per domain
- New table: `dkim_keys` (id, domain_id, selector, private_key_path, public_key, created_at)
- Auto-add DKIM TXT record to DNS: `{selector}._domainkey.{domain}`
- Configure OpenDKIM or Postfix milter to sign outgoing mail
- Endpoint: `GET /api/domains/:id/email/dkim`, `POST /api/domains/:id/email/dkim/generate`
- Key rotation: generate new key with new selector, keep old for transition

### 6B. SPF & DMARC
- Auto SPF record: `v=spf1 a mx ip4:{server_ip} ~all` (already partially done in DNS)
- Auto DMARC record: `v=DMARC1; p=none; rua=mailto:postmaster@{domain}`
- DMARC policy editor: none → quarantine → reject progression
- Endpoints for viewing/editing SPF and DMARC records (via existing DNS API)

### 6C. SpamAssassin
- Agent commands: `spamassassin_setup`, `spamassassin_enable`, `spamassassin_disable`
- Per-domain spam settings: enable/disable, score threshold, action (mark/move/delete)
- New table: `spam_settings` (id, domain_id, enabled, threshold, action)
- Whitelist/blacklist per domain
- Endpoint: `GET/PUT /api/domains/:id/email/spam`

### 6D. ClamAV
- Agent commands: `clamav_setup`, `clamav_status`
- Integrate with Postfix (amavisd-new or clamav-milter)
- Action on virus: reject at SMTP level
- Auto-update virus definitions

### 6E. Roundcube Webmail
- Agent command: `roundcube_install` — installs Roundcube, configures NGINX vhost
- Accessible at `webmail.{domain}` or panel subdomain
- Auto-configure IMAP/SMTP connection settings
- SSO: `POST /api/email/:id/webmail-token` → generates one-time login token
- Panel UI: "Open Webmail" button per email account

### 6F. Mail Queue Management
- Agent command: `mail_queue` — reads Postfix queue (mailq)
- Endpoints: `GET /api/email/queue`, `POST /api/email/queue/flush`, `DELETE /api/email/queue/:id`
- UI: mail queue viewer with flush/delete actions

### 6G. Mail Autodiscovery
- Auto-create SRV records for mail autodiscovery
- Agent: set up `autoconfig.{domain}` XML endpoint (Thunderbird)
- Agent: set up `autodiscover.{domain}` XML endpoint (Outlook)

### Tests
- [ ] DKIM signing: outgoing mail has valid DKIM signature (verify with external tool)
- [ ] DKIM DNS record resolves correctly
- [ ] SPF record present and valid
- [ ] DMARC record present
- [ ] SpamAssassin marks high-score mail as spam
- [ ] SpamAssassin respects per-domain threshold
- [ ] ClamAV rejects mail with EICAR test virus
- [ ] Roundcube accessible and functional
- [ ] SSO token logs into Roundcube without password
- [ ] Mail queue shows stuck messages, flush works
- [ ] Autodiscovery: Thunderbird auto-detects server settings

---

## Phase 7: Enhancements & Cron (v1.5.0)

_DNS/PHP/Backup enhancements and cron job management. V1 release._

### 7A. DNS Enhancements
- Additional record types: SRV, CAA
- DNS templates: predefined record sets (basic hosting, hosting+email, Google Workspace)
- Apply template on domain creation (selectable)
- Template CRUD: `GET/POST /api/dns/templates`, `GET/PUT/DELETE /api/dns/templates/:id`

### 7B. PHP Enhancements
- **phpinfo viewer**: `GET /api/domains/:id/php/info` (from Phase 1 if not done)
- **Extensions list**: `GET /api/php/:version/extensions` — agent reads `php -m`
- **Toggle extensions**: `PUT /api/php/:version/extensions/:ext` — agent enables/disables via ini
- **Composer**: agent commands `composer_install`, `composer_update` per domain
- Endpoint: `POST /api/domains/:id/composer/:action`

### 7C. Backup Enhancements
- **Incremental backups**: rsync-based, only changed files since last full
- Agent command: `backup_incremental` using rsync + hard links
- **Selective restore**: `GET /api/backups/:id/contents` — list archive contents
- Restore specific paths: `POST /api/backups/:id/restore` with `{ paths: [...] }` body
- **Per-user backups**: users can create/restore own backups (if plan allows)
- Backup size counts against user's disk quota

### 7D. Cron Job Management
- New table: `cron_jobs` (id, admin_id, domain_id, schedule, command, enabled, last_run, last_status, last_output, created_at)
- Agent commands: `cron_set` (writes user crontab), `cron_remove`, `cron_list`
- Endpoints: `GET/POST /api/domains/:id/cron`, `GET/PUT/DELETE /api/cron/:id`
- Visual schedule builder in frontend (minute/hour/day/month/weekday pickers)
- Common presets: every minute, 5 min, hourly, daily, weekly, monthly
- Max cron jobs per user (from service plan)
- Run history: last N runs with exit code and output
- Lock files to prevent overlap (flock)

### 7E. Frontend: Enhancement UIs
- DNS template selector on domain creation
- PHP extensions toggle page
- Composer actions in file manager context menu
- Backup contents browser + selective restore dialog
- Cron jobs page with visual schedule builder

### Tests
- [ ] SRV and CAA records create and resolve correctly
- [ ] DNS template applies full record set on domain creation
- [ ] PHP extension toggle enables/disables module
- [ ] Composer install runs in domain directory
- [ ] Incremental backup only transfers changed files
- [ ] Selective restore restores chosen paths only
- [ ] Cron job executes at scheduled time
- [ ] Cron run history shows output and exit code
- [ ] Plan limit on cron jobs enforced
- [ ] Lock file prevents concurrent cron execution

---

## Summary

| Phase | Version | Focus | Key Deliverable |
|-------|---------|-------|-----------------|
| 1 | 0.2.0-alpha | Alpha Polish | Let's Encrypt, scheduled backups, WebSocket dashboard |
| 2 | 1.0.0-beta | Multi-User | User isolation, roles, scoped resources |
| 3 | 1.1.0-beta | Plans & Quotas | Service plans, resource limits, usage tracking |
| 4 | 1.2.0-beta | Security | 2FA, Fail2ban, ModSecurity |
| 5 | 1.3.0-beta | Email Core | Postfix/Dovecot, email accounts, forwarding |
| 6 | 1.4.0-beta | Email Advanced | DKIM/DMARC, spam, antivirus, Roundcube |
| 7 | 1.5.0 | Enhancements | DNS/PHP/Backup upgrades, cron jobs → **V1 Release** |

## Architecture Notes

**Critical for Phase 2**: Adding `admin_id` to all resource tables is a breaking migration. The migration must:
1. Add nullable `admin_id` column
2. Set all existing rows to `admin_id = 1` (original admin)
3. Add NOT NULL constraint
4. Add foreign key + index

**File path migration (Phase 2)**: Existing domains use `/var/www/{domain}/`. New user-scoped domains use `/home/{username}/domains/{domain}/public`. The migration should NOT move existing domains — they keep their current paths. Only new domains created by non-super-admin users use the new path structure.

**JWT backward compatibility (Phase 2)**: Old tokens without `Role` claim should be treated as `super_admin` during transition. Add version field to claims if needed.
