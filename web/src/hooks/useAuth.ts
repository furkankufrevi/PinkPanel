import { useMutation, useQueryClient } from "@tanstack/react-query";
import { useNavigate } from "react-router-dom";
import { login, logout } from "@/api/auth";
import { useAuthStore } from "@/stores/auth";
import type { LoginRequest } from "@/types/api";

export function useLogin() {
  const setAuthenticated = useAuthStore((s) => s.setAuthenticated);
  const navigate = useNavigate();

  return useMutation({
    mutationFn: (data: LoginRequest) => login(data),
    onSuccess: (data, variables) => {
      setAuthenticated(variables.username, (data.role as "super_admin" | "admin" | "user") ?? "super_admin");
      navigate("/");
    },
  });
}

export function useLogout() {
  const clearAuth = useAuthStore((s) => s.clearAuth);
  const navigate = useNavigate();
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: () => logout(),
    onSettled: () => {
      clearAuth();
      queryClient.clear();
      navigate("/login");
    },
  });
}
