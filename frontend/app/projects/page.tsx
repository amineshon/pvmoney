"use client";

import { useEffect, useState } from "react";
import { api } from "@/lib/api";
import { Project } from "@/lib/types";
import { ProjectCard } from "@/components/ProjectCard";
import { ProjectForm } from "@/components/forms";
import { Btn, Empty } from "@/components/ui";
import { toman } from "@/lib/format";
import { useI18n } from "@/lib/i18n";

export default function ProjectsPage() {
  const { t, locale } = useI18n();
  const [items, setItems] = useState<Project[] | null>(null);
  const [open, setOpen] = useState(false);

  function load() {
    api.projects().then(setItems);
  }
  useEffect(() => {
    load();
  }, []);

  if (!items) return <div className="animate-pulse text-white/40">{t("projects.loading")}</div>;
  const spent = items.reduce((s, p) => s + p.current_amount, 0);

  return (
    <div className="mx-auto max-w-6xl">
      <div className="mb-8 flex flex-wrap items-end justify-between gap-4">
        <div>
          <h1 className="text-3xl font-extrabold">{t("projects.title")}</h1>
          <p className="mt-2 text-white/45">
            {t("projects.spent")} {toman(spent, true, locale)}
          </p>
        </div>
        <Btn onClick={() => setOpen(true)}>{t("projects.add")}</Btn>
      </div>
      {items.length === 0 ? (
        <Empty title={t("projects.empty.title")} text={t("projects.empty.text")} action={<Btn onClick={() => setOpen(true)}>{t("projects.empty.cta")}</Btn>} />
      ) : (
        <div className="grid gap-5 md:grid-cols-2">
          {items.map((p) => (
            <ProjectCard key={p.id} project={p} />
          ))}
        </div>
      )}
      {open && <ProjectForm onClose={() => setOpen(false)} onSaved={load} />}
    </div>
  );
}
