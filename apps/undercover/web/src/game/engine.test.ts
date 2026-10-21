import { describe, expect, it } from "vitest";
import type { DrawResponse } from "../api/types";
import {
  acceptGuess,
  continueGame,
  createGame,
  defaultSettings,
  eliminate,
  type Game,
  outcomeOf,
  type Player,
  recordGuess,
  type Rng,
  type Settings,
  shuffle,
  skipElimination,
  startRound,
  validateNames,
  validateSettings,
  wordSeen,
} from "./engine";

const pair: DrawResponse = {
  id: 1,
  civilWord: "Chat",
  undercoverWord: "Tigre",
  civilSide: "a",
  extreme: false,
};

/** A deterministic pseudo-random generator (mulberry32). */
function seeded(seed: number): Rng {
  return () => {
    seed = (seed + 0x6d2b79f5) | 0;
    let t = Math.imul(seed ^ (seed >>> 15), 1 | seed);
    t = (t + Math.imul(t ^ (t >>> 7), 61 | t)) ^ t;
    return ((t ^ (t >>> 14)) >>> 0) / 4294967296;
  };
}

const names = (n: number) => Array.from({ length: n }, (_, i) => `Joueur ${i + 1}`);

function settings(overrides: Partial<Settings>): Settings {
  return { ...defaultSettings, ...overrides };
}

/** Builds a game in round 1 with the given roles, in order. */
function gameWith(roles: Player["role"][], overrides: Partial<Settings> = {}): Game {
  const players = roles.map((role, i): Player => ({
    name: `J${i}`,
    role,
    word: role === "civil" ? "Chat" : role === "undercover" ? "Tigre" : null,
    alive: true,
  }));
  const s = settings({
    playerCount: roles.length,
    undercoverCount: roles.filter((r) => r === "undercover").length,
    mrWhiteCount: roles.filter((r) => r === "mrwhite").length,
    ...overrides,
  });
  return startRound(
    { settings: s, pair, players, round: 0, phase: { kind: "distribution", next: 0 } },
    seeded(1),
  );
}

describe("validateSettings", () => {
  it("accepts the defaults", () => {
    expect(validateSettings(defaultSettings)).toEqual([]);
  });

  it.each([
    ["too few players", { playerCount: 2 }],
    ["too many players", { playerCount: 21 }],
    ["no impostor", { undercoverCount: 0, mrWhiteCount: 0 }],
    ["civils not a majority", { playerCount: 4, undercoverCount: 1, mrWhiteCount: 1 }],
    ["negative count", { mrWhiteCount: -1 }],
    ["non-integer count", { playerCount: 4.5 }],
  ])("rejects %s", (_, overrides) => {
    expect(validateSettings(settings(overrides))).not.toEqual([]);
  });

  it("accepts the smallest game", () => {
    expect(
      validateSettings(settings({ playerCount: 3, undercoverCount: 0, mrWhiteCount: 1 })),
    ).toEqual([]);
  });
});

describe("validateNames", () => {
  it("accepts distinct names", () => {
    expect(validateNames(["Alice", "Bob", "Chloé"])).toEqual([]);
  });

  it("rejects empty, too long and duplicate names", () => {
    expect(validateNames(["Alice", " "])).toHaveLength(1);
    expect(validateNames(["A".repeat(21)])).toHaveLength(1);
    expect(validateNames(["Alice", "alice "])).toHaveLength(1);
  });
});

describe("shuffle", () => {
  it("keeps every item", () => {
    const items = [1, 2, 3, 4, 5, 6];
    expect(shuffle(items, seeded(42)).sort()).toEqual(items);
  });
});

describe("createGame", () => {
  it("deals the roles and words", () => {
    const s = settings({ playerCount: 7, undercoverCount: 2, mrWhiteCount: 1 });
    const game = createGame(s, names(7), pair, seeded(3));

    const count = (role: Player["role"]) => game.players.filter((p) => p.role === role).length;
    expect([count("civil"), count("undercover"), count("mrwhite")]).toEqual([4, 2, 1]);
    for (const p of game.players) {
      const expected = { civil: "Chat", undercover: "Tigre", mrwhite: null }[p.role];
      expect(p.word).toBe(expected);
    }
    expect(game.phase).toEqual({ kind: "distribution", next: 0 });
  });

  it("refuses invalid settings or names", () => {
    expect(() => createGame(settings({ playerCount: 3 }), names(2), pair, seeded(1))).toThrow();
    expect(() => createGame(settings({ undercoverCount: 0 }), names(5), pair, seeded(1))).toThrow();
  });
});

describe("distribution", () => {
  // The start of the game never announces a role: civilians and undercovers only get
  // their word (the screens are covered in ui/screens.test.ts).
  it("never goes through a reveal before round 1", () => {
    const s = settings({ playerCount: 7, undercoverCount: 2, mrWhiteCount: 1 });
    let game = createGame(s, names(7), pair, seeded(4));
    while (game.phase.kind === "distribution") {
      game = wordSeen(game, seeded(4));
    }
    expect(game.phase.kind).toBe("round");
    expect(game.players.every((p) => p.alive)).toBe(true);
  });

  it("shows each word once, then starts round 1", () => {
    let game = createGame(settings({ playerCount: 3 }), names(3), pair, seeded(1));
    game = wordSeen(game, seeded(1));
    game = wordSeen(game, seeded(1));
    expect(game.phase).toEqual({ kind: "distribution", next: 2 });
    game = wordSeen(game, seeded(1));
    expect(game.phase.kind).toBe("round");
    expect(game.round).toBe(1);
  });
});

describe("startRound", () => {
  it("orders every living player", () => {
    const game = gameWith(["civil", "civil", "civil", "undercover"]);
    const next = continueGame(eliminate(game, 0), seeded(5));
    if (next.phase.kind !== "round") throw new Error("expected a round");
    expect([...next.phase.order].sort()).toEqual([1, 2, 3]);
  });

  it("never lets a forbidden role start", () => {
    for (let seed = 0; seed < 200; seed++) {
      const game = startRound(
        gameWith(["mrwhite", "undercover", "civil", "civil", "civil"], {
          mrWhiteCannotStart: true,
          undercoverCannotStart: true,
        }),
        seeded(seed),
      );
      if (game.phase.kind !== "round") throw new Error("expected a round");
      expect(game.players[game.phase.order[0]!]!.role).toBe("civil");
    }
  });

  it("lets anyone start when allowed", () => {
    const starters = new Set<string>();
    for (let seed = 0; seed < 200; seed++) {
      const game = startRound(
        gameWith(["mrwhite", "undercover", "civil", "civil", "civil"], {
          mrWhiteCannotStart: false,
          undercoverCannotStart: false,
        }),
        seeded(seed),
      );
      if (game.phase.kind !== "round") throw new Error("expected a round");
      starters.add(game.players[game.phase.order[0]!]!.role);
    }
    expect(starters).toEqual(new Set(["mrwhite", "undercover", "civil"]));
  });
});

describe("elimination and victory", () => {
  it("announces an eliminated civil", () => {
    const game = eliminate(gameWith(["civil", "civil", "civil", "civil", "undercover"]), 0);
    expect(game.phase).toEqual({ kind: "reveal", eliminated: 0 });
    expect(game.round).toBe(1);
    expect(game.players[0]!.alive).toBe(false);
  });

  it("announces an eliminated undercover", () => {
    const game = eliminate(gameWith(["civil", "civil", "civil", "undercover", "undercover"]), 3);
    expect(game.phase).toEqual({ kind: "reveal", eliminated: 3 });
  });

  it("starts the next round after the announcement while the game is undecided", () => {
    const game = continueGame(
      eliminate(gameWith(["civil", "civil", "civil", "civil", "undercover"]), 0),
      seeded(1),
    );
    expect(game.phase.kind).toBe("round");
    expect(game.round).toBe(2);
  });

  it("announces the elimination that lets civilians win, then ends the game", () => {
    const game = eliminate(gameWith(["civil", "civil", "undercover"]), 2);
    expect(game.phase).toEqual({ kind: "reveal", eliminated: 2 });
    expect(continueGame(game, seeded(1)).phase).toEqual({
      kind: "over",
      outcome: { winner: "civils" },
    });
  });

  it("announces the elimination that lets impostors win, then ends the game", () => {
    const game = eliminate(gameWith(["civil", "civil", "undercover"]), 0);
    expect(game.phase).toEqual({ kind: "reveal", eliminated: 0 });
    expect(continueGame(game, seeded(1)).phase).toEqual({
      kind: "over",
      outcome: { winner: "impostors" },
    });
  });

  it("starts a new round without elimination nor announcement", () => {
    const game = skipElimination(gameWith(["civil", "civil", "undercover"]), seeded(2));
    expect(game.phase.kind).toBe("round");
    expect(game.round).toBe(2);
    expect(game.players.every((p) => p.alive)).toBe(true);
  });

  it("refuses to eliminate a dead player", () => {
    const game = continueGame(
      eliminate(gameWith(["civil", "civil", "civil", "civil", "undercover"]), 0),
      seeded(1),
    );
    expect(() => eliminate(game, 0)).toThrow();
  });

  it("refuses to continue from a phase that is not a reveal or a guess result", () => {
    expect(() => continueGame(gameWith(["civil", "civil", "undercover"]), seeded(1))).toThrow();
  });
});

describe("Mr. White", () => {
  const roles: Player["role"][] = ["civil", "civil", "civil", "mrwhite", "mrwhite"];

  it("asks an eliminated Mr. White to guess", () => {
    const game = eliminate(gameWith(roles), 3);
    expect(game.phase).toEqual({ kind: "guess", eliminated: 3 });
  });

  it("makes the first Mr. White who guesses right win alone", () => {
    let game = recordGuess(eliminate(gameWith(roles), 3), "chat", "Chat", true);
    game = continueGame(game, seeded(1));
    expect(game.phase).toEqual({ kind: "over", outcome: { winner: "mrwhite", mrWhite: 3 } });
  });

  it("continues after a wrong guess", () => {
    const game = recordGuess(eliminate(gameWith(roles), 3), "chien", "Chat", false);
    expect(continueGame(game, seeded(1)).phase.kind).toBe("round");
  });

  it("lets players accept a refused guess", () => {
    let game = recordGuess(eliminate(gameWith(roles), 3), "chqt", "Chat", false);
    game = continueGame(acceptGuess(game), seeded(1));
    expect(game.phase).toMatchObject({ kind: "over", outcome: { winner: "mrwhite" } });
  });

  it("lets civilians win when the last Mr. White guesses wrong", () => {
    const game = recordGuess(
      eliminate(gameWith(["civil", "civil", "mrwhite"]), 2),
      "x",
      "Chat",
      false,
    );
    expect(continueGame(game, seeded(1)).phase).toEqual({
      kind: "over",
      outcome: { winner: "civils" },
    });
  });
});

describe("outcomeOf", () => {
  it("is undecided at the start", () => {
    expect(outcomeOf(gameWith(["civil", "civil", "undercover"]).players)).toBeNull();
  });
});
