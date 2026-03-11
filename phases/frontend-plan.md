# PinkPanel — Frontend Architecture & UI/UX Plan

---

## Design Philosophy

**Clean, confident, and information-dense** — like Vercel meets a hosting panel. Every pixel earns its place. No clutter, no unnecessary decoration, no walls of text. The UI should feel like a premium developer tool, not a 2010 cPanel clone.

### Principles
1. **Clarity over decoration** — Data-first layouts. No ornamental borders, gradients, or drop shadows unless they serve hierarchy.
2. **Fast by default** — Optimistic updates, skeleton loading, no full-page reloads. Every action feels instant.
3. **Progressive disclosure** — Show essentials first, reveal complexity on demand. Beginners aren't overwhelmed, power users aren't slowed down.
4. **Consistent patterns** — Same interaction for same action everywhere. Delete always asks confirmation. Create always opens a sheet/dialog. Status always uses the same color coding.
5. **Dark-first design** — Dark mode is the primary design target (server admins work late). Light mode is equally polished.

---

## Design System

### Color Palette

**Using Shadcn "Zinc" base with a pink/rose accent** (matching the PinkPanel brand).

```
// Dark mode (primary)
--background:       hsl(240 10% 3.9%)      // Near-black
--foreground:       hsl(0 0% 98%)           // Near-white
--card:             hsl(240 10% 5.5%)       // Slightly lifted
--card-foreground:  hsl(0 0% 98%)
--popover:          hsl(240 10% 5.5%)
--muted:            hsl(240 3.7% 15.9%)     // Subtle backgrounds
--muted-foreground: hsl(240 5% 64.9%)       // Secondary text
--border:           hsl(240 3.7% 15.9%)     // Subtle borders
--input:            hsl(240 3.7% 15.9%)

// Brand accent (Pink/Rose)
--primary:          hsl(346 77% 59%)        // PinkPanel pink
--primary-foreground: hsl(0 0% 100%)
--accent:           hsl(346 77% 59% / 0.1)  // Pink tint for hover

// Semantic colors
--success:          hsl(142 71% 45%)        // Green — running, active, healthy
--warning:          hsl(38 92% 50%)         // Amber — warning, expiring
--destructive:      hsl(0 84% 60%)          // Red — error, stopped, delete
--info:             hsl(217 91% 60%)        // Blue — info, pending

// Light mode overrides
--background:       hsl(0 0% 100%)
--foreground:       hsl(240 10% 3.9%)
--card:             hsl(0 0% 99%)
--border:           hsl(240 5.9% 90%)
--muted:            hsl(240 4.8% 95.9%)
```

### Typography

**Font: Geist Sans + Geist Mono**
- Geist Sans — UI text, headings, labels (same font Vercel uses, clean and modern)
- Geist Mono — code, file paths, IP addresses, config values, terminal output

```
// Scale
text-xs:   12px / 1.5     // Labels, badges, timestamps
text-sm:   14px / 1.5     // Body text, table cells, form inputs
text-base: 16px / 1.5     // Paragraph text (rarely used in panels)
text-lg:   18px / 1.75    // Section titles within pages
text-xl:   20px / 1.75    // Page subtitles
text-2xl:  24px / 1.33    // Page titles
text-3xl:  30px / 1.33    // Dashboard hero numbers (CPU 42%)
```

### Spacing System
Tailwind's default 4px grid. Key spacings:
- `gap-1` (4px) — between icon and label
- `gap-2` (8px) — between form elements
- `gap-4` (16px) — between cards, sections
- `gap-6` (24px) — between major page sections
- `p-4` (16px) — card padding
- `p-6` (24px) — page padding

### Border Radius
- `rounded-md` (6px) — buttons, inputs, small cards
- `rounded-lg` (8px) — cards, dialogs, dropdowns
- `rounded-xl` (12px) — large cards, page sections
- `rounded-full` — avatars, status dots, badges

### Shadows (minimal)
- Cards: no shadow in dark mode, `shadow-sm` in light mode
- Dropdowns/popovers: `shadow-md`
- Dialogs/modals: `shadow-lg` + backdrop blur
- Borders preferred over shadows for dark mode hierarchy

### Icons
**Lucide React** — consistent 24px stroke icons, 1.5px stroke weight.
- Navigation: `LayoutDashboard`, `Globe`, `Database`, `FolderOpen`, `Shield`, `Mail`, `HardDrive`, `Settings`
- Actions: `Plus`, `Pencil`, `Trash2`, `RotateCcw`, `Download`, `Upload`, `Copy`
- Status: `CheckCircle2` (success), `AlertTriangle` (warning), `XCircle` (error), `Loader2` (loading, animated spin)

---

## Layout Architecture

### App Shell

```
┌──────────────────────────────────────────────────────────────┐
│ ┌──────────┐ ┌──────────────────────────────────────────────┐│
│ │          │ │  Header                                      ││
│ │          │ │  ┌─────────────────────────────┐  ┌────────┐ ││
│ │ Sidebar  │ │  │ Breadcrumb / Page Title     │  │ ⚙ 👤  │ ││
│ │          │ │  └─────────────────────────────┘  └────────┘ ││
│ │ ┌──────┐ │ ├──────────────────────────────────────────────┤│
│ │ │ Logo │ │ │                                              ││
│ │ └──────┘ │ │  Main Content Area                           ││
│ │          │ │                                              ││
│ │ Dashboard│ │  ┌────────────┐ ┌────────────┐ ┌──────────┐ ││
│ │ Domains  │ │  │ Card       │ │ Card       │ │ Card     │ ││
│ │ Databases│ │  │            │ │            │ │          │ ││
│ │ Files    │ │  └────────────┘ └────────────┘ └──────────┘ ││
│ │ Email    │ │                                              ││
│ │ DNS      │ │  ┌──────────────────────────────────────────┐││
│ │ SSL      │ │  │ Table / Content                          │││
│ │ Backups  │ │  │                                          │││
│ │ Logs     │ │  │                                          │││
│ │          │ │  │                                          │││
│ │ ─────── │ │  │                                          │││
│ │ Settings │ │  └──────────────────────────────────────────┘││
│ └──────────┘ └──────────────────────────────────────────────┘│
└──────────────────────────────────────────────────────────────┘
```

### Sidebar
- **Width**: 256px expanded, 48px collapsed (icon-only mode)
- **Collapse**: `Cmd+B` / `Ctrl+B` keyboard shortcut, or click trigger button
- **Mobile**: Off-canvas drawer (slides in from left), triggered by hamburger icon
- **Style**: Subtle border-right separator, no heavy background contrast
- **Logo**: PinkPanel logo at top, collapses to icon "P" mark
- **Navigation groups**: separated by subtle dividers with optional group labels
- **Active state**: pink accent background tint + left border indicator
- **Hover state**: subtle muted background
- **Badges**: notification count badges on items (e.g., "3 updates" on WordPress)
- **Footer**: server hostname, panel version, collapse toggle

### Header
- **Height**: 48px
- **Left**: breadcrumb trail (Home > Domains > example.com) or page title
- **Right**: notification bell (dropdown), dark/light toggle, user avatar dropdown
- **User dropdown**: profile, change password, language, logout
- **Sticky**: stays fixed at top of content area during scroll

### Main Content
- **Max width**: none (fluid, fills available space)
- **Padding**: 24px all sides
- **Scroll**: main content scrolls independently (sidebar + header stay fixed)

### Responsive Breakpoints
```
sm:  640px   — Mobile (single column, off-canvas sidebar)
md:  768px   — Tablet (collapsed sidebar, adjusted grid)
lg:  1024px  — Desktop (expanded sidebar)
xl:  1280px  — Wide desktop (more columns in grids)
2xl: 1536px  — Ultra-wide (max content width optional)
```

---

## Component Library

### Core Components (Shadcn/ui base + customizations)

#### Status Badge
Used everywhere to indicate state. Consistent color coding across the entire app.

| Status | Color | Dot | Example |
|--------|-------|-----|---------|
| Active / Running | Green | ● | Service status, domain active |
| Inactive / Stopped | Red | ● | Service stopped, domain suspended |
| Warning / Expiring | Amber | ● | SSL expiring soon, disk 90% |
| Pending / Processing | Blue | ● (pulse) | Backup in progress, DNS propagating |
| Disabled | Gray | ● | Feature off, cron paused |

```tsx
<StatusBadge status="active" />    // Green dot + "Active" text
<StatusBadge status="running" />   // Green dot + "Running" text
<StatusBadge status="stopped" />   // Red dot + "Stopped" text
<StatusBadge status="expiring" />  // Amber dot + "Expiring" text
```

#### Data Table
Based on TanStack Table + Shadcn table styling. Used for: domains, databases, email accounts, FTP accounts, backups, logs, users.

Features:
- Column sorting (click header to toggle asc/desc)
- Column visibility toggle (show/hide columns)
- Row selection with checkbox (for bulk actions)
- Bulk action bar (appears when rows selected: "3 selected — Delete | Suspend | Export")
- Search/filter input above table
- Pagination (10/25/50/100 per page)
- Empty state illustration + CTA when no data
- Loading skeleton (animated rows)
- Row click → navigate to detail page
- Row action menu (three-dot `...` dropdown: Edit, Delete, Suspend, etc.)
- Responsive: horizontal scroll on mobile, or card view for small tables

#### Stat Card
Dashboard overview cards showing a metric with trend.

```
┌─────────────────────┐
│ Total Domains    ↑  │
│                     │
│    47               │
│    ███████████░░░░  │
│    +3 this month    │
└─────────────────────┘
```

- Icon + label at top
- Large metric number (text-3xl, monospace for alignment)
- Optional sparkline mini-chart (last 7 days)
- Optional trend indicator (+12% ↑ in green, -5% ↓ in red)
- Click → navigate to related page

#### Command Palette (Cmd+K)
Global command palette for power users. Inspired by Vercel/Linear.

- `Cmd+K` / `Ctrl+K` opens palette
- Search across: domains, databases, users, email accounts, settings
- Quick actions: "Add domain", "Create backup", "Restart NGINX", "View logs"
- Recent items section
- Keyboard navigation (arrow keys + enter)
- Fuzzy search matching
- Context-aware: shows relevant actions based on current page

#### Sheet (Side Panel)
Used for create/edit forms. Slides in from right, doesn't lose page context.

- **Create domain**, **Create database**, **Create email** → all open in sheets
- Width: 480px (default), 640px (wide, for complex forms)
- Overlay backdrop (click outside or Esc to close)
- Form inside with validation
- Submit button at bottom (sticky if form scrolls)

#### Confirmation Dialog
Used for destructive actions (delete, suspend, restart service).

- Modal centered dialog
- Clear description of what will happen
- Type-to-confirm for critical actions: "Type `example.com` to delete this domain"
- Cancel (secondary) + Confirm (destructive red) buttons

#### Toast Notifications
Using Sonner — bottom-right stacked toasts.

- **Success**: green accent, "Domain created successfully"
- **Error**: red accent, "Failed to create database: quota exceeded"
- **Info**: blue accent, "Backup in progress..."
- **Loading**: spinner, transforms into success/error when done
- Auto-dismiss: 5 seconds (success/info), persistent (error)
- Action button in toast: "Undo" for reversible actions

#### Empty State
Shown when a section has no data yet.

```
┌─────────────────────────────────┐
│                                 │
│         (illustration)          │
│                                 │
│     No domains yet              │
│     Add your first domain       │
│     to get started.             │
│                                 │
│     [+ Add Domain]              │
│                                 │
└─────────────────────────────────┘
```

- Illustration/icon (subtle, not cartoonish)
- Clear title + one line description
- Primary CTA button

---

## Page Designs

### 1. Login Page

```
┌──────────────────────────────────────────────────────────────┐
│                                                              │
│                                                              │
│              ┌──────────────────────────┐                    │
│              │                          │                    │
│              │     🩷 PinkPanel         │                    │
│              │                          │                    │
│              │  Email                   │                    │
│              │  ┌──────────────────────┐│                    │
│              │  │ admin@example.com    ││                    │
│              │  └──────────────────────┘│                    │
│              │                          │                    │
│              │  Password                │                    │
│              │  ┌──────────────────────┐│                    │
│              │  │ ••••••••••      👁  ││                    │
│              │  └──────────────────────┘│                    │
│              │                          │                    │
│              │  ┌──────────────────────┐│                    │
│              │  │     Sign In          ││                    │
│              │  └──────────────────────┘│                    │
│              │                          │                    │
│              │  Forgot password?        │                    │
│              │                          │                    │
│              └──────────────────────────┘                    │
│                                                              │
│              server.example.com • v1.0.0                     │
│                                                              │
└──────────────────────────────────────────────────────────────┘
```

- Centered card on dark/subtle gradient background
- Minimal: logo, email, password, sign in button
- Password visibility toggle
- Loading spinner on button during auth
- Error shake animation + inline error message on failure
- 2FA step: after password, show TOTP code input (6-digit with auto-focus between digits)
- Server hostname and version in subtle footer text

### 2. Dashboard

```
┌─ Dashboard ──────────────────────────────────────────────────┐
│                                                              │
│  ┌──────────┐ ┌──────────┐ ┌──────────┐ ┌──────────┐       │
│  │ CPU      │ │ RAM      │ │ Disk     │ │ Uptime   │       │
│  │  23%     │ │ 1.2/4 GB │ │ 34/80 GB │ │ 47d 12h  │       │
│  │  ▃▅▂▄▆▃ │ │  ▃▅▂▄▆▃ │ │  ████░░░ │ │          │       │
│  └──────────┘ └──────────┘ └──────────┘ └──────────┘       │
│                                                              │
│  ┌───────────────────────────────┐ ┌───────────────────────┐│
│  │ Services                      │ │ Quick Stats           ││
│  │                               │ │                       ││
│  │  ● NGINX          Running    │ │  47  Domains          ││
│  │  ● PHP-FPM 8.3    Running    │ │  12  Databases        ││
│  │  ● MariaDB        Running    │ │  89  Email Accounts   ││
│  │  ● Postfix        Running    │ │   5  Active Backups   ││
│  │  ● Dovecot        Running    │ │   3  SSL Expiring     ││
│  │  ● BIND9          Running    │ │   0  Security Issues  ││
│  │  ● Redis          Running    │ │                       ││
│  │  ● vsftpd         Running    │ │                       ││
│  └───────────────────────────────┘ └───────────────────────┘│
│                                                              │
│  ┌──────────────────────────────────────────────────────────┐│
│  │ Resource Usage (24h)                          [1h▾]     ││
│  │                                                          ││
│  │  CPU ━━━  RAM ━━━  Network ━━━                          ││
│  │  80%│                                                    ││
│  │     │      ╱╲                                            ││
│  │  40%│  ╱╲╱╱  ╲╱╲   ╱╲                                  ││
│  │     │╱╱        ╲╱╲╱╱  ╲                                 ││
│  │   0%└────────────────────────────────────                ││
│  │     00:00    06:00    12:00    18:00    24:00            ││
│  └──────────────────────────────────────────────────────────┘│
│                                                              │
│  ┌──────────────────────────────────────────────────────────┐│
│  │ Recent Activity                                          ││
│  │                                                          ││
│  │  🕐 2m ago   Domain blog.example.com created     admin  ││
│  │  🕐 15m ago  SSL renewed for example.com         system ││
│  │  🕐 1h ago   Backup completed (2.3 GB)           system ││
│  │  🕐 3h ago   PHP version changed to 8.3          admin  ││
│  │  🕐 1d ago   Database shop_db created            admin  ││
│  └──────────────────────────────────────────────────────────┘│
└──────────────────────────────────────────────────────────────┘
```

**Stat cards**: 4-column grid on desktop, 2-column on tablet, 1-column on mobile.
- CPU + RAM: live-updating every 5 seconds via WebSocket
- CPU card: area sparkline chart (last 60 data points)
- RAM card: used/total with progress bar
- Disk card: used/total with progress bar, color shifts yellow→red as usage increases
- Uptime: `Xd Xh` format

**Services card**: list of managed services with status dots. Click a service → popover with: Restart, Stop, View Logs actions.

**Resource chart**: Recharts area chart. Time range selector: 1h, 6h, 24h, 7d, 30d. Toggle datasets (CPU, RAM, Network). Tooltip shows exact values on hover.

**Activity feed**: chronological, relative timestamps. Click → navigate to related resource.

### 3. Domains List Page

```
┌─ Domains ────────────────────────────────────────────────────┐
│                                                              │
│  ┌────────────────────────────────────┐  ┌────────────────┐ │
│  │ 🔍 Search domains...              │  │ + Add Domain   │ │
│  └────────────────────────────────────┘  └────────────────┘ │
│                                                              │
│  ┌──────────────────────────────────────────────────────────┐│
│  │ □  Domain           Status    SSL     PHP   Disk   ⋯   ││
│  │────────────────────────────────────────────────────────── ││
│  │ □  example.com      ● Active  🔒 LE   8.3  245MB  ⋯   ││
│  │ □  blog.example.com ● Active  🔒 LE   8.3   48MB  ⋯   ││
│  │ □  shop.mysite.com  ● Active  🔒 LE   8.2  1.2GB  ⋯   ││
│  │ □  oldsite.net      ○ Susp.   ⚠ Exp.  8.1   89MB  ⋯   ││
│  │ □  test.dev         ● Active  ✕ None  8.3   12MB  ⋯   ││
│  │                                                          ││
│  │                     ◄ 1 2 3 ... 5 ►                     ││
│  └──────────────────────────────────────────────────────────┘│
└──────────────────────────────────────────────────────────────┘
```

- Search filters: status (all/active/suspended), has SSL (yes/no)
- SSL column shows: 🔒 with issuer (LE=Let's Encrypt, Custom) or ⚠ Expiring or ✕ None
- Row click → domain detail page
- Row `⋯` menu → Edit, Suspend, File Manager, Logs, Delete
- Bulk select → "Suspend selected", "Issue SSL for selected"

### 4. Domain Detail Page

```
┌─ Domains > example.com ─────────────────────────────────────┐
│                                                              │
│  example.com                              ● Active          │
│  /home/admin/domains/example.com/public                     │
│                                                              │
│  ┌──────┐ ┌──────┐ ┌──────┐ ┌──────┐ ┌──────┐ ┌──────┐   │
│  │ Files│ │ DNS  │ │ SSL  │ │ PHP  │ │ Logs │ │ Back │   │
│  └──────┘ └──────┘ └──────┘ └──────┘ └──────┘ └──────┘   │
│  ─────────────────────────────────────────────────────────── │
│                                                              │
│  (Tab content renders here)                                  │
│                                                              │
└──────────────────────────────────────────────────────────────┘
```

- Domain name as page title with status badge
- Document root path shown in monospace
- **Tabbed interface** for sub-sections: Overview, Files, DNS, SSL, PHP, Databases, FTP, Logs, Backups, Settings
- Tabs persist in URL (`/domains/1/dns`, `/domains/1/ssl`)
- Each tab loads its own data independently

### 5. File Manager Page

```
┌─ Files: example.com ─────────────────────────────────────────┐
│                                                              │
│  /public ──────────────────────────  [Upload] [+ New] [⋯]  │
│                                                              │
│  ┌──────────────────────────────────────────────────────────┐│
│  │ ← ..                                                     ││
│  │ 📁 css/                          —      2024-01-15 14:30 ││
│  │ 📁 js/                           —      2024-01-15 14:30 ││
│  │ 📁 images/                       —      2024-01-14 09:12 ││
│  │ 📁 wp-admin/                     —      2024-01-10 11:00 ││
│  │ 📁 wp-content/                   —      2024-01-15 16:45 ││
│  │ 📁 wp-includes/                  —      2024-01-10 11:00 ││
│  │ 📄 index.php                   1.2 KB   2024-01-10 11:00 ││
│  │ 📄 wp-config.php               3.8 KB   2024-01-12 08:30 ││
│  │ 📄 .htaccess                   0.4 KB   2024-01-10 11:00 ││
│  │ 📄 robots.txt                  0.1 KB   2024-01-10 11:00 ││
│  └──────────────────────────────────────────────────────────┘│
│                                                              │
│  12 items • 24.5 MB total                                    │
└──────────────────────────────────────────────────────────────┘
```

- **Breadcrumb path** at top (clickable segments: `/` > `public` > `css`)
- Actions toolbar: Upload (opens dropzone overlay), New File, New Folder, more (⋯: Extract, Compress)
- **Context menu** on right-click: Rename, Copy, Move, Delete, Download, Edit, Permissions
- **Drag-and-drop upload**: full-page dropzone overlay when dragging files over
- **Code editor**: click a text file → opens CodeMirror editor in a sheet panel (right side, 60% width)
  - Syntax highlighting, line numbers, word wrap toggle
  - Save (`Cmd+S`), undo/redo, find & replace
- **Image preview**: click an image → lightbox preview
- **Multi-select**: shift+click for range, cmd+click for individual
- Keyboard shortcuts: `Delete` to delete, `Enter` to open, `Backspace` to go up

### 6. Database Management

```
┌─ Databases ──────────────────────────────────────────────────┐
│                                                              │
│  ┌────────────────────────┐  ┌──────────────┐ ┌───────────┐│
│  │ 🔍 Search databases... │  │ + Create DB  │ │ phpMyAdmin││
│  └────────────────────────┘  └──────────────┘ └───────────┘│
│                                                              │
│  ┌──────────────────────────────────────────────────────────┐│
│  │  Database          Domain          Size     Users   ⋯   ││
│  │────────────────────────────────────────────────────────── ││
│  │  example_wp        example.com     156 MB   1       ⋯   ││
│  │  shop_main         shop.mysite.com 892 MB   2       ⋯   ││
│  │  blog_data         blog.example.   12 MB    1       ⋯   ││
│  └──────────────────────────────────────────────────────────┘│
│                                                              │
│  ┌──────────────────────────────────────────────────────────┐│
│  │ Database Users                        [+ Create User]    ││
│  │────────────────────────────────────────────────────────── ││
│  │  example_user     example_wp       All Privileges   ⋯   ││
│  │  shop_admin       shop_main        All Privileges   ⋯   ││
│  │  shop_readonly    shop_main        SELECT only      ⋯   ││
│  └──────────────────────────────────────────────────────────┘│
└──────────────────────────────────────────────────────────────┘
```

- Two sections: Databases + Database Users
- phpMyAdmin button opens in new tab with SSO
- Create DB sheet: name, linked domain (optional), create user simultaneously
- `⋯` menu: Backup, Restore, Repair, Optimize, Delete
- Connection string copy button (one-click copy)

### 7. SSL Management (Domain Tab)

```
┌─ SSL: example.com ───────────────────────────────────────────┐
│                                                              │
│  ┌──────────────────────────────────────────────────────────┐│
│  │ 🔒 SSL Certificate Active                               ││
│  │                                                          ││
│  │  Issuer:      Let's Encrypt                              ││
│  │  Issued:      2024-01-01                                 ││
│  │  Expires:     2024-03-31 (76 days remaining)             ││
│  │  Domains:     example.com, www.example.com               ││
│  │  Auto-Renew:  ● Enabled                                  ││
│  │                                                          ││
│  │  [Renew Now]  [Replace Certificate]  [Remove SSL]       ││
│  └──────────────────────────────────────────────────────────┘│
│                                                              │
│  Settings                                                    │
│  ┌──────────────────────────────────────────────────────────┐│
│  │  Force HTTPS           [━━━━●]  on                      ││
│  │  HTTP/2                [━━━━●]  on                      ││
│  │  HSTS                  [━━━━●]  on                      ││
│  │  HSTS Max-Age          [365] days                        ││
│  └──────────────────────────────────────────────────────────┘│
│                                                              │
│  ── or issue a new certificate ──                            │
│                                                              │
│  ┌──────────────────────────────────────────────────────────┐│
│  │  ○ Let's Encrypt (free, auto-renewing)                  ││
│  │  ○ Upload Custom Certificate                             ││
│  │                                                          ││
│  │  [Issue Certificate]                                     ││
│  └──────────────────────────────────────────────────────────┘│
└──────────────────────────────────────────────────────────────┘
```

### 8. Log Viewer

```
┌─ Logs: example.com ──────────────────────────────────────────┐
│                                                              │
│  [Access ▾] [All Status ▾] [Last 100 ▾]  🔍 Filter...  [⟳]│
│                                                              │
│  ┌──────────────────────────────────────────────────────────┐│
│  │ 192.168.1.1  GET /index.php        200  45ms   14:32:01 ││
│  │ 10.0.0.5     POST /wp-login.php    302  120ms  14:31:58 ││
│  │ 10.0.0.5     GET /wp-admin/        200  89ms   14:31:59 ││
│  │ 51.12.3.4    GET /xmlrpc.php       403  2ms    14:31:45 ││
│  │ 51.12.3.4    GET /wp-config.php    403  1ms    14:31:44 ││
│  │ 192.168.1.1  GET /style.css        304  3ms    14:31:40 ││
│  └──────────────────────────────────────────────────────────┘│
│                                                              │
│  ◉ Live tail                                                 │
└──────────────────────────────────────────────────────────────┘
```

- Log type selector: Access, Error, PHP-FPM, FTP
- Status code filter: All, 2xx, 3xx, 4xx, 5xx
- Text search with regex support
- Color coding: 2xx green, 3xx blue, 4xx amber, 5xx red
- **Live tail mode**: WebSocket stream, new lines appear at bottom with slide-in animation
- Monospace font (Geist Mono)
- Click IP → copy or ban option
- Click URL → view full request details in popover

### 9. Settings Page

```
┌─ Settings ───────────────────────────────────────────────────┐
│                                                              │
│  ┌────────────┐                                              │
│  │ General    │  General Settings                            │
│  │ Security   │  ───────────────────────                     │
│  │ Web Server │                                              │
│  │ PHP        │  Panel Hostname  ┌────────────────────────┐ │
│  │ Database   │                  │ panel.example.com      │ │
│  │ Backup     │                  └────────────────────────┘ │
│  │ Email      │                                              │
│  │ DNS        │  Panel Port      ┌────────────────────────┐ │
│  │ FTP        │                  │ 8443                   │ │
│  │ Advanced   │                  └────────────────────────┘ │
│  └────────────┘                                              │
│                  Timezone        ┌────────────────────────┐ │
│                                  │ Europe/Istanbul     ▾  │ │
│                                  └────────────────────────┘ │
│                                                              │
│                  [Save Changes]                              │
└──────────────────────────────────────────────────────────────┘
```

- **Left sub-navigation** for settings categories
- Each category is its own form section
- Settings changes require explicit "Save" (no auto-save for settings)
- Dangerous settings (panel port change, etc.) show warning before save

---

## Interaction Patterns

### Loading States
- **Page load**: full-page skeleton (matching the layout shape)
- **Table load**: skeleton rows (5 rows with pulsing animation)
- **Card load**: skeleton with shimmer effect
- **Button action**: spinner replaces button text, button disabled
- **Background action**: toast with spinner "Creating backup..."
- **Never**: blank white/black page or generic "Loading..."

### Optimistic Updates
- **Toggle switch** (e.g., enable/disable SSL, suspend domain): UI updates immediately, rolls back on error
- **Delete**: item fades out immediately, reappears on error with toast
- **Create**: new item appears in table immediately with "saving" indicator

### Animations & Transitions
Keep minimal and purposeful. No flashy animations.

- **Page transitions**: none (instant swap, no slide/fade between pages)
- **Sheet/dialog open**: slide in from right (sheet), fade + scale up (dialog) — 200ms
- **Sheet/dialog close**: reverse of open — 150ms
- **Toast appear**: slide up from bottom — 200ms
- **Table row appear**: fade in — 150ms (for new items)
- **Sidebar collapse**: width transition — 200ms ease
- **Hover states**: instant (no transition delay on hover backgrounds)
- **Status dot pulse**: infinite pulse for "pending/processing" states

### Keyboard Shortcuts
| Shortcut | Action |
|----------|--------|
| `Cmd+K` | Open command palette |
| `Cmd+B` | Toggle sidebar |
| `Cmd+S` | Save (in editors/forms) |
| `Escape` | Close sheet/dialog/palette |
| `?` | Show keyboard shortcuts help |
| `g d` | Go to Dashboard |
| `g o` | Go to Domains |
| `g b` | Go to Databases |
| `g f` | Go to Files |
| `g l` | Go to Logs |
| `g s` | Go to Settings |
| `c d` | Create domain |
| `c b` | Create database |

### Error Handling
- **Form validation**: inline errors below each field, red border on invalid field
- **API error (4xx)**: toast with specific message from API
- **Network error**: persistent toast "Connection lost. Retrying..." with retry button
- **Auth expired**: auto-refresh token, if fails → redirect to login with "Session expired" message
- **500 error**: toast "Something went wrong" + log error for admin review

---

## Real-Time Features

### WebSocket Architecture
Single WebSocket connection at `ws://panel:8443/api/ws`

**Channels the client subscribes to:**
```
server.metrics      → CPU, RAM, network (every 5s)
server.services     → Service status changes (event-driven)
domain.{id}.logs    → Live log tail for a domain
backup.{id}.progress → Backup progress percentage
```

**Message format:**
```json
{
  "channel": "server.metrics",
  "data": {
    "cpu": 23.5,
    "ram": { "used": 1258291200, "total": 4294967296 },
    "network": { "rx": 1024, "tx": 512 }
  }
}
```

### Auto-Refresh Strategy
- Dashboard stats: WebSocket (real-time)
- Domain list: refetch on window focus + every 30s
- Database list: refetch on window focus
- Service status: WebSocket (real-time)
- Logs: WebSocket when live tail enabled, manual refresh otherwise
- SSL status: refetch every 5 minutes
- Backup list: refetch on window focus + WebSocket for active backups

---

## Frontend File Structure

```
web/
├── src/
│   ├── main.tsx                      # Entry point
│   ├── App.tsx                       # Router setup
│   │
│   ├── components/
│   │   ├── ui/                       # Shadcn base components (auto-generated)
│   │   │   ├── button.tsx
│   │   │   ├── card.tsx
│   │   │   ├── dialog.tsx
│   │   │   ├── dropdown-menu.tsx
│   │   │   ├── input.tsx
│   │   │   ├── select.tsx
│   │   │   ├── sheet.tsx
│   │   │   ├── switch.tsx
│   │   │   ├── table.tsx
│   │   │   ├── tabs.tsx
│   │   │   ├── toast.tsx (sonner)
│   │   │   └── ...
│   │   │
│   │   ├── layout/                   # App shell components
│   │   │   ├── app-shell.tsx         # Main layout wrapper
│   │   │   ├── sidebar.tsx           # Navigation sidebar
│   │   │   ├── header.tsx            # Top header bar
│   │   │   ├── breadcrumb.tsx        # Dynamic breadcrumbs
│   │   │   ├── mobile-nav.tsx        # Mobile off-canvas nav
│   │   │   └── user-menu.tsx         # User avatar dropdown
│   │   │
│   │   ├── shared/                   # Reusable domain-specific components
│   │   │   ├── status-badge.tsx      # Universal status indicator
│   │   │   ├── data-table.tsx        # Generic data table wrapper
│   │   │   ├── data-table-toolbar.tsx
│   │   │   ├── data-table-pagination.tsx
│   │   │   ├── data-table-row-actions.tsx
│   │   │   ├── stat-card.tsx         # Metric card with sparkline
│   │   │   ├── empty-state.tsx       # No-data placeholder
│   │   │   ├── confirm-dialog.tsx    # Destructive action confirmation
│   │   │   ├── command-palette.tsx   # Cmd+K palette
│   │   │   ├── code-editor.tsx       # CodeMirror wrapper
│   │   │   ├── log-viewer.tsx        # Log display with coloring
│   │   │   ├── metric-chart.tsx      # Recharts area/line chart
│   │   │   ├── skeleton-table.tsx    # Table loading skeleton
│   │   │   ├── skeleton-cards.tsx    # Card grid loading skeleton
│   │   │   ├── copy-button.tsx       # Click-to-copy with feedback
│   │   │   ├── file-dropzone.tsx     # Drag-and-drop upload area
│   │   │   └── form-field.tsx        # Form field with label + error
│   │
│   ├── pages/
│   │   ├── auth/
│   │   │   ├── login.tsx
│   │   │   ├── two-factor.tsx
│   │   │   └── setup.tsx             # Initial admin setup (first run)
│   │   │
│   │   ├── dashboard/
│   │   │   ├── index.tsx             # Main dashboard page
│   │   │   ├── stat-cards.tsx        # CPU/RAM/Disk/Uptime cards
│   │   │   ├── services-card.tsx     # Service status list
│   │   │   ├── resource-chart.tsx    # 24h resource usage chart
│   │   │   └── activity-feed.tsx     # Recent activity list
│   │   │
│   │   ├── domains/
│   │   │   ├── index.tsx             # Domain list page
│   │   │   ├── columns.tsx           # TanStack Table column defs
│   │   │   ├── create-domain-sheet.tsx
│   │   │   ├── [id]/
│   │   │   │   ├── layout.tsx        # Domain detail layout (tabs)
│   │   │   │   ├── overview.tsx      # Domain overview tab
│   │   │   │   ├── files.tsx         # File manager tab
│   │   │   │   ├── dns.tsx           # DNS records tab
│   │   │   │   ├── ssl.tsx           # SSL management tab
│   │   │   │   ├── php.tsx           # PHP settings tab
│   │   │   │   ├── databases.tsx     # Linked databases tab
│   │   │   │   ├── ftp.tsx           # FTP accounts tab
│   │   │   │   ├── logs.tsx          # Domain logs tab
│   │   │   │   ├── backups.tsx       # Domain backups tab
│   │   │   │   └── settings.tsx      # Domain settings tab
│   │   │
│   │   ├── databases/
│   │   │   ├── index.tsx             # Database list page
│   │   │   ├── columns.tsx
│   │   │   ├── create-database-sheet.tsx
│   │   │   └── create-user-sheet.tsx
│   │   │
│   │   ├── backups/
│   │   │   ├── index.tsx             # Backup list page
│   │   │   ├── columns.tsx
│   │   │   └── create-backup-sheet.tsx
│   │   │
│   │   ├── logs/
│   │   │   └── index.tsx             # Server-level log viewer
│   │   │
│   │   └── settings/
│   │       ├── layout.tsx            # Settings layout (left sub-nav)
│   │       ├── general.tsx
│   │       ├── security.tsx
│   │       ├── web-server.tsx
│   │       ├── php.tsx
│   │       ├── database.tsx
│   │       ├── backup.tsx
│   │       └── advanced.tsx
│   │
│   ├── hooks/
│   │   ├── use-auth.ts               # Auth state + login/logout
│   │   ├── use-websocket.ts          # WebSocket connection manager
│   │   ├── use-server-metrics.ts     # Real-time CPU/RAM/network
│   │   ├── use-keyboard-shortcut.ts  # Register keyboard shortcuts
│   │   ├── use-clipboard.ts          # Copy-to-clipboard
│   │   ├── use-confirm.ts            # Confirmation dialog state
│   │   ├── use-debounce.ts           # Input debouncing
│   │   └── use-media-query.ts        # Responsive breakpoint detection
│   │
│   ├── api/
│   │   ├── client.ts                 # Axios instance + interceptors
│   │   ├── auth.ts                   # Auth API hooks (useLogin, etc.)
│   │   ├── domains.ts                # Domain API hooks
│   │   ├── dns.ts                    # DNS API hooks
│   │   ├── ssl.ts                    # SSL API hooks
│   │   ├── php.ts                    # PHP API hooks
│   │   ├── files.ts                  # File manager API hooks
│   │   ├── databases.ts              # Database API hooks
│   │   ├── ftp.ts                    # FTP API hooks
│   │   ├── backups.ts                # Backup API hooks
│   │   ├── logs.ts                   # Logs API hooks
│   │   ├── settings.ts               # Settings API hooks
│   │   └── dashboard.ts              # Dashboard API hooks
│   │
│   ├── stores/
│   │   ├── sidebar-store.ts          # Sidebar collapsed state
│   │   ├── theme-store.ts            # Dark/light mode
│   │   └── command-palette-store.ts  # Palette open state
│   │
│   ├── lib/
│   │   ├── utils.ts                  # cn() helper, formatBytes, etc.
│   │   ├── constants.ts              # Status colors, route paths
│   │   ├── validators.ts             # Zod schemas for forms
│   │   └── format.ts                 # Date, byte, number formatters
│   │
│   └── types/
│       ├── domain.ts                 # Domain, Subdomain types
│       ├── dns.ts                    # DnsRecord type
│       ├── ssl.ts                    # Certificate type
│       ├── database.ts               # Database, DatabaseUser types
│       ├── backup.ts                 # Backup type
│       ├── user.ts                   # User, AuthResponse types
│       ├── server.ts                 # ServerMetrics, ServiceStatus
│       └── api.ts                    # ApiError, PaginatedResponse
│
├── public/
│   ├── favicon.ico
│   └── logo.svg
│
├── index.html
├── package.json
├── tsconfig.json
├── tsconfig.app.json
├── tailwind.config.ts
├── vite.config.ts
├── components.json                   # Shadcn config
└── playwright.config.ts
```

---

## Routing Structure

```tsx
/login                          → Login page
/setup                          → Initial admin setup (first run only)

/                               → Dashboard
/domains                        → Domain list
/domains/:id                    → Domain detail (redirects to /overview)
/domains/:id/overview           → Domain overview tab
/domains/:id/files              → File manager tab
/domains/:id/files?path=/css    → File manager at specific path
/domains/:id/dns                → DNS records tab
/domains/:id/ssl                → SSL management tab
/domains/:id/php                → PHP settings tab
/domains/:id/databases          → Domain databases tab
/domains/:id/ftp                → FTP accounts tab
/domains/:id/logs               → Domain logs tab
/domains/:id/backups            → Domain backups tab
/domains/:id/settings           → Domain settings tab

/databases                      → All databases list
/backups                        → All backups list
/logs                           → Server log viewer

/settings                       → Settings (redirects to /general)
/settings/general               → General settings
/settings/security              → Security settings
/settings/web-server            → NGINX/Apache settings
/settings/php                   → PHP versions & config
/settings/database              → MySQL/MariaDB settings
/settings/backup                → Backup settings
/settings/advanced              → Advanced settings

# V1 additions
/users                          → User list
/users/:id                      → User detail
/plans                          → Service plan list
/email                          → Email accounts (all domains)
/domains/:id/email              → Domain email tab

# V2 additions
/wordpress                      → WP Toolkit dashboard
/wordpress/:id                  → WP instance detail
/domains/:id/git                → Git deployment tab
/domains/:id/nodejs             → Node.js app tab
/domains/:id/python             → Python app tab
/monitoring                     → Server monitoring
/api-keys                       → API key management

# V3 additions
/resellers                      → Reseller management
/reports                        → Reports dashboard
/migration                      → Migration wizard
/branding                       → White-label settings

# V4 additions
/extensions                     → Extension marketplace
/builder/:id                    → Website builder
/servers                        → Multi-server management
```

---

## Implementation Phases

### Phase F1 — Foundation (aligns with V0.4)
_Get the shell rendering with auth._

1. Initialize Vite + React + TypeScript project
2. Install and configure Tailwind CSS v4
3. Install Shadcn/ui, configure with Zinc + Pink theme
4. Install Geist font (Sans + Mono)
5. Add base Shadcn components: Button, Card, Input, Label, Dropdown Menu, Dialog, Sheet, Tabs, Table, Switch, Select, Tooltip, Popover, Separator, Skeleton, Badge
6. Create `app-shell.tsx` layout with sidebar + header + content area
7. Create `sidebar.tsx` with Shadcn Sidebar component (collapsible to icons, mobile drawer)
8. Create `header.tsx` with breadcrumbs + user menu
9. Set up React Router with layout routes
10. Create login page with form validation (Zod)
11. Set up Axios client with JWT interceptor
12. Set up TanStack Query provider
13. Set up Zustand stores (sidebar, theme)
14. Create protected route wrapper
15. Implement dark/light theme toggle (persisted in localStorage)
16. Create `status-badge.tsx` component
17. Create `empty-state.tsx` component
18. Set up Sonner toast provider
19. Responsive sidebar (collapsed on tablet, drawer on mobile)
20. Keyboard shortcut: `Cmd+B` toggle sidebar

### Phase F2 — Dashboard & Shared Components
_Build the dashboard and reusable components that all pages need._

1. Create `data-table.tsx` (TanStack Table wrapper with sorting, pagination, search, row selection, bulk actions, row actions menu, skeleton loading, empty state)
2. Create `stat-card.tsx` with sparkline support
3. Create `confirm-dialog.tsx` with type-to-confirm
4. Create `command-palette.tsx` (Cmd+K)
5. Create `copy-button.tsx`
6. Create `form-field.tsx` (label + input + error wrapper)
7. Create `skeleton-table.tsx` and `skeleton-cards.tsx`
8. Build dashboard page:
   - Stat cards row (CPU, RAM, Disk, Uptime) — mock data initially
   - Services card with status dots
   - Resource usage chart (Recharts area chart, time range selector)
   - Recent activity feed
9. Set up WebSocket hook (`use-websocket.ts`)
10. Create `use-server-metrics.ts` hook (WebSocket → stat cards)
11. Wire dashboard to real API + WebSocket

### Phase F3 — Domain Management
_Core domain CRUD and detail pages._

1. Domain list page with data table (columns: name, status, SSL, PHP, disk, actions)
2. Create domain sheet (form: domain name, create www, PHP version)
3. Domain detail layout with tabs
4. Domain overview tab (stats, quick actions)
5. Domain settings tab (document root, PHP version, suspend/delete)
6. DNS tab: record list table + create/edit record sheet
7. SSL tab: certificate status card + issue/upload forms + settings toggles
8. PHP tab: version selector + php.ini settings editor
9. FTP tab: FTP account list + create account sheet
10. Domain logs tab: access + error log viewer with filters
11. Domain backups tab: backup list + create backup

### Phase F4 — File Manager
_Complete file management experience._

1. File listing component (icon, name, size, date, permissions)
2. Breadcrumb path navigation
3. Directory navigation (click folder to enter, `..` to go up)
4. File/folder creation (dialog)
5. Rename (inline edit)
6. Delete (confirmation dialog)
7. Move/copy (dialog with path selector)
8. File upload with react-dropzone (drag-and-drop overlay, progress bar, multi-file)
9. File download (single file + zip folder)
10. Code editor sheet (CodeMirror with syntax highlighting, save, find/replace)
11. Image preview lightbox
12. Archive extraction (zip/tar)
13. Permission editor (chmod dialog)
14. Context menu (right-click)
15. Multi-select with shift+click and cmd+click
16. Keyboard shortcuts (Delete, Enter, Backspace)

### Phase F5 — Databases, Backups & Logs
_Remaining MVP pages._

1. Database list page with data table
2. Create database sheet (name, linked domain, create user option)
3. Create database user sheet (username, password generator, permissions checkboxes)
4. Database row actions (backup, restore, repair, optimize, delete)
5. phpMyAdmin SSO link
6. Connection string copy
7. Backup list page with data table
8. Create backup sheet (scope: full server or specific domain)
9. Restore backup flow (confirmation, selective restore options)
10. Backup download
11. Server-level log viewer page (select service, filter, search)
12. `log-viewer.tsx` component with syntax coloring by status code
13. Live tail mode with WebSocket
14. Settings pages (all categories with forms)

### Phase F6 — Polish & E2E Tests
_Final quality pass._

1. Review all pages for consistent spacing, alignment, typography
2. All empty states have illustrations + CTAs
3. All loading states use proper skeletons (no spinners blocking the page)
4. All error states show helpful messages
5. Mobile responsive pass: every page works on 375px width
6. Accessibility pass: all interactive elements keyboard-accessible, ARIA labels, focus rings
7. Playwright E2E tests:
   - Login flow (success + failure + 2FA)
   - Dashboard loads with data
   - Create domain → verify in list
   - Manage DNS records
   - Issue SSL certificate
   - File manager: upload, edit, delete
   - Create database
   - Create backup → verify in list
   - Settings: change value → save → verify persisted
8. Performance: ensure bundle size < 500KB gzipped (code-split pages)
9. Lighthouse audit: target 90+ on Performance, Accessibility, Best Practices

---

## Performance Budget

| Metric | Target |
|--------|--------|
| Initial JS bundle (gzipped) | < 150KB |
| Per-page chunk (gzipped) | < 50KB |
| Total bundle (gzipped) | < 500KB |
| First Contentful Paint | < 1.0s |
| Time to Interactive | < 1.5s |
| Largest Contentful Paint | < 2.0s |
| Layout shifts (CLS) | < 0.1 |

### Optimization Strategies
- **Code splitting**: React.lazy + Suspense per page route
- **Tree shaking**: import only used Shadcn components
- **Font loading**: `font-display: swap`, preload Geist Sans weight 400+600
- **Image optimization**: lazy load images in file manager
- **TanStack Query caching**: staleTime 30s for lists, 5min for settings
- **Virtualization**: `@tanstack/react-virtual` for long log lists (1000+ lines)
- **Debounce**: search inputs debounced 300ms
