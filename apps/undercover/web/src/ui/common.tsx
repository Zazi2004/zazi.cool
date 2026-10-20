import type { ComponentChildren } from "preact";
import { useEffect, useRef } from "preact/hooks";
import { buzz, pulses } from "./haptics";
import type { Player, Role } from "../game/engine";

export const roleLabels: Record<Role, string> = {
  civil: "Civil",
  undercover: "Undercover",
  mrwhite: "Mr. White",
};

export function ErrorText({ children }: { children: ComponentChildren }) {
  return (
    <p class="error" role="alert">
      {children}
    </p>
  );
}

interface StepperProps {
  label: string;
  value: number;
  min: number;
  max: number;
  onChange: (value: number) => void;
}

/** A number picker with large − / + buttons, easier than an input on a phone. */
export function Stepper({ label, value, min, max, onChange }: StepperProps) {
  return (
    <div class="stepper">
      <span class="stepper-label">{label}</span>
      <div class="stepper-controls">
        <button
          type="button"
          aria-label={`${label} : moins`}
          disabled={value <= min}
          onClick={() => onChange(value - 1)}
        >
          −
        </button>
        <output aria-live="polite">{value}</output>
        <button
          type="button"
          aria-label={`${label} : plus`}
          disabled={value >= max}
          onClick={() => onChange(value + 1)}
        >
          +
        </button>
      </div>
    </div>
  );
}

/** The secret shown to a single player: their word, or their Mr. White role. */
export function SecretCard({ player }: { player: Player }) {
  const ref = useFocusOnMount<HTMLDivElement>(pulses.secret);
  if (player.word === null) {
    return (
      <div class="secret" ref={ref} tabIndex={-1}>
        <p class="secret-label">Tu es</p>
        <p class="secret-word">Mr. White</p>
        <p class="hint">Tu n'as pas de mot : écoute les indices et fonds-toi dans la masse.</p>
      </div>
    );
  }
  return (
    <div class="secret" ref={ref} tabIndex={-1}>
      <p class="secret-label">Ton mot</p>
      <p class="secret-word">{player.word}</p>
    </div>
  );
}

/** Round badge with the player's initial; the tone follows the role once it is public. */
export function Avatar({ name, tone, large }: { name: string; tone?: Role; large?: boolean }) {
  const classes = ["avatar", tone ? `avatar-${tone}` : "", large ? "avatar-large" : ""];
  return (
    <span class={classes.filter(Boolean).join(" ")} aria-hidden="true">
      {[...name.trim()][0]?.toUpperCase() ?? "?"}
    </span>
  );
}

/**
 * Calls `onHide` when the page goes to the background. A secret must never stay on screen
 * after the phone is locked or the player switches app, so the caller re-hides it.
 */
export function useHideOnLeave(onHide: () => void) {
  useEffect(() => {
    const handler = () => {
      if (document.hidden) onHide();
    };
    document.addEventListener("visibilitychange", handler);
    return () => document.removeEventListener("visibilitychange", handler);
  }, [onHide]);
}

/** Moves the focus to the main content without scrolling. */
export function focusMain(): void {
  document.getElementById("main")?.focus({ preventScroll: true });
}

/**
 * Focus target for content that appears in place of the control that was just pressed
 * (secret word): without it the focus would fall back to the page for keyboard
 * and screen-reader users.
 */
export function useFocusOnMount<T extends HTMLElement>(vibrate?: Parameters<typeof buzz>[0]) {
  const ref = useRef<T>(null);
  useEffect(() => {
    ref.current?.focus({ preventScroll: true });
    if (vibrate !== undefined) buzz(vibrate);
  }, []);
  return ref;
}

export { pulses };
