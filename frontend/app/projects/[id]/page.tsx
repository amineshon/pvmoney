"use client";

import { useEffect, useState } from "react";
import { useParams, useRouter } from "next/navigation";
import Link from "next/link";
import { api } from "@/lib/api";
import { Account, Project, Tx } from "@/lib/types";
import { faDate, progress, toman } from "@/lib/format";
import { Btn } from "@/components/ui";
import { ConvertAssetForm, ItemForm, PayItemForm, ProjectForm } from "@/components/forms";
import { TransactionRow } from "@/components/TransactionRow";
import { apiError, useI18n } from "@/lib/i18n";

export default function ProjectDetailPage() {
  const { id } = useParams<{ id: string }>();
  const router = useRouter();
  const { t, label, locale } = useI18n();
  const [project, setProject] = useState<Project | null>(null);
  const [accounts, setAccounts] = useState<Account[]>([]);
  const [txs, setTxs] = useState<Tx[]>([]);
  const [edit, setEdit] = useState(false);
  const [itemOpen, setItemOpen] = useState(false);
  const [pay, setPay] = useState<{ id: string; name: string; remaining: number } | null>(null);
  const [convert, setConvert] = useState(false);

  async function load() {
    const [p, a, tx] = await Promise.all([api.project(id), api.accounts(), api.transactions(`?project_id=${id}`)]);
    setProject(p);
    setAccounts(a);
    setTxs(tx);
  }

  useEffect(() => {
    load();
  }, [id]);

  if (!project) return <div className="text-white/40">{t("project.loading")}</div>;
  const pct = progress(project.current_amount, project.target_amount);
  const items = project.items || [];
  const itemsSum = items.reduce((s, it) => s + it.planned_amount, 0);
  const remaining = project.target_amount - project.current_amount;

  async function remove() {
    try {
      await api.deleteProject(project!.id);
      router.push("/projects");
    } catch (e) {
      alert(apiError(e instanceof Error ? e.message : "", t));
    }
  }

  async function removeItem(itemId: string) {
    try {
      await api.deleteItem(project!.id, itemId);
      load();
    } catch (e) {
      alert(apiError(e instanceof Error ? e.message : "", t));
    }
  }

  return (
    <div className="mx-auto max-w-3xl">
      <div className="glass relative overflow-hidden rounded-[32px] p-8">
        <div className="absolute inset-x-0 bottom-0" style={{ height: `${Math.max(6, pct)}%`, background: `${project.color}33` }} />
        <div className="relative">
          <div className="text-sm text-white/40">{project.status === "completed" ? t("project.completed") : t("project.active")}</div>
          <h1 className="mt-1 text-3xl font-black">{project.name}</h1>
          {project.description && <p className="mt-3 text-white/55">{project.description}</p>}
          {project.result_asset_type && (
            <div className="mt-3 inline-flex rounded-full bg-gold-400/10 px-3 py-1 text-xs text-gold-200">
              {t("project.dest")} {label(project.result_asset_type)}
            </div>
          )}
          <div className="mt-8 grid grid-cols-2 gap-3 sm:grid-cols-4">
            <Stat label={t("projects.base")} value={toman(project.base_amount || 0, true, locale)} />
            <Stat label={t("projects.itemsSum")} value={toman(itemsSum, true, locale)} />
            <Stat label={t("projects.budget")} value={toman(project.target_amount, true, locale)} gold />
            <Stat label={t("projects.spentLabel")} value={toman(project.current_amount, true, locale)} rose />
          </div>
          <div className="mt-4 flex items-end justify-between">
            <div>
              <div className="text-xs text-white/40">{t("projects.remaining")}</div>
              <div className={`text-2xl font-bold ${remaining < 0 ? "text-rose-300" : "text-emerald-300"}`}>
                {toman(remaining, true, locale)}
              </div>
            </div>
            <div className="text-5xl font-black text-gold-200">{pct}%</div>
          </div>
          <div className="mt-5 h-3 overflow-hidden rounded-full bg-white/10">
            <div className="h-full rounded-full transition-all" style={{ width: `${pct}%`, background: project.color }} />
          </div>
          {project.deadline && (
            <div className="mt-3 text-sm text-white/40">
              {t("projects.deadline")} {faDate(project.deadline, false, locale)}
            </div>
          )}
          {project.asset && (
            <Link href="/assets" className="mt-4 block rounded-2xl bg-gold-400/10 px-4 py-3 text-sm text-gold-200">
              {t("project.converted", { name: project.asset.name, value: toman(project.asset.value, true, locale) })}
            </Link>
          )}
          <div className="mt-8 flex flex-wrap gap-2">
            <Btn onClick={() => setItemOpen(true)}>{t("project.addItem")}</Btn>
            {!project.asset && (
              <Btn kind="ghost" onClick={() => setConvert(true)}>
                {t("project.toAsset")}
              </Btn>
            )}
            <Btn kind="ghost" onClick={() => setEdit(true)}>
              {t("common.edit")}
            </Btn>
            <Btn kind="danger" onClick={remove}>
              {t("common.delete")}
            </Btn>
          </div>
        </div>
      </div>

      <div className="mt-8 space-y-3">
        <div className="flex items-center justify-between">
          <h2 className="font-bold">{t("project.items")}</h2>
          <button onClick={() => setItemOpen(true)} className="text-sm text-gold-200">
            {t("common.add")}
          </button>
        </div>
        {items.length === 0 ? (
          <div className="glass rounded-[24px] p-6 text-sm text-white/45">{t("project.itemsEmpty")}</div>
        ) : (
          items.map((it) => {
            const p = progress(it.paid_amount, it.planned_amount || it.paid_amount || 1);
            const remaining = Math.max(0, it.planned_amount - it.paid_amount);
            return (
              <div key={it.id} className="glass rounded-[24px] p-4">
                <div className="flex items-start justify-between gap-3">
                  <div>
                    <div className="font-bold">{it.name}</div>
                    {it.notes && <div className="text-xs text-white/40">{it.notes}</div>}
                  </div>
                  <div className="text-end text-sm">
                    <div className="text-rose-300">
                      {toman(it.paid_amount, true, locale)} {t("project.spentOf")}
                    </div>
                    <div className="text-white/40">
                      {t("project.from")} {toman(it.planned_amount, true, locale)}
                    </div>
                  </div>
                </div>
                <div className="mt-3 h-1.5 overflow-hidden rounded-full bg-white/10">
                  <div className="h-full rounded-full bg-rose-400" style={{ width: `${p}%` }} />
                </div>
                <div className="mt-3 flex gap-2">
                  <Btn className="flex-1 py-2 text-xs" onClick={() => setPay({ id: it.id, name: it.name, remaining })}>
                    {t("common.pay")}
                  </Btn>
                  <Btn kind="ghost" className="py-2 text-xs" onClick={() => removeItem(it.id)}>
                    {t("common.delete")}
                  </Btn>
                </div>
              </div>
            );
          })
        )}
      </div>

      <div className="mt-8 glass rounded-[28px] p-5">
        <h2 className="mb-3 font-bold">{t("project.history")}</h2>
        {txs.length === 0 ? <p className="text-sm text-white/40">{t("project.noPay")}</p> : txs.map((tx) => <TransactionRow key={tx.id} tx={tx} />)}
      </div>

      {itemOpen && (
        <ItemForm projectId={project.id} budgetTotal={project.target_amount} onClose={() => setItemOpen(false)} onSaved={load} />
      )}
      {pay && (
        <PayItemForm
          projectId={project.id}
          itemId={pay.id}
          itemName={pay.name}
          remaining={pay.remaining}
          accounts={accounts}
          onClose={() => setPay(null)}
          onSaved={load}
        />
      )}
      {convert && (
        <ConvertAssetForm
          projectId={project.id}
          defaultName={project.name}
          defaultType={project.result_asset_type}
          spent={project.current_amount}
          onClose={() => setConvert(false)}
          onSaved={load}
        />
      )}
      {edit && <ProjectForm initial={project} onClose={() => setEdit(false)} onSaved={load} />}
    </div>
  );
}

function Stat({ label, value, gold, rose }: { label: string; value: string; gold?: boolean; rose?: boolean }) {
  return (
    <div className="rounded-2xl bg-white/[0.04] px-3 py-3">
      <div className="text-[11px] text-white/40">{label}</div>
      <div className={`mt-1 text-sm font-bold ${gold ? "text-gold-200" : rose ? "text-rose-300" : "text-white/85"}`}>{value}</div>
    </div>
  );
}
