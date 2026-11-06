package server

import (
	"io/fs"
	"mime"
	"net/http"
	"path"
	"strings"
)

func init() {
	// The distroless image has no /etc/mime.types and Go knows no .webmanifest:
	// without this the manifest would be sniffed as text/plain and browsers ignore it.
	if err := mime.AddExtensionType(".webmanifest", "application/manifest+json"); err != nil {
		panic(err)
	}
}

// staticHandler serves the built front (NF1). A path that is a file of the build
// (favicon, manifest…) is served as is; any other path is a client-side route
// (/habits, /stats…) and gets index.html. Under /assets/ a missing file is a plain
// 404: serving the page there would hand HTML to a <script> tag.
func staticHandler(static fs.FS) http.Handler {
	files := http.FileServerFS(static)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			w.Header().Set("Allow", "GET, HEAD")
			writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Méthode non autorisée.")
			return
		}
		name := strings.TrimPrefix(path.Clean(r.URL.Path), "/")
		// Vite puts a content hash in asset file names: they never change.
		if strings.HasPrefix(name, "assets/") {
			w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
			files.ServeHTTP(w, r)
			return
		}
		w.Header().Set("Cache-Control", "no-cache")
		info, err := fs.Stat(static, name)
		if name != "" && name != "index.html" && err == nil && !info.IsDir() {
			files.ServeHTTP(w, r)
			return
		}
		http.ServeFileFS(w, r, static, "index.html")
	})
}
