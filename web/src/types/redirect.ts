export interface Redirect {
  id: number;
  domain_id: number;
  source_path: string;
  target_url: string;
  redirect_type: number;
  enabled: boolean;
  created_at: string;
  updated_at: string;
}

export interface CreateRedirectRequest {
  source_path: string;
  target_url: string;
  redirect_type?: number;
}

export interface UpdateRedirectRequest {
  source_path?: string;
  target_url?: string;
  redirect_type?: number;
  enabled?: boolean;
}
