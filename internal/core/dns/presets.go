package dns

// PresetTemplate represents a built-in DNS template with placeholder records.
// Variables: {{domain}}, {{ip}}, {{ipv6}}, {{hostname}}
type PresetTemplate struct {
	ID          int64
	Name        string
	Description string
	Category    string
	Records     []PresetRecord
}

// PresetRecord is a DNS record within a preset template.
type PresetRecord struct {
	Type     string
	Name     string
	Value    string
	TTL      int
	Priority *int
}

func intPtr(v int) *int { return &v }

// Presets returns all built-in DNS templates. IDs are negative to avoid DB collision.
func Presets() []PresetTemplate {
	return []PresetTemplate{
		{
			ID:          -1,
			Name:        "PinkPanel Default",
			Description: "Standard hosting records with mail, SPF, DMARC, and autodiscovery",
			Category:    "hosting",
			Records: []PresetRecord{
				{Type: "SOA", Name: "@", Value: "ns1.{{domain}}. admin.{{domain}}.", TTL: 86400},
				{Type: "NS", Name: "@", Value: "ns1.{{domain}}.", TTL: 86400},
				{Type: "NS", Name: "@", Value: "ns2.{{domain}}.", TTL: 86400},
				{Type: "A", Name: "ns1", Value: "{{ip}}", TTL: 3600},
				{Type: "A", Name: "ns2", Value: "{{ip}}", TTL: 3600},
				{Type: "A", Name: "@", Value: "{{ip}}", TTL: 3600},
				{Type: "A", Name: "www", Value: "{{ip}}", TTL: 3600},
				{Type: "A", Name: "mail", Value: "{{ip}}", TTL: 3600},
				{Type: "MX", Name: "@", Value: "mail.{{domain}}.", TTL: 3600, Priority: intPtr(10)},
				{Type: "TXT", Name: "@", Value: "v=spf1 a mx ip4:{{ip}} ~all", TTL: 3600},
				{Type: "TXT", Name: "_dmarc", Value: "v=DMARC1; p=quarantine; rua=mailto:postmaster@{{domain}}", TTL: 3600},
				{Type: "SRV", Name: "_imaps._tcp", Value: "0 993 mail.{{domain}}.", TTL: 3600, Priority: intPtr(10)},
				{Type: "SRV", Name: "_submission._tcp", Value: "0 587 mail.{{domain}}.", TTL: 3600, Priority: intPtr(10)},
				{Type: "SRV", Name: "_autodiscover._tcp", Value: "0 443 mail.{{domain}}.", TTL: 3600, Priority: intPtr(10)},
			},
		},
		{
			ID:          -2,
			Name:        "Google Workspace",
			Description: "MX, SPF, and verification records for Google Workspace",
			Category:    "email",
			Records: []PresetRecord{
				{Type: "MX", Name: "@", Value: "aspmx.l.google.com.", TTL: 3600, Priority: intPtr(1)},
				{Type: "MX", Name: "@", Value: "alt1.aspmx.l.google.com.", TTL: 3600, Priority: intPtr(5)},
				{Type: "MX", Name: "@", Value: "alt2.aspmx.l.google.com.", TTL: 3600, Priority: intPtr(5)},
				{Type: "MX", Name: "@", Value: "alt3.aspmx.l.google.com.", TTL: 3600, Priority: intPtr(10)},
				{Type: "MX", Name: "@", Value: "alt4.aspmx.l.google.com.", TTL: 3600, Priority: intPtr(10)},
				{Type: "TXT", Name: "@", Value: "v=spf1 include:_spf.google.com ~all", TTL: 3600},
			},
		},
		{
			ID:          -3,
			Name:        "Microsoft 365",
			Description: "MX, SPF, and autodiscover records for Microsoft 365",
			Category:    "email",
			Records: []PresetRecord{
				{Type: "MX", Name: "@", Value: "{{domain}}.mail.protection.outlook.com.", TTL: 3600, Priority: intPtr(0)},
				{Type: "TXT", Name: "@", Value: "v=spf1 include:spf.protection.outlook.com -all", TTL: 3600},
				{Type: "CNAME", Name: "autodiscover", Value: "autodiscover.outlook.com.", TTL: 3600},
				{Type: "CNAME", Name: "sip", Value: "sipdir.online.lync.com.", TTL: 3600},
				{Type: "CNAME", Name: "lyncdiscover", Value: "webdir.online.lync.com.", TTL: 3600},
				{Type: "CNAME", Name: "enterpriseregistration", Value: "enterpriseregistration.windows.net.", TTL: 3600},
				{Type: "CNAME", Name: "enterpriseenrollment", Value: "enterpriseenrollment.manage.microsoft.com.", TTL: 3600},
				{Type: "SRV", Name: "_sip._tls", Value: "100 443 sipdir.online.lync.com.", TTL: 3600, Priority: intPtr(1)},
				{Type: "SRV", Name: "_sipfederationtls._tcp", Value: "100 5061 sipfed.online.lync.com.", TTL: 3600, Priority: intPtr(1)},
			},
		},
		{
			ID:          -4,
			Name:        "Zoho Mail",
			Description: "MX, SPF, and verification records for Zoho Mail",
			Category:    "email",
			Records: []PresetRecord{
				{Type: "MX", Name: "@", Value: "mx.zoho.com.", TTL: 3600, Priority: intPtr(10)},
				{Type: "MX", Name: "@", Value: "mx2.zoho.com.", TTL: 3600, Priority: intPtr(20)},
				{Type: "MX", Name: "@", Value: "mx3.zoho.com.", TTL: 3600, Priority: intPtr(50)},
				{Type: "TXT", Name: "@", Value: "v=spf1 include:zoho.com ~all", TTL: 3600},
			},
		},
		{
			ID:          -5,
			Name:        "ProtonMail",
			Description: "MX and SPF records for ProtonMail",
			Category:    "email",
			Records: []PresetRecord{
				{Type: "MX", Name: "@", Value: "mail.protonmail.ch.", TTL: 3600, Priority: intPtr(10)},
				{Type: "MX", Name: "@", Value: "mailsec.protonmail.ch.", TTL: 3600, Priority: intPtr(20)},
				{Type: "TXT", Name: "@", Value: "v=spf1 include:_spf.protonmail.ch mx ~all", TTL: 3600},
			},
		},
		{
			ID:          -6,
			Name:        "Fastmail",
			Description: "MX, SPF, and autodiscover records for Fastmail",
			Category:    "email",
			Records: []PresetRecord{
				{Type: "MX", Name: "@", Value: "in1-smtp.messagingengine.com.", TTL: 3600, Priority: intPtr(10)},
				{Type: "MX", Name: "@", Value: "in2-smtp.messagingengine.com.", TTL: 3600, Priority: intPtr(20)},
				{Type: "TXT", Name: "@", Value: "v=spf1 include:spf.messagingengine.com ~all", TTL: 3600},
				{Type: "SRV", Name: "_submission._tcp", Value: "0 587 smtp.fastmail.com.", TTL: 3600, Priority: intPtr(1)},
				{Type: "SRV", Name: "_imaps._tcp", Value: "0 993 imap.fastmail.com.", TTL: 3600, Priority: intPtr(1)},
				{Type: "SRV", Name: "_carddavs._tcp", Value: "0 443 carddav.fastmail.com.", TTL: 3600, Priority: intPtr(1)},
				{Type: "SRV", Name: "_caldavs._tcp", Value: "0 443 caldav.fastmail.com.", TTL: 3600, Priority: intPtr(1)},
			},
		},
		{
			ID:          -7,
			Name:        "AWS SES",
			Description: "MX and SPF records for Amazon Simple Email Service",
			Category:    "email",
			Records: []PresetRecord{
				{Type: "MX", Name: "@", Value: "inbound-smtp.us-east-1.amazonaws.com.", TTL: 3600, Priority: intPtr(10)},
				{Type: "TXT", Name: "@", Value: "v=spf1 include:amazonses.com ~all", TTL: 3600},
			},
		},
		{
			ID:          -8,
			Name:        "Vercel",
			Description: "A and CNAME records for Vercel hosting",
			Category:    "hosting",
			Records: []PresetRecord{
				{Type: "A", Name: "@", Value: "76.76.21.21", TTL: 3600},
				{Type: "CNAME", Name: "www", Value: "cname.vercel-dns.com.", TTL: 3600},
			},
		},
		{
			ID:          -9,
			Name:        "Netlify",
			Description: "A and CNAME records for Netlify hosting",
			Category:    "hosting",
			Records: []PresetRecord{
				{Type: "A", Name: "@", Value: "75.2.60.5", TTL: 3600},
				{Type: "CNAME", Name: "www", Value: "{{domain}}.netlify.app.", TTL: 3600},
			},
		},
		{
			ID:          -10,
			Name:        "GitHub Pages",
			Description: "A and CNAME records for GitHub Pages",
			Category:    "hosting",
			Records: []PresetRecord{
				{Type: "A", Name: "@", Value: "185.199.108.153", TTL: 3600},
				{Type: "A", Name: "@", Value: "185.199.109.153", TTL: 3600},
				{Type: "A", Name: "@", Value: "185.199.110.153", TTL: 3600},
				{Type: "A", Name: "@", Value: "185.199.111.153", TTL: 3600},
				{Type: "CNAME", Name: "www", Value: "{{domain}}.", TTL: 3600},
			},
		},
		{
			ID:          -11,
			Name:        "Cloudflare Pages",
			Description: "CNAME records for Cloudflare Pages",
			Category:    "hosting",
			Records: []PresetRecord{
				{Type: "CNAME", Name: "@", Value: "{{domain}}.pages.dev.", TTL: 3600},
				{Type: "CNAME", Name: "www", Value: "{{domain}}.pages.dev.", TTL: 3600},
			},
		},
	}
}
