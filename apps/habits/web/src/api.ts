// Client of the JSON API (spec §7). Errors carry the code and the French message
// sent by the server, so screens can show `message` as is.

import type { Day, Habit, HabitInput, HabitList, Moment } from "./types";

export class ApiError extends Error {
  readonly status: number;
  readonly code: string;

  constructor(status: number, code: string, message: string) {
    super(message);
    this.name = "ApiError";
    this.status = status;
    this.code = code;
  }
}

const networkMessage = "Impossible de joindre le serveur, réessayez.";

async function request<T>(method: string, path: string, body?: unknown): Promise<T> {
  let res: Response;
  try {
    res = await fetch(path, {
      method,
      credentials: "same-origin",
      headers: body === undefined ? undefined : { "Content-Type": "application/json" },
      body: body === undefined ? undefined : JSON.stringify(body),
    });
  } catch {
    throw new ApiError(0, "network", networkMessage);
  }
  if (res.ok) {
    return (res.status === 204 ? undefined : await res.json()) as T;
  }
  let code = "error";
  let message = "Une erreur est survenue, réessayez.";
  try {
    const payload = (await res.json()) as { error?: string; message?: string };
    code = payload.error ?? code;
    message = payload.message ?? message;
  } catch {
    // The body is not JSON (a proxy error page, for instance): keep the defaults.
  }
  throw new ApiError(res.status, code, message);
}

export interface Session {
  authenticated: boolean;
}

/** Resolves to false when there is no valid session; other failures are thrown. */
export async function hasSession(): Promise<boolean> {
  try {
    const session = await request<Session>("GET", "/api/session");
    return session.authenticated;
  } catch (e) {
    if (e instanceof ApiError && e.status === 401) return false;
    throw e;
  }
}

export function login(password: string): Promise<void> {
  return request("POST", "/api/login", { password });
}

export function logout(): Promise<void> {
  return request("POST", "/api/logout");
}

export function listHabits(): Promise<HabitList> {
  return request("GET", "/api/habits");
}

export function createHabit(input: HabitInput): Promise<Habit> {
  return request("POST", "/api/habits", input);
}

export function updateHabit(id: number, patch: Partial<HabitInput>): Promise<Habit> {
  return request("PATCH", `/api/habits/${id}`, patch);
}

export function archiveHabit(id: number): Promise<Habit> {
  return request("POST", `/api/habits/${id}/archive`);
}

export function restoreHabit(id: number): Promise<Habit> {
  return request("POST", `/api/habits/${id}/restore`);
}

/** Sets the manual order of the active habits of a moment (all of them, once each). */
export function reorderHabits(moment: Moment, ids: number[]): Promise<void> {
  return request("PUT", "/api/habits/order", { moment, ids });
}

export function getDay(date: string): Promise<Day> {
  return request("GET", `/api/days/${date}`);
}

export function putEntry(date: string, habitId: number, value: number): Promise<Day> {
  return request("PUT", `/api/days/${date}/entries/${habitId}`, { value });
}

export function putNote(date: string, text: string): Promise<void> {
  return request("PUT", `/api/days/${date}/note`, { text });
}
