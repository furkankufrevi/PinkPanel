import { useState, useEffect, useRef, useCallback } from "react";
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
import { toast } from "sonner";
import {
  checkForUpdates,
  getReleases,
  getUpgradeHistory,
  getUpgradeStatus,
  triggerUpgrade,
} from "@/api/updates";
import type { Release } from "@/api/updates";
import type { AxiosError } from "axios";
import type { APIError } from "@/types/api";
import {
  Download,
  CheckCircle,
  CheckCircle2,
  AlertCircle,
  ArrowUpCircle,
  Clock,
  ExternalLink,
  Tag,
  History,
  Loader2,
  Terminal,
  XCircle,
  RefreshCw,
  Rocket,
  Server,
} from "lucide-react";

export function UpdatesPage() {
  const [upgrading, setUpgrading] = useState(false);

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold">Updates</h1>
        <p className="text-muted-foreground">
          Check for new versions and manage upgrades
        </p>
      </div>

      {upgrading ? (
        <UpgradeProgressCard onDone={() => setUpgrading(false)} />
      ) : (
        <>
          <UpdateCheckCard onUpgradeStart={() => setUpgrading(true)} />
          <ReleasesCard />
          <UpgradeHistoryCard />
        </>
      )}
    </div>
  );
}

// ─── Update Check ────────────────────────────────

function UpdateCheckCard({
  onUpgradeStart,
}: {
  onUpgradeStart: () => void;
}) {
  const { data, isLoading, refetch, isRefetching } = useQuery({
    queryKey: ["update-check"],
    queryFn: checkForUpdates,
    retry: false,
  });

  const upgradeMut = useMutation({
    mutationFn: triggerUpgrade,
    onSuccess: () => {
      onUpgradeStart();
    },
    onError: (err: AxiosError<APIError>) => {
      toast.error(
        err.response?.data?.error?.message ?? "Failed to start upgrade"
      );
    },
  });

  if (isLoading) return <Skeleton className="h-40 w-full" />;

  const hasUpdate = data?.update_available;

  return (
    <Card>
      <CardHeader>
        <div className="flex items-center justify-between">
          <div>
            <CardTitle className="flex items-center gap-2">
              {hasUpdate ? (
                <ArrowUpCircle className="h-5 w-5 text-pink-500" />
              ) : (
                <CheckCircle className="h-5 w-5 text-green-500" />
              )}
              {hasUpdate ? "Update Available" : "Up to Date"}
            </CardTitle>
            <CardDescription>
              Running version{" "}
              <span className="font-mono font-medium text-foreground">
                v{data?.current_version}
              </span>
            </CardDescription>
          </div>
          <Button
            variant="outline"
            size="sm"
            onClick={() => refetch()}
            disabled={isRefetching}
          >
            {isRefetching ? (
              <Loader2 className="h-3.5 w-3.5 mr-1.5 animate-spin" />
            ) : (
              <RefreshCw className="h-3.5 w-3.5 mr-1.5" />
            )}
            Check Now
          </Button>
        </div>
      </CardHeader>
      <CardContent className="space-y-4">
        {hasUpdate && data?.latest_version && (
          <div className="rounded-lg border border-pink-500/20 bg-pink-500/5 p-4 space-y-4">
            <div className="flex items-center justify-between">
              <div className="space-y-1">
                <h3 className="font-semibold text-lg">
                  {data.release_name || `v${data.latest_version}`}
                </h3>
                <div className="flex items-center gap-3 text-sm text-muted-foreground">
                  <Badge className="bg-pink-500/10 text-pink-500 border-pink-500/20 font-mono">
                    v{data.latest_version}
                  </Badge>
                  {data.published_at && (
                    <span className="flex items-center gap-1">
                      <Clock className="h-3 w-3" />
                      {new Date(data.published_at).toLocaleDateString()}
                    </span>
                  )}
                </div>
              </div>
            </div>
            {data.release_notes && (
              <div className="text-sm text-muted-foreground whitespace-pre-wrap max-h-48 overflow-y-auto border-t border-pink-500/10 pt-3 leading-relaxed">
                {data.release_notes}
              </div>
            )}
            <div className="flex items-center gap-2 pt-1">
              <Button
                onClick={() => upgradeMut.mutate()}
                disabled={upgradeMut.isPending}
                className="bg-pink-500 hover:bg-pink-600"
              >
                {upgradeMut.isPending ? (
                  <Loader2 className="h-4 w-4 mr-2 animate-spin" />
                ) : (
                  <Rocket className="h-4 w-4 mr-2" />
                )}
                Upgrade Now
              </Button>
              {data.release_url && (
                <a
                  href={data.release_url}
                  target="_blank"
                  rel="noopener noreferrer"
                >
                  <Button variant="outline" size="sm">
                    <ExternalLink className="h-3.5 w-3.5 mr-1.5" />
                    View on GitHub
                  </Button>
                </a>
              )}
            </div>
          </div>
        )}

        {!hasUpdate && !data?.error && (
          <div className="flex items-center gap-3 rounded-lg border p-4">
            <div className="flex h-10 w-10 items-center justify-center rounded-full bg-green-500/10">
              <CheckCircle2 className="h-5 w-5 text-green-500" />
            </div>
            <div>
              <p className="text-sm font-medium">You're on the latest version</p>
              <p className="text-xs text-muted-foreground">
                No updates available at this time.
              </p>
            </div>
          </div>
        )}

        {data?.error && (
          <div className="rounded-lg border border-yellow-500/20 bg-yellow-500/5 p-4 space-y-3">
            <div className="flex items-center gap-2 text-sm text-yellow-600 dark:text-yellow-400">
              <AlertCircle className="h-4 w-4 shrink-0" />
              <span>{data.error}</span>
            </div>
            <Button
              size="sm"
              onClick={() => upgradeMut.mutate()}
              disabled={upgradeMut.isPending}
            >
              {upgradeMut.isPending ? (
                <Loader2 className="h-4 w-4 mr-2 animate-spin" />
              ) : (
                <Download className="h-4 w-4 mr-2" />
              )}
              Upgrade to Latest
            </Button>
          </div>
        )}
      </CardContent>
    </Card>
  );
}

// ─── Upgrade Progress ────────────────────────────

function UpgradeProgressCard({ onDone }: { onDone: () => void }) {
  const queryClient = useQueryClient();
  const [logContent, setLogContent] = useState("");
  const [offset, setOffset] = useState(0);
  const [status, setStatus] = useState<string>("in_progress");
  const [running, setRunning] = useState(true);
  const logRef = useRef<HTMLPreElement>(null);
  const autoScroll = useRef(true);

  const pollLog = useCallback(async () => {
    try {
      const data = await getUpgradeStatus(offset);
      if (data.log) {
        setLogContent((prev) => prev + data.log);
        setOffset(data.total_size);
      }
      setStatus(data.status);
      setRunning(data.running);
      return data;
    } catch {
      // Server may be restarting during upgrade
      return null;
    }
  }, [offset]);

  useEffect(() => {
    // Initial poll
    pollLog();

    const interval = setInterval(async () => {
      const data = await pollLog();
      if (data && !data.running && data.status !== "in_progress") {
        clearInterval(interval);
        // Invalidate queries so history and version refresh
        queryClient.invalidateQueries({ queryKey: ["update-check"] });
        queryClient.invalidateQueries({ queryKey: ["upgrade-history"] });
      }
    }, 2000);

    return () => clearInterval(interval);
  }, [pollLog, queryClient]);

  // Auto-scroll to bottom
  useEffect(() => {
    if (autoScroll.current && logRef.current) {
      logRef.current.scrollTop = logRef.current.scrollHeight;
    }
  }, [logContent]);

  const handleScroll = () => {
    if (!logRef.current) return;
    const { scrollTop, scrollHeight, clientHeight } = logRef.current;
    autoScroll.current = scrollHeight - scrollTop - clientHeight < 50;
  };

  const isFinished = !running && status !== "in_progress";
  const isSuccess = status === "completed";
  const isFailed = status === "failed";

  // Parse log lines for display
  const logLines = logContent.split("\n");

  return (
    <Card className="overflow-hidden">
      <CardHeader>
        <div className="flex items-center justify-between">
          <div>
            <CardTitle className="flex items-center gap-2">
              {isFinished ? (
                isSuccess ? (
                  <CheckCircle2 className="h-5 w-5 text-green-500" />
                ) : (
                  <XCircle className="h-5 w-5 text-red-500" />
                )
              ) : (
                <Loader2 className="h-5 w-5 text-pink-500 animate-spin" />
              )}
              {isFinished
                ? isSuccess
                  ? "Upgrade Complete"
                  : "Upgrade Failed"
                : "Upgrading..."}
            </CardTitle>
            <CardDescription>
              {isFinished
                ? isSuccess
                  ? "PinkPanel has been updated successfully."
                  : "The upgrade encountered an error. Check the log below."
                : "Please wait while PinkPanel is being updated. Do not close this page."}
            </CardDescription>
          </div>
          <div className="flex items-center gap-2">
            {!isFinished && (
              <Badge className="bg-blue-500/10 text-blue-500 border-blue-500/20 gap-1.5 animate-pulse">
                <Server className="h-3 w-3" />
                In Progress
              </Badge>
            )}
            {isSuccess && (
              <Badge className="bg-green-500/10 text-green-500 border-green-500/20 gap-1.5">
                <CheckCircle2 className="h-3 w-3" />
                Success
              </Badge>
            )}
            {isFailed && (
              <Badge className="bg-red-500/10 text-red-500 border-red-500/20 gap-1.5">
                <XCircle className="h-3 w-3" />
                Failed
              </Badge>
            )}
          </div>
        </div>
      </CardHeader>
      <CardContent className="space-y-4">
        {/* Progress Steps */}
        <ProgressSteps logContent={logContent} isFinished={isFinished} isSuccess={isSuccess} />

        {/* Log Output */}
        <div className="space-y-2">
          <div className="flex items-center justify-between">
            <h4 className="text-sm font-medium flex items-center gap-2">
              <Terminal className="h-4 w-4" />
              Upgrade Log
            </h4>
            <span className="text-xs text-muted-foreground font-mono">
              {logLines.length} lines
            </span>
          </div>
          <pre
            ref={logRef}
            onScroll={handleScroll}
            className="bg-[oklch(0.13_0_0)] text-[oklch(0.8_0_0)] rounded-lg p-4 text-xs font-mono leading-relaxed max-h-[400px] overflow-y-auto whitespace-pre-wrap break-words border"
          >
            {logContent || "Waiting for output..."}
            {!isFinished && (
              <span className="inline-block w-2 h-4 bg-pink-500 animate-pulse ml-0.5" />
            )}
          </pre>
        </div>

        {/* Actions */}
        {isFinished && (
          <div className="flex items-center gap-2 pt-2">
            <Button
              onClick={() => {
                onDone();
                if (isSuccess) {
                  window.location.reload();
                }
              }}
              className={isSuccess ? "bg-green-600 hover:bg-green-700" : "bg-pink-500 hover:bg-pink-600"}
            >
              {isSuccess ? (
                <>
                  <CheckCircle2 className="h-4 w-4 mr-2" />
                  Done — Reload Panel
                </>
              ) : (
                <>
                  <RefreshCw className="h-4 w-4 mr-2" />
                  Back to Updates
                </>
              )}
            </Button>
          </div>
        )}
      </CardContent>
    </Card>
  );
}

// ─── Progress Steps ──────────────────────────────

function ProgressSteps({
  logContent,
  isFinished,
  isSuccess,
}: {
  logContent: string;
  isFinished: boolean;
  isSuccess: boolean;
}) {
  const steps = [
    { label: "Cloning repository", pattern: /Cloning latest/i, donePattern: /Build complete|Building/i },
    { label: "Building from source", pattern: /Building/i, donePattern: /Build complete/i },
    { label: "Installing binaries", pattern: /Installing new binaries|Stopping PinkPanel/i, donePattern: /Configuring|setup_/i },
    { label: "Configuring services", pattern: /Configuring|setup_/i, donePattern: /Starting PinkPanel/i },
    { label: "Starting services", pattern: /Starting PinkPanel/i, donePattern: /Upgraded successfully|is running/i },
  ];

  return (
    <div className="grid grid-cols-5 gap-2">
      {steps.map((step, i) => {
        const started = step.pattern.test(logContent);
        const done = step.donePattern.test(logContent);
        const isCurrent = started && !done && !isFinished;
        const failed = isFinished && !isSuccess && started && !done;

        return (
          <div key={i} className="flex flex-col items-center gap-1.5">
            <div
              className={`flex h-8 w-8 items-center justify-center rounded-full border-2 transition-colors ${
                done
                  ? "border-green-500 bg-green-500/10"
                  : failed
                    ? "border-red-500 bg-red-500/10"
                    : isCurrent
                      ? "border-pink-500 bg-pink-500/10"
                      : "border-muted bg-muted/30"
              }`}
            >
              {done ? (
                <CheckCircle2 className="h-4 w-4 text-green-500" />
              ) : failed ? (
                <XCircle className="h-4 w-4 text-red-500" />
              ) : isCurrent ? (
                <Loader2 className="h-4 w-4 text-pink-500 animate-spin" />
              ) : (
                <span className="text-xs text-muted-foreground font-medium">
                  {i + 1}
                </span>
              )}
            </div>
            <span
              className={`text-[10px] text-center leading-tight ${
                done
                  ? "text-green-500 font-medium"
                  : isCurrent
                    ? "text-pink-500 font-medium"
                    : failed
                      ? "text-red-500"
                      : "text-muted-foreground"
              }`}
            >
              {step.label}
            </span>
          </div>
        );
      })}
    </div>
  );
}

// ─── Releases ────────────────────────────────────

function ReleasesCard() {
  const { data, isLoading } = useQuery({
    queryKey: ["releases"],
    queryFn: getReleases,
    retry: false,
  });

  if (isLoading) return <Skeleton className="h-64 w-full" />;
  if (!data?.releases?.length) return null;

  return (
    <Card>
      <CardHeader>
        <CardTitle className="flex items-center gap-2">
          <Tag className="h-5 w-5 text-pink-500" />
          Release History
        </CardTitle>
        <CardDescription>Recent versions and changelogs</CardDescription>
      </CardHeader>
      <CardContent>
        <div className="space-y-3">
          {data.releases.map((release) => (
            <ReleaseItem key={release.version} release={release} />
          ))}
        </div>
      </CardContent>
    </Card>
  );
}

function ReleaseItem({ release }: { release: Release }) {
  return (
    <div className="border rounded-lg p-4 space-y-2">
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-2">
          <h3 className="font-medium">
            {release.name || `v${release.version}`}
          </h3>
          {release.is_current && (
            <Badge
              variant="outline"
              className="text-green-500 border-green-500/30"
            >
              Current
            </Badge>
          )}
          {release.is_newer && (
            <Badge className="bg-pink-500/10 text-pink-500 border-pink-500/20">
              New
            </Badge>
          )}
          {release.prerelease && (
            <Badge
              variant="outline"
              className="text-yellow-500 border-yellow-500/30"
            >
              Pre-release
            </Badge>
          )}
        </div>
        <div className="flex items-center gap-2 text-xs text-muted-foreground">
          <Clock className="h-3 w-3" />
          {new Date(release.published_at).toLocaleDateString()}
        </div>
      </div>
      {release.notes && (
        <p className="text-sm text-muted-foreground whitespace-pre-wrap line-clamp-4">
          {release.notes}
        </p>
      )}
      {release.url && (
        <a
          href={release.url}
          target="_blank"
          rel="noopener noreferrer"
          className="text-xs text-pink-500 hover:underline inline-flex items-center gap-1"
        >
          <ExternalLink className="h-3 w-3" />
          View release
        </a>
      )}
    </div>
  );
}

// ─── Upgrade History ─────────────────────────────

function UpgradeHistoryCard() {
  const { data, isLoading } = useQuery({
    queryKey: ["upgrade-history"],
    queryFn: getUpgradeHistory,
    retry: false,
  });

  if (isLoading) return <Skeleton className="h-32 w-full" />;
  if (!data?.history?.length) return null;

  return (
    <Card>
      <CardHeader>
        <CardTitle className="flex items-center gap-2">
          <History className="h-5 w-5 text-pink-500" />
          Upgrade History
        </CardTitle>
        <CardDescription>Past upgrade attempts</CardDescription>
      </CardHeader>
      <CardContent>
        <div className="border rounded-lg divide-y">
          {data.history.map((entry) => (
            <div
              key={entry.id}
              className="flex items-center justify-between p-3 text-sm"
            >
              <div className="flex items-center gap-3">
                {entry.status === "completed" ? (
                  <CheckCircle2 className="h-4 w-4 text-green-500" />
                ) : entry.status === "in_progress" ? (
                  <Loader2 className="h-4 w-4 text-blue-500 animate-spin" />
                ) : (
                  <XCircle className="h-4 w-4 text-red-500" />
                )}
                <Badge
                  variant="outline"
                  className={
                    entry.status === "completed"
                      ? "text-green-500 border-green-500/30"
                      : entry.status === "in_progress"
                        ? "text-blue-500 border-blue-500/30"
                        : "text-red-500 border-red-500/30"
                  }
                >
                  {entry.status === "completed"
                    ? "Completed"
                    : entry.status === "in_progress"
                      ? "In Progress"
                      : "Failed"}
                </Badge>
                <span>
                  {entry.previous_version && (
                    <span className="text-muted-foreground font-mono text-xs">
                      v{entry.previous_version}
                    </span>
                  )}
                  {entry.previous_version && (
                    <span className="text-muted-foreground mx-1.5">→</span>
                  )}
                  <span className="font-mono text-xs font-medium">
                    v{entry.version}
                  </span>
                </span>
              </div>
              <span className="text-xs text-muted-foreground">
                {new Date(entry.created_at).toLocaleString()}
              </span>
            </div>
          ))}
        </div>
      </CardContent>
    </Card>
  );
}
