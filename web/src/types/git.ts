export interface GitRepository {
  id: number;
  domain_id: number;
  name: string;
  repo_type: "remote" | "local";
  remote_url: string | null;
  branch: string;
  deploy_mode: "automatic" | "manual" | "disabled";
  deploy_path: string;
  post_deploy_cmd: string | null;
  webhook_secret: string | null;
  last_deploy_at: string | null;
  last_commit: string | null;
  created_at: string;
  updated_at: string;
}

export interface GitDeployment {
  id: number;
  repo_id: number;
  commit_hash: string | null;
  branch: string | null;
  status: "pending" | "running" | "completed" | "failed";
  log: string | null;
  duration_ms: number | null;
  triggered_by: string;
  created_at: string;
}

export interface CreateGitRepoRequest {
  name: string;
  repo_type: "remote" | "local";
  remote_url?: string;
  branch?: string;
  deploy_path?: string;
  deploy_mode?: string;
}

export interface UpdateGitRepoRequest {
  branch?: string;
  deploy_mode?: string;
  deploy_path?: string;
  post_deploy_cmd?: string;
}
