import { useCallback, useEffect, useRef, useState } from "react";
import { ApiError, getDay, putEntry } from "../api";
import { localDate, subtitle, tapValue, withValue } from "../today";
import { momentLabels, type Day, type DayHabit } from "../types";

type State =
  { status: "loading" } | { status: "error"; message: string } | { status: "ready"; day: Day };

const fallback = "Impossible de charger la journée.";

export function Today() {
  const [date] = useState(() => localDate(new Date()));
  const [state, setState] = useState<State>({ status: "loading" });
  const [notice, setNotice] = useState("");
  // Only the answer of the latest tap replaces the day: an older, slower one is dropped.
  const latest = useRef(0);

  useEffect(() => {
    let current = true;
    getDay(date).then(
      (day) => current && setState({ status: "ready", day }),
      (err: unknown) =>
        current &&
        setState({ status: "error", message: err instanceof ApiError ? err.message : fallback }),
    );
    return () => {
      current = false;
    };
  }, [date]);

  const tap = useCallback(
    async (item: DayHabit) => {
      if (state.status !== "ready") return;
      const before = state.day;
      const value = tapValue(item);
      const ticket = ++latest.current;
      setNotice("");
      setState({ status: "ready", day: withValue(before, item.habit.id, value) });
      try {
        const day = await putEntry(before.date, item.habit.id, value);
        if (ticket === latest.current) setState({ status: "ready", day });
      } catch (err) {
        if (ticket !== latest.current) return;
        setState({ status: "ready", day: before });
        setNotice(err instanceof ApiError ? err.message : "La saisie n'a pas été enregistrée.");
      }
    },
    [state],
  );

  return (
    <main className="page">
      <h1>Aujourd'hui</h1>
      {state.status === "loading" && <p className="muted">Chargement…</p>}
      {state.status === "error" && (
        <p className="error" role="alert">
          {state.message}
        </p>
      )}
      {state.status === "ready" && <DayView day={state.day} onTap={tap} />}
      <p className="toast" role="alert" hidden={notice === ""}>
        {notice}
      </p>
    </main>
  );
}

function DayView({ day, onTap }: { day: Day; onTap: (item: DayHabit) => void }) {
  return (
    <>
      <Ring progress={day.progress} done={day.done} total={day.total} />
      {day.groups.length === 0 && (
        <p className="muted">Aucune habitude aujourd'hui. Créez-en une dans l'onglet Habitudes.</p>
      )}
      {day.groups.map((group) => (
        <section key={group.moment} className="group" aria-labelledby={`today-${group.moment}`}>
          <h2 id={`today-${group.moment}`}>{momentLabels[group.moment]}</h2>
          <ul className="habit-list">
            {group.habits.map((item) => (
              <Row key={item.habit.id} item={item} onTap={onTap} />
            ))}
          </ul>
        </section>
      ))}
    </>
  );
}

function Row({ item, onTap }: { item: DayHabit; onTap: (item: DayHabit) => void }) {
  const h = item.habit;
  return (
    <li className={`habit color-${h.color}${item.complete ? " is-done" : ""}`}>
      <button
        type="button"
        className="habit-tap"
        aria-pressed={item.complete}
        onClick={() => onTap(item)}
      >
        <span className="habit-emoji" aria-hidden="true">
          {h.emoji}
        </span>
        <span className="habit-text">
          <span className="habit-name">{h.name}</span>
          <span className="habit-rule muted">{subtitle(item)}</span>
        </span>
        <MiniRing progress={item.progress} />
      </button>
    </li>
  );
}

const bigRadius = 64;
const bigLength = 2 * Math.PI * bigRadius;

function Ring({ progress, done, total }: { progress: number; done: number; total: number }) {
  const percent = Math.round(progress * 100);
  return (
    <div
      className="ring"
      role="img"
      aria-label={`${percent} % de la journée, ${done} sur ${total}`}
    >
      <svg viewBox="0 0 160 160" aria-hidden="true">
        <defs>
          <linearGradient id="ring-gradient" x1="0" y1="0" x2="1" y2="1">
            <stop offset="0%" stopColor="var(--ring-from)" />
            <stop offset="100%" stopColor="var(--ring-to)" />
          </linearGradient>
        </defs>
        <circle className="ring-track" cx="80" cy="80" r={bigRadius} />
        <circle
          className="ring-value"
          cx="80"
          cy="80"
          r={bigRadius}
          stroke="url(#ring-gradient)"
          strokeDasharray={bigLength}
          strokeDashoffset={bigLength * (1 - progress)}
        />
      </svg>
      <div className="ring-label" aria-hidden="true">
        <strong>{percent} %</strong>
        <span>
          {done} / {total} du jour
        </span>
      </div>
    </div>
  );
}

const miniRadius = 14;
const miniLength = 2 * Math.PI * miniRadius;

function MiniRing({ progress }: { progress: number }) {
  return (
    <svg className="mini-ring" viewBox="0 0 36 36" aria-hidden="true">
      <circle className="mini-track" cx="18" cy="18" r={miniRadius} />
      <circle
        className="mini-value"
        cx="18"
        cy="18"
        r={miniRadius}
        strokeDasharray={miniLength}
        strokeDashoffset={miniLength * (1 - progress)}
      />
    </svg>
  );
}
