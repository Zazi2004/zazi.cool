import { randomBytes } from "node:crypto";
import { describe, expect, it } from "vitest";
import { BOOT_BUDGET, MODULE_BUDGET, measure, overBudget, type Manifest } from "./budget.ts";

const manifest: Manifest = {
  "src/boot.ts": { file: "assets/boot.js", isEntry: true },
  "src/backgrounds/aurore.ts": {
    file: "assets/aurore.js",
    isDynamicEntry: true,
    imports: ["_gl.js"],
  },
  "_gl.js": { file: "assets/gl.js" },
};

// Random bytes do not compress: the gzip size is about the raw size.
const files = (boot: number, aurore: number, shared: number) => {
  const data: Record<string, Uint8Array> = {
    "assets/boot.js": randomBytes(boot),
    "assets/aurore.js": randomBytes(aurore),
    "assets/gl.js": randomBytes(shared),
  };
  return (file: string) => data[file] ?? new Uint8Array();
};

describe("budget (CA25)", () => {
  it("accepts a build within budget", () => {
    const measures = measure(manifest, files(1000, 10_000, 5000));
    expect(measures).toHaveLength(2);
    expect(overBudget(measures)).toEqual([]);
  });

  it("rejects a bootstrap script over 5 KiB gzip", () => {
    const failures = overBudget(measure(manifest, files(BOOT_BUDGET + 500, 1000, 1000)));
    expect(failures.map((m) => m.name)).toEqual(["src/boot.ts"]);
  });

  it("rejects a module over 30 KiB gzip, counting the chunks it imports", () => {
    const failures = overBudget(measure(manifest, files(1000, MODULE_BUDGET - 1000, 2000)));
    expect(failures.map((m) => m.name)).toEqual(["src/backgrounds/aurore.ts"]);
  });
});
