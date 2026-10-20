import { useState } from "preact/hooks";
import { loadGame, loadHaptics, saveHaptics } from "../game/storage";
import { hapticsSupported } from "./haptics";
import { Link } from "./router";

export function HomePage() {
  const saved = loadGame();
  const inProgress = saved !== null && saved.phase.kind !== "over";
  const [haptics, setHaptics] = useState(loadHaptics);

  const toggleHaptics = (enabled: boolean) => {
    saveHaptics(enabled);
    setHaptics(enabled);
    // Feedback right away, so the player knows what to expect (and that it works).
    if (enabled) navigator.vibrate(40);
  };

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
      {hapticsSupported() && (
        <label class="checkbox">
          <input
            type="checkbox"
            checked={haptics}
            onChange={(e) => toggleHaptics(e.currentTarget.checked)}
          />
          <span>
            Vibrations
            <span class="hint">
              {" "}
              — une courte vibration au mot secret, à la révélation et à la fin.
            </span>
          </span>
        </label>
      )}
    </section>
  );
}
