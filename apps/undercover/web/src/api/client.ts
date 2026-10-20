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
   * Retry once when the request never reached the server. Only for requests that are
   * safe to repeat: a connection reused after a long pause may have been closed in
   * the meantime (Mr. White thinking, phone asleep), and browsers do not retry a POST
   * on their own.
   */
  retryOnNetworkError?: boolean;
}

async function request<T>(method: string, path: string, body?: unknown, opts: RequestOptions = {}) {
  const send = () =>
    fetch(path, {
      method,
      headers: body === undefined ? undefined : { "Content-Type": "application/json" },
      body: body === undefined ? undefined : JSON.stringify(body),
      signal: opts.signal,
    });
  let res: Response;
  try {
    try {
      res = await send();
    } catch (err) {
      if (!opts.retryOnNetworkError || opts.signal?.aborted) throw err;
      res = await send();
    }
  } catch (err) {
    if (err instanceof DOMException && err.name === "AbortError") throw err;
    throw new Error("Impossible de joindre le serveur. Vérifiez la connexion.", { cause: err });
  }

  const text = await res.text();
  const data: unknown = text === "" ? undefined : JSON.parse(text);
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

export function drawPair({ extreme, suggestionsOnly, exclude }: DrawOptions) {
  const params = new URLSearchParams({
    extreme: extremeParam[extreme],
    suggestionsOnly: String(suggestionsOnly),
  });
  if (exclude.length > 0) params.set("exclude", exclude.join(","));
  return request<DrawResponse>("GET", `/api/pairs/random?${params}`);
}

export function checkGuess(pairId: number, req: GuessRequest) {
  return request<GuessResponse>("POST", `/api/pairs/${pairId}/guess`, req, {
    retryOnNetworkError: true, // a pure comparison: repeating it changes nothing
  });
}

export function checkPair(a: string, b: string, signal?: AbortSignal) {
  const params = new URLSearchParams({ a, b });
  return request<CheckResponse>("GET", `/api/pairs/check?${params}`, undefined, { signal });
}

export type CreateResult =
  | { ok: true; response: CreatePairResponse }
  | { ok: false; check: CheckResponse }
  | { ok: false; error: string };

/** Submits a pair. A refusal (duplicate, similar, invalid) is a result, not an error. */
export async function createPair(req: CreatePairRequest): Promise<CreateResult> {
  try {
    const response = await request<CreatePairResponse>("POST", "/api/pairs", req);
    return { ok: true, response };
  } catch (err) {
    if (err instanceof ApiError && isCheckResponse(err.body)) {
      return { ok: false, check: err.body };
    }
    if (err instanceof Error) return { ok: false, error: err.message };
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
