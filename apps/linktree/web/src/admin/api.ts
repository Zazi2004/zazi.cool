// HTTP client of the admin API. Every call goes through `request`, which turns
// error responses into ApiError (see ErrorResponse in api/types.ts).
import type {
  CreatePageRequest,
  Document,
  ErrorCode,
  ErrorResponse,
  ImageResponse,
  ImageVisual,
  LoginRequest,
  PageSummary,
  PagesResponse,
  ReorderRequest,
  SaveResponse,
  SlugCheckResponse,
} from "@/api/types";

export class ApiError extends Error {
  constructor(
    readonly status: number,
    readonly code: ErrorCode | "network",
    message: string,
    readonly fields: Record<string, string> = {},
  ) {
    super(message);
  }
}

/** Called on any 401 except the login itself: the app goes back to the login page. */
let onUnauthorized: () => void = () => {};
export function setUnauthorizedHandler(fn: () => void) {
  onUnauthorized = fn;
}

async function request<T>(method: string, path: string, body?: unknown): Promise<T> {
  let res: Response;
  try {
    res = await fetch(`/admin/api${path}`, {
      method,
      headers:
        body instanceof FormData || body === undefined
          ? {}
          : { "Content-Type": "application/json" },
      body: body instanceof FormData ? body : body === undefined ? undefined : JSON.stringify(body),
      credentials: "same-origin",
    });
  } catch {
    throw new ApiError(0, "network", "Serveur injoignable : vérifiez la connexion.");
  }
  if (res.status === 204) return undefined as T;
  const data: unknown = await res.json().catch(() => null);
  if (!res.ok) {
    const err = (data as ErrorResponse | null)?.error;
    if (res.status === 401 && path !== "/login") onUnauthorized();
    throw new ApiError(
      res.status,
      err?.code ?? "internal",
      err?.message ?? `Erreur ${res.status}.`,
      err?.fields ?? {},
    );
  }
  return data as T;
}

export const api = {
  login: (body: LoginRequest) => request<void>("POST", "/login", body),
  logout: () => request<void>("POST", "/logout"),
  session: () => request<void>("GET", "/session"),
  pages: () => request<PagesResponse>("GET", "/pages").then((r) => r.pages),
  createPage: (body: CreatePageRequest) => request<Document>("POST", "/pages", body),
  page: (id: string) => request<Document>("GET", `/pages/${id}`),
  savePage: (doc: Document) => request<SaveResponse>("PUT", `/pages/${doc.id}`, doc),
  deletePage: (id: string) => request<void>("DELETE", `/pages/${id}`),
  reorder: (body: ReorderRequest) => request<void>("PUT", "/pages/order", body),
  checkSlug: (slug: string, except?: string) =>
    request<SlugCheckResponse>(
      "GET",
      `/slugs/${encodeURIComponent(slug)}${except ? `?except=${except}` : ""}`,
    ),
  uploadImage: (file: Blob) => {
    const form = new FormData();
    form.append("file", file, "image");
    return request<ImageResponse>("POST", "/images", form).then((r): ImageVisual => r.visual);
  },
};

export type { PageSummary };
