// PROTOTYPE — Variant B « Mosaïque »: bold bento tiles that fill up like
// liquid, aurora background, gradient title, spotlight on each tile.
import type { CSSProperties } from "react";
import { motion } from "motion/react";
import type { VProps } from "./main";
import {
  MOMENTS, MOMENT_ICON, MOMENT_LABEL, TODAY, addDays, dayRatio, fmtDayMonth, fmtWeekday, grouped, line, relLabel,
  streak, type Habit,
} from "./model";
import { CountUp, DaySwiper, GradientText, NoteBox, Spotlight, TABS, usePress } from "./effects";

export function VariantB({ date, setDate, store, onTap, onLong }: VProps) {
  const { logs } = store;
  const ratio = dayRatio(logs, date);
  return (
    <div className="vb">
      <div className="vb-aurora" aria-hidden><span /><span /><span /></div>

      <header className="vb-head">
        <div className="vb-nav">
          <button onClick={() => setDate(addDays(date, -1))} aria-label="Jour précédent">‹</button>
          <span>{relLabel(date)}</span>
          <button onClick={() => setDate(addDays(date, 1))} disabled={date >= TODAY} aria-label="Jour suivant">›</button>
        </div>
        <h1 className="vb-title"><GradientText>{fmtWeekday(date)}</GradientText></h1>
        <div className="vb-meta">
          <span>{fmtDayMonth(date)}</span>
          <span className="vb-chip">🔥 {streak(logs)} jours</span>
          <span className="vb-chip dark"><CountUp value={Math.round(ratio * 100)} /> %</span>
        </div>
      </header>

      <DaySwiper date={date} setDate={setDate}>
        {MOMENTS.map((m) => {
          const hs = grouped(logs, date, m);
          if (!hs.length) return null;
          return (
            <section key={m} className="vb-sec">
              <h2><span>{MOMENT_ICON[m]}</span>{MOMENT_LABEL[m]}</h2>
              <div className="vb-grid">
                {hs.map((h) => <Tile key={h.id} h={h} date={date} {...{ store, onTap, onLong }} />)}
              </div>
            </section>
          );
        })}
        <NoteBox date={date} store={store} className="vb-note" label="📝 Note du jour" />
      </DaySwiper>

      <nav className="vb-tabs">
        {TABS.map((t, i) => (
          <span key={t.key} className={i === 0 ? "on" : ""}><b>{t.icon}</b><small>{t.label}</small></span>
        ))}
      </nav>
    </div>
  );
}

function Tile({ h, date, store, onTap, onLong }: { h: Habit; date: string } & Pick<VProps, "store" | "onTap" | "onLong">) {
  const l = line(store.logs, h, date);
  const press = usePress((e) => onTap(h, e), () => onLong(h));
  const wide = h.period !== "day";
  return (
    <Spotlight className={"vb-tile" + (wide ? " wide" : "") + (l.complete ? " done" : "")} style={{ "--c": h.color } as CSSProperties} {...press}>
      <motion.div className="vb-fill" initial={false} animate={{ height: `${l.progress * 100}%` }} transition={{ type: "spring", stiffness: 140, damping: 20 }} />
      <div className="vb-tile-top">
        <motion.span className="vb-emoji" key={l.total} initial={{ scale: 0.7 }} animate={{ scale: 1 }} transition={{ type: "spring", stiffness: 500, damping: 12 }}>{h.emoji}</motion.span>
        {l.complete && <span className="vb-check">✓</span>}
      </div>
      <div className="vb-name">{h.name}</div>
      <div className="vb-count">
        {h.target === 1 && !h.unit ? (l.complete ? "Fait" : "Touche pour cocher") : (
          <><b>{l.total}</b><span>/ {h.target}{h.unit ? " " + h.unit : ""}</span></>
        )}
      </div>
      {l.pace && (
        <div className="vb-pace">
          <div className="vb-bar">
            <i style={{ width: `${l.progress * 100}%` }} />
            <em style={{ left: `${Math.min(100, (l.pace.expected / h.target) * 100)}%` }} />
          </div>
          <span>{l.periodText} · <b>{l.pace.text}</b></span>
        </div>
      )}
    </Spotlight>
  );
}
