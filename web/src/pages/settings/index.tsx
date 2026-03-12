import { useState } from "react";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
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
import { getActivityLog, getServerInfo, getSessions, revokeSession } from "@/api/settings";
import { get2FAStatus, setup2FA, enable2FA, disable2FA, regenerateRecoveryCodes } from "@/api/auth";
import type { ActivityEntry, SessionEntry } from "@/api/settings";
import {
  Server,
  Activity,
  Clock,
  Cpu,
  HardDrive,
  MemoryStick,
  Monitor,
  ShieldCheck,
  X,
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
        <TwoFactorCard />
        <Separator />
        <ActiveSessionsCard />
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

function parseUserAgent(ua: string): string {
  if (!ua) return "Unknown";
  if (ua.includes("Firefox")) return "Firefox";
  if (ua.includes("Edg/")) return "Edge";
  if (ua.includes("Chrome")) return "Chrome";
  if (ua.includes("Safari")) return "Safari";
  return ua.substring(0, 40);
}

function ActiveSessionsCard() {
  const queryClient = useQueryClient();
  const { data, isLoading } = useQuery({
    queryKey: ["sessions"],
    queryFn: getSessions,
  });

  async function handleRevoke(id: number) {
    try {
      await revokeSession(id);
      toast.success("Session revoked");
      queryClient.invalidateQueries({ queryKey: ["sessions"] });
    } catch {
      toast.error("Failed to revoke session");
    }
  }

  return (
    <Card>
      <CardHeader>
        <CardTitle className="flex items-center gap-2">
          <Monitor className="h-5 w-5" />
          Active Sessions
        </CardTitle>
        <CardDescription>
          Devices currently logged into your account
        </CardDescription>
      </CardHeader>
      <CardContent>
        {isLoading ? (
          <Skeleton className="h-20 w-full" />
        ) : !data?.data || data.data.length === 0 ? (
          <p className="text-sm text-muted-foreground">No active sessions</p>
        ) : (
          <div className="space-y-2">
            {data.data.map((session: SessionEntry) => (
              <div
                key={session.id}
                className="flex items-center justify-between p-3 rounded border text-sm"
              >
                <div className="space-y-0.5">
                  <p className="font-medium flex items-center gap-2">
                    {parseUserAgent(session.user_agent)}
                    {session.current && (
                      <Badge variant="outline" className="text-xs text-green-600 border-green-600">
                        Current
                      </Badge>
                    )}
                  </p>
                  <p className="text-xs text-muted-foreground">
                    IP: {session.ip_address} — Logged in{" "}
                    {new Date(session.created_at).toLocaleString()}
                  </p>
                </div>
                {!session.current && (
                  <Button
                    variant="ghost"
                    size="sm"
                    onClick={() => handleRevoke(session.id)}
                    className="text-destructive hover:text-destructive"
                  >
                    <X className="h-4 w-4" />
                  </Button>
                )}
              </div>
            ))}
          </div>
        )}
      </CardContent>
    </Card>
  );
}

function TwoFactorCard() {
  const queryClient = useQueryClient();
  const [step, setStep] = useState<"idle" | "setup" | "verify" | "codes">("idle");
  const [qrCode, setQrCode] = useState("");
  const [secret, setSecret] = useState("");
  const [verifyCode, setVerifyCode] = useState("");
  const [recoveryCodes, setRecoveryCodes] = useState<string[]>([]);
  const [disablePassword, setDisablePassword] = useState("");

  const { data: status, isLoading } = useQuery({
    queryKey: ["2fa-status"],
    queryFn: get2FAStatus,
  });

  const setupMut = useMutation({
    mutationFn: setup2FA,
    onSuccess: (data) => {
      setQrCode(data.qr_code);
      setSecret(data.secret);
      setStep("verify");
    },
    onError: (err: any) => {
      toast.error(err.response?.data?.error?.message ?? "Failed to setup 2FA");
    },
  });

  const enableMut = useMutation({
    mutationFn: () => enable2FA(verifyCode),
    onSuccess: (data) => {
      setRecoveryCodes(data.recovery_codes);
      setStep("codes");
      queryClient.invalidateQueries({ queryKey: ["2fa-status"] });
      toast.success("Two-factor authentication enabled");
    },
    onError: (err: any) => {
      toast.error(err.response?.data?.error?.message ?? "Invalid code");
    },
  });

  const disableMut = useMutation({
    mutationFn: () => disable2FA(disablePassword),
    onSuccess: () => {
      setDisablePassword("");
      queryClient.invalidateQueries({ queryKey: ["2fa-status"] });
      toast.success("Two-factor authentication disabled");
    },
    onError: (err: any) => {
      toast.error(err.response?.data?.error?.message ?? "Failed to disable 2FA");
    },
  });

  const regenMut = useMutation({
    mutationFn: regenerateRecoveryCodes,
    onSuccess: (data) => {
      setRecoveryCodes(data.recovery_codes);
      setStep("codes");
      queryClient.invalidateQueries({ queryKey: ["2fa-status"] });
    },
    onError: (err: any) => {
      toast.error(err.response?.data?.error?.message ?? "Failed to regenerate codes");
    },
  });

  if (isLoading) return <Skeleton className="h-40 w-full" />;

  const is2FAEnabled = status?.enabled ?? false;

  return (
    <Card>
      <CardHeader>
        <CardTitle className="flex items-center gap-2">
          <ShieldCheck className="h-5 w-5" />
          Two-Factor Authentication
        </CardTitle>
        <CardDescription>
          {is2FAEnabled
            ? "Your account is protected with 2FA"
            : "Add an extra layer of security to your account"}
        </CardDescription>
      </CardHeader>
      <CardContent className="space-y-4">
        {/* Already enabled — show disable option */}
        {is2FAEnabled && step === "idle" && (
          <div className="space-y-4">
            <div className="flex items-center gap-2">
              <Badge className="bg-green-500/10 text-green-500 border-green-500/20">
                Enabled
              </Badge>
              {status && (
                <span className="text-xs text-muted-foreground">
                  {status.recovery_remaining} recovery codes remaining
                </span>
              )}
            </div>
            <div className="flex gap-2">
              <Button
                size="sm"
                variant="outline"
                onClick={() => regenMut.mutate()}
                disabled={regenMut.isPending}
              >
                {regenMut.isPending ? "Regenerating..." : "Regenerate Recovery Codes"}
              </Button>
            </div>
            <Separator />
            <div className="space-y-2">
              <Label className="text-sm text-destructive">Disable 2FA</Label>
              <div className="flex gap-2">
                <Input
                  type="password"
                  placeholder="Enter your password"
                  value={disablePassword}
                  onChange={(e) => setDisablePassword(e.target.value)}
                  className="max-w-xs"
                />
                <Button
                  variant="destructive"
                  size="sm"
                  disabled={!disablePassword || disableMut.isPending}
                  onClick={() => disableMut.mutate()}
                >
                  {disableMut.isPending ? "Disabling..." : "Disable"}
                </Button>
              </div>
            </div>
          </div>
        )}

        {/* Not enabled — show enable button */}
        {!is2FAEnabled && step === "idle" && (
          <Button
            className="bg-pink-500 hover:bg-pink-600"
            onClick={() => setupMut.mutate()}
            disabled={setupMut.isPending}
          >
            {setupMut.isPending ? "Setting up..." : "Enable Two-Factor Authentication"}
          </Button>
        )}

        {/* Step: QR code + verify */}
        {step === "verify" && (
          <div className="space-y-4">
            <p className="text-sm text-muted-foreground">
              Scan this QR code with your authenticator app (Google Authenticator, Authy, etc.)
            </p>
            <div className="flex justify-center">
              <img src={qrCode} alt="2FA QR Code" className="rounded border p-2 bg-white" />
            </div>
            <div className="space-y-1">
              <Label className="text-xs text-muted-foreground">Manual entry key</Label>
              <code className="block text-xs bg-muted px-2 py-1 rounded select-all">
                {secret}
              </code>
            </div>
            <form
              onSubmit={(e) => {
                e.preventDefault();
                enableMut.mutate();
              }}
              className="space-y-3"
            >
              <div className="space-y-2">
                <Label htmlFor="verify-2fa">Verification Code</Label>
                <Input
                  id="verify-2fa"
                  type="text"
                  value={verifyCode}
                  onChange={(e) => setVerifyCode(e.target.value)}
                  placeholder="000000"
                  autoComplete="one-time-code"
                  autoFocus
                  className="max-w-[200px] text-center text-lg tracking-widest"
                />
              </div>
              <div className="flex gap-2">
                <Button
                  type="submit"
                  className="bg-pink-500 hover:bg-pink-600"
                  disabled={!verifyCode || enableMut.isPending}
                >
                  {enableMut.isPending ? "Verifying..." : "Verify & Enable"}
                </Button>
                <Button
                  type="button"
                  variant="outline"
                  onClick={() => {
                    setStep("idle");
                    setVerifyCode("");
                  }}
                >
                  Cancel
                </Button>
              </div>
            </form>
          </div>
        )}

        {/* Step: Show recovery codes */}
        {step === "codes" && recoveryCodes.length > 0 && (
          <div className="space-y-4">
            <p className="text-sm text-muted-foreground">
              Save these recovery codes in a safe place. Each code can only be used once.
            </p>
            <div className="grid grid-cols-2 gap-2 p-3 bg-muted rounded">
              {recoveryCodes.map((code) => (
                <code key={code} className="text-sm font-mono">
                  {code}
                </code>
              ))}
            </div>
            <Button
              variant="outline"
              size="sm"
              onClick={() => {
                navigator.clipboard.writeText(recoveryCodes.join("\n"));
                toast.success("Recovery codes copied to clipboard");
              }}
            >
              Copy All
            </Button>
            <div>
              <Button
                size="sm"
                onClick={() => {
                  setStep("idle");
                  setRecoveryCodes([]);
                  setVerifyCode("");
                }}
              >
                Done
              </Button>
            </div>
          </div>
        )}
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
