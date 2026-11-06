import { describe, expect, it } from "vitest";
import dayGolden from "./golden/day.json";
import {
  addDays,
  cancelsHold,
  isBinary,
  localDate,
  longDate,
  noteLength,
  parseValue,
  relativeLabel,
  shouldCelebrate,
  subtitle,
  swipeStep,
  swipeTarget,
  tapValue,
  weekdayInitial,
  withValue,
} from "./today";
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

describe("display", () => {
  it("subtitle", () => {
    expect(subtitle({ ...item({ target: 2, unit: "pilules" }, 1), total: 1 })).toBe(
      "1 / 2 pilules",
    );
    expect(subtitle({ ...item({ target: 1, unit: "" }, 0), total: 0 })).toBe("0 / 1");
  });

  it("localDate pads", () => {
    expect(localDate(new Date(2026, 0, 5))).toBe("2026-01-05");
  });
});

describe("dates", () => {
  it("addDays crosses months and years", () => {
    expect(addDays("2026-11-06", -6)).toBe("2026-10-31");
    expect(addDays("2026-12-31", 1)).toBe("2027-01-01");
    expect(addDays("2026-03-29", 1)).toBe("2026-03-30"); // daylight saving change
  });

  it("relativeLabel", () => {
    expect(relativeLabel("2026-11-06", "2026-11-06")).toBe("Aujourd'hui");
    expect(relativeLabel("2026-11-05", "2026-11-06")).toBe("Hier");
    expect(relativeLabel("2026-11-01", "2026-11-06")).toBe("Il y a 5 jours");
  });

  it("longDate and weekdayInitial", () => {
    expect(longDate("2026-11-06")).toBe("vendredi 6 novembre");
    expect(["2026-11-02", "2026-11-06", "2026-11-08"].map(weekdayInitial)).toEqual(["L", "V", "D"]);
  });
});

describe("gestures", () => {
  it("a swipe needs distance and a mostly horizontal drag", () => {
    expect(swipeStep(-80, 5)).toBe(1);
    expect(swipeStep(80, 5)).toBe(-1);
    expect(swipeStep(-30, 0)).toBe(0);
    expect(swipeStep(-80, 70)).toBe(0); // a vertical scroll
  });

  it("never goes past today", () => {
    expect(swipeTarget("2026-11-06", "2026-11-06", -90, 0)).toBeNull();
    expect(swipeTarget("2026-11-06", "2026-11-06", 90, 0)).toBe("2026-11-05");
    expect(swipeTarget("2026-11-05", "2026-11-06", -90, 0)).toBe("2026-11-06");
    expect(swipeTarget("2026-11-05", "2026-11-06", -10, 0)).toBeNull();
  });

  it("a movement over 10 px cancels the long press", () => {
    expect(cancelsHold(6, 6)).toBe(false);
    expect(cancelsHold(8, 8)).toBe(true);
    expect(cancelsHold(0, -11)).toBe(true);
  });
});

describe("exact value and note", () => {
  it("parseValue accepts whole numbers up to 100000", () => {
    expect(parseValue("6")).toBe(6);
    expect(parseValue("0")).toBe(0);
    expect(parseValue(" 12 ")).toBe(12);
    expect(parseValue("")).toBeNull();
    expect(parseValue("-1")).toBeNull();
    expect(parseValue("1.5")).toBeNull();
    expect(parseValue("100001")).toBeNull();
  });

  it("noteLength counts characters like the server", () => {
    expect(noteLength("é💊")).toBe(2);
  });
});

describe("celebration", () => {
  it("only when the day turns complete", () => {
    expect(shouldCelebrate(false, { ...day, complete: true })).toBe(true);
    expect(shouldCelebrate(true, { ...day, complete: true })).toBe(false); // already complete
    expect(shouldCelebrate(false, { ...day, complete: false })).toBe(false);
  });
});
