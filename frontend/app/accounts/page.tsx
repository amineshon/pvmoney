"use client";

import { useEffect, useState } from "react";
import { api } from "@/lib/api";
import { Account } from "@/lib/types";
import { AccountCard } from "@/components/AccountCard";
import { AccountForm, AdjustForm } from "@/components/forms";
import { Btn, Empty } from "@/components/ui";
import { toman } from "@/lib/format";
import { useI18n } from "@/lib/i18n";

export default function AccountsPage() {
  const { t, locale } = useI18n();
  const [items, setItems] = useState<Account[] | null>(null);
  const [open, setOpen] = useState(false);
  const [edit, setEdit] = useState<Account | undefined>();
  const [adjust, setAdjust] = useState<Account | undefined>();

  function load() {
    api.accounts().then(setItems);
  }
  useEffect(() => {
    load();
  }, []);

  if (!items) return <div className="animate-pulse text-white/40">{t("accounts.loading")}</div>;
  const total = items.reduce((s, a) => s + a.balance, 0);

  return (
    <div className="mx-auto max-w-6xl">
      <div className="mb-8 flex flex-wrap items-end justify-between gap-4">
        <div>
          <h1 className="text-3xl font-extrabold">{t("accounts.title")}</h1>
          <p className="mt-2 text-white/45">
            {t("accounts.total")} {toman(total, true, locale)}
          </p>
        </div>
        <Btn
          onClick={() => {
            setEdit(undefined);
            setOpen(true);
          }}
        >
          {t("accounts.add")}
        </Btn>
      </div>
      {items.length === 0 ? (
        <Empty title={t("accounts.empty.title")} text={t("accounts.empty.text")} action={<Btn onClick={() => setOpen(true)}>{t("accounts.empty.cta")}</Btn>} />
      ) : (
        <div className="grid gap-5 sm:grid-cols-2 xl:grid-cols-3">
          {items.map((a) => (
            <AccountCard
              key={a.id}
              account={a}
              onEdit={() => {
                setEdit(a);
                setOpen(true);
              }}
              onAdjust={() => setAdjust(a)}
            />
          ))}
        </div>
      )}
      {open && <AccountForm initial={edit} onClose={() => setOpen(false)} onSaved={load} />}
      {adjust && <AdjustForm account={adjust} onClose={() => setAdjust(undefined)} onSaved={load} />}
    </div>
  );
}
