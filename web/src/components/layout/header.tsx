import { Menu, Moon, Sun, LogOut, User } from "lucide-react";
import { Button } from "@/components/ui/button";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { Avatar, AvatarFallback } from "@/components/ui/avatar";
import { useAuthStore } from "@/stores/auth";
import { useUIStore } from "@/stores/ui";
import { useLogout } from "@/hooks/useAuth";

export function Header() {
  const username = useAuthStore((s) => s.username);
  const theme = useUIStore((s) => s.theme);
  const toggleTheme = useUIStore((s) => s.toggleTheme);
  const setMobileSidebarOpen = useUIStore((s) => s.setMobileSidebarOpen);
  const logoutMutation = useLogout();

  return (
    <header className="flex h-14 items-center gap-4 border-b border-border bg-card px-4">
      <Button
        variant="ghost"
        size="icon"
        className="md:hidden"
        onClick={() => setMobileSidebarOpen(true)}
      >
        <Menu className="h-5 w-5" />
      </Button>

      <div className="flex-1" />

      <Button variant="ghost" size="icon" onClick={toggleTheme}>
        {theme === "dark" ? (
          <Sun className="h-4 w-4" />
        ) : (
          <Moon className="h-4 w-4" />
        )}
      </Button>

      <DropdownMenu>
        <DropdownMenuTrigger className="flex items-center gap-2 rounded-md px-2 py-1.5 hover:bg-accent outline-none cursor-pointer">
          <Avatar className="h-7 w-7">
            <AvatarFallback className="bg-pink-500/20 text-pink-500 text-xs">
              {username?.[0]?.toUpperCase() ?? "A"}
            </AvatarFallback>
          </Avatar>
          <span className="hidden sm:inline text-sm">{username}</span>
        </DropdownMenuTrigger>
        <DropdownMenuContent align="end">
          <DropdownMenuItem disabled>
            <User className="mr-2 h-4 w-4" />
            {username}
          </DropdownMenuItem>
          <DropdownMenuSeparator />
          <DropdownMenuItem
            onClick={() => logoutMutation.mutate()}
            className="text-destructive"
          >
            <LogOut className="mr-2 h-4 w-4" />
            Logout
          </DropdownMenuItem>
        </DropdownMenuContent>
      </DropdownMenu>
    </header>
  );
}
