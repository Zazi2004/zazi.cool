import { useCallback, useEffect, useRef, useState, type MouseEvent } from "react";
import { ApiError, getDay, putEntry } from "../api";
import { navigate } from "../router";
import {
  addDays,
  dayOfMonth,
  isBinary,
  localDate,
  longDate,
  paceTone,
  relativeLabel,
  shouldCelebrate,
  subtitle,
  tapValue,
  weekdayInitial,
  withValue,
} from "../today";
import { momentLabels, type Day, type DayHabit } from "../types";
import { DaySwiper } from "./DaySwiper";
import { DayNote } from "./DayNote";
import { EntrySheet } from "./EntrySheet";
import { CountUp } from "./effects/CountUp";
import { confetti, spark } from "./effects/fx";
import { ShinyText } from "./effects/ShinyText";
import { Ring } from "./Ring";
import { useLongPress } from "./useLongPress";

type State =
  { status: "loading" } | { status: "error"; message: string } | { status: "ready"; day: Day };

const fallback = "Impossible de charger la journée.";

// The last known version of each day, so that changing day slides to a full screen at
// once; the server's answer replaces it as soon as it arrives.
const cache = new Map<string, Day>();

function stateFor(date: string): State {
  const day = cache.get(date);
  return day ? { status: "ready", day } : { status: "loading" };
}

function dayPath(date: string, today: string): string {
  return date === today ? "/" : `/day/${date}`;
}

function preload(date: string, today: string) {
  if (date > today || cache.has(date)) return;
  // Best effort: on failure, this day simply loads when it is opened.
  getDay(date).then(
    (day) => cache.set(date, day),
    () => undefined,
  );
}

/** `date` is the day of the route; null means today. */
export function Today({ date: routeDate }: { date: string | null }) {
  const today = localDate(new Date());
  const date = routeDate ?? today;
  const [shown, setShown] = useState(date);
  const [state, setState] = useState<State>(() => stateFor(date));
  // Whether `state` comes from the server rather than from the cache.
  const [fresh, setFresh] = useState(false);
  const [notice, setNotice] = useState("");
  const [entering, setEntering] = useState<DayHabit | null>(null);
  if (shown !== date) {
    setShown(date);
    setState(stateFor(date));
    setFresh(false);
    setNotice("");
  }
  // Only the answer of the latest request replaces the day: an older, slower one is dropped.
  const latest = useRef(0);
  // Whether the day was complete before the action in flight: opening a complete day
  // must not celebrate, only an action that completes it does.
  const wasComplete = useRef(false);
  // The header and the week strip stay in place while another day loads.
  const lastDay = useRef<Day | null>(null);

  useEffect(() => {
    let current = true;
    const ticket = ++latest.current;
    wasComplete.current = cache.get(date)?.complete ?? false;
    getDay(date).then(
      (day) => {
        cache.set(date, day);
        if (!current || ticket !== latest.current) return;
        wasComplete.current = day.complete;
        setState({ status: "ready", day });
        setFresh(true);
        preload(addDays(date, -1), today);
        preload(addDays(date, 1), today);
      },
      (err: unknown) => {
        if (!current || ticket !== latest.current) return;
        const message = err instanceof ApiError ? err.message : fallback;
        if (cache.has(date)) setNotice(message);
        else setState({ status: "error", message });
      },
    );
    return () => {
      current = false;
    };
  }, [date, today]);

  const save = useCallback(
    async (habitId: number, value: number) => {
      if (state.status !== "ready" || !state.day.editable) return;
      const before = state.day;
      const ticket = ++latest.current;
      setNotice("");
      setState({ status: "ready", day: withValue(before, habitId, value) });
      try {
        const day = await putEntry(before.date, habitId, value);
        cache.set(before.date, day);
        if (ticket !== latest.current) return;
        if (shouldCelebrate(wasComplete.current, day)) confetti();
        wasComplete.current = day.complete;
        setState({ status: "ready", day });
        setFresh(true);
      } catch (err) {
        if (ticket !== latest.current) return;
        setState({ status: "ready", day: before });
        setNotice(err instanceof ApiError ? err.message : "La saisie n'a pas été enregistrée.");
      }
    },
    [state],
  );

  const day = state.status === "ready" && state.day.date === date ? state.day : null;
  if (day) lastDay.current = day;
  const summary = day ?? lastDay.current;
  const canGoForward = addDays(date, 1) <= today;
  const go = (target: string) => navigate(dayPath(target, today));

  function tap(item: DayHabit, point: Point) {
    const next = tapValue(item);
    if (next > item.value) spark(point.x, point.y, point.color);
    void save(item.habit.id, next);
  }

  return (
    <main className="page today">
      <header className="today-header">
        <div>
          <p className="eyebrow">{relativeLabel(date, today)}</p>
          <h1>{longDate(date)}</h1>
        </div>
        {summary && (
          <p
            className="streak"
            aria-label={`Série de ${summary.streak} jour${summary.streak > 1 ? "s" : ""}`}
          >
            <span aria-hidden="true">🔥</span> <ShinyText>{summary.streak} j</ShinyText>
          </p>
        )}
      </header>
      {summary && <WeekStrip week={summary.week} selected={date} today={today} onSelect={go} />}
      <DaySwiper date={date} today={today} onChange={(d) => entering === null && go(d)}>
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
              onTap={tap}
              onEnter={setEntering}
            />
            {/* The note waits for the server: typing over a cached text could erase a newer one. */}
            {fresh && <DayNote key={date} date={date} initial={day.note} editable={day.editable} />}
          </>
        )}
      </DaySwiper>
      {/* Keyboard alternative to the swipe; appears when it receives focus. */}
      <nav className="day-nav" aria-label="Changer de jour">
        <button type="button" className="ghost" onClick={() => go(addDays(date, -1))}>
          ← Jour précédent
        </button>
        <button
          type="button"
          className="ghost"
          disabled={!canGoForward}
          onClick={() => go(addDays(date, 1))}
        >
          Jour suivant →
        </button>
      </nav>
      {entering && (
        <EntrySheet
          item={entering}
          onSave={(value) => void save(entering.habit.id, value)}
          onClose={() => setEntering(null)}
        />
      )}
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
  onSelect,
}: {
  week: Day["week"];
  selected: string;
  today: string;
  onSelect: (date: string) => void;
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
            onClick={() => onSelect(d.date)}
          >
            <span className="week-initial" aria-hidden="true">
              {weekdayInitial(d.date)}
            </span>
            <Ring size={32} stroke={4} progress={future ? 0 : d.progress} />
            <span className="week-number" aria-hidden="true">
              {dayOfMonth(d.date)}
            </span>
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
  onTap: (item: DayHabit, point: Point) => void;
  onEnter: (item: DayHabit) => void;
}) {
  const percent = Math.round(day.progress * 100);
  return (
    <>
      <section
        className="hero"
        role="img"
        aria-label={`${percent} % de la journée, ${day.done} sur ${day.total}`}
      >
        <Ring size={176} stroke={18} progress={day.progress} gradient>
          <span className="hero-percent">
            <CountUp to={percent} />
            <small>%</small>
          </span>
          <span className="hero-count">
            {day.done} / {day.total} du jour
          </span>
        </Ring>
      </section>
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

/** Where a tap happened, and the colour of its sparks. */
interface Point {
  x: number;
  y: number;
  color: string;
}

function tapPoint(e: MouseEvent<HTMLButtonElement>): Point {
  const box = e.currentTarget.getBoundingClientRect();
  // A click from the keyboard has no position (detail 0): spark from the row's centre.
  const keyboard = e.detail === 0;
  return {
    x: keyboard ? box.left + box.width / 2 : e.clientX,
    y: keyboard ? box.top + box.height / 2 : e.clientY,
    color: getComputedStyle(e.currentTarget).getPropertyValue("--tone").trim(),
  };
}

function Row({
  item,
  editable,
  onTap,
  onEnter,
}: {
  item: DayHabit;
  editable: boolean;
  onTap: (item: DayHabit, point: Point) => void;
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
        onClick={(e) => !press.consumeClick() && onTap(item, tapPoint(e))}
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
        <Ring size={44} stroke={5} progress={item.progress}>
          <span className="habit-mark" aria-hidden="true">
            {item.complete ? "✓" : isBinary(h) ? "" : "+"}
          </span>
        </Ring>
      </button>
      {/* Keyboard alternative to the long press; appears when it receives focus. */}
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
