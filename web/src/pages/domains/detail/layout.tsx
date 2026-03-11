import { useParams, useNavigate, useLocation, Outlet } from "react-router-dom";
import { useQuery } from "@tanstack/react-query";
import { ArrowLeft } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Tabs, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { StatusBadge } from "@/components/shared/status-badge";
import { Skeleton } from "@/components/ui/skeleton";
import { getDomain } from "@/api/domains";

const tabs = [
  { value: "overview", label: "Overview" },
  { value: "subdomains", label: "Subdomains" },
  { value: "dns", label: "DNS" },
  { value: "ssl", label: "SSL" },
  { value: "php", label: "PHP" },
  { value: "files", label: "Files" },
  { value: "databases", label: "Databases" },
  { value: "ftp", label: "FTP" },
  { value: "logs", label: "Logs" },
  { value: "backups", label: "Backups" },
  { value: "settings", label: "Settings" },
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

  // Determine active tab from URL path
  const pathParts = location.pathname.split("/");
  const activeTab = pathParts[3] || "overview";

  function handleTabChange(tab: string) {
    navigate(`/domains/${id}/${tab}`);
  }

  return (
    <div className="space-y-6">
      <div className="flex items-center gap-4">
        <Button variant="ghost" size="icon" onClick={() => navigate("/domains")}>
          <ArrowLeft className="h-4 w-4" />
        </Button>
        {isLoading ? (
          <Skeleton className="h-8 w-48" />
        ) : (
          <div className="flex items-center gap-3">
            <h1 className="text-2xl font-bold">{domain?.name}</h1>
            {domain && <StatusBadge status={domain.status} />}
          </div>
        )}
      </div>

      <Tabs value={activeTab} onValueChange={handleTabChange}>
        <TabsList className="w-full justify-start overflow-x-auto">
          {tabs.map((tab) => (
            <TabsTrigger key={tab.value} value={tab.value}>
              {tab.label}
            </TabsTrigger>
          ))}
        </TabsList>
      </Tabs>

      <Outlet context={{ domain, isLoading }} />
    </div>
  );
}
