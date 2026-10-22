import { useState } from "preact/hooks";
import {
  civilCount,
  type ExtremeMode,
  type Game,
  MAX_NAME_LENGTH,
  MAX_PLAYERS,
  MIN_PLAYERS,
  type Settings,
  validateNames,
  validateSettings,
} from "../game/engine";
import { loadSetup } from "../game/storage";
import { toUiError, type UiError } from "../api/client";
import { ErrorText, RetryableError, Stepper } from "./common";
import { newGame } from "./newGame";
import { Link } from "./router";

const EXTREME_CHOICES: [ExtremeMode, string][] = [
  ["off", "Tout public"],
  ["on", "Tout public et extrême"],
  ["only", "Extrême uniquement"],
];

/** Keeps the names already typed and pads or trims the list to `count`. */
function resize(names: string[], count: number): string[] {
  return Array.from({ length: count }, (_, i) => names[i] ?? "");
}

export function SetupScreen({ onStart }: { onStart: (game: Game) => void }) {
  const [initial] = useState(loadSetup);
  const [settings, setSettings] = useState<Settings>(initial.settings);
  const [names, setNames] = useState(() => resize(initial.names, initial.settings.playerCount));
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<UiError | null>(null);

  const set = <K extends keyof Settings>(key: K, value: Settings[K]) => {
    const next = { ...settings, [key]: value };
    setSettings(next);
    if (key === "playerCount") setNames(resize(names, next.playerCount));
  };

  const problems = [...validateSettings(settings), ...validateNames(names)];
  const impostorMax = settings.playerCount; // the validation explains the real limit

  const start = async (e?: Event) => {
    e?.preventDefault();
    if (problems.length > 0 || loading) return;
    setLoading(true);
    setError(null);
    try {
      onStart(await newGame(settings, names));
    } catch (err) {
      setError(toUiError(err));
      setLoading(false);
    }
  };

  return (
    <form class="stack" onSubmit={start}>
      <h1>Nouvelle partie</h1>

      <fieldset class="card stack">
        <legend>Joueurs</legend>
        <Stepper
          label="Nombre de joueurs"
          value={settings.playerCount}
          min={MIN_PLAYERS}
          max={MAX_PLAYERS}
          onChange={(n) => set("playerCount", n)}
        />
        <p class="hint">Saisissez les prénoms dans le sens de la table.</p>
        <ol class="names">
          {names.map((name, i) => (
            <li key={i}>
              <input
                type="text"
                value={name}
                maxLength={MAX_NAME_LENGTH}
                placeholder={`Joueur ${i + 1}`}
                aria-label={`Prénom du joueur ${i + 1}`}
                autocomplete="off"
                onInput={(e) =>
                  setNames(names.map((n, j) => (j === i ? e.currentTarget.value : n)))
                }
              />
            </li>
          ))}
        </ol>
      </fieldset>

      <fieldset class="card stack">
        <legend>Rôles</legend>
        <Stepper
          label="Undercovers"
          value={settings.undercoverCount}
          min={0}
          max={impostorMax}
          onChange={(n) => set("undercoverCount", n)}
        />
        <Stepper
          label="Mr. White"
          value={settings.mrWhiteCount}
          min={0}
          max={impostorMax}
          onChange={(n) => set("mrWhiteCount", n)}
        />
        <p class="hint">Civils : {Math.max(civilCount(settings), 0)}</p>
        <Checkbox
          label="Mr. White ne peut pas commencer un tour"
          checked={settings.mrWhiteCannotStart}
          onChange={(v) => set("mrWhiteCannotStart", v)}
        />
        <Checkbox
          label="Un undercover ne peut pas commencer un tour"
          checked={settings.undercoverCannotStart}
          onChange={(v) => set("undercoverCannotStart", v)}
        />
      </fieldset>

      <fieldset class="card stack">
        <legend>Mots</legend>
        {EXTREME_CHOICES.map(([mode, label]) => (
          <label key={mode} class="checkbox">
            <input
              type="radio"
              name="extreme"
              checked={settings.extreme === mode}
              onChange={() => set("extreme", mode)}
            />
            <span>{label}</span>
          </label>
        ))}
        <p class="hint">Extrême : paires limites, réservé aux adultes.</p>
        <Checkbox
          label="Uniquement les paires proposées par les joueurs"
          checked={settings.suggestionsOnly}
          onChange={(v) => set("suggestionsOnly", v)}
        />
      </fieldset>

      {problems.map((p) => (
        <ErrorText key={p}>{p}</ErrorText>
      ))}
      {error && (
        <RetryableError error={error} busy={loading} onRetry={start}>
          {!error.retryable && <Link href="/proposer">Proposer une paire</Link>}
        </RetryableError>
      )}

      <button
        type="submit"
        class="button primary"
        disabled={problems.length > 0 || loading}
        aria-busy={loading}
      >
        {loading ? "Tirage des mots…" : "Distribuer les mots"}
      </button>
    </form>
  );
}

interface CheckboxProps {
  label: string;
  checked: boolean;
  onChange: (checked: boolean) => void;
}

function Checkbox({ label, checked, onChange }: CheckboxProps) {
  return (
    <label class="checkbox">
      <input
        type="checkbox"
        checked={checked}
        onChange={(e) => onChange(e.currentTarget.checked)}
      />
      <span>{label}</span>
    </label>
  );
}
