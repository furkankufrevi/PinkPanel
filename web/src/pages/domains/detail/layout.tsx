import { useMemo } from "react";
import { useParams, useNavigate, useLocation, Outlet } from "react-router-dom";
import { useQuery } from "@tanstack/react-query";
import { ArrowLeft, ChevronRight } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { StatusBadge } from "@/components/shared/status-badge";
import { Skeleton } from "@/components/ui/skeleton";
import { cn } from "@/lib/utils";
import { getDomain } from "@/api/domains";

// Plesk-style: only 4 top-level tabs instead of 14
const topTabs = [
  { value: "overview", label: "Dashboard" },
  { value: "dns", label: "Hosting & DNS" },
  { value: "email", label: "Mail" },
  { value: "settings", label: "Settings" },
];

// Labels for sub-pages (shown in breadcrumb)
const subPageLabels: Record<string, string> = {
  files: "Files",
  databases: "Databases",
  ftp: "FTP",
  backups: "Backups",
  php: "PHP",
  ssl: "SSL/TLS",
  redirects: "Redirects",
  crons: "Cron Jobs",
  git: "Git",
  logs: "Logs",
  apps: "Applications",
};

export function DomainDetailLayout() {
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const location = useLocation();

  const domainId = Number(id);
  const { data: domain, isLoading } = useQuery({
    queryKey: ["domain", domainId],
    queryFn: () => getDomain(domainId),
    enabled: !!domainId,
  });

  // Fetch parent domain name if this is a subdomain
  const parentId = domain?.parent_id;
  const { data: parentDomain } = useQuery({
    queryKey: ["domain", parentId],
    queryFn: () => getDomain(parentId!),
    enabled: !!parentId,
  });

  const isSubdomain = !!domain?.parent_id;
  const hasSeparateDNS = domain?.separate_dns ?? false;

  const tabs = useMemo(() => {
    return topTabs.filter((tab) => {
      // Hide Hosting & DNS tab for subdomains without separate DNS
      if (tab.value === "dns" && isSubdomain && !hasSeparateDNS) return false;
      return true;
    });
  }, [isSubdomain, hasSeparateDNS]);

  const pathParts = location.pathname.split("/");
  const activeRoute = pathParts[3] || "overview";
  const isTopTab = topTabs.some((t) => t.value === activeRoute);
  const subPageLabel = subPageLabels[activeRoute];

  return (
    <div className="space-y-4">
      {/* Header */}
      <div className="flex items-center gap-3">
        <Button
          variant="ghost"
          size="icon"
          className="h-8 w-8 shrink-0"
          onClick={() => navigate("/domains")}
        >
          <ArrowLeft className="h-4 w-4" />
        </Button>
        {isLoading ? (
          <Skeleton className="h-8 w-48" />
        ) : (
          <div className="flex items-center gap-3 min-w-0">
            <h1 className="text-xl font-bold truncate">{domain?.name}</h1>
            {domain && <StatusBadge status={domain.status} />}
            {isSubdomain && (
              <Badge variant="outline" className="text-xs">Subdomain</Badge>
            )}
            {isSubdomain && parentDomain && (
              <span className="text-xs text-muted-foreground">
                of{" "}
                <button
                  className="text-pink-500 hover:underline"
                  onClick={() => navigate(`/domains/${parentDomain.id}/overview`)}
                >
                  {parentDomain.name}
                </button>
              </span>
            )}
          </div>
        )}
      </div>

      {/* Top-level tabs (Plesk-style: only 4) */}
      <div className="border-b border-border">
        <nav className="flex gap-6">
          {tabs.map((tab) => {
            const isActive =
              activeRoute === tab.value ||
              (tab.value === "overview" && !isTopTab);
            return (
              <button
                key={tab.value}
                onClick={() => navigate(`/domains/${id}/${tab.value}`)}
                className={cn(
                  "pb-2.5 text-sm font-medium border-b-2 transition-colors",
                  isActive
                    ? "border-pink-500 text-foreground"
                    : "border-transparent text-muted-foreground hover:text-foreground"
                )}
              >
                {tab.label}
              </button>
            );
          })}
        </nav>
      </div>

      {/* Breadcrumb for sub-pages */}
      {!isTopTab && subPageLabel && (
        <div className="flex items-center gap-1.5 text-sm">
          <button
            onClick={() => navigate(`/domains/${id}/overview`)}
            className="text-muted-foreground hover:text-foreground transition-colors"
          >
            Dashboard
          </button>
          <ChevronRight className="h-3.5 w-3.5 text-muted-foreground" />
          <span className="font-medium">{subPageLabel}</span>
        </div>
      )}

      {/* Content */}
      <Outlet context={{ domain, isLoading }} />

      {/* Bottom info bar */}
      {domain && (
        <div className="border-t border-border pt-3 mt-6 flex flex-wrap items-center gap-x-6 gap-y-1 text-xs text-muted-foreground">
          <span>
            Website at{" "}
            <code className="text-foreground font-mono">
              {domain.document_root}
            </code>
          </span>
          <span>
            Created{" "}
            {new Date(domain.created_at).toLocaleDateString()}
          </span>
        </div>
      )}
    </div>
  );
}
