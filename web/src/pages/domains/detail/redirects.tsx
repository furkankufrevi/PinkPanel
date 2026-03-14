import { useState } from "react";
import { useParams } from "react-router-dom";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { Card, CardContent, CardHeader } from "@/components/ui/card";
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
  listRedirects,
  createRedirect,
  updateRedirect,
  deleteRedirect,
} from "@/api/redirect";
import type { Redirect } from "@/types/redirect";
import type { AxiosError } from "axios";
import type { APIError } from "@/types/api";
import { ArrowRight, Plus, Pencil, Trash2, ExternalLink } from "lucide-react";

export function DomainRedirects() {
  const { id } = useParams<{ id: string }>();
  const domainId = Number(id);
  const queryClient = useQueryClient();

  const [showCreate, setShowCreate] = useState(false);
  const [editRedirect, setEditRedirect] = useState<Redirect | null>(null);
  const [deleteTarget, setDeleteTarget] = useState<Redirect | null>(null);

  // Form state
  const [sourcePath, setSourcePath] = useState("");
  const [targetUrl, setTargetUrl] = useState("");
  const [redirectType, setRedirectType] = useState("301");

  const { data, isLoading } = useQuery({
    queryKey: ["redirects", domainId],
    queryFn: () => listRedirects(domainId),
    enabled: !!domainId,
  });

  const createMutation = useMutation({
    mutationFn: () =>
      createRedirect(domainId, {
        source_path: sourcePath,
        target_url: targetUrl,
        redirect_type: Number(redirectType),
      }),
    onSuccess: () => {
      toast.success("Redirect created");
      setShowCreate(false);
      resetForm();
      queryClient.invalidateQueries({ queryKey: ["redirects", domainId] });
    },
    onError: (err: AxiosError<APIError>) => {
      toast.error(err.response?.data?.error?.message ?? "Failed to create redirect");
    },
  });

  const updateMutation = useMutation({
    mutationFn: (req: { id: number; source_path?: string; target_url?: string; redirect_type?: number; enabled?: boolean }) =>
      updateRedirect(req.id, req),
    onSuccess: () => {
      toast.success("Redirect updated");
      setEditRedirect(null);
      resetForm();
      queryClient.invalidateQueries({ queryKey: ["redirects", domainId] });
    },
    onError: (err: AxiosError<APIError>) => {
      toast.error(err.response?.data?.error?.message ?? "Failed to update redirect");
    },
  });

  const deleteMutation = useMutation({
    mutationFn: () => deleteRedirect(deleteTarget!.id),
    onSuccess: () => {
      toast.success("Redirect deleted");
      setDeleteTarget(null);
      queryClient.invalidateQueries({ queryKey: ["redirects", domainId] });
    },
    onError: (err: AxiosError<APIError>) => {
      toast.error(err.response?.data?.error?.message ?? "Failed to delete redirect");
    },
  });

  function resetForm() {
    setSourcePath("");
    setTargetUrl("");
    setRedirectType("301");
  }

  function openEdit(r: Redirect) {
    setEditRedirect(r);
    setSourcePath(r.source_path);
    setTargetUrl(r.target_url);
    setRedirectType(String(r.redirect_type));
  }

  function handleToggleEnabled(r: Redirect) {
    updateMutation.mutate({ id: r.id, enabled: !r.enabled });
  }

  if (isLoading) {
    return <Skeleton className="h-48 w-full max-w-3xl" />;
  }

  const redirects = data?.data ?? [];

  return (
    <div className="space-y-4 max-w-3xl">
      <div className="flex items-center justify-between">
        <h3 className="text-lg font-medium">Redirects</h3>
        <Button
          size="sm"
          onClick={() => {
            resetForm();
            setShowCreate(true);
          }}
          className="bg-pink-500 hover:bg-pink-600"
        >
          <Plus className="h-4 w-4 mr-1" />
          Add Redirect
        </Button>
      </div>

      {redirects.length === 0 ? (
        <Card>
          <CardContent className="py-8 text-center">
            <ExternalLink className="h-10 w-10 mx-auto text-muted-foreground mb-3" />
            <p className="text-sm text-muted-foreground">
              No redirects configured for this domain
            </p>
          </CardContent>
        </Card>
      ) : (
        <div className="space-y-2">
          {redirects.map((r) => (
            <Card key={r.id}>
              <CardHeader className="py-3">
                <div className="flex items-center justify-between gap-2">
                  <div className="flex items-center gap-2 min-w-0 flex-1">
                    <Badge
                      variant="outline"
                      className={
                        r.redirect_type === 301
                          ? "text-blue-500 border-blue-500/30 shrink-0"
                          : "text-amber-500 border-amber-500/30 shrink-0"
                      }
                    >
                      {r.redirect_type}
                    </Badge>
                    <code className="text-sm truncate">{r.source_path}</code>
                    <ArrowRight className="h-3 w-3 text-muted-foreground shrink-0" />
                    <span className="text-sm text-muted-foreground truncate">
                      {r.target_url}
                    </span>
                  </div>
                  <div className="flex items-center gap-2 shrink-0">
                    <Switch
                      checked={r.enabled}
                      onCheckedChange={() => handleToggleEnabled(r)}
                    />
                    <Button
                      size="icon"
                      variant="ghost"
                      className="h-7 w-7"
                      onClick={() => openEdit(r)}
                    >
                      <Pencil className="h-3 w-3" />
                    </Button>
                    <Button
                      size="icon"
                      variant="ghost"
                      className="h-7 w-7 text-destructive"
                      onClick={() => setDeleteTarget(r)}
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
            <DialogTitle>Add Redirect</DialogTitle>
            <DialogDescription>
              Create a URL redirect for this domain
            </DialogDescription>
          </DialogHeader>
          <div className="space-y-4">
            <div className="space-y-2">
              <Label>Source Path</Label>
              <Input
                value={sourcePath}
                onChange={(e) => setSourcePath(e.target.value)}
                placeholder="/old-page"
                autoFocus
              />
              <p className="text-xs text-muted-foreground">
                Must start with /
              </p>
            </div>
            <div className="space-y-2">
              <Label>Target URL</Label>
              <Input
                value={targetUrl}
                onChange={(e) => setTargetUrl(e.target.value)}
                placeholder="https://example.com/new-page"
              />
            </div>
            <div className="space-y-2">
              <Label>Redirect Type</Label>
              <Select value={redirectType} onValueChange={(v) => v && setRedirectType(v)}>
                <SelectTrigger>
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="301">301 — Permanent</SelectItem>
                  <SelectItem value="302">302 — Temporary</SelectItem>
                </SelectContent>
              </Select>
            </div>
          </div>
          <DialogFooter>
            <Button variant="outline" onClick={() => setShowCreate(false)}>
              Cancel
            </Button>
            <Button
              onClick={() => createMutation.mutate()}
              disabled={!sourcePath || !targetUrl || createMutation.isPending}
              className="bg-pink-500 hover:bg-pink-600"
            >
              {createMutation.isPending ? "Creating..." : "Create"}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* Edit Dialog */}
      <Dialog open={!!editRedirect} onOpenChange={() => setEditRedirect(null)}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Edit Redirect</DialogTitle>
            <DialogDescription>
              Update the redirect configuration
            </DialogDescription>
          </DialogHeader>
          <div className="space-y-4">
            <div className="space-y-2">
              <Label>Source Path</Label>
              <Input
                value={sourcePath}
                onChange={(e) => setSourcePath(e.target.value)}
                placeholder="/old-page"
                autoFocus
              />
            </div>
            <div className="space-y-2">
              <Label>Target URL</Label>
              <Input
                value={targetUrl}
                onChange={(e) => setTargetUrl(e.target.value)}
                placeholder="https://example.com/new-page"
              />
            </div>
            <div className="space-y-2">
              <Label>Redirect Type</Label>
              <Select value={redirectType} onValueChange={(v) => v && setRedirectType(v)}>
                <SelectTrigger>
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="301">301 — Permanent</SelectItem>
                  <SelectItem value="302">302 — Temporary</SelectItem>
                </SelectContent>
              </Select>
            </div>
          </div>
          <DialogFooter>
            <Button variant="outline" onClick={() => setEditRedirect(null)}>
              Cancel
            </Button>
            <Button
              onClick={() =>
                editRedirect &&
                updateMutation.mutate({
                  id: editRedirect.id,
                  source_path: sourcePath,
                  target_url: targetUrl,
                  redirect_type: Number(redirectType),
                })
              }
              disabled={!sourcePath || !targetUrl || updateMutation.isPending}
              className="bg-pink-500 hover:bg-pink-600"
            >
              {updateMutation.isPending ? "Saving..." : "Save"}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* Delete Confirmation */}
      <ConfirmDialog
        open={!!deleteTarget}
        onOpenChange={() => setDeleteTarget(null)}
        title="Delete Redirect"
        description={`Delete redirect for "${deleteTarget?.source_path}"?`}
        confirmText="Delete"
        destructive
        loading={deleteMutation.isPending}
        onConfirm={() => deleteMutation.mutate()}
      />
    </div>
  );
}
