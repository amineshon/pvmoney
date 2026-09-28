"use client";

import { useEffect, useState } from "react";
import {
  Area,
  AreaChart,
  Bar,
  BarChart,
  CartesianGrid,
  ComposedChart,
  Line,
  ResponsiveContainer,
  Tooltip,
  XAxis,
  YAxis,
} from "recharts";
import { api } from "@/lib/api";
import { Dashboard } from "@/lib/types";
import { faDate, faMonth, formatMoney, toman } from "@/lib/format";
import { useI18n } from "@/lib/i18n";
import { calcNetWorth, type Rates } from "@/lib/rates";

const COLORS = ["#e0c36a", "#3ee0a2", "#fb7185", "#7dd3fc", "#a78bfa", "#f9a8d4", "#fbbf24", "#34d399"];

function movingAvg(values: number[], window = 7) {
  return values.map((_, i) => {
    const from = Math.max(0, i - window + 1);
    const slice = values.slice(from, i + 1);
    return Math.round(slice.reduce((s, n) => s + n, 0) / slice.length);
  });
}

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

  const monthly = data.monthly.map((m, i, arr) => {
    const prev = i > 0 ? arr[i - 1].income : 0;
    const mom = prev > 0 ? Math.round(((m.income - prev) / prev) * 100) : 0;
    return {
      ...m,
      label: faMonth(m.month, locale),
      incomeM: m.income,
      expenseM: m.expense,
      mom,
    };
  });
  const incomeCumMonth = monthly.reduce<number[]>((acc, m) => {
    acc.push((acc[acc.length - 1] || 0) + m.incomeM);
    return acc;
  }, []);
  const monthlyGrowth = monthly.map((m, i) => ({ ...m, cum: incomeCumMonth[i] }));

  const expenses = data.cashflow.map((c) => c.expense);
  const avg = movingAvg(expenses, 7);
  let run = 0;
  const daily = data.cashflow.map((c, i) => {
    run += c.income;
    return {
      ...c,
      label: faDate(c.date, false, locale),
      tick: i % 7 === 0 ? faDate(c.date, false, locale) : "",
      avg: avg[i],
      cumIncome: run,
    };
  });

  const lastMom = monthlyGrowth[monthlyGrowth.length - 1]?.mom || 0;
  const lastIncome = monthlyGrowth[monthlyGrowth.length - 1]?.incomeM || 0;
  const prevIncome = monthlyGrowth[monthlyGrowth.length - 2]?.incomeM || 0;

  return (
    <div className="mx-auto max-w-6xl">
      <h1 className="mb-8 text-3xl font-extrabold">{t("analytics.title")}</h1>
      <div className="grid gap-4 md:grid-cols-3">
        <Stat label={t("dash.netWorth")} value={toman(netWorth, true, locale)} />
        <Stat label={t("dash.incomeMonth")} value={toman(data.monthly_income, true, locale)} hint={prevIncome ? `${lastMom >= 0 ? "+" : ""}${lastMom}% ${t("analytics.mom")}` : undefined} tone={lastMom >= 0 ? "up" : "down"} />
        <Stat label={t("dash.expenseMonth")} value={toman(data.monthly_expense, true, locale)} />
      </div>

      <div className="glass mt-6 rounded-[28px] p-5">
        <div className="mb-4 flex flex-wrap items-end justify-between gap-2">
          <div>
            <h2 className="font-bold">{t("analytics.dailySpend")}</h2>
            <p className="mt-1 text-xs text-white/40">{t("analytics.ninety")}</p>
          </div>
          <div className="text-xs text-white/40">{t("analytics.avg7")}</div>
        </div>
        <div className="h-80">
          <ResponsiveContainer width="100%" height="100%">
            <ComposedChart data={daily} barCategoryGap={1}>
              <CartesianGrid stroke="rgba(255,255,255,0.06)" vertical={false} />
              <XAxis dataKey="label" stroke="#9a9384" fontSize={11} minTickGap={28} interval="preserveStartEnd" />
              <YAxis stroke="#9a9384" fontSize={11} width={56} tickFormatter={(v) => formatMoney(v, true, locale)} />
              <Tooltip
                contentStyle={{ background: "#12151e", border: "1px solid rgba(224,195,106,.2)", borderRadius: 16 }}
                labelFormatter={(_, pts) => (pts?.[0]?.payload?.label as string) || ""}
                formatter={(v, name) => [toman(Number(v), true, locale), String(name)]}
              />
              <Bar dataKey="expense" name={t("type.expense")} fill="#fb7185" radius={[3, 3, 0, 0]} maxBarSize={10} />
              <Line type="monotone" dataKey="avg" name={t("analytics.avg7")} stroke="#e0c36a" strokeWidth={2} dot={false} />
            </ComposedChart>
          </ResponsiveContainer>
        </div>
      </div>

      <div className="mt-6 grid gap-6 lg:grid-cols-2">
        <div className="glass rounded-[28px] p-5">
          <div className="mb-4">
            <h2 className="font-bold">{t("analytics.incomeGrowth")}</h2>
            <p className="mt-1 text-xs text-white/40">
              {t("analytics.twelve")}
              {lastIncome ? ` · ${toman(lastIncome, true, locale)}` : ""}
            </p>
          </div>
          <div className="h-72">
            <ResponsiveContainer width="100%" height="100%">
              <ComposedChart data={monthlyGrowth}>
                <defs>
                  <linearGradient id="incFill" x1="0" y1="0" x2="0" y2="1">
                    <stop offset="0%" stopColor="#3ee0a2" stopOpacity={0.35} />
                    <stop offset="100%" stopColor="#3ee0a2" stopOpacity={0} />
                  </linearGradient>
                </defs>
                <CartesianGrid stroke="rgba(255,255,255,0.06)" vertical={false} />
                <XAxis dataKey="label" stroke="#9a9384" fontSize={11} />
                <YAxis stroke="#9a9384" fontSize={11} width={56} tickFormatter={(v) => formatMoney(v, true, locale)} />
                <Tooltip
                  contentStyle={{ background: "#12151e", border: "1px solid rgba(224,195,106,.2)", borderRadius: 16 }}
                  formatter={(v, name) => [toman(Number(v), true, locale), String(name)]}
                />
                <Bar dataKey="incomeM" name={t("type.income")} fill="#3ee0a2" radius={[8, 8, 0, 0]} />
                <Line type="monotone" dataKey="cum" name={t("analytics.incomeCum")} stroke="#e0c36a" strokeWidth={2.5} dot={false} />
              </ComposedChart>
            </ResponsiveContainer>
          </div>
        </div>

        <div className="glass rounded-[28px] p-5">
          <h2 className="mb-4 font-bold">{t("analytics.twelve")}</h2>
          <div className="h-72">
            <ResponsiveContainer width="100%" height="100%">
              <BarChart data={monthlyGrowth}>
                <CartesianGrid stroke="rgba(255,255,255,0.06)" vertical={false} />
                <XAxis dataKey="label" stroke="#9a9384" fontSize={11} />
                <YAxis stroke="#9a9384" fontSize={11} width={56} tickFormatter={(v) => formatMoney(v, true, locale)} />
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
      </div>

      <div className="mt-6 grid gap-6 lg:grid-cols-2">
        <div className="glass rounded-[28px] p-5">
          <h2 className="mb-4 font-bold">{t("analytics.incomeCum")}</h2>
          <div className="h-64">
            <ResponsiveContainer width="100%" height="100%">
              <AreaChart data={daily}>
                <defs>
                  <linearGradient id="gCum" x1="0" y1="0" x2="0" y2="1">
                    <stop offset="0%" stopColor="#3ee0a2" stopOpacity={0.45} />
                    <stop offset="100%" stopColor="#3ee0a2" stopOpacity={0} />
                  </linearGradient>
                </defs>
                <XAxis dataKey="label" stroke="#9a9384" fontSize={11} minTickGap={28} interval="preserveStartEnd" />
                <YAxis hide />
                <Tooltip
                  contentStyle={{ background: "#12151e", border: "1px solid rgba(224,195,106,.2)", borderRadius: 16 }}
                  labelFormatter={(_, pts) => (pts?.[0]?.payload?.label as string) || ""}
                  formatter={(v) => toman(Number(v), true, locale)}
                />
                <Area type="monotone" dataKey="cumIncome" name={t("analytics.incomeCum")} stroke="#3ee0a2" fill="url(#gCum)" />
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

function Stat({
  label,
  value,
  hint,
  tone,
}: {
  label: string;
  value: string;
  hint?: string;
  tone?: "up" | "down";
}) {
  return (
    <div className="glass rounded-[24px] p-5">
      <div className="text-xs text-white/40">{label}</div>
      <div className={`mt-1 text-xl font-bold ${tone === "down" ? "text-rose-300" : tone === "up" ? "text-emerald-300" : ""}`}>{value}</div>
      {hint && <div className={`mt-1 text-xs ${tone === "down" ? "text-rose-300/70" : "text-emerald-300/70"}`}>{hint}</div>}
    </div>
  );
}
