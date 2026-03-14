import api from "./client";
import type {
  AppDefinition,
  InstalledApp,
  InstallAppRequest,
  WPInfo,
} from "@/types/app";

export async function getAppCatalog() {
  const { data } = await api.get<{ data: AppDefinition[] }>("/apps/catalog");
  return data;
}

export async function listInstalledApps(domainId: number) {
  const { data } = await api.get<{ data: InstalledApp[] }>(
    `/domains/${domainId}/apps`
  );
  return data;
}

export async function installApp(domainId: number, req: InstallAppRequest) {
  const { data } = await api.post<InstalledApp>(
    `/domains/${domainId}/apps/install`,
    req
  );
  return data;
}

export async function getInstalledApp(id: number) {
  const { data } = await api.get<InstalledApp>(`/apps/${id}`);
  return data;
}

export async function uninstallApp(id: number, dropDb?: boolean) {
  const params = dropDb ? { drop_db: "true" } : {};
  const { data } = await api.delete(`/apps/${id}`, { params });
  return data;
}

export async function updateApp(id: number) {
  const { data } = await api.post(`/apps/${id}/update`);
  return data;
}

export async function getAppLogs(id: number) {
  const { data } = await api.get<{ log: string }>(`/apps/${id}/logs`);
  return data;
}

export async function getWPInfo(id: number) {
  const { data } = await api.get<WPInfo>(`/apps/${id}/wp/info`);
  return data;
}

export async function wpMaintenance(id: number, enable: boolean) {
  const { data } = await api.post(`/apps/${id}/wp/maintenance`, { enable });
  return data;
}
