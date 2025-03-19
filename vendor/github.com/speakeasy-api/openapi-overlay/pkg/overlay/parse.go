package overlay

import (
	"fmt"
	"gopkg.in/yaml.v3"
	"io"
	"os"
	"path/filepath"
)

// Parse will parse the given reader as an overlay file.
func Parse(path string) (*Overlay, error) {
	filePath, err := filepath.Abs(path)
	if err != nil {
		return nil, fmt.Errorf("failed to get absolute path for %q: %w", path, err)
	}

	ro, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open overlay file at path %q: %w", path, err)
	}
	defer ro.Close()

	var overlay Overlay
	dec := yaml.NewDecoder(ro)

	err = dec.Decode(&overlay)
	if err != nil {
		return nil, err
	}

	return &overlay, err
}

// Format will validate reformat the given file
func Format(path string) error {
	overlay, err := Parse(path)
	if err != nil {
		return err
	}
	filePath, err := filepath.Abs(path)
	if err != nil {
		return fmt.Errorf("failed to open overlay file at path %q: %w", path, err)
	}
	formatted, err := overlay.ToString()
	if err != nil {
		return err
	}

	return os.WriteFile(filePath, []byte(formatted), 0644)
}

// Format writes the file back out as YAML.
func (o *Overlay) Format(w io.Writer) error {
	enc := yaml.NewEncoder(w)
	enc.SetIndent(2)
	return enc.Encode(o)
}
