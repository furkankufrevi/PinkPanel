#!/usr/bin/env bash
set -euo pipefail

# PinkPanel Deploy Script
# Run on Ubuntu server as root:
#   curl -fsSL https://raw.githubusercontent.com/furkankufrevi/PinkPanel/master/scripts/deploy.sh | sudo bash

REPO="https://github.com/furkankufrevi/PinkPanel.git"
BUILD_DIR="/tmp/pinkpanel-build"
PINKPANEL_USER="pinkpanel"
PINKPANEL_HOME="/opt/pinkpanel"
PINKPANEL_DATA="/var/lib/pinkpanel"
PINKPANEL_LOG="/var/log/pinkpanel"
PINKPANEL_PORT="${PINKPANEL_PORT:-8443}"

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

# ── Pre-flight ────────────────────────────────

check_root() {
    [[ $EUID -eq 0 ]] || die "Run as root: sudo bash deploy.sh"
}

check_os() {
    [[ -f /etc/os-release ]] || die "Cannot detect OS."
    source /etc/os-release
    case "$ID-$VERSION_ID" in
        ubuntu-22.04|ubuntu-24.04|debian-11|debian-12) ;;
        *) die "Unsupported: $PRETTY_NAME. Need Ubuntu 22.04/24.04 or Debian 11/12." ;;
    esac
    log "OS: $PRETTY_NAME"
}

check_resources() {
    local mem_mb=$(( $(grep MemTotal /proc/meminfo | awk '{print $2}') / 1024 ))
    local disk_gb=$(( $(df / | tail -1 | awk '{print $4}') / 1024 / 1024 ))
    (( mem_mb >= 900 )) || die "Need 1GB+ RAM (found ${mem_mb}MB)"
    (( disk_gb >= 5 ))  || die "Need 5GB+ disk (found ${disk_gb}GB)"
    log "Resources OK: ${mem_mb}MB RAM, ${disk_gb}GB disk"
}

# ── Install build deps ────────────────────────

install_build_deps() {
    log "Installing build dependencies..."
    export DEBIAN_FRONTEND=noninteractive
    apt-get update -qq

    # Go
    if ! command -v go &>/dev/null; then
        log "Installing Go..."
        local GO_VERSION="1.23.6"
        local ARCH=$(dpkg --print-architecture)
        curl -fsSL "https://go.dev/dl/go${GO_VERSION}.linux-${ARCH}.tar.gz" | tar -C /usr/local -xzf -
        export PATH="/usr/local/go/bin:$PATH"
        echo 'export PATH="/usr/local/go/bin:$PATH"' > /etc/profile.d/golang.sh
    fi
    log "Go $(go version | awk '{print $3}')"

    # Node.js
    if ! command -v node &>/dev/null || [[ $(node -v | cut -d. -f1 | tr -d v) -lt 18 ]]; then
        log "Installing Node.js 20..."
        curl -fsSL https://deb.nodesource.com/setup_20.x | bash - > /dev/null 2>&1
        apt-get install -y -qq nodejs > /dev/null
    fi
    log "Node $(node -v)"

    apt-get install -y -qq git curl wget tar make > /dev/null
}

# ── Install runtime packages ─────────────────

install_packages() {
    log "Installing server packages..."
    export DEBIAN_FRONTEND=noninteractive

    apt-get install -y -qq \
        nginx \
        mariadb-server \
        bind9 bind9utils \
        vsftpd \
        certbot \
        zip \
        unzip \
        ufw \
        > /dev/null

    # PHP repo
    if ! command -v add-apt-repository &>/dev/null; then
        apt-get install -y -qq software-properties-common > /dev/null
    fi

    source /etc/os-release
    if [[ "$ID" == "ubuntu" ]]; then
        add-apt-repository -y ppa:ondrej/php > /dev/null 2>&1
    else
        if [[ ! -f /usr/share/keyrings/deb.sury.org-archive-keyring.gpg ]]; then
            curl -sSLo /tmp/debsuryorg-archive-keyring.deb https://packages.sury.org/debsuryorg-archive-keyring.deb
            dpkg -i /tmp/debsuryorg-archive-keyring.deb > /dev/null
            echo "deb [signed-by=/usr/share/keyrings/deb.sury.org-archive-keyring.gpg] https://packages.sury.org/php/ $(lsb_release -sc) main" \
                > /etc/apt/sources.list.d/sury-php.list
        fi
        apt-get update -qq
    fi

    apt-get install -y -qq \
        php8.3-fpm php8.3-cli php8.3-common php8.3-mysql php8.3-xml \
        php8.3-curl php8.3-gd php8.3-mbstring php8.3-zip php8.3-bcmath \
        php8.3-intl php8.3-readline php8.3-opcache \
        > /dev/null

    log "Server packages installed"
}

# ── Clone & build ─────────────────────────────

build_pinkpanel() {
    log "Cloning PinkPanel..."
    rm -rf "$BUILD_DIR"
    git clone --depth 1 "$REPO" "$BUILD_DIR" 2>/dev/null

    cd "$BUILD_DIR"
    log "Building PinkPanel..."
    export PATH="/usr/local/go/bin:$PATH"
    make build 2>&1 | tail -5

    log "Build complete"
}

# ── Setup user & directories ─────────────────

setup_system() {
    if ! id "$PINKPANEL_USER" &>/dev/null; then
        useradd -r -s /usr/sbin/nologin -d "$PINKPANEL_HOME" "$PINKPANEL_USER"
    fi

    mkdir -p "$PINKPANEL_HOME/bin" "$PINKPANEL_DATA" "$PINKPANEL_LOG" \
             /var/run/pinkpanel /var/backups/pinkpanel /var/www /etc/pinkpanel

    # Stop existing services before replacing binaries (avoids "Text file busy")
    systemctl stop pinkpanel 2>/dev/null || true
    systemctl stop pinkpanel-agent 2>/dev/null || true

    # Install binaries
    cp "$BUILD_DIR/dist/pinkpanel"       "$PINKPANEL_HOME/bin/"
    cp "$BUILD_DIR/dist/pinkpanel-agent" "$PINKPANEL_HOME/bin/"
    cp "$BUILD_DIR/dist/pinkpanel-cli"   "$PINKPANEL_HOME/bin/"
    chmod +x "$PINKPANEL_HOME/bin/"*
    ln -sf "$PINKPANEL_HOME/bin/pinkpanel-cli" /usr/local/bin/pinkpanel

    chown -R "$PINKPANEL_USER:$PINKPANEL_USER" \
        "$PINKPANEL_HOME" "$PINKPANEL_DATA" "$PINKPANEL_LOG" \
        /var/run/pinkpanel /var/backups/pinkpanel

    log "Binaries installed"
}

# ── Configuration ─────────────────────────────

setup_config() {
    if [[ -f /etc/pinkpanel/pinkpanel.yml ]]; then
        log "Config exists, keeping current"
        return
    fi

    openssl rand -base64 64 > /etc/pinkpanel/jwt.key
    chmod 600 /etc/pinkpanel/jwt.key
    chown "$PINKPANEL_USER:$PINKPANEL_USER" /etc/pinkpanel/jwt.key

    cat > /etc/pinkpanel/pinkpanel.yml <<CONF
server:
  host: "0.0.0.0"
  port: ${PINKPANEL_PORT}

database:
  path: "${PINKPANEL_DATA}/pinkpanel.db"

security:
  jwt_secret_file: "/etc/pinkpanel/jwt.key"
  access_token_ttl: 15m
  refresh_token_ttl: 168h
  bcrypt_cost: 12

agent:
  socket: "/var/run/pinkpanel/agent.sock"

logging:
  level: "info"
  file: "${PINKPANEL_LOG}/pinkpanel.log"
  max_size: 100
  max_backups: 5
  max_age: 30
CONF

    chown "$PINKPANEL_USER:$PINKPANEL_USER" /etc/pinkpanel/pinkpanel.yml
    log "Config created"
}

# ── Systemd ───────────────────────────────────

setup_systemd() {
    cat > /etc/systemd/system/pinkpanel.service <<SVC
[Unit]
Description=PinkPanel Server
After=network.target mariadb.service
Wants=pinkpanel-agent.service

[Service]
Type=simple
User=${PINKPANEL_USER}
Group=${PINKPANEL_USER}
WorkingDirectory=${PINKPANEL_HOME}
ExecStart=${PINKPANEL_HOME}/bin/pinkpanel
Restart=on-failure
RestartSec=5
LimitNOFILE=65535
Environment=PINKPANEL_CONFIG=/etc/pinkpanel/pinkpanel.yml
NoNewPrivileges=true
ProtectSystem=strict
ProtectHome=true
ReadWritePaths=${PINKPANEL_DATA} ${PINKPANEL_LOG} /var/run/pinkpanel
PrivateTmp=true

[Install]
WantedBy=multi-user.target
SVC

    cat > /etc/systemd/system/pinkpanel-agent.service <<SVC
[Unit]
Description=PinkPanel Agent
After=network.target

[Service]
Type=simple
User=root
WorkingDirectory=${PINKPANEL_HOME}
ExecStart=${PINKPANEL_HOME}/bin/pinkpanel-agent --socket /var/run/pinkpanel/agent.sock
Restart=on-failure
RestartSec=5
LimitNOFILE=65535

[Install]
WantedBy=multi-user.target
SVC

    systemctl daemon-reload
    log "Systemd services created"
}

# ── MariaDB ───────────────────────────────────

secure_mariadb() {
    systemctl enable --now mariadb > /dev/null 2>&1

    if [[ ! -f /etc/pinkpanel/mysql.cnf ]]; then
        local root_pass=$(openssl rand -base64 24)
        mysql -u root <<-SQL || true
            ALTER USER 'root'@'localhost' IDENTIFIED BY '${root_pass}';
            DELETE FROM mysql.user WHERE User='';
            DELETE FROM mysql.user WHERE User='root' AND Host NOT IN ('localhost','127.0.0.1','::1');
            DROP DATABASE IF EXISTS test;
            FLUSH PRIVILEGES;
SQL
        cat > /etc/pinkpanel/mysql.cnf <<CNF
[client]
user=root
password=${root_pass}
CNF
        chmod 600 /etc/pinkpanel/mysql.cnf
        log "MariaDB secured"
    fi
}

# ── BIND config repair helper ────────────────

repair_named_conf_local() {
    local conf="/etc/bind/named.conf.local"
    [[ -f "$conf" ]] || return
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
}

# ── Configure BIND9 ──────────────────────────

setup_bind() {
    mkdir -p /etc/bind/zones
    chown bind:bind /etc/bind/zones 2>/dev/null || true

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

    # Ensure named.conf.local exists and is clean
    if [[ ! -f /etc/bind/named.conf.local ]]; then
        echo "// PinkPanel managed zones" > /etc/bind/named.conf.local
    else
        repair_named_conf_local
    fi

    # Generate rndc key if missing (needed for rndc reconfig/reload)
    if [[ ! -f /etc/bind/rndc.key ]]; then
        rndc-confgen -a -b 256 > /dev/null 2>&1 || true
        chown bind:bind /etc/bind/rndc.key 2>/dev/null || true
    fi

    # Ensure main named.conf includes named.conf.local
    if [[ -f /etc/bind/named.conf ]] && ! grep -q "named.conf.local" /etc/bind/named.conf; then
        echo 'include "/etc/bind/named.conf.local";' >> /etc/bind/named.conf
    fi

    # Fix ownership
    chown bind:bind /etc/bind/named.conf.local 2>/dev/null || true
    chown -R bind:bind /etc/bind/zones/ 2>/dev/null || true

    # Validate config before starting
    if command -v named-checkconf &>/dev/null; then
        if named-checkconf > /dev/null 2>&1; then
            log "BIND config check passed"
        else
            warn "BIND config check failed — review /etc/bind/"
        fi
    fi

    log "BIND9 configured (authoritative only, rndc enabled)"
}

# ── Enable services ───────────────────────────

enable_services() {
    systemctl enable --now nginx > /dev/null 2>&1
    # Reset failed state in case BIND crashed previously
    systemctl reset-failed named 2>/dev/null || systemctl reset-failed bind9 2>/dev/null || true
    systemctl enable named > /dev/null 2>&1 || systemctl enable bind9 > /dev/null 2>&1 || true
    systemctl restart named 2>/dev/null || systemctl restart bind9 2>/dev/null || true
    if systemctl is-active --quiet named 2>/dev/null || systemctl is-active --quiet bind9 2>/dev/null; then
        log "BIND9 running"
    else
        warn "BIND9 failed to start — check: journalctl -u named -n 20"
    fi
    systemctl enable --now vsftpd > /dev/null 2>&1
    systemctl enable --now php8.3-fpm > /dev/null 2>&1
    log "Services enabled"
}

# ── Firewall ──────────────────────────────────

setup_firewall() {
    command -v ufw &>/dev/null || return
    ufw --force reset > /dev/null 2>&1
    ufw default deny incoming > /dev/null 2>&1
    ufw default allow outgoing > /dev/null 2>&1
    ufw allow 22/tcp > /dev/null 2>&1
    ufw allow 80/tcp > /dev/null 2>&1
    ufw allow 443/tcp > /dev/null 2>&1
    ufw allow "$PINKPANEL_PORT/tcp" > /dev/null 2>&1
    ufw allow 21/tcp > /dev/null 2>&1
    ufw allow 53 > /dev/null 2>&1
    ufw allow 40000:50000/tcp > /dev/null 2>&1
    ufw --force enable > /dev/null 2>&1
    log "Firewall configured"
}

# ── Start PinkPanel ───────────────────────────

start_pinkpanel() {
    systemctl enable --now pinkpanel-agent 2>/dev/null
    sleep 1
    systemctl enable --now pinkpanel 2>/dev/null
    log "PinkPanel started"
}

# ── Cleanup ───────────────────────────────────

cleanup() {
    rm -rf "$BUILD_DIR"
}

# ── Print result ──────────────────────────────

print_done() {
    local ip=$(hostname -I | awk '{print $1}')

    print_banner
    echo -e "  ${BOLD}${GREEN}Deployed successfully!${NC}"
    echo ""
    echo -e "  Open in browser:"
    echo -e "  ${BOLD}http://${ip}:${PINKPANEL_PORT}${NC}"
    echo ""
    echo -e "  Create your admin account on first visit."
    echo ""
    echo -e "  Commands:"
    echo -e "    systemctl status pinkpanel"
    echo -e "    systemctl status pinkpanel-agent"
    echo -e "    journalctl -u pinkpanel -f"
    echo ""
    echo -e "  Config:  /etc/pinkpanel/pinkpanel.yml"
    echo -e "  Data:    ${PINKPANEL_DATA}/"
    echo -e "  Logs:    ${PINKPANEL_LOG}/"
    echo -e "  MySQL:   /etc/pinkpanel/mysql.cnf"
    echo ""
}

# ── Main ──────────────────────────────────────

print_banner() {
    echo ""
    echo -e "${PINK}    ____  _       __   ____                  __${NC}"
    echo -e "${PINK}   / __ \\(_)___  / /__/ __ \\____ _____  ___  / /${NC}"
    echo -e "${PINK}  / /_/ / / __ \\/ //_/ /_/ / __ \`/ __ \\/ _ \\/ / ${NC}"
    echo -e "${PINK} / ____/ / / / / ,< / ____/ /_/ / / / /  __/ /  ${NC}"
    echo -e "${PINK}/_/   /_/_/ /_/_/|_/_/    \\__,_/_/ /_/\\___/_/   ${NC}"
    echo ""
}

main() {
    print_banner
    echo -e "  ${BOLD}${GREEN}Deployer${NC}"
    echo -e "  Repository: ${REPO}"
    echo ""

    check_root
    check_os
    check_resources

    install_build_deps
    install_packages
    build_pinkpanel
    setup_system
    setup_config
    setup_systemd
    secure_mariadb
    setup_bind
    enable_services
    setup_firewall
    start_pinkpanel
    cleanup

    print_done
}

main "$@"
