import api from "@/api/client";
import type { FileListResponse, SearchResult } from "@/types/files";

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

export async function uploadFiles(
  domainId: number,
  destPath: string,
  files: File[],
  onProgress?: (percent: number) => void
): Promise<string[]> {
  const formData = new FormData();
  formData.append("path", destPath);
  for (const file of files) {
    formData.append("files", file);
  }
  const { data } = await api.post(`/domains/${domainId}/files/upload`, formData, {
    headers: { "Content-Type": "multipart/form-data" },
    onUploadProgress: (e) => {
      if (onProgress && e.total) {
        onProgress(Math.round((e.loaded * 100) / e.total));
      }
    },
  });
  return data.uploaded;
}

export async function downloadFile(domainId: number, path: string): Promise<void> {
  const { data } = await api.get(`/domains/${domainId}/files/download`, {
    params: { path },
    responseType: "blob",
  });
  const url = window.URL.createObjectURL(data);
  const a = document.createElement("a");
  a.href = url;
  a.download = path.split("/").pop() || "download";
  document.body.appendChild(a);
  a.click();
  a.remove();
  window.URL.revokeObjectURL(url);
}

export async function compressFiles(
  domainId: number,
  sources: string[],
  output: string,
  format: string
): Promise<void> {
  await api.post(`/domains/${domainId}/files/compress`, { sources, output, format });
}

export async function searchFiles(
  domainId: number,
  query: string,
  path?: string
): Promise<SearchResult[]> {
  const params: Record<string, string> = { q: query };
  if (path) params.path = path;
  const { data } = await api.get(`/domains/${domainId}/files/search`, { params });
  return data.data ?? [];
}

export function getDownloadUrl(domainId: number, path: string): string {
  return `/api/domains/${domainId}/files/download?path=${encodeURIComponent(path)}`;
}
