import { useEffect, useMemo, type CSSProperties } from "react";

const pieces = 28;
const palette = ["#7c5cf0", "#2f7cf6", "#2f9e5f", "#d98a00", "#e5559b", "#e5484d"];

/** A short burst of confetti, in CSS only; `onDone` fires when it has played. */
export function Confetti({ onDone }: { onDone: () => void }) {
  const items = useMemo(
    () =>
      Array.from({ length: pieces }, (_, i) => ({
        id: i,
        x: Math.round((Math.random() - 0.5) * 320),
        y: -Math.round(120 + Math.random() * 220),
        turn: Math.round((Math.random() - 0.5) * 900),
        delay: Math.round(Math.random() * 120),
        color: palette[i % palette.length] ?? "#7c5cf0",
      })),
    [],
  );

  useEffect(() => {
    const id = window.setTimeout(onDone, 1800);
    return () => window.clearTimeout(id);
  }, [onDone]);

  return (
    <div className="confetti" aria-hidden="true">
      {items.map((p) => (
        <span
          key={p.id}
          style={
            {
              "--x": `${p.x}px`,
              "--y": `${p.y}px`,
              "--turn": `${p.turn}deg`,
              "--delay": `${p.delay}ms`,
              "--color": p.color,
            } as CSSProperties
          }
        />
      ))}
    </div>
  );
}
