import { useState, useEffect } from "react";
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
import { createDNSRecord, updateDNSRecord } from "@/api/dns";
import type { DNSRecord } from "@/types/dns";
import type { AxiosError } from "axios";
import type { APIError } from "@/types/api";

const recordTypes = ["A", "AAAA", "CNAME", "MX", "TXT", "NS", "SRV", "CAA"];

interface DNSRecordSheetProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  domainId: number;
  record: DNSRecord | null;
}

export function DNSRecordSheet({
  open,
  onOpenChange,
  domainId,
  record,
}: DNSRecordSheetProps) {
  const isEdit = !!record;
  const queryClient = useQueryClient();

  const [type, setType] = useState("A");
  const [name, setName] = useState("@");
  const [value, setValue] = useState("");
  const [ttl, setTtl] = useState("3600");
  const [priority, setPriority] = useState("");

  useEffect(() => {
    if (record) {
      setType(record.type);
      setName(record.name);
      setValue(record.value);
      setTtl(String(record.ttl));
      setPriority(record.priority != null ? String(record.priority) : "");
    } else {
      setType("A");
      setName("@");
      setValue("");
      setTtl("3600");
      setPriority("");
    }
  }, [record, open]);

  const createMutation = useMutation({
    mutationFn: () =>
      createDNSRecord(domainId, {
        type,
        name,
        value,
        ttl: Number(ttl),
        priority: priority ? Number(priority) : null,
      }),
    onSuccess: () => {
      toast.success("DNS record created");
      queryClient.invalidateQueries({ queryKey: ["dns-records", domainId] });
      onOpenChange(false);
    },
    onError: (err: AxiosError<APIError>) => {
      toast.error(err.response?.data?.error?.message ?? "Failed to create record");
    },
  });

  const updateMutation = useMutation({
    mutationFn: () =>
      updateDNSRecord(record!.id, {
        type,
        name,
        value,
        ttl: Number(ttl),
        priority: priority ? Number(priority) : null,
      }),
    onSuccess: () => {
      toast.success("DNS record updated");
      queryClient.invalidateQueries({ queryKey: ["dns-records", domainId] });
      onOpenChange(false);
    },
    onError: (err: AxiosError<APIError>) => {
      toast.error(err.response?.data?.error?.message ?? "Failed to update record");
    },
  });

  const isPending = createMutation.isPending || updateMutation.isPending;
  const showPriority = type === "MX" || type === "SRV";

  function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    if (!value.trim()) {
      toast.error("Value is required");
      return;
    }
    if (isEdit) {
      updateMutation.mutate();
    } else {
      createMutation.mutate();
    }
  }

  return (
    <Sheet open={open} onOpenChange={onOpenChange}>
      <SheetContent>
        <SheetHeader>
          <SheetTitle>{isEdit ? "Edit DNS Record" : "Add DNS Record"}</SheetTitle>
          <SheetDescription>
            {isEdit ? "Modify the DNS record" : "Add a new DNS record to this domain"}
          </SheetDescription>
        </SheetHeader>
        <form onSubmit={handleSubmit} className="space-y-4 py-6">
          <div className="space-y-2">
            <Label>Type</Label>
            <Select value={type} onValueChange={(v) => v && setType(v)}>
              <SelectTrigger>
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                {recordTypes.map((t) => (
                  <SelectItem key={t} value={t}>
                    {t}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>
          <div className="space-y-2">
            <Label>Name</Label>
            <Input
              value={name}
              onChange={(e) => setName(e.target.value)}
              placeholder="@ or subdomain"
            />
            <p className="text-xs text-muted-foreground">
              Use @ for the root domain, or enter a subdomain name
            </p>
          </div>
          <div className="space-y-2">
            <Label>Value</Label>
            <Input
              value={value}
              onChange={(e) => setValue(e.target.value)}
              placeholder={type === "A" ? "1.2.3.4" : type === "CNAME" ? "target.example.com" : "value"}
            />
          </div>
          <div className="grid grid-cols-2 gap-4">
            <div className="space-y-2">
              <Label>TTL (seconds)</Label>
              <Input
                type="number"
                value={ttl}
                onChange={(e) => setTtl(e.target.value)}
                min={60}
                max={86400}
              />
            </div>
            {showPriority && (
              <div className="space-y-2">
                <Label>Priority</Label>
                <Input
                  type="number"
                  value={priority}
                  onChange={(e) => setPriority(e.target.value)}
                  placeholder="10"
                  min={0}
                  max={65535}
                />
              </div>
            )}
          </div>
          <SheetFooter>
            <Button
              type="submit"
              disabled={isPending}
              className="bg-pink-500 hover:bg-pink-600"
            >
              {isPending
                ? "Saving..."
                : isEdit
                  ? "Update Record"
                  : "Add Record"}
            </Button>
          </SheetFooter>
        </form>
      </SheetContent>
    </Sheet>
  );
}
