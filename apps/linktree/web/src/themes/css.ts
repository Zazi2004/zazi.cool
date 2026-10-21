// Turns the theme tokens into CSS variables, scoped by the data-theme and
// data-accent attributes (on <html> for the public pages, on the preview frame in
// the admin). The Vite plugin "theme-tokens" (vite.config.ts) puts the result in
// the CSS bundle, so the colors are there before any JavaScript runs.
import { mix, rgba } from "./color";
import { accentIds, themeIds, themes, type Palette, type Theme } from "./tokens";

function paletteVars(theme: Theme, p: Palette): string {
  const hover =
    p.scheme === "dark"
      ? rgba(p.surface, Math.min(1, p.surfaceAlpha + 0.06))
      : rgba(mix(p.text, p.surface, 0.05), p.surfaceAlpha);
  const vars: Record<string, string> = {
    "color-scheme": p.scheme,
    "--lt-bg": theme.gradient(p),
    "--lt-bg-base": p.background[0]!,
    "--lt-text": p.text,
    "--lt-muted": p.muted,
    "--lt-surface": rgba(p.surface, p.surfaceAlpha),
    "--lt-surface-strong": hover,
    "--lt-border": p.border,
    "--lt-accent": p.accent,
    "--lt-accent-text": p.accentText,
    "--lt-accent-soft": rgba(p.accent, 0.16),
  };
  return Object.entries(vars)
    .map(([k, v]) => `${k}:${v};`)
    .join("");
}

export function themeCss(): string {
  const rules: string[] = [];
  for (const id of themeIds) {
    const theme = themes[id];
    rules.push(
      `[data-theme="${id}"]{--lt-radius:${theme.radius};--lt-grain:${theme.grain};` +
        `--lt-font-body:${theme.fontBody};--lt-font-heading:${theme.fontHeading};}`,
    );
    for (const accent of accentIds) {
      const sel = `[data-theme="${id}"][data-accent="${accent}"]`;
      const { light, dark } = theme.palettes(accent);
      // A theme with both variants follows the system preference.
      const base = light ?? dark;
      if (base) rules.push(`${sel}{${paletteVars(theme, base)}}`);
      if (light && dark) {
        rules.push(`@media (prefers-color-scheme: dark){${sel}{${paletteVars(theme, dark)}}}`);
      }
    }
  }
  return rules.join("\n");
}
