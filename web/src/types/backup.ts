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

export interface BackupSchedule {
  id: number;
  domain_id: number | null;
  frequency: "daily" | "weekly" | "monthly";
  time: string;
  retention_count: number;
  enabled: boolean;
  last_run: string | null;
  next_run: string | null;
  created_at: string;
  updated_at: string;
}

export interface CreateScheduleRequest {
  domain_id?: number | null;
  frequency: string;
  time: string;
  retention_count: number;
}

export interface UpdateScheduleRequest {
  frequency: string;
  time: string;
  retention_count: number;
  enabled: boolean;
}
