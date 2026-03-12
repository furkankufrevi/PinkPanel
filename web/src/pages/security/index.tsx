import { useState } from "react";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Badge } from "@/components/ui/badge";
import { Skeleton } from "@/components/ui/skeleton";
import { toast } from "sonner";
import {
  getFail2banStatus,
  getFail2banJailStatus,
  banIP,
  unbanIP,
} from "@/api/security";
import type { Fail2banJailStatus } from "@/api/security";
import type { AxiosError } from "axios";
import type { APIError } from "@/types/api";
import {
  Shield,
  ShieldAlert,
  ShieldCheck,
  Ban,
  Unlock,
  Plus,
  AlertTriangle,
} from "lucide-react";

export function SecurityPage() {
  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold">Security</h1>
        <p className="text-muted-foreground">
          Fail2ban intrusion prevention and IP management
        </p>
      </div>
      <Fail2banOverview />
      <div className="grid gap-6 lg:grid-cols-2">
        <JailCard jail="pinkpanel" title="PinkPanel" description="Panel login brute-force protection" />
        <JailCard jail="sshd" title="SSH" description="SSH brute-force protection" />
      </div>
    </div>
  );
}

function Fail2banOverview() {
  const { data, isLoading, error } = useQuery({
    queryKey: ["fail2ban-status"],
    queryFn: getFail2banStatus,
    retry: false,
  });

  if (isLoading) return <Skeleton className="h-24 w-full" />;

  if (error) {
    return (
      <Card>
        <CardContent className="py-8 text-center">
          <AlertTriangle className="h-10 w-10 mx-auto text-yellow-500 mb-3" />
          <h3 className="font-medium">Fail2ban Not Available</h3>
          <p className="text-sm text-muted-foreground mt-1">
            Fail2ban is not installed or the agent cannot reach it.
          </p>
        </CardContent>
      </Card>
    );
  }

  const jails = data?.jails ?? [];

  return (
    <Card>
      <CardHeader className="pb-3">
        <CardTitle className="flex items-center gap-2">
          <Shield className="h-5 w-5 text-pink-500" />
          Fail2ban Status
        </CardTitle>
        <CardDescription>
          Intrusion detection and prevention service
        </CardDescription>
      </CardHeader>
      <CardContent>
        <div className="flex items-center gap-3">
          <Badge className="bg-green-500/10 text-green-500 border-green-500/20">
            <ShieldCheck className="h-3 w-3 mr-1" />
            Active
          </Badge>
          <span className="text-sm text-muted-foreground">
            {jails.length} jail{jails.length !== 1 ? "s" : ""} configured: {jails.join(", ")}
          </span>
        </div>
      </CardContent>
    </Card>
  );
}

function JailCard({ jail, title, description }: { jail: string; title: string; description: string }) {
  const queryClient = useQueryClient();
  const [banInput, setBanInput] = useState("");

  const { data, isLoading, error } = useQuery({
    queryKey: ["fail2ban-jail", jail],
    queryFn: () => getFail2banJailStatus(jail),
    refetchInterval: 15000,
    retry: false,
  });

  const banMut = useMutation({
    mutationFn: (ip: string) => banIP(ip, jail),
    onSuccess: () => {
      toast.success("IP banned");
      setBanInput("");
      queryClient.invalidateQueries({ queryKey: ["fail2ban-jail", jail] });
    },
    onError: (err: AxiosError<APIError>) => {
      toast.error(err.response?.data?.error?.message ?? "Failed to ban IP");
    },
  });

  const unbanMut = useMutation({
    mutationFn: (ip: string) => unbanIP(ip, jail),
    onSuccess: () => {
      toast.success("IP unbanned");
      queryClient.invalidateQueries({ queryKey: ["fail2ban-jail", jail] });
    },
    onError: (err: AxiosError<APIError>) => {
      toast.error(err.response?.data?.error?.message ?? "Failed to unban IP");
    },
  });

  if (error) {
    return (
      <Card>
        <CardHeader>
          <CardTitle className="text-sm flex items-center gap-2">
            <ShieldAlert className="h-4 w-4 text-yellow-500" />
            {title}
          </CardTitle>
          <CardDescription>{description}</CardDescription>
        </CardHeader>
        <CardContent>
          <p className="text-sm text-muted-foreground">Jail not active or not configured</p>
        </CardContent>
      </Card>
    );
  }

  if (isLoading) return <Skeleton className="h-48 w-full" />;

  const jailData = data as Fail2banJailStatus;
  const bannedIPs = jailData.banned_ips ?? [];

  return (
    <Card>
      <CardHeader className="pb-3">
        <CardTitle className="text-sm flex items-center gap-2">
          <Shield className="h-4 w-4 text-pink-500" />
          {title} Jail
        </CardTitle>
        <CardDescription>{description}</CardDescription>
      </CardHeader>
      <CardContent className="space-y-4">
        {/* Stats */}
        <div className="grid grid-cols-2 gap-3">
          <div className="p-2 rounded border text-center">
            <p className="text-2xl font-bold">{jailData.currently_banned ?? 0}</p>
            <p className="text-xs text-muted-foreground">Banned</p>
          </div>
          <div className="p-2 rounded border text-center">
            <p className="text-2xl font-bold">{jailData.total_failed ?? 0}</p>
            <p className="text-xs text-muted-foreground">Total Failed</p>
          </div>
        </div>

        {/* Ban IP Form */}
        <div className="flex gap-2">
          <Input
            placeholder="IP address to ban"
            value={banInput}
            onChange={(e) => setBanInput(e.target.value)}
            className="text-sm"
            onKeyDown={(e) => {
              if (e.key === "Enter" && banInput) banMut.mutate(banInput);
            }}
          />
          <Button
            size="sm"
            variant="destructive"
            disabled={!banInput || banMut.isPending}
            onClick={() => banMut.mutate(banInput)}
          >
            <Plus className="h-3 w-3 mr-1" />
            Ban
          </Button>
        </div>

        {/* Banned IPs List */}
        {bannedIPs.length > 0 ? (
          <div className="space-y-1">
            <p className="text-xs text-muted-foreground font-medium">Banned IPs</p>
            {bannedIPs.map((ip) => (
              <div
                key={ip}
                className="flex items-center justify-between p-2 rounded border text-sm"
              >
                <span className="flex items-center gap-2">
                  <Ban className="h-3 w-3 text-red-500" />
                  <code className="text-xs">{ip}</code>
                </span>
                <Button
                  size="sm"
                  variant="ghost"
                  className="h-6 text-xs"
                  disabled={unbanMut.isPending}
                  onClick={() => unbanMut.mutate(ip)}
                >
                  <Unlock className="h-3 w-3 mr-1" />
                  Unban
                </Button>
              </div>
            ))}
          </div>
        ) : (
          <p className="text-xs text-muted-foreground">No IPs currently banned</p>
        )}
      </CardContent>
    </Card>
  );
}
