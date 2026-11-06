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

/** Shows `value` on one row right away; the server answer replaces the whole day. */
export function withValue(day: Day, habitId: number, value: number): Day {
  return {
    ...day,
    groups: day.groups.map((g) => ({
      ...g,
      habits: g.habits.map((it) =>
        it.habit.id === habitId
          ? {
              ...it,
              value,
              total: value,
              progress: Math.min(1, value / it.habit.target),
              complete: value >= it.habit.target,
            }
          : it,
      ),
    })),
  };
}

/** Local civil date as YYYY-MM-DD. */
export function localDate(now: Date): string {
  const pad = (n: number) => String(n).padStart(2, "0");
  return `${now.getFullYear()}-${pad(now.getMonth() + 1)}-${pad(now.getDate())}`;
}

/** « 1 / 2 pilules », « 1 / 1 » for a yes/no habit shows nothing more than the state. */
export function subtitle(item: DayHabit): string {
  const unit = item.habit.unit ? ` ${item.habit.unit}` : "";
  return `${item.total} / ${item.habit.target}${unit}`;
}
