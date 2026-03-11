import { create } from "zustand";

type Theme = "light" | "dark";

interface UIState {
  sidebarCollapsed: boolean;
  mobileSidebarOpen: boolean;
  theme: Theme;
  toggleSidebar: () => void;
  setMobileSidebarOpen: (open: boolean) => void;
  setTheme: (theme: Theme) => void;
  toggleTheme: () => void;
}

function getInitialTheme(): Theme {
  if (typeof window === "undefined") return "dark";
  const stored = localStorage.getItem("pinkpanel-theme");
  if (stored === "light" || stored === "dark") return stored;
  return window.matchMedia("(prefers-color-scheme: dark)").matches
    ? "dark"
    : "light";
}

export const useUIStore = create<UIState>((set, get) => ({
  sidebarCollapsed: false,
  mobileSidebarOpen: false,
  theme: getInitialTheme(),
  toggleSidebar: () =>
    set((state) => ({ sidebarCollapsed: !state.sidebarCollapsed })),
  setMobileSidebarOpen: (open) => set({ mobileSidebarOpen: open }),
  setTheme: (theme) => {
    localStorage.setItem("pinkpanel-theme", theme);
    document.documentElement.classList.toggle("dark", theme === "dark");
    set({ theme });
  },
  toggleTheme: () => {
    const next = get().theme === "dark" ? "light" : "dark";
    get().setTheme(next);
  },
}));
