import api from "@/api/client";
import type {
  Domain,
  CreateDomainRequest,
  UpdateDomainRequest,
  DomainListResponse,
} from "@/types/domain";

export async function listDomains(params?: {
  search?: string;
  status?: string;
  page?: number;
  per_page?: number;
}): Promise<DomainListResponse> {
  const { data } = await api.get("/domains", { params });
  return data;
}

export async function getDomain(id: number): Promise<Domain> {
  const { data } = await api.get(`/domains/${id}`);
  return data;
}

export interface CreateDomainResult {
  domain: Domain;
  warnings: string[];
}

export async function createDomain(
  req: CreateDomainRequest
): Promise<CreateDomainResult> {
  const { data } = await api.post("/domains", req);
  // Response is either { data: Domain, warnings: [...] } or just Domain
  if (data.data) {
    return { domain: data.data, warnings: data.warnings ?? [] };
  }
  return { domain: data, warnings: [] };
}

export async function updateDomain(
  id: number,
  req: UpdateDomainRequest
): Promise<Domain> {
  const { data } = await api.put(`/domains/${id}`, req);
  return data;
}

export async function deleteDomain(
  id: number,
  removeFiles = false
): Promise<void> {
  await api.delete(`/domains/${id}`, {
    params: removeFiles ? { remove_files: "true" } : undefined,
  });
}

export async function suspendDomain(id: number): Promise<void> {
  await api.post(`/domains/${id}/suspend`);
}

export async function activateDomain(id: number): Promise<void> {
  await api.post(`/domains/${id}/activate`);
}
