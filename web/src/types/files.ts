export interface FileEntry {
  name: string;
  path: string;
  is_dir: boolean;
  size: number;
  permissions: string;
  owner: string;
  group: string;
  mod_time: string;
}

export interface FileListResponse {
  data: FileEntry[];
  path: string;
  base: string;
}
