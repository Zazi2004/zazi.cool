// Pure helpers of the Today screen. The server owns every business rule (NF5): this only
// decides what one tap sends and patches a row optimistically until the server answers.
import type { Day, DayHabit } from "./types";

/** Yes/no habit: a target of 1 with no unit. */
export function isBinary(h: { target: number; unit: string }): boolean {
  return h.target === 1 && h.unit === "";
}

/** Value to send after a tap: yes/no toggles, a quantity adds `step` (target may be exceeded). */
export function tapValue(item: DayHabit): number {
  const h = item.habit;
  return isBinary(h) ? (item.value ? 0 : 1) : item.value + (h.step || 1);
}

/**
 * One row with the day's value changed: the period total moves by the same amount. The pace
 * keeps the server's figures until its answer replaces it.
 */
function patched(it: DayHabit, value: number): DayHabit {
  const total = Math.max(0, it.total + value - it.value);
  return {
    ...it,
    value,
    total,
    progress: Math.min(1, total / it.habit.target),
    complete: total >= it.habit.target,
  };
}

/** Shows `value` on one row right away; the server answer replaces the whole day. */
export function withValue(day: Day, habitId: number, value: number): Day {
  return {
    ...day,
    groups: day.groups.map((g) => ({
      ...g,
      habits: g.habits.map((it) => (it.habit.id === habitId ? patched(it, value) : it)),
    })),
  };
}

/** Local civil date as YYYY-MM-DD. */
export function localDate(now: Date): string {
  const pad = (n: number) => String(n).padStart(2, "0");
  return `${now.getFullYear()}-${pad(now.getMonth() + 1)}-${pad(now.getDate())}`;
}

const periodSuffix = {
  day: "",
  week: " cette semaine",
  month: " ce mois-ci",
  year: " cette année",
};

/** « 1 / 2 pilules », « 132 / 150 séances cette année ». */
export function subtitle(item: DayHabit): string {
  const unit = item.habit.unit ? ` ${item.habit.unit}` : "";
  return `${item.total} / ${item.habit.target}${unit}${periodSuffix[item.habit.period]}`;
}

/** The pace label of a period habit and its tone; the label is computed by the server. */
export function paceTone(item: DayHabit): "done" | "ahead" | "behind" | "on" | null {
  const pace = item.pace;
  if (!pace) return null;
  if (item.complete) return "done";
  return pace.diff > 0 ? "ahead" : pace.diff < 0 ? "behind" : "on";
}
