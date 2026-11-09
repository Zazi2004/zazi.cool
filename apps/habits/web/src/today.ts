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

/** « Fait », « À faire », « 1 / 2 pilules », « 132 / 150 séances cette année ». */
export function subtitle(item: DayHabit): string {
  const h = item.habit;
  const suffix = periodSuffix[h.period];
  if (isBinary(h)) return `${item.complete ? "Fait" : "À faire"}${suffix}`;
  const unit = h.unit ? ` ${h.unit}` : "";
  return `${item.total} / ${h.target}${unit}${suffix}`;
}

/** The pace label of a period habit and its tone; the label is computed by the server. */
export function paceTone(item: DayHabit): "done" | "ahead" | "behind" | "on" | null {
  const pace = item.pace;
  if (!pace) return null;
  if (item.complete) return "done";
  return pace.diff > 0 ? "ahead" : pace.diff < 0 ? "behind" : "on";
}

const dayMs = 24 * 60 * 60 * 1000;

function parseDate(date: string): Date {
  const [y = 0, m = 1, d = 1] = date.split("-").map(Number);
  return new Date(y, m - 1, d);
}

/** The civil date `n` days after `date` (before when negative), both as YYYY-MM-DD. */
export function addDays(date: string, n: number): string {
  const d = parseDate(date);
  d.setDate(d.getDate() + n);
  return localDate(d);
}

/** « Aujourd'hui », « Hier », « Il y a N jours ». */
export function relativeLabel(date: string, today: string): string {
  const days = Math.round((parseDate(today).getTime() - parseDate(date).getTime()) / dayMs);
  if (days <= 0) return "Aujourd'hui";
  if (days === 1) return "Hier";
  return `Il y a ${days} jours`;
}

const longFormat = new Intl.DateTimeFormat("fr-FR", {
  weekday: "long",
  day: "numeric",
  month: "long",
});

/** « vendredi 6 novembre ». */
export function longDate(date: string): string {
  return longFormat.format(parseDate(date));
}

/** Day of the month of a YYYY-MM-DD date. */
export function dayOfMonth(date: string): number {
  return Number(date.slice(8));
}

/** One-letter weekday of the week strip: L M M J V S D. */
export function weekdayInitial(date: string): string {
  return "LMMJVSD"[(parseDate(date).getDay() + 6) % 7] ?? "";
}

/** Horizontal travel (px) that turns a drag into a day change. */
export const swipeDistance = 60;
/** A press that moves farther than this (px) is a drag, not a long press. */
export const holdTolerance = 10;
/** How long (ms) a press must last to open the exact-value sheet. */
export const holdDelay = 450;

/** -1 for the previous day, 1 for the next one, 0 when the drag is short or mostly vertical. */
export function swipeStep(dx: number, dy: number): -1 | 0 | 1 {
  if (Math.abs(dx) < swipeDistance || Math.abs(dx) < 1.5 * Math.abs(dy)) return 0;
  return dx < 0 ? 1 : -1;
}

/** Whether a pointer that moved by (dx, dy) cancels a pending long press. */
export function cancelsHold(dx: number, dy: number): boolean {
  return Math.hypot(dx, dy) > holdTolerance;
}

/** The day a swipe leads to, or null: never past today. */
export function swipeTarget(date: string, today: string, dx: number, dy: number): string | null {
  const step = swipeStep(dx, dy);
  if (step === 0) return null;
  const target = addDays(date, step);
  return target > today ? null : target;
}

export const maxValue = 100000;

/** A whole number from the exact-value field, or null when it is not valid. */
export function parseValue(text: string): number | null {
  if (!/^\d+$/.test(text.trim())) return null;
  const n = Number(text);
  return n <= maxValue ? n : null;
}

/** Number of characters as the server counts them (code points, not UTF-16 units). */
export function noteLength(text: string): number {
  return Array.from(text).length;
}

export const noteLimit = 1000;

/** Confetti only when an action turned an incomplete day into a complete one. */
export function shouldCelebrate(wasComplete: boolean, day: Day): boolean {
  return !wasComplete && day.complete;
}
