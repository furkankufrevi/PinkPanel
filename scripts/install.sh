#!/usr/bin/env bash
set -euo pipefail

# PinkPanel Installer
# Usage: curl -fsSL https://get.pinkpanel.com | bash

PINKPANEL_VERSION="${PINKPANEL_VERSION:-latest}"
PINKPANEL_USER="pinkpanel"
PINKPANEL_HOME="/opt/pinkpanel"
PINKPANEL_DATA="/var/lib/pinkpanel"
PINKPANEL_LOG="/var/log/pinkpanel"
PINKPANEL_PORT="${PINKPANEL_PORT:-8443}"

BOLD='\033[1m'
GREEN='\033[0;32m'
YELLOW='\033[0;33m'
RED='\033[0;31m'
NC='\033[0m'

log()  { echo -e "${GREEN}[PinkPanel]${NC} $*"; }
warn() { echo -e "${YELLOW}[PinkPanel]${NC} $*"; }
err()  { echo -e "${RED}[PinkPanel]${NC} $*" >&2; }
die()  { err "$@"; exit 1; }

# ---------------------------------------------------------------------------
# Pre-flight checks
# ---------------------------------------------------------------------------

check_root() {
    if [[ $EUID -ne 0 ]]; then
        die "This installer must be run as root. Use: sudo bash install.sh"
    fi
}

check_os() {
    if [[ ! -f /etc/os-release ]]; then
        die "Cannot detect OS. Only Ubuntu 22.04/24.04 and Debian 11/12 are supported."
    fi
    source /etc/os-release

    case "$ID" in
        ubuntu)
            case "$VERSION_ID" in
                22.04|24.04) ;;
                *) die "Unsupported Ubuntu version: $VERSION_ID. Supported: 22.04, 24.04" ;;
            esac
            ;;
        debian)
            case "$VERSION_ID" in
                11|12) ;;
                *) die "Unsupported Debian version: $VERSION_ID. Supported: 11, 12" ;;
            esac
            ;;
        *)
            die "Unsupported OS: $ID. Only Ubuntu and Debian are supported."
            ;;
    esac

    log "Detected $PRETTY_NAME"
}

check_requirements() {
    local cpus mem_kb disk_avail_kb

    cpus=$(nproc)
    mem_kb=$(grep MemTotal /proc/meminfo | awk '{print $2}')
    disk_avail_kb=$(df / | tail -1 | awk '{print $4}')

    if (( cpus < 1 )); then
        die "At least 1 CPU core is required."
    fi
    if (( mem_kb < 900000 )); then
        die "At least 1 GB of RAM is required (found ~$((mem_kb / 1024)) MB)."
    fi
    if (( disk_avail_kb < 5000000 )); then
        die "At least 5 GB of free disk space is required (found ~$((disk_avail_kb / 1024 / 1024)) GB)."
    fi

    log "System requirements met: ${cpus} CPU(s), $((mem_kb / 1024)) MB RAM, $((disk_avail_kb / 1024 / 1024)) GB disk"
}

# ---------------------------------------------------------------------------
# Package installation
# ---------------------------------------------------------------------------

install_packages() {
    log "Updating package lists..."
    export DEBIAN_FRONTEND=noninteractive
    apt-get update -qq

    log "Installing system packages..."
    apt-get install -y -qq \
        nginx \
        mariadb-server \
        bind9 bind9utils \
        vsftpd \
        certbot \
        curl \
        wget \
        tar \
        zip \
        unzip \
        ufw \
        > /dev/null

    # PHP — install multiple versions via ondrej PPA (Ubuntu) or sury (Debian)
    if ! command -v add-apt-repository &>/dev/null; then
        apt-get install -y -qq software-properties-common > /dev/null
    fi

    source /etc/os-release
    if [[ "$ID" == "ubuntu" ]]; then
        add-apt-repository -y ppa:ondrej/php > /dev/null 2>&1
    else
        curl -sSLo /tmp/debsuryorg-archive-keyring.deb https://packages.sury.org/debsuryorg-archive-keyring.deb
        dpkg -i /tmp/debsuryorg-archive-keyring.deb > /dev/null
        echo "deb [signed-by=/usr/share/keyrings/deb.sury.org-archive-keyring.gpg] https://packages.sury.org/php/ $(lsb_release -sc) main" \
            > /etc/apt/sources.list.d/sury-php.list
        apt-get update -qq
    fi

    # Install PHP 8.3 as default
    apt-get install -y -qq \
        php8.3-fpm php8.3-cli php8.3-common php8.3-mysql php8.3-xml \
        php8.3-curl php8.3-gd php8.3-mbstring php8.3-zip php8.3-bcmath \
        php8.3-intl php8.3-readline php8.3-opcache \
        > /dev/null

    log "System packages installed"
}

# ---------------------------------------------------------------------------
# System user & directories
# ---------------------------------------------------------------------------

setup_user() {
    if ! id "$PINKPANEL_USER" &>/dev/null; then
        useradd -r -s /usr/sbin/nologin -d "$PINKPANEL_HOME" "$PINKPANEL_USER"
        log "Created system user: $PINKPANEL_USER"
    fi
}

setup_directories() {
    mkdir -p "$PINKPANEL_HOME/bin"
    mkdir -p "$PINKPANEL_DATA"
    mkdir -p "$PINKPANEL_LOG"
    mkdir -p /var/run/pinkpanel
    mkdir -p /var/backups/pinkpanel
    mkdir -p /var/www
    mkdir -p /etc/pinkpanel
    mkdir -p "$PINKPANEL_DATA/acme"

    chown -R "$PINKPANEL_USER:$PINKPANEL_USER" "$PINKPANEL_HOME" "$PINKPANEL_DATA" "$PINKPANEL_LOG" /var/run/pinkpanel /var/backups/pinkpanel
    log "Directories created"
}

# ---------------------------------------------------------------------------
# Download & install binaries
# ---------------------------------------------------------------------------

install_binaries() {
    local arch
    arch=$(uname -m)
    case "$arch" in
        x86_64)  arch="amd64" ;;
        aarch64) arch="arm64" ;;
        *)       die "Unsupported architecture: $arch" ;;
    esac

    if [[ "$PINKPANEL_VERSION" == "latest" ]]; then
        log "Downloading latest PinkPanel binaries (linux/$arch)..."
    else
        log "Downloading PinkPanel $PINKPANEL_VERSION (linux/$arch)..."
    fi

    # For now, check if binaries are available locally (dev mode)
    if [[ -f "./dist/pinkpanel" ]]; then
        log "Using local binaries from ./dist/"
        cp ./dist/pinkpanel "$PINKPANEL_HOME/bin/pinkpanel"
        cp ./dist/pinkpanel-agent "$PINKPANEL_HOME/bin/pinkpanel-agent"
        cp ./dist/pinkpanel-cli "$PINKPANEL_HOME/bin/pinkpanel-cli"
    else
        die "Binary download not yet implemented. Build locally with 'make build' and run installer from project root."
    fi

    chmod +x "$PINKPANEL_HOME/bin/"*
    ln -sf "$PINKPANEL_HOME/bin/pinkpanel-cli" /usr/local/bin/pinkpanel

    log "Binaries installed to $PINKPANEL_HOME/bin/"
}

# ---------------------------------------------------------------------------
# Configuration
# ---------------------------------------------------------------------------

setup_config() {
    if [[ -f /etc/pinkpanel/pinkpanel.yml ]]; then
        log "Configuration already exists, skipping..."
        return
    fi

    # Generate JWT secret
    openssl rand -base64 64 > /etc/pinkpanel/jwt.key
    chmod 600 /etc/pinkpanel/jwt.key
    chown "$PINKPANEL_USER:$PINKPANEL_USER" /etc/pinkpanel/jwt.key

    cat > /etc/pinkpanel/pinkpanel.yml <<EOF
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
EOF

    chown "$PINKPANEL_USER:$PINKPANEL_USER" /etc/pinkpanel/pinkpanel.yml
    log "Configuration created at /etc/pinkpanel/pinkpanel.yml"
}

# ---------------------------------------------------------------------------
# Systemd services
# ---------------------------------------------------------------------------

setup_systemd() {
    cat > /etc/systemd/system/pinkpanel.service <<EOF
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

# Security hardening
NoNewPrivileges=true
ProtectSystem=strict
ProtectHome=true
ReadWritePaths=${PINKPANEL_DATA} ${PINKPANEL_LOG} /var/run/pinkpanel
PrivateTmp=true

[Install]
WantedBy=multi-user.target
EOF

    cat > /etc/systemd/system/pinkpanel-agent.service <<EOF
[Unit]
Description=PinkPanel Agent
After=network.target

[Service]
Type=simple
User=root
WorkingDirectory=${PINKPANEL_HOME}
ExecStart=${PINKPANEL_HOME}/bin/pinkpanel-agent
Restart=on-failure
RestartSec=5
LimitNOFILE=65535

[Install]
WantedBy=multi-user.target
EOF

    systemctl daemon-reload
    log "Systemd services created"
}

# ---------------------------------------------------------------------------
# BIND config repair helper
# ---------------------------------------------------------------------------

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

# ---------------------------------------------------------------------------
# Service configuration
# ---------------------------------------------------------------------------

configure_services() {
    # Enable and start MariaDB
    systemctl enable --now mariadb > /dev/null 2>&1
    log "MariaDB enabled"

    # Enable NGINX
    systemctl enable nginx > /dev/null 2>&1
    log "NGINX enabled"

    # Configure and enable BIND9
    mkdir -p /etc/bind/zones
    chown bind:bind /etc/bind/zones 2>/dev/null || true

    # Write authoritative-only BIND options
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
        # Repair: remove stray orphan "};" lines left by old zone removal bug
        repair_named_conf_local
    fi

    # Ensure named.conf includes named.conf.local
    if [[ -f /etc/bind/named.conf ]] && ! grep -q "named.conf.local" /etc/bind/named.conf; then
        echo 'include "/etc/bind/named.conf.local";' >> /etc/bind/named.conf
    fi

    # Generate rndc key if missing
    if [[ ! -f /etc/bind/rndc.key ]]; then
        rndc-confgen -a -b 256 > /dev/null 2>&1 || true
        chown bind:bind /etc/bind/rndc.key 2>/dev/null || true
    fi

    # Fix ownership
    chown bind:bind /etc/bind/named.conf.local 2>/dev/null || true
    chown -R bind:bind /etc/bind/zones/ 2>/dev/null || true

    # Validate config before starting
    if command -v named-checkconf &>/dev/null; then
        if ! named-checkconf > /dev/null 2>&1; then
            warn "BIND config check failed — review /etc/bind/"
        fi
    fi

    # Reset failed state (in case BIND crashed previously) then start
    systemctl reset-failed named 2>/dev/null || systemctl reset-failed bind9 2>/dev/null || true
    systemctl enable named > /dev/null 2>&1 || systemctl enable bind9 > /dev/null 2>&1 || true
    systemctl restart named 2>/dev/null || systemctl restart bind9 2>/dev/null || true

    if systemctl is-active --quiet named 2>/dev/null || systemctl is-active --quiet bind9 2>/dev/null; then
        log "BIND9 configured and running"
    else
        warn "BIND9 configured but failed to start — check: journalctl -u named -n 20"
    fi

    # Enable vsftpd
    systemctl enable vsftpd > /dev/null 2>&1
    log "vsftpd enabled"

    # Enable PHP-FPM
    systemctl enable php8.3-fpm > /dev/null 2>&1
    log "PHP-FPM 8.3 enabled"
}

# ---------------------------------------------------------------------------
# Firewall
# ---------------------------------------------------------------------------

setup_firewall() {
    if ! command -v ufw &>/dev/null; then
        warn "ufw not found, skipping firewall configuration"
        return
    fi

    ufw --force reset > /dev/null 2>&1
    ufw default deny incoming > /dev/null 2>&1
    ufw default allow outgoing > /dev/null 2>&1

    ufw allow 22/tcp comment "SSH" > /dev/null 2>&1
    ufw allow 80/tcp comment "HTTP" > /dev/null 2>&1
    ufw allow 443/tcp comment "HTTPS" > /dev/null 2>&1
    ufw allow "$PINKPANEL_PORT/tcp" comment "PinkPanel" > /dev/null 2>&1
    ufw allow 21/tcp comment "FTP" > /dev/null 2>&1
    ufw allow 53 comment "DNS" > /dev/null 2>&1
    ufw allow 40000:50000/tcp comment "FTP Passive" > /dev/null 2>&1

    ufw --force enable > /dev/null 2>&1
    log "Firewall configured (ports: 22, 53, 80, 443, $PINKPANEL_PORT, 21, 40000-50000)"
}

# ---------------------------------------------------------------------------
# Start services
# ---------------------------------------------------------------------------

start_services() {
    systemctl start pinkpanel-agent
    log "PinkPanel Agent started"

    systemctl start pinkpanel
    log "PinkPanel Server started"

    systemctl enable pinkpanel pinkpanel-agent > /dev/null 2>&1
}

# ---------------------------------------------------------------------------
# MariaDB secure installation
# ---------------------------------------------------------------------------

secure_mariadb() {
    # Ensure root uses unix_socket auth (default on MariaDB 10.4+)
    # The agent runs as root, so unix_socket works natively without a password file.
    mysql -u root <<-EOSQL || true
        ALTER USER 'root'@'localhost' IDENTIFIED VIA unix_socket;
        DELETE FROM mysql.user WHERE User='';
        DELETE FROM mysql.user WHERE User='root' AND Host NOT IN ('localhost', '127.0.0.1', '::1');
        DROP DATABASE IF EXISTS test;
        DELETE FROM mysql.db WHERE Db='test' OR Db='test\\_%';
        FLUSH PRIVILEGES;
EOSQL
    log "MariaDB secured (using unix_socket auth for root)"
}

# ---------------------------------------------------------------------------
# Print completion
# ---------------------------------------------------------------------------

print_complete() {
    local ip
    ip=$(hostname -I | awk '{print $1}')

    echo ""
    echo -e "${BOLD}═══════════════════════════════════════════════════════${NC}"
    echo -e "${BOLD}  ${GREEN}PinkPanel installed successfully!${NC}"
    echo -e "${BOLD}═══════════════════════════════════════════════════════${NC}"
    echo ""
    echo -e "  Access your panel at:"
    echo -e "  ${BOLD}http://${ip}:${PINKPANEL_PORT}${NC}"
    echo ""
    echo -e "  Complete setup by creating your admin account"
    echo -e "  in the browser on first visit."
    echo ""
    echo -e "  Useful commands:"
    echo -e "    ${BOLD}pinkpanel status${NC}      — Check panel status"
    echo -e "    ${BOLD}systemctl status pinkpanel${NC}"
    echo -e "    ${BOLD}journalctl -u pinkpanel -f${NC}"
    echo ""
    echo -e "  Config: /etc/pinkpanel/pinkpanel.yml"
    echo -e "  Data:   ${PINKPANEL_DATA}/"
    echo -e "  Logs:   ${PINKPANEL_LOG}/"
    echo ""
    echo -e "${BOLD}═══════════════════════════════════════════════════════${NC}"
}

# ---------------------------------------------------------------------------
# Main
# ---------------------------------------------------------------------------

main() {
    echo ""
    echo -e "${BOLD}${GREEN}PinkPanel Installer${NC}"
    echo ""

    check_root
    check_os
    check_requirements

    install_packages
    setup_user
    setup_directories
    install_binaries
    setup_config
    setup_systemd
    configure_services
    secure_mariadb
    setup_firewall
    start_services

    print_complete
}

main "$@"
