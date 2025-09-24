package helper

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestValkeyClientPing(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping functional tests")
	}

	ctx := t.Context()

	vc, err := StartValkey(t)
	require.NoError(t, err)

	// Create client with the stored address (which is now the full connectAddr)
	rc, err := NewValkeyClient(vc.Addr, vc.Username, vc.Password, false)
	require.NoError(t, err)

	res, err := rc.Do(ctx, rc.B().Ping().Build()).ToString()
	require.NoError(t, err)
	require.Equal(t, "PONG", res)
}

func TestValkeyCustomConfiguration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping functional tests")
	}

	ctx := t.Context()

	vc, err := StartValkey(t, WithValkeyConfiguration(&ValkeyConfiguration{
		MaxMemory: "100mb",
		Loglevel:  "verbose",
	}))
	require.NoError(t, err)

	// Create client with the stored address (which is now the full connectAddr)
	rc, err := NewValkeyClient(vc.Addr, vc.Username, vc.Password, false)
	require.NoError(t, err)

	res, err := rc.Do(ctx, rc.B().Ping().Build()).ToString()
	require.NoError(t, err)
	require.Equal(t, "PONG", res)

	// Test that configuration was applied by checking memory config
	// CONFIG GET returns a map in newer Redis/Valkey versions
	configRes, err := rc.Do(ctx, rc.B().Arbitrary("CONFIG", "GET", "maxmemory").Build()).ToMap()
	if err != nil {
		// Try array format for older versions
		configArr, err2 := rc.Do(ctx, rc.B().Arbitrary("CONFIG", "GET", "maxmemory").Build()).ToArray()
		if err2 != nil {
			t.Fatalf("Failed to get config as map or array: map error: %v, array error: %v", err, err2)
		}
		// Handle array response
		require.Len(t, configArr, 2)
		configName, err := configArr[0].ToString()
		require.NoError(t, err)
		require.Equal(t, "maxmemory", configName)
		
		configValue, err := configArr[1].ToString()
		require.NoError(t, err)
		// Valkey returns the config value in bytes, so "100mb" should be converted
		// The actual value will be "104857600" (100 * 1024 * 1024)
		require.Equal(t, "104857600", configValue)
	} else {
		// Handle map response
		maxMemoryValue, exists := configRes["maxmemory"]
		require.True(t, exists, "maxmemory config not found")
		
		configValue, err := maxMemoryValue.ToString()
		require.NoError(t, err)
		require.Equal(t, "104857600", configValue)
	}
}

func TestValkeyCustomImage(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping functional tests")
	}

	ctx := t.Context()

	// Test with a custom image (using the same image for testing purposes)
	customImage := "valkey/valkey:8.0.1"
	vc, err := StartValkey(t, WithValkeyImage(customImage))
	require.NoError(t, err)

	// Create client with the stored address
	rc, err := NewValkeyClient(vc.Addr, vc.Username, vc.Password, false)
	require.NoError(t, err)

	res, err := rc.Do(ctx, rc.B().Ping().Build()).ToString()
	require.NoError(t, err)
	require.Equal(t, "PONG", res)
}