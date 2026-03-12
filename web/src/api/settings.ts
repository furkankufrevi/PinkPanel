import api from "./client";

export interface ActivityEntry {
  id: number;
  admin_id: number;
  username: string;
  action: string;
  target_type?: string;
  target_id?: number;
  details?: string;
  ip_address?: string;
  created_at: string;
}

export interface ServerInfo {
  panel_version: string;
  system: {
    os: string;
    arch: string;
    hostname: string;
    cpu_usage: number;
    ram: { total: number; used: number; free: number };
    disk: { mount: string; filesystem: string; total: string; used: string; available: string; use_percent: string }[];
    uptime: string;
    load_avg: string;
  };
}

export async function getActivityLog(limit: number = 50) {
  const { data } = await api.get<{ data: ActivityEntry[] }>("/settings/activity", {
    params: { limit },
  });
  return data;
}

export async function getServerInfo() {
  const { data } = await api.get<ServerInfo>("/settings/server-info");
  return data;
}

export interface SessionEntry {
  id: number;
  admin_id: number;
  ip_address: string;
  user_agent: string;
  created_at: string;
  expires_at: string;
  current: boolean;
}

export async function getSessions() {
  const { data } = await api.get<{ data: SessionEntry[] }>("/auth/sessions");
  return data;
}

export async function revokeSession(id: number) {
  await api.delete(`/auth/sessions/${id}`);
}
