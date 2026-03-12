import { useState } from "react";
import { useNavigate } from "react-router-dom";
import { useMutation, useQuery } from "@tanstack/react-query";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { login, verify2FA, getSetupStatus, setupAdmin } from "@/api/auth";
import { useAuthStore } from "@/stores/auth";
import { toast } from "sonner";
import type { LoginRequest, SetupRequest } from "@/types/api";
import type { AxiosError } from "axios";
import type { APIError } from "@/types/api";
import { ShieldCheck } from "lucide-react";

export function LoginPage() {
  const { data: setupStatus, isLoading: setupLoading } = useQuery({
    queryKey: ["setup", "status"],
    queryFn: getSetupStatus,
  });

  if (setupLoading) {
    return (
      <div className="flex min-h-screen items-center justify-center">
        <p className="text-muted-foreground">Loading...</p>
      </div>
    );
  }

  if (setupStatus?.setup_required) {
    return <SetupForm />;
  }

  return <LoginForm />;
}

function LoginForm() {
  const [username, setUsername] = useState("");
  const [password, setPassword] = useState("");
  const [tempToken, setTempToken] = useState<string | null>(null);
  const [totpCode, setTotpCode] = useState("");
  const [useRecovery, setUseRecovery] = useState(false);
  const navigate = useNavigate();
  const setAuthenticated = useAuthStore((s) => s.setAuthenticated);

  const loginMutation = useMutation({
    mutationFn: (data: LoginRequest) => login(data),
    onSuccess: (data, variables) => {
      if (data.requires_2fa && data.temp_token) {
        setTempToken(data.temp_token);
        return;
      }
      setAuthenticated(variables.username, (data.role as "super_admin" | "admin" | "user") ?? "super_admin");
      navigate("/");
    },
    onError: (error: AxiosError<APIError>) => {
      const message =
        error.response?.data?.error?.message ?? "Login failed";
      toast.error(message);
    },
  });

  const verifyMutation = useMutation({
    mutationFn: () =>
      verify2FA({
        temp_token: tempToken!,
        ...(useRecovery ? { recovery_code: totpCode } : { code: totpCode }),
      }),
    onSuccess: (data) => {
      setAuthenticated(username, (data.role as "super_admin" | "admin" | "user") ?? "super_admin");
      navigate("/");
    },
    onError: (error: AxiosError<APIError>) => {
      toast.error(error.response?.data?.error?.message ?? "Invalid code");
    },
  });

  function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    if (!username || !password) {
      toast.error("Username and password are required");
      return;
    }
    loginMutation.mutate({ username, password });
  }

  function handleVerify(e: React.FormEvent) {
    e.preventDefault();
    if (!totpCode) {
      toast.error("Verification code is required");
      return;
    }
    verifyMutation.mutate();
  }

  // 2FA verification step
  if (tempToken) {
    return (
      <div className="flex min-h-screen items-center justify-center p-4">
        <Card className="w-full max-w-sm">
          <CardHeader className="text-center">
            <div className="flex justify-center mb-2">
              <ShieldCheck className="h-12 w-12 text-pink-500" />
            </div>
            <CardTitle className="text-2xl">Two-Factor Authentication</CardTitle>
            <CardDescription>
              {useRecovery
                ? "Enter a recovery code"
                : "Enter the 6-digit code from your authenticator app"}
            </CardDescription>
          </CardHeader>
          <CardContent>
            <form onSubmit={handleVerify} className="space-y-4">
              <div className="space-y-2">
                <Label htmlFor="totp-code">
                  {useRecovery ? "Recovery Code" : "Verification Code"}
                </Label>
                <Input
                  id="totp-code"
                  type="text"
                  value={totpCode}
                  onChange={(e) => setTotpCode(e.target.value)}
                  placeholder={useRecovery ? "xxxxx-xxxxx" : "000000"}
                  autoComplete="one-time-code"
                  autoFocus
                  className="text-center text-lg tracking-widest"
                />
              </div>
              <Button
                type="submit"
                className="w-full bg-pink-500 hover:bg-pink-600"
                disabled={verifyMutation.isPending}
              >
                {verifyMutation.isPending ? "Verifying..." : "Verify"}
              </Button>
              <div className="text-center">
                <button
                  type="button"
                  className="text-xs text-muted-foreground hover:text-foreground underline"
                  onClick={() => {
                    setUseRecovery(!useRecovery);
                    setTotpCode("");
                  }}
                >
                  {useRecovery ? "Use authenticator app instead" : "Use a recovery code"}
                </button>
              </div>
              <div className="text-center">
                <button
                  type="button"
                  className="text-xs text-muted-foreground hover:text-foreground"
                  onClick={() => {
                    setTempToken(null);
                    setTotpCode("");
                    setUseRecovery(false);
                  }}
                >
                  Back to login
                </button>
              </div>
            </form>
          </CardContent>
        </Card>
      </div>
    );
  }

  return (
    <div className="flex min-h-screen items-center justify-center p-4">
      <Card className="w-full max-w-sm">
        <CardHeader className="text-center">
          <div className="flex justify-center mb-2">
            <img src="/logo.png" alt="PinkPanel" className="h-12 w-12 rounded-lg" />
          </div>
          <CardTitle className="text-2xl">
            <span className="text-pink-500">Pink</span>Panel
          </CardTitle>
          <CardDescription>Sign in to your panel</CardDescription>
        </CardHeader>
        <CardContent>
          <form onSubmit={handleSubmit} className="space-y-4">
            <div className="space-y-2">
              <Label htmlFor="username">Username</Label>
              <Input
                id="username"
                type="text"
                value={username}
                onChange={(e) => setUsername(e.target.value)}
                placeholder="admin"
                autoComplete="username"
                autoFocus
              />
            </div>
            <div className="space-y-2">
              <Label htmlFor="password">Password</Label>
              <Input
                id="password"
                type="password"
                value={password}
                onChange={(e) => setPassword(e.target.value)}
                placeholder="Password"
                autoComplete="current-password"
              />
            </div>
            <Button
              type="submit"
              className="w-full bg-pink-500 hover:bg-pink-600"
              disabled={loginMutation.isPending}
            >
              {loginMutation.isPending ? "Signing in..." : "Sign in"}
            </Button>
          </form>
        </CardContent>
      </Card>
    </div>
  );
}

function SetupForm() {
  const [username, setUsername] = useState("");
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const navigate = useNavigate();
  const setAuthenticated = useAuthStore((s) => s.setAuthenticated);

  const setupMutation = useMutation({
    mutationFn: (data: SetupRequest) => setupAdmin(data),
    onSuccess: (_data, variables) => {
      setAuthenticated(variables.username, "super_admin");
      toast.success("Admin account created");
      navigate("/");
    },
    onError: (error: AxiosError<APIError>) => {
      const message =
        error.response?.data?.error?.message ?? "Setup failed";
      toast.error(message);
    },
  });

  function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    if (!username || !email || !password) {
      toast.error("All fields are required");
      return;
    }
    if (password.length < 8) {
      toast.error("Password must be at least 8 characters");
      return;
    }
    setupMutation.mutate({ username, email, password });
  }

  return (
    <div className="flex min-h-screen items-center justify-center p-4">
      <Card className="w-full max-w-sm">
        <CardHeader className="text-center">
          <div className="flex justify-center mb-2">
            <img src="/logo.png" alt="PinkPanel" className="h-12 w-12 rounded-lg" />
          </div>
          <CardTitle className="text-2xl">
            <span className="text-pink-500">Pink</span>Panel Setup
          </CardTitle>
          <CardDescription>Create your admin account</CardDescription>
        </CardHeader>
        <CardContent>
          <form onSubmit={handleSubmit} className="space-y-4">
            <div className="space-y-2">
              <Label htmlFor="setup-username">Username</Label>
              <Input
                id="setup-username"
                type="text"
                value={username}
                onChange={(e) => setUsername(e.target.value)}
                placeholder="admin"
                autoFocus
              />
            </div>
            <div className="space-y-2">
              <Label htmlFor="setup-email">Email</Label>
              <Input
                id="setup-email"
                type="email"
                value={email}
                onChange={(e) => setEmail(e.target.value)}
                placeholder="admin@example.com"
              />
            </div>
            <div className="space-y-2">
              <Label htmlFor="setup-password">Password</Label>
              <Input
                id="setup-password"
                type="password"
                value={password}
                onChange={(e) => setPassword(e.target.value)}
                placeholder="Min. 8 characters"
              />
            </div>
            <Button
              type="submit"
              className="w-full bg-pink-500 hover:bg-pink-600"
              disabled={setupMutation.isPending}
            >
              {setupMutation.isPending ? "Creating..." : "Create Admin"}
            </Button>
          </form>
        </CardContent>
      </Card>
    </div>
  );
}
