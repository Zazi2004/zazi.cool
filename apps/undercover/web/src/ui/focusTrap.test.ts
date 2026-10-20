import { describe, expect, it } from "vitest";
import { nextFocusIndex } from "./focusTrap";

describe("nextFocusIndex", () => {
  it("moves forward and wraps to the first element", () => {
    expect(nextFocusIndex(3, 0, false)).toBe(1);
    expect(nextFocusIndex(3, 2, false)).toBe(0);
  });

  it("moves backward and wraps to the last element", () => {
    expect(nextFocusIndex(3, 1, true)).toBe(0);
    expect(nextFocusIndex(3, 0, true)).toBe(2);
  });

  it("pulls the focus back in when it is outside the trap", () => {
    expect(nextFocusIndex(3, -1, false)).toBe(0);
    expect(nextFocusIndex(3, -1, true)).toBe(2);
  });
});
