import { useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Badge } from "@/components/ui/badge";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { Separator } from "@/components/ui/separator";
import { Skeleton } from "@/components/ui/skeleton";
import { Switch } from "@/components/ui/switch";
import { useUIStore } from "@/stores/ui";
import { toast } from "sonner";
import api from "@/api/client";
import { getActivityLog, getServerInfo } from "@/api/settings";
import type { ActivityEntry } from "@/api/settings";
import {
  Server,
  Activity,
  Clock,
  Cpu,
  HardDrive,
  MemoryStick,
} from "lucide-react";

export function SettingsPage() {
  return (
    <div className="space-y-6">
      <h1 className="text-2xl font-bold">Settings</h1>
      <div className="grid gap-6 max-w-3xl">
        <ServerInfoCard />
        <Separator />
        <AppearanceSettings />
        <Separator />
        <ChangePasswordCard />
        <Separator />
        <ActivityLogCard />
      </div>
    </div>
  );
}

function ServerInfoCard() {
  const { data, isLoading } = useQuery({
    queryKey: ["server-info"],
    queryFn: getServerInfo,
    retry: false,
  });

  if (isLoading) {
    return <Skeleton className="h-40 w-full" />;
  }

  if (!data) {
    return (
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <Server className="h-5 w-5" />
            Server Information
          </CardTitle>
          <CardDescription>
            Unable to fetch server info — check agent connection
          </CardDescription>
        </CardHeader>
      </Card>
    );
  }

  const sys = data.system;

  return (
    <Card>
      <CardHeader>
        <CardTitle className="flex items-center gap-2">
          <Server className="h-5 w-5" />
          Server Information
        </CardTitle>
        <CardDescription>
          PinkPanel {data.panel_version} — {sys.hostname}
        </CardDescription>
      </CardHeader>
      <CardContent>
        <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
          <div className="space-y-1">
            <p className="text-xs text-muted-foreground flex items-center gap-1">
              <Cpu className="h-3 w-3" />
              CPU
            </p>
            <p className="text-sm font-medium">{(sys.cpu_usage ?? 0).toFixed(1)}%</p>
          </div>
          <div className="space-y-1">
            <p className="text-xs text-muted-foreground flex items-center gap-1">
              <MemoryStick className="h-3 w-3" />
              Memory
            </p>
            <p className="text-sm font-medium">
              {sys.ram?.total ? ((sys.ram.used / sys.ram.total) * 100).toFixed(1) : "0.0"}%
            </p>
          </div>
          {sys.disk?.[0] && (
            <div className="space-y-1">
              <p className="text-xs text-muted-foreground flex items-center gap-1">
                <HardDrive className="h-3 w-3" />
                Disk
              </p>
              <p className="text-sm font-medium">
                {sys.disk[0].use_percent ?? "0%"}
              </p>
            </div>
          )}
          <div className="space-y-1">
            <p className="text-xs text-muted-foreground flex items-center gap-1">
              <Clock className="h-3 w-3" />
              Uptime
            </p>
            <p className="text-sm font-medium">{sys.uptime}</p>
          </div>
        </div>
        <div className="mt-3 flex items-center gap-2 text-xs text-muted-foreground">
          <Badge variant="outline" className="text-xs">
            {sys.os}
          </Badge>
          <Badge variant="outline" className="text-xs">
            {sys.arch}
          </Badge>
          <span>Load: {sys.load_avg}</span>
        </div>
      </CardContent>
    </Card>
  );
}

function AppearanceSettings() {
  const theme = useUIStore((s) => s.theme);
  const toggleTheme = useUIStore((s) => s.toggleTheme);

  return (
    <Card>
      <CardHeader>
        <CardTitle>Appearance</CardTitle>
        <CardDescription>Customize the panel look and feel</CardDescription>
      </CardHeader>
      <CardContent>
        <div className="flex items-center justify-between">
          <div>
            <Label>Dark Mode</Label>
            <p className="text-sm text-muted-foreground">
              Toggle between light and dark theme
            </p>
          </div>
          <Switch checked={theme === "dark"} onCheckedChange={toggleTheme} />
        </div>
      </CardContent>
    </Card>
  );
}

function ChangePasswordCard() {
  const [currentPassword, setCurrentPassword] = useState("");
  const [newPassword, setNewPassword] = useState("");
  const [confirmPassword, setConfirmPassword] = useState("");
  const [loading, setLoading] = useState(false);

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault();

    if (!currentPassword || !newPassword || !confirmPassword) {
      toast.error("All fields are required");
      return;
    }
    if (newPassword.length < 8) {
      toast.error("New password must be at least 8 characters");
      return;
    }
    if (newPassword !== confirmPassword) {
      toast.error("Passwords do not match");
      return;
    }

    setLoading(true);
    try {
      await api.post("/auth/change-password", {
        current_password: currentPassword,
        new_password: newPassword,
      });
      toast.success("Password changed successfully");
      setCurrentPassword("");
      setNewPassword("");
      setConfirmPassword("");
    } catch (error: any) {
      const message =
        error.response?.data?.error?.message ?? "Failed to change password";
      toast.error(message);
    } finally {
      setLoading(false);
    }
  }

  return (
    <Card>
      <CardHeader>
        <CardTitle>Change Password</CardTitle>
        <CardDescription>
          Update your admin account password
        </CardDescription>
      </CardHeader>
      <CardContent>
        <form onSubmit={handleSubmit} className="space-y-4">
          <div className="space-y-2">
            <Label htmlFor="current-password">Current Password</Label>
            <Input
              id="current-password"
              type="password"
              value={currentPassword}
              onChange={(e) => setCurrentPassword(e.target.value)}
            />
          </div>
          <div className="space-y-2">
            <Label htmlFor="new-password">New Password</Label>
            <Input
              id="new-password"
              type="password"
              value={newPassword}
              onChange={(e) => setNewPassword(e.target.value)}
              placeholder="Min. 8 characters"
            />
          </div>
          <div className="space-y-2">
            <Label htmlFor="confirm-password">Confirm New Password</Label>
            <Input
              id="confirm-password"
              type="password"
              value={confirmPassword}
              onChange={(e) => setConfirmPassword(e.target.value)}
            />
          </div>
          <Button
            type="submit"
            disabled={loading}
            className="bg-pink-500 hover:bg-pink-600"
          >
            {loading ? "Changing..." : "Change Password"}
          </Button>
        </form>
      </CardContent>
    </Card>
  );
}

function formatAction(action: string): string {
  return action.replace(/_/g, " ").replace(/\b\w/g, (c) => c.toUpperCase());
}

function formatDate(dateStr: string): string {
  return new Date(dateStr).toLocaleString();
}

function ActivityLogCard() {
  const { data, isLoading } = useQuery({
    queryKey: ["activity-log"],
    queryFn: () => getActivityLog(50),
  });

  return (
    <Card>
      <CardHeader>
        <CardTitle className="flex items-center gap-2">
          <Activity className="h-5 w-5" />
          Activity Log
        </CardTitle>
        <CardDescription>Recent admin actions</CardDescription>
      </CardHeader>
      <CardContent>
        {isLoading ? (
          <Skeleton className="h-32 w-full" />
        ) : !data?.data || data.data.length === 0 ? (
          <p className="text-sm text-muted-foreground">No activity yet</p>
        ) : (
          <div className="space-y-2 max-h-[400px] overflow-auto">
            {data.data.map((entry: ActivityEntry) => (
              <div
                key={entry.id}
                className="flex items-start justify-between p-2 rounded border text-sm"
              >
                <div className="space-y-0.5">
                  <p className="font-medium">
                    {formatAction(entry.action)}
                    {entry.details && (
                      <span className="font-normal text-muted-foreground">
                        {" "}
                        — {entry.details}
                      </span>
                    )}
                  </p>
                  <p className="text-xs text-muted-foreground">
                    by {entry.username}
                    {entry.ip_address && ` from ${entry.ip_address}`}
                  </p>
                </div>
                <span className="text-xs text-muted-foreground whitespace-nowrap ml-4">
                  {formatDate(entry.created_at)}
                </span>
              </div>
            ))}
          </div>
        )}
      </CardContent>
    </Card>
  );
}
