import { useState } from "react";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { type ColumnDef } from "@tanstack/react-table";
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
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import {
  Sheet,
  SheetContent,
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
import { DataTable } from "@/components/shared/data-table";
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
  ExternalLink,
  Copy,
  Eye,
  EyeOff,
  RefreshCw,
  Server,
  Users,
} from "lucide-react";

function formatSize(bytes: number): string {
  if (bytes === 0) return "0 B";
  const units = ["B", "KB", "MB", "GB"];
  const i = Math.floor(Math.log(bytes) / Math.log(1024));
  return `${(bytes / Math.pow(1024, i)).toFixed(i > 0 ? 1 : 0)} ${units[i]}`;
}

function generatePassword(length = 20): string {
  const chars = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789!@#$%^&*";
  const array = new Uint8Array(length);
  crypto.getRandomValues(array);
  return Array.from(array, (b) => chars[b % chars.length]).join("");
}

function copyToClipboard(text: string, label: string) {
  navigator.clipboard.writeText(text);
  toast.success(`${label} copied to clipboard`);
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
  const [newHost, setNewHost] = useState("localhost");
  const [newPermissions, setNewPermissions] = useState("ALL");
  const [showPassword, setShowPassword] = useState(false);
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
        host: newHost,
        permissions: newPermissions,
      }),
    onSuccess: () => {
      toast.success("Database user created");
      setShowAddUser(false);
      setNewUsername("");
      setNewPassword("");
      setNewHost("localhost");
      setNewPermissions("ALL");
      setShowPassword(false);
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

  const columns: ColumnDef<Database, unknown>[] = [
    {
      accessorKey: "name",
      header: "Database",
      cell: ({ row }) => (
        <button
          className="flex items-center gap-2 font-medium hover:text-pink-500 transition-colors text-left"
          onClick={() => setSelectedDb(row.original)}
        >
          <DatabaseIcon className="h-4 w-4 text-pink-500 shrink-0" />
          <span className="font-mono text-sm">{row.original.name}</span>
        </button>
      ),
    },
    {
      accessorKey: "type",
      header: "Engine",
      cell: ({ row }) => (
        <Badge variant="outline" className="text-xs font-normal">
          <Server className="h-3 w-3 mr-1" />
          {row.original.type === "mysql" ? "MySQL" : "MariaDB"}
        </Badge>
      ),
    },
    {
      accessorKey: "size_bytes",
      header: "Size",
      cell: ({ row }) => (
        <span className="flex items-center gap-1 text-sm text-muted-foreground">
          <HardDrive className="h-3 w-3" />
          {formatSize(row.original.size_bytes)}
        </span>
      ),
    },
    {
      accessorKey: "created_at",
      header: "Created",
      cell: ({ row }) => (
        <span className="text-sm text-muted-foreground">
          {new Date(row.original.created_at).toLocaleDateString()}
        </span>
      ),
    },
    {
      id: "actions",
      header: "",
      enableSorting: false,
      cell: ({ row }) => (
        <div className="flex items-center justify-end gap-1">
          <Button
            size="sm"
            variant="ghost"
            onClick={() => setSelectedDb(row.original)}
          >
            <Users className="h-3.5 w-3.5 mr-1" />
            Manage
          </Button>
          <Button
            size="icon"
            variant="ghost"
            className="h-8 w-8 text-destructive hover:text-destructive"
            onClick={() => setDeleteDb(row.original)}
          >
            <Trash2 className="h-3.5 w-3.5" />
          </Button>
        </div>
      ),
    },
  ];

  const phpmyadminUrl = `${window.location.protocol}//${window.location.hostname}/phpmyadmin/`;

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold">Databases</h1>
          <p className="text-muted-foreground">Manage MySQL databases and users</p>
        </div>
        <div className="flex items-center gap-2">
          <Button
            variant="outline"
            onClick={() => window.open(phpmyadminUrl, "_blank")}
          >
            <ExternalLink className="h-4 w-4 mr-1" />
            phpMyAdmin
          </Button>
          <Button
            onClick={() => { setShowCreate(true); setNewDbName(""); }}
            className="bg-pink-500 hover:bg-pink-600"
          >
            <Plus className="h-4 w-4 mr-1" />
            New Database
          </Button>
        </div>
      </div>

      {/* Connection Info Card */}
      <Card>
        <CardHeader className="pb-3">
          <CardTitle className="text-sm font-medium text-muted-foreground">Connection Details</CardTitle>
        </CardHeader>
        <CardContent>
          <div className="grid grid-cols-2 md:grid-cols-4 gap-4 text-sm">
            <div>
              <span className="text-muted-foreground text-xs">Host</span>
              <div className="font-mono mt-0.5 flex items-center gap-1">
                localhost
                <button onClick={() => copyToClipboard("localhost", "Host")} className="text-muted-foreground hover:text-foreground">
                  <Copy className="h-3 w-3" />
                </button>
              </div>
            </div>
            <div>
              <span className="text-muted-foreground text-xs">Port</span>
              <div className="font-mono mt-0.5 flex items-center gap-1">
                3306
                <button onClick={() => copyToClipboard("3306", "Port")} className="text-muted-foreground hover:text-foreground">
                  <Copy className="h-3 w-3" />
                </button>
              </div>
            </div>
            <div>
              <span className="text-muted-foreground text-xs">Server</span>
              <div className="font-mono mt-0.5">MariaDB</div>
            </div>
            <div>
              <span className="text-muted-foreground text-xs">phpMyAdmin</span>
              <div className="mt-0.5">
                <a
                  href={phpmyadminUrl}
                  target="_blank"
                  rel="noopener noreferrer"
                  className="text-pink-500 hover:text-pink-600 flex items-center gap-1 text-xs"
                >
                  Open in new tab <ExternalLink className="h-3 w-3" />
                </a>
              </div>
            </div>
          </div>
        </CardContent>
      </Card>

      {/* Databases Table */}
      <DataTable
        columns={columns}
        data={databases}
        loading={isLoading}
        emptyTitle="No databases yet"
        emptyDescription="Create your first MySQL database to get started"
        emptyIcon={<DatabaseIcon className="h-12 w-12" />}
        emptyAction={
          <Button
            onClick={() => { setShowCreate(true); setNewDbName(""); }}
            className="bg-pink-500 hover:bg-pink-600"
          >
            <Plus className="h-4 w-4 mr-1" />
            New Database
          </Button>
        }
      />

      {/* Database Detail Sheet */}
      <Sheet open={!!selectedDb} onOpenChange={() => setSelectedDb(null)}>
        <SheetContent className="sm:max-w-lg overflow-y-auto">
          <SheetHeader>
            <SheetTitle className="flex items-center gap-2">
              <DatabaseIcon className="h-5 w-5 text-pink-500" />
              {selectedDb?.name}
            </SheetTitle>
          </SheetHeader>

          <div className="space-y-6 py-6">
            {/* Database Info */}
            <div className="grid grid-cols-2 gap-4 text-sm p-4 rounded-lg bg-muted/50">
              <div>
                <span className="text-muted-foreground text-xs">Engine</span>
                <div className="font-medium mt-0.5">
                  {selectedDb?.type === "mysql" ? "MySQL" : "MariaDB"}
                </div>
              </div>
              <div>
                <span className="text-muted-foreground text-xs">Size</span>
                <div className="font-medium mt-0.5">{formatSize(selectedDb?.size_bytes ?? 0)}</div>
              </div>
              <div>
                <span className="text-muted-foreground text-xs">Created</span>
                <div className="font-medium mt-0.5">
                  {selectedDb?.created_at ? new Date(selectedDb.created_at).toLocaleDateString() : "—"}
                </div>
              </div>
              <div>
                <span className="text-muted-foreground text-xs">Database Name</span>
                <div className="font-mono mt-0.5 flex items-center gap-1">
                  {selectedDb?.name}
                  <button onClick={() => copyToClipboard(selectedDb?.name ?? "", "Database name")} className="text-muted-foreground hover:text-foreground">
                    <Copy className="h-3 w-3" />
                  </button>
                </div>
              </div>
            </div>

            {/* Users Section */}
            <div>
              <div className="flex items-center justify-between mb-3">
                <h3 className="text-sm font-semibold flex items-center gap-1.5">
                  <Users className="h-4 w-4" />
                  Database Users
                  {detail?.users && detail.users.length > 0 && (
                    <Badge variant="secondary" className="text-xs ml-1">
                      {detail.users.length}
                    </Badge>
                  )}
                </h3>
                <Button
                  size="sm"
                  onClick={() => {
                    setShowAddUser(true);
                    setNewUsername("");
                    setNewPassword("");
                    setNewHost("localhost");
                    setNewPermissions("ALL");
                    setShowPassword(false);
                  }}
                  className="bg-pink-500 hover:bg-pink-600"
                >
                  <UserPlus className="h-3 w-3 mr-1" />
                  Add User
                </Button>
              </div>

              {(!detail?.users || detail.users.length === 0) ? (
                <div className="text-center py-8 border rounded-lg border-dashed">
                  <User className="h-8 w-8 mx-auto text-muted-foreground mb-2" />
                  <p className="text-sm text-muted-foreground">No users assigned</p>
                  <p className="text-xs text-muted-foreground mt-1">
                    Add a user to connect to this database
                  </p>
                </div>
              ) : (
                <div className="space-y-2">
                  {detail.users.map((u) => (
                    <div
                      key={u.id}
                      className="flex items-center justify-between p-3 rounded-lg border bg-card"
                    >
                      <div className="space-y-1">
                        <div className="flex items-center gap-2">
                          <User className="h-4 w-4 text-muted-foreground" />
                          <span className="font-mono text-sm font-medium">{u.username}</span>
                          <span className="text-muted-foreground text-xs">@{u.host}</span>
                          <button
                            onClick={() => copyToClipboard(u.username, "Username")}
                            className="text-muted-foreground hover:text-foreground"
                          >
                            <Copy className="h-3 w-3" />
                          </button>
                        </div>
                        <div className="flex items-center gap-2 ml-6">
                          <Badge
                            variant="outline"
                            className={`text-xs ${u.permissions === "ALL" ? "border-green-500/30 text-green-500" : "border-amber-500/30 text-amber-500"}`}
                          >
                            {u.permissions === "ALL" ? "Full Access" : u.permissions}
                          </Badge>
                        </div>
                      </div>
                      <Button
                        size="icon"
                        variant="ghost"
                        className="h-8 w-8 text-destructive hover:text-destructive"
                        onClick={() => setDeleteUser(u)}
                      >
                        <Trash2 className="h-3.5 w-3.5" />
                      </Button>
                    </div>
                  ))}
                </div>
              )}
            </div>

            {/* Quick Actions */}
            <div className="pt-2 border-t space-y-2">
              <Button
                variant="outline"
                size="sm"
                className="w-full justify-start"
                onClick={() => window.open(phpmyadminUrl, "_blank")}
              >
                <ExternalLink className="h-4 w-4 mr-2" />
                Open in phpMyAdmin
              </Button>
              <Button
                variant="outline"
                size="sm"
                className="w-full justify-start text-destructive hover:text-destructive"
                onClick={() => {
                  setDeleteDb(selectedDb);
                }}
              >
                <Trash2 className="h-4 w-4 mr-2" />
                Delete Database
              </Button>
            </div>
          </div>
        </SheetContent>
      </Sheet>

      {/* Create Database Dialog */}
      <Dialog open={showCreate} onOpenChange={setShowCreate}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Create Database</DialogTitle>
            <DialogDescription>
              Create a new MySQL database on this server
            </DialogDescription>
          </DialogHeader>
          <div className="space-y-4">
            <div className="space-y-2">
              <Label>Database Name</Label>
              <Input
                value={newDbName}
                onChange={(e) => setNewDbName(e.target.value)}
                placeholder="my_database"
                className="font-mono"
                autoFocus
              />
              <p className="text-xs text-muted-foreground">
                Letters, numbers, underscores, and hyphens. Max 64 characters.
              </p>
            </div>
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
        <DialogContent className="sm:max-w-md">
          <DialogHeader>
            <DialogTitle>Add Database User</DialogTitle>
            <DialogDescription>
              Create a MySQL user with access to <span className="font-mono font-medium">{selectedDb?.name}</span>
            </DialogDescription>
          </DialogHeader>
          <div className="space-y-4">
            <div className="space-y-2">
              <Label>Username</Label>
              <Input
                value={newUsername}
                onChange={(e) => setNewUsername(e.target.value)}
                placeholder="db_user"
                className="font-mono"
                autoFocus
              />
            </div>
            <div className="space-y-2">
              <div className="flex items-center justify-between">
                <Label>Password</Label>
                <Button
                  type="button"
                  variant="ghost"
                  size="sm"
                  className="h-6 text-xs"
                  onClick={() => setNewPassword(generatePassword())}
                >
                  <RefreshCw className="h-3 w-3 mr-1" />
                  Generate
                </Button>
              </div>
              <div className="relative">
                <Input
                  type={showPassword ? "text" : "password"}
                  value={newPassword}
                  onChange={(e) => setNewPassword(e.target.value)}
                  placeholder="Strong password"
                  className="font-mono pr-20"
                />
                <div className="absolute right-1 top-1/2 -translate-y-1/2 flex items-center gap-0.5">
                  <Button
                    type="button"
                    variant="ghost"
                    size="icon"
                    className="h-7 w-7"
                    onClick={() => setShowPassword(!showPassword)}
                  >
                    {showPassword ? <EyeOff className="h-3.5 w-3.5" /> : <Eye className="h-3.5 w-3.5" />}
                  </Button>
                  {newPassword && (
                    <Button
                      type="button"
                      variant="ghost"
                      size="icon"
                      className="h-7 w-7"
                      onClick={() => copyToClipboard(newPassword, "Password")}
                    >
                      <Copy className="h-3.5 w-3.5" />
                    </Button>
                  )}
                </div>
              </div>
            </div>
            <div className="grid grid-cols-2 gap-4">
              <div className="space-y-2">
                <Label>Host</Label>
                <Select value={newHost} onValueChange={(v) => v && setNewHost(v)}>
                  <SelectTrigger>
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="localhost">localhost</SelectItem>
                    <SelectItem value="%">% (any host)</SelectItem>
                    <SelectItem value="127.0.0.1">127.0.0.1</SelectItem>
                  </SelectContent>
                </Select>
              </div>
              <div className="space-y-2">
                <Label>Permissions</Label>
                <Select value={newPermissions} onValueChange={(v) => v && setNewPermissions(v)}>
                  <SelectTrigger>
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="ALL">Full Access</SelectItem>
                    <SelectItem value="SELECT, INSERT, UPDATE, DELETE">Read/Write</SelectItem>
                    <SelectItem value="SELECT">Read Only</SelectItem>
                  </SelectContent>
                </Select>
              </div>
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
        description={`This will permanently delete the database "${deleteDb?.name}" and all its data including users. This action cannot be undone.`}
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
        description={`Remove user "${deleteUser?.username}@${deleteUser?.host}" and revoke all database permissions?`}
        confirmText="Delete User"
        destructive
        loading={deleteUserMutation.isPending}
        onConfirm={() => deleteUserMutation.mutate()}
      />
    </div>
  );
}
