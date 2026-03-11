import api, { setTokens, clearTokens, getRefreshToken } from "./client";
import type {
  LoginRequest,
  TokenPair,
  SetupRequest,
  SetupResponse,
  SetupStatus,
} from "@/types/api";

export async function login(data: LoginRequest): Promise<TokenPair> {
  const response = await api.post<TokenPair>("/auth/login", data);
  setTokens(response.data);
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
