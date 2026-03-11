export interface FTPAccount {
  id: number;
  domain_id: number;
  username: string;
  home_dir: string;
  quota_mb: number;
  created_at: string;
  updated_at: string;
}

export interface CreateFTPAccountRequest {
  domain_id: number;
  username: string;
  password: string;
  home_dir?: string;
  quota_mb?: number;
}
