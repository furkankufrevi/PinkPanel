import { useMemo } from "react";
import { useParams, useNavigate, useLocation, Outlet } from "react-router-dom";
import { useQuery } from "@tanstack/react-query";
import {
  ArrowLeft,
  Globe,
  LayoutGrid,
  Network,
  Shield,
  Code,
  FolderOpen,
  HardDrive,
  Upload,
  Clock,
  GitBranch,
  Mail,
  ScrollText,
  Archive,
  Settings,
} from "lucide-react";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { StatusBadge } from "@/components/shared/status-badge";
import { Skeleton } from "@/components/ui/skeleton";
import { cn } from "@/lib/utils";
import { getDomain } from "@/api/domains";

const allTabs = [
  { value: "overview", label: "Overview", icon: LayoutGrid },
  { value: "dns", label: "DNS", icon: Globe },
  { value: "ssl", label: "SSL", icon: Shield },
  { value: "php", label: "PHP", icon: Code },
  { value: "files", label: "Files", icon: FolderOpen },
  { value: "databases", label: "Databases", icon: HardDrive },
  { value: "ftp", label: "FTP", icon: Upload },
  { value: "email", label: "Email", icon: Mail },
  { value: "crons", label: "Cron Jobs", icon: Clock },
  { value: "git", label: "Git", icon: GitBranch },
  { value: "logs", label: "Logs", icon: ScrollText },
  { value: "backups", label: "Backups", icon: Archive },
  { value: "settings", label: "Settings", icon: Settings },
];

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
    return allTabs.filter((tab) => {
      // Hide DNS tab for subdomains without separate DNS
      if (tab.value === "dns" && isSubdomain && !hasSeparateDNS) return false;
      return true;
    });
  }, [isSubdomain, hasSeparateDNS]);

  const pathParts = location.pathname.split("/");
  const activeTab = pathParts[3] || "overview";

  return (
    <div className="space-y-6">
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
            <div className="rounded-lg bg-pink-500/10 p-2">
              {isSubdomain ? (
                <Network className="h-5 w-5 text-pink-500" />
              ) : (
                <Globe className="h-5 w-5 text-pink-500" />
              )}
            </div>
            <div className="min-w-0">
              <h1 className="text-xl font-bold truncate">{domain?.name}</h1>
              {isSubdomain && parentDomain && (
                <p className="text-xs text-muted-foreground">
                  Subdomain of{" "}
                  <button
                    className="text-pink-500 hover:underline"
                    onClick={() => navigate(`/domains/${parentDomain.id}/overview`)}
                  >
                    {parentDomain.name}
                  </button>
                </p>
              )}
            </div>
            {domain && <StatusBadge status={domain.status} />}
            {isSubdomain && (
              <Badge variant="outline" className="text-xs">Subdomain</Badge>
            )}
          </div>
        )}
      </div>

      {/* Tab navigation */}
      <div className="-mx-1 overflow-x-auto pb-px">
        <nav className="flex gap-1 px-1 min-w-max">
          {tabs.map((tab) => {
            const isActive = activeTab === tab.value;
            return (
              <button
                key={tab.value}
                onClick={() => navigate(`/domains/${id}/${tab.value}`)}
                className={cn(
                  "flex items-center gap-1.5 rounded-md px-3 py-1.5 text-sm font-medium transition-colors whitespace-nowrap",
                  isActive
                    ? "bg-pink-500/10 text-pink-500"
                    : "text-muted-foreground hover:bg-accent hover:text-foreground"
                )}
              >
                <tab.icon className="h-3.5 w-3.5" />
                {tab.label}
              </button>
            );
          })}
        </nav>
      </div>

      {/* Content */}
      <Outlet context={{ domain, isLoading }} />
    </div>
  );
}
