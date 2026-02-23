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
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/telemetry/metrics"
	"github.com/inngest/inngest/pkg/util/errs"
	"github.com/jonboulle/clockwork"
	"github.com/redis/rueidis"
)

//go:embed lua/*
var embedded embed.FS

var (
	// scripts stores all embedded lua scripts on initialization
	scripts              = map[string]*rueidis.Lua{}
	include              = regexp.MustCompile(`(?m)^-- \$include\(([\w./]+)\)$`)
	langServerAnnotation = regexp.MustCompile(`(?m)^---@.*$|---@[^\n]*`)
	comments             = regexp.MustCompile(`(?m)^--.*$|--[^\n]*`)
	emptyLines           = regexp.MustCompile(`(?m)^\s*$`)
)

func init() {
	// read the lua scripts
	entries, err := embedded.ReadDir("lua")
	if err != nil {
		panic(fmt.Errorf("error reading redis lua dir: %w", err))
	}
	readRedisScripts("lua", entries)
}

// processLuaScript processes a single Lua script by handling includes, removing language server annotations, comments, and empty lines
func processLuaScript(name, content string, fs embed.FS) (string, error) {
	val := content

	// Add any includes.
	items := include.FindAllStringSubmatch(val, -1)
	if len(items) > 0 {
		// Replace each include
		for _, include := range items {
			byt, err := fs.ReadFile(fmt.Sprintf("lua/%s", include[1]))
			if err != nil {
				return "", fmt.Errorf("error reading redis lua include: %w", err)
			}
			val = strings.ReplaceAll(val, include[0], string(byt))
		}
	}

	// Remove language server annotations (lines starting with ---@)
	val = langServerAnnotation.ReplaceAllString(val, "")

	// Remove comments (lines starting with --)
	val = comments.ReplaceAllString(val, "")

	// Remove empty lines
	val = emptyLines.ReplaceAllString(val, "")

	// Clean up multiple consecutive newlines
	val = regexp.MustCompile(`\n\n+`).ReplaceAllString(val, "\n")

	// Trim leading/trailing whitespace
	val = strings.TrimSpace(val)

	return val, nil
}

func readRedisScripts(path string, entries []fs.DirEntry) {
	for _, e := range entries {
		// NOTE: When using embed go always uses forward slashes as a path
		// prefix. filepath.Join uses OS-specific prefixes which fails on
		// windows, so we construct the path using Sprintf for all platforms
		if e.IsDir() {
			entries, _ := embedded.ReadDir(fmt.Sprintf("%s/%s", path, e.Name()))
			readRedisScripts(path+"/"+e.Name(), entries)
			continue
		}

		byt, err := embedded.ReadFile(fmt.Sprintf("%s/%s", path, e.Name()))
		if err != nil {
			panic(fmt.Errorf("error reading redis lua script: %w", err))
		}

		name := path + "/" + e.Name()
		name = strings.TrimPrefix(name, "lua/")
		name = strings.TrimSuffix(name, ".lua")

		processedScript, err := processLuaScript(name, string(byt), embedded)
		if err != nil {
			panic(fmt.Errorf("error processing lua script %s: %w", name, err))
		}

		scripts[name] = rueidis.NewLuaScript(processedScript)
	}
}

// SerializedConstraintItem represents a minimal, Lua-friendly version of ConstraintItem
// with short JSON field names and integer enums to reduce Redis storage size.
type SerializedConstraintItem struct {
	// k = Kind as integer: 1=rate_limit, 2=concurrency, 3=throttle
	Kind int `json:"k"`

	// Concurrency constraint (only populated when Kind=2)
	Concurrency *SerializedConcurrencyConstraint `json:"c,omitempty"`

	// Throttle constraint (only populated when Kind=3)
	Throttle *SerializedThrottleConstraint `json:"t,omitempty"`

	// RateLimit constraint (only populated when Kind=1)
	RateLimit *SerializedRateLimitConstraint `json:"r,omitempty"`
}

// SerializedConcurrencyConstraint represents a minimal version of ConcurrencyConstraint
type SerializedConcurrencyConstraint struct {
	// m = Mode as integer: 0=Step, 1=Run
	Mode int `json:"m,omitempty"`

	// s = Scope as integer: 0=Fn, 1=Env, 2=Account
	Scope int `json:"s,omitempty"`

	// h = KeyExpressionHash
	KeyExpressionHash string `json:"h,omitempty"`

	// eh = EvaluatedKeyHash
	EvaluatedKeyHash string `json:"eh,omitempty"`

	// l = Limit (embedded from config)
	Limit int `json:"l"`

	// InProgressLeaseKey represents the Redis key holding the ZSET for this constraint
	InProgressLeaseKey string `json:"ilk"`

	// RetryAfterMS determines the retry duration in milliseconds if this concurrency constraint is limiting
	RetryAfterMS int `json:"ra,omitempty"`
}

// SerializedThrottleConstraint represents a minimal version of ThrottleConstraint
type SerializedThrottleConstraint struct {
	// s = Scope as integer: 0=Fn, 1=Env, 2=Account
	Scope int `json:"s,omitempty"`

	// h = KeyExpressionHash
	KeyExpressionHash string `json:"h,omitempty"`

	// eh = EvaluatedKeyHash
	EvaluatedKeyHash string `json:"eh,omitempty"`

	// l = Limit (embedded from config)
	Limit int `json:"l"`

	// b = Burst (embedded from config)
	Burst int `json:"b"`

	// p = Period in ms (embedded from config)
	Period int `json:"p"`

	// k = Key (fully-qualified Redis key)
	Key string `json:"k,omitempty"`
}

// SerializedRateLimitConstraint represents a minimal version of RateLimitConstraint
type SerializedRateLimitConstraint struct {
	// s = Scope as integer: 0=Fn, 1=Env, 2=Account
	Scope int `json:"s,omitempty"`

	// h = KeyExpressionHash
	KeyExpressionHash string `json:"h,omitempty"`

	// eh = EvaluatedKeyHash
	EvaluatedKeyHash string `json:"eh,omitempty"`

	// l = Limit (embedded from config)
	Limit int `json:"l"`

	// p = Period in ns (embedded from config)
	Period int `json:"p"`

	// b = Burst (embedded from config)
	Burst int `json:"b"`

	// k = Key (fully-qualified Redis key: concatenated evaluated key hash with prefix)
	Key string `json:"k,omitempty"`
}

// ToSerializedConstraintItem converts a ConstraintItem to a SerializedConstraintItem
// for efficient storage in Redis and easy consumption in Lua scripts.
// The config parameter is used to embed matching configuration limits directly into the constraint.
func (ci ConstraintItem) ToSerializedConstraintItem(
	config ConstraintConfig,
	accountID uuid.UUID,
	envID uuid.UUID,
	functionID uuid.UUID,
) SerializedConstraintItem {
	serialized := SerializedConstraintItem{}

	// Convert ConstraintKind to integer
	switch ci.Kind {
	case ConstraintKindRateLimit:
		serialized.Kind = 1
		if ci.RateLimit != nil {
			rateLimitConstraint := &SerializedRateLimitConstraint{
				Scope:             int(ci.RateLimit.Scope),
				KeyExpressionHash: ci.RateLimit.KeyExpressionHash,
				EvaluatedKeyHash:  ci.RateLimit.EvaluatedKeyHash,
				Key:               ci.RateLimit.StateKey(accountID, envID, functionID),
			}

			// Find matching rate limit config
			for _, rlConfig := range config.RateLimit {
				if rlConfig.Scope == ci.RateLimit.Scope && rlConfig.KeyExpressionHash == ci.RateLimit.KeyExpressionHash {
					rateLimitConstraint.Limit = rlConfig.Limit
					rateLimitConstraint.Burst = int(rlConfig.Limit / 10)
					// Ensure rate limiting period is encoded as nanoseconds
					rateLimitConstraint.Period = int((time.Duration(rlConfig.Period) * time.Second).Nanoseconds())
					break
				}
			}

			serialized.RateLimit = rateLimitConstraint
		}
	case ConstraintKindConcurrency:
		serialized.Kind = 2
		if ci.Concurrency != nil {
			concurrencyConstraint := &SerializedConcurrencyConstraint{
				Mode:               int(ci.Concurrency.Mode),
				Scope:              int(ci.Concurrency.Scope),
				KeyExpressionHash:  ci.Concurrency.KeyExpressionHash,
				EvaluatedKeyHash:   ci.Concurrency.EvaluatedKeyHash,
				InProgressLeaseKey: ci.Concurrency.InProgressLeasesKey(accountID, envID, functionID),
				RetryAfterMS:       int(ci.Concurrency.RetryAfter().Milliseconds()),
			}

			// Embed appropriate limit based on scope and mode
			if ci.Concurrency.KeyExpressionHash != "" {
				// Custom concurrency key - find matching custom limit
				for _, customLimit := range config.Concurrency.CustomConcurrencyKeys {
					if customLimit.Mode == ci.Concurrency.Mode &&
						customLimit.Scope == ci.Concurrency.Scope &&
						customLimit.KeyExpressionHash == ci.Concurrency.KeyExpressionHash {
						concurrencyConstraint.Limit = customLimit.Limit
						break
					}
				}
			} else {
				// Standard concurrency limits based on scope and mode
				switch ci.Concurrency.Scope {
				case 0: // Function scope
					if ci.Concurrency.Mode == 0 { // Step mode
						concurrencyConstraint.Limit = config.Concurrency.FunctionConcurrency
					} else { // Run mode
						concurrencyConstraint.Limit = config.Concurrency.FunctionRunConcurrency
					}
				case 2: // Account scope
					if ci.Concurrency.Mode == 0 { // Step mode
						concurrencyConstraint.Limit = config.Concurrency.AccountConcurrency
					} else { // Run mode
						concurrencyConstraint.Limit = config.Concurrency.AccountRunConcurrency
					}
				}
			}

			serialized.Concurrency = concurrencyConstraint
		}
	case ConstraintKindThrottle:
		serialized.Kind = 3
		if ci.Throttle != nil {
			throttleConstraint := &SerializedThrottleConstraint{
				Scope:             int(ci.Throttle.Scope),
				KeyExpressionHash: ci.Throttle.KeyExpressionHash,
				EvaluatedKeyHash:  ci.Throttle.EvaluatedKeyHash,
				Key:               ci.Throttle.StateKey(accountID, envID, functionID),
			}

			// Find matching throttle config
			for _, tConfig := range config.Throttle {
				if tConfig.Scope == ci.Throttle.Scope && tConfig.KeyExpressionHash == ci.Throttle.KeyExpressionHash {
					throttleConstraint.Limit = tConfig.Limit
					throttleConstraint.Burst = tConfig.Burst
					throttleConstraint.Period = tConfig.Period * 1000 // Convert seconds to milliseconds
					break
				}
			}

			serialized.Throttle = throttleConstraint
		}
	}

	return serialized
}

func strSlice(args []any) ([]string, error) {
	res := make([]string, len(args))
	for i, item := range args {
		if s, ok := item.(fmt.Stringer); ok {
			res[i] = s.String()
			continue
		}

		switch v := item.(type) {
		case string:
			res[i] = v
		case []byte:
			res[i] = rueidis.BinaryString(v)
		case int:
			res[i] = strconv.Itoa(v)
		case bool:
			// Use 1 and 0 to signify true/false.
			if v {
				res[i] = "1"
			} else {
				res[i] = "0"
			}
		default:
			byt, err := json.Marshal(item)
			if err != nil {
				return nil, err
			}
			res[i] = rueidis.BinaryString(byt)
		}
	}
	return res, nil
}

func isTimeout(err error) bool {
	if errors.Is(err, os.ErrDeadlineExceeded) {
		return true
	}

	if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
		return true
	}

	return false
}

func isNetworkError(err error) bool {
	var opErr *net.OpError
	if errors.As(err, &opErr) {
		return true
	}
	var dnsErr *net.DNSError
	if errors.As(err, &dnsErr) {
		return true
	}
	return false
}

func executeLuaScript(
	ctx context.Context,
	name string,
	shard string,
	source LeaseSource,
	client rueidis.Client,
	clock clockwork.Clock,
	keys []string,
	args []string,
) ([]byte, errs.InternalError) {
	// Get current time for duration metrics
	start := clock.Now()

	// Execute script and convert response to bytes (we return JSON from all scripts)
	rawRes, err := scripts[name].Exec(ctx, client, keys, args).AsBytes()

	status, retry := luaError(err)

	// Report duration
	metrics.HistogramConstraintAPILuaScriptDuration(ctx, clock.Since(start), metrics.HistogramOpt{
		PkgName: pkgName,
		Tags: map[string]any{
			"operation":       name,
			"status":          status,
			"source_location": source.Location.String(),
			"source_service":  source.Service.String(),
			"shard":           shard,
		},
	})

	if err != nil {
		return nil, errs.Wrap(0, retry, "%s script failed: %w", name, err)
	}

	return rawRes, nil
}

func luaError(err error) (status string, retry bool) {
	if isTimeout(err) {
		return "timeout", true
	}
	if isNetworkError(err) {
		return "network_error", true
	}
	if err != nil {
		return "error", false
	}
	return "success", false
}
