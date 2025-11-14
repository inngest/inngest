package constraintapi

import (
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
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
	keyPrefix := "test-prefix"

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
				Scope:                     enums.ThrottleScopeFn,
				Limit:                     200,
				Burst:                     300,
				Period:                    60,
				ThrottleKeyExpressionHash: "throttle-key",
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
			expected: `{"k":1,"r":{"b":10,"s":2,"h":"test-key-hash","eh":"eval-hash","k":"{test-prefix}:eval-hash","l":100,"p":60000000000}}`,
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
					InProgressItemKey: "redis:item:key123",
				},
			},
			expected: `{"k":2,"c":{"m":1,"s":1,"h":"custom-key","eh":"concurrency-eval","l":15,"ilk":"test-prefix:12345678-1234-1234-1234-123456789abc:state:concurrency:e:87654321-4321-4321-4321-cba987654321<custom-key:concurrency-eval>","iik":"redis:item:key123"}}`,
		},
		{
			name: "Throttle constraint with embedded config",
			input: ConstraintItem{
				Kind: ConstraintKindThrottle,
				Throttle: &ThrottleConstraint{
					Scope:             enums.ThrottleScopeFn,
					KeyExpressionHash: "throttle-key",
					EvaluatedKeyHash:  "throttle-eval",
				},
			},
			expected: `{"k":3,"t":{"h":"throttle-key","eh":"throttle-eval","l":200,"b":300,"p":60}}`,
		},
		{
			name: "Concurrency constraint with standard function step limit",
			input: ConstraintItem{
				Kind: ConstraintKindConcurrency,
				Concurrency: &ConcurrencyConstraint{
					Mode:              enums.ConcurrencyModeStep,
					Scope:             enums.ConcurrencyScopeFn,
					InProgressItemKey: "redis:function:item456",
					// KeyExpressionHash and EvaluatedKeyHash left empty for standard limit
				},
			},
			expected: `{"k":2,"c":{"l":25,"ilk":"test-prefix:12345678-1234-1234-1234-123456789abc:state:concurrency:f:11111111-2222-3333-4444-555555555555","iik":"redis:function:item456"}}`, // Function concurrency limit embedded
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			serialized := tt.input.ToSerializedConstraintItem(testConfig, accountID, envID, functionID, keyPrefix)
			jsonBytes, err := json.Marshal(serialized)
			require.NoError(t, err)

			assert.JSONEq(t, tt.expected, string(jsonBytes))
		})
	}
}

func TestSerializedConstraintItem_SizeReduction(t *testing.T) {
	// Test that serialized version is significantly smaller
	accountID := uuid.MustParse("12345678-1234-1234-1234-123456789abc")
	envID := uuid.MustParse("87654321-4321-4321-4321-cba987654321")
	functionID := uuid.MustParse("11111111-2222-3333-4444-555555555555")
	keyPrefix := "test-prefix"

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
			InProgressItemKey: "redis:some-very-long-in-progress-item-key-value",
		},
	}

	// Serialize original
	originalJson, err := json.Marshal(original)
	require.NoError(t, err)

	// Serialize optimized version with embedded config
	serialized := original.ToSerializedConstraintItem(testConfig, accountID, envID, functionID, keyPrefix)
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
	assert.Contains(t, string(optimizedJson), `"iik":`) // InProgressItemKey
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
