import api from "./client";
import type { Subdomain } from "@/types/subdomain";

export async function listSubdomains(domainId: number) {
  const { data } = await api.get<{ data: Subdomain[] }>(
    `/domains/${domainId}/subdomains`
  );
  return data;
}

export async function createSubdomain(domainId: number, name: string) {
  const { data } = await api.post<Subdomain>(
    `/domains/${domainId}/subdomains`,
    { name }
  );
  return data;
}

export async function deleteSubdomain(domainId: number, subdomainId: number) {
  const { data } = await api.delete(
    `/domains/${domainId}/subdomains/${subdomainId}`
  );
  return data;
}
