package scaffold

import (
	"context"
	"embed"
	"io/fs"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

//go:embed scaffold_fixtures
var dir embed.FS
var fixtures fs.FS

func init() {
	var err error

	// Also set the CacheDir to the same directory as the local fixtures.
	// This is a bit of a hack, but it's the easiest way to test the scaffold
	// using both `fs.FS` and `os` for Windows compatibility.
	//
	// See https://github.com/inngest/inngest/pull/188
	CacheDir = filepath.Join(".", "scaffold_fixtures")

	fixtures, err = fs.Sub(dir, filepath.Join(".", "scaffold_fixtures"))
	if err != nil {
		panic(err)
	}
}

func TestParse(t *testing.T) {
	mapping, err := parse(context.Background(), fixtures)
	require.NoError(t, err)
	require.NotNil(t, mapping)
	require.Equal(t, 1, len(mapping.Languages))
	require.Equal(t, 2, len(mapping.Languages["typescript"]))

	for _, v := range mapping.Languages["typescript"] {
		require.NotEmpty(t, v.root)
	}
}
