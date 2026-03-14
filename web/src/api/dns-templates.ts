import api from "@/api/client";
import type {
  DNSTemplate,
  CreateDNSTemplateRequest,
  ApplyTemplateRequest,
  SaveAsTemplateRequest,
} from "@/types/dns-template";
import type { DNSRecord } from "@/types/dns";

export async function listDNSTemplates(): Promise<{ data: DNSTemplate[] }> {
  const { data } = await api.get("/dns/templates");
  return data;
}

export async function getDNSTemplate(id: number): Promise<{ data: DNSTemplate }> {
  const { data } = await api.get(`/dns/templates/${id}`);
  return data;
}

export async function createDNSTemplate(req: CreateDNSTemplateRequest): Promise<{ data: DNSTemplate }> {
  const { data } = await api.post("/dns/templates", req);
  return data;
}

export async function updateDNSTemplate(id: number, req: CreateDNSTemplateRequest): Promise<{ data: DNSTemplate }> {
  const { data } = await api.put(`/dns/templates/${id}`, req);
  return data;
}

export async function deleteDNSTemplate(id: number): Promise<void> {
  await api.delete(`/dns/templates/${id}`);
}

export async function exportDNSTemplate(id: number): Promise<Blob> {
  const { data } = await api.get(`/dns/templates/${id}/export`, { responseType: "blob" });
  return data;
}

export async function importDNSTemplate(json: string): Promise<{ data: DNSTemplate }> {
  const { data } = await api.post("/dns/templates/import", json, {
    headers: { "Content-Type": "application/json" },
  });
  return data;
}

export async function applyDNSTemplate(
  domainId: number,
  req: ApplyTemplateRequest
): Promise<{ data: DNSRecord[]; message: string }> {
  const { data } = await api.post(`/domains/${domainId}/dns/apply-template`, req);
  return data;
}

export async function saveAsTemplate(
  domainId: number,
  req: SaveAsTemplateRequest
): Promise<{ data: DNSTemplate }> {
  const { data } = await api.post(`/domains/${domainId}/dns/save-template`, req);
  return data;
}
