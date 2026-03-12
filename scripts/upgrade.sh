#!/usr/bin/env bash
set -euo pipefail

# PinkPanel Upgrade Script
# Run on your server as root:
#   curl -fsSL https://raw.githubusercontent.com/furkankufrevi/PinkPanel/master/scripts/upgrade.sh | sudo bash

REPO="https://github.com/furkankufrevi/PinkPanel.git"
BUILD_DIR="/tmp/pinkpanel-build"
PINKPANEL_HOME="/opt/pinkpanel"
PINKPANEL_DATA="/var/lib/pinkpanel"
VERSION_FILE="/etc/pinkpanel/version"

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

# ── Version helpers ────────────────────────
# Strips pre-release suffix and converts "0.3.0-alpha" → "000300"
# so we can do numeric comparisons.
version_to_num() {
    local ver="${1%%-*}"  # strip -alpha, -beta, etc.
    local major minor patch
    IFS='.' read -r major minor patch <<< "$ver"
    printf '%d%02d%02d' "${major:-0}" "${minor:-0}" "${patch:-0}"
}

# Returns true if $1 < $2
version_lt() {
    [[ $(version_to_num "$1") -lt $(version_to_num "$2") ]]
}

get_current_version() {
    # Try version file first, then binary, then fall back to "0.0.0"
    if [[ -f "$VERSION_FILE" ]]; then
        cat "$VERSION_FILE"
    elif [[ -x "$PINKPANEL_HOME/bin/pinkpanel-cli" ]]; then
        "$PINKPANEL_HOME/bin/pinkpanel-cli" version 2>/dev/null | awk '{print $2}' || echo "0.0.0"
    elif [[ -x "$PINKPANEL_HOME/bin/pinkpanel" ]]; then
        "$PINKPANEL_HOME/bin/pinkpanel" version 2>/dev/null | awk '{print $2}' || echo "0.0.0"
    else
        echo "0.0.0"
    fi
}

save_version() {
    echo "$1" > "$VERSION_FILE"
}

# ══════════════════════════════════════════════
# Version-specific migrations
# Each function runs only if upgrading FROM a version older than its target.
# Add new migrations at the bottom with incrementing version numbers.
# ══════════════════════════════════════════════

migrate_to_0_3_0() {
    log "Running migrations for 0.3.0..."

    # --- ACME data dir: /etc/pinkpanel/acme → /var/lib/pinkpanel/acme ---
    local old_acme="/etc/pinkpanel/acme"
    local new_acme="$PINKPANEL_DATA/acme"
    mkdir -p "$new_acme"
    if [[ -d "$old_acme" ]] && ls "$old_acme"/* &>/dev/null 2>&1; then
        log "  Migrating ACME data to $new_acme..."
        cp -a "$old_acme"/* "$new_acme/" 2>/dev/null || true
        rm -rf "$old_acme"
    fi
    chown -R pinkpanel:pinkpanel "$new_acme" 2>/dev/null || true

    # --- MySQL: password auth → unix_socket ---
    if [[ -f /etc/pinkpanel/mysql.cnf ]]; then
        log "  Migrating MySQL root auth to unix_socket..."
        local migrate_sql="ALTER USER 'root'@'localhost' IDENTIFIED VIA unix_socket; FLUSH PRIVILEGES;"
        if mysql --defaults-file=/etc/pinkpanel/mysql.cnf -e "$migrate_sql" 2>/dev/null; then
            log "  MySQL auth migrated via password file"
        elif mysql -u root -e "$migrate_sql" 2>/dev/null; then
            log "  MySQL auth migrated via unix_socket"
        else
            warn "  Could not migrate MySQL auth — manual intervention may be needed"
        fi
        rm -f /etc/pinkpanel/mysql.cnf
        log "  Removed legacy mysql.cnf"
    fi

    log "Migrations for 0.3.0 complete"
}

# ── Run all applicable migrations ──────────
run_migrations() {
    local from_version="$1"

    # Add migration entries here in order. Format:
    #   version_lt "$from_version" "X.Y.Z" && migrate_to_X_Y_Z
    version_lt "$from_version" "0.3.0" && migrate_to_0_3_0

    # Future migrations go here:
    # version_lt "$from_version" "0.4.0" && migrate_to_0_4_0
    # version_lt "$from_version" "0.5.0" && migrate_to_0_5_0
}

# ══════════════════════════════════════════════
# Main upgrade flow
# ══════════════════════════════════════════════

[[ $EUID -eq 0 ]] || die "Run as root: sudo bash upgrade.sh"

print_banner
echo -e "  ${BOLD}${GREEN}Upgrader${NC}"
echo ""

# Get current version
CURRENT=$(get_current_version)
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

# Get new version from the freshly built binary
NEW_VERSION=$("$BUILD_DIR/dist/pinkpanel-cli" version 2>/dev/null | awk '{print $2}' || echo "unknown")
log "New version: $NEW_VERSION"

# Check if already up to date
if [[ "$CURRENT" == "$NEW_VERSION" ]]; then
    warn "Already running $CURRENT — no upgrade needed"
    rm -rf "$BUILD_DIR"
    exit 0
fi

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

    # Reset failed state (BIND may have crashed previously and given up retrying)
    systemctl reset-failed named 2>/dev/null || systemctl reset-failed bind9 2>/dev/null || true

    # Restart BIND
    systemctl restart named 2>/dev/null || systemctl restart bind9 2>/dev/null || true

    if systemctl is-active --quiet named 2>/dev/null || systemctl is-active --quiet bind9 2>/dev/null; then
        log "BIND9 running"
    else
        warn "BIND9 failed to start — check: journalctl -u named -n 20"
    fi

    log "BIND9 configuration updated"
}

install_missing_packages
fix_bind

# ── Run version-specific migrations ────────
run_migrations "$CURRENT"

# ── Save new version ──────────────────────
save_version "$NEW_VERSION"

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

# Confirm version from running binary
INSTALLED=$(get_current_version)

print_banner
echo -e "  ${BOLD}${GREEN}Upgraded successfully!${NC}"
echo ""
echo -e "  ${CURRENT} → ${BOLD}${INSTALLED}${NC}"
echo ""
