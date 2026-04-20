// Package uiembed bundles the built Vite UI into the harness binary.
//
// The embedded filesystem is rooted at `dist/`. When building the UI, the
// Vite output must be written to the sibling `dist/` directory so go:embed
// picks it up at compile time.
package uiembed

import (
	"embed"
	"io/fs"
	"net/http"
)

//go:embed all:dist
var dist embed.FS

// FS returns an http.FileSystem rooted at the embedded UI dist directory.
// If the UI has not been built, it serves the stub index.html that ships
// checked-in so the binary always compiles and runs.
func FS() http.FileSystem {
	sub, err := fs.Sub(dist, "dist")
	if err != nil {
		// Impossible unless embed is broken; panic so misconfig is loud.
		panic(err)
	}
	return http.FS(sub)
}

// SPAHandler serves the embedded SPA. Unknown paths fall back to index.html
// so TanStack Router's client-side routes resolve.
func SPAHandler() http.Handler {
	fsrv := http.FileServer(FS())
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Serve the file if it exists; otherwise rewrite to /index.html.
		if f, err := FS().Open(r.URL.Path); err == nil {
			_ = f.Close()
			fsrv.ServeHTTP(w, r)
			return
		}
		r2 := r.Clone(r.Context())
		r2.URL.Path = "/"
		fsrv.ServeHTTP(w, r2)
	})
}
