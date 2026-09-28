"use client";

import { useMemo, useState } from "react";
import Link from "next/link";
import { ChevronLeft, ChevronRight } from "lucide-react";
import { Installment } from "@/lib/types";
import { faDate, toman } from "@/lib/format";
import { calendarKind, formatCalNum, gregorianToParts, monthName, todayParts } from "@/lib/calendar";
import { DebtLogo } from "./DebtLogo";
import { Modal } from "./ui";
import { useI18n } from "@/lib/i18n";

function pad(n: number) {
  return String(n).padStart(2, "0");
}

export function DueBoard({
  remaining,
  items,
}: {
  remaining: number;
  items: Installment[];
}) {
  const { t, locale } = useI18n();
  const kind = calendarKind(locale);
  const today = todayParts(kind);
  const todayKey = `${today.y}-${pad(today.m)}`;
  const todayISO = `${new Date().getFullYear()}-${pad(new Date().getMonth() + 1)}-${pad(new Date().getDate())}`;

  const { overdue, thisMonth, later } = useMemo(() => {
    const overdue: Installment[] = [];
    const thisMonth: Installment[] = [];
    const laterMap = new Map<string, { y: number; m: number; items: Installment[]; total: number }>();
    for (const it of items) {
      const iso = (it.due_date || "").slice(0, 10);
      const parts = gregorianToParts(iso, kind);
      if (!parts) continue;
      const key = `${parts.y}-${pad(parts.m)}`;
      if (iso < todayISO) {
        overdue.push(it);
        continue;
      }
      if (key === todayKey) {
        thisMonth.push(it);
        continue;
      }
      if (key < todayKey) {
        overdue.push(it);
        continue;
      }
      const bucket = laterMap.get(key) || { y: parts.y, m: parts.m, items: [], total: 0 };
      bucket.items.push(it);
      bucket.total += it.amount;
      laterMap.set(key, bucket);
    }
    const later = [...laterMap.values()].sort((a, b) => a.y - b.y || a.m - b.m);
    return { overdue, thisMonth, later };
  }, [items, kind, todayISO, todayKey]);

  const monthItems = [...overdue, ...thisMonth];
  const monthTotal = monthItems.reduce((s, it) => s + it.amount, 0);
  const [openMonth, setOpenMonth] = useState<(typeof later)[0] | null>(null);

  return (
    <div className="overflow-hidden rounded-[28px] border border-rose-400/15 bg-gradient-to-bl from-[#1c1216] via-ink-900 to-[#10141c]">
      <div className="px-5 pt-5 md:px-6">
        <div className="text-xs text-white/40">{t("dash.debtTotal")}</div>
        <div className="mt-1 text-3xl font-black text-rose-300 md:text-4xl">{toman(remaining, true, locale)}</div>
      </div>

      <div className="mt-5 border-t border-white/6 px-5 py-5 md:px-6">
        <div className="flex items-end justify-between gap-3">
          <div>
            <h3 className="font-bold">{t("dash.dueThisMonth")}</h3>
            <p className="mt-1 text-xs text-white/40">
              {t("dash.dueThisMonthCount", { n: monthItems.length })}
              {overdue.length > 0 ? ` · ${t("dash.overdueN", { n: overdue.length })}` : ""}
            </p>
          </div>
          <div className="text-end">
            <div className="text-[11px] text-white/35">{t("dash.dueMonthTotal")}</div>
            <div className="text-lg font-extrabold text-gold-200">{toman(monthTotal, true, locale)}</div>
          </div>
        </div>

        {monthItems.length === 0 ? (
          <p className="mt-4 text-sm text-white/40">{t("dash.noThisMonth")}</p>
        ) : (
          <div className="mt-4 space-y-2">
            {monthItems.map((it) => (
              <DueRow key={it.id} it={it} overdue={(it.due_date || "").slice(0, 10) < todayISO} />
            ))}
          </div>
        )}
      </div>

      {later.length > 0 && (
        <div className="border-t border-white/6 px-5 py-4 md:px-6">
          <h3 className="mb-3 text-sm font-bold text-white/70">{t("dash.dueLater")}</h3>
          <div className="space-y-2">
            {later.map((m) => (
              <button
                key={`${m.y}-${m.m}`}
                type="button"
                onClick={() => setOpenMonth(m)}
                className="flex w-full items-center justify-between gap-3 rounded-2xl bg-white/5 px-4 py-3 text-start transition hover:bg-white/8"
              >
                <div>
                  <div className="font-bold">
                    {monthName(m.m, locale)} {formatCalNum(m.y, locale)}
                  </div>
                  <div className="mt-0.5 text-xs text-white/40">{t("dash.dueThisMonthCount", { n: m.items.length })}</div>
                </div>
                <div className="flex items-center gap-2">
                  <span className="font-extrabold text-rose-200">{toman(m.total, true, locale)}</span>
                  <ChevronLeft className="h-4 w-4 text-white/35 ltr:hidden" />
                  <ChevronRight className="hidden h-4 w-4 text-white/35 ltr:block" />
                </div>
              </button>
            ))}
          </div>
        </div>
      )}

      {openMonth && (
        <Modal
          title={t("dash.monthPopup", { month: `${monthName(openMonth.m, locale)} ${formatCalNum(openMonth.y, locale)}` })}
          subtitle={toman(openMonth.total, true, locale)}
          onClose={() => setOpenMonth(null)}
        >
          <div className="space-y-2">
            {openMonth.items.map((it) => (
              <DueRow key={it.id} it={it} />
            ))}
          </div>
        </Modal>
      )}
    </div>
  );
}

function DueRow({ it, overdue }: { it: Installment; overdue?: boolean }) {
  const { locale } = useI18n();
  return (
    <Link
      href={`/debts/${it.debt_id}`}
      className={`flex items-center gap-3 rounded-2xl px-3 py-2.5 ${overdue ? "bg-rose-500/12" : "bg-white/5"}`}
    >
      <DebtLogo debtId={it.debt_id} hasLogo={it.has_logo} name={it.debt_name || ""} color={it.color} size="sm" />
      <div className="min-w-0 flex-1">
        <div className="truncate font-bold">{it.debt_name}</div>
        <div className="text-xs text-white/40">
          {it.creditor ? `${it.creditor} · ` : ""}
          {faDate(it.due_date, false, locale)}
        </div>
      </div>
      <div className={`shrink-0 font-extrabold ${overdue ? "text-rose-300" : "text-white"}`}>{toman(it.amount, true, locale)}</div>
    </Link>
  );
}
