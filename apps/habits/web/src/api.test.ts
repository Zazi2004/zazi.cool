import { afterEach, describe, expect, it, vi } from "vitest";
import { ApiError, hasSession, login } from "./api";

function stubFetch(response: Response | Error) {
  const fetchMock = vi.fn(async () => {
    if (response instanceof Error) throw response;
    return response;
  });
  vi.stubGlobal("fetch", fetchMock);
  return fetchMock;
}

afterEach(() => vi.unstubAllGlobals());

describe("hasSession", () => {
  it("is true with a session", async () => {
    stubFetch(Response.json({ authenticated: true }));
    expect(await hasSession()).toBe(true);
  });

  it("is false on 401", async () => {
    stubFetch(Response.json({ error: "unauthorized" }, { status: 401 }));
    expect(await hasSession()).toBe(false);
  });

  it("throws on other failures", async () => {
    stubFetch(new Response("boom", { status: 500 }));
    await expect(hasSession()).rejects.toMatchObject({ status: 500 });
  });
});

describe("login", () => {
  it("posts the password as JSON", async () => {
    const fetchMock = stubFetch(new Response(null, { status: 204 }));
    await login("secret");
    const [path, init] = fetchMock.mock.calls[0] as unknown as [string, RequestInit];
    expect(path).toBe("/api/login");
    expect(init.method).toBe("POST");
    expect(new Headers(init.headers).get("Content-Type")).toBe("application/json");
    expect(init.body).toBe(JSON.stringify({ password: "secret" }));
  });

  it("exposes the server message on a wrong password", async () => {
    stubFetch(
      Response.json({ error: "bad_password", message: "Mot de passe incorrect." }, { status: 401 }),
    );
    const err = await login("nope").catch((e: unknown) => e);
    expect(err).toBeInstanceOf(ApiError);
    expect(err).toMatchObject({
      status: 401,
      code: "bad_password",
      message: "Mot de passe incorrect.",
    });
  });

  it("reports a network failure", async () => {
    stubFetch(new TypeError("Failed to fetch"));
    await expect(login("x")).rejects.toMatchObject({ status: 0, code: "network" });
  });
});
