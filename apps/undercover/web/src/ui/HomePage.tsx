import { loadGame } from "../game/storage";
import { Link } from "./router";

export function HomePage() {
  const saved = loadGame();
  const inProgress = saved !== null && saved.phase.kind !== "over";

  return (
    <section class="stack">
      <div class="hero">
        <h1>Undercover</h1>
        <p>
          Tout le monde a le même mot… sauf les imposteurs. Un seul téléphone, qui passe de main en
          main.
        </p>
      </div>
      <Link href="/partie" class="button primary">
        {inProgress ? "Reprendre la partie" : "Nouvelle partie"}
      </Link>
      <Link href="/proposer" class="button">
        Proposer une paire de mots
      </Link>
      <Link href="/regles" class="button ghost">
        Règles du jeu
      </Link>
    </section>
  );
}
