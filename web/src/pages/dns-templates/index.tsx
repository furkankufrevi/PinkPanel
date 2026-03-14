import { useState, useRef } from "react";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { type ColumnDef } from "@tanstack/react-table";
import {
  Plus,
  Pencil,
  Trash2,
  Download,
  Upload,
  LayoutTemplate,
  Lock,
} from "lucide-react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Badge } from "@/components/ui/badge";
import { DataTable } from "@/components/shared/data-table";
import { ConfirmDialog } from "@/components/shared/confirm-dialog";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import {
  listDNSTemplates,
  createDNSTemplate,
  updateDNSTemplate,
  deleteDNSTemplate,
  exportDNSTemplate,
  importDNSTemplate,
} from "@/api/dns-templates";
import { toast } from "sonner";
import type { DNSTemplate, DNSTemplateRecord } from "@/types/dns-template";
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

const categoryColors: Record<string, string> = {
  hosting: "bg-blue-500/10 text-blue-500",
  email: "bg-amber-500/10 text-amber-500",
  custom: "bg-emerald-500/10 text-emerald-500",
};

const emptyRecord = { type: "A", name: "@", value: "", ttl: 3600, priority: null };

export default function DNSTemplatesPage() {
  const queryClient = useQueryClient();
  const fileInputRef = useRef<HTMLInputElement>(null);
  const [editDialogOpen, setEditDialogOpen] = useState(false);
  const [editTemplate, setEditTemplate] = useState<DNSTemplate | null>(null);
  const [deleteTarget, setDeleteTarget] = useState<DNSTemplate | null>(null);
  const [viewTemplate, setViewTemplate] = useState<DNSTemplate | null>(null);

  // Form state
  const [formName, setFormName] = useState("");
  const [formDesc, setFormDesc] = useState("");
  const [formCategory, setFormCategory] = useState("custom");
  const [formRecords, setFormRecords] = useState<Omit<DNSTemplateRecord, "id" | "template_id">[]>([
    { ...emptyRecord },
  ]);

  const { data, isLoading } = useQuery({
    queryKey: ["dns-templates"],
    queryFn: listDNSTemplates,
  });

  const templates = data?.data ?? [];

  const createMutation = useMutation({
    mutationFn: () =>
      createDNSTemplate({
        name: formName,
        description: formDesc,
        category: formCategory,
        records: formRecords,
      }),
    onSuccess: () => {
      toast.success("Template created");
      queryClient.invalidateQueries({ queryKey: ["dns-templates"] });
      closeEdit();
    },
    onError: (err: AxiosError<APIError>) => {
      toast.error(err.response?.data?.error?.message ?? "Failed to create template");
    },
  });

  const updateMutation = useMutation({
    mutationFn: () =>
      updateDNSTemplate(editTemplate!.id, {
        name: formName,
        description: formDesc,
        category: formCategory,
        records: formRecords,
      }),
    onSuccess: () => {
      toast.success("Template updated");
      queryClient.invalidateQueries({ queryKey: ["dns-templates"] });
      closeEdit();
    },
    onError: (err: AxiosError<APIError>) => {
      toast.error(err.response?.data?.error?.message ?? "Failed to update template");
    },
  });

  const deleteMutation = useMutation({
    mutationFn: (id: number) => deleteDNSTemplate(id),
    onSuccess: () => {
      toast.success("Template deleted");
      setDeleteTarget(null);
      queryClient.invalidateQueries({ queryKey: ["dns-templates"] });
    },
    onError: (err: AxiosError<APIError>) => {
      toast.error(err.response?.data?.error?.message ?? "Failed to delete template");
    },
  });

  const importMutation = useMutation({
    mutationFn: (json: string) => importDNSTemplate(json),
    onSuccess: () => {
      toast.success("Template imported");
      queryClient.invalidateQueries({ queryKey: ["dns-templates"] });
    },
    onError: (err: AxiosError<APIError>) => {
      toast.error(err.response?.data?.error?.message ?? "Failed to import template");
    },
  });

  function openCreate() {
    setEditTemplate(null);
    setFormName("");
    setFormDesc("");
    setFormCategory("custom");
    setFormRecords([{ ...emptyRecord }]);
    setEditDialogOpen(true);
  }

  function openEdit(t: DNSTemplate) {
    setEditTemplate(t);
    setFormName(t.name);
    setFormDesc(t.description);
    setFormCategory(t.category);
    setFormRecords(t.records.map((r) => ({ type: r.type, name: r.name, value: r.value, ttl: r.ttl, priority: r.priority })));
    setEditDialogOpen(true);
  }

  function closeEdit() {
    setEditDialogOpen(false);
    setEditTemplate(null);
  }

  function addRecord() {
    setFormRecords([...formRecords, { ...emptyRecord }]);
  }

  function removeRecord(index: number) {
    setFormRecords(formRecords.filter((_, i) => i !== index));
  }

  function updateRecord(index: number, field: string, value: string | number | null) {
    const updated = [...formRecords];
    updated[index] = { ...updated[index], [field]: value };
    setFormRecords(updated);
  }

  async function handleExport(id: number) {
    try {
      const blob = await exportDNSTemplate(id);
      const url = URL.createObjectURL(blob);
      const a = document.createElement("a");
      a.href = url;
      a.download = "dns-template.json";
      a.click();
      URL.revokeObjectURL(url);
    } catch {
      toast.error("Failed to export template");
    }
  }

  function handleImportClick() {
    fileInputRef.current?.click();
  }

  async function handleImportFile(e: React.ChangeEvent<HTMLInputElement>) {
    const file = e.target.files?.[0];
    if (!file) return;
    const text = await file.text();
    importMutation.mutate(text);
    e.target.value = "";
  }

  const columns: ColumnDef<DNSTemplate, unknown>[] = [
    {
      accessorKey: "name",
      header: "Name",
      cell: ({ row }) => (
        <div className="flex items-center gap-2">
          <span className="font-medium">{row.original.name}</span>
          {row.original.is_preset && (
            <Lock className="h-3 w-3 text-muted-foreground" />
          )}
        </div>
      ),
    },
    {
      accessorKey: "category",
      header: "Category",
      cell: ({ row }) => (
        <Badge variant="secondary" className={categoryColors[row.original.category] ?? ""}>
          {row.original.category}
        </Badge>
      ),
    },
    {
      accessorKey: "description",
      header: "Description",
      cell: ({ row }) => (
        <span className="text-sm text-muted-foreground truncate max-w-[300px] block">
          {row.original.description}
        </span>
      ),
    },
    {
      id: "records",
      header: "Records",
      cell: ({ row }) => (
        <button
          className="text-sm text-pink-500 hover:underline cursor-pointer"
          onClick={() => setViewTemplate(row.original)}
        >
          {row.original.records.length} record{row.original.records.length !== 1 ? "s" : ""}
        </button>
      ),
    },
    {
      id: "actions",
      header: "",
      enableSorting: false,
      cell: ({ row }) => {
        const t = row.original;
        return (
          <div className="flex items-center gap-1">
            <Button
              variant="ghost"
              size="icon"
              className="h-8 w-8"
              title="Export"
              onClick={() => handleExport(t.id)}
            >
              <Download className="h-3.5 w-3.5" />
            </Button>
            {!t.is_preset && (
              <>
                <Button
                  variant="ghost"
                  size="icon"
                  className="h-8 w-8"
                  onClick={() => openEdit(t)}
                >
                  <Pencil className="h-3.5 w-3.5" />
                </Button>
                <Button
                  variant="ghost"
                  size="icon"
                  className="h-8 w-8 text-destructive"
                  onClick={() => setDeleteTarget(t)}
                >
                  <Trash2 className="h-3.5 w-3.5" />
                </Button>
              </>
            )}
          </div>
        );
      },
    },
  ];

  const saving = createMutation.isPending || updateMutation.isPending;

  return (
    <div className="space-y-4 p-6">
      <div className="flex items-center justify-between">
        <h1 className="text-2xl font-bold">DNS Templates</h1>
        <div className="flex items-center gap-2">
          <input
            ref={fileInputRef}
            type="file"
            accept=".json"
            className="hidden"
            onChange={handleImportFile}
          />
          <Button variant="outline" onClick={handleImportClick}>
            <Upload className="mr-2 h-4 w-4" />
            Import
          </Button>
          <Button onClick={openCreate} className="bg-pink-500 hover:bg-pink-600">
            <Plus className="mr-2 h-4 w-4" />
            New Template
          </Button>
        </div>
      </div>

      <DataTable
        columns={columns}
        data={templates}
        loading={isLoading}
        emptyTitle="No DNS templates"
        emptyDescription="Create custom templates or import from JSON"
        emptyIcon={<LayoutTemplate className="h-12 w-12" />}
        emptyAction={
          <Button onClick={openCreate} className="bg-pink-500 hover:bg-pink-600">
            <Plus className="mr-2 h-4 w-4" />
            New Template
          </Button>
        }
      />

      {/* Create/Edit Dialog */}
      <Dialog open={editDialogOpen} onOpenChange={(v) => !v && closeEdit()}>
        <DialogContent className="max-w-2xl max-h-[90vh] overflow-y-auto">
          <DialogHeader>
            <DialogTitle>{editTemplate ? "Edit Template" : "New DNS Template"}</DialogTitle>
            <DialogDescription>
              {editTemplate
                ? "Update the template name, description, and records."
                : "Create a reusable set of DNS records. Use {{domain}}, {{ip}}, {{ipv6}}, {{hostname}} as variables."}
            </DialogDescription>
          </DialogHeader>

          <div className="space-y-4">
            <div className="grid grid-cols-2 gap-4">
              <div className="space-y-2">
                <Label>Name</Label>
                <Input value={formName} onChange={(e) => setFormName(e.target.value)} placeholder="Template name" />
              </div>
              <div className="space-y-2">
                <Label>Category</Label>
                <Select value={formCategory} onValueChange={(v) => v && setFormCategory(v)}>
                  <SelectTrigger>
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="hosting">Hosting</SelectItem>
                    <SelectItem value="email">Email</SelectItem>
                    <SelectItem value="custom">Custom</SelectItem>
                  </SelectContent>
                </Select>
              </div>
            </div>
            <div className="space-y-2">
              <Label>Description</Label>
              <Input value={formDesc} onChange={(e) => setFormDesc(e.target.value)} placeholder="Brief description" />
            </div>

            <div className="space-y-2">
              <div className="flex items-center justify-between">
                <Label>Records</Label>
                <Button variant="outline" size="sm" onClick={addRecord}>
                  <Plus className="mr-1 h-3 w-3" /> Add Record
                </Button>
              </div>
              <div className="space-y-2 max-h-[40vh] overflow-y-auto">
                {formRecords.map((r, i) => (
                  <div key={i} className="flex items-center gap-2 rounded-md border p-2">
                    <Select value={r.type} onValueChange={(v) => v && updateRecord(i, "type", v)}>
                      <SelectTrigger className="w-24">
                        <SelectValue />
                      </SelectTrigger>
                      <SelectContent>
                        {["A", "AAAA", "CNAME", "MX", "TXT", "NS", "SOA", "SRV", "CAA"].map((t) => (
                          <SelectItem key={t} value={t}>{t}</SelectItem>
                        ))}
                      </SelectContent>
                    </Select>
                    <Input
                      className="w-28"
                      value={r.name}
                      onChange={(e) => updateRecord(i, "name", e.target.value)}
                      placeholder="Name"
                    />
                    <Input
                      className="flex-1"
                      value={r.value}
                      onChange={(e) => updateRecord(i, "value", e.target.value)}
                      placeholder="Value (use {{domain}}, {{ip}})"
                    />
                    <Input
                      className="w-20"
                      type="number"
                      value={r.ttl}
                      onChange={(e) => updateRecord(i, "ttl", parseInt(e.target.value) || 3600)}
                      placeholder="TTL"
                    />
                    {(r.type === "MX" || r.type === "SRV") && (
                      <Input
                        className="w-20"
                        type="number"
                        value={r.priority ?? ""}
                        onChange={(e) =>
                          updateRecord(i, "priority", e.target.value ? parseInt(e.target.value) : null)
                        }
                        placeholder="Pri"
                      />
                    )}
                    <Button
                      variant="ghost"
                      size="icon"
                      className="h-8 w-8 shrink-0 text-destructive"
                      onClick={() => removeRecord(i)}
                    >
                      <Trash2 className="h-3.5 w-3.5" />
                    </Button>
                  </div>
                ))}
              </div>
            </div>
          </div>

          <DialogFooter>
            <Button variant="outline" onClick={closeEdit} disabled={saving}>
              Cancel
            </Button>
            <Button
              onClick={() => (editTemplate ? updateMutation.mutate() : createMutation.mutate())}
              disabled={!formName.trim() || formRecords.length === 0 || saving}
              className="bg-pink-500 hover:bg-pink-600"
            >
              {saving ? "Saving..." : editTemplate ? "Update Template" : "Create Template"}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* View Records Dialog */}
      <Dialog open={!!viewTemplate} onOpenChange={(v) => !v && setViewTemplate(null)}>
        <DialogContent className="max-w-lg">
          <DialogHeader>
            <DialogTitle>{viewTemplate?.name} — Records</DialogTitle>
            <DialogDescription>{viewTemplate?.description}</DialogDescription>
          </DialogHeader>
          <div className="space-y-1 max-h-[60vh] overflow-y-auto">
            {viewTemplate?.records.map((r, i) => (
              <div key={i} className="flex items-center gap-2 rounded-md border p-2 text-sm">
                <span className={`inline-flex items-center rounded px-1.5 py-0.5 text-xs font-semibold ${typeColors[r.type] ?? "bg-muted"}`}>
                  {r.type}
                </span>
                <span className="font-mono">{r.name}</span>
                <span className="text-muted-foreground">→</span>
                <span className="font-mono truncate flex-1">{r.value}</span>
                <span className="text-xs text-muted-foreground">{r.ttl}s</span>
                {r.priority != null && (
                  <span className="text-xs text-muted-foreground">pri:{r.priority}</span>
                )}
              </div>
            ))}
          </div>
        </DialogContent>
      </Dialog>

      {/* Delete Confirm */}
      <ConfirmDialog
        open={!!deleteTarget}
        onOpenChange={(v) => !v && setDeleteTarget(null)}
        title="Delete Template"
        description={`Delete the template "${deleteTarget?.name}"? This cannot be undone.`}
        confirmText="Delete"
        destructive
        loading={deleteMutation.isPending}
        onConfirm={() => deleteTarget && deleteMutation.mutate(deleteTarget.id)}
      />
    </div>
  );
}
