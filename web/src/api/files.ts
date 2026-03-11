import api from "@/api/client";
import type { FileListResponse } from "@/types/files";

export async function listFiles(domainId: number, path?: string): Promise<FileListResponse> {
  const params = path ? { path } : {};
  const { data } = await api.get(`/domains/${domainId}/files`, { params });
  return data;
}

export async function readFile(domainId: number, path: string): Promise<{ content: string }> {
  const { data } = await api.get(`/domains/${domainId}/files/read`, { params: { path } });
  return data;
}

export async function saveFile(
  domainId: number,
  path: string,
  content: string
): Promise<void> {
  await api.post(`/domains/${domainId}/files/save`, { path, content });
}

export async function deleteFile(
  domainId: number,
  path: string,
  recursive: boolean = false
): Promise<void> {
  await api.post(`/domains/${domainId}/files/delete`, { path, recursive });
}

export async function renameFile(
  domainId: number,
  oldPath: string,
  newPath: string
): Promise<void> {
  await api.post(`/domains/${domainId}/files/rename`, { old_path: oldPath, new_path: newPath });
}

export async function createDirectory(domainId: number, path: string): Promise<void> {
  await api.post(`/domains/${domainId}/files/mkdir`, { path });
}

export async function extractArchive(
  domainId: number,
  archive: string,
  dest: string
): Promise<void> {
  await api.post(`/domains/${domainId}/files/extract`, { archive, dest });
}
