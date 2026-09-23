import type { Locale } from "./i18n";

export type YMD = { y: number; m: number; d: number };
export type CalendarKind = "jalali" | "gregory";

export function calendarKind(locale: Locale): CalendarKind {
  return locale === "fa" ? "jalali" : "gregory";
}

export function weekStart(locale: Locale): number {
  if (locale === "fa") return 6;
  if (locale === "de") return 1;
  return 0;
}

const JALALI_MONTHS = [
  "فروردین",
  "اردیبهشت",
  "خرداد",
  "تیر",
  "مرداد",
  "شهریور",
  "مهر",
  "آبان",
  "آذر",
  "دی",
  "بهمن",
  "اسفند",
] as const;

const GREG_MONTHS: Record<"en" | "de", readonly string[]> = {
  en: ["January", "February", "March", "April", "May", "June", "July", "August", "September", "October", "November", "December"],
  de: ["Januar", "Februar", "März", "April", "Mai", "Juni", "Juli", "August", "September", "Oktober", "November", "Dezember"],
};

const WEEKDAYS: Record<Locale, readonly string[]> = {
  fa: ["ش", "ی", "د", "س", "چ", "پ", "ج"],
  en: ["Su", "Mo", "Tu", "We", "Th", "Fr", "Sa"],
  de: ["Mo", "Di", "Mi", "Do", "Fr", "Sa", "So"],
};

export function monthName(month: number, locale: Locale): string {
  if (locale === "fa") return JALALI_MONTHS[month - 1] || "";
  return GREG_MONTHS[locale][month - 1] || "";
}

export function weekdayLabels(locale: Locale): readonly string[] {
  return WEEKDAYS[locale];
}

export function parseISODate(iso: string): YMD | null {
  const m = /^(\d{4})-(\d{2})-(\d{2})/.exec((iso || "").trim());
  if (!m) return null;
  const y = Number(m[1]);
  const mo = Number(m[2]);
  const d = Number(m[3]);
  if (!y || mo < 1 || mo > 12 || d < 1 || d > 31) return null;
  return { y, m: mo, d };
}

export function toISODate(y: number, m: number, d: number): string {
  return `${y}-${pad(m)}-${pad(d)}`;
}

export function isoToLocalDate(iso: string): Date | null {
  const hasTime = /T\d{2}:/.test(iso);
  if (!hasTime) {
    const p = parseISODate(iso);
    if (p) return new Date(p.y, p.m - 1, p.d);
  }
  const d = new Date(iso);
  return Number.isNaN(d.getTime()) ? null : d;
}

function pad(n: number) {
  return String(n).padStart(2, "0");
}

function div(a: number, b: number) {
  return ~~(a / b);
}

export function toJalali(gy: number, gm: number, gd: number): YMD {
  const gdm = [0, 31, 59, 90, 120, 151, 181, 212, 243, 273, 304, 334];
  const gy2 = gm > 2 ? gy + 1 : gy;
  let days =
    355666 +
    365 * gy +
    div(gy2 + 3, 4) -
    div(gy2 + 99, 100) +
    div(gy2 + 399, 400) +
    gd +
    gdm[gm - 1];
  let jy = -1595 + 33 * div(days, 12053);
  days %= 12053;
  jy += 4 * div(days, 1461);
  days %= 1461;
  if (days > 365) {
    jy += div(days - 1, 365);
    days = (days - 1) % 365;
  }
  let jm: number;
  let jd: number;
  if (days < 186) {
    jm = 1 + div(days, 31);
    jd = 1 + (days % 31);
  } else {
    jm = 7 + div(days - 186, 30);
    jd = 1 + ((days - 186) % 30);
  }
  return { y: jy, m: jm, d: jd };
}

export function toGregorian(jy: number, jm: number, jd: number): YMD {
  jy += 1595;
  let days = -355668 + 365 * jy + div(jy, 33) * 8 + div((jy % 33) + 3, 4) + jd + (jm < 7 ? (jm - 1) * 31 : (jm - 7) * 30 + 186);
  let gy = 400 * div(days, 146097);
  days %= 146097;
  if (days > 36524) {
    gy += 100 * div(--days, 36524);
    days %= 36524;
    if (days >= 365) days += 1;
  }
  gy += 4 * div(days, 1461);
  days %= 1461;
  if (days > 365) {
    gy += div(days - 1, 365);
    days = (days - 1) % 365;
  }
  const gd = days + 1;
  const leap = (gy % 4 === 0 && gy % 100 !== 0) || gy % 400 === 0;
  const sal = [0, 31, leap ? 29 : 28, 31, 30, 31, 30, 31, 31, 30, 31, 30, 31];
  let gm = 0;
  let rem = gd;
  for (; gm < 13 && rem > sal[gm]; gm += 1) rem -= sal[gm];
  return { y: gy, m: gm, d: rem };
}

const JALALI_LEAP_REMAINDERS = [1, 5, 9, 13, 17, 22, 26, 30];

export function isLeapJalali(jy: number): boolean {
  return JALALI_LEAP_REMAINDERS.includes(((jy % 33) + 33) % 33);
}

export function daysInMonth(y: number, m: number, kind: CalendarKind): number {
  if (kind === "jalali") {
    if (m <= 6) return 31;
    if (m <= 11) return 30;
    return isLeapJalali(y) ? 30 : 29;
  }
  return new Date(y, m, 0).getDate();
}

export function gregorianToParts(iso: string, kind: CalendarKind): YMD | null {
  const g = parseISODate(iso);
  if (!g) return null;
  return kind === "jalali" ? toJalali(g.y, g.m, g.d) : g;
}

export function partsToISO(parts: YMD, kind: CalendarKind): string {
  const g = kind === "jalali" ? toGregorian(parts.y, parts.m, parts.d) : parts;
  return toISODate(g.y, g.m, g.d);
}

export function todayParts(kind: CalendarKind): YMD {
  const n = new Date();
  const g = { y: n.getFullYear(), m: n.getMonth() + 1, d: n.getDate() };
  return kind === "jalali" ? toJalali(g.y, g.m, g.d) : g;
}

export function weekdayIndex(parts: YMD, kind: CalendarKind): number {
  const g = kind === "jalali" ? toGregorian(parts.y, parts.m, parts.d) : parts;
  return new Date(g.y, g.m - 1, g.d).getDay();
}

export function toFaDigits(n: number | string): string {
  return String(n).replace(/\d/g, (ch) => "۰۱۲۳۴۵۶۷۸۹"[Number(ch)]);
}

export function formatCalNum(n: number, locale: Locale): string {
  return locale === "fa" ? toFaDigits(n) : String(n);
}

export function formatDisplayDate(iso: string, locale: Locale): string {
  const g = parseISODate(iso);
  if (!g) return "";
  if (locale === "fa") {
    const j = toJalali(g.y, g.m, g.d);
    return `${toFaDigits(j.d)} ${JALALI_MONTHS[j.m - 1]} ${toFaDigits(j.y)}`;
  }
  const tag = locale === "de" ? "de-DE" : "en-US";
  return new Date(g.y, g.m - 1, g.d).toLocaleDateString(tag, {
    year: "numeric",
    month: "short",
    day: "numeric",
  });
}

export function formatDisplayMonth(isoOrYm: string, locale: Locale): string {
  const raw = isoOrYm.length === 7 ? `${isoOrYm}-01` : isoOrYm;
  const g = parseISODate(raw);
  if (!g) return isoOrYm;
  if (locale === "fa") {
    const j = toJalali(g.y, g.m, g.d);
    return `${JALALI_MONTHS[j.m - 1]} ${toFaDigits(j.y)}`;
  }
  const tag = locale === "de" ? "de-DE" : "en-US";
  return new Date(g.y, g.m - 1, 1).toLocaleDateString(tag, { month: "long", year: "numeric" });
}

export function sameDay(a: YMD, b: YMD) {
  return a.y === b.y && a.m === b.m && a.d === b.d;
}

export function clampDay(y: number, m: number, d: number, kind: CalendarKind): YMD {
  return { y, m, d: Math.min(d, daysInMonth(y, m, kind)) };
}

export function shiftMonth(y: number, m: number, delta: number, kind: CalendarKind): YMD {
  const idx = y * 12 + (m - 1) + delta;
  const ny = Math.floor(idx / 12);
  const nm = (idx % 12) + 1;
  return { y: ny, m: nm, d: 1 };
}

export function yearRange(kind: CalendarKind, around: number): number[] {
  const min = kind === "jalali" ? 1350 : 1970;
  const max = kind === "jalali" ? 1450 : 2070;
  const start = Math.max(min, around - 40);
  const end = Math.min(max, around + 20);
  const out: number[] = [];
  for (let y = start; y <= end; y += 1) out.push(y);
  return out;
}
