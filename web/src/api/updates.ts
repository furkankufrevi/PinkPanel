import api from "./client";

export interface UpdateCheck {
  current_version: string;
  latest_version?: string;
  update_available: boolean;
  release_name?: string;
  release_notes?: string;
  release_url?: string;
  published_at?: string;
  error?: string;
}

export interface Release {
  version: string;
  name: string;
  notes: string;
  published_at: string;
  url: string;
  prerelease: boolean;
  is_current: boolean;
  is_newer: boolean;
}

export interface ReleasesResponse {
  current_version: string;
  releases: Release[];
}

export interface UpgradeHistoryEntry {
  id: number;
  version: string;
  previous_version: string | null;
  changelog: string | null;
  status: string;
  created_at: string;
}

export interface UpgradeHistoryResponse {
  current_version: string;
  history: UpgradeHistoryEntry[];
}

export async function checkForUpdates(): Promise<UpdateCheck> {
  const response = await api.get<UpdateCheck>("/updates/check");
  return response.data;
}

export async function getReleases(): Promise<ReleasesResponse> {
  const response = await api.get<ReleasesResponse>("/updates/releases");
  return response.data;
}

export interface UpgradeResult {
  status: string;
  log_file: string;
}

export interface UpgradeStatus {
  status: string;
  running: boolean;
  log: string;
  log_file: string;
  total_size: number;
}

export async function triggerUpgrade(): Promise<UpgradeResult> {
  const response = await api.post<UpgradeResult>("/updates/upgrade");
  return response.data;
}

export async function getUpgradeStatus(offset = 0): Promise<UpgradeStatus> {
  const response = await api.get<UpgradeStatus>("/updates/upgrade/status", {
    params: { offset },
  });
  return response.data;
}

export async function getUpgradeHistory(): Promise<UpgradeHistoryResponse> {
  const response = await api.get<UpgradeHistoryResponse>("/updates/history");
  return response.data;
}
