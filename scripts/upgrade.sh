#!/usr/bin/env bash
set -euo pipefail

# PinkPanel Upgrade Script
# Run on your server as root:
#   curl -fsSL https://raw.githubusercontent.com/furkankufrevi/PinkPanel/master/scripts/upgrade.sh | sudo bash

REPO="https://github.com/furkankufrevi/PinkPanel.git"
BUILD_DIR="/tmp/pinkpanel-build"
PINKPANEL_HOME="/opt/pinkpanel"

BOLD='\033[1m'
GREEN='\033[0;32m'
YELLOW='\033[0;33m'
RED='\033[0;31m'
NC='\033[0m'

log()  { echo -e "${GREEN}[PinkPanel]${NC} $*"; }
warn() { echo -e "${YELLOW}[PinkPanel]${NC} $*"; }
err()  { echo -e "${RED}[PinkPanel]${NC} $*" >&2; }
die()  { err "$@"; exit 1; }

[[ $EUID -eq 0 ]] || die "Run as root: sudo bash upgrade.sh"

echo ""
echo -e "${BOLD}${GREEN}PinkPanel Upgrader${NC}"
echo ""

# Get current version
CURRENT="unknown"
if [[ -x "$PINKPANEL_HOME/bin/pinkpanel" ]]; then
    CURRENT=$("$PINKPANEL_HOME/bin/pinkpanel" version 2>/dev/null || echo "unknown")
fi
log "Current version: $CURRENT"

# Ensure build tools exist
export PATH="/usr/local/go/bin:$PATH"
command -v go &>/dev/null || die "Go not found. Was PinkPanel installed with deploy.sh?"
command -v node &>/dev/null || die "Node.js not found. Was PinkPanel installed with deploy.sh?"

# Clone & build
log "Cloning latest PinkPanel..."
rm -rf "$BUILD_DIR"
git clone --depth 1 "$REPO" "$BUILD_DIR" 2>/dev/null

cd "$BUILD_DIR"
log "Building..."
make build 2>&1 | tail -5
log "Build complete"

# Stop services
log "Stopping PinkPanel services..."
systemctl stop pinkpanel 2>/dev/null || true
systemctl stop pinkpanel-agent 2>/dev/null || true

# Backup old binaries
if [[ -d "$PINKPANEL_HOME/bin" ]]; then
    cp -a "$PINKPANEL_HOME/bin" "$PINKPANEL_HOME/bin.bak.$(date +%Y%m%d%H%M%S)"
fi

# Install new binaries
log "Installing new binaries..."
cp "$BUILD_DIR/dist/pinkpanel"       "$PINKPANEL_HOME/bin/"
cp "$BUILD_DIR/dist/pinkpanel-agent" "$PINKPANEL_HOME/bin/"
cp "$BUILD_DIR/dist/pinkpanel-cli"   "$PINKPANEL_HOME/bin/"
chmod +x "$PINKPANEL_HOME/bin/"*

# ── Fix BIND9 configuration ─────────────────
fix_bind() {
    log "Checking BIND9 configuration..."

    # Ensure zones directory exists
    mkdir -p /etc/bind/zones
    chown bind:bind /etc/bind/zones 2>/dev/null || true

    # Fix named.conf.options — authoritative only, allow all queries
    cat > /etc/bind/named.conf.options <<'BINDOPTS'
options {
    directory "/var/cache/bind";
    listen-on { any; };
    listen-on-v6 { any; };
    allow-query { any; };
    recursion no;
    allow-recursion { none; };
    dnssec-validation auto;
    version "not disclosed";
};
BINDOPTS

    # Generate rndc key if missing
    if [[ ! -f /etc/bind/rndc.key ]]; then
        rndc-confgen -a -b 256 > /dev/null 2>&1 || true
        chown bind:bind /etc/bind/rndc.key 2>/dev/null || true
    fi

    # Ensure named.conf includes named.conf.local
    if [[ -f /etc/bind/named.conf ]] && ! grep -q "named.conf.local" /etc/bind/named.conf; then
        echo 'include "/etc/bind/named.conf.local";' >> /etc/bind/named.conf
    fi

    # Fix zone file ownership
    if [[ -d /etc/bind/zones ]]; then
        chown -R bind:bind /etc/bind/zones 2>/dev/null || true
    fi

    # Restart BIND
    systemctl restart named 2>/dev/null || systemctl restart bind9 2>/dev/null || true

    log "BIND9 configuration updated"
}

fix_bind

# Start services
log "Starting PinkPanel services..."
systemctl start pinkpanel-agent
systemctl start pinkpanel

# Cleanup
rm -rf "$BUILD_DIR"

# Verify
sleep 2
if systemctl is-active --quiet pinkpanel; then
    log "PinkPanel is running"
else
    err "PinkPanel failed to start — check: journalctl -u pinkpanel -n 50"
fi

NEW="unknown"
if [[ -x "$PINKPANEL_HOME/bin/pinkpanel" ]]; then
    NEW=$("$PINKPANEL_HOME/bin/pinkpanel" version 2>/dev/null || echo "unknown")
fi

echo ""
echo -e "${BOLD}═══════════════════════════════════════════════════════${NC}"
echo -e "${BOLD}  ${GREEN}PinkPanel upgraded successfully!${NC}"
echo -e "${BOLD}═══════════════════════════════════════════════════════${NC}"
echo ""
echo -e "  $CURRENT → $NEW"
echo ""
echo -e "${BOLD}═══════════════════════════════════════════════════════${NC}"
