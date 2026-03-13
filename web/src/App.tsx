import { BrowserRouter, Routes, Route, Navigate } from "react-router-dom";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { Toaster } from "@/components/ui/sonner";
import { TooltipProvider } from "@/components/ui/tooltip";
import { ErrorBoundary } from "@/components/shared/error-boundary";
import { ProtectedRoute } from "@/components/shared/protected-route";
import { AppShell } from "@/components/layout/app-shell";
import { useUIStore } from "@/stores/ui";
import { useEffect, lazy, Suspense } from "react";
import { Skeleton } from "@/components/ui/skeleton";

// Lazy-loaded pages for code-splitting
const LoginPage = lazy(() => import("@/pages/login").then((m) => ({ default: m.LoginPage })));
const DashboardPage = lazy(() => import("@/pages/dashboard").then((m) => ({ default: m.DashboardPage })));
const DomainsPage = lazy(() => import("@/pages/domains").then((m) => ({ default: m.DomainsPage })));
const DomainDetailLayout = lazy(() => import("@/pages/domains/detail/layout").then((m) => ({ default: m.DomainDetailLayout })));
const DomainOverview = lazy(() => import("@/pages/domains/detail/overview").then((m) => ({ default: m.DomainOverview })));
const DomainSettings = lazy(() => import("@/pages/domains/detail/settings").then((m) => ({ default: m.DomainSettings })));
const DomainDNS = lazy(() => import("@/pages/domains/detail/dns").then((m) => ({ default: m.DomainDNS })));
const DomainPHP = lazy(() => import("@/pages/domains/detail/php").then((m) => ({ default: m.DomainPHP })));
const DomainSSL = lazy(() => import("@/pages/domains/detail/ssl").then((m) => ({ default: m.DomainSSL })));
const DomainFiles = lazy(() => import("@/pages/domains/detail/files").then((m) => ({ default: m.DomainFiles })));
const DomainDatabases = lazy(() => import("@/pages/domains/detail/databases").then((m) => ({ default: m.DomainDatabases })));
const DomainFTP = lazy(() => import("@/pages/domains/detail/ftp").then((m) => ({ default: m.DomainFTP })));
const DomainEmail = lazy(() => import("@/pages/domains/detail/email").then((m) => ({ default: m.DomainEmail })));
const DomainGit = lazy(() => import("@/pages/domains/detail/git").then((m) => ({ default: m.DomainGit })));
const DomainBackups = lazy(() => import("@/pages/domains/detail/backups").then((m) => ({ default: m.DomainBackups })));
const DomainLogs = lazy(() => import("@/pages/domains/detail/logs").then((m) => ({ default: m.DomainLogs })));
const FilesPage = lazy(() => import("@/pages/files").then((m) => ({ default: m.FilesPage })));
const DatabasesPage = lazy(() => import("@/pages/databases").then((m) => ({ default: m.DatabasesPage })));
const BackupsPage = lazy(() => import("@/pages/backups").then((m) => ({ default: m.BackupsPage })));
const LogsPage = lazy(() => import("@/pages/logs").then((m) => ({ default: m.LogsPage })));
const SettingsPage = lazy(() => import("@/pages/settings").then((m) => ({ default: m.SettingsPage })));
const UsersPage = lazy(() => import("@/pages/users").then((m) => ({ default: m.UsersPage })));
const SecurityPage = lazy(() => import("@/pages/security").then((m) => ({ default: m.SecurityPage })));
const UpdatesPage = lazy(() => import("@/pages/updates").then((m) => ({ default: m.UpdatesPage })));

const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      staleTime: 30000,
      retry: (failureCount, error: any) => {
        if (error?.response?.status < 500) return false;
        return failureCount < 3;
      },
    },
  },
});

function PageLoader() {
  return <Skeleton className="h-64 w-full" />;
}

function ThemeInitializer() {
  const theme = useUIStore((s) => s.theme);

  useEffect(() => {
    document.documentElement.classList.toggle("dark", theme === "dark");
  }, [theme]);

  return null;
}

function App() {
  return (
    <ErrorBoundary>
      <QueryClientProvider client={queryClient}>
        <TooltipProvider>
          <BrowserRouter>
            <ThemeInitializer />
            <Suspense fallback={<PageLoader />}>
              <Routes>
                <Route path="/login" element={<LoginPage />} />
                <Route
                  element={
                    <ProtectedRoute>
                      <AppShell />
                    </ProtectedRoute>
                  }
                >
                  <Route index element={<DashboardPage />} />
                  <Route path="domains" element={<DomainsPage />} />
                  <Route path="domains/:id" element={<DomainDetailLayout />}>
                    <Route index element={<Navigate to="overview" replace />} />
                    <Route path="overview" element={<DomainOverview />} />
                    <Route path="dns" element={<DomainDNS />} />
                    <Route path="ssl" element={<DomainSSL />} />
                    <Route path="php" element={<DomainPHP />} />
                    <Route path="files" element={<DomainFiles />} />
                    <Route path="databases" element={<DomainDatabases />} />
                    <Route path="ftp" element={<DomainFTP />} />
                    <Route path="email" element={<DomainEmail />} />
                    <Route path="git" element={<DomainGit />} />
                    <Route path="logs" element={<DomainLogs />} />
                    <Route path="backups" element={<DomainBackups />} />
                    <Route path="settings" element={<DomainSettings />} />
                  </Route>
                  <Route path="files" element={<FilesPage />} />
                  <Route path="databases" element={<DatabasesPage />} />
                  <Route path="backups" element={<BackupsPage />} />
                  <Route path="logs" element={<LogsPage />} />
                  <Route path="users" element={<UsersPage />} />
                  <Route path="security" element={<SecurityPage />} />
                  <Route path="updates" element={<UpdatesPage />} />
                  <Route path="settings" element={<SettingsPage />} />
                </Route>
              </Routes>
            </Suspense>
            <Toaster />
          </BrowserRouter>
        </TooltipProvider>
      </QueryClientProvider>
    </ErrorBoundary>
  );
}

export default App;
