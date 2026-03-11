package template

import (
	"bytes"
	"fmt"
	"time"
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

// RenderZoneFile renders a BIND9 zone file from the given data.
// SOA is always rendered first (required by BIND), then NS, then the rest.
func RenderZoneFile(data ZoneFileData) (string, error) {
	var buf bytes.Buffer
	serial := time.Now().Format("2006010215")

	buf.WriteString(fmt.Sprintf("$TTL 3600\n$ORIGIN %s.\n\n", data.Domain))

	// 1. SOA — find mname/rname from records, or use defaults
	mname := fmt.Sprintf("ns1.%s. admin.%s.", data.Domain, data.Domain)
	for _, r := range data.Records {
		if r.Type == "SOA" {
			mname = r.Value
			break
		}
	}

	buf.WriteString(fmt.Sprintf("@\tIN\tSOA\t%s (\n", mname))
	buf.WriteString(fmt.Sprintf("\t\t\t\t%s\t; serial\n", serial))
	buf.WriteString("\t\t\t\t3600\t\t; refresh\n")
	buf.WriteString("\t\t\t\t900\t\t; retry\n")
	buf.WriteString("\t\t\t\t1209600\t\t; expire\n")
	buf.WriteString("\t\t\t\t86400 )\t\t; minimum\n\n")

	// 2. NS records
	for _, r := range data.Records {
		if r.Type == "NS" {
			buf.WriteString(fmt.Sprintf("%s\t%d\tIN\tNS\t%s\n", r.Name, r.TTL, r.Value))
		}
	}
	buf.WriteString("\n")

	// 3. All other records (skip SOA and NS, already rendered)
	for _, r := range data.Records {
		if r.Type == "SOA" || r.Type == "NS" {
			continue
		}
		buf.WriteString(renderRecord(r))
		buf.WriteString("\n")
	}

	return buf.String(), nil
}

// renderRecord formats a single ZoneRecord as a BIND9 zone file line.
func renderRecord(r ZoneRecord) string {
	switch r.Type {
	case "MX":
		return fmt.Sprintf("%s\t%d\tIN\tMX\t%d\t%s",
			r.Name, r.TTL, r.Priority, r.Value)
	case "SRV":
		return fmt.Sprintf("%s\t%d\tIN\tSRV\t%d\t%s",
			r.Name, r.TTL, r.Priority, r.Value)
	case "TXT":
		return fmt.Sprintf("%s\t%d\tIN\tTXT\t\"%s\"",
			r.Name, r.TTL, r.Value)
	default:
		return fmt.Sprintf("%s\t%d\tIN\t%s\t%s",
			r.Name, r.TTL, r.Type, r.Value)
	}
}
