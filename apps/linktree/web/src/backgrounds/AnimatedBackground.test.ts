import { describe, expect, it } from "vitest";
import { animationAllowed } from "./AnimatedBackground";

describe("animationAllowed", () => {
  const ok = { webgl2: true, reducedMotion: false, saveData: false };

  it("runs when nothing prevents it", () => {
    expect(animationAllowed(ok)).toBe(true);
  });

  it.each([
    ["without WebGL 2", { ...ok, webgl2: false }],
    ["with prefers-reduced-motion", { ...ok, reducedMotion: true }],
    ["with Save-Data", { ...ok, saveData: true }],
  ])("falls back to the CSS background %s", (_, env) => {
    expect(animationAllowed(env)).toBe(false);
  });
});
