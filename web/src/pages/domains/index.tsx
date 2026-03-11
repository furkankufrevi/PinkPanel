import { useState, useMemo } from "react";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { Globe, Plus, Search } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { DataTable, DataTablePagination } from "@/components/shared/data-table";
import { ConfirmDialog } from "@/components/shared/confirm-dialog";
import { getDomainColumns } from "./columns";
import { CreateDomainSheet } from "./create-domain-sheet";
import { listDomains, suspendDomain, activateDomain, deleteDomain } from "@/api/domains";
import { toast } from "sonner";
import type { Domain } from "@/types/domain";
import type { AxiosError } from "axios";
import type { APIError } from "@/types/api";

export function DomainsPage() {
  const [search, setSearch] = useState("");
  const [statusFilter, setStatusFilter] = useState("all");
  const [page, setPage] = useState(1);
  const [createOpen, setCreateOpen] = useState(false);
  const [deleteTarget, setDeleteTarget] = useState<Domain | null>(null);
  const queryClient = useQueryClient();

  const { data, isLoading } = useQuery({
    queryKey: ["domains", { search, status: statusFilter, page }],
    queryFn: () =>
      listDomains({
        search: search || undefined,
        status: statusFilter === "all" ? undefined : statusFilter,
        page,
        per_page: 20,
      }),
  });

  const suspendMutation = useMutation({
    mutationFn: (id: number) => suspendDomain(id),
    onSuccess: () => {
      toast.success("Domain suspended");
      queryClient.invalidateQueries({ queryKey: ["domains"] });
    },
    onError: (err: AxiosError<APIError>) => {
      toast.error(err.response?.data?.error?.message ?? "Failed to suspend domain");
    },
  });

  const activateMutation = useMutation({
    mutationFn: (id: number) => activateDomain(id),
    onSuccess: () => {
      toast.success("Domain activated");
      queryClient.invalidateQueries({ queryKey: ["domains"] });
    },
    onError: (err: AxiosError<APIError>) => {
      toast.error(err.response?.data?.error?.message ?? "Failed to activate domain");
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
      toast.error(err.response?.data?.error?.message ?? "Failed to delete domain");
    },
  });

  const columns = useMemo(
    () =>
      getDomainColumns({
        onSuspend: (d) => suspendMutation.mutate(d.id),
        onActivate: (d) => activateMutation.mutate(d.id),
        onDelete: (d) => setDeleteTarget(d),
      }),
    [suspendMutation, activateMutation]
  );

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <h1 className="text-2xl font-bold">Domains</h1>
        <Button
          onClick={() => setCreateOpen(true)}
          className="bg-pink-500 hover:bg-pink-600"
        >
          <Plus className="mr-2 h-4 w-4" />
          Add Domain
        </Button>
      </div>

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

      <DataTable
        columns={columns}
        data={data?.data ?? []}
        loading={isLoading}
        emptyTitle="No domains yet"
        emptyDescription="Add your first domain to get started"
        emptyIcon={<Globe className="h-12 w-12" />}
        emptyAction={
          <Button
            onClick={() => setCreateOpen(true)}
            className="bg-pink-500 hover:bg-pink-600"
          >
            <Plus className="mr-2 h-4 w-4" />
            Add Domain
          </Button>
        }
      />

      {data && (
        <DataTablePagination
          page={page}
          perPage={20}
          total={data.total}
          onPageChange={setPage}
        />
      )}

      <CreateDomainSheet open={createOpen} onOpenChange={setCreateOpen} />

      <ConfirmDialog
        open={!!deleteTarget}
        onOpenChange={(open) => !open && setDeleteTarget(null)}
        title="Delete Domain"
        description={`This will permanently delete ${deleteTarget?.name} and all its files, configuration, and data. This action cannot be undone.`}
        typeToConfirm={deleteTarget?.name}
        confirmText="Delete Domain"
        destructive
        loading={deleteMutation.isPending}
        onConfirm={() => deleteTarget && deleteMutation.mutate(deleteTarget.id)}
      />
    </div>
  );
}
