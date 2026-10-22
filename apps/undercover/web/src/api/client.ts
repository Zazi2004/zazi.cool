import type {
  CheckResponse,
  CreatePairRequest,
  CreatePairResponse,
  DrawResponse,
  ErrorResponse,
  GuessRequest,
  GuessResponse,
  ReportReason,
  ReportRequest,
} from "./types";
import type { ExtremeMode } from "../game/engine";

/** An HTTP error answered by the server, with its decoded JSON body if any. */
export class ApiError extends Error {
  constructor(
    readonly status: number,
    readonly body: unknown,
  ) {
    super(errorMessage(body) ?? `Erreur ${status}`);
  }
}

export type NetworkErrorKind = "offline" | "timeout" | "unreachable";

/** The request never got an answer: no connection, too slow, or dropped on the way. */
export class NetworkError extends Error {
  constructor(
    readonly kind: NetworkErrorKind,
    options?: ErrorOptions,
  ) {
    super(networkMessages[kind], options);
  }
}

const networkMessages: Record<NetworkErrorKind, string> = {
  offline: "Vous êtes hors ligne. Vérifiez votre connexion.",
  timeout: "Le serveur met trop de temps à répondre.",
  unreachable: "Impossible de joindre le serveur. Vérifiez la connexion.",
};

/** How long a request may take before it is given up as a timeout. */
export const REQUEST_TIMEOUT_MS = 15_000;

/** An error shaped for the screens: a French message and whether trying again can help. */
export interface UiError {
  message: string;
  retryable: boolean;
}

/** Turns anything thrown by this client into a message and a retry hint. */
export function toUiError(err: unknown): UiError {
  if (err instanceof NetworkError) return { message: err.message, retryable: true };
  if (err instanceof ApiError) {
    if (err.status === 429) {
      return {
        message: "Trop de requêtes. Patientez quelques secondes puis réessayez.",
        retryable: true,
      };
    }
    if (err.status >= 500) {
      return {
        message: `Le serveur a rencontré un problème (erreur ${err.status}).`,
        retryable: true,
      };
    }
    return { message: err.message, retryable: false };
  }
  return { message: err instanceof Error ? err.message : "Erreur inattendue.", retryable: false };
}

function errorMessage(body: unknown): string | undefined {
  if (body && typeof body === "object" && "error" in body) {
    const { error } = body as ErrorResponse;
    if (typeof error === "string" && error !== "") return error;
  }
  return undefined;
}

interface RequestOptions {
  signal?: AbortSignal;
  /**
   * Retry once when the request never reached the server, or when a gateway answered
   * 502/503/504 (redeploy, restarting pod). Only for requests that are safe to repeat:
   * a connection reused after a long pause may have been closed in the meantime (Mr.
   * White thinking, phone asleep), and browsers do not retry a POST on their own.
   */
  idempotent?: boolean;
}

const RETRY_DELAY_MS = 300;
const RETRY_STATUSES = new Set([502, 503, 504]);

function isAbortError(err: unknown) {
  return err instanceof DOMException && err.name === "AbortError";
}

/** Resolves after `ms`, or rejects with the signal's abort reason if it fires first. */
function sleep(ms: number, signal?: AbortSignal) {
  return new Promise<void>((resolve, reject) => {
    if (signal?.aborted) return reject(signal.reason);
    const onAbort = () => {
      clearTimeout(timer);
      reject(signal?.reason);
    };
    const timer = setTimeout(() => {
      signal?.removeEventListener("abort", onAbort);
      resolve();
    }, ms);
    signal?.addEventListener("abort", onAbort, { once: true });
  });
}

async function request<T>(method: string, path: string, body?: unknown, opts: RequestOptions = {}) {
  const send = async () => {
    const controller = new AbortController();
    let timedOut = false;
    const timer = setTimeout(() => {
      timedOut = true;
      controller.abort();
    }, REQUEST_TIMEOUT_MS);
    const onAbort = () => controller.abort();
    opts.signal?.addEventListener("abort", onAbort);
    try {
      const res = await fetch(path, {
        method,
        headers: body === undefined ? undefined : { "Content-Type": "application/json" },
        body: body === undefined ? undefined : JSON.stringify(body),
        signal: controller.signal,
      });
      return { res, text: await res.text() };
    } catch (err) {
      if (timedOut) throw new NetworkError("timeout", { cause: err });
      throw err;
    } finally {
      clearTimeout(timer);
      opts.signal?.removeEventListener("abort", onAbort);
    }
  };
  const attempts = opts.idempotent ? 2 : 1;
  let res: Response;
  let text: string;
  try {
    for (let attempt = 1; ; attempt++) {
      try {
        ({ res, text } = await send());
      } catch (err) {
        if (attempt >= attempts || opts.signal?.aborted || err instanceof NetworkError) throw err;
        await sleep(RETRY_DELAY_MS, opts.signal);
        continue;
      }
      if (attempt < attempts && RETRY_STATUSES.has(res.status)) {
        await sleep(RETRY_DELAY_MS, opts.signal);
        continue;
      }
      break;
    }
  } catch (err) {
    if (err instanceof NetworkError) throw err;
    if (isAbortError(err)) throw err;
    const offline = typeof navigator !== "undefined" && navigator.onLine === false;
    throw new NetworkError(offline ? "offline" : "unreachable", { cause: err });
  }

  let data: unknown;
  try {
    data = text === "" ? undefined : JSON.parse(text);
  } catch {
    // Not JSON: typically an HTML error page from the ingress.
    if (!res.ok) throw new ApiError(res.status, undefined);
    throw new Error("Réponse inattendue du serveur.");
  }
  if (!res.ok) throw new ApiError(res.status, data);
  return data as T;
}

export interface DrawOptions {
  extreme: ExtremeMode;
  suggestionsOnly: boolean;
  exclude: number[];
}

/** Values of the "extreme" query parameter; "true" predates the "only" mode. */
const extremeParam: Record<ExtremeMode, string> = { off: "false", on: "true", only: "only" };

export function drawPair({ extreme, suggestionsOnly, exclude }: DrawOptions, signal?: AbortSignal) {
  const params = new URLSearchParams({
    extreme: extremeParam[extreme],
    suggestionsOnly: String(suggestionsOnly),
  });
  if (exclude.length > 0) params.set("exclude", exclude.join(","));
  return request<DrawResponse>("GET", `/api/pairs/random?${params}`, undefined, {
    signal,
    idempotent: true,
  });
}

export function checkGuess(pairId: number, req: GuessRequest) {
  return request<GuessResponse>("POST", `/api/pairs/${pairId}/guess`, req, {
    idempotent: true, // a pure comparison: repeating it changes nothing
  });
}

export function checkPair(a: string, b: string, signal?: AbortSignal) {
  const params = new URLSearchParams({ a, b });
  return request<CheckResponse>("GET", `/api/pairs/check?${params}`, undefined, {
    signal,
    idempotent: true,
  });
}

export type CreateResult =
  | { ok: true; response: CreatePairResponse }
  | { ok: false; check: CheckResponse }
  | { ok: false; error: string; retryable: boolean };

/** Submits a pair. A refusal (duplicate, similar, invalid) is a result, not an error. */
export async function createPair(req: CreatePairRequest): Promise<CreateResult> {
  try {
    const response = await request<CreatePairResponse>("POST", "/api/pairs", req);
    return { ok: true, response };
  } catch (err) {
    if (err instanceof ApiError && isCheckResponse(err.body)) {
      return { ok: false, check: err.body };
    }
    if (err instanceof Error) {
      const { message, retryable } = toUiError(err);
      return { ok: false, error: message, retryable };
    }
    throw err;
  }
}

function isCheckResponse(body: unknown): body is CheckResponse {
  return !!body && typeof body === "object" && "status" in body;
}

export function reportPair(pairId: number, reason: ReportReason) {
  const req: ReportRequest = { reason };
  return request<void>("POST", `/api/pairs/${pairId}/reports`, req);
}
