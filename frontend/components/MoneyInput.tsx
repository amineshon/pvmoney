"use client";

import { useMemo, useState } from "react";
import { inputClass } from "./ui";
import { formatMoney } from "@/lib/format";
import { MAX_MONEY } from "@/lib/constants";
import { useI18n } from "@/lib/i18n";

function pickUnit(v: number) {
  if (v >= 1_000_000_000) return 1_000_000_000;
  if (v >= 1_000_000) return 1_000_000;
  if (v >= 1_000) return 1_000;
  return 1;
}

function toLatinDigits(s: string) {
  const fa = "۰۱۲۳۴۵۶۷۸۹";
  const ar = "٠١٢٣٤٥٦٧٨٩";
  let out = "";
  for (const ch of s) {
    const i = fa.indexOf(ch);
    if (i >= 0) {
      out += String(i);
      continue;
    }
    const j = ar.indexOf(ch);
    if (j >= 0) {
      out += String(j);
      continue;
    }
    out += ch;
  }
  return out;
}

function parseRaw(s: string): number {
  let cleaned = "";
  let dot = false;
  for (const ch of toLatinDigits(s)) {
    if (ch === "," || ch === "،" || ch === " " || ch === "_" || ch === "٬") continue;
    if ((ch === "." || ch === "/") && !dot) {
      cleaned += ".";
      dot = true;
      continue;
    }
    if (ch >= "0" && ch <= "9") cleaned += ch;
  }
  if (!cleaned || cleaned === ".") return 0;
  const n = Number(cleaned);
  return Number.isFinite(n) ? n : 0;
}

function prettyRaw(n: number) {
  if (!n) return "";
  if (Number.isInteger(n)) return String(n);
  return String(Math.round(n * 10000) / 10000);
}

export function MoneyInput({
  value,
  onChange,
  placeholder,
}: {
  value: number;
  onChange: (n: number) => void;
  placeholder?: string;
}) {
  const { t, locale } = useI18n();
  const units = [
    { id: 1, label: t("currency.toman") },
    { id: 1_000, label: t("currency.thousand") },
    { id: 1_000_000, label: t("currency.million") },
    { id: 1_000_000_000, label: t("currency.billion") },
  ];
  const safeValue = Number.isFinite(value) && value >= 0 && value <= MAX_MONEY ? value : 0;
  const [unit, setUnit] = useState(safeValue ? pickUnit(safeValue) : 1_000_000);
  const [raw, setRaw] = useState(safeValue ? prettyRaw(safeValue / pickUnit(safeValue)) : "");
  const [tooBig, setTooBig] = useState(false);

  function emit(nextRaw: string, nextUnit: number) {
    const n = parseRaw(nextRaw);
    const toman = Math.round(n * nextUnit);
    if (!Number.isFinite(toman) || toman > MAX_MONEY) {
      setTooBig(true);
      onChange(MAX_MONEY);
      return;
    }
    setTooBig(false);
    onChange(Math.max(0, toman));
  }

  const preview = useMemo(() => {
    const n = parseRaw(raw);
    const toman = n * unit;
    if (!Number.isFinite(toman) || toman > MAX_MONEY) return MAX_MONEY;
    return Math.max(0, Math.round(toman));
  }, [raw, unit]);

  return (
    <div>
      <div className="mb-2 flex gap-1.5 overflow-x-auto pb-0.5 [-ms-overflow-style:none] [scrollbar-width:none] [&::-webkit-scrollbar]:hidden">
        {units.map((u) => (
          <button
            key={u.id}
            type="button"
            onClick={() => {
              const current = parseRaw(raw) * unit;
              const nextRaw = current ? prettyRaw(current / u.id) : raw;
              setUnit(u.id);
              setRaw(nextRaw);
              emit(nextRaw, u.id);
            }}
            className={`shrink-0 rounded-full px-3 py-1.5 text-xs font-bold transition ${
              unit === u.id ? "bg-gold-400 text-ink-950" : "border border-white/12 bg-[#121722] text-white/70 hover:bg-white/10"
            }`}
          >
            {u.label}
          </button>
        ))}
      </div>
      <input
        className={inputClass(tooBig ? "border-rose-400/50 focus:border-rose-400/70" : "")}
        inputMode="decimal"
        placeholder={placeholder || t("money.placeholder")}
        value={raw}
        onChange={(e) => {
          const v = e.target.value;
          setRaw(v);
          emit(v, unit);
        }}
      />
      <div className={`mt-1.5 text-xs ${tooBig ? "text-rose-300" : "text-gold-200/80"}`}>
        {tooBig ? t("money.tooBig") : `${formatMoney(preview, false, locale)} ${t("currency.toman")}`}
      </div>
    </div>
  );
}
