package template

import (
	"bytes"
	"text/template"
)

// NginxVhostData holds all configuration values needed to render an NGINX virtual host.
type NginxVhostData struct {
	Domain       string
	DocumentRoot string
	PHPVersion   string
	SSLEnabled   bool
	SSLCertPath  string
	SSLKeyPath   string
	SSLChainPath string
	ForceHTTPS   bool
	HTTP2        bool
	HSTS               bool
	HSTSMaxAge         int
	ModSecurityEnabled bool
	Suspended          bool
}

// NginxSuspendedData holds the domain for a suspended vhost.
type NginxSuspendedData struct {
	Domain string
}

const nginxVhostTemplate = `server {
    listen 80;
    listen [::]:80;
    server_name {{ .Domain }} www.{{ .Domain }};
    root {{ .DocumentRoot }};

    # Allow ACME challenge for Let's Encrypt
    location ^~ /.well-known/acme-challenge/ {
        allow all;
        default_type "text/plain";
    }
{{- if and .SSLEnabled .ForceHTTPS }}

    # Redirect HTTP to HTTPS
    location / {
        return 301 https://$host$request_uri;
    }
}

server {
    listen 443 ssl{{ if .HTTP2 }} http2{{ end }};
    listen [::]:443 ssl{{ if .HTTP2 }} http2{{ end }};
    server_name {{ .Domain }} www.{{ .Domain }};
    root {{ .DocumentRoot }};

    ssl_certificate {{ .SSLCertPath }};
    ssl_certificate_key {{ .SSLKeyPath }};
{{- if .SSLChainPath }}
    ssl_trusted_certificate {{ .SSLChainPath }};
{{- end }}

    ssl_protocols TLSv1.2 TLSv1.3;
    ssl_ciphers HIGH:!aNULL:!MD5;
    ssl_prefer_server_ciphers on;
    ssl_session_cache shared:SSL:10m;
    ssl_session_timeout 10m;
{{- if .HSTS }}

    # HSTS
    add_header Strict-Transport-Security "max-age={{ .HSTSMaxAge }}; includeSubDomains; preload" always;
{{- end }}

    # Security headers
    add_header X-Frame-Options SAMEORIGIN always;
    add_header X-Content-Type-Options nosniff always;
    add_header X-XSS-Protection "1; mode=block" always;
{{- if .ModSecurityEnabled }}

    # ModSecurity WAF
    modsecurity on;
    modsecurity_rules_file /etc/nginx/modsecurity/modsecurity.conf;
{{- end }}

    # Gzip
    gzip on;
    gzip_types text/plain text/css application/json application/javascript text/xml application/xml application/xml+rss text/javascript;
    gzip_vary on;

    index index.php index.html index.htm;

    # Allow ACME challenge for Let's Encrypt
    location ^~ /.well-known/acme-challenge/ {
        allow all;
        default_type "text/plain";
    }

    location / {
        try_files $uri $uri/ /index.php?$query_string;
    }

    location ~ \.php$ {
        fastcgi_pass unix:/run/php/php{{ .PHPVersion }}-fpm-{{ .Domain }}.sock;
        fastcgi_index index.php;
        fastcgi_param SCRIPT_FILENAME $document_root$fastcgi_script_name;
        include fastcgi_params;
    }

    # Deny access to .ht files
    location ~ /\.ht {
        deny all;
    }
}
{{- else if .SSLEnabled }}

    # SSL without forced redirect — serve content on both HTTP and HTTPS
{{- if .HSTS }}
    add_header Strict-Transport-Security "max-age={{ .HSTSMaxAge }}; includeSubDomains; preload" always;
{{- end }}

    # Security headers
    add_header X-Frame-Options SAMEORIGIN always;
    add_header X-Content-Type-Options nosniff always;
    add_header X-XSS-Protection "1; mode=block" always;
{{- if .ModSecurityEnabled }}

    # ModSecurity WAF
    modsecurity on;
    modsecurity_rules_file /etc/nginx/modsecurity/modsecurity.conf;
{{- end }}

    # Gzip
    gzip on;
    gzip_types text/plain text/css application/json application/javascript text/xml application/xml application/xml+rss text/javascript;
    gzip_vary on;

    index index.php index.html index.htm;

    location / {
        try_files $uri $uri/ /index.php?$query_string;
    }

    location ~ \.php$ {
        fastcgi_pass unix:/run/php/php{{ .PHPVersion }}-fpm-{{ .Domain }}.sock;
        fastcgi_index index.php;
        fastcgi_param SCRIPT_FILENAME $document_root$fastcgi_script_name;
        include fastcgi_params;
    }

    # Deny access to .ht files
    location ~ /\.ht {
        deny all;
    }
}

server {
    listen 443 ssl{{ if .HTTP2 }} http2{{ end }};
    listen [::]:443 ssl{{ if .HTTP2 }} http2{{ end }};
    server_name {{ .Domain }} www.{{ .Domain }};
    root {{ .DocumentRoot }};

    ssl_certificate {{ .SSLCertPath }};
    ssl_certificate_key {{ .SSLKeyPath }};
{{- if .SSLChainPath }}
    ssl_trusted_certificate {{ .SSLChainPath }};
{{- end }}

    ssl_protocols TLSv1.2 TLSv1.3;
    ssl_ciphers HIGH:!aNULL:!MD5;
    ssl_prefer_server_ciphers on;
    ssl_session_cache shared:SSL:10m;
    ssl_session_timeout 10m;
{{- if .HSTS }}

    # HSTS
    add_header Strict-Transport-Security "max-age={{ .HSTSMaxAge }}; includeSubDomains; preload" always;
{{- end }}

    # Security headers
    add_header X-Frame-Options SAMEORIGIN always;
    add_header X-Content-Type-Options nosniff always;
    add_header X-XSS-Protection "1; mode=block" always;
{{- if .ModSecurityEnabled }}

    # ModSecurity WAF
    modsecurity on;
    modsecurity_rules_file /etc/nginx/modsecurity/modsecurity.conf;
{{- end }}

    # Gzip
    gzip on;
    gzip_types text/plain text/css application/json application/javascript text/xml application/xml application/xml+rss text/javascript;
    gzip_vary on;

    index index.php index.html index.htm;

    # Allow ACME challenge for Let's Encrypt
    location ^~ /.well-known/acme-challenge/ {
        allow all;
        default_type "text/plain";
    }

    location / {
        try_files $uri $uri/ /index.php?$query_string;
    }

    location ~ \.php$ {
        fastcgi_pass unix:/run/php/php{{ .PHPVersion }}-fpm-{{ .Domain }}.sock;
        fastcgi_index index.php;
        fastcgi_param SCRIPT_FILENAME $document_root$fastcgi_script_name;
        include fastcgi_params;
    }

    # Deny access to .ht files
    location ~ /\.ht {
        deny all;
    }
}
{{- else }}

    # HTTP only
{{- if .HSTS }}
    add_header Strict-Transport-Security "max-age={{ .HSTSMaxAge }}; includeSubDomains; preload" always;
{{- end }}

    # Security headers
    add_header X-Frame-Options SAMEORIGIN always;
    add_header X-Content-Type-Options nosniff always;
    add_header X-XSS-Protection "1; mode=block" always;
{{- if .ModSecurityEnabled }}

    # ModSecurity WAF
    modsecurity on;
    modsecurity_rules_file /etc/nginx/modsecurity/modsecurity.conf;
{{- end }}

    # Gzip
    gzip on;
    gzip_types text/plain text/css application/json application/javascript text/xml application/xml application/xml+rss text/javascript;
    gzip_vary on;

    index index.php index.html index.htm;

    location / {
        try_files $uri $uri/ /index.php?$query_string;
    }

    location ~ \.php$ {
        fastcgi_pass unix:/run/php/php{{ .PHPVersion }}-fpm-{{ .Domain }}.sock;
        fastcgi_index index.php;
        fastcgi_param SCRIPT_FILENAME $document_root$fastcgi_script_name;
        include fastcgi_params;
    }

    # Deny access to .ht files
    location ~ /\.ht {
        deny all;
    }
}
{{- end }}
`

const nginxSuspendedTemplate = `server {
    listen 80;
    listen [::]:80;
    server_name {{ .Domain }} www.{{ .Domain }};

    location / {
        default_type "text/html";
        return 503 '<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Site Suspended</title>
    <style>
        body {
            font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif;
            display: flex;
            justify-content: center;
            align-items: center;
            min-height: 100vh;
            margin: 0;
            background-color: #f5f5f5;
            color: #333;
        }
        .container {
            text-align: center;
            padding: 2rem;
        }
        h1 { font-size: 2rem; margin-bottom: 0.5rem; }
        p { font-size: 1.1rem; color: #666; }
    </style>
</head>
<body>
    <div class="container">
        <h1>This site has been suspended</h1>
        <p>Please contact the site administrator for more information.</p>
    </div>
</body>
</html>';
    }
}
`

// RenderNginxVhost renders an NGINX virtual host configuration from the given data.
func RenderNginxVhost(data NginxVhostData) (string, error) {
	tmpl, err := template.New("nginx-vhost").Parse(nginxVhostTemplate)
	if err != nil {
		return "", err
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", err
	}

	return buf.String(), nil
}

// RenderNginxSuspended renders a suspended site NGINX configuration.
func RenderNginxSuspended(data NginxSuspendedData) (string, error) {
	tmpl, err := template.New("nginx-suspended").Parse(nginxSuspendedTemplate)
	if err != nil {
		return "", err
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", err
	}

	return buf.String(), nil
}
