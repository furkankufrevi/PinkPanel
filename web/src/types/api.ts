export interface TokenPair {
  access_token: string;
  refresh_token: string;
  expires_at: number;
  role?: string;
}

export interface LoginRequest {
  username: string;
  password: string;
}

export interface LoginResponse {
  access_token?: string;
  refresh_token?: string;
  expires_at?: number;
  role?: string;
  requires_2fa?: boolean;
  temp_token?: string;
}

export interface TwoFactorSetupResponse {
  secret: string;
  qr_code: string;
  otpauth: string;
}

export interface TwoFactorEnableResponse {
  message: string;
  recovery_codes: string[];
}

export interface TwoFactorStatusResponse {
  enabled: boolean;
  recovery_remaining: number;
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
