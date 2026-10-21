// Mirror of internal/api/types.go: the data injected in the public pages and the
// admin API messages. contract.test.ts checks both stay aligned (see contract.json).
import type { AccentId, ThemeId } from "@/themes/tokens";

export interface ImageVisual {
  kind: "image";
  id: string;
  url: string;
  width: number;
  height: number;
}

export interface ShapeVisual {
  kind: "shape";
  shape: string;
}

export type Visual = ImageVisual | ShapeVisual;

// --- Public pages ---

export interface IndexEntry {
  slug: string;
  title: string;
  description: string;
  avatar: Visual | null;
}

export interface IndexView {
  view: "index";
  pages: IndexEntry[];
}

export interface PublicLink {
  kind: "link";
  id: string;
  title: string;
  url: string;
  visual: Visual | null;
}

export interface PublicGroup {
  kind: "group";
  id: string;
  title: string;
  subtitle: string;
  visual: Visual | null;
  openByDefault: boolean;
  links: PublicLink[];
}

export type PublicItem = PublicLink | PublicGroup;

export interface PublicPage {
  id: string;
  slug: string;
  title: string;
  description: string;
  avatar: Visual | null;
  theme: ThemeId;
  accent: AccentId;
  items: PublicItem[];
}

export interface PageView {
  view: "page";
  page: PublicPage;
}

export interface NotFoundView {
  view: "notFound";
}

export type PageData = IndexView | PageView | NotFoundView;

// --- Admin API ---

export interface LoginRequest {
  password: string;
}

export interface SessionResponse {
  authenticated: boolean;
}

export interface PageSummary {
  id: string;
  slug: string;
  title: string;
  description: string;
  avatar: Visual | null;
  published: boolean;
  listed: boolean;
  linkCount: number;
}

export interface PagesResponse {
  pages: PageSummary[];
}

export interface CreatePageRequest {
  title: string;
  slug: string;
}

export interface DocItem {
  id: string;
  kind: "link" | "group";
  title: string;
  subtitle: string;
  url: string;
  visual: Visual | null;
  openByDefault: boolean;
  published: boolean;
  links: DocItem[];
}

export interface Document {
  id: string;
  slug: string;
  title: string;
  description: string;
  avatar: Visual | null;
  theme: ThemeId;
  accent: AccentId;
  published: boolean;
  listed: boolean;
  version: number;
  items: DocItem[];
}

export interface SaveResponse {
  version: number;
}

export interface ReorderRequest {
  ids: string[];
}

export interface SlugCheckResponse {
  slug: string;
  available: boolean;
  error: string;
}

export interface ImageResponse {
  visual: ImageVisual;
}

export type ErrorCode =
  | "bad_request"
  | "validation"
  | "unauthorized"
  | "bad_password"
  | "not_found"
  | "conflict"
  | "too_large"
  | "rate_limited"
  | "internal";

export interface ErrorResponse {
  error: {
    code: ErrorCode;
    message: string;
    fields?: Record<string, string>;
  };
}

/** One example of each message: the shape of contract.json. */
export interface Contract {
  themes: readonly ThemeId[];
  accents: readonly AccentId[];
  imageVisual: ImageVisual;
  shapeVisual: ShapeVisual;
  indexView: IndexView;
  notFoundView: NotFoundView;
  pageView: PageView;
  loginRequest: LoginRequest;
  sessionResponse: SessionResponse;
  pagesResponse: PagesResponse;
  createPageRequest: CreatePageRequest;
  document: Document;
  saveResponse: SaveResponse;
  reorderRequest: ReorderRequest;
  slugCheckResponse: SlugCheckResponse;
  imageResponse: ImageResponse;
  errorResponse: ErrorResponse;
}
