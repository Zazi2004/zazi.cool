// Non-regression suite for ONBO-52 ("erreur de connexion au serveur"): every request the
// UI makes is replayed against the same network failures. Each journey goes through the
// function the screen really calls, with `fetch` mocked.
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { ApiError, checkGuess, checkPair, createPair, reportPair } from "./client";
import { defaultSettings } from "../game/engine";
import { newGame } from "../ui/newGame";

const store = new Map<string, string>();
vi.stubGlobal("localStorage", {
  getItem: (k: string) => store.get(k) ?? null,
  setItem: (k: string, v: string) => void store.set(k, v),
  removeItem: (k: string) => void store.delete(k),
});

const fetchMock = vi.fn();
const json = (body: unknown, status = 200) => new Response(JSON.stringify(body), { status });
const html = (status: number) => new Response("<html>error page</html>", { status });
const names = ["Ana", "Ben", "Cyd", "Dan", "Eve"];
const UNREACHABLE = "Impossible de joindre le serveur. Vérifiez la connexion.";

interface Journey {
  name: string;
  /** Screen action, as the UI triggers it. */
  run: () => Promise<unknown>;
  /** Body of a successful answer. */
  ok: unknown;
  /** Checks what the action resolves to on success. */
  expectOk: (result: unknown) => void;
  /** True when the client repeats the request itself (safe to repeat). */
  retried: boolean;
  /** Maps whatever the action does with a failure to the error the screen would show. */
  failure: (run: () => Promise<unknown>) => Promise<unknown>;
}

const thrown = (run: () => Promise<unknown>) =>
  run().then(
    () => undefined,
    (e: unknown) => e,
  );
const returnedError = async (run: () => Promise<unknown>) => {
  const result = (await run()) as { ok: boolean; error?: string; check?: unknown };
  return result.ok ? undefined : result;
};

const pair = { id: 7, civilWord: "Chat", undercoverWord: "Tigre", civilSide: "a", extreme: false };

const journeys: Journey[] = [
  {
    // "Rejouer" (OverScreen) calls the very same newGame() with the saved settings.
    name: "start a game / Rejouer",
    run: () => newGame(defaultSettings, names),
    ok: pair,
    expectOk: (game) => expect(game).toMatchObject({ pair: { id: 7 }, phase: expect.anything() }),
    retried: true,
    failure: thrown,
  },
  {
    name: "Mr. White guess",
    run: () => checkGuess(7, { civilSide: "a", guess: "chat" }),
    ok: { correct: true },
    expectOk: (r) => expect(r).toEqual({ correct: true }),
    retried: true,
    failure: thrown,
  },
  {
    name: "live redundancy check while proposing a pair",
    run: () => checkPair("chat", "tigre"),
    ok: { status: "ok" },
    expectOk: (r) => expect(r).toEqual({ status: "ok" }),
    retried: true,
    failure: thrown,
  },
  {
    // Not repeated automatically: the user decides (button), the server counts reports.
    name: "report a pair",
    run: () => reportPair(7, "remove"),
    ok: undefined,
    expectOk: (r) => expect(r).toBeUndefined(),
    retried: false,
    failure: thrown,
  },
  {
    // Not repeated automatically: it would publish a pair twice on a lost answer.
    name: "propose a pair",
    run: () => createPair({ wordA: "chat", wordB: "tigre", extreme: false } as never),
    ok: { id: 9 },
    expectOk: (r) => expect(r).toMatchObject({ ok: true }),
    retried: false,
    failure: returnedError,
  },
];

function answer(ok: unknown) {
  return ok === undefined ? new Response(null, { status: 204 }) : json(ok);
}

beforeEach(() => {
  vi.useFakeTimers();
  vi.stubGlobal("fetch", fetchMock);
});

afterEach(() => {
  vi.useRealTimers();
  fetchMock.mockReset();
  store.clear();
});

/** Runs the action to completion, letting retry delays elapse. */
async function settle<T>(p: Promise<T>): Promise<{ value?: T; error?: unknown }> {
  const outcome = p.then(
    (value): { value?: T; error?: unknown } => ({ value }),
    (error: unknown): { value?: T; error?: unknown } => ({ error }),
  );
  await vi.runAllTimersAsync();
  return outcome;
}

describe.each(journeys)("$name", (j) => {
  it("network cut, then success", async () => {
    fetchMock
      .mockRejectedValueOnce(new TypeError("Failed to fetch"))
      .mockResolvedValue(answer(j.ok));
    const first = await settle(j.failure(j.run));
    if (j.retried) {
      expect(first.error).toBeUndefined();
      expect(fetchMock).toHaveBeenCalledTimes(2);
      return;
    }
    // One attempt only, a clear message, and the user's retry works.
    expect(fetchMock).toHaveBeenCalledTimes(1);
    const shown = first.value as { error?: string } | Error | undefined;
    expect(shown instanceof Error ? shown.message : shown?.error).toBe(UNREACHABLE);
    j.expectOk((await settle(j.run())).value);
    expect(fetchMock).toHaveBeenCalledTimes(2);
  });

  it("502 then success", async () => {
    fetchMock.mockResolvedValueOnce(html(502)).mockResolvedValue(answer(j.ok));
    const first = await settle(j.failure(j.run));
    if (j.retried) {
      expect(first.error).toBeUndefined();
      expect(fetchMock).toHaveBeenCalledTimes(2);
      return;
    }
    expect(fetchMock).toHaveBeenCalledTimes(1);
    j.expectOk((await settle(j.run())).value);
  });

  it("HTML body never leaks a JSON parse error", async () => {
    fetchMock.mockImplementation(() => html(500));
    const { value, error } = await settle(j.failure(j.run));
    expect(error).toBeUndefined(); // failure() already caught it
    const shown = value as Error | { error?: string } | undefined;
    const message = shown instanceof Error ? shown.message : (shown?.error ?? "");
    expect(message).not.toMatch(/JSON|Unexpected token|SyntaxError/i);
    if (shown instanceof ApiError) expect(shown.status).toBe(500);
    expect(message).not.toBe("");
  });

  it("HTML body on a 200 gives a plain message", async () => {
    fetchMock.mockImplementation(() => new Response("<html></html>", { status: 200 }));
    const { value } = await settle(j.failure(j.run));
    const shown = value as Error | { error?: string };
    const message = shown instanceof Error ? shown.message : (shown.error ?? "");
    expect(message).toBe("Réponse inattendue du serveur.");
  });

  it("429 is not retried and keeps its status", async () => {
    fetchMock.mockImplementation(() => json({ error: "Trop de requêtes" }, 429));
    const { value } = await settle(j.failure(j.run));
    expect(fetchMock).toHaveBeenCalledTimes(1);
    const shown = value as Error | { error?: string };
    if (shown instanceof ApiError) expect(shown.status).toBe(429);
    expect(shown instanceof Error ? shown.message : shown.error).toBe("Trop de requêtes");
  });

  it("429 with an HTML body still reports the status", async () => {
    fetchMock.mockImplementation(() => html(429));
    const { value } = await settle(j.failure(j.run));
    expect(fetchMock).toHaveBeenCalledTimes(1);
    const shown = value as Error | { error?: string };
    expect(shown instanceof Error ? shown.message : shown.error).toBe("Erreur 429");
  });
});
