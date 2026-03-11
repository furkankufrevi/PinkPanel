import { create } from "zustand";
import { hasTokens } from "@/api/client";

interface AuthState {
  isAuthenticated: boolean;
  username: string | null;
  setAuthenticated: (username: string) => void;
  clearAuth: () => void;
}

export const useAuthStore = create<AuthState>((set) => ({
  isAuthenticated: hasTokens(),
  username: localStorage.getItem("pinkpanel_username"),
  setAuthenticated: (username) => {
    localStorage.setItem("pinkpanel_username", username);
    set({ isAuthenticated: true, username });
  },
  clearAuth: () => {
    localStorage.removeItem("pinkpanel_username");
    set({ isAuthenticated: false, username: null });
  },
}));
