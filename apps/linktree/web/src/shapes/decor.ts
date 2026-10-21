import { shapeIds } from "./list";

// Digits look odd as decoration: they are kept for the avatar picker and the 404.
const decorPool = shapeIds.filter((id) => !id.startsWith("number-"));

/** FNV-1a: a tiny, stable string hash. */
function hash(s: string): number {
  let h = 0x811c9dc5;
  for (let i = 0; i < s.length; i++) {
    h ^= s.charCodeAt(i);
    h = Math.imul(h, 0x01000193);
  }
  return h >>> 0;
}

/** mulberry32: a small seeded random generator, returning numbers in [0, 1). */
function random(seed: number): () => number {
  let a = seed;
  return () => {
    a = (a + 0x6d2b79f5) | 0;
    let t = Math.imul(a ^ (a >>> 15), 1 | a);
    t = (t + Math.imul(t ^ (t >>> 7), 61 | t)) ^ t;
    return ((t ^ (t >>> 14)) >>> 0) / 4294967296;
  };
}

/**
 * Picks `count` distinct decoration shapes for a page. The choice depends only on
 * the seed (the page ID), so a page always keeps the same decoration.
 */
export function decorShapes(seed: string, count: number): string[] {
  const next = random(hash(seed));
  const pool = [...decorPool];
  const picked: string[] = [];
  while (picked.length < count && pool.length > 0) {
    picked.push(pool.splice(Math.floor(next() * pool.length), 1)[0]!);
  }
  return picked;
}

/** The shapes spelling a number, e.g. "404" (number-1 is 0, number-10 is 9). */
export function digitShapes(n: string): string[] {
  return [...n].map((d) => `number-${Number(d) + 1}`);
}
