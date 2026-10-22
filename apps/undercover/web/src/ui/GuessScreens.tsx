import { useState } from "preact/hooks";
import { checkGuess, toUiError, type UiError } from "../api/client";
import { acceptGuess, continueGame, recordGuess } from "../game/engine";
import { Avatar, RetryableError } from "./common";
import type { ScreenProps } from "./GamePage";

/** An eliminated Mr. White types the word they think the civilians have. */
export function GuessScreen({ game, onChange }: ScreenProps) {
  const [guess, setGuess] = useState("");
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<UiError | null>(null);
  if (game.phase.kind !== "guess") return null;
  const player = game.players[game.phase.eliminated]!;

  const submit = async (e?: Event) => {
    e?.preventDefault();
    if (guess.trim() === "" || loading) return;
    setLoading(true);
    setError(null);
    try {
      const res = await checkGuess(game.pair.id, { civilSide: game.pair.civilSide, guess });
      onChange(recordGuess(game, guess.trim(), res.expected, res.correct));
    } catch (err) {
      setError(toUiError(err));
      setLoading(false);
    }
  };

  return (
    <form class="stack center" onSubmit={submit}>
      <p class="eyebrow">Tour {game.round} · Élimination</p>
      <Avatar name={player.name} tone="mrwhite" large />
      <h1>{player.name} était…</h1>
      <p class="role stamp role-mrwhite">Mr. White</p>
      <p>Dernière chance : quel est le mot des civils ?</p>
      <input
        type="text"
        value={guess}
        maxLength={40}
        placeholder="Le mot des civils"
        aria-label="Mot proposé par Mr. White"
        autocomplete="off"
        onInput={(e) => setGuess(e.currentTarget.value)}
      />
      {error && <RetryableError error={error} busy={loading} onRetry={submit} />}
      <button
        type="submit"
        class="button primary"
        disabled={guess.trim() === "" || loading}
        aria-busy={loading}
      >
        {loading ? "Vérification…" : "Valider"}
      </button>
    </form>
  );
}

export function GuessResultScreen({ game, onChange }: ScreenProps) {
  if (game.phase.kind !== "guessResult") return null;
  const { guess, expected, correct } = game.phase;
  const next = () => onChange(continueGame(game, Math.random));

  return (
    <section class="stack center">
      <h1 class={correct ? "verdict-win" : "verdict-miss"}>
        {correct ? "Mr. White a trouvé !" : "Raté !"}
      </h1>
      <dl class="card result">
        <dt>Réponse</dt>
        <dd>{guess}</dd>
        <dt>Mot des civils</dt>
        <dd>{expected}</dd>
      </dl>
      {correct ? (
        <button type="button" class="button primary" onClick={next}>
          Voir les résultats
        </button>
      ) : (
        <>
          <button type="button" class="button primary" onClick={next}>
            Continuer
          </button>
          <button type="button" class="button" onClick={() => onChange(acceptGuess(game))}>
            C'était bon : valider quand même
          </button>
        </>
      )}
    </section>
  );
}
