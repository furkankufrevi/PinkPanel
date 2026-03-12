import { useState } from "react";
import { useParams } from "react-router-dom";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Label } from "@/components/ui/label";
import { Textarea } from "@/components/ui/textarea";
import { Switch } from "@/components/ui/switch";
import { Badge } from "@/components/ui/badge";
import { Skeleton } from "@/components/ui/skeleton";
import { ConfirmDialog } from "@/components/shared/confirm-dialog";
import { toast } from "sonner";
import {
  getSSLCertificate,
  installSSLCertificate,
  deleteSSLCertificate,
  toggleSSLAutoRenew,
  issueLetsEncrypt,
} from "@/api/ssl";
import type { AxiosError } from "axios";
import type { APIError } from "@/types/api";
import { ShieldCheck, ShieldX, Upload, Trash2, Zap, Loader2 } from "lucide-react";

export function DomainSSL() {
  const { id } = useParams<{ id: string }>();
  const domainId = Number(id);
  const queryClient = useQueryClient();

  const [showInstall, setShowInstall] = useState(false);
  const [showDelete, setShowDelete] = useState(false);
  const [certificate, setCertificate] = useState("");
  const [privateKey, setPrivateKey] = useState("");
  const [chain, setChain] = useState("");
  const [forceHttps, setForceHttps] = useState(true);

  const { data: ssl, isLoading } = useQuery({
    queryKey: ["ssl", domainId],
    queryFn: () => getSSLCertificate(domainId),
    enabled: !!domainId,
  });

  const installMutation = useMutation({
    mutationFn: () =>
      installSSLCertificate(domainId, {
        certificate,
        private_key: privateKey,
        chain: chain || undefined,
        force_https: forceHttps,
      }),
    onSuccess: () => {
      toast.success("SSL certificate installed");
      queryClient.invalidateQueries({ queryKey: ["ssl", domainId] });
      setShowInstall(false);
      setCertificate("");
      setPrivateKey("");
      setChain("");
    },
    onError: (err: AxiosError<APIError>) => {
      toast.error(
        err.response?.data?.error?.message ?? "Failed to install certificate"
      );
    },
  });

  const deleteMutation = useMutation({
    mutationFn: () => deleteSSLCertificate(domainId),
    onSuccess: () => {
      toast.success("SSL certificate removed");
      queryClient.invalidateQueries({ queryKey: ["ssl", domainId] });
      setShowDelete(false);
    },
    onError: (err: AxiosError<APIError>) => {
      toast.error(
        err.response?.data?.error?.message ?? "Failed to remove certificate"
      );
    },
  });

  const letsEncryptMutation = useMutation({
    mutationFn: () => issueLetsEncrypt(domainId, true),
    onSuccess: () => {
      toast.success("Let's Encrypt certificate issued successfully");
      queryClient.invalidateQueries({ queryKey: ["ssl", domainId] });
    },
    onError: (err: AxiosError<APIError>) => {
      toast.error(
        err.response?.data?.error?.message ?? "Failed to issue Let's Encrypt certificate"
      );
    },
  });

  const autoRenewMutation = useMutation({
    mutationFn: (enabled: boolean) => toggleSSLAutoRenew(domainId, enabled),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["ssl", domainId] });
    },
    onError: (err: AxiosError<APIError>) => {
      toast.error(
        err.response?.data?.error?.message ?? "Failed to update auto-renew"
      );
    },
  });

  if (isLoading) {
    return (
      <div className="space-y-4 max-w-2xl">
        <Skeleton className="h-48 w-full" />
      </div>
    );
  }

  const isInstalled = ssl?.installed;
  const isExpired =
    ssl?.expires_at && new Date(ssl.expires_at) < new Date();
  const daysUntilExpiry = ssl?.expires_at
    ? Math.ceil(
        (new Date(ssl.expires_at).getTime() - Date.now()) / (1000 * 60 * 60 * 24)
      )
    : null;

  return (
    <div className="space-y-6 max-w-2xl">
      {/* Status Card */}
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            {isInstalled ? (
              <ShieldCheck className="h-5 w-5 text-green-500" />
            ) : (
              <ShieldX className="h-5 w-5 text-muted-foreground" />
            )}
            SSL Certificate
          </CardTitle>
          <CardDescription>
            {isInstalled
              ? "An SSL certificate is installed for this domain"
              : "No SSL certificate is installed for this domain"}
          </CardDescription>
        </CardHeader>
        {isInstalled && ssl && (
          <CardContent className="space-y-3">
            <div className="grid grid-cols-2 gap-4 text-sm">
              <div>
                <span className="text-muted-foreground">Type</span>
                <div className="mt-1">
                  <Badge variant="outline">
                    {ssl.type === "letsencrypt" ? "Let's Encrypt" : "Custom"}
                  </Badge>
                </div>
              </div>
              <div>
                <span className="text-muted-foreground">Status</span>
                <div className="mt-1">
                  {isExpired ? (
                    <Badge variant="destructive">Expired</Badge>
                  ) : daysUntilExpiry !== null && daysUntilExpiry <= 30 ? (
                    <Badge className="bg-amber-500 text-white">
                      Expires in {daysUntilExpiry}d
                    </Badge>
                  ) : (
                    <Badge className="bg-green-500 text-white">Valid</Badge>
                  )}
                </div>
              </div>
              {ssl.issuer && (
                <div>
                  <span className="text-muted-foreground">Issuer</span>
                  <div className="mt-1 font-medium">{ssl.issuer}</div>
                </div>
              )}
              {ssl.expires_at && (
                <div>
                  <span className="text-muted-foreground">Expires</span>
                  <div className="mt-1 font-medium">
                    {new Date(ssl.expires_at).toLocaleDateString()}
                  </div>
                </div>
              )}
            </div>

            <div className="flex items-center justify-between pt-2 border-t">
              <div>
                <Label className="text-sm font-medium">Auto-Renew</Label>
                <p className="text-xs text-muted-foreground">
                  Automatically renew before expiration
                </p>
              </div>
              <Switch
                checked={ssl.auto_renew}
                onCheckedChange={(checked) => autoRenewMutation.mutate(checked)}
              />
            </div>

            <div className="flex gap-2 pt-2">
              <Button
                variant="outline"
                size="sm"
                onClick={() => setShowInstall(true)}
              >
                <Upload className="h-4 w-4 mr-1" />
                Replace
              </Button>
              <Button
                variant="destructive"
                size="sm"
                onClick={() => setShowDelete(true)}
              >
                <Trash2 className="h-4 w-4 mr-1" />
                Remove
              </Button>
            </div>
          </CardContent>
        )}
        {!isInstalled && (
          <CardContent className="flex gap-2">
            <Button
              onClick={() => letsEncryptMutation.mutate()}
              disabled={letsEncryptMutation.isPending}
              className="bg-pink-500 hover:bg-pink-600"
            >
              {letsEncryptMutation.isPending ? (
                <Loader2 className="h-4 w-4 mr-1 animate-spin" />
              ) : (
                <Zap className="h-4 w-4 mr-1" />
              )}
              {letsEncryptMutation.isPending ? "Issuing..." : "Issue Let's Encrypt"}
            </Button>
            <Button
              variant="outline"
              onClick={() => setShowInstall(true)}
            >
              <Upload className="h-4 w-4 mr-1" />
              Custom Certificate
            </Button>
          </CardContent>
        )}
      </Card>

      {/* Install Form */}
      {showInstall && (
        <Card>
          <CardHeader>
            <CardTitle>Install SSL Certificate</CardTitle>
            <CardDescription>
              Paste your certificate and private key in PEM format
            </CardDescription>
          </CardHeader>
          <CardContent className="space-y-4">
            <div className="space-y-2">
              <Label>Certificate (PEM)</Label>
              <Textarea
                value={certificate}
                onChange={(e) => setCertificate(e.target.value)}
                placeholder="-----BEGIN CERTIFICATE-----&#10;...&#10;-----END CERTIFICATE-----"
                rows={6}
                className="font-mono text-xs"
              />
            </div>
            <div className="space-y-2">
              <Label>Private Key (PEM)</Label>
              <Textarea
                value={privateKey}
                onChange={(e) => setPrivateKey(e.target.value)}
                placeholder="-----BEGIN PRIVATE KEY-----&#10;...&#10;-----END PRIVATE KEY-----"
                rows={6}
                className="font-mono text-xs"
              />
            </div>
            <div className="space-y-2">
              <Label>
                CA Chain (optional)
              </Label>
              <Textarea
                value={chain}
                onChange={(e) => setChain(e.target.value)}
                placeholder="-----BEGIN CERTIFICATE-----&#10;...&#10;-----END CERTIFICATE-----"
                rows={4}
                className="font-mono text-xs"
              />
            </div>
            <div className="flex items-center justify-between">
              <div>
                <Label className="text-sm font-medium">Force HTTPS</Label>
                <p className="text-xs text-muted-foreground">
                  Redirect all HTTP traffic to HTTPS
                </p>
              </div>
              <Switch checked={forceHttps} onCheckedChange={setForceHttps} />
            </div>
            <div className="flex gap-2">
              <Button
                onClick={() => installMutation.mutate()}
                disabled={
                  installMutation.isPending || !certificate || !privateKey
                }
                className="bg-pink-500 hover:bg-pink-600"
              >
                {installMutation.isPending ? "Installing..." : "Install Certificate"}
              </Button>
              <Button
                variant="outline"
                onClick={() => setShowInstall(false)}
              >
                Cancel
              </Button>
            </div>
          </CardContent>
        </Card>
      )}

      {/* Delete Confirmation */}
      <ConfirmDialog
        open={showDelete}
        onOpenChange={setShowDelete}
        title="Remove SSL Certificate"
        description="This will remove the SSL certificate and revert the domain to HTTP only. This action cannot be undone."
        confirmText="Remove Certificate"
        destructive
        loading={deleteMutation.isPending}
        onConfirm={() => deleteMutation.mutate()}
      />
    </div>
  );
}
