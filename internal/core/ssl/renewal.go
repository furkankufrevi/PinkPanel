package ssl

import (
	"fmt"
	"strings"
	"time"

	"github.com/rs/zerolog/log"

	"github.com/pinkpanel/pinkpanel/internal/agent"
	tmpl "github.com/pinkpanel/pinkpanel/internal/template"
)

// RenewalService periodically checks for expiring certificates and renews them.
type RenewalService struct {
	SSLSvc      *Service
	ACMESvc     *ACMEService
	AgentClient *agent.Client
	stop        chan struct{}
}

// CertForRenewal holds certificate + domain info needed for renewal.
type CertForRenewal struct {
	Certificate
	DomainName   string
	DocumentRoot string
	PHPVersion   string
	// ChallengeType, MailSSL, DomainID are inherited from Certificate
}

// ListExpiringCerts returns Let's Encrypt certs expiring within the given duration.
func (s *Service) ListExpiringCerts(within time.Duration) ([]CertForRenewal, error) {
	cutoff := time.Now().Add(within).Format(time.RFC3339)
	rows, err := s.DB.Query(`
		SELECT c.id, c.domain_id, c.type, c.cert_path, c.key_path, c.chain_path,
		       c.issuer, c.domains, c.expires_at, c.auto_renew, c.force_https,
		       c.hsts, c.mail_ssl, c.challenge_type,
		       d.name, d.document_root, d.php_version
		FROM ssl_certificates c
		JOIN domains d ON d.id = c.domain_id
		WHERE c.type = 'letsencrypt'
		  AND c.auto_renew = 1
		  AND c.expires_at <= ?
	`, cutoff)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var certs []CertForRenewal
	for rows.Next() {
		var c CertForRenewal
		var chainPath, issuer, domains *string
		var forceHTTPS, hsts, mailSSL int
		var challengeType string
		err := rows.Scan(
			&c.ID, &c.DomainID, &c.Type, &c.CertPath, &c.KeyPath, &chainPath,
			&issuer, &domains, &c.ExpiresAt, &c.AutoRenew, &forceHTTPS,
			&hsts, &mailSSL, &challengeType,
			&c.DomainName, &c.DocumentRoot, &c.PHPVersion,
		)
		if err != nil {
			return nil, err
		}
		c.ChainPath = chainPath
		c.Issuer = issuer
		c.Domains = domains
		c.ForceHTTPS = forceHTTPS == 1
		c.HSTS = hsts == 1
		c.MailSSL = mailSSL == 1
		c.ChallengeType = challengeType
		if c.ChallengeType == "" {
			c.ChallengeType = "http-01"
		}
		certs = append(certs, c)
	}
	return certs, nil
}

// Start begins the renewal check loop. It checks every 12 hours.
func (r *RenewalService) Start() {
	r.stop = make(chan struct{})
	go func() {
		// Check immediately on startup
		r.checkAndRenew()

		ticker := time.NewTicker(12 * time.Hour)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				r.checkAndRenew()
			case <-r.stop:
				return
			}
		}
	}()
	log.Info().Msg("SSL auto-renewal service started (checks every 12h)")
}

// Stop stops the renewal service.
func (r *RenewalService) Stop() {
	if r.stop != nil {
		close(r.stop)
	}
}

func (r *RenewalService) checkAndRenew() {
	if r.ACMESvc == nil {
		return
	}

	certs, err := r.SSLSvc.ListExpiringCerts(30 * 24 * time.Hour) // 30 days
	if err != nil {
		log.Error().Err(err).Msg("failed to list expiring certificates")
		return
	}

	if len(certs) == 0 {
		return
	}

	log.Info().Int("count", len(certs)).Msg("found certificates due for renewal")

	for _, cert := range certs {
		r.renewCert(cert)
	}
}

func (r *RenewalService) renewCert(cert CertForRenewal) {
	logger := log.With().Str("domain", cert.DomainName).Int64("cert_id", cert.ID).Logger()
	logger.Info().Msg("renewing Let's Encrypt certificate")

	// Build domain list from stored domains field
	domains := []string{cert.DomainName}
	if cert.Domains != nil && *cert.Domains != "" {
		// Use stored domains list
		parts := splitDomains(*cert.Domains)
		if len(parts) > 0 {
			domains = parts
		}
	}

	// Issue certificate using the appropriate challenge type
	var issued *IssuedCert
	var err error
	if cert.ChallengeType == "dns-01" {
		issued, err = r.ACMESvc.IssueCertificateDNS01(domains, cert.DomainID, cert.DomainName)
	} else {
		issued, err = r.ACMESvc.IssueCertificate(domains, cert.DocumentRoot)
	}
	if err != nil {
		logger.Error().Err(err).Msg("renewal failed")
		return
	}

	// Write cert files
	resp, err := r.AgentClient.Call("ssl_write_cert", map[string]any{
		"domain": cert.DomainName, "cert": issued.Certificate, "key": issued.PrivateKey, "chain": issued.IssuerCert,
	})
	if err != nil {
		logger.Error().Err(err).Msg("failed to write renewed certificate")
		return
	}

	result, _ := resp.Result.(map[string]interface{})
	certPath, _ := result["cert_path"].(string)
	keyPath, _ := result["key_path"].(string)
	chainPath := ""
	if cp, ok := result["chain_path"].(string); ok {
		chainPath = cp
	}

	// Update database (preserve force_https, hsts, mail_ssl, challenge_type settings)
	if _, err := r.SSLSvc.Install(cert.DomainID, "letsencrypt", certPath, keyPath, chainPath, issued.Issuer, issued.Domains, issued.ExpiresAt, cert.ForceHTTPS, InstallOpts{
		HSTS:          cert.HSTS,
		MailSSL:       cert.MailSSL,
		ChallengeType: cert.ChallengeType,
	}); err != nil {
		logger.Error().Err(err).Msg("failed to update certificate in database")
		return
	}

	// If mail SSL is enabled, re-configure Postfix/Dovecot with renewed cert
	if cert.MailSSL {
		if _, err := r.AgentClient.Call("email_configure_ssl", map[string]any{
			"domain":    cert.DomainName,
			"cert_path": certPath,
			"key_path":  keyPath,
		}); err != nil {
			logger.Error().Err(err).Msg("failed to update mail SSL after renewal")
		}
	}

	// Update NGINX (preserve force_https and hsts settings)
	vhostContent, err := tmpl.RenderNginxVhost(tmpl.NginxVhostData{
		Domain: cert.DomainName, DocumentRoot: cert.DocumentRoot, PHPVersion: cert.PHPVersion,
		SSLEnabled: true, SSLCertPath: certPath, SSLKeyPath: keyPath, SSLChainPath: chainPath,
		ForceHTTPS: cert.ForceHTTPS, HTTP2: true, HSTS: cert.HSTS, HSTSMaxAge: 31536000,
	})
	if err != nil {
		logger.Error().Err(err).Msg("failed to render NGINX vhost")
		return
	}
	configPath := fmt.Sprintf("/etc/nginx/sites-available/%s.conf", cert.DomainName)
	enabledPath := fmt.Sprintf("/etc/nginx/sites-enabled/%s.conf", cert.DomainName)
	if _, err := r.AgentClient.Call("file_write", map[string]any{"path": configPath, "content": vhostContent, "mode": "0644"}); err != nil {
		logger.Error().Err(err).Msg("failed to write NGINX sites-available config")
		return
	}
	if _, err := r.AgentClient.Call("file_write", map[string]any{"path": enabledPath, "content": vhostContent, "mode": "0644"}); err != nil {
		logger.Error().Err(err).Msg("failed to write NGINX sites-enabled config")
	}
	if _, err := r.AgentClient.Call("nginx_reload", nil); err != nil {
		logger.Error().Err(err).Msg("failed to reload NGINX")
		return
	}

	logger.Info().Time("expires_at", issued.ExpiresAt).Msg("certificate renewed successfully")
}

func splitDomains(s string) []string {
	var result []string
	for _, part := range strings.Split(s, ",") {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}
