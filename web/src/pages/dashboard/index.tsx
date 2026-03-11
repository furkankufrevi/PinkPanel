import {
  Activity,
  Database,
  HardDrive,
  Server,
} from "lucide-react";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { useDetailedHealth } from "@/hooks/useHealth";

export function DashboardPage() {
  const { data: health, isLoading, error } = useDetailedHealth();

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

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <h1 className="text-2xl font-bold">Dashboard</h1>
        <Badge
          variant={health?.status === "ok" ? "default" : "destructive"}
          className={
            health?.status === "ok" ? "bg-green-500/10 text-green-500 border-green-500/20" : ""
          }
        >
          {health?.status === "ok" ? "All Systems OK" : "Degraded"}
        </Badge>
      </div>

      <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-4">
        <StatusCard
          title="Panel"
          value={health?.status ?? "unknown"}
          icon={<Server className="h-4 w-4" />}
          detail={`Version ${health?.version ?? "?"}`}
        />
        <StatusCard
          title="Uptime"
          value={health?.uptime ?? "?"}
          icon={<Activity className="h-4 w-4" />}
          detail="Since last restart"
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
