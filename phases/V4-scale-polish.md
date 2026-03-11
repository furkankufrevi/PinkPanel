# V4 - Scale & Polish

_Enterprise features, application marketplace, mobile app, multi-server management, internationalization, and website builder._

**Depends on**: V3 complete

---

## 1. Application Marketplace

### Extension System Architecture
- Plugin API: Go interface that extensions implement
- Extensions can add: UI pages, API endpoints, background jobs, hooks
- Extension manifest (`pinkpanel-extension.yml`):
  ```yaml
  name: seo-toolkit
  version: 1.0.0
  author: PinkPanel
  description: SEO analysis and optimization
  permissions:
    - domains.read
    - files.read
  hooks:
    - domain.created
    - domain.deleted
  ui:
    - path: /seo
      label: SEO Toolkit
      icon: search
  ```
- Extension lifecycle: install, enable, disable, uninstall, update
- Extension isolation (separate process or sandboxed execution)

### Marketplace
- Centralized extension repository (hosted by PinkPanel project)
- Browse/search extensions by category
- Categories: security, backup, performance, SEO, developer tools, email, monitoring, CMS
- Extension ratings and reviews
- One-click install from marketplace
- Auto-update extensions

### Built-in Extensions to Publish
- **SEO Toolkit**: per-site SEO score, meta tag analysis, sitemap generator, robots.txt editor
- **Uptime Monitor**: external HTTP checks, response time tracking, downtime alerts
- **Image Optimizer**: auto-compress uploaded images (WebP conversion, lossy/lossless)
- **Malware Scanner**: file scanning for known malware signatures, scheduled scans
- **Performance Analyzer**: PageSpeed Insights integration, Core Web Vitals tracking
- **Database Optimizer**: automated table optimization, slow query detection

### Third-Party Extension SDK
- Documentation for building extensions
- CLI scaffolding tool: `pinkpanel ext create my-extension`
- Testing framework for extensions
- Extension submission process for marketplace

---

## 2. CMS Toolkits

### Joomla! Toolkit
- One-click Joomla installation
- Detect existing Joomla instances
- Extension (plugin/module/component) management
- Template management
- Security hardening (one-click):
  - Disable user registration
  - Block directory listing
  - Strong admin password enforcement
  - Remove installer directory warning
- Core updates with backup
- Mass management across all Joomla instances

### Drupal Toolkit
- One-click Drupal installation (Composer-based)
- Module management
- Theme management
- Drush CLI integration
- Security updates
- Instance detection

### Generic CMS Detection
- Auto-detect CMS type from document root:
  - WordPress, Joomla, Drupal, Magento, PrestaShop, Laravel, etc.
- Display CMS type and version on domain listing
- Link to relevant toolkit if available
- Update notifications for detected CMS

---

## 3. Website Builder

### Builder Features
- Drag-and-drop page builder (embedded in panel)
- Component library:
  - Headers, footers, navigation
  - Hero sections, feature grids
  - Text blocks, image galleries, sliders
  - Contact forms, maps
  - Pricing tables, testimonials
  - Blog post layouts
  - Call-to-action buttons
- Responsive design: desktop, tablet, mobile preview
- Visual styling: colors, fonts, spacing, borders (no CSS knowledge needed)
- Page management: create, edit, delete, reorder pages
- Global styles: site-wide font, color palette, button styles

### Templates
- Template library: 30+ pre-built website templates
- Categories: business, portfolio, restaurant, agency, blog, e-commerce, landing page
- One-click template apply
- Template customization

### E-Commerce (Basic)
- Product catalog pages
- Shopping cart (Snipcart or similar embeddable solution)
- Payment gateway integration (Stripe, PayPal)
- Order notification emails
- Product image galleries

### Blog
- Blog post editor (rich text / Markdown)
- Categories and tags
- Post scheduling
- RSS feed auto-generation
- Social sharing meta tags (Open Graph, Twitter Cards)

### SEO for Builder
- Page title and meta description per page
- Custom URL slugs
- Auto-generate sitemap.xml
- Open Graph tags editor
- Structured data (JSON-LD) for basic schema types

### Technical
- Builder generates static HTML + CSS + JS
- Output deployed to domain's document root
- No runtime dependencies (pure static output)
- Version history (restore previous versions)
- Export site as zip

---

## 4. Mobile App

### Platform
- **React Native** (shared codebase for iOS and Android)
- Communicates with PinkPanel REST API
- API key authentication (stored securely on device)

### Features

#### Dashboard
- Server status overview (CPU, RAM, disk, services)
- Quick stats (domains, users, alerts)
- Pull-to-refresh
- Service start/stop/restart

#### Domain Management
- List domains with status
- Add/delete domains
- Suspend/activate
- View domain details
- Quick SSL issue

#### Monitoring
- Real-time server metrics graphs
- Alert history
- Service status with quick actions

#### User Management
- List users
- Suspend/activate users
- View user usage

#### Notifications
- Push notifications for:
  - Service down alerts
  - SSL expiring
  - Backup failures
  - Disk/CPU/RAM threshold alerts
  - Security events (brute force detected)
- Notification preferences (which alerts to receive)
- Notification history

#### Quick Actions
- Restart NGINX / PHP / MySQL
- Create quick backup
- Ban/unban IP
- View active connections

---

## 5. Internationalization (i18n)

### Translation System
- All UI strings externalized to translation files
- JSON-based translation files per language
- Frontend: `react-i18next` for React components
- Backend: translated API error messages and email templates

### Supported Languages (Initial)
- English (default)
- Turkish
- German
- French
- Spanish
- Portuguese (Brazilian)
- Russian
- Chinese (Simplified)
- Japanese
- Arabic (RTL support)

### RTL Support
- Full right-to-left layout for Arabic, Hebrew, Persian
- RTL-aware components (Tailwind CSS RTL plugin)
- Mirrored navigation and layouts

### Community Translations
- Translation contribution system
- Web-based translation editor
- Translation completion percentage per language
- Translator credits

### Locale Settings
- Per-user language selection
- Date/time format per locale
- Number format per locale
- Timezone per user

---

## 6. Advanced Security

### PCI DSS Compliance Tools
- PCI DSS checklist with automated checks
- One-click fixes for common PCI requirements:
  - TLS 1.2+ enforcement
  - Disable weak ciphers
  - Enable firewall
  - Enforce password policies
  - Enable audit logging
  - Disable unnecessary services
- Compliance report generation
- Scheduled compliance scans

### Security Advisor
- Overall security score for the server (0-100)
- Categorized recommendations:
  - SSL: all sites have HTTPS, HSTS, strong ciphers
  - Firewall: active, rules configured
  - Updates: all packages up to date
  - Authentication: 2FA enabled, strong passwords
  - Services: only necessary ports open
  - WordPress: all sites hardened and updated
- One-click fix for each recommendation
- Historical score tracking

### File Integrity Monitoring
- Baseline scan of critical system files
- Periodic comparison against baseline
- Alert on unexpected changes to:
  - System binaries, configs, cron jobs
  - Web server configs
  - PHP configs
  - Panel binaries
- Exclude list for expected changes

### Vulnerability Scanner
- Scan all WordPress/Joomla/Drupal instances for known vulnerabilities
- CVE database integration
- Scheduled scans (daily/weekly)
- Vulnerability report with severity ratings
- Remediation guidance per vulnerability

### Automated Security Patching
- Auto-apply critical security updates for OS packages
- Configurable: auto-patch all, security only, or manual
- Patch notification (email before auto-patching)
- Patch rollback capability

---

## 7. Multi-Server Management

### Server Registry
- Add remote PinkPanel servers to a central control panel
- Connection via PinkPanel REST API + API key
- Server metadata: name, IP, OS, location, role (web, mail, DB)
- Server health status in central dashboard

### Central Dashboard
- Overview of all servers: status, resource usage, alerts
- Aggregate stats: total domains, total users, total storage across all servers
- Per-server drill-down to full server management

### Cross-Server Operations
- Migrate domains/users between servers
- Create user on specific server
- Central backup management (view/trigger backups on any server)
- Central SSL management
- Unified search across all servers

### DNS Cluster
- Synchronized DNS across multiple servers
- Primary/secondary DNS server configuration
- Auto-replicate zone changes to all DNS servers
- Split DNS support

### Load Balancing Configuration
- NGINX-based load balancer configuration
- Round-robin, least connections, IP hash
- Health checks for backend servers
- SSL termination at load balancer
- Session persistence options

---

## 8. Advanced Developer Tools

### Online Code Editor
- VS Code-like editor embedded in panel (Monaco Editor)
- Syntax highlighting for 50+ languages
- IntelliSense/autocomplete for common languages
- Multiple file tabs
- File tree sidebar
- Integrated terminal
- Git diff viewer
- Search and replace across files
- Theme selection (dark, light, high contrast)

### Advanced Site Preview
- Preview website without DNS changes (hosts file bypass)
- Preview with different screen sizes (responsive testing)
- Dynamic site support (PHP, Node.js rendered preview)
- Screenshot comparison tool
- Share preview link (temporary public URL)

### Webhook & Event System
- Extended event catalog:
  - Server: boot, shutdown, service start/stop, high resource usage
  - Domain: created, deleted, suspended, SSL issued/expired
  - User: created, deleted, login, password change
  - Email: account created, mailbox full, delivery failure
  - Backup: started, completed, failed
  - Security: login failure, IP banned, WAF blocked, malware detected
  - WordPress: installed, updated, plugin activated
- Event log viewer with filtering
- Custom automation scripts triggered by events
- Event-driven email notifications
- Integration templates: Slack, Discord, Teams, PagerDuty

### CI/CD Integration
- GitHub Actions webhook integration
- GitLab CI/CD webhook integration
- Generic CI/CD webhook endpoint
- Deploy status badges
- Deploy lock (prevent deploys during maintenance)

---

## 9. Advanced Resource Management

### Cgroups v2 Integration
- Per-user CPU limits (CPU shares, CPU quota)
- Per-user RAM limits (memory.max)
- Per-user disk I/O limits (io.max)
- Per-user process count limits
- Resource usage vs limits displayed in user dashboard
- Admin override for temporary resource burst

### CloudLinux Integration (Optional)
- LVE (Lightweight Virtual Environment) integration
- PHP Selector integration
- CageFS integration
- MySQL Governor integration
- Auto-detect CloudLinux and enable features

### Resource Abuse Detection
- Monitor per-user resource consumption
- Detect abnormal spikes (sudden CPU/RAM/disk usage)
- Auto-notify admin on abuse detection
- Optional auto-suspend on sustained abuse
- Abuse report per user

---

## 10. Cloud Platform Integration

### One-Click Server Provisioning
- Provision new PinkPanel servers from panel:
  - **DigitalOcean**: select region, size, create droplet, auto-install PinkPanel
  - **Vultr**: select region, plan, deploy instance, auto-install
  - **Hetzner**: select location, type, create server, auto-install
  - **AWS Lightsail**: select region, blueprint, launch instance, auto-install
- API key management for cloud providers
- Server provisioning history

### Cloud DNS Integration
- Use cloud DNS instead of local BIND:
  - Cloudflare DNS API
  - AWS Route 53
  - DigitalOcean DNS
  - Google Cloud DNS
- Auto-sync DNS zones to cloud DNS
- Hybrid: local BIND + cloud DNS secondary

---

## 11. Social & SEO Tools

### SEO Toolkit (Extension)
- Per-site SEO score
- Meta tag analysis and suggestions
- Sitemap.xml management
- Robots.txt editor
- Structured data validator
- Google Search Console integration (optional)
- Keyword density checker
- Broken link checker

### Uptime Monitoring (Extension)
- External HTTP/HTTPS checks from multiple locations
- Check intervals: 1, 5, 15, 30 minutes
- Response time tracking and graphing
- Downtime alerts (email, push notification, webhook)
- Uptime percentage reporting
- Status page generator (public status page per domain)

---

## New API Endpoints (V4 additions)

### Extensions
- `GET /api/extensions/marketplace` — Browse marketplace
- `POST /api/extensions/install` — Install extension
- `PUT /api/extensions/:id` — Enable/disable/update
- `DELETE /api/extensions/:id` — Uninstall extension
- `GET /api/extensions` — List installed extensions

### Multi-Server
- `GET /api/servers` — List managed servers
- `POST /api/servers` — Add server
- `GET /api/servers/:id/status` — Server status
- `POST /api/servers/:id/migrate` — Migrate domain to server
- `POST /api/servers/provision` — Provision new cloud server

### Website Builder
- `GET /api/builder/templates` — List templates
- `GET /api/domains/:id/builder` — Get builder data
- `PUT /api/domains/:id/builder` — Save builder data
- `POST /api/domains/:id/builder/publish` — Publish site
- `GET /api/domains/:id/builder/versions` — Version history

### i18n
- `GET /api/i18n/languages` — Available languages
- `GET /api/i18n/:lang` — Translation strings
- `PUT /api/users/:id/locale` — Set user locale

### Security
- `GET /api/security/score` — Security score
- `GET /api/security/recommendations` — Security recommendations
- `POST /api/security/recommendations/:id/fix` — Apply fix
- `GET /api/security/vulnerabilities` — Vulnerability scan results
- `POST /api/security/vulnerabilities/scan` — Trigger scan
- `GET /api/security/pci` — PCI compliance status
- `GET /api/security/integrity` — File integrity status

### Cloud
- `GET /api/cloud/providers` — Configured providers
- `POST /api/cloud/providers` — Add provider credentials
- `GET /api/cloud/:provider/regions` — Available regions
- `GET /api/cloud/:provider/sizes` — Available sizes
- `POST /api/cloud/:provider/provision` — Provision server

---

## New Database Tables (V4 additions)

- `extensions` — id, name, version, manifest (JSON), status (installed/enabled/disabled), installed_at
- `extension_settings` — id, extension_id, key, value
- `servers` — id, name, ip, api_key_hash, os, location, role, status, last_seen_at, added_at
- `server_metrics` — server_id, timestamp, cpu, ram, disk, network
- `builder_sites` — id, domain_id, template_id, pages (JSON), styles (JSON), version, published_at
- `builder_templates` — id, name, category, preview_url, data (JSON)
- `builder_versions` — id, builder_site_id, data (JSON), created_at
- `translations` — locale, key, value
- `user_locales` — user_id, locale, timezone, date_format
- `security_scores` — id, score, breakdown (JSON), scanned_at
- `security_recommendations` — id, category, severity, title, description, fix_command, status
- `vulnerability_scans` — id, cms_instance_id, cve_id, severity, description, remediation, scanned_at
- `file_integrity_baselines` — id, file_path, hash, permissions, owner, scanned_at
- `file_integrity_changes` — id, baseline_id, change_type, old_hash, new_hash, detected_at
- `cloud_providers` — id, type (digitalocean/vultr/hetzner/aws), name, credentials (encrypted JSON)
- `provisioned_servers` — id, cloud_provider_id, instance_id, name, ip, status, provisioned_at
- `uptime_checks` — id, domain_id, url, interval_minutes, locations (JSON), enabled
- `uptime_results` — id, check_id, location, status_code, response_time_ms, checked_at
