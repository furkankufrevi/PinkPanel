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
import { Switch } from "@/components/ui/switch";
import { Textarea } from "@/components/ui/textarea";
import { Separator } from "@/components/ui/separator";
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
  listGitRepos,
  createGitRepo,
  deleteGitRepo,
  updateGitRepo,
  triggerDeploy,
  listDeployments,
} from "@/api/git";
import type { GitRepository, GitDeployment } from "@/types/git";
import type { AxiosError } from "axios";
import type { APIError } from "@/types/api";
import { getDomain } from "@/api/domains";
import {
  GitBranch,
  Plus,
  Trash2,
  Clock,
  Copy,
  Settings,
  CheckCircle2,
  XCircle,
  Loader2,
  Globe,
  HardDrive,
  Rocket,
  Download,
  ChevronDown,
  ChevronRight,
  FolderSync,
  Webhook,
} from "lucide-react";

function copyToClipboard(text: string) {
  navigator.clipboard.writeText(text);
  toast.success("Copied to clipboard");
}

export function DomainGit() {
  const { id } = useParams<{ id: string }>();
  const domainId = Number(id);
  const queryClient = useQueryClient();

  const [showCreate, setShowCreate] = useState(false);
  const [deleteTarget, setDeleteTarget] = useState<GitRepository | null>(null);
  const [editRepo, setEditRepo] = useState<GitRepository | null>(null);

  // Create form state
  const [newType, setNewType] = useState<"remote" | "local">("remote");
  const [newRemoteURL, setNewRemoteURL] = useState("");
  const [newName, setNewName] = useState("");
  const [newBranch, setNewBranch] = useState("main");
  const [newDeployMode, setNewDeployMode] = useState("automatic");
  const [newDeployPath, setNewDeployPath] = useState("");
  const [newPostDeployCmd, setNewPostDeployCmd] = useState("");
  const [showPostDeploy, setShowPostDeploy] = useState(false);

  // Edit form state
  const [editBranch, setEditBranch] = useState("");
  const [editDeployMode, setEditDeployMode] = useState("");
  const [editDeployPath, setEditDeployPath] = useState("");
  const [editPostDeployCmd, setEditPostDeployCmd] = useState("");

  const { data: domain } = useQuery({
    queryKey: ["domain", domainId],
    queryFn: () => getDomain(domainId),
    enabled: !!domainId,
  });

  const { data, isLoading } = useQuery({
    queryKey: ["git-repos", domainId],
    queryFn: () => listGitRepos(domainId),
    enabled: !!domainId,
  });

  const createMutation = useMutation({
    mutationFn: () =>
      createGitRepo(domainId, {
        name: newName,
        repo_type: newType,
        remote_url: newType === "remote" ? newRemoteURL : undefined,
        branch: newBranch,
        deploy_mode: newDeployMode,
        deploy_path: newDeployPath || undefined,
      }),
    onSuccess: () => {
      toast.success("Repository created — cloning in progress");
      setShowCreate(false);
      resetCreateForm();
      queryClient.invalidateQueries({ queryKey: ["git-repos", domainId] });
    },
    onError: (err: AxiosError<APIError>) => {
      toast.error(
        err.response?.data?.error?.message ?? "Failed to create repository"
      );
    },
  });

  const deleteMutation = useMutation({
    mutationFn: () => deleteGitRepo(domainId, deleteTarget!.id),
    onSuccess: () => {
      toast.success("Repository deleted");
      setDeleteTarget(null);
      queryClient.invalidateQueries({ queryKey: ["git-repos", domainId] });
    },
    onError: (err: AxiosError<APIError>) => {
      toast.error(
        err.response?.data?.error?.message ?? "Failed to delete repository"
      );
    },
  });

  const updateMutation = useMutation({
    mutationFn: () =>
      updateGitRepo(domainId, editRepo!.id, {
        branch: editBranch,
        deploy_mode: editDeployMode,
        deploy_path: editDeployPath,
        post_deploy_cmd: editPostDeployCmd,
      }),
    onSuccess: () => {
      toast.success("Repository settings updated");
      setEditRepo(null);
      queryClient.invalidateQueries({ queryKey: ["git-repos", domainId] });
    },
    onError: (err: AxiosError<APIError>) => {
      toast.error(
        err.response?.data?.error?.message ?? "Failed to update settings"
      );
    },
  });

  function resetCreateForm() {
    setNewType("remote");
    setNewRemoteURL("");
    setNewName("");
    setNewBranch("main");
    setNewDeployMode("automatic");
    setNewDeployPath("");
    setNewPostDeployCmd("");
    setShowPostDeploy(false);
  }

  function openEdit(repo: GitRepository) {
    setEditRepo(repo);
    setEditBranch(repo.branch);
    setEditDeployMode(repo.deploy_mode);
    setEditDeployPath(repo.deploy_path);
    setEditPostDeployCmd(repo.post_deploy_cmd ?? "");
  }

  if (isLoading) {
    return (
      <div className="space-y-4 max-w-5xl">
        <Skeleton className="h-12 w-full" />
        <Skeleton className="h-64 w-full" />
      </div>
    );
  }

  const repos = data?.data ?? [];
  const domainName = domain?.name ?? "";

  return (
    <div className="space-y-6 max-w-5xl">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h2 className="text-lg font-semibold flex items-center gap-2">
            <GitBranch className="h-5 w-5 text-pink-500" />
            Git Repositories
          </h2>
          <p className="text-sm text-muted-foreground">
            Deploy code from Git repositories to {domainName || "this domain"}
          </p>
        </div>
        <Button
          onClick={() => {
            resetCreateForm();
            setShowCreate(true);
          }}
          className="bg-pink-500 hover:bg-pink-600"
        >
          <Plus className="h-4 w-4 mr-1.5" />
          Add Repository
        </Button>
      </div>

      {/* Repository Grid */}
      {repos.length === 0 ? (
        <Card className="border-dashed">
          <CardContent className="py-16 text-center">
            <div className="flex h-16 w-16 mx-auto items-center justify-center rounded-full bg-muted mb-4">
              <GitBranch className="h-8 w-8 text-muted-foreground" />
            </div>
            <h3 className="text-sm font-medium mb-1">No repositories yet</h3>
            <p className="text-sm text-muted-foreground max-w-md mx-auto mb-4">
              Connect a remote repository from GitHub, GitLab, or Bitbucket to
              automatically deploy your code, or create a local repository for
              push-to-deploy.
            </p>
            <Button
              variant="outline"
              onClick={() => {
                resetCreateForm();
                setShowCreate(true);
              }}
            >
              <Plus className="h-4 w-4 mr-1.5" />
              Add Repository
            </Button>
          </CardContent>
        </Card>
      ) : (
        <div className="grid grid-cols-1 lg:grid-cols-2 gap-4">
          {repos.map((repo) => (
            <RepoCard
              key={repo.id}
              repo={repo}
              domainId={domainId}
              onEdit={() => openEdit(repo)}
              onDelete={() => setDeleteTarget(repo)}
            />
          ))}
          {/* Add Repo card */}
          <button
            onClick={() => {
              resetCreateForm();
              setShowCreate(true);
            }}
            className="border-2 border-dashed rounded-lg p-8 flex flex-col items-center justify-center gap-2 text-muted-foreground hover:text-pink-500 hover:border-pink-500/30 transition-colors min-h-[200px]"
          >
            <Plus className="h-6 w-6" />
            <span className="text-sm font-medium">Add Repository</span>
          </button>
        </div>
      )}

      {/* Create Repository Dialog */}
      <Dialog open={showCreate} onOpenChange={setShowCreate}>
        <DialogContent className="sm:max-w-lg">
          <DialogHeader>
            <DialogTitle>Create repository</DialogTitle>
            <DialogDescription>
              Connect a Git repository to deploy code to this domain
            </DialogDescription>
          </DialogHeader>

          <div className="space-y-6">
            {/* Code Location */}
            <div className="space-y-3">
              <Label className="text-base font-semibold">Code location</Label>
              <div className="grid grid-cols-2 gap-3">
                <button
                  type="button"
                  onClick={() => setNewType("remote")}
                  className={`flex flex-col gap-2 p-4 rounded-lg border-2 text-left transition-colors ${
                    newType === "remote"
                      ? "border-pink-500 bg-pink-500/5"
                      : "border-muted hover:border-muted-foreground/30"
                  }`}
                >
                  <Globe className="h-5 w-5 text-pink-500" />
                  <div>
                    <p className="text-sm font-medium">Remote repository</p>
                    <p className="text-xs text-muted-foreground mt-0.5">
                      Pull from GitHub, GitLab, or Bitbucket
                    </p>
                  </div>
                </button>
                <button
                  type="button"
                  onClick={() => setNewType("local")}
                  className={`flex flex-col gap-2 p-4 rounded-lg border-2 text-left transition-colors ${
                    newType === "local"
                      ? "border-pink-500 bg-pink-500/5"
                      : "border-muted hover:border-muted-foreground/30"
                  }`}
                >
                  <HardDrive className="h-5 w-5 text-pink-500" />
                  <div>
                    <p className="text-sm font-medium">Local repository</p>
                    <p className="text-xs text-muted-foreground mt-0.5">
                      Push code to the server yourself
                    </p>
                  </div>
                </button>
              </div>
            </div>

            {/* Repository URL (remote only) */}
            {newType === "remote" && (
              <div className="space-y-2">
                <Label>
                  Repository URL <span className="text-red-500">*</span>
                </Label>
                <Input
                  value={newRemoteURL}
                  onChange={(e) => setNewRemoteURL(e.target.value)}
                  placeholder="https://github.com/user/repo.git"
                  autoFocus
                />
                <p className="text-xs text-muted-foreground">
                  Both HTTP(S) and SSH protocols are supported
                </p>
              </div>
            )}

            {/* Repository Name */}
            <div className="space-y-2">
              <Label>
                Repository name <span className="text-red-500">*</span>
              </Label>
              <Input
                value={newName}
                onChange={(e) => setNewName(e.target.value)}
                placeholder="my-project.git"
                autoFocus={newType === "local"}
              />
              <p className="text-xs text-muted-foreground">
                Specify a name unique within a domain
              </p>
            </div>

            <Separator />

            {/* Deployment Settings */}
            <div className="space-y-4">
              <Label className="text-base font-semibold">
                Deployment settings
              </Label>

              {/* Deployment Mode */}
              <div className="space-y-2">
                <Label>
                  Deployment mode <span className="text-red-500">*</span>
                </Label>
                <div className="flex rounded-lg border overflow-hidden">
                  {(["automatic", "manual", "disabled"] as const).map(
                    (mode) => (
                      <button
                        key={mode}
                        type="button"
                        onClick={() => setNewDeployMode(mode)}
                        className={`flex-1 px-4 py-2 text-sm font-medium transition-colors ${
                          newDeployMode === mode
                            ? "bg-foreground text-background"
                            : "hover:bg-muted"
                        }`}
                      >
                        {mode.charAt(0).toUpperCase() + mode.slice(1)}
                      </button>
                    )
                  )}
                </div>
                <p className="text-xs text-muted-foreground">
                  {newDeployMode === "automatic"
                    ? "Files will be deployed automatically on every push or pull."
                    : newDeployMode === "manual"
                      ? "You will manually trigger deployments from this panel."
                      : "Repository will be cloned but not deployed."}
                </p>
              </div>

              {/* Server Path */}
              {newDeployMode !== "disabled" && (
                <div className="space-y-2">
                  <Label>
                    Server path <span className="text-red-500">*</span>
                  </Label>
                  <Input
                    value={newDeployPath}
                    onChange={(e) => setNewDeployPath(e.target.value)}
                    placeholder="Leave empty for document root"
                  />
                  <p className="text-xs text-muted-foreground">
                    Directory on the server where files will be deployed
                  </p>
                </div>
              )}

              {/* Post-deploy actions */}
              {newDeployMode !== "disabled" && (
                <div className="space-y-2">
                  <label className="flex items-center gap-2 cursor-pointer">
                    <Switch
                      checked={showPostDeploy}
                      onCheckedChange={setShowPostDeploy}
                    />
                    <span className="text-sm font-medium">
                      Enable additional deployment actions
                    </span>
                  </label>
                  <p className="text-xs text-muted-foreground">
                    Specify shell commands to run every time upon deployment
                  </p>
                  {showPostDeploy && (
                    <Textarea
                      value={newPostDeployCmd}
                      onChange={(e) => setNewPostDeployCmd(e.target.value)}
                      placeholder="npm install && npm run build"
                      rows={3}
                      className="font-mono text-sm"
                    />
                  )}
                </div>
              )}
            </div>
          </div>

          <DialogFooter>
            <Button variant="outline" onClick={() => setShowCreate(false)}>
              Cancel
            </Button>
            <Button
              onClick={() => createMutation.mutate()}
              disabled={
                !newName ||
                (newType === "remote" && !newRemoteURL) ||
                createMutation.isPending
              }
              className="bg-pink-500 hover:bg-pink-600"
            >
              {createMutation.isPending && (
                <Loader2 className="h-4 w-4 mr-1.5 animate-spin" />
              )}
              {createMutation.isPending ? "Creating..." : "OK"}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* Edit Settings Dialog */}
      <Dialog open={!!editRepo} onOpenChange={() => setEditRepo(null)}>
        <DialogContent className="sm:max-w-lg">
          <DialogHeader>
            <DialogTitle>Repository Settings</DialogTitle>
            <DialogDescription>
              Configure {editRepo?.name}
            </DialogDescription>
          </DialogHeader>
          <div className="space-y-4">
            <div className="space-y-2">
              <Label>Branch</Label>
              <Input
                value={editBranch}
                onChange={(e) => setEditBranch(e.target.value)}
              />
            </div>
            <div className="space-y-2">
              <Label>Deployment Mode</Label>
              <div className="flex rounded-lg border overflow-hidden">
                {(["automatic", "manual", "disabled"] as const).map((mode) => (
                  <button
                    key={mode}
                    type="button"
                    onClick={() => setEditDeployMode(mode)}
                    className={`flex-1 px-4 py-2 text-sm font-medium transition-colors ${
                      editDeployMode === mode
                        ? "bg-foreground text-background"
                        : "hover:bg-muted"
                    }`}
                  >
                    {mode.charAt(0).toUpperCase() + mode.slice(1)}
                  </button>
                ))}
              </div>
            </div>
            <div className="space-y-2">
              <Label>Deploy Path</Label>
              <Input
                value={editDeployPath}
                onChange={(e) => setEditDeployPath(e.target.value)}
              />
            </div>
            <div className="space-y-2">
              <Label>Post-Deploy Command</Label>
              <Textarea
                value={editPostDeployCmd}
                onChange={(e) => setEditPostDeployCmd(e.target.value)}
                placeholder="npm install && npm run build"
                rows={3}
                className="font-mono text-sm"
              />
              <p className="text-xs text-muted-foreground">
                Shell commands to run after files are deployed
              </p>
            </div>
          </div>
          <DialogFooter>
            <Button variant="outline" onClick={() => setEditRepo(null)}>
              Cancel
            </Button>
            <Button
              onClick={() => updateMutation.mutate()}
              disabled={updateMutation.isPending}
              className="bg-pink-500 hover:bg-pink-600"
            >
              {updateMutation.isPending ? "Saving..." : "Save Changes"}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* Delete Confirmation */}
      <ConfirmDialog
        open={!!deleteTarget}
        onOpenChange={() => setDeleteTarget(null)}
        title="Delete Git Repository"
        description={`This will permanently delete "${deleteTarget?.name}" and all its deployment history. The deployed files will remain on the server.`}
        confirmText="Delete"
        typeToConfirm={deleteTarget?.name}
        destructive
        loading={deleteMutation.isPending}
        onConfirm={() => deleteMutation.mutate()}
      />
    </div>
  );
}

// ─── Repo Card (Plesk-style) ─────────────────────

function RepoCard({
  repo,
  domainId,
  onEdit,
  onDelete,
}: {
  repo: GitRepository;
  domainId: number;
  onEdit: () => void;
  onDelete: () => void;
}) {
  const queryClient = useQueryClient();
  const [showDeployments, setShowDeployments] = useState(false);

  const deployMutation = useMutation({
    mutationFn: () => triggerDeploy(domainId, repo.id),
    onSuccess: () => {
      toast.success("Deployment triggered");
      queryClient.invalidateQueries({ queryKey: ["git-repos", domainId] });
      queryClient.invalidateQueries({
        queryKey: ["git-deployments", repo.id],
      });
    },
    onError: (err: AxiosError<APIError>) => {
      toast.error(
        err.response?.data?.error?.message ?? "Failed to trigger deployment"
      );
    },
  });

  const { data: deploymentsData } = useQuery({
    queryKey: ["git-deployments", repo.id],
    queryFn: () => listDeployments(domainId, repo.id, 5),
    enabled: showDeployments,
  });

  const deployments = deploymentsData?.data ?? [];
  const webhookUrl = repo.webhook_secret
    ? `${window.location.origin}/api/git/webhook/${repo.webhook_secret}`
    : null;

  const isRemote = repo.repo_type === "remote";

  return (
    <Card className="flex flex-col">
      {/* Header */}
      <CardHeader className="pb-3">
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-2">
            {isRemote ? (
              <Globe className="h-4 w-4 text-pink-500 shrink-0" />
            ) : (
              <HardDrive className="h-4 w-4 text-pink-500 shrink-0" />
            )}
            <CardTitle className="text-base">{repo.name}</CardTitle>
          </div>
          <DeployModeBadge mode={repo.deploy_mode} />
        </div>
      </CardHeader>

      <CardContent className="flex-1 space-y-4 pt-0">
        {/* URL */}
        {isRemote && repo.remote_url && (
          <div className="space-y-1">
            <p className="text-xs font-medium text-muted-foreground">URL</p>
            <div className="flex items-center gap-1.5">
              <code className="text-xs bg-muted rounded px-2 py-1.5 flex-1 truncate font-mono">
                {repo.remote_url}
              </code>
              <Button
                size="icon"
                variant="ghost"
                className="h-7 w-7 shrink-0"
                onClick={() => copyToClipboard(repo.remote_url!)}
              >
                <Copy className="h-3 w-3" />
              </Button>
            </div>
          </div>
        )}

        {/* Clone URL for local repos */}
        {!isRemote && (
          <div className="space-y-1">
            <p className="text-xs font-medium text-muted-foreground">
              Push URL
            </p>
            <div className="flex items-center gap-1.5">
              <code className="text-xs bg-muted rounded px-2 py-1.5 flex-1 truncate font-mono">
                ssh://pinkpanel@{window.location.hostname}
                /var/lib/pinkpanel/git/{repo.domain_id}/{repo.name}.git
              </code>
              <Button
                size="icon"
                variant="ghost"
                className="h-7 w-7 shrink-0"
                onClick={() =>
                  copyToClipboard(
                    `ssh://pinkpanel@${window.location.hostname}/var/lib/pinkpanel/git/${repo.domain_id}/${repo.name}.git`
                  )
                }
              >
                <Copy className="h-3 w-3" />
              </Button>
            </div>
          </div>
        )}

        {/* Branch */}
        <div className="flex items-center gap-4 text-sm">
          <div>
            <span className="text-xs text-muted-foreground">Branch</span>
            <p className="font-mono text-sm">{repo.branch}</p>
          </div>
          {repo.last_commit && (
            <div>
              <span className="text-xs text-muted-foreground">
                Last commit
              </span>
              <p className="font-mono text-sm">
                {repo.last_commit.slice(0, 7)}
              </p>
            </div>
          )}
        </div>

        {/* Webhook URL */}
        {webhookUrl && repo.deploy_mode === "automatic" && (
          <div className="space-y-1">
            <p className="text-xs font-medium text-muted-foreground flex items-center gap-1">
              <Webhook className="h-3 w-3" />
              Webhook URL
            </p>
            <div className="flex items-center gap-1.5">
              <code className="text-[11px] bg-muted rounded px-2 py-1.5 flex-1 truncate font-mono">
                {webhookUrl}
              </code>
              <Button
                size="icon"
                variant="ghost"
                className="h-7 w-7 shrink-0"
                onClick={() => copyToClipboard(webhookUrl)}
              >
                <Copy className="h-3 w-3" />
              </Button>
            </div>
          </div>
        )}

        <Separator />

        {/* Deployment Info */}
        <div className="space-y-2">
          <div className="flex items-center gap-1.5 text-sm">
            <FolderSync className="h-3.5 w-3.5 text-muted-foreground" />
            <span>
              <span className="font-mono font-medium">{repo.branch}</span>
              <span className="text-muted-foreground">
                {" "}
                branch{" "}
                {repo.deploy_mode === "automatic"
                  ? "automatically"
                  : repo.deploy_mode === "manual"
                    ? "manually"
                    : ""}{" "}
                to{" "}
              </span>
              <span className="font-mono text-pink-500">
                {repo.deploy_path}
              </span>
            </span>
          </div>
          {repo.last_deploy_at && (
            <p className="text-xs text-muted-foreground flex items-center gap-1">
              <Clock className="h-3 w-3" />
              Last deployed{" "}
              {new Date(repo.last_deploy_at).toLocaleString()}
            </p>
          )}
        </div>

        {/* Deployment History Toggle */}
        <button
          onClick={() => setShowDeployments(!showDeployments)}
          className="flex items-center gap-1 text-xs text-pink-500 hover:text-pink-600 font-medium"
        >
          {showDeployments ? (
            <ChevronDown className="h-3 w-3" />
          ) : (
            <ChevronRight className="h-3 w-3" />
          )}
          Deployment history
        </button>

        {showDeployments && (
          <div className="space-y-1.5">
            {deployments.length === 0 ? (
              <p className="text-xs text-muted-foreground py-2">
                No deployments yet
              </p>
            ) : (
              deployments.map((dep) => (
                <DeploymentRow key={dep.id} deployment={dep} />
              ))
            )}
          </div>
        )}
      </CardContent>

      {/* Footer Actions */}
      <div className="border-t px-6 py-3 mt-auto flex items-center justify-between">
        <div className="flex items-center gap-1.5">
          {isRemote && repo.deploy_mode !== "disabled" && (
            <Button
              size="sm"
              variant="outline"
              className="h-8 text-xs"
              onClick={() => deployMutation.mutate()}
              disabled={deployMutation.isPending}
            >
              {deployMutation.isPending ? (
                <Loader2 className="h-3.5 w-3.5 mr-1.5 animate-spin" />
              ) : (
                <Download className="h-3.5 w-3.5 mr-1.5" />
              )}
              Pull now
            </Button>
          )}
          {repo.deploy_mode !== "disabled" && (
            <Button
              size="sm"
              variant="outline"
              className="h-8 text-xs"
              onClick={() => deployMutation.mutate()}
              disabled={deployMutation.isPending}
            >
              {deployMutation.isPending ? (
                <Loader2 className="h-3.5 w-3.5 mr-1.5 animate-spin" />
              ) : (
                <Rocket className="h-3.5 w-3.5 mr-1.5" />
              )}
              Deploy now
            </Button>
          )}
        </div>
        <div className="flex items-center gap-1">
          <Button
            size="icon"
            variant="ghost"
            className="h-8 w-8"
            onClick={onEdit}
            title="Settings"
          >
            <Settings className="h-3.5 w-3.5" />
          </Button>
          <Button
            size="icon"
            variant="ghost"
            className="h-8 w-8 text-destructive hover:text-destructive"
            onClick={onDelete}
            title="Delete"
          >
            <Trash2 className="h-3.5 w-3.5" />
          </Button>
        </div>
      </div>
    </Card>
  );
}

// ─── Helpers ─────────────────────────────────────

function DeployModeBadge({ mode }: { mode: string }) {
  const config: Record<string, { label: string; className: string }> = {
    automatic: {
      label: "Automatic",
      className: "bg-green-500/10 text-green-500 border-green-500/20",
    },
    manual: {
      label: "Manual",
      className: "bg-blue-500/10 text-blue-500 border-blue-500/20",
    },
    disabled: {
      label: "Disabled",
      className: "bg-gray-500/10 text-muted-foreground border-muted",
    },
  };
  const c = config[mode] ?? config.disabled;
  return (
    <Badge variant="outline" className={`text-xs ${c.className}`}>
      {c.label}
    </Badge>
  );
}

function DeploymentRow({ deployment }: { deployment: GitDeployment }) {
  const [showLog, setShowLog] = useState(false);

  return (
    <div className="rounded-lg border text-xs">
      <div
        className="flex items-center justify-between px-3 py-2 cursor-pointer hover:bg-muted/50 transition-colors"
        onClick={() => deployment.log && setShowLog(!showLog)}
      >
        <div className="flex items-center gap-2">
          {deployment.status === "completed" ? (
            <CheckCircle2 className="h-3.5 w-3.5 text-green-500" />
          ) : deployment.status === "running" ? (
            <Loader2 className="h-3.5 w-3.5 text-blue-500 animate-spin" />
          ) : deployment.status === "failed" ? (
            <XCircle className="h-3.5 w-3.5 text-red-500" />
          ) : (
            <Clock className="h-3.5 w-3.5 text-muted-foreground" />
          )}
          <span className="capitalize font-medium">{deployment.status}</span>
          {deployment.commit_hash && (
            <span className="font-mono text-muted-foreground">
              {deployment.commit_hash.slice(0, 7)}
            </span>
          )}
          {deployment.duration_ms != null && (
            <span className="text-muted-foreground">
              {(deployment.duration_ms / 1000).toFixed(1)}s
            </span>
          )}
        </div>
        <span className="text-muted-foreground">
          {new Date(deployment.created_at).toLocaleString()}
        </span>
      </div>
      {showLog && deployment.log && (
        <pre className="px-3 pb-3 text-[10px] bg-[oklch(0.13_0_0)] text-[oklch(0.8_0_0)] rounded-b-lg p-3 overflow-x-auto max-h-40 whitespace-pre-wrap font-mono leading-relaxed">
          {deployment.log}
        </pre>
      )}
    </div>
  );
}
