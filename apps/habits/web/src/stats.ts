// Display helpers of the Stats screens. Every figure comes from the server (NF5); this
// only formats it and lays the heatmap out.
import type { HabitStats, HabitSummary, Period } from "./types";

/** « 67 % », or « — » when no period counts (the server sends null). */
export function formatRate(rate: number | null): string {
  return rate === null ? "—" : `${Math.round(rate * 100)} %`;
}

const streakUnits: Record<Period, [string, string]> = {
  day: ["jour", "jours"],
  week: ["semaine", "semaines"],
  month: ["mois", "mois"],
  year: ["an", "ans"],
};

/** « 3 semaines »; « — » for a habit that has no streak (yearly). */
export function formatStreak(period: Period, n: number | null): string {
  if (n === null) return "—";
  const [one, many] = streakUnits[period];
  return `${n} ${n > 1 ? many : one}`;
}

const periodSuffix: Record<Period, string> = {
  day: "",
  week: " cette semaine",
  month: " ce mois-ci",
  year: " cette année",
};

/** « 3 / 3 sorties cette semaine ». */
export function totalLine(s: HabitSummary): string {
  const unit = s.habit.unit ? ` ${s.habit.unit}` : "";
  return `${s.total} / ${s.habit.target}${unit}${periodSuffix[s.habit.period]}`;
}

export const statusLabels: Record<HabitStats["periods"][number]["status"], string> = {
  success: "Réussie",
  missed: "Manquée",
  current: "En cours",
  interrupted: "Interrompue",
};

const shortDate = new Intl.DateTimeFormat("fr-FR", { day: "numeric", month: "short" });
const longDate = new Intl.DateTimeFormat("fr-FR", { day: "numeric", month: "long" });

function parse(date: string): Date {
  const [y = 0, m = 1, d = 1] = date.split("-").map(Number);
  return new Date(y, m - 1, d);
}

/** « 26 oct. – 1 nov. » for a period; one date when it lasts a single day. */
export function rangeLabel(start: string, end: string): string {
  return start === end
    ? shortDate.format(parse(start))
    : `${shortDate.format(parse(start))} – ${shortDate.format(parse(end))}`;
}

/** « 6 novembre », for the tooltip of a heatmap cell. */
export function cellDate(date: string): string {
  return longDate.format(parse(date));
}

/** Row (1 = Monday … 7 = Sunday) of the first day of the heatmap, whose weeks are columns. */
export function firstRow(days: HabitStats["days"]): number {
  const first = days[0];
  return first ? ((parse(first.date).getDay() + 6) % 7) + 1 : 1;
}

/** True when the habit applies at least one day of the year. */
export function hasData(days: HabitStats["days"]): boolean {
  return days.some((d) => d.level !== null);
}
