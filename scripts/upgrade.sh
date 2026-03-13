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

migrate_to_0_6_2() {
    log "Running migrations for 0.6.2..."

    # --- Git integration data directory ---
    mkdir -p "$PINKPANEL_DATA/git"
    chown pinkpanel:pinkpanel "$PINKPANEL_DATA/git" 2>/dev/null || true

    log "Migrations for 0.6.2 complete"
}

# ── Run all applicable migrations ──────────
run_migrations() {
    local from_version="$1"

    # Add migration entries here in order. Format:
    #   version_lt "$from_version" "X.Y.Z" && migrate_to_X_Y_Z
    version_lt "$from_version" "0.3.0" && migrate_to_0_3_0 || true

    version_lt "$from_version" "0.6.2" && migrate_to_0_6_2 || true

    return 0
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

# Install new binaries (even if same version — scripts/config may have changed)
SKIP_BINARY=false
if [[ "$CURRENT" == "$NEW_VERSION" ]]; then
    log "Already running $CURRENT — updating config only"
    SKIP_BINARY=true
fi

if [[ "$SKIP_BINARY" == false ]]; then
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
else
    # Stop services briefly for config updates
    log "Stopping PinkPanel services..."
    systemctl stop pinkpanel 2>/dev/null || true
    systemctl stop pinkpanel-agent 2>/dev/null || true
fi

# ── Ensure MySQL root uses unix_socket auth ──
fix_mysql_auth() {
    # Check if root can connect without password (unix_socket)
    if mysql -u root -e "SELECT 1" &>/dev/null; then
        # Already working — ensure plugin is unix_socket
        local plugin
        plugin=$(mysql -u root -N -e "SELECT plugin FROM mysql.user WHERE user='root' AND host='localhost';" 2>/dev/null || echo "")
        if [[ "$plugin" != "unix_socket" ]]; then
            log "Switching MySQL root to unix_socket auth..."
            mysql -u root -e "ALTER USER 'root'@'localhost' IDENTIFIED VIA unix_socket; FLUSH PRIVILEGES;" 2>/dev/null || true
        fi
        # Clean up legacy password file
        rm -f /etc/pinkpanel/mysql.cnf
        return
    fi

    # Root can't connect — try with legacy password file
    if [[ -f /etc/pinkpanel/mysql.cnf ]]; then
        log "Migrating MySQL root auth to unix_socket via password file..."
        if mysql --defaults-file=/etc/pinkpanel/mysql.cnf -e "ALTER USER 'root'@'localhost' IDENTIFIED VIA unix_socket; FLUSH PRIVILEGES;" 2>/dev/null; then
            log "MySQL auth migrated"
            rm -f /etc/pinkpanel/mysql.cnf
            return
        fi
    fi

    # Last resort: reset via skip-grant-tables
    warn "MySQL root access denied — attempting recovery via skip-grant-tables..."
    systemctl stop mariadb 2>/dev/null || true
    mysqld_safe --skip-grant-tables --skip-networking &
    local bg_pid=$!
    sleep 3
    if mysql -u root -e "FLUSH PRIVILEGES; ALTER USER 'root'@'localhost' IDENTIFIED VIA unix_socket; FLUSH PRIVILEGES;" 2>/dev/null; then
        log "MySQL root auth recovered"
    else
        warn "Could not recover MySQL root auth — manual intervention needed"
    fi
    kill "$bg_pid" 2>/dev/null || true
    wait "$bg_pid" 2>/dev/null || true
    sleep 2
    systemctl start mariadb 2>/dev/null || true
    rm -f /etc/pinkpanel/mysql.cnf

    if mysql -u root -e "SELECT 1" &>/dev/null; then
        log "MySQL root access confirmed"
    else
        warn "MySQL root still not accessible — check manually"
    fi
}

# ── Ensure phpMyAdmin is installed ──────────
setup_phpmyadmin() {
    export DEBIAN_FRONTEND=noninteractive

    if [[ ! -d /usr/share/phpmyadmin ]]; then
        log "Installing phpMyAdmin..."
        echo "phpmyadmin phpmyadmin/dbconfig-install boolean false" | debconf-set-selections
        echo "phpmyadmin phpmyadmin/reconfigure-webserver multiselect none" | debconf-set-selections
        apt-get update -qq
        apt-get install -y -qq phpmyadmin > /dev/null 2>&1 || {
            warn "phpMyAdmin package not available — skipping"
            return
        }
    fi

    # Create token directory (world-readable so www-data PHP can access)
    mkdir -p /var/lib/pinkpanel/pma-tokens
    chown www-data:www-data /var/lib/pinkpanel/pma-tokens
    chmod 755 /var/lib/pinkpanel/pma-tokens

    # Patch phpMyAdmin config.inc.php directly for signon auth (conf.d may not override)
    local pma_config="/etc/phpmyadmin/config.inc.php"
    if [[ -f "$pma_config" ]]; then
        # Remove any existing auth_type/SignonSession/SignonURL lines
        sed -i "/auth_type.*=.*'cookie'/d" "$pma_config"
        sed -i "/auth_type.*=.*'signon'/d" "$pma_config"
        sed -i "/SignonSession/d" "$pma_config"
        sed -i "/SignonURL/d" "$pma_config"
        # Append signon config before the closing ?>  or at end of file
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

    # Deploy signon.php script (always update)
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

    # Create/update NGINX snippet (always update to fix alias+php issues)
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

    # Include in default NGINX server block (both sites-available and sites-enabled,
    # since sites-enabled may be a separate file instead of a symlink)
    for f in /etc/nginx/sites-available/default /etc/nginx/sites-enabled/default; do
        if [[ -f "$f" ]] && ! grep -q "phpmyadmin" "$f"; then
            sed -i '/server_name _;/a\\n\tinclude snippets/phpmyadmin.conf;' "$f"
        fi
    done

    # Reload NGINX
    nginx -t > /dev/null 2>&1 && systemctl reload nginx 2>/dev/null || true

    log "phpMyAdmin configured with auto-login"
}

# ── Ensure required packages ────────────────
install_missing_packages() {
    local missing=()
    command -v git &>/dev/null || missing+=("git")
    command -v zip &>/dev/null || missing+=("zip")
    command -v unzip &>/dev/null || missing+=("unzip")
    command -v fail2ban-client &>/dev/null || missing+=("fail2ban")
    dpkg -l libmodsecurity3 &>/dev/null 2>&1 || missing+=("libmodsecurity3")
    command -v postfix &>/dev/null || missing+=("postfix")
    command -v dovecot &>/dev/null || missing+=("dovecot-core" "dovecot-imapd" "dovecot-lmtpd")
    command -v opendkim-genkey &>/dev/null || missing+=("opendkim" "opendkim-tools")
    dpkg -l roundcube-core &>/dev/null 2>&1 || missing+=("roundcube" "roundcube-plugins")
    command -v spamd &>/dev/null || missing+=("spamassassin" "spamass-milter")
    command -v clamd &>/dev/null || missing+=("clamav" "clamav-daemon" "clamav-milter")
    dpkg -l dovecot-sieve &>/dev/null 2>&1 || missing+=("dovecot-sieve" "dovecot-managesieved")
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

# ── Ensure ModSecurity is configured ─────────
setup_modsecurity() {
    mkdir -p /etc/nginx/modsecurity

    # Install NGINX ModSecurity connector if not present
    if ! dpkg -l libnginx-mod-http-modsecurity &>/dev/null 2>&1; then
        apt-get install -y -qq libnginx-mod-http-modsecurity > /dev/null 2>&1 || true
    fi

    # Create modsecurity config if missing
    if [[ ! -f /etc/nginx/modsecurity/modsecurity.conf ]]; then
        log "Creating ModSecurity config..."
        if [[ -f /etc/modsecurity/modsecurity.conf-recommended ]]; then
            cp /etc/modsecurity/modsecurity.conf-recommended /etc/nginx/modsecurity/modsecurity.conf
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

    # Install OWASP CRS if missing
    if [[ ! -d /etc/nginx/modsecurity/crs ]]; then
        apt-get install -y -qq modsecurity-crs > /dev/null 2>&1 || true
        if [[ -d /usr/share/modsecurity-crs ]]; then
            ln -sf /usr/share/modsecurity-crs /etc/nginx/modsecurity/crs
            if ! grep -q "crs" /etc/nginx/modsecurity/modsecurity.conf 2>/dev/null; then
                cat >> /etc/nginx/modsecurity/modsecurity.conf <<'CRS'

# OWASP Core Rule Set
Include /etc/nginx/modsecurity/crs/rules/*.conf
CRS
            fi
        fi
    fi
}

# ── Ensure Fail2ban is configured ───────────
setup_fail2ban() {
    if ! command -v fail2ban-client &>/dev/null; then
        return
    fi

    # Create PinkPanel filter if missing
    if [[ ! -f /etc/fail2ban/filter.d/pinkpanel.conf ]]; then
        log "Creating Fail2ban PinkPanel filter..."
        cat > /etc/fail2ban/filter.d/pinkpanel.conf <<'FILTER'
[Definition]
failregex = ^.*"POST /api/auth/login".*<HOST>.*401.*$
ignoreregex =
FILTER
    fi

    # Create PinkPanel jail if missing
    if [[ ! -f /etc/fail2ban/jail.d/pinkpanel.conf ]]; then
        log "Creating Fail2ban PinkPanel jail..."
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
    fi

    systemctl enable fail2ban > /dev/null 2>&1
    systemctl restart fail2ban > /dev/null 2>&1
}

# ── Ensure mail server is configured ───────
setup_mail() {
    # Idempotent: only create configs if missing

    # Create vmail user
    if ! id vmail &>/dev/null; then
        groupadd -g 5000 vmail 2>/dev/null || true
        useradd -g vmail -u 5000 vmail -d /var/mail/vhosts -s /usr/sbin/nologin 2>/dev/null || true
    fi
    mkdir -p /var/mail/vhosts
    chown -R vmail:vmail /var/mail/vhosts

    # Create virtual map files if missing
    touch /etc/postfix/virtual-mailbox-domains 2>/dev/null || true
    touch /etc/postfix/virtual-mailbox-maps 2>/dev/null || true
    touch /etc/postfix/virtual 2>/dev/null || true

    # Configure Postfix for virtual mailboxes if not already done
    if ! grep -q "virtual_mailbox_domains" /etc/postfix/main.cf 2>/dev/null; then
        log "Configuring Postfix for virtual mailboxes..."
        local hostname
        hostname=$(hostname -f 2>/dev/null || hostname)

        # Append virtual mailbox settings
        cat >> /etc/postfix/main.cf <<POSTFIX

# PinkPanel virtual mailbox settings
virtual_mailbox_domains = /etc/postfix/virtual-mailbox-domains
virtual_mailbox_maps = hash:/etc/postfix/virtual-mailbox-maps
virtual_mailbox_base = /var/mail/vhosts
virtual_transport = lmtp:unix:private/dovecot-lmtp
virtual_alias_maps = hash:/etc/postfix/virtual
virtual_minimum_uid = 5000
virtual_uid_maps = static:5000
virtual_gid_maps = static:5000

smtpd_sasl_auth_enable = yes
smtpd_sasl_type = dovecot
smtpd_sasl_path = private/auth
smtpd_sasl_security_options = noanonymous

milter_protocol = 6
milter_default_action = accept
smtpd_milters = inet:localhost:8891
non_smtpd_milters = inet:localhost:8891
POSTFIX
        postmap /etc/postfix/virtual-mailbox-maps 2>/dev/null || true
        postmap /etc/postfix/virtual 2>/dev/null || true
    fi

    # Enable submission port if missing
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

    # Always ensure Dovecot is configured for virtual mailbox auth
    # (overwrites system auth config that breaks virtual email accounts)
    log "Configuring Dovecot for virtual mailbox auth..."
    cat > /etc/dovecot/conf.d/10-auth.conf <<'DOVEAUTH'
disable_plaintext_auth = no
auth_mechanisms = plain login
!include auth-passwdfile.conf.ext
DOVEAUTH

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

    cat > /etc/dovecot/conf.d/10-mail.conf <<'DOVEMAIL'
mail_location = maildir:/var/mail/vhosts/%d/%n/Maildir
namespace inbox {
  inbox = yes
}
mail_privileged_group = vmail
DOVEMAIL

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

    if ! grep -q "ssl_cert.*ssl-cert-snakeoil\|ssl_cert.*pinkpanel" /etc/dovecot/conf.d/10-ssl.conf 2>/dev/null; then
        cat > /etc/dovecot/conf.d/10-ssl.conf <<'DOVESSL'
ssl = yes
ssl_cert = </etc/ssl/certs/ssl-cert-snakeoil.pem
ssl_key = </etc/ssl/private/ssl-cert-snakeoil.key
ssl_min_protocol = TLSv1.2
DOVESSL
    fi

    # Dovecot users file
    touch /etc/dovecot/users
    chown dovecot:dovecot /etc/dovecot/users 2>/dev/null || true
    chmod 640 /etc/dovecot/users

    # OpenDKIM
    mkdir -p /etc/opendkim/keys
    chown -R opendkim:opendkim /etc/opendkim 2>/dev/null || true
    if [[ ! -f /etc/opendkim.conf ]] || ! grep -q "KeyTable" /etc/opendkim.conf 2>/dev/null; then
        log "Configuring OpenDKIM..."
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
        cat > /etc/opendkim/trusted.hosts <<'TRUSTED'
127.0.0.1
localhost
TRUSTED
        touch /etc/opendkim/key.table /etc/opendkim/signing.table
    fi
    mkdir -p /run/opendkim
    chown opendkim:opendkim /run/opendkim 2>/dev/null || true

    # Open mail ports if ufw is active
    if command -v ufw &>/dev/null && ufw status | grep -q "Status: active"; then
        ufw allow 25/tcp > /dev/null 2>&1 || true
        ufw allow 587/tcp > /dev/null 2>&1 || true
        ufw allow 993/tcp > /dev/null 2>&1 || true
        ufw allow 143/tcp > /dev/null 2>&1 || true
    fi

    # Enable and restart services (restart needed to pick up config changes)
    systemctl enable postfix > /dev/null 2>&1 || true
    systemctl enable dovecot > /dev/null 2>&1 || true
    systemctl enable opendkim > /dev/null 2>&1 || true
    systemctl restart dovecot > /dev/null 2>&1 || true
    systemctl restart postfix > /dev/null 2>&1 || true
    systemctl restart opendkim > /dev/null 2>&1 || true

    log "Mail server ready"
}

# ── SpamAssassin & ClamAV setup ────────────
setup_spam_antivirus() {
    log "Checking spam & antivirus..."

    # Configure spamass-milter socket inside Postfix chroot
    mkdir -p /var/spool/postfix/spamass
    chown postfix:postfix /var/spool/postfix/spamass 2>/dev/null || true

    if [[ ! -f /etc/default/spamass-milter ]] || ! grep -q "spamass.sock" /etc/default/spamass-milter 2>/dev/null; then
        cat > /etc/default/spamass-milter <<'SPAMILTER'
OPTIONS="-u spamass-milter -p /var/spool/postfix/spamass/spamass.sock"
SPAMILTER
    fi

    # ClamAV milter config
    mkdir -p /var/spool/postfix/clamav
    chown clamav:postfix /var/spool/postfix/clamav 2>/dev/null || true

    if [[ ! -f /etc/clamav/clamav-milter.conf ]] || ! grep -q "OnInfected" /etc/clamav/clamav-milter.conf 2>/dev/null; then
        cat > /etc/clamav/clamav-milter.conf <<'CMILTER'
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
    fi

    # Add milters to Postfix if not present
    if ! grep -q "spamass" /etc/postfix/main.cf 2>/dev/null; then
        postconf -e "smtpd_milters = inet:localhost:8891, unix:/var/spool/postfix/spamass/spamass.sock, unix:/var/spool/postfix/clamav/clamav-milter.sock" 2>/dev/null || true
        postconf -e "non_smtpd_milters = inet:localhost:8891" 2>/dev/null || true
        postconf -e "milter_default_action = accept" 2>/dev/null || true
    fi

    # Dovecot sieve
    mkdir -p /var/lib/dovecot/sieve
    if [[ ! -f /var/lib/dovecot/sieve/spam-to-junk.sieve ]]; then
        cat > /var/lib/dovecot/sieve/spam-to-junk.sieve <<'SIEVE'
require ["fileinto", "mailbox"];
if header :contains "X-Spam-Flag" "YES" {
    fileinto :create "Junk";
    stop;
}
SIEVE
        sievec /var/lib/dovecot/sieve/spam-to-junk.sieve 2>/dev/null || true
        chown -R vmail:vmail /var/lib/dovecot/sieve
    fi

    if [[ ! -f /etc/dovecot/conf.d/90-sieve.conf ]]; then
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
    chown -R www-data:www-data /var/www/autoconfig /var/www/autodiscover 2>/dev/null || true

    # Enable services
    systemctl enable --now clamav-freshclam > /dev/null 2>&1 || true
    systemctl enable --now clamav-daemon > /dev/null 2>&1 || true
    systemctl enable --now spamassassin > /dev/null 2>&1 || true
    systemctl enable --now spamass-milter > /dev/null 2>&1 || true
    systemctl enable --now clamav-milter > /dev/null 2>&1 || true
    systemctl reload dovecot > /dev/null 2>&1 || true

    log "Spam & antivirus ready"
}

# ── Roundcube Webmail setup ────────────────
setup_roundcube() {
    log "Checking Roundcube Webmail..."

    # Create token directory
    mkdir -p /var/lib/pinkpanel/roundcube-tokens
    chown www-data:www-data /var/lib/pinkpanel/roundcube-tokens
    chmod 755 /var/lib/pinkpanel/roundcube-tokens

    # Configure Roundcube
    local rc_config="/etc/roundcube/config.inc.php"
    if [[ -f "$rc_config" ]]; then
        if ! grep -q "imap_host.*localhost" "$rc_config"; then
            sed -i "s|\\\$config\['imap_host'\].*|\\\$config['imap_host'] = ['localhost:143'];|" "$rc_config"
        fi
        if ! grep -q "smtp_host.*localhost" "$rc_config"; then
            sed -i "s|\\\$config\['smtp_host'\].*|\\\$config['smtp_host'] = 'localhost:587';|" "$rc_config"
        fi
        if ! grep -q "PinkPanel" "$rc_config"; then
            echo "\$config['product_name'] = 'PinkPanel Webmail';" >> "$rc_config"
        fi
    fi

    # Deploy signon.php — reads one-time token, auto-submits login form
    if [[ -d /usr/share/roundcube ]]; then
        cat > /usr/share/roundcube/signon.php <<'RCSIGNON'
<?php
$token = isset($_GET['token']) ? preg_replace('/[^a-f0-9]/', '', $_GET['token']) : '';
if (!$token) { die('Missing token'); }

$tokenFile = '/var/lib/pinkpanel/roundcube-tokens/' . $token . '.json';
if (!file_exists($tokenFile)) { die('Invalid or expired token. Please try again from the panel.'); }

$data = json_decode(file_get_contents($tokenFile), true);
@unlink($tokenFile);

if (!$data || empty($data['username']) || empty($data['password'])) { die('Invalid token data'); }

$user = htmlspecialchars($data['username'], ENT_QUOTES, 'UTF-8');
$pass = htmlspecialchars($data['password'], ENT_QUOTES, 'UTF-8');
?><!DOCTYPE html>
<html><head><meta charset="utf-8"><title>Signing in...</title></head>
<body><p>Signing in to webmail...</p>
<form id="rclogin" method="post" action="/roundcube/">
  <input type="hidden" name="_task" value="login">
  <input type="hidden" name="_action" value="login">
  <input type="hidden" name="_user" value="<?php echo $user; ?>">
  <input type="hidden" name="_pass" value="<?php echo $pass; ?>">
  <input type="hidden" name="_timezone" value="UTC">
</form>
<script>document.getElementById('rclogin').submit();</script>
</body></html>
RCSIGNON
        chown www-data:www-data /usr/share/roundcube/signon.php
    fi

    # Always recreate NGINX config (fix paths on upgrade)
    local php_sock
    php_sock=$(ls /run/php/php*-fpm.sock 2>/dev/null | head -1)
    [[ -z "$php_sock" ]] && php_sock="/run/php/php-fpm.sock"

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

    for f in /etc/nginx/sites-available/default /etc/nginx/sites-enabled/default; do
        if [[ -f "$f" ]] && ! grep -q "roundcube" "$f"; then
            sed -i '/server_name _;/a\\n\tinclude snippets/roundcube.conf;' "$f"
        fi
    done

    nginx -t > /dev/null 2>&1 && systemctl reload nginx > /dev/null 2>&1 || true

    log "Roundcube Webmail ready"
}

# ── Always-run fixups (run even when version matches) ──
install_missing_packages
fix_bind
fix_mysql_auth
setup_phpmyadmin
setup_modsecurity
setup_fail2ban
setup_mail
setup_spam_antivirus
setup_roundcube

# ── Run version-specific migrations ────────
if [[ "$SKIP_BINARY" == false ]]; then
    run_migrations "$CURRENT"
fi

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
