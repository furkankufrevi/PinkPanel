# V1 - Multi-User & Email

_Transform from single-admin to multi-tenant. Add email hosting, service plans, and proper security._

**Depends on**: MVP complete

---

## 1. User & Account Management

### User System
- Admin creates user accounts (email + password)
- Users log in to their own scoped dashboard
- Users can only see/manage their own domains, databases, email, files
- User profile: change password, update email, timezone preference

### Roles & Permissions
- **Super Admin**: full server access, manage all users, server settings
- **Admin**: manage assigned users and their resources
- **User**: manage own domains and services only
- Granular permissions per user:
  - Can create domains (yes/no, max count)
  - Can create databases (yes/no, max count)
  - Can create email accounts (yes/no, max count)
  - Can access SSH/FTP (yes/no)
  - Can create backups (yes/no)
  - Can manage DNS (yes/no)
  - Can manage PHP settings (yes/no)

### System User Isolation
- Each panel user gets a dedicated Linux system user
- Document roots under `/home/{username}/domains/{domain}/public`
- PHP-FPM pools run as the user's system user
- File permission isolation between users
- No cross-user file access

---

## 2. Service Plans

### Plan Configuration
- **Name** and description
- **Resource Limits**:
  - Max domains
  - Max subdomains
  - Max databases
  - Max email accounts
  - Max FTP accounts
  - Disk space quota (MB)
  - Bandwidth limit (GB/month)
  - Max mailbox size (MB)
- **Feature Toggles**:
  - PHP access (yes/no)
  - SSH access (yes/no)
  - SSL management (yes/no)
  - Cron jobs (yes/no, max count)
  - DNS editor access (yes/no)
  - Backup access (yes/no)
  - File manager access (yes/no)

### Plan Operations
- Create/edit/delete plans
- Clone plan as template
- View users assigned to a plan
- Default plan for new users
- Cannot delete a plan with active users (must reassign first)

---

## 3. Subscription Management

### Subscriptions
- Assign a user to a service plan → creates a subscription
- Subscription tracks: start date, status, resource usage
- **Statuses**: active, suspended, expired
- Suspend subscription → suspends all user's domains, email, FTP
- Activate subscription → restores all services
- Upgrade/downgrade plan (resource limits adjust immediately)

### Resource Enforcement
- Disk quota enforced at OS level (Linux quota or monitoring)
- Domain/database/email count enforced at application level
- Bandwidth tracking per domain (parsed from NGINX logs)
- Warning notifications at 80% and 95% of limits
- Auto-suspend option when limits exceeded (configurable)

### Usage Tracking
- Per-user disk usage (files + databases + email)
- Per-domain bandwidth (daily aggregation from access logs)
- Current resource usage vs plan limits displayed in user dashboard
- Admin view: all users' usage at a glance

---

## 4. Email Services

### Mail Server Setup
- **Postfix** for SMTP (sending/receiving)
- **Dovecot** for IMAP/POP3 (mailbox access)
- Virtual mailbox configuration (no system users per email account)
- Mailbox storage under `/var/mail/vhosts/{domain}/{user}/`

### Email Account Management
- Create email accounts: `user@domain.com`
- Set mailbox quota per account
- Password management (admin reset, user self-change)
- Enable/disable individual accounts
- Delete account (with option to keep/delete mail data)
- Bulk operations: create multiple accounts, bulk password reset

### Email Features
- **Forwarding**: forward email to external address (with/without keeping local copy)
- **Aliases**: map `alias@domain.com` → `real@domain.com`
- **Catch-all**: route all unmatched addresses to one mailbox
- **Autoresponders**: set vacation/out-of-office replies with date range
- **Mailbox quota**: per-account storage limit with warnings

### Email Operations
- View mailbox usage (size, message count)
- View mail queue (Postfix queue)
- Flush/delete stuck queue entries
- Test email sending from panel

---

## 5. Webmail

### Roundcube Integration
- Install Roundcube as a subdomain/path (`webmail.domain.com` or `domain.com/webmail`)
- Auto-configure IMAP/SMTP settings
- Single sign-on from panel → Roundcube
- Per-domain Roundcube branding (logo, title)
- Mobile-responsive webmail access

---

## 6. Email Security & Authentication

### DKIM
- Auto-generate DKIM key pair per domain (2048-bit RSA)
- Auto-add DKIM TXT record to DNS zone
- Sign outgoing mail with DKIM
- Key rotation capability
- DKIM verification status display

### SPF
- Auto-create SPF TXT record on domain creation
- Default: `v=spf1 a mx ip4:{server_ip} ~all`
- SPF record editor in DNS UI
- SPF validation tool

### DMARC
- Auto-create DMARC record: `v=DMARC1; p=none; rua=mailto:postmaster@{domain}`
- DMARC policy editor (none → quarantine → reject progression)
- DMARC report email configuration

### Anti-Spam
- **SpamAssassin** integration
- Per-domain spam filtering enable/disable
- Spam score threshold configuration (default: 5.0)
- Spam actions: mark as spam, move to junk folder, delete
- Custom spam rules per domain
- Whitelist/blacklist per domain and per account

### Antivirus
- **ClamAV** integration for email scanning
- Scan incoming and outgoing mail
- Action on virus: reject, quarantine, delete
- Virus definition auto-updates

---

## 7. Security Enhancements

### Fail2ban
- Auto-install and configure Fail2ban
- Jails for: SSH, panel login, FTP, SMTP, IMAP, POP3
- Panel UI to view:
  - Currently banned IPs
  - Ban history
  - Jail status (enabled/disabled, ban count)
- Manual ban/unban IP from UI
- Configurable ban time and max retries per jail
- Whitelist admin IPs

### ModSecurity WAF
- Install ModSecurity with NGINX
- OWASP Core Rule Set (CRS) as default
- Per-domain enable/disable
- Audit log viewer (blocked requests with details)
- Rule exception management (whitelist false positives)
- Paranoia level configuration (1-4)

### Two-Factor Authentication
- TOTP-based 2FA (Google Authenticator, Authy compatible)
- QR code setup flow
- Recovery codes (one-time use)
- Admin can enforce 2FA for all users
- Admin can reset user's 2FA

### Login Security
- Login attempt logging (IP, user, timestamp, success/fail)
- Account lockout after N failed attempts
- IP-based login restrictions (optional whitelist per user)
- Session management: view active sessions, revoke specific sessions

---

## 8. DNS Enhancements

### Additional Record Types
- **SRV** records (for mail autodiscovery, SIP, etc.)
- **CAA** records (Certificate Authority Authorization)
- **PTR** records (reverse DNS — informational, actual PTR set by ISP)

### DNS Templates
- Create DNS record templates for common setups
- Apply template on domain creation
- Templates for: basic hosting, hosting + email, hosting + Google Workspace, etc.

### Mail Autodiscovery
- Auto-create SRV records for mail client autodiscovery
- Auto-create `autoconfig.domain.com` (Thunderbird) and `autodiscover.domain.com` (Outlook) entries

---

## 9. PHP Enhancements

### Multiple PHP Versions
- Install multiple PHP versions side-by-side (7.4, 8.0, 8.1, 8.2, 8.3)
- Per-domain PHP version selection
- Switch PHP versions without downtime (FPM pool restart only)
- Install/remove PHP versions from panel UI

### PHP Handlers
- PHP-FPM (default, recommended)
- FastCGI
- Per-domain handler selection

### PHP Extensions
- List installed extensions per PHP version
- Enable/disable common extensions from UI (not compiling, toggle ini)
- Install additional extensions via system packages

### PHP Composer
- Run Composer commands per domain from panel
- `composer install`, `composer update`
- View `composer.json` and lock file

---

## 10. Backup Enhancements

### Scheduled Backups
- Cron-based backup scheduling
- Configure: daily, weekly, monthly
- Set backup time (off-peak hours)
- Retention policy: keep last N backups per schedule

### Incremental Backups
- Only backup changed files since last full backup
- Reduces storage usage and backup duration
- Full backup weekly + daily incrementals

### Selective Restore
- Browse backup contents before restoring
- Restore individual items:
  - Specific domain files
  - Specific database
  - Specific email accounts
  - Configuration only
- Preview diff before restore (what will change)

### Per-User Backups
- Users can create backups of their own sites (if plan allows)
- Users can restore their own backups
- Backup count/size counts against user's disk quota

---

## 11. Cron Job Management

### User Cron Jobs
- Users can create cron jobs (if plan allows)
- Visual cron schedule builder (minute, hour, day, month, weekday)
- Common presets: every minute, every 5 min, hourly, daily, weekly, monthly
- Command input with validation
- Max cron jobs per user (from service plan)

### Cron Features
- Enable/disable individual cron jobs
- View last run time and exit status
- View cron job output (last N runs)
- Email notification on failure (optional)
- Lock files to prevent overlap

---

## New API Endpoints (V1 additions)

### Users
- `POST /api/users` — Create user
- `GET /api/users` — List users (admin)
- `GET /api/users/:id` — User detail
- `PUT /api/users/:id` — Update user
- `DELETE /api/users/:id` — Delete user
- `POST /api/users/:id/suspend` — Suspend user
- `POST /api/users/:id/activate` — Activate user

### Plans
- `CRUD /api/plans` — Full CRUD for service plans
- `GET /api/plans/:id/users` — Users on this plan

### Subscriptions
- `POST /api/subscriptions` — Create subscription (assign user to plan)
- `PUT /api/subscriptions/:id` — Change plan
- `GET /api/users/:id/usage` — Resource usage

### Email
- `CRUD /api/domains/:id/email/accounts` — Email account management
- `PUT /api/email/:id/password` — Change email password
- `PUT /api/email/:id/quota` — Set quota
- `CRUD /api/domains/:id/email/forwarders` — Forwarders
- `CRUD /api/domains/:id/email/aliases` — Aliases
- `PUT /api/domains/:id/email/catchall` — Catch-all config
- `CRUD /api/email/:id/autoresponder` — Autoresponder
- `GET /api/domains/:id/email/dkim` — DKIM status
- `POST /api/domains/:id/email/dkim/generate` — Generate DKIM keys
- `PUT /api/domains/:id/email/spam` — Spam filter settings
- `GET /api/email/queue` — Mail queue

### Security
- `GET /api/security/fail2ban/status` — Fail2ban status
- `GET /api/security/fail2ban/banned` — Banned IPs
- `POST /api/security/fail2ban/ban` — Manual ban
- `POST /api/security/fail2ban/unban` — Manual unban
- `GET /api/security/modsec/log` — ModSecurity audit log
- `PUT /api/domains/:id/modsec` — Toggle ModSecurity per domain
- `POST /api/auth/2fa/setup` — Initialize 2FA
- `POST /api/auth/2fa/verify` — Verify and enable 2FA
- `DELETE /api/auth/2fa` — Disable 2FA

### Cron
- `CRUD /api/domains/:id/cron` — Cron job management

---

## New Database Tables (V1 additions)

- `users` — id, email, password_hash, role, status, plan_id, system_username, created_at
- `plans` — id, name, description, limits (JSON), features (JSON), created_at
- `subscriptions` — id, user_id, plan_id, status, started_at, expires_at
- `email_accounts` — id, domain_id, local_part, password_hash, quota_mb, status, created_at
- `email_forwarders` — id, domain_id, source, destination
- `email_aliases` — id, domain_id, alias, target_account_id
- `email_autoresponders` — id, email_account_id, subject, body, start_date, end_date, enabled
- `dkim_keys` — id, domain_id, selector, private_key, public_key, created_at
- `fail2ban_events` — id, jail, ip, action (ban/unban), timestamp
- `cron_jobs` — id, user_id, domain_id, schedule, command, enabled, last_run, last_status
- `sessions` — id, user_id, ip, user_agent, created_at, expires_at
- `login_attempts` — id, email, ip, success, timestamp
