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
	Email      string
	Staging    bool // use staging server for testing
	WebRoot    string
}

// IssueCertificate obtains a Let's Encrypt certificate for the given domains.
// It uses the HTTP-01 challenge with the webroot method.
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

	// Use HTTP-01 challenge with webroot
	challengeDir := filepath.Join(webRoot, ".well-known", "acme-challenge")
	if err := os.MkdirAll(challengeDir, 0755); err != nil {
		return nil, fmt.Errorf("creating challenge directory: %w", err)
	}

	err = client.Challenge.SetHTTP01Provider(&webrootProvider{root: webRoot})
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
			issuer = cert.Issuer.Organization[0]
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

// webrootProvider implements the challenge.Provider interface for HTTP-01.
type webrootProvider struct {
	root string
}

func (w *webrootProvider) Present(domain, token, keyAuth string) error {
	challengeDir := filepath.Join(w.root, ".well-known", "acme-challenge")
	if err := os.MkdirAll(challengeDir, 0755); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(challengeDir, token), []byte(keyAuth), 0644)
}

func (w *webrootProvider) CleanUp(domain, token, keyAuth string) error {
	path := filepath.Join(w.root, ".well-known", "acme-challenge", token)
	os.Remove(path)
	return nil
}
