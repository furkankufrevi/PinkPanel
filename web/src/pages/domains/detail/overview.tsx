import { useNavigate, useOutletContext } from "react-router-dom";
import {
  Globe,
  FolderOpen,
  Code,
  Shield,
  Network,
  HardDrive,
  Upload,
  ScrollText,
  Archive,
  Clock,
  GitBranch,
  ExternalLink,
  Mail,
  RefreshCw,
} from "lucide-react";
import { Card, CardContent } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Skeleton } from "@/components/ui/skeleton";
import { useQuery } from "@tanstack/react-query";
import { getSSLCertificate } from "@/api/ssl";
import { getDomainMetrics } from "@/api/metrics";
import type { Domain } from "@/types/domain";

interface DomainContext {
  domain: Domain | undefined;
  isLoading: boolean;
}

export function DomainOverview() {
  const { domain, isLoading } = useOutletContext<DomainContext>();
  const navigate = useNavigate();
  const domainId = domain?.id;

  const { data: ssl } = useQuery({
    queryKey: ["ssl", domainId],
    queryFn: () => getSSLCertificate(domainId!),
    enabled: !!domainId,
  });

  const { data: metricsData } = useQuery({
    queryKey: ["domain-metrics", domainId],
    queryFn: () => getDomainMetrics(domainId!, 168),
    enabled: !!domainId,
  });

  if (isLoading) {
    return (
      <div className="grid gap-6 md:grid-cols-[200px_1fr]">
        <Skeleton className="h-48" />
        <div className="space-y-4">
          <Skeleton className="h-8 w-32" />
          <div className="grid grid-cols-3 gap-3">
            {Array.from({ length: 6 }).map((_, i) => (
              <Skeleton key={i} className="h-14" />
            ))}
          </div>
        </div>
      </div>
    );
  }

  if (!domain) return null;

  const sslInstalled = ssl?.installed ?? false;
  const currentMetrics = metricsData?.data?.current;

  const categories = buildCategories(domain, sslInstalled);

  return (
    <div className="grid gap-6 md:grid-cols-[200px_1fr]">
      {/* Left sidebar — Statistics */}
      <div className="space-y-4">
        {/* Domain preview placeholder */}
        <Card className="overflow-hidden">
          <div className="h-28 bg-gradient-to-br from-pink-500/10 via-violet-500/10 to-cyan-500/10 flex items-center justify-center">
            <Globe className="h-10 w-10 text-muted-foreground/30" />
          </div>
        </Card>

        {/* Statistics */}
        <Card>
          <CardContent className="pt-4 space-y-3">
            <div className="flex items-center justify-between">
              <span className="text-sm font-medium">Statistics</span>
              <RefreshCw className="h-3.5 w-3.5 text-muted-foreground" />
            </div>
            <div className="space-y-2 text-sm">
              <div className="flex justify-between">
                <span className="text-muted-foreground">Disk space</span>
                <span className="font-medium">
                  {currentMetrics
                    ? formatBytes(currentMetrics.disk_usage_bytes)
                    : "0 MB"}
                </span>
              </div>
              <div className="flex justify-between">
                <span className="text-muted-foreground">Traffic</span>
                <span className="font-medium">
                  {currentMetrics
                    ? formatBytes(currentMetrics.bandwidth_bytes)
                    : "0 MB"}
                </span>
              </div>
            </div>
          </CardContent>
        </Card>

        {/* Visit site button */}
        <Button
          variant="outline"
          size="sm"
          className="w-full"
          onClick={() => window.open(`http://${domain.name}`, "_blank")}
        >
          <ExternalLink className="mr-2 h-3.5 w-3.5" />
          Visit Site
        </Button>
      </div>

      {/* Main content — Categorized feature cards */}
      <div className="space-y-6">
        {categories.map((category) => (
          <div key={category.title}>
            <h3 className="text-sm font-semibold text-foreground mb-3">
              {category.title}
            </h3>
            <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-1">
              {category.items.map((item) => (
                <button
                  key={item.title}
                  onClick={() => navigate(item.route)}
                  className="flex items-center gap-3 rounded-lg px-3 py-2.5 text-left transition-colors hover:bg-accent group"
                >
                  <div
                    className={`h-9 w-9 rounded-lg ${item.bg} flex items-center justify-center shrink-0`}
                  >
                    <item.icon className={`h-4.5 w-4.5 ${item.color}`} />
                  </div>
                  <div className="min-w-0">
                    <div className="text-sm font-medium group-hover:text-foreground">
                      {item.title}
                    </div>
                    {item.subtitle && (
                      <div
                        className={`text-xs ${
                          item.subtitleColor || "text-muted-foreground"
                        }`}
                      >
                        {item.subtitle}
                      </div>
                    )}
                  </div>
                </button>
              ))}
            </div>
          </div>
        ))}
      </div>
    </div>
  );
}

interface FeatureItem {
  title: string;
  subtitle?: string;
  subtitleColor?: string;
  icon: typeof Globe;
  color: string;
  bg: string;
  route: string;
}

interface Category {
  title: string;
  items: FeatureItem[];
}

function buildCategories(domain: Domain, sslInstalled: boolean): Category[] {
  const id = domain.id;
  const isSubdomain = !!domain.parent_id;

  const categories: Category[] = [];

  // Files & Databases
  categories.push({
    title: "Files & Databases",
    items: [
      {
        title: "Files",
        icon: FolderOpen,
        color: "text-blue-500",
        bg: "bg-blue-500/10",
        route: `/domains/${id}/files`,
      },
      {
        title: "Databases",
        icon: HardDrive,
        color: "text-emerald-500",
        bg: "bg-emerald-500/10",
        route: `/domains/${id}/databases`,
      },
      {
        title: "FTP",
        icon: Upload,
        color: "text-purple-500",
        bg: "bg-purple-500/10",
        route: `/domains/${id}/ftp`,
      },
      {
        title: "Backups",
        icon: Archive,
        color: "text-orange-500",
        bg: "bg-orange-500/10",
        route: `/domains/${id}/backups`,
      },
    ],
  });

  // Dev Tools
  const devTools: FeatureItem[] = [
    {
      title: "PHP",
      subtitle: `Version ${domain.php_version}`,
      icon: Code,
      color: "text-indigo-500",
      bg: "bg-indigo-500/10",
      route: `/domains/${id}/php`,
    },
    {
      title: "Logs",
      icon: ScrollText,
      color: "text-slate-500",
      bg: "bg-slate-500/10",
      route: `/domains/${id}/logs`,
    },
    {
      title: "Cron Jobs",
      icon: Clock,
      color: "text-teal-500",
      bg: "bg-teal-500/10",
      route: `/domains/${id}/crons`,
    },
    {
      title: "Git",
      icon: GitBranch,
      color: "text-red-500",
      bg: "bg-red-500/10",
      route: `/domains/${id}/git`,
    },
  ];
  categories.push({ title: "Dev Tools", items: devTools });

  // Security
  const security: FeatureItem[] = [
    {
      title: "SSL/TLS Certificates",
      subtitle: sslInstalled ? "Secured" : "Not secured",
      subtitleColor: sslInstalled ? "text-emerald-500" : "text-red-500",
      icon: Shield,
      color: sslInstalled ? "text-emerald-500" : "text-amber-500",
      bg: sslInstalled ? "bg-emerald-500/10" : "bg-amber-500/10",
      route: `/domains/${id}/ssl`,
    },
    {
      title: "Redirects",
      icon: ExternalLink,
      color: "text-amber-500",
      bg: "bg-amber-500/10",
      route: `/domains/${id}/redirects`,
    },
  ];
  categories.push({ title: "Security", items: security });

  // Email (only for root domains typically, but show for all)
  categories.push({
    title: "Mail",
    items: [
      {
        title: "Email Accounts",
        icon: Mail,
        color: "text-blue-500",
        bg: "bg-blue-500/10",
        route: `/domains/${id}/email`,
      },
    ],
  });

  // Subdomains card for root domains
  if (!isSubdomain) {
    categories[0].items.unshift({
      title: "Subdomains",
      icon: Network,
      color: "text-violet-500",
      bg: "bg-violet-500/10",
      route: "/domains",
    });
  }

  return categories;
}

function formatBytes(bytes: number): string {
  if (!bytes || bytes === 0) return "0 B";
  const units = ["B", "KB", "MB", "GB", "TB"];
  const i = Math.floor(Math.log(bytes) / Math.log(1024));
  return `${(bytes / Math.pow(1024, i)).toFixed(i > 0 ? 1 : 0)} ${units[i]}`;
}
