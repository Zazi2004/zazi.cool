import { useEffect, useState } from "react";
import { ApiError, getHabitStats } from "../api";
import { navigate } from "../router";
import {
  cellDate,
  firstRow,
  formatRate,
  formatStreak,
  hasData,
  rangeLabel,
  statusLabels,
  totalLine,
} from "../stats";
import { localDate } from "../today";
import type { HabitStats } from "../types";
import { MiniRing } from "./MiniRing";
import { paceClass } from "./paceClass";

type State =
  | { status: "loading" }
  | { status: "error"; message: string }
  | { status: "ready"; stats: HabitStats };

/** The page of one habit: figures, the heatmap of a year and the list of periods. */
export function StatsHabit({ id }: { id: number }) {
  const thisYear = Number(localDate(new Date()).slice(0, 4));
  const [year, setYear] = useState(thisYear);
  const [state, setState] = useState<State>({ status: "loading" });

  useEffect(() => {
    let current = true;
    setState({ status: "loading" });
    getHabitStats(id, year).then(
      (stats) => current && setState({ status: "ready", stats }),
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
  }, [id, year]);

  const stats = state.status === "ready" ? state.stats : null;
  return (
    <main className={`page${stats ? ` color-${stats.habit.color}` : ""}`}>
      <a
        className="back"
        href="/stats"
        onClick={(e) => {
          e.preventDefault();
          navigate("/stats");
        }}
      >
        ← Stats
      </a>
      {state.status === "loading" && <p className="muted">Chargement…</p>}
      {state.status === "error" && (
        <p className="error" role="alert">
          {state.message}
        </p>
      )}
      {stats && (
        <>
          <Header stats={stats} />
          <Figures stats={stats} />
          <section aria-label="Calendrier de l'année">
            <div className="year-nav">
              <button
                type="button"
                className="ghost"
                aria-label="Année précédente"
                disabled={!hasData(stats.days)}
                onClick={() => setYear(year - 1)}
              >
                ‹
              </button>
              <h2>{stats.year}</h2>
              <button
                type="button"
                className="ghost"
                aria-label="Année suivante"
                disabled={year >= thisYear}
                onClick={() => setYear(year + 1)}
              >
                ›
              </button>
            </div>
            <Heatmap stats={stats} />
          </section>
          {stats.periods.length > 0 && <Periods stats={stats} />}
        </>
      )}
    </main>
  );
}

function Header({ stats }: { stats: HabitStats }) {
  return (
    <header className="stats-header">
      <span className="habit-emoji" aria-hidden="true">
        {stats.habit.emoji}
      </span>
      <div className="habit-text">
        <h1>{stats.habit.name}</h1>
        <p className="muted">
          {totalLine(stats)}
          {stats.pace && (
            <>
              {" · "}
              <span className={paceClass(stats)}>{stats.pace.label}</span>
            </>
          )}
        </p>
      </div>
      <MiniRing progress={stats.progress} />
    </header>
  );
}

function Figures({ stats }: { stats: HabitStats }) {
  const period = stats.habit.period;
  const figures = [
    { label: "Série en cours", value: formatStreak(period, stats.streak) },
    { label: "Meilleure série", value: formatStreak(period, stats.bestStreak) },
    { label: "Réussite sur 30 jours", value: formatRate(stats.rate30) },
    { label: `Réussite en ${stats.year}`, value: formatRate(stats.rateYear) },
  ];
  return (
    <dl className="figures">
      {figures.map((f) => (
        <div key={f.label} className="figure">
          <dt className="muted">{f.label}</dt>
          <dd>{f.value}</dd>
        </div>
      ))}
    </dl>
  );
}

function Heatmap({ stats }: { stats: HabitStats }) {
  return (
    <ol
      className={`heatmap row-${firstRow(stats.days)}`}
      aria-label={`Saisies de ${stats.year}, une case par jour`}
    >
      {stats.days.map((d) => (
        <li
          key={d.date}
          className={d.level === null ? "lvl-none" : `lvl-${d.level}`}
          title={`${cellDate(d.date)}${d.level === null ? "" : ` : ${d.value}`}`}
        >
          <span className="sr-only">
            {cellDate(d.date)}
            {d.level === null ? ", non prévu" : ` : ${d.value}`}
          </span>
        </li>
      ))}
    </ol>
  );
}

function Periods({ stats }: { stats: HabitStats }) {
  const unit = stats.habit.unit ? ` ${stats.habit.unit}` : "";
  return (
    <section aria-label="Périodes">
      <h2>Périodes</h2>
      <ul className="period-list">
        {stats.periods.map((p) => (
          <li key={p.start} className={`period period-${p.status}`}>
            <span>{rangeLabel(p.start, p.end)}</span>
            <span className="muted">
              {p.total} / {p.target}
              {unit}
            </span>
            <span className="period-status">{statusLabels[p.status]}</span>
          </li>
        ))}
      </ul>
    </section>
  );
}
