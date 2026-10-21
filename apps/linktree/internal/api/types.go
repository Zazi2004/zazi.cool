package api

// This file is the single definition of the JSON messages exchanged with the front:
// the data injected in the public pages and the admin API messages. The TypeScript
// counterpart is web/src/api/types.ts; contract_test.go keeps both aligned through
// the web/src/api/contract.json fixture.

// VisualKind tells an uploaded image from a coolshapes shape.
type VisualKind string

const (
	VisualImage VisualKind = "image"
	VisualShape VisualKind = "shape"
)

// Visual is the picture of a page avatar, a group or a link. A missing visual is
// JSON null. Images carry ID, URL and dimensions; shapes carry Shape. The admin
// sends back only Kind and ID (image) or Shape (shape): the rest is ignored.
type Visual struct {
	Kind   VisualKind `json:"kind"`
	ID     string     `json:"id,omitempty"`
	URL    string     `json:"url,omitempty"`
	Width  int        `json:"width,omitempty"`
	Height int        `json:"height,omitempty"`
	Shape  string     `json:"shape,omitempty"`
}

// --- Public pages (JSON in <script id="page-data">) ---

// View tells the public app which screen to show.
type View string

const (
	ViewIndex    View = "index"
	ViewPage     View = "page"
	ViewNotFound View = "notFound"
)

// IndexView is the data of the index (/).
type IndexView struct {
	View  View         `json:"view"`
	Pages []IndexEntry `json:"pages"`
}

// IndexEntry is a page listed in the index.
type IndexEntry struct {
	Slug        string  `json:"slug"`
	Title       string  `json:"title"`
	Description string  `json:"description"`
	Avatar      *Visual `json:"avatar"`
}

// PageView is the data of a links page (/{slug}). Drafts are never in it.
type PageView struct {
	View View       `json:"view"`
	Page PublicPage `json:"page"`
}

// NotFoundView is the data of the 404 page.
type NotFoundView struct {
	View View `json:"view"`
}

// PublicPage is a page as visitors see it.
type PublicPage struct {
	ID          string  `json:"id"` // seeds the theme decoration, so a page keeps its look
	Slug        string  `json:"slug"`
	Title       string  `json:"title"`
	Description string  `json:"description"`
	Avatar      *Visual `json:"avatar"`
	Theme       string  `json:"theme"`
	Accent      string  `json:"accent"`
	// Items holds PublicLink and PublicGroup values, told apart by their "kind".
	Items []any `json:"items"`
}

// PublicLink is a link of a public page.
type PublicLink struct {
	Kind   string  `json:"kind"` // always "link"
	ID     string  `json:"id"`
	Title  string  `json:"title"`
	URL    string  `json:"url"`
	Visual *Visual `json:"visual"`
}

// PublicGroup is a group (accordion) of a public page.
type PublicGroup struct {
	Kind          string       `json:"kind"` // always "group"
	ID            string       `json:"id"`
	Title         string       `json:"title"`
	Subtitle      string       `json:"subtitle"`
	Visual        *Visual      `json:"visual"`
	OpenByDefault bool         `json:"openByDefault"`
	Links         []PublicLink `json:"links"`
}

// --- Admin API ---

// LoginRequest is the body of POST /admin/api/login.
type LoginRequest struct {
	Password string `json:"password"`
}

// SessionResponse answers GET /admin/api/session when logged in.
type SessionResponse struct {
	Authenticated bool `json:"authenticated"`
}

// PageSummary is a page in the admin list.
type PageSummary struct {
	ID          string  `json:"id"`
	Slug        string  `json:"slug"`
	Title       string  `json:"title"`
	Description string  `json:"description"`
	Avatar      *Visual `json:"avatar"`
	Published   bool    `json:"published"`
	Listed      bool    `json:"listed"`
	LinkCount   int     `json:"linkCount"`
}

// PagesResponse answers GET /admin/api/pages.
type PagesResponse struct {
	Pages []PageSummary `json:"pages"`
}

// CreatePageRequest is the body of POST /admin/api/pages.
type CreatePageRequest struct {
	Title string `json:"title"`
	Slug  string `json:"slug"`
}

// Document is the full page edited in the admin: GET and PUT /admin/api/pages/{id}.
type Document struct {
	ID          string    `json:"id"`
	Slug        string    `json:"slug"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	Avatar      *Visual   `json:"avatar"`
	Theme       string    `json:"theme"`
	Accent      string    `json:"accent"`
	Published   bool      `json:"published"`
	Listed      bool      `json:"listed"`
	Version     int       `json:"version"`
	Items       []DocItem `json:"items"`
}

// DocItem is a link or a group of a Document. Every field is always present, so the
// editor works with one simple type; fields that do not apply are empty.
type DocItem struct {
	ID            string    `json:"id"`
	Kind          string    `json:"kind"` // "link" or "group"
	Title         string    `json:"title"`
	Subtitle      string    `json:"subtitle"`      // groups only
	URL           string    `json:"url"`           // links only
	Visual        *Visual   `json:"visual"`        // shapes for groups only
	OpenByDefault bool      `json:"openByDefault"` // groups only
	Published     bool      `json:"published"`
	Links         []DocItem `json:"links"` // groups only, [] for links
}

// SaveResponse answers PUT /admin/api/pages/{id}.
type SaveResponse struct {
	Version int `json:"version"`
}

// ReorderRequest is the body of PUT /admin/api/pages/order.
type ReorderRequest struct {
	IDs []string `json:"ids"`
}

// SlugCheckResponse answers GET /admin/api/slugs/{slug}. Error explains why the
// slug cannot be used ("" when it can).
type SlugCheckResponse struct {
	Slug      string `json:"slug"`
	Available bool   `json:"available"`
	Error     string `json:"error"`
}

// ImageResponse answers POST /admin/api/images.
type ImageResponse struct {
	Visual Visual `json:"visual"`
}

// ErrorCode identifies the kind of error for the front.
type ErrorCode string

const (
	CodeBadRequest   ErrorCode = "bad_request"
	CodeValidation   ErrorCode = "validation"
	CodeUnauthorized ErrorCode = "unauthorized"
	CodeBadPassword  ErrorCode = "bad_password"
	CodeNotFound     ErrorCode = "not_found"
	CodeConflict     ErrorCode = "conflict"
	CodeTooLarge     ErrorCode = "too_large"
	CodeRateLimited  ErrorCode = "rate_limited"
	CodeInternal     ErrorCode = "internal"
)

// ErrorResponse is the body of every 4xx/5xx JSON response.
type ErrorResponse struct {
	Error ErrorBody `json:"error"`
}

// ErrorBody details an error. Fields maps a field to its message, e.g. "slug" or
// "items.<uuid>.url", so that the editor can highlight it.
type ErrorBody struct {
	Code    ErrorCode         `json:"code"`
	Message string            `json:"message"`
	Fields  map[string]string `json:"fields,omitempty"`
}
