import { useState } from "react";
import { useParams } from "react-router-dom";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import {
  Card,
  CardContent,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
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
  listFTPAccounts,
  createFTPAccount,
  deleteFTPAccount,
} from "@/api/ftp";
import type { FTPAccount } from "@/types/ftp";
import type { AxiosError } from "axios";
import type { APIError } from "@/types/api";
import { Upload, Plus, Trash2, User, FolderOpen } from "lucide-react";

export function DomainFTP() {
  const { id } = useParams<{ id: string }>();
  const domainId = Number(id);
  const queryClient = useQueryClient();

  const [showCreate, setShowCreate] = useState(false);
  const [newUsername, setNewUsername] = useState("");
  const [newPassword, setNewPassword] = useState("");
  const [newQuota, setNewQuota] = useState("0");
  const [deleteAccount, setDeleteAccount] = useState<FTPAccount | null>(null);

  const { data, isLoading } = useQuery({
    queryKey: ["ftp-accounts", domainId],
    queryFn: () => listFTPAccounts(domainId),
    enabled: !!domainId,
  });

  const createMutation = useMutation({
    mutationFn: () =>
      createFTPAccount({
        domain_id: domainId,
        username: newUsername,
        password: newPassword,
        quota_mb: Number(newQuota) || 0,
      }),
    onSuccess: () => {
      toast.success("FTP account created");
      setShowCreate(false);
      setNewUsername("");
      setNewPassword("");
      setNewQuota("0");
      queryClient.invalidateQueries({ queryKey: ["ftp-accounts"] });
    },
    onError: (err: AxiosError<APIError>) => {
      toast.error(
        err.response?.data?.error?.message ?? "Failed to create FTP account"
      );
    },
  });

  const deleteMutation = useMutation({
    mutationFn: () => deleteFTPAccount(deleteAccount!.id),
    onSuccess: () => {
      toast.success("FTP account deleted");
      setDeleteAccount(null);
      queryClient.invalidateQueries({ queryKey: ["ftp-accounts"] });
    },
    onError: (err: AxiosError<APIError>) => {
      toast.error(
        err.response?.data?.error?.message ?? "Failed to delete FTP account"
      );
    },
  });

  if (isLoading) {
    return <Skeleton className="h-48 w-full max-w-2xl" />;
  }

  const accounts = data?.data ?? [];

  return (
    <div className="space-y-4 max-w-2xl">
      <div className="flex items-center justify-between">
        <h3 className="text-lg font-medium">FTP Accounts</h3>
        <Button
          size="sm"
          onClick={() => {
            setShowCreate(true);
            setNewUsername("");
            setNewPassword("");
            setNewQuota("0");
          }}
          className="bg-pink-500 hover:bg-pink-600"
        >
          <Plus className="h-4 w-4 mr-1" />
          New Account
        </Button>
      </div>

      {accounts.length === 0 ? (
        <Card>
          <CardContent className="py-8 text-center">
            <Upload className="h-10 w-10 mx-auto text-muted-foreground mb-3" />
            <p className="text-sm text-muted-foreground">
              No FTP accounts for this domain
            </p>
          </CardContent>
        </Card>
      ) : (
        <div className="space-y-2">
          {accounts.map((account) => (
            <Card key={account.id}>
              <CardHeader className="py-3">
                <div className="flex items-center justify-between">
                  <CardTitle className="text-sm flex items-center gap-2">
                    <User className="h-4 w-4 text-pink-500" />
                    {account.username}
                  </CardTitle>
                  <div className="flex items-center gap-3">
                    <span className="flex items-center gap-1 text-xs text-muted-foreground">
                      <FolderOpen className="h-3 w-3" />
                      {account.home_dir}
                    </span>
                    {account.quota_mb > 0 && (
                      <Badge variant="outline" className="text-xs">
                        {account.quota_mb} MB
                      </Badge>
                    )}
                    {account.quota_mb === 0 && (
                      <Badge variant="outline" className="text-xs">
                        Unlimited
                      </Badge>
                    )}
                    <Button
                      size="icon"
                      variant="ghost"
                      className="h-7 w-7 text-destructive"
                      onClick={() => setDeleteAccount(account)}
                    >
                      <Trash2 className="h-3 w-3" />
                    </Button>
                  </div>
                </div>
              </CardHeader>
            </Card>
          ))}
        </div>
      )}

      {/* Create Dialog */}
      <Dialog open={showCreate} onOpenChange={setShowCreate}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>New FTP Account</DialogTitle>
            <DialogDescription>
              Create an FTP account for this domain
            </DialogDescription>
          </DialogHeader>
          <div className="space-y-4">
            <div className="space-y-2">
              <Label>Username</Label>
              <Input
                value={newUsername}
                onChange={(e) => setNewUsername(e.target.value)}
                placeholder="ftp_user"
                autoFocus
              />
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
              onClick={() => createMutation.mutate()}
              disabled={
                !newUsername || !newPassword || createMutation.isPending
              }
              className="bg-pink-500 hover:bg-pink-600"
            >
              {createMutation.isPending ? "Creating..." : "Create"}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* Delete Confirmation */}
      <ConfirmDialog
        open={!!deleteAccount}
        onOpenChange={() => setDeleteAccount(null)}
        title="Delete FTP Account"
        description={`Permanently delete FTP account "${deleteAccount?.username}"?`}
        confirmText="Delete"
        typeToConfirm={deleteAccount?.username}
        destructive
        loading={deleteMutation.isPending}
        onConfirm={() => deleteMutation.mutate()}
      />
    </div>
  );
}
