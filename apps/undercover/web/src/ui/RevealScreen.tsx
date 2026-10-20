import { useState } from "preact/hooks";
import { continueGame, type Role } from "../game/engine";
import { Avatar, pulses, roleLabels, useFocusOnMount } from "./common";
import type { ScreenProps } from "./GamePage";

const blurbs: Record<Role, string> = {
  civil: "Un civil de moins… les imposteurs respirent.",
  undercover: "Un undercover démasqué !",
  mrwhite: "Mr. White est démasqué !",
};

function Verdict({ role }: { role: Role }) {
  const ref = useFocusOnMount<HTMLDivElement>(pulses.verdict);
  return (
    <div class="verdict" role="status" ref={ref} tabIndex={-1}>
      <p class={`role stamp role-${role}`}>{roleLabels[role]}</p>
      <p class="hint">{blurbs[role]}</p>
    </div>
  );
}

export function RevealScreen({ game, onChange }: ScreenProps) {
  const [shown, setShown] = useState(false);
  if (game.phase.kind !== "reveal") return null;
  const player = game.players[game.phase.eliminated]!;

  return (
    <section class="stack center">
      <p class="eyebrow">Tour {game.round} · Élimination</p>
      <Avatar name={player.name} tone={shown ? player.role : undefined} large />
      <h1>{player.name} était…</h1>
      {shown ? (
        <Verdict role={player.role} />
      ) : (
        <button type="button" class="button suspense" onClick={() => setShown(true)}>
          Révéler le rôle
        </button>
      )}
      {shown && (
        <button
          type="button"
          class="button primary"
          onClick={() => onChange(continueGame(game, Math.random))}
        >
          {game.phase.outcome ? "Voir les résultats" : "Tour suivant"}
        </button>
      )}
    </section>
  );
}
