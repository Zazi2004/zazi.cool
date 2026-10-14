import { useState } from "preact/hooks";
import { wordSeen } from "../game/engine";
import { SecretCard } from "./common";
import type { ScreenProps } from "./GamePage";

export function DistributionScreen({ game, onChange }: ScreenProps) {
  const [revealed, setRevealed] = useState(false);
  if (game.phase.kind !== "distribution") return null;

  const index = game.phase.next;
  const player = game.players[index]!;

  return (
    <section class="stack center">
      <p class="eyebrow">
        Distribution des mots · {index + 1} / {game.players.length}
      </p>
      {revealed ? (
        <>
          <SecretCard player={player} />
          <button
            type="button"
            class="button primary"
            onClick={() => onChange(wordSeen(game, Math.random))}
          >
            C'est retenu, cacher
          </button>
        </>
      ) : (
        <>
          <h1>
            Passe le téléphone à <strong>{player.name}</strong>
          </h1>
          <p class="hint">Les autres ne regardent pas l'écran.</p>
          <button type="button" class="button primary" onClick={() => setRevealed(true)}>
            Je suis {player.name}, voir mon mot
          </button>
        </>
      )}
    </section>
  );
}
