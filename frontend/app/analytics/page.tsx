"use client";

import { useEffect, useState } from "react";
import {
  Area,
  AreaChart,
  Bar,
  BarChart,
  CartesianGrid,
  ResponsiveContainer,
  Tooltip,
  XAxis,
  YAxis,
} from "recharts";
import { api } from "@/lib/api";
import { Dashboard } from "@/lib/types";
import { faMonth, formatMoney, toman } from "@/lib/format";
import { useI18n } from "@/lib/i18n";
import { calcNetWorth, type Rates } from "@/lib/rates";

const COLORS = ["#e0c36a", "#3ee0a2", "#fb7185", "#7dd3fc", "#a78bfa", "#f9a8d4", "#fbbf24", "#34d399"];

export default function AnalyticsPage() {
  const { t, label, locale } = useI18n();
  const [data, setData] = useState<Dashboard | null>(null);
  const [rates, setRates] = useState<Rates | null>(null);
  useEffect(() => {
    api.dashboard().then(setData);
    api.rates().then(setRates).catch(() => setRates(null));
  }, []);
  if (!data) return <div className="text-white/40">{t("analytics.loading")}</div>;
  const { netWorth } = calcNetWorth(data.liquid, data.assets || [], data.debts_remaining || 0, rates);

  const monthly = data.monthly.map((m) => ({
    ...m,
    label: faMonth(m.month, locale),
    incomeM: m.income,
    expenseM: m.expense,
  }));
  const cash = data.cashflow.map((c) => ({
    ...c,
    label: new Date(c.date).toLocaleDateString(locale === "fa" ? "fa-IR" : locale === "de" ? "de-DE" : "en-US", {
      day: "numeric",
      month: "short",
    }),
  }));

  return (
    <div className="mx-auto max-w-6xl">
      <h1 className="mb-8 text-3xl font-extrabold">{t("analytics.title")}</h1>
      <div className="grid gap-4 md:grid-cols-3">
        <Stat label={t("dash.netWorth")} value={toman(netWorth, true, locale)} />
        <Stat label={t("dash.incomeMonth")} value={toman(data.monthly_income, true, locale)} />
        <Stat label={t("dash.expenseMonth")} value={toman(data.monthly_expense, true, locale)} />
      </div>

      <div className="glass mt-6 rounded-[28px] p-5">
        <h2 className="mb-4 font-bold">{t("analytics.six")}</h2>
        <div className="h-72">
          <ResponsiveContainer width="100%" height="100%">
            <BarChart data={monthly}>
              <CartesianGrid stroke="rgba(255,255,255,0.06)" vertical={false} />
              <XAxis dataKey="label" stroke="#9a9384" fontSize={12} />
              <YAxis stroke="#9a9384" fontSize={11} tickFormatter={(v) => formatMoney(v, true, locale)} />
              <Tooltip
                contentStyle={{ background: "#12151e", border: "1px solid rgba(224,195,106,.2)", borderRadius: 16 }}
                formatter={(v) => toman(Number(v), true, locale)}
              />
              <Bar dataKey="incomeM" name={t("type.income")} fill="#3ee0a2" radius={[8, 8, 0, 0]} />
              <Bar dataKey="expenseM" name={t("type.expense")} fill="#fb7185" radius={[8, 8, 0, 0]} />
            </BarChart>
          </ResponsiveContainer>
        </div>
      </div>

      <div className="mt-6 grid gap-6 lg:grid-cols-2">
        <div className="glass rounded-[28px] p-5">
          <h2 className="mb-4 font-bold">{t("analytics.thirty")}</h2>
          <div className="h-64">
            <ResponsiveContainer width="100%" height="100%">
              <AreaChart data={cash}>
                <defs>
                  <linearGradient id="g1" x1="0" y1="0" x2="0" y2="1">
                    <stop offset="0%" stopColor="#3ee0a2" stopOpacity={0.4} />
                    <stop offset="100%" stopColor="#3ee0a2" stopOpacity={0} />
                  </linearGradient>
                </defs>
                <XAxis dataKey="label" stroke="#9a9384" fontSize={11} />
                <YAxis hide />
                <Tooltip
                  contentStyle={{ background: "#12151e", border: "1px solid rgba(224,195,106,.2)", borderRadius: 16 }}
                  formatter={(v) => toman(Number(v), true, locale)}
                />
                <Area type="monotone" dataKey="income" name={t("type.income")} stroke="#3ee0a2" fill="url(#g1)" />
                <Area type="monotone" dataKey="expense" name={t("type.expense")} stroke="#fb7185" fill="transparent" />
              </AreaChart>
            </ResponsiveContainer>
          </div>
        </div>

        <div className="glass rounded-[28px] p-5">
          <h2 className="mb-4 font-bold">{t("analytics.byCat")}</h2>
          {data.category_spend.length === 0 ? (
            <p className="py-16 text-center text-sm text-white/40">{t("analytics.noCat")}</p>
          ) : (
            <div className="space-y-4">
              {data.category_spend.map((c, i) => {
                const max = data.category_spend[0]?.amount || 1;
                return (
                  <div key={c.category}>
                    <div className="mb-1 flex justify-between text-sm">
                      <span>{label(c.category)}</span>
                      <span className="text-white/50">{toman(c.amount, true, locale)}</span>
                    </div>
                    <div className="h-2 rounded-full bg-white/5">
                      <div
                        className="h-full rounded-full"
                        style={{ width: `${(c.amount / max) * 100}%`, background: COLORS[i % COLORS.length] }}
                      />
                    </div>
                  </div>
                );
              })}
            </div>
          )}
        </div>
      </div>
    </div>
  );
}

function Stat({ label, value }: { label: string; value: string }) {
  return (
    <div className="glass rounded-[24px] p-5">
      <div className="text-xs text-white/40">{label}</div>
      <div className="mt-1 text-xl font-bold">{value}</div>
    </div>
  );
}
