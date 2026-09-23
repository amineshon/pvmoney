"use client";

import { useMemo, useState } from "react";
import { motion } from "framer-motion";
import { CalendarClock, Check, CircleAlert } from "lucide-react";
import { Debt, Installment } from "@/lib/types";
import { faDate, toman } from "@/lib/format";
import { Btn } from "@/components/ui";
import { useI18n } from "@/lib/i18n";

type Tab = "pending" | "paid";

function daysUntil(iso: string) {
  const due = new Date(`${iso.slice(0, 10)}T00:00:00`);
  const today = new Date();
  today.setHours(0, 0, 0, 0);
  return Math.round((due.getTime() - today.getTime()) / 86400000);
}

function isPrior(it: Installment) {
  return it.status === "paid" && !it.account_id;
}

function isOpen(it: Installment) {
  return it.status !== "paid";
}

export function InstallmentTimeline({
  debt,
  onPay,
  onMarkPrior,
  onUndoPrior,
}: {
  debt: Debt;
  onPay: (it: Installment) => void;
  onMarkPrior: (it: Installment) => void;
  onUndoPrior: (it: Installment) => void;
}) {
  const { t, locale } = useI18n();
  const items = debt.installments || [];
  const [tab, setTab] = useState<Tab>("pending");

  const { pending, paid, next } = useMemo(() => {
    const pending = items.filter(isOpen).sort((a, b) => a.due_date.localeCompare(b.due_date));
    const paid = items.filter((it) => it.status === "paid").sort((a, b) => (b.paid_at || b.due_date).localeCompare(a.paid_at || a.due_date));
    return { pending, paid, next: pending[0] || null };
  }, [items]);

  if (items.length === 0) {
    return <div className="glass rounded-[24px] p-6 text-sm text-white/45">{t("debts.noInst")}</div>;
  }

  const last = items[items.length - 1];
  const lastIsRemainder = items.length > 1 && last && last.amount !== (debt.monthly_amount || items[0].amount);
  const restPending = pending.filter((it) => it.id !== next?.id);
  const shown = tab === "pending" ? restPending : paid;
  const done = paid.length;
  const pct = Math.round((done / items.length) * 100);

  return (
    <div>
      <div className="mb-4 flex items-end justify-between gap-3">
        <div>
          <h2 className="font-bold">{t("debts.installments")}</h2>
          <p className="mt-1 text-xs text-white/40">{t("debts.progressPaid", { paid: done, total: items.length })}</p>
        </div>
        <div className="text-sm font-black text-gold-200">{pct}%</div>
      </div>
      <div className="mb-5 flex gap-1">
        {items.map((it) => (
          <div
            key={it.id}
            className={`h-1.5 flex-1 rounded-full ${it.status === "paid" ? "bg-emerald-400/80" : it.status === "overdue" ? "bg-rose-400/80" : "bg-white/12"}`}
          />
        ))}
      </div>

      {next && (
        <NextCard
          it={next}
          last={lastIsRemainder && next.id === last.id}
          onPay={() => onPay(next)}
          onMarkPrior={() => onMarkPrior(next)}
        />
      )}

      <div className="mt-5 flex rounded-2xl bg-white/5 p-1">
        <TabBtn active={tab === "pending"} onClick={() => setTab("pending")} count={pending.length} label={t("debts.tabPending")} />
        <TabBtn active={tab === "paid"} onClick={() => setTab("paid")} count={paid.length} label={t("debts.tabPaid")} />
      </div>

      <div className="mt-3 space-y-2">
        {shown.length === 0 ? (
          <div className="rounded-[20px] border border-white/8 bg-white/[0.03] px-4 py-8 text-center text-sm text-white/40">
            {tab === "pending" ? t("debts.noPending") : t("debts.noPaid")}
          </div>
        ) : (
          shown.map((it, i) => (
            <motion.div
              key={it.id}
              initial={{ opacity: 0, y: 8 }}
              animate={{ opacity: 1, y: 0 }}
              transition={{ delay: i * 0.03 }}
            >
              <Row
                it={it}
                last={lastIsRemainder && it.id === last.id}
                onPay={() => onPay(it)}
                onMarkPrior={() => onMarkPrior(it)}
                onUndoPrior={() => onUndoPrior(it)}
              />
            </motion.div>
          ))
        )}
      </div>
    </div>
  );
}

function NextCard({
  it,
  last,
  onPay,
  onMarkPrior,
}: {
  it: Installment;
  last: boolean;
  onPay: () => void;
  onMarkPrior: () => void;
}) {
  const { t, locale } = useI18n();
  const days = daysUntil(it.due_date);
  const overdue = days < 0 || it.status === "overdue";

  return (
    <div
      className={`relative overflow-hidden rounded-[28px] border p-5 ${
        overdue ? "border-rose-400/30 bg-gradient-to-bl from-rose-500/15 to-[#141018]" : "border-gold-400/25 bg-gradient-to-bl from-[#2a2212] to-[#121016]"
      }`}
    >
      <div className="flex items-start justify-between gap-3">
        <div className="flex items-center gap-2 text-xs font-bold uppercase tracking-wide">
          {overdue ? <CircleAlert className="h-4 w-4 text-rose-300" /> : <CalendarClock className="h-4 w-4 text-gold-200" />}
          <span className={overdue ? "text-rose-300" : "text-gold-200"}>{t("debts.next")}</span>
        </div>
        <DueChip days={days} overdue={overdue} />
      </div>
      <div className="mt-4 text-2xl font-black md:text-3xl">{faDate(it.due_date, false, locale)}</div>
      <div className={`mt-2 text-3xl font-black ${overdue ? "text-rose-300" : "text-gold-200"}`}>{toman(it.amount, true, locale)}</div>
      {last && <div className="mt-2 text-xs text-gold-200/80">{t("debts.lastInst")}</div>}
      <div className="mt-5 flex flex-wrap gap-2">
        <Btn className="flex-1" onClick={onPay}>
          {t("debts.payInst")}
        </Btn>
        <Btn kind="ghost" onClick={onMarkPrior}>
          {t("debts.markPrior")}
        </Btn>
      </div>
    </div>
  );
}

function Row({
  it,
  last,
  onPay,
  onMarkPrior,
  onUndoPrior,
}: {
  it: Installment;
  last: boolean;
  onPay: () => void;
  onMarkPrior: () => void;
  onUndoPrior: () => void;
}) {
  const { t, locale } = useI18n();
  const prior = isPrior(it);
  const days = daysUntil(it.due_date);
  const overdue = isOpen(it) && (days < 0 || it.status === "overdue");

  return (
    <div className={`glass flex flex-wrap items-center justify-between gap-3 rounded-[20px] px-4 py-3 ${prior ? "opacity-75" : ""}`}>
      <div className="min-w-0">
        <div className="flex flex-wrap items-center gap-2 font-medium">
          <span>{faDate(it.due_date, false, locale)}</span>
          {prior && <span className="rounded-full bg-emerald-400/15 px-2 py-0.5 text-[10px] text-emerald-300">{t("form.debt.priorBadge")}</span>}
          {last && !prior && <span className="rounded-full bg-gold-400/15 px-2 py-0.5 text-[10px] text-gold-200">{t("debts.lastInst")}</span>}
          {overdue && <span className="rounded-full bg-rose-400/15 px-2 py-0.5 text-[10px] text-rose-300">{t("debts.overdue")}</span>}
        </div>
        <div className="mt-0.5 text-xs text-white/40">
          {it.status === "paid"
            ? `${prior ? t("form.debt.priorBadge") : t("debts.paid")}${it.paid_at ? ` · ${faDate(it.paid_at, true, locale)}` : ""}`
            : relativeDue(days, t)}
        </div>
      </div>
      <div className="flex flex-wrap items-center gap-2">
        <div className={`font-bold ${it.status === "paid" ? "text-emerald-300" : overdue ? "text-rose-300" : "text-white"}`}>
          {toman(it.amount, true, locale)}
        </div>
        {isOpen(it) && (
          <>
            <Btn className="py-2 text-xs" onClick={onPay}>
              {t("debts.payInst")}
            </Btn>
            <Btn kind="ghost" className="py-2 text-xs" onClick={onMarkPrior}>
              {t("debts.markPrior")}
            </Btn>
          </>
        )}
        {prior && (
          <Btn kind="ghost" className="py-2 text-xs" onClick={onUndoPrior}>
            {t("debts.undoPrior")}
          </Btn>
        )}
        {it.status === "paid" && !prior && <Check className="h-4 w-4 text-emerald-300" />}
      </div>
    </div>
  );
}

function TabBtn({ active, onClick, count, label }: { active: boolean; onClick: () => void; count: number; label: string }) {
  return (
    <button
      type="button"
      onClick={onClick}
      className={`flex flex-1 items-center justify-center gap-2 rounded-xl px-3 py-2.5 text-sm font-bold transition ${
        active ? "bg-white/12 text-white shadow-sm" : "text-white/45 hover:text-white/70"
      }`}
    >
      {label}
      <span className={`rounded-full px-1.5 py-0.5 text-[10px] ${active ? "bg-gold-400/20 text-gold-200" : "bg-white/8 text-white/40"}`}>
        {count}
      </span>
    </button>
  );
}

function DueChip({ days, overdue }: { days: number; overdue: boolean }) {
  const { t } = useI18n();
  return (
    <span className={`rounded-full px-2.5 py-1 text-[11px] font-bold ${overdue ? "bg-rose-400/15 text-rose-300" : "bg-gold-400/15 text-gold-200"}`}>
      {relativeDue(days, t)}
    </span>
  );
}

function relativeDue(days: number, t: (key: string, vars?: Record<string, string | number>) => string) {
  if (days < 0) return t("debts.overdueBy", { n: Math.abs(days) });
  if (days === 0) return t("debts.dueToday");
  if (days === 1) return t("debts.dueTomorrow");
  return t("debts.dueIn", { n: days });
}
