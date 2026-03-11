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
    ram: { total: number; used: number; free: number; percent: number };
    disk: { mount: string; total: number; used: number; free: number; percent: number }[];
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
