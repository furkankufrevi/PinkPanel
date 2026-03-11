package template

import (
	"bytes"
	"text/template"
)

type PHPPoolData struct {
	Domain       string
	User         string
	Group        string
	ListenSocket string
	PHPVersion   string
	Settings     map[string]string
}

const phpPoolTmpl = `[{{ .Domain }}]
user = {{ .User }}
group = {{ .Group }}

listen = {{ .ListenSocket }}
listen.owner = {{ .User }}
listen.group = {{ .Group }}
listen.mode = 0660

pm = dynamic
pm.max_children = 10
pm.start_servers = 2
pm.min_spare_servers = 1
pm.max_spare_servers = 3
pm.max_requests = 500
{{ range $key, $val := .Settings }}
php_admin_value[{{ $key }}] = {{ $val }}
{{- end }}

php_admin_value[error_log] = /var/log/php-fpm/{{ .Domain }}-error.log
php_admin_value[access.log] = /var/log/php-fpm/{{ .Domain }}-access.log
`

func RenderPHPPool(data PHPPoolData) (string, error) {
	t, err := template.New("phppool").Parse(phpPoolTmpl)
	if err != nil {
		return "", err
	}
	var buf bytes.Buffer
	if err := t.Execute(&buf, data); err != nil {
		return "", err
	}
	return buf.String(), nil
}
