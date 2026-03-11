import { useState, useEffect } from "react";
import { useParams } from "react-router-dom";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { Skeleton } from "@/components/ui/skeleton";
import { toast } from "sonner";
import { getDomainPHP, updateDomainPHP, getPHPVersions } from "@/api/php";
import type { AxiosError } from "axios";
import type { APIError } from "@/types/api";

const phpDirectives = [
  { key: "upload_max_filesize", label: "Upload Max Filesize", placeholder: "64M" },
  { key: "post_max_size", label: "Post Max Size", placeholder: "64M" },
  { key: "max_execution_time", label: "Max Execution Time", placeholder: "300" },
  { key: "memory_limit", label: "Memory Limit", placeholder: "256M" },
  { key: "display_errors", label: "Display Errors", placeholder: "Off" },
  { key: "error_reporting", label: "Error Reporting", placeholder: "E_ALL & ~E_DEPRECATED & ~E_STRICT" },
  { key: "date.timezone", label: "Timezone", placeholder: "UTC" },
];

export function DomainPHP() {
  const { id } = useParams<{ id: string }>();
  const domainId = Number(id);
  const queryClient = useQueryClient();

  const [version, setVersion] = useState("");
  const [settings, setSettings] = useState<Record<string, string>>({});

  const { data: phpData, isLoading } = useQuery({
    queryKey: ["domain-php", domainId],
    queryFn: () => getDomainPHP(domainId),
    enabled: !!domainId,
  });

  const { data: versionsData } = useQuery({
    queryKey: ["php-versions"],
    queryFn: getPHPVersions,
  });

  useEffect(() => {
    if (phpData) {
      setVersion(phpData.version);
      setSettings(phpData.settings || {});
    }
  }, [phpData]);

  const updateMutation = useMutation({
    mutationFn: () => updateDomainPHP(domainId, { version, settings }),
    onSuccess: () => {
      toast.success("PHP settings updated");
      queryClient.invalidateQueries({ queryKey: ["domain-php", domainId] });
      queryClient.invalidateQueries({ queryKey: ["domain", domainId] });
    },
    onError: (err: AxiosError<APIError>) => {
      toast.error(err.response?.data?.error?.message ?? "Failed to update PHP settings");
    },
  });

  function updateSetting(key: string, value: string) {
    setSettings((prev) => ({ ...prev, [key]: value }));
  }

  if (isLoading) {
    return (
      <div className="space-y-4 max-w-2xl">
        <Skeleton className="h-48 w-full" />
        <Skeleton className="h-64 w-full" />
      </div>
    );
  }

  const versions = versionsData?.data ?? ["8.3", "8.2", "8.1", "8.0", "7.4"];

  return (
    <div className="space-y-6 max-w-2xl">
      <Card>
        <CardHeader>
          <CardTitle>PHP Version</CardTitle>
          <CardDescription>
            Select the PHP version for this domain
          </CardDescription>
        </CardHeader>
        <CardContent>
          <Select value={version} onValueChange={(v) => v && setVersion(v)}>
            <SelectTrigger className="w-[200px]">
              <SelectValue placeholder="Select version" />
            </SelectTrigger>
            <SelectContent>
              {versions.map((v) => (
                <SelectItem key={v} value={v}>
                  PHP {v}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle>PHP Settings</CardTitle>
          <CardDescription>
            Configure php.ini directives for this domain. Changes are applied via FPM pool reload.
          </CardDescription>
        </CardHeader>
        <CardContent className="space-y-4">
          {phpDirectives.map((dir) => (
            <div key={dir.key} className="grid grid-cols-2 gap-4 items-center">
              <Label className="text-sm">{dir.label}</Label>
              <Input
                value={settings[dir.key] ?? ""}
                onChange={(e) => updateSetting(dir.key, e.target.value)}
                placeholder={dir.placeholder}
              />
            </div>
          ))}
        </CardContent>
      </Card>

      <Button
        onClick={() => updateMutation.mutate()}
        disabled={updateMutation.isPending}
        className="bg-pink-500 hover:bg-pink-600"
      >
        {updateMutation.isPending ? "Saving..." : "Save PHP Settings"}
      </Button>
    </div>
  );
}
