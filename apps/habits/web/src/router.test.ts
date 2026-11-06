import { describe, expect, it } from "vitest";
import { matchRoute } from "./router";

describe("matchRoute", () => {
  it.each([
    ["/", { name: "today", date: null }],
    ["/day/2026-11-06", { name: "today", date: "2026-11-06" }],
    ["/stats", { name: "stats" }],
    ["/stats/12", { name: "statsHabit", id: 12 }],
    ["/habits", { name: "habits" }],
    ["/habits/", { name: "habits" }],
    ["/login", { name: "login" }],
  ])("%s", (path, want) => {
    expect(matchRoute(path)).toEqual(want);
  });

  it.each(["/day", "/day/yesterday", "/stats/abc", "/stats/0", "/stats/1/2", "/nope"])(
    "%s is not found",
    (path) => {
      expect(matchRoute(path)).toEqual({ name: "notFound" });
    },
  );
});
