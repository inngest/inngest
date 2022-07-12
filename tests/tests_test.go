//go:build integration

package main

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAll(t *testing.T) {
	if testing.Short() {
		return
	}

	err := do(context.Background())
	require.NoError(t, err)
}
