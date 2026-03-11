#!/usr/bin/env bash
set -euo pipefail

# PinkPanel Uninstall Script
# Run on your server as root:
#   curl -fsSL https://raw.githubusercontent.com/furkankufrevi/PinkPanel/master/scripts/uninstall.sh | sudo bash

PINKPANEL_USER="pinkpanel"
PINKPANEL_HOME="/opt/pinkpanel"
PINKPANEL_DATA="/var/lib/pinkpanel"
PINKPANEL_LOG="/var/log/pinkpanel"

BOLD='\033[1m'
GREEN='\033[0;32m'
YELLOW='\033[0;33m'
RED='\033[0;31m'
PINK='\033[1;35m'
NC='\033[0m'

log()  { echo -e "${GREEN}[PinkPanel]${NC} $*"; }
warn() { echo -e "${YELLOW}[PinkPanel]${NC} $*"; }
err()  { echo -e "${RED}[PinkPanel]${NC} $*" >&2; }
die()  { err "$@"; exit 1; }

print_banner() {
    echo ""
    echo -e "${PINK}    ____  _       __   ____                  __${NC}"
    echo -e "${PINK}   / __ \\(_)___  / /__/ __ \\____ _____  ___  / /${NC}"
    echo -e "${PINK}  / /_/ / / __ \\/ //_/ /_/ / __ \`/ __ \\/ _ \\/ / ${NC}"
    echo -e "${PINK} / ____/ / / / / ,< / ____/ /_/ / / / /  __/ /  ${NC}"
    echo -e "${PINK}/_/   /_/_/ /_/_/|_/_/    \\__,_/_/ /_/\\___/_/   ${NC}"
    echo ""
}

[[ $EUID -eq 0 ]] || die "Run as root: sudo bash uninstall.sh"

print_banner
echo -e "  ${BOLD}${RED}Uninstaller${NC}"
echo ""
echo -e "This will remove PinkPanel from your server."
echo -e "By default it will ${BOLD}NOT${NC} remove:"
echo -e "  - System packages (nginx, mariadb, php, bind9, vsftpd)"
echo -e "  - Website files in /var/www/"
echo -e "  - MariaDB databases"
echo ""

# ── Confirmation ─────────────────────────────

read -rp "Are you sure you want to uninstall PinkPanel? (yes/no): " CONFIRM
if [[ "$CONFIRM" != "yes" ]]; then
    log "Uninstall cancelled."
    exit 0
fi

echo ""

# ── Optional cleanup prompts ─────────────────

REMOVE_DATA="no"
read -rp "Remove PinkPanel data (database, config, backups)? (yes/no): " REMOVE_DATA

REMOVE_WEBSITES="no"
read -rp "Remove all website files in /var/www/? (yes/no): " REMOVE_WEBSITES

REMOVE_DNS="no"
read -rp "Remove DNS zone files in /etc/bind/zones/? (yes/no): " REMOVE_DNS

echo ""

# ── Stop and disable services ────────────────

log "Stopping PinkPanel services..."
systemctl stop pinkpanel 2>/dev/null || true
systemctl stop pinkpanel-agent 2>/dev/null || true
systemctl disable pinkpanel 2>/dev/null || true
systemctl disable pinkpanel-agent 2>/dev/null || true
log "Services stopped"

# ── Remove systemd units ────────────────────

log "Removing systemd units..."
rm -f /etc/systemd/system/pinkpanel.service
rm -f /etc/systemd/system/pinkpanel-agent.service
systemctl daemon-reload
log "Systemd units removed"

# ── Remove binaries ─────────────────────────

log "Removing binaries..."
rm -rf "$PINKPANEL_HOME"
rm -f /usr/local/bin/pinkpanel
log "Binaries removed"

# ── Remove runtime files ────────────────────

log "Removing runtime files..."
rm -rf /var/run/pinkpanel
rm -rf "$PINKPANEL_LOG"
log "Runtime files removed"

# ── Remove PinkPanel NGINX vhosts ───────────

log "Removing PinkPanel-managed NGINX vhosts..."
for conf in /etc/nginx/sites-enabled/*.conf; do
    [[ -f "$conf" ]] || continue
    rm -f "$conf"
done
for conf in /etc/nginx/sites-available/*.conf; do
    [[ -f "$conf" ]] || continue
    rm -f "$conf"
done
if systemctl is-active --quiet nginx 2>/dev/null; then
    systemctl reload nginx 2>/dev/null || true
fi
log "NGINX vhosts removed"

# ── Remove PHP-FPM pools ───────────────────

log "Removing PinkPanel-managed PHP-FPM pools..."
for pooldir in /etc/php/*/fpm/pool.d/; do
    [[ -d "$pooldir" ]] || continue
    for pool in "$pooldir"*.conf; do
        [[ -f "$pool" ]] || continue
        poolname=$(basename "$pool")
        # Keep the default www.conf pool
        [[ "$poolname" == "www.conf" ]] && continue
        rm -f "$pool"
    done
done
# Reload running PHP-FPM instances
for fpm in /run/php/php*-fpm.pid; do
    [[ -f "$fpm" ]] || continue
    version=$(echo "$fpm" | grep -oP 'php\K[0-9.]+')
    systemctl reload "php${version}-fpm" 2>/dev/null || true
done
log "PHP-FPM pools removed"

# ── Conditional: remove data ────────────────

if [[ "$REMOVE_DATA" == "yes" ]]; then
    log "Removing PinkPanel data..."
    rm -rf "$PINKPANEL_DATA"
    rm -rf /etc/pinkpanel
    rm -rf /var/backups/pinkpanel
    log "Data removed"
else
    warn "Keeping PinkPanel data:"
    warn "  Config:  /etc/pinkpanel/"
    warn "  Data:    $PINKPANEL_DATA/"
    warn "  Backups: /var/backups/pinkpanel/"
fi

# ── Conditional: remove websites ────────────

if [[ "$REMOVE_WEBSITES" == "yes" ]]; then
    log "Removing website files..."
    rm -rf /var/www/*
    log "Website files removed"
else
    warn "Keeping website files in /var/www/"
fi

# ── Conditional: remove DNS zones ───────────

if [[ "$REMOVE_DNS" == "yes" ]]; then
    log "Removing DNS zone files..."
    rm -rf /etc/bind/zones/*
    # Remove PinkPanel zone blocks from named.conf.local
    if [[ -f /etc/bind/named.conf.local ]]; then
        sed -i '/^zone "/,/^};$/d' /etc/bind/named.conf.local
    fi
    rndc reconfig 2>/dev/null || systemctl reload named 2>/dev/null || systemctl reload bind9 2>/dev/null || true
    log "DNS zones removed"
else
    warn "Keeping DNS zone files in /etc/bind/zones/"
fi

# ── Remove system user ──────────────────────

if id "$PINKPANEL_USER" &>/dev/null; then
    log "Removing pinkpanel system user..."
    userdel "$PINKPANEL_USER" 2>/dev/null || true
    log "User removed"
fi

# ── Done ────────────────────────────────────

print_banner
echo -e "  ${BOLD}${GREEN}Uninstalled successfully.${NC}"
echo ""
echo -e "  System packages (nginx, mariadb, php, bind9, vsftpd)"
echo -e "  were ${BOLD}not${NC} removed. To remove them manually:"
echo ""
echo -e "    apt remove --purge nginx mariadb-server php8.3-fpm \\"
echo -e "      bind9 vsftpd certbot ufw"
echo ""
