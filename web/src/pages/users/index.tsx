import { useState } from "react";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import {
  listUsers,
  createUser,
  deleteUser,
  suspendUser,
  activateUser,
  resetUserPassword,
  updateUser,
} from "@/api/users";
import type { UserWithStats, CreateUserRequest } from "@/api/users";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Badge } from "@/components/ui/badge";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
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
  UserPlus,
  Shield,
  ShieldCheck,
  User,
  Ban,
  CheckCircle,
  Trash2,
  KeyRound,
  Globe,
  Database,
  Wifi,
  Pencil,
} from "lucide-react";
import { useAuthStore } from "@/stores/auth";

export function UsersPage() {
  const [search, setSearch] = useState("");
  const [showCreate, setShowCreate] = useState(false);
  const [showResetPw, setShowResetPw] = useState<UserWithStats | null>(null);
  const [showEdit, setShowEdit] = useState<UserWithStats | null>(null);
  const [confirmDelete, setConfirmDelete] = useState<UserWithStats | null>(null);
  const queryClient = useQueryClient();
  const currentRole = useAuthStore((s) => s.role);

  const { data: users = [], isLoading } = useQuery({
    queryKey: ["users", search],
    queryFn: () => listUsers(search || undefined),
  });

  const suspendMutation = useMutation({
    mutationFn: (id: number) => suspendUser(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["users"] });
      toast.success("User suspended");
    },
    onError: () => toast.error("Failed to suspend user"),
  });

  const activateMutation = useMutation({
    mutationFn: (id: number) => activateUser(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["users"] });
      toast.success("User activated");
    },
    onError: () => toast.error("Failed to activate user"),
  });

  const deleteMutation = useMutation({
    mutationFn: (id: number) => deleteUser(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["users"] });
      toast.success("User deleted");
      setConfirmDelete(null);
    },
    onError: (err: any) => {
      toast.error(err?.response?.data?.error?.message ?? "Failed to delete user");
      setConfirmDelete(null);
    },
  });

  const roleIcon = (role: string) => {
    switch (role) {
      case "super_admin":
        return <ShieldCheck className="h-3.5 w-3.5" />;
      case "admin":
        return <Shield className="h-3.5 w-3.5" />;
      default:
        return <User className="h-3.5 w-3.5" />;
    }
  };

  const roleLabel = (role: string) => {
    switch (role) {
      case "super_admin":
        return "Super Admin";
      case "admin":
        return "Admin";
      default:
        return "User";
    }
  };

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <h1 className="text-2xl font-bold">Users</h1>
        {currentRole === "super_admin" && (
          <Button
            className="bg-pink-500 hover:bg-pink-600"
            onClick={() => setShowCreate(true)}
          >
            <UserPlus className="h-4 w-4 mr-2" />
            Create User
          </Button>
        )}
      </div>

      <Input
        placeholder="Search users..."
        value={search}
        onChange={(e) => setSearch(e.target.value)}
        className="max-w-sm"
      />

      {isLoading ? (
        <p className="text-muted-foreground">Loading...</p>
      ) : users.length === 0 ? (
        <p className="text-muted-foreground">No users found</p>
      ) : (
        <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-3">
          {users.map((u) => (
            <Card key={u.id}>
              <CardHeader className="flex flex-row items-center justify-between pb-2">
                <CardTitle className="text-base font-medium">
                  {u.username}
                </CardTitle>
                <div className="flex items-center gap-2">
                  <Badge
                    variant="outline"
                    className={
                      u.status === "active"
                        ? "bg-green-500/10 text-green-500 border-green-500/20"
                        : "bg-red-500/10 text-red-500 border-red-500/20"
                    }
                  >
                    {u.status}
                  </Badge>
                  <Badge variant="outline" className="gap-1">
                    {roleIcon(u.role)}
                    {roleLabel(u.role)}
                  </Badge>
                </div>
              </CardHeader>
              <CardContent className="space-y-3">
                <p className="text-sm text-muted-foreground">{u.email}</p>

                <div className="flex gap-4 text-sm text-muted-foreground">
                  <span className="flex items-center gap-1">
                    <Globe className="h-3.5 w-3.5" />
                    {u.domain_count} domains
                  </span>
                  <span className="flex items-center gap-1">
                    <Database className="h-3.5 w-3.5" />
                    {u.database_count} DBs
                  </span>
                  <span className="flex items-center gap-1">
                    <Wifi className="h-3.5 w-3.5" />
                    {u.ftp_count} FTP
                  </span>
                </div>

                <div className="flex gap-2 pt-1">
                  {currentRole === "super_admin" && (
                    <>
                      <Button
                        variant="outline"
                        size="sm"
                        onClick={() => setShowEdit(u)}
                      >
                        <Pencil className="h-3.5 w-3.5 mr-1" />
                        Edit
                      </Button>
                      <Button
                        variant="outline"
                        size="sm"
                        onClick={() => setShowResetPw(u)}
                      >
                        <KeyRound className="h-3.5 w-3.5 mr-1" />
                        Reset PW
                      </Button>
                    </>
                  )}
                  {u.status === "active" ? (
                    <Button
                      variant="outline"
                      size="sm"
                      className="text-amber-500"
                      onClick={() => suspendMutation.mutate(u.id)}
                    >
                      <Ban className="h-3.5 w-3.5 mr-1" />
                      Suspend
                    </Button>
                  ) : (
                    <Button
                      variant="outline"
                      size="sm"
                      className="text-green-500"
                      onClick={() => activateMutation.mutate(u.id)}
                    >
                      <CheckCircle className="h-3.5 w-3.5 mr-1" />
                      Activate
                    </Button>
                  )}
                  {currentRole === "super_admin" && (
                    <Button
                      variant="outline"
                      size="sm"
                      className="text-red-500"
                      onClick={() => setConfirmDelete(u)}
                    >
                      <Trash2 className="h-3.5 w-3.5" />
                    </Button>
                  )}
                </div>

                <p className="text-xs text-muted-foreground">
                  Created {new Date(u.created_at).toLocaleDateString()}
                </p>
              </CardContent>
            </Card>
          ))}
        </div>
      )}

      {showCreate && (
        <CreateUserDialog
          onClose={() => setShowCreate(false)}
          onSuccess={() => {
            setShowCreate(false);
            queryClient.invalidateQueries({ queryKey: ["users"] });
          }}
        />
      )}

      {showResetPw && (
        <ResetPasswordDialog
          user={showResetPw}
          onClose={() => setShowResetPw(null)}
        />
      )}

      {showEdit && (
        <EditUserDialog
          user={showEdit}
          onClose={() => setShowEdit(null)}
          onSuccess={() => {
            setShowEdit(null);
            queryClient.invalidateQueries({ queryKey: ["users"] });
          }}
        />
      )}

      <ConfirmDialog
        open={!!confirmDelete}
        onOpenChange={(open) => !open && setConfirmDelete(null)}
        title="Delete User"
        description={`Are you sure you want to delete "${confirmDelete?.username}"? This will remove all their resources.`}
        onConfirm={() => confirmDelete && deleteMutation.mutate(confirmDelete.id)}
        destructive
      />
    </div>
  );
}

function CreateUserDialog({
  onClose,
  onSuccess,
}: {
  onClose: () => void;
  onSuccess: () => void;
}) {
  const [form, setForm] = useState<CreateUserRequest>({
    username: "",
    email: "",
    password: "",
    role: "user",
  });

  const createMutation = useMutation({
    mutationFn: () => createUser(form),
    onSuccess: () => {
      toast.success("User created");
      onSuccess();
    },
    onError: (err: any) => {
      toast.error(err?.response?.data?.error?.message ?? "Failed to create user");
    },
  });

  return (
    <Dialog open onOpenChange={(open) => !open && onClose()}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Create User</DialogTitle>
          <DialogDescription>
            Create a new user account with a role.
          </DialogDescription>
        </DialogHeader>
        <div className="space-y-4">
          <div className="space-y-2">
            <Label>Username</Label>
            <Input
              value={form.username}
              onChange={(e) => setForm({ ...form, username: e.target.value })}
              placeholder="johndoe"
            />
          </div>
          <div className="space-y-2">
            <Label>Email</Label>
            <Input
              type="email"
              value={form.email}
              onChange={(e) => setForm({ ...form, email: e.target.value })}
              placeholder="john@example.com"
            />
          </div>
          <div className="space-y-2">
            <Label>Password</Label>
            <Input
              type="password"
              value={form.password}
              onChange={(e) => setForm({ ...form, password: e.target.value })}
              placeholder="Min. 8 characters"
            />
          </div>
          <div className="space-y-2">
            <Label>Role</Label>
            <Select
              value={form.role}
              onValueChange={(v) => v && setForm({ ...form, role: v })}
            >
              <SelectTrigger>
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="user">User</SelectItem>
                <SelectItem value="admin">Admin</SelectItem>
                <SelectItem value="super_admin">Super Admin</SelectItem>
              </SelectContent>
            </Select>
          </div>
        </div>
        <DialogFooter>
          <Button variant="outline" onClick={onClose}>
            Cancel
          </Button>
          <Button
            className="bg-pink-500 hover:bg-pink-600"
            onClick={() => createMutation.mutate()}
            disabled={createMutation.isPending}
          >
            {createMutation.isPending ? "Creating..." : "Create"}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}

function EditUserDialog({
  user,
  onClose,
  onSuccess,
}: {
  user: UserWithStats;
  onClose: () => void;
  onSuccess: () => void;
}) {
  const [email, setEmail] = useState(user.email);
  const [role, setRole] = useState(user.role);

  const editMutation = useMutation({
    mutationFn: () => updateUser(user.id, { email, role }),
    onSuccess: () => {
      toast.success("User updated");
      onSuccess();
    },
    onError: (err: any) => {
      toast.error(err?.response?.data?.error?.message ?? "Failed to update user");
    },
  });

  return (
    <Dialog open onOpenChange={(open) => !open && onClose()}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Edit User: {user.username}</DialogTitle>
        </DialogHeader>
        <div className="space-y-4">
          <div className="space-y-2">
            <Label>Email</Label>
            <Input
              type="email"
              value={email}
              onChange={(e) => setEmail(e.target.value)}
            />
          </div>
          <div className="space-y-2">
            <Label>Role</Label>
            <Select value={role} onValueChange={(v) => v && setRole(v)}>
              <SelectTrigger>
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="user">User</SelectItem>
                <SelectItem value="admin">Admin</SelectItem>
                <SelectItem value="super_admin">Super Admin</SelectItem>
              </SelectContent>
            </Select>
          </div>
        </div>
        <DialogFooter>
          <Button variant="outline" onClick={onClose}>
            Cancel
          </Button>
          <Button
            className="bg-pink-500 hover:bg-pink-600"
            onClick={() => editMutation.mutate()}
            disabled={editMutation.isPending}
          >
            {editMutation.isPending ? "Saving..." : "Save"}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}

function ResetPasswordDialog({
  user,
  onClose,
}: {
  user: UserWithStats;
  onClose: () => void;
}) {
  const [password, setPassword] = useState("");

  const resetMutation = useMutation({
    mutationFn: () => resetUserPassword(user.id, password),
    onSuccess: () => {
      toast.success("Password reset");
      onClose();
    },
    onError: (err: any) => {
      toast.error(
        err?.response?.data?.error?.message ?? "Failed to reset password"
      );
    },
  });

  return (
    <Dialog open onOpenChange={(open) => !open && onClose()}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Reset Password: {user.username}</DialogTitle>
          <DialogDescription>
            Set a new password. The user will be logged out of all sessions.
          </DialogDescription>
        </DialogHeader>
        <div className="space-y-2">
          <Label>New Password</Label>
          <Input
            type="password"
            value={password}
            onChange={(e) => setPassword(e.target.value)}
            placeholder="Min. 8 characters"
          />
        </div>
        <DialogFooter>
          <Button variant="outline" onClick={onClose}>
            Cancel
          </Button>
          <Button
            className="bg-pink-500 hover:bg-pink-600"
            onClick={() => resetMutation.mutate()}
            disabled={resetMutation.isPending || password.length < 8}
          >
            {resetMutation.isPending ? "Resetting..." : "Reset Password"}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
