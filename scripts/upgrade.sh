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

[[ $EUID -eq 0 ]] || die "Run as root: sudo bash upgrade.sh"

print_banner
echo -e "  ${BOLD}${GREEN}Upgrader${NC}"
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

# ── Ensure required packages ────────────────
install_missing_packages() {
    local missing=()
    command -v zip &>/dev/null || missing+=("zip")
    command -v unzip &>/dev/null || missing+=("unzip")
    if (( ${#missing[@]} > 0 )); then
        log "Installing missing packages: ${missing[*]}..."
        export DEBIAN_FRONTEND=noninteractive
        apt-get update -qq
        apt-get install -y -qq "${missing[@]}" > /dev/null
    fi
}

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

    # Repair named.conf.local: remove stray "};" lines left by old zone removal bug
    if [[ -f /etc/bind/named.conf.local ]]; then
        local conf="/etc/bind/named.conf.local"
        # Rebuild config: keep comments and zone blocks, discard orphan "};" lines
        local tmp_conf
        tmp_conf=$(mktemp)
        local in_zone=0
        while IFS= read -r line; do
            if [[ "$line" =~ ^zone[[:space:]] ]]; then
                in_zone=1
                echo "$line" >> "$tmp_conf"
            elif (( in_zone )); then
                echo "$line" >> "$tmp_conf"
                if [[ "$line" =~ ^\}\; ]]; then
                    in_zone=0
                fi
            elif [[ "$line" =~ ^[[:space:]]*\}\;[[:space:]]*$ ]]; then
                # Stray orphan "};" — skip it
                warn "Removed stray '};' from named.conf.local"
            else
                echo "$line" >> "$tmp_conf"
            fi
        done < "$conf"
        cp "$tmp_conf" "$conf"
        rm -f "$tmp_conf"
        chown bind:bind "$conf" 2>/dev/null || true
    fi

    # Fix zone file ownership
    if [[ -d /etc/bind/zones ]]; then
        chown -R bind:bind /etc/bind/zones 2>/dev/null || true
    fi

    # Test config before restarting
    if command -v named-checkconf &>/dev/null; then
        if named-checkconf > /dev/null 2>&1; then
            log "BIND config check passed"
        else
            warn "BIND config check failed — review /etc/bind/"
        fi
    fi

    # Restart BIND
    systemctl restart named 2>/dev/null || systemctl restart bind9 2>/dev/null || true

    log "BIND9 configuration updated"
}

install_missing_packages
fix_bind

# Start services
log "Starting PinkPanel services..."
systemctl start pinkpanel-agent
sleep 1
# Verify agent socket is accessible
if [[ -S /var/run/pinkpanel/agent.sock ]]; then
    log "Agent socket ready"
else
    warn "Agent socket not found — check: journalctl -u pinkpanel-agent -n 20"
fi
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

print_banner
echo -e "  ${BOLD}${GREEN}Upgraded successfully!${NC}"
echo ""
echo -e "  $CURRENT → $NEW"
echo ""
