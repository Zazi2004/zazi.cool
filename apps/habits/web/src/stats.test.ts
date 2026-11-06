import { describe, expect, it } from "vitest";
import {
  cellDate,
  firstRow,
  formatRate,
  formatStreak,
  hasData,
  rangeLabel,
  totalLine,
} from "./stats";
import type { Habit, HabitSummary } from "./types";

const course: Habit = {
  id: 2,
  name: "Course",
  emoji: "🏃",
  color: "green",
  moment: "evening",
  position: 0,
  unit: "sorties",
  step: 1,
  period: "week",
  target: 3,
  weekdays: 127,
  archivedOn: null,
};
const summary = (over: Partial<HabitSummary> = {}): HabitSummary => ({
  habit: course,
  total: 1,
  progress: 1 / 3,
  complete: false,
  pace: null,
  streak: 2,
  ...over,
});

describe("stats formatting", () => {
  it("shows a dash when no period counts", () => {
    expect(formatRate(null)).toBe("—");
    expect(formatRate(0.6667)).toBe("67 %");
    expect(formatRate(0)).toBe("0 %");
  });

  it("pluralises the streak by period", () => {
    expect(formatStreak("week", 1)).toBe("1 semaine");
    expect(formatStreak("week", 3)).toBe("3 semaines");
    expect(formatStreak("month", 2)).toBe("2 mois");
    expect(formatStreak("day", 0)).toBe("0 jour");
    expect(formatStreak("year", null)).toBe("—");
  });

  it("writes the total of the period", () => {
    expect(totalLine(summary())).toBe("1 / 3 sorties cette semaine");
    const sport = { ...course, period: "year" as const, unit: "", target: 150 };
    expect(totalLine(summary({ habit: sport, total: 128 }))).toBe("128 / 150 cette année");
  });

  it("labels dates", () => {
    expect(rangeLabel("2026-10-26", "2026-10-31")).toMatch(/^26 oct\.? – 31 oct\.?$/);
    expect(rangeLabel("2026-11-02", "2026-11-02")).toMatch(/^2 nov\.?$/);
    expect(cellDate("2026-11-06")).toBe("6 novembre");
  });
});

describe("heatmap layout", () => {
  it("starts on the weekday of 1 January (2026: Thursday)", () => {
    expect(firstRow([{ date: "2026-01-01", value: 0, level: 0 }])).toBe(4);
    expect(firstRow([{ date: "2024-01-01", value: 0, level: 0 }])).toBe(1);
    expect(firstRow([])).toBe(1);
  });

  it("detects a year without any day of the habit", () => {
    expect(hasData([{ date: "2025-01-01", value: 0, level: null }])).toBe(false);
    expect(hasData([{ date: "2025-01-01", value: 0, level: 0 }])).toBe(true);
  });
});
