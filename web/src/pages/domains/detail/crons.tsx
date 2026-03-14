import { useState } from "react";
import { useParams } from "react-router-dom";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import {
  Card,
  CardContent,
} from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Badge } from "@/components/ui/badge";
import { Skeleton } from "@/components/ui/skeleton";
import { Switch } from "@/components/ui/switch";
import { Textarea } from "@/components/ui/textarea";
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
  listCronJobs,
  createCronJob,
  updateCronJob,
  deleteCronJob,
  runCronJob,
  getCronLogs,
} from "@/api/cron";
import type { CronJob, CronLog } from "@/types/cron";
import type { AxiosError } from "axios";
import type { APIError } from "@/types/api";
import {
  Clock,
  Plus,
  Trash2,
  Pencil,
  Play,
  ScrollText,
  Loader2,
  CheckCircle2,
  XCircle,
  Terminal,
} from "lucide-react";

const SCHEDULE_PRESETS = [
  { label: "Every minute", value: "* * * * *" },
  { label: "Every 5 minutes", value: "*/5 * * * *" },
  { label: "Every 15 minutes", value: "*/15 * * * *" },
  { label: "Every 30 minutes", value: "*/30 * * * *" },
  { label: "Hourly", value: "0 * * * *" },
  { label: "Daily (midnight)", value: "0 0 * * *" },
  { label: "Weekly (Sunday)", value: "0 0 * * 0" },
  { label: "Monthly (1st)", value: "0 0 1 * *" },
  { label: "Custom", value: "custom" },
];

function describeSchedule(schedule: string): string {
  const presets: Record<string, string> = {
    "* * * * *": "Every minute",
    "*/5 * * * *": "Every 5 minutes",
    "*/15 * * * *": "Every 15 minutes",
    "*/30 * * * *": "Every 30 minutes",
    "0 * * * *": "Hourly",
    "0 0 * * *": "Daily at midnight",
    "0 0 * * 0": "Weekly on Sunday",
    "0 0 1 * *": "Monthly on the 1st",
  };
  return presets[schedule] || schedule;
}

export function DomainCrons() {
  const { id } = useParams<{ id: string }>();
  const domainId = Number(id);
  const queryClient = useQueryClient();

  const [showCreate, setShowCreate] = useState(false);
  const [editJob, setEditJob] = useState<CronJob | null>(null);
  const [deleteTarget, setDeleteTarget] = useState<CronJob | null>(null);
  const [runResult, setRunResult] = useState<{
    exit_code: number;
    output: string;
    duration_ms: number;
  } | null>(null);
  const [logsJob, setLogsJob] = useState<CronJob | null>(null);

  // Create form
  const [preset, setPreset] = useState("0 * * * *");
  const [schedule, setSchedule] = useState("0 * * * *");
  const [command, setCommand] = useState("");
  const [description, setDescription] = useState("");

  // Edit form
  const [editPreset, setEditPreset] = useState("custom");
  const [editSchedule, setEditSchedule] = useState("");
  const [editCommand, setEditCommand] = useState("");
  const [editDescription, setEditDescription] = useState("");

  const { data, isLoading } = useQuery({
    queryKey: ["cron-jobs", domainId],
    queryFn: () => listCronJobs(domainId),
    enabled: !!domainId,
  });
  const jobs = data?.data ?? [];

  const { data: logsData, isLoading: logsLoading } = useQuery({
    queryKey: ["cron-logs", logsJob?.id],
    queryFn: () => getCronLogs(logsJob!.id),
    enabled: !!logsJob,
  });
  const logs: CronLog[] = logsData?.data ?? [];

  const createMut = useMutation({
    mutationFn: () =>
      createCronJob(domainId, { schedule, command, description }),
    onSuccess: () => {
      toast.success("Cron job created");
      queryClient.invalidateQueries({ queryKey: ["cron-jobs", domainId] });
      resetCreateForm();
    },
    onError: (e: AxiosError<APIError>) =>
      toast.error(e.response?.data?.error?.message || "Failed to create cron job"),
  });

  const updateMut = useMutation({
    mutationFn: () =>
      updateCronJob(editJob!.id, {
        schedule: editSchedule,
        command: editCommand,
        description: editDescription,
      }),
    onSuccess: () => {
      toast.success("Cron job updated");
      queryClient.invalidateQueries({ queryKey: ["cron-jobs", domainId] });
      setEditJob(null);
    },
    onError: (e: AxiosError<APIError>) =>
      toast.error(e.response?.data?.error?.message || "Failed to update cron job"),
  });

  const toggleMut = useMutation({
    mutationFn: ({ id, enabled }: { id: number; enabled: boolean }) =>
      updateCronJob(id, { enabled }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["cron-jobs", domainId] });
    },
    onError: (e: AxiosError<APIError>) =>
      toast.error(e.response?.data?.error?.message || "Failed to toggle cron job"),
  });

  const deleteMut = useMutation({
    mutationFn: (id: number) => deleteCronJob(id),
    onSuccess: () => {
      toast.success("Cron job deleted");
      queryClient.invalidateQueries({ queryKey: ["cron-jobs", domainId] });
      setDeleteTarget(null);
    },
    onError: (e: AxiosError<APIError>) =>
      toast.error(e.response?.data?.error?.message || "Failed to delete cron job"),
  });

  const [runningId, setRunningId] = useState<number | null>(null);
  const runMut = useMutation({
    mutationFn: (id: number) => {
      setRunningId(id);
      return runCronJob(id);
    },
    onSuccess: (data) => {
      setRunResult(data);
      setRunningId(null);
      queryClient.invalidateQueries({ queryKey: ["cron-logs"] });
    },
    onError: (e: AxiosError<APIError>) => {
      toast.error(e.response?.data?.error?.message || "Failed to run cron job");
      setRunningId(null);
    },
  });

  function resetCreateForm() {
    setShowCreate(false);
    setPreset("0 * * * *");
    setSchedule("0 * * * *");
    setCommand("");
    setDescription("");
  }

  function openEdit(job: CronJob) {
    setEditJob(job);
    setEditSchedule(job.schedule);
    setEditCommand(job.command);
    setEditDescription(job.description);
    const match = SCHEDULE_PRESETS.find((p) => p.value === job.schedule);
    setEditPreset(match ? match.value : "custom");
  }

  if (isLoading) {
    return <Skeleton className="h-64 w-full" />;
  }

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-2">
          <Clock className="h-5 w-5 text-pink-500" />
          <h2 className="text-lg font-semibold">Cron Jobs</h2>
          {jobs.length > 0 && (
            <Badge variant="secondary">{jobs.length}</Badge>
          )}
        </div>
        <Button size="sm" onClick={() => setShowCreate(true)}>
          <Plus className="h-4 w-4 mr-1" />
          Add Cron Job
        </Button>
      </div>

      {/* List */}
      {jobs.length === 0 ? (
        <Card>
          <CardContent className="flex flex-col items-center justify-center py-12 text-center">
            <div className="rounded-full bg-muted p-3 mb-3">
              <Clock className="h-6 w-6 text-muted-foreground" />
            </div>
            <p className="text-sm text-muted-foreground">
              No cron jobs configured
            </p>
            <p className="text-xs text-muted-foreground mt-1">
              Schedule recurring tasks like PHP scripts, cleanup commands, or
              report generation.
            </p>
          </CardContent>
        </Card>
      ) : (
        <div className="space-y-3">
          {jobs.map((job) => (
            <Card key={job.id}>
              <CardContent className="flex items-center justify-between py-4 px-5">
                <div className="min-w-0 flex-1 space-y-1">
                  <div className="flex items-center gap-2">
                    <span className="font-medium text-sm truncate">
                      {job.description || "Untitled"}
                    </span>
                    {!job.enabled && (
                      <Badge variant="outline" className="text-xs">
                        Disabled
                      </Badge>
                    )}
                  </div>
                  <div className="flex items-center gap-3 text-xs text-muted-foreground">
                    <span className="flex items-center gap-1">
                      <Clock className="h-3 w-3" />
                      {describeSchedule(job.schedule)}
                    </span>
                    <code className="bg-muted px-1.5 py-0.5 rounded text-[11px] max-w-[300px] truncate">
                      {job.command}
                    </code>
                  </div>
                </div>
                <div className="flex items-center gap-2 ml-4 shrink-0">
                  <Switch
                    checked={job.enabled}
                    onCheckedChange={(checked: boolean) =>
                      toggleMut.mutate({ id: job.id, enabled: checked })
                    }
                  />
                  <Button
                    variant="ghost"
                    size="icon"
                    className="h-8 w-8"
                    onClick={() => runMut.mutate(job.id)}
                    disabled={runningId === job.id}
                    title="Run now"
                  >
                    {runningId === job.id ? (
                      <Loader2 className="h-4 w-4 animate-spin" />
                    ) : (
                      <Play className="h-4 w-4" />
                    )}
                  </Button>
                  <Button
                    variant="ghost"
                    size="icon"
                    className="h-8 w-8"
                    onClick={() => setLogsJob(job)}
                    title="View logs"
                  >
                    <ScrollText className="h-4 w-4" />
                  </Button>
                  <Button
                    variant="ghost"
                    size="icon"
                    className="h-8 w-8"
                    onClick={() => openEdit(job)}
                    title="Edit"
                  >
                    <Pencil className="h-4 w-4" />
                  </Button>
                  <Button
                    variant="ghost"
                    size="icon"
                    className="h-8 w-8 text-destructive"
                    onClick={() => setDeleteTarget(job)}
                    title="Delete"
                  >
                    <Trash2 className="h-4 w-4" />
                  </Button>
                </div>
              </CardContent>
            </Card>
          ))}
        </div>
      )}

      {/* Create Dialog */}
      <Dialog open={showCreate} onOpenChange={(v) => !v && resetCreateForm()}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Add Cron Job</DialogTitle>
            <DialogDescription>
              Schedule a recurring command for this domain.
            </DialogDescription>
          </DialogHeader>
          <div className="space-y-4">
            <div className="space-y-2">
              <Label>Description</Label>
              <Input
                placeholder="e.g., Run daily cleanup"
                value={description}
                onChange={(e) => setDescription(e.target.value)}
              />
            </div>
            <div className="space-y-2">
              <Label>Schedule</Label>
              <Select
                value={preset}
                onValueChange={(v) => {
                  if (!v) return;
                  setPreset(v);
                  if (v !== "custom") setSchedule(v);
                }}
              >
                <SelectTrigger>
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  {SCHEDULE_PRESETS.map((p) => (
                    <SelectItem key={p.value} value={p.value}>
                      {p.label}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
              {preset === "custom" && (
                <Input
                  placeholder="* * * * * (min hour day month weekday)"
                  value={schedule}
                  onChange={(e) => setSchedule(e.target.value)}
                  className="font-mono text-sm"
                />
              )}
              {preset !== "custom" && (
                <p className="text-xs text-muted-foreground font-mono">
                  {schedule}
                </p>
              )}
            </div>
            <div className="space-y-2">
              <Label>Command</Label>
              <Textarea
                placeholder="e.g., php /var/www/domain.com/cron.php"
                value={command}
                onChange={(e) => setCommand(e.target.value)}
                className="font-mono text-sm"
                rows={3}
              />
            </div>
          </div>
          <DialogFooter>
            <Button variant="outline" onClick={resetCreateForm}>
              Cancel
            </Button>
            <Button
              onClick={() => createMut.mutate()}
              disabled={createMut.isPending || !schedule || !command}
            >
              {createMut.isPending && (
                <Loader2 className="h-4 w-4 mr-1 animate-spin" />
              )}
              Create
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* Edit Dialog */}
      <Dialog open={!!editJob} onOpenChange={(v) => !v && setEditJob(null)}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Edit Cron Job</DialogTitle>
          </DialogHeader>
          <div className="space-y-4">
            <div className="space-y-2">
              <Label>Description</Label>
              <Input
                value={editDescription}
                onChange={(e) => setEditDescription(e.target.value)}
              />
            </div>
            <div className="space-y-2">
              <Label>Schedule</Label>
              <Select
                value={editPreset}
                onValueChange={(v) => {
                  if (!v) return;
                  setEditPreset(v);
                  if (v !== "custom") setEditSchedule(v);
                }}
              >
                <SelectTrigger>
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  {SCHEDULE_PRESETS.map((p) => (
                    <SelectItem key={p.value} value={p.value}>
                      {p.label}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
              {editPreset === "custom" && (
                <Input
                  placeholder="* * * * *"
                  value={editSchedule}
                  onChange={(e) => setEditSchedule(e.target.value)}
                  className="font-mono text-sm"
                />
              )}
              {editPreset !== "custom" && (
                <p className="text-xs text-muted-foreground font-mono">
                  {editSchedule}
                </p>
              )}
            </div>
            <div className="space-y-2">
              <Label>Command</Label>
              <Textarea
                value={editCommand}
                onChange={(e) => setEditCommand(e.target.value)}
                className="font-mono text-sm"
                rows={3}
              />
            </div>
          </div>
          <DialogFooter>
            <Button variant="outline" onClick={() => setEditJob(null)}>
              Cancel
            </Button>
            <Button
              onClick={() => updateMut.mutate()}
              disabled={updateMut.isPending || !editSchedule || !editCommand}
            >
              {updateMut.isPending && (
                <Loader2 className="h-4 w-4 mr-1 animate-spin" />
              )}
              Save
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* Delete Confirm */}
      <ConfirmDialog
        open={!!deleteTarget}
        onOpenChange={(v) => !v && setDeleteTarget(null)}
        title="Delete Cron Job"
        description={`Are you sure you want to delete "${deleteTarget?.description || "this cron job"}"? This action cannot be undone.`}
        confirmText="Delete"
        destructive
        loading={deleteMut.isPending}
        onConfirm={() => deleteTarget && deleteMut.mutate(deleteTarget.id)}
      />

      {/* Run Result Dialog */}
      <Dialog open={!!runResult} onOpenChange={(v) => !v && setRunResult(null)}>
        <DialogContent className="max-w-lg">
          <DialogHeader>
            <DialogTitle className="flex items-center gap-2">
              <Terminal className="h-4 w-4" />
              Execution Result
            </DialogTitle>
          </DialogHeader>
          {runResult && (
            <div className="space-y-3">
              <div className="flex items-center gap-3">
                {runResult.exit_code === 0 ? (
                  <Badge className="bg-green-500/10 text-green-500 border-green-500/20">
                    <CheckCircle2 className="h-3 w-3 mr-1" />
                    Success
                  </Badge>
                ) : (
                  <Badge variant="destructive">
                    <XCircle className="h-3 w-3 mr-1" />
                    Exit code {runResult.exit_code}
                  </Badge>
                )}
                <span className="text-xs text-muted-foreground">
                  {runResult.duration_ms}ms
                </span>
              </div>
              <pre className="bg-muted rounded-md p-3 text-xs font-mono overflow-auto max-h-64 whitespace-pre-wrap">
                {runResult.output || "(no output)"}
              </pre>
            </div>
          )}
          <DialogFooter>
            <Button variant="outline" onClick={() => setRunResult(null)}>
              Close
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* Logs Dialog */}
      <Dialog open={!!logsJob} onOpenChange={(v) => !v && setLogsJob(null)}>
        <DialogContent className="max-w-2xl">
          <DialogHeader>
            <DialogTitle className="flex items-center gap-2">
              <ScrollText className="h-4 w-4" />
              Execution Logs — {logsJob?.description || "Cron Job"}
            </DialogTitle>
          </DialogHeader>
          {logsLoading ? (
            <Skeleton className="h-32 w-full" />
          ) : logs.length === 0 ? (
            <p className="text-sm text-muted-foreground text-center py-8">
              No execution logs yet.
            </p>
          ) : (
            <div className="max-h-96 overflow-auto space-y-2">
              {logs.map((l) => (
                <LogEntry key={l.id} log={l} />
              ))}
            </div>
          )}
          <DialogFooter>
            <Button variant="outline" onClick={() => setLogsJob(null)}>
              Close
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  );
}

function LogEntry({ log }: { log: CronLog }) {
  const [expanded, setExpanded] = useState(false);

  return (
    <div className="border rounded-md p-3 text-sm">
      <div
        className="flex items-center justify-between cursor-pointer"
        onClick={() => setExpanded(!expanded)}
      >
        <div className="flex items-center gap-2">
          {log.exit_code === 0 ? (
            <CheckCircle2 className="h-3.5 w-3.5 text-green-500" />
          ) : (
            <XCircle className="h-3.5 w-3.5 text-destructive" />
          )}
          <span className="text-xs text-muted-foreground">
            {new Date(log.started_at).toLocaleString()}
          </span>
          <span className="text-xs text-muted-foreground">
            {log.duration_ms}ms
          </span>
          {log.exit_code !== 0 && (
            <Badge variant="outline" className="text-xs">
              exit {log.exit_code}
            </Badge>
          )}
        </div>
        <span className="text-xs text-muted-foreground">
          {expanded ? "▲" : "▼"}
        </span>
      </div>
      {expanded && (
        <pre className="mt-2 bg-muted rounded p-2 text-xs font-mono overflow-auto max-h-40 whitespace-pre-wrap">
          {log.output || "(no output)"}
        </pre>
      )}
    </div>
  );
}
