"use client";

import { useEffect, useState } from "react";
import { api } from "@/lib/api";
import { Asset } from "@/lib/types";
import { faDate, toman } from "@/lib/format";
import { Btn, Empty } from "@/components/ui";
import { AssetForm } from "@/components/forms";
import { FxHint } from "@/components/FxHint";
import Link from "next/link";
import { useI18n } from "@/lib/i18n";
import { isMarketAsset, liveAssetValue, liveUnitPrice, type Rates } from "@/lib/rates";

export default function AssetsPage() {
  const { t, label, locale } = useI18n();
  const [items, setItems] = useState<Asset[] | null>(null);
  const [rates, setRates] = useState<Rates | null>(null);
  const [open, setOpen] = useState(false);
  const [edit, setEdit] = useState<Asset | undefined>();

  function load() {
    api.assets().then(setItems);
    api
      .rates()
      .then(setRates)
      .catch(() => setRates(null));
  }
  useEffect(() => {
    load();
  }, []);

  if (!items) return <div className="animate-pulse text-white/40">{t("assets.loading")}</div>;

  const valued = items.map((a) => {
    const live = liveAssetValue(a.type, a.quantity, a.value, rates);
    return { a, live };
  });
  const total = valued.reduce((s, x) => s + x.live, 0);
  const byType = new Map<string, number>();
  for (const { a, live } of valued) byType.set(a.type, (byType.get(a.type) || 0) + live);

  return (
    <div className="mx-auto max-w-6xl">
      <div className="mb-6 flex flex-wrap items-end justify-between gap-4">
        <div>
          <h1 className="text-3xl font-extrabold">{t("assets.title")}</h1>
          <p className="mt-2 text-white/45">
            {t("assets.total")} {toman(total, true, locale)}
          </p>
          {rates && total > 0 && (
            <FxHint toman={total} rates={rates} className="mt-1 flex flex-wrap gap-x-3 text-xs text-white/45" />
          )}
        </div>
        <Btn
          onClick={() => {
            setEdit(undefined);
            setOpen(true);
          }}
        >
          {t("assets.add")}
        </Btn>
      </div>

      {rates && (
        <div className="mb-6 overflow-x-auto">
          <div className="flex min-w-max gap-2">
            <RateChip label={t("rates.usd")} value={toman(rates.usd_toman, true, locale)} />
            <RateChip label={t("rates.eur")} value={toman(rates.eur_toman, true, locale)} />
            <RateChip label={t("rates.gold18")} value={`${toman(rates.gold18_toman, true, locale)} / ${t("assets.perGram")}`} />
            <RateChip label={t("rates.gold24")} value={`${toman(rates.gold24_toman, true, locale)} / ${t("assets.perGram")}`} />
          </div>
          {rates.updated_at && (
            <div className="mt-2 text-[11px] text-white/30">
              {t("rates.updated")} · {faDate(rates.updated_at, true, locale)}
            </div>
          )}
        </div>
      )}

      {byType.size > 0 && (
        <div className="mb-6 flex flex-wrap gap-2">
          {[...byType.entries()].map(([type, val]) => (
            <div key={type} className="glass rounded-full px-4 py-2 text-sm">
              {label(type)} · {toman(val, true, locale)}
            </div>
          ))}
        </div>
      )}

      {items.length === 0 ? (
        <Empty title={t("assets.empty.title")} text={t("assets.empty.text")} action={<Btn onClick={() => setOpen(true)}>{t("assets.empty.cta")}</Btn>} />
      ) : (
        <div className="grid gap-4 sm:grid-cols-2 xl:grid-cols-3">
          {valued.map(({ a, live }) => {
            const unitLive = liveUnitPrice(a.type, rates);
            return (
              <button
                key={a.id}
                onClick={() => {
                  setEdit(a);
                  setOpen(true);
                }}
                className="glass relative overflow-hidden rounded-[28px] p-5 text-start"
              >
                <div className="absolute -start-6 -top-8 h-24 w-24 rounded-full opacity-30" style={{ background: a.color }} />
                <div className="relative">
                  <div className="text-xs text-white/40">{label(a.type)}</div>
                  <h3 className="mt-1 text-lg font-bold">{a.name}</h3>
                  <div className="mt-4 text-sm text-white/55">
                    {a.quantity} {label(a.unit) === a.unit ? a.unit : label(a.unit)}
                    {unitLive > 0 && (
                      <span className="ms-2 text-white/35">
                        · {toman(unitLive, true, locale)}
                        {(a.type === "gold18" || a.type === "gold24" || a.type === "gold") && ` / ${t("assets.perGram")}`}
                      </span>
                    )}
                  </div>
                  <div className="mt-1 text-xl font-extrabold text-gold-200">{toman(live, true, locale)}</div>
                  <FxHint toman={live} rates={rates} />
                  {isMarketAsset(a.type) && a.value !== live && (
                    <div className="mt-1 text-[11px] text-white/35">
                      {t("assets.book")}: {toman(a.value, true, locale)}
                    </div>
                  )}
                  {a.source_project_name && (
                    <Link href="/projects" onClick={(e) => e.stopPropagation()} className="mt-3 block text-[11px] text-white/35">
                      {t("assets.fromProject")} {a.source_project_name}
                    </Link>
                  )}
                </div>
              </button>
            );
          })}
        </div>
      )}
      {open && <AssetForm initial={edit} onClose={() => setOpen(false)} onSaved={load} />}
    </div>
  );
}

function RateChip({ label, value }: { label: string; value: string }) {
  return (
    <div className="glass rounded-2xl px-4 py-2.5">
      <div className="text-[10px] uppercase tracking-wide text-white/40">{label}</div>
      <div className="mt-0.5 text-sm font-bold text-gold-200">{value}</div>
    </div>
  );
}
