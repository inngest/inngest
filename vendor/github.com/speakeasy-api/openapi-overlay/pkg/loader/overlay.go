package loader

import (
	"fmt"
	"github.com/speakeasy-api/openapi-overlay/pkg/overlay"
)

// LoadOverlay is a tool for loading and parsing an overlay file from the file
// system.
func LoadOverlay(path string) (*overlay.Overlay, error) {
	o, err := overlay.Parse(path)
	if err != nil {
		return nil, fmt.Errorf("failed to parse overlay from path %q: %w", path, err)
	}

	return o, nil
}
