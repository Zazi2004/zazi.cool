import { describe, expect, it } from "vitest";
import { hexToRgb, toOklab, unwrapHues } from "./color.ts";
import { renderSize } from "./gl.ts";

describe("renderSize", () => {
  it("caps the device ratio at 1.5", () => {
    expect(renderSize(400, 800, 3)).toEqual({ width: 600, height: 1200 });
  });
  it("keeps a ratio of 1", () => {
    expect(renderSize(1280, 720, 1)).toEqual({ width: 1280, height: 720 });
  });
  it("caps the longest side at 1920 pixels", () => {
    const size = renderSize(2560, 1440, 2);
    expect(size.width).toBe(1920);
    expect(size.height).toBe(1080);
  });
  it("never gives an empty canvas", () => {
    expect(renderSize(0, 0, 2)).toEqual({ width: 1, height: 1 });
  });
});

describe("color", () => {
  it("parses #rrggbb", () => {
    expect(hexToRgb("#ff8000")).toEqual([1, 128 / 255, 0]);
  });
  it("converts white and black to Oklab", () => {
    const [l, a, b] = toOklab([1, 1, 1]);
    expect(l).toBeCloseTo(1, 3);
    expect(Math.hypot(a, b)).toBeLessThan(0.001);
    expect(toOklab([0, 0, 0])[0]).toBeCloseTo(0, 5);
  });
  it("keeps hues continuous between stops", () => {
    const [, second] = unwrapHues([
      [0.7, 0.1, 3],
      [0.7, 0.1, -3],
    ]);
    expect(second?.[2]).toBeCloseTo(-3 + 2 * Math.PI, 5);
  });
});
