"use client";

import { useEffect, useState } from "react";
import Link from "next/link";
import { api } from "@/lib/api";
import { Debt, TelegramStatus } from "@/lib/types";
import { faDate, toman } from "@/lib/format";
import { Btn, Empty } from "@/components/ui";
import { DebtForm } from "@/components/forms";
import { apiError, useI18n } from "@/lib/i18n";

export default function DebtsPage() {
  const { t, label, locale } = useI18n();
  const [items, setItems] = useState<Debt[] | null>(null);
  const [tg, setTg] = useState<TelegramStatus | null>(null);
  const [open, setOpen] = useState(false);
  const [edit, setEdit] = useState<Debt | undefined>();
  const [tgMsg, setTgMsg] = useState("");

  function load() {
    Promise.all([api.debts(), api.telegram()]).then(([d, s]) => {
      setItems(d);
      setTg(s);
    });
  }

  useEffect(() => {
    load();
  }, []);

  async function sendTest() {
    setTgMsg("");
    try {
      await api.telegramTest();
      setTgMsg(t("tg.sent"));
    } catch (e) {
      setTgMsg(apiError(e instanceof Error ? e.message : "", t));
    }
  }

  if (!items) return <div className="animate-pulse text-white/40">{t("debts.loading")}</div>;
  const remaining = items.filter((d) => d.status === "active").reduce((s, d) => s + d.remaining, 0);
  const botLink = tg?.bot_username ? `https://t.me/${tg.bot_username}` : "https://t.me";

  return (
    <div className="mx-auto max-w-6xl">
      <div className="mb-8 flex flex-wrap items-end justify-between gap-4">
        <div>
          <h1 className="text-3xl font-extrabold">{t("debts.title")}</h1>
          <p className="mt-2 text-white/45">{t("debts.sub")}</p>
          <p className="mt-1 text-rose-300">
            {t("debts.remaining")} {toman(remaining, true, locale)}
          </p>
        </div>
        <Btn
          onClick={() => {
            setEdit(undefined);
            setOpen(true);
          }}
        >
          {t("debts.add")}
        </Btn>
      </div>

      <div className="glass mb-8 rounded-[28px] p-5">
        <div className="flex flex-wrap items-start justify-between gap-4">
          <div>
            <div className="text-xs text-white/40">{t("tg.title")}</div>
            <div className={`mt-1 font-bold ${tg?.telegram_connected ? "text-emerald-300" : "text-gold-200"}`}>
              {tg?.telegram_connected ? t("tg.connected") : t("tg.waiting")}
            </div>
            {tg?.bot_username && (
              <div className="mt-1 text-sm text-white/45">
                {t("tg.bot")}: @{tg.bot_username}
              </div>
            )}
            <p className="mt-3 max-w-xl text-sm text-white/50">{t("tg.help")}</p>
            <p className="mt-2 max-w-xl text-xs text-white/35">{t("tg.schedule")}</p>
            {tgMsg && <p className="mt-2 text-sm text-gold-200">{tgMsg}</p>}
          </div>
          <div className="flex flex-wrap gap-2">
            <a href={botLink} target="_blank" rel="noreferrer" className="rounded-2xl bg-white/5 px-4 py-3 text-sm font-bold hover:bg-white/10">
              {t("tg.open")}
            </a>
            <Btn kind="ghost" onClick={sendTest}>
              {t("tg.test")}
            </Btn>
          </div>
        </div>
      </div>

      {items.length === 0 ? (
        <Empty title={t("debts.empty.title")} text={t("debts.empty.text")} action={<Btn onClick={() => setOpen(true)}>{t("debts.empty.cta")}</Btn>} />
      ) : (
        <div className="grid gap-4 md:grid-cols-2">
          {items.map((d) => (
            <Link key={d.id} href={`/debts/${d.id}`} className="glass relative overflow-hidden rounded-[28px] p-5">
              <div className="absolute -end-8 -top-8 h-24 w-24 rounded-full opacity-25" style={{ background: d.color }} />
              <div className="relative">
                <div className="flex items-start justify-between gap-3">
                  <div>
                    <div className="text-xs text-white/40">{label(d.type)}</div>
                    <h3 className="mt-1 text-lg font-bold">{d.name}</h3>
                    {d.creditor && <div className="text-sm text-white/45">{d.creditor}</div>}
                  </div>
                  <span className={`rounded-full px-2 py-1 text-[11px] ${d.status === "paid" ? "bg-emerald-400/15 text-emerald-300" : "bg-rose-400/15 text-rose-300"}`}>
                    {d.status === "paid" ? t("debts.paid") : t("debts.active")}
                  </span>
                </div>
                <div className="mt-4 flex items-end justify-between">
                  <div>
                    <div className="text-[11px] text-white/40">{t("common.remaining")}</div>
                    <div className="text-xl font-extrabold text-rose-300">{toman(d.remaining, true, locale)}</div>
                    {(d.commission_amount || 0) > 0 && (
                      <div className="mt-1 text-[11px] text-white/40">
                        {t("debts.commission")} {toman(d.commission_amount || 0, true, locale)}
                        <span className="text-white/25"> · </span>
                        {t("debts.netReceived")} {toman(d.net_received || 0, true, locale)}
                      </div>
                    )}
                  </div>
                  <div className="text-end text-xs text-white/40">
                    {d.has_schedule ? t("debts.schedule") : t("debts.noSchedule")}
                    {d.start_date && d.end_date && (
                      <div>
                        {faDate(d.start_date, false, locale)} {t("debts.until")} {faDate(d.end_date, false, locale)}
                      </div>
                    )}
                    {d.next_due && (
                      <div className="mt-1 text-gold-200">
                        {t("debts.next")}: {faDate(d.next_due, false, locale)}
                      </div>
                    )}
                  </div>
                </div>
              </div>
            </Link>
          ))}
        </div>
      )}
      {open && <DebtForm initial={edit} onClose={() => setOpen(false)} onSaved={load} />}
    </div>
  );
}
