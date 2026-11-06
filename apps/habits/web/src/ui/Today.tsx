import { useCallback, useEffect, useRef, useState, type PointerEvent } from "react";
import { ApiError, getDay, putEntry } from "../api";
import { navigate } from "../router";
import {
  addDays,
  localDate,
  longDate,
  paceTone,
  relativeLabel,
  shouldCelebrate,
  subtitle,
  swipeTarget,
  tapValue,
  weekdayInitial,
  withValue,
} from "../today";
import { momentLabels, type Day, type DayHabit } from "../types";
import { Confetti } from "./Confetti";
import { DayNote } from "./DayNote";
import { EntrySheet } from "./EntrySheet";
import { useLongPress } from "./useLongPress";

type State =
  { status: "loading" } | { status: "error"; message: string } | { status: "ready"; day: Day };

const fallback = "Impossible de charger la journée.";

function dayPath(date: string, today: string): string {
  return date === today ? "/" : `/day/${date}`;
}

function prefersReducedMotion(): boolean {
  return window.matchMedia("(prefers-reduced-motion: reduce)").matches;
}

/** `date` is the day of the route; null means today. The screen is keyed by date. */
export function Today({ date: routeDate }: { date: string | null }) {
  const today = localDate(new Date());
  const date = routeDate ?? today;
  const [state, setState] = useState<State>({ status: "loading" });
  const [notice, setNotice] = useState("");
  const [entering, setEntering] = useState<DayHabit | null>(null);
  const [celebrating, setCelebrating] = useState(false);
  // Only the answer of the latest save replaces the day: an older, slower one is dropped.
  const latest = useRef(0);
  // Whether the day was complete before the action in flight: opening a complete day
  // must not celebrate, only an action that completes it does.
  const wasComplete = useRef(false);

  useEffect(() => {
    let current = true;
    getDay(date).then(
      (day) => {
        if (!current) return;
        wasComplete.current = day.complete;
        setState({ status: "ready", day });
      },
      (err: unknown) =>
        current &&
        setState({ status: "error", message: err instanceof ApiError ? err.message : fallback }),
    );
    return () => {
      current = false;
    };
  }, [date]);

  const save = useCallback(
    async (habitId: number, value: number) => {
      if (state.status !== "ready" || !state.day.editable) return;
      const before = state.day;
      const ticket = ++latest.current;
      setNotice("");
      setState({ status: "ready", day: withValue(before, habitId, value) });
      try {
        const day = await putEntry(before.date, habitId, value);
        if (ticket !== latest.current) return;
        if (shouldCelebrate(wasComplete.current, day) && !prefersReducedMotion()) {
          setCelebrating(true);
        }
        wasComplete.current = day.complete;
        setState({ status: "ready", day });
      } catch (err) {
        if (ticket !== latest.current) return;
        setState({ status: "ready", day: before });
        setNotice(err instanceof ApiError ? err.message : "La saisie n'a pas été enregistrée.");
      }
    },
    [state],
  );

  const swipe = useRef<{ x: number; y: number } | null>(null);
  function swipeStart(e: PointerEvent) {
    const ignored = (e.target as HTMLElement).closest("textarea, input, dialog, .week");
    swipe.current = ignored ? null : { x: e.clientX, y: e.clientY };
  }
  function swipeEnd(e: PointerEvent) {
    const from = swipe.current;
    swipe.current = null;
    if (!from) return;
    const target = swipeTarget(date, today, e.clientX - from.x, e.clientY - from.y);
    if (target) navigate(dayPath(target, today));
  }

  const day = state.status === "ready" ? state.day : null;
  const canGoForward = addDays(date, 1) <= today;

  return (
    <main
      className="page today"
      onPointerDown={swipeStart}
      onPointerUp={swipeEnd}
      onPointerCancel={() => (swipe.current = null)}
    >
      <header className="today-header">
        <div>
          <h1>{relativeLabel(date, today)}</h1>
          <p className="muted today-date">{longDate(date)}</p>
        </div>
        {day && (
          <p
            className="streak"
            aria-label={`Série de ${day.streak} jour${day.streak > 1 ? "s" : ""}`}
          >
            <span aria-hidden="true">🔥</span> {day.streak}
          </p>
        )}
      </header>
      {day && <WeekStrip week={day.week} selected={date} today={today} />}
      {state.status === "loading" && <p className="muted">Chargement…</p>}
      {state.status === "error" && (
        <p className="error" role="alert">
          {state.message}
        </p>
      )}
      {day && (
        <>
          {!day.editable && (
            <p className="readonly" role="status">
              Lecture seule
            </p>
          )}
          <DayView
            day={day}
            wording={date === today ? "aujourd'hui" : "ce jour-là"}
            onTap={(item) => save(item.habit.id, tapValue(item))}
            onEnter={setEntering}
          />
          <DayNote date={date} initial={day.note} editable={day.editable} />
          <nav className="day-nav" aria-label="Changer de jour">
            <button
              type="button"
              className="ghost"
              onClick={() => navigate(dayPath(addDays(date, -1), today))}
            >
              ← Jour précédent
            </button>
            <button
              type="button"
              className="ghost"
              disabled={!canGoForward}
              onClick={() => navigate(dayPath(addDays(date, 1), today))}
            >
              Jour suivant →
            </button>
          </nav>
        </>
      )}
      {entering && (
        <EntrySheet
          item={entering}
          onSave={(value) => void save(entering.habit.id, value)}
          onClose={() => setEntering(null)}
        />
      )}
      {celebrating && <Confetti onDone={() => setCelebrating(false)} />}
      <p className="toast" role="alert" hidden={notice === ""}>
        {notice}
      </p>
    </main>
  );
}

function WeekStrip({
  week,
  selected,
  today,
}: {
  week: Day["week"];
  selected: string;
  today: string;
}) {
  return (
    <nav className="week" aria-label="Semaine">
      {week.map((d) => {
        const future = d.date > today;
        return (
          <button
            key={d.date}
            type="button"
            className="week-day"
            disabled={future}
            aria-current={d.date === selected ? "date" : undefined}
            aria-label={longDate(d.date)}
            onClick={() => navigate(dayPath(d.date, today))}
          >
            <span className="week-initial" aria-hidden="true">
              {weekdayInitial(d.date)}
            </span>
            <MiniRing progress={future ? 0 : d.progress} />
          </button>
        );
      })}
    </nav>
  );
}

function DayView({
  day,
  wording,
  onTap,
  onEnter,
}: {
  day: Day;
  wording: string;
  onTap: (item: DayHabit) => void;
  onEnter: (item: DayHabit) => void;
}) {
  return (
    <>
      <Ring progress={day.progress} done={day.done} total={day.total} />
      {day.groups.length === 0 && (
        <p className="muted">Aucune habitude {wording}. Créez-en une dans l'onglet Habitudes.</p>
      )}
      {day.groups.map((group) => (
        <section key={group.moment} className="group" aria-labelledby={`today-${group.moment}`}>
          <h2 id={`today-${group.moment}`}>{momentLabels[group.moment]}</h2>
          <ul className="habit-list">
            {group.habits.map((item) => (
              <Row
                key={item.habit.id}
                item={item}
                editable={day.editable}
                onTap={onTap}
                onEnter={onEnter}
              />
            ))}
          </ul>
        </section>
      ))}
    </>
  );
}

function Row({
  item,
  editable,
  onTap,
  onEnter,
}: {
  item: DayHabit;
  editable: boolean;
  onTap: (item: DayHabit) => void;
  onEnter: (item: DayHabit) => void;
}) {
  const h = item.habit;
  const press = useLongPress(() => onEnter(item), editable);
  return (
    <li className={`habit color-${h.color}${item.complete ? " is-done" : ""}`}>
      <button
        type="button"
        className="habit-tap"
        aria-pressed={item.complete}
        disabled={!editable}
        onClick={() => !press.consumeClick() && onTap(item)}
        {...press.handlers}
      >
        <span className="habit-emoji" aria-hidden="true">
          {h.emoji}
        </span>
        <span className="habit-text">
          <span className="habit-name">{h.name}</span>
          <span className="habit-rule muted">
            {subtitle(item)}
            {item.pace && (
              <>
                {" · "}
                <span className={`pace pace-${paceTone(item)}`}>{item.pace.label}</span>
              </>
            )}
          </span>
        </span>
        <MiniRing progress={item.progress} />
      </button>
      <button
        type="button"
        className="habit-enter"
        disabled={!editable}
        aria-label={`Saisir une valeur pour ${h.name}`}
        onClick={() => onEnter(item)}
      >
        <span aria-hidden="true">✎</span>
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
