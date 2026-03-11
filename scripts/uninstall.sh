#!/usr/bin/env bash
set -euo pipefail

# PinkPanel Uninstaller

PINKPANEL_USER="pinkpanel"
PINKPANEL_HOME="/opt/pinkpanel"
PINKPANEL_DATA="/var/lib/pinkpanel"
PINKPANEL_LOG="/var/log/pinkpanel"

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[0;33m'
BOLD='\033[1m'
NC='\033[0m'

log()  { echo -e "${GREEN}[PinkPanel]${NC} $*"; }
warn() { echo -e "${YELLOW}[PinkPanel]${NC} $*"; }

if [[ $EUID -ne 0 ]]; then
    echo -e "${RED}This script must be run as root.${NC}" >&2
    exit 1
fi

echo ""
echo -e "${BOLD}${RED}PinkPanel Uninstaller${NC}"
echo ""
echo -e "${YELLOW}This will remove PinkPanel binaries, services, and configuration.${NC}"
echo -e "${YELLOW}Your website files in /var/www/ and databases will NOT be removed.${NC}"
echo ""

read -rp "Are you sure you want to uninstall PinkPanel? (y/N) " confirm
if [[ "$confirm" != "y" && "$confirm" != "Y" ]]; then
    echo "Cancelled."
    exit 0
fi

echo ""

# Stop services
log "Stopping services..."
systemctl stop pinkpanel 2>/dev/null || true
systemctl stop pinkpanel-agent 2>/dev/null || true
systemctl disable pinkpanel 2>/dev/null || true
systemctl disable pinkpanel-agent 2>/dev/null || true

# Remove systemd units
log "Removing systemd services..."
rm -f /etc/systemd/system/pinkpanel.service
rm -f /etc/systemd/system/pinkpanel-agent.service
systemctl daemon-reload

# Remove binaries
log "Removing binaries..."
rm -rf "$PINKPANEL_HOME"
rm -f /usr/local/bin/pinkpanel

# Remove configuration
log "Removing configuration..."
rm -rf /etc/pinkpanel

# Remove runtime files
log "Removing runtime files..."
rm -rf /var/run/pinkpanel

# Optionally remove data
read -rp "Remove PinkPanel database and logs? (y/N) " remove_data
if [[ "$remove_data" == "y" || "$remove_data" == "Y" ]]; then
    rm -rf "$PINKPANEL_DATA"
    rm -rf "$PINKPANEL_LOG"
    log "Data and logs removed"
else
    warn "Data preserved at $PINKPANEL_DATA"
    warn "Logs preserved at $PINKPANEL_LOG"
fi

# Remove user
if id "$PINKPANEL_USER" &>/dev/null; then
    userdel "$PINKPANEL_USER" 2>/dev/null || true
    log "System user removed"
fi

echo ""
echo -e "${GREEN}PinkPanel has been uninstalled.${NC}"
echo -e "Website files in /var/www/ and databases were preserved."
echo ""
