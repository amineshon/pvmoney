"use client";

import { ReactNode } from "react";
import { useAuth } from "@/lib/auth";
import { AppShell } from "./AppShell";
import { AuthScreen, AuthSplash } from "./AuthScreen";

export function AuthGate({ children }: { children: ReactNode }) {
  const { ready, user } = useAuth();
  if (!ready) return <AuthSplash />;
  if (!user) return <AuthScreen />;
  return <AppShell>{children}</AppShell>;
}
