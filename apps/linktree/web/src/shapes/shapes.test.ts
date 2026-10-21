import { readdirSync } from "node:fs";
import { describe, expect, it } from "vitest";
import { decorShapes, digitShapes } from "./decor";
import { shapeIds } from "./list";

describe("shapes", () => {
  it("list.ts matches the files of public/shapes", () => {
    const files = readdirSync(new URL("../../public/shapes/", import.meta.url))
      .filter((f) => f.endsWith(".svg"))
      .map((f) => f.slice(0, -4));
    expect([...shapeIds].sort()).toEqual(files.sort());
  });

  it("picks the same decoration for the same page", () => {
    const a = decorShapes("0199e6a8-0000-7000-8000-000000000001", 3);
    expect(a).toEqual(decorShapes("0199e6a8-0000-7000-8000-000000000001", 3));
    expect(new Set(a).size).toBe(3);
    expect(a.every((id) => shapeIds.includes(id) && !id.startsWith("number-"))).toBe(true);
    expect(decorShapes("another page", 3)).not.toEqual(a);
  });

  it("spells numbers with digit shapes", () => {
    expect(digitShapes("404")).toEqual(["number-5", "number-1", "number-5"]);
  });
});
