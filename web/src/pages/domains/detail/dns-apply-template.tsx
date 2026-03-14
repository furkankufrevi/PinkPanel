import { useState } from "react";
import { useQuery, useMutation } from "@tanstack/react-query";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Button } from "@/components/ui/button";
import { Label } from "@/components/ui/label";
import {
  Select,
  SelectContent,
  SelectGroup,
  SelectItem,
  SelectLabel,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { listDNSTemplates, applyDNSTemplate } from "@/api/dns-templates";
import { toast } from "sonner";
import type { DNSTemplate } from "@/types/dns-template";
import type { AxiosError } from "axios";
import type { APIError } from "@/types/api";

interface ApplyTemplateDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  domainId: number;
  onApplied: () => void;
}

const categoryLabels: Record<string, string> = {
  hosting: "Hosting",
  email: "Email Providers",
  custom: "Custom Templates",
};

export function ApplyTemplateDialog({
  open,
  onOpenChange,
  domainId,
  onApplied,
}: ApplyTemplateDialogProps) {
  const [selectedId, setSelectedId] = useState<string>("");
  const [mode, setMode] = useState<string>("merge");

  const { data: templatesData } = useQuery({
    queryKey: ["dns-templates"],
    queryFn: listDNSTemplates,
    enabled: open,
  });

  const templates = templatesData?.data ?? [];

  // Group templates by category
  const grouped = templates.reduce<Record<string, DNSTemplate[]>>((acc, t) => {
    const cat = t.category || "custom";
    if (!acc[cat]) acc[cat] = [];
    acc[cat].push(t);
    return acc;
  }, {});

  const selectedTemplate = templates.find((t) => String(t.id) === selectedId);

  const applyMutation = useMutation({
    mutationFn: () =>
      applyDNSTemplate(domainId, {
        template_id: Number(selectedId),
        mode: mode as "merge" | "replace",
      }),
    onSuccess: (result) => {
      toast.success(result.message || "Template applied");
      onApplied();
      handleClose();
    },
    onError: (err: AxiosError<APIError>) => {
      toast.error(err.response?.data?.error?.message ?? "Failed to apply template");
    },
  });

  function handleClose() {
    setSelectedId("");
    setMode("merge");
    onOpenChange(false);
  }

  return (
    <Dialog open={open} onOpenChange={(v) => !v && handleClose()}>
      <DialogContent className="max-w-lg">
        <DialogHeader>
          <DialogTitle>Apply DNS Template</DialogTitle>
          <DialogDescription>
            Select a template to apply its DNS records to this domain. Variables
            like {"{{domain}}"} and {"{{ip}}"} will be automatically resolved.
          </DialogDescription>
        </DialogHeader>

        <div className="space-y-4">
          <div className="space-y-2">
            <Label>Template</Label>
            <Select value={selectedId} onValueChange={(v) => v && setSelectedId(v)}>
              <SelectTrigger>
                <SelectValue placeholder="Choose a template..." />
              </SelectTrigger>
              <SelectContent>
                {Object.entries(grouped).map(([category, items]) => (
                  <SelectGroup key={category}>
                    <SelectLabel>{categoryLabels[category] ?? category}</SelectLabel>
                    {items.map((t) => (
                      <SelectItem key={t.id} value={String(t.id)}>
                        {t.name}
                        {t.is_preset && (
                          <span className="ml-2 text-xs text-muted-foreground">(preset)</span>
                        )}
                      </SelectItem>
                    ))}
                  </SelectGroup>
                ))}
              </SelectContent>
            </Select>
          </div>

          {selectedTemplate && (
            <div className="rounded-md border p-3 text-sm space-y-2">
              <p className="text-muted-foreground">{selectedTemplate.description}</p>
              <p className="font-medium">
                {selectedTemplate.records.length} record{selectedTemplate.records.length !== 1 ? "s" : ""}
              </p>
              <div className="flex flex-wrap gap-1">
                {selectedTemplate.records.map((r, i) => (
                  <span
                    key={i}
                    className="inline-flex items-center rounded px-1.5 py-0.5 text-xs font-mono bg-muted"
                  >
                    {r.type} {r.name}
                  </span>
                ))}
              </div>
            </div>
          )}

          <div className="space-y-2">
            <Label>Apply Mode</Label>
            <Select value={mode} onValueChange={(v) => v && setMode(v)}>
              <SelectTrigger>
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="merge">Merge — Add records, keep existing</SelectItem>
                <SelectItem value="replace">Replace — Delete all existing records first</SelectItem>
              </SelectContent>
            </Select>
            {mode === "replace" && (
              <p className="text-xs text-destructive">
                Warning: This will delete all existing DNS records before applying the template.
              </p>
            )}
          </div>
        </div>

        <DialogFooter>
          <Button variant="outline" onClick={handleClose} disabled={applyMutation.isPending}>
            Cancel
          </Button>
          <Button
            onClick={() => applyMutation.mutate()}
            disabled={!selectedId || applyMutation.isPending}
            className="bg-pink-500 hover:bg-pink-600"
          >
            {applyMutation.isPending ? "Applying..." : "Apply Template"}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
