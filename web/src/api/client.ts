import axios from "axios";
import type { TokenPair } from "@/types/api";

const api = axios.create({
  baseURL: "/api",
  headers: {
    "Content-Type": "application/json",
  },
});

const TOKEN_KEY = "pinkpanel_tokens";

// Load tokens from localStorage on init
function loadTokens(): { access: string | null; refresh: string | null } {
  try {
    const raw = localStorage.getItem(TOKEN_KEY);
    if (raw) {
      const parsed = JSON.parse(raw);
      return { access: parsed.access_token ?? null, refresh: parsed.refresh_token ?? null };
    }
  } catch {
    // corrupted — clear it
    localStorage.removeItem(TOKEN_KEY);
  }
  return { access: null, refresh: null };
}

let { access: accessToken, refresh: refreshToken } = loadTokens();

export function setTokens(tokens: TokenPair) {
  accessToken = tokens.access_token;
  refreshToken = tokens.refresh_token;
  localStorage.setItem(TOKEN_KEY, JSON.stringify(tokens));
}

export function clearTokens() {
  accessToken = null;
  refreshToken = null;
  localStorage.removeItem(TOKEN_KEY);
}

export function getAccessToken() {
  return accessToken;
}

export function getRefreshToken() {
  return refreshToken;
}

export function hasTokens() {
  return !!accessToken && !!refreshToken;
}

// Request interceptor — attach access token
api.interceptors.request.use((config) => {
  if (accessToken) {
    config.headers.Authorization = `Bearer ${accessToken}`;
  }
  return config;
});

// Response interceptor — auto-refresh on 401
let isRefreshing = false;
let failedQueue: Array<{
  resolve: (value: unknown) => void;
  reject: (reason: unknown) => void;
}> = [];

function processQueue(error: unknown) {
  failedQueue.forEach((prom) => {
    if (error) {
      prom.reject(error);
    } else {
      prom.resolve(undefined);
    }
  });
  failedQueue = [];
}

api.interceptors.response.use(
  (response) => response,
  async (error) => {
    const originalRequest = error.config;

    if (error.response?.status !== 401 || originalRequest._retry) {
      return Promise.reject(error);
    }

    // Don't retry auth endpoints
    if (
      originalRequest.url?.includes("/auth/login") ||
      originalRequest.url?.includes("/auth/refresh")
    ) {
      return Promise.reject(error);
    }

    if (isRefreshing) {
      return new Promise((resolve, reject) => {
        failedQueue.push({ resolve, reject });
      }).then(() => {
        originalRequest.headers.Authorization = `Bearer ${accessToken}`;
        return api(originalRequest);
      });
    }

    originalRequest._retry = true;
    isRefreshing = true;

    try {
      if (!refreshToken) {
        throw new Error("No refresh token");
      }

      const { data } = await axios.post<TokenPair>("/api/auth/refresh", {
        refresh_token: refreshToken,
      });

      setTokens(data);
      // Store role if returned by refresh endpoint
      if (data.role) {
        localStorage.setItem("pinkpanel_role", data.role);
      }
      processQueue(null);

      originalRequest.headers.Authorization = `Bearer ${data.access_token}`;
      return api(originalRequest);
    } catch (refreshError) {
      processQueue(refreshError);
      clearTokens();
      window.location.href = "/login";
      return Promise.reject(refreshError);
    } finally {
      isRefreshing = false;
    }
  }
);

export default api;
