// The weight budgets of NF3, checked on the Vite build (CA25): the bootstrap script
// weighs 5 KiB gzip at most, and each background module 30 KiB gzip at most.

import { gzipSync } from "node:zlib";

export const BOOT_BUDGET = 5 * 1024;
export const MODULE_BUDGET = 30 * 1024;

interface Chunk {
  file: string;
  isEntry?: boolean;
  isDynamicEntry?: boolean;
  imports?: string[];
}

export type Manifest = Record<string, Chunk>;

export interface Measure {
  name: string;
  bytes: number;
  budget: number;
}

/**
 * Gzip size of the bootstrap and of each background module. A module counts with the
 * chunks it imports statically (the code shared between backgrounds), as it needs
 * them to run; the bootstrap does not count the modules it loads later.
 */
export function measure(manifest: Manifest, read: (file: string) => Uint8Array): Measure[] {
  const size = (file: string) => gzipSync(read(file)).length;
  const total = (key: string, seen = new Set<string>()): number => {
    const chunk = manifest[key];
    if (!chunk || seen.has(key)) return 0;
    seen.add(key);
    return size(chunk.file) + (chunk.imports ?? []).reduce((sum, imp) => sum + total(imp, seen), 0);
  };
  const results: Measure[] = [];
  for (const [key, chunk] of Object.entries(manifest)) {
    if (chunk.isEntry) results.push({ name: key, bytes: total(key), budget: BOOT_BUDGET });
    else if (chunk.isDynamicEntry) {
      results.push({ name: key, bytes: total(key), budget: MODULE_BUDGET });
    }
  }
  return results;
}

export function overBudget(measures: Measure[]): Measure[] {
  return measures.filter((m) => m.bytes > m.budget);
}
