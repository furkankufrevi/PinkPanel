import { useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { Card, CardContent } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Skeleton } from "@/components/ui/skeleton";
import { getSystemLogs } from "@/api/logs";
import { ScrollText, RefreshCw, Search } from "lucide-react";

const SYSTEM_LOG_TYPES = [
  { key: "syslog", label: "Syslog" },
  { key: "auth", label: "Auth" },
  { key: "nginx", label: "NGINX" },
  { key: "mysql", label: "MySQL" },
];

export function LogsPage() {
  const [logType, setLogType] = useState("syslog");
  const [lines, setLines] = useState(200);
  const [filter, setFilter] = useState("");
  const [appliedFilter, setAppliedFilter] = useState("");

  const { data, isLoading, refetch, isFetching } = useQuery({
    queryKey: ["system-logs", logType, lines, appliedFilter],
    queryFn: () => getSystemLogs(logType, lines, appliedFilter),
  });

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold">System Logs</h1>
          <p className="text-muted-foreground">
            View server logs in real time
          </p>
        </div>
        <Button
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

      {/* Controls */}
      <div className="flex items-end gap-3 flex-wrap">
        <div className="space-y-1">
          <Label className="text-xs">Log Source</Label>
          <div className="flex gap-1">
            {SYSTEM_LOG_TYPES.map((t) => (
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
            onChange={(e) => setLines(Number(e.target.value) || 200)}
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
          <CardContent className="py-12 text-center">
            <ScrollText className="h-12 w-12 mx-auto text-muted-foreground mb-4" />
            <h3 className="text-lg font-medium">No log data</h3>
            <p className="text-muted-foreground text-sm mt-1">
              Unable to read logs — check agent connection
            </p>
          </CardContent>
        </Card>
      )}
    </div>
  );
}
