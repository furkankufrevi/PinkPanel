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
  Settings,
  ExternalLink,
  Copy,
  CheckCircle2,
  XCircle,
} from "lucide-react";
import {
  Card,
  CardContent,
  CardHeader,
  CardTitle,
  CardDescription,
} from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Separator } from "@/components/ui/separator";
import { Skeleton } from "@/components/ui/skeleton";
import { toast } from "sonner";
import { useQuery } from "@tanstack/react-query";
import { getSSLCertificate } from "@/api/ssl";
import { listDomains } from "@/api/domains";
import { listDNSRecords } from "@/api/dns";
import type { Domain } from "@/types/domain";

interface DomainContext {
  domain: Domain | undefined;
  isLoading: boolean;
}

function copyToClipboard(text: string) {
  navigator.clipboard.writeText(text);
  toast.success("Copied to clipboard");
}

export function DomainOverview() {
  const { domain, isLoading } = useOutletContext<DomainContext>();
  const navigate = useNavigate();
  const domainId = domain?.id;
  const isSubdomain = !!domain?.parent_id;

  const { data: ssl } = useQuery({
    queryKey: ["ssl", domainId],
    queryFn: () => getSSLCertificate(domainId!),
    enabled: !!domainId,
  });

  // Fetch child subdomains for root domains
  const { data: allDomains } = useQuery({
    queryKey: ["domains", { per_page: 100 }],
    queryFn: () => listDomains({ per_page: 100 }),
    enabled: !!domainId && !isSubdomain,
  });

  const { data: dnsRecords } = useQuery({
    queryKey: ["dns", domainId],
    queryFn: () => listDNSRecords(domainId!),
    enabled: !!domainId && (!isSubdomain || domain?.separate_dns),
  });

  if (isLoading) {
    return (
      <div className="space-y-6">
        <div className="grid gap-4 md:grid-cols-3">
          {Array.from({ length: 6 }).map((_, i) => (
            <Skeleton key={i} className="h-32 rounded-xl" />
          ))}
        </div>
      </div>
    );
  }

  if (!domain) return null;

  const sslInstalled = ssl?.installed ?? false;
  const childSubdomains = allDomains?.data?.filter((d) => d.parent_id === domain.id) ?? [];
  const subdomainCount = childSubdomains.length;
  const dnsCount = dnsRecords?.data?.length ?? 0;

  const featureCards = [];

  // Only show subdomains card for root domains
  if (!isSubdomain) {
    featureCards.push({
      title: "Subdomains",
      description: `${subdomainCount} subdomain${subdomainCount !== 1 ? "s" : ""} configured`,
      icon: Network,
      action: () => navigate("/domains"),
      color: "text-blue-500",
      bg: "bg-blue-500/10",
    });
  }

  // DNS card: show for root domains, or subdomains with separate_dns
  if (!isSubdomain || domain.separate_dns) {
    featureCards.push({
      title: "DNS Zone",
      description: `${dnsCount} record${dnsCount !== 1 ? "s" : ""} configured`,
      icon: Globe,
      action: () => navigate(`/domains/${domain.id}/dns`),
      color: "text-violet-500",
      bg: "bg-violet-500/10",
    });
  } else {
    featureCards.push({
      title: "DNS Zone",
      description: "Managed by parent domain",
      icon: Globe,
      action: () => navigate(`/domains/${domain.parent_id}/dns`),
      color: "text-violet-500/50",
      bg: "bg-violet-500/5",
    });
  }

  featureCards.push(
    {
      title: "SSL/TLS",
      description: sslInstalled ? "Certificate installed" : "No certificate",
      icon: Shield,
      action: () => navigate(`/domains/${domain.id}/ssl`),
      color: sslInstalled ? "text-emerald-500" : "text-amber-500",
      bg: sslInstalled ? "bg-emerald-500/10" : "bg-amber-500/10",
      badge: sslInstalled ? (
        <Badge variant="outline" className="bg-emerald-500/10 text-emerald-500 border-emerald-500/20">
          Secure
        </Badge>
      ) : (
        <Badge variant="outline" className="bg-amber-500/10 text-amber-500 border-amber-500/20">
          Not Secure
        </Badge>
      ),
    },
    {
      title: "PHP",
      description: `PHP ${domain.php_version} configured`,
      icon: Code,
      action: () => navigate(`/domains/${domain.id}/php`),
      color: "text-indigo-500",
      bg: "bg-indigo-500/10",
      badge: (
        <Badge variant="outline" className="bg-indigo-500/10 text-indigo-500 border-indigo-500/20">
          {domain.php_version}
        </Badge>
      ),
    },
    {
      title: "File Manager",
      description: "Browse and manage files",
      icon: FolderOpen,
      action: () => navigate(`/domains/${domain.id}/files`),
      color: "text-orange-500",
      bg: "bg-orange-500/10",
    },
    {
      title: "Databases",
      description: "MySQL databases",
      icon: HardDrive,
      action: () => navigate(`/domains/${domain.id}/databases`),
      color: "text-cyan-500",
      bg: "bg-cyan-500/10",
    },
    {
      title: "FTP Accounts",
      description: "FTP access management",
      icon: Upload,
      action: () => navigate(`/domains/${domain.id}/ftp`),
      color: "text-pink-500",
      bg: "bg-pink-500/10",
    },
    {
      title: "Logs",
      description: "Access & error logs",
      icon: ScrollText,
      action: () => navigate(`/domains/${domain.id}/logs`),
      color: "text-gray-500",
      bg: "bg-gray-500/10",
    },
    {
      title: "Backups",
      description: "Backup & restore",
      icon: Archive,
      action: () => navigate(`/domains/${domain.id}/backups`),
      color: "text-teal-500",
      bg: "bg-teal-500/10",
    },
    {
      title: "Settings",
      description: "Domain configuration",
      icon: Settings,
      action: () => navigate(`/domains/${domain.id}/settings`),
      color: "text-rose-500",
      bg: "bg-rose-500/10",
    },
  );

  return (
    <div className="space-y-6">
      {/* Domain info summary */}
      <Card>
        <CardContent className="pt-0">
          <div className="flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
            <div className="space-y-1">
              <div className="flex items-center gap-2">
                <span className="text-sm text-muted-foreground">Document Root</span>
              </div>
              <div className="flex items-center gap-2">
                <code className="text-sm font-mono bg-muted px-2 py-1 rounded">
                  {domain.document_root}
                </code>
                <Button
                  variant="ghost"
                  size="icon"
                  className="h-6 w-6"
                  onClick={() => copyToClipboard(domain.document_root)}
                >
                  <Copy className="h-3 w-3" />
                </Button>
              </div>
            </div>
            <Separator orientation="vertical" className="hidden sm:block h-10" />
            <div className="space-y-1">
              <span className="text-sm text-muted-foreground">Status</span>
              <div className="flex items-center gap-2">
                {domain.status === "active" ? (
                  <CheckCircle2 className="h-4 w-4 text-emerald-500" />
                ) : (
                  <XCircle className="h-4 w-4 text-amber-500" />
                )}
                <span className="text-sm font-medium capitalize">{domain.status}</span>
              </div>
            </div>
            <Separator orientation="vertical" className="hidden sm:block h-10" />
            <div className="space-y-1">
              <span className="text-sm text-muted-foreground">Created</span>
              <p className="text-sm font-medium">
                {new Date(domain.created_at).toLocaleDateString()}
              </p>
            </div>
            <div className="sm:ml-auto">
              <Button
                variant="outline"
                size="sm"
                onClick={() => window.open(`http://${domain.name}`, "_blank")}
              >
                <ExternalLink className="mr-2 h-3.5 w-3.5" />
                Visit Site
              </Button>
            </div>
          </div>
        </CardContent>
      </Card>

      {/* Feature cards grid */}
      <div className="grid gap-3 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-5">
        {featureCards.map((card) => (
          <Card
            key={card.title}
            className="cursor-pointer transition-all hover:ring-2 hover:ring-pink-500/20 hover:shadow-md"
            onClick={card.action}
          >
            <CardHeader className="pb-2">
              <div className="flex items-center justify-between">
                <div className={`rounded-lg p-2 ${card.bg}`}>
                  <card.icon className={`h-5 w-5 ${card.color}`} />
                </div>
                {card.badge}
              </div>
            </CardHeader>
            <CardContent>
              <CardTitle className="text-sm">{card.title}</CardTitle>
              <CardDescription className="text-xs mt-0.5">
                {card.description}
              </CardDescription>
            </CardContent>
          </Card>
        ))}
      </div>
    </div>
  );
}
