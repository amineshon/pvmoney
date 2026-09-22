"use client";

import { useEffect, useState } from "react";
import { api } from "@/lib/api";
import { Account, Project, Tx } from "@/lib/types";
import { TransactionRow } from "@/components/TransactionRow";
import { TxForm } from "@/components/forms";
import { Btn, Empty, inputClass } from "@/components/ui";
import { useI18n } from "@/lib/i18n";

const TX_TYPES = ["income", "expense", "transfer", "contribution", "withdrawal", "opening", "adjustment"];

export default function TransactionsPage() {
  const { t, label } = useI18n();
  const [items, setItems] = useState<Tx[] | null>(null);
  const [accounts, setAccounts] = useState<Account[]>([]);
  const [projects, setProjects] = useState<Project[]>([]);
  const [type, setType] = useState("");
  const [open, setOpen] = useState(false);

  function load() {
    const q = type ? `?type=${type}` : "";
    Promise.all([api.transactions(q), api.accounts(), api.projects()]).then(([tx, a, p]) => {
      setItems(tx);
      setAccounts(a);
      setProjects(p);
    });
  }

  useEffect(() => {
    load();
  }, [type]);

  async function remove(id: string) {
    if (!confirm(t("tx.deleteAsk"))) return;
    await api.deleteTx(id);
    load();
  }

  return (
    <div className="mx-auto max-w-4xl">
      <div className="mb-8 flex flex-wrap items-end justify-between gap-4">
        <div>
          <h1 className="text-3xl font-extrabold">{t("tx.title")}</h1>
          <p className="mt-2 text-white/45">{t("tx.sub")}</p>
        </div>
        <Btn onClick={() => setOpen(true)}>{t("tx.add")}</Btn>
      </div>
      {items === null ? (
        <div className="animate-pulse text-white/40">{t("tx.loading")}</div>
      ) : (
        <>
          <select className={inputClass("mb-4 max-w-xs")} value={type} onChange={(e) => setType(e.target.value)}>
            <option value="">{t("tx.allTypes")}</option>
            {TX_TYPES.map((item) => (
              <option key={item} value={item}>
                {label(item)}
              </option>
            ))}
          </select>
          <div className="glass rounded-[28px] p-3 md:p-5">
            {items.length === 0 ? (
              <Empty title={t("tx.empty.title")} text={t("tx.empty.text")} />
            ) : (
              items.map((tx) => <TransactionRow key={tx.id} tx={tx} onDelete={() => remove(tx.id)} />)
            )}
          </div>
        </>
      )}
      {open && <TxForm accounts={accounts} projects={projects} onClose={() => setOpen(false)} onSaved={load} />}
    </div>
  );
}
