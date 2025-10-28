package embeddocs

import (
	"embed"
	"io/fs"
)

//go:embed website/pages/docs
var EmbeddedDocs embed.FS

// GetDocsFS returns the embedded docs filesystem
func GetDocsFS() (fs.FS, error) {
	return fs.Sub(EmbeddedDocs, "website/pages/docs")
}
