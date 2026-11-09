import { useEffect, useRef } from "react";
import { prefersReducedMotion } from "./motion";

type Particle = {
  x: number;
  y: number;
  vx: number;
  vy: number;
  life: number;
  max: number;
  color: string;
  kind: "spark" | "confetti";
  rotation: number;
  spin: number;
  width: number;
};

const confettiColors = [
  "#f59e0b",
  "#8b5cf6",
  "#3b82f6",
  "#f43f5e",
  "#10b981",
  "#0ea5a4",
  "#ec4899",
];

// One canvas and one animation loop serve the whole app: effects are fired from anywhere.
const particles: Particle[] = [];
let context: CanvasRenderingContext2D | null = null;
let running = false;

function drawSpark(ctx: CanvasRenderingContext2D, p: Particle) {
  p.x += p.vx;
  p.y += p.vy;
  p.vx *= 0.9;
  p.vy *= 0.9;
  ctx.strokeStyle = p.color;
  ctx.lineWidth = 2.5;
  ctx.lineCap = "round";
  ctx.beginPath();
  ctx.moveTo(p.x, p.y);
  ctx.lineTo(p.x - p.vx * 3, p.y - p.vy * 3);
  ctx.stroke();
}

function drawConfetti(ctx: CanvasRenderingContext2D, p: Particle) {
  p.vy += 0.32;
  p.vx *= 0.99;
  p.x += p.vx;
  p.y += p.vy;
  p.rotation += p.spin;
  ctx.save();
  ctx.translate(p.x, p.y);
  ctx.rotate(p.rotation);
  ctx.fillStyle = p.color;
  ctx.fillRect(-p.width / 2, -p.width / 4, p.width, p.width / 2);
  ctx.restore();
}

function frame() {
  const ctx = context;
  if (!ctx) {
    running = false;
    return;
  }
  ctx.clearRect(0, 0, window.innerWidth, window.innerHeight);
  for (let i = particles.length - 1; i >= 0; i--) {
    const p = particles[i];
    if (!p) continue;
    p.life++;
    if (p.life > p.max) {
      particles.splice(i, 1);
      continue;
    }
    ctx.globalAlpha = 1 - p.life / p.max;
    if (p.kind === "spark") drawSpark(ctx, p);
    else drawConfetti(ctx, p);
  }
  ctx.globalAlpha = 1;
  if (particles.length > 0) requestAnimationFrame(frame);
  else running = false;
}

function start() {
  if (running) return;
  running = true;
  requestAnimationFrame(frame);
}

/** Sparks at a point, in the colour of the habit touched (after React Bits ClickSpark). */
export function spark(x: number, y: number, color: string) {
  if (prefersReducedMotion()) return;
  const count = 10;
  for (let i = 0; i < count; i++) {
    const angle = (i / count) * Math.PI * 2;
    particles.push({
      x,
      y,
      vx: Math.cos(angle) * 6,
      vy: Math.sin(angle) * 6,
      life: 0,
      max: 20,
      color,
      kind: "spark",
      rotation: 0,
      spin: 0,
      width: 0,
    });
  }
  start();
}

/** The burst that celebrates a complete day. */
export function confetti() {
  if (prefersReducedMotion()) return;
  for (let i = 0; i < 160; i++) {
    particles.push({
      x: window.innerWidth / 2 + (Math.random() - 0.5) * 120,
      y: window.innerHeight * 0.4,
      vx: (Math.random() - 0.5) * 16,
      vy: -Math.random() * 15 - 5,
      life: 0,
      max: 110 + Math.random() * 40,
      color: confettiColors[i % confettiColors.length] ?? "#6366f1",
      kind: "confetti",
      rotation: Math.random() * 6,
      spin: (Math.random() - 0.5) * 0.4,
      width: 8 + Math.random() * 6,
    });
  }
  start();
}

/** The full-screen canvas the effects draw on; mounted once, above everything. */
export function FxCanvas() {
  const canvas = useRef<HTMLCanvasElement>(null);

  useEffect(() => {
    const el = canvas.current;
    const ctx = el?.getContext("2d");
    if (!el || !ctx) return;
    const resize = () => {
      const ratio = window.devicePixelRatio || 1;
      el.width = window.innerWidth * ratio;
      el.height = window.innerHeight * ratio;
      ctx.setTransform(ratio, 0, 0, ratio, 0, 0);
    };
    resize();
    context = ctx;
    window.addEventListener("resize", resize);
    return () => {
      window.removeEventListener("resize", resize);
      context = null;
    };
  }, []);

  return <canvas ref={canvas} className="fx-canvas" aria-hidden="true" />;
}
