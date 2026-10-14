"use client";

import { useMemo, useState } from "react";
import Link from "next/link";
import { Check, ChevronLeft, ChevronRight } from "lucide-react";
import { Installment } from "@/lib/types";
import { faDate, toman } from "@/lib/format";
import { calendarKind, formatCalNum, gregorianToParts, monthName, todayParts, type CalendarKind } from "@/lib/calendar";
import { DebtLogo } from "./DebtLogo";
import { Modal } from "./ui";
import { useI18n } from "@/lib/i18n";

function pad(n: number) {
  return String(n).padStart(2, "0");
}

function dueYMD(raw: string) {
  const m = /(\d{4}-\d{2}-\d{2})/.exec(raw || "");
  return m ? m[1] : "";
}

function rankInst(it: Installment) {
  if (it.status === "paid" && it.account_id) return 2;
  if (it.status === "paid") return 1;
  return 0;
}

function onePerDebt(items: Installment[]) {
  const map = new Map<string, Installment>();
  for (const it of items) {
    const prev = map.get(it.debt_id);
    if (!prev || rankInst(it) > rankInst(prev) || (rankInst(it) === rankInst(prev) && dueYMD(it.due_date) < dueYMD(prev.due_date))) {
      map.set(it.debt_id, it);
    }
  }
  return [...map.values()];
}

function onePerDebtMonth(items: Installment[], kind: CalendarKind) {
  const map = new Map<string, Installment>();
  for (const it of items) {
    const iso = dueYMD(it.due_date);
    const parts = gregorianToParts(iso, kind);
    const key = `${it.debt_id}-${parts?.y ?? 0}-${pad(parts?.m ?? 0)}`;
    const prev = map.get(key);
    if (!prev || rankInst(it) > rankInst(prev) || (rankInst(it) === rankInst(prev) && iso < dueYMD(prev.due_date))) {
      map.set(key, it);
    }
  }
  return [...map.values()];
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
  const todayISO = `${todayParts("gregory").y}-${pad(todayParts("gregory").m)}-${pad(todayParts("gregory").d)}`;

  const { overdue, thisMonth, later } = useMemo(() => {
    const overdue: Installment[] = [];
    const thisMonth: Installment[] = [];
    const laterMap = new Map<string, { y: number; m: number; items: Installment[]; total: number }>();
    for (const it of items) {
      const iso = dueYMD(it.due_date);
      const parts = gregorianToParts(iso, kind);
      if (!parts) continue;
      const key = `${parts.y}-${pad(parts.m)}`;
      const paid = it.status === "paid";
      if (!paid && iso && iso < todayISO) {
        overdue.push(it);
        continue;
      }
      if (key === todayKey) {
        thisMonth.push(it);
        continue;
      }
      if (paid) continue;
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
    for (const bucket of later) {
      bucket.items = onePerDebt(bucket.items);
      bucket.total = bucket.items.reduce((s, it) => s + it.amount, 0);
    }
    const thisMonthOnce = onePerDebt(thisMonth);
    thisMonthOnce.sort((a, b) => {
      const pa = a.status === "paid" ? 1 : 0;
      const pb = b.status === "paid" ? 1 : 0;
      if (pa !== pb) return pa - pb;
      return dueYMD(a.due_date).localeCompare(dueYMD(b.due_date));
    });
    const overdueOnce = onePerDebtMonth(overdue, kind);
    overdueOnce.sort((a, b) => dueYMD(a.due_date).localeCompare(dueYMD(b.due_date)));
    return { overdue: overdueOnce, thisMonth: thisMonthOnce, later };
  }, [items, kind, todayISO, todayKey]);

  const openItems = [...overdue, ...thisMonth.filter((it) => it.status !== "paid")];
  const paidItems = thisMonth.filter((it) => it.status === "paid");
  const hasThisMonth = openItems.length > 0 || paidItems.length > 0;
  const [picked, setPicked] = useState<"open" | "paid" | null>(null);
  const activeTab = picked ?? (openItems.length > 0 ? "open" : "paid");
  const shown = activeTab === "paid" ? paidItems : openItems;

  const primary = hasThisMonth
    ? { title: t("dash.dueThisMonth"), items: shown, y: today.y, m: today.m }
    : later[0]
      ? { title: `${monthName(later[0].m, locale)} ${formatCalNum(later[0].y, locale)}`, items: later[0].items, y: later[0].y, m: later[0].m }
      : null;
  const restLater = hasThisMonth ? later : later.slice(1);
  const monthTotal = openItems.reduce((s, it) => s + it.amount, 0);
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
            <h3 className="font-bold">{primary ? primary.title : t("dash.dueThisMonth")}</h3>
            <p className="mt-1 text-xs text-white/40">
              {t("dash.dueThisMonthCount", { n: openItems.length || primary?.items.length || 0 })}
              {overdue.length > 0 ? ` · ${t("dash.overdueN", { n: overdue.length })}` : ""}
              {paidItems.length > 0 ? ` · ${t("dash.monthPaid", { n: paidItems.length })}` : ""}
            </p>
          </div>
          <div className="text-end">
            <div className="text-[11px] text-white/35">{t("dash.dueMonthTotal")}</div>
            <div className="text-lg font-extrabold text-gold-200">{toman(monthTotal, true, locale)}</div>
          </div>
        </div>

        {hasThisMonth && (
          <div className="mt-4 flex rounded-2xl bg-white/5 p-1">
            <TabBtn
              active={activeTab === "open"}
              onClick={() => setPicked("open")}
              count={openItems.length}
              label={t("debts.tabPending")}
            />
            <TabBtn
              active={activeTab === "paid"}
              onClick={() => setPicked("paid")}
              count={paidItems.length}
              label={t("debts.tabPaid")}
            />
          </div>
        )}

        {!primary ? (
          <p className="mt-4 text-sm text-white/40">{t("dash.noThisMonth")}</p>
        ) : shown.length === 0 && hasThisMonth ? (
          <p className="mt-4 text-sm text-white/40">
            {activeTab === "paid" ? t("dash.noPaidThisMonth") : t("dash.noThisMonth")}
          </p>
        ) : (
          <div className="mt-4 space-y-2">
            {primary.items.map((it) => (
              <DueRow key={it.id} it={it} overdue={it.status === "overdue" || (it.status !== "paid" && dueYMD(it.due_date) < todayISO)} />
            ))}
          </div>
        )}
      </div>

      {restLater.length > 0 && (
        <div className="border-t border-white/6 px-5 py-4 md:px-6">
          <h3 className="mb-3 text-sm font-bold text-white/70">{t("dash.dueLater")}</h3>
          <div className="space-y-2">
            {restLater.map((m) => (
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

function TabBtn({
  active,
  onClick,
  count,
  label,
}: {
  active: boolean;
  onClick: () => void;
  count: number;
  label: string;
}) {
  return (
    <button
      type="button"
      onClick={onClick}
      className={`flex flex-1 items-center justify-center gap-2 rounded-xl px-3 py-2.5 text-sm font-bold transition ${
        active ? "bg-white/12 text-white" : "text-white/45 hover:text-white/70"
      }`}
    >
      {label}
      <span className={`rounded-full px-1.5 py-0.5 text-[11px] ${active ? "bg-white/15" : "bg-white/8"}`}>{count}</span>
    </button>
  );
}

function DueRow({ it, overdue }: { it: Installment; overdue?: boolean }) {
  const { t, locale } = useI18n();
  const paid = it.status === "paid";
  return (
    <Link
      href={`/debts/${it.debt_id}`}
      className={`flex items-center gap-3 rounded-2xl px-3 py-2.5 ${
        paid ? "bg-emerald-500/8" : overdue ? "bg-rose-500/12" : "bg-white/5"
      }`}
    >
      <DebtLogo debtId={it.debt_id} hasLogo={it.has_logo} name={it.debt_name || ""} color={it.color} size="sm" />
      <div className="min-w-0 flex-1">
        <div className={`truncate font-bold ${paid ? "text-white/70" : ""}`}>{it.debt_name}</div>
        <div className="text-xs text-white/40">
          {it.creditor ? `${it.creditor} · ` : ""}
          {faDate(it.due_date, false, locale)}
          {paid ? ` · ${t("debts.paid")}` : ""}
        </div>
      </div>
      <div className="flex shrink-0 items-center gap-2">
        {paid && <Check className="h-4 w-4 text-emerald-300" />}
        <div className={`font-extrabold ${paid ? "text-white/45 line-through" : overdue ? "text-rose-300" : "text-white"}`}>
          {toman(it.amount, true, locale)}
        </div>
      </div>
    </Link>
  );
}
