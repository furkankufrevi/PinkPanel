export interface CronJob {
  id: number;
  domain_id: number;
  schedule: string;
  command: string;
  description: string;
  enabled: boolean;
  created_at: string;
  updated_at: string;
}

export interface CronLog {
  id: number;
  cron_job_id: number;
  exit_code: number;
  output: string;
  duration_ms: number;
  started_at: string;
}

export interface CreateCronJobRequest {
  schedule: string;
  command: string;
  description?: string;
}

export interface UpdateCronJobRequest {
  schedule?: string;
  command?: string;
  description?: string;
  enabled?: boolean;
}
