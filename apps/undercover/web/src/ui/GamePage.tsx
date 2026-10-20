import { useState } from "preact/hooks";
import type { Game } from "../game/engine";
import { loadGame, saveGame } from "../game/storage";
import { focusMain } from "./common";
import { DistributionScreen } from "./DistributionScreen";
import { GuessResultScreen, GuessScreen } from "./GuessScreens";
import { Modal } from "./Modal";
import { OverScreen } from "./OverScreen";
import { RoundScreen } from "./RoundScreen";
import { SetupScreen } from "./SetupScreen";

/** Props shared by every in-game screen. */
export interface ScreenProps {
  game: Game;
  onChange: (game: Game) => void;
}

export function GamePage() {
  const [game, setGame] = useState<Game | null>(loadGame);
  const [confirming, setConfirming] = useState(false);

  const update = (next: Game | null) => {
    saveGame(next);
    setGame(next);
    window.scrollTo(0, 0);
    focusMain();
  };

  if (!game) {
    return <SetupScreen onStart={update} />;
  }

  const abandon = () => {
    setConfirming(false);
    update(null);
  };
  const props: ScreenProps = { game, onChange: update };

  return (
    <>
      <Screen {...props} onNewGame={() => update(null)} />
      {game.phase.kind !== "over" && (
        <button type="button" class="button ghost small" onClick={() => setConfirming(true)}>
          Abandonner la partie
        </button>
      )}
      {confirming && (
        <Modal title="Abandonner la partie ?" onClose={() => setConfirming(false)}>
          <p>La partie en cours sera perdue, les mots et les rôles aussi.</p>
          <div class="modal-actions">
            <button
              type="button"
              class="button"
              data-autofocus
              onClick={() => setConfirming(false)}
            >
              Continuer la partie
            </button>
            <button type="button" class="button danger" onClick={abandon}>
              Abandonner
            </button>
          </div>
        </Modal>
      )}
    </>
  );
}

function Screen(props: ScreenProps & { onNewGame: () => void }) {
  const { phase } = props.game;
  // The key resets each screen's local state when the phase changes.
  switch (phase.kind) {
    case "distribution":
      return <DistributionScreen key={`d${phase.next}`} {...props} />;
    case "round":
      return <RoundScreen key={`r${props.game.round}`} {...props} />;
    case "guess":
      return <GuessScreen {...props} />;
    case "guessResult":
      return <GuessResultScreen {...props} />;
    case "over":
      return <OverScreen {...props} />;
  }
}
