import { Link, useLocation } from "react-router-dom";
import {
  LayoutDashboard,
  Globe,
  Database,
  FolderOpen,
  Archive,
  ScrollText,
  Settings,
  PanelLeftClose,
  PanelLeft,
  Users,
} from "lucide-react";
import { cn } from "@/lib/utils";
import { useUIStore } from "@/stores/ui";
import { useAuthStore } from "@/stores/auth";
import { Button } from "@/components/ui/button";

interface NavItem {
  path: string;
  label: string;
  icon: typeof LayoutDashboard;
  adminOnly?: boolean;
}

const navItems: NavItem[] = [
  { path: "/", label: "Dashboard", icon: LayoutDashboard },
  { path: "/domains", label: "Domains", icon: Globe },
  { path: "/databases", label: "Databases", icon: Database },
  { path: "/files", label: "Files", icon: FolderOpen },
  { path: "/backups", label: "Backups", icon: Archive },
  { path: "/logs", label: "Logs", icon: ScrollText },
  { path: "/users", label: "Users", icon: Users, adminOnly: true },
  { path: "/settings", label: "Settings", icon: Settings },
];

export function Sidebar() {
  const location = useLocation();
  const collapsed = useUIStore((s) => s.sidebarCollapsed);
  const toggleSidebar = useUIStore((s) => s.toggleSidebar);
  const role = useAuthStore((s) => s.role);

  const isAdmin = role === "super_admin" || role === "admin";

  const visibleItems = navItems.filter(
    (item) => !item.adminOnly || isAdmin
  );

  return (
    <aside
      className={cn(
        "hidden md:flex flex-col border-r border-border bg-card transition-all duration-200",
        collapsed ? "w-16" : "w-60"
      )}
    >
      <div className="flex h-14 items-center border-b border-border px-4">
        <Link to="/" className="flex items-center gap-2">
          <img src="/logo.png" alt="PinkPanel" className="h-7 w-7 rounded" />
          {!collapsed && (
            <span className="text-lg font-bold">
              <span className="text-pink-500">Pink</span>
              <span className="text-foreground">Panel</span>
            </span>
          )}
        </Link>
        <Button
          variant="ghost"
          size="icon"
          className={cn("ml-auto h-8 w-8", collapsed && "mx-auto")}
          onClick={toggleSidebar}
        >
          {collapsed ? (
            <PanelLeft className="h-4 w-4" />
          ) : (
            <PanelLeftClose className="h-4 w-4" />
          )}
        </Button>
      </div>

      <nav className="flex-1 space-y-1 p-2">
        {visibleItems.map((item) => {
          const isActive =
            item.path === "/"
              ? location.pathname === "/"
              : location.pathname.startsWith(item.path);

          return (
            <Link
              key={item.path}
              to={item.path}
              className={cn(
                "flex items-center gap-3 rounded-md px-3 py-2 text-sm font-medium transition-colors",
                isActive
                  ? "bg-pink-500/10 text-pink-500"
                  : "text-muted-foreground hover:bg-accent hover:text-foreground",
                collapsed && "justify-center px-2"
              )}
            >
              <item.icon className="h-4 w-4 shrink-0" />
              {!collapsed && <span>{item.label}</span>}
            </Link>
          );
        })}
      </nav>
    </aside>
  );
}
