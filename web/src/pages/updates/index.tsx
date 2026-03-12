import { useQuery, useMutation } from "@tanstack/react-query";
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
  triggerUpgrade,
} from "@/api/updates";
import type { Release } from "@/api/updates";
import type { AxiosError } from "axios";
import type { APIError } from "@/types/api";
import {
  Download,
  CheckCircle,
  AlertCircle,
  ArrowUpCircle,
  Clock,
  ExternalLink,
  Tag,
  History,
  Loader2,
} from "lucide-react";

export function UpdatesPage() {
  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold">Updates</h1>
        <p className="text-muted-foreground">
          Check for new versions and manage upgrades
        </p>
      </div>
      <UpdateCheckCard />
      <ReleasesCard />
      <UpgradeHistoryCard />
    </div>
  );
}

function UpdateCheckCard() {
  const { data, isLoading, refetch, isRefetching } = useQuery({
    queryKey: ["update-check"],
    queryFn: checkForUpdates,
    retry: false,
  });

  const upgradeMut = useMutation({
    mutationFn: triggerUpgrade,
    onSuccess: () => {
      toast.success(
        "Upgrade started. The panel will restart automatically when complete."
      );
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
      <CardHeader className="pb-3">
        <CardTitle className="flex items-center gap-2">
          {hasUpdate ? (
            <ArrowUpCircle className="h-5 w-5 text-pink-500" />
          ) : (
            <CheckCircle className="h-5 w-5 text-green-500" />
          )}
          {hasUpdate ? "Update Available" : "Up to Date"}
        </CardTitle>
        <CardDescription>
          Current version: <strong>v{data?.current_version}</strong>
        </CardDescription>
      </CardHeader>
      <CardContent className="space-y-4">
        {hasUpdate && data?.latest_version && (
          <div className="rounded-lg border border-pink-500/20 bg-pink-500/5 p-4 space-y-3">
            <div className="flex items-center justify-between">
              <div>
                <h3 className="font-semibold">
                  {data.release_name || `v${data.latest_version}`}
                </h3>
                {data.published_at && (
                  <p className="text-xs text-muted-foreground flex items-center gap-1 mt-1">
                    <Clock className="h-3 w-3" />
                    {new Date(data.published_at).toLocaleDateString()}
                  </p>
                )}
              </div>
              <Badge className="bg-pink-500/10 text-pink-500 border-pink-500/20">
                v{data.latest_version}
              </Badge>
            </div>
            {data.release_notes && (
              <div className="text-sm text-muted-foreground whitespace-pre-wrap max-h-48 overflow-y-auto border-t pt-3">
                {data.release_notes}
              </div>
            )}
            <div className="flex items-center gap-2 pt-1">
              <Button
                onClick={() => upgradeMut.mutate()}
                disabled={upgradeMut.isPending}
              >
                {upgradeMut.isPending ? (
                  <Loader2 className="h-4 w-4 mr-2 animate-spin" />
                ) : (
                  <Download className="h-4 w-4 mr-2" />
                )}
                Upgrade Now
              </Button>
              {data.release_url && (
                <Button variant="outline" asChild>
                  <a
                    href={data.release_url}
                    target="_blank"
                    rel="noopener noreferrer"
                  >
                    <ExternalLink className="h-4 w-4 mr-2" />
                    View on GitHub
                  </a>
                </Button>
              )}
            </div>
          </div>
        )}

        {!hasUpdate && (
          <p className="text-sm text-muted-foreground">
            You are running the latest version.
          </p>
        )}

        {data?.error && (
          <div className="flex items-center gap-2 text-sm text-yellow-500">
            <AlertCircle className="h-4 w-4" />
            {data.error}
          </div>
        )}

        <Button
          variant="outline"
          size="sm"
          onClick={() => refetch()}
          disabled={isRefetching}
        >
          {isRefetching ? (
            <Loader2 className="h-3 w-3 mr-2 animate-spin" />
          ) : (
            <ArrowUpCircle className="h-3 w-3 mr-2" />
          )}
          Check Again
        </Button>
      </CardContent>
    </Card>
  );
}

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
        <div className="space-y-4">
          {data.releases.map((release) => (
            <ReleaseItem
              key={release.version}
              release={release}
            />
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
            <Badge variant="outline" className="text-green-500 border-green-500/30">
              Current
            </Badge>
          )}
          {release.is_newer && (
            <Badge className="bg-pink-500/10 text-pink-500 border-pink-500/20">
              New
            </Badge>
          )}
          {release.prerelease && (
            <Badge variant="outline" className="text-yellow-500 border-yellow-500/30">
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
        <div className="space-y-2">
          {data.history.map((entry) => (
            <div
              key={entry.id}
              className="flex items-center justify-between p-3 rounded border text-sm"
            >
              <div className="flex items-center gap-3">
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
                  {entry.status}
                </Badge>
                <span>
                  {entry.previous_version && (
                    <span className="text-muted-foreground">
                      v{entry.previous_version} →{" "}
                    </span>
                  )}
                  <strong>v{entry.version}</strong>
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
