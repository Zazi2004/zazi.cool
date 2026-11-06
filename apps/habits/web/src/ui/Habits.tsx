import { useEffect, useState } from "react";
import { ApiError, listHabits } from "../api";
import { describeRule, groupByMoment, withHabit } from "../habits";
import { momentLabels, type Habit } from "../types";
import { HabitSheet } from "./HabitSheet";

type State =
  | { status: "loading" }
  | { status: "error"; message: string }
  | { status: "ready"; active: Habit[]; archived: Habit[] };

export function Habits() {
  const [state, setState] = useState<State>({ status: "loading" });
  const [creating, setCreating] = useState(false);

  useEffect(() => {
    let current = true;
    listHabits().then(
      ({ active, archived }) => current && setState({ status: "ready", active, archived }),
      (err: unknown) =>
        current &&
        setState({
          status: "error",
          message: err instanceof ApiError ? err.message : "Impossible de charger les habitudes.",
        }),
    );
    return () => {
      current = false;
    };
  }, []);

  function created(habit: Habit) {
    setState((s) => (s.status === "ready" ? { ...s, active: withHabit(s.active, habit) } : s));
    setCreating(false);
  }

  return (
    <main className="page">
      <h1>Habitudes</h1>
      {state.status === "loading" && <p className="muted">Chargement…</p>}
      {state.status === "error" && (
        <p className="error" role="alert">
          {state.message}
        </p>
      )}
      {state.status === "ready" && <HabitList habits={state.active} />}

      <button
        type="button"
        className="fab"
        aria-label="Nouvelle habitude"
        onClick={() => setCreating(true)}
      >
        +
      </button>
      {creating && <HabitSheet onClose={() => setCreating(false)} onCreated={created} />}
    </main>
  );
}

function HabitList({ habits }: { habits: Habit[] }) {
  if (habits.length === 0) {
    return (
      <p className="muted">Aucune habitude pour l'instant. Touchez « + » pour en créer une.</p>
    );
  }
  return (
    <>
      {groupByMoment(habits).map((group) => (
        <section key={group.moment} className="group" aria-labelledby={`group-${group.moment}`}>
          <h2 id={`group-${group.moment}`}>{momentLabels[group.moment]}</h2>
          <ul className="habit-list">
            {group.habits.map((h) => (
              <li key={h.id} className={`habit color-${h.color}`}>
                <span className="habit-emoji" aria-hidden="true">
                  {h.emoji}
                </span>
                <span className="habit-text">
                  <span className="habit-name">{h.name}</span>
                  <span className="habit-rule muted">{describeRule(h)}</span>
                </span>
              </li>
            ))}
          </ul>
        </section>
      ))}
    </>
  );
}
