// Color conversions shared by the shaders' palettes. Adapted from the React Bits
// components of the linktree (see the shader files); colors come from backgrounds.json
// as #rrggbb, so no CSS parsing is needed.

export type Vec3 = [number, number, number];

export function hexToRgb(hex: string): Vec3 {
  const n = parseInt(hex.slice(1), 16);
  return [(n >> 16) / 255, ((n >> 8) & 255) / 255, (n & 255) / 255];
}

const toLinear = (c: number) => (c <= 0.04045 ? c / 12.92 : ((c + 0.055) / 1.055) ** 2.4);

/** sRGB (0–1) to Oklab. */
export function toOklab([r, g, b]: Vec3): Vec3 {
  const lr = toLinear(r);
  const lg = toLinear(g);
  const lb = toLinear(b);
  const l = Math.cbrt(0.4122214708 * lr + 0.5363325363 * lg + 0.0514459929 * lb);
  const m = Math.cbrt(0.2119034982 * lr + 0.6806995451 * lg + 0.1073969566 * lb);
  const s = Math.cbrt(0.0883024619 * lr + 0.2817188376 * lg + 0.6299787005 * lb);
  return [
    0.2104542553 * l + 0.793617785 * m - 0.0040720468 * s,
    1.9779984951 * l - 2.428592205 * m + 0.4505937099 * s,
    0.0259040371 * l + 0.7827717662 * m - 0.808675766 * s,
  ];
}

/** Oklab to lightness, chroma, hue (radians). */
export function toOklch(rgb: Vec3): Vec3 {
  const [l, a, b] = toOklab(rgb);
  return [l, Math.hypot(a, b), Math.atan2(b, a)];
}

/**
 * Makes the hues of a ramp continuous (no jump of 2π between two stops), so that
 * mixing two stops goes the short way round the color wheel.
 */
export function unwrapHues(stops: Vec3[]): Vec3[] {
  const known = stops.find((stop) => stop[1] > 0.02);
  let previous = known ? known[2] : 0;
  return stops.map(([l, c, h]) => {
    let hue = c > 0.02 ? h : previous;
    while (hue - previous > Math.PI) hue -= Math.PI * 2;
    while (hue - previous < -Math.PI) hue += Math.PI * 2;
    previous = hue;
    return [l, c, hue];
  });
}
