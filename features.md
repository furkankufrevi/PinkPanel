# PinkPanel - Feature Roadmap

Feature reference based on Plesk Panel, organized into development phases.
Detailed specifications for each phase are in the `phases/` directory.

---

## Phase Overview

| Phase | Focus | Key Deliverables | Details |
|-------|-------|-----------------|---------|
| **V0** | Technology Stack | Go + React + SQLite + Redis, architecture design | [V0-stack.md](phases/V0-stack.md) |
| **Frontend** | UI/UX Architecture | Design system, component library, page designs, interaction patterns | [frontend-plan.md](phases/frontend-plan.md) |
| **MVP** | Core Foundation | Dashboard, domains, NGINX/Apache, PHP, SSL, file manager, MySQL, backup, logs | [MVP.md](phases/MVP.md) |
| **V1** | Multi-User & Email | Users, roles, service plans, subscriptions, Postfix/Dovecot email, DKIM/SPF/DMARC, Fail2ban, ModSecurity, 2FA | [V1-multiuser-email.md](phases/V1-multiuser-email.md) |
| **V2** | Developers & WordPress | WP Toolkit (staging, cloning, hardening), Git deploy, Node.js/Python/Ruby, Docker, CLI, REST API, monitoring | [V2-devtools-wordpress.md](phases/V2-devtools-wordpress.md) |
| **V3** | Hosting Business | Resellers, white-label branding, cPanel/DirectAdmin migration, S3/GDrive backup, Grafana monitoring, reporting, DNSSEC | [V3-hosting-business.md](phases/V3-hosting-business.md) |
| **V4** | Scale & Polish | Extension marketplace, CMS toolkits, website builder, mobile app, multi-server, i18n, PCI DSS, cloud provisioning | [V4-scale-polish.md](phases/V4-scale-polish.md) |

---

## Feature Count by Phase

| Phase | Features | API Endpoints | DB Tables |
|-------|----------|--------------|-----------|
| MVP | ~50 | ~40 | 10 |
| V1 | ~70 | ~30 | 11 |
| V2 | ~90 | ~40 | 15 |
| V3 | ~60 | ~25 | 11 |
| V4 | ~80 | ~30 | 16 |

---

## Stack Summary (V0)

```
Backend:    Go (Fiber) — single binary, low memory, system-level ops
Frontend:   React + TypeScript + Vite + Shadcn/ui + Tailwind
Database:   SQLite (data) + Redis (cache/sessions/realtime)
Agent:      Separate Go binary (root privileges, Unix socket)
CLI:        Cobra (Go) — talks to REST API
Deploy:     Two binaries: pinkpanel + pinkpanel-agent
Target OS:  Ubuntu 22.04/24.04, Debian 11/12 (initial)
```
