import { useState } from "preact/hooks";
import type { Outcome, Player } from "../game/engine";
import { ErrorText, roleLabels } from "./common";
import type { ScreenProps } from "./GamePage";
import { newGame } from "./newGame";
import { ReportButtons } from "./ReportButtons";
import { Link } from "./router";

function title(outcome: Outcome, players: Player[]): string {
  switch (outcome.winner) {
    case "civils":
      return "Les civils gagnent !";
    case "impostors":
      return "Les imposteurs gagnent !";
    case "mrwhite":
      return `${players[outcome.mrWhite ?? -1]?.name ?? "Mr. White"} (Mr. White) gagne !`;
  }
}

export function OverScreen({ game, onChange, onNewGame }: ScreenProps & { onNewGame: () => void }) {
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  if (game.phase.kind !== "over") return null;
  const { outcome } = game.phase;

  const replay = async () => {
    setLoading(true);
    setError(null);
    try {
      onChange(
        await newGame(
          game.settings,
          game.players.map((p) => p.name),
        ),
      );
    } catch (err) {
      setError(err instanceof Error ? err.message : "Erreur inattendue.");
      setLoading(false);
    }
  };

  return (
    <section class="stack">
      <h1 class="center">{title(outcome, game.players)}</h1>

      <dl class="card result">
        <dt>Mot des civils</dt>
        <dd>{game.pair.civilWord}</dd>
        <dt>Mot des undercovers</dt>
        <dd>{game.pair.undercoverWord}</dd>
      </dl>

      <ul class="card players">
        {game.players.map((p) => (
          <li key={p.name} class={p.alive ? "" : "eliminated"}>
            <span class="player-name">{p.name}</span>
            <span class={`tag role-${p.role}`}>{roleLabels[p.role]}</span>
          </li>
        ))}
      </ul>

      {error && <ErrorText>{error}</ErrorText>}
      <button type="button" class="button primary" disabled={loading} onClick={replay}>
        {loading ? "Tirage des mots…" : "Rejouer avec les mêmes joueurs"}
      </button>
      <button type="button" class="button" onClick={onNewGame}>
        Changer les réglages
      </button>

      <details class="card">
        <summary>Cette paire pose problème ?</summary>
        <ReportButtons pairId={game.pair.id} extreme={game.pair.extreme} />
      </details>

      <Link href="/" class="button ghost">
        Accueil
      </Link>
    </section>
  );
}
