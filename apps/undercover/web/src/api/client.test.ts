import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { ApiError, checkGuess, checkPair, createPair, drawPair, type DrawOptions } from "./client";

const draw: DrawOptions = { extreme: "off", suggestionsOnly: false, exclude: [] };
const json = (body: unknown, status = 200) => new Response(JSON.stringify(body), { status });
const html = (status: number) => new Response("<html>Bad Gateway</html>", { status });

const fetchMock = vi.fn();

beforeEach(() => {
  vi.useFakeTimers();
  vi.stubGlobal("fetch", fetchMock);
});

afterEach(() => {
  vi.useRealTimers();
  vi.unstubAllGlobals();
  fetchMock.mockReset();
});

describe("request retries", () => {
  it("retries a GET once after a network error", async () => {
    fetchMock
      .mockRejectedValueOnce(new TypeError("network"))
      .mockResolvedValueOnce(json({ id: 1 }));
    const p = drawPair(draw);
    await vi.runAllTimersAsync();
    await expect(p).resolves.toEqual({ id: 1 });
    expect(fetchMock).toHaveBeenCalledTimes(2);
  });

  it.each([502, 503, 504])("retries a GET once after a %i", async (status) => {
    fetchMock.mockResolvedValueOnce(html(status)).mockResolvedValueOnce(json({ status: "ok" }));
    const p = checkPair("a", "b");
    await vi.runAllTimersAsync();
    await expect(p).resolves.toEqual({ status: "ok" });
    expect(fetchMock).toHaveBeenCalledTimes(2);
  });

  it("retries checkGuess after a network error", async () => {
    fetchMock
      .mockRejectedValueOnce(new TypeError("network"))
      .mockResolvedValueOnce(json({ correct: true }));
    const p = checkGuess(1, { civilSide: "a", guess: "x" });
    await vi.runAllTimersAsync();
    await expect(p).resolves.toEqual({ correct: true });
  });

  it("gives up after two attempts", async () => {
    fetchMock.mockRejectedValue(new TypeError("network"));
    const p = drawPair(draw);
    const assertion = expect(p).rejects.toThrow("Impossible de joindre le serveur");
    await vi.runAllTimersAsync();
    await assertion;
    expect(fetchMock).toHaveBeenCalledTimes(2);
  });

  it("does not retry non-idempotent requests", async () => {
    fetchMock.mockRejectedValue(new TypeError("network"));
    await expect(createPair({ wordA: "a", wordB: "b" } as never)).resolves.toMatchObject({
      ok: false,
    });
    expect(fetchMock).toHaveBeenCalledTimes(1);

    fetchMock.mockReset().mockImplementation(() => html(502));
    await expect(createPair({ wordA: "a", wordB: "b" } as never)).resolves.toMatchObject({
      ok: false,
    });
    expect(fetchMock).toHaveBeenCalledTimes(1);
  });

  it("does not retry other statuses", async () => {
    fetchMock.mockResolvedValue(json({ error: "nope" }, 404));
    await expect(drawPair(draw)).rejects.toMatchObject({ status: 404, message: "nope" });
    expect(fetchMock).toHaveBeenCalledTimes(1);
  });

  it("stops retrying when aborted during the delay", async () => {
    fetchMock.mockRejectedValue(new TypeError("network"));
    const controller = new AbortController();
    const p = checkPair("a", "b", controller.signal);
    const assertion = expect(p).rejects.toMatchObject({ name: "AbortError" });
    await vi.advanceTimersByTimeAsync(10);
    controller.abort(new DOMException("aborted", "AbortError"));
    await assertion;
    await vi.runAllTimersAsync();
    expect(fetchMock).toHaveBeenCalledTimes(1);
  });
});

describe("request non-JSON bodies", () => {
  it("turns an HTML error page into an ApiError with the status", async () => {
    fetchMock.mockResolvedValue(html(500));
    const err = await drawPair(draw).catch((e: unknown) => e);
    expect(err).toBeInstanceOf(ApiError);
    expect(err).toMatchObject({ status: 500, message: "Erreur 500" });
  });

  it("reports a non-JSON success as a plain error", async () => {
    fetchMock.mockResolvedValue(new Response("<html></html>", { status: 200 }));
    await expect(drawPair(draw)).rejects.toThrow("Réponse inattendue du serveur.");
  });

  it("surfaces the final 502 as an ApiError once retries are spent", async () => {
    fetchMock.mockImplementation(() => html(502));
    const p = drawPair(draw).catch((e: unknown) => e);
    await vi.runAllTimersAsync();
    expect(await p).toMatchObject({ status: 502 });
    expect(fetchMock).toHaveBeenCalledTimes(2);
  });
});
