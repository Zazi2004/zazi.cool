import { describe, expect, it } from "vitest";
import { animationAllowed, backgroundOf } from "./policy.ts";

const env = { webgl2: true, reducedMotion: false, saveData: false };

// CA23 (F23): the rule that decides whether a background module is downloaded.
describe("animationAllowed", () => {
  it("allows animation with WebGL 2 and no preference", () => {
    expect(animationAllowed(env)).toBe(true);
  });
  it("refuses without WebGL 2", () => {
    expect(animationAllowed({ ...env, webgl2: false })).toBe(false);
  });
  it("refuses with prefers-reduced-motion", () => {
    expect(animationAllowed({ ...env, reducedMotion: true })).toBe(false);
  });
  it("refuses with Save-Data", () => {
    expect(animationAllowed({ ...env, saveData: true })).toBe(false);
  });
});

describe("backgroundOf", () => {
  it("knows the three animated backgrounds", () => {
    expect(backgroundOf("aurore")).toBe("aurore");
    expect(backgroundOf("fils")).toBe("fils");
    expect(backgroundOf("grain")).toBe("grain");
  });
  it("gives null for the static background, an unknown name, or none", () => {
    expect(backgroundOf("statique")).toBeNull();
    expect(backgroundOf("disco")).toBeNull();
    expect(backgroundOf(undefined)).toBeNull();
  });
});
