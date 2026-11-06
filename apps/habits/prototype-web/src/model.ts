// PROTOTYPE — throwaway. In-memory model mirroring the v2 decisions
// (period + target + optional allowed days). No persistence on purpose.
import { useState } from "react";

export type Period = "day" | "week" | "month" | "year";
export type Moment = "morning" | "day" | "evening";

export interface Habit {
  id: string;
  name: string;
  emoji: string;
  color: string;
  moment: Moment;
  period: Period;
  target: number;
  unit?: string;
  step?: number; // tap increment (open question: +1 is awkward for minutes)
  days?: number[]; // allowed weekdays, 0 = Monday
}

export type Logs = Record<string, Record<string, number>>;

export const TODAY = "2026-11-06";
export const EDIT_WINDOW = 7;

export const PALETTE = {
  amber: "#F59E0B",
  violet: "#8B5CF6",
  blue: "#3B82F6",
  red: "#F43F5E",
  green: "#10B981",
  teal: "#0EA5A4",
  slate: "#64748B",
  pink: "#EC4899",
};

export const MOMENTS: Moment[] = ["morning", "day", "evening"];
export const MOMENT_LABEL: Record<Moment, string> = {
  morning: "Matin",
  day: "Journée",
  evening: "Soir",
};
export const MOMENT_ICON: Record<Moment, string> = {
  morning: "🌅",
  day: "☀️",
  evening: "🌙",
};

export const HABITS: Habit[] = [
  { id: "vitd", name: "Vitamine D", emoji: "💊", color: PALETTE.amber, moment: "morning", period: "day", target: 2, unit: "pilules" },
  { id: "medit", name: "Méditation", emoji: "🧘", color: PALETTE.violet, moment: "morning", period: "day", target: 1 },
  { id: "eau", name: "Boire de l'eau", emoji: "💧", color: PALETTE.blue, moment: "day", period: "day", target: 8, unit: "verres" },
  { id: "sport", name: "Séance de sport", emoji: "🏋️", color: PALETTE.red, moment: "day", period: "year", target: 150, unit: "séances" },
  { id: "course", name: "Course à pied", emoji: "🏃", color: PALETTE.green, moment: "day", period: "week", target: 3, unit: "sorties" },
  { id: "lecture", name: "Lecture", emoji: "📚", color: PALETTE.teal, moment: "evening", period: "day", target: 20, unit: "min", step: 5 },
  { id: "ecran", name: "Pas d'écran après 22 h", emoji: "📵", color: PALETTE.slate, moment: "evening", period: "day", target: 1, days: [0, 1, 2, 3, 4] },
  { id: "famille", name: "Appeler la famille", emoji: "☎️", color: PALETTE.pink, moment: "evening", period: "month", target: 4, unit: "appels" },
];

// ---- dates (civil days as YYYY-MM-DD strings) ----
const toD = (s: string) => new Date(s + "T00:00:00Z");
const toS = (d: Date) => d.toISOString().slice(0, 10);
export const addDays = (s: string, n: number) => {
  const d = toD(s);
  d.setUTCDate(d.getUTCDate() + n);
  return toS(d);
};
export const weekday = (s: string) => (toD(s).getUTCDay() + 6) % 7;
export const daysBetween = (a: string, b: string) =>
  Math.round((toD(b).getTime() - toD(a).getTime()) / 864e5);

export function periodRange(p: Period, s: string): [string, string] {
  const y = Number(s.slice(0, 4));
  const m = Number(s.slice(5, 7));
  switch (p) {
    case "day":
      return [s, s];
    case "week": {
      const start = addDays(s, -weekday(s));
      return [start, addDays(start, 6)];
    }
    case "month":
      return [s.slice(0, 7) + "-01", toS(new Date(Date.UTC(y, m, 0)))];
    case "year":
      return [`${y}-01-01`, `${y}-12-31`];
  }
}

const fmt = (o: Intl.DateTimeFormatOptions) =>
  new Intl.DateTimeFormat("fr-FR", { ...o, timeZone: "UTC" });
export const fmtLong = (s: string) => fmt({ weekday: "long", day: "numeric", month: "long" }).format(toD(s));
export const fmtWeekday = (s: string) => fmt({ weekday: "long" }).format(toD(s));
export const fmtDayMonth = (s: string) => fmt({ day: "numeric", month: "long" }).format(toD(s));
export const fmtLetter = (s: string) => fmt({ weekday: "narrow" }).format(toD(s));
export const dayNum = (s: string) => Number(s.slice(8));
export const weekDates = (s: string) => {
  const start = addDays(s, -weekday(s));
  return Array.from({ length: 7 }, (_, i) => addDays(start, i));
};
export function relLabel(s: string) {
  const n = daysBetween(s, TODAY);
  if (n === 0) return "Aujourd'hui";
  if (n === 1) return "Hier";
  return `Il y a ${n} jours`;
}
export const editable = (s: string) => s <= TODAY && daysBetween(s, TODAY) < EDIT_WINDOW;

// ---- rules ----
export const applies = (h: Habit, s: string) => !h.days || h.days.includes(weekday(s));
export const isBinary = (h: Habit) => h.target === 1 && !h.unit;
export const value = (logs: Logs, s: string, id: string) => logs[s]?.[id] ?? 0;

export function periodTotal(logs: Logs, h: Habit, s: string) {
  const [start] = periodRange(h.period, s);
  let total = 0;
  for (let d = start; d <= s; d = addDays(d, 1)) total += value(logs, d, h.id);
  return total;
}

export function progress(logs: Logs, h: Habit, s: string) {
  return Math.min(1, periodTotal(logs, h, s) / h.target);
}
export const isComplete = (logs: Logs, h: Habit, s: string) => progress(logs, h, s) >= 1;

const PERIOD_SUFFIX: Record<Period, string> = {
  day: "",
  week: "cette semaine",
  month: "ce mois-ci",
  year: "cette année",
};

export interface Line {
  today: number;
  total: number;
  progress: number;
  complete: boolean;
  count: string; // "1 / 2 pilules"
  periodText: string; // "cette année"
  pace: { expected: number; diff: number; text: string } | null;
}

export function line(logs: Logs, h: Habit, s: string): Line {
  const total = periodTotal(logs, h, s);
  const p = Math.min(1, total / h.target);
  let pace: Line["pace"] = null;
  if (h.period !== "day") {
    const [start, end] = periodRange(h.period, s);
    const expected = (h.target * (daysBetween(start, s) + 1)) / (daysBetween(start, end) + 1);
    const diff = total - expected;
    const r = Math.round(Math.abs(diff));
    const text =
      total >= h.target ? "objectif atteint" : r === 0 ? "dans le rythme" : diff > 0 ? `${r} d'avance` : `${r} de retard`;
    pace = { expected, diff, text };
  }
  return {
    today: value(logs, s, h.id),
    total,
    progress: p,
    complete: p >= 1,
    count: isBinary(h) ? (p >= 1 ? "Fait" : "À faire") : `${total} / ${h.target}${h.unit ? " " + h.unit : ""}`,
    periodText: PERIOD_SUFFIX[h.period],
    pace,
  };
}

export const visible = (s: string) => HABITS.filter((h) => applies(h, s));
export const dailyHabits = (s: string) => visible(s).filter((h) => h.period === "day");

export function grouped(logs: Logs, s: string, m: Moment) {
  const hs = visible(s).filter((h) => h.moment === m);
  return [...hs.filter((h) => !isComplete(logs, h, s)), ...hs.filter((h) => isComplete(logs, h, s))];
}

export function dayRatio(logs: Logs, s: string) {
  const hs = dailyHabits(s);
  if (!hs.length) return 0;
  return hs.reduce((a, h) => a + progress(logs, h, s), 0) / hs.length;
}
export const dayComplete = (logs: Logs, s: string) => dailyHabits(s).every((h) => isComplete(logs, h, s));

export function streak(logs: Logs) {
  let d = dayComplete(logs, TODAY) ? TODAY : addDays(TODAY, -1);
  let n = 0;
  while (n < 500 && dayComplete(logs, d)) {
    n++;
    d = addDays(d, -1);
  }
  return n;
}

// ---- fake history ----
function rng(seed: number) {
  return () => {
    seed |= 0;
    seed = (seed + 0x6d2b79f5) | 0;
    let t = Math.imul(seed ^ (seed >>> 15), 1 | seed);
    t = (t + Math.imul(t ^ (t >>> 7), 61 | t)) ^ t;
    return ((t ^ (t >>> 14)) >>> 0) / 4294967296;
  };
}

function seedLogs(): Logs {
  const r = rng(42);
  const logs: Logs = {};
  const odds: Record<string, number> = { sport: 0.43, course: 0.42, famille: 0.14 };
  for (let i = 400; i >= 1; i--) {
    const d = addDays(TODAY, -i);
    logs[d] = {};
    for (const h of HABITS) {
      if (!applies(h, d)) continue;
      const x = r();
      if (h.period === "day") {
        const full = i <= 12 || x < 0.8;
        logs[d][h.id] = full ? h.target : x < 0.92 ? Math.floor(h.target * r()) : 0;
      } else if (x < odds[h.id]) {
        logs[d][h.id] = 1;
      }
    }
  }
  logs[TODAY] = { vitd: 1, medit: 1, eau: 3 };
  return logs;
}

export function useStore() {
  const [logs, setLogs] = useState<Logs>(seedLogs);
  const [notes, setNotes] = useState<Record<string, string>>({
    [addDays(TODAY, -1)]: "Genou un peu douloureux, séance écourtée.",
  });
  return {
    logs,
    notes,
    setValue: (s: string, id: string, v: number) =>
      setLogs((l) => ({ ...l, [s]: { ...l[s], [id]: Math.max(0, v) } })),
    setNote: (s: string, t: string) => setNotes((n) => ({ ...n, [s]: t })),
  };
}
export type Store = ReturnType<typeof useStore>;
