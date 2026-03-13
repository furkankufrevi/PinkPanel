import { useState } from "react";
import { useParams } from "react-router-dom";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import {
  Card,
  CardContent,
  CardHeader,
  CardTitle,
  CardDescription,
} from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Badge } from "@/components/ui/badge";
import { Skeleton } from "@/components/ui/skeleton";
import { Textarea } from "@/components/ui/textarea";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
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
import {
  GitBranch,
  Plus,
  Trash2,
  Play,
  Clock,
  ExternalLink,
  Copy,
  Settings,
  ChevronDown,
  ChevronUp,
  CheckCircle2,
  XCircle,
  Loader2,
  Globe,
  HardDrive,
} from "lucide-react";

export function DomainGit() {
  const { id } = useParams<{ id: string }>();
  const domainId = Number(id);
  const queryClient = useQueryClient();

  const [showCreate, setShowCreate] = useState(false);
  const [deleteRepo, setDeleteRepo] = useState<GitRepository | null>(null);
  const [editRepo, setEditRepo] = useState<GitRepository | null>(null);
  const [expandedRepo, setExpandedRepo] = useState<number | null>(null);

  // Create form state
  const [newName, setNewName] = useState("");
  const [newType, setNewType] = useState<"remote" | "local">("remote");
  const [newRemoteURL, setNewRemoteURL] = useState("");
  const [newBranch, setNewBranch] = useState("main");
  const [newDeployMode, setNewDeployMode] = useState("manual");
  const [newDeployPath, setNewDeployPath] = useState("");

  // Edit form state
  const [editBranch, setEditBranch] = useState("");
  const [editDeployMode, setEditDeployMode] = useState("");
  const [editDeployPath, setEditDeployPath] = useState("");
  const [editPostDeployCmd, setEditPostDeployCmd] = useState("");

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
      toast.success("Git repository created");
      setShowCreate(false);
      resetCreateForm();
      queryClient.invalidateQueries({ queryKey: ["git-repos"] });
    },
    onError: (err: AxiosError<APIError>) => {
      toast.error(
        err.response?.data?.error?.message ?? "Failed to create repository"
      );
    },
  });

  const deleteMutation = useMutation({
    mutationFn: () => deleteGitRepo(domainId, deleteRepo!.id),
    onSuccess: () => {
      toast.success("Git repository deleted");
      setDeleteRepo(null);
      queryClient.invalidateQueries({ queryKey: ["git-repos"] });
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
      queryClient.invalidateQueries({ queryKey: ["git-repos"] });
    },
    onError: (err: AxiosError<APIError>) => {
      toast.error(
        err.response?.data?.error?.message ?? "Failed to update repository"
      );
    },
  });

  function resetCreateForm() {
    setNewName("");
    setNewType("remote");
    setNewRemoteURL("");
    setNewBranch("main");
    setNewDeployMode("manual");
    setNewDeployPath("");
  }

  function openEdit(repo: GitRepository) {
    setEditRepo(repo);
    setEditBranch(repo.branch);
    setEditDeployMode(repo.deploy_mode);
    setEditDeployPath(repo.deploy_path);
    setEditPostDeployCmd(repo.post_deploy_cmd ?? "");
  }

  if (isLoading) {
    return <Skeleton className="h-48 w-full max-w-3xl" />;
  }

  const repos = data?.data ?? [];

  return (
    <div className="space-y-4 max-w-3xl">
      <div className="flex items-center justify-between">
        <h3 className="text-lg font-medium">Git Repositories</h3>
        <Button
          size="sm"
          onClick={() => {
            resetCreateForm();
            setShowCreate(true);
          }}
          className="bg-pink-500 hover:bg-pink-600"
        >
          <Plus className="h-4 w-4 mr-1" />
          Add Repository
        </Button>
      </div>

      {repos.length === 0 ? (
        <Card>
          <CardContent className="py-8 text-center">
            <GitBranch className="h-10 w-10 mx-auto text-muted-foreground mb-3" />
            <p className="text-sm text-muted-foreground">
              No git repositories configured for this domain
            </p>
            <p className="text-xs text-muted-foreground mt-1">
              Add a remote repository to pull from or a local repository for
              push-to-deploy
            </p>
          </CardContent>
        </Card>
      ) : (
        <div className="space-y-3">
          {repos.map((repo) => (
            <RepoCard
              key={repo.id}
              repo={repo}
              domainId={domainId}
              expanded={expandedRepo === repo.id}
              onToggle={() =>
                setExpandedRepo(expandedRepo === repo.id ? null : repo.id)
              }
              onEdit={() => openEdit(repo)}
              onDelete={() => setDeleteRepo(repo)}
            />
          ))}
        </div>
      )}

      {/* Create Dialog */}
      <Dialog open={showCreate} onOpenChange={setShowCreate}>
        <DialogContent className="max-w-lg">
          <DialogHeader>
            <DialogTitle>Add Git Repository</DialogTitle>
            <DialogDescription>
              Connect a remote repository or create a local push-to-deploy repo
            </DialogDescription>
          </DialogHeader>
          <div className="space-y-4">
            <div className="space-y-2">
              <Label>Repository Type</Label>
              <Select
                value={newType}
                onValueChange={(v) =>
                  v && setNewType(v as "remote" | "local")
                }
              >
                <SelectTrigger>
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="remote">
                    Remote (Pull from URL)
                  </SelectItem>
                  <SelectItem value="local">
                    Local (Push to Server)
                  </SelectItem>
                </SelectContent>
              </Select>
            </div>
            <div className="space-y-2">
              <Label>Repository Name</Label>
              <Input
                value={newName}
                onChange={(e) => setNewName(e.target.value)}
                placeholder="my-project"
                autoFocus
              />
            </div>
            {newType === "remote" && (
              <div className="space-y-2">
                <Label>Remote URL</Label>
                <Input
                  value={newRemoteURL}
                  onChange={(e) => setNewRemoteURL(e.target.value)}
                  placeholder="https://github.com/user/repo.git"
                />
              </div>
            )}
            <div className="space-y-2">
              <Label>Branch</Label>
              <Input
                value={newBranch}
                onChange={(e) => setNewBranch(e.target.value)}
                placeholder="main"
              />
            </div>
            <div className="space-y-2">
              <Label>Deployment Mode</Label>
              <Select
                value={newDeployMode}
                onValueChange={(v) => v && setNewDeployMode(v)}
              >
                <SelectTrigger>
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="automatic">Automatic</SelectItem>
                  <SelectItem value="manual">Manual</SelectItem>
                  <SelectItem value="disabled">Disabled</SelectItem>
                </SelectContent>
              </Select>
            </div>
            <div className="space-y-2">
              <Label>Deploy Path</Label>
              <Input
                value={newDeployPath}
                onChange={(e) => setNewDeployPath(e.target.value)}
                placeholder="Leave empty for domain document root"
              />
              <p className="text-xs text-muted-foreground">
                Where files will be deployed to on the server
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
                !newName ||
                (newType === "remote" && !newRemoteURL) ||
                createMutation.isPending
              }
              className="bg-pink-500 hover:bg-pink-600"
            >
              {createMutation.isPending ? "Creating..." : "Create"}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* Edit Dialog */}
      <Dialog open={!!editRepo} onOpenChange={() => setEditRepo(null)}>
        <DialogContent className="max-w-lg">
          <DialogHeader>
            <DialogTitle>Repository Settings</DialogTitle>
            <DialogDescription>
              Update settings for {editRepo?.name}
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
              <Select
                value={editDeployMode}
                onValueChange={(v) => v && setEditDeployMode(v)}
              >
                <SelectTrigger>
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="automatic">Automatic</SelectItem>
                  <SelectItem value="manual">Manual</SelectItem>
                  <SelectItem value="disabled">Disabled</SelectItem>
                </SelectContent>
              </Select>
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
              />
              <p className="text-xs text-muted-foreground">
                Command to run after files are deployed
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
              {updateMutation.isPending ? "Saving..." : "Save"}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* Delete Confirmation */}
      <ConfirmDialog
        open={!!deleteRepo}
        onOpenChange={() => setDeleteRepo(null)}
        title="Delete Git Repository"
        description={`Permanently delete repository "${deleteRepo?.name}"? This will remove the repository and all deployment history.`}
        confirmText="Delete"
        typeToConfirm={deleteRepo?.name}
        destructive
        loading={deleteMutation.isPending}
        onConfirm={() => deleteMutation.mutate()}
      />
    </div>
  );
}

function RepoCard({
  repo,
  domainId,
  expanded,
  onToggle,
  onEdit,
  onDelete,
}: {
  repo: GitRepository;
  domainId: number;
  expanded: boolean;
  onToggle: () => void;
  onEdit: () => void;
  onDelete: () => void;
}) {
  const queryClient = useQueryClient();

  const deployMutation = useMutation({
    mutationFn: () => triggerDeploy(domainId, repo.id),
    onSuccess: () => {
      toast.success("Deployment triggered");
      queryClient.invalidateQueries({ queryKey: ["git-repos"] });
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
    queryFn: () => listDeployments(domainId, repo.id, 10),
    enabled: expanded,
  });

  const deployments = deploymentsData?.data ?? [];
  const webhookUrl = repo.webhook_secret
    ? `${window.location.origin}/api/git/webhook/${repo.webhook_secret}`
    : null;

  return (
    <Card>
      <CardHeader className="py-3">
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-2 min-w-0">
            <GitBranch className="h-4 w-4 text-pink-500 shrink-0" />
            <CardTitle className="text-sm truncate">{repo.name}</CardTitle>
            <Badge
              variant="outline"
              className="text-xs shrink-0"
            >
              {repo.repo_type === "remote" ? (
                <Globe className="h-3 w-3 mr-1" />
              ) : (
                <HardDrive className="h-3 w-3 mr-1" />
              )}
              {repo.repo_type === "remote" ? "Remote" : "Local"}
            </Badge>
            <DeployModeBadge mode={repo.deploy_mode} />
          </div>
          <div className="flex items-center gap-1 shrink-0">
            {repo.deploy_mode !== "disabled" && (
              <Button
                size="sm"
                variant="outline"
                className="h-7 text-xs"
                onClick={() => deployMutation.mutate()}
                disabled={deployMutation.isPending}
              >
                {deployMutation.isPending ? (
                  <Loader2 className="h-3 w-3 mr-1 animate-spin" />
                ) : (
                  <Play className="h-3 w-3 mr-1" />
                )}
                Deploy
              </Button>
            )}
            <Button
              size="icon"
              variant="ghost"
              className="h-7 w-7"
              onClick={onEdit}
            >
              <Settings className="h-3 w-3" />
            </Button>
            <Button
              size="icon"
              variant="ghost"
              className="h-7 w-7 text-destructive"
              onClick={onDelete}
            >
              <Trash2 className="h-3 w-3" />
            </Button>
            <Button
              size="icon"
              variant="ghost"
              className="h-7 w-7"
              onClick={onToggle}
            >
              {expanded ? (
                <ChevronUp className="h-3 w-3" />
              ) : (
                <ChevronDown className="h-3 w-3" />
              )}
            </Button>
          </div>
        </div>
        <CardDescription className="text-xs mt-1 flex items-center gap-3 flex-wrap">
          <span className="font-mono">{repo.branch}</span>
          {repo.remote_url && (
            <span className="flex items-center gap-1 truncate max-w-xs">
              <ExternalLink className="h-3 w-3 shrink-0" />
              {repo.remote_url}
            </span>
          )}
          {repo.last_deploy_at && (
            <span className="flex items-center gap-1">
              <Clock className="h-3 w-3" />
              {new Date(repo.last_deploy_at).toLocaleDateString()}
            </span>
          )}
          {repo.last_commit && (
            <span className="font-mono text-[10px]">
              {repo.last_commit.slice(0, 7)}
            </span>
          )}
        </CardDescription>
      </CardHeader>

      {expanded && (
        <CardContent className="pt-0 space-y-3">
          {/* Webhook URL */}
          {webhookUrl && repo.deploy_mode === "automatic" && (
            <div className="rounded-md bg-muted p-3 space-y-1">
              <p className="text-xs font-medium">Webhook URL</p>
              <div className="flex items-center gap-2">
                <code className="text-xs bg-background rounded px-2 py-1 flex-1 truncate font-mono">
                  {webhookUrl}
                </code>
                <Button
                  size="icon"
                  variant="ghost"
                  className="h-7 w-7 shrink-0"
                  onClick={() => {
                    navigator.clipboard.writeText(webhookUrl);
                    toast.success("Webhook URL copied");
                  }}
                >
                  <Copy className="h-3 w-3" />
                </Button>
              </div>
            </div>
          )}

          {/* Clone URL for local repos */}
          {repo.repo_type === "local" && (
            <div className="rounded-md bg-muted p-3 space-y-1">
              <p className="text-xs font-medium">Clone URL</p>
              <div className="flex items-center gap-2">
                <code className="text-xs bg-background rounded px-2 py-1 flex-1 truncate font-mono">
                  {`ssh://pinkpanel@${window.location.hostname}/var/lib/pinkpanel/git/${repo.domain_id}/${repo.name}.git`}
                </code>
                <Button
                  size="icon"
                  variant="ghost"
                  className="h-7 w-7 shrink-0"
                  onClick={() => {
                    navigator.clipboard.writeText(
                      `ssh://pinkpanel@${window.location.hostname}/var/lib/pinkpanel/git/${repo.domain_id}/${repo.name}.git`
                    );
                    toast.success("Clone URL copied");
                  }}
                >
                  <Copy className="h-3 w-3" />
                </Button>
              </div>
            </div>
          )}

          {/* Deploy path */}
          <div className="text-xs text-muted-foreground">
            Deploy path:{" "}
            <span className="font-mono">{repo.deploy_path}</span>
          </div>

          {/* Deployment History */}
          <div>
            <p className="text-xs font-medium mb-2">Deployment History</p>
            {deployments.length === 0 ? (
              <p className="text-xs text-muted-foreground">
                No deployments yet
              </p>
            ) : (
              <div className="space-y-1">
                {deployments.map((dep) => (
                  <DeploymentRow key={dep.id} deployment={dep} />
                ))}
              </div>
            )}
          </div>
        </CardContent>
      )}
    </Card>
  );
}

function DeployModeBadge({ mode }: { mode: string }) {
  const colors: Record<string, string> = {
    automatic: "bg-green-500/10 text-green-500 border-green-500/20",
    manual: "bg-blue-500/10 text-blue-500 border-blue-500/20",
    disabled: "bg-gray-500/10 text-gray-500 border-gray-500/20",
  };
  return (
    <Badge variant="outline" className={`text-[10px] ${colors[mode] ?? ""}`}>
      {mode}
    </Badge>
  );
}

function DeploymentRow({ deployment }: { deployment: GitDeployment }) {
  const [showLog, setShowLog] = useState(false);

  const statusIcon = {
    pending: <Clock className="h-3 w-3 text-muted-foreground" />,
    running: <Loader2 className="h-3 w-3 text-blue-500 animate-spin" />,
    completed: <CheckCircle2 className="h-3 w-3 text-green-500" />,
    failed: <XCircle className="h-3 w-3 text-red-500" />,
  }[deployment.status];

  return (
    <div className="rounded border px-3 py-2">
      <div
        className="flex items-center justify-between cursor-pointer"
        onClick={() => deployment.log && setShowLog(!showLog)}
      >
        <div className="flex items-center gap-2 text-xs">
          {statusIcon}
          <span className="capitalize">{deployment.status}</span>
          {deployment.commit_hash && (
            <span className="font-mono text-muted-foreground">
              {deployment.commit_hash.slice(0, 7)}
            </span>
          )}
          <span className="text-muted-foreground">
            {deployment.triggered_by}
          </span>
        </div>
        <div className="flex items-center gap-2 text-xs text-muted-foreground">
          {deployment.duration_ms != null && (
            <span>{(deployment.duration_ms / 1000).toFixed(1)}s</span>
          )}
          <span>{new Date(deployment.created_at).toLocaleString()}</span>
        </div>
      </div>
      {showLog && deployment.log && (
        <pre className="mt-2 text-[10px] bg-muted rounded p-2 overflow-x-auto max-h-40 whitespace-pre-wrap font-mono">
          {deployment.log}
        </pre>
      )}
    </div>
  );
}
