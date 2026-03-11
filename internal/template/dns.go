package template

import (
	"bytes"
	"fmt"
	"text/template"
)

// ZoneRecord represents a single DNS resource record in a BIND9 zone file.
type ZoneRecord struct {
	Name     string // record name (e.g., "@", "www", "mail")
	TTL      int    // time-to-live in seconds
	Class    string // always "IN"
	Type     string // A, AAAA, CNAME, MX, TXT, NS, SOA, SRV, CAA
	Priority int    // used for MX and SRV records
	Value    string // record value
}

// ZoneFileData holds all data needed to render a BIND9 zone file.
type ZoneFileData struct {
	Domain  string
	Records []ZoneRecord
}

const zoneFileTemplate = `$TTL 3600
$ORIGIN {{ .Domain }}.

{{ range .Records -}}
{{ renderRecord . }}
{{ end -}}
`

// renderRecord formats a single ZoneRecord as a BIND9 zone file line.
func renderRecord(r ZoneRecord) string {
	switch r.Type {
	case "SOA":
		return fmt.Sprintf("%s\t%d\t%s\t%s\t%s",
			r.Name, r.TTL, r.Class, r.Type, r.Value)
	case "MX":
		return fmt.Sprintf("%s\t%d\t%s\t%s\t%d\t%s",
			r.Name, r.TTL, r.Class, r.Type, r.Priority, r.Value)
	case "SRV":
		return fmt.Sprintf("%s\t%d\t%s\t%s\t%d\t%s",
			r.Name, r.TTL, r.Class, r.Type, r.Priority, r.Value)
	case "TXT":
		return fmt.Sprintf("%s\t%d\t%s\t%s\t\"%s\"",
			r.Name, r.TTL, r.Class, r.Type, r.Value)
	default:
		return fmt.Sprintf("%s\t%d\t%s\t%s\t%s",
			r.Name, r.TTL, r.Class, r.Type, r.Value)
	}
}

// RenderZoneFile renders a BIND9 zone file from the given data.
func RenderZoneFile(data ZoneFileData) (string, error) {
	funcMap := template.FuncMap{
		"renderRecord": renderRecord,
	}

	tmpl, err := template.New("zone-file").Funcs(funcMap).Parse(zoneFileTemplate)
	if err != nil {
		return "", err
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", err
	}

	return buf.String(), nil
}
