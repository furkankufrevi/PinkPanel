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
  if (domainId === 0) return `/api/files/download?path=${encodeURIComponent(path)}`;
  return `/api/domains/${domainId}/files/download?path=${encodeURIComponent(path)}`;
}

// ── Routing helpers: domainId=0 routes to global API ──

export function routedListFiles(domainId: number, path?: string): Promise<FileListResponse> {
  return domainId === 0 ? globalListFiles(path) : listFiles(domainId, path);
}

export function routedReadFile(domainId: number, path: string): Promise<{ content: string }> {
  return domainId === 0 ? globalReadFile(path) : readFile(domainId, path);
}

export function routedSaveFile(domainId: number, path: string, content: string): Promise<void> {
  return domainId === 0 ? globalSaveFile(path, content) : saveFile(domainId, path, content);
}

export function routedDeleteFile(domainId: number, path: string, recursive = false): Promise<void> {
  return domainId === 0 ? globalDeleteFile(path, recursive) : deleteFile(domainId, path, recursive);
}

export function routedRenameFile(domainId: number, oldPath: string, newPath: string): Promise<void> {
  return domainId === 0 ? globalRenameFile(oldPath, newPath) : renameFile(domainId, oldPath, newPath);
}

export function routedCreateDirectory(domainId: number, path: string): Promise<void> {
  return domainId === 0 ? globalCreateDirectory(path) : createDirectory(domainId, path);
}

export function routedUploadFiles(domainId: number, destPath: string, files: File[], onProgress?: (percent: number) => void): Promise<string[]> {
  return domainId === 0 ? globalUploadFiles(destPath, files, onProgress) : uploadFiles(domainId, destPath, files, onProgress);
}

export function routedDownloadFile(domainId: number, path: string): Promise<void> {
  return domainId === 0 ? globalDownloadFile(path) : downloadFile(domainId, path);
}

export function routedCompressFiles(domainId: number, sources: string[], output: string, format: string): Promise<void> {
  return domainId === 0 ? globalCompressFiles(sources, output, format) : compressFiles(domainId, sources, output, format);
}

export function routedExtractArchive(domainId: number, archive: string, dest: string): Promise<void> {
  return domainId === 0 ? globalExtractArchive(archive, dest) : extractArchive(domainId, archive, dest);
}

export function routedSearchFiles(domainId: number, query: string, path?: string): Promise<SearchResult[]> {
  return domainId === 0 ? globalSearchFiles(query, path) : searchFiles(domainId, query, path);
}

// ── Global file manager API (all websites, rooted at /var/www) ──

export async function globalListFiles(path?: string): Promise<FileListResponse> {
  const params = path ? { path } : {};
  const { data } = await api.get("/files", { params });
  return data;
}

export async function globalReadFile(path: string): Promise<{ content: string }> {
  const { data } = await api.get("/files/read", { params: { path } });
  return data;
}

export async function globalSaveFile(path: string, content: string): Promise<void> {
  await api.post("/files/save", { path, content });
}

export async function globalDeleteFile(path: string, recursive = false): Promise<void> {
  await api.post("/files/delete", { path, recursive });
}

export async function globalRenameFile(oldPath: string, newPath: string): Promise<void> {
  await api.post("/files/rename", { old_path: oldPath, new_path: newPath });
}

export async function globalCreateDirectory(path: string): Promise<void> {
  await api.post("/files/mkdir", { path });
}

export async function globalExtractArchive(archive: string, dest: string): Promise<void> {
  await api.post("/files/extract", { archive, dest });
}

export async function globalUploadFiles(
  destPath: string,
  files: File[],
  onProgress?: (percent: number) => void
): Promise<string[]> {
  const formData = new FormData();
  formData.append("path", destPath);
  for (const file of files) {
    formData.append("files", file);
  }
  const { data } = await api.post("/files/upload", formData, {
    headers: { "Content-Type": "multipart/form-data" },
    onUploadProgress: (e) => {
      if (onProgress && e.total) {
        onProgress(Math.round((e.loaded * 100) / e.total));
      }
    },
  });
  return data.uploaded;
}

export async function globalDownloadFile(path: string): Promise<void> {
  const { data } = await api.get("/files/download", {
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

export async function globalCompressFiles(
  sources: string[],
  output: string,
  format: string
): Promise<void> {
  await api.post("/files/compress", { sources, output, format });
}

export async function globalSearchFiles(
  query: string,
  path?: string
): Promise<SearchResult[]> {
  const params: Record<string, string> = { q: query };
  if (path) params.path = path;
  const { data } = await api.get("/files/search", { params });
  return data.data ?? [];
}

export function globalGetDownloadUrl(path: string): string {
  return `/api/files/download?path=${encodeURIComponent(path)}`;
}
