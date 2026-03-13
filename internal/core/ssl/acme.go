package ssl

import (
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-acme/lego/v4/certcrypto"
	"github.com/go-acme/lego/v4/certificate"
	"github.com/go-acme/lego/v4/lego"
	"github.com/go-acme/lego/v4/registration"
	"github.com/rs/zerolog/log"

	"github.com/pinkpanel/pinkpanel/internal/agent"
	"github.com/pinkpanel/pinkpanel/internal/core/dns"
)

const (
	acmeDataDir = "/var/lib/pinkpanel/acme"
	accountFile = "account.json"
	keyFile     = "account-key.pem"
)

// ACMEUser implements the registration.User interface for lego.
type ACMEUser struct {
	Email        string                 `json:"email"`
	Registration *registration.Resource `json:"registration"`
	key          crypto.PrivateKey
}

func (u *ACMEUser) GetEmail() string                        { return u.Email }
func (u *ACMEUser) GetRegistration() *registration.Resource { return u.Registration }
func (u *ACMEUser) GetPrivateKey() crypto.PrivateKey         { return u.key }

// ACMEService handles Let's Encrypt certificate issuance and renewal.
type ACMEService struct {
	Email       string
	Staging     bool // use staging server for testing
	AgentClient *agent.Client
	DNSSvc      *dns.Service
}

// IssueCertificate obtains a Let's Encrypt certificate for the given domains.
// It uses the HTTP-01 challenge with the webroot method.
// Challenge files are written via the agent (which runs as root) to avoid
// read-only filesystem issues under ProtectSystem=strict.
func (a *ACMEService) IssueCertificate(domains []string, webRoot string) (*IssuedCert, error) {
	user, err := a.loadOrCreateAccount()
	if err != nil {
		return nil, fmt.Errorf("loading ACME account: %w", err)
	}

	config := lego.NewConfig(user)
	config.Certificate.KeyType = certcrypto.EC256

	if a.Staging {
		config.CADirURL = lego.LEDirectoryStaging
	}

	client, err := lego.NewClient(config)
	if err != nil {
		return nil, fmt.Errorf("creating ACME client: %w", err)
	}

	// Use HTTP-01 challenge with agent-backed webroot provider
	provider := &agentWebrootProvider{
		root:        webRoot,
		agentClient: a.AgentClient,
	}

	err = client.Challenge.SetHTTP01Provider(provider)
	if err != nil {
		return nil, fmt.Errorf("setting HTTP-01 provider: %w", err)
	}

	// Register if needed
	if user.Registration == nil {
		reg, err := client.Registration.Register(registration.RegisterOptions{TermsOfServiceAgreed: true})
		if err != nil {
			return nil, fmt.Errorf("registering ACME account: %w", err)
		}
		user.Registration = reg
		if err := a.saveAccount(user); err != nil {
			log.Error().Err(err).Msg("failed to save ACME account")
		}
	}

	request := certificate.ObtainRequest{
		Domains: domains,
		Bundle:  true,
	}

	certificates, err := client.Certificate.Obtain(request)
	if err != nil {
		return nil, fmt.Errorf("obtaining certificate: %w", err)
	}

	// Parse expiry from the certificate
	expiresAt := time.Now().Add(90 * 24 * time.Hour) // default 90 days
	if block, _ := pem.Decode(certificates.Certificate); block != nil {
		if cert, err := x509.ParseCertificate(block.Bytes); err == nil {
			expiresAt = cert.NotAfter
		}
	}

	// Parse issuer
	issuer := "Let's Encrypt"
	if block, _ := pem.Decode(certificates.Certificate); block != nil {
		if cert, err := x509.ParseCertificate(block.Bytes); err == nil {
			if len(cert.Issuer.Organization) > 0 {
				issuer = cert.Issuer.Organization[0]
			}
		}
	}

	return &IssuedCert{
		Certificate: string(certificates.Certificate),
		PrivateKey:  string(certificates.PrivateKey),
		IssuerCert:  string(certificates.IssuerCertificate),
		Domain:      certificates.Domain,
		Domains:     strings.Join(domains, ","),
		Issuer:      issuer,
		ExpiresAt:   expiresAt,
	}, nil
}

// IssueCertificateDNS01 obtains a Let's Encrypt certificate using the DNS-01 challenge.
// This is required for wildcard certificates. It uses PinkPanel's own BIND server
// to create and clean up _acme-challenge TXT records.
func (a *ACMEService) IssueCertificateDNS01(domains []string, domainID int64, domainName string) (*IssuedCert, error) {
	if a.DNSSvc == nil {
		return nil, fmt.Errorf("DNS service not configured for DNS-01 challenges")
	}

	user, err := a.loadOrCreateAccount()
	if err != nil {
		return nil, fmt.Errorf("loading ACME account: %w", err)
	}

	config := lego.NewConfig(user)
	config.Certificate.KeyType = certcrypto.EC256

	if a.Staging {
		config.CADirURL = lego.LEDirectoryStaging
	}

	client, err := lego.NewClient(config)
	if err != nil {
		return nil, fmt.Errorf("creating ACME client: %w", err)
	}

	// Use DNS-01 challenge with BIND provider
	provider := &bindDNS01Provider{
		dnsSvc:      a.DNSSvc,
		agentClient: a.AgentClient,
		domainID:    domainID,
		domainName:  domainName,
	}

	err = client.Challenge.SetDNS01Provider(provider)
	if err != nil {
		return nil, fmt.Errorf("setting DNS-01 provider: %w", err)
	}

	// Register if needed
	if user.Registration == nil {
		reg, err := client.Registration.Register(registration.RegisterOptions{TermsOfServiceAgreed: true})
		if err != nil {
			return nil, fmt.Errorf("registering ACME account: %w", err)
		}
		user.Registration = reg
		if err := a.saveAccount(user); err != nil {
			log.Error().Err(err).Msg("failed to save ACME account")
		}
	}

	request := certificate.ObtainRequest{
		Domains: domains,
		Bundle:  true,
	}

	certificates, err := client.Certificate.Obtain(request)
	if err != nil {
		return nil, fmt.Errorf("obtaining certificate via DNS-01: %w", err)
	}

	// Parse expiry from the certificate
	expiresAt := time.Now().Add(90 * 24 * time.Hour)
	if block, _ := pem.Decode(certificates.Certificate); block != nil {
		if cert, err := x509.ParseCertificate(block.Bytes); err == nil {
			expiresAt = cert.NotAfter
		}
	}

	// Parse issuer
	issuer := "Let's Encrypt"
	if block, _ := pem.Decode(certificates.Certificate); block != nil {
		if cert, err := x509.ParseCertificate(block.Bytes); err == nil {
			if len(cert.Issuer.Organization) > 0 {
				issuer = cert.Issuer.Organization[0]
			}
		}
	}

	return &IssuedCert{
		Certificate: string(certificates.Certificate),
		PrivateKey:  string(certificates.PrivateKey),
		IssuerCert:  string(certificates.IssuerCertificate),
		Domain:      certificates.Domain,
		Domains:     strings.Join(domains, ","),
		Issuer:      issuer,
		ExpiresAt:   expiresAt,
	}, nil
}

// IssuedCert holds the result of a successful certificate issuance.
type IssuedCert struct {
	Certificate string
	PrivateKey  string
	IssuerCert  string
	Domain      string
	Domains     string
	Issuer      string
	ExpiresAt   time.Time
}

func (a *ACMEService) loadOrCreateAccount() (*ACMEUser, error) {
	if err := os.MkdirAll(acmeDataDir, 0700); err != nil {
		return nil, err
	}

	keyPath := filepath.Join(acmeDataDir, keyFile)
	accountPath := filepath.Join(acmeDataDir, accountFile)

	// Try loading existing
	if _, err := os.Stat(keyPath); err == nil {
		return a.loadAccount(keyPath, accountPath)
	}

	// Generate new key
	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("generating key: %w", err)
	}

	// Save key
	keyBytes, err := x509.MarshalECPrivateKey(privateKey)
	if err != nil {
		return nil, fmt.Errorf("marshaling key: %w", err)
	}
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: keyBytes})
	if err := os.WriteFile(keyPath, keyPEM, 0600); err != nil {
		return nil, fmt.Errorf("writing key: %w", err)
	}

	return &ACMEUser{
		Email: a.Email,
		key:   privateKey,
	}, nil
}

func (a *ACMEService) loadAccount(keyPath, accountPath string) (*ACMEUser, error) {
	// Load private key
	keyPEM, err := os.ReadFile(keyPath)
	if err != nil {
		return nil, fmt.Errorf("reading key: %w", err)
	}
	block, _ := pem.Decode(keyPEM)
	if block == nil {
		return nil, fmt.Errorf("invalid PEM key")
	}
	privateKey, err := x509.ParseECPrivateKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("parsing key: %w", err)
	}

	user := &ACMEUser{
		Email: a.Email,
		key:   privateKey,
	}

	// Try loading account registration
	if data, err := os.ReadFile(accountPath); err == nil {
		if err := json.Unmarshal(data, user); err != nil {
			log.Warn().Err(err).Msg("failed to parse ACME account, will re-register")
		}
	}

	return user, nil
}

func (a *ACMEService) saveAccount(user *ACMEUser) error {
	data, err := json.MarshalIndent(user, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(acmeDataDir, accountFile), data, 0600)
}

// agentWebrootProvider implements the challenge.Provider interface for HTTP-01.
// It writes challenge tokens via the agent (which runs as root) so the server
// process doesn't need write access to document roots.
type agentWebrootProvider struct {
	root        string
	agentClient *agent.Client
}

func (w *agentWebrootProvider) Present(domain, token, keyAuth string) error {
	challengePath := filepath.Join(w.root, ".well-known", "acme-challenge", token)

	// Create challenge directory and write token file via agent
	challengeDir := filepath.Join(w.root, ".well-known", "acme-challenge")
	if _, err := w.agentClient.Call("dir_create", map[string]any{
		"path": challengeDir,
		"mode": "0755",
	}); err != nil {
		return fmt.Errorf("creating challenge directory via agent: %w", err)
	}

	if _, err := w.agentClient.Call("file_write", map[string]any{
		"path":    challengePath,
		"content": keyAuth,
		"mode":    "0644",
	}); err != nil {
		return fmt.Errorf("writing challenge token via agent: %w", err)
	}

	return nil
}

func (w *agentWebrootProvider) CleanUp(domain, token, keyAuth string) error {
	challengePath := filepath.Join(w.root, ".well-known", "acme-challenge", token)
	if _, err := w.agentClient.Call("file_delete", map[string]any{
		"path": challengePath,
	}); err != nil {
		log.Warn().Err(err).Str("path", challengePath).Msg("failed to clean up ACME challenge token")
	}
	return nil
}
