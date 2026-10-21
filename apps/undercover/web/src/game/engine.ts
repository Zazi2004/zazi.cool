// Game engine: pure functions that take a game and return the next one. The UI only
// calls these functions and renders the resulting phase. Randomness is injected
// (`rng`) so that tests are deterministic.

import type { DrawResponse } from "../api/types";

export type Role = "civil" | "undercover" | "mrwhite";

/** Which pairs can be drawn: all-ages only, all-ages and extreme, or extreme only. */
export type ExtremeMode = "off" | "on" | "only";

export interface Settings {
  playerCount: number;
  undercoverCount: number;
  mrWhiteCount: number;
  mrWhiteCannotStart: boolean;
  undercoverCannotStart: boolean;
  extreme: ExtremeMode;
  suggestionsOnly: boolean;
}

export const MIN_PLAYERS = 3;
export const MAX_PLAYERS = 20;
export const MAX_NAME_LENGTH = 20;

export const defaultSettings: Settings = {
  playerCount: 5,
  undercoverCount: 1,
  mrWhiteCount: 0,
  mrWhiteCannotStart: true,
  undercoverCannotStart: false,
  extreme: "off",
  suggestionsOnly: false,
};

export interface Player {
  name: string;
  role: Role;
  /** null for Mr. White, who gets no word. */
  word: string | null;
  alive: boolean;
}

export type Winner = "civils" | "impostors" | "mrwhite";

export interface Outcome {
  winner: Winner;
  /** Index of the Mr. White who guessed the word, when winner is "mrwhite". */
  mrWhite?: number;
}

export type Phase =
  /** Players look at their word one after the other; `next` is the next player. */
  | { kind: "distribution"; next: number }
  /** Players give a clue in `order` (player indexes), then vote orally. */
  | { kind: "round"; order: number[] }
  /** The role of an eliminated civil or undercover is announced to everyone. */
  | { kind: "reveal"; eliminated: number }
  /** An eliminated Mr. White types the word they think the civilians have. */
  | { kind: "guess"; eliminated: number }
  | { kind: "guessResult"; eliminated: number; guess: string; expected: string; correct: boolean }
  | { kind: "over"; outcome: Outcome };

export interface Game {
  settings: Settings;
  pair: DrawResponse;
  players: Player[];
  round: number;
  phase: Phase;
}

/** Returns a number in [0, 1), like Math.random. */
export type Rng = () => number;

export function impostorCount(s: Settings): number {
  return s.undercoverCount + s.mrWhiteCount;
}

export function civilCount(s: Settings): number {
  return s.playerCount - impostorCount(s);
}

/** Returns the problems with the settings, in French; empty when valid. */
export function validateSettings(s: Settings): string[] {
  const errors: string[] = [];
  const counts = [s.playerCount, s.undercoverCount, s.mrWhiteCount];
  if (!counts.every((n) => Number.isInteger(n) && n >= 0)) {
    return ["Les nombres de joueurs doivent être des entiers positifs."];
  }
  if (s.playerCount < MIN_PLAYERS || s.playerCount > MAX_PLAYERS) {
    errors.push(`Il faut entre ${MIN_PLAYERS} et ${MAX_PLAYERS} joueurs.`);
  }
  if (impostorCount(s) < 1) {
    errors.push("Il faut au moins un undercover ou un Mr. White.");
  }
  const civils = civilCount(s);
  if (civils < 2 || civils <= impostorCount(s)) {
    errors.push("Les civils doivent être au moins 2 et plus nombreux que les imposteurs.");
  }
  return errors;
}

/** Returns the problems with the player names, in French; empty when valid. */
export function validateNames(names: string[]): string[] {
  const errors: string[] = [];
  const trimmed = names.map((n) => n.trim());
  if (trimmed.some((n) => n === "")) {
    errors.push("Chaque joueur doit avoir un prénom.");
  }
  if (trimmed.some((n) => n.length > MAX_NAME_LENGTH)) {
    errors.push(`Les prénoms font au plus ${MAX_NAME_LENGTH} caractères.`);
  }
  const keys = trimmed.filter((n) => n !== "").map((n) => n.toLocaleLowerCase("fr"));
  if (new Set(keys).size !== keys.length) {
    errors.push("Deux joueurs ne peuvent pas avoir le même prénom.");
  }
  return errors;
}

/** Fisher–Yates shuffle, returning a new array. */
export function shuffle<T>(items: readonly T[], rng: Rng): T[] {
  const out = [...items];
  for (let i = out.length - 1; i > 0; i--) {
    const j = Math.floor(rng() * (i + 1));
    [out[i], out[j]] = [out[j]!, out[i]!];
  }
  return out;
}

export function createGame(
  settings: Settings,
  names: string[],
  pair: DrawResponse,
  rng: Rng,
): Game {
  const problems = [...validateSettings(settings), ...validateNames(names)];
  if (names.length !== settings.playerCount) {
    problems.push("Le nombre de prénoms ne correspond pas au nombre de joueurs.");
  }
  if (problems.length > 0) {
    throw new Error(problems.join(" "));
  }

  const roles: Role[] = shuffle(
    [
      ...Array<Role>(civilCount(settings)).fill("civil"),
      ...Array<Role>(settings.undercoverCount).fill("undercover"),
      ...Array<Role>(settings.mrWhiteCount).fill("mrwhite"),
    ],
    rng,
  );
  const players = names.map((name, i): Player => {
    const role = roles[i]!;
    return { name: name.trim(), role, word: wordFor(role, pair), alive: true };
  });
  return { settings, pair, players, round: 0, phase: { kind: "distribution", next: 0 } };
}

function wordFor(role: Role, pair: DrawResponse): string | null {
  switch (role) {
    case "civil":
      return pair.civilWord;
    case "undercover":
      return pair.undercoverWord;
    case "mrwhite":
      return null;
  }
}

/** The current player has seen their word: pass to the next one, or start round 1. */
export function wordSeen(game: Game, rng: Rng): Game {
  const phase = expectPhase(game, "distribution");
  const next = phase.next + 1;
  if (next < game.players.length) {
    return { ...game, phase: { kind: "distribution", next } };
  }
  return startRound(game, rng);
}

/**
 * Starts a new round with a random speaking order. The first speaker is drawn among
 * the players allowed to start; with valid settings there is always a living civil.
 */
export function startRound(game: Game, rng: Rng): Game {
  const alive = game.players.flatMap((p, i) => (p.alive ? [i] : []));
  const order = shuffle(alive, rng);
  const first = order.findIndex((i) => canStart(game.players[i]!, game.settings));
  if (first > 0) {
    // Swap the first allowed player to the front; the order stays random.
    [order[0], order[first]] = [order[first]!, order[0]!];
  }
  return { ...game, round: game.round + 1, phase: { kind: "round", order } };
}

function canStart(p: Player, s: Settings): boolean {
  if (p.role === "mrwhite" && s.mrWhiteCannotStart) return false;
  if (p.role === "undercover" && s.undercoverCannotStart) return false;
  return true;
}

/** The vote ended in a tie or without a decision: nobody leaves, next round. */
export function skipElimination(game: Game, rng: Rng): Game {
  expectPhase(game, "round");
  return startRound(game, rng);
}

/**
 * Eliminates the voted player. An eliminated Mr. White guesses the word; any other role
 * is announced first, even when this elimination ends the game.
 */
export function eliminate(game: Game, index: number): Game {
  expectPhase(game, "round");
  const target = game.players[index];
  if (!target || !target.alive) {
    throw new Error(`Player ${index} cannot be eliminated.`);
  }
  const players = game.players.map((p, i) => (i === index ? { ...p, alive: false } : p));
  const kind = target.role === "mrwhite" ? "guess" : "reveal";
  return { ...game, players, phase: { kind, eliminated: index } };
}

/** Records Mr. White's guess, as judged by the server. */
export function recordGuess(game: Game, guess: string, expected: string, correct: boolean): Game {
  const phase = expectPhase(game, "guess");
  return {
    ...game,
    phase: { kind: "guessResult", eliminated: phase.eliminated, guess, expected, correct },
  };
}

/** The players judge a refused guess as right after all. */
export function acceptGuess(game: Game): Game {
  const phase = expectPhase(game, "guessResult");
  return { ...game, phase: { ...phase, correct: true } };
}

/** Leaves a reveal or a guess result: the game ends or a new round starts. */
export function continueGame(game: Game, rng: Rng): Game {
  const { phase } = game;
  if (phase.kind !== "reveal" && phase.kind !== "guessResult") {
    throw new Error(`Expected phase reveal or guessResult, got ${phase.kind}.`);
  }
  const outcome =
    phase.kind === "guessResult" && phase.correct
      ? { winner: "mrwhite" as const, mrWhite: phase.eliminated }
      : outcomeOf(game.players);
  if (outcome) {
    return { ...game, phase: { kind: "over", outcome } };
  }
  return startRound(game, rng);
}

/**
 * Civilians win when no impostor is left; impostors win when a single civilian is
 * left. Mr. White's own victory is handled by his guess.
 */
export function outcomeOf(players: Player[]): Outcome | null {
  const alive = players.filter((p) => p.alive);
  const civils = alive.filter((p) => p.role === "civil").length;
  const impostors = alive.length - civils;
  if (impostors === 0) return { winner: "civils" };
  if (civils <= 1) return { winner: "impostors" };
  return null;
}

function expectPhase<K extends Phase["kind"]>(game: Game, kind: K): Extract<Phase, { kind: K }> {
  if (game.phase.kind !== kind) {
    throw new Error(`Expected phase ${kind}, got ${game.phase.kind}.`);
  }
  return game.phase as Extract<Phase, { kind: K }>;
}
