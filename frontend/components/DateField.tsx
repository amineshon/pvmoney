"use client";

import { useEffect, useLayoutEffect, useMemo, useRef, useState } from "react";
import { createPortal } from "react-dom";
import { CalendarDays, ChevronLeft, ChevronRight, X } from "lucide-react";
import { useI18n } from "@/lib/i18n";
import {
  calendarKind,
  clampDay,
  daysInMonth,
  formatCalNum,
  formatDisplayDate,
  gregorianToParts,
  monthName,
  partsToISO,
  sameDay,
  shiftMonth,
  todayParts,
  weekdayIndex,
  weekdayLabels,
  weekStart,
  yearRange,
  type YMD,
} from "@/lib/calendar";
import { inputClass } from "./ui";

export function DateField({
  value,
  onChange,
  required,
  allowEmpty,
}: {
  value: string;
  onChange: (iso: string) => void;
  required?: boolean;
  allowEmpty?: boolean;
}) {
  const { t, locale } = useI18n();
  const kind = calendarKind(locale);
  const canClear = allowEmpty ?? !required;
  const triggerRef = useRef<HTMLButtonElement>(null);
  const panelRef = useRef<HTMLDivElement>(null);
  const [open, setOpen] = useState(false);
  const [view, setView] = useState<"days" | "months" | "years">("days");
  const [pos, setPos] = useState<{ mode: "sheet" | "pop"; top?: number; left?: number; width?: number }>({ mode: "pop" });

  const selected = useMemo(() => (value ? gregorianToParts(value, kind) : null), [value, kind]);
  const today = useMemo(() => todayParts(kind), [kind]);
  const [cursor, setCursor] = useState<YMD>(selected || today);

  useEffect(() => {
    if (open) {
      setCursor(selected || todayParts(kind));
      setView("days");
    }
  }, [open, kind, selected]);

  function place() {
    const el = triggerRef.current;
    if (!el) return;
    const r = el.getBoundingClientRect();
    if (window.innerWidth < 640) {
      setPos({ mode: "sheet" });
      return;
    }
    const width = Math.max(320, Math.min(360, r.width));
    const height = 420;
    const spaceBelow = window.innerHeight - r.bottom;
    const top = spaceBelow > height + 16 || r.top < height ? r.bottom + 8 : r.top - height - 8;
    let left = r.left;
    if (document.documentElement.dir === "rtl") left = r.right - width;
    if (left + width > window.innerWidth - 12) left = window.innerWidth - width - 12;
    if (left < 12) left = 12;
    setPos({ mode: "pop", top: Math.max(12, top), left, width });
  }

  useLayoutEffect(() => {
    if (!open) return;
    place();
    const on = () => place();
    window.addEventListener("resize", on);
    window.addEventListener("scroll", on, true);
    return () => {
      window.removeEventListener("resize", on);
      window.removeEventListener("scroll", on, true);
    };
  }, [open]);

  useEffect(() => {
    if (!open) return;
    function onKey(e: KeyboardEvent) {
      if (e.key === "Escape") setOpen(false);
    }
    function onDown(e: MouseEvent) {
      const t = e.target as Node;
      if (triggerRef.current?.contains(t) || panelRef.current?.contains(t)) return;
      setOpen(false);
    }
    document.addEventListener("keydown", onKey);
    document.addEventListener("mousedown", onDown);
    return () => {
      document.removeEventListener("keydown", onKey);
      document.removeEventListener("mousedown", onDown);
    };
  }, [open]);

  function pick(p: YMD) {
    onChange(partsToISO(p, kind));
    setOpen(false);
  }

  const start = weekStart(locale);
  const labels = weekdayLabels(locale);
  const firstWeekday = weekdayIndex({ y: cursor.y, m: cursor.m, d: 1 }, kind);
  const lead = (firstWeekday - start + 7) % 7;
  const dim = daysInMonth(cursor.y, cursor.m, kind);
  const years = yearRange(kind, cursor.y);
  const yearAnchor = years[0] ?? cursor.y;
  const yearStart = Math.floor((cursor.y - yearAnchor) / 12) * 12 + yearAnchor;
  const yearPage = years.filter((y) => y >= yearStart && y < yearStart + 12);

  const panel = open
    ? createPortal(
        <>
        {pos.mode === "sheet" && (
          <div className="fixed inset-0 z-[119] bg-black/50" onClick={() => setOpen(false)} />
        )}
        <div
          ref={panelRef}
          role="dialog"
          aria-label={t("date.pick")}
          className={
            pos.mode === "sheet"
              ? "fixed inset-x-3 bottom-3 z-[120] rounded-[28px] border border-white/12 bg-[#121722] p-4 shadow-card"
              : "fixed z-[120] rounded-[24px] border border-white/12 bg-[#121722] p-4 shadow-card"
          }
          style={pos.mode === "pop" ? { top: pos.top, left: pos.left, width: pos.width } : undefined}
          onMouseDown={(e) => e.preventDefault()}
        >
          <div className="mb-3 flex items-center justify-between gap-2">
            <button
              type="button"
              className="flex h-10 w-10 items-center justify-center rounded-xl text-white/70 hover:bg-white/8"
              onClick={() => {
                if (view === "days") setCursor((c) => ({ ...shiftMonth(c.y, c.m, -1, kind), d: 1 }));
                else if (view === "years") setCursor((c) => ({ ...c, y: c.y - 12 }));
              }}
              aria-label={t("date.prev")}
            >
              <ChevronRight className="h-5 w-5 ltr:hidden" />
              <ChevronLeft className="hidden h-5 w-5 ltr:block" />
            </button>
            <div className="flex min-w-0 flex-1 items-center justify-center gap-1">
              <button
                type="button"
                className="rounded-xl px-2.5 py-1.5 text-sm font-bold hover:bg-white/8"
                onClick={() => setView((v) => (v === "months" ? "days" : "months"))}
              >
                {monthName(cursor.m, locale)}
              </button>
              <button
                type="button"
                className="rounded-xl px-2.5 py-1.5 text-sm font-bold hover:bg-white/8"
                onClick={() => setView((v) => (v === "years" ? "days" : "years"))}
              >
                {formatCalNum(cursor.y, locale)}
              </button>
            </div>
            <button
              type="button"
              className="flex h-10 w-10 items-center justify-center rounded-xl text-white/70 hover:bg-white/8"
              onClick={() => {
                if (view === "days") setCursor((c) => ({ ...shiftMonth(c.y, c.m, 1, kind), d: 1 }));
                else if (view === "years") setCursor((c) => ({ ...c, y: c.y + 12 }));
              }}
              aria-label={t("date.next")}
            >
              <ChevronLeft className="h-5 w-5 ltr:hidden" />
              <ChevronRight className="hidden h-5 w-5 ltr:block" />
            </button>
          </div>

          {view === "months" && (
            <div className="grid grid-cols-3 gap-2">
              {Array.from({ length: 12 }, (_, i) => i + 1).map((m) => (
                <button
                  key={m}
                  type="button"
                  onClick={() => {
                    setCursor((c) => clampDay(c.y, m, c.d, kind));
                    setView("days");
                  }}
                  className={`min-h-11 rounded-2xl px-2 text-sm font-semibold ${
                    cursor.m === m ? "bg-gold-400 text-ink-950" : "bg-white/6 hover:bg-white/10"
                  }`}
                >
                  {monthName(m, locale)}
                </button>
              ))}
            </div>
          )}

          {view === "years" && (
            <div className="grid grid-cols-3 gap-2">
              {yearPage.map((y) => (
                <button
                  key={y}
                  type="button"
                  onClick={() => {
                    setCursor((c) => clampDay(y, c.m, c.d, kind));
                    setView("days");
                  }}
                  className={`min-h-11 rounded-2xl text-sm font-semibold ${
                    cursor.y === y ? "bg-gold-400 text-ink-950" : "bg-white/6 hover:bg-white/10"
                  }`}
                >
                  {formatCalNum(y, locale)}
                </button>
              ))}
            </div>
          )}

          {view === "days" && (
            <>
              <div className="mb-1 grid grid-cols-7">
                {labels.map((w) => (
                  <div key={w} className="py-1 text-center text-[11px] font-bold text-white/35">
                    {w}
                  </div>
                ))}
              </div>
              <div className="grid grid-cols-7 gap-y-1">
                {Array.from({ length: lead + dim }, (_, i) => {
                  if (i < lead) return <div key={`e${i}`} />;
                  const d = i - lead + 1;
                  const p = { y: cursor.y, m: cursor.m, d };
                  const isSel = selected ? sameDay(p, selected) : false;
                  const isToday = sameDay(p, today);
                  return (
                    <button
                      key={d}
                      type="button"
                      onClick={() => pick(p)}
                      className={`mx-auto flex h-10 w-10 items-center justify-center rounded-2xl text-sm font-semibold transition ${
                        isSel
                          ? "bg-gold-400 text-ink-950 shadow-glow"
                          : isToday
                            ? "ring-1 ring-gold-400/50 text-gold-200 hover:bg-gold-400/10"
                            : "text-white/85 hover:bg-white/8"
                      }`}
                    >
                      {formatCalNum(d, locale)}
                    </button>
                  );
                })}
              </div>
            </>
          )}

          <div className="mt-4 flex items-center justify-between gap-2">
            <button
              type="button"
              className="rounded-xl px-3 py-2 text-sm font-bold text-gold-200 hover:bg-gold-400/10"
              onClick={() => pick(today)}
            >
              {t("date.today")}
            </button>
            <div className="flex gap-1">
              {canClear && (
                <button
                  type="button"
                  className="rounded-xl px-3 py-2 text-sm text-white/50 hover:bg-white/8"
                  onClick={() => {
                    onChange("");
                    setOpen(false);
                  }}
                >
                  {t("date.clear")}
                </button>
              )}
              <button
                type="button"
                className="flex h-10 w-10 items-center justify-center rounded-xl text-white/40 hover:bg-white/8 md:hidden"
                onClick={() => setOpen(false)}
                aria-label={t("date.close")}
              >
                <X className="h-4 w-4" />
              </button>
            </div>
          </div>
        </div>
        </>,
        document.body,
      )
    : null;

  return (
    <div className="relative">
      <input
        tabIndex={-1}
        required={required}
        value={value}
        onChange={() => undefined}
        className="pointer-events-none absolute h-0 w-0 opacity-0"
        aria-hidden
      />
      <button
        ref={triggerRef}
        type="button"
        onClick={() => setOpen((v) => !v)}
        aria-haspopup="dialog"
        aria-expanded={open}
        className={`${inputClass("flex items-center justify-between gap-3 text-start")} ${open ? "border-gold-400/55 ring-2 ring-gold-400/20" : ""}`}
      >
        <span className={value ? "font-semibold" : "text-white/30"}>{value ? formatDisplayDate(value, locale) : t("date.pick")}</span>
        <CalendarDays className="h-5 w-5 shrink-0 text-gold-300/80" />
      </button>
      {panel}
    </div>
  );
}
