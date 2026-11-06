import { describe, expect, it } from "vitest";
import {
  allDays,
  describeRule,
  emptyForm,
  groupByMoment,
  hasWeekday,
  toggleWeekday,
  toInput,
  withHabit,
} from "./habits";
import type { Habit } from "./types";

function habit(overrides: Partial<Habit>): Habit {
  return {
    id: 1,
    name: "Vitamine D",
    emoji: "💊",
    color: "amber",
    moment: "morning",
    position: 0,
    unit: "pilules",
    step: 1,
    period: "day",
    target: 2,
    weekdays: allDays,
    archivedOn: null,
    ...overrides,
  };
}

describe("weekdays mask", () => {
  it("toggles one day, Monday being bit 0", () => {
    const withoutMonday = toggleWeekday(allDays, 0);
    expect(withoutMonday).toBe(0b1111110);
    expect(hasWeekday(withoutMonday, 0)).toBe(false);
    expect(hasWeekday(withoutMonday, 6)).toBe(true);
    expect(toggleWeekday(withoutMonday, 0)).toBe(allDays);
  });
});

describe("toInput", () => {
  it("sends a yes/no habit as a target of 1 without unit", () => {
    const input = toInput({ ...emptyForm, name: " Méditer ", unit: "min", target: "5", step: "5" });
    expect(input).toMatchObject({ name: "Méditer", unit: "", target: 1, step: 1 });
  });

  it("sends the quantity fields as numbers", () => {
    const input = toInput({
      ...emptyForm,
      kind: "quantity",
      target: "2",
      unit: " pilules ",
      step: "1",
    });
    expect(input).toMatchObject({ unit: "pilules", target: 2, step: 1 });
  });
});

describe("groupByMoment", () => {
  it("orders the moments and hides the empty ones", () => {
    const groups = groupByMoment([
      habit({ id: 1, moment: "evening" }),
      habit({ id: 2, moment: "morning" }),
    ]);
    expect(groups.map((g) => g.moment)).toEqual(["morning", "evening"]);
  });
});

describe("withHabit", () => {
  it("puts the new habit last in its moment", () => {
    const list = [
      habit({ id: 1, moment: "morning", position: 0 }),
      habit({ id: 2, moment: "evening", position: 0 }),
    ];
    const next = withHabit(list, habit({ id: 3, moment: "morning", position: 1 }));
    expect(next.map((h) => h.id)).toEqual([1, 3, 2]);
  });
});

describe("describeRule", () => {
  it("describes a quantity", () => {
    expect(describeRule(habit({}))).toBe("2 pilules par jour");
  });

  it("describes a yes/no habit on some days", () => {
    expect(describeRule(habit({ unit: "", target: 1, weekdays: 0b0011111 }))).toBe(
      "Oui / non par jour · lun, mar, mer, jeu, ven",
    );
  });

  it("describes a goal on a period", () => {
    expect(describeRule(habit({ unit: "séances", target: 150, period: "year" }))).toBe(
      "150 séances par an",
    );
  });
});
