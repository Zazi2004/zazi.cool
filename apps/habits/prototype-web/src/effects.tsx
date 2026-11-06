// PROTOTYPE — throwaway. React Bits–style effects (CountUp, ShinyText,
// GradientText, SpotlightCard, ClickSpark, confetti) plus shared gestures.
import { useEffect, useId, useRef, useState, type CSSProperties, type PointerEvent, type ReactNode } from "react";
import { AnimatePresence, animate, motion } from "motion/react";
import { TODAY, addDays, editable, isBinary, type Habit, type Store, PALETTE } from "./model";

// ---- CountUp ----
export function CountUp({ value }: { value: number }) {
  const [shown, setShown] = useState(value);
  const prev = useRef(value);
  useEffect(() => {
    const c = animate(prev.current, value, { duration: 0.6, ease: "easeOut", onUpdate: (v) => setShown(Math.round(v)) });
    prev.current = value;
    return () => c.stop();
  }, [value]);
  return <>{shown}</>;
}

export const ShinyText = ({ children, className = "" }: { children: ReactNode; className?: string }) => (
  <span className={"fx-shiny " + className}>{children}</span>
);
export const GradientText = ({ children, className = "" }: { children: ReactNode; className?: string }) => (
  <span className={"fx-gradient " + className}>{children}</span>
);

// ---- SpotlightCard: radial light following the pointer ----
export function Spotlight({ className = "", style, children, ...rest }: { className?: string; style?: CSSProperties; children: ReactNode } & Record<string, unknown>) {
  const ref = useRef<HTMLDivElement>(null);
  return (
    <div
      ref={ref}
      className={"fx-spot " + className}
      style={style}
      onPointerMove={(e) => {
        const r = ref.current!.getBoundingClientRect();
        ref.current!.style.setProperty("--mx", `${e.clientX - r.left}px`);
        ref.current!.style.setProperty("--my", `${e.clientY - r.top}px`);
      }}
      {...rest}
    >
      {children}
    </div>
  );
}

// ---- Ring ----
export function Ring({ size, stroke, progress, color, gradient, track = "rgba(15,23,42,.08)", children }: {
  size: number; stroke: number; progress: number; color?: string; gradient?: [string, string]; track?: string; children?: ReactNode;
}) {
  const id = useId();
  const r = (size - stroke) / 2;
  const c = 2 * Math.PI * r;
  return (
    <div className="fx-ring" style={{ width: size, height: size }}>
      <svg width={size} height={size}>
        {gradient && (
          <defs>
            <linearGradient id={id} x1="0" y1="0" x2="1" y2="1">
              <stop offset="0" stopColor={gradient[0]} />
              <stop offset="1" stopColor={gradient[1]} />
            </linearGradient>
          </defs>
        )}
        <circle cx={size / 2} cy={size / 2} r={r} fill="none" stroke={track} strokeWidth={stroke} />
        <motion.circle
          cx={size / 2} cy={size / 2} r={r} fill="none"
          stroke={gradient ? `url(#${id})` : color} strokeWidth={stroke} strokeLinecap="round"
          strokeDasharray={c}
          initial={false}
          animate={{ strokeDashoffset: c * (1 - Math.max(0.0001, progress)) }}
          transition={{ type: "spring", stiffness: 120, damping: 20 }}
          transform={`rotate(-90 ${size / 2} ${size / 2})`}
        />
      </svg>
      {children && <div className="fx-ring-in">{children}</div>}
    </div>
  );
}

// ---- FX canvas: click sparks + confetti ----
type P = { x: number; y: number; vx: number; vy: number; life: number; max: number; color: string; kind: "spark" | "conf"; rot: number; vr: number; w: number };
const parts: P[] = [];
let ctx: CanvasRenderingContext2D | null = null;
let running = false;
function loop() {
  if (!ctx) return;
  const { width, height } = ctx.canvas;
  ctx.clearRect(0, 0, width, height);
  for (let i = parts.length - 1; i >= 0; i--) {
    const p = parts[i];
    p.life++;
    if (p.life > p.max) { parts.splice(i, 1); continue; }
    const a = 1 - p.life / p.max;
    ctx.globalAlpha = a;
    if (p.kind === "spark") {
      p.x += p.vx; p.y += p.vy; p.vx *= 0.9; p.vy *= 0.9;
      ctx.strokeStyle = p.color; ctx.lineWidth = 2.5; ctx.lineCap = "round";
      ctx.beginPath(); ctx.moveTo(p.x, p.y); ctx.lineTo(p.x - p.vx * 3, p.y - p.vy * 3); ctx.stroke();
    } else {
      p.vy += 0.32; p.vx *= 0.99; p.x += p.vx; p.y += p.vy; p.rot += p.vr;
      ctx.save(); ctx.translate(p.x, p.y); ctx.rotate(p.rot); ctx.fillStyle = p.color;
      ctx.fillRect(-p.w / 2, -p.w / 4, p.w, p.w / 2); ctx.restore();
    }
  }
  ctx.globalAlpha = 1;
  if (parts.length) requestAnimationFrame(loop); else running = false;
}
function kick() { if (!running) { running = true; requestAnimationFrame(loop); } }
export const fx = {
  spark(x: number, y: number, color: string) {
    for (let i = 0; i < 10; i++) {
      const a = (i / 10) * Math.PI * 2;
      parts.push({ x, y, vx: Math.cos(a) * 6, vy: Math.sin(a) * 6, life: 0, max: 20, color, kind: "spark", rot: 0, vr: 0, w: 0 });
    }
    kick();
  },
  confetti() {
    const colors = Object.values(PALETTE);
    const w = window.innerWidth;
    for (let i = 0; i < 160; i++) {
      parts.push({
        x: w / 2 + (Math.random() - 0.5) * 120, y: window.innerHeight * 0.4,
        vx: (Math.random() - 0.5) * 16, vy: -Math.random() * 15 - 5, life: 0, max: 110 + Math.random() * 40,
        color: colors[i % colors.length], kind: "conf", rot: Math.random() * 6, vr: (Math.random() - 0.5) * 0.4, w: 8 + Math.random() * 6,
      });
    }
    kick();
  },
};
export function FxCanvas() {
  const ref = useRef<HTMLCanvasElement>(null);
  useEffect(() => {
    const c = ref.current!;
    const resize = () => { c.width = window.innerWidth; c.height = window.innerHeight; };
    resize();
    ctx = c.getContext("2d");
    window.addEventListener("resize", resize);
    return () => window.removeEventListener("resize", resize);
  }, []);
  return <canvas ref={ref} className="fx-canvas" />;
}

// ---- tap vs long-press (cancelled by movement so swiping still works) ----
export type Pt = { clientX: number; clientY: number };
export function usePress(onTap: (e: Pt) => void, onLong: () => void) {
  const st = useRef<{ x: number; y: number; t: number; fired: boolean; moved: boolean } | null>(null);
  return {
    onPointerDown: (e: PointerEvent) => {
      const s = { x: e.clientX, y: e.clientY, t: 0, fired: false, moved: false };
      s.t = window.setTimeout(() => { s.fired = true; navigator.vibrate?.(10); onLong(); }, 450);
      st.current = s;
    },
    onPointerMove: (e: PointerEvent) => {
      const s = st.current;
      if (s && Math.hypot(e.clientX - s.x, e.clientY - s.y) > 10) { clearTimeout(s.t); s.moved = true; }
    },
    onPointerUp: () => { if (st.current) clearTimeout(st.current.t); },
    onPointerCancel: () => { if (st.current) clearTimeout(st.current.t); },
    onClick: (e: Pt) => {
      const s = st.current;
      st.current = null;
      if (s && (s.fired || s.moved)) return;
      onTap(e);
    },
    onContextMenu: (e: { preventDefault: () => void }) => e.preventDefault(),
  };
}

export function tapValue(h: Habit, current: number) {
  return isBinary(h) ? (current ? 0 : 1) : current + (h.step ?? 1);
}

// ---- swipe between days ----
export function DaySwiper({ date, setDate, children }: { date: string; setDate: (d: string) => void; children: ReactNode }) {
  const [dir, setDir] = useState(0);
  const go = (n: number) => {
    const d = addDays(date, n);
    if (d > TODAY) return;
    setDir(n);
    setDate(d);
  };
  return (
    <div className="fx-swiper">
      <AnimatePresence mode="popLayout" initial={false} custom={dir}>
        <motion.div
          key={date}
          custom={dir}
          variants={{
            enter: (d: number) => ({ x: d > 0 ? "60%" : "-60%", opacity: 0 }),
            center: { x: 0, opacity: 1 },
            exit: (d: number) => ({ x: d > 0 ? "-60%" : "60%", opacity: 0 }),
          }}
          initial="enter" animate="center" exit="exit"
          transition={{ type: "spring", stiffness: 320, damping: 34 }}
          drag="x" dragDirectionLock dragConstraints={{ left: 0, right: 0 }} dragElastic={0.3}
          style={{ touchAction: "pan-y" }}
          onDragEnd={(_, i) => { if (i.offset.x < -70) go(1); else if (i.offset.x > 70) go(-1); }}
        >
          {!editable(date) && <div className="fx-readonly">Lecture seule · au-delà de 7 jours</div>}
          {children}
        </motion.div>
      </AnimatePresence>
    </div>
  );
}

// ---- long-press sheet: exact value ----
export function ValueSheet({ habit, value, onClose, onSave }: { habit: Habit | null; value: number; onClose: () => void; onSave: (v: number) => void }) {
  const [v, setV] = useState(value);
  useEffect(() => setV(value), [value, habit]);
  const step = habit?.step ?? 1;
  return (
    <AnimatePresence>
      {habit && (
        <>
          <motion.div className="sheet-backdrop" initial={{ opacity: 0 }} animate={{ opacity: 1 }} exit={{ opacity: 0 }} onClick={onClose} />
          <motion.div
            className="sheet" initial={{ y: "100%" }} animate={{ y: 0 }} exit={{ y: "100%" }}
            transition={{ type: "spring", stiffness: 380, damping: 36 }}
            drag="y" dragConstraints={{ top: 0, bottom: 0 }} dragElastic={{ top: 0, bottom: 0.6 }}
            onDragEnd={(_, i) => { if (i.offset.y > 90) onClose(); }}
          >
            <div className="sheet-grip" />
            <div className="sheet-head" style={{ "--c": habit.color } as CSSProperties}>
              <span className="sheet-emoji">{habit.emoji}</span>
              <div>
                <div className="sheet-name">{habit.name}</div>
                <div className="sheet-sub">Aujourd'hui · objectif {habit.target}{habit.unit ? " " + habit.unit : ""}{habit.period !== "day" ? " sur la période" : ""}</div>
              </div>
            </div>
            <div className="sheet-stepper">
              <button onClick={() => setV(Math.max(0, v - step))} aria-label="Moins">−</button>
              <input inputMode="numeric" value={v} onChange={(e) => setV(Number(e.target.value.replace(/\D/g, "")) || 0)} />
              <button onClick={() => setV(v + step)} aria-label="Plus">+</button>
            </div>
            <div className="sheet-unit">{habit.unit ?? "fois"}</div>
            <button className="sheet-save" style={{ background: habit.color }} onClick={() => onSave(v)}>Enregistrer</button>
          </motion.div>
        </>
      )}
    </AnimatePresence>
  );
}

// ---- daily note ----
export function NoteBox({ date, store, className, label = "Note du jour", placeholder = "Comment s'est passée la journée ?" }: {
  date: string; store: Store; className: string; label?: string; placeholder?: string;
}) {
  const text = store.notes[date] ?? "";
  return (
    <label className={className}>
      <span className="note-label">{label}<span className="note-count">{text.length} / 1000</span></span>
      <textarea
        value={text} maxLength={1000} rows={3} placeholder={placeholder} disabled={!editable(date)}
        onChange={(e) => store.setNote(date, e.target.value)}
      />
    </label>
  );
}

export const TABS = [
  { key: "today", label: "Aujourd'hui", icon: "◉" },
  { key: "stats", label: "Stats", icon: "▦" },
  { key: "habits", label: "Habitudes", icon: "☰" },
];
