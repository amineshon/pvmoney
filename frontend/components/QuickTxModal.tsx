"use client";

import { useEffect, useState } from "react";
import { api } from "@/lib/api";
import { Account, Project } from "@/lib/types";
import { TxForm } from "./forms";
import { Modal } from "./ui";
import { useI18n } from "@/lib/i18n";

export function QuickTxModal({ onClose }: { onClose: () => void }) {
  const { t } = useI18n();
  const [accounts, setAccounts] = useState<Account[]>([]);
  const [projects, setProjects] = useState<Project[]>([]);
  const [ready, setReady] = useState(false);

  useEffect(() => {
    Promise.all([api.accounts(), api.projects()]).then(([a, p]) => {
      setAccounts(a);
      setProjects(p);
      setReady(true);
    });
  }, []);

  if (!ready) return null;
  if (!accounts.length) {
    return (
      <Modal title={t("quick.needAccount.title")} subtitle={t("quick.needAccount.sub")} onClose={onClose}>
        <p className="text-sm text-white/55">{t("quick.needAccount.text")}</p>
      </Modal>
    );
  }
  return (
    <TxForm
      accounts={accounts}
      projects={projects}
      onClose={onClose}
      onSaved={() => {
        window.location.reload();
      }}
    />
  );
}
