"use client";

import { createContext, useCallback, useContext, useEffect, useMemo, useState, type ReactNode } from "react";
import { api, SESSION_KEY } from "./api";
import type { AuthStatus, AuthUser } from "./types";

type AuthCtx = {
  ready: boolean;
  user: AuthUser | null;
  status: AuthStatus | null;
  setSession: (token: string, user: AuthUser) => void;
  logout: () => Promise<void>;
  refresh: () => Promise<void>;
};

const Ctx = createContext<AuthCtx | null>(null);

export function AuthProvider({ children }: { children: ReactNode }) {
  const [ready, setReady] = useState(false);
  const [user, setUser] = useState<AuthUser | null>(null);
  const [status, setStatus] = useState<AuthStatus | null>(null);

  const refresh = useCallback(async () => {
    try {
      const s = await api.authStatus();
      setStatus(s);
      setUser(s.authenticated && s.user ? s.user : null);
      if (!s.authenticated) localStorage.removeItem(SESSION_KEY);
    } catch {
      setStatus(null);
      setUser(null);
    } finally {
      setReady(true);
    }
  }, []);

  useEffect(() => {
    refresh();
    const onDenied = () => {
      localStorage.removeItem(SESSION_KEY);
      setUser(null);
    };
    window.addEventListener("pvmoney:unauthorized", onDenied);
    return () => window.removeEventListener("pvmoney:unauthorized", onDenied);
  }, [refresh]);

  const setSession = useCallback((token: string, next: AuthUser) => {
    localStorage.setItem(SESSION_KEY, token);
    setUser(next);
  }, []);

  const logout = useCallback(async () => {
    try {
      await api.logout();
    } catch {
      /* still clear locally */
    }
    localStorage.removeItem(SESSION_KEY);
    setUser(null);
    await refresh();
  }, [refresh]);

  const value = useMemo(
    () => ({ ready, user, status, setSession, logout, refresh }),
    [ready, user, status, setSession, logout, refresh],
  );

  return <Ctx.Provider value={value}>{children}</Ctx.Provider>;
}

export function useAuth() {
  const ctx = useContext(Ctx);
  if (!ctx) throw new Error("useAuth outside provider");
  return ctx;
}
