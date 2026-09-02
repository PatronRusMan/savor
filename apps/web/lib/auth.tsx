"use client";

import { createContext, useCallback, useContext, useEffect, useMemo, useState } from "react";
import { api, type AuthPayload, type User } from "./api";

type AuthState = {
  user: User | null;
  token: string | null;
  ready: boolean;
  login: (email: string, password: string) => Promise<User>;
  register: (input: { email: string; password: string; name: string; role: User["role"] }) => Promise<User>;
  logout: () => void;
};

const Ctx = createContext<AuthState | null>(null);

const TOKEN_KEY = "savor.access";
const REFRESH_KEY = "savor.refresh";
const USER_KEY = "savor.user";

export function AuthProvider({ children }: { children: React.ReactNode }) {
  const [user, setUser] = useState<User | null>(null);
  const [token, setToken] = useState<string | null>(null);
  const [ready, setReady] = useState(false);

  const apply = useCallback((p: AuthPayload) => {
    localStorage.setItem(TOKEN_KEY, p.accessToken);
    localStorage.setItem(REFRESH_KEY, p.refreshToken);
    localStorage.setItem(USER_KEY, JSON.stringify(p.user));
    setToken(p.accessToken);
    setUser(p.user);
  }, []);

  useEffect(() => {
    const t = localStorage.getItem(TOKEN_KEY);
    const u = localStorage.getItem(USER_KEY);
    if (t && u) {
      setToken(t);
      setUser(JSON.parse(u) as User);
    }
    setReady(true);
  }, []);

  const login = useCallback(async (email: string, password: string) => {
    const p = await api<AuthPayload>("/api/v1/auth/login", { method: "POST", body: JSON.stringify({ email, password }) });
    apply(p);
    return p.user;
  }, [apply]);

  const register = useCallback(async (input: { email: string; password: string; name: string; role: User["role"] }) => {
    const p = await api<AuthPayload>("/api/v1/auth/register", { method: "POST", body: JSON.stringify(input) });
    apply(p);
    return p.user;
  }, [apply]);

  const logout = useCallback(() => {
    const refresh = localStorage.getItem(REFRESH_KEY);
    if (refresh) {
      void api("/api/v1/auth/logout", { method: "POST", token, body: JSON.stringify({ refreshToken: refresh }) }).catch(() => null);
    }
    localStorage.removeItem(TOKEN_KEY);
    localStorage.removeItem(REFRESH_KEY);
    localStorage.removeItem(USER_KEY);
    setToken(null);
    setUser(null);
  }, [token]);

  const value = useMemo(() => ({ user, token, ready, login, register, logout }), [user, token, ready, login, register, logout]);
  return <Ctx.Provider value={value}>{children}</Ctx.Provider>;
}

export function useAuth() {
  const v = useContext(Ctx);
  if (!v) throw new Error("AuthProvider missing");
  return v;
}

export function homeFor(role?: User["role"]) {
  if (role === "restaurant") return "/restaurant";
  if (role === "courier") return "/courier";
  return "/";
}
