import api, { setTokens, clearTokens, getRefreshToken } from "./client";
import type {
  LoginRequest,
  LoginResponse,
  TokenPair,
  SetupRequest,
  SetupResponse,
  SetupStatus,
  TwoFactorSetupResponse,
  TwoFactorEnableResponse,
  TwoFactorStatusResponse,
} from "@/types/api";

export async function login(data: LoginRequest): Promise<LoginResponse> {
  const response = await api.post<LoginResponse>("/auth/login", data);
  // Only set tokens if this is a full login (no 2FA required)
  if (response.data.access_token && !response.data.requires_2fa) {
    setTokens(response.data as TokenPair);
  }
  return response.data;
}

export async function verify2FA(data: {
  temp_token: string;
  code?: string;
  recovery_code?: string;
}): Promise<TokenPair> {
  const response = await api.post<TokenPair>("/auth/2fa/verify", data);
  setTokens(response.data);
  return response.data;
}

export async function get2FAStatus(): Promise<TwoFactorStatusResponse> {
  const response = await api.get<TwoFactorStatusResponse>("/auth/2fa/status");
  return response.data;
}

export async function setup2FA(): Promise<TwoFactorSetupResponse> {
  const response = await api.post<TwoFactorSetupResponse>("/auth/2fa/setup");
  return response.data;
}

export async function enable2FA(code: string): Promise<TwoFactorEnableResponse> {
  const response = await api.post<TwoFactorEnableResponse>("/auth/2fa/enable", { code });
  return response.data;
}

export async function disable2FA(password: string): Promise<void> {
  await api.post("/auth/2fa/disable", { password });
}

export async function regenerateRecoveryCodes(): Promise<{ recovery_codes: string[] }> {
  const response = await api.post<{ recovery_codes: string[] }>("/auth/2fa/recovery-codes/regenerate");
  return response.data;
}

export async function logout(): Promise<void> {
  const refreshToken = getRefreshToken();
  try {
    await api.post("/auth/logout", { refresh_token: refreshToken });
  } finally {
    clearTokens();
  }
}

export async function refreshTokens(): Promise<TokenPair> {
  const refreshToken = getRefreshToken();
  const response = await api.post<TokenPair>("/auth/refresh", {
    refresh_token: refreshToken,
  });
  setTokens(response.data);
  return response.data;
}

export async function getSetupStatus(): Promise<SetupStatus> {
  const response = await api.get<SetupStatus>("/setup/status");
  return response.data;
}

export async function setupAdmin(data: SetupRequest): Promise<SetupResponse> {
  const response = await api.post<SetupResponse>("/setup/admin", data);
  if (response.data.access_token) {
    setTokens(response.data);
  }
  return response.data;
}
