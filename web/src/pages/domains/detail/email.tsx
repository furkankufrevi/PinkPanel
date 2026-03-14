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
import { Tabs, TabsList, TabsTrigger, TabsContent } from "@/components/ui/tabs";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { ConfirmDialog } from "@/components/shared/confirm-dialog";
import { toast } from "sonner";
import {
  listEmailAccounts,
  createEmailAccount,
  deleteEmailAccount,
  changeEmailPassword,
  toggleEmailAccount,
  updateEmailQuota,
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
  getMailSSLStatus,
  configureMailSSL,
} from "@/api/email";
import type { EmailAccount, EmailForwarder, SpamListEntry } from "@/api/email";
import type { AxiosError } from "axios";
import type { APIError } from "@/types/api";
import { getDomain } from "@/api/domains";
import {
  Mail,
  Plus,
  Trash2,
  Key,
  ArrowRight,
  Check,
  Loader2,
  ExternalLink,
  Shield,
  Wifi,
  Lock,
  ShieldCheck,
  ShieldAlert,
  MailPlus,
  Forward,
  Settings,
  Eye,
  EyeOff,
  Copy,
  RefreshCw,
  Inbox,
  Ban,
  CheckCircle2,
  XCircle,
  AlertCircle,
} from "lucide-react";

function generatePassword(length = 16): string {
  const chars = "abcdefghijkmnopqrstuvwxyzABCDEFGHJKLMNPQRSTUVWXYZ23456789!@#$%&*";
  const array = new Uint8Array(length);
  crypto.getRandomValues(array);
  return Array.from(array, (b) => chars[b % chars.length]).join("");
}

function copyToClipboard(text: string) {
  navigator.clipboard.writeText(text);
  toast.success("Copied to clipboard");
}

export function DomainEmail() {
  const { id } = useParams<{ id: string }>();
  const domainId = Number(id);

  const { data: domain } = useQuery({
    queryKey: ["domain", domainId],
    queryFn: () => getDomain(domainId),
    enabled: !!domainId,
  });

  const { data: accountsData, isLoading: accountsLoading } = useQuery({
    queryKey: ["email-accounts", domainId],
    queryFn: () => listEmailAccounts(domainId),
    enabled: !!domainId,
  });

  const { data: forwardersData, isLoading: forwardersLoading } = useQuery({
    queryKey: ["email-forwarders", domainId],
    queryFn: () => listEmailForwarders(domainId),
    enabled: !!domainId,
  });

  const accounts = accountsData?.data ?? [];
  const forwarders = forwardersData?.data ?? [];
  const domainName = domain?.name ?? "";
  const activeCount = accounts.filter((a) => a.enabled).length;

  if (accountsLoading || forwardersLoading) {
    return (
      <div className="space-y-4 max-w-5xl">
        <Skeleton className="h-24 w-full" />
        <Skeleton className="h-64 w-full" />
      </div>
    );
  }

  return (
    <div className="space-y-6 max-w-5xl">
      {/* Summary Stats */}
      <div className="grid grid-cols-1 sm:grid-cols-3 gap-4">
        <Card>
          <CardContent className="pt-6">
            <div className="flex items-center gap-3">
              <div className="flex h-10 w-10 items-center justify-center rounded-lg bg-pink-500/10">
                <Mail className="h-5 w-5 text-pink-500" />
              </div>
              <div>
                <p className="text-2xl font-bold">{accounts.length}</p>
                <p className="text-xs text-muted-foreground">
                  {accounts.length === 1 ? "Mailbox" : "Mailboxes"}
                </p>
              </div>
            </div>
          </CardContent>
        </Card>
        <Card>
          <CardContent className="pt-6">
            <div className="flex items-center gap-3">
              <div className="flex h-10 w-10 items-center justify-center rounded-lg bg-blue-500/10">
                <Forward className="h-5 w-5 text-blue-500" />
              </div>
              <div>
                <p className="text-2xl font-bold">{forwarders.length}</p>
                <p className="text-xs text-muted-foreground">
                  {forwarders.length === 1 ? "Forwarder" : "Forwarders"}
                </p>
              </div>
            </div>
          </CardContent>
        </Card>
        <Card>
          <CardContent className="pt-6">
            <div className="flex items-center gap-3">
              <div className="flex h-10 w-10 items-center justify-center rounded-lg bg-green-500/10">
                <CheckCircle2 className="h-5 w-5 text-green-500" />
              </div>
              <div>
                <p className="text-2xl font-bold">
                  {activeCount}
                  <span className="text-sm font-normal text-muted-foreground">
                    /{accounts.length}
                  </span>
                </p>
                <p className="text-xs text-muted-foreground">Active</p>
              </div>
            </div>
          </CardContent>
        </Card>
      </div>

      {/* Tabbed Content */}
      <Tabs defaultValue="mailboxes">
        <TabsList>
          <TabsTrigger value="mailboxes">
            <Mail className="h-4 w-4" />
            Mailboxes
          </TabsTrigger>
          <TabsTrigger value="forwarders">
            <Forward className="h-4 w-4" />
            Forwarders
          </TabsTrigger>
          <TabsTrigger value="spam">
            <Shield className="h-4 w-4" />
            Spam Protection
          </TabsTrigger>
          <TabsTrigger value="settings">
            <Settings className="h-4 w-4" />
            Settings
          </TabsTrigger>
        </TabsList>

        <TabsContent value="mailboxes">
          <AccountsSection
            domainId={domainId}
            domainName={domainName}
            accounts={accounts}
          />
        </TabsContent>

        <TabsContent value="forwarders">
          <ForwardersSection
            domainId={domainId}
            domainName={domainName}
            forwarders={forwarders}
          />
        </TabsContent>

        <TabsContent value="spam">
          <SpamFilterSection domainId={domainId} />
        </TabsContent>

        <TabsContent value="settings">
          <div className="space-y-6">
            <AutodiscoverySection domainId={domainId} />
            <MailSSLSection domainId={domainId} />
            <DNSRecordsSection domainId={domainId} />
          </div>
        </TabsContent>
      </Tabs>
    </div>
  );
}

// ─── Accounts ────────────────────────────────────

function AccountsSection({
  domainId,
  domainName,
  accounts,
}: {
  domainId: number;
  domainName: string;
  accounts: EmailAccount[];
}) {
  const queryClient = useQueryClient();
  const [showCreate, setShowCreate] = useState(false);
  const [newAddress, setNewAddress] = useState("");
  const [newPassword, setNewPassword] = useState("");
  const [newQuota, setNewQuota] = useState("0");
  const [showNewPassword, setShowNewPassword] = useState(false);
  const [deleteTarget, setDeleteTarget] = useState<EmailAccount | null>(null);
  const [passwordTarget, setPasswordTarget] = useState<EmailAccount | null>(null);
  const [passwordValue, setPasswordValue] = useState("");
  const [showChangePassword, setShowChangePassword] = useState(false);
  const [quotaTarget, setQuotaTarget] = useState<EmailAccount | null>(null);
  const [quotaValue, setQuotaValue] = useState("");

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
      setShowNewPassword(false);
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
      setShowChangePassword(false);
    },
    onError: (err: AxiosError<APIError>) => {
      toast.error(err.response?.data?.error?.message ?? "Failed to change password");
    },
  });

  const quotaMut = useMutation({
    mutationFn: () => updateEmailQuota(domainId, quotaTarget!.id, Number(quotaValue) || 0),
    onSuccess: () => {
      toast.success("Quota updated");
      setQuotaTarget(null);
      setQuotaValue("");
      queryClient.invalidateQueries({ queryKey: ["email-accounts", domainId] });
    },
    onError: (err: AxiosError<APIError>) => {
      toast.error(err.response?.data?.error?.message ?? "Failed to update quota");
    },
  });

  return (
    <>
      <Card>
        <CardHeader>
          <div className="flex items-center justify-between">
            <div>
              <CardTitle className="flex items-center gap-2">
                <MailPlus className="h-5 w-5 text-pink-500" />
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
                const pwd = generatePassword();
                setNewPassword(pwd);
                setNewQuota("0");
                setShowNewPassword(true);
              }}
              className="bg-pink-500 hover:bg-pink-600"
            >
              <Plus className="h-4 w-4 mr-1" />
              New Mailbox
            </Button>
          </div>
        </CardHeader>
        <CardContent>
          {accounts.length === 0 ? (
            <div className="py-12 text-center">
              <div className="flex h-16 w-16 mx-auto items-center justify-center rounded-full bg-muted mb-4">
                <Inbox className="h-8 w-8 text-muted-foreground" />
              </div>
              <h3 className="text-sm font-medium mb-1">No mailboxes yet</h3>
              <p className="text-sm text-muted-foreground max-w-sm mx-auto">
                Create your first email account to start sending and receiving
                emails on {domainName}.
              </p>
            </div>
          ) : (
            <div className="border rounded-lg overflow-hidden">
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead>Email Address</TableHead>
                    <TableHead className="w-[120px]">Quota</TableHead>
                    <TableHead className="w-[100px]">Status</TableHead>
                    <TableHead className="w-[180px] text-right">Actions</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {accounts.map((account) => (
                    <TableRow key={account.id}>
                      <TableCell>
                        <div className="flex items-center gap-2.5">
                          <div className="flex h-8 w-8 items-center justify-center rounded-full bg-pink-500/10">
                            <Mail className="h-4 w-4 text-pink-500" />
                          </div>
                          <div>
                            <p className="font-medium text-sm">
                              {account.address}@{domainName}
                            </p>
                            <p className="text-xs text-muted-foreground">
                              Created{" "}
                              {new Date(account.created_at).toLocaleDateString()}
                            </p>
                          </div>
                        </div>
                      </TableCell>
                      <TableCell>
                        <button
                          className="text-left hover:text-pink-500 transition-colors"
                          onClick={() => {
                            setQuotaTarget(account);
                            setQuotaValue(String(account.quota_mb));
                          }}
                          title="Click to edit quota"
                        >
                          {account.quota_mb > 0 ? (
                            <Badge variant="outline" className="font-mono text-xs">
                              {account.quota_mb} MB
                            </Badge>
                          ) : (
                            <Badge variant="outline" className="text-xs text-muted-foreground">
                              Unlimited
                            </Badge>
                          )}
                        </button>
                      </TableCell>
                      <TableCell>
                        <div className="flex items-center gap-2">
                          <Switch
                            checked={account.enabled}
                            onCheckedChange={(checked) =>
                              toggleMut.mutate({ accountId: account.id, enabled: !!checked })
                            }
                          />
                          <span className={`text-xs ${account.enabled ? "text-green-500" : "text-muted-foreground"}`}>
                            {account.enabled ? "Active" : "Disabled"}
                          </span>
                        </div>
                      </TableCell>
                      <TableCell className="text-right">
                        <div className="flex items-center justify-end gap-1">
                          <Button
                            size="sm"
                            variant="ghost"
                            className="h-8 px-2 text-xs"
                            onClick={async () => {
                              try {
                                const res = await openWebmail(domainId, account.id);
                                window.open(res.url, "_blank");
                              } catch {
                                toast.error(
                                  "Failed to open webmail. Change the password first to enable access."
                                );
                              }
                            }}
                            title="Open Webmail"
                          >
                            <ExternalLink className="h-3.5 w-3.5 mr-1" />
                            Webmail
                          </Button>
                          <Button
                            size="icon"
                            variant="ghost"
                            className="h-8 w-8"
                            onClick={() => {
                              setPasswordTarget(account);
                              setPasswordValue("");
                              setShowChangePassword(false);
                            }}
                            title="Change Password"
                          >
                            <Key className="h-3.5 w-3.5" />
                          </Button>
                          <Button
                            size="icon"
                            variant="ghost"
                            className="h-8 w-8 text-destructive hover:text-destructive"
                            onClick={() => setDeleteTarget(account)}
                            title="Delete Account"
                          >
                            <Trash2 className="h-3.5 w-3.5" />
                          </Button>
                        </div>
                      </TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
            </div>
          )}
        </CardContent>
      </Card>

      {/* Create Dialog */}
      <Dialog open={showCreate} onOpenChange={setShowCreate}>
        <DialogContent className="sm:max-w-md">
          <DialogHeader>
            <DialogTitle>Create Email Account</DialogTitle>
            <DialogDescription>
              Create a new mailbox for {domainName}
            </DialogDescription>
          </DialogHeader>
          <div className="space-y-4">
            <div className="space-y-2">
              <Label>Email Address</Label>
              <div className="flex items-center gap-0">
                <Input
                  value={newAddress}
                  onChange={(e) => setNewAddress(e.target.value)}
                  placeholder="user"
                  className="rounded-r-none border-r-0"
                  autoFocus
                />
                <div className="flex h-9 items-center rounded-r-md border bg-muted px-3 text-sm text-muted-foreground">
                  @{domainName}
                </div>
              </div>
            </div>
            <div className="space-y-2">
              <Label>Password</Label>
              <div className="flex gap-1.5">
                <div className="relative flex-1">
                  <Input
                    type={showNewPassword ? "text" : "password"}
                    value={newPassword}
                    onChange={(e) => setNewPassword(e.target.value)}
                    placeholder="Strong password"
                    className="pr-16"
                  />
                  <div className="absolute inset-y-0 right-0 flex items-center gap-0.5 pr-1">
                    <Button
                      type="button"
                      variant="ghost"
                      size="icon"
                      className="h-7 w-7"
                      onClick={() => setShowNewPassword(!showNewPassword)}
                      tabIndex={-1}
                    >
                      {showNewPassword ? (
                        <EyeOff className="h-3.5 w-3.5" />
                      ) : (
                        <Eye className="h-3.5 w-3.5" />
                      )}
                    </Button>
                    <Button
                      type="button"
                      variant="ghost"
                      size="icon"
                      className="h-7 w-7"
                      onClick={() => copyToClipboard(newPassword)}
                      tabIndex={-1}
                    >
                      <Copy className="h-3.5 w-3.5" />
                    </Button>
                  </div>
                </div>
                <Button
                  type="button"
                  variant="outline"
                  size="icon"
                  className="h-9 w-9 shrink-0"
                  onClick={() => {
                    setNewPassword(generatePassword());
                    setShowNewPassword(true);
                  }}
                  title="Generate password"
                >
                  <RefreshCw className="h-3.5 w-3.5" />
                </Button>
              </div>
            </div>
            <div className="space-y-2">
              <Label>
                Mailbox Quota{" "}
                <span className="font-normal text-muted-foreground">(MB)</span>
              </Label>
              <Input
                type="number"
                value={newQuota}
                onChange={(e) => setNewQuota(e.target.value)}
                placeholder="0 = unlimited"
              />
              <p className="text-xs text-muted-foreground">
                Set to 0 for unlimited storage
              </p>
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
              {createMut.isPending && <Loader2 className="h-4 w-4 mr-1 animate-spin" />}
              {createMut.isPending ? "Creating..." : "Create Account"}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* Change Password Dialog */}
      <Dialog open={!!passwordTarget} onOpenChange={() => setPasswordTarget(null)}>
        <DialogContent className="sm:max-w-md">
          <DialogHeader>
            <DialogTitle>Change Password</DialogTitle>
            <DialogDescription>
              {passwordTarget?.address}@{domainName}
            </DialogDescription>
          </DialogHeader>
          <div className="space-y-2">
            <Label>New Password</Label>
            <div className="flex gap-1.5">
              <div className="relative flex-1">
                <Input
                  type={showChangePassword ? "text" : "password"}
                  value={passwordValue}
                  onChange={(e) => setPasswordValue(e.target.value)}
                  placeholder="New password"
                  autoFocus
                  className="pr-16"
                />
                <div className="absolute inset-y-0 right-0 flex items-center gap-0.5 pr-1">
                  <Button
                    type="button"
                    variant="ghost"
                    size="icon"
                    className="h-7 w-7"
                    onClick={() => setShowChangePassword(!showChangePassword)}
                    tabIndex={-1}
                  >
                    {showChangePassword ? (
                      <EyeOff className="h-3.5 w-3.5" />
                    ) : (
                      <Eye className="h-3.5 w-3.5" />
                    )}
                  </Button>
                  <Button
                    type="button"
                    variant="ghost"
                    size="icon"
                    className="h-7 w-7"
                    onClick={() => copyToClipboard(passwordValue)}
                    tabIndex={-1}
                  >
                    <Copy className="h-3.5 w-3.5" />
                  </Button>
                </div>
              </div>
              <Button
                type="button"
                variant="outline"
                size="icon"
                className="h-9 w-9 shrink-0"
                onClick={() => {
                  setPasswordValue(generatePassword());
                  setShowChangePassword(true);
                }}
                title="Generate password"
              >
                <RefreshCw className="h-3.5 w-3.5" />
              </Button>
            </div>
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
              {passwordMut.isPending && <Loader2 className="h-4 w-4 mr-1 animate-spin" />}
              {passwordMut.isPending ? "Changing..." : "Change Password"}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* Edit Quota Dialog */}
      <Dialog open={!!quotaTarget} onOpenChange={() => setQuotaTarget(null)}>
        <DialogContent className="sm:max-w-sm">
          <DialogHeader>
            <DialogTitle>Edit Quota</DialogTitle>
            <DialogDescription>
              {quotaTarget?.address}@{domainName}
            </DialogDescription>
          </DialogHeader>
          <div className="space-y-2">
            <Label>
              Quota{" "}
              <span className="font-normal text-muted-foreground">(MB)</span>
            </Label>
            <Input
              type="number"
              value={quotaValue}
              onChange={(e) => setQuotaValue(e.target.value)}
              placeholder="0 = unlimited"
              autoFocus
            />
            <p className="text-xs text-muted-foreground">
              Set to 0 for unlimited storage
            </p>
          </div>
          <DialogFooter>
            <Button variant="outline" onClick={() => setQuotaTarget(null)}>
              Cancel
            </Button>
            <Button
              onClick={() => quotaMut.mutate()}
              disabled={quotaMut.isPending}
              className="bg-pink-500 hover:bg-pink-600"
            >
              {quotaMut.isPending ? "Updating..." : "Update Quota"}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* Delete Confirmation */}
      <ConfirmDialog
        open={!!deleteTarget}
        onOpenChange={() => setDeleteTarget(null)}
        title="Delete Email Account"
        description={`This will permanently delete "${deleteTarget?.address}@${domainName}" and all its mail data. This action cannot be undone.`}
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

function ForwardersSection({
  domainId,
  domainName,
  forwarders,
}: {
  domainId: number;
  domainName: string;
  forwarders: EmailForwarder[];
}) {
  const queryClient = useQueryClient();
  const [showCreate, setShowCreate] = useState(false);
  const [newSource, setNewSource] = useState("");
  const [newDest, setNewDest] = useState("");
  const [deleteTarget, setDeleteTarget] = useState<EmailForwarder | null>(null);

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

  return (
    <>
      <Card>
        <CardHeader>
          <div className="flex items-center justify-between">
            <div>
              <CardTitle className="flex items-center gap-2">
                <Forward className="h-5 w-5 text-blue-500" />
                Email Forwarders
              </CardTitle>
              <CardDescription>
                Automatically forward incoming mail to external addresses
              </CardDescription>
            </div>
            <Button
              size="sm"
              onClick={() => {
                setShowCreate(true);
                setNewSource("");
                setNewDest("");
              }}
              className="bg-pink-500 hover:bg-pink-600"
            >
              <Plus className="h-4 w-4 mr-1" />
              New Forwarder
            </Button>
          </div>
        </CardHeader>
        <CardContent>
          {forwarders.length === 0 ? (
            <div className="py-12 text-center">
              <div className="flex h-16 w-16 mx-auto items-center justify-center rounded-full bg-muted mb-4">
                <Forward className="h-8 w-8 text-muted-foreground" />
              </div>
              <h3 className="text-sm font-medium mb-1">No forwarders configured</h3>
              <p className="text-sm text-muted-foreground max-w-sm mx-auto">
                Create forwarding rules to redirect incoming mail to other addresses.
              </p>
            </div>
          ) : (
            <div className="border rounded-lg overflow-hidden">
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead>Source Address</TableHead>
                    <TableHead className="w-[40px]" />
                    <TableHead>Destination</TableHead>
                    <TableHead className="w-[60px]" />
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {forwarders.map((fwd) => (
                    <TableRow key={fwd.id}>
                      <TableCell>
                        <div className="flex items-center gap-2">
                          <Mail className="h-4 w-4 text-muted-foreground" />
                          <span className="font-medium text-sm">
                            {fwd.source_address}@{domainName}
                          </span>
                        </div>
                      </TableCell>
                      <TableCell>
                        <ArrowRight className="h-4 w-4 text-muted-foreground" />
                      </TableCell>
                      <TableCell>
                        <span className="text-sm text-muted-foreground">
                          {fwd.destination}
                        </span>
                      </TableCell>
                      <TableCell className="text-right">
                        <Button
                          size="icon"
                          variant="ghost"
                          className="h-8 w-8 text-destructive hover:text-destructive"
                          onClick={() => setDeleteTarget(fwd)}
                        >
                          <Trash2 className="h-3.5 w-3.5" />
                        </Button>
                      </TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
            </div>
          )}
        </CardContent>
      </Card>

      {/* Create Forwarder Dialog */}
      <Dialog open={showCreate} onOpenChange={setShowCreate}>
        <DialogContent className="sm:max-w-md">
          <DialogHeader>
            <DialogTitle>Create Email Forwarder</DialogTitle>
            <DialogDescription>
              Forward incoming mail to another address
            </DialogDescription>
          </DialogHeader>
          <div className="space-y-4">
            <div className="space-y-2">
              <Label>Source Address</Label>
              <div className="flex items-center gap-0">
                <Input
                  value={newSource}
                  onChange={(e) => setNewSource(e.target.value)}
                  placeholder="sales"
                  className="rounded-r-none border-r-0"
                  autoFocus
                />
                <div className="flex h-9 items-center rounded-r-md border bg-muted px-3 text-sm text-muted-foreground">
                  @{domainName}
                </div>
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
              {createMut.isPending && <Loader2 className="h-4 w-4 mr-1 animate-spin" />}
              {createMut.isPending ? "Creating..." : "Create Forwarder"}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* Delete Confirmation */}
      <ConfirmDialog
        open={!!deleteTarget}
        onOpenChange={() => setDeleteTarget(null)}
        title="Delete Forwarder"
        description={`Stop forwarding mail from "${deleteTarget?.source_address}@${domainName}"?`}
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
  const [newListType, setNewListType] = useState<"whitelist" | "blacklist">("blacklist");

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

  if (isLoading) return <Skeleton className="h-48 w-full" />;

  const whitelist = whitelistData?.data ?? [];
  const blacklist = blacklistData?.data ?? [];

  return (
    <div className="space-y-6">
      {/* Main Settings */}
      <Card>
        <CardHeader>
          <div className="flex items-center justify-between">
            <div>
              <CardTitle className="flex items-center gap-2">
                <Shield className="h-5 w-5 text-pink-500" />
                Spam Protection
              </CardTitle>
              <CardDescription>
                SpamAssassin filters incoming messages and scores them for spam likelihood
              </CardDescription>
            </div>
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
        </CardHeader>
        {settings?.enabled && (
          <CardContent className="space-y-6">
            <div className="grid grid-cols-1 sm:grid-cols-2 gap-6">
              <div className="space-y-3">
                <Label>Spam Score Threshold</Label>
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
                <p className="text-xs text-muted-foreground">
                  Messages scoring above this threshold are treated as spam.
                  Lower values are stricter. Default: 5.0
                </p>
              </div>
              <div className="space-y-3">
                <Label>Action on Spam</Label>
                <Select
                  value={settings.action}
                  onValueChange={(v) =>
                    v &&
                    updateMut.mutate({
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
                    <SelectItem value="mark">Mark as spam (add headers)</SelectItem>
                    <SelectItem value="junk">Move to Junk folder</SelectItem>
                    <SelectItem value="delete">Delete silently</SelectItem>
                  </SelectContent>
                </Select>
                <p className="text-xs text-muted-foreground">
                  What to do when a message is identified as spam
                </p>
              </div>
            </div>
          </CardContent>
        )}
      </Card>

      {/* Whitelist & Blacklist */}
      {settings?.enabled && (
        <Card>
          <CardHeader>
            <CardTitle className="text-base">Whitelist & Blacklist</CardTitle>
            <CardDescription>
              Always allow or always block specific email addresses or domains
            </CardDescription>
          </CardHeader>
          <CardContent className="space-y-4">
            {/* Add entry */}
            <div className="flex items-center gap-2">
              <Select
                value={newListType}
                onValueChange={(v) => v && setNewListType(v as "whitelist" | "blacklist")}
              >
                <SelectTrigger className="w-[130px]">
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
                onKeyDown={(e) => {
                  if (e.key === "Enter" && newEntry) addMut.mutate();
                }}
              />
              <Button
                size="sm"
                onClick={() => addMut.mutate()}
                disabled={!newEntry || addMut.isPending}
                className="bg-pink-500 hover:bg-pink-600"
              >
                <Plus className="h-4 w-4 mr-1" />
                Add
              </Button>
            </div>

            {/* Lists */}
            {whitelist.length === 0 && blacklist.length === 0 && (
              <p className="text-sm text-muted-foreground text-center py-4">
                No entries yet. Add email addresses or patterns above.
              </p>
            )}

            {whitelist.length > 0 && (
              <div>
                <h4 className="text-xs font-semibold text-muted-foreground uppercase tracking-wider mb-2">
                  Whitelist ({whitelist.length})
                </h4>
                <div className="border rounded-lg divide-y">
                  {whitelist.map((e) => (
                    <div
                      key={e.id}
                      className="flex items-center justify-between px-3 py-2"
                    >
                      <div className="flex items-center gap-2">
                        <CheckCircle2 className="h-4 w-4 text-green-500" />
                        <span className="text-sm">{e.entry}</span>
                      </div>
                      <Button
                        size="icon"
                        variant="ghost"
                        className="h-7 w-7 text-muted-foreground hover:text-destructive"
                        onClick={() => deleteMut.mutate(e)}
                      >
                        <Trash2 className="h-3.5 w-3.5" />
                      </Button>
                    </div>
                  ))}
                </div>
              </div>
            )}

            {blacklist.length > 0 && (
              <div>
                <h4 className="text-xs font-semibold text-muted-foreground uppercase tracking-wider mb-2">
                  Blacklist ({blacklist.length})
                </h4>
                <div className="border rounded-lg divide-y">
                  {blacklist.map((e) => (
                    <div
                      key={e.id}
                      className="flex items-center justify-between px-3 py-2"
                    >
                      <div className="flex items-center gap-2">
                        <Ban className="h-4 w-4 text-red-500" />
                        <span className="text-sm">{e.entry}</span>
                      </div>
                      <Button
                        size="icon"
                        variant="ghost"
                        className="h-7 w-7 text-muted-foreground hover:text-destructive"
                        onClick={() => deleteMut.mutate(e)}
                      >
                        <Trash2 className="h-3.5 w-3.5" />
                      </Button>
                    </div>
                  ))}
                </div>
              </div>
            )}
          </CardContent>
        </Card>
      )}
    </div>
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
      <CardHeader>
        <div className="flex items-center justify-between">
          <div>
            <CardTitle className="flex items-center gap-2 text-base">
              <Wifi className="h-4 w-4 text-pink-500" />
              Mail Autodiscovery
            </CardTitle>
            <CardDescription>
              {allConfigured
                ? "Email clients will auto-detect server settings"
                : "Configure auto-detection for Thunderbird, Outlook, and other clients"}
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
          {[
            { key: "srv_records", label: "SRV Records", ok: data?.srv_records },
            { key: "autoconfig", label: "Thunderbird", ok: data?.autoconfig },
            { key: "autodiscover", label: "Outlook", ok: data?.autodiscover },
          ].map((item) => (
            <div
              key={item.key}
              className="flex items-center gap-2 p-3 rounded-lg border"
            >
              {item.ok ? (
                <CheckCircle2 className="h-4 w-4 text-green-500 shrink-0" />
              ) : (
                <XCircle className="h-4 w-4 text-muted-foreground shrink-0" />
              )}
              <span className="text-sm">{item.label}</span>
            </div>
          ))}
        </div>
      </CardContent>
    </Card>
  );
}

// ─── Mail SSL ─────────────────────────────────────

function MailSSLSection({ domainId }: { domainId: number }) {
  const queryClient = useQueryClient();

  const { data, isLoading } = useQuery({
    queryKey: ["mail-ssl", domainId],
    queryFn: () => getMailSSLStatus(domainId),
    enabled: !!domainId,
    retry: false,
  });

  const configureMut = useMutation({
    mutationFn: () => configureMailSSL(domainId),
    onSuccess: () => {
      toast.success("Mail SSL/TLS configured for Postfix and Dovecot");
      queryClient.invalidateQueries({ queryKey: ["mail-ssl", domainId] });
    },
    onError: (err: AxiosError<APIError>) => {
      toast.error(err.response?.data?.error?.message ?? "Failed to configure mail SSL");
    },
  });

  if (isLoading) return <Skeleton className="h-24 w-full" />;

  return (
    <Card>
      <CardHeader>
        <div className="flex items-center justify-between">
          <div>
            <CardTitle className="flex items-center gap-2 text-base">
              <Lock className="h-4 w-4 text-pink-500" />
              Mail SSL/TLS
            </CardTitle>
            <CardDescription>
              {data?.mail_ssl
                ? "Encrypted connections are active for IMAP and SMTP"
                : data?.has_ssl_cert
                  ? "SSL certificate available — enable it for mail services"
                  : "Issue an SSL certificate first on the SSL/TLS tab"}
            </CardDescription>
          </div>
          {data?.has_ssl_cert && !data?.mail_ssl && (
            <Button
              size="sm"
              onClick={() => configureMut.mutate()}
              disabled={configureMut.isPending}
              className="bg-pink-500 hover:bg-pink-600"
            >
              {configureMut.isPending ? (
                <Loader2 className="h-4 w-4 mr-1 animate-spin" />
              ) : (
                <Lock className="h-4 w-4 mr-1" />
              )}
              Enable SSL
            </Button>
          )}
        </div>
      </CardHeader>
      <CardContent>
        <div className="grid grid-cols-2 gap-3">
          {[
            { label: "SSL Certificate", ok: data?.has_ssl_cert },
            { label: "Mail TLS", ok: data?.mail_ssl },
          ].map((item) => (
            <div
              key={item.label}
              className="flex items-center gap-2 p-3 rounded-lg border"
            >
              {item.ok ? (
                <CheckCircle2 className="h-4 w-4 text-green-500 shrink-0" />
              ) : (
                <XCircle className="h-4 w-4 text-muted-foreground shrink-0" />
              )}
              <span className="text-sm">{item.label}</span>
            </div>
          ))}
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
      <CardHeader>
        <div className="flex items-center justify-between">
          <div>
            <CardTitle className="flex items-center gap-2 text-base">
              {allExist ? (
                <ShieldCheck className="h-4 w-4 text-green-500" />
              ) : (
                <ShieldAlert className="h-4 w-4 text-yellow-500" />
              )}
              Email DNS Records
            </CardTitle>
            <CardDescription>
              {allExist
                ? "All recommended DNS records are configured (SPF, DKIM, DMARC)"
                : `${missingCount} missing record${missingCount !== 1 ? "s" : ""} — deliverability may be affected`}
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
          <div className="py-6 text-center">
            <AlertCircle className="h-8 w-8 mx-auto text-muted-foreground mb-2" />
            <p className="text-sm text-muted-foreground">
              Create at least one email account to see DNS recommendations.
            </p>
          </div>
        ) : (
          <div className="border rounded-lg divide-y">
            {records.map((rec) => (
              <div key={rec.label} className="p-3 space-y-1.5">
                <div className="flex items-center justify-between">
                  <div className="flex items-center gap-2">
                    <Badge
                      variant="outline"
                      className="text-xs font-mono px-2"
                    >
                      {rec.label}
                    </Badge>
                    <span className="text-xs text-muted-foreground">
                      {rec.name}
                    </span>
                  </div>
                  {rec.exists ? (
                    <Badge className="bg-green-500/10 text-green-500 border-green-500/20 gap-1">
                      <Check className="h-3 w-3" />
                      Configured
                    </Badge>
                  ) : (
                    <Badge className="bg-yellow-500/10 text-yellow-500 border-yellow-500/20">
                      Missing
                    </Badge>
                  )}
                </div>
                <p className="text-xs font-mono text-muted-foreground break-all leading-relaxed">
                  {rec.value.length > 140 ? rec.value.slice(0, 140) + "..." : rec.value}
                </p>
              </div>
            ))}
          </div>
        )}
      </CardContent>
    </Card>
  );
}
