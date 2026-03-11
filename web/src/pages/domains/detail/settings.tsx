import { useState, useEffect } from "react";
import { useOutletContext, useNavigate, useParams } from "react-router-dom";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
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
import { Separator } from "@/components/ui/separator";
import { ConfirmDialog } from "@/components/shared/confirm-dialog";
import { toast } from "sonner";
import { updateDomain, suspendDomain, activateDomain, deleteDomain } from "@/api/domains";
import type { Domain } from "@/types/domain";
import type { AxiosError } from "axios";
import type { APIError } from "@/types/api";

const phpVersions = ["8.3", "8.2", "8.1", "8.0", "7.4"];

interface DomainContext {
  domain: Domain | undefined;
  isLoading: boolean;
}

export function DomainSettings() {
  const { domain } = useOutletContext<DomainContext>();
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const queryClient = useQueryClient();

  const [documentRoot, setDocumentRoot] = useState("");
  const [phpVersion, setPhpVersion] = useState("8.3");
  const [separateDNS, setSeparateDNS] = useState(false);
  const [deleteOpen, setDeleteOpen] = useState(false);

  const isSubdomain = !!domain?.parent_id;

  useEffect(() => {
    if (domain) {
      setDocumentRoot(domain.document_root);
      setPhpVersion(domain.php_version);
      setSeparateDNS(domain.separate_dns);
    }
  }, [domain]);

  const updateMutation = useMutation({
    mutationFn: () =>
      updateDomain(Number(id), {
        document_root: documentRoot,
        php_version: phpVersion,
        ...(isSubdomain ? { separate_dns: separateDNS } : {}),
      }),
    onSuccess: () => {
      toast.success("Domain settings updated");
      queryClient.invalidateQueries({ queryKey: ["domain", Number(id)] });
    },
    onError: (err: AxiosError<APIError>) => {
      toast.error(err.response?.data?.error?.message ?? "Failed to update domain");
    },
  });

  const suspendMutation = useMutation({
    mutationFn: () => suspendDomain(Number(id)),
    onSuccess: () => {
      toast.success("Domain suspended");
      queryClient.invalidateQueries({ queryKey: ["domain", Number(id)] });
    },
    onError: (err: AxiosError<APIError>) => {
      toast.error(err.response?.data?.error?.message ?? "Suspend failed");
    },
  });

  const activateMutation = useMutation({
    mutationFn: () => activateDomain(Number(id)),
    onSuccess: () => {
      toast.success("Domain activated");
      queryClient.invalidateQueries({ queryKey: ["domain", Number(id)] });
    },
    onError: (err: AxiosError<APIError>) => {
      toast.error(err.response?.data?.error?.message ?? "Activate failed");
    },
  });

  const deleteMutation = useMutation({
    mutationFn: () => deleteDomain(Number(id), true),
    onSuccess: () => {
      toast.success("Domain deleted");
      navigate("/domains");
    },
    onError: (err: AxiosError<APIError>) => {
      toast.error(err.response?.data?.error?.message ?? "Delete failed");
    },
  });

  if (!domain) return null;

  return (
    <div className="space-y-6 max-w-2xl">
      <Card>
        <CardHeader>
          <CardTitle>General</CardTitle>
          <CardDescription>Domain configuration settings</CardDescription>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="space-y-2">
            <Label>Document Root</Label>
            <Input
              value={documentRoot}
              onChange={(e) => setDocumentRoot(e.target.value)}
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
          {isSubdomain && (
            <>
              <Separator />
              <div className="flex items-center justify-between">
                <div>
                  <Label>Separate DNS Zone</Label>
                  <p className="text-sm text-muted-foreground">
                    Create an independent DNS zone for this subdomain instead of using the parent's zone
                  </p>
                </div>
                <Switch
                  checked={separateDNS}
                  onCheckedChange={setSeparateDNS}
                />
              </div>
            </>
          )}
          <Button
            onClick={() => updateMutation.mutate()}
            disabled={updateMutation.isPending}
            className="bg-pink-500 hover:bg-pink-600"
          >
            {updateMutation.isPending ? "Saving..." : "Save Changes"}
          </Button>
        </CardContent>
      </Card>

      <Separator />

      <Card className="border-destructive/50">
        <CardHeader>
          <CardTitle className="text-destructive">Danger Zone</CardTitle>
          <CardDescription>Irreversible actions for this {isSubdomain ? "subdomain" : "domain"}</CardDescription>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="flex items-center justify-between">
            <div>
              <p className="font-medium">
                {domain.status === "active" ? "Suspend Domain" : "Activate Domain"}
              </p>
              <p className="text-sm text-muted-foreground">
                {domain.status === "active"
                  ? "Temporarily disable this domain"
                  : "Re-enable this domain"}
              </p>
            </div>
            {domain.status === "active" ? (
              <Button
                variant="outline"
                onClick={() => suspendMutation.mutate()}
                disabled={suspendMutation.isPending}
              >
                {suspendMutation.isPending ? "Suspending..." : "Suspend"}
              </Button>
            ) : (
              <Button
                variant="outline"
                onClick={() => activateMutation.mutate()}
                disabled={activateMutation.isPending}
              >
                {activateMutation.isPending ? "Activating..." : "Activate"}
              </Button>
            )}
          </div>

          <Separator />

          <div className="flex items-center justify-between">
            <div>
              <p className="font-medium">Delete {isSubdomain ? "Subdomain" : "Domain"}</p>
              <p className="text-sm text-muted-foreground">
                Permanently delete this {isSubdomain ? "subdomain" : "domain"} and all its data
              </p>
            </div>
            <Button variant="destructive" onClick={() => setDeleteOpen(true)}>
              Delete
            </Button>
          </div>
        </CardContent>
      </Card>

      <ConfirmDialog
        open={deleteOpen}
        onOpenChange={setDeleteOpen}
        title={`Delete ${isSubdomain ? "Subdomain" : "Domain"}`}
        description={`This will permanently delete ${domain.name} and all associated files, databases, and configuration.`}
        typeToConfirm={domain.name}
        confirmText={`Delete ${isSubdomain ? "Subdomain" : "Domain"}`}
        destructive
        loading={deleteMutation.isPending}
        onConfirm={() => deleteMutation.mutate()}
      />
    </div>
  );
}
