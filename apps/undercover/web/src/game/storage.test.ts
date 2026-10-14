import { describe, expect, it } from "vitest";
import { defaultSettings, type Settings } from "./engine";
import { loadSettings } from "./storage";

describe("loadSettings", () => {
  it("fills missing settings with the defaults", () => {
    expect(loadSettings(undefined)).toEqual(defaultSettings);
    expect(loadSettings({ playerCount: 7 })).toEqual({ ...defaultSettings, playerCount: 7 });
  });

  it("keeps a valid extreme mode", () => {
    expect(loadSettings({ extreme: "only" }).extreme).toBe("only");
  });

  it("converts the boolean extreme setting of earlier versions", () => {
    const legacy = (extreme: unknown) => ({ extreme }) as unknown as Partial<Settings>;
    expect(loadSettings(legacy(true)).extreme).toBe("on");
    expect(loadSettings(legacy(false)).extreme).toBe("off");
    expect(loadSettings(legacy("nonsense")).extreme).toBe("off");
  });
});
