import { describe, expect, it } from "vitest";
import { fitWithin } from "./image";

describe("fitWithin", () => {
  it("keeps small images as they are", () => {
    expect(fitWithin(800, 600, 1024)).toEqual({ width: 800, height: 600 });
  });

  it("shrinks the largest side and keeps the ratio", () => {
    expect(fitWithin(4032, 3024, 1024)).toEqual({ width: 1024, height: 768 });
    expect(fitWithin(1000, 3000, 1024)).toEqual({ width: 341, height: 1024 });
  });

  it("never returns a zero side", () => {
    expect(fitWithin(5000, 1, 1024)).toEqual({ width: 1024, height: 1 });
  });
});
