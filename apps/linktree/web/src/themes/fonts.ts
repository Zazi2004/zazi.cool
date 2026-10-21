import type { ThemeId } from "./tokens";

// Each import is a separate CSS chunk: a page only downloads the fonts of its theme.
const loaders: Record<ThemeId, () => Promise<unknown>> = {
  zazi: () => import("@fontsource-variable/geist"),
  nuit: () => import("@fontsource-variable/space-grotesk"),
  // The italic is used by the titles.
  papier: () =>
    Promise.all([
      import("@fontsource-variable/fraunces"),
      import("@fontsource-variable/fraunces/wght-italic.css"),
    ]),
  neon: () =>
    Promise.all([import("@fontsource-variable/unbounded"), import("@fontsource-variable/geist")]),
  pop: () => import("@fontsource-variable/bricolage-grotesque"),
};

const loaded = new Map<ThemeId, Promise<unknown>>();

/** Loads the fonts of a theme once. Until then, the system font is shown. */
export function loadThemeFonts(theme: ThemeId): Promise<unknown> {
  let p = loaded.get(theme);
  if (!p) {
    p = loaders[theme]().catch((err: unknown) => console.error("font loading failed", err));
    loaded.set(theme, p);
  }
  return p;
}
