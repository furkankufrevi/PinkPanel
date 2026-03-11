import api from "@/api/client";
import type {
  DNSRecord,
  CreateDNSRecordRequest,
  UpdateDNSRecordRequest,
} from "@/types/dns";

export async function listDNSRecords(domainId: number): Promise<{ data: DNSRecord[] }> {
  const { data } = await api.get(`/domains/${domainId}/dns`);
  return data;
}

export async function createDNSRecord(
  domainId: number,
  req: CreateDNSRecordRequest
): Promise<DNSRecord> {
  const { data } = await api.post(`/domains/${domainId}/dns`, req);
  return data;
}

export async function updateDNSRecord(
  recordId: number,
  req: UpdateDNSRecordRequest
): Promise<DNSRecord> {
  const { data } = await api.put(`/dns/${recordId}`, req);
  return data;
}

export async function deleteDNSRecord(recordId: number): Promise<void> {
  await api.delete(`/dns/${recordId}`);
}

export async function resetDNSDefaults(domainId: number): Promise<{ data: DNSRecord[]; message: string }> {
  const { data } = await api.post(`/domains/${domainId}/dns/reset`);
  return data;
}
