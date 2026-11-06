import { describe, expect, it } from "vitest";
import dayGolden from "./golden/day.json";
import { isBinary, localDate, paceTone, subtitle, tapValue, withValue } from "./today";
import type { Day, DayHabit } from "./types";

const day = dayGolden as unknown as Day;
const rows = day.groups[0]!.habits;
const find = (name: string) => rows.find((r) => r.habit.name === name)!;
// The done habit is listed last in its group.
const vitamin = find("Vitamine D");
const stretch = find("Étirements");

function item(h: Partial<DayHabit["habit"]>, value: number): DayHabit {
  return { ...vitamin, habit: { ...vitamin.habit, ...h }, value };
}

describe("tapValue", () => {
  it("toggles a yes/no habit", () => {
    const yesNo = { target: 1, unit: "" };
    expect(isBinary(yesNo)).toBe(true);
    expect(tapValue(item(yesNo, 0))).toBe(1);
    expect(tapValue(item(yesNo, 1))).toBe(0);
  });

  it("adds step to a quantity, past the target", () => {
    expect(tapValue(item({ target: 2, unit: "pilules", step: 1 }, 0))).toBe(1);
    expect(tapValue(item({ target: 2, unit: "pilules", step: 1 }, 2))).toBe(3);
    expect(tapValue(item({ target: 20, unit: "pages", step: 5 }, 10))).toBe(15);
  });

  it("a target above 1 without unit is a quantity", () => {
    expect(isBinary({ target: 3, unit: "" })).toBe(false);
    expect(isBinary({ target: 1, unit: "min" })).toBe(false);
  });
});

describe("withValue", () => {
  it("patches only the tapped row", () => {
    const next = withValue(day, stretch.habit.id, 1);
    const patched = next.groups[0]!.habits;
    expect(patched.find((r) => r.habit.id === stretch.habit.id)).toMatchObject({
      value: 1,
      complete: true,
      progress: 1,
    });
    expect(patched.find((r) => r.habit.id === vitamin.habit.id)).toEqual(vitamin);
    expect(day.groups[0]!.habits.find((r) => r.habit.id === stretch.habit.id)!.value).toBe(0);
  });
});

describe("withValue on a period habit", () => {
  it("moves the period total by the change of the day value", () => {
    const sport: DayHabit = {
      ...item({ target: 3, unit: "sorties", period: "week" }, 1),
      total: 3,
      complete: true,
      progress: 1,
    };
    const base: Day = { ...day, groups: [{ moment: "day", habits: [sport] }] };
    const next = withValue(base, sport.habit.id, 2).groups[0]!.habits[0]!;
    expect(next).toMatchObject({ value: 2, total: 4, complete: true, progress: 1 });
    const back = withValue(base, sport.habit.id, 0).groups[0]!.habits[0]!;
    expect(back).toMatchObject({ value: 0, total: 2, complete: false });
  });
});

describe("display", () => {
  it("subtitle", () => {
    expect(subtitle({ ...item({ target: 2, unit: "pilules" }, 1), total: 1 })).toBe(
      "1 / 2 pilules",
    );
    expect(subtitle({ ...item({ target: 1, unit: "" }, 0), total: 0 })).toBe("0 / 1");
  });

  it("subtitle names the period", () => {
    const sport = { ...item({ target: 150, unit: "séances", period: "year" }, 1), total: 132 };
    expect(subtitle(sport)).toBe("132 / 150 séances cette année");
    expect(subtitle({ ...sport, habit: { ...sport.habit, period: "week" } })).toContain(
      "cette semaine",
    );
  });

  it("paceTone", () => {
    const pace = { expected: 10, label: "x" };
    const row = (diff: number, complete = false) => ({
      ...item({ period: "year" }, 0),
      complete,
      pace: { ...pace, diff },
    });
    expect(paceTone(row(2))).toBe("ahead");
    expect(paceTone(row(-2))).toBe("behind");
    expect(paceTone(row(0))).toBe("on");
    expect(paceTone(row(-2, true))).toBe("done");
    expect(paceTone({ ...row(0), pace: null })).toBeNull();
  });

  it("localDate pads", () => {
    expect(localDate(new Date(2026, 0, 5))).toBe("2026-01-05");
  });
});
