import { describe, expect, it } from "vitest";
import { contrast, mix } from "./color";
import { themeCss } from "./css";
import { accentIds, themeIds, themes, type Palette } from "./tokens";

// WCAG AA: 4.5 for body text. Muted text (descriptions, subtitles) is body text too.
const AA = 4.5;

function variants() {
  return themeIds.flatMap((t) =>
    accentIds.flatMap((a) =>
      Object.entries(themes[t].palettes(a)).map(([name, p]) => ({
        label: `${t} × ${a} (${name})`,
        p: p as Palette,
      })),
    ),
  );
}

/** The color of a semi-transparent surface seen over one background color. */
function surfaceOver(p: Palette, bg: string): string {
  return mix(p.surface, bg, p.surfaceAlpha);
}

describe("theme tokens", () => {
  it.each(variants())("$label: text on the background is readable", ({ p }) => {
    for (const bg of p.background) {
      expect(contrast(p.text, bg)).toBeGreaterThanOrEqual(AA);
      expect(contrast(p.muted, bg)).toBeGreaterThanOrEqual(AA);
    }
  });

  it.each(variants())("$label: text on the cards is readable", ({ p }) => {
    for (const bg of p.background) {
      const surface = surfaceOver(p, bg);
      expect(contrast(p.text, surface)).toBeGreaterThanOrEqual(AA);
      expect(contrast(p.muted, surface)).toBeGreaterThanOrEqual(AA);
    }
  });

  it.each(variants())("$label: text on the accent is readable", ({ p }) => {
    expect(contrast(p.accentText, p.accent)).toBeGreaterThanOrEqual(AA);
  });

  it("generates a CSS rule for every theme and accent", () => {
    const css = themeCss();
    for (const t of themeIds) {
      for (const a of accentIds) {
        expect(css).toContain(`[data-theme="${t}"][data-accent="${a}"]{`);
      }
    }
    // zazi follows the system: its dark variant sits in a media query.
    expect(css).toContain('@media (prefers-color-scheme: dark){[data-theme="zazi"]');
  });
});
