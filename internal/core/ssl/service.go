package ssl

import (
	"database/sql"
	"fmt"
	"time"
)

type Certificate struct {
	ID            int64      `json:"id"`
	DomainID      int64      `json:"domain_id"`
	Type          string     `json:"type"`
	CertPath      string     `json:"cert_path"`
	KeyPath       string     `json:"key_path"`
	ChainPath     *string    `json:"chain_path"`
	Issuer        *string    `json:"issuer"`
	Domains       *string    `json:"domains"`
	IssuedAt      *time.Time `json:"issued_at"`
	ExpiresAt     time.Time  `json:"expires_at"`
	AutoRenew     bool       `json:"auto_renew"`
	ForceHTTPS    bool       `json:"force_https"`
	HSTS          bool       `json:"hsts"`
	MailSSL       bool       `json:"mail_ssl"`
	ChallengeType string     `json:"challenge_type"`
	CreatedAt     string     `json:"created_at"`
	UpdatedAt     string     `json:"updated_at"`
}

type Service struct {
	DB *sql.DB
}

// GetByDomainID returns the SSL certificate for a domain, or nil if none exists.
func (s *Service) GetByDomainID(domainID int64) (*Certificate, error) {
	cert := &Certificate{}
	var autoRenew, forceHTTPS, hsts, mailSSL int
	var issuedAt sql.NullString
	var chainPath, issuer, domains, challengeType sql.NullString
	err := s.DB.QueryRow(`
		SELECT id, domain_id, type, cert_path, key_path, chain_path, issuer, domains,
		       issued_at, expires_at, auto_renew, force_https, hsts, mail_ssl, challenge_type,
		       created_at, updated_at
		FROM ssl_certificates WHERE domain_id = ?`, domainID,
	).Scan(
		&cert.ID, &cert.DomainID, &cert.Type, &cert.CertPath, &cert.KeyPath,
		&chainPath, &issuer, &domains, &issuedAt, &cert.ExpiresAt,
		&autoRenew, &forceHTTPS, &hsts, &mailSSL, &challengeType,
		&cert.CreatedAt, &cert.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("querying ssl certificate: %w", err)
	}
	cert.AutoRenew = autoRenew == 1
	cert.ForceHTTPS = forceHTTPS == 1
	cert.HSTS = hsts == 1
	cert.MailSSL = mailSSL == 1
	if challengeType.Valid && challengeType.String != "" {
		cert.ChallengeType = challengeType.String
	} else {
		cert.ChallengeType = "http-01"
	}
	if chainPath.Valid {
		cert.ChainPath = &chainPath.String
	}
	if issuer.Valid {
		cert.Issuer = &issuer.String
	}
	if domains.Valid {
		cert.Domains = &domains.String
	}
	if issuedAt.Valid {
		t, _ := time.Parse("2006-01-02T15:04:05Z", issuedAt.String)
		cert.IssuedAt = &t
	}
	return cert, nil
}

// Install creates or replaces the SSL certificate for a domain.
// InstallOpts holds optional fields for certificate installation.
type InstallOpts struct {
	HSTS          bool
	MailSSL       bool
	ChallengeType string // "http-01" or "dns-01"
}

func (s *Service) Install(domainID int64, certType, certPath, keyPath, chainPath, issuer, domainNames string, expiresAt time.Time, forceHTTPS bool, opts ...InstallOpts) (*Certificate, error) {
	var chainPathPtr, issuerPtr, domainsPtr *string
	if chainPath != "" {
		chainPathPtr = &chainPath
	}
	if issuer != "" {
		issuerPtr = &issuer
	}
	if domainNames != "" {
		domainsPtr = &domainNames
	}

	forceHTTPSVal := 0
	if forceHTTPS {
		forceHTTPSVal = 1
	}

	var opt InstallOpts
	if len(opts) > 0 {
		opt = opts[0]
	}
	if opt.ChallengeType == "" {
		opt.ChallengeType = "http-01"
	}
	hstsVal := 0
	if opt.HSTS {
		hstsVal = 1
	}
	mailSSLVal := 0
	if opt.MailSSL {
		mailSSLVal = 1
	}

	now := time.Now().UTC()
	res, err := s.DB.Exec(`
		INSERT INTO ssl_certificates (domain_id, type, cert_path, key_path, chain_path, issuer, domains, issued_at, expires_at, auto_renew, force_https, hsts, mail_ssl, challenge_type, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, 1, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(domain_id) DO UPDATE SET
			type = excluded.type,
			cert_path = excluded.cert_path,
			key_path = excluded.key_path,
			chain_path = excluded.chain_path,
			issuer = excluded.issuer,
			domains = excluded.domains,
			issued_at = excluded.issued_at,
			expires_at = excluded.expires_at,
			force_https = excluded.force_https,
			hsts = excluded.hsts,
			mail_ssl = excluded.mail_ssl,
			challenge_type = excluded.challenge_type,
			updated_at = excluded.updated_at`,
		domainID, certType, certPath, keyPath, chainPathPtr, issuerPtr, domainsPtr,
		now, expiresAt, forceHTTPSVal, hstsVal, mailSSLVal, opt.ChallengeType, now, now,
	)
	if err != nil {
		return nil, fmt.Errorf("installing ssl certificate: %w", err)
	}

	id, _ := res.LastInsertId()
	return &Certificate{
		ID:            id,
		DomainID:      domainID,
		Type:          certType,
		CertPath:      certPath,
		KeyPath:       keyPath,
		ChainPath:     chainPathPtr,
		Issuer:        issuerPtr,
		Domains:       domainsPtr,
		IssuedAt:      &now,
		ExpiresAt:     expiresAt,
		AutoRenew:     true,
		ForceHTTPS:    forceHTTPS,
		HSTS:          opt.HSTS,
		MailSSL:       opt.MailSSL,
		ChallengeType: opt.ChallengeType,
		CreatedAt:     now.Format(time.RFC3339),
		UpdatedAt:     now.Format(time.RFC3339),
	}, nil
}

// Delete removes the SSL certificate record for a domain.
func (s *Service) Delete(domainID int64) error {
	res, err := s.DB.Exec("DELETE FROM ssl_certificates WHERE domain_id = ?", domainID)
	if err != nil {
		return fmt.Errorf("deleting ssl certificate: %w", err)
	}
	rows, _ := res.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("no ssl certificate found for domain")
	}
	return nil
}

// ToggleAutoRenew enables or disables auto-renewal.
func (s *Service) ToggleAutoRenew(domainID int64, enabled bool) error {
	val := 0
	if enabled {
		val = 1
	}
	_, err := s.DB.Exec("UPDATE ssl_certificates SET auto_renew = ?, updated_at = datetime('now') WHERE domain_id = ?", val, domainID)
	return err
}

// ToggleForceHTTPS enables or disables HTTP→HTTPS redirect.
func (s *Service) ToggleForceHTTPS(domainID int64, enabled bool) error {
	val := 0
	if enabled {
		val = 1
	}
	_, err := s.DB.Exec("UPDATE ssl_certificates SET force_https = ?, updated_at = datetime('now') WHERE domain_id = ?", val, domainID)
	return err
}

// ToggleHSTS enables or disables HSTS for a domain.
func (s *Service) ToggleHSTS(domainID int64, enabled bool) error {
	val := 0
	if enabled {
		val = 1
	}
	_, err := s.DB.Exec("UPDATE ssl_certificates SET hsts = ?, updated_at = datetime('now') WHERE domain_id = ?", val, domainID)
	return err
}

// SetMailSSL sets whether the SSL cert is assigned to mail services.
func (s *Service) SetMailSSL(domainID int64, enabled bool) error {
	val := 0
	if enabled {
		val = 1
	}
	_, err := s.DB.Exec("UPDATE ssl_certificates SET mail_ssl = ?, updated_at = datetime('now') WHERE domain_id = ?", val, domainID)
	return err
}
