import api from "./client";
import type { Redirect, CreateRedirectRequest, UpdateRedirectRequest } from "@/types/redirect";

export async function listRedirects(domainId: number) {
  const { data } = await api.get<{ data: Redirect[] }>(`/domains/${domainId}/redirects`);
  return data;
}

export async function createRedirect(domainId: number, req: CreateRedirectRequest) {
  const { data } = await api.post<Redirect>(`/domains/${domainId}/redirects`, req);
  return data;
}

export async function updateRedirect(id: number, req: UpdateRedirectRequest) {
  const { data } = await api.put<Redirect>(`/redirects/${id}`, req);
  return data;
}

export async function deleteRedirect(id: number) {
  const { data } = await api.delete(`/redirects/${id}`);
  return data;
}
