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

export async function triggerUpgrade(): Promise<unknown> {
  const response = await api.post("/updates/upgrade");
  return response.data;
}

export async function getUpgradeHistory(): Promise<UpgradeHistoryResponse> {
  const response = await api.get<UpgradeHistoryResponse>("/updates/history");
  return response.data;
}
