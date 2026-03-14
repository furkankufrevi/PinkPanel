import {
  Activity,
  Cpu,
  Database,
  HardDrive,
  MemoryStick,
  Server,
  Wifi,
  WifiOff,
  TrendingUp,
} from "lucide-react";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { useDetailedHealth } from "@/hooks/useHealth";
import { useSystemMetrics } from "@/hooks/useSystemMetrics";
import { useQuery } from "@tanstack/react-query";
import { getSystemMetricsHistory } from "@/api/metrics";
import type { SystemMetricPoint } from "@/api/metrics";
import {
  AreaChart,
  Area,
  XAxis,
  YAxis,
  Tooltip,
  ResponsiveContainer,
} from "recharts";

export function DashboardPage() {
  const { data: health, isLoading, error } = useDetailedHealth();
  const { metrics, connected } = useSystemMetrics();

  const { data: historyData } = useQuery({
    queryKey: ["system-metrics-history"],
    queryFn: () => getSystemMetricsHistory(24),
    refetchInterval: 5 * 60 * 1000, // refresh every 5 minutes
  });
  const history: SystemMetricPoint[] = historyData?.data ?? [];

  if (isLoading) {
    return (
      <div className="space-y-6">
        <h1 className="text-2xl font-bold">Dashboard</h1>
        <p className="text-muted-foreground">Loading...</p>
      </div>
    );
  }

  if (error) {
    return (
      <div className="space-y-6">
        <h1 className="text-2xl font-bold">Dashboard</h1>
        <p className="text-destructive">Failed to load server status</p>
      </div>
    );
  }

  const cpuUsage = metrics?.cpu_usage ?? 0;
  const ramUsed = metrics?.ram?.used ?? 0;
  const ramTotal = metrics?.ram?.total ?? 1;
  const ramPercent = ramTotal > 0 ? (ramUsed / ramTotal) * 100 : 0;
  const diskPercent = metrics?.disk?.[0]?.use_percent
    ? parseFloat(metrics.disk[0].use_percent)
    : 0;

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <h1 className="text-2xl font-bold">Dashboard</h1>
        <div className="flex items-center gap-2">
          <Badge
            variant="outline"
            className={
              connected
                ? "bg-green-500/10 text-green-500 border-green-500/20"
                : "bg-muted text-muted-foreground"
            }
          >
            {connected ? (
              <Wifi className="h-3 w-3 mr-1" />
            ) : (
              <WifiOff className="h-3 w-3 mr-1" />
            )}
            {connected ? "Live" : "Connecting..."}
          </Badge>
          <Badge
            variant={health?.status === "ok" ? "default" : "destructive"}
            className={
              health?.status === "ok" ? "bg-green-500/10 text-green-500 border-green-500/20" : ""
            }
          >
            {health?.status === "ok" ? "All Systems OK" : "Degraded"}
          </Badge>
        </div>
      </div>

      {/* Status Cards */}
      <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-4">
        <StatusCard
          title="Panel"
          value={health?.status ?? "unknown"}
          icon={<Server className="h-4 w-4" />}
          detail={`Version ${health?.version ?? "?"}`}
        />
        <StatusCard
          title="Uptime"
          value={metrics?.uptime ?? health?.uptime ?? "?"}
          icon={<Activity className="h-4 w-4" />}
          detail={metrics?.hostname ?? "Since last restart"}
        />
        <StatusCard
          title="Database"
          value={health?.components.database ?? "unknown"}
          icon={<Database className="h-4 w-4" />}
          detail="SQLite"
        />
        <StatusCard
          title="Agent"
          value={health?.components.agent ?? "unknown"}
          icon={<HardDrive className="h-4 w-4" />}
          detail="System agent"
        />
      </div>

      {/* Real-Time Metrics with Sparklines */}
      {metrics && (
        <div className="grid gap-4 md:grid-cols-3">
          <MetricCard
            title="CPU Usage"
            icon={<Cpu className="h-4 w-4" />}
            value={cpuUsage}
            label={`${cpuUsage.toFixed(1)}%`}
            detail={`Load: ${metrics.load_avg}`}
            sparkline={metrics.cpu_history}
            color="#ec4899"
          />
          <MetricCard
            title="Memory"
            icon={<MemoryStick className="h-4 w-4" />}
            value={ramPercent}
            label={`${ramPercent.toFixed(1)}%`}
            detail={`${formatBytes(ramUsed)} / ${formatBytes(ramTotal)}`}
            sparkline={metrics.ram_history}
            color="#8b5cf6"
          />
          <MetricCard
            title="Disk"
            icon={<HardDrive className="h-4 w-4" />}
            value={diskPercent}
            label={metrics.disk?.[0]?.use_percent ?? "0%"}
            detail={
              metrics.disk?.[0]
                ? `${metrics.disk[0].used} / ${metrics.disk[0].total}`
                : "N/A"
            }
          />
        </div>
      )}

      {/* System Trends (24h) */}
      {history.length > 1 && (
        <div className="space-y-4">
          <div className="flex items-center gap-2">
            <TrendingUp className="h-4 w-4 text-muted-foreground" />
            <h2 className="text-sm font-medium text-muted-foreground">
              System Trends (24h)
            </h2>
          </div>
          <div className="grid gap-4 md:grid-cols-3">
            <TrendChart
              title="CPU Usage"
              data={history}
              dataKey="cpu_usage"
              color="#ec4899"
              unit="%"
            />
            <TrendChart
              title="Memory Usage"
              data={history.map((p) => ({
                ...p,
                ram_percent:
                  p.ram_total > 0
                    ? (p.ram_used / p.ram_total) * 100
                    : 0,
              }))}
              dataKey="ram_percent"
              color="#8b5cf6"
              unit="%"
            />
            <TrendChart
              title="Load Average"
              data={history}
              dataKey="load_avg_1"
              color="#06b6d4"
              unit=""
            />
          </div>
        </div>
      )}

      {/* System Info */}
      {metrics && (
        <div className="flex items-center gap-2 text-xs text-muted-foreground">
          <Badge variant="outline" className="text-xs">{metrics.os}</Badge>
          <Badge variant="outline" className="text-xs">{metrics.arch}</Badge>
          <span>{metrics.hostname}</span>
        </div>
      )}
    </div>
  );
}

function StatusCard({
  title,
  value,
  icon,
  detail,
}: {
  title: string;
  value: string;
  icon: React.ReactNode;
  detail: string;
}) {
  const isOk = value === "ok" || value === "active";

  return (
    <Card>
      <CardHeader className="flex flex-row items-center justify-between pb-2">
        <CardTitle className="text-sm font-medium text-muted-foreground">
          {title}
        </CardTitle>
        <span className="text-muted-foreground">{icon}</span>
      </CardHeader>
      <CardContent>
        <div
          className={`text-xl font-bold ${
            isOk ? "text-green-500" : value === "unknown" ? "text-muted-foreground" : "text-amber-500"
          }`}
        >
          {value}
        </div>
        <p className="text-xs text-muted-foreground mt-1">{detail}</p>
      </CardContent>
    </Card>
  );
}

function MetricCard({
  title,
  icon,
  value,
  label,
  detail,
  sparkline,
  color,
}: {
  title: string;
  icon: React.ReactNode;
  value: number;
  label: string;
  detail: string;
  sparkline?: number[];
  color?: string;
}) {
  const barColor =
    value > 90 ? "bg-red-500" : value > 70 ? "bg-amber-500" : "bg-green-500";

  return (
    <Card>
      <CardHeader className="flex flex-row items-center justify-between pb-2">
        <CardTitle className="text-sm font-medium text-muted-foreground">
          {title}
        </CardTitle>
        <span className="text-muted-foreground">{icon}</span>
      </CardHeader>
      <CardContent className="space-y-2">
        <div className="text-2xl font-bold">{label}</div>
        <div className="w-full bg-muted rounded-full h-2">
          <div
            className={`h-2 rounded-full transition-all duration-500 ${barColor}`}
            style={{ width: `${Math.min(value, 100)}%` }}
          />
        </div>
        {sparkline && sparkline.length > 1 && (
          <div className="h-10 w-full">
            <ResponsiveContainer width="100%" height="100%">
              <AreaChart data={sparkline.map((v, i) => ({ i, v }))}>
                <defs>
                  <linearGradient id={`spark-${title}`} x1="0" y1="0" x2="0" y2="1">
                    <stop offset="0%" stopColor={color || "#ec4899"} stopOpacity={0.3} />
                    <stop offset="100%" stopColor={color || "#ec4899"} stopOpacity={0} />
                  </linearGradient>
                </defs>
                <Area
                  type="monotone"
                  dataKey="v"
                  stroke={color || "#ec4899"}
                  strokeWidth={1.5}
                  fill={`url(#spark-${title})`}
                  dot={false}
                  isAnimationActive={false}
                />
              </AreaChart>
            </ResponsiveContainer>
          </div>
        )}
        <p className="text-xs text-muted-foreground">{detail}</p>
      </CardContent>
    </Card>
  );
}

function TrendChart({
  title,
  data,
  dataKey,
  color,
  unit,
}: {
  title: string;
  data: Record<string, any>[];
  dataKey: string;
  color: string;
  unit: string;
}) {
  return (
    <Card>
      <CardHeader className="pb-2">
        <CardTitle className="text-sm font-medium text-muted-foreground">
          {title}
        </CardTitle>
      </CardHeader>
      <CardContent>
        <div className="h-32">
          <ResponsiveContainer width="100%" height="100%">
            <AreaChart data={data}>
              <defs>
                <linearGradient id={`trend-${dataKey}`} x1="0" y1="0" x2="0" y2="1">
                  <stop offset="0%" stopColor={color} stopOpacity={0.3} />
                  <stop offset="100%" stopColor={color} stopOpacity={0} />
                </linearGradient>
              </defs>
              <XAxis
                dataKey="collected_at"
                tick={false}
                axisLine={false}
                tickLine={false}
              />
              <YAxis
                hide
                domain={dataKey.includes("percent") || dataKey === "cpu_usage" ? [0, 100] : ["auto", "auto"]}
              />
              <Tooltip
                contentStyle={{
                  background: "hsl(var(--card))",
                  border: "1px solid hsl(var(--border))",
                  borderRadius: "6px",
                  fontSize: "12px",
                }}
                labelFormatter={(label) => {
                  try {
                    return new Date(label).toLocaleTimeString();
                  } catch {
                    return label;
                  }
                }}
                formatter={(value) => [
                  `${Number(value).toFixed(1)}${unit}`,
                  title,
                ]}
              />
              <Area
                type="monotone"
                dataKey={dataKey}
                stroke={color}
                strokeWidth={2}
                fill={`url(#trend-${dataKey})`}
                dot={false}
              />
            </AreaChart>
          </ResponsiveContainer>
        </div>
      </CardContent>
    </Card>
  );
}

function formatBytes(bytes: number): string {
  if (bytes === 0) return "0 B";
  const units = ["B", "KB", "MB", "GB", "TB"];
  const i = Math.floor(Math.log(bytes) / Math.log(1024));
  return `${(bytes / Math.pow(1024, i)).toFixed(i > 0 ? 1 : 0)} ${units[i]}`;
}
