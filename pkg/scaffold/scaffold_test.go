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
		require.NotNil(t, v.FS)
	}
}
