import { useEffect, useRef, useState, type PointerEvent } from "react";
import { ApiError, listHabits, reorderHabits, restoreHabit } from "../api";
import { describeRule, groupByMoment, moveId, replaceHabit } from "../habits";
import { momentLabels, type Habit, type Moment, type Period } from "../types";
import { HabitSheet } from "./HabitSheet";

type State =
  | { status: "loading" }
  | { status: "error"; message: string }
  | {
      status: "ready";
      active: Habit[];
      archived: Habit[];
      ruleStarts: Record<Period, string>;
    };

type Ready = Extract<State, { status: "ready" }>;

/** What the bottom sheet is open for: a new habit, or an existing one. */
type Editing = { habit?: Habit } | null;

function messageOf(err: unknown, fallback: string): string {
  return err instanceof ApiError ? err.message : fallback;
}

export function Habits() {
  const [state, setState] = useState<State>({ status: "loading" });
  const [editing, setEditing] = useState<Editing>(null);
  const [notice, setNotice] = useState<string | null>(null);
  // The latest state, for the drag that commits after many renders.
  const latest = useRef(state);
  latest.current = state;

  useEffect(() => {
    let current = true;
    listHabits().then(
      (list) => current && setState({ status: "ready", ...list }),
      (err: unknown) =>
        current &&
        setState({
          status: "error",
          message: messageOf(err, "Impossible de charger les habitudes."),
        }),
    );
    return () => {
      current = false;
    };
  }, []);

  function update(change: (s: Ready) => Ready) {
    setState((s) => (s.status === "ready" ? change(s) : s));
  }

  function saved(habit: Habit) {
    update((s) => ({ ...s, active: replaceHabit(s.active, habit) }));
    setEditing(null);
  }

  function archived(habit: Habit) {
    update((s) => ({
      ...s,
      active: s.active.filter((h) => h.id !== habit.id),
      archived: [habit, ...s.archived],
    }));
    setEditing(null);
  }

  async function restore(habit: Habit) {
    setNotice(null);
    try {
      const restored = await restoreHabit(habit.id);
      update((s) => ({
        ...s,
        active: replaceHabit(s.active, restored),
        archived: s.archived.filter((h) => h.id !== habit.id),
      }));
    } catch (err) {
      setNotice(messageOf(err, "Impossible de restaurer l'habitude."));
    }
  }

  /** Shows an order without saving it (while dragging). */
  function preview(moment: Moment, ids: number[]) {
    update((s) => ({ ...s, active: withOrder(s.active, moment, ids) }));
  }

  /** Saves the order currently shown; on failure, goes back to `before`. */
  async function commit(moment: Moment, before: Habit[]) {
    const current = latest.current;
    if (current.status !== "ready") return;
    const ids = current.active.filter((h) => h.moment === moment).map((h) => h.id);
    if (ids.every((id, i) => id === before.filter((h) => h.moment === moment)[i]?.id)) return;
    setNotice(null);
    try {
      await reorderHabits(moment, ids);
    } catch (err) {
      update((s) => ({ ...s, active: before }));
      setNotice(messageOf(err, "Impossible d'enregistrer l'ordre."));
    }
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
      {notice && (
        <p className="error" role="alert">
          {notice}
        </p>
      )}
      {state.status === "ready" && (
        <>
          <HabitList
            habits={state.active}
            onEdit={(habit) => setEditing({ habit })}
            onPreview={preview}
            onCommit={commit}
          />
          {state.archived.length > 0 && (
            <details className="archived">
              <summary>Archivées ({state.archived.length})</summary>
              <ul className="habit-list">
                {state.archived.map((h) => (
                  <li key={h.id} className={`habit color-${h.color}`}>
                    <span className="habit-emoji" aria-hidden="true">
                      {h.emoji}
                    </span>
                    <span className="habit-text">
                      <span className="habit-name">{h.name}</span>
                      <span className="habit-rule muted">{describeRule(h)}</span>
                    </span>
                    <button type="button" className="ghost" onClick={() => restore(h)}>
                      Restaurer
                    </button>
                  </li>
                ))}
              </ul>
            </details>
          )}
        </>
      )}

      <button
        type="button"
        className="fab"
        aria-label="Nouvelle habitude"
        onClick={() => setEditing({})}
      >
        <svg viewBox="0 0 24 24" width="24" height="24" aria-hidden="true">
          <path
            d="M12 5v14M5 12h14"
            stroke="currentColor"
            strokeWidth="2.5"
            strokeLinecap="round"
            fill="none"
          />
        </svg>
      </button>
      {editing && state.status === "ready" && (
        <HabitSheet
          key={editing.habit?.id ?? "new"}
          habit={editing.habit}
          ruleStarts={state.ruleStarts}
          onClose={() => setEditing(null)}
          onSaved={saved}
          onArchived={archived}
        />
      )}
    </main>
  );
}

/** Gives the habits of a moment the positions of `ids`; the others are untouched. */
function withOrder(habits: Habit[], moment: Moment, ids: number[]): Habit[] {
  const position = new Map(ids.map((id, i) => [id, i]));
  return habits
    .map((h) => (h.moment === moment ? { ...h, position: position.get(h.id) ?? h.position } : h))
    .sort((a, b) => momentRank(a.moment) - momentRank(b.moment) || a.position - b.position);
}

function momentRank(m: Moment): number {
  return ["morning", "day", "evening"].indexOf(m);
}

interface ListProps {
  habits: Habit[];
  onEdit: (habit: Habit) => void;
  onPreview: (moment: Moment, ids: number[]) => void;
  onCommit: (moment: Moment, before: Habit[]) => void;
}

function HabitList({ habits, onEdit, onPreview, onCommit }: ListProps) {
  if (habits.length === 0) {
    return (
      <p className="muted">Aucune habitude pour l'instant. Touchez « + » pour en créer une.</p>
    );
  }
  return (
    <>
      {groupByMoment(habits).map((group) => (
        <MomentGroup
          key={group.moment}
          moment={group.moment}
          habits={group.habits}
          all={habits}
          onEdit={onEdit}
          onPreview={onPreview}
          onCommit={onCommit}
        />
      ))}
    </>
  );
}

interface GroupProps extends Omit<ListProps, "habits"> {
  moment: Moment;
  habits: Habit[];
  all: Habit[];
}

function MomentGroup({ moment, habits, all, onEdit, onPreview, onCommit }: GroupProps) {
  const list = useRef<HTMLUListElement>(null);
  const [dragging, setDragging] = useState<number | null>(null);
  // The order before a drag, to restore it if saving fails.
  const before = useRef<Habit[]>(all);
  const ids = habits.map((h) => h.id);

  function arrow(id: number, delta: number) {
    const next = moveId(ids, id, ids.indexOf(id) + delta);
    if (next.every((x, i) => x === ids[i])) return;
    before.current = all;
    onPreview(moment, next);
    onCommit(moment, all);
  }

  function startDrag(e: PointerEvent<HTMLButtonElement>, id: number) {
    e.currentTarget.setPointerCapture(e.pointerId);
    before.current = all;
    setDragging(id);
  }

  function drag(e: PointerEvent<HTMLButtonElement>) {
    if (dragging === null || !list.current) return;
    // The new index is the number of other rows whose middle is above the pointer.
    const rows = Array.from(list.current.children).filter(
      (row) => (row as HTMLElement).dataset.id !== String(dragging),
    );
    const index = rows.filter((row) => {
      const box = row.getBoundingClientRect();
      return box.top + box.height / 2 < e.clientY;
    }).length;
    const next = moveId(ids, dragging, index);
    if (next.some((x, i) => x !== ids[i])) onPreview(moment, next);
  }

  function endDrag() {
    if (dragging === null) return;
    setDragging(null);
    onCommit(moment, before.current);
  }

  return (
    <section className="group" aria-labelledby={`group-${moment}`}>
      <h2 id={`group-${moment}`}>{momentLabels[moment]}</h2>
      <ul className="habit-list" ref={list}>
        {habits.map((h, i) => (
          <li
            key={h.id}
            data-id={h.id}
            className={`habit color-${h.color}${dragging === h.id ? " is-dragging" : ""}`}
          >
            <button
              type="button"
              className="habit-edit"
              aria-label={`Modifier ${h.name}`}
              onClick={() => onEdit(h)}
            >
              <span className="habit-emoji" aria-hidden="true">
                {h.emoji}
              </span>
              <span className="habit-text">
                <span className="habit-name">{h.name}</span>
                <span className="habit-rule muted">{describeRule(h)}</span>
              </span>
            </button>
            <div className="reorder">
              <button
                type="button"
                className="icon"
                aria-label={`Monter ${h.name}`}
                disabled={i === 0}
                onClick={() => arrow(h.id, -1)}
              >
                ↑
              </button>
              <button
                type="button"
                className="icon"
                aria-label={`Descendre ${h.name}`}
                disabled={i === habits.length - 1}
                onClick={() => arrow(h.id, 1)}
              >
                ↓
              </button>
              <button
                type="button"
                className="icon handle"
                aria-label={`Déplacer ${h.name} en le faisant glisser`}
                onPointerDown={(e) => startDrag(e, h.id)}
                onPointerMove={drag}
                onPointerUp={endDrag}
                onPointerCancel={endDrag}
              >
                ⠿
              </button>
            </div>
          </li>
        ))}
      </ul>
    </section>
  );
}
