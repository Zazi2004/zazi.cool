// PROTOTYPE — Variant A « Anneaux »: Apple Fitness–like. Week strip of mini
// rings, a big hero ring for the day, clean list rows with a ring button.
import type { CSSProperties } from "react";
import type { VProps } from "./main";
import {
  MOMENTS, MOMENT_LABEL, TODAY, dailyHabits, dayNum, dayRatio, fmtLong, fmtLetter, grouped, isComplete,
  line, relLabel, streak, weekDates, type Habit,
} from "./model";
import { CountUp, DaySwiper, NoteBox, Ring, ShinyText, TABS, usePress } from "./effects";

export function VariantA({ date, setDate, store, onTap, onLong }: VProps) {
  const { logs } = store;
  const daily = dailyHabits(date);
  const done = daily.filter((h) => isComplete(logs, h, date)).length;
  const ratio = dayRatio(logs, date);

  return (
    <div className="va">
      <header className="va-top">
        <div>
          <div className="va-eyebrow">{relLabel(date)}</div>
          <h1 className="va-title">{fmtLong(date)}</h1>
        </div>
        <div className="va-streak">🔥 <ShinyText>{streak(logs)} j</ShinyText></div>
      </header>

      <div className="va-week">
        {weekDates(date).map((d) => (
          <button key={d} className={"va-wd" + (d === date ? " on" : "")} disabled={d > TODAY} onClick={() => setDate(d)}>
            <span className="va-wd-l">{fmtLetter(d)}</span>
            <Ring size={32} stroke={4} progress={d > TODAY ? 0 : dayRatio(logs, d)} color="#6366F1" />
            <span className="va-wd-n">{dayNum(d)}</span>
          </button>
        ))}
      </div>

      <DaySwiper date={date} setDate={setDate}>
        <section className="va-hero">
          <Ring size={176} stroke={18} progress={ratio} gradient={["#6366F1", "#EC4899"]}>
            <div className="va-hero-pct"><CountUp value={Math.round(ratio * 100)} /><small>%</small></div>
            <div className="va-hero-sub">{done} / {daily.length} du jour</div>
          </Ring>
        </section>

        {MOMENTS.map((m) => {
          const hs = grouped(logs, date, m);
          if (!hs.length) return null;
          return (
            <section key={m} className="va-group">
              <h2>{MOMENT_LABEL[m]}</h2>
              <div className="va-card">
                {hs.map((h) => <RowA key={h.id} h={h} date={date} {...{ store, onTap, onLong }} />)}
              </div>
            </section>
          );
        })}

        <NoteBox date={date} store={store} className="va-note" />
      </DaySwiper>

      <nav className="va-tabs">
        {TABS.map((t, i) => (
          <span key={t.key} className={i === 0 ? "on" : ""}><b>{t.icon}</b>{t.label}</span>
        ))}
      </nav>
    </div>
  );
}

function RowA({ h, date, store, onTap, onLong }: { h: Habit; date: string } & Pick<VProps, "store" | "onTap" | "onLong">) {
  const l = line(store.logs, h, date);
  const press = usePress((e) => onTap(h, e), () => onLong(h));
  return (
    <div className={"va-row" + (l.complete ? " done" : "")} {...press} style={{ "--c": h.color } as CSSProperties}>
      <span className="va-emoji">{h.emoji}</span>
      <div className="va-txt">
        <div className="va-name">{h.name}</div>
        <div className="va-sub">
          {l.count}{l.periodText && ` ${l.periodText}`}
          {l.pace && <span className={"va-pace " + (l.pace.diff >= -0.5 ? "up" : "down")}> · {l.pace.text}</span>}
        </div>
      </div>
      <Ring size={44} stroke={5} progress={l.progress} color={h.color}>
        <span className="va-btn">{l.complete ? "✓" : h.target === 1 && !h.unit ? "" : "+"}</span>
      </Ring>
    </div>
  );
}
