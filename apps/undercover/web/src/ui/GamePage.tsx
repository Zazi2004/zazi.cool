import { useState } from "preact/hooks";
import type { Game } from "../game/engine";
import { loadGame, saveGame } from "../game/storage";
import { focusMain } from "./common";
import { DistributionScreen } from "./DistributionScreen";
import { GuessResultScreen, GuessScreen } from "./GuessScreens";
import { OverScreen } from "./OverScreen";
import { RevealScreen } from "./RevealScreen";
import { RoundScreen } from "./RoundScreen";
import { SetupScreen } from "./SetupScreen";

/** Props shared by every in-game screen. */
export interface ScreenProps {
  game: Game;
  onChange: (game: Game) => void;
}

export function GamePage() {
  const [game, setGame] = useState<Game | null>(loadGame);

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
    if (confirm("Abandonner la partie en cours ?")) update(null);
  };
  const props: ScreenProps = { game, onChange: update };

  return (
    <>
      <Screen {...props} onNewGame={() => update(null)} />
      {game.phase.kind !== "over" && (
        <button type="button" class="button ghost small" onClick={abandon}>
          Abandonner la partie
        </button>
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
    case "reveal":
      return <RevealScreen {...props} />;
    case "guess":
      return <GuessScreen {...props} />;
    case "guessResult":
      return <GuessResultScreen {...props} />;
    case "over":
      return <OverScreen {...props} />;
  }
}
