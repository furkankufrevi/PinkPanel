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
import { Badge } from "@/components/ui/badge";
import { Skeleton } from "@/components/ui/skeleton";
import { ConfirmDialog } from "@/components/shared/confirm-dialog";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Switch } from "@/components/ui/switch";
import { toast } from "sonner";
import {
  listBackups,
  createBackup,
  deleteBackup,
  restoreBackup,
  downloadBackup,
  listSchedules,
  createSchedule,
  updateSchedule,
  deleteSchedule,
} from "@/api/backups";
import type { Backup, BackupSchedule } from "@/types/backup";
import type { AxiosError } from "axios";
import type { APIError } from "@/types/api";
import {
  Archive,
  Plus,
  Trash2,
  RotateCcw,
  Download,
  HardDrive,
  Clock,
  CalendarClock,
  CheckCircle2,
  XCircle,
  Loader2,
} from "lucide-react";

function formatSize(bytes: number): string {
  if (bytes === 0) return "0 B";
  const units = ["B", "KB", "MB", "GB"];
  const i = Math.floor(Math.log(bytes) / Math.log(1024));
  return `${(bytes / Math.pow(1024, i)).toFixed(i > 0 ? 1 : 0)} ${units[i]}`;
}

function formatDate(dateStr: string): string {
  return new Date(dateStr).toLocaleString();
}

function StatusBadge({ status }: { status: string }) {
  switch (status) {
    case "completed":
      return (
        <Badge className="bg-green-500/10 text-green-500 border-green-500/20">
          <CheckCircle2 className="h-3 w-3 mr-1" />
          Completed
        </Badge>
      );
    case "running":
      return (
        <Badge className="bg-blue-500/10 text-blue-500 border-blue-500/20">
          <Loader2 className="h-3 w-3 mr-1 animate-spin" />
          Running
        </Badge>
      );
    case "failed":
      return (
        <Badge className="bg-red-500/10 text-red-500 border-red-500/20">
          <XCircle className="h-3 w-3 mr-1" />
          Failed
        </Badge>
      );
    default:
      return (
        <Badge variant="outline">
          <Clock className="h-3 w-3 mr-1" />
          Pending
        </Badge>
      );
  }
}

export function BackupsPage() {
  const queryClient = useQueryClient();
  const [deleteTarget, setDeleteTarget] = useState<Backup | null>(null);
  const [restoreTarget, setRestoreTarget] = useState<Backup | null>(null);

  const { data, isLoading } = useQuery({
    queryKey: ["backups"],
    queryFn: () => listBackups(),
    refetchInterval: 10000,
  });

  const createFullMutation = useMutation({
    mutationFn: () => createBackup({ type: "full" }),
    onSuccess: () => {
      toast.success("Full backup started");
      queryClient.invalidateQueries({ queryKey: ["backups"] });
    },
    onError: (err: AxiosError<APIError>) => {
      toast.error(err.response?.data?.error?.message ?? "Failed to create backup");
    },
  });

  const deleteMutation = useMutation({
    mutationFn: () => deleteBackup(deleteTarget!.id),
    onSuccess: () => {
      toast.success("Backup deleted");
      setDeleteTarget(null);
      queryClient.invalidateQueries({ queryKey: ["backups"] });
    },
    onError: (err: AxiosError<APIError>) => {
      toast.error(err.response?.data?.error?.message ?? "Failed to delete backup");
    },
  });

  const restoreMutation = useMutation({
    mutationFn: () => restoreBackup(restoreTarget!.id),
    onSuccess: () => {
      toast.success("Backup restored successfully");
      setRestoreTarget(null);
    },
    onError: (err: AxiosError<APIError>) => {
      toast.error(err.response?.data?.error?.message ?? "Failed to restore backup");
    },
  });

  if (isLoading) {
    return (
      <div className="space-y-4">
        <Skeleton className="h-10 w-full" />
        <Skeleton className="h-64 w-full" />
      </div>
    );
  }

  const backups = data?.data ?? [];

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold">Backups</h1>
          <p className="text-muted-foreground">
            Create and manage server backups
          </p>
        </div>
        <Button
          onClick={() => createFullMutation.mutate()}
          disabled={createFullMutation.isPending}
          className="bg-pink-500 hover:bg-pink-600"
        >
          <Plus className="h-4 w-4 mr-1" />
          {createFullMutation.isPending ? "Starting..." : "Full Backup"}
        </Button>
      </div>

      {backups.length === 0 ? (
        <Card>
          <CardContent className="py-12 text-center">
            <Archive className="h-12 w-12 mx-auto text-muted-foreground mb-4" />
            <h3 className="text-lg font-medium">No backups yet</h3>
            <p className="text-muted-foreground text-sm mt-1">
              Create your first backup to protect your data
            </p>
          </CardContent>
        </Card>
      ) : (
        <div className="space-y-3">
          {backups.map((backup) => (
            <Card key={backup.id}>
              <CardHeader className="py-3">
                <div className="flex items-center justify-between">
                  <div className="space-y-1">
                    <CardTitle className="text-sm flex items-center gap-2">
                      <Archive className="h-4 w-4 text-pink-500" />
                      {backup.file_path.split("/").pop()}
                    </CardTitle>
                    <CardDescription className="flex items-center gap-4 text-xs">
                      <span className="flex items-center gap-1">
                        <Clock className="h-3 w-3" />
                        {formatDate(backup.created_at)}
                      </span>
                      {backup.size_bytes > 0 && (
                        <span className="flex items-center gap-1">
                          <HardDrive className="h-3 w-3" />
                          {formatSize(backup.size_bytes)}
                        </span>
                      )}
                      <Badge variant="outline" className="text-xs">
                        {backup.type}
                      </Badge>
                    </CardDescription>
                  </div>
                  <div className="flex items-center gap-2">
                    <StatusBadge status={backup.status} />
                    {backup.status === "completed" && (
                      <>
                        <Button
                          size="sm"
                          variant="outline"
                          className="h-7 text-xs"
                          onClick={() => {
                            toast.promise(downloadBackup(backup.id), {
                              loading: "Preparing download...",
                              success: "Download started",
                              error: "Failed to download backup",
                            });
                          }}
                        >
                          <Download className="h-3 w-3 mr-1" />
                          Download
                        </Button>
                        <Button
                          size="sm"
                          variant="outline"
                          className="h-7 text-xs"
                          onClick={() => setRestoreTarget(backup)}
                        >
                          <RotateCcw className="h-3 w-3 mr-1" />
                          Restore
                        </Button>
                      </>
                    )}
                    <Button
                      size="icon"
                      variant="ghost"
                      className="h-7 w-7 text-destructive"
                      onClick={() => setDeleteTarget(backup)}
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

      {/* Scheduled Backups */}
      <BackupScheduleCard />

      {/* Delete Confirmation */}
      <ConfirmDialog
        open={!!deleteTarget}
        onOpenChange={() => setDeleteTarget(null)}
        title="Delete Backup"
        description="Permanently delete this backup? This action cannot be undone."
        confirmText="Delete"
        destructive
        loading={deleteMutation.isPending}
        onConfirm={() => deleteMutation.mutate()}
      />

      {/* Restore Confirmation */}
      <ConfirmDialog
        open={!!restoreTarget}
        onOpenChange={() => setRestoreTarget(null)}
        title="Restore Backup"
        description="This will restore files from the backup. Existing files may be overwritten. Are you sure?"
        confirmText="Restore"
        destructive
        loading={restoreMutation.isPending}
        onConfirm={() => restoreMutation.mutate()}
      />
    </div>
  );
}

function BackupScheduleCard() {
  const queryClient = useQueryClient();
  const [showCreate, setShowCreate] = useState(false);
  const [frequency, setFrequency] = useState("daily");
  const [time, setTime] = useState("03:00");
  const [retention, setRetention] = useState(5);

  const { data: schedulesData } = useQuery({
    queryKey: ["backup-schedules"],
    queryFn: listSchedules,
  });

  const createMut = useMutation({
    mutationFn: () =>
      createSchedule({
        domain_id: null,
        frequency,
        time,
        retention_count: retention,
      }),
    onSuccess: () => {
      toast.success("Backup schedule created");
      queryClient.invalidateQueries({ queryKey: ["backup-schedules"] });
      setShowCreate(false);
    },
    onError: (err: AxiosError<APIError>) => {
      toast.error(err.response?.data?.error?.message ?? "Failed to create schedule");
    },
  });

  const toggleMut = useMutation({
    mutationFn: ({ id, schedule }: { id: number; schedule: BackupSchedule }) =>
      updateSchedule(id, {
        frequency: schedule.frequency,
        time: schedule.time,
        retention_count: schedule.retention_count,
        enabled: !schedule.enabled,
      }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["backup-schedules"] });
    },
  });

  const deleteMut = useMutation({
    mutationFn: (id: number) => deleteSchedule(id),
    onSuccess: () => {
      toast.success("Schedule deleted");
      queryClient.invalidateQueries({ queryKey: ["backup-schedules"] });
    },
  });

  // Show only global (non-domain) schedules
  const schedules = (schedulesData?.data ?? []).filter((s) => s.domain_id === null);

  return (
    <Card>
      <CardHeader className="pb-3">
        <div className="flex items-center justify-between">
          <div>
            <CardTitle className="text-sm flex items-center gap-2">
              <CalendarClock className="h-4 w-4 text-pink-500" />
              Scheduled Backups
            </CardTitle>
            <CardDescription className="text-xs mt-1">
              Automatically create full server backups on a schedule
            </CardDescription>
          </div>
          {!showCreate && (
            <Button size="sm" variant="outline" onClick={() => setShowCreate(true)}>
              <Plus className="h-3 w-3 mr-1" />
              Add Schedule
            </Button>
          )}
        </div>
      </CardHeader>
      <CardContent className="space-y-3">
        {showCreate && (
          <div className="p-3 border rounded space-y-3">
            <div className="grid grid-cols-3 gap-3">
              <div className="space-y-1">
                <Label className="text-xs">Frequency</Label>
                <Select value={frequency} onValueChange={(v) => v && setFrequency(v)}>
                  <SelectTrigger className="h-8 text-xs">
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="daily">Daily</SelectItem>
                    <SelectItem value="weekly">Weekly</SelectItem>
                    <SelectItem value="monthly">Monthly</SelectItem>
                  </SelectContent>
                </Select>
              </div>
              <div className="space-y-1">
                <Label className="text-xs">Time (UTC)</Label>
                <Input
                  type="time"
                  value={time}
                  onChange={(e) => setTime(e.target.value)}
                  className="h-8 text-xs"
                />
              </div>
              <div className="space-y-1">
                <Label className="text-xs">Keep Last</Label>
                <Input
                  type="number"
                  min={1}
                  max={50}
                  value={retention}
                  onChange={(e) => setRetention(Number(e.target.value))}
                  className="h-8 text-xs"
                />
              </div>
            </div>
            <div className="flex gap-2">
              <Button
                size="sm"
                onClick={() => createMut.mutate()}
                disabled={createMut.isPending}
                className="bg-pink-500 hover:bg-pink-600 h-7 text-xs"
              >
                {createMut.isPending ? "Creating..." : "Create"}
              </Button>
              <Button size="sm" variant="outline" onClick={() => setShowCreate(false)} className="h-7 text-xs">
                Cancel
              </Button>
            </div>
          </div>
        )}

        {schedules.length === 0 && !showCreate && (
          <p className="text-xs text-muted-foreground">No schedules configured</p>
        )}

        {schedules.map((sc) => (
          <div key={sc.id} className="flex items-center justify-between p-2 border rounded text-sm">
            <div className="space-y-0.5">
              <p className="text-xs font-medium capitalize">
                {sc.frequency} at {sc.time} UTC
              </p>
              <p className="text-xs text-muted-foreground">
                Keep {sc.retention_count} backups
                {sc.next_run && ` · Next: ${new Date(sc.next_run).toLocaleString()}`}
              </p>
            </div>
            <div className="flex items-center gap-2">
              <Switch
                checked={sc.enabled}
                onCheckedChange={() => toggleMut.mutate({ id: sc.id, schedule: sc })}
              />
              <Button
                size="icon"
                variant="ghost"
                className="h-7 w-7 text-destructive"
                onClick={() => deleteMut.mutate(sc.id)}
              >
                <Trash2 className="h-3 w-3" />
              </Button>
            </div>
          </div>
        ))}
      </CardContent>
    </Card>
  );
}
