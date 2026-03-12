import api from "./client";
import type { Backup, CreateBackupRequest, BackupSchedule, CreateScheduleRequest, UpdateScheduleRequest } from "@/types/backup";

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

// Backup schedule API
export async function listSchedules() {
  const { data } = await api.get<{ data: BackupSchedule[] }>("/backup-schedules");
  return data;
}

export async function createSchedule(req: CreateScheduleRequest) {
  const { data } = await api.post<BackupSchedule>("/backup-schedules", req);
  return data;
}

export async function updateSchedule(id: number, req: UpdateScheduleRequest) {
  const { data } = await api.put<BackupSchedule>(`/backup-schedules/${id}`, req);
  return data;
}

export async function deleteSchedule(id: number) {
  await api.delete(`/backup-schedules/${id}`);
}

export async function downloadBackup(id: number) {
  const response = await api.get(`/backups/${id}/download`, {
    responseType: "blob",
  });
  const url = window.URL.createObjectURL(new Blob([response.data]));
  const link = document.createElement("a");
  link.href = url;
  const disposition = response.headers["content-disposition"];
  const match = disposition?.match(/filename="?([^"]+)"?/);
  link.download = match?.[1] ?? `backup-${id}.tar.gz`;
  document.body.appendChild(link);
  link.click();
  link.remove();
  window.URL.revokeObjectURL(url);
}
