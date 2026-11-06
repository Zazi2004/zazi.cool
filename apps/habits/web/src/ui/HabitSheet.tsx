import { useEffect, useRef, useState, type FormEvent } from "react";
import { ApiError, archiveHabit, createHabit, updateHabit } from "../api";
import {
  changesRule,
  emptyForm,
  formatDayMonth,
  formFromHabit,
  hasWeekday,
  toggleWeekday,
  toInput,
  toPatch,
  weekdayNames,
  type HabitForm,
} from "../habits";
import {
  colorLabels,
  colors,
  momentLabels,
  moments,
  periodLabels,
  periods,
  type Habit,
  type Period,
} from "../types";

interface Props {
  /** The habit to edit; absent to create one. */
  habit?: Habit;
  /** Where a rule change would start, per period (from the server). */
  ruleStarts: Record<Period, string>;
  onClose: () => void;
  onSaved: (habit: Habit) => void;
  onArchived: (habit: Habit) => void;
}

/** Bottom sheet to create or edit a habit. A native <dialog> gives focus handling and Escape. */
export function HabitSheet({ habit, ruleStarts, onClose, onSaved, onArchived }: Props) {
  const dialog = useRef<HTMLDialogElement>(null);
  const [form, setForm] = useState<HabitForm>(habit ? formFromHabit(habit) : emptyForm);
  const [error, setError] = useState<string | null>(null);
  const [pending, setPending] = useState(false);

  useEffect(() => {
    const el = dialog.current;
    if (el && !el.open) el.showModal();
  }, []);

  function set<K extends keyof HabitForm>(key: K, value: HabitForm[K]) {
    setForm((f) => ({ ...f, [key]: value }));
  }

  async function submit(e: FormEvent) {
    e.preventDefault();
    setPending(true);
    setError(null);
    try {
      onSaved(habit ? await updateHabit(habit.id, patch) : await createHabit(toInput(form)));
    } catch (err) {
      fail(err);
    }
  }

  async function archive() {
    if (!habit) return;
    setPending(true);
    setError(null);
    try {
      onArchived(await archiveHabit(habit.id));
    } catch (err) {
      fail(err);
    }
  }

  function fail(err: unknown) {
    setError(err instanceof ApiError ? err.message : "Une erreur est survenue, réessayez.");
    setPending(false);
  }

  const quantity = form.kind === "quantity";
  const patch = habit ? toPatch(habit, form) : {};
  // A new version starts at the beginning of the period it is saved in, under the new period.
  const versionStart = habit && changesRule(patch) ? ruleStarts[form.period] : null;

  return (
    <dialog
      ref={dialog}
      className="sheet"
      aria-labelledby="sheet-title"
      onClose={onClose}
      onClick={(e) => e.target === dialog.current && dialog.current?.close()}
    >
      <form onSubmit={submit}>
        <header className="sheet-header">
          <h2 id="sheet-title">{habit ? "Modifier l'habitude" : "Nouvelle habitude"}</h2>
          <button type="button" className="ghost" onClick={() => dialog.current?.close()}>
            Fermer
          </button>
        </header>

        <div className="field-row">
          <div className="field emoji-field">
            <label htmlFor="habit-emoji">Emoji</label>
            <input
              id="habit-emoji"
              value={form.emoji}
              onChange={(e) => set("emoji", e.target.value)}
              required
            />
          </div>
          <div className="field grow">
            <label htmlFor="habit-name">Nom</label>
            <input
              id="habit-name"
              value={form.name}
              maxLength={60}
              autoFocus={!habit}
              required
              onChange={(e) => set("name", e.target.value)}
            />
          </div>
        </div>

        <fieldset className="field">
          <legend>Couleur</legend>
          <div className="swatches">
            {colors.map((c) => (
              <label key={c} className={`swatch color-${c}`}>
                <input
                  type="radio"
                  name="color"
                  value={c}
                  checked={form.color === c}
                  onChange={() => set("color", c)}
                />
                <span className="visually-hidden">{colorLabels[c]}</span>
              </label>
            ))}
          </div>
        </fieldset>

        <Choice
          legend="Moment"
          name="moment"
          value={form.moment}
          options={moments.map((m) => ({ value: m, label: momentLabels[m] }))}
          onChange={(v) => set("moment", v)}
        />

        <Choice
          legend="Type"
          name="kind"
          value={form.kind}
          options={[
            { value: "binary", label: "Oui / non" },
            { value: "quantity", label: "Quantité" },
          ]}
          onChange={(v) => set("kind", v)}
        />

        {quantity && (
          <div className="field-row">
            <div className="field grow">
              <label htmlFor="habit-target">Cible</label>
              <input
                id="habit-target"
                type="number"
                inputMode="numeric"
                min={1}
                max={10000}
                required
                value={form.target}
                onChange={(e) => set("target", e.target.value)}
              />
            </div>
            <div className="field grow">
              <label htmlFor="habit-unit">Unité</label>
              <input
                id="habit-unit"
                value={form.unit}
                maxLength={20}
                placeholder="pilules"
                onChange={(e) => set("unit", e.target.value)}
              />
            </div>
            <div className="field grow">
              <label htmlFor="habit-step">Pas</label>
              <input
                id="habit-step"
                type="number"
                inputMode="numeric"
                min={1}
                max={10000}
                required
                value={form.step}
                onChange={(e) => set("step", e.target.value)}
              />
            </div>
          </div>
        )}

        <Choice
          legend="Période"
          name="period"
          value={form.period}
          options={periods.map((p) => ({ value: p, label: periodLabels[p] }))}
          onChange={(v) => set("period", v)}
        />

        <fieldset className="field">
          <legend>Jours autorisés</legend>
          <div className="weekdays">
            {weekdayNames.map((d, i) => (
              <button
                key={d.long}
                type="button"
                className="weekday"
                aria-label={d.long}
                aria-pressed={hasWeekday(form.weekdays, i)}
                onClick={() => set("weekdays", toggleWeekday(form.weekdays, i))}
              >
                {d.short}
              </button>
            ))}
          </div>
        </fieldset>

        {versionStart && (
          <p className="rule-note" role="status">
            S'applique à partir du {formatDayMonth(versionStart)} : les périodes passées gardent
            leur règle.
          </p>
        )}
        {error && (
          <p className="error" role="alert">
            {error}
          </p>
        )}
        <button type="submit" className="primary" disabled={pending}>
          {habit ? "Enregistrer" : "Créer l'habitude"}
        </button>
        {habit && (
          <button type="button" className="ghost danger" disabled={pending} onClick={archive}>
            Archiver
          </button>
        )}
      </form>
    </dialog>
  );
}

interface ChoiceProps<T extends string> {
  legend: string;
  name: string;
  value: T;
  options: { value: T; label: string }[];
  onChange: (value: T) => void;
}

/** A segmented control built on radio buttons (keyboard and screen reader friendly). */
function Choice<T extends string>({ legend, name, value, options, onChange }: ChoiceProps<T>) {
  return (
    <fieldset className="field">
      <legend>{legend}</legend>
      <div className="segmented">
        {options.map((o) => (
          <label key={o.value}>
            <input
              type="radio"
              name={name}
              value={o.value}
              checked={value === o.value}
              onChange={() => onChange(o.value)}
            />
            <span>{o.label}</span>
          </label>
        ))}
      </div>
    </fieldset>
  );
}
