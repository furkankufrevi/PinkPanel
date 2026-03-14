import { useState } from "react";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { saveAsTemplate } from "@/api/dns-templates";
import { toast } from "sonner";
import type { AxiosError } from "axios";
import type { APIError } from "@/types/api";

interface SaveTemplateDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  domainId: number;
}

export function SaveTemplateDialog({
  open,
  onOpenChange,
  domainId,
}: SaveTemplateDialogProps) {
  const queryClient = useQueryClient();
  const [name, setName] = useState("");
  const [description, setDescription] = useState("");

  const saveMutation = useMutation({
    mutationFn: () => saveAsTemplate(domainId, { name, description }),
    onSuccess: (result) => {
      toast.success(`Template "${result.data.name}" saved`);
      queryClient.invalidateQueries({ queryKey: ["dns-templates"] });
      handleClose();
    },
    onError: (err: AxiosError<APIError>) => {
      toast.error(err.response?.data?.error?.message ?? "Failed to save template");
    },
  });

  function handleClose() {
    setName("");
    setDescription("");
    onOpenChange(false);
  }

  return (
    <Dialog open={open} onOpenChange={(v) => !v && handleClose()}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Save as DNS Template</DialogTitle>
          <DialogDescription>
            Save this domain's current DNS records as a reusable template.
          </DialogDescription>
        </DialogHeader>

        <div className="space-y-4">
          <div className="space-y-2">
            <Label htmlFor="template-name">Template Name</Label>
            <Input
              id="template-name"
              value={name}
              onChange={(e) => setName(e.target.value)}
              placeholder="e.g., My Standard Setup"
              autoFocus
            />
          </div>
          <div className="space-y-2">
            <Label htmlFor="template-desc">Description (optional)</Label>
            <Input
              id="template-desc"
              value={description}
              onChange={(e) => setDescription(e.target.value)}
              placeholder="Brief description of what this template includes"
            />
          </div>
        </div>

        <DialogFooter>
          <Button variant="outline" onClick={handleClose} disabled={saveMutation.isPending}>
            Cancel
          </Button>
          <Button
            onClick={() => saveMutation.mutate()}
            disabled={!name.trim() || saveMutation.isPending}
            className="bg-pink-500 hover:bg-pink-600"
          >
            {saveMutation.isPending ? "Saving..." : "Save Template"}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
