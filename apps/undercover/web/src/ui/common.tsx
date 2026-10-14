import type { ComponentChildren } from "preact";
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
  if (player.word === null) {
    return (
      <div class="secret">
        <p class="secret-label">Tu es</p>
        <p class="secret-word">Mr. White</p>
        <p class="hint">Tu n'as pas de mot : écoute les indices et fonds-toi dans la masse.</p>
      </div>
    );
  }
  return (
    <div class="secret">
      <p class="secret-label">Ton mot</p>
      <p class="secret-word">{player.word}</p>
    </div>
  );
}
