import { describe, expect, it } from "vitest";
import habitGolden from "./golden/habit.json";
import habitListGolden from "./golden/habit-list.json";
import type { Habit, HabitList } from "./types";

// JSON imports widen string literals to `string`; this maps a type to that widened shape
// so the golden files can be checked against the real types with `satisfies`.
type Widen<T> = T extends string
  ? string
  : T extends number
    ? number
    : T extends boolean
      ? boolean
      : T extends null
        ? null
        : T extends readonly (infer U)[]
          ? Widen<U>[]
          : { [K in keyof T]: Widen<T[K]> };

// Same keys on both sides: `satisfies` alone would accept a field the type lacks.
type SameKeys<A, B> = [keyof A] extends [keyof B]
  ? [keyof B] extends [keyof A]
    ? true
    : never
  : never;
const same = <A, B>(_: SameKeys<A, B>) => undefined;

const habit = habitGolden satisfies Widen<Habit>;
same<typeof habitGolden, Habit>(true);

const habitList = habitListGolden satisfies Widen<HabitList>;
same<typeof habitListGolden, HabitList>(true);
same<(typeof habitListGolden.active)[number], Habit>(true);
same<(typeof habitListGolden.archived)[number], Habit>(true);

// The checks above are compile-time (tsc runs before vitest in `npm test`); these make
// sure the files are loaded and carry what the screens rely on.
describe("golden files", () => {
  it("Habit", () => {
    expect(habit.name).toBe("Vitamine D");
    expect(habit.archivedOn).toBeNull();
  });

  it("habit list", () => {
    expect(habitList.active.length).toBeGreaterThan(0);
    expect(habitList.archived[0]?.archivedOn).toMatch(/^\d{4}-\d{2}-\d{2}$/);
  });
});
