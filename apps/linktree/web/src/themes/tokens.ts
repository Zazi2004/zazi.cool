// Single definition of the themes' colors. From it come the CSS variables (see
// css.ts, built into the bundle by a Vite plugin), the contrast tests and the
// colors of the animated backgrounds. The identifiers must match the Go list
// (internal/pages): the contract test checks it.
import { mix, readableOn, rgba } from "./color";

export const themeIds = ["zazi", "nuit", "papier", "neon", "pop"] as const;
export type ThemeId = (typeof themeIds)[number];

export const accentIds = [
  "violet",
  "bleu",
  "cyan",
  "vert",
  "ambre",
  "orange",
  "rose",
  "neutre",
] as const;
export type AccentId = (typeof accentIds)[number];

/** Each accent has a deep shade (for light themes) and a bright one (for dark themes). */
export const accents: Record<AccentId, { label: string; deep: string; bright: string }> = {
  violet: { label: "Violet", deep: "#6d28d9", bright: "#a78bfa" },
  bleu: { label: "Bleu", deep: "#1d4ed8", bright: "#60a5fa" },
  cyan: { label: "Cyan", deep: "#0e7490", bright: "#22d3ee" },
  vert: { label: "Vert", deep: "#15803d", bright: "#4ade80" },
  ambre: { label: "Ambre", deep: "#b45309", bright: "#fbbf24" },
  orange: { label: "Orange", deep: "#c2410c", bright: "#fb923c" },
  rose: { label: "Rose", deep: "#be185d", bright: "#f472b6" },
  neutre: { label: "Neutre", deep: "#3f3f46", bright: "#d4d4d8" },
};

/** The colors of one theme variant (light or dark) for one accent. */
export interface Palette {
  scheme: "light" | "dark";
  /**
   * Colors the background goes through (gradient stops, or the colors of the
   * animated background as seen through its opacity). Text drawn directly on the
   * background must be readable on every one of them.
   */
  background: string[];
  text: string;
  muted: string;
  /** Card color and opacity: links and groups sit on it. */
  surface: string;
  surfaceAlpha: number;
  border: string;
  accent: string;
  accentText: string;
}

export interface Theme {
  id: ThemeId;
  label: string;
  description: string;
  /** Variants: zazi follows the system light/dark mode, the others have one. */
  palettes(accent: AccentId): { light?: Palette; dark?: Palette };
  /** CSS gradient of the static background, from the palette's background colors. */
  gradient(p: Palette): string;
  /**
   * Opacity of the animated background over the base color, if the theme has one.
   * The palette's background colors are computed with it: the contrast tests only
   * hold if the animation is drawn with this exact opacity.
   */
  animatedOpacity?: number;
  /** Opacity of the grain texture over the background. */
  grain: number;
  radius: string;
  fontBody: string;
  fontHeading: string;
}

// Opacity of the animated backgrounds of the dark themes (see Theme.animatedOpacity).
const NUIT_OPACITY = 0.3;
const NEON_OPACITY = 0.25;

const system = 'ui-sans-serif, system-ui, -apple-system, "Segoe UI", sans-serif';

function palette(p: Omit<Palette, "accentText">): Palette {
  return { ...p, accentText: readableOn(p.accent) };
}

export const themes: Record<ThemeId, Theme> = {
  zazi: {
    id: "zazi",
    label: "Zazi",
    description: "Sobre et net, suit le mode clair ou sombre de l'appareil.",
    palettes: (a) => ({
      light: palette({
        scheme: "light",
        background: ["#f7f6f3", mix(accents[a].deep, "#f7f6f3", 0.1)],
        text: "#18181b",
        muted: "#52525b",
        surface: "#ffffff",
        surfaceAlpha: 0.78,
        border: rgba("#18181b", 0.08),
        accent: accents[a].deep,
      }),
      dark: palette({
        scheme: "dark",
        background: ["#0c0c10", mix(accents[a].bright, "#0c0c10", 0.12)],
        text: "#fafafa",
        muted: "#a1a1aa",
        surface: "#ffffff",
        surfaceAlpha: 0.06,
        border: rgba("#ffffff", 0.09),
        accent: accents[a].bright,
      }),
    }),
    gradient: (p) =>
      `radial-gradient(120% 80% at 50% 0%, ${p.background[1]} 0%, ${p.background[0]} 70%)`,
    grain: 0.14,
    radius: "1.1rem",
    fontBody: `"Geist Variable", ${system}`,
    fontHeading: `"Geist Variable", ${system}`,
  },

  nuit: {
    id: "nuit",
    label: "Nuit",
    description: "Sombre, aurore animée et cartes en verre dépoli.",
    palettes: (a) => ({
      // The aurora is drawn at NUIT_OPACITY over the base: these are its extremes.
      dark: palette({
        scheme: "dark",
        background: [
          "#05060a",
          mix(accents[a].bright, "#05060a", NUIT_OPACITY),
          mix("#22d3ee", "#05060a", NUIT_OPACITY * 0.8),
        ],
        text: "#f8fafc",
        muted: "#c9d0de",
        surface: "#ffffff",
        surfaceAlpha: 0.07,
        border: rgba("#ffffff", 0.12),
        accent: accents[a].bright,
      }),
    }),
    gradient: (p) =>
      `radial-gradient(90% 60% at 20% 0%, ${p.background[1]} 0%, transparent 70%), ` +
      `radial-gradient(70% 50% at 90% 10%, ${p.background[2]} 0%, transparent 70%), ${p.background[0]}`,
    animatedOpacity: NUIT_OPACITY,
    grain: 0.1,
    radius: "1.25rem",
    fontBody: `"Space Grotesk Variable", ${system}`,
    fontHeading: `"Space Grotesk Variable", ${system}`,
  },

  papier: {
    id: "papier",
    label: "Papier",
    description: "Clair et chaleureux, typographie à empattements, grain marqué.",
    palettes: (a) => ({
      light: palette({
        scheme: "light",
        background: [
          "#f4efe6",
          mix(accents[a].deep, "#f4efe6", 0.22),
          mix("#f59e0b", "#f4efe6", 0.2),
        ],
        text: "#1c1917",
        muted: "#57534e",
        surface: "#fffdf8",
        surfaceAlpha: 0.92,
        border: rgba("#1c1917", 0.1),
        accent: accents[a].deep,
      }),
    }),
    gradient: (p) =>
      `linear-gradient(160deg, ${p.background[1]} 0%, ${p.background[0]} 45%, ${p.background[2]} 100%)`,
    grain: 0.35,
    radius: "0.9rem",
    fontBody: `"Fraunces Variable", Georgia, serif`,
    fontHeading: `"Fraunces Variable", Georgia, serif`,
  },

  neon: {
    id: "neon",
    label: "Néon",
    description: "Noir profond, lignes lumineuses et bordures qui brillent.",
    palettes: (a) => ({
      // The threads are drawn at NEON_OPACITY over the base.
      dark: palette({
        scheme: "dark",
        background: ["#050507", mix(accents[a].bright, "#050507", NEON_OPACITY)],
        text: "#f5f5f7",
        muted: "#c4c4cf",
        surface: "#0c0c11",
        surfaceAlpha: 0.88,
        border: rgba(accents[a].bright, 0.45),
        accent: accents[a].bright,
      }),
    }),
    gradient: (p) =>
      `radial-gradient(80% 50% at 50% 100%, ${p.background[1]} 0%, ${p.background[0]} 70%)`,
    animatedOpacity: NEON_OPACITY,
    grain: 0.08,
    radius: "0.75rem",
    fontBody: `"Geist Variable", ${system}`,
    fontHeading: `"Unbounded Variable", ${system}`,
  },

  pop: {
    id: "pop",
    label: "Pop",
    description: "Dégradé vif et granuleux, formes géantes, boutons épais.",
    palettes: (a) => ({
      light: palette({
        scheme: "light",
        background: [mix(accents[a].deep, "#ffffff", 0.45), "#fde68a", "#a5f3fc"],
        text: "#111111",
        muted: "#2e2e33",
        surface: "#ffffff",
        surfaceAlpha: 1,
        border: "#111111",
        accent: accents[a].deep,
      }),
    }),
    gradient: (p) =>
      `linear-gradient(135deg, ${p.background[0]} 0%, ${p.background[1]} 55%, ${p.background[2]} 100%)`,
    grain: 0.3,
    radius: "1.5rem",
    fontBody: `"Bricolage Grotesque Variable", ${system}`,
    fontHeading: `"Bricolage Grotesque Variable", ${system}`,
  },
};

export function isThemeId(s: string): s is ThemeId {
  return (themeIds as readonly string[]).includes(s);
}

export function isAccentId(s: string): s is AccentId {
  return (accentIds as readonly string[]).includes(s);
}
