// Browser persistence: the current game survives a reload or the phone going to
// sleep, and the last settings and names are remembered for the next game.
// Storage may be unavailable (private browsing): every access is guarded and the
// app keeps working, only without memory.

import { defaultSettings, type ExtremeMode, type Game, type Settings } from "./engine";

const KEYS = {
  game: "undercover.v1.game",
  setup: "undercover.v1.setup",
  played: "undercover.v1.played",
  haptics: "undercover.v1.haptics",
} as const;

/** How many played pairs are remembered to avoid repeats. */
const MAX_PLAYED = 200;

function read<T>(key: string): T | null {
  try {
    const raw = localStorage.getItem(key);
    return raw === null ? null : (JSON.parse(raw) as T);
  } catch {
    return null;
  }
}

function write(key: string, value: unknown): void {
  try {
    if (value === null) localStorage.removeItem(key);
    else localStorage.setItem(key, JSON.stringify(value));
  } catch {
    // Storage full or disabled: the game goes on without persistence.
  }
}

export function loadGame(): Game | null {
  const game = read<Game>(KEYS.game);
  // A minimal sanity check: anything else is a stale or foreign value.
  if (!game || !Array.isArray(game.players) || !game.phase || !game.pair) return null;
  return { ...game, settings: loadSettings(game.settings) };
}

export function saveGame(game: Game | null): void {
  write(KEYS.game, game);
}

/**
 * Forgets a finished game. Without this, "New game" from the home page reopened the
 * previous end screen, because the finished game was still the saved one.
 */
export function discardFinishedGame(): void {
  if (loadGame()?.phase.kind === "over") saveGame(null);
}

export interface Setup {
  settings: Settings;
  names: string[];
}

const EXTREME_MODES: readonly unknown[] = ["off", "on", "only"] satisfies ExtremeMode[];

/**
 * Fills missing settings with the defaults. The extreme mode used to be a boolean:
 * values saved by earlier versions are converted.
 */
export function loadSettings(saved: Partial<Settings> | undefined): Settings {
  const settings = { ...defaultSettings, ...saved };
  const extreme: unknown = settings.extreme;
  if (extreme === true) settings.extreme = "on";
  else if (!EXTREME_MODES.includes(extreme)) settings.extreme = "off";
  return settings;
}

export function loadSetup(): Setup {
  const setup = read<Partial<Setup>>(KEYS.setup);
  return {
    settings: loadSettings(setup?.settings),
    names: Array.isArray(setup?.names) ? setup.names : [],
  };
}

export function saveSetup(setup: Setup): void {
  write(KEYS.setup, setup);
}

export function loadPlayed(): number[] {
  const played = read<unknown>(KEYS.played);
  return Array.isArray(played) ? played.filter((id) => Number.isInteger(id)) : [];
}

export function addPlayed(id: number): void {
  const played = loadPlayed().filter((p) => p !== id);
  played.push(id);
  write(KEYS.played, played.slice(-MAX_PLAYED));
}

/** Vibration feedback is opt-in: off until the player turns it on. */
export function loadHaptics(): boolean {
  return read<unknown>(KEYS.haptics) === true;
}

export function saveHaptics(enabled: boolean): void {
  write(KEYS.haptics, enabled ? true : null);
}
