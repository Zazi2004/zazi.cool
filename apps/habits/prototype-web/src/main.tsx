// PROTOTYPE — throwaway. Three radically different "Aujourd'hui" screens for
// Habits v2, switchable via ?variant=A|B|C and the floating switcher.
import { StrictMode, useEffect, useRef, useState } from "react";
import { createRoot } from "react-dom/client";
import { TODAY, dayComplete, editable, value, useStore, type Habit, type Store } from "./model";
import { FxCanvas, ValueSheet, fx, tapValue, type Pt } from "./effects";
import { VariantA } from "./VariantA";
import { VariantB } from "./VariantB";
import { VariantC } from "./VariantC";
import "./styles.css";

export interface VProps {
  date: string;
  setDate: (d: string) => void;
  store: Store;
  onTap: (h: Habit, e: Pt) => void;
  onLong: (h: Habit) => void;
}

const VARIANTS = [
  { key: "A", name: "Anneaux", C: VariantA },
  { key: "B", name: "Mosaïque", C: VariantB },
  { key: "C", name: "Journal", C: VariantC },
];

function App() {
  const store = useStore();
  const [date, setDate] = useState(TODAY);
  const [sheet, setSheet] = useState<Habit | null>(null);
  const [vk, setVk] = useState(() => new URLSearchParams(location.search).get("variant") ?? "A");
  const idx = Math.max(0, VARIANTS.findIndex((v) => v.key === vk));
  const V = VARIANTS[idx];

  const cycle = (n: number) => {
    const next = VARIANTS[(idx + n + VARIANTS.length) % VARIANTS.length].key;
    const u = new URL(location.href);
    u.searchParams.set("variant", next);
    history.replaceState(null, "", u);
    setVk(next);
  };
  useEffect(() => {
    const onKey = (e: KeyboardEvent) => {
      const t = e.target as HTMLElement;
      if (t.closest("input, textarea, [contenteditable]")) return;
      if (e.key === "ArrowLeft") cycle(-1);
      if (e.key === "ArrowRight") cycle(1);
    };
    window.addEventListener("keydown", onKey);
    return () => window.removeEventListener("keydown", onKey);
  });

  // Celebrate when the day flips to complete.
  const wasComplete = useRef(dayComplete(store.logs, date));
  const lastDate = useRef(date);
  useEffect(() => {
    const now = dayComplete(store.logs, date);
    if (lastDate.current === date && now && !wasComplete.current) fx.confetti();
    wasComplete.current = now;
    lastDate.current = date;
  }, [store.logs, date]);

  const onTap = (h: Habit, e: Pt) => {
    if (!editable(date)) return;
    const cur = value(store.logs, date, h.id);
    const next = tapValue(h, cur);
    if (next > cur) fx.spark(e.clientX, e.clientY, h.color);
    store.setValue(date, h.id, next);
  };
  const onLong = (h: Habit) => editable(date) && setSheet(h);

  return (
    <>
      <div className="stage">
        <div className="phone" key={V.key}>
          <div className="scroller"><V.C date={date} setDate={setDate} store={store} onTap={onTap} onLong={onLong} /></div>
          <ValueSheet
            habit={sheet}
            value={sheet ? value(store.logs, date, sheet.id) : 0}
            onClose={() => setSheet(null)}
            onSave={(v) => { if (sheet) store.setValue(date, sheet.id, v); setSheet(null); }}
          />
        </div>
      </div>
      <div className="proto-switcher">
        <button onClick={() => cycle(-1)} aria-label="Variante précédente">←</button>
        <span>{V.key} · {V.name}</span>
        <button onClick={() => cycle(1)} aria-label="Variante suivante">→</button>
      </div>
      <FxCanvas />
    </>
  );
}

createRoot(document.getElementById("root")!).render(<StrictMode><App /></StrictMode>);
