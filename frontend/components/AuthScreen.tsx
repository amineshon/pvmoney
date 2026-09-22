"use client";

import { FormEvent, useEffect, useRef, useState } from "react";
import { AnimatePresence, motion } from "framer-motion";
import { Check, Languages, MessageCircle, Shield, Smartphone, Wallet } from "lucide-react";
import { api } from "@/lib/api";
import { useAuth } from "@/lib/auth";
import { LOCALES, apiError, useI18n } from "@/lib/i18n";
import type { AuthChallenge } from "@/lib/types";
import { Btn, ErrorBox, inputClass } from "./ui";

type Mode = "register" | "login";
type Step = "identity" | "telegram" | "code";

export function AuthScreen() {
  const { t, locale, setLocale } = useI18n();
  const { status, setSession } = useAuth();
  const [mode, setMode] = useState<Mode>(status?.mode || "register");
  const [step, setStep] = useState<Step>("identity");
  const [username, setUsername] = useState("");
  const [phone, setPhone] = useState("");
  const [code, setCode] = useState("");
  const [busy, setBusy] = useState(false);
  const [err, setErr] = useState("");
  const [challenge, setChallenge] = useState<AuthChallenge | null>(null);
  const [wait, setWait] = useState(0);

  const pollGen = useRef(0);

  useEffect(() => {
    if (status?.mode) setMode(status.mode);
  }, [status?.mode]);

  useEffect(() => {
    if (!wait) return;
    const id = setInterval(() => setWait((s) => Math.max(0, s - 1)), 1000);
    return () => clearInterval(id);
  }, [wait]);

  useEffect(() => {
    if (step !== "telegram" || !challenge?.id) return;
    const challengeId = challenge.id;
    const gen = pollGen.current;
    let stop = false;
    async function tick() {
      try {
        const next = await api.challenge(challengeId);
        if (stop || gen !== pollGen.current) return;
        setChallenge(next);
        if (next.status === "otp_sent") {
          setStep("code");
          setWait(30);
        }
      } catch (e) {
        if (!stop && gen === pollGen.current) setErr(apiError(e instanceof Error ? e.message : "", t));
      }
    }
    tick();
    const timer = setInterval(tick, 1600);
    return () => {
      stop = true;
      clearInterval(timer);
    };
  }, [step, challenge?.id, t]);

  function switchMode(next: Mode) {
    pollGen.current += 1;
    setMode(next);
    setStep("identity");
    setChallenge(null);
    setCode("");
    setErr("");
  }

  async function onIdentity(e: FormEvent) {
    e.preventDefault();
    setErr("");
    setBusy(true);
    try {
      if (mode === "register") {
        const ch = await api.registerStart({ username: username.trim(), phone, lang: locale });
        setChallenge(ch);
        setStep("telegram");
      } else {
        const ch = await api.loginStart({ username: username.trim(), lang: locale });
        setChallenge(ch);
        setStep("code");
        setWait(30);
      }
    } catch (e) {
      setErr(apiError(e instanceof Error ? e.message : "", t));
    } finally {
      setBusy(false);
    }
  }

  async function onVerify(e: FormEvent) {
    e.preventDefault();
    if (!challenge) return;
    setErr("");
    setBusy(true);
    try {
      const out = await api.verifyAuth({ challenge_id: challenge.id, code: code.replace(/\D/g, "") });
      setSession(out.token, out.user);
    } catch (e) {
      setErr(apiError(e instanceof Error ? e.message : "", t));
    } finally {
      setBusy(false);
    }
  }

  async function resend() {
    if (!challenge || wait) return;
    setErr("");
    setBusy(true);
    try {
      const next = await api.resendAuth({ challenge_id: challenge.id });
      setChallenge(next);
      setWait(30);
    } catch (e) {
      setErr(apiError(e instanceof Error ? e.message : "", t));
    } finally {
      setBusy(false);
    }
  }

  const botReady = status?.bot_ready !== false;
  const steps: Step[] = mode === "register" ? ["identity", "telegram", "code"] : ["identity", "code"];
  const view: Step = mode === "login" && step === "telegram" ? "identity" : step;

  return (
    <div className="relative flex min-h-dvh items-center justify-center bg-mesh bg-ink-950 px-4 py-10">
      <div className="absolute inset-x-0 top-0 flex items-center justify-between px-4 pt-[max(1rem,env(safe-area-inset-top))] md:px-8">
        <div className="flex items-center gap-3">
          <div className="flex h-10 w-10 items-center justify-center rounded-2xl bg-gradient-to-br from-gold-200 to-gold-700 shadow-glow">
            <Wallet className="h-5 w-5 text-ink-950" />
          </div>
          <div>
            <div className="text-sm font-extrabold gold-text">{t("app.name")}</div>
            <div className="text-[11px] text-white/40">{t("app.tagline")}</div>
          </div>
        </div>
        <div className="flex items-center gap-2 rounded-2xl bg-white/5 p-1">
          <Languages className="ms-2 h-3.5 w-3.5 text-white/35" />
          {LOCALES.map((l) => (
            <button
              key={l.id}
              onClick={() => setLocale(l.id)}
              className={`rounded-xl px-2.5 py-1.5 text-[11px] font-bold ${
                locale === l.id ? "bg-gold-400 text-ink-950" : "text-white/50 hover:text-white"
              }`}
            >
              {l.id.toUpperCase()}
            </button>
          ))}
        </div>
      </div>

      <motion.div
        initial={{ opacity: 0, y: 18 }}
        animate={{ opacity: 1, y: 0 }}
        className="relative w-full max-w-[440px] overflow-hidden rounded-[32px] border border-white/12 bg-[#121722]/96 p-6 shadow-card backdrop-blur-2xl md:p-8"
      >
        <div className="mb-6">
          <div className="text-xs font-semibold tracking-[0.2em] text-gold-200/70">{t("auth.kicker")}</div>
          <h1 className="mt-2 text-3xl font-black tracking-tight">
            {mode === "register" ? t("auth.registerTitle") : t("auth.loginTitle")}
          </h1>
          <p className="mt-2 text-sm leading-relaxed text-white/50">{t("auth.tagline")}</p>
        </div>

        <div className="mb-6 flex gap-2">
          {steps.map((s, i) => {
            const active = view === s;
            const done = steps.indexOf(view) > i;
            return (
              <div key={s} className="flex flex-1 flex-col gap-1.5">
                <div className={`h-1 rounded-full ${done || active ? "bg-gold-400" : "bg-white/10"}`} />
                <div className={`text-[10px] font-bold ${active ? "text-gold-200" : "text-white/35"}`}>
                  {i + 1}. {t(`auth.step.${s}`)}
                </div>
              </div>
            );
          })}
        </div>

        {!botReady && (
          <div className="mb-4 rounded-2xl bg-amber-500/10 px-4 py-3 text-sm text-amber-200">{t("auth.botMissing")}</div>
        )}
        <ErrorBox message={err} />

        <AnimatePresence mode="wait">
          {view === "identity" && (
            <motion.form
              key="id"
              onSubmit={onIdentity}
              initial={{ opacity: 0, x: 16 }}
              animate={{ opacity: 1, x: 0 }}
              exit={{ opacity: 0, x: -16 }}
              className="space-y-4"
            >
              <label className="block">
                <span className="mb-1.5 block text-xs font-semibold text-white/65">{t("auth.username")}</span>
                <input
                  className={inputClass()}
                  value={username}
                  onChange={(e) => setUsername(e.target.value)}
                  placeholder={t("auth.usernamePh")}
                  autoComplete="username"
                  autoFocus
                  required
                  minLength={3}
                  maxLength={24}
                />
                <span className="mt-1.5 block text-[11px] text-white/35">{t("auth.usernameHint")}</span>
              </label>
              {mode === "register" && (
                <label className="block">
                  <span className="mb-1.5 block text-xs font-semibold text-white/65">{t("auth.phone")}</span>
                  <div className="flex gap-2">
                    <div className="flex min-h-12 items-center rounded-2xl border border-white/14 bg-[#121722] px-3 text-sm text-white/55">
                      +98
                    </div>
                    <input
                      className={inputClass()}
                      dir="ltr"
                      inputMode="tel"
                      value={phone}
                      onChange={(e) => setPhone(formatPhone(e.target.value))}
                      placeholder="912 000 0000"
                      required
                    />
                  </div>
                  <span className="mt-1.5 block text-[11px] text-white/35">{t("auth.phoneHint")}</span>
                </label>
              )}
              <Btn type="submit" disabled={busy || !botReady} className="w-full">
                {busy ? t("common.saving") : t("auth.continue")}
              </Btn>
            </motion.form>
          )}

          {view === "telegram" && mode === "register" && challenge && (
            <motion.div key="tg" initial={{ opacity: 0, x: 16 }} animate={{ opacity: 1, x: 0 }} exit={{ opacity: 0, x: -16 }}>
              <div className="mb-4 rounded-2xl bg-white/4 p-4">
                <StatusPill status={challenge.status} t={t} />
                <p className="mt-3 text-sm text-white/55">{t("auth.tgBody")}</p>
              </div>
              <ol className="mb-5 space-y-3 text-sm">
                <Guide n={1} icon={MessageCircle} text={t("auth.tgStep1")} />
                <Guide n={2} icon={Smartphone} text={t("auth.tgStep2")} />
                <Guide n={3} icon={Shield} text={t("auth.tgStep3")} />
              </ol>
              <a
                href={challenge.bot_link || (challenge.bot_username ? `https://t.me/${challenge.bot_username}` : "https://t.me")}
                target="_blank"
                rel="noreferrer"
                className="mb-3 flex min-h-12 items-center justify-center rounded-2xl bg-gradient-to-r from-gold-400 to-gold-700 text-sm font-bold text-ink-950 shadow-glow rtl:bg-gradient-to-l rtl:from-gold-700 rtl:to-gold-400"
              >
                {t("auth.openTelegram")}
              </a>
              <button type="button" onClick={() => setStep("identity")} className="w-full py-2 text-sm text-white/40 hover:text-white/70">
                {t("auth.back")}
              </button>
            </motion.div>
          )}

          {view === "code" && challenge && (
            <motion.form
              key="code"
              onSubmit={onVerify}
              initial={{ opacity: 0, x: 16 }}
              animate={{ opacity: 1, x: 0 }}
              exit={{ opacity: 0, x: -16 }}
            >
              <p className="mb-1 text-sm text-white/60">{t("auth.codeBody")}</p>
              {challenge.phone_masked && (
                <p className="mb-4 font-mono text-xs text-gold-200/80" dir="ltr">
                  {challenge.phone_masked}
                </p>
              )}
              <OtpInput value={code} onChange={setCode} />
              <Btn type="submit" disabled={busy || code.replace(/\D/g, "").length !== 6} className="mt-5 w-full">
                {busy ? t("common.saving") : t("auth.verify")}
              </Btn>
              <button
                type="button"
                disabled={busy || wait > 0}
                onClick={resend}
                className="mt-3 w-full py-2 text-sm text-white/40 hover:text-white/70 disabled:opacity-40"
              >
                {wait > 0 ? t("auth.resendIn", { n: wait }) : t("auth.resend")}
              </button>
              {mode === "register" && (
                <button type="button" onClick={() => setStep("telegram")} className="w-full py-1 text-sm text-white/30 hover:text-white/60">
                  {t("auth.back")}
                </button>
              )}
            </motion.form>
          )}
        </AnimatePresence>

        <div className="mt-6 border-t border-white/8 pt-4 text-center text-sm text-white/45">
          {mode === "register" ? (
            <button type="button" className="text-gold-200 hover:underline" onClick={() => switchMode("login")}>
              {t("auth.orLogin")}
            </button>
          ) : (
            <button type="button" className="text-gold-200 hover:underline" onClick={() => switchMode("register")}>
              {t("auth.orRegister")}
            </button>
          )}
        </div>
      </motion.div>
    </div>
  );
}

function Guide({ n, icon: Icon, text }: { n: number; icon: typeof Shield; text: string }) {
  return (
    <li className="flex items-start gap-3">
      <span className="mt-0.5 flex h-7 w-7 shrink-0 items-center justify-center rounded-full bg-gold-400/15 text-xs font-bold text-gold-200">
        {n}
      </span>
      <span className="flex-1 pt-0.5 text-white/70">{text}</span>
      <Icon className="mt-0.5 h-4 w-4 shrink-0 text-gold-200/50" />
    </li>
  );
}

function StatusPill({ status, t }: { status: string; t: (k: string) => string }) {
  const map: Record<string, { cls: string; key: string }> = {
    pending_start: { cls: "bg-gold-400/15 text-gold-200", key: "auth.waitingStart" },
    waiting_contact: { cls: "bg-sky-400/15 text-sky-200", key: "auth.waitingContact" },
    otp_sent: { cls: "bg-emerald-400/15 text-emerald-200", key: "auth.otpSent" },
    phone_mismatch: { cls: "bg-rose-400/15 text-rose-200", key: "auth.phoneMismatch" },
  };
  const it = map[status] || map.pending_start;
  return (
    <div className={`inline-flex items-center gap-2 rounded-full px-3 py-1 text-xs font-bold ${it.cls}`}>
      {status === "otp_sent" ? <Check className="h-3.5 w-3.5" /> : <span className="h-1.5 w-1.5 animate-pulse rounded-full bg-current" />}
      {t(it.key)}
    </div>
  );
}

function OtpInput({ value, onChange }: { value: string; onChange: (v: string) => void }) {
  const refs = useRef<Array<HTMLInputElement | null>>([]);
  const digits = value.replace(/\D/g, "").slice(0, 6).split("");
  while (digits.length < 6) digits.push("");

  function write(raw: string, start: number) {
    const next = value.replace(/\D/g, "").split("");
    const incoming = raw.replace(/\D/g, "");
    if (!incoming) {
      next[start] = "";
      onChange(next.join("").slice(0, 6));
      return;
    }
    incoming.split("").forEach((ch, i) => {
      if (start + i < 6) next[start + i] = ch;
    });
    onChange(next.join("").slice(0, 6));
    refs.current[Math.min(5, start + incoming.length)]?.focus();
  }

  return (
    <div className="flex justify-between gap-2" dir="ltr">
      {digits.map((d, i) => (
        <input
          key={i}
          ref={(el) => {
            refs.current[i] = el;
          }}
          value={d}
          inputMode="numeric"
          maxLength={6}
          className="h-14 w-11 rounded-2xl border border-white/14 bg-[#121722] text-center text-xl font-bold text-white outline-none focus:border-gold-400/60 focus:ring-2 focus:ring-gold-400/20 md:w-12"
          onChange={(e) => write(e.target.value, i)}
          onKeyDown={(e) => {
            if (e.key === "Backspace" && !digits[i] && i > 0) {
              refs.current[i - 1]?.focus();
              write("", i - 1);
            }
          }}
          onPaste={(e) => {
            e.preventDefault();
            write(e.clipboardData.getData("text"), 0);
          }}
        />
      ))}
    </div>
  );
}

function formatPhone(raw: string) {
  const map: Record<string, string> = {
    "۰": "0",
    "۱": "1",
    "۲": "2",
    "۳": "3",
    "۴": "4",
    "۵": "5",
    "۶": "6",
    "۷": "7",
    "۸": "8",
    "۹": "9",
  };
  let s = "";
  for (const ch of raw) s += map[ch] || ch;
  s = s.replace(/\D/g, "");
  if (s.startsWith("98")) s = s.slice(2);
  if (s.startsWith("0")) s = s.slice(1);
  return s.slice(0, 10);
}

export function AuthSplash() {
  const { t } = useI18n();
  return (
    <div className="flex min-h-dvh flex-col items-center justify-center bg-mesh bg-ink-950">
      <div className="flex h-16 w-16 items-center justify-center rounded-3xl bg-gradient-to-br from-gold-200 to-gold-700 shadow-glow">
        <Wallet className="h-7 w-7 text-ink-950" />
      </div>
      <div className="mt-5 text-lg font-extrabold gold-text">{t("app.name")}</div>
      <div className="mt-2 text-sm text-white/40">{t("auth.checking")}</div>
    </div>
  );
}
