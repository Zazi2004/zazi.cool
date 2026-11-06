// Types of the JSON API (spec §7). The Go handlers produce this shape; the golden files
// in src/golden/ (written by the Go tests) are checked against these types in
// golden.test.ts, so a field renamed on one side only fails the build.

export type Moment = "morning" | "day" | "evening";
export type Period = "day" | "week" | "month" | "year";
export type Color = "amber" | "violet" | "blue" | "red" | "green" | "teal" | "slate" | "pink";

export interface Habit {
  id: number;
  name: string;
  emoji: string;
  color: Color;
  moment: Moment;
  position: number;
  unit: string;
  step: number;
  period: Period;
  target: number;
  /** 7-bit mask, bit 0 = Monday (the rule in force today). */
  weekdays: number;
  archivedOn: string | null;
}

export type HabitInput = Omit<Habit, "id" | "position" | "archivedOn">;

export interface HabitList {
  active: Habit[];
  archived: Habit[];
}

export const moments: readonly Moment[] = ["morning", "day", "evening"];
export const periods: readonly Period[] = ["day", "week", "month", "year"];
export const colors: readonly Color[] = [
  "amber",
  "violet",
  "blue",
  "red",
  "green",
  "teal",
  "slate",
  "pink",
];

export const momentLabels: Record<Moment, string> = {
  morning: "Matin",
  day: "Journée",
  evening: "Soir",
};

export const periodLabels: Record<Period, string> = {
  day: "Jour",
  week: "Semaine",
  month: "Mois",
  year: "Année",
};

export const colorLabels: Record<Color, string> = {
  amber: "Ambre",
  violet: "Violet",
  blue: "Bleu",
  red: "Rouge",
  green: "Vert",
  teal: "Turquoise",
  slate: "Ardoise",
  pink: "Rose",
};
