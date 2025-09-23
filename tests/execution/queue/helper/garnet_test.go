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
