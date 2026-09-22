"use client";

import { Account } from "@/lib/types";
import { maskCard, toman } from "@/lib/format";
import { Pencil, SlidersHorizontal } from "lucide-react";
import { useI18n } from "@/lib/i18n";

export function AccountCard({
  account,
  onEdit,
  onAdjust,
}: {
  account: Account;
  onEdit?: () => void;
  onAdjust?: () => void;
}) {
  const { label, locale } = useI18n();
  return (
    <div
      className="card-shine relative aspect-[1.62/1] overflow-hidden rounded-[28px] p-5 text-white shadow-card"
      style={{
        background: `linear-gradient(145deg, ${account.color} 0%, #0b0d12 78%)`,
      }}
    >
      <div className="absolute -start-8 -top-10 h-36 w-36 rounded-full bg-white/10 blur-2xl" />
      <div className="absolute bottom-0 left-0 right-0 h-24 bg-gradient-to-t from-black/40 to-transparent" />
      <div className="relative flex h-full flex-col justify-between">
        <div className="flex items-start justify-between">
          <div>
            <div className="text-xs text-white/70">{account.bank_name || label(account.type)}</div>
            <div className="mt-1 text-lg font-bold">{account.name}</div>
          </div>
          <div className="flex items-center gap-1">
            {onAdjust && (
              <button onClick={onAdjust} className="rounded-lg bg-white/10 p-1.5 hover:bg-white/20">
                <SlidersHorizontal className="h-3.5 w-3.5" />
              </button>
            )}
            {onEdit && (
              <button onClick={onEdit} className="rounded-lg bg-white/10 p-1.5 hover:bg-white/20">
                <Pencil className="h-3.5 w-3.5" />
              </button>
            )}
            <div className="h-8 w-11 rounded-md bg-gradient-to-br from-yellow-200/80 to-amber-600/80 shadow-inner" />
          </div>
        </div>
        <div>
          <div className="font-mono text-sm tracking-[0.28em] text-white/70">{maskCard(account.account_number, locale)}</div>
          <div className="mt-3 text-2xl font-extrabold tracking-tight">{toman(account.balance, true, locale)}</div>
        </div>
      </div>
    </div>
  );
}
