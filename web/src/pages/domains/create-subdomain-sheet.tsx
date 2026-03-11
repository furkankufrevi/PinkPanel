import { useState } from "react";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import {
  Sheet,
  SheetContent,
  SheetDescription,
  SheetFooter,
  SheetHeader,
  SheetTitle,
} from "@/components/ui/sheet";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { toast } from "sonner";
import { createDomain } from "@/api/domains";
import type { Domain } from "@/types/domain";
import type { AxiosError } from "axios";
import type { APIError } from "@/types/api";

const phpVersions = ["8.3", "8.2", "8.1", "8.0", "7.4"];

interface CreateSubdomainSheetProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  parentDomains: Domain[];
}

export function CreateSubdomainSheet({ open, onOpenChange, parentDomains }: CreateSubdomainSheetProps) {
  const [name, setName] = useState("");
  const [parentId, setParentId] = useState<string>("");
  const [phpVersion, setPhpVersion] = useState("8.3");
  const queryClient = useQueryClient();

  const selectedParent = parentDomains.find((d) => d.id === Number(parentId));

  const mutation = useMutation({
    mutationFn: () =>
      createDomain({
        name,
        php_version: phpVersion,
        create_www: false,
        parent_id: Number(parentId),
      }),
    onSuccess: () => {
      toast.success(`Subdomain ${name}.${selectedParent?.name} created`);
      queryClient.invalidateQueries({ queryKey: ["domains"] });
      onOpenChange(false);
      resetForm();
    },
    onError: (error: AxiosError<APIError>) => {
      const message = error.response?.data?.error?.message ?? "Failed to create subdomain";
      toast.error(message);
    },
  });

  function resetForm() {
    setName("");
    setParentId("");
    setPhpVersion("8.3");
  }

  function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    if (!name.trim()) {
      toast.error("Subdomain name is required");
      return;
    }
    if (!parentId) {
      toast.error("Please select a parent domain");
      return;
    }
    mutation.mutate();
  }

  return (
    <Sheet open={open} onOpenChange={onOpenChange}>
      <SheetContent>
        <SheetHeader>
          <SheetTitle>Add Subdomain</SheetTitle>
          <SheetDescription>
            Create a subdomain under an existing domain
          </SheetDescription>
        </SheetHeader>
        <form onSubmit={handleSubmit} className="space-y-6 py-6">
          <div className="space-y-2">
            <Label>Parent Domain</Label>
            <Select value={parentId} onValueChange={(v) => v && setParentId(v)}>
              <SelectTrigger>
                <SelectValue placeholder="Select domain..." />
              </SelectTrigger>
              <SelectContent>
                {parentDomains.map((d) => (
                  <SelectItem key={d.id} value={String(d.id)}>
                    {d.name}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>
          <div className="space-y-2">
            <Label htmlFor="sub-name">Subdomain Name</Label>
            <div className="flex items-center gap-2">
              <Input
                id="sub-name"
                value={name}
                onChange={(e) => setName(e.target.value)}
                placeholder="blog"
                autoFocus
              />
              {selectedParent && (
                <span className="text-sm text-muted-foreground whitespace-nowrap">
                  .{selectedParent.name}
                </span>
              )}
            </div>
            <p className="text-xs text-muted-foreground">
              Alphanumeric characters and hyphens only
            </p>
          </div>
          <div className="space-y-2">
            <Label>PHP Version</Label>
            <Select value={phpVersion} onValueChange={(v) => v && setPhpVersion(v)}>
              <SelectTrigger>
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                {phpVersions.map((v) => (
                  <SelectItem key={v} value={v}>
                    PHP {v}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>
          <SheetFooter>
            <Button
              type="submit"
              disabled={mutation.isPending || !parentId}
              className="bg-pink-500 hover:bg-pink-600"
            >
              {mutation.isPending ? "Creating..." : "Create Subdomain"}
            </Button>
          </SheetFooter>
        </form>
      </SheetContent>
    </Sheet>
  );
}
