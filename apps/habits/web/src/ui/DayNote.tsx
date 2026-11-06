import { useEffect, useRef, useState } from "react";
import { ApiError, putNote } from "../api";
import { noteLength, noteLimit } from "../today";

const debounceMs = 800;

type Saved = "idle" | "saving" | "saved" | "error";

/**
 * The note of a day, saved automatically ~800 ms after the last keystroke. `initial` is
 * only read on mount: the component is keyed by date, so a server answer never
 * overwrites what is being typed. Unsaved text is sent when leaving the day.
 */
export function DayNote({
  date,
  initial,
  editable,
}: {
  date: string;
  initial: string;
  editable: boolean;
}) {
  const [text, setText] = useState(initial);
  const [saved, setSaved] = useState<Saved>("idle");
  const [error, setError] = useState("");
  const timer = useRef<number | undefined>(undefined);
  const pending = useRef<string | null>(null);

  async function save(value: string) {
    pending.current = null;
    setSaved("saving");
    try {
      await putNote(date, value);
      setSaved("saved");
      setError("");
    } catch (err) {
      setSaved("error");
      setError(err instanceof ApiError ? err.message : "La note n'a pas été enregistrée.");
    }
  }

  // Leaving the day (swipe, tab) must not lose a note typed less than 800 ms ago.
  useEffect(
    () => () => {
      window.clearTimeout(timer.current);
      if (pending.current !== null) void putNote(date, pending.current).catch(() => undefined);
    },
    [date],
  );

  function change(value: string) {
    setText(value);
    window.clearTimeout(timer.current);
    if (noteLength(value) > noteLimit) {
      pending.current = null;
      setSaved("error");
      setError(`La note ne peut pas dépasser ${noteLimit} caractères.`);
      return;
    }
    setError("");
    setSaved("idle");
    pending.current = value;
    timer.current = window.setTimeout(() => void save(value), debounceMs);
  }

  const length = noteLength(text);
  return (
    <section className="note" aria-labelledby="note-title">
      <h2 id="note-title">Note du jour</h2>
      <textarea
        value={text}
        rows={4}
        readOnly={!editable}
        placeholder={editable ? "Comment s'est passée la journée ?" : ""}
        aria-describedby="note-count"
        onChange={(e) => change(e.target.value)}
      />
      <p id="note-count" className="muted note-meta">
        <span aria-live="polite">
          {saved === "saving" && "Enregistrement…"}
          {saved === "saved" && "Enregistré"}
        </span>
        <span className={length > noteLimit ? "error" : undefined}>
          {length} / {noteLimit}
        </span>
      </p>
      {error && (
        <p className="error" role="alert">
          {error}
        </p>
      )}
    </section>
  );
}
