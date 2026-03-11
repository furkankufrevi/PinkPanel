import api from "./client";
import type { FTPAccount, CreateFTPAccountRequest } from "@/types/ftp";

export async function listFTPAccounts(domainId?: number) {
  const params = domainId ? { domain_id: domainId } : {};
  const { data } = await api.get<{ data: FTPAccount[] }>("/ftp", { params });
  return data;
}

export async function getFTPAccount(id: number) {
  const { data } = await api.get<FTPAccount>(`/ftp/${id}`);
  return data;
}

export async function createFTPAccount(req: CreateFTPAccountRequest) {
  const { data } = await api.post<FTPAccount>("/ftp", req);
  return data;
}

export async function deleteFTPAccount(id: number) {
  const { data } = await api.delete(`/ftp/${id}`);
  return data;
}

export async function updateFTPQuota(id: number, quotaMB: number) {
  const { data } = await api.put(`/ftp/${id}/quota`, { quota_mb: quotaMB });
  return data;
}
