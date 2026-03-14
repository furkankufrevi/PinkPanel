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
        fail2ban \
        libmodsecurity3 \
        git \
        postfix \
        dovecot-core dovecot-imapd dovecot-lmtpd dovecot-sieve dovecot-managesieved \
        opendkim opendkim-tools \
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
    mkdir -p "$PINKPANEL_DATA/git"

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
# phpMyAdmin
# ---------------------------------------------------------------------------

setup_phpmyadmin() {
    log "Setting up phpMyAdmin..."

    # Install phpMyAdmin non-interactively
    export DEBIAN_FRONTEND=noninteractive
    echo "phpmyadmin phpmyadmin/dbconfig-install boolean false" | debconf-set-selections
    echo "phpmyadmin phpmyadmin/reconfigure-webserver multiselect none" | debconf-set-selections
    apt-get install -y -qq phpmyadmin > /dev/null 2>&1 || {
        warn "phpMyAdmin package not available — skipping"
        return
    }

    # Create token directory (world-readable so www-data PHP can access)
    mkdir -p /var/lib/pinkpanel/pma-tokens
    chown www-data:www-data /var/lib/pinkpanel/pma-tokens
    chmod 755 /var/lib/pinkpanel/pma-tokens

    # Patch phpMyAdmin config.inc.php directly for signon auth
    local pma_config="/etc/phpmyadmin/config.inc.php"
    if [[ -f "$pma_config" ]]; then
        sed -i "/auth_type.*=.*'cookie'/d" "$pma_config"
        sed -i "/auth_type.*=.*'signon'/d" "$pma_config"
        sed -i "/SignonSession/d" "$pma_config"
        sed -i "/SignonURL/d" "$pma_config"
        cat >> "$pma_config" <<'PMACFG'

/* PinkPanel signon auth */
$cfg['Servers'][1]['auth_type'] = 'signon';
$cfg['Servers'][1]['SignonSession'] = 'PinkPanelPMA';
$cfg['Servers'][1]['SignonURL'] = '/phpmyadmin/signon.php';
$cfg['Servers'][1]['host'] = 'localhost';
PMACFG
    fi

    # Also keep conf.d as fallback
    mkdir -p /etc/phpmyadmin/conf.d
    cat > /etc/phpmyadmin/conf.d/pinkpanel.php <<'PMACONF'
<?php
$cfg['Servers'][1]['auth_type'] = 'signon';
$cfg['Servers'][1]['SignonSession'] = 'PinkPanelPMA';
$cfg['Servers'][1]['SignonURL'] = '/phpmyadmin/signon.php';
$cfg['Servers'][1]['host'] = 'localhost';
PMACONF

    # Deploy signon.php script
    cat > /usr/share/phpmyadmin/signon.php <<'SIGNON'
<?php
$token = isset($_GET['token']) ? preg_replace('/[^a-f0-9]/', '', $_GET['token']) : '';
if (!$token) {
    die('Missing token');
}

$tokenFile = '/var/lib/pinkpanel/pma-tokens/' . $token . '.json';
if (!file_exists($tokenFile)) {
    die('Invalid or expired token. Please try again from the panel.');
}

$data = json_decode(file_get_contents($tokenFile), true);
@unlink($tokenFile);

if (!$data || empty($data['username']) || empty($data['password'])) {
    die('Invalid token data');
}

session_name('PinkPanelPMA');
session_start();
$_SESSION['PMA_single_signon_user'] = $data['username'];
$_SESSION['PMA_single_signon_password'] = $data['password'];
$_SESSION['PMA_single_signon_host'] = 'localhost';

$db = isset($data['database']) ? $data['database'] : '';
header('Location: /phpmyadmin/index.php' . ($db ? '?db=' . urlencode($db) : ''));
exit;
SIGNON
    chown www-data:www-data /usr/share/phpmyadmin/signon.php

    # Create NGINX config for phpMyAdmin
    local php_sock
    php_sock=$(ls /run/php/php*-fpm.sock 2>/dev/null | head -1)
    [[ -z "$php_sock" ]] && php_sock="/run/php/php-fpm.sock"

    cat > /etc/nginx/snippets/phpmyadmin.conf <<PMA
location /phpmyadmin/ {
    alias /usr/share/phpmyadmin/;
    index index.php;
}

location ~ ^/phpmyadmin/(.+\\.php)\$ {
    alias /usr/share/phpmyadmin/\$1;
    fastcgi_pass unix:${php_sock};
    fastcgi_index index.php;
    fastcgi_param SCRIPT_FILENAME /usr/share/phpmyadmin/\$1;
    include fastcgi_params;
}
PMA

    # Include in default NGINX server block (both sites-available and sites-enabled)
    for f in /etc/nginx/sites-available/default /etc/nginx/sites-enabled/default; do
        if [[ -f "$f" ]] && ! grep -q "phpmyadmin" "$f"; then
            sed -i '/server_name _;/a\\n\tinclude snippets/phpmyadmin.conf;' "$f"
        fi
    done

    log "phpMyAdmin configured at /phpmyadmin/ with auto-login"
}

# ---------------------------------------------------------------------------
# ModSecurity
# ---------------------------------------------------------------------------

setup_modsecurity() {
    log "Configuring ModSecurity..."

    # Install NGINX ModSecurity connector if available
    apt-get install -y -qq libnginx-mod-http-modsecurity > /dev/null 2>&1 || true

    # Create modsecurity config directory
    mkdir -p /etc/nginx/modsecurity

    # Create main modsecurity config if not present
    if [[ ! -f /etc/nginx/modsecurity/modsecurity.conf ]]; then
        if [[ -f /etc/modsecurity/modsecurity.conf-recommended ]]; then
            cp /etc/modsecurity/modsecurity.conf-recommended /etc/nginx/modsecurity/modsecurity.conf
            # Switch from DetectionOnly to On
            sed -i 's/SecRuleEngine DetectionOnly/SecRuleEngine On/' /etc/nginx/modsecurity/modsecurity.conf
        else
            cat > /etc/nginx/modsecurity/modsecurity.conf <<'MODSEC'
SecRuleEngine On
SecRequestBodyAccess On
SecResponseBodyAccess Off
SecRequestBodyLimit 13107200
SecRequestBodyNoFilesLimit 131072
SecResponseBodyLimit 524288
SecTmpDir /tmp/
SecDataDir /tmp/
SecAuditEngine RelevantOnly
SecAuditLogRelevantStatus "^(?:5|4(?!04))"
SecAuditLogParts ABIJDEFHZ
SecAuditLogType Serial
SecAuditLog /var/log/nginx/modsec_audit.log
SecArgumentSeparator &
SecCookieFormat 0
SecUnicodeMapFile unicode.mapping 20127
MODSEC
        fi
    fi

    # Install OWASP CRS if available
    if [[ ! -d /etc/nginx/modsecurity/crs ]]; then
        apt-get install -y -qq modsecurity-crs > /dev/null 2>&1 || true
        if [[ -d /usr/share/modsecurity-crs ]]; then
            ln -sf /usr/share/modsecurity-crs /etc/nginx/modsecurity/crs
            # Include CRS rules in modsecurity config
            if ! grep -q "crs" /etc/nginx/modsecurity/modsecurity.conf 2>/dev/null; then
                cat >> /etc/nginx/modsecurity/modsecurity.conf <<'CRS'

# OWASP Core Rule Set
Include /etc/nginx/modsecurity/crs/rules/*.conf
CRS
            fi
        fi
    fi

    log "ModSecurity configured"
}

# Fail2ban
# ---------------------------------------------------------------------------

setup_fail2ban() {
    if ! command -v fail2ban-client &>/dev/null; then
        warn "fail2ban not installed, skipping"
        return
    fi

    log "Configuring Fail2ban..."

    # Create PinkPanel filter for panel authentication failures
    cat > /etc/fail2ban/filter.d/pinkpanel.conf <<'FILTER'
[Definition]
failregex = ^.*"POST /api/auth/login".*<HOST>.*401.*$
ignoreregex =
FILTER

    # Create PinkPanel jail configuration
    cat > /etc/fail2ban/jail.d/pinkpanel.conf <<'JAIL'
[pinkpanel]
enabled = true
port = http,https
filter = pinkpanel
logpath = /var/log/pinkpanel/server.log
maxretry = 10
findtime = 600
bantime = 3600
action = %(action_)s

[sshd]
enabled = true
maxretry = 5
findtime = 600
bantime = 3600
JAIL

    systemctl enable fail2ban > /dev/null 2>&1
    systemctl restart fail2ban > /dev/null 2>&1

    log "Fail2ban configured with PinkPanel jail"
}

# ---------------------------------------------------------------------------
# Mail server (Postfix + Dovecot + OpenDKIM)
# ---------------------------------------------------------------------------

setup_mail() {
    log "Configuring mail server..."

    local hostname
    hostname=$(hostname -f 2>/dev/null || hostname)

    # Create vmail user for virtual mailboxes
    if ! id vmail &>/dev/null; then
        groupadd -g 5000 vmail
        useradd -g vmail -u 5000 vmail -d /var/mail/vhosts -s /usr/sbin/nologin
    fi
    mkdir -p /var/mail/vhosts
    chown -R vmail:vmail /var/mail/vhosts

    # Create empty virtual map files
    touch /etc/postfix/virtual-mailbox-domains
    touch /etc/postfix/virtual-mailbox-maps
    touch /etc/postfix/virtual

    # ── Postfix main.cf ──
    cat > /etc/postfix/main.cf <<POSTFIX
# PinkPanel Postfix Configuration
smtpd_banner = \$myhostname ESMTP
biff = no
append_dot_mydomain = no
readme_directory = no

myhostname = ${hostname}
mydomain = ${hostname}
myorigin = \$mydomain
mydestination = localhost
mynetworks = 127.0.0.0/8 [::ffff:127.0.0.0]/104 [::1]/128

# Virtual mailbox settings
virtual_mailbox_domains = /etc/postfix/virtual-mailbox-domains
virtual_mailbox_maps = hash:/etc/postfix/virtual-mailbox-maps
virtual_mailbox_base = /var/mail/vhosts
virtual_transport = lmtp:unix:private/dovecot-lmtp
virtual_alias_maps = hash:/etc/postfix/virtual
virtual_minimum_uid = 5000
virtual_uid_maps = static:5000
virtual_gid_maps = static:5000

# TLS
smtpd_tls_cert_file = /etc/ssl/certs/ssl-cert-snakeoil.pem
smtpd_tls_key_file = /etc/ssl/private/ssl-cert-snakeoil.key
smtpd_tls_security_level = may
smtp_tls_security_level = may

# SASL (Dovecot)
smtpd_sasl_auth_enable = yes
smtpd_sasl_type = dovecot
smtpd_sasl_path = private/auth
smtpd_sasl_security_options = noanonymous
smtpd_sasl_local_domain = \$myhostname

# Restrictions
smtpd_recipient_restrictions = permit_sasl_authenticated, permit_mynetworks, reject_unauth_destination

# OpenDKIM milter
milter_protocol = 6
milter_default_action = accept
smtpd_milters = inet:localhost:8891
non_smtpd_milters = inet:localhost:8891

# Limits
mailbox_size_limit = 0
message_size_limit = 52428800

# Rate limiting (prevent abuse if accounts get compromised)
smtpd_client_message_rate_limit = 100
smtpd_client_recipient_rate_limit = 500
anvil_rate_time_unit = 3600s
POSTFIX

    # ── Postfix master.cf: enable submission port ──
    if ! grep -q "^submission" /etc/postfix/master.cf 2>/dev/null; then
        cat >> /etc/postfix/master.cf <<'MASTER'

submission inet n       -       y       -       -       smtpd
  -o syslog_name=postfix/submission
  -o smtpd_tls_security_level=encrypt
  -o smtpd_sasl_auth_enable=yes
  -o smtpd_reject_unlisted_recipient=no
  -o smtpd_recipient_restrictions=permit_sasl_authenticated,reject
MASTER
    fi

    # Postmap hash files
    postmap /etc/postfix/virtual-mailbox-maps 2>/dev/null || true
    postmap /etc/postfix/virtual 2>/dev/null || true

    # ── Dovecot ──
    # Auth config
    cat > /etc/dovecot/conf.d/10-auth.conf <<'DOVEAUTH'
disable_plaintext_auth = no
auth_mechanisms = plain login
!include auth-passwdfile.conf.ext
DOVEAUTH

    # Passwd-file auth
    cat > /etc/dovecot/conf.d/auth-passwdfile.conf.ext <<'DOVEPWD'
passdb {
  driver = passwd-file
  args = scheme=SHA512-CRYPT /etc/dovecot/users
}
userdb {
  driver = static
  args = uid=vmail gid=vmail home=/var/mail/vhosts/%d/%n
}
DOVEPWD

    # Mail location
    cat > /etc/dovecot/conf.d/10-mail.conf <<'DOVEMAIL'
mail_location = maildir:/var/mail/vhosts/%d/%n/Maildir
namespace inbox {
  inbox = yes
}
mail_privileged_group = vmail
DOVEMAIL

    # Master config (LMTP + auth sockets for Postfix)
    cat > /etc/dovecot/conf.d/10-master.conf <<'DOVEMASTER'
service imap-login {
  inet_listener imap {
    port = 143
  }
  inet_listener imaps {
    port = 993
    ssl = yes
  }
}
service lmtp {
  unix_listener /var/spool/postfix/private/dovecot-lmtp {
    mode = 0600
    user = postfix
    group = postfix
  }
}
service auth {
  unix_listener /var/spool/postfix/private/auth {
    mode = 0660
    user = postfix
    group = postfix
  }
  unix_listener auth-userdb {
    mode = 0600
    user = vmail
  }
}
service auth-worker {
  user = vmail
}
DOVEMASTER

    # SSL config
    cat > /etc/dovecot/conf.d/10-ssl.conf <<'DOVESSL'
ssl = yes
ssl_cert = </etc/ssl/certs/ssl-cert-snakeoil.pem
ssl_key = </etc/ssl/private/ssl-cert-snakeoil.key
ssl_min_protocol = TLSv1.2
DOVESSL

    # Create empty users file
    touch /etc/dovecot/users
    chown dovecot:dovecot /etc/dovecot/users
    chmod 640 /etc/dovecot/users

    # ── OpenDKIM ──
    mkdir -p /etc/opendkim/keys
    chown -R opendkim:opendkim /etc/opendkim

    cat > /etc/opendkim.conf <<'DKIM'
AutoRestart             yes
AutoRestartRate         10/1h
Syslog                  yes
SyslogSuccess           yes
LogWhy                  yes
Canonicalization        relaxed/simple
ExternalIgnoreList      refile:/etc/opendkim/trusted.hosts
InternalHosts           refile:/etc/opendkim/trusted.hosts
KeyTable                refile:/etc/opendkim/key.table
SigningTable            refile:/etc/opendkim/signing.table
Mode                    sv
PidFile                 /run/opendkim/opendkim.pid
SignatureAlgorithm      rsa-sha256
UserID                  opendkim:opendkim
Socket                  inet:8891@localhost
DKIM

    # Trusted hosts
    cat > /etc/opendkim/trusted.hosts <<'TRUSTED'
127.0.0.1
localhost
TRUSTED

    # Create empty key and signing tables
    touch /etc/opendkim/key.table
    touch /etc/opendkim/signing.table

    # Ensure opendkim run directory
    mkdir -p /run/opendkim
    chown opendkim:opendkim /run/opendkim

    # Enable and start services
    systemctl enable --now postfix > /dev/null 2>&1
    systemctl enable --now dovecot > /dev/null 2>&1
    systemctl enable --now opendkim > /dev/null 2>&1

    log "Mail server configured (Postfix + Dovecot + OpenDKIM)"
}

# SpamAssassin & ClamAV
# ---------------------------------------------------------------------------

setup_spam_antivirus() {
    log "Configuring spam & antivirus filtering..."

    export DEBIAN_FRONTEND=noninteractive
    apt-get install -y -qq spamassassin spamass-milter \
        clamav clamav-daemon clamav-milter \
        dovecot-sieve dovecot-managesieved 2>&1 | tail -5 || {
        warn "Some spam/antivirus packages failed to install — filtering may be incomplete"
    }

    # Enable SpamAssassin daemon
    sed -i 's/ENABLED=0/ENABLED=1/' /etc/default/spamassassin 2>/dev/null || true
    mkdir -p /etc/spamassassin/local.d

    # Configure spamass-milter socket inside Postfix chroot
    mkdir -p /var/spool/postfix/spamass
    chown postfix:postfix /var/spool/postfix/spamass

    if [[ ! -f /etc/default/spamass-milter ]] || ! grep -q "spamass.sock" /etc/default/spamass-milter 2>/dev/null; then
        cat > /etc/default/spamass-milter <<'SPAMILTER'
OPTIONS="-u spamass-milter -p /var/spool/postfix/spamass/spamass.sock"
SPAMILTER
    fi

    # ClamAV milter config
    mkdir -p /var/spool/postfix/clamav
    chown clamav:postfix /var/spool/postfix/clamav

    cat > /etc/clamav/clamav-milter.conf <<'CMILTER'
PidFile /var/run/clamav/clamav-milter.pid
MilterSocket /var/spool/postfix/clamav/clamav-milter.sock
MilterSocketMode 660
MilterSocketGroup postfix
FixStaleSocket true
User clamav
ClamdSocket unix:/run/clamav/clamd.ctl
OnInfected Reject
LogInfected Basic
LogClean Off
CMILTER

    # Add milters to Postfix (SpamAssassin + ClamAV alongside OpenDKIM)
    if ! grep -q "spamass" /etc/postfix/main.cf 2>/dev/null; then
        postconf -e "smtpd_milters = inet:localhost:8891, unix:/var/spool/postfix/spamass/spamass.sock, unix:/var/spool/postfix/clamav/clamav-milter.sock"
        postconf -e "non_smtpd_milters = inet:localhost:8891"
        postconf -e "milter_default_action = accept"
    fi

    # Dovecot sieve: move spam to Junk folder
    mkdir -p /var/lib/dovecot/sieve
    cat > /var/lib/dovecot/sieve/spam-to-junk.sieve <<'SIEVE'
require ["fileinto", "mailbox"];
if header :contains "X-Spam-Flag" "YES" {
    fileinto :create "Junk";
    stop;
}
SIEVE
    sievec /var/lib/dovecot/sieve/spam-to-junk.sieve 2>/dev/null || true
    chown -R vmail:vmail /var/lib/dovecot/sieve

    # Enable sieve plugin in Dovecot
    if [[ ! -f /etc/dovecot/conf.d/90-sieve.conf ]] || ! grep -q "sieve" /etc/dovecot/conf.d/90-sieve.conf 2>/dev/null; then
        cat > /etc/dovecot/conf.d/90-sieve.conf <<'DSIEVE'
plugin {
    sieve = /var/lib/dovecot/sieve/spam-to-junk.sieve
    sieve_global_dir = /var/lib/dovecot/sieve/
}

protocol lmtp {
    mail_plugins = $mail_plugins sieve
}
DSIEVE
    fi

    # Autoconfig directories
    mkdir -p /var/www/autoconfig /var/www/autodiscover
    chown -R www-data:www-data /var/www/autoconfig /var/www/autodiscover

    # Enable and start services
    systemctl enable --now clamav-freshclam > /dev/null 2>&1 || true
    systemctl enable --now clamav-daemon > /dev/null 2>&1 || true
    systemctl enable --now spamassassin > /dev/null 2>&1 || true
    systemctl enable --now spamass-milter > /dev/null 2>&1 || true
    systemctl enable --now clamav-milter > /dev/null 2>&1 || true
    systemctl reload dovecot > /dev/null 2>&1 || true
    postfix reload > /dev/null 2>&1 || true

    log "Spam & antivirus filtering ready (SpamAssassin + ClamAV)"
}

# ---------------------------------------------------------------------------
# Roundcube Webmail
# ---------------------------------------------------------------------------

setup_roundcube() {
    log "Setting up Roundcube Webmail..."

    export DEBIAN_FRONTEND=noninteractive
    echo "roundcube-core roundcube/dbconfig-install boolean true" | debconf-set-selections
    echo "roundcube-core roundcube/database-type select sqlite3" | debconf-set-selections
    apt-get install -y -qq roundcube roundcube-plugins > /dev/null 2>&1 || {
        warn "Roundcube package not available — skipping"
        return
    }

    # Create token directory
    mkdir -p /var/lib/pinkpanel/roundcube-tokens
    chown www-data:www-data /var/lib/pinkpanel/roundcube-tokens
    chmod 755 /var/lib/pinkpanel/roundcube-tokens

    # Configure Roundcube
    local rc_config="/etc/roundcube/config.inc.php"
    if [[ -f "$rc_config" ]]; then
        # Set IMAP host
        if ! grep -q "imap_host.*localhost" "$rc_config"; then
            sed -i "s|\\\$config\['imap_host'\].*|\\\$config['imap_host'] = ['localhost:143'];|" "$rc_config"
        fi
        # Set SMTP host
        if ! grep -q "smtp_host.*localhost" "$rc_config"; then
            sed -i "s|\\\$config\['smtp_host'\].*|\\\$config['smtp_host'] = 'tls://localhost:587';|" "$rc_config"
        fi
        # Set product name
        if ! grep -q "PinkPanel" "$rc_config"; then
            echo "\$config['product_name'] = 'PinkPanel Webmail';" >> "$rc_config"
        fi
        # Disable TLS peer verification for localhost connections
        if ! grep -q "smtp_conn_options" "$rc_config"; then
            cat >> "$rc_config" <<'RCOPTS'

$config['smtp_conn_options'] = [
    'ssl' => [
        'verify_peer' => false,
        'verify_peer_name' => false,
    ],
];
$config['imap_conn_options'] = [
    'ssl' => [
        'verify_peer' => false,
        'verify_peer_name' => false,
    ],
];
RCOPTS
        fi
    fi

    # Deploy signon.php — server-side login via cURL (handles CSRF token)
    cat > /usr/share/roundcube/signon.php <<'RCSIGNON'
<?php
// PinkPanel Roundcube SSO — server-side cURL login
// 1. Reads one-time token from disk
// 2. GETs Roundcube login page to obtain session cookie + CSRF token
// 3. POSTs login with credentials + CSRF token
// 4. Forwards authenticated session cookie to browser and redirects

$token = isset($_GET['token']) ? preg_replace('/[^a-f0-9]/', '', $_GET['token']) : '';
if (!$token) { die('Missing token'); }

$tokenFile = '/var/lib/pinkpanel/roundcube-tokens/' . $token . '.json';
if (!file_exists($tokenFile)) { die('Invalid or expired token. Please try again from the panel.'); }

$data = json_decode(file_get_contents($tokenFile), true);
@unlink($tokenFile);

if (!$data || empty($data['username']) || empty($data['password'])) { die('Invalid token data'); }

$rcBase = 'http://127.0.0.1/roundcube/';
$cookieJar = tempnam(sys_get_temp_dir(), 'rc_');

// Step 1: GET login page to obtain session cookie and CSRF token
$ch = curl_init();
curl_setopt_array($ch, [
    CURLOPT_URL            => $rcBase . '?_task=login',
    CURLOPT_RETURNTRANSFER => true,
    CURLOPT_COOKIEJAR      => $cookieJar,
    CURLOPT_COOKIEFILE     => $cookieJar,
    CURLOPT_FOLLOWLOCATION => true,
    CURLOPT_TIMEOUT        => 10,
]);
$loginPage = curl_exec($ch);
$httpCode = curl_getinfo($ch, CURLINFO_HTTP_CODE);
curl_close($ch);

if (!$loginPage || $httpCode !== 200) {
    @unlink($cookieJar);
    die('Failed to load Roundcube login page');
}

// Extract CSRF token from the login form
if (!preg_match('/name="_token"\s+value="([^"]+)"/', $loginPage, $m)) {
    @unlink($cookieJar);
    die('Could not extract CSRF token from Roundcube');
}
$csrfToken = $m[1];

// Step 2: POST login with credentials + CSRF token
// Capture Set-Cookie headers from response (sessauth is not in cookie jar)
$responseCookies = [];
$ch = curl_init();
curl_setopt_array($ch, [
    CURLOPT_URL            => $rcBase . '?_task=login&_action=login',
    CURLOPT_RETURNTRANSFER => true,
    CURLOPT_POST           => true,
    CURLOPT_POSTFIELDS     => http_build_query([
        '_task'     => 'login',
        '_action'   => 'login',
        '_timezone' => 'UTC',
        '_token'    => $csrfToken,
        '_user'     => $data['username'],
        '_pass'     => $data['password'],
    ]),
    CURLOPT_COOKIEJAR      => $cookieJar,
    CURLOPT_COOKIEFILE     => $cookieJar,
    CURLOPT_FOLLOWLOCATION => false,
    CURLOPT_TIMEOUT        => 10,
    CURLOPT_HEADERFUNCTION => function($ch, $header) use (&$responseCookies) {
        if (stripos($header, 'Set-Cookie:') === 0) {
            $cookiePart = trim(substr($header, 11));
            $nameValue = explode(';', $cookiePart)[0];
            $eq = strpos($nameValue, '=');
            if ($eq !== false) {
                $name = trim(substr($nameValue, 0, $eq));
                $value = trim(substr($nameValue, $eq + 1));
                $responseCookies[$name] = $value;
            }
        }
        return strlen($header);
    },
]);
$response = curl_exec($ch);
$loginHttpCode = curl_getinfo($ch, CURLINFO_HTTP_CODE);
curl_close($ch);
@unlink($cookieJar);

// 302 redirect = login succeeded
if ($loginHttpCode !== 302) {
    die('Roundcube login failed — check email credentials. You may need to change the password in the panel first.');
}

// Forward all session cookies to the browser
foreach ($responseCookies as $name => $value) {
    setcookie($name, $value, 0, '/roundcube/', '', true, true);
}

// Step 3: Redirect to authenticated Roundcube session
header('Location: /roundcube/');
exit;
RCSIGNON
    chown www-data:www-data /usr/share/roundcube/signon.php
    # Symlink into public docroot (nginx may serve from /var/lib/roundcube/public_html/)
    if [[ -d /var/lib/roundcube/public_html ]] && [[ ! -e /var/lib/roundcube/public_html/signon.php ]]; then
        ln -sf /usr/share/roundcube/signon.php /var/lib/roundcube/public_html/signon.php
    fi

    # Create NGINX config for Roundcube
    local php_sock
    php_sock=$(ls /run/php/php*-fpm.sock 2>/dev/null | head -1)
    [[ -z "$php_sock" ]] && php_sock="/run/php/php-fpm.sock"

    # Determine Roundcube web root (varies by distro)
    local rc_root="/usr/share/roundcube"
    [[ -d /var/lib/roundcube/public_html ]] && rc_root="/var/lib/roundcube/public_html"

    cat > /etc/nginx/snippets/roundcube.conf <<RCNGINX
location /roundcube/ {
    alias ${rc_root}/;
    index index.php;

    location ~ ^/roundcube/(.*\\.php)\$ {
        alias ${rc_root}/\$1;
        fastcgi_pass unix:${php_sock};
        fastcgi_index index.php;
        fastcgi_param SCRIPT_FILENAME ${rc_root}/\$1;
        include fastcgi_params;
    }
}

location ~ ^/roundcube/(config|temp|logs) {
    deny all;
}
RCNGINX

    # Include in default NGINX server block
    for f in /etc/nginx/sites-available/default /etc/nginx/sites-enabled/default; do
        if [[ -f "$f" ]] && ! grep -q "roundcube" "$f"; then
            sed -i '/server_name _;/a\\n\tinclude snippets/roundcube.conf;' "$f"
        fi
    done

    nginx -t > /dev/null 2>&1 && systemctl reload nginx || warn "NGINX config test failed after Roundcube setup"

    log "Roundcube Webmail configured at /roundcube/ with auto-login"
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
    ufw allow 25/tcp comment "SMTP" > /dev/null 2>&1
    ufw allow 587/tcp comment "SMTP Submission" > /dev/null 2>&1
    ufw allow 993/tcp comment "IMAPS" > /dev/null 2>&1
    ufw allow 143/tcp comment "IMAP" > /dev/null 2>&1

    ufw --force enable > /dev/null 2>&1
    log "Firewall configured (ports: 22, 25, 53, 80, 143, 443, 587, 993, $PINKPANEL_PORT, 21, 40000-50000)"
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
    setup_phpmyadmin
    setup_mail
    setup_roundcube
    setup_spam_antivirus
    setup_firewall
    setup_modsecurity
    setup_fail2ban
    start_services

    # Save installed version
    local ver
    ver=$("$PINKPANEL_HOME/bin/pinkpanel-cli" version 2>/dev/null | awk '{print $2}' || echo "unknown")
    echo "$ver" > /etc/pinkpanel/version
    log "Version $ver"

    print_complete
}

main "$@"
