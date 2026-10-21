// Screen content tests. There is no DOM in the test environment: components are called
// as plain functions with stubbed hooks, and the returned virtual nodes are flattened
// into the text and attributes they would put in the DOM.

import type { ComponentChildren, VNode } from "preact";
import { describe, expect, it, vi } from "vitest";
import type { DrawResponse } from "../api/types";
import {
  createGame,
  defaultSettings,
  eliminate,
  type Game,
  type Player,
  startRound,
  wordSeen,
} from "../game/engine";
import { SecretCard } from "./common";
import { DistributionScreen } from "./DistributionScreen";
import { RevealScreen } from "./RevealScreen";

vi.mock("preact/hooks", () => ({
  useState: <T>(initial: T) => [initial, () => {}],
  useCallback: <T>(fn: T) => fn,
  useEffect: () => {},
  useRef: () => ({ current: null }),
}));

/** Function component, the only kind this app uses. */
type Component = (props: Record<string, unknown>) => ComponentChildren;

/**
 * Every text and attribute value the node would put in the DOM. Adjacent texts are
 * joined as in `textContent`; elements are separated by spaces.
 */
function markup(node: ComponentChildren): string {
  if (node === null || node === undefined || typeof node === "boolean") return "";
  if (typeof node === "string" || typeof node === "number") return String(node);
  if (Array.isArray(node)) return node.map(markup).join("");
  const { type, props } = node as VNode<Record<string, unknown>>;
  if (typeof type === "function") return markup((type as Component)(props));
  const attributes = Object.entries(props)
    .filter(([name, value]) => name !== "children" && typeof value === "string")
    .map(([, value]) => value);
  return ` ${attributes.join(" ")} ${markup(props.children as ComponentChildren)} `;
}

const pair: DrawResponse = {
  id: 1,
  civilWord: "Chat",
  undercoverWord: "Tigre",
  civilSide: "a",
  extreme: false,
};
const onChange = () => {};

/** A game in round 1 with the given roles, in order. */
function gameWith(roles: Player["role"][]): Game {
  const players = roles.map((role, i): Player => ({
    name: `J${i}`,
    role,
    word: role === "civil" ? "Chat" : role === "undercover" ? "Tigre" : null,
    alive: true,
  }));
  const settings = {
    ...defaultSettings,
    playerCount: roles.length,
    undercoverCount: roles.filter((r) => r === "undercover").length,
    mrWhiteCount: roles.filter((r) => r === "mrwhite").length,
  };
  return startRound(
    { settings, pair, players, round: 0, phase: { kind: "distribution", next: 0 } },
    () => 0,
  );
}

describe("distribution", () => {
  it("never announces the civil or undercover role", () => {
    const settings = { ...defaultSettings, playerCount: 7, undercoverCount: 2, mrWhiteCount: 1 };
    const names = ["Ana", "Bob", "Cléo", "Dan", "Eva", "Fil", "Gus"];
    let game = createGame(settings, names, pair, Math.random);
    while (game.phase.kind === "distribution") {
      const player = game.players[game.phase.next]!;
      // The covered card, then the secret shown once the player touches it.
      const screens =
        markup(DistributionScreen({ game, onChange })) + markup(SecretCard({ player }));
      expect(screens.toLowerCase()).not.toContain("civil");
      expect(screens.toLowerCase()).not.toContain("undercover");
      if (player.role === "mrwhite") expect(screens).toContain("Mr. White");
      else expect(screens).not.toContain("Mr. White");
      game = wordSeen(game, Math.random);
    }
  });
});

describe("RevealScreen", () => {
  it.each([
    ["civil", "Civil"],
    ["undercover", "Undercover"],
  ] as const)("announces an eliminated %s", (role, label) => {
    const game = eliminate(gameWith(["civil", "civil", "civil", role, "undercover"]), 3);
    const screen = markup(RevealScreen({ game, onChange }));
    expect(screen).toContain("Tour 1");
    expect(screen).toContain("J3 était…");
    expect(screen).toContain(label);
    expect(screen).toContain(`avatar-${role}`);
    expect(screen).toContain("Continuer");
  });

  it("shows neither the eliminated player's word nor the civilians' word", () => {
    for (const index of [0, 3]) {
      const game = eliminate(
        gameWith(["civil", "civil", "civil", "undercover", "undercover"]),
        index,
      );
      const screen = markup(RevealScreen({ game, onChange })).toLowerCase();
      expect(screen).not.toContain("chat");
      expect(screen).not.toContain("tigre");
    }
  });
});
