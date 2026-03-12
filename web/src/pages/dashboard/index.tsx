import {
  Activity,
  Cpu,
  Database,
  HardDrive,
  MemoryStick,
  Server,
  Wifi,
  WifiOff,
} from "lucide-react";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { useDetailedHealth } from "@/hooks/useHealth";
import { useSystemMetrics } from "@/hooks/useSystemMetrics";

export function DashboardPage() {
  const { data: health, isLoading, error } = useDetailedHealth();
  const { metrics, connected } = useSystemMetrics();

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

      {/* Real-Time Metrics */}
      {metrics && (
        <div className="grid gap-4 md:grid-cols-3">
          <MetricCard
            title="CPU Usage"
            icon={<Cpu className="h-4 w-4" />}
            value={cpuUsage}
            label={`${cpuUsage.toFixed(1)}%`}
            detail={`Load: ${metrics.load_avg}`}
          />
          <MetricCard
            title="Memory"
            icon={<MemoryStick className="h-4 w-4" />}
            value={ramPercent}
            label={`${ramPercent.toFixed(1)}%`}
            detail={`${formatBytes(ramUsed)} / ${formatBytes(ramTotal)}`}
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
}: {
  title: string;
  icon: React.ReactNode;
  value: number;
  label: string;
  detail: string;
}) {
  const color =
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
            className={`h-2 rounded-full transition-all duration-500 ${color}`}
            style={{ width: `${Math.min(value, 100)}%` }}
          />
        </div>
        <p className="text-xs text-muted-foreground">{detail}</p>
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
