import api from "./client";
import type {
  CronJob,
  CronLog,
  CreateCronJobRequest,
  UpdateCronJobRequest,
} from "@/types/cron";

export async function listCronJobs(domainId: number) {
  const { data } = await api.get<{ data: CronJob[] }>(
    `/domains/${domainId}/crons`
  );
  return data;
}

export async function getCronJob(id: number) {
  const { data } = await api.get<CronJob>(`/crons/${id}`);
  return data;
}

export async function createCronJob(
  domainId: number,
  req: CreateCronJobRequest
) {
  const { data } = await api.post<CronJob>(`/domains/${domainId}/crons`, req);
  return data;
}

export async function updateCronJob(id: number, req: UpdateCronJobRequest) {
  const { data } = await api.put<CronJob>(`/crons/${id}`, req);
  return data;
}

export async function deleteCronJob(id: number) {
  const { data } = await api.delete(`/crons/${id}`);
  return data;
}

export async function runCronJob(id: number) {
  const { data } = await api.post<{
    exit_code: number;
    output: string;
    duration_ms: number;
  }>(`/crons/${id}/run`);
  return data;
}

export async function getCronLogs(id: number, limit?: number) {
  const params = limit ? { limit } : {};
  const { data } = await api.get<{ data: CronLog[] }>(`/crons/${id}/logs`, {
    params,
  });
  return data;
}
