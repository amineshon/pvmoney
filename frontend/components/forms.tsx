"use client";

import { FormEvent, useEffect, useState } from "react";
import { Account, Debt, Project } from "@/lib/types";
import { ACCOUNT_COLORS, ACCOUNT_TYPES, ASSET_TYPES, DEBT_COLORS, DEBT_TYPES, EXPENSE_CATS, INCOME_CATS } from "@/lib/constants";
import { api } from "@/lib/api";
import { Btn, ErrorBox, Field, Modal, inputClass } from "./ui";
import { MoneyInput } from "./MoneyInput";
import { FxHint } from "./FxHint";
import { apiError, useI18n } from "@/lib/i18n";
import { formatMoney, toman } from "@/lib/format";
import { isMarketAsset, liveUnitPrice, type Rates } from "@/lib/rates";

export function AccountForm({
  initial,
  onClose,
  onSaved,
}: {
  initial?: Account;
  onClose: () => void;
  onSaved: () => void;
}) {
  const { t, label } = useI18n();
  const [name, setName] = useState(initial?.name || "");
  const [bank, setBank] = useState(initial?.bank_name || "");
  const [number, setNumber] = useState(initial?.account_number || "");
  const [type, setType] = useState(initial?.type || "bank");
  const [color, setColor] = useState(initial?.color || ACCOUNT_COLORS[0]);
  const [balance, setBalance] = useState(initial?.balance || 0);
  const [error, setError] = useState("");
  const [loading, setLoading] = useState(false);

  async function onSubmit(e: FormEvent) {
    e.preventDefault();
    setLoading(true);
    setError("");
    try {
      const body = { name, bank_name: bank, account_number: number, type, color, icon: "landmark", balance };
      if (initial) await api.updateAccount(initial.id, body);
      else await api.createAccount(body);
      onSaved();
      onClose();
    } catch (err) {
      setError(apiError(err instanceof Error ? err.message : "", t));
    } finally {
      setLoading(false);
    }
  }

  async function remove() {
    if (!initial) return;
    if (!confirm(t("form.account.deleteAsk"))) return;
    try {
      await api.deleteAccount(initial.id);
      onSaved();
      onClose();
    } catch (err) {
      setError(apiError(err instanceof Error ? err.message : "", t));
    }
  }

  return (
    <Modal title={initial ? t("form.account.edit") : t("form.account.new")} subtitle={t("form.account.sub")} onClose={onClose}>
      <form onSubmit={onSubmit}>
        <ErrorBox message={error} />
        <Field label={t("form.account.name")}>
          <input className={inputClass()} value={name} onChange={(e) => setName(e.target.value)} placeholder={t("form.account.namePh")} required />
        </Field>
        <div className="grid grid-cols-2 gap-3">
          <Field label={t("form.account.bank")}>
            <input className={inputClass()} value={bank} onChange={(e) => setBank(e.target.value)} placeholder={t("form.account.bankPh")} />
          </Field>
          <Field label={t("form.account.number")}>
            <input className={inputClass()} value={number} onChange={(e) => setNumber(e.target.value)} placeholder="6037..." />
          </Field>
        </div>
        <Field label={t("common.type")}>
          <select className={inputClass()} value={type} onChange={(e) => setType(e.target.value)}>
            {ACCOUNT_TYPES.map((item) => (
              <option key={item.id} value={item.id}>
                {label(item.id)}
              </option>
            ))}
          </select>
        </Field>
        {!initial && (
          <Field label={t("form.account.balance")}>
            <MoneyInput value={balance} onChange={setBalance} />
          </Field>
        )}
        <Field label={t("form.account.color")}>
          <div className="flex flex-wrap gap-2">
            {ACCOUNT_COLORS.map((c) => (
              <button
                key={c}
                type="button"
                onClick={() => setColor(c)}
                className={`h-8 w-8 rounded-full ${color === c ? "ring-2 ring-gold-200 ring-offset-2 ring-offset-ink-900" : ""}`}
                style={{ background: c }}
              />
            ))}
          </div>
        </Field>
        <div className="mt-4 flex gap-2">
          <Btn type="submit" className="flex-1" disabled={loading}>
            {loading ? t("common.saving") : t("form.account.save")}
          </Btn>
          {initial && (
            <Btn kind="danger" onClick={remove}>
              {t("common.delete")}
            </Btn>
          )}
        </div>
      </form>
    </Modal>
  );
}

export function AdjustForm({ account, onClose, onSaved }: { account: Account; onClose: () => void; onSaved: () => void }) {
  const { t } = useI18n();
  const [balance, setBalance] = useState(account.balance);
  const [desc, setDesc] = useState(t("form.adjust.desc"));
  const [error, setError] = useState("");

  async function onSubmit(e: FormEvent) {
    e.preventDefault();
    try {
      await api.adjustAccount(account.id, { balance, description: desc });
      onSaved();
      onClose();
    } catch (err) {
      setError(apiError(err instanceof Error ? err.message : "", t));
    }
  }

  return (
    <Modal title={t("form.adjust.title")} subtitle={account.name} onClose={onClose}>
      <form onSubmit={onSubmit}>
        <ErrorBox message={error} />
        <Field label={t("form.adjust.new")}>
          <MoneyInput value={balance} onChange={setBalance} />
        </Field>
        <Field label={t("common.description")}>
          <input className={inputClass()} value={desc} onChange={(e) => setDesc(e.target.value)} />
        </Field>
        <Btn type="submit" className="w-full">
          {t("common.apply")}
        </Btn>
      </form>
    </Modal>
  );
}

export function ProjectForm({
  initial,
  onClose,
  onSaved,
}: {
  initial?: { id: string; name: string; description: string; target_amount: number; base_amount?: number; deadline?: string | null; color: string; result_asset_type?: string };
  onClose: () => void;
  onSaved: () => void;
}) {
  const { t, label } = useI18n();
  const [name, setName] = useState(initial?.name || "");
  const [description, setDescription] = useState(initial?.description || "");
  const [target, setTarget] = useState(initial?.base_amount ?? initial?.target_amount ?? 0);
  const [deadline, setDeadline] = useState(initial?.deadline ? initial.deadline.slice(0, 10) : "");
  const [color, setColor] = useState(initial?.color || "#3ee0a2");
  const [resultType, setResultType] = useState(initial?.result_asset_type || "");
  const [error, setError] = useState("");
  const colors = ["#3ee0a2", "#e0c36a", "#7dd3fc", "#f9a8d4", "#a78bfa", "#fb7185"];

  async function onSubmit(e: FormEvent) {
    e.preventDefault();
    try {
      const body = { name, description, base_amount: target, target_amount: target, deadline: deadline || null, color, result_asset_type: resultType };
      if (initial) await api.updateProject(initial.id, body);
      else await api.createProject(body);
      onSaved();
      onClose();
    } catch (err) {
      setError(apiError(err instanceof Error ? err.message : "", t));
    }
  }

  return (
    <Modal title={initial ? t("form.project.edit") : t("form.project.new")} subtitle={t("form.project.sub")} onClose={onClose}>
      <form onSubmit={onSubmit}>
        <ErrorBox message={error} />
        <Field label={t("form.project.name")}>
          <input className={inputClass()} value={name} onChange={(e) => setName(e.target.value)} placeholder={t("form.project.namePh")} required />
        </Field>
        <Field label={t("common.description")}>
          <textarea className={inputClass("min-h-20")} value={description} onChange={(e) => setDescription(e.target.value)} />
        </Field>
        <Field label={t("form.project.budget")}>
          <MoneyInput value={target} onChange={setTarget} />
          <p className="mt-1.5 text-xs text-white/40">{t("form.project.budgetHint")}</p>
        </Field>
        <Field label={t("form.project.result")}>
          <select className={inputClass()} value={resultType} onChange={(e) => setResultType(e.target.value)}>
            <option value="">{t("form.project.resultNone")}</option>
            {ASSET_TYPES.map((item) => (
              <option key={item.id} value={item.id}>
                {label(item.id)}
              </option>
            ))}
          </select>
        </Field>
        <Field label={t("form.project.deadline")}>
          <input type="date" className={inputClass()} value={deadline} onChange={(e) => setDeadline(e.target.value)} />
        </Field>
        <Field label={t("common.color")}>
          <div className="flex gap-2">
            {colors.map((c) => (
              <button
                key={c}
                type="button"
                onClick={() => setColor(c)}
                className={`h-8 w-8 rounded-full ${color === c ? "ring-2 ring-white" : ""}`}
                style={{ background: c }}
              />
            ))}
          </div>
        </Field>
        <Btn type="submit" className="w-full">
          {t("form.project.save")}
        </Btn>
      </form>
    </Modal>
  );
}

export function ItemForm({
  projectId,
  initial,
  budgetTotal = 0,
  onClose,
  onSaved,
}: {
  projectId: string;
  initial?: { id: string; name: string; planned_amount: number; notes: string };
  budgetTotal?: number;
  onClose: () => void;
  onSaved: () => void;
}) {
  const { t, locale } = useI18n();
  const [name, setName] = useState(initial?.name || "");
  const [planned, setPlanned] = useState(initial?.planned_amount || 0);
  const [notes, setNotes] = useState(initial?.notes || "");
  const [error, setError] = useState("");
  const replacing = initial?.planned_amount || 0;
  const nextBudget = Math.max(0, budgetTotal - replacing + planned);

  async function onSubmit(e: FormEvent) {
    e.preventDefault();
    try {
      const body = { name, planned_amount: planned, notes };
      if (initial) await api.updateItem(projectId, initial.id, body);
      else await api.createItem(projectId, body);
      onSaved();
      onClose();
    } catch (err) {
      setError(apiError(err instanceof Error ? err.message : "", t));
    }
  }

  return (
    <Modal title={initial ? t("form.item.edit") : t("form.item.new")} subtitle={t("form.item.sub")} onClose={onClose}>
      <form onSubmit={onSubmit}>
        <ErrorBox message={error} />
        <Field label={t("form.item.title")}>
          <input className={inputClass()} value={name} onChange={(e) => setName(e.target.value)} placeholder={t("form.item.titlePh")} required />
        </Field>
        <Field label={t("form.item.estimate")}>
          <MoneyInput value={planned} onChange={setPlanned} />
          <p className="mt-1.5 text-xs text-white/40">{t("form.item.budgetHint")}</p>
          {planned > 0 && (
            <p className="mt-1 text-xs font-semibold text-gold-200">
              {t("form.item.nextBudget", { amount: toman(nextBudget, true, locale) })}
            </p>
          )}
        </Field>
        <Field label={t("common.note")}>
          <input className={inputClass()} value={notes} onChange={(e) => setNotes(e.target.value)} />
        </Field>
        <Btn type="submit" className="w-full">
          {t("common.save")}
        </Btn>
      </form>
    </Modal>
  );
}

export function PayItemForm({
  projectId,
  itemId,
  itemName,
  remaining,
  accounts,
  onClose,
  onSaved,
}: {
  projectId: string;
  itemId: string;
  itemName: string;
  remaining: number;
  accounts: { id: string; name: string }[];
  onClose: () => void;
  onSaved: () => void;
}) {
  const { t } = useI18n();
  const [accountId, setAccountId] = useState(accounts[0]?.id || "");
  const [amount, setAmount] = useState(Math.max(0, remaining));
  const [desc, setDesc] = useState(itemName);
  const [error, setError] = useState("");

  async function onSubmit(e: FormEvent) {
    e.preventDefault();
    try {
      await api.payItem(projectId, itemId, { account_id: accountId, amount, description: desc });
      onSaved();
      onClose();
    } catch (err) {
      setError(apiError(err instanceof Error ? err.message : "", t));
    }
  }

  return (
    <Modal title={t("form.payItem.title")} subtitle={t("form.payItem.sub", { name: itemName })} onClose={onClose}>
      <form onSubmit={onSubmit}>
        <ErrorBox message={error} />
        <Field label={t("common.fromAccount")}>
          <select className={inputClass()} value={accountId} onChange={(e) => setAccountId(e.target.value)}>
            {accounts.map((a) => (
              <option key={a.id} value={a.id}>
                {a.name}
              </option>
            ))}
          </select>
        </Field>
        <Field label={t("form.payItem.amount")}>
          <MoneyInput value={amount} onChange={setAmount} />
        </Field>
        <Field label={t("common.description")}>
          <input className={inputClass()} value={desc} onChange={(e) => setDesc(e.target.value)} />
        </Field>
        <Btn type="submit" className="w-full">
          {t("form.payItem.save")}
        </Btn>
      </form>
    </Modal>
  );
}

export function AssetForm({
  initial,
  sourceProjectId,
  onClose,
  onSaved,
}: {
  initial?: { id: string; name: string; type: string; quantity: number; unit: string; unit_value: number; notes: string; color: string };
  sourceProjectId?: string;
  onClose: () => void;
  onSaved: () => void;
}) {
  const { t, label, locale } = useI18n();
  const [name, setName] = useState(initial?.name || "");
  const [type, setType] = useState(initial?.type || "gold18");
  const [quantity, setQuantity] = useState(String(initial?.quantity ?? 1));
  const [unit, setUnit] = useState(initial?.unit || label("gram"));
  const [unitValue, setUnitValue] = useState(initial?.unit_value || 0);
  const [notes, setNotes] = useState(initial?.notes || "");
  const [error, setError] = useState("");
  const [rates, setRates] = useState<Rates | null>(null);

  useEffect(() => {
    api
      .rates()
      .then(setRates)
      .catch(() => setRates(null));
  }, []);

  useEffect(() => {
    if (!rates || initial) return;
    const live = liveUnitPrice(type, rates);
    if (live > 0 && unitValue === 0) setUnitValue(live);
  }, [rates, type, initial, unitValue]);

  function changeType(next: string) {
    setType(next);
    const found = ASSET_TYPES.find((item) => item.id === next);
    if (found) setUnit(label(found.unit));
    const live = liveUnitPrice(next, rates);
    if (live > 0 && !initial) setUnitValue(live);
  }

  const qty = Number(quantity) || 0;
  const live = liveUnitPrice(type, rates);
  const bookTotal = Math.round(qty * unitValue);
  const marketTotal = live > 0 ? Math.round(qty * live) : bookTotal;

  async function onSubmit(e: FormEvent) {
    e.preventDefault();
    try {
      const body = {
        name,
        type,
        quantity: Number(quantity) || 1,
        unit,
        unit_value: unitValue,
        notes,
        source_project_id: sourceProjectId || null,
      };
      if (initial) await api.updateAsset(initial.id, body);
      else await api.createAsset(body);
      onSaved();
      onClose();
    } catch (err) {
      setError(apiError(err instanceof Error ? err.message : "", t));
    }
  }

  async function remove() {
    if (!initial) return;
    if (!confirm(t("form.asset.deleteAsk"))) return;
    await api.deleteAsset(initial.id);
    onSaved();
    onClose();
  }

  return (
    <Modal title={initial ? t("form.asset.edit") : t("form.asset.new")} subtitle={t("form.asset.sub")} onClose={onClose}>
      <form onSubmit={onSubmit}>
        <ErrorBox message={error} />
        <Field label={t("common.name")}>
          <input className={inputClass()} value={name} onChange={(e) => setName(e.target.value)} placeholder={t("form.asset.namePh")} required />
        </Field>
        <Field label={t("common.type")}>
          <select className={inputClass()} value={type} onChange={(e) => changeType(e.target.value)}>
            {ASSET_TYPES.map((item) => (
              <option key={item.id} value={item.id}>
                {label(item.id)}
              </option>
            ))}
          </select>
        </Field>
        <div className="grid grid-cols-2 gap-3">
          <Field label={t("form.asset.qty")}>
            <input className={inputClass()} inputMode="decimal" value={quantity} onChange={(e) => setQuantity(e.target.value)} />
          </Field>
          <Field label={t("form.asset.unit")}>
            <input className={inputClass()} value={unit} onChange={(e) => setUnit(e.target.value)} />
          </Field>
        </div>
        <Field label={t("form.asset.unitValue")}>
          <MoneyInput value={unitValue} onChange={setUnitValue} />
        </Field>
        {isMarketAsset(type) && live > 0 && (
          <div className="mb-3 rounded-2xl border border-gold-400/20 bg-gold-400/5 px-4 py-3">
            <div className="text-xs text-white/50">{t("form.asset.live")}</div>
            <div className="mt-1 text-sm font-bold text-gold-200">
              {toman(live, true, locale)}
              {(type === "gold18" || type === "gold24" || type === "gold") && <span className="ms-1 text-white/40">· {t("assets.perGram")}</span>}
            </div>
            <button
              type="button"
              className="mt-2 text-xs font-bold text-gold-300"
              onClick={() => setUnitValue(live)}
            >
              {t("form.asset.useLive")}
            </button>
          </div>
        )}
        <div className="mb-3 text-sm text-white/55">
          {toman(bookTotal, true, locale)}
          <FxHint toman={marketTotal} rates={rates} />
        </div>
        <Field label={t("common.note")}>
          <input className={inputClass()} value={notes} onChange={(e) => setNotes(e.target.value)} />
        </Field>
        <div className="mt-2 flex gap-2">
          <Btn type="submit" className="flex-1">
            {t("form.asset.save")}
          </Btn>
          {initial && (
            <Btn kind="danger" onClick={remove}>
              {t("common.delete")}
            </Btn>
          )}
        </div>
      </form>
    </Modal>
  );
}

export function ConvertAssetForm({
  projectId,
  defaultName,
  defaultType,
  spent,
  onClose,
  onSaved,
}: {
  projectId: string;
  defaultName: string;
  defaultType: string;
  spent: number;
  onClose: () => void;
  onSaved: () => void;
}) {
  const { t, label, locale } = useI18n();
  const [name, setName] = useState(defaultName);
  const [type, setType] = useState(defaultType || "house");
  const [quantity, setQuantity] = useState("1");
  const [unit, setUnit] = useState(label(ASSET_TYPES.find((item) => item.id === (defaultType || "house"))?.unit || "property"));
  const [unitValue, setUnitValue] = useState(spent);
  const [error, setError] = useState("");

  function changeType(next: string) {
    setType(next);
    const found = ASSET_TYPES.find((item) => item.id === next);
    if (found) setUnit(label(found.unit));
  }

  async function onSubmit(e: FormEvent) {
    e.preventDefault();
    try {
      await api.convertAsset(projectId, {
        name,
        type,
        quantity: Number(quantity) || 1,
        unit,
        unit_value: unitValue,
      });
      onSaved();
      onClose();
    } catch (err) {
      setError(apiError(err instanceof Error ? err.message : "", t));
    }
  }

  return (
    <Modal title={t("form.convert.title")} subtitle={t("form.convert.sub")} onClose={onClose}>
      <form onSubmit={onSubmit}>
        <ErrorBox message={error} />
        <Field label={t("form.convert.assetName")}>
          <input className={inputClass()} value={name} onChange={(e) => setName(e.target.value)} required />
        </Field>
        <Field label={t("common.type")}>
          <select className={inputClass()} value={type} onChange={(e) => changeType(e.target.value)}>
            {ASSET_TYPES.map((item) => (
              <option key={item.id} value={item.id}>
                {label(item.id)}
              </option>
            ))}
          </select>
        </Field>
        <div className="grid grid-cols-2 gap-3">
          <Field label={t("form.asset.qty")}>
            <input className={inputClass()} inputMode="decimal" value={quantity} onChange={(e) => setQuantity(e.target.value)} />
          </Field>
          <Field label={t("form.asset.unit")}>
            <input className={inputClass()} value={unit} onChange={(e) => setUnit(e.target.value)} />
          </Field>
        </div>
        <Field label={t("form.convert.unitValue")}>
          <MoneyInput value={unitValue} onChange={setUnitValue} />
        </Field>
        <p className="mb-3 text-xs text-white/40">{t("form.convert.hint", { amount: toman(spent, true, locale) })}</p>
        <Btn type="submit" className="w-full">
          {t("form.convert.save")}
        </Btn>
      </form>
    </Modal>
  );
}

export function MoneyMoveForm({
  mode,
  projectId,
  accounts,
  onClose,
  onSaved,
}: {
  mode: "contribute" | "withdraw";
  projectId: string;
  accounts: { id: string; name: string; balance: number }[];
  onClose: () => void;
  onSaved: () => void;
}) {
  const { t } = useI18n();
  const [accountId, setAccountId] = useState(accounts[0]?.id || "");
  const [amount, setAmount] = useState(0);
  const [desc, setDesc] = useState(mode === "contribute" ? t("form.move.inDesc") : t("form.move.outDesc"));
  const [error, setError] = useState("");

  async function onSubmit(e: FormEvent) {
    e.preventDefault();
    try {
      const body = { account_id: accountId, amount, description: desc };
      if (mode === "contribute") await api.contribute(projectId, body);
      else await api.withdraw(projectId, body);
      onSaved();
      onClose();
    } catch (err) {
      setError(apiError(err instanceof Error ? err.message : "", t));
    }
  }

  return (
    <Modal title={mode === "contribute" ? t("form.move.in") : t("form.move.out")} onClose={onClose}>
      <form onSubmit={onSubmit}>
        <ErrorBox message={error} />
        <Field label={t("common.account")}>
          <select className={inputClass()} value={accountId} onChange={(e) => setAccountId(e.target.value)}>
            {accounts.map((a) => (
              <option key={a.id} value={a.id}>
                {a.name}
              </option>
            ))}
          </select>
        </Field>
        <Field label={t("common.amount")}>
          <MoneyInput value={amount} onChange={setAmount} />
        </Field>
        <Field label={t("common.description")}>
          <input className={inputClass()} value={desc} onChange={(e) => setDesc(e.target.value)} />
        </Field>
        <Btn type="submit" className="w-full">
          {t("common.confirm")}
        </Btn>
      </form>
    </Modal>
  );
}

export function TxForm({
  accounts,
  projects,
  onClose,
  onSaved,
}: {
  accounts: { id: string; name: string }[];
  projects: Project[];
  onClose: () => void;
  onSaved: () => void;
}) {
  const { t, label, locale } = useI18n();
  const [type, setType] = useState<"income" | "expense" | "transfer">("expense");
  const [accountId, setAccountId] = useState(accounts[0]?.id || "");
  const [toId, setToId] = useState(accounts[1]?.id || accounts[0]?.id || "");
  const [projectId, setProjectId] = useState("");
  const [itemId, setItemId] = useState("");
  const [amount, setAmount] = useState(0);
  const [category, setCategory] = useState(EXPENSE_CATS[0]);
  const [description, setDescription] = useState("");
  const [date, setDate] = useState(new Date().toISOString().slice(0, 10));
  const [error, setError] = useState("");

  const selectedProject = projects.find((p) => p.id === projectId);
  const projectItems = selectedProject?.items || [];

  useEffect(() => {
    setCategory(type === "income" ? INCOME_CATS[0] : type === "expense" ? EXPENSE_CATS[0] : "transfer");
  }, [type]);

  function pickProject(id: string) {
    setProjectId(id);
    setItemId("");
  }

  function pickItem(id: string) {
    setItemId(id);
    const item = projectItems.find((it) => it.id === id);
    if (!item) return;
    if (!description) setDescription(item.name);
    const remain = item.planned_amount - item.paid_amount;
    if (remain > 0 && amount === 0) setAmount(remain);
  }

  async function onSubmit(e: FormEvent) {
    e.preventDefault();
    try {
      await api.createTx({
        type,
        amount: Math.round(amount),
        account_id: accountId,
        to_account_id: type === "transfer" ? toId : null,
        project_id: projectId || null,
        item_id: type === "expense" && itemId ? itemId : null,
        category,
        description,
        occurred_at: date,
      });
      onSaved();
      onClose();
    } catch (err) {
      setError(apiError(err instanceof Error ? err.message : "", t));
    }
  }

  const cats = type === "income" ? INCOME_CATS : type === "expense" ? EXPENSE_CATS : ["transfer"];

  return (
    <Modal title={t("form.tx.title")} subtitle={t("form.tx.sub")} onClose={onClose} wide>
      <form onSubmit={onSubmit}>
        <ErrorBox message={error} />
        <div className="mb-4 grid grid-cols-3 gap-2">
          {(["expense", "income", "transfer"] as const).map((id) => (
            <button
              key={id}
              type="button"
              onClick={() => setType(id)}
              className={`min-h-11 rounded-2xl py-2.5 text-sm font-bold ${type === id ? "bg-gold-400 text-ink-950" : "bg-white/8 text-white/70"}`}
            >
              {label(id)}
            </button>
          ))}
        </div>
        <div className="grid gap-3 md:grid-cols-2">
          <Field label={type === "transfer" ? t("common.fromAccount") : t("common.account")}>
            <select className={inputClass()} value={accountId} onChange={(e) => setAccountId(e.target.value)}>
              {accounts.map((a) => (
                <option key={a.id} value={a.id}>
                  {a.name}
                </option>
              ))}
            </select>
          </Field>
          {type === "transfer" ? (
            <Field label={t("common.toAccount")}>
              <select className={inputClass()} value={toId} onChange={(e) => setToId(e.target.value)}>
                {accounts.map((a) => (
                  <option key={a.id} value={a.id}>
                    {a.name}
                  </option>
                ))}
              </select>
            </Field>
          ) : (
            <Field label={t("form.tx.category")}>
              <select className={inputClass()} value={category} onChange={(e) => setCategory(e.target.value)}>
                {cats.map((c) => (
                  <option key={c} value={c}>
                    {label(c)}
                  </option>
                ))}
              </select>
            </Field>
          )}
        </div>
        {type === "expense" && (
          <>
            <Field label={t("form.tx.projectOpt")}>
              <select className={inputClass()} value={projectId} onChange={(e) => pickProject(e.target.value)}>
                <option value="">{t("form.tx.noProject")}</option>
                {projects.map((p) => (
                  <option key={p.id} value={p.id}>
                    {p.name}
                    {p.items?.length ? ` (${p.items.length})` : ""}
                  </option>
                ))}
              </select>
            </Field>
            {projectItems.length > 0 && (
              <Field label={t("form.tx.item")}>
                <select className={inputClass()} value={itemId} onChange={(e) => pickItem(e.target.value)}>
                  <option value="">{t("form.tx.noItem")}</option>
                  {projectItems.map((it) => {
                    const remain = it.planned_amount - it.paid_amount;
                    return (
                      <option key={it.id} value={it.id}>
                        {it.name}
                        {remain > 0 ? ` · ${formatMoney(remain, true, locale)}` : ""}
                      </option>
                    );
                  })}
                </select>
                <p className="mt-1.5 text-xs text-white/40">{t("form.tx.itemHint")}</p>
              </Field>
            )}
          </>
        )}
        <Field label={t("common.amount")}>
          <MoneyInput value={amount} onChange={setAmount} />
        </Field>
        <div className="grid gap-3 md:grid-cols-2">
          <Field label={t("common.date")}>
            <input type="date" className={inputClass()} value={date} onChange={(e) => setDate(e.target.value)} />
          </Field>
          <Field label={t("common.description")}>
            <input className={inputClass()} value={description} onChange={(e) => setDescription(e.target.value)} placeholder={t("form.tx.descPh")} />
          </Field>
        </div>
        <Btn type="submit" className="w-full">
          {t("form.tx.save")}
        </Btn>
      </form>
    </Modal>
  );
}

export function DebtForm({
  initial,
  onClose,
  onSaved,
}: {
  initial?: Debt;
  onClose: () => void;
  onSaved: () => void;
}) {
  const { t, label } = useI18n();
  const [name, setName] = useState(initial?.name || "");
  const [type, setType] = useState(initial?.type || "loan");
  const [creditor, setCreditor] = useState(initial?.creditor || "");
  const [total, setTotal] = useState(initial?.total_amount || 0);
  const [notes, setNotes] = useState(initial?.notes || "");
  const [color, setColor] = useState(initial?.color || DEBT_COLORS[0]);
  const [hasSchedule, setHasSchedule] = useState(initial?.has_schedule ?? true);
  const [start, setStart] = useState(initial?.start_date ? initial.start_date.slice(0, 10) : "");
  const [end, setEnd] = useState(initial?.end_date ? initial.end_date.slice(0, 10) : "");
  const [monthly, setMonthly] = useState(initial?.monthly_amount || 0);
  const [dueDay, setDueDay] = useState(initial?.due_day || 1);
  const [error, setError] = useState("");
  const [loading, setLoading] = useState(false);

  async function onSubmit(e: FormEvent) {
    e.preventDefault();
    setLoading(true);
    setError("");
    try {
      const body = {
        name,
        type,
        creditor,
        total_amount: total,
        notes,
        color,
        has_schedule: hasSchedule,
        start_date: start || null,
        end_date: end || null,
        monthly_amount: monthly,
        due_day: dueDay,
      };
      if (initial) await api.updateDebt(initial.id, { ...body, has_schedule: initial.has_schedule });
      else await api.createDebt(body);
      onSaved();
      onClose();
    } catch (err) {
      setError(apiError(err instanceof Error ? err.message : "", t));
    } finally {
      setLoading(false);
    }
  }

  async function remove() {
    if (!initial) return;
    if (!confirm(t("debts.deleteAsk"))) return;
    await api.deleteDebt(initial.id);
    onSaved();
    onClose();
  }

  return (
    <Modal title={initial ? t("form.debt.edit") : t("form.debt.new")} subtitle={t("form.debt.sub")} onClose={onClose}>
      <form onSubmit={onSubmit}>
        <ErrorBox message={error} />
        <Field label={t("common.name")}>
          <input className={inputClass()} value={name} onChange={(e) => setName(e.target.value)} placeholder={t("form.debt.namePh")} required />
        </Field>
        <div className="grid grid-cols-2 gap-3">
          <Field label={t("common.type")}>
            <select className={inputClass()} value={type} onChange={(e) => setType(e.target.value)}>
              {DEBT_TYPES.map((id) => (
                <option key={id} value={id}>
                  {label(id)}
                </option>
              ))}
            </select>
          </Field>
          <Field label={t("form.debt.creditor")}>
            <input className={inputClass()} value={creditor} onChange={(e) => setCreditor(e.target.value)} placeholder={t("form.debt.creditorPh")} />
          </Field>
        </div>
        {!initial && (
          <Field label={t("form.debt.total")}>
            <MoneyInput value={total} onChange={setTotal} />
          </Field>
        )}
        {!initial && (
          <label className="mb-4 flex items-center gap-3 rounded-2xl bg-white/5 px-4 py-3 text-sm">
            <input type="checkbox" checked={hasSchedule} onChange={(e) => setHasSchedule(e.target.checked)} />
            {t("form.debt.hasSchedule")}
          </label>
        )}
        {!initial && hasSchedule && (
          <>
            <div className="grid grid-cols-2 gap-3">
              <Field label={t("form.debt.start")}>
                <input type="date" className={inputClass()} value={start} onChange={(e) => setStart(e.target.value)} required />
              </Field>
              <Field label={t("form.debt.end")}>
                <input type="date" className={inputClass()} value={end} onChange={(e) => setEnd(e.target.value)} required />
              </Field>
            </div>
            <Field label={t("form.debt.monthly")}>
              <MoneyInput value={monthly} onChange={setMonthly} />
            </Field>
            <Field label={t("form.debt.dueDay")}>
              <input
                type="number"
                min={1}
                max={31}
                className={inputClass()}
                value={dueDay}
                onChange={(e) => setDueDay(Number(e.target.value) || 1)}
              />
            </Field>
          </>
        )}
        {!initial && !hasSchedule && (
          <Field label={t("common.date")}>
            <input type="date" className={inputClass()} value={start} onChange={(e) => setStart(e.target.value)} />
          </Field>
        )}
        <Field label={t("common.note")}>
          <input className={inputClass()} value={notes} onChange={(e) => setNotes(e.target.value)} />
        </Field>
        <Field label={t("common.color")}>
          <div className="flex flex-wrap gap-2">
            {DEBT_COLORS.map((c) => (
              <button
                key={c}
                type="button"
                onClick={() => setColor(c)}
                className={`h-8 w-8 rounded-full ${color === c ? "ring-2 ring-white" : ""}`}
                style={{ background: c }}
              />
            ))}
          </div>
        </Field>
        <div className="mt-4 flex gap-2">
          <Btn type="submit" className="flex-1" disabled={loading}>
            {loading ? t("common.saving") : t("form.debt.save")}
          </Btn>
          {initial && (
            <Btn kind="danger" onClick={remove}>
              {t("common.delete")}
            </Btn>
          )}
        </div>
      </form>
    </Modal>
  );
}

export function PayDebtForm({
  accounts,
  amountHint,
  onSubmitPay,
  onClose,
}: {
  accounts: { id: string; name: string }[];
  amountHint: number;
  onSubmitPay: (body: { account_id: string; amount: number; description: string }) => Promise<void>;
  onClose: () => void;
}) {
  const { t } = useI18n();
  const [accountId, setAccountId] = useState(accounts[0]?.id || "");
  const [amount, setAmount] = useState(Math.max(0, amountHint));
  const [desc, setDesc] = useState("");
  const [error, setError] = useState("");

  async function onSubmit(e: FormEvent) {
    e.preventDefault();
    try {
      await onSubmitPay({ account_id: accountId, amount, description: desc });
      onClose();
    } catch (err) {
      setError(apiError(err instanceof Error ? err.message : "", t));
    }
  }

  return (
    <Modal title={t("form.payDebt.title")} subtitle={t("form.payDebt.sub")} onClose={onClose}>
      <form onSubmit={onSubmit}>
        <ErrorBox message={error} />
        <Field label={t("common.fromAccount")}>
          <select className={inputClass()} value={accountId} onChange={(e) => setAccountId(e.target.value)}>
            {accounts.map((a) => (
              <option key={a.id} value={a.id}>
                {a.name}
              </option>
            ))}
          </select>
        </Field>
        <Field label={t("common.amount")}>
          <MoneyInput value={amount} onChange={setAmount} />
        </Field>
        <Field label={t("common.description")}>
          <input className={inputClass()} value={desc} onChange={(e) => setDesc(e.target.value)} />
        </Field>
        <Btn type="submit" className="w-full">
          {t("common.pay")}
        </Btn>
      </form>
    </Modal>
  );
}
