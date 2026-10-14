import { describe, expect, it } from "vitest";
import goContract from "./contract.json";
import type { Contract } from "./types";

// Every field set, optional ones included (`Required`), so that a field missing on
// either side shows up as a difference.
type Complete<T> = { [K in keyof T]-?: Required<T[K]> };

const view = { id: 1, wordA: "Chat", wordB: "Tigre", extreme: false };
const tsContract: Complete<Contract> = {
  pairView: view,
  drawResponse: {
    id: 1,
    civilWord: "Tigre",
    undercoverWord: "Chat",
    civilSide: "b",
    extreme: false,
  },
  guessRequest: { civilSide: "b", guess: "tigre" },
  guessResponse: { correct: true, expected: "Tigre" },
  checkResponse: {
    status: "duplicate",
    error: "message",
    duplicate: view,
    similar: [view],
    related: [view],
  },
  createPairRequest: {
    wordA: "Chat",
    wordB: "Tigre",
    extreme: true,
    confirmSimilar: true,
    website: "",
  },
  createPairResponse: { pair: view },
  reportRequest: { reason: "extreme" },
  errorResponse: { error: "message" },
};

/** Replaces every value by its type, keeping the structure and the field names. */
function shape(value: unknown): unknown {
  if (Array.isArray(value)) return value.map(shape);
  if (value !== null && typeof value === "object") {
    return Object.fromEntries(
      Object.entries(value)
        .sort(([a], [b]) => a.localeCompare(b))
        .map(([k, v]) => [k, shape(v)]),
    );
  }
  return typeof value;
}

describe("API contract", () => {
  it("matches the Go message types (contract.json)", () => {
    expect(shape(tsContract)).toEqual(shape(goContract));
  });

  it("uses values allowed by the TypeScript unions", () => {
    // contract.json comes from Go: its values must be valid for the TS types.
    expect(tsContract).toEqual(goContract);
  });
});
