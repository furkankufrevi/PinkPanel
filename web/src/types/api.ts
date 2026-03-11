export interface TokenPair {
  access_token: string;
  refresh_token: string;
  expires_at: number;
}

export interface LoginRequest {
  username: string;
  password: string;
}

export interface SetupRequest {
  username: string;
  email: string;
  password: string;
}

export interface SetupStatus {
  setup_required: boolean;
}

export interface SetupResponse extends TokenPair {
  message: string;
}

export interface HealthResponse {
  status: string;
  components: {
    database: string;
    agent: string;
  };
}

export interface DetailedHealthResponse extends HealthResponse {
  version: string;
  uptime: string;
}

export interface APIError {
  error: {
    code: string;
    message: string;
    details?: Record<string, unknown>;
  };
}
