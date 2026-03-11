import api from "./client";

export interface LogResponse {
  log_type: string;
  path: string;
  content: string;
}

export interface LogSource {
  key: string;
  label: string;
}

export async function getLogSources() {
  const { data } = await api.get<{ data: LogSource[] }>("/logs/sources");
  return data;
}

export async function getDomainLogs(
  domainId: number,
  type: string = "access",
  lines: number = 100,
  filter: string = ""
) {
  const { data } = await api.get<LogResponse>(
    `/domains/${domainId}/logs`,
    { params: { type, lines, filter } }
  );
  return data;
}

export async function getSystemLogs(
  type: string = "syslog",
  lines: number = 100,
  filter: string = ""
) {
  const { data } = await api.get<LogResponse>(
    "/logs/system",
    { params: { type, lines, filter } }
  );
  return data;
}
