import api from "./client";

export interface EmailAccount {
  id: number;
  domain_id: number;
  address: string;
  quota_mb: number;
  enabled: boolean;
  created_at: string;
  updated_at: string;
}

export interface EmailForwarder {
  id: number;
  domain_id: number;
  source_address: string;
  destination: string;
  created_at: string;
}

export interface EmailDNSRecord {
  type: string;
  name: string;
  value: string;
  label: string;
  exists: boolean;
}

export interface MailQueueItem {
  queue_id: string;
  sender: string;
  recipients: string[];
  message_size: number;
  arrival_time: number;
  reason?: string;
}

// Accounts
export async function listEmailAccounts(domainId: number) {
  const res = await api.get<{ data: EmailAccount[] }>(`/domains/${domainId}/email/accounts`);
  return res.data;
}

export async function createEmailAccount(domainId: number, req: { address: string; password: string; quota_mb: number }) {
  const res = await api.post<EmailAccount>(`/domains/${domainId}/email/accounts`, req);
  return res.data;
}

export async function deleteEmailAccount(domainId: number, accountId: number) {
  await api.delete(`/domains/${domainId}/email/accounts/${accountId}`);
}

export async function updateEmailQuota(domainId: number, accountId: number, quota_mb: number) {
  await api.put(`/domains/${domainId}/email/accounts/${accountId}/quota`, { quota_mb });
}

export async function changeEmailPassword(domainId: number, accountId: number, password: string) {
  await api.put(`/domains/${domainId}/email/accounts/${accountId}/password`, { password });
}

export async function toggleEmailAccount(domainId: number, accountId: number, enabled: boolean) {
  await api.put(`/domains/${domainId}/email/accounts/${accountId}/toggle`, { enabled });
}

// Webmail
export async function openWebmail(domainId: number, accountId: number) {
  const res = await api.post<{ url: string }>(`/domains/${domainId}/email/accounts/${accountId}/webmail`);
  return res.data;
}

// Forwarders
export async function listEmailForwarders(domainId: number) {
  const res = await api.get<{ data: EmailForwarder[] }>(`/domains/${domainId}/email/forwarders`);
  return res.data;
}

export async function createEmailForwarder(domainId: number, req: { source_address: string; destination: string }) {
  const res = await api.post<EmailForwarder>(`/domains/${domainId}/email/forwarders`, req);
  return res.data;
}

export async function deleteEmailForwarder(domainId: number, fwdId: number) {
  await api.delete(`/domains/${domainId}/email/forwarders/${fwdId}`);
}

// DNS Records
export async function getEmailDNSRecords(domainId: number) {
  const res = await api.get<{ records: EmailDNSRecord[] }>(`/domains/${domainId}/email/dns-records`);
  return res.data;
}

export async function applyEmailDNSRecords(domainId: number) {
  const res = await api.post<{ status: string; created: number }>(`/domains/${domainId}/email/dns-records`);
  return res.data;
}

// SpamAssassin
export interface SpamSettings {
  id: number;
  domain_id: number;
  enabled: boolean;
  score_threshold: number;
  action: string;
  created_at: string;
  updated_at: string;
}

export interface SpamListEntry {
  id: number;
  domain_id: number;
  list_type: string;
  entry: string;
  created_at: string;
}

export async function getSpamSettings(domainId: number) {
  const res = await api.get<SpamSettings>(`/domains/${domainId}/email/spam`);
  return res.data;
}

export async function updateSpamSettings(domainId: number, settings: { enabled: boolean; score_threshold: number; action: string }) {
  await api.put(`/domains/${domainId}/email/spam`, settings);
}

export async function getSpamList(domainId: number, type: 'whitelist' | 'blacklist') {
  const res = await api.get<{ data: SpamListEntry[] }>(`/domains/${domainId}/email/spam/list/${type}`);
  return res.data;
}

export async function addSpamEntry(domainId: number, req: { list_type: string; entry: string }) {
  const res = await api.post<SpamListEntry>(`/domains/${domainId}/email/spam/list`, req);
  return res.data;
}

export async function deleteSpamEntry(domainId: number, entryId: number) {
  await api.delete(`/domains/${domainId}/email/spam/list/${entryId}`);
}

// ClamAV
export interface ClamAVStatus {
  enabled: boolean;
  clamav_running: boolean;
  freshclam_running: boolean;
  milter_running: boolean;
  version?: string;
}

export async function getClamAVStatus() {
  const res = await api.get<ClamAVStatus>("/email/clamav");
  return res.data;
}

export async function toggleClamAV(enabled: boolean) {
  await api.put("/email/clamav", { enabled });
}

// Autodiscovery
export interface AutodiscoveryStatus {
  configured: boolean;
  srv_records: boolean;
  autoconfig: boolean;
  autodiscover: boolean;
}

export async function getAutodiscoveryStatus(domainId: number) {
  const res = await api.get<AutodiscoveryStatus>(`/domains/${domainId}/email/autodiscovery`);
  return res.data;
}

export async function setupAutodiscovery(domainId: number) {
  const res = await api.post<{ status: string; created: number }>(`/domains/${domainId}/email/autodiscovery`);
  return res.data;
}

// Mail Queue
export async function getMailQueue() {
  const res = await api.get<{ queue: MailQueueItem[] }>("/email/queue");
  return res.data;
}

export async function flushMailQueue() {
  await api.post("/email/queue/flush");
}

export async function deleteMailQueueItem(queueId: string) {
  await api.delete(`/email/queue/${queueId}`);
}
