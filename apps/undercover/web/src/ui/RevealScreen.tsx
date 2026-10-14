import { continueGame } from "../game/engine";
import { roleLabels } from "./common";
import type { ScreenProps } from "./GamePage";

export function RevealScreen({ game, onChange }: ScreenProps) {
  if (game.phase.kind !== "reveal") return null;
  const player = game.players[game.phase.eliminated]!;

  return (
    <section class="stack center">
      <p class="eyebrow">Tour {game.round} · Élimination</p>
      <h1>{player.name} était…</h1>
      <p class={`role role-${player.role}`}>{roleLabels[player.role]}</p>
      <button
        type="button"
        class="button primary"
        onClick={() => onChange(continueGame(game, Math.random))}
      >
        {game.phase.outcome ? "Voir les résultats" : "Tour suivant"}
      </button>
    </section>
  );
}
