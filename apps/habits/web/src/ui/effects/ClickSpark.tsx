import { useEffect, useRef, type PointerEvent, type ReactNode } from "react";
import { prefersReducedMotion } from "./motion";

type Spark = { x: number; y: number; angle: number; color: string; start: number };

const sparkCount = 8;
const duration = 450;
const reach = 22;

/**
 * Sparks at the touch point, in the colour of the habit touched (adapted from React Bits
 * ClickSpark). The colour is read from the `--tone` of the closest `.habit`; nothing is
 * drawn when the user prefers reduced motion.
 */
export function ClickSpark({ children }: { children: ReactNode }) {
  const canvas = useRef<HTMLCanvasElement>(null);
  const sparks = useRef<Spark[]>([]);
  const frame = useRef(0);

  useEffect(() => {
    const el = canvas.current;
    if (!el) return;
    const resize = () => {
      el.width = window.innerWidth;
      el.height = window.innerHeight;
    };
    resize();
    window.addEventListener("resize", resize);
    return () => {
      window.removeEventListener("resize", resize);
      cancelAnimationFrame(frame.current);
    };
  }, []);

  function draw(now: number) {
    const el = canvas.current;
    const ctx = el?.getContext("2d");
    if (!el || !ctx) return;
    ctx.clearRect(0, 0, el.width, el.height);
    sparks.current = sparks.current.filter((s) => now - s.start < duration);
    for (const s of sparks.current) {
      const t = (now - s.start) / duration;
      const eased = 1 - (1 - t) * (1 - t);
      const from = eased * reach;
      const to = from + (1 - t) * 8;
      ctx.strokeStyle = s.color;
      ctx.globalAlpha = 1 - t;
      ctx.lineWidth = 2;
      ctx.lineCap = "round";
      ctx.beginPath();
      ctx.moveTo(s.x + from * Math.cos(s.angle), s.y + from * Math.sin(s.angle));
      ctx.lineTo(s.x + to * Math.cos(s.angle), s.y + to * Math.sin(s.angle));
      ctx.stroke();
    }
    ctx.globalAlpha = 1;
    if (sparks.current.length > 0) frame.current = requestAnimationFrame(draw);
  }

  function spark(e: PointerEvent) {
    if (prefersReducedMotion()) return;
    const habit = (e.target as HTMLElement).closest<HTMLElement>(".habit-tap");
    if (!habit) return;
    const color = getComputedStyle(habit).getPropertyValue("--tone").trim() || "#7c5cf0";
    const start = performance.now();
    for (let i = 0; i < sparkCount; i++) {
      sparks.current.push({
        x: e.clientX,
        y: e.clientY,
        angle: (2 * Math.PI * i) / sparkCount,
        color,
        start,
      });
    }
    cancelAnimationFrame(frame.current);
    frame.current = requestAnimationFrame(draw);
  }

  return (
    <div onPointerDown={spark}>
      {children}
      <canvas ref={canvas} className="click-spark" aria-hidden="true" />
    </div>
  );
}
