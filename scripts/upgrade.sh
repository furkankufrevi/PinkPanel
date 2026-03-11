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
