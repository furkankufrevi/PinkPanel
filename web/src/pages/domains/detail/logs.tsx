import { useState } from "react";
import { useParams } from "react-router-dom";
import { useQuery } from "@tanstack/react-query";
import { Card, CardContent } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Skeleton } from "@/components/ui/skeleton";
import { getDomainLogs, downloadDomainLog } from "@/api/logs";
import { toast } from "sonner";
import { ScrollText, RefreshCw, Search, Download } from "lucide-react";

const LOG_TYPES = [
  { key: "access", label: "Access" },
  { key: "error", label: "Error" },
  { key: "php", label: "PHP" },
];

export function DomainLogs() {
  const { id } = useParams<{ id: string }>();
  const domainId = Number(id);

  const [logType, setLogType] = useState("access");
  const [lines, setLines] = useState(100);
  const [filter, setFilter] = useState("");
  const [appliedFilter, setAppliedFilter] = useState("");

  const { data, isLoading, refetch, isFetching } = useQuery({
    queryKey: ["domain-logs", domainId, logType, lines, appliedFilter],
    queryFn: () => getDomainLogs(domainId, logType, lines, appliedFilter),
    enabled: !!domainId,
  });

  return (
    <div className="space-y-4 max-w-4xl">
      <div className="flex items-center justify-between">
        <h3 className="text-lg font-medium">Logs</h3>
        <div className="flex gap-2">
          <Button
            size="sm"
            variant="outline"
            onClick={() => {
              toast.promise(downloadDomainLog(domainId, logType), {
                loading: "Preparing download...",
                success: "Download started",
                error: "Failed to download log",
              });
            }}
          >
            <Download className="h-4 w-4 mr-1" />
            Download
          </Button>
          <Button
            size="sm"
            variant="outline"
            onClick={() => refetch()}
            disabled={isFetching}
          >
            <RefreshCw
              className={`h-4 w-4 mr-1 ${isFetching ? "animate-spin" : ""}`}
            />
            Refresh
          </Button>
        </div>
      </div>

      {/* Controls */}
      <div className="flex items-end gap-3 flex-wrap">
        <div className="space-y-1">
          <Label className="text-xs">Log Type</Label>
          <div className="flex gap-1">
            {LOG_TYPES.map((t) => (
              <Button
                key={t.key}
                size="sm"
                variant={logType === t.key ? "default" : "outline"}
                className={
                  logType === t.key ? "bg-pink-500 hover:bg-pink-600" : ""
                }
                onClick={() => setLogType(t.key)}
              >
                {t.label}
              </Button>
            ))}
          </div>
        </div>
        <div className="space-y-1">
          <Label className="text-xs">Lines</Label>
          <Input
            type="number"
            value={lines}
            onChange={(e) => setLines(Number(e.target.value) || 100)}
            className="w-24 h-9"
          />
        </div>
        <div className="space-y-1 flex-1 min-w-[200px]">
          <Label className="text-xs">Filter</Label>
          <div className="flex gap-1">
            <Input
              value={filter}
              onChange={(e) => setFilter(e.target.value)}
              placeholder="Search logs..."
              className="h-9"
              onKeyDown={(e) => {
                if (e.key === "Enter") setAppliedFilter(filter);
              }}
            />
            <Button
              size="sm"
              variant="outline"
              className="h-9"
              onClick={() => setAppliedFilter(filter)}
            >
              <Search className="h-4 w-4" />
            </Button>
          </div>
        </div>
      </div>

      {/* Log Output */}
      {isLoading ? (
        <Skeleton className="h-96 w-full" />
      ) : data?.content ? (
        <Card>
          <CardContent className="p-0">
            <pre className="p-4 text-xs font-mono overflow-auto max-h-[600px] whitespace-pre-wrap break-all">
              {typeof data.content === "string"
                ? data.content
                : JSON.stringify(data.content, null, 2)}
            </pre>
          </CardContent>
        </Card>
      ) : (
        <Card>
          <CardContent className="py-8 text-center">
            <ScrollText className="h-10 w-10 mx-auto text-muted-foreground mb-3" />
            <p className="text-sm text-muted-foreground">
              No log data available
            </p>
          </CardContent>
        </Card>
      )}
    </div>
  );
}
