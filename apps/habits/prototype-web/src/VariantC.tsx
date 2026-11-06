// PROTOTYPE — Variant C « Journal »: editorial, warm paper, serif titles.
// A vertical timeline of the day; quantities shown as pills/dots, not numbers.
import { useEffect, useRef, type CSSProperties } from "react";
import { motion } from "motion/react";
import type { VProps } from "./main";
import {
  MOMENTS, MOMENT_LABEL, TODAY, addDays, dailyHabits, dayComplete, dayNum, fmtDayMonth, fmtLetter, fmtWeekday,
  grouped, isComplete, line, relLabel, streak, type Habit,
} from "./model";
import { DaySwiper, NoteBox, ShinyText, TABS, usePress } from "./effects";

export function VariantC({ date, setDate, store, onTap, onLong }: VProps) {
  const { logs } = store;
  const days = Array.from({ length: 14 }, (_, i) => addDays(TODAY, i - 13));
  const daily = dailyHabits(date);
  const strip = useRef<HTMLDivElement>(null);
  useEffect(() => { strip.current?.scrollTo({ left: 9999 }); }, []);
  return (
    <div className="vc">
      <header className="vc-head">
        <div className="vc-kicker">{relLabel(date)}</div>
        <h1 className="vc-date">{fmtWeekday(date)} <em>{fmtDayMonth(date)}</em></h1>
        <p className="vc-streak"><ShinyText>{streak(logs)} jours complets d'affilée</ShinyText></p>
      </header>

      <div className="vc-days" ref={strip}>
        {days.map((d) => (
          <button key={d} className={"vc-day" + (d === date ? " on" : "")} onClick={() => setDate(d)}>
            <span>{fmtLetter(d)}</span>
            <b>{dayNum(d)}</b>
            <i className={dayComplete(logs, d) ? "full" : ""} />
          </button>
        ))}
      </div>

      <div className="vc-seg">
        {daily.map((h) => (
          <motion.i key={h.id} initial={false} animate={{ backgroundColor: isComplete(logs, h, date) ? h.color : "#E7E1D6" }} />
        ))}
      </div>

      <DaySwiper date={date} setDate={setDate}>
        <div className="vc-timeline">
          {MOMENTS.map((m) => {
            const hs = grouped(logs, date, m);
            if (!hs.length) return null;
            return (
              <section key={m} className="vc-moment">
                <h2>{MOMENT_LABEL[m]}</h2>
                {hs.map((h) => <Item key={h.id} h={h} date={date} {...{ store, onTap, onLong }} />)}
              </section>
            );
          })}
        </div>
        <NoteBox date={date} store={store} className="vc-note" label="Carnet" placeholder="Une ligne sur ta journée…" />
      </DaySwiper>

      <nav className="vc-tabs">
        {TABS.map((t, i) => <span key={t.key} className={i === 0 ? "on" : ""}>{t.label}</span>)}
      </nav>
    </div>
  );
}

function Item({ h, date, store, onTap, onLong }: { h: Habit; date: string } & Pick<VProps, "store" | "onTap" | "onLong">) {
  const l = line(store.logs, h, date);
  const press = usePress((e) => onTap(h, e), () => onLong(h));
  const dots = h.period === "day" && h.target > 1 && h.target <= 10;
  return (
    <div className={"vc-item" + (l.complete ? " done" : "")} style={{ "--c": h.color, "--p": l.progress } as CSSProperties} {...press}>
      <div className="vc-node"><span>{l.complete ? "✓" : h.emoji}</span></div>
      <div className="vc-body">
        <div className="vc-name">{h.name}</div>
        {dots ? (
          <div className="vc-dots">
            {Array.from({ length: h.target }, (_, i) => (
              <motion.i key={i} initial={false} animate={{ scale: i < l.total ? 1 : 0.75, backgroundColor: i < l.total ? h.color : "#E7E1D6" }} />
            ))}
            <span>{l.total}/{h.target} {h.unit}</span>
          </div>
        ) : h.target === 1 && !h.unit ? (
          <div className="vc-meta">{l.complete ? "fait" : "touche pour cocher"}</div>
        ) : (
          <div className="vc-line">
            <div className="vc-track">
              <motion.i initial={false} animate={{ width: `${l.progress * 100}%` }} />
              {l.pace && <em style={{ left: `${Math.min(100, (l.pace.expected / h.target) * 100)}%` }} />}
            </div>
            <div className="vc-meta">
              {l.total} / {h.target} {h.unit} {l.periodText}
              {l.pace && <> — <b className={l.pace.diff >= -0.5 ? "up" : "down"}>{l.pace.text}</b></>}
            </div>
          </div>
        )}
      </div>
    </div>
  );
}
