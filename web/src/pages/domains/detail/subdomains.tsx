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
  listSubdomains,
  createSubdomain,
  deleteSubdomain,
} from "@/api/subdomains";
import { useQuery as useDomainQuery } from "@tanstack/react-query";
import api from "@/api/client";
import type { Subdomain } from "@/types/subdomain";
import type { AxiosError } from "axios";
import type { APIError } from "@/types/api";
import { Globe, Plus, Trash2, FolderOpen } from "lucide-react";

export function DomainSubdomains() {
  const { id } = useParams<{ id: string }>();
  const domainId = Number(id);
  const queryClient = useQueryClient();

  const [showCreate, setShowCreate] = useState(false);
  const [newName, setNewName] = useState("");
  const [deleteTarget, setDeleteTarget] = useState<Subdomain | null>(null);

  // Get parent domain for display
  const { data: domainData } = useDomainQuery({
    queryKey: ["domain", domainId],
    queryFn: async () => {
      const { data } = await api.get(`/domains/${domainId}`);
      return data;
    },
    enabled: !!domainId,
  });
  const domainName = domainData?.name ?? "";

  const { data, isLoading } = useQuery({
    queryKey: ["subdomains", domainId],
    queryFn: () => listSubdomains(domainId),
    enabled: !!domainId,
  });

  const createMutation = useMutation({
    mutationFn: () => createSubdomain(domainId, newName),
    onSuccess: () => {
      toast.success("Subdomain created");
      setShowCreate(false);
      setNewName("");
      queryClient.invalidateQueries({ queryKey: ["subdomains", domainId] });
    },
    onError: (err: AxiosError<APIError>) => {
      toast.error(
        err.response?.data?.error?.message ?? "Failed to create subdomain"
      );
    },
  });

  const deleteMutation = useMutation({
    mutationFn: () => deleteSubdomain(domainId, deleteTarget!.id),
    onSuccess: () => {
      toast.success("Subdomain deleted");
      setDeleteTarget(null);
      queryClient.invalidateQueries({ queryKey: ["subdomains", domainId] });
    },
    onError: (err: AxiosError<APIError>) => {
      toast.error(
        err.response?.data?.error?.message ?? "Failed to delete subdomain"
      );
    },
  });

  if (isLoading) {
    return <Skeleton className="h-48 w-full max-w-2xl" />;
  }

  const subdomains = data?.data ?? [];

  return (
    <div className="space-y-4 max-w-2xl">
      <div className="flex items-center justify-between">
        <h3 className="text-lg font-medium">Subdomains</h3>
        <Button
          size="sm"
          onClick={() => {
            setShowCreate(true);
            setNewName("");
          }}
          className="bg-pink-500 hover:bg-pink-600"
        >
          <Plus className="h-4 w-4 mr-1" />
          New Subdomain
        </Button>
      </div>

      {subdomains.length === 0 ? (
        <Card>
          <CardContent className="py-8 text-center">
            <Globe className="h-10 w-10 mx-auto text-muted-foreground mb-3" />
            <p className="text-sm text-muted-foreground">
              No subdomains for this domain
            </p>
          </CardContent>
        </Card>
      ) : (
        <div className="space-y-2">
          {subdomains.map((sub) => (
            <Card key={sub.id}>
              <CardHeader className="py-3">
                <div className="flex items-center justify-between">
                  <div className="space-y-1">
                    <CardTitle className="text-sm flex items-center gap-2">
                      <Globe className="h-4 w-4 text-pink-500" />
                      {sub.name}.{domainName}
                    </CardTitle>
                    <CardDescription className="flex items-center gap-1 text-xs">
                      <FolderOpen className="h-3 w-3" />
                      {sub.document_root}
                    </CardDescription>
                  </div>
                  <Button
                    size="icon"
                    variant="ghost"
                    className="h-7 w-7 text-destructive"
                    onClick={() => setDeleteTarget(sub)}
                  >
                    <Trash2 className="h-3 w-3" />
                  </Button>
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
            <DialogTitle>New Subdomain</DialogTitle>
            <DialogDescription>
              Create a subdomain for {domainName}
            </DialogDescription>
          </DialogHeader>
          <div className="space-y-2">
            <Label>Subdomain Name</Label>
            <div className="flex items-center gap-2">
              <Input
                value={newName}
                onChange={(e) => setNewName(e.target.value)}
                placeholder="blog"
                autoFocus
              />
              <span className="text-sm text-muted-foreground whitespace-nowrap">
                .{domainName}
              </span>
            </div>
            <p className="text-xs text-muted-foreground">
              Alphanumeric characters and hyphens only
            </p>
          </div>
          <DialogFooter>
            <Button variant="outline" onClick={() => setShowCreate(false)}>
              Cancel
            </Button>
            <Button
              onClick={() => createMutation.mutate()}
              disabled={!newName || createMutation.isPending}
              className="bg-pink-500 hover:bg-pink-600"
            >
              {createMutation.isPending ? "Creating..." : "Create"}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* Delete Confirmation */}
      <ConfirmDialog
        open={!!deleteTarget}
        onOpenChange={() => setDeleteTarget(null)}
        title="Delete Subdomain"
        description={`Permanently delete subdomain "${deleteTarget?.name}.${domainName}"? This will remove the NGINX configuration.`}
        confirmText="Delete"
        destructive
        loading={deleteMutation.isPending}
        onConfirm={() => deleteMutation.mutate()}
      />
    </div>
  );
}
