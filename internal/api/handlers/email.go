package handlers

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/gofiber/fiber/v2"
	"github.com/rs/zerolog/log"

	"github.com/pinkpanel/pinkpanel/internal/agent"
	"github.com/pinkpanel/pinkpanel/internal/core/dns"
	"github.com/pinkpanel/pinkpanel/internal/core/domain"
	"github.com/pinkpanel/pinkpanel/internal/core/email"
	sslpkg "github.com/pinkpanel/pinkpanel/internal/core/ssl"
	"github.com/pinkpanel/pinkpanel/internal/db"
	tmpl "github.com/pinkpanel/pinkpanel/internal/template"
)

type EmailHandler struct {
	DB          *sql.DB
	EmailSvc    *email.Service
	DomainSvc   *domain.Service
	DNSSvc      *dns.Service
	AgentClient *agent.Client
	ACMESvc     *sslpkg.ACMEService
}

// ---------- Accounts ----------

// ListAccounts returns email accounts for a domain.
func (h *EmailHandler) ListAccounts(c *fiber.Ctx) error {
	domainID, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": fiber.Map{"code": "bad_request", "message": "invalid domain ID"}})
	}

	accounts, err := h.EmailSvc.ListAccounts(domainID)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": fiber.Map{"code": "internal_error", "message": err.Error()}})
	}
	if accounts == nil {
		accounts = []email.Account{}
	}
	return c.JSON(fiber.Map{"data": accounts})
}

// CreateAccount creates a new email account.
func (h *EmailHandler) CreateAccount(c *fiber.Ctx) error {
	domainID, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": fiber.Map{"code": "bad_request", "message": "invalid domain ID"}})
	}

	var req struct {
		Address  string `json:"address"`
		Password string `json:"password"`
		QuotaMB  int64  `json:"quota_mb"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": fiber.Map{"code": "bad_request", "message": "invalid request body"}})
	}
	if req.Address == "" || req.Password == "" {
		return c.Status(400).JSON(fiber.Map{"error": fiber.Map{"code": "validation_error", "message": "address and password are required"}})
	}

	dom, err := h.DomainSvc.GetByID(domainID)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{"error": fiber.Map{"code": "not_found", "message": "domain not found"}})
	}

	// Create via agent (Dovecot user + maildir)
	if _, err := h.AgentClient.Call("email_create_account", map[string]any{
		"domain":   dom.Name,
		"address":  req.Address,
		"password": req.Password,
		"quota_mb": req.QuotaMB,
	}); err != nil {
		return c.Status(500).JSON(fiber.Map{"error": fiber.Map{"code": "agent_error", "message": "failed to create email account: " + err.Error()}})
	}

	// Store in DB
	account, err := h.EmailSvc.CreateAccount(domainID, req.Address, req.QuotaMB)
	if err != nil {
		// Rollback agent
		if _, rollbackErr := h.AgentClient.Call("email_delete_account", map[string]any{
			"domain":  dom.Name,
			"address": req.Address,
		}); rollbackErr != nil {
			log.Error().Err(rollbackErr).Msg("failed to rollback email account creation")
		}
		return c.Status(400).JSON(fiber.Map{"error": fiber.Map{"code": "validation_error", "message": err.Error()}})
	}

	// Store encrypted password for webmail SSO
	if err := h.EmailSvc.StorePassword(account.ID, req.Password); err != nil {
		log.Error().Err(err).Msg("failed to store email password for webmail")
	}

	// Update Postfix virtual maps
	h.syncVirtualMaps()

	// Auto-setup DKIM + OpenDKIM tables for this domain if not already done
	go h.ensureDKIM(domainID, dom.Name)

	// Auto-setup RFC 5321 required forwarders (postmaster@, abuse@) on first account
	go h.ensureRFCForwarders(domainID, req.Address+"@"+dom.Name)

	adminID, _ := c.Locals("admin_id").(int64)
	db.LogActivity(h.DB, adminID, "create_email_account", "email", account.ID, req.Address+"@"+dom.Name, c.IP())

	return c.Status(201).JSON(account)
}

// DeleteAccount deletes an email account.
func (h *EmailHandler) DeleteAccount(c *fiber.Ctx) error {
	domainID, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": fiber.Map{"code": "bad_request", "message": "invalid domain ID"}})
	}

	accountID, err := strconv.ParseInt(c.Params("accountId"), 10, 64)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": fiber.Map{"code": "bad_request", "message": "invalid account ID"}})
	}

	account, err := h.EmailSvc.DeleteAccount(accountID)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{"error": fiber.Map{"code": "not_found", "message": err.Error()}})
	}

	// Verify account belongs to this domain
	if account.DomainID != domainID {
		return c.Status(404).JSON(fiber.Map{"error": fiber.Map{"code": "not_found", "message": "account not found for this domain"}})
	}

	dom, _ := h.DomainSvc.GetByID(domainID)
	domName := ""
	if dom != nil {
		domName = dom.Name
	}

	// Delete via agent
	if _, err := h.AgentClient.Call("email_delete_account", map[string]any{
		"domain":  domName,
		"address": account.Address,
	}); err != nil {
		log.Error().Err(err).Str("address", account.Address).Msg("failed to delete email account via agent")
	}

	h.syncVirtualMaps()

	adminID, _ := c.Locals("admin_id").(int64)
	db.LogActivity(h.DB, adminID, "delete_email_account", "email", accountID, account.Address+"@"+domName, c.IP())

	return c.JSON(fiber.Map{"status": "ok"})
}

// UpdateQuota updates the mailbox quota.
func (h *EmailHandler) UpdateQuota(c *fiber.Ctx) error {
	accountID, err := strconv.ParseInt(c.Params("accountId"), 10, 64)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": fiber.Map{"code": "bad_request", "message": "invalid account ID"}})
	}

	var req struct {
		QuotaMB int64 `json:"quota_mb"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": fiber.Map{"code": "bad_request", "message": "invalid request body"}})
	}

	if err := h.EmailSvc.UpdateQuota(accountID, req.QuotaMB); err != nil {
		return c.Status(500).JSON(fiber.Map{"error": fiber.Map{"code": "internal_error", "message": err.Error()}})
	}
	return c.JSON(fiber.Map{"status": "ok"})
}

// ChangePassword changes an email account's password.
func (h *EmailHandler) ChangePassword(c *fiber.Ctx) error {
	domainID, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": fiber.Map{"code": "bad_request", "message": "invalid domain ID"}})
	}

	accountID, err := strconv.ParseInt(c.Params("accountId"), 10, 64)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": fiber.Map{"code": "bad_request", "message": "invalid account ID"}})
	}

	var req struct {
		Password string `json:"password"`
	}
	if err := c.BodyParser(&req); err != nil || req.Password == "" {
		return c.Status(400).JSON(fiber.Map{"error": fiber.Map{"code": "validation_error", "message": "password is required"}})
	}

	account, err := h.EmailSvc.GetAccountByID(accountID)
	if err != nil || account.DomainID != domainID {
		return c.Status(404).JSON(fiber.Map{"error": fiber.Map{"code": "not_found", "message": "account not found"}})
	}

	dom, err := h.DomainSvc.GetByID(domainID)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{"error": fiber.Map{"code": "not_found", "message": "domain not found"}})
	}

	if _, err := h.AgentClient.Call("email_change_password", map[string]any{
		"domain":   dom.Name,
		"address":  account.Address,
		"password": req.Password,
	}); err != nil {
		return c.Status(500).JSON(fiber.Map{"error": fiber.Map{"code": "agent_error", "message": "failed to change password: " + err.Error()}})
	}

	// Update stored password for webmail SSO
	if err := h.EmailSvc.StorePassword(accountID, req.Password); err != nil {
		log.Error().Err(err).Msg("failed to update stored email password")
	}

	return c.JSON(fiber.Map{"status": "ok"})
}

// ToggleAccount enables or disables an email account.
func (h *EmailHandler) ToggleAccount(c *fiber.Ctx) error {
	accountID, err := strconv.ParseInt(c.Params("accountId"), 10, 64)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": fiber.Map{"code": "bad_request", "message": "invalid account ID"}})
	}

	var req struct {
		Enabled bool `json:"enabled"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": fiber.Map{"code": "bad_request", "message": "invalid request body"}})
	}

	if err := h.EmailSvc.ToggleAccount(accountID, req.Enabled); err != nil {
		return c.Status(500).JSON(fiber.Map{"error": fiber.Map{"code": "internal_error", "message": err.Error()}})
	}

	// Rebuild virtual maps (disabled accounts should be excluded)
	h.syncVirtualMaps()

	return c.JSON(fiber.Map{"status": "ok"})
}

// ---------- Forwarders ----------

// ListForwarders returns email forwarders for a domain.
func (h *EmailHandler) ListForwarders(c *fiber.Ctx) error {
	domainID, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": fiber.Map{"code": "bad_request", "message": "invalid domain ID"}})
	}

	forwarders, err := h.EmailSvc.ListForwarders(domainID)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": fiber.Map{"code": "internal_error", "message": err.Error()}})
	}
	if forwarders == nil {
		forwarders = []email.Forwarder{}
	}
	return c.JSON(fiber.Map{"data": forwarders})
}

// CreateForwarder creates a new email forwarder.
func (h *EmailHandler) CreateForwarder(c *fiber.Ctx) error {
	domainID, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": fiber.Map{"code": "bad_request", "message": "invalid domain ID"}})
	}

	var req struct {
		SourceAddress string `json:"source_address"`
		Destination   string `json:"destination"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": fiber.Map{"code": "bad_request", "message": "invalid request body"}})
	}
	if req.SourceAddress == "" || req.Destination == "" {
		return c.Status(400).JSON(fiber.Map{"error": fiber.Map{"code": "validation_error", "message": "source_address and destination are required"}})
	}

	fwd, err := h.EmailSvc.CreateForwarder(domainID, req.SourceAddress, req.Destination)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": fiber.Map{"code": "validation_error", "message": err.Error()}})
	}

	h.syncVirtualMaps()

	adminID, _ := c.Locals("admin_id").(int64)
	dom, _ := h.DomainSvc.GetByID(domainID)
	domName := ""
	if dom != nil {
		domName = dom.Name
	}
	db.LogActivity(h.DB, adminID, "create_email_forwarder", "email", fwd.ID, req.SourceAddress+"@"+domName+" -> "+req.Destination, c.IP())

	return c.Status(201).JSON(fwd)
}

// DeleteForwarder deletes an email forwarder.
func (h *EmailHandler) DeleteForwarder(c *fiber.Ctx) error {
	fwdID, err := strconv.ParseInt(c.Params("fwdId"), 10, 64)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": fiber.Map{"code": "bad_request", "message": "invalid forwarder ID"}})
	}

	fwd, err := h.EmailSvc.DeleteForwarder(fwdID)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{"error": fiber.Map{"code": "not_found", "message": err.Error()}})
	}

	h.syncVirtualMaps()

	adminID, _ := c.Locals("admin_id").(int64)
	db.LogActivity(h.DB, adminID, "delete_email_forwarder", "email", fwdID, fwd.SourceAddress, c.IP())

	return c.JSON(fiber.Map{"status": "ok"})
}

// ---------- DNS Records ----------

// GetDNSRecords returns recommended email DNS records (SPF, DKIM, DMARC).
func (h *EmailHandler) GetDNSRecords(c *fiber.Ctx) error {
	domainID, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": fiber.Map{"code": "bad_request", "message": "invalid domain ID"}})
	}

	dom, err := h.DomainSvc.GetByID(domainID)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{"error": fiber.Map{"code": "not_found", "message": "domain not found"}})
	}

	// Get server IPs
	serverIP := getServerIP()
	serverIPv6 := getServerIPv6()

	// Build recommended SPF value with optional IPv6
	spfValue := fmt.Sprintf("v=spf1 a mx ip4:%s", serverIP)
	if serverIPv6 != "" {
		spfValue += fmt.Sprintf(" ip6:%s", serverIPv6)
	}
	spfValue += " ~all"

	// Check existing DNS records
	existingRecords, _ := h.DNSSvc.ListByDomain(domainID)
	hasSPF := false
	hasDKIM := false
	hasDMARC := false
	for _, r := range existingRecords {
		if r.Type == "TXT" {
			if r.Name == "@" && containsSubstr(r.Value, "v=spf1") {
				hasSPF = true
			}
			if r.Name == "mail._domainkey" {
				hasDKIM = true
			}
			if r.Name == "_dmarc" {
				hasDMARC = true
			}
		}
	}

	recommendations := []fiber.Map{
		{
			"type":   "TXT",
			"name":   "@",
			"value":  spfValue,
			"label":  "SPF",
			"exists": hasSPF,
		},
		{
			"type":   "TXT",
			"name":   "_dmarc",
			"value":  fmt.Sprintf("v=DMARC1; p=quarantine; rua=mailto:postmaster@%s", dom.Name),
			"label":  "DMARC",
			"exists": hasDMARC,
		},
	}

	// Try to get DKIM public key
	dkimResp, err := h.AgentClient.Call("email_generate_dkim", map[string]any{"domain": dom.Name})
	if err == nil && dkimResp != nil {
		var dkimResult struct {
			PublicKey string `json:"public_key"`
			Selector  string `json:"selector"`
		}
		if raw, err := json.Marshal(dkimResp.Result); err == nil {
			json.Unmarshal(raw, &dkimResult)
		}
		if dkimResult.PublicKey != "" {
			recommendations = append(recommendations, fiber.Map{
				"type":   "TXT",
				"name":   dkimResult.Selector + "._domainkey",
				"value":  dkimResult.PublicKey,
				"label":  "DKIM",
				"exists": hasDKIM,
			})
		}
	}

	return c.JSON(fiber.Map{"records": recommendations})
}

// ApplyDNSRecords creates missing email DNS records.
func (h *EmailHandler) ApplyDNSRecords(c *fiber.Ctx) error {
	domainID, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": fiber.Map{"code": "bad_request", "message": "invalid domain ID"}})
	}

	dom, err := h.DomainSvc.GetByID(domainID)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{"error": fiber.Map{"code": "not_found", "message": "domain not found"}})
	}

	serverIP := getServerIP()
	serverIPv6 := getServerIPv6()
	existingRecords, _ := h.DNSSvc.ListByDomain(domainID)

	hasSPF, hasDKIM, hasDMARC := false, false, false
	var existingSPFID int64
	var existingSPFValue string
	for _, r := range existingRecords {
		if r.Type == "TXT" {
			if r.Name == "@" && containsSubstr(r.Value, "v=spf1") {
				hasSPF = true
				existingSPFID = r.ID
				existingSPFValue = r.Value
			}
			if r.Name == "mail._domainkey" {
				hasDKIM = true
			}
			if r.Name == "_dmarc" {
				hasDMARC = true
			}
		}
	}

	created := 0

	// Build correct SPF value
	spf := fmt.Sprintf("v=spf1 a mx ip4:%s", serverIP)
	if serverIPv6 != "" {
		spf += fmt.Sprintf(" ip6:%s", serverIPv6)
	}
	spf += " ~all"

	if !hasSPF {
		_, err := h.DNSSvc.Create(domainID, "TXT", "@", spf, 3600, nil)
		if err == nil {
			created++
		}
	} else if serverIPv6 != "" && !containsSubstr(existingSPFValue, "ip6:") {
		// Update existing SPF to include IPv6
		if _, err := h.DNSSvc.Update(existingSPFID, "TXT", "@", spf, 3600, nil); err == nil {
			created++
		}
	}

	if !hasDMARC {
		_, err := h.DNSSvc.Create(domainID, "TXT", "_dmarc", fmt.Sprintf("v=DMARC1; p=quarantine; rua=mailto:postmaster@%s", dom.Name), 3600, nil)
		if err == nil {
			created++
		}
	}

	if !hasDKIM {
		dkimResp, err := h.AgentClient.Call("email_generate_dkim", map[string]any{"domain": dom.Name})
		if err == nil && dkimResp != nil {
			var dkimResult struct {
				PublicKey string `json:"public_key"`
				Selector  string `json:"selector"`
			}
			if raw, err := json.Marshal(dkimResp.Result); err == nil {
				json.Unmarshal(raw, &dkimResult)
			}
			if dkimResult.PublicKey != "" {
				_, err := h.DNSSvc.Create(domainID, "TXT", dkimResult.Selector+"._domainkey", dkimResult.PublicKey, 3600, nil)
				if err == nil {
					created++
				}
			}
		}
	}

	// Regenerate DNS zone if records were created
	if created > 0 {
		// Use the DNS handler's regenerateZone via a direct agent call
		h.regenerateZone(domainID, dom.Name)
	}

	adminID, _ := c.Locals("admin_id").(int64)
	db.LogActivity(h.DB, adminID, "apply_email_dns", "email", domainID, fmt.Sprintf("%d records created", created), c.IP())

	return c.JSON(fiber.Map{"status": "ok", "created": created})
}

// ensureDKIM generates DKIM keys and publishes the DNS record for a domain
// if not already present. Safe to call multiple times — idempotent.
func (h *EmailHandler) ensureDKIM(domainID int64, domainName string) {
	defer func() {
		if r := recover(); r != nil {
			log.Error().Interface("panic", r).Str("domain", domainName).Msg("panic in ensureDKIM")
		}
	}()

	// Check if DKIM record already exists
	records, _ := h.DNSSvc.ListByDomain(domainID)
	for _, r := range records {
		if r.Type == "TXT" && containsSubstr(r.Name, "._domainkey") {
			// DKIM DNS record exists but ensure OpenDKIM tables are populated too
			if _, err := h.AgentClient.Call("email_generate_dkim", map[string]any{"domain": domainName}); err != nil {
				log.Error().Err(err).Str("domain", domainName).Msg("failed to ensure OpenDKIM tables")
			}
			return
		}
	}

	// Generate DKIM key + populate OpenDKIM tables
	dkimResp, err := h.AgentClient.Call("email_generate_dkim", map[string]any{"domain": domainName})
	if err != nil {
		log.Error().Err(err).Str("domain", domainName).Msg("failed to generate DKIM key")
		return
	}

	var dkimResult struct {
		PublicKey string `json:"public_key"`
		Selector  string `json:"selector"`
	}
	if raw, err := json.Marshal(dkimResp.Result); err == nil {
		json.Unmarshal(raw, &dkimResult)
	}

	if dkimResult.PublicKey == "" {
		log.Error().Str("domain", domainName).Msg("DKIM generation returned empty public key")
		return
	}

	// Create DNS record
	_, err = h.DNSSvc.Create(domainID, "TXT", dkimResult.Selector+"._domainkey", dkimResult.PublicKey, 3600, nil)
	if err != nil {
		log.Error().Err(err).Str("domain", domainName).Msg("failed to create DKIM DNS record")
		return
	}

	// Regenerate zone
	h.regenerateZone(domainID, domainName)
	log.Info().Str("domain", domainName).Msg("DKIM key generated and DNS record published")
}

// ensureRFCForwarders creates postmaster@ and abuse@ forwarders for a domain
// if they don't already exist (RFC 5321 requirement). The first email account
// created for the domain is used as the destination.
func (h *EmailHandler) ensureRFCForwarders(domainID int64, firstAccountEmail string) {
	defer func() {
		if r := recover(); r != nil {
			log.Error().Interface("panic", r).Msg("panic in ensureRFCForwarders")
		}
	}()

	// Check existing forwarders
	forwarders, _ := h.EmailSvc.ListForwarders(domainID)
	existing := make(map[string]bool)
	for _, f := range forwarders {
		existing[f.SourceAddress] = true
	}

	created := false
	for _, alias := range []string{"postmaster", "abuse"} {
		if existing[alias] {
			continue
		}
		if _, err := h.EmailSvc.CreateForwarder(domainID, alias, firstAccountEmail); err != nil {
			log.Error().Err(err).Str("alias", alias).Msg("failed to create RFC forwarder")
			continue
		}
		created = true
		log.Info().Str("alias", alias).Str("destination", firstAccountEmail).Msg("RFC forwarder created")
	}

	if created {
		h.syncVirtualMaps()
	}
}

// ---------- Mail Queue ----------

// ListQueue returns the Postfix mail queue.
func (h *EmailHandler) ListQueue(c *fiber.Ctx) error {
	resp, err := h.AgentClient.Call("email_queue_list", nil)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": fiber.Map{"code": "agent_error", "message": "failed to get mail queue: " + err.Error()}})
	}
	return c.JSON(resp.Result)
}

// FlushQueue flushes the Postfix mail queue.
func (h *EmailHandler) FlushQueue(c *fiber.Ctx) error {
	_, err := h.AgentClient.Call("email_queue_flush", nil)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": fiber.Map{"code": "agent_error", "message": "failed to flush mail queue: " + err.Error()}})
	}

	adminID, _ := c.Locals("admin_id").(int64)
	db.LogActivity(h.DB, adminID, "flush_mail_queue", "email", 0, "", c.IP())

	return c.JSON(fiber.Map{"status": "ok"})
}

// DeleteQueueItem deletes a specific item from the mail queue.
func (h *EmailHandler) DeleteQueueItem(c *fiber.Ctx) error {
	queueID := c.Params("queueId")
	if queueID == "" {
		return c.Status(400).JSON(fiber.Map{"error": fiber.Map{"code": "bad_request", "message": "queue ID is required"}})
	}

	_, err := h.AgentClient.Call("email_queue_delete", map[string]any{"queue_id": queueID})
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": fiber.Map{"code": "agent_error", "message": "failed to delete queue item: " + err.Error()}})
	}

	return c.JSON(fiber.Map{"status": "ok"})
}

// ---------- SpamAssassin ----------

// GetSpamSettings returns spam filter settings for a domain.
func (h *EmailHandler) GetSpamSettings(c *fiber.Ctx) error {
	domainID, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": fiber.Map{"code": "bad_request", "message": "invalid domain ID"}})
	}

	settings, err := h.EmailSvc.GetSpamSettings(domainID)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": fiber.Map{"code": "internal_error", "message": err.Error()}})
	}

	return c.JSON(settings)
}

// UpdateSpamSettings updates spam filter settings for a domain.
func (h *EmailHandler) UpdateSpamSettings(c *fiber.Ctx) error {
	domainID, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": fiber.Map{"code": "bad_request", "message": "invalid domain ID"}})
	}

	var req struct {
		Enabled        bool    `json:"enabled"`
		ScoreThreshold float64 `json:"score_threshold"`
		Action         string  `json:"action"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": fiber.Map{"code": "bad_request", "message": "invalid request body"}})
	}

	if err := h.EmailSvc.UpdateSpamSettings(domainID, req.Enabled, req.ScoreThreshold, req.Action); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": fiber.Map{"code": "validation_error", "message": err.Error()}})
	}

	// Get domain name and lists for agent config
	dom, _ := h.DomainSvc.GetByID(domainID)
	if dom != nil {
		whitelist, _ := h.EmailSvc.ListSpamEntries(domainID, "whitelist")
		blacklist, _ := h.EmailSvc.ListSpamEntries(domainID, "blacklist")

		var wl, bl []string
		for _, e := range whitelist {
			wl = append(wl, e.Entry)
		}
		for _, e := range blacklist {
			bl = append(bl, e.Entry)
		}

		h.AgentClient.Call("spam_configure", map[string]any{
			"domain":          dom.Name,
			"enabled":         req.Enabled,
			"score_threshold": req.ScoreThreshold,
			"action":          req.Action,
			"whitelist":       wl,
			"blacklist":       bl,
		})
	}

	adminID, _ := c.Locals("admin_id").(int64)
	db.LogActivity(h.DB, adminID, "update_spam_settings", "email", domainID, req.Action, c.IP())

	return c.JSON(fiber.Map{"status": "ok"})
}

// ListSpamEntries returns whitelist or blacklist entries.
func (h *EmailHandler) ListSpamEntries(c *fiber.Ctx) error {
	domainID, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": fiber.Map{"code": "bad_request", "message": "invalid domain ID"}})
	}

	listType := c.Params("type")
	if listType != "whitelist" && listType != "blacklist" {
		return c.Status(400).JSON(fiber.Map{"error": fiber.Map{"code": "bad_request", "message": "type must be whitelist or blacklist"}})
	}

	entries, err := h.EmailSvc.ListSpamEntries(domainID, listType)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": fiber.Map{"code": "internal_error", "message": err.Error()}})
	}
	if entries == nil {
		entries = []email.SpamListEntry{}
	}
	return c.JSON(fiber.Map{"data": entries})
}

// AddSpamEntry adds a whitelist or blacklist entry.
func (h *EmailHandler) AddSpamEntry(c *fiber.Ctx) error {
	domainID, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": fiber.Map{"code": "bad_request", "message": "invalid domain ID"}})
	}

	var req struct {
		ListType string `json:"list_type"`
		Entry    string `json:"entry"`
	}
	if err := c.BodyParser(&req); err != nil || req.Entry == "" {
		return c.Status(400).JSON(fiber.Map{"error": fiber.Map{"code": "validation_error", "message": "list_type and entry are required"}})
	}

	entry, err := h.EmailSvc.AddSpamEntry(domainID, req.ListType, req.Entry)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": fiber.Map{"code": "validation_error", "message": err.Error()}})
	}

	// Reconfigure SpamAssassin with updated lists
	h.reconfigureSpam(domainID)

	return c.Status(201).JSON(entry)
}

// DeleteSpamEntry removes a whitelist or blacklist entry.
func (h *EmailHandler) DeleteSpamEntry(c *fiber.Ctx) error {
	domainID, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": fiber.Map{"code": "bad_request", "message": "invalid domain ID"}})
	}

	entryID, err := strconv.ParseInt(c.Params("entryId"), 10, 64)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": fiber.Map{"code": "bad_request", "message": "invalid entry ID"}})
	}

	if err := h.EmailSvc.DeleteSpamEntry(entryID); err != nil {
		return c.Status(500).JSON(fiber.Map{"error": fiber.Map{"code": "internal_error", "message": err.Error()}})
	}

	h.reconfigureSpam(domainID)

	return c.JSON(fiber.Map{"status": "ok"})
}

// reconfigureSpam pushes current spam settings + lists to the agent.
func (h *EmailHandler) reconfigureSpam(domainID int64) {
	dom, _ := h.DomainSvc.GetByID(domainID)
	settings, _ := h.EmailSvc.GetSpamSettings(domainID)
	if dom == nil || settings == nil {
		return
	}

	whitelist, _ := h.EmailSvc.ListSpamEntries(domainID, "whitelist")
	blacklist, _ := h.EmailSvc.ListSpamEntries(domainID, "blacklist")

	var wl, bl []string
	for _, e := range whitelist {
		wl = append(wl, e.Entry)
	}
	for _, e := range blacklist {
		bl = append(bl, e.Entry)
	}

	h.AgentClient.Call("spam_configure", map[string]any{
		"domain":          dom.Name,
		"enabled":         settings.Enabled,
		"score_threshold": settings.ScoreThreshold,
		"action":          settings.Action,
		"whitelist":       wl,
		"blacklist":       bl,
	})
}

// ---------- ClamAV ----------

// GetClamAVStatus returns ClamAV service status.
func (h *EmailHandler) GetClamAVStatus(c *fiber.Ctx) error {
	resp, err := h.AgentClient.Call("clamav_status", nil)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": fiber.Map{"code": "agent_error", "message": err.Error()}})
	}
	return c.JSON(resp.Result)
}

// ToggleClamAV enables or disables ClamAV scanning.
func (h *EmailHandler) ToggleClamAV(c *fiber.Ctx) error {
	var req struct {
		Enabled bool `json:"enabled"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": fiber.Map{"code": "bad_request", "message": "invalid request body"}})
	}

	_, err := h.AgentClient.Call("clamav_configure", map[string]any{"enabled": req.Enabled})
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": fiber.Map{"code": "agent_error", "message": err.Error()}})
	}

	adminID, _ := c.Locals("admin_id").(int64)
	action := "disabled"
	if req.Enabled {
		action = "enabled"
	}
	db.LogActivity(h.DB, adminID, "toggle_clamav", "email", 0, "ClamAV "+action, c.IP())

	return c.JSON(fiber.Map{"status": "ok"})
}

// ---------- Mail Autodiscovery ----------

// GetAutodiscoveryStatus returns autodiscovery configuration status.
func (h *EmailHandler) GetAutodiscoveryStatus(c *fiber.Ctx) error {
	domainID, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": fiber.Map{"code": "bad_request", "message": "invalid domain ID"}})
	}

	records, _ := h.DNSSvc.ListByDomain(domainID)

	hasSRV := false
	hasAutoconfig := false
	hasAutodiscover := false

	for _, r := range records {
		switch {
		case r.Type == "SRV" && (r.Name == "_imaps._tcp" || r.Name == "_submission._tcp"):
			hasSRV = true
		case r.Type == "CNAME" && r.Name == "autoconfig":
			hasAutoconfig = true
		case r.Type == "CNAME" && r.Name == "autodiscover":
			hasAutodiscover = true
		}
	}

	return c.JSON(fiber.Map{
		"configured":   hasSRV && hasAutoconfig && hasAutodiscover,
		"srv_records":  hasSRV,
		"autoconfig":   hasAutoconfig,
		"autodiscover": hasAutodiscover,
	})
}

// SetupAutodiscovery creates DNS records and XML files for mail autodiscovery.
func (h *EmailHandler) SetupAutodiscovery(c *fiber.Ctx) error {
	domainID, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": fiber.Map{"code": "bad_request", "message": "invalid domain ID"}})
	}

	dom, err := h.DomainSvc.GetByID(domainID)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{"error": fiber.Map{"code": "not_found", "message": "domain not found"}})
	}

	records, _ := h.DNSSvc.ListByDomain(domainID)
	created := 0

	// Check existing records
	hasSRVIMAP, hasSRVSMTP, hasAutoconfig, hasAutodiscover := false, false, false, false
	for _, r := range records {
		switch {
		case r.Type == "SRV" && r.Name == "_imaps._tcp":
			hasSRVIMAP = true
		case r.Type == "SRV" && r.Name == "_submission._tcp":
			hasSRVSMTP = true
		case r.Type == "CNAME" && r.Name == "autoconfig":
			hasAutoconfig = true
		case r.Type == "CNAME" && r.Name == "autodiscover":
			hasAutodiscover = true
		}
	}

	// Create SRV records for IMAPS
	if !hasSRVIMAP {
		priority := 0
		_, err := h.DNSSvc.Create(domainID, "SRV", "_imaps._tcp", fmt.Sprintf("0 1 993 mail.%s.", dom.Name), 3600, &priority)
		if err == nil {
			created++
		}
	}

	// Create SRV records for submission
	if !hasSRVSMTP {
		priority := 0
		_, err := h.DNSSvc.Create(domainID, "SRV", "_submission._tcp", fmt.Sprintf("0 1 587 mail.%s.", dom.Name), 3600, &priority)
		if err == nil {
			created++
		}
	}

	// Create CNAME for autoconfig
	if !hasAutoconfig {
		_, err := h.DNSSvc.Create(domainID, "CNAME", "autoconfig", fmt.Sprintf("mail.%s.", dom.Name), 3600, nil)
		if err == nil {
			created++
		}
	}

	// Create CNAME for autodiscover
	if !hasAutodiscover {
		_, err := h.DNSSvc.Create(domainID, "CNAME", "autodiscover", fmt.Sprintf("mail.%s.", dom.Name), 3600, nil)
		if err == nil {
			created++
		}
	}

	// Write autoconfig/autodiscover XML files via agent
	h.AgentClient.Call("email_write_autoconfig", map[string]any{
		"domain":   dom.Name,
		"hostname": "mail." + dom.Name,
	})

	// Regenerate DNS zone
	if created > 0 {
		h.regenerateZone(domainID, dom.Name)
	}

	// Create mail vhost + SSL cert for mail.<domain>
	mailDomain := "mail." + dom.Name
	go h.setupMailVhost(mailDomain, dom.DocumentRoot)

	adminID, _ := c.Locals("admin_id").(int64)
	db.LogActivity(h.DB, adminID, "setup_autodiscovery", "email", domainID, fmt.Sprintf("%d DNS records created", created), c.IP())

	return c.JSON(fiber.Map{"status": "ok", "created": created})
}

// SetupMailVhost is an API endpoint to create/recreate the mail.<domain> vhost with SSL.
func (h *EmailHandler) SetupMailVhost(c *fiber.Ctx) error {
	domainID, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": fiber.Map{"code": "bad_request", "message": "invalid domain ID"}})
	}

	dom, err := h.DomainSvc.GetByID(domainID)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{"error": fiber.Map{"code": "not_found", "message": "domain not found"}})
	}

	mailDomain := "mail." + dom.Name
	go h.setupMailVhost(mailDomain, dom.DocumentRoot)

	adminID, _ := c.Locals("admin_id").(int64)
	db.LogActivity(h.DB, adminID, "setup_mail_vhost", "email", domainID, mailDomain, c.IP())

	return c.JSON(fiber.Map{"status": "ok", "message": "Mail vhost setup started for " + mailDomain})
}

// setupMailVhost creates an nginx vhost for mail.<domain> with SSL.
// Runs async — first creates HTTP-only vhost (for ACME challenge), issues SSL, then upgrades to HTTPS.
func (h *EmailHandler) setupMailVhost(mailDomain, documentRoot string) {
	defer func() {
		if r := recover(); r != nil {
			log.Error().Interface("panic", r).Str("domain", mailDomain).Msg("panic in setupMailVhost")
		}
	}()

	log.Info().Str("domain", mailDomain).Msg("setting up mail vhost")

	configPath := fmt.Sprintf("/etc/nginx/sites-available/%s.conf", mailDomain)
	enabledPath := fmt.Sprintf("/etc/nginx/sites-enabled/%s.conf", mailDomain)

	// Step 1: Create HTTP-only mail vhost (needed for ACME HTTP-01 challenge)
	httpVhost, err := tmpl.RenderNginxMailVhost(tmpl.NginxMailVhostData{
		Domain: mailDomain,
	})
	if err != nil {
		log.Error().Err(err).Str("domain", mailDomain).Msg("failed to render mail vhost")
		return
	}

	log.Info().Str("path", configPath).Msg("writing mail vhost config")
	if _, err := h.AgentClient.Call("file_write", map[string]any{
		"path": configPath, "content": httpVhost, "mode": "0644",
	}); err != nil {
		log.Error().Err(err).Str("path", configPath).Msg("failed to write mail nginx config")
		return
	}
	log.Info().Str("path", enabledPath).Msg("writing mail vhost enabled config")
	if _, err := h.AgentClient.Call("file_write", map[string]any{
		"path": enabledPath, "content": httpVhost, "mode": "0644",
	}); err != nil {
		log.Error().Err(err).Str("path", enabledPath).Msg("failed to write mail nginx enabled config")
		return
	}
	log.Info().Msg("testing nginx config for mail vhost")
	if _, err := h.AgentClient.Call("nginx_test", nil); err != nil {
		log.Error().Err(err).Msg("mail vhost nginx test failed — check snippets/roundcube.conf exists")
		return
	}
	if _, err := h.AgentClient.Call("nginx_reload", nil); err != nil {
		log.Error().Err(err).Msg("failed to reload nginx for mail vhost")
		return
	}
	log.Info().Str("domain", mailDomain).Msg("HTTP mail vhost created successfully")

	// Step 2: Issue SSL certificate for mail.<domain>
	if h.ACMESvc == nil {
		log.Warn().Str("domain", mailDomain).Msg("ACME service not configured, mail vhost created without SSL")
		return
	}

	// Use /var/www/html as webroot since the mail vhost uses that for ACME challenges
	issued, err := h.ACMESvc.IssueCertificate([]string{mailDomain}, "/var/www/html")
	if err != nil {
		log.Error().Err(err).Str("domain", mailDomain).Msg("failed to issue SSL for mail domain")
		return
	}

	// Write cert files via agent
	resp, err := h.AgentClient.Call("ssl_write_cert", map[string]any{
		"domain": mailDomain,
		"cert":   issued.Certificate,
		"key":    issued.PrivateKey,
		"chain":  issued.IssuerCert,
	})
	if err != nil {
		log.Error().Err(err).Msg("failed to write mail SSL cert files")
		return
	}

	result, _ := resp.Result.(map[string]interface{})
	certPath, _ := result["cert_path"].(string)
	keyPath, _ := result["key_path"].(string)
	chainPath := ""
	if cp, ok := result["chain_path"].(string); ok {
		chainPath = cp
	}

	// Step 3: Update vhost with SSL
	sslVhost, err := tmpl.RenderNginxMailVhost(tmpl.NginxMailVhostData{
		Domain:       mailDomain,
		SSLEnabled:   true,
		SSLCertPath:  certPath,
		SSLKeyPath:   keyPath,
		SSLChainPath: chainPath,
	})
	if err != nil {
		log.Error().Err(err).Msg("failed to render SSL mail vhost")
		return
	}

	if _, err := h.AgentClient.Call("file_write", map[string]any{
		"path": configPath, "content": sslVhost, "mode": "0644",
	}); err != nil {
		log.Error().Err(err).Msg("failed to write SSL mail nginx config")
		return
	}
	if _, err := h.AgentClient.Call("file_write", map[string]any{
		"path": enabledPath, "content": sslVhost, "mode": "0644",
	}); err != nil {
		log.Error().Err(err).Msg("failed to write SSL mail nginx enabled config")
		return
	}
	if _, err := h.AgentClient.Call("nginx_test", nil); err != nil {
		log.Error().Err(err).Msg("SSL mail vhost nginx test failed")
		return
	}
	if _, err := h.AgentClient.Call("nginx_reload", nil); err != nil {
		log.Error().Err(err).Msg("failed to reload nginx after mail SSL")
	}

	log.Info().Str("domain", mailDomain).Msg("mail vhost with SSL created successfully")
}

// ---------- Webmail ----------

// Webmail generates a one-time Roundcube signon token for auto-login.
func (h *EmailHandler) Webmail(c *fiber.Ctx) error {
	domainID, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": fiber.Map{"code": "bad_request", "message": "invalid domain ID"}})
	}

	accountID, err := strconv.ParseInt(c.Params("accountId"), 10, 64)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": fiber.Map{"code": "bad_request", "message": "invalid account ID"}})
	}

	account, err := h.EmailSvc.GetAccountByID(accountID)
	if err != nil || account.DomainID != domainID {
		return c.Status(404).JSON(fiber.Map{"error": fiber.Map{"code": "not_found", "message": "email account not found"}})
	}

	dom, err := h.DomainSvc.GetByID(domainID)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{"error": fiber.Map{"code": "not_found", "message": "domain not found"}})
	}

	password, err := h.EmailSvc.GetPassword(accountID)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": fiber.Map{"code": "no_password", "message": "No stored password. Change the account password first to enable webmail access."}})
	}

	// Generate one-time token
	tokenBytes := make([]byte, 32)
	if _, err := rand.Read(tokenBytes); err != nil {
		return c.Status(500).JSON(fiber.Map{"error": fiber.Map{"code": "internal_error", "message": "failed to generate token"}})
	}
	token := hex.EncodeToString(tokenBytes)

	fullEmail := account.Address + "@" + dom.Name
	tokenData, _ := json.Marshal(map[string]string{
		"username": fullEmail,
		"password": password,
	})

	// Write token file via agent
	tokenPath := fmt.Sprintf("/var/lib/pinkpanel/roundcube-tokens/%s.json", token)
	if _, err := h.AgentClient.Call("dir_create", map[string]any{
		"path": "/var/lib/pinkpanel/roundcube-tokens", "mode": "0755",
	}); err != nil {
		log.Error().Err(err).Msg("failed to create roundcube-tokens directory")
	}
	if _, err := h.AgentClient.Call("set_ownership", map[string]any{
		"owner": "www-data", "group": "www-data", "path": "/var/lib/pinkpanel/roundcube-tokens",
	}); err != nil {
		log.Error().Err(err).Msg("failed to chown roundcube-tokens directory")
	}
	if _, err := h.AgentClient.Call("file_write", map[string]any{
		"path": tokenPath, "content": string(tokenData), "mode": "0644",
	}); err != nil {
		return c.Status(500).JSON(fiber.Map{"error": fiber.Map{"code": "agent_error", "message": "failed to write signon token"}})
	}

	adminID, _ := c.Locals("admin_id").(int64)
	db.LogActivity(h.DB, adminID, "webmail_login", "email", accountID, fullEmail, c.IP())

	// Build absolute URL pointing to the mail vhost for this domain
	webmailURL := fmt.Sprintf("https://mail.%s/roundcube/signon.php?token=%s", dom.Name, token)

	return c.JSON(fiber.Map{
		"url": webmailURL,
	})
}

// ---------- Mail SSL ----------

// ConfigureMailSSL sets up SSL/TLS for Postfix and Dovecot using the domain's certificate.
func (h *EmailHandler) ConfigureMailSSL(c *fiber.Ctx) error {
	domainID, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": fiber.Map{"code": "bad_request", "message": "invalid domain ID"}})
	}

	dom, err := h.DomainSvc.GetByID(domainID)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{"error": fiber.Map{"code": "not_found", "message": "domain not found"}})
	}

	resp, err := h.AgentClient.Call("email_configure_ssl", map[string]any{
		"domain": dom.Name,
	})
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": fiber.Map{"code": "ssl_error", "message": err.Error()}})
	}

	adminID, _ := c.Locals("admin_id").(int64)
	db.LogActivity(h.DB, adminID, "email_ssl_configure", "email", domainID, dom.Name, c.IP())

	return c.JSON(resp.Result)
}

// GetMailSSLStatus checks if SSL is configured for mail services.
func (h *EmailHandler) GetMailSSLStatus(c *fiber.Ctx) error {
	domainID, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": fiber.Map{"code": "bad_request", "message": "invalid domain ID"}})
	}

	dom, err := h.DomainSvc.GetByID(domainID)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{"error": fiber.Map{"code": "not_found", "message": "domain not found"}})
	}

	resp, err := h.AgentClient.Call("email_ssl_status", map[string]any{
		"domain": dom.Name,
	})
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": fiber.Map{"code": "agent_error", "message": err.Error()}})
	}

	// Parse agent response and add domain
	var result map[string]any
	if b, err := json.Marshal(resp.Result); err == nil {
		json.Unmarshal(b, &result)
	}
	if result == nil {
		result = map[string]any{}
	}
	result["domain"] = dom.Name

	return c.JSON(result)
}

// ---------- Helpers ----------

// syncVirtualMaps rebuilds all Postfix virtual maps from the database.
func (h *EmailHandler) syncVirtualMaps() {
	// Get all domains that have email accounts
	rows, err := h.DB.Query(`
		SELECT DISTINCT d.id, d.name
		FROM domains d
		INNER JOIN email_accounts ea ON ea.domain_id = d.id
		WHERE ea.enabled = 1
	`)
	if err != nil {
		log.Error().Err(err).Msg("failed to query email domains")
		return
	}
	defer rows.Close()

	var domains []string
	mailboxes := make(map[string]string)
	aliases := make(map[string]string)

	for rows.Next() {
		var domID int64
		var domName string
		rows.Scan(&domID, &domName)
		domains = append(domains, domName)

		// Get enabled accounts for this domain
		accounts, _ := h.EmailSvc.ListAccounts(domID)
		for _, a := range accounts {
			if a.Enabled {
				fullAddr := a.Address + "@" + domName
				mailboxes[fullAddr] = domName + "/" + a.Address + "/"
			}
		}

		// Get forwarders for this domain
		forwarders, _ := h.EmailSvc.ListForwarders(domID)
		for _, f := range forwarders {
			fullAddr := f.SourceAddress + "@" + domName
			aliases[fullAddr] = f.Destination
		}
	}

	// Also include domains that only have forwarders
	fwdRows, err := h.DB.Query(`
		SELECT DISTINCT d.name
		FROM domains d
		INNER JOIN email_forwarders ef ON ef.domain_id = d.id
	`)
	if err == nil {
		defer fwdRows.Close()
		for fwdRows.Next() {
			var name string
			fwdRows.Scan(&name)
			found := false
			for _, d := range domains {
				if d == name {
					found = true
					break
				}
			}
			if !found {
				domains = append(domains, name)
			}
		}
	}

	if _, err := h.AgentClient.Call("email_update_virtual_maps", map[string]any{
		"domains":   domains,
		"mailboxes": mailboxes,
		"aliases":   aliases,
	}); err != nil {
		log.Error().Err(err).Msg("failed to update Postfix virtual maps")
	}
}

// regenerateZone triggers a DNS zone regeneration via the agent.
func (h *EmailHandler) regenerateZone(domainID int64, domainName string) {
	zoneRecords, err := buildZoneRecords(h.DNSSvc, h.DomainSvc, domainID, domainName)
	if err != nil {
		log.Error().Err(err).Str("domain", domainName).Msg("failed to build zone records")
		return
	}

	zoneContent, err := tmpl.RenderZoneFile(tmpl.ZoneFileData{
		Domain:  domainName,
		Records: zoneRecords,
	})
	if err != nil {
		log.Error().Err(err).Str("domain", domainName).Msg("failed to render zone file")
		return
	}

	if _, err := h.AgentClient.Call("dns_write_zone", map[string]any{
		"domain":  domainName,
		"content": zoneContent,
	}); err != nil {
		log.Error().Err(err).Str("domain", domainName).Msg("failed to write zone file")
		return
	}
	if _, err := h.AgentClient.Call("dns_reload", nil); err != nil {
		log.Error().Err(err).Str("domain", domainName).Msg("failed to reload DNS after zone update")
	}
}

func containsSubstr(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
