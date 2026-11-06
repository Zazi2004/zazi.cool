import { useSyncExternalStore } from "react";

// A minimal client-side router: the app has five routes and no need for a library.

export type Route =
  | { name: "today"; date: string | null }
  | { name: "stats" }
  | { name: "statsHabit"; id: number }
  | { name: "habits" }
  | { name: "login" }
  | { name: "notFound" };

const datePattern = /^\d{4}-\d{2}-\d{2}$/;

export function matchRoute(path: string): Route {
  const parts = path.split("/").filter(Boolean);
  const [first, second] = parts;
  if (first === undefined) return { name: "today", date: null };
  if (parts.length === 1) {
    if (first === "stats") return { name: "stats" };
    if (first === "habits") return { name: "habits" };
    if (first === "login") return { name: "login" };
  }
  if (parts.length === 2 && second !== undefined) {
    if (first === "day" && datePattern.test(second)) return { name: "today", date: second };
    if (first === "stats" && /^[1-9]\d*$/.test(second))
      return { name: "statsHabit", id: Number(second) };
  }
  return { name: "notFound" };
}

const changeEvent = "habits:navigate";

export function navigate(to: string, options: { replace?: boolean } = {}) {
  if (to === window.location.pathname) return;
  if (options.replace) window.history.replaceState(null, "", to);
  else window.history.pushState(null, "", to);
  window.dispatchEvent(new Event(changeEvent));
}

function subscribe(onChange: () => void) {
  window.addEventListener("popstate", onChange);
  window.addEventListener(changeEvent, onChange);
  return () => {
    window.removeEventListener("popstate", onChange);
    window.removeEventListener(changeEvent, onChange);
  };
}

/** The current path, re-rendering the caller whenever it changes. */
export function usePath(): string {
  return useSyncExternalStore(subscribe, () => window.location.pathname);
}
