import { useEffect, useState } from "react";
import { ApiError, getStats } from "../api";
import { navigate } from "../router";
import { formatStreak, totalLine } from "../stats";
import { paceClass } from "./paceClass";
import type { HabitSummary } from "../types";
import { Ring } from "./Ring";
import { SpotlightCard } from "./effects/SpotlightCard";

type State =
  | { status: "loading" }
  | { status: "error"; message: string }
  | { status: "ready"; habits: HabitSummary[] };

/** The overview: one card per active habit, linking to its page. */
export function Stats() {
  const [state, setState] = useState<State>({ status: "loading" });

  useEffect(() => {
    let current = true;
    getStats().then(
      (stats) => current && setState({ status: "ready", habits: stats.habits }),
      (err: unknown) =>
        current &&
        setState({
          status: "error",
          message:
            err instanceof ApiError ? err.message : "Impossible de charger les statistiques.",
        }),
    );
    return () => {
      current = false;
    };
  }, []);

  return (
    <main className="page">
      <h1>Stats</h1>
      {state.status === "loading" && <p className="muted">Chargement…</p>}
      {state.status === "error" && (
        <p className="error" role="alert">
          {state.message}
        </p>
      )}
      {state.status === "ready" && state.habits.length === 0 && (
        <p className="muted">
          Aucune habitude pour l'instant : créez-en une dans l'onglet Habitudes.
        </p>
      )}
      {state.status === "ready" && state.habits.length > 0 && (
        <ul className="habit-list stats-list">
          {state.habits.map((s) => (
            <SummaryCard key={s.habit.id} summary={s} />
          ))}
        </ul>
      )}
    </main>
  );
}

function SummaryCard({ summary }: { summary: HabitSummary }) {
  const h = summary.habit;
  const href = `/stats/${h.id}`;
  return (
    <li className={`habit color-${h.color}`}>
      <SpotlightCard>
        <a
          className="habit-tap"
          href={href}
          onClick={(e) => {
            if (e.button !== 0 || e.metaKey || e.ctrlKey || e.shiftKey || e.altKey) return;
            e.preventDefault();
            navigate(href);
          }}
        >
          <span className="habit-emoji" aria-hidden="true">
            {h.emoji}
          </span>
          <span className="habit-text">
            <span className="habit-name">{h.name}</span>
            <span className="habit-rule muted">
              {totalLine(summary)}
              {summary.pace && (
                <>
                  {" · "}
                  <span className={paceClass(summary)}>{summary.pace.label}</span>
                </>
              )}
            </span>
            {summary.streak !== null && (
              <span className="habit-rule muted">
                <span aria-hidden="true">🔥</span> Série : {formatStreak(h.period, summary.streak)}
              </span>
            )}
          </span>
          <Ring size={44} stroke={5} progress={summary.progress} />
        </a>
      </SpotlightCard>
    </li>
  );
}
