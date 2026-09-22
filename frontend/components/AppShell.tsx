"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";
import {
  LayoutDashboard,
  Landmark,
  Target,
  ArrowLeftRight,
  PieChart,
  Plus,
  Wallet,
  Gem,
  HandCoins,
  MoreHorizontal,
  Languages,
  LogOut,
} from "lucide-react";
import { useState } from "react";
import { QuickTxModal } from "./QuickTxModal";
import { LOCALES, useI18n } from "@/lib/i18n";
import { useAuth } from "@/lib/auth";

const NAV = [
  { href: "/", key: "nav.dashboard", icon: LayoutDashboard },
  { href: "/accounts", key: "nav.accounts", icon: Landmark },
  { href: "/assets", key: "nav.assets", icon: Gem },
  { href: "/projects", key: "nav.projects", icon: Target },
  { href: "/debts", key: "nav.debts", icon: HandCoins },
  { href: "/transactions", key: "nav.transactions", icon: ArrowLeftRight },
  { href: "/analytics", key: "nav.analytics", icon: PieChart },
];

const MORE_HREFS = ["/assets", "/projects", "/transactions", "/analytics"];

export function AppShell({ children }: { children: React.ReactNode }) {
  const path = usePathname();
  const [open, setOpen] = useState(false);
  const [more, setMore] = useState(false);
  const { t, locale, setLocale } = useI18n();
  const { user, logout } = useAuth();
  const moreActive = MORE_HREFS.some((href) => path.startsWith(href));

  return (
    <div className="min-h-dvh bg-mesh bg-ink-950">
      <aside className="fixed inset-x-0 bottom-0 z-40 border-t border-[var(--line)] bg-ink-900/95 backdrop-blur-xl md:inset-x-auto md:bottom-auto md:top-0 md:h-screen md:w-[260px] md:start-0 md:border-t-0 md:border-e md:border-[var(--line)]">
        <div className="hidden items-center gap-3 px-6 py-7 md:flex">
          <div className="flex h-11 w-11 items-center justify-center rounded-2xl bg-gradient-to-br from-gold-200 to-gold-700 shadow-glow">
            <Wallet className="h-5 w-5 text-ink-950" />
          </div>
          <div>
            <div className="text-lg font-extrabold tracking-tight gold-text">{t("app.name")}</div>
            <div className="text-[11px] text-gold-200/60">{t("app.tagline")}</div>
          </div>
        </div>

        <nav className="hidden md:flex md:flex-col md:gap-1 md:px-4 md:py-2">
          {NAV.map((item) => {
            const active = item.href === "/" ? path === "/" : path.startsWith(item.href);
            const Icon = item.icon;
            return (
              <Link
                key={item.href}
                href={item.href}
                className={`flex items-center gap-3 rounded-2xl px-4 py-3 text-sm transition ${
                  active
                    ? "bg-gold-400/10 text-gold-200 shadow-[inset_0_0_0_1px_rgba(224,195,106,0.18)]"
                    : "text-white/45 hover:bg-white/5 hover:text-white/80"
                }`}
              >
                <Icon className="h-5 w-5" />
                <span className="font-medium">{t(item.key)}</span>
              </Link>
            );
          })}
        </nav>

        <nav className="grid grid-cols-5 items-end px-1 pt-1 pb-[max(0.35rem,env(safe-area-inset-bottom))] md:hidden">
          <MobileTab href="/" icon={LayoutDashboard} label={t("nav.dashboard")} active={path === "/"} />
          <MobileTab href="/accounts" icon={Landmark} label={t("nav.accounts")} active={path.startsWith("/accounts")} />
          <button
            onClick={() => setOpen(true)}
            className="relative -mt-5 flex flex-col items-center justify-end"
            aria-label={t("common.quickTx")}
          >
            <span className="flex h-14 w-14 items-center justify-center rounded-full bg-gradient-to-br from-gold-200 to-gold-700 text-ink-950 shadow-glow ring-4 ring-ink-950">
              <Plus className="h-6 w-6" />
            </span>
          </button>
          <MobileTab href="/debts" icon={HandCoins} label={t("nav.debts")} active={path.startsWith("/debts")} />
          <button
            onClick={() => setMore(true)}
            className={`flex min-h-[3.4rem] flex-col items-center justify-center gap-0.5 rounded-2xl px-1 text-[10px] ${
              moreActive ? "text-gold-200" : "text-white/45"
            }`}
          >
            <MoreHorizontal className="h-5 w-5" />
            <span className="max-w-[4.6rem] truncate font-medium">{t("nav.more")}</span>
          </button>
        </nav>

        <div className="hidden px-4 pt-3 md:block">
          <div className="mb-3 flex rounded-2xl bg-white/5 p-1">
            {LOCALES.map((l) => (
              <button
                key={l.id}
                onClick={() => setLocale(l.id)}
                className={`flex-1 rounded-xl py-1.5 text-[11px] font-bold ${
                  locale === l.id ? "bg-gold-400 text-ink-950" : "text-white/50 hover:text-white"
                }`}
              >
                {l.id.toUpperCase()}
              </button>
            ))}
          </div>
          <button
            onClick={() => setOpen(true)}
            className="flex w-full items-center justify-center gap-2 rounded-2xl bg-gradient-to-r from-gold-400 to-gold-700 py-3 text-sm font-bold text-ink-950 shadow-glow transition hover:brightness-110 rtl:bg-gradient-to-l rtl:from-gold-700 rtl:to-gold-400"
          >
            <Plus className="h-4 w-4" />
            {t("common.quickTx")}
          </button>
          {user && (
            <div className="mt-4 flex items-center justify-between gap-2 rounded-2xl bg-white/5 px-3 py-2">
              <div className="min-w-0">
                <div className="truncate text-xs font-bold text-white/80">{user.username}</div>
                <div className="text-[10px] text-white/35">{t("auth.signedIn")}</div>
              </div>
              <button
                onClick={() => logout()}
                className="rounded-xl p-2 text-white/40 hover:bg-white/10 hover:text-rose-300"
                aria-label={t("auth.logout")}
              >
                <LogOut className="h-4 w-4" />
              </button>
            </div>
          )}
        </div>
      </aside>

      {more && (
        <div className="fixed inset-0 z-[70] md:hidden" onClick={() => setMore(false)}>
          <div className="absolute inset-0 bg-black/60" />
          <div
            onClick={(e) => e.stopPropagation()}
            className="absolute inset-x-0 bottom-0 rounded-t-3xl border border-white/10 bg-[#0e121a] p-5 pb-[max(1.25rem,env(safe-area-inset-bottom))]"
          >
            <div className="mx-auto mb-4 h-1 w-10 rounded-full bg-white/20" />
            <div className="mb-4 text-sm font-bold">{t("nav.moreTitle")}</div>
            <div className="grid grid-cols-2 gap-2">
              {NAV.filter((item) => MORE_HREFS.includes(item.href)).map((item) => {
                const Icon = item.icon;
                const active = path.startsWith(item.href);
                return (
                  <Link
                    key={item.href}
                    href={item.href}
                    onClick={() => setMore(false)}
                    className={`flex items-center gap-3 rounded-2xl px-4 py-3 text-sm ${
                      active ? "bg-gold-400/10 text-gold-200" : "bg-white/5 text-white/75"
                    }`}
                  >
                    <Icon className="h-5 w-5" />
                    {t(item.key)}
                  </Link>
                );
              })}
            </div>
            <div className="mt-5 flex items-center gap-2 text-xs text-white/40">
              <Languages className="h-4 w-4" />
              {LOCALES.find((l) => l.id === locale)?.label}
            </div>
            <div className="mt-2 flex rounded-2xl bg-white/5 p-1">
              {LOCALES.map((l) => (
                <button
                  key={l.id}
                  onClick={() => setLocale(l.id)}
                  className={`flex-1 rounded-xl py-2 text-xs font-bold ${
                    locale === l.id ? "bg-gold-400 text-ink-950" : "text-white/50"
                  }`}
                >
                  {l.label}
                </button>
              ))}
            </div>
            <button
              onClick={() => logout()}
              className="mt-4 flex w-full items-center justify-center gap-2 rounded-2xl bg-white/5 py-3 text-sm text-white/60"
            >
              <LogOut className="h-4 w-4" />
              {t("auth.logout")}
            </button>
          </div>
        </div>
      )}

      <main className="px-4 pb-[calc(5.75rem+env(safe-area-inset-bottom))] pt-[max(1rem,env(safe-area-inset-top))] md:ps-[280px] md:pe-8 md:pb-10 md:pt-8">
        {children}
      </main>
      {open && <QuickTxModal onClose={() => setOpen(false)} />}
    </div>
  );
}

function MobileTab({
  href,
  icon: Icon,
  label,
  active,
}: {
  href: string;
  icon: typeof LayoutDashboard;
  label: string;
  active: boolean;
}) {
  return (
    <Link
      href={href}
      className={`flex min-h-[3.4rem] flex-col items-center justify-center gap-0.5 rounded-2xl px-1 text-[10px] ${
        active ? "text-gold-200" : "text-white/45"
      }`}
    >
      <Icon className="h-5 w-5" />
      <span className="max-w-[4.6rem] truncate font-medium">{label}</span>
    </Link>
  );
}
