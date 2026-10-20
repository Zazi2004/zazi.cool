import { afterEach, describe, expect, it, vi } from "vitest";
import { defaultSettings, type Game, type Settings } from "./engine";
import { discardFinishedGame, loadGame, loadHaptics, loadSettings, saveGame } from "./storage";

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

describe("loadHaptics", () => {
  it("is off by default, including when storage is unavailable", () => {
    expect(loadHaptics()).toBe(false);
  });
});

describe("discardFinishedGame", () => {
  const store = new Map<string, string>();
  vi.stubGlobal("localStorage", {
    getItem: (k: string) => store.get(k) ?? null,
    setItem: (k: string, v: string) => void store.set(k, v),
    removeItem: (k: string) => void store.delete(k),
  });
  afterEach(() => store.clear());

  const game = (phase: Game["phase"]): Game => ({
    settings: defaultSettings,
    pair: { id: 1, civilWord: "Chat", undercoverWord: "Tigre", civilSide: "a", extreme: false },
    players: [],
    round: 1,
    phase,
  });

  // Regression: "Home" then "New game" reopened the previous end screen.
  it("forgets a finished game so a new one can start", () => {
    saveGame(game({ kind: "over", outcome: { winner: "civils" } as never }));
    discardFinishedGame();
    expect(loadGame()).toBeNull();
  });

  it("keeps a game in progress", () => {
    saveGame(game({ kind: "round", order: [] }));
    discardFinishedGame();
    expect(loadGame()).not.toBeNull();
  });
});
