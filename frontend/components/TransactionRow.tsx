"use client";

import { Tx } from "@/lib/types";
import { faDate, toman } from "@/lib/format";
import { ArrowDownLeft, ArrowUpRight, ArrowLeftRight, Target, Undo2, Settings2, Landmark } from "lucide-react";
import { useI18n } from "@/lib/i18n";

const ICONS: Record<string, typeof ArrowDownLeft> = {
  income: ArrowDownLeft,
  expense: ArrowUpRight,
  transfer: ArrowLeftRight,
  contribution: Target,
  withdrawal: Undo2,
  opening: Landmark,
  adjustment: Settings2,
};

export function TransactionRow({ tx, onDelete }: { tx: Tx; onDelete?: () => void }) {
  const { t, label, locale } = useI18n();
  const Icon = ICONS[tx.type] || ArrowLeftRight;
  const positive = tx.type === "income" || tx.type === "withdrawal" || tx.type === "opening" || (tx.type === "adjustment" && tx.amount > 0);
  const amount = tx.type === "adjustment" ? tx.amount : Math.abs(tx.amount);

  return (
    <div className="flex items-center gap-3 rounded-2xl px-2 py-3 hover:bg-white/[0.03]">
      <div
        className={`flex h-11 w-11 items-center justify-center rounded-2xl ${
          positive ? "bg-emerald-400/10 text-emerald-300" : tx.type === "expense" || tx.type === "contribution" ? "bg-rose-400/10 text-rose-300" : "bg-white/5 text-gold-200"
        }`}
      >
        <Icon className="h-4 w-4" />
      </div>
      <div className="min-w-0 flex-1">
        <div className="truncate font-medium">{tx.description || label(tx.type)}</div>
        <div className="truncate text-xs text-white/40">
          {label(tx.type)}
          {tx.category ? ` · ${label(tx.category)}` : ""}
          {tx.account_name ? ` · ${tx.account_name}` : ""}
          {tx.to_account_name ? ` → ${tx.to_account_name}` : ""}
          {tx.project_name ? ` · ${tx.project_name}` : ""}
        </div>
      </div>
      <div className="text-end">
        <div className={`font-bold ${positive ? "text-emerald-300" : tx.type === "expense" || tx.type === "contribution" ? "text-rose-300" : "text-white"}`}>
          {positive ? "+" : tx.type === "expense" || tx.type === "contribution" ? "−" : ""}
          {toman(amount, true, locale)}
        </div>
        <div className="text-[11px] text-white/35">{faDate(tx.occurred_at, false, locale)}</div>
      </div>
      {onDelete && (
        <button onClick={onDelete} className="ms-1 text-xs text-white/25 hover:text-rose-300">
          {t("common.delete")}
        </button>
      )}
    </div>
  );
}
