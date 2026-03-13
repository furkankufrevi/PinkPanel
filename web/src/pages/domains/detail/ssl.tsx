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
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { ConfirmDialog } from "@/components/shared/confirm-dialog";
import { toast } from "sonner";
import {
  getSSLCertificate,
  installSSLCertificate,
  deleteSSLCertificate,
  toggleSSLAutoRenew,
  toggleForceHTTPS,
  toggleHSTS,
  issueLetsEncrypt,
} from "@/api/ssl";
import type { AxiosError } from "axios";
import type { APIError } from "@/types/api";
import type { IssueLetsEncryptRequest, SecuredComponent } from "@/types/ssl";
import {
  ShieldCheck,
  ShieldX,
  Upload,
  Trash2,
  Zap,
  Loader2,
  RefreshCw,
  Info,
  CheckCircle2,
  XCircle,
} from "lucide-react";
import { useQuery as useDomainQuery } from "@tanstack/react-query";
import { getDomain } from "@/api/domains";

export function DomainSSL() {
  const { id } = useParams<{ id: string }>();
  const domainId = Number(id);
  const queryClient = useQueryClient();

  const [showInstall, setShowInstall] = useState(false);
  const [showDelete, setShowDelete] = useState(false);
  const [showIssueDialog, setShowIssueDialog] = useState(false);
  const [certificate, setCertificate] = useState("");
  const [privateKey, setPrivateKey] = useState("");
  const [chain, setChain] = useState("");
  const [forceHttps, setForceHttps] = useState(true);

  // Issue dialog state
  const [issueOpts, setIssueOpts] = useState<IssueLetsEncryptRequest>({
    secure_domain: true,
    secure_wildcard: false,
    include_www: true,
    secure_webmail: false,
    secure_mail: false,
    assign_to_mail: false,
  });

  const { data: domainData } = useDomainQuery({
    queryKey: ["domain", domainId],
    queryFn: () => getDomain(domainId),
    enabled: !!domainId,
  });
  const domainName = domainData?.name ?? "example.com";

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
    mutationFn: (req: IssueLetsEncryptRequest) =>
      issueLetsEncrypt(domainId, req),
    onSuccess: () => {
      toast.success("Let's Encrypt certificate issued successfully");
      queryClient.invalidateQueries({ queryKey: ["ssl", domainId] });
      setShowIssueDialog(false);
    },
    onError: (err: AxiosError<APIError>) => {
      toast.error(
        err.response?.data?.error?.message ??
          "Failed to issue Let's Encrypt certificate"
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

  const forceHttpsMutation = useMutation({
    mutationFn: (enabled: boolean) => toggleForceHTTPS(domainId, enabled),
    onSuccess: () => {
      toast.success("HTTPS redirect updated");
      queryClient.invalidateQueries({ queryKey: ["ssl", domainId] });
    },
    onError: (err: AxiosError<APIError>) => {
      toast.error(
        err.response?.data?.error?.message ?? "Failed to update HTTPS redirect"
      );
    },
  });

  const hstsMutation = useMutation({
    mutationFn: (enabled: boolean) => toggleHSTS(domainId, enabled),
    onSuccess: () => {
      toast.success("HSTS setting updated");
      queryClient.invalidateQueries({ queryKey: ["ssl", domainId] });
    },
    onError: (err: AxiosError<APIError>) => {
      toast.error(
        err.response?.data?.error?.message ?? "Failed to update HSTS"
      );
    },
  });

  if (isLoading) {
    return (
      <div className="space-y-4 max-w-4xl">
        <Skeleton className="h-48 w-full" />
      </div>
    );
  }

  const isInstalled = ssl?.installed;
  const isExpired =
    ssl?.expires_at && new Date(ssl.expires_at) < new Date();
  const daysUntilExpiry = ssl?.expires_at
    ? Math.ceil(
        (new Date(ssl.expires_at).getTime() - Date.now()) /
          (1000 * 60 * 60 * 24)
      )
    : null;

  // Render secured component row
  function ComponentRow({
    component,
  }: {
    component: SecuredComponent;
  }) {
    return (
      <div className="flex items-center justify-between py-2">
        <span className="text-sm font-medium">{component.name}</span>
        {component.secured ? (
          <Badge className="bg-green-500/10 text-green-600 border-green-200 gap-1">
            <CheckCircle2 className="h-3.5 w-3.5" />
            Secured
          </Badge>
        ) : (
          <Badge
            variant="outline"
            className="text-muted-foreground gap-1"
          >
            <XCircle className="h-3.5 w-3.5" />
            Not Secured
          </Badge>
        )}
      </div>
    );
  }

  return (
    <div className="space-y-6 max-w-4xl">
      {/* No cert installed — show options */}
      {!isInstalled && (
        <Card>
          <CardHeader>
            <CardTitle className="flex items-center gap-2">
              <ShieldX className="h-5 w-5 text-muted-foreground" />
              SSL/TLS Certificate
            </CardTitle>
            <CardDescription>
              No SSL certificate is installed for this domain. Choose an option
              below to secure it.
            </CardDescription>
          </CardHeader>
          <CardContent className="flex gap-3">
            <Button
              onClick={() => setShowIssueDialog(true)}
              className="bg-pink-500 hover:bg-pink-600"
            >
              <Zap className="h-4 w-4 mr-1" />
              Let's Encrypt
            </Button>
            <Button variant="outline" onClick={() => setShowInstall(true)}>
              <Upload className="h-4 w-4 mr-1" />
              Custom Certificate
            </Button>
          </CardContent>
        </Card>
      )}

      {/* Cert installed — two-column layout */}
      {isInstalled && ssl && (
        <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
          {/* Main column */}
          <div className="lg:col-span-2 space-y-6">
            {/* Certificate Info */}
            <Card>
              <CardHeader>
                <CardTitle className="flex items-center gap-2">
                  <ShieldCheck className="h-5 w-5 text-green-500" />
                  SSL/TLS Certificate
                </CardTitle>
              </CardHeader>
              <CardContent className="space-y-4">
                {/* Info bar */}
                <div className="flex flex-wrap items-center gap-3 text-sm">
                  <Badge variant="outline">
                    {ssl.type === "letsencrypt" ? "Let's Encrypt" : "Custom"}
                  </Badge>
                  {ssl.domains && (
                    <span className="text-muted-foreground">{ssl.domains}</span>
                  )}
                  <span className="text-muted-foreground">|</span>
                  {ssl.expires_at && (
                    <span className="text-muted-foreground">
                      Valid until{" "}
                      {new Date(ssl.expires_at).toLocaleDateString()}
                    </span>
                  )}
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

                {/* Secured Components */}
                {ssl.secured_components && (
                  <div className="border rounded-lg p-4">
                    <h4 className="text-sm font-semibold mb-3">
                      Secured Components
                    </h4>
                    <div className="divide-y">
                      <ComponentRow
                        component={ssl.secured_components.domain}
                      />
                      <ComponentRow component={ssl.secured_components.www} />
                      <ComponentRow component={ssl.secured_components.mail} />
                      <ComponentRow
                        component={ssl.secured_components.webmail}
                      />
                      <ComponentRow
                        component={ssl.secured_components.wildcard}
                      />
                      <ComponentRow
                        component={ssl.secured_components.mail_services}
                      />
                    </div>
                  </div>
                )}

                {/* Actions */}
                <div className="flex gap-2 pt-2">
                  <Button
                    variant="outline"
                    size="sm"
                    onClick={() => setShowIssueDialog(true)}
                  >
                    <RefreshCw className="h-4 w-4 mr-1" />
                    Reissue Certificate
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
            </Card>
          </div>

          {/* Side column — Options */}
          <div className="space-y-6">
            <Card>
              <CardHeader>
                <CardTitle className="text-base">Options</CardTitle>
              </CardHeader>
              <CardContent className="space-y-4">
                <div className="flex items-center justify-between">
                  <div>
                    <Label className="text-sm font-medium">Force HTTPS</Label>
                    <p className="text-xs text-muted-foreground">
                      Redirect HTTP to HTTPS
                    </p>
                  </div>
                  <Switch
                    checked={ssl.force_https}
                    onCheckedChange={(checked) =>
                      forceHttpsMutation.mutate(checked)
                    }
                  />
                </div>

                <div className="flex items-center justify-between">
                  <div>
                    <Label className="text-sm font-medium">HSTS</Label>
                    <p className="text-xs text-muted-foreground">
                      Strict Transport Security
                    </p>
                  </div>
                  <Switch
                    checked={ssl.hsts}
                    onCheckedChange={(checked) => hstsMutation.mutate(checked)}
                  />
                </div>

                <div className="flex items-center justify-between">
                  <div>
                    <Label className="text-sm font-medium">Auto-Renew</Label>
                    <p className="text-xs text-muted-foreground">
                      Renew before expiration
                    </p>
                  </div>
                  <Switch
                    checked={ssl.auto_renew}
                    onCheckedChange={(checked) =>
                      autoRenewMutation.mutate(checked)
                    }
                  />
                </div>
              </CardContent>
            </Card>
          </div>
        </div>
      )}

      {/* Issue Let's Encrypt Dialog */}
      <Dialog
        open={showIssueDialog}
        onOpenChange={setShowIssueDialog}
      >
        <DialogContent className="sm:max-w-md">
          <DialogHeader>
            <DialogTitle>Issue Let's Encrypt Certificate</DialogTitle>
            <DialogDescription>
              Select which components to secure with SSL/TLS.
            </DialogDescription>
          </DialogHeader>

          <div className="space-y-3 py-2">
            {/* Secure domain — always checked, disabled */}
            <label className="flex items-start gap-3 p-3 rounded-lg border bg-muted/30">
              <input
                type="checkbox"
                checked={issueOpts.secure_domain}
                disabled
                className="mt-0.5 h-4 w-4 rounded border-input accent-pink-500"
              />
              <div className="flex-1 min-w-0">
                <div className="text-sm font-medium">Secure the domain</div>
                <div className="text-xs text-muted-foreground">
                  {domainName}
                </div>
              </div>
            </label>

            {/* Include www */}
            <label className="flex items-start gap-3 p-3 rounded-lg border cursor-pointer hover:bg-muted/30 transition-colors">
              <input
                type="checkbox"
                checked={issueOpts.include_www}
                onChange={(e) =>
                  setIssueOpts((o) => ({ ...o, include_www: e.target.checked }))
                }
                className="mt-0.5 h-4 w-4 rounded border-input accent-pink-500"
              />
              <div className="flex-1 min-w-0">
                <div className="text-sm font-medium">Include www</div>
                <div className="text-xs text-muted-foreground">
                  www.{domainName}
                </div>
              </div>
            </label>

            {/* Secure wildcard */}
            <label className="flex items-start gap-3 p-3 rounded-lg border cursor-pointer hover:bg-muted/30 transition-colors">
              <input
                type="checkbox"
                checked={issueOpts.secure_wildcard}
                onChange={(e) =>
                  setIssueOpts((o) => ({
                    ...o,
                    secure_wildcard: e.target.checked,
                  }))
                }
                className="mt-0.5 h-4 w-4 rounded border-input accent-pink-500"
              />
              <div className="flex-1 min-w-0">
                <div className="text-sm font-medium flex items-center gap-1.5">
                  Secure wildcard
                  <span
                    title="Uses DNS-01 challenge via your BIND server. Covers all subdomains."
                    className="cursor-help"
                  >
                    <Info className="h-3.5 w-3.5 text-muted-foreground" />
                  </span>
                </div>
                <div className="text-xs text-muted-foreground">
                  *.{domainName}
                </div>
              </div>
            </label>

            {/* Secure mail */}
            <label className="flex items-start gap-3 p-3 rounded-lg border cursor-pointer hover:bg-muted/30 transition-colors">
              <input
                type="checkbox"
                checked={issueOpts.secure_mail}
                onChange={(e) =>
                  setIssueOpts((o) => ({
                    ...o,
                    secure_mail: e.target.checked,
                  }))
                }
                className="mt-0.5 h-4 w-4 rounded border-input accent-pink-500"
              />
              <div className="flex-1 min-w-0">
                <div className="text-sm font-medium">Secure mail</div>
                <div className="text-xs text-muted-foreground">
                  mail.{domainName}
                </div>
              </div>
            </label>

            {/* Secure webmail */}
            <label className="flex items-start gap-3 p-3 rounded-lg border cursor-pointer hover:bg-muted/30 transition-colors">
              <input
                type="checkbox"
                checked={issueOpts.secure_webmail}
                onChange={(e) =>
                  setIssueOpts((o) => ({
                    ...o,
                    secure_webmail: e.target.checked,
                  }))
                }
                className="mt-0.5 h-4 w-4 rounded border-input accent-pink-500"
              />
              <div className="flex-1 min-w-0">
                <div className="text-sm font-medium">Secure webmail</div>
                <div className="text-xs text-muted-foreground">
                  webmail.{domainName}
                </div>
              </div>
            </label>

            {/* Assign to mail services */}
            <label className="flex items-start gap-3 p-3 rounded-lg border cursor-pointer hover:bg-muted/30 transition-colors">
              <input
                type="checkbox"
                checked={issueOpts.assign_to_mail}
                onChange={(e) =>
                  setIssueOpts((o) => ({
                    ...o,
                    assign_to_mail: e.target.checked,
                  }))
                }
                className="mt-0.5 h-4 w-4 rounded border-input accent-pink-500"
              />
              <div className="flex-1 min-w-0">
                <div className="text-sm font-medium">
                  Assign to mail services
                </div>
                <div className="text-xs text-muted-foreground">
                  Configure Postfix/Dovecot (IMAP, POP, SMTP)
                </div>
              </div>
            </label>
          </div>

          {letsEncryptMutation.isError && (
            <p className="text-sm text-destructive">
              {(letsEncryptMutation.error as AxiosError<APIError>)?.response
                ?.data?.error?.message ?? "Issuance failed"}
            </p>
          )}

          <DialogFooter>
            <Button
              variant="outline"
              onClick={() => setShowIssueDialog(false)}
              disabled={letsEncryptMutation.isPending}
            >
              Cancel
            </Button>
            <Button
              onClick={() => letsEncryptMutation.mutate(issueOpts)}
              disabled={letsEncryptMutation.isPending}
              className="bg-pink-500 hover:bg-pink-600"
            >
              {letsEncryptMutation.isPending ? (
                <Loader2 className="h-4 w-4 mr-1 animate-spin" />
              ) : (
                <Zap className="h-4 w-4 mr-1" />
              )}
              {letsEncryptMutation.isPending
                ? "Issuing..."
                : "Issue Certificate"}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* Custom Install Form */}
      {showInstall && (
        <Card>
          <CardHeader>
            <CardTitle>Install Custom Certificate</CardTitle>
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
              <Label>CA Chain (optional)</Label>
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
                {installMutation.isPending
                  ? "Installing..."
                  : "Install Certificate"}
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
