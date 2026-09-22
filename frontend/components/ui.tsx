"use client";

import { motion, AnimatePresence } from "framer-motion";
import { X } from "lucide-react";
import { ReactNode } from "react";

export function Modal({
  title,
  subtitle,
  onClose,
  children,
  wide,
}: {
  title: string;
  subtitle?: string;
  onClose: () => void;
  children: ReactNode;
  wide?: boolean;
}) {
  return (
    <AnimatePresence>
      <motion.div
        className="fixed inset-0 z-[80] flex items-end justify-center bg-black/65 p-0 backdrop-blur-sm md:items-center md:p-6"
        initial={{ opacity: 0 }}
        animate={{ opacity: 1 }}
        exit={{ opacity: 0 }}
        onClick={onClose}
      >
        <motion.div
          onClick={(e) => e.stopPropagation()}
          initial={{ y: 48, opacity: 0 }}
          animate={{ y: 0, opacity: 1 }}
          className={`w-full overflow-y-auto overscroll-contain rounded-t-[28px] border border-white/10 bg-[#10141c] p-5 shadow-card md:rounded-3xl md:p-6 ${
            wide ? "max-w-2xl" : "max-w-lg"
          } max-h-[min(92dvh,840px)] pb-[max(1.25rem,env(safe-area-inset-bottom))]`}
        >
          <div className="mb-5 flex items-start justify-between gap-4">
            <div>
              <h2 className="text-xl font-bold">{title}</h2>
              {subtitle && <p className="mt-1 text-sm text-white/50">{subtitle}</p>}
            </div>
            <button onClick={onClose} className="min-h-11 min-w-11 rounded-xl p-2 text-white/50 hover:bg-white/5">
              <X className="h-5 w-5" />
            </button>
          </div>
          {children}
        </motion.div>
      </motion.div>
    </AnimatePresence>
  );
}

export function Field({ label, children }: { label: string; children: ReactNode }) {
  return (
    <label className="mb-3 block">
      <span className="mb-1.5 block text-xs font-semibold tracking-wide text-white/65">{label}</span>
      {children}
    </label>
  );
}

export function inputClass(extra = "") {
  return `w-full min-h-12 rounded-2xl border border-white/14 bg-[#121722] px-4 py-3 text-[16px] text-white shadow-[inset_0_1px_0_rgba(255,255,255,0.04)] outline-none transition placeholder:text-white/30 focus:border-gold-400/55 focus:bg-[#171c28] focus:ring-2 focus:ring-gold-400/20 ${extra}`;
}

export function Btn({
  children,
  onClick,
  type = "button",
  kind = "gold",
  disabled,
  className = "",
}: {
  children: ReactNode;
  onClick?: () => void;
  type?: "button" | "submit";
  kind?: "gold" | "ghost" | "danger";
  disabled?: boolean;
  className?: string;
}) {
  const map = {
    gold: "bg-gradient-to-r from-gold-400 to-gold-700 text-ink-950 shadow-glow hover:brightness-110 rtl:bg-gradient-to-l rtl:from-gold-700 rtl:to-gold-400",
    ghost: "bg-white/8 text-white hover:bg-white/12",
    danger: "bg-rose-500/15 text-rose-300 hover:bg-rose-500/25",
  };
  return (
    <button
      type={type}
      disabled={disabled}
      onClick={onClick}
      className={`min-h-12 rounded-2xl px-4 py-3 text-sm font-bold transition disabled:opacity-50 ${map[kind]} ${className}`}
    >
      {children}
    </button>
  );
}

export function Empty({ title, text, action }: { title: string; text: string; action?: ReactNode }) {
  return (
    <div className="glass flex flex-col items-center rounded-3xl px-6 py-16 text-center">
      <div className="mb-4 h-16 w-16 rounded-full bg-gold-400/10 ring-1 ring-gold-400/20" />
      <h3 className="text-lg font-bold">{title}</h3>
      <p className="mt-2 max-w-sm text-sm text-white/45">{text}</p>
      {action && <div className="mt-6">{action}</div>}
    </div>
  );
}

export function ErrorBox({ message }: { message: string }) {
  if (!message) return null;
  return <div className="mb-3 rounded-2xl bg-rose-500/10 px-4 py-3 text-sm text-rose-300">{message}</div>;
}
