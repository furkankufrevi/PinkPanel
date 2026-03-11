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
import { Switch } from "@/components/ui/switch";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { toast } from "sonner";
import { createDomain } from "@/api/domains";
import type { AxiosError } from "axios";
import type { APIError } from "@/types/api";

const phpVersions = ["8.3", "8.2", "8.1", "8.0", "7.4"];

interface CreateDomainSheetProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
}

export function CreateDomainSheet({ open, onOpenChange }: CreateDomainSheetProps) {
  const [name, setName] = useState("");
  const [phpVersion, setPhpVersion] = useState("8.3");
  const [createWww, setCreateWww] = useState(true);
  const queryClient = useQueryClient();

  const mutation = useMutation({
    mutationFn: () =>
      createDomain({ name, php_version: phpVersion, create_www: createWww }),
    onSuccess: () => {
      toast.success(`Domain ${name} created`);
      queryClient.invalidateQueries({ queryKey: ["domains"] });
      onOpenChange(false);
      resetForm();
    },
    onError: (error: AxiosError<APIError>) => {
      const message = error.response?.data?.error?.message ?? "Failed to create domain";
      toast.error(message);
    },
  });

  function resetForm() {
    setName("");
    setPhpVersion("8.3");
    setCreateWww(true);
  }

  function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    if (!name.trim()) {
      toast.error("Domain name is required");
      return;
    }
    mutation.mutate();
  }

  return (
    <Sheet open={open} onOpenChange={onOpenChange}>
      <SheetContent>
        <SheetHeader>
          <SheetTitle>Add Domain</SheetTitle>
          <SheetDescription>
            Create a new domain with NGINX configuration
          </SheetDescription>
        </SheetHeader>
        <form onSubmit={handleSubmit} className="space-y-6 py-6">
          <div className="space-y-2">
            <Label htmlFor="domain-name">Domain Name</Label>
            <Input
              id="domain-name"
              value={name}
              onChange={(e) => setName(e.target.value)}
              placeholder="example.com"
              autoFocus
            />
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
          <div className="flex items-center justify-between">
            <div>
              <Label>Create www subdomain</Label>
              <p className="text-sm text-muted-foreground">
                Also serve www.{name || "example.com"}
              </p>
            </div>
            <Switch checked={createWww} onCheckedChange={setCreateWww} />
          </div>
          <SheetFooter>
            <Button
              type="submit"
              disabled={mutation.isPending}
              className="bg-pink-500 hover:bg-pink-600"
            >
              {mutation.isPending ? "Creating..." : "Create Domain"}
            </Button>
          </SheetFooter>
        </form>
      </SheetContent>
    </Sheet>
  );
}
