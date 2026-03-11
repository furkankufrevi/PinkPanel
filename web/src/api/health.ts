import api from "./client";
import type { HealthResponse, DetailedHealthResponse } from "@/types/api";

export async function getHealth(): Promise<HealthResponse> {
  const response = await api.get<HealthResponse>("/health");
  return response.data;
}

export async function getDetailedHealth(): Promise<DetailedHealthResponse> {
  const response = await api.get<DetailedHealthResponse>("/health/detailed");
  return response.data;
}
