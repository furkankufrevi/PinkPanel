import { useNavigate } from "react-router-dom";
import { useQuery } from "@tanstack/react-query";
import {
  Card,
  CardContent,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { Skeleton } from "@/components/ui/skeleton";
import { StatusBadge } from "@/components/shared/status-badge";
import { listDomains } from "@/api/domains";
import { Globe, Network, FolderOpen } from "lucide-react";

export function FilesPage() {
  const navigate = useNavigate();

  const { data, isLoading } = useQuery({
    queryKey: ["domains", { per_page: 100 }],
    queryFn: () => listDomains({ per_page: 100 }),
  });

  const domains = data?.data ?? [];

  if (isLoading) {
    return (
      <div className="space-y-4">
        <Skeleton className="h-10 w-full" />
        <Skeleton className="h-64 w-full" />
      </div>
    );
  }

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold">File Manager</h1>
        <p className="text-muted-foreground">
          Select a domain to manage its files
        </p>
      </div>

      {domains.length === 0 ? (
        <Card>
          <CardContent className="py-12 text-center">
            <FolderOpen className="h-12 w-12 mx-auto text-muted-foreground mb-4" />
            <h3 className="text-lg font-medium">No domains yet</h3>
            <p className="text-muted-foreground text-sm mt-1">
              Create a domain first to manage its files
            </p>
          </CardContent>
        </Card>
      ) : (
        <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-3">
          {domains.map((domain) => (
            <Card
              key={domain.id}
              className="cursor-pointer hover:border-pink-500/50 transition-colors"
              onClick={() => navigate(`/domains/${domain.id}/files`)}
            >
              <CardHeader className="pb-3">
                <div className="flex items-center justify-between">
                  <CardTitle className="text-base flex items-center gap-2">
                    {domain.parent_id ? (
                      <Network className="h-4 w-4 text-pink-500" />
                    ) : (
                      <Globe className="h-4 w-4 text-pink-500" />
                    )}
                    {domain.name}
                  </CardTitle>
                  <StatusBadge status={domain.status} />
                </div>
                <p className="text-xs text-muted-foreground truncate">
                  {domain.document_root}
                </p>
              </CardHeader>
            </Card>
          ))}
        </div>
      )}
    </div>
  );
}
