import { m } from "motion/react";
import { useEffect, useState, type ComponentType } from "react";
import type { AccentId, ThemeId } from "@/themes/tokens";

export interface ThemeBackgroundProps {
  accent: AccentId;
}

type Loader = () => Promise<{ default: ComponentType<ThemeBackgroundProps> }>;

// One module per animated background: a page only downloads its own, after its
// content is on screen. Themes missing here (zazi) have a static background only.
const loaders: Partial<Record<ThemeId, Loader>> = {
  nuit: () => import("./nuit"),
  papier: () => import("./papier"),
  neon: () => import("./neon"),
  pop: () => import("./pop"),
};

/**
 * Whether an animated background may run: WebGL 2 is available, and the visitor
 * asked neither for less motion nor to save data. Otherwise the CSS background of
 * the theme stays alone.
 */
export function animationAllowed(env: {
  webgl2: boolean;
  reducedMotion: boolean;
  saveData: boolean;
}): boolean {
  return env.webgl2 && !env.reducedMotion && !env.saveData;
}

function currentEnv() {
  const connection = (navigator as Navigator & { connection?: { saveData?: boolean } }).connection;
  return {
    webgl2: typeof WebGL2RenderingContext !== "undefined",
    reducedMotion: window.matchMedia("(prefers-reduced-motion: reduce)").matches,
    saveData: connection?.saveData === true,
  };
}

/** Calls fn when the browser is idle (or soon, where requestIdleCallback is missing). */
function whenIdle(fn: () => void): () => void {
  if ("requestIdleCallback" in window) {
    const id = window.requestIdleCallback(fn, { timeout: 1500 });
    return () => window.cancelIdleCallback(id);
  }
  const id = setTimeout(fn, 200);
  return () => clearTimeout(id);
}

/**
 * Loads and shows the animated background of a theme, fading it in over the CSS
 * background. The background components pause themselves when hidden or off
 * screen, and size themselves on their container (the admin preview frame works).
 */
export function AnimatedBackground({ theme, accent }: { theme: ThemeId; accent: AccentId }) {
  const [Background, setBackground] = useState<ComponentType<ThemeBackgroundProps> | null>(null);

  useEffect(() => {
    setBackground(null);
    const load = loaders[theme];
    if (!load || !animationAllowed(currentEnv())) return;
    let cancelled = false;
    const cancelIdle = whenIdle(() => {
      load()
        .then((mod) => {
          // A function given to a state setter is called by React: wrap the component.
          if (!cancelled) setBackground(() => mod.default);
        })
        .catch((err: unknown) => console.error("animated background failed to load", err));
    });
    return () => {
      cancelled = true;
      cancelIdle();
    };
  }, [theme]);

  if (!Background) return null;
  return (
    <m.div
      className="absolute inset-0"
      initial={{ opacity: 0 }}
      animate={{ opacity: 1 }}
      transition={{ duration: 1.2, ease: "easeOut" }}
    >
      <Background accent={accent} />
    </m.div>
  );
}
