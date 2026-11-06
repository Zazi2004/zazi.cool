// Pure helpers of the Habits screen. No business rule lives here (NF5): the server
// validates and computes; this only shapes what the form sends and what the list shows.
import { moments, type Habit, type HabitInput, type Moment } from "./types";

export const allDays = 0b1111111;

/** Day labels, Monday first, matching bit 0 = Monday of the weekdays mask. */
export const weekdayNames = [
  { short: "L", long: "Lundi" },
  { short: "M", long: "Mardi" },
  { short: "M", long: "Mercredi" },
  { short: "J", long: "Jeudi" },
  { short: "V", long: "Vendredi" },
  { short: "S", long: "Samedi" },
  { short: "D", long: "Dimanche" },
] as const;

export function toggleWeekday(mask: number, index: number): number {
  return mask ^ (1 << index);
}

export function hasWeekday(mask: number, index: number): boolean {
  return (mask & (1 << index)) !== 0;
}

export interface HabitForm {
  name: string;
  emoji: string;
  color: HabitInput["color"];
  moment: Moment;
  kind: "binary" | "quantity";
  target: string;
  unit: string;
  step: string;
  period: HabitInput["period"];
  weekdays: number;
}

export const emptyForm: HabitForm = {
  name: "",
  emoji: "✨",
  color: "violet",
  moment: "morning",
  kind: "binary",
  target: "1",
  unit: "",
  step: "1",
  period: "day",
  weekdays: allDays,
};

/** Builds the request body. A yes/no habit is a target of 1 with no unit. */
export function toInput(form: HabitForm): HabitInput {
  const quantity = form.kind === "quantity";
  return {
    name: form.name.trim(),
    emoji: form.emoji.trim(),
    color: form.color,
    moment: form.moment,
    unit: quantity ? form.unit.trim() : "",
    step: quantity ? Number(form.step) : 1,
    period: form.period,
    target: quantity ? Number(form.target) : 1,
    weekdays: form.weekdays,
  };
}

/** Groups habits by moment in display order, hiding empty moments. */
export function groupByMoment(habits: Habit[]): { moment: Moment; habits: Habit[] }[] {
  return moments
    .map((moment) => ({ moment, habits: habits.filter((h) => h.moment === moment) }))
    .filter((group) => group.habits.length > 0);
}

/** Places a new habit last in its moment, like the server does. */
export function withHabit(habits: Habit[], created: Habit): Habit[] {
  return [...habits, created].sort(
    (a, b) => moments.indexOf(a.moment) - moments.indexOf(b.moment) || a.position - b.position,
  );
}

/** One-line description of the rule, e.g. "2 pilules par jour" or "Tous les jours". */
export function describeRule(h: Habit): string {
  const per = { day: "par jour", week: "par semaine", month: "par mois", year: "par an" }[h.period];
  const quantity = h.target !== 1 || h.unit !== "";
  const goal = quantity ? `${h.target}${h.unit ? ` ${h.unit}` : ""} ${per}` : `Oui / non ${per}`;
  if (h.weekdays === allDays) return goal;
  const days = weekdayNames
    .filter((_, i) => hasWeekday(h.weekdays, i))
    .map((d) => d.long.slice(0, 3).toLowerCase());
  return `${goal} · ${days.join(", ")}`;
}
