import { create } from "zustand";
import { hasTokens } from "@/api/client";

export type UserRole = "super_admin" | "admin" | "user";

interface AuthState {
  isAuthenticated: boolean;
  username: string | null;
  role: UserRole | null;
  setAuthenticated: (username: string, role: UserRole) => void;
  clearAuth: () => void;
}

export const useAuthStore = create<AuthState>((set) => ({
  isAuthenticated: hasTokens(),
  username: localStorage.getItem("pinkpanel_username"),
  role: (localStorage.getItem("pinkpanel_role") as UserRole) || null,
  setAuthenticated: (username, role) => {
    localStorage.setItem("pinkpanel_username", username);
    localStorage.setItem("pinkpanel_role", role);
    set({ isAuthenticated: true, username, role });
  },
  clearAuth: () => {
    localStorage.removeItem("pinkpanel_username");
    localStorage.removeItem("pinkpanel_role");
    set({ isAuthenticated: false, username: null, role: null });
  },
}));
