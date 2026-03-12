package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/gofiber/fiber/v2"
	"github.com/rs/zerolog/log"

	"github.com/pinkpanel/pinkpanel/internal/agent"
	"github.com/pinkpanel/pinkpanel/internal/core/dns"
	"github.com/pinkpanel/pinkpanel/internal/core/domain"
	"github.com/pinkpanel/pinkpanel/internal/core/email"
	"github.com/pinkpanel/pinkpanel/internal/db"
)

type EmailHandler struct {
	DB          *sql.DB
	EmailSvc    *email.Service
	DomainSvc   *domain.Service
	DNSSvc      *dns.Service
	AgentClient *agent.Client
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

	// Update Postfix virtual maps
	h.syncVirtualMaps()

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

	// Get server IP
	serverIP := getServerIP()

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
			"value":  fmt.Sprintf("v=spf1 a mx ip4:%s ~all", serverIP),
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
	existingRecords, _ := h.DNSSvc.ListByDomain(domainID)

	hasSPF, hasDKIM, hasDMARC := false, false, false
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

	created := 0

	if !hasSPF {
		_, err := h.DNSSvc.Create(domainID, "@", "TXT", fmt.Sprintf("v=spf1 a mx ip4:%s ~all", serverIP), 3600, nil)
		if err == nil {
			created++
		}
	}

	if !hasDMARC {
		_, err := h.DNSSvc.Create(domainID, "_dmarc", "TXT", fmt.Sprintf("v=DMARC1; p=quarantine; rua=mailto:postmaster@%s", dom.Name), 3600, nil)
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
				_, err := h.DNSSvc.Create(domainID, dkimResult.Selector+"._domainkey", "TXT", dkimResult.PublicKey, 3600, nil)
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
	records, err := h.DNSSvc.ListByDomain(domainID)
	if err != nil {
		return
	}

	// Build zone content (simplified — use the same template as DNS handler)
	// We call the agent to write and reload the zone
	type zoneRecord struct {
		Name     string `json:"name"`
		TTL      int    `json:"ttl"`
		Class    string `json:"class"`
		Type     string `json:"type"`
		Value    string `json:"value"`
		Priority int    `json:"priority,omitempty"`
	}
	var zoneRecords []zoneRecord
	for _, r := range records {
		zr := zoneRecord{Name: r.Name, TTL: r.TTL, Class: "IN", Type: r.Type, Value: r.Value}
		if r.Priority != nil {
			zr.Priority = *r.Priority
		}
		zoneRecords = append(zoneRecords, zr)
	}

	// Use the template package to render
	// For simplicity, call dns_write_zone + dns_reload through agent
	// The DNS handler's regenerateZone is not easily reusable since it takes *DNSHandler
	// Instead, we use the template directly
	tmplPkg, _ := getZoneContent(domainName, records)
	if tmplPkg != "" {
		h.AgentClient.Call("dns_write_zone", map[string]any{
			"domain":  domainName,
			"content": tmplPkg,
		})
		h.AgentClient.Call("dns_reload", nil)
	}
}

// getZoneContent builds a zone file from records (lightweight version).
func getZoneContent(domain string, records []dns.Record) (string, error) {
	// Use the template package
	zoneRecords := make([]struct {
		Name     string
		TTL      int
		Class    string
		Type     string
		Value    string
		Priority int
	}, 0, len(records))

	for _, r := range records {
		zr := struct {
			Name     string
			TTL      int
			Class    string
			Type     string
			Value    string
			Priority int
		}{Name: r.Name, TTL: r.TTL, Class: "IN", Type: r.Type, Value: r.Value}
		if r.Priority != nil {
			zr.Priority = *r.Priority
		}
		zoneRecords = append(zoneRecords, zr)
	}

	// Build simple zone file
	content := fmt.Sprintf("$TTL 3600\n$ORIGIN %s.\n\n", domain)
	for _, r := range zoneRecords {
		name := r.Name
		if name == "@" {
			name = domain + "."
		} else if name != "" {
			name = name + "." + domain + "."
		}

		switch r.Type {
		case "MX":
			content += fmt.Sprintf("%-30s %d IN %-6s %d %s\n", name, r.TTL, r.Type, r.Priority, r.Value)
		case "TXT":
			content += fmt.Sprintf("%-30s %d IN %-6s \"%s\"\n", name, r.TTL, r.Type, r.Value)
		default:
			content += fmt.Sprintf("%-30s %d IN %-6s %s\n", name, r.TTL, r.Type, r.Value)
		}
	}
	return content, nil
}

func containsSubstr(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
