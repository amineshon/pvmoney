"use client";

import { useEffect, useState, type ReactNode } from "react";
import { motion } from "framer-motion";
import { api } from "@/lib/api";
import { Dashboard } from "@/lib/types";
import { greeting, toman, progress, faDate, formatUSD, formatEUR } from "@/lib/format";
import { AccountCard } from "@/components/AccountCard";
import { ProjectCard } from "@/components/ProjectCard";
import { TransactionRow } from "@/components/TransactionRow";
import { Btn, Empty } from "@/components/ui";
import { AccountForm, ProjectForm } from "@/components/forms";
import Link from "next/link";
import { ArrowUpRight, Sparkles, WalletCards, Gem, HandCoins } from "lucide-react";
import { useI18n } from "@/lib/i18n";
import { FxHint } from "@/components/FxHint";
import { convertToman, liveAssetValue, type Rates } from "@/lib/rates";

export default function HomePage() {
  const { t, locale } = useI18n();
  const [data, setData] = useState<Dashboard | null>(null);
  const [rates, setRates] = useState<Rates | null>(null);
  const [err, setErr] = useState("");
  const [accountOpen, setAccountOpen] = useState(false);
  const [projectOpen, setProjectOpen] = useState(false);

  function load() {
    api
      .dashboard()
      .then(setData)
      .catch((e) => setErr(e.message));
    api
      .rates()
      .then(setRates)
      .catch(() => setRates(null));
  }

  useEffect(() => {
    load();
  }, []);

  if (err) {
    return <Empty title={t("err.offline.title")} text={t("err.offline.text")} />;
  }
  if (!data) {
    return <div className="animate-pulse text-white/40">{t("common.loading")}</div>;
  }

  const net = data.monthly_income - data.monthly_expense;

  return (
    <div className="mx-auto max-w-6xl">
      <div className="mb-8 flex flex-wrap items-end justify-between gap-4">
        <div>
          <div className="text-sm text-gold-200/70">{greeting(locale)}</div>
          <h1 className="mt-1 text-3xl font-extrabold md:text-4xl">{t("dash.title")}</h1>
        </div>
        <div className="flex gap-2">
          <Btn kind="ghost" onClick={() => setAccountOpen(true)}>
            {t("dash.addAccount")}
          </Btn>
          <Btn onClick={() => setProjectOpen(true)}>{t("dash.addProject")}</Btn>
        </div>
      </div>

      <motion.section
        initial={{ opacity: 0, y: 16 }}
        animate={{ opacity: 1, y: 0 }}
        className="relative overflow-hidden rounded-[32px] border border-gold-400/20 bg-gradient-to-bl from-[#1b160c] via-ink-900 to-[#0c1210] p-6 shadow-glow md:p-10"
      >
        <Sparkles className="absolute start-8 top-8 h-6 w-6 text-gold-400/40" />
        <div className="text-sm text-white/50">{t("dash.netWorth")}</div>
        <div className="gold-text mt-2 text-4xl font-black tracking-tight md:text-6xl">{toman(data.net_worth, true, locale)}</div>
        {rates && (
          <div className="mt-2 flex flex-wrap gap-x-4 gap-y-1 text-sm text-white/50">
            <span>≈ {formatUSD(convertToman(data.net_worth, rates).usd, locale)}</span>
            <span>≈ {formatEUR(convertToman(data.net_worth, rates).eur, locale)}</span>
          </div>
        )}
        <div className="mt-6 grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
          <HeroStat icon={<WalletCards className="h-4 w-4" />} label={t("dash.liquid")} value={toman(data.liquid, true, locale)} />
          <HeroStat icon={<Gem className="h-4 w-4" />} label={t("dash.assets")} value={toman(data.assets_total, true, locale)} />
          <HeroStat icon={<HandCoins className="h-4 w-4" />} label={t("dash.debts")} value={toman(data.debts_remaining || 0, true, locale)} tone="down" />
          <HeroStat
            icon={<ArrowUpRight className="h-4 w-4" />}
            label={t("dash.monthNet")}
            value={toman(net, true, locale)}
            tone={net >= 0 ? "up" : "down"}
          />
        </div>
      </motion.section>

      <div className="mt-6 grid gap-4 md:grid-cols-2">
        <div className="glass rounded-[28px] p-5">
          <div className="text-xs text-white/40">{t("dash.incomeMonth")}</div>
          <div className="mt-1 text-2xl font-bold text-emerald-300">{toman(data.monthly_income, true, locale)}</div>
        </div>
        <div className="glass rounded-[28px] p-5">
          <div className="text-xs text-white/40">{t("dash.expenseMonth")}</div>
          <div className="mt-1 text-2xl font-bold text-rose-300">{toman(data.monthly_expense, true, locale)}</div>
        </div>
      </div>

      <section className="mt-10">
        <Header title={t("dash.accounts")} href="/accounts" />
        {data.accounts.length === 0 ? (
          <Empty title={t("dash.noAccounts.title")} text={t("dash.noAccounts.text")} action={<Btn onClick={() => setAccountOpen(true)}>{t("dash.noAccounts.cta")}</Btn>} />
        ) : (
          <div className="grid gap-4 sm:grid-cols-2 xl:grid-cols-3">
            {data.accounts.slice(0, 3).map((a) => (
              <AccountCard key={a.id} account={a} />
            ))}
          </div>
        )}
      </section>

      <section className="mt-10">
        <Header title={t("dash.assetsTitle")} href="/assets" />
        {!data.assets?.length ? (
          <Empty title={t("dash.noAssets.title")} text={t("dash.noAssets.text")} />
        ) : (
          <div className="grid gap-3 sm:grid-cols-2 xl:grid-cols-3">
            {data.assets.slice(0, 3).map((a) => (
              <Link key={a.id} href="/assets" className="glass rounded-[24px] p-4">
                <div className="mt-1 font-bold">{a.name}</div>
                <div className="mt-2 text-gold-200">{toman(liveAssetValue(a.type, a.quantity, a.value, rates), true, locale)}</div>
                <FxHint toman={liveAssetValue(a.type, a.quantity, a.value, rates)} rates={rates} />
              </Link>
            ))}
          </div>
        )}
      </section>

      <section className="mt-10">
        <Header title={t("dash.debtsTitle")} href="/debts" />
        {!data.upcoming?.length ? (
          <Empty title={t("dash.noDebts")} text={t("debts.empty.text")} />
        ) : (
          <div className="grid gap-3 sm:grid-cols-2 xl:grid-cols-3">
            {data.upcoming.slice(0, 6).map((it) => (
              <Link key={it.id} href={`/debts/${it.debt_id}`} className="glass rounded-[24px] p-4">
                <div className="text-xs text-white/40">{it.creditor || t("dash.due")}</div>
                <div className="mt-1 font-bold">{it.debt_name}</div>
                <div className="mt-2 flex items-end justify-between">
                  <div className="text-rose-300">{toman(it.amount, true, locale)}</div>
                  <div className="text-xs text-white/40">{faDate(it.due_date, false, locale)}</div>
                </div>
              </Link>
            ))}
          </div>
        )}
      </section>

      <section className="mt-10">
        <Header title={t("dash.projects")} href="/projects" />
        {data.projects.length === 0 ? (
          <Empty title={t("dash.noProjects.title")} text={t("dash.noProjects.text")} action={<Btn onClick={() => setProjectOpen(true)}>{t("dash.noProjects.cta")}</Btn>} />
        ) : (
          <div className="grid gap-4 md:grid-cols-2">
            {data.projects.slice(0, 4).map((p) => (
              <ProjectCard key={p.id} project={p} />
            ))}
          </div>
        )}
      </section>

      <section className="mt-10 grid gap-6 lg:grid-cols-5">
        <div className="glass rounded-[28px] p-5 lg:col-span-3">
          <Header title={t("dash.recent")} href="/transactions" compact />
          {data.recent.length === 0 ? (
            <p className="py-8 text-center text-sm text-white/40">{t("dash.noTx")}</p>
          ) : (
            data.recent.map((tx) => <TransactionRow key={tx.id} tx={tx} />)
          )}
        </div>
        <div className="glass rounded-[28px] p-5 lg:col-span-2">
          <h3 className="mb-4 font-bold">{t("dash.projectSpend")}</h3>
          <div className="space-y-4">
            {data.projects.slice(0, 5).map((p) => {
              const pct = progress(p.current_amount, p.target_amount);
              return (
                <div key={p.id}>
                  <div className="mb-1 flex justify-between text-sm">
                    <span>{p.name}</span>
                    <span className="text-white/40">{pct}%</span>
                  </div>
                  <div className="h-2 overflow-hidden rounded-full bg-white/5">
                    <div className="h-full rounded-full" style={{ width: `${pct}%`, background: p.color }} />
                  </div>
                </div>
              );
            })}
            {data.projects.length === 0 && <p className="text-sm text-white/40">{t("dash.noProjectList")}</p>}
          </div>
        </div>
      </section>

      {accountOpen && <AccountForm onClose={() => setAccountOpen(false)} onSaved={load} />}
      {projectOpen && <ProjectForm onClose={() => setProjectOpen(false)} onSaved={load} />}
    </div>
  );
}

function Header({ title, href, compact }: { title: string; href: string; compact?: boolean }) {
  const { t } = useI18n();
  return (
    <div className={`flex items-center justify-between ${compact ? "mb-3" : "mb-4"}`}>
      <h2 className="text-xl font-bold">{title}</h2>
      <Link href={href} className="text-sm text-gold-200/80 hover:text-gold-200">
        {t("common.all")}
      </Link>
    </div>
  );
}

function HeroStat({
  icon,
  label,
  value,
  tone,
}: {
  icon: ReactNode;
  label: string;
  value: string;
  tone?: "up" | "down";
}) {
  return (
    <div className="rounded-2xl bg-white/5 px-4 py-3">
      <div className="flex items-center gap-2 text-xs text-white/45">
        {icon}
        {label}
      </div>
      <div className={`mt-1 font-bold ${tone === "down" ? "text-rose-300" : tone === "up" ? "text-emerald-300" : ""}`}>{value}</div>
    </div>
  );
}
