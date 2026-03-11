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
import { Label } from "@/components/ui/label";
import { Badge } from "@/components/ui/badge";
import { Skeleton } from "@/components/ui/skeleton";
import {
  Sheet,
  SheetContent,
  SheetDescription,
  SheetHeader,
  SheetTitle,
} from "@/components/ui/sheet";
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
  getDatabase,
  createDatabase,
  deleteDatabase,
  createDatabaseUser,
  deleteDatabaseUser,
} from "@/api/databases";
import type { Database, DatabaseUser } from "@/types/database";
import type { AxiosError } from "axios";
import type { APIError } from "@/types/api";
import {
  Database as DatabaseIcon,
  Plus,
  Trash2,
  UserPlus,
  User,
  HardDrive,
} from "lucide-react";

function formatSize(bytes: number): string {
  if (bytes === 0) return "0 B";
  const units = ["B", "KB", "MB", "GB"];
  const i = Math.floor(Math.log(bytes) / Math.log(1024));
  return `${(bytes / Math.pow(1024, i)).toFixed(i > 0 ? 1 : 0)} ${units[i]}`;
}

export function DatabasesPage() {
  const queryClient = useQueryClient();

  const [showCreate, setShowCreate] = useState(false);
  const [newDbName, setNewDbName] = useState("");
  const [selectedDb, setSelectedDb] = useState<Database | null>(null);
  const [deleteDb, setDeleteDb] = useState<Database | null>(null);
  const [showAddUser, setShowAddUser] = useState(false);
  const [newUsername, setNewUsername] = useState("");
  const [newPassword, setNewPassword] = useState("");
  const [deleteUser, setDeleteUser] = useState<DatabaseUser | null>(null);

  const { data, isLoading } = useQuery({
    queryKey: ["databases"],
    queryFn: () => listDatabases(),
  });

  const { data: detail } = useQuery({
    queryKey: ["database-detail", selectedDb?.id],
    queryFn: () => getDatabase(selectedDb!.id),
    enabled: !!selectedDb,
  });

  const createMutation = useMutation({
    mutationFn: () => createDatabase({ name: newDbName }),
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
      if (selectedDb?.id === deleteDb?.id) setSelectedDb(null);
      queryClient.invalidateQueries({ queryKey: ["databases"] });
    },
    onError: (err: AxiosError<APIError>) => {
      toast.error(err.response?.data?.error?.message ?? "Failed to delete database");
    },
  });

  const createUserMutation = useMutation({
    mutationFn: () =>
      createDatabaseUser(selectedDb!.id, {
        username: newUsername,
        password: newPassword,
      }),
    onSuccess: () => {
      toast.success("User created");
      setShowAddUser(false);
      setNewUsername("");
      setNewPassword("");
      queryClient.invalidateQueries({ queryKey: ["database-detail", selectedDb?.id] });
    },
    onError: (err: AxiosError<APIError>) => {
      toast.error(err.response?.data?.error?.message ?? "Failed to create user");
    },
  });

  const deleteUserMutation = useMutation({
    mutationFn: () => deleteDatabaseUser(selectedDb!.id, deleteUser!.id),
    onSuccess: () => {
      toast.success("User deleted");
      setDeleteUser(null);
      queryClient.invalidateQueries({ queryKey: ["database-detail", selectedDb?.id] });
    },
    onError: (err: AxiosError<APIError>) => {
      toast.error(err.response?.data?.error?.message ?? "Failed to delete user");
    },
  });

  const databases = data?.data ?? [];

  if (isLoading) {
    return (
      <div className="space-y-4">
        <Skeleton className="h-10 w-full" />
        <Skeleton className="h-64 w-full" />
      </div>
    );
  }

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold">Databases</h1>
          <p className="text-muted-foreground">Manage MySQL databases and users</p>
        </div>
        <Button
          onClick={() => { setShowCreate(true); setNewDbName(""); }}
          className="bg-pink-500 hover:bg-pink-600"
        >
          <Plus className="h-4 w-4 mr-1" />
          New Database
        </Button>
      </div>

      {databases.length === 0 ? (
        <Card>
          <CardContent className="py-12 text-center">
            <DatabaseIcon className="h-12 w-12 mx-auto text-muted-foreground mb-4" />
            <h3 className="text-lg font-medium">No databases yet</h3>
            <p className="text-muted-foreground text-sm mt-1">
              Create your first MySQL database to get started
            </p>
          </CardContent>
        </Card>
      ) : (
        <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-3">
          {databases.map((db) => (
            <Card
              key={db.id}
              className="cursor-pointer hover:border-pink-500/50 transition-colors"
              onClick={() => setSelectedDb(db)}
            >
              <CardHeader className="pb-3">
                <div className="flex items-center justify-between">
                  <CardTitle className="text-base flex items-center gap-2">
                    <DatabaseIcon className="h-4 w-4 text-pink-500" />
                    {db.name}
                  </CardTitle>
                  <Button
                    size="icon"
                    variant="ghost"
                    className="h-7 w-7 text-destructive"
                    onClick={(e) => {
                      e.stopPropagation();
                      setDeleteDb(db);
                    }}
                  >
                    <Trash2 className="h-3 w-3" />
                  </Button>
                </div>
                <CardDescription className="flex items-center gap-3 text-xs">
                  <span className="flex items-center gap-1">
                    <HardDrive className="h-3 w-3" />
                    {formatSize(db.size_bytes)}
                  </span>
                  <Badge variant="outline" className="text-xs">
                    {db.type}
                  </Badge>
                </CardDescription>
              </CardHeader>
            </Card>
          ))}
        </div>
      )}

      {/* Database Detail Sheet */}
      <Sheet open={!!selectedDb} onOpenChange={() => setSelectedDb(null)}>
        <SheetContent className="sm:max-w-lg">
          <SheetHeader>
            <SheetTitle className="flex items-center gap-2">
              <DatabaseIcon className="h-5 w-5 text-pink-500" />
              {selectedDb?.name}
            </SheetTitle>
            <SheetDescription>
              {selectedDb?.type} — {formatSize(selectedDb?.size_bytes ?? 0)}
            </SheetDescription>
          </SheetHeader>

          <div className="space-y-6 py-6">
            {/* Users */}
            <div>
              <div className="flex items-center justify-between mb-3">
                <h3 className="text-sm font-medium">Database Users</h3>
                <Button
                  size="sm"
                  variant="outline"
                  onClick={() => {
                    setShowAddUser(true);
                    setNewUsername("");
                    setNewPassword("");
                  }}
                >
                  <UserPlus className="h-3 w-3 mr-1" />
                  Add User
                </Button>
              </div>

              {(!detail?.users || detail.users.length === 0) ? (
                <p className="text-sm text-muted-foreground">No users assigned</p>
              ) : (
                <div className="space-y-2">
                  {detail.users.map((u) => (
                    <div
                      key={u.id}
                      className="flex items-center justify-between p-2 rounded border text-sm"
                    >
                      <div className="flex items-center gap-2">
                        <User className="h-4 w-4 text-muted-foreground" />
                        <span className="font-mono">{u.username}@{u.host}</span>
                        <Badge variant="outline" className="text-xs">
                          {u.permissions}
                        </Badge>
                      </div>
                      <Button
                        size="icon"
                        variant="ghost"
                        className="h-6 w-6 text-destructive"
                        onClick={() => setDeleteUser(u)}
                      >
                        <Trash2 className="h-3 w-3" />
                      </Button>
                    </div>
                  ))}
                </div>
              )}
            </div>
          </div>
        </SheetContent>
      </Sheet>

      {/* Create Database Dialog */}
      <Dialog open={showCreate} onOpenChange={setShowCreate}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>New Database</DialogTitle>
            <DialogDescription>
              Create a new MySQL database
            </DialogDescription>
          </DialogHeader>
          <div className="space-y-2">
            <Label>Database Name</Label>
            <Input
              value={newDbName}
              onChange={(e) => setNewDbName(e.target.value)}
              placeholder="my_database"
              autoFocus
            />
            <p className="text-xs text-muted-foreground">
              Alphanumeric characters, underscores, and hyphens only
            </p>
          </div>
          <DialogFooter>
            <Button variant="outline" onClick={() => setShowCreate(false)}>
              Cancel
            </Button>
            <Button
              onClick={() => createMutation.mutate()}
              disabled={!newDbName || createMutation.isPending}
              className="bg-pink-500 hover:bg-pink-600"
            >
              {createMutation.isPending ? "Creating..." : "Create Database"}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* Add User Dialog */}
      <Dialog open={showAddUser} onOpenChange={setShowAddUser}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Add Database User</DialogTitle>
            <DialogDescription>
              Create a MySQL user with access to {selectedDb?.name}
            </DialogDescription>
          </DialogHeader>
          <div className="space-y-4">
            <div className="space-y-2">
              <Label>Username</Label>
              <Input
                value={newUsername}
                onChange={(e) => setNewUsername(e.target.value)}
                placeholder="db_user"
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
          </div>
          <DialogFooter>
            <Button variant="outline" onClick={() => setShowAddUser(false)}>
              Cancel
            </Button>
            <Button
              onClick={() => createUserMutation.mutate()}
              disabled={!newUsername || !newPassword || createUserMutation.isPending}
              className="bg-pink-500 hover:bg-pink-600"
            >
              {createUserMutation.isPending ? "Creating..." : "Add User"}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* Delete Database Confirmation */}
      <ConfirmDialog
        open={!!deleteDb}
        onOpenChange={() => setDeleteDb(null)}
        title="Delete Database"
        description={`This will permanently delete the database "${deleteDb?.name}" and all its data. This action cannot be undone.`}
        confirmText="Delete Database"
        typeToConfirm={deleteDb?.name}
        destructive
        loading={deleteMutation.isPending}
        onConfirm={() => deleteMutation.mutate()}
      />

      {/* Delete User Confirmation */}
      <ConfirmDialog
        open={!!deleteUser}
        onOpenChange={() => setDeleteUser(null)}
        title="Delete User"
        description={`Remove user "${deleteUser?.username}@${deleteUser?.host}" and revoke all permissions?`}
        confirmText="Delete User"
        destructive
        loading={deleteUserMutation.isPending}
        onConfirm={() => deleteUserMutation.mutate()}
      />
    </div>
  );
}
