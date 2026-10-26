// Fails the build when the bootstrap script or a background module is over budget
// (NF3, CA25). Run by `npm run build` after Vite, so `docker build` stops too.
import { readFileSync } from "node:fs";
import { measure, overBudget, type Manifest } from "../src/budget.ts";

const dist = new URL("../dist/", import.meta.url);
const manifest: Manifest = JSON.parse(readFileSync(new URL(".vite/manifest.json", dist), "utf8"));
const measures = measure(manifest, (file) => readFileSync(new URL(file, dist)));

const kib = (n: number) => `${(n / 1024).toFixed(1)} KiB`;
for (const m of measures) console.log(`${m.name}: ${kib(m.bytes)} gzip (budget ${kib(m.budget)})`);

const failures = overBudget(measures);
if (failures.length) {
  console.error(`Bundle budget exceeded: ${failures.map((m) => m.name).join(", ")}`);
  process.exit(1);
}
