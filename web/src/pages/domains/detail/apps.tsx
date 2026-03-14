import { useState, useEffect } from "react";
import { useParams } from "react-router-dom";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Switch } from "@/components/ui/switch";
import { Skeleton } from "@/components/ui/skeleton";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import {
  Sheet,
  SheetContent,
  SheetDescription,
  SheetHeader,
  SheetTitle,
} from "@/components/ui/sheet";
import { ConfirmDialog } from "@/components/shared/confirm-dialog";
import { toast } from "sonner";
import {
  getAppCatalog,
  listInstalledApps,
  installApp,
  getInstalledApp,
  uninstallApp,
  updateApp,
  getAppLogs,
  getWPInfo,
  wpMaintenance,
} from "@/api/apps";
import type {
  AppDefinition,
  InstalledApp,
  InstallAppRequest,
  WPPlugin,
  WPTheme,
} from "@/types/app";
import type { AxiosError } from "axios";
import type { APIError } from "@/types/api";
import {
  Package,
  Plus,
  Trash2,
  RefreshCw,
  ExternalLink,
  CheckCircle2,
  XCircle,
  Loader2,
  Clock,
  ArrowUpCircle,
  Settings,
  Globe,
  Database,
  Shield,
  Eye,
  Terminal,
} from "lucide-react";

function AppStatusBadge({ status }: { status: string }) {
  switch (status) {
    case "completed":
      return (
        <Badge className="bg-green-500/10 text-green-500 border-green-500/20">
          <CheckCircle2 className="h-3 w-3 mr-1" />
          Installed
        </Badge>
      );
    case "installing":
      return (
        <Badge className="bg-blue-500/10 text-blue-500 border-blue-500/20">
          <Loader2 className="h-3 w-3 mr-1 animate-spin" />
          Installing
        </Badge>
      );
    case "updating":
      return (
        <Badge className="bg-amber-500/10 text-amber-500 border-amber-500/20">
          <Loader2 className="h-3 w-3 mr-1 animate-spin" />
          Updating
        </Badge>
      );
    case "uninstalling":
      return (
        <Badge className="bg-orange-500/10 text-orange-500 border-orange-500/20">
          <Loader2 className="h-3 w-3 mr-1 animate-spin" />
          Uninstalling
        </Badge>
      );
    case "failed":
      return (
        <Badge className="bg-red-500/10 text-red-500 border-red-500/20">
          <XCircle className="h-3 w-3 mr-1" />
          Failed
        </Badge>
      );
    default:
      return (
        <Badge variant="outline">
          <Clock className="h-3 w-3 mr-1" />
          Pending
        </Badge>
      );
  }
}

const categoryLabels: Record<string, string> = {
  all: "All",
  cms: "CMS",
  ecommerce: "E-Commerce",
  tools: "Tools",
};

const appIcons: Record<string, string> = {
  wordpress: "W",
  joomla: "J",
  drupal: "D",
  prestashop: "P",
  phpmyadmin: "pMA",
};

function AppIcon({ slug, size = "md" }: { slug: string; size?: "sm" | "md" }) {
  const label = appIcons[slug] || slug[0]?.toUpperCase() || "?";
  const sizeClass = size === "sm" ? "h-8 w-8 text-xs" : "h-10 w-10 text-sm";
  return (
    <div
      className={`${sizeClass} rounded-lg bg-pink-500/10 flex items-center justify-center font-bold text-pink-500`}
    >
      {label}
    </div>
  );
}

export function DomainApps() {
  const { id } = useParams<{ id: string }>();
  const domainId = Number(id);
  const queryClient = useQueryClient();

  const [catalogOpen, setCatalogOpen] = useState(false);
  const [wizardOpen, setWizardOpen] = useState(false);
  const [progressOpen, setProgressOpen] = useState(false);
  const [uninstallOpen, setUninstallOpen] = useState(false);
  const [wpSheetOpen, setWpSheetOpen] = useState(false);
  const [selectedApp, setSelectedApp] = useState<AppDefinition | null>(null);
  const [selectedInstalled, setSelectedInstalled] =
    useState<InstalledApp | null>(null);
  const [progressAppId, setProgressAppId] = useState<number | null>(null);
  const [dropDB, setDropDB] = useState(false);
  const [wizardStep, setWizardStep] = useState(1);
  const [installForm, setInstallForm] = useState<InstallAppRequest>({
    app_type: "",
    site_title: "",
    admin_user: "admin",
    admin_pass: "",
    admin_email: "",
    db_name: "",
    db_user: "",
    db_pass: "",
    install_path: "",
  });
  const [catalogFilter, setCatalogFilter] = useState("all");

  // Queries
  const { data: appsData, isLoading } = useQuery({
    queryKey: ["installed-apps", domainId],
    queryFn: () => listInstalledApps(domainId),
    enabled: !!domainId,
  });

  const { data: catalogData } = useQuery({
    queryKey: ["app-catalog"],
    queryFn: getAppCatalog,
    enabled: catalogOpen,
  });

  // Polling for in-progress app
  const { data: progressApp } = useQuery({
    queryKey: ["app", progressAppId],
    queryFn: () => getInstalledApp(progressAppId!),
    enabled: !!progressAppId && progressOpen,
    refetchInterval: (q) => {
      const status = q.state.data?.status;
      return status === "installing" || status === "updating" ? 2000 : false;
    },
  });

  const { data: progressLogs } = useQuery({
    queryKey: ["app-logs", progressAppId],
    queryFn: () => getAppLogs(progressAppId!),
    enabled: !!progressAppId && progressOpen,
    refetchInterval: (q) => {
      const appStatus = progressApp?.status;
      return appStatus === "installing" || appStatus === "updating"
        ? 2000
        : false;
    },
  });

  // When progress app finishes, refresh the list
  useEffect(() => {
    if (
      progressApp &&
      (progressApp.status === "completed" || progressApp.status === "failed")
    ) {
      queryClient.invalidateQueries({ queryKey: ["installed-apps", domainId] });
    }
  }, [progressApp?.status, domainId, queryClient]);

  // Mutations
  const installMutation = useMutation({
    mutationFn: (req: InstallAppRequest) => installApp(domainId, req),
    onSuccess: (data) => {
      toast.success("Installation started");
      setWizardOpen(false);
      setProgressAppId(data.id);
      setProgressOpen(true);
      queryClient.invalidateQueries({ queryKey: ["installed-apps", domainId] });
    },
    onError: (err: AxiosError<APIError>) => {
      toast.error(
        err.response?.data?.error?.message || "Failed to start installation"
      );
    },
  });

  const uninstallMutation = useMutation({
    mutationFn: () => uninstallApp(selectedInstalled!.id, dropDB),
    onSuccess: () => {
      toast.success("App uninstalled");
      setUninstallOpen(false);
      setSelectedInstalled(null);
      queryClient.invalidateQueries({ queryKey: ["installed-apps", domainId] });
    },
    onError: (err: AxiosError<APIError>) => {
      toast.error(err.response?.data?.error?.message || "Uninstall failed");
    },
  });

  const updateMutation = useMutation({
    mutationFn: (appId: number) => updateApp(appId),
    onSuccess: (_, appId) => {
      toast.success("Update started");
      setProgressAppId(appId);
      setProgressOpen(true);
      queryClient.invalidateQueries({ queryKey: ["installed-apps", domainId] });
    },
    onError: (err: AxiosError<APIError>) => {
      toast.error(err.response?.data?.error?.message || "Update failed");
    },
  });

  const apps = appsData?.data ?? [];
  const catalog = catalogData?.data ?? [];
  const filteredCatalog =
    catalogFilter === "all"
      ? catalog
      : catalog.filter((a) => a.category === catalogFilter);
  const categories = [
    "all",
    ...new Set(catalog.map((a) => a.category)),
  ];

  function startInstallWizard(appDef: AppDefinition) {
    setSelectedApp(appDef);
    setCatalogOpen(false);
    setWizardStep(1);
    setInstallForm({
      app_type: appDef.slug,
      site_title: "",
      admin_user: "admin",
      admin_pass: "",
      admin_email: "",
      db_name: "",
      db_user: "",
      db_pass: "",
      install_path: "",
    });
    setWizardOpen(true);
  }

  function submitInstall() {
    installMutation.mutate(installForm);
  }

  if (isLoading) {
    return (
      <div className="space-y-4">
        <Skeleton className="h-10 w-full" />
        <Skeleton className="h-32 w-full" />
      </div>
    );
  }

  return (
    <div className="space-y-4">
      <Card>
        <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-4">
          <div>
            <CardTitle className="text-lg">Applications</CardTitle>
            <CardDescription>
              Install and manage web applications
            </CardDescription>
          </div>
          <Button onClick={() => setCatalogOpen(true)}>
            <Plus className="mr-2 h-4 w-4" />
            Install App
          </Button>
        </CardHeader>
        <CardContent>
          {apps.length === 0 ? (
            <div className="text-center py-12 text-muted-foreground">
              <Package className="h-12 w-12 mx-auto mb-3 opacity-30" />
              <p className="text-sm">No applications installed yet</p>
              <Button
                variant="link"
                size="sm"
                className="mt-2"
                onClick={() => setCatalogOpen(true)}
              >
                Browse catalog
              </Button>
            </div>
          ) : (
            <div className="space-y-2">
              {apps.map((app) => (
                <div
                  key={app.id}
                  className="flex items-center justify-between p-3 rounded-lg border border-border hover:bg-accent/50 transition-colors"
                >
                  <div className="flex items-center gap-3">
                    <AppIcon slug={app.app_type} />
                    <div>
                      <div className="font-medium text-sm">{app.app_name}</div>
                      <div className="text-xs text-muted-foreground flex items-center gap-2">
                        {app.version && <span>v{app.version}</span>}
                        {app.install_path && <span>{app.install_path}</span>}
                      </div>
                    </div>
                  </div>
                  <div className="flex items-center gap-2">
                    <AppStatusBadge status={app.status} />
                    {app.admin_url && (
                      <Button
                        variant="ghost"
                        size="icon"
                        className="h-8 w-8"
                        onClick={() => window.open(app.admin_url!, "_blank")}
                        title="Open admin panel"
                      >
                        <ExternalLink className="h-3.5 w-3.5" />
                      </Button>
                    )}
                    {app.status === "installing" ||
                    app.status === "updating" ? (
                      <Button
                        variant="ghost"
                        size="icon"
                        className="h-8 w-8"
                        onClick={() => {
                          setProgressAppId(app.id);
                          setProgressOpen(true);
                        }}
                        title="View progress"
                      >
                        <Eye className="h-3.5 w-3.5" />
                      </Button>
                    ) : null}
                    {app.app_type === "wordpress" &&
                      app.status === "completed" && (
                        <Button
                          variant="ghost"
                          size="icon"
                          className="h-8 w-8"
                          onClick={() => {
                            setSelectedInstalled(app);
                            setWpSheetOpen(true);
                          }}
                          title="WordPress settings"
                        >
                          <Settings className="h-3.5 w-3.5" />
                        </Button>
                      )}
                    {app.status === "completed" && (
                      <Button
                        variant="ghost"
                        size="icon"
                        className="h-8 w-8"
                        onClick={() => updateMutation.mutate(app.id)}
                        title="Update to latest"
                      >
                        <ArrowUpCircle className="h-3.5 w-3.5" />
                      </Button>
                    )}
                    {(app.status === "completed" ||
                      app.status === "failed") && (
                      <Button
                        variant="ghost"
                        size="icon"
                        className="h-8 w-8 text-red-500 hover:text-red-600"
                        onClick={() => {
                          setSelectedInstalled(app);
                          setDropDB(false);
                          setUninstallOpen(true);
                        }}
                        title="Uninstall"
                      >
                        <Trash2 className="h-3.5 w-3.5" />
                      </Button>
                    )}
                  </div>
                </div>
              ))}
            </div>
          )}
        </CardContent>
      </Card>

      {/* App Catalog Dialog */}
      <Dialog open={catalogOpen} onOpenChange={setCatalogOpen}>
        <DialogContent className="max-w-2xl">
          <DialogHeader>
            <DialogTitle>Install Application</DialogTitle>
            <DialogDescription>
              Choose an application to install on this domain
            </DialogDescription>
          </DialogHeader>
          <div className="flex gap-2 mb-4">
            {categories.map((cat) => (
              <Button
                key={cat}
                variant={catalogFilter === cat ? "default" : "outline"}
                size="sm"
                onClick={() => setCatalogFilter(cat)}
              >
                {categoryLabels[cat] || cat}
              </Button>
            ))}
          </div>
          <div className="grid grid-cols-1 sm:grid-cols-2 gap-3 max-h-[400px] overflow-y-auto">
            {filteredCatalog.map((appDef) => (
              <button
                key={appDef.slug}
                onClick={() => startInstallWizard(appDef)}
                className="flex items-start gap-3 p-3 rounded-lg border border-border hover:bg-accent text-left transition-colors"
              >
                <AppIcon slug={appDef.slug} />
                <div className="min-w-0">
                  <div className="font-medium text-sm">{appDef.name}</div>
                  <div className="text-xs text-muted-foreground line-clamp-2">
                    {appDef.description}
                  </div>
                  <div className="flex gap-2 mt-1.5">
                    <Badge variant="outline" className="text-[10px] px-1.5 py-0">
                      PHP {appDef.min_php}+
                    </Badge>
                    {appDef.needs_db && (
                      <Badge
                        variant="outline"
                        className="text-[10px] px-1.5 py-0"
                      >
                        MySQL
                      </Badge>
                    )}
                  </div>
                </div>
              </button>
            ))}
          </div>
        </DialogContent>
      </Dialog>

      {/* Install Wizard Dialog */}
      <Dialog open={wizardOpen} onOpenChange={setWizardOpen}>
        <DialogContent className="max-w-md">
          <DialogHeader>
            <DialogTitle>
              Install {selectedApp?.name} — Step {wizardStep} of{" "}
              {selectedApp?.has_cli ? 3 : selectedApp?.needs_db ? 2 : 1}
            </DialogTitle>
            <DialogDescription>
              {wizardStep === 1 && "Configure app settings"}
              {wizardStep === 2 && "Configure database"}
              {wizardStep === 3 && "Configure admin account"}
            </DialogDescription>
          </DialogHeader>

          {wizardStep === 1 && (
            <div className="space-y-3">
              <div>
                <Label>Site Title</Label>
                <Input
                  value={installForm.site_title}
                  onChange={(e) =>
                    setInstallForm({ ...installForm, site_title: e.target.value })
                  }
                  placeholder="My Website"
                />
              </div>
              <div>
                <Label>Install Path (optional)</Label>
                <Input
                  value={installForm.install_path}
                  onChange={(e) =>
                    setInstallForm({
                      ...installForm,
                      install_path: e.target.value,
                    })
                  }
                  placeholder="Leave empty for document root"
                />
                <p className="text-xs text-muted-foreground mt-1">
                  Subdirectory relative to document root (e.g., "blog")
                </p>
              </div>
            </div>
          )}

          {wizardStep === 2 && selectedApp?.needs_db && (
            <div className="space-y-3">
              <div>
                <Label>Database Name</Label>
                <Input
                  value={installForm.db_name}
                  onChange={(e) =>
                    setInstallForm({ ...installForm, db_name: e.target.value })
                  }
                  placeholder="Auto-generated if empty"
                />
              </div>
              <div>
                <Label>Database User</Label>
                <Input
                  value={installForm.db_user}
                  onChange={(e) =>
                    setInstallForm({ ...installForm, db_user: e.target.value })
                  }
                  placeholder="Same as database name if empty"
                />
              </div>
              <div>
                <Label>Database Password</Label>
                <Input
                  type="password"
                  value={installForm.db_pass}
                  onChange={(e) =>
                    setInstallForm({ ...installForm, db_pass: e.target.value })
                  }
                  placeholder="Auto-generated if empty"
                />
              </div>
            </div>
          )}

          {wizardStep === 3 && selectedApp?.has_cli && (
            <div className="space-y-3">
              <div>
                <Label>Admin Username</Label>
                <Input
                  value={installForm.admin_user}
                  onChange={(e) =>
                    setInstallForm({
                      ...installForm,
                      admin_user: e.target.value,
                    })
                  }
                />
              </div>
              <div>
                <Label>Admin Password</Label>
                <Input
                  type="password"
                  value={installForm.admin_pass}
                  onChange={(e) =>
                    setInstallForm({
                      ...installForm,
                      admin_pass: e.target.value,
                    })
                  }
                />
              </div>
              <div>
                <Label>Admin Email</Label>
                <Input
                  type="email"
                  value={installForm.admin_email}
                  onChange={(e) =>
                    setInstallForm({
                      ...installForm,
                      admin_email: e.target.value,
                    })
                  }
                  placeholder="admin@example.com"
                />
              </div>
            </div>
          )}

          <DialogFooter className="flex justify-between">
            <div>
              {wizardStep > 1 && (
                <Button
                  variant="outline"
                  onClick={() => setWizardStep(wizardStep - 1)}
                >
                  Back
                </Button>
              )}
            </div>
            <div>
              {(() => {
                const totalSteps = selectedApp?.has_cli
                  ? 3
                  : selectedApp?.needs_db
                    ? 2
                    : 1;
                if (wizardStep < totalSteps) {
                  return (
                    <Button onClick={() => setWizardStep(wizardStep + 1)}>
                      Next
                    </Button>
                  );
                }
                return (
                  <Button
                    onClick={submitInstall}
                    disabled={installMutation.isPending}
                  >
                    {installMutation.isPending && (
                      <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                    )}
                    Install
                  </Button>
                );
              })()}
            </div>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* Install Progress Dialog */}
      <Dialog open={progressOpen} onOpenChange={setProgressOpen}>
        <DialogContent className="max-w-lg">
          <DialogHeader>
            <DialogTitle className="flex items-center gap-2">
              {progressApp?.status === "installing" ||
              progressApp?.status === "updating" ? (
                <Loader2 className="h-4 w-4 animate-spin" />
              ) : progressApp?.status === "completed" ? (
                <CheckCircle2 className="h-4 w-4 text-green-500" />
              ) : progressApp?.status === "failed" ? (
                <XCircle className="h-4 w-4 text-red-500" />
              ) : null}
              {progressApp?.status === "installing"
                ? "Installing..."
                : progressApp?.status === "updating"
                  ? "Updating..."
                  : progressApp?.status === "completed"
                    ? "Completed!"
                    : progressApp?.status === "failed"
                      ? "Failed"
                      : "Processing..."}
            </DialogTitle>
          </DialogHeader>
          <div className="bg-muted rounded-lg p-3 max-h-[300px] overflow-y-auto font-mono text-xs whitespace-pre-wrap">
            {progressLogs?.log || "Waiting for progress..."}
            {progressApp?.error_message && (
              <div className="text-red-500 mt-2">
                Error: {progressApp.error_message}
              </div>
            )}
          </div>
          <DialogFooter>
            <Button variant="outline" onClick={() => setProgressOpen(false)}>
              {progressApp?.status === "completed" ||
              progressApp?.status === "failed"
                ? "Close"
                : "Run in Background"}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* Uninstall Confirm Dialog */}
      <ConfirmDialog
        open={uninstallOpen}
        onOpenChange={setUninstallOpen}
        title={`Uninstall ${selectedInstalled?.app_name}?`}
        description={
          <div className="space-y-3">
            <p>
              This will delete all application files at{" "}
              <code className="text-xs bg-muted px-1 py-0.5 rounded">
                {selectedInstalled?.install_path}
              </code>
            </p>
            {selectedInstalled?.db_name && (
              <div className="flex items-center gap-2">
                <Switch
                  checked={dropDB}
                  onCheckedChange={setDropDB}
                  id="drop-db"
                />
                <Label htmlFor="drop-db" className="text-sm">
                  Also drop database{" "}
                  <code className="text-xs bg-muted px-1 py-0.5 rounded">
                    {selectedInstalled.db_name}
                  </code>
                </Label>
              </div>
            )}
          </div>
        }
        confirmText="Uninstall"
        destructive
        loading={uninstallMutation.isPending}
        onConfirm={() => uninstallMutation.mutate()}
      />

      {/* WordPress Manage Sheet */}
      {selectedInstalled?.app_type === "wordpress" && (
        <WPManageSheet
          app={selectedInstalled}
          open={wpSheetOpen}
          onOpenChange={setWpSheetOpen}
        />
      )}
    </div>
  );
}

function WPManageSheet({
  app,
  open,
  onOpenChange,
}: {
  app: InstalledApp;
  open: boolean;
  onOpenChange: (open: boolean) => void;
}) {
  const { data: wpInfo, isLoading } = useQuery({
    queryKey: ["wp-info", app.id],
    queryFn: () => getWPInfo(app.id),
    enabled: open,
  });

  const maintenanceMutation = useMutation({
    mutationFn: (enable: boolean) => wpMaintenance(app.id, enable),
    onSuccess: () => toast.success("Maintenance mode toggled"),
    onError: () => toast.error("Failed to toggle maintenance mode"),
  });

  let plugins: WPPlugin[] = [];
  let themes: WPTheme[] = [];

  if (wpInfo) {
    try {
      plugins = JSON.parse(wpInfo.plugins_json);
    } catch {
      plugins = [];
    }
    try {
      themes = JSON.parse(wpInfo.themes_json);
    } catch {
      themes = [];
    }
  }

  return (
    <Sheet open={open} onOpenChange={onOpenChange}>
      <SheetContent className="w-full sm:max-w-lg overflow-y-auto">
        <SheetHeader>
          <SheetTitle>WordPress Manager</SheetTitle>
          <SheetDescription>{app.install_path}</SheetDescription>
        </SheetHeader>

        <div className="space-y-6 mt-6">
          {isLoading ? (
            <div className="space-y-3">
              <Skeleton className="h-8 w-32" />
              <Skeleton className="h-24 w-full" />
            </div>
          ) : (
            <>
              {/* Version */}
              <div className="flex items-center justify-between">
                <div>
                  <div className="text-sm font-medium">WordPress Version</div>
                  <div className="text-xs text-muted-foreground">
                    {wpInfo?.version || app.version || "Unknown"}
                  </div>
                </div>
                <Badge variant="outline">
                  <Globe className="h-3 w-3 mr-1" />
                  {wpInfo?.version || app.version}
                </Badge>
              </div>

              {/* Maintenance Mode */}
              <div className="flex items-center justify-between p-3 rounded-lg border border-border">
                <div>
                  <div className="text-sm font-medium">Maintenance Mode</div>
                  <div className="text-xs text-muted-foreground">
                    Show maintenance page to visitors
                  </div>
                </div>
                <div className="flex gap-2">
                  <Button
                    variant="outline"
                    size="sm"
                    onClick={() => maintenanceMutation.mutate(true)}
                    disabled={maintenanceMutation.isPending}
                  >
                    Enable
                  </Button>
                  <Button
                    variant="outline"
                    size="sm"
                    onClick={() => maintenanceMutation.mutate(false)}
                    disabled={maintenanceMutation.isPending}
                  >
                    Disable
                  </Button>
                </div>
              </div>

              {/* Plugins */}
              <div>
                <h4 className="text-sm font-medium mb-2">
                  Plugins ({plugins.length})
                </h4>
                {plugins.length === 0 ? (
                  <p className="text-xs text-muted-foreground">
                    No plugins found
                  </p>
                ) : (
                  <div className="space-y-1">
                    {plugins.map((p) => (
                      <div
                        key={p.name}
                        className="flex items-center justify-between py-1.5 px-2 rounded text-xs hover:bg-accent/50"
                      >
                        <div>
                          <span className="font-medium">{p.name}</span>
                          <span className="text-muted-foreground ml-2">
                            v{p.version}
                          </span>
                        </div>
                        <div className="flex items-center gap-2">
                          <Badge
                            variant="outline"
                            className={
                              p.status === "active"
                                ? "text-green-500 border-green-500/20"
                                : "text-muted-foreground"
                            }
                          >
                            {p.status}
                          </Badge>
                          {p.update && p.update !== "none" && (
                            <Badge className="bg-amber-500/10 text-amber-500 border-amber-500/20">
                              {p.update}
                            </Badge>
                          )}
                        </div>
                      </div>
                    ))}
                  </div>
                )}
              </div>

              {/* Themes */}
              <div>
                <h4 className="text-sm font-medium mb-2">
                  Themes ({themes.length})
                </h4>
                {themes.length === 0 ? (
                  <p className="text-xs text-muted-foreground">
                    No themes found
                  </p>
                ) : (
                  <div className="space-y-1">
                    {themes.map((t) => (
                      <div
                        key={t.name}
                        className="flex items-center justify-between py-1.5 px-2 rounded text-xs hover:bg-accent/50"
                      >
                        <div>
                          <span className="font-medium">{t.name}</span>
                          <span className="text-muted-foreground ml-2">
                            v{t.version}
                          </span>
                        </div>
                        <div className="flex items-center gap-2">
                          <Badge
                            variant="outline"
                            className={
                              t.status === "active"
                                ? "text-green-500 border-green-500/20"
                                : "text-muted-foreground"
                            }
                          >
                            {t.status}
                          </Badge>
                          {t.update && t.update !== "none" && (
                            <Badge className="bg-amber-500/10 text-amber-500 border-amber-500/20">
                              {t.update}
                            </Badge>
                          )}
                        </div>
                      </div>
                    ))}
                  </div>
                )}
              </div>

              {/* Quick links */}
              {app.admin_url && (
                <Button
                  variant="outline"
                  size="sm"
                  className="w-full"
                  onClick={() => window.open(app.admin_url!, "_blank")}
                >
                  <ExternalLink className="mr-2 h-3.5 w-3.5" />
                  Open WP Admin
                </Button>
              )}
            </>
          )}
        </div>
      </SheetContent>
    </Sheet>
  );
}
