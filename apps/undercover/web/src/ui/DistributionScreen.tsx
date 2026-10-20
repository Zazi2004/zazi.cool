import { useCallback, useState } from "preact/hooks";
import { wordSeen } from "../game/engine";
import { Avatar, SecretCard, useHideOnLeave } from "./common";
import type { ScreenProps } from "./GamePage";

export function DistributionScreen({ game, onChange }: ScreenProps) {
  const [revealed, setRevealed] = useState(false);
  const hide = useCallback(() => setRevealed(false), []);
  useHideOnLeave(hide);
  if (game.phase.kind !== "distribution") return null;

  const index = game.phase.next;
  const player = game.players[index]!;

  return (
    <section class="stack center">
      <p class="eyebrow">
        Distribution des mots · {index + 1} / {game.players.length}
      </p>
      <div class="progress" aria-hidden="true">
        {game.players.map((p, i) => (
          <span key={p.name} class={i < index ? "done" : i === index ? "current" : ""} />
        ))}
      </div>
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
          <Avatar name={player.name} large />
          <h1>
            Passe le téléphone à <strong>{player.name}</strong>
          </h1>
          <p class="hint">Les autres ne regardent pas l'écran.</p>
          <button
            type="button"
            class="secret-cover"
            aria-label={`Je suis ${player.name}, voir mon mot`}
            onClick={() => setRevealed(true)}
          >
            <span class="secret-cover-mark" aria-hidden="true">
              ?
            </span>
            <span>Je suis {player.name}</span>
            <span class="secret-cover-hint">Touche pour retourner la carte</span>
          </button>
        </>
      )}
    </section>
  );
}
