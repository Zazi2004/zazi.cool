import { useEffect, useRef, useState, type FormEvent, type PointerEvent } from "react";
import { maxValue, parseValue } from "../today";
import type { DayHabit } from "../types";

interface Props {
  item: DayHabit;
  onSave: (value: number) => void;
  onClose: () => void;
}

const closeDistance = 80;

/** Bottom sheet to enter the exact value of a habit; 0 removes the entry. */
export function EntrySheet({ item, onSave, onClose }: Props) {
  const dialog = useRef<HTMLDialogElement>(null);
  const drag = useRef<{ y: number; dy: number } | null>(null);
  const [text, setText] = useState(String(item.value));
  const h = item.habit;
  const value = parseValue(text);

  useEffect(() => {
    const el = dialog.current;
    if (el && !el.open) el.showModal();
  }, []);

  function step(delta: number) {
    setText(String(Math.min(maxValue, Math.max(0, (value ?? 0) + delta))));
  }

  function submit(e: FormEvent) {
    e.preventDefault();
    if (value === null) return;
    onSave(value);
    dialog.current?.close();
  }

  function setOffset(dy: number) {
    dialog.current?.style.setProperty("transform", dy > 0 ? `translateY(${dy}px)` : "");
  }

  function dragStart(e: PointerEvent) {
    drag.current = { y: e.clientY, dy: 0 };
    e.currentTarget.setPointerCapture(e.pointerId);
  }

  function dragMove(e: PointerEvent) {
    if (!drag.current) return;
    drag.current.dy = e.clientY - drag.current.y;
    setOffset(drag.current.dy);
  }

  function dragEnd() {
    const dy = drag.current?.dy ?? 0;
    drag.current = null;
    setOffset(0);
    if (dy > closeDistance) dialog.current?.close();
  }

  return (
    <dialog
      ref={dialog}
      className={`sheet color-${h.color}`}
      aria-labelledby="entry-title"
      onClose={onClose}
      onClick={(e) => e.target === dialog.current && dialog.current?.close()}
    >
      <div
        className="sheet-grip"
        aria-hidden="true"
        onPointerDown={dragStart}
        onPointerMove={dragMove}
        onPointerUp={dragEnd}
        onPointerCancel={dragEnd}
      />
      <form onSubmit={submit}>
        <header className="sheet-header">
          <h2 id="entry-title">
            <span aria-hidden="true">{h.emoji}</span> {h.name}
          </h2>
          <button type="button" className="ghost" onClick={() => dialog.current?.close()}>
            Fermer
          </button>
        </header>
        <div className="stepper">
          <button
            type="button"
            aria-label="Moins"
            disabled={(value ?? 0) <= 0}
            onClick={() => step(-(h.step || 1))}
          >
            −
          </button>
          <div className="field">
            <label htmlFor="entry-value">Valeur{h.unit ? ` (${h.unit})` : ""}</label>
            <input
              id="entry-value"
              inputMode="numeric"
              autoComplete="off"
              value={text}
              onChange={(e) => setText(e.target.value)}
              aria-invalid={value === null}
            />
          </div>
          <button type="button" aria-label="Plus" onClick={() => step(h.step || 1)}>
            +
          </button>
        </div>
        <p className="muted">
          Objectif : {h.target}
          {h.unit ? ` ${h.unit}` : ""}. 0 supprime la saisie.
        </p>
        {value === null && (
          <p className="error" role="alert">
            Saisissez un nombre entier entre 0 et 100 000.
          </p>
        )}
        <button type="submit" disabled={value === null}>
          Enregistrer
        </button>
      </form>
    </dialog>
  );
}
