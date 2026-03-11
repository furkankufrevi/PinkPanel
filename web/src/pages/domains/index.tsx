import { useState, useMemo } from "react";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { useNavigate } from "react-router-dom";
import {
  Globe,
  Plus,
  Search,
  MoreVertical,
  ExternalLink,
  Pause,
  Play,
  Trash2,
  Code,
  FolderOpen,
  Network,
} from "lucide-react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import {
  Card,
  CardContent,
} from "@/components/ui/card";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { Skeleton } from "@/components/ui/skeleton";
import { ConfirmDialog } from "@/components/shared/confirm-dialog";
import { StatusBadge } from "@/components/shared/status-badge";
import { CreateDomainSheet } from "./create-domain-sheet";
import { CreateSubdomainSheet } from "./create-subdomain-sheet";
import {
  listDomains,
  suspendDomain,
  activateDomain,
  deleteDomain,
} from "@/api/domains";
import { toast } from "sonner";
import type { Domain } from "@/types/domain";
import type { AxiosError } from "axios";
import type { APIError } from "@/types/api";

export function DomainsPage() {
  const [search, setSearch] = useState("");
  const [statusFilter, setStatusFilter] = useState("all");
  const [page, setPage] = useState(1);
  const [createOpen, setCreateOpen] = useState(false);
  const [createSubOpen, setCreateSubOpen] = useState(false);
  const [deleteTarget, setDeleteTarget] = useState<Domain | null>(null);
  const queryClient = useQueryClient();
  const navigate = useNavigate();

  const { data, isLoading } = useQuery({
    queryKey: ["domains", { search, status: statusFilter, page }],
    queryFn: () =>
      listDomains({
        search: search || undefined,
        status: statusFilter === "all" ? undefined : statusFilter,
        page,
        per_page: 100,
      }),
  });

  const suspendMutation = useMutation({
    mutationFn: (id: number) => suspendDomain(id),
    onSuccess: () => {
      toast.success("Domain suspended");
      queryClient.invalidateQueries({ queryKey: ["domains"] });
    },
    onError: (err: AxiosError<APIError>) => {
      toast.error(
        err.response?.data?.error?.message ?? "Failed to suspend domain"
      );
    },
  });

  const activateMutation = useMutation({
    mutationFn: (id: number) => activateDomain(id),
    onSuccess: () => {
      toast.success("Domain activated");
      queryClient.invalidateQueries({ queryKey: ["domains"] });
    },
    onError: (err: AxiosError<APIError>) => {
      toast.error(
        err.response?.data?.error?.message ?? "Failed to activate domain"
      );
    },
  });

  const deleteMutation = useMutation({
    mutationFn: (id: number) => deleteDomain(id, true),
    onSuccess: () => {
      toast.success("Domain deleted");
      setDeleteTarget(null);
      queryClient.invalidateQueries({ queryKey: ["domains"] });
    },
    onError: (err: AxiosError<APIError>) => {
      toast.error(
        err.response?.data?.error?.message ?? "Failed to delete domain"
      );
    },
  });

  const allDomains = data?.data ?? [];

  // Group: root domains and their children
  const { rootDomains, childrenMap } = useMemo(() => {
    const roots: Domain[] = [];
    const children: Record<number, Domain[]> = {};

    for (const d of allDomains) {
      if (d.parent_id == null) {
        roots.push(d);
      } else {
        if (!children[d.parent_id]) children[d.parent_id] = [];
        children[d.parent_id].push(d);
      }
    }

    return { rootDomains: roots, childrenMap: children };
  }, [allDomains]);

  function DomainCard({ domain, isChild }: { domain: Domain; isChild?: boolean }) {
    return (
      <Card
        className={`cursor-pointer transition-all hover:ring-2 hover:ring-pink-500/20 hover:shadow-md ${isChild ? "ml-6 border-l-2 border-l-pink-500/30" : ""}`}
        onClick={() => navigate(`/domains/${domain.id}/overview`)}
      >
        <CardContent className="pt-0">
          <div className="flex items-start justify-between">
            <div className="flex items-center gap-3 min-w-0">
              <div className={`rounded-lg p-2.5 shrink-0 ${isChild ? "bg-blue-500/10" : "bg-pink-500/10"}`}>
                {isChild ? (
                  <Network className="h-5 w-5 text-blue-500" />
                ) : (
                  <Globe className="h-5 w-5 text-pink-500" />
                )}
              </div>
              <div className="min-w-0">
                <p className="font-semibold truncate">{domain.name}</p>
                <div className="flex items-center gap-2 mt-1">
                  <StatusBadge status={domain.status} />
                  {isChild && (
                    <span className="text-xs text-muted-foreground">Subdomain</span>
                  )}
                </div>
              </div>
            </div>
            <DropdownMenu>
              <DropdownMenuTrigger
                onClick={(e) => e.stopPropagation()}
              >
                <Button variant="ghost" size="icon" className="h-8 w-8 shrink-0">
                  <MoreVertical className="h-4 w-4" />
                </Button>
              </DropdownMenuTrigger>
              <DropdownMenuContent align="end">
                <DropdownMenuItem
                  onClick={(e) => {
                    e.stopPropagation();
                    window.open(`http://${domain.name}`, "_blank");
                  }}
                >
                  <ExternalLink className="mr-2 h-4 w-4" />
                  Visit Site
                </DropdownMenuItem>
                <DropdownMenuSeparator />
                {domain.status === "active" ? (
                  <DropdownMenuItem
                    onClick={(e) => {
                      e.stopPropagation();
                      suspendMutation.mutate(domain.id);
                    }}
                  >
                    <Pause className="mr-2 h-4 w-4" />
                    Suspend
                  </DropdownMenuItem>
                ) : (
                  <DropdownMenuItem
                    onClick={(e) => {
                      e.stopPropagation();
                      activateMutation.mutate(domain.id);
                    }}
                  >
                    <Play className="mr-2 h-4 w-4" />
                    Activate
                  </DropdownMenuItem>
                )}
                <DropdownMenuSeparator />
                <DropdownMenuItem
                  className="text-red-500 focus:text-red-500"
                  onClick={(e) => {
                    e.stopPropagation();
                    setDeleteTarget(domain);
                  }}
                >
                  <Trash2 className="mr-2 h-4 w-4" />
                  Delete
                </DropdownMenuItem>
              </DropdownMenuContent>
            </DropdownMenu>
          </div>

          {/* Info row */}
          <div className="flex items-center gap-4 mt-3 pt-3 border-t border-border">
            <div className="flex items-center gap-1.5 text-xs text-muted-foreground">
              <Code className="h-3 w-3" />
              PHP {domain.php_version}
            </div>
            <div className="flex items-center gap-1.5 text-xs text-muted-foreground truncate">
              <FolderOpen className="h-3 w-3 shrink-0" />
              <span className="truncate">{domain.document_root}</span>
            </div>
          </div>
        </CardContent>
      </Card>
    );
  }

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <h1 className="text-2xl font-bold">Domains</h1>
        <div className="flex items-center gap-2">
          <Button
            variant="outline"
            onClick={() => setCreateSubOpen(true)}
          >
            <Network className="mr-2 h-4 w-4" />
            Add Subdomain
          </Button>
          <Button
            onClick={() => setCreateOpen(true)}
            className="bg-pink-500 hover:bg-pink-600"
          >
            <Plus className="mr-2 h-4 w-4" />
            Add Domain
          </Button>
        </div>
      </div>

      {/* Filters */}
      <div className="flex items-center gap-3">
        <div className="relative flex-1 max-w-sm">
          <Search className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground" />
          <Input
            placeholder="Search domains..."
            value={search}
            onChange={(e) => {
              setSearch(e.target.value);
              setPage(1);
            }}
            className="pl-9"
          />
        </div>
        <Select
          value={statusFilter}
          onValueChange={(v) => {
            if (v) setStatusFilter(v);
            setPage(1);
          }}
        >
          <SelectTrigger className="w-[140px]">
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="all">All Status</SelectItem>
            <SelectItem value="active">Active</SelectItem>
            <SelectItem value="suspended">Suspended</SelectItem>
          </SelectContent>
        </Select>
      </div>

      {/* Domain cards */}
      {isLoading ? (
        <div className="grid gap-3 sm:grid-cols-2 lg:grid-cols-3">
          {Array.from({ length: 6 }).map((_, i) => (
            <Skeleton key={i} className="h-36 rounded-xl" />
          ))}
        </div>
      ) : rootDomains.length === 0 && allDomains.length === 0 ? (
        <Card>
          <CardContent className="flex flex-col items-center justify-center py-16 text-center">
            <div className="rounded-full bg-muted p-4 mb-4">
              <Globe className="h-10 w-10 text-muted-foreground" />
            </div>
            <h3 className="text-lg font-medium mb-1">No domains yet</h3>
            <p className="text-sm text-muted-foreground mb-4">
              Add your first domain to get started
            </p>
            <Button
              onClick={() => setCreateOpen(true)}
              className="bg-pink-500 hover:bg-pink-600"
            >
              <Plus className="mr-2 h-4 w-4" />
              Add Domain
            </Button>
          </CardContent>
        </Card>
      ) : (
        <div className="space-y-3">
          {rootDomains.map((domain) => (
            <div key={domain.id} className="space-y-2">
              <DomainCard domain={domain} />
              {childrenMap[domain.id]?.map((child) => (
                <DomainCard key={child.id} domain={child} isChild />
              ))}
            </div>
          ))}
        </div>
      )}

      {/* Pagination */}
      {data && data.total > 100 && (
        <div className="flex items-center justify-center gap-2">
          <Button
            variant="outline"
            size="sm"
            disabled={page <= 1}
            onClick={() => setPage(page - 1)}
          >
            Previous
          </Button>
          <span className="text-sm text-muted-foreground">
            Page {page} of {Math.ceil(data.total / 100)}
          </span>
          <Button
            variant="outline"
            size="sm"
            disabled={page >= Math.ceil(data.total / 100)}
            onClick={() => setPage(page + 1)}
          >
            Next
          </Button>
        </div>
      )}

      <CreateDomainSheet open={createOpen} onOpenChange={setCreateOpen} />
      <CreateSubdomainSheet
        open={createSubOpen}
        onOpenChange={setCreateSubOpen}
        parentDomains={rootDomains}
      />

      <ConfirmDialog
        open={!!deleteTarget}
        onOpenChange={(open) => !open && setDeleteTarget(null)}
        title={deleteTarget?.parent_id ? "Delete Subdomain" : "Delete Domain"}
        description={`This will permanently delete ${deleteTarget?.name} and all its files, configuration, and data. This action cannot be undone.`}
        typeToConfirm={deleteTarget?.name}
        confirmText={deleteTarget?.parent_id ? "Delete Subdomain" : "Delete Domain"}
        destructive
        loading={deleteMutation.isPending}
        onConfirm={() =>
          deleteTarget && deleteMutation.mutate(deleteTarget.id)
        }
      />
    </div>
  );
}
