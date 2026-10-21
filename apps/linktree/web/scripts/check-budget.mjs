// Fails the build when the public bundle exceeds its budget: 150 KB gzip of
// JavaScript loaded initially by the public entry, 30 KB gzip for each animated
// background chunk (see docs/specs/linktree.md).
import { readFileSync } from "node:fs";
import { gzipSync } from "node:zlib";

const dist = new URL("../dist/", import.meta.url);
const manifest = JSON.parse(readFileSync(new URL(".vite/manifest.json", dist), "utf8"));
const gzip = (file) => gzipSync(readFileSync(new URL(file, dist))).length;
const kb = (n) => `${(n / 1024).toFixed(1)} KB`;

// Initial JS of the public entry: the entry and its static imports, recursively.
const seen = new Set();
function initial(key) {
  if (seen.has(key)) return 0;
  seen.add(key);
  const chunk = manifest[key];
  return gzip(chunk.file) + (chunk.imports ?? []).reduce((sum, imp) => sum + initial(imp), 0);
}

const failures = [];
const publicSize = initial("src/public/main.tsx");
console.log(`public initial JS: ${kb(publicSize)} gzip (budget 150 KB)`);
if (publicSize > 150 * 1024) failures.push("public initial JS over 150 KB");

for (const [key, chunk] of Object.entries(manifest)) {
  if (!key.startsWith("src/backgrounds/") || key.endsWith("AnimatedBackground.tsx")) continue;
  // Only the background's own code: shared chunks are already counted above.
  const size = gzip(chunk.file);
  console.log(`${key}: ${kb(size)} gzip (budget 30 KB)`);
  if (size > 30 * 1024) failures.push(`${key} over 30 KB`);
}

if (failures.length) {
  console.error("Bundle budget exceeded: " + failures.join(", "));
  process.exit(1);
}
