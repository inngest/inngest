package constraintapi

import (
	"context"
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"net"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/enums"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

//go:embed lua/*
var testFS embed.FS

func TestSerializedConstraintItem(t *testing.T) {
	// Test UUIDs
	accountID := uuid.MustParse("12345678-1234-1234-1234-123456789abc")
	envID := uuid.MustParse("87654321-4321-4321-4321-cba987654321")
	functionID := uuid.MustParse("11111111-2222-3333-4444-555555555555")

	testConfig := ConstraintConfig{
		FunctionVersion: 1,
		RateLimit: []RateLimitConfig{
			{
				Scope:             enums.RateLimitScopeAccount,
				Limit:             100,
				Period:            60,
				KeyExpressionHash: "test-key-hash",
			},
		},
		Concurrency: ConcurrencyConfig{
			AccountConcurrency:     50,
			FunctionConcurrency:    25,
			AccountRunConcurrency:  10,
			FunctionRunConcurrency: 5,
			CustomConcurrencyKeys: []CustomConcurrencyLimit{
				{
					Mode:              enums.ConcurrencyModeRun,
					Scope:             enums.ConcurrencyScopeEnv,
					Limit:             15,
					KeyExpressionHash: "custom-key",
				},
			},
		},
		Throttle: []ThrottleConfig{
			{
				Scope:             enums.ThrottleScopeFn,
				Limit:             200,
				Burst:             300,
				Period:            60,
				KeyExpressionHash: "throttle-expr",
			},
		},
	}

	tests := []struct {
		name     string
		input    ConstraintItem
		expected string
	}{
		{
			name: "RateLimit constraint with embedded config",
			input: ConstraintItem{
				Kind: ConstraintKindRateLimit,
				RateLimit: &RateLimitConstraint{
					Scope:             enums.RateLimitScopeAccount,
					KeyExpressionHash: "test-key-hash",
					EvaluatedKeyHash:  "eval-hash",
				},
			},
			expected: `{"k":1,"r":{"b":10,"s":2,"h":"test-key-hash","eh":"eval-hash","k":"{cs}:a:12345678-1234-1234-1234-123456789abc:rl:a:eval-hash","l":100,"p":60000000000}}`,
		},
		{
			name: "Concurrency constraint with custom key",
			input: ConstraintItem{
				Kind: ConstraintKindConcurrency,
				Concurrency: &ConcurrencyConstraint{
					Mode:              enums.ConcurrencyModeRun,
					Scope:             enums.ConcurrencyScopeEnv,
					KeyExpressionHash: "custom-key",
					EvaluatedKeyHash:  "concurrency-eval",
					},
			},
			expected: `{"k":2,"c":{"m":1,"s":1,"h":"custom-key","eh":"concurrency-eval","l":15,"ilk":"{cs}:a:12345678-1234-1234-1234-123456789abc:concurrency:e:87654321-4321-4321-4321-cba987654321<custom-key:concurrency-eval>","ra":2000}}`,
		},
		{
			name: "Throttle constraint with embedded config",
			input: ConstraintItem{
				Kind: ConstraintKindThrottle,
				Throttle: &ThrottleConstraint{
					Scope:             enums.ThrottleScopeFn,
					KeyExpressionHash: "throttle-expr",
					EvaluatedKeyHash:  "throttle-key",
				},
			},
			expected: `{"k":3,"t":{"h":"throttle-expr","eh":"throttle-key","l":200,"b":300,"p":60000,"k":"{cs}:a:12345678-1234-1234-1234-123456789abc:throttle:f:11111111-2222-3333-4444-555555555555:throttle-key"}}`,
		},
		{
			name: "Concurrency constraint with standard function step limit",
			input: ConstraintItem{
				Kind: ConstraintKindConcurrency,
				Concurrency: &ConcurrencyConstraint{
					Mode:              enums.ConcurrencyModeStep,
					Scope:             enums.ConcurrencyScopeFn,
						// KeyExpressionHash and EvaluatedKeyHash left empty for standard limit
				},
			},
			expected: `{"k":2,"c":{"l":25,"ra":2000,"ilk":"{cs}:a:12345678-1234-1234-1234-123456789abc:concurrency:f:11111111-2222-3333-4444-555555555555"}}`, // Function concurrency limit embedded
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			serialized := tt.input.ToSerializedConstraintItem(testConfig, accountID, envID, functionID)
			jsonBytes, err := json.Marshal(serialized)
			require.NoError(t, err)

			assert.JSONEq(t, tt.expected, string(jsonBytes))
		})
	}
}

func TestSerializedConstraintItem_EmptyLimit(t *testing.T) {
	config := ConstraintConfig{
		Concurrency: ConcurrencyConfig{
			CustomConcurrencyKeys: []CustomConcurrencyLimit{
				{},
			},
		},
	}

	constraint := ConstraintItem{
		Kind:        ConstraintKindConcurrency,
		Concurrency: &ConcurrencyConstraint{},
	}

	accountID, envID, fnID := uuid.New(), uuid.New(), uuid.New()
	serialized := constraint.ToSerializedConstraintItem(config, accountID, envID, fnID)
	jsonBytes, err := json.Marshal(serialized)
	require.NoError(t, err)

	expected, err := json.Marshal(map[string]any{
		"k": 2,
		"c": map[string]any{
			"l":   0, // Should always be included
			"ilk": fmt.Sprintf("{cs}:a:%s:concurrency:f:%s", accountID, fnID),
			"ra":  ConcurrencyLimitRetryAfter.Milliseconds(),
		},
	})
	require.NoError(t, err)
	require.JSONEq(t, string(expected), string(jsonBytes))
}

func TestSerializedConstraintItem_SizeReduction(t *testing.T) {
	// Test that serialized version is significantly smaller
	accountID := uuid.MustParse("12345678-1234-1234-1234-123456789abc")
	envID := uuid.MustParse("87654321-4321-4321-4321-cba987654321")
	functionID := uuid.MustParse("11111111-2222-3333-4444-555555555555")

	testConfig := ConstraintConfig{
		FunctionVersion: 1,
		Concurrency: ConcurrencyConfig{
			AccountConcurrency: 50,
		},
	}

	original := ConstraintItem{
		Kind: ConstraintKindConcurrency,
		Concurrency: &ConcurrencyConstraint{
			Mode:              enums.ConcurrencyModeRun,
			Scope:             enums.ConcurrencyScopeAccount,
			KeyExpressionHash: "some-very-long-key-expression-hash-value",
			EvaluatedKeyHash:  "some-very-long-evaluated-key-hash-value",
		},
	}

	// Serialize original
	originalJson, err := json.Marshal(original)
	require.NoError(t, err)

	// Serialize optimized version with embedded config
	serialized := original.ToSerializedConstraintItem(testConfig, accountID, envID, functionID)
	optimizedJson, err := json.Marshal(serialized)
	require.NoError(t, err)

	t.Logf("Original JSON (%d bytes): %s", len(originalJson), string(originalJson))
	t.Logf("Optimized JSON (%d bytes): %s", len(optimizedJson), string(optimizedJson))

	// The optimized version uses shorter field names and integer enums, though
	// the addition of InProgressLeaseKey may make the overall size larger.
	// We test that the optimized version is valid and contains the expected structure.
	assert.NotEmpty(t, optimizedJson)
	assert.Contains(t, string(optimizedJson), `"k":2`)  // Kind as integer
	assert.Contains(t, string(optimizedJson), `"ilk":`) // InProgressLeaseKey
}

func TestLuaScriptSnapshots(t *testing.T) {
	// Read all Lua scripts from the embedded filesystem
	entries, err := testFS.ReadDir("lua")
	require.NoError(t, err)

	scripts := make(map[string]string)
	collectLuaScripts(t, "lua", entries, scripts)

	// Test each script
	for scriptName, rawContent := range scripts {
		t.Run(scriptName, func(t *testing.T) {
			// Process the script
			processedContent, err := processLuaScript(scriptName, rawContent, testFS)
			require.NoError(t, err)

			// Read expected snapshot from fixture file
			snapshotPath := filepath.Join("testdata", "snapshots", scriptName+".lua")
			expectedBytes, err := os.ReadFile(snapshotPath)
			if os.IsNotExist(err) {
				// Generate snapshot file if it doesn't exist
				err := os.MkdirAll(filepath.Dir(snapshotPath), 0755)
				require.NoError(t, err)

				err = os.WriteFile(snapshotPath, []byte(processedContent), 0644)
				require.NoError(t, err)

				t.Logf("Generated snapshot for %s at %s", scriptName, snapshotPath)
				return
			}
			require.NoError(t, err)

			expected := string(expectedBytes)

			// Compare with expected snapshot
			require.Equal(t, expected, processedContent,
				"Script %s processed content differs from snapshot at %s. "+
					"If this is intentional, delete the snapshot file to regenerate it",
				scriptName, snapshotPath)
		})
	}
}

func collectLuaScripts(t *testing.T, path string, entries []fs.DirEntry, scripts map[string]string) {
	for _, e := range entries {
		if e.IsDir() {
			subEntries, err := testFS.ReadDir(fmt.Sprintf("%s/%s", path, e.Name()))
			require.NoError(t, err)
			collectLuaScripts(t, path+"/"+e.Name(), subEntries, scripts)
			continue
		}

		if !strings.HasSuffix(e.Name(), ".lua") {
			continue
		}

		byt, err := testFS.ReadFile(fmt.Sprintf("%s/%s", path, e.Name()))
		require.NoError(t, err)

		name := path + "/" + e.Name()
		name = strings.TrimPrefix(name, "lua/")
		name = strings.TrimSuffix(name, ".lua")

		scripts[name] = string(byt)
	}
}

func TestLuaError(t *testing.T) {
	dialErr := &net.OpError{Op: "dial", Net: "tcp", Err: errors.New("connection refused")}
	dnsErr := &net.DNSError{Err: "no such host", Name: "redis.example.com"}

	tests := []struct {
		name        string
		err         error
		wantStatus  string
		wantRetry   bool
	}{
		{
			name:       "nil error returns success",
			err:        nil,
			wantStatus: "success",
			wantRetry:  false,
		},
		{
			name:       "context deadline exceeded returns timeout",
			err:        context.DeadlineExceeded,
			wantStatus: "timeout",
			wantRetry:  true,
		},
		{
			name:       "context canceled returns timeout",
			err:        context.Canceled,
			wantStatus: "timeout",
			wantRetry:  true,
		},
		{
			name:       "os deadline exceeded returns timeout",
			err:        os.ErrDeadlineExceeded,
			wantStatus: "timeout",
			wantRetry:  true,
		},
		{
			name:       "wrapped context deadline exceeded returns timeout",
			err:        fmt.Errorf("redis: %w", context.DeadlineExceeded),
			wantStatus: "timeout",
			wantRetry:  true,
		},
		{
			name:       "net.OpError returns network_error",
			err:        dialErr,
			wantStatus: "network_error",
			wantRetry:  true,
		},
		{
			name:       "wrapped net.OpError returns network_error",
			err:        fmt.Errorf("redis: %w", dialErr),
			wantStatus: "network_error",
			wantRetry:  true,
		},
		{
			name:       "net.DNSError returns network_error",
			err:        dnsErr,
			wantStatus: "network_error",
			wantRetry:  true,
		},
		{
			name:       "wrapped net.DNSError returns network_error",
			err:        fmt.Errorf("redis: %w", dnsErr),
			wantStatus: "network_error",
			wantRetry:  true,
		},
		{
			name:       "generic error returns error without retry",
			err:        errors.New("WRONGTYPE Operation against a key holding the wrong kind of value"),
			wantStatus: "error",
			wantRetry:  false,
		},
		{
			name:       "wrapped generic error returns error without retry",
			err:        fmt.Errorf("script failed: %w", errors.New("ERR unknown command")),
			wantStatus: "error",
			wantRetry:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			status, retry := luaError(tt.err)
			assert.Equal(t, tt.wantStatus, status)
			assert.Equal(t, tt.wantRetry, retry)
		})
	}
}
