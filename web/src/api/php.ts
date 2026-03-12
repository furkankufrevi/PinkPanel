import api from "@/api/client";

export interface PHPSettings {
  version: string;
  settings: Record<string, string>;
}

export async function getPHPVersions(): Promise<{ data: string[] }> {
  const { data } = await api.get("/php/versions");
  return data;
}

export async function getDomainPHP(domainId: number): Promise<PHPSettings> {
  const { data } = await api.get(`/domains/${domainId}/php`);
  return data;
}

export async function updateDomainPHP(
  domainId: number,
  req: { version: string; settings: Record<string, string> }
): Promise<PHPSettings> {
  const { data } = await api.put(`/domains/${domainId}/php`, req);
  return data;
}

export interface PHPInfo {
  version: string;
  info: { info: string; extensions: string };
}

export async function getPHPInfo(domainId: number): Promise<PHPInfo> {
  const { data } = await api.get(`/domains/${domainId}/php/info`);
  return data;
}
