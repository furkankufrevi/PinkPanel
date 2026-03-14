import api from "./client";

export interface SystemMetricPoint {
  id: number;
  cpu_usage: number;
  ram_used: number;
  ram_total: number;
  ram_percent: number;
  load_avg_1: number;
  load_avg_5: number;
  load_avg_15: number;
  collected_at: string;
}

export interface DomainMetricPoint {
  id: number;
  domain_id: number;
  disk_usage_bytes: number;
  bandwidth_bytes: number;
  collected_at: string;
}

export async function getSystemMetricsHistory(hours = 24) {
  const { data } = await api.get<{ data: SystemMetricPoint[] }>(
    "/metrics/system",
    { params: { hours } }
  );
  return data;
}

export async function getSystemMetricsCurrent() {
  const { data } = await api.get<SystemMetricPoint>("/metrics/system/current");
  return data;
}

export async function getDomainMetrics(domainId: number, hours = 168) {
  const { data } = await api.get<{
    data: {
      current: DomainMetricPoint | null;
      history: DomainMetricPoint[];
    };
  }>(`/domains/${domainId}/metrics`, { params: { hours } });
  return data;
}
