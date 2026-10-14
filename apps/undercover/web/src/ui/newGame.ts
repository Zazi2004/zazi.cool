import { drawPair } from "../api/client";
import { createGame, type Game, type Settings } from "../game/engine";
import { addPlayed, loadPlayed, saveSetup } from "../game/storage";

/** Draws a new pair and deals it. Throws with a French message on failure. */
export async function newGame(settings: Settings, names: string[]): Promise<Game> {
  saveSetup({ settings, names });
  const pair = await drawPair({
    extreme: settings.extreme,
    suggestionsOnly: settings.suggestionsOnly,
    exclude: loadPlayed(),
  });
  addPlayed(pair.id);
  return createGame(settings, names, pair, Math.random);
}
