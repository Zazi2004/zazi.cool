import { loadHaptics } from "../game/storage";

/** Short pulses for the key moments of the game. */
export const pulses = {
  secret: 40,
  verdict: [60, 40, 120],
  victory: [80, 50, 80, 50, 160],
} as const;

/** Whether this browser can vibrate at all (iOS Safari cannot). */
export function hapticsSupported(): boolean {
  return typeof navigator.vibrate === "function";
}

/** Vibrates only when the player opted in; a no-op everywhere else. */
export function buzz(pattern: number | readonly number[]): void {
  if (!hapticsSupported() || !loadHaptics()) return;
  navigator.vibrate(typeof pattern === "number" ? pattern : [...pattern]);
}
