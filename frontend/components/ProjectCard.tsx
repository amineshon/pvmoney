"use client";

import Link from "next/link";
import { Project } from "@/lib/types";
import { progress, toman, faDate } from "@/lib/format";
import { CheckCircle2 } from "lucide-react";
import { useI18n } from "@/lib/i18n";

export function ProjectCard({ project }: { project: Project }) {
  const { t, label, locale } = useI18n();
  const pct = progress(project.current_amount, project.target_amount);
  const done = project.status === "completed" || (project.target_amount > 0 && pct >= 100);
  const remaining = project.target_amount - project.current_amount;
  const itemCount = project.items?.length || 0;

  return (
    <Link href={`/projects/${project.id}`} className="glass group relative block overflow-hidden rounded-[28px] p-5">
      <div
        className="absolute inset-x-0 bottom-0 opacity-80 transition-all duration-700"
        style={{ height: `${Math.max(8, pct)}%`, background: `linear-gradient(180deg, ${project.color}33, ${project.color}88)` }}
      >
        <svg className="wave absolute -top-4 w-[220%] text-white/20" viewBox="0 0 120 12" preserveAspectRatio="none">
          <path fill="currentColor" d="M0 6 Q 15 0 30 6 T 60 6 T 90 6 T 120 6 V12 H0Z" />
        </svg>
      </div>
      <div className="relative">
        <div className="flex items-start justify-between gap-3">
          <div>
            <div className="text-xs text-white/45">{done ? t("projects.done") : t("projects.active")}</div>
            <h3 className="mt-1 text-lg font-bold">{project.name}</h3>
          </div>
          {done ? (
            <CheckCircle2 className="h-6 w-6 text-emerald-400" />
          ) : (
            <div
              className="flex h-12 w-12 items-center justify-center rounded-full text-xs font-bold"
              style={{ background: `${project.color}22`, color: project.color, boxShadow: `inset 0 0 0 2px ${project.color}55` }}
            >
              {pct}%
            </div>
          )}
        </div>
        {project.description && <p className="mt-2 line-clamp-2 text-sm text-white/50">{project.description}</p>}
        <div className="mt-5 grid grid-cols-2 gap-3">
          <div>
            <div className="text-[11px] text-white/40">{t("projects.spentLabel")}</div>
            <div className="font-bold text-rose-300">{toman(project.current_amount, true, locale)}</div>
          </div>
          <div className="text-end">
            <div className="text-[11px] text-white/40">{t("projects.budget")}</div>
            <div className="font-bold text-gold-200">{toman(project.target_amount, true, locale)}</div>
          </div>
        </div>
        <div className="mt-3 flex items-center justify-between text-[11px] text-white/45">
          <span>
            {t("projects.remaining")} {toman(remaining, true, locale)}
          </span>
          {itemCount > 0 && (
            <span>
              {t("projects.itemsSum")} · {itemCount}
            </span>
          )}
        </div>
        {project.result_asset_type && (
          <div className="mt-3 text-[11px] text-gold-200/80">
            {t("projects.assetDest")} {label(project.result_asset_type)}
          </div>
        )}
        {project.deadline && (
          <div className="mt-1 text-[11px] text-white/35">
            {t("projects.deadline")} {faDate(project.deadline, false, locale)}
          </div>
        )}
      </div>
    </Link>
  );
}
