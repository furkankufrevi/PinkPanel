package ssl

import (
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/rs/zerolog/log"

	"github.com/pinkpanel/pinkpanel/internal/agent"
	"github.com/pinkpanel/pinkpanel/internal/core/dns"
	"github.com/pinkpanel/pinkpanel/internal/core/domain"
	tmpl "github.com/pinkpanel/pinkpanel/internal/template"
)

// bindDNS01Provider implements lego's challenge.Provider and challenge.ProviderTimeout
// using PinkPanel's own BIND DNS server for DNS-01 challenges (needed for wildcards).
type bindDNS01Provider struct {
	dnsSvc      *dns.Service
	domainSvc   *domain.Service
	agentClient *agent.Client
	domainID    int64
	domainName  string
	records     map[string]int64 // fqdn -> dns record ID for cleanup
	mu          sync.Mutex
}

// Present creates a _acme-challenge TXT record in the domain's zone.
func (p *bindDNS01Provider) Present(domain, token, keyAuth string) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	// Create _acme-challenge TXT record
	rec, err := p.dnsSvc.Create(p.domainID, "TXT", "_acme-challenge", keyAuth, 60, nil)
	if err != nil {
		return fmt.Errorf("creating ACME challenge TXT record: %w", err)
	}

	if p.records == nil {
		p.records = make(map[string]int64)
	}
	p.records[domain] = rec.ID

	// Regenerate zone file and reload BIND
	if err := p.reloadZone(); err != nil {
		return fmt.Errorf("reloading DNS zone after challenge present: %w", err)
	}

	log.Info().Str("domain", domain).Msg("DNS-01 challenge TXT record created")
	return nil
}

// CleanUp removes the _acme-challenge TXT record.
func (p *bindDNS01Provider) CleanUp(domain, token, keyAuth string) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	recID, ok := p.records[domain]
	if !ok {
		return nil
	}

	if err := p.dnsSvc.Delete(recID); err != nil {
		log.Warn().Err(err).Str("domain", domain).Msg("failed to delete ACME challenge TXT record")
		return nil
	}
	delete(p.records, domain)

	// Regenerate zone file and reload BIND
	if err := p.reloadZone(); err != nil {
		log.Warn().Err(err).Str("domain", domain).Msg("failed to reload DNS zone after challenge cleanup")
	}

	log.Info().Str("domain", domain).Msg("DNS-01 challenge TXT record cleaned up")
	return nil
}

// Timeout returns generous timeouts since we control the DNS server directly.
func (p *bindDNS01Provider) Timeout() (timeout, interval time.Duration) {
	return 180 * time.Second, 5 * time.Second
}

// reloadZone regenerates the BIND zone file and reloads the DNS server.
func (p *bindDNS01Provider) reloadZone() error {
	records, err := p.dnsSvc.ListByDomain(p.domainID)
	if err != nil {
		return fmt.Errorf("listing DNS records: %w", err)
	}

	zoneRecords := make([]tmpl.ZoneRecord, 0, len(records))
	for _, r := range records {
		zr := tmpl.ZoneRecord{
			Name:  r.Name,
			TTL:   r.TTL,
			Class: "IN",
			Type:  r.Type,
			Value: r.Value,
		}
		if r.Priority != nil {
			zr.Priority = *r.Priority
		}
		zoneRecords = append(zoneRecords, zr)
	}

	// Include A/AAAA records for subdomains with separate_dns=0
	if p.domainSvc != nil {
		children, err := p.domainSvc.GetChildren(p.domainID)
		if err == nil {
			serverIP := getProviderServerIP()
			serverIPv6 := getProviderServerIPv6()
			existing := make(map[string]bool)
			for _, zr := range zoneRecords {
				existing[zr.Name+"/"+zr.Type] = true
			}
			for _, child := range children {
				if child.SeparateDNS {
					continue
				}
				prefix := extractPrefix(child.Name, p.domainName)
				if prefix == "" || prefix == child.Name {
					continue
				}
				if !existing[prefix+"/A"] {
					zoneRecords = append(zoneRecords, tmpl.ZoneRecord{
						Name: prefix, TTL: 3600, Class: "IN", Type: "A", Value: serverIP,
					})
				}
				if serverIPv6 != "" && !existing[prefix+"/AAAA"] {
					zoneRecords = append(zoneRecords, tmpl.ZoneRecord{
						Name: prefix, TTL: 3600, Class: "IN", Type: "AAAA", Value: serverIPv6,
					})
				}
			}
		}
	}

	zoneContent, err := tmpl.RenderZoneFile(tmpl.ZoneFileData{
		Domain:  p.domainName,
		Records: zoneRecords,
	})
	if err != nil {
		return fmt.Errorf("rendering zone file: %w", err)
	}

	if _, err := p.agentClient.Call("dns_write_zone", map[string]interface{}{
		"domain":  p.domainName,
		"content": zoneContent,
	}); err != nil {
		return fmt.Errorf("writing zone file: %w", err)
	}

	if _, err := p.agentClient.Call("dns_add_zone", map[string]interface{}{
		"domain": p.domainName,
	}); err != nil {
		return fmt.Errorf("adding zone to BIND: %w", err)
	}

	if _, err := p.agentClient.Call("dns_reload", nil); err != nil {
		return fmt.Errorf("reloading DNS: %w", err)
	}

	return nil
}

// extractPrefix extracts subdomain prefix from FQDN given parent name.
func extractPrefix(fqdn, parentName string) string {
	if len(fqdn) > len(parentName)+1 {
		return fqdn[:len(fqdn)-len(parentName)-1]
	}
	return fqdn
}

// getProviderServerIP returns the server's primary public IPv4 address.
func getProviderServerIP() string {
	addrs, _ := net.InterfaceAddrs()
	var fallback string
	for _, addr := range addrs {
		ipNet, ok := addr.(*net.IPNet)
		if !ok || ipNet.IP.IsLoopback() || ipNet.IP.To4() == nil {
			continue
		}
		if !ipNet.IP.IsPrivate() {
			return ipNet.IP.String()
		}
		if fallback == "" {
			fallback = ipNet.IP.String()
		}
	}
	if fallback != "" {
		return fallback
	}
	return "127.0.0.1"
}

// getProviderServerIPv6 returns the server's primary public IPv6 address.
func getProviderServerIPv6() string {
	addrs, _ := net.InterfaceAddrs()
	for _, addr := range addrs {
		ipNet, ok := addr.(*net.IPNet)
		if !ok || ipNet.IP.IsLoopback() || ipNet.IP.To4() != nil {
			continue
		}
		if ipNet.IP.IsLinkLocalUnicast() || ipNet.IP.IsLinkLocalMulticast() {
			continue
		}
		if !ipNet.IP.IsPrivate() {
			return ipNet.IP.String()
		}
	}
	return ""
}
