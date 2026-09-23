"use client";

import { useEffect, useState } from "react";
import { useParams, useRouter } from "next/navigation";
import { api } from "@/lib/api";
import { Account, Debt } from "@/lib/types";
import { faDate, toman } from "@/lib/format";
import { Btn } from "@/components/ui";
import { DebtForm, PayDebtForm } from "@/components/forms";
import { InstallmentTimeline } from "@/components/InstallmentTimeline";
import { apiError, useI18n } from "@/lib/i18n";

export default function DebtDetailPage() {
  const { id } = useParams<{ id: string }>();
  const router = useRouter();
  const { t, label, locale } = useI18n();
  const [debt, setDebt] = useState<Debt | null>(null);
  const [accounts, setAccounts] = useState<Account[]>([]);
  const [edit, setEdit] = useState(false);
  const [pay, setPay] = useState<{ instId?: string; amount: number } | null>(null);

  async function load() {
    const [d, a] = await Promise.all([api.debt(id), api.accounts()]);
    setDebt(d);
    setAccounts(a);
  }

  useEffect(() => {
    load();
  }, [id]);

  if (!debt) return <div className="text-white/40">{t("common.loading")}</div>;

  async function remove() {
    if (!confirm(t("debts.deleteAsk"))) return;
    await api.deleteDebt(debt!.id);
    router.push("/debts");
  }

  return (
    <div className="mx-auto max-w-3xl">
      <div className="glass relative overflow-hidden rounded-[32px] p-8">
        <div className="absolute -end-10 -top-10 h-32 w-32 rounded-full opacity-20" style={{ background: debt.color }} />
        <div className="relative">
          <div className="text-sm text-white/40">{label(debt.type)}</div>
          <h1 className="mt-1 text-3xl font-black">{debt.name}</h1>
          {debt.creditor && <p className="mt-2 text-white/55">{debt.creditor}</p>}
          {debt.notes && <p className="mt-2 text-sm text-white/40">{debt.notes}</p>}
          <div className="mt-8 flex items-end justify-between">
            <div>
              <div className="text-xs text-white/40">{t("common.remaining")}</div>
              <div className="text-3xl font-bold text-rose-300">{toman(debt.remaining, true, locale)}</div>
            </div>
            <div className="text-end">
              <div className="text-xs text-white/40">{t("common.total")}</div>
              <div className="text-2xl font-bold">{toman(debt.total_amount, true, locale)}</div>
            </div>
          </div>
          {(debt.commission_amount || 0) > 0 && (
            <div className="mt-5 grid gap-2 rounded-2xl border border-gold-400/20 bg-gold-400/8 px-4 py-3 text-sm sm:grid-cols-3">
              <div>
                <div className="text-[11px] text-white/40">{t("debts.commission")}</div>
                <div className="font-bold">{toman(debt.commission_amount || 0, true, locale)}</div>
              </div>
              <div>
                <div className="text-[11px] text-white/40">{t("debts.netReceived")}</div>
                <div className="font-bold text-gold-200">{toman(debt.net_received || 0, true, locale)}</div>
              </div>
              <div>
                <div className="text-[11px] text-white/40">{t("debts.totalCost")}</div>
                <div className="font-extrabold">{toman(debt.total_cost || 0, true, locale)}</div>
              </div>
            </div>
          )}
          {debt.has_schedule && debt.start_date && debt.end_date && (
            <div className="mt-4 text-sm text-white/45">
              {t("debts.schedule")}: {faDate(debt.start_date, false, locale)} {t("debts.until")} {faDate(debt.end_date, false, locale)}
              {debt.monthly_amount > 0 && ` · ${toman(debt.monthly_amount, true, locale)}`}
            </div>
          )}
          <div className="mt-8 flex flex-wrap gap-2">
            {debt.status !== "paid" && (
              <Btn onClick={() => setPay({ amount: debt.remaining })}>{t("debts.payRemaining")}</Btn>
            )}
            <Btn kind="ghost" onClick={() => setEdit(true)}>
              {t("common.edit")}
            </Btn>
            <Btn kind="danger" onClick={remove}>
              {t("common.delete")}
            </Btn>
          </div>
        </div>
      </div>

      <div className="mt-8">
        <InstallmentTimeline
          debt={debt}
          onPay={(it) => setPay({ instId: it.id, amount: it.amount })}
          onMarkPrior={async (it) => {
            if (!confirm(t("debts.priorAsk"))) return;
            await api.payInstallment(debt.id, it.id, { amount: it.amount, already_paid: true });
            load();
          }}
          onUndoPrior={async (it) => {
            await api.undoPriorInstallment(debt.id, it.id);
            load();
          }}
        />
      </div>

      {edit && <DebtForm initial={debt} onClose={() => setEdit(false)} onSaved={load} />}
      {pay && (
        <PayDebtForm
          accounts={accounts}
          amountHint={pay.amount}
          onClose={() => setPay(null)}
          onSubmitPay={async (body) => {
            if (pay.instId) await api.payInstallment(debt.id, pay.instId, body);
            else await api.payDebt(debt.id, { ...body, already_paid: body.already_paid });
            await load();
          }}
        />
      )}
    </div>
  );
}
