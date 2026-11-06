import type { HabitSummary } from "../types";

/** CSS classes of a pace label: green when ahead or done, red when behind. */
export function paceClass(s: Pick<HabitSummary, "complete" | "pace">): string {
  const pace = s.pace;
  if (!pace) return "";
  const tone = s.complete ? "done" : pace.diff > 0 ? "ahead" : pace.diff < 0 ? "behind" : "on";
  return `pace pace-${tone}`;
}
