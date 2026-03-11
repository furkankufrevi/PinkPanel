import api from "./client";
import type { Backup, CreateBackupRequest } from "@/types/backup";

export async function listBackups(domainId?: number) {
  const params = domainId ? { domain_id: domainId } : {};
  const { data } = await api.get<{ data: Backup[] }>("/backups", { params });
  return data;
}

export async function getBackup(id: number) {
  const { data } = await api.get<Backup>(`/backups/${id}`);
  return data;
}

export async function createBackup(req: CreateBackupRequest) {
  const { data } = await api.post<Backup>("/backups", req);
  return data;
}

export async function deleteBackup(id: number) {
  const { data } = await api.delete(`/backups/${id}`);
  return data;
}

export async function restoreBackup(id: number) {
  const { data } = await api.post(`/backups/${id}/restore`);
  return data;
}
