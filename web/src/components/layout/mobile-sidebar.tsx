import { Link, useLocation } from "react-router-dom";
import {
  LayoutDashboard,
  Globe,
  Database,
  FolderOpen,
  Archive,
  ScrollText,
  Settings,
} from "lucide-react";
import {
  Sheet,
  SheetContent,
  SheetHeader,
  SheetTitle,
} from "@/components/ui/sheet";
import { cn } from "@/lib/utils";
import { useUIStore } from "@/stores/ui";

const navItems = [
  { path: "/", label: "Dashboard", icon: LayoutDashboard },
  { path: "/domains", label: "Domains", icon: Globe },
  { path: "/databases", label: "Databases", icon: Database },
  { path: "/files", label: "Files", icon: FolderOpen },
  { path: "/backups", label: "Backups", icon: Archive },
  { path: "/logs", label: "Logs", icon: ScrollText },
  { path: "/settings", label: "Settings", icon: Settings },
];

export function MobileSidebar() {
  const location = useLocation();
  const open = useUIStore((s) => s.mobileSidebarOpen);
  const setOpen = useUIStore((s) => s.setMobileSidebarOpen);

  return (
    <Sheet open={open} onOpenChange={setOpen}>
      <SheetContent side="left" className="w-60 p-0">
        <SheetHeader className="border-b border-border px-4 py-4">
          <SheetTitle className="text-left">
            <span className="text-pink-500">Pink</span>
            <span>Panel</span>
          </SheetTitle>
        </SheetHeader>
        <nav className="space-y-1 p-2">
          {navItems.map((item) => {
            const isActive =
              item.path === "/"
                ? location.pathname === "/"
                : location.pathname.startsWith(item.path);

            return (
              <Link
                key={item.path}
                to={item.path}
                onClick={() => setOpen(false)}
                className={cn(
                  "flex items-center gap-3 rounded-md px-3 py-2 text-sm font-medium transition-colors",
                  isActive
                    ? "bg-pink-500/10 text-pink-500"
                    : "text-muted-foreground hover:bg-accent hover:text-foreground"
                )}
              >
                <item.icon className="h-4 w-4" />
                <span>{item.label}</span>
              </Link>
            );
          })}
        </nav>
      </SheetContent>
    </Sheet>
  );
}
