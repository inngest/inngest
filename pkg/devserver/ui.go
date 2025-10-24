package devserver

import (
	"context"
	"embed"
	"mime"
	"path"
	"strings"
)

//go:embed all:static
var static embed.FS

func init() {
	//
	// Fix invalid mime type errors when loading JS from our assets on windows
	_ = mime.AddExtensionType(".js", "application/javascript")
}

// serve implements SPA routing for Tanstack assets:
// - Serves static files from static/client if they exist
// - Falls back to _shell.html for all other routes (client-side routing)
func serve(ctx context.Context, requestPath string) []byte {
	//
	// Try to serve the file directly from static/client
	filePath := path.Join("static/client", requestPath)

	if byt, err := static.ReadFile(filePath); err == nil {
		return byt
	}

	//
	// If the path has a file extension, it was likely a missing asset
	// Don't fallback to shell in this case
	if strings.Contains(path.Base(requestPath), ".") {
		return nil
	}

	//
	// Fall back to _shell.html for client-side routing
	byt, _ := static.ReadFile("static/client/_shell.html")
	return byt
}
