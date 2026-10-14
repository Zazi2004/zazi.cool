import { useState } from "preact/hooks";
import { eliminate, skipElimination } from "../game/engine";
import { SecretCard } from "./common";
import type { ScreenProps } from "./GamePage";

/** What the round screen shows: the speaking order, the vote, or a word check. */
type View =
  | { kind: "order" }
  | { kind: "vote"; selected: number | null }
  | { kind: "peekPick" }
  | { kind: "peekConfirm"; player: number }
  | { kind: "peekShow"; player: number };

export function RoundScreen({ game, onChange }: ScreenProps) {
  const [view, setView] = useState<View>({ kind: "order" });
  if (game.phase.kind !== "round") return null;

  const { order } = game.phase;
  const alive = game.players.flatMap((p, i) => (p.alive ? [i] : []));
  const back = () => setView({ kind: "order" });

  switch (view.kind) {
    case "order":
      return (
        <section class="stack">
          <p class="eyebrow">Tour {game.round}</p>
          <h1>Ordre de passage</h1>
          <p class="hint">Chacun donne un indice sur son mot, à l'oral, dans cet ordre.</p>
          <ol class="order card">
            {order.map((i) => (
              <li key={i}>{game.players[i]!.name}</li>
            ))}
          </ol>
          <button
            type="button"
            class="button primary"
            onClick={() => setView({ kind: "vote", selected: null })}
          >
            Passer au vote
          </button>
          <button type="button" class="button" onClick={() => setView({ kind: "peekPick" })}>
            Revoir mon mot
          </button>
        </section>
      );

    case "vote": {
      const { selected } = view;
      return (
        <section class="stack">
          <p class="eyebrow">Tour {game.round} · Vote</p>
          <h1>Qui est éliminé ?</h1>
          <p class="hint">Votez à l'oral, puis désignez le joueur éliminé.</p>
          <div class="choices" role="radiogroup" aria-label="Joueur éliminé">
            {alive.map((i) => (
              <button
                key={i}
                type="button"
                role="radio"
                aria-checked={selected === i}
                class={selected === i ? "choice selected" : "choice"}
                onClick={() => setView({ kind: "vote", selected: i })}
              >
                {game.players[i]!.name}
              </button>
            ))}
          </div>
          <button
            type="button"
            class="button danger"
            disabled={selected === null}
            onClick={() => selected !== null && onChange(eliminate(game, selected))}
          >
            {selected === null
              ? "Choisissez un joueur"
              : `Éliminer ${game.players[selected]!.name}`}
          </button>
          <button
            type="button"
            class="button"
            onClick={() => onChange(skipElimination(game, Math.random))}
          >
            Personne n'est éliminé
          </button>
          <button type="button" class="button ghost" onClick={back}>
            Retour à l'ordre de passage
          </button>
        </section>
      );
    }

    case "peekPick":
      return (
        <section class="stack">
          <h1>Qui veut revoir son mot ?</h1>
          <div class="choices">
            {alive.map((i) => (
              <button
                key={i}
                type="button"
                class="choice"
                onClick={() => setView({ kind: "peekConfirm", player: i })}
              >
                {game.players[i]!.name}
              </button>
            ))}
          </div>
          <button type="button" class="button ghost" onClick={back}>
            Retour
          </button>
        </section>
      );

    case "peekConfirm": {
      const name = game.players[view.player]!.name;
      return (
        <section class="stack center">
          <h1>
            Seul·e <strong>{name}</strong> doit regarder
          </h1>
          <button
            type="button"
            class="button primary"
            onClick={() => setView({ kind: "peekShow", player: view.player })}
          >
            Je suis {name}, voir mon mot
          </button>
          <button type="button" class="button ghost" onClick={back}>
            Annuler
          </button>
        </section>
      );
    }

    case "peekShow":
      return (
        <section class="stack center">
          <SecretCard player={game.players[view.player]!} />
          <button type="button" class="button primary" onClick={back}>
            Cacher
          </button>
        </section>
      );
  }
}
