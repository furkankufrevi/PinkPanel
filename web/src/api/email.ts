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
