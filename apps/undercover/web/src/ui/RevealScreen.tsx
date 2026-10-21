import { continueGame, type Role } from "../game/engine";
import { Avatar, roleLabels, useFocusOnMount } from "./common";
import type { ScreenProps } from "./GamePage";

function Verdict({ role }: { role: Role }) {
  const ref = useFocusOnMount<HTMLDivElement>();
  return (
    <div class="verdict" role="status" ref={ref} tabIndex={-1}>
      <p class={`role stamp role-${role}`}>{roleLabels[role]}</p>
    </div>
  );
}

/**
 * Announces the role of an eliminated civil or undercover. Never their word: it would
 * give away the civilians' word before the end of the game.
 */
export function RevealScreen({ game, onChange }: ScreenProps) {
  if (game.phase.kind !== "reveal") return null;
  const player = game.players[game.phase.eliminated]!;

  return (
    <section class="stack center">
      <p class="eyebrow">Tour {game.round} · Élimination</p>
      <Avatar name={player.name} tone={player.role} large />
      <h1>{player.name} était…</h1>
      <Verdict role={player.role} />
      <button
        type="button"
        class="button primary"
        onClick={() => onChange(continueGame(game, Math.random))}
      >
        Continuer
      </button>
    </section>
  );
}
