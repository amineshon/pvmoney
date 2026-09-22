import type { Locale } from "./i18n";

const intlMap: Record<Locale, string> = { fa: "fa-IR", en: "en-US", de: "de-DE" };

export function localeTag(locale: Locale = "fa") {
  return intlMap[locale] || "fa-IR";
}

export function formatMoney(n: number, compact = false, locale: Locale = "fa"): string {
  if (!Number.isFinite(n)) return formatNumber(0, locale);
  const abs = Math.abs(n);
  if (compact) {
    if (abs >= 1_000_000_000) return `${formatNumber(round(n / 1_000_000_000), locale)} ${compactUnit(locale, "billion")}`;
    if (abs >= 1_000_000) return `${formatNumber(round(n / 1_000_000), locale)} ${compactUnit(locale, "million")}`;
    if (abs >= 1_000) return `${formatNumber(round(n / 1_000), locale)} ${compactUnit(locale, "thousand")}`;
  }
  return formatNumber(n, locale);
}

function compactUnit(locale: Locale, kind: "thousand" | "million" | "billion") {
  const map = {
    fa: { thousand: "هزار", million: "میلیون", billion: "میلیارد" },
    en: { thousand: "k", million: "m", billion: "b" },
    de: { thousand: "Tsd.", million: "Mio.", billion: "Mrd." },
  };
  return map[locale][kind];
}

export function toman(n: number, compact = false, locale: Locale = "fa"): string {
  const unit = locale === "fa" ? "تومان" : "Toman";
  return `${formatMoney(n, compact, locale)} ${unit}`;
}

export function formatUSD(n: number, locale: Locale = "fa"): string {
  const digits = Math.abs(n) >= 100 ? 0 : 2;
  return new Intl.NumberFormat(localeTag(locale), { style: "currency", currency: "USD", maximumFractionDigits: digits }).format(n);
}

export function formatEUR(n: number, locale: Locale = "fa"): string {
  const digits = Math.abs(n) >= 100 ? 0 : 2;
  return new Intl.NumberFormat(localeTag(locale), { style: "currency", currency: "EUR", maximumFractionDigits: digits }).format(n);
}

export function formatNumber(n: number, locale: Locale = "fa"): string {
  return new Intl.NumberFormat(localeTag(locale), { maximumFractionDigits: 1 }).format(n);
}

export const toFa = (n: number) => formatNumber(n, "fa");

function round(n: number): number {
  return Math.round(n * 10) / 10;
}

export function faDate(iso: string, withTime = false, locale: Locale = "fa"): string {
  const d = new Date(iso);
  if (Number.isNaN(d.getTime())) return iso;
  return d.toLocaleDateString(localeTag(locale), {
    year: "numeric",
    month: "short",
    day: "numeric",
    ...(withTime ? { hour: "2-digit", minute: "2-digit" } : {}),
  });
}

export function faMonth(ym: string, locale: Locale = "fa"): string {
  const [y, m] = ym.split("-").map(Number);
  if (!y || !m) return ym;
  return new Date(y, m - 1, 1).toLocaleDateString(localeTag(locale), { month: "long", year: "numeric" });
}

export function maskCard(num: string, locale: Locale = "fa"): string {
  const digits = (num || "").replace(/\D/g, "");
  if (digits.length < 4) return "••••  ••••";
  const last = formatNumber(Number(digits.slice(-4)), locale).replace(/[^\d]/g, "");
  return `••••  ${last || digits.slice(-4)}`;
}

export function greeting(locale: Locale = "fa"): string {
  const h = new Date().getHours();
  const map = {
    fa: ["صبح بخیر", "ظهر بخیر", "عصر بخیر", "شب بخیر"],
    en: ["Good morning", "Good afternoon", "Good evening", "Good night"],
    de: ["Guten Morgen", "Guten Tag", "Guten Abend", "Gute Nacht"],
  }[locale];
  if (h < 12) return map[0];
  if (h < 17) return map[1];
  if (h < 21) return map[2];
  return map[3];
}

export function progress(current: number, target: number): number {
  if (target <= 0) return 0;
  return Math.min(100, Math.round((current / target) * 1000) / 10);
}
