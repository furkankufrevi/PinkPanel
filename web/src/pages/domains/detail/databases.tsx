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
  listDatabases,
  createDatabase,
  deleteDatabase,
} from "@/api/databases";
import type { Database } from "@/types/database";
import type { AxiosError } from "axios";
import type { APIError } from "@/types/api";
import {
  Database as DatabaseIcon,
  Plus,
  Trash2,
  HardDrive,
} from "lucide-react";

function formatSize(bytes: number): string {
  if (bytes === 0) return "0 B";
  const units = ["B", "KB", "MB", "GB"];
  const i = Math.floor(Math.log(bytes) / Math.log(1024));
  return `${(bytes / Math.pow(1024, i)).toFixed(i > 0 ? 1 : 0)} ${units[i]}`;
}

export function DomainDatabases() {
  const { id } = useParams<{ id: string }>();
  const domainId = Number(id);
  const queryClient = useQueryClient();

  const [showCreate, setShowCreate] = useState(false);
  const [newDbName, setNewDbName] = useState("");
  const [deleteDb, setDeleteDb] = useState<Database | null>(null);

  const { data, isLoading } = useQuery({
    queryKey: ["databases", domainId],
    queryFn: () => listDatabases(domainId),
    enabled: !!domainId,
  });

  const createMutation = useMutation({
    mutationFn: () => createDatabase({ name: newDbName, domain_id: domainId }),
    onSuccess: () => {
      toast.success("Database created");
      setShowCreate(false);
      setNewDbName("");
      queryClient.invalidateQueries({ queryKey: ["databases"] });
    },
    onError: (err: AxiosError<APIError>) => {
      toast.error(err.response?.data?.error?.message ?? "Failed to create database");
    },
  });

  const deleteMutation = useMutation({
    mutationFn: () => deleteDatabase(deleteDb!.id),
    onSuccess: () => {
      toast.success("Database deleted");
      setDeleteDb(null);
      queryClient.invalidateQueries({ queryKey: ["databases"] });
    },
    onError: (err: AxiosError<APIError>) => {
      toast.error(err.response?.data?.error?.message ?? "Failed to delete database");
    },
  });

  if (isLoading) {
    return <Skeleton className="h-48 w-full max-w-2xl" />;
  }

  const databases = data?.data ?? [];

  return (
    <div className="space-y-4 max-w-2xl">
      <div className="flex items-center justify-between">
        <h3 className="text-lg font-medium">Databases</h3>
        <Button
          size="sm"
          onClick={() => { setShowCreate(true); setNewDbName(""); }}
          className="bg-pink-500 hover:bg-pink-600"
        >
          <Plus className="h-4 w-4 mr-1" />
          New Database
        </Button>
      </div>

      {databases.length === 0 ? (
        <Card>
          <CardContent className="py-8 text-center">
            <DatabaseIcon className="h-10 w-10 mx-auto text-muted-foreground mb-3" />
            <p className="text-sm text-muted-foreground">No databases for this domain</p>
          </CardContent>
        </Card>
      ) : (
        <div className="space-y-2">
          {databases.map((db) => (
            <Card key={db.id}>
              <CardHeader className="py-3">
                <div className="flex items-center justify-between">
                  <CardTitle className="text-sm flex items-center gap-2">
                    <DatabaseIcon className="h-4 w-4 text-pink-500" />
                    {db.name}
                  </CardTitle>
                  <div className="flex items-center gap-3">
                    <CardDescription className="flex items-center gap-1 text-xs">
                      <HardDrive className="h-3 w-3" />
                      {formatSize(db.size_bytes)}
                    </CardDescription>
                    <Badge variant="outline" className="text-xs">{db.type}</Badge>
                    <Button
                      size="icon"
                      variant="ghost"
                      className="h-7 w-7 text-destructive"
                      onClick={() => setDeleteDb(db)}
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
            <DialogTitle>New Database</DialogTitle>
            <DialogDescription>Create a MySQL database for this domain</DialogDescription>
          </DialogHeader>
          <div className="space-y-2">
            <Label>Database Name</Label>
            <Input
              value={newDbName}
              onChange={(e) => setNewDbName(e.target.value)}
              placeholder="my_database"
              autoFocus
            />
          </div>
          <DialogFooter>
            <Button variant="outline" onClick={() => setShowCreate(false)}>Cancel</Button>
            <Button
              onClick={() => createMutation.mutate()}
              disabled={!newDbName || createMutation.isPending}
              className="bg-pink-500 hover:bg-pink-600"
            >
              {createMutation.isPending ? "Creating..." : "Create"}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* Delete Confirmation */}
      <ConfirmDialog
        open={!!deleteDb}
        onOpenChange={() => setDeleteDb(null)}
        title="Delete Database"
        description={`Permanently delete "${deleteDb?.name}" and all data?`}
        confirmText="Delete"
        typeToConfirm={deleteDb?.name}
        destructive
        loading={deleteMutation.isPending}
        onConfirm={() => deleteMutation.mutate()}
      />
    </div>
  );
}
