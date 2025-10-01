package helper

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRedisClientPing(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping functional tests")
	}

	ctx := t.Context()

	gc, err := StartGarnet(t)
	require.NoError(t, err)

	// Create client with the stored address (which is now the full connectAddr)
	rc, err := NewRedisClient(gc.Addr, gc.Username, gc.Password)
	require.NoError(t, err)

	res, err := rc.Do(ctx, rc.B().Ping().Build()).ToString()
	require.NoError(t, err)
	require.Equal(t, "PONG", res)
}

func TestCustomConfiguration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping functional tests")
	}

	ctx := t.Context()

	gc, err := StartGarnet(t, WithConfiguration(&GarnetConfiguration{
		EnableLua: true,
	}))
	require.NoError(t, err)

	// Create client with the stored address (which is now the full connectAddr)
	rc, err := NewRedisClient(gc.Addr, gc.Username, gc.Password)
	require.NoError(t, err)

	res, err := rc.Do(ctx, rc.B().Ping().Build()).ToString()
	require.NoError(t, err)
	require.Equal(t, "PONG", res)

	// Test Lua functionality with hello world eval
	luaScript := `return "hello world"`
	evalRes, err := rc.Do(ctx, rc.B().Eval().Script(luaScript).Numkeys(0).Build()).ToString()
	require.NoError(t, err)
	require.Equal(t, "hello world", evalRes)
}

func TestCustomImage(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping functional tests")
	}

	ctx := t.Context()

	// Test with a custom image (using the same image for testing purposes)
	customImage := "ghcr.io/microsoft/garnet:1.0.84"
	gc, err := StartGarnet(t, WithImage(customImage))
	require.NoError(t, err)

	// Create client with the stored address
	rc, err := NewRedisClient(gc.Addr, gc.Username, gc.Password)
	require.NoError(t, err)

	res, err := rc.Do(ctx, rc.B().Ping().Build()).ToString()
	require.NoError(t, err)
	require.Equal(t, "PONG", res)
}
