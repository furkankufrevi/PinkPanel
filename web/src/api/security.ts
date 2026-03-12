import api from "./client";

export interface Fail2banStatus {
  raw: string;
  jails?: string[];
}

export interface Fail2banJailStatus {
  jail: string;
  raw: string;
  currently_failed?: number;
  total_failed?: number;
  currently_banned?: number;
  total_banned?: number;
  banned_ips?: string[];
}

export interface Fail2banBannedIPs {
  jail: string;
  banned_ips: string[];
}

export async function getFail2banStatus(): Promise<Fail2banStatus> {
  const response = await api.get<Fail2banStatus>("/security/fail2ban/status");
  return response.data;
}

export async function getFail2banJailStatus(jail: string): Promise<Fail2banJailStatus> {
  const response = await api.get<Fail2banJailStatus>(`/security/fail2ban/jails/${jail}`);
  return response.data;
}

export async function getFail2banBannedIPs(jail = "pinkpanel"): Promise<Fail2banBannedIPs> {
  const response = await api.get<Fail2banBannedIPs>("/security/fail2ban/banned", {
    params: { jail },
  });
  return response.data;
}

export async function banIP(ip: string, jail = "pinkpanel"): Promise<void> {
  await api.post("/security/fail2ban/ban", { ip, jail });
}

export async function unbanIP(ip: string, jail = "pinkpanel"): Promise<void> {
  await api.post("/security/fail2ban/unban", { ip, jail });
}
