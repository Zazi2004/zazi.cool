import { afterEach, describe, expect, it, vi } from "vitest";
import {
  ApiError,
  type DrawOptions,
  NetworkError,
  REQUEST_TIMEOUT_MS,
  checkGuess,
  drawPair,
  reportPair,
  toUiError,
} from "./client";

const draw: DrawOptions = { extreme: "off", suggestionsOnly: false, exclude: [] };

function respond(status: number, body: unknown = {}) {
  return new Response(JSON.stringify(body), { status });
}

afterEach(() => {
  vi.unstubAllGlobals();
  vi.useRealTimers();
});

describe("network errors", () => {
  it("reports an unreachable server", async () => {
    vi.stubGlobal("navigator", { onLine: true });
    vi.stubGlobal("fetch", vi.fn().mockRejectedValue(new TypeError("Failed to fetch")));
    const err = await drawPair(draw).catch((e: unknown) => e);
    expect(err).toBeInstanceOf(NetworkError);
    expect(toUiError(err)).toEqual({
      message: "Impossible de joindre le serveur. Vérifiez la connexion.",
      retryable: true,
    });
  });

  it("reports offline distinctly", async () => {
    vi.stubGlobal("navigator", { onLine: false });
    vi.stubGlobal("fetch", vi.fn().mockRejectedValue(new TypeError("Failed to fetch")));
    const err = await reportPair(1, "remove").catch((e: unknown) => e);
    expect(toUiError(err)).toEqual({
      message: "Vous êtes hors ligne. Vérifiez votre connexion.",
      retryable: true,
    });
  });

  it("reports a timeout and does not retry it", async () => {
    vi.useFakeTimers();
    vi.stubGlobal("navigator", { onLine: true });
    const fetchMock = vi.fn(
      (_path: string, init: RequestInit) =>
        new Promise<Response>((_, reject) => {
          init.signal?.addEventListener("abort", () => reject(new DOMException("", "AbortError")));
        }),
    );
    vi.stubGlobal("fetch", fetchMock);
    const result = checkGuess(1, { civilSide: "a", guess: "chat" }).catch((e: unknown) => e);
    await vi.advanceTimersByTimeAsync(REQUEST_TIMEOUT_MS);
    const err = await result;
    expect(err).toBeInstanceOf(NetworkError);
    expect((err as NetworkError).kind).toBe("timeout");
    expect(fetchMock).toHaveBeenCalledTimes(1);
  });
});

describe("HTTP errors", () => {
  it("reports 5xx as retryable", () => {
    const ui = toUiError(new ApiError(503, undefined));
    expect(ui.retryable).toBe(true);
    expect(ui.message).toContain("503");
  });

  it("reports 429 as too many requests, retryable", () => {
    expect(toUiError(new ApiError(429, { error: "rate limited" }))).toEqual({
      message: "Trop de requêtes. Patientez quelques secondes puis réessayez.",
      retryable: true,
    });
  });

  it("keeps the server message for other 4xx and does not offer a retry", async () => {
    vi.stubGlobal("fetch", vi.fn().mockResolvedValue(respond(404, { error: "Aucune paire" })));
    const err = await drawPair(draw).catch((e: unknown) => e);
    expect(toUiError(err)).toEqual({ message: "Aucune paire", retryable: false });
  });
});
