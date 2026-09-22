"use client";

import { formatEUR, formatUSD } from "@/lib/format";
import { convertToman, type Rates } from "@/lib/rates";
import { useI18n } from "@/lib/i18n";

export function FxHint({
  toman,
  rates,
  className,
}: {
  toman: number;
  rates?: Rates | null;
  className?: string;
}) {
  const { locale } = useI18n();
  if (!rates || !Number.isFinite(toman) || toman <= 0) return null;
  const fx = convertToman(toman, rates);
  return (
    <div className={className || "mt-1.5 flex flex-wrap gap-x-3 gap-y-1 text-xs text-white/50"}>
      <span>≈ {formatUSD(fx.usd, locale)}</span>
      <span>≈ {formatEUR(fx.eur, locale)}</span>
    </div>
  );
}
