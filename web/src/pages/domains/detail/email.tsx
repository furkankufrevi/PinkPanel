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
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Badge } from "@/components/ui/badge";
import { Switch } from "@/components/ui/switch";
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
  listEmailAccounts,
  createEmailAccount,
  deleteEmailAccount,
  changeEmailPassword,
  toggleEmailAccount,
  openWebmail,
  listEmailForwarders,
  createEmailForwarder,
  deleteEmailForwarder,
  getEmailDNSRecords,
  applyEmailDNSRecords,
  getSpamSettings,
  updateSpamSettings,
  getSpamList,
  addSpamEntry,
  deleteSpamEntry,
  getAutodiscoveryStatus,
  setupAutodiscovery,
} from "@/api/email";
import type { EmailAccount, EmailForwarder, SpamListEntry } from "@/api/email";
import type { AxiosError } from "axios";
import type { APIError } from "@/types/api";
import { getDomain } from "@/api/domains";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import {
  Mail,
  Plus,
  Trash2,
  Key,
  ArrowRight,
  ShieldCheck,
  ShieldAlert,
  Check,
  Loader2,
  ExternalLink,
  Shield,
  Wifi,
} from "lucide-react";

export function DomainEmail() {
  const { id } = useParams<{ id: string }>();
  const domainId = Number(id);

  const { data: domain } = useQuery({
    queryKey: ["domain", domainId],
    queryFn: () => getDomain(domainId),
    enabled: !!domainId,
  });

  return (
    <div className="space-y-6 max-w-3xl">
      <AccountsSection domainId={domainId} domainName={domain?.name ?? ""} />
      <ForwardersSection domainId={domainId} domainName={domain?.name ?? ""} />
      <SpamFilterSection domainId={domainId} />
      <AutodiscoverySection domainId={domainId} />
      <DNSRecordsSection domainId={domainId} />
    </div>
  );
}

// ─── Accounts ────────────────────────────────────

function AccountsSection({ domainId, domainName }: { domainId: number; domainName: string }) {
  const queryClient = useQueryClient();
  const [showCreate, setShowCreate] = useState(false);
  const [newAddress, setNewAddress] = useState("");
  const [newPassword, setNewPassword] = useState("");
  const [newQuota, setNewQuota] = useState("0");
  const [deleteTarget, setDeleteTarget] = useState<EmailAccount | null>(null);
  const [passwordTarget, setPasswordTarget] = useState<EmailAccount | null>(null);
  const [passwordValue, setPasswordValue] = useState("");

  const { data, isLoading } = useQuery({
    queryKey: ["email-accounts", domainId],
    queryFn: () => listEmailAccounts(domainId),
    enabled: !!domainId,
  });

  const createMut = useMutation({
    mutationFn: () =>
      createEmailAccount(domainId, {
        address: newAddress,
        password: newPassword,
        quota_mb: Number(newQuota) || 0,
      }),
    onSuccess: () => {
      toast.success("Email account created");
      setShowCreate(false);
      setNewAddress("");
      setNewPassword("");
      setNewQuota("0");
      queryClient.invalidateQueries({ queryKey: ["email-accounts", domainId] });
    },
    onError: (err: AxiosError<APIError>) => {
      toast.error(err.response?.data?.error?.message ?? "Failed to create account");
    },
  });

  const deleteMut = useMutation({
    mutationFn: () => deleteEmailAccount(domainId, deleteTarget!.id),
    onSuccess: () => {
      toast.success("Email account deleted");
      setDeleteTarget(null);
      queryClient.invalidateQueries({ queryKey: ["email-accounts", domainId] });
    },
    onError: (err: AxiosError<APIError>) => {
      toast.error(err.response?.data?.error?.message ?? "Failed to delete account");
    },
  });

  const toggleMut = useMutation({
    mutationFn: ({ accountId, enabled }: { accountId: number; enabled: boolean }) =>
      toggleEmailAccount(domainId, accountId, enabled),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["email-accounts", domainId] });
    },
    onError: (err: AxiosError<APIError>) => {
      toast.error(err.response?.data?.error?.message ?? "Failed to toggle account");
    },
  });

  const passwordMut = useMutation({
    mutationFn: () => changeEmailPassword(domainId, passwordTarget!.id, passwordValue),
    onSuccess: () => {
      toast.success("Password changed");
      setPasswordTarget(null);
      setPasswordValue("");
    },
    onError: (err: AxiosError<APIError>) => {
      toast.error(err.response?.data?.error?.message ?? "Failed to change password");
    },
  });

  if (isLoading) return <Skeleton className="h-48 w-full" />;

  const accounts = data?.data ?? [];

  return (
    <>
      <Card>
        <CardHeader className="pb-3">
          <div className="flex items-center justify-between">
            <div>
              <CardTitle className="flex items-center gap-2">
                <Mail className="h-5 w-5 text-pink-500" />
                Email Accounts
              </CardTitle>
              <CardDescription>
                Manage mailboxes for {domainName || "this domain"}
              </CardDescription>
            </div>
            <Button
              size="sm"
              onClick={() => {
                setShowCreate(true);
                setNewAddress("");
                setNewPassword("");
                setNewQuota("0");
              }}
              className="bg-pink-500 hover:bg-pink-600"
            >
              <Plus className="h-4 w-4 mr-1" />
              New Account
            </Button>
          </div>
        </CardHeader>
        <CardContent>
          {accounts.length === 0 ? (
            <div className="py-6 text-center">
              <Mail className="h-10 w-10 mx-auto text-muted-foreground mb-3" />
              <p className="text-sm text-muted-foreground">No email accounts yet</p>
            </div>
          ) : (
            <div className="space-y-2">
              {accounts.map((account) => (
                <div
                  key={account.id}
                  className="flex items-center justify-between p-3 rounded-lg border"
                >
                  <div className="flex items-center gap-3">
                    <Mail className="h-4 w-4 text-pink-500" />
                    <div>
                      <p className="text-sm font-medium">
                        {account.address}@{domainName}
                      </p>
                      <p className="text-xs text-muted-foreground">
                        {account.quota_mb > 0 ? `${account.quota_mb} MB quota` : "Unlimited"}
                      </p>
                    </div>
                  </div>
                  <div className="flex items-center gap-2">
                    <Button
                      size="sm"
                      variant="outline"
                      className="h-7 text-xs"
                      onClick={async () => {
                        try {
                          const res = await openWebmail(domainId, account.id);
                          window.open(res.url, "_blank");
                        } catch {
                          toast.error("Failed to open webmail. Change the password first to enable access.");
                        }
                      }}
                    >
                      <ExternalLink className="h-3 w-3 mr-1" />
                      Inbox
                    </Button>
                    <Switch
                      checked={account.enabled}
                      onCheckedChange={(checked) =>
                        toggleMut.mutate({ accountId: account.id, enabled: !!checked })
                      }
                    />
                    <Button
                      size="icon"
                      variant="ghost"
                      className="h-7 w-7"
                      onClick={() => {
                        setPasswordTarget(account);
                        setPasswordValue("");
                      }}
                    >
                      <Key className="h-3 w-3" />
                    </Button>
                    <Button
                      size="icon"
                      variant="ghost"
                      className="h-7 w-7 text-destructive"
                      onClick={() => setDeleteTarget(account)}
                    >
                      <Trash2 className="h-3 w-3" />
                    </Button>
                  </div>
                </div>
              ))}
            </div>
          )}
        </CardContent>
      </Card>

      {/* Create Dialog */}
      <Dialog open={showCreate} onOpenChange={setShowCreate}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>New Email Account</DialogTitle>
            <DialogDescription>
              Create a mailbox for {domainName}
            </DialogDescription>
          </DialogHeader>
          <div className="space-y-4">
            <div className="space-y-2">
              <Label>Address</Label>
              <div className="flex items-center gap-2">
                <Input
                  value={newAddress}
                  onChange={(e) => setNewAddress(e.target.value)}
                  placeholder="user"
                  autoFocus
                />
                <span className="text-sm text-muted-foreground whitespace-nowrap">
                  @{domainName}
                </span>
              </div>
            </div>
            <div className="space-y-2">
              <Label>Password</Label>
              <Input
                type="password"
                value={newPassword}
                onChange={(e) => setNewPassword(e.target.value)}
                placeholder="Strong password"
              />
            </div>
            <div className="space-y-2">
              <Label>Quota (MB)</Label>
              <Input
                type="number"
                value={newQuota}
                onChange={(e) => setNewQuota(e.target.value)}
                placeholder="0 = unlimited"
              />
              <p className="text-xs text-muted-foreground">0 for unlimited</p>
            </div>
          </div>
          <DialogFooter>
            <Button variant="outline" onClick={() => setShowCreate(false)}>
              Cancel
            </Button>
            <Button
              onClick={() => createMut.mutate()}
              disabled={!newAddress || !newPassword || createMut.isPending}
              className="bg-pink-500 hover:bg-pink-600"
            >
              {createMut.isPending ? "Creating..." : "Create"}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* Change Password Dialog */}
      <Dialog open={!!passwordTarget} onOpenChange={() => setPasswordTarget(null)}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Change Password</DialogTitle>
            <DialogDescription>
              {passwordTarget?.address}@{domainName}
            </DialogDescription>
          </DialogHeader>
          <div className="space-y-2">
            <Label>New Password</Label>
            <Input
              type="password"
              value={passwordValue}
              onChange={(e) => setPasswordValue(e.target.value)}
              placeholder="New password"
              autoFocus
            />
          </div>
          <DialogFooter>
            <Button variant="outline" onClick={() => setPasswordTarget(null)}>
              Cancel
            </Button>
            <Button
              onClick={() => passwordMut.mutate()}
              disabled={!passwordValue || passwordMut.isPending}
              className="bg-pink-500 hover:bg-pink-600"
            >
              {passwordMut.isPending ? "Changing..." : "Change Password"}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* Delete Confirmation */}
      <ConfirmDialog
        open={!!deleteTarget}
        onOpenChange={() => setDeleteTarget(null)}
        title="Delete Email Account"
        description={`Permanently delete "${deleteTarget?.address}@${domainName}" and all its mail?`}
        confirmText="Delete"
        typeToConfirm={deleteTarget ? `${deleteTarget.address}@${domainName}` : undefined}
        destructive
        loading={deleteMut.isPending}
        onConfirm={() => deleteMut.mutate()}
      />
    </>
  );
}

// ─── Forwarders ──────────────────────────────────

function ForwardersSection({ domainId, domainName }: { domainId: number; domainName: string }) {
  const queryClient = useQueryClient();
  const [showCreate, setShowCreate] = useState(false);
  const [newSource, setNewSource] = useState("");
  const [newDest, setNewDest] = useState("");
  const [deleteTarget, setDeleteTarget] = useState<EmailForwarder | null>(null);

  const { data, isLoading } = useQuery({
    queryKey: ["email-forwarders", domainId],
    queryFn: () => listEmailForwarders(domainId),
    enabled: !!domainId,
  });

  const createMut = useMutation({
    mutationFn: () =>
      createEmailForwarder(domainId, {
        source_address: newSource,
        destination: newDest,
      }),
    onSuccess: () => {
      toast.success("Forwarder created");
      setShowCreate(false);
      setNewSource("");
      setNewDest("");
      queryClient.invalidateQueries({ queryKey: ["email-forwarders", domainId] });
    },
    onError: (err: AxiosError<APIError>) => {
      toast.error(err.response?.data?.error?.message ?? "Failed to create forwarder");
    },
  });

  const deleteMut = useMutation({
    mutationFn: () => deleteEmailForwarder(domainId, deleteTarget!.id),
    onSuccess: () => {
      toast.success("Forwarder deleted");
      setDeleteTarget(null);
      queryClient.invalidateQueries({ queryKey: ["email-forwarders", domainId] });
    },
    onError: (err: AxiosError<APIError>) => {
      toast.error(err.response?.data?.error?.message ?? "Failed to delete forwarder");
    },
  });

  if (isLoading) return <Skeleton className="h-32 w-full" />;

  const forwarders = data?.data ?? [];

  return (
    <>
      <Card>
        <CardHeader className="pb-3">
          <div className="flex items-center justify-between">
            <div>
              <CardTitle className="text-sm">Email Forwarders</CardTitle>
              <CardDescription>Forward mail to external addresses</CardDescription>
            </div>
            <Button
              size="sm"
              variant="outline"
              onClick={() => {
                setShowCreate(true);
                setNewSource("");
                setNewDest("");
              }}
            >
              <Plus className="h-4 w-4 mr-1" />
              New Forwarder
            </Button>
          </div>
        </CardHeader>
        <CardContent>
          {forwarders.length === 0 ? (
            <p className="text-sm text-muted-foreground text-center py-4">
              No forwarders configured
            </p>
          ) : (
            <div className="space-y-2">
              {forwarders.map((fwd) => (
                <div
                  key={fwd.id}
                  className="flex items-center justify-between p-3 rounded-lg border text-sm"
                >
                  <div className="flex items-center gap-2">
                    <span className="font-medium">
                      {fwd.source_address}@{domainName}
                    </span>
                    <ArrowRight className="h-3 w-3 text-muted-foreground" />
                    <span className="text-muted-foreground">{fwd.destination}</span>
                  </div>
                  <Button
                    size="icon"
                    variant="ghost"
                    className="h-7 w-7 text-destructive"
                    onClick={() => setDeleteTarget(fwd)}
                  >
                    <Trash2 className="h-3 w-3" />
                  </Button>
                </div>
              ))}
            </div>
          )}
        </CardContent>
      </Card>

      {/* Create Forwarder Dialog */}
      <Dialog open={showCreate} onOpenChange={setShowCreate}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>New Email Forwarder</DialogTitle>
            <DialogDescription>
              Forward mail from this domain to another address
            </DialogDescription>
          </DialogHeader>
          <div className="space-y-4">
            <div className="space-y-2">
              <Label>Source</Label>
              <div className="flex items-center gap-2">
                <Input
                  value={newSource}
                  onChange={(e) => setNewSource(e.target.value)}
                  placeholder="sales"
                  autoFocus
                />
                <span className="text-sm text-muted-foreground whitespace-nowrap">
                  @{domainName}
                </span>
              </div>
            </div>
            <div className="space-y-2">
              <Label>Forward to</Label>
              <Input
                type="email"
                value={newDest}
                onChange={(e) => setNewDest(e.target.value)}
                placeholder="user@example.com"
              />
            </div>
          </div>
          <DialogFooter>
            <Button variant="outline" onClick={() => setShowCreate(false)}>
              Cancel
            </Button>
            <Button
              onClick={() => createMut.mutate()}
              disabled={!newSource || !newDest || createMut.isPending}
              className="bg-pink-500 hover:bg-pink-600"
            >
              {createMut.isPending ? "Creating..." : "Create"}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* Delete Confirmation */}
      <ConfirmDialog
        open={!!deleteTarget}
        onOpenChange={() => setDeleteTarget(null)}
        title="Delete Forwarder"
        description={`Delete forwarder "${deleteTarget?.source_address}@${domainName}"?`}
        confirmText="Delete"
        destructive
        loading={deleteMut.isPending}
        onConfirm={() => deleteMut.mutate()}
      />
    </>
  );
}

// ─── Spam Filter ─────────────────────────────────

function SpamFilterSection({ domainId }: { domainId: number }) {
  const queryClient = useQueryClient();
  const [newEntry, setNewEntry] = useState("");
  const [newListType, setNewListType] = useState<"whitelist" | "blacklist">("whitelist");

  const { data: settings, isLoading } = useQuery({
    queryKey: ["spam-settings", domainId],
    queryFn: () => getSpamSettings(domainId),
    enabled: !!domainId,
  });

  const { data: whitelistData } = useQuery({
    queryKey: ["spam-list", domainId, "whitelist"],
    queryFn: () => getSpamList(domainId, "whitelist"),
    enabled: !!domainId,
  });

  const { data: blacklistData } = useQuery({
    queryKey: ["spam-list", domainId, "blacklist"],
    queryFn: () => getSpamList(domainId, "blacklist"),
    enabled: !!domainId,
  });

  const updateMut = useMutation({
    mutationFn: (s: { enabled: boolean; score_threshold: number; action: string }) =>
      updateSpamSettings(domainId, s),
    onSuccess: () => {
      toast.success("Spam settings updated");
      queryClient.invalidateQueries({ queryKey: ["spam-settings", domainId] });
    },
    onError: (err: AxiosError<APIError>) => {
      toast.error(err.response?.data?.error?.message ?? "Failed to update spam settings");
    },
  });

  const addMut = useMutation({
    mutationFn: () => addSpamEntry(domainId, { list_type: newListType, entry: newEntry }),
    onSuccess: () => {
      toast.success("Entry added");
      setNewEntry("");
      queryClient.invalidateQueries({ queryKey: ["spam-list", domainId, newListType] });
    },
    onError: (err: AxiosError<APIError>) => {
      toast.error(err.response?.data?.error?.message ?? "Failed to add entry");
    },
  });

  const deleteMut = useMutation({
    mutationFn: (entry: SpamListEntry) => deleteSpamEntry(domainId, entry.id),
    onSuccess: (_: void, entry: SpamListEntry) => {
      queryClient.invalidateQueries({ queryKey: ["spam-list", domainId, entry.list_type] });
    },
  });

  if (isLoading) return <Skeleton className="h-32 w-full" />;

  const whitelist = whitelistData?.data ?? [];
  const blacklist = blacklistData?.data ?? [];

  return (
    <Card>
      <CardHeader className="pb-3">
        <CardTitle className="text-sm flex items-center gap-2">
          <Shield className="h-4 w-4 text-pink-500" />
          Spam Filter (SpamAssassin)
        </CardTitle>
        <CardDescription>
          Configure spam filtering for this domain
        </CardDescription>
      </CardHeader>
      <CardContent className="space-y-4">
        <div className="flex items-center justify-between">
          <Label>Enable Spam Filtering</Label>
          <Switch
            checked={settings?.enabled ?? false}
            onCheckedChange={(checked) =>
              updateMut.mutate({
                enabled: !!checked,
                score_threshold: settings?.score_threshold ?? 5.0,
                action: settings?.action ?? "mark",
              })
            }
          />
        </div>

        {settings?.enabled && (
          <>
            <div className="grid grid-cols-2 gap-4">
              <div className="space-y-2">
                <Label>Score Threshold</Label>
                <Input
                  type="number"
                  step="0.5"
                  min="1"
                  max="20"
                  value={settings.score_threshold}
                  onChange={(e) =>
                    updateMut.mutate({
                      enabled: true,
                      score_threshold: parseFloat(e.target.value) || 5.0,
                      action: settings.action,
                    })
                  }
                />
                <p className="text-xs text-muted-foreground">Lower = stricter (default: 5.0)</p>
              </div>
              <div className="space-y-2">
                <Label>Action on Spam</Label>
                <Select
                  value={settings.action}
                  onValueChange={(v) =>
                    v && updateMut.mutate({
                      enabled: true,
                      score_threshold: settings.score_threshold,
                      action: v,
                    })
                  }
                >
                  <SelectTrigger>
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="mark">Mark as spam (headers)</SelectItem>
                    <SelectItem value="junk">Move to Junk folder</SelectItem>
                    <SelectItem value="delete">Delete</SelectItem>
                  </SelectContent>
                </Select>
              </div>
            </div>

            <div className="border-t pt-4 space-y-3">
              <div className="flex items-center gap-2">
                <Select
                  value={newListType}
                  onValueChange={(v) => v && setNewListType(v as "whitelist" | "blacklist")}
                >
                  <SelectTrigger className="w-32">
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="whitelist">Whitelist</SelectItem>
                    <SelectItem value="blacklist">Blacklist</SelectItem>
                  </SelectContent>
                </Select>
                <Input
                  value={newEntry}
                  onChange={(e) => setNewEntry(e.target.value)}
                  placeholder="user@example.com or *@example.com"
                  className="flex-1"
                />
                <Button
                  size="sm"
                  onClick={() => addMut.mutate()}
                  disabled={!newEntry || addMut.isPending}
                >
                  <Plus className="h-4 w-4" />
                </Button>
              </div>

              {whitelist.length > 0 && (
                <div>
                  <p className="text-xs font-medium text-muted-foreground mb-1">Whitelist</p>
                  <div className="space-y-1">
                    {whitelist.map((e) => (
                      <div key={e.id} className="flex items-center justify-between px-2 py-1 rounded border text-sm">
                        <span className="text-green-500">{e.entry}</span>
                        <Button size="icon" variant="ghost" className="h-6 w-6" onClick={() => deleteMut.mutate(e)}>
                          <Trash2 className="h-3 w-3" />
                        </Button>
                      </div>
                    ))}
                  </div>
                </div>
              )}

              {blacklist.length > 0 && (
                <div>
                  <p className="text-xs font-medium text-muted-foreground mb-1">Blacklist</p>
                  <div className="space-y-1">
                    {blacklist.map((e) => (
                      <div key={e.id} className="flex items-center justify-between px-2 py-1 rounded border text-sm">
                        <span className="text-red-500">{e.entry}</span>
                        <Button size="icon" variant="ghost" className="h-6 w-6" onClick={() => deleteMut.mutate(e)}>
                          <Trash2 className="h-3 w-3" />
                        </Button>
                      </div>
                    ))}
                  </div>
                </div>
              )}
            </div>
          </>
        )}
      </CardContent>
    </Card>
  );
}

// ─── Autodiscovery ───────────────────────────────

function AutodiscoverySection({ domainId }: { domainId: number }) {
  const queryClient = useQueryClient();

  const { data, isLoading } = useQuery({
    queryKey: ["autodiscovery", domainId],
    queryFn: () => getAutodiscoveryStatus(domainId),
    enabled: !!domainId,
    retry: false,
  });

  const setupMut = useMutation({
    mutationFn: () => setupAutodiscovery(domainId),
    onSuccess: (res) => {
      toast.success(`Autodiscovery configured (${res.created} DNS records created)`);
      queryClient.invalidateQueries({ queryKey: ["autodiscovery", domainId] });
    },
    onError: (err: AxiosError<APIError>) => {
      toast.error(err.response?.data?.error?.message ?? "Failed to setup autodiscovery");
    },
  });

  if (isLoading) return <Skeleton className="h-24 w-full" />;

  const allConfigured = data?.configured ?? false;

  return (
    <Card>
      <CardHeader className="pb-3">
        <div className="flex items-center justify-between">
          <div>
            <CardTitle className="text-sm flex items-center gap-2">
              <Wifi className="h-4 w-4 text-pink-500" />
              Mail Autodiscovery
            </CardTitle>
            <CardDescription>
              {allConfigured
                ? "Thunderbird & Outlook will auto-detect mail settings"
                : "Auto-configure email clients (Thunderbird, Outlook)"}
            </CardDescription>
          </div>
          {!allConfigured && (
            <Button
              size="sm"
              onClick={() => setupMut.mutate()}
              disabled={setupMut.isPending}
              className="bg-pink-500 hover:bg-pink-600"
            >
              {setupMut.isPending ? (
                <Loader2 className="h-4 w-4 mr-1 animate-spin" />
              ) : (
                <Plus className="h-4 w-4 mr-1" />
              )}
              Setup
            </Button>
          )}
        </div>
      </CardHeader>
      <CardContent>
        <div className="grid grid-cols-3 gap-3">
          <div className="p-3 rounded-lg border text-center">
            <Badge
              variant="outline"
              className={data?.srv_records ? "text-green-500 border-green-500/30" : "text-yellow-500 border-yellow-500/30"}
            >
              {data?.srv_records ? <Check className="h-3 w-3 mr-1" /> : null}
              SRV Records
            </Badge>
          </div>
          <div className="p-3 rounded-lg border text-center">
            <Badge
              variant="outline"
              className={data?.autoconfig ? "text-green-500 border-green-500/30" : "text-yellow-500 border-yellow-500/30"}
            >
              {data?.autoconfig ? <Check className="h-3 w-3 mr-1" /> : null}
              Thunderbird
            </Badge>
          </div>
          <div className="p-3 rounded-lg border text-center">
            <Badge
              variant="outline"
              className={data?.autodiscover ? "text-green-500 border-green-500/30" : "text-yellow-500 border-yellow-500/30"}
            >
              {data?.autodiscover ? <Check className="h-3 w-3 mr-1" /> : null}
              Outlook
            </Badge>
          </div>
        </div>
      </CardContent>
    </Card>
  );
}

// ─── DNS Records ─────────────────────────────────

function DNSRecordsSection({ domainId }: { domainId: number }) {
  const queryClient = useQueryClient();

  const { data, isLoading } = useQuery({
    queryKey: ["email-dns", domainId],
    queryFn: () => getEmailDNSRecords(domainId),
    enabled: !!domainId,
    retry: false,
  });

  const applyMut = useMutation({
    mutationFn: () => applyEmailDNSRecords(domainId),
    onSuccess: (res) => {
      toast.success(`${res.created} DNS record(s) created`);
      queryClient.invalidateQueries({ queryKey: ["email-dns", domainId] });
    },
    onError: (err: AxiosError<APIError>) => {
      toast.error(err.response?.data?.error?.message ?? "Failed to apply DNS records");
    },
  });

  if (isLoading) return <Skeleton className="h-32 w-full" />;

  const records = data?.records ?? [];
  const allExist = records.length > 0 && records.every((r) => r.exists);
  const missingCount = records.filter((r) => !r.exists).length;

  return (
    <Card>
      <CardHeader className="pb-3">
        <div className="flex items-center justify-between">
          <div>
            <CardTitle className="text-sm flex items-center gap-2">
              {allExist ? (
                <ShieldCheck className="h-4 w-4 text-green-500" />
              ) : (
                <ShieldAlert className="h-4 w-4 text-yellow-500" />
              )}
              Email DNS Records
            </CardTitle>
            <CardDescription>
              {allExist
                ? "All recommended records are configured"
                : `${missingCount} missing record(s)`}
            </CardDescription>
          </div>
          {!allExist && (
            <Button
              size="sm"
              onClick={() => applyMut.mutate()}
              disabled={applyMut.isPending}
              className="bg-pink-500 hover:bg-pink-600"
            >
              {applyMut.isPending ? (
                <Loader2 className="h-4 w-4 mr-1 animate-spin" />
              ) : (
                <Plus className="h-4 w-4 mr-1" />
              )}
              Apply Missing
            </Button>
          )}
        </div>
      </CardHeader>
      <CardContent>
        {records.length === 0 ? (
          <p className="text-sm text-muted-foreground text-center py-4">
            Could not generate DNS recommendations
          </p>
        ) : (
          <div className="space-y-3">
            {records.map((rec) => (
              <div key={rec.label} className="p-3 rounded-lg border space-y-1">
                <div className="flex items-center justify-between">
                  <div className="flex items-center gap-2">
                    <Badge variant="outline" className="text-xs font-mono">
                      {rec.label}
                    </Badge>
                    <span className="text-xs text-muted-foreground">
                      {rec.name}
                    </span>
                  </div>
                  {rec.exists ? (
                    <Badge className="bg-green-500/10 text-green-500 border-green-500/20">
                      <Check className="h-3 w-3 mr-1" />
                      Configured
                    </Badge>
                  ) : (
                    <Badge className="bg-yellow-500/10 text-yellow-500 border-yellow-500/20">
                      Missing
                    </Badge>
                  )}
                </div>
                <p className="text-xs font-mono text-muted-foreground break-all">
                  {rec.value.length > 120 ? rec.value.slice(0, 120) + "..." : rec.value}
                </p>
              </div>
            ))}
          </div>
        )}
      </CardContent>
    </Card>
  );
}
