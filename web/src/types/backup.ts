export interface Backup {
  id: number;
  domain_id: number | null;
  type: "full" | "domain";
  file_path: string;
  size_bytes: number;
  status: "pending" | "running" | "completed" | "failed";
  created_at: string;
  completed_at: string | null;
}

export interface CreateBackupRequest {
  type: "full" | "domain";
  domain_id?: number;
}
