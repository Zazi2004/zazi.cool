import { describe, expect, it } from "vitest";
import palettes from "./backgrounds.json";
import { backgroundNames } from "./policy.ts";

// The contrast test of the Go side (internal/web/contrast_test.go) reads this file:
// every animated background needs both modes, an opacity and #rrggbb colors.
describe("backgrounds.json", () => {
  for (const name of backgroundNames) {
    for (const mode of ["light", "dark"] as const) {
      it(`${name} / ${mode}`, () => {
        const { opacity, colors } = palettes[name][mode];
        expect(opacity).toBeGreaterThan(0);
        expect(opacity).toBeLessThan(1);
        expect(colors.length).toBeGreaterThanOrEqual(2);
        for (const color of colors) expect(color).toMatch(/^#[0-9a-f]{6}$/);
      });
    }
  }
});
