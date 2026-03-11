import { useState } from "react";
import { useParams } from "react-router-dom";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { type ColumnDef } from "@tanstack/react-table";
import { Plus, Pencil, Trash2, Globe } from "lucide-react";
import { Button } from "@/components/ui/button";
import { DataTable } from "@/components/shared/data-table";
import { ConfirmDialog } from "@/components/shared/confirm-dialog";
import { DNSRecordSheet } from "./dns-record-sheet";
import { listDNSRecords, deleteDNSRecord } from "@/api/dns";
import { toast } from "sonner";
import type { DNSRecord } from "@/types/dns";
import type { AxiosError } from "axios";
import type { APIError } from "@/types/api";

const typeColors: Record<string, string> = {
  A: "bg-blue-500/10 text-blue-500",
  AAAA: "bg-indigo-500/10 text-indigo-500",
  CNAME: "bg-purple-500/10 text-purple-500",
  MX: "bg-amber-500/10 text-amber-500",
  TXT: "bg-emerald-500/10 text-emerald-500",
  NS: "bg-pink-500/10 text-pink-500",
  SOA: "bg-red-500/10 text-red-500",
  SRV: "bg-cyan-500/10 text-cyan-500",
  CAA: "bg-orange-500/10 text-orange-500",
};

export function DomainDNS() {
  const { id } = useParams<{ id: string }>();
  const domainId = Number(id);
  const queryClient = useQueryClient();

  const [sheetOpen, setSheetOpen] = useState(false);
  const [editRecord, setEditRecord] = useState<DNSRecord | null>(null);
  const [deleteTarget, setDeleteTarget] = useState<DNSRecord | null>(null);

  const { data, isLoading } = useQuery({
    queryKey: ["dns-records", domainId],
    queryFn: () => listDNSRecords(domainId),
    enabled: !!domainId,
  });

  const deleteMutation = useMutation({
    mutationFn: (recordId: number) => deleteDNSRecord(recordId),
    onSuccess: () => {
      toast.success("DNS record deleted");
      setDeleteTarget(null);
      queryClient.invalidateQueries({ queryKey: ["dns-records", domainId] });
    },
    onError: (err: AxiosError<APIError>) => {
      toast.error(err.response?.data?.error?.message ?? "Failed to delete record");
    },
  });

  const columns: ColumnDef<DNSRecord, unknown>[] = [
    {
      accessorKey: "type",
      header: "Type",
      cell: ({ row }) => {
        const t = row.original.type;
        return (
          <span
            className={`inline-flex items-center rounded px-2 py-0.5 text-xs font-semibold ${typeColors[t] ?? "bg-muted text-muted-foreground"}`}
          >
            {t}
          </span>
        );
      },
    },
    {
      accessorKey: "name",
      header: "Name",
      cell: ({ row }) => (
        <span className="font-mono text-sm">{row.original.name}</span>
      ),
    },
    {
      accessorKey: "value",
      header: "Value",
      cell: ({ row }) => (
        <span className="font-mono text-sm truncate max-w-[300px] block">
          {row.original.value}
        </span>
      ),
    },
    {
      accessorKey: "ttl",
      header: "TTL",
      cell: ({ row }) => (
        <span className="text-sm text-muted-foreground">{row.original.ttl}</span>
      ),
    },
    {
      accessorKey: "priority",
      header: "Priority",
      cell: ({ row }) => (
        <span className="text-sm text-muted-foreground">
          {row.original.priority ?? "—"}
        </span>
      ),
    },
    {
      id: "actions",
      header: "",
      enableSorting: false,
      cell: ({ row }) => {
        const record = row.original;
        if (record.type === "SOA") return null;
        return (
          <div className="flex items-center gap-1">
            <Button
              variant="ghost"
              size="icon"
              className="h-8 w-8"
              onClick={() => {
                setEditRecord(record);
                setSheetOpen(true);
              }}
            >
              <Pencil className="h-3.5 w-3.5" />
            </Button>
            <Button
              variant="ghost"
              size="icon"
              className="h-8 w-8 text-destructive"
              onClick={() => setDeleteTarget(record)}
            >
              <Trash2 className="h-3.5 w-3.5" />
            </Button>
          </div>
        );
      },
    },
  ];

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <h2 className="text-lg font-semibold">DNS Records</h2>
        <Button
          onClick={() => {
            setEditRecord(null);
            setSheetOpen(true);
          }}
          className="bg-pink-500 hover:bg-pink-600"
        >
          <Plus className="mr-2 h-4 w-4" />
          Add Record
        </Button>
      </div>

      <DataTable
        columns={columns}
        data={data?.data ?? []}
        loading={isLoading}
        emptyTitle="No DNS records"
        emptyDescription="Add DNS records for this domain"
        emptyIcon={<Globe className="h-12 w-12" />}
        emptyAction={
          <Button
            onClick={() => {
              setEditRecord(null);
              setSheetOpen(true);
            }}
            className="bg-pink-500 hover:bg-pink-600"
          >
            <Plus className="mr-2 h-4 w-4" />
            Add Record
          </Button>
        }
      />

      <DNSRecordSheet
        open={sheetOpen}
        onOpenChange={(open) => {
          setSheetOpen(open);
          if (!open) setEditRecord(null);
        }}
        domainId={domainId}
        record={editRecord}
      />

      <ConfirmDialog
        open={!!deleteTarget}
        onOpenChange={(open) => !open && setDeleteTarget(null)}
        title="Delete DNS Record"
        description={`Delete the ${deleteTarget?.type} record for "${deleteTarget?.name}"?`}
        confirmText="Delete"
        destructive
        loading={deleteMutation.isPending}
        onConfirm={() => deleteTarget && deleteMutation.mutate(deleteTarget.id)}
      />
    </div>
  );
}
