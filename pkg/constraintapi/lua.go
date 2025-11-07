package constraintapi

import (
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/redis/rueidis"
)

//go:embed lua/*
var embedded embed.FS

var (
	// scripts stores all embedded lua scripts on initialization
	scripts = map[string]*rueidis.Lua{}
	include = regexp.MustCompile(`-- \$include\(([\w.]+)\)`)
)

func init() {
	// read the lua scripts
	entries, err := embedded.ReadDir("lua")
	if err != nil {
		panic(fmt.Errorf("error reading redis lua dir: %w", err))
	}
	readRedisScripts("lua", entries)
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
		val := string(byt)

		// Add any includes.
		items := include.FindAllStringSubmatch(val, -1)
		if len(items) > 0 {
			// Replace each include
			for _, include := range items {
				byt, err = embedded.ReadFile(fmt.Sprintf("lua/%s", include[1]))
				if err != nil {
					panic(fmt.Errorf("error reading redis lua include: %w", err))
				}
				val = strings.ReplaceAll(val, include[0], string(byt))
			}
		}
		scripts[name] = rueidis.NewLuaScript(val)
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
	Limit int `json:"l,omitempty"`

	// InProgressLeaseKey represents the Redis key holding the ZSET for this constraint
	InProgressLeaseKey string `json:"ilk"`

	// InProgressItemKey represents the in progress item (concurrency) ZSET key for this constraint
	InProgressItemKey string `json:"iik"`
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
	Limit int `json:"l,omitempty"`

	// b = Burst (embedded from config)
	Burst int `json:"b,omitempty"`

	// p = Period (embedded from config)
	Period int `json:"p,omitempty"`
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
	Limit int `json:"l,omitempty"`

	// p = Period (embedded from config)
	Period int `json:"p,omitempty"`
}

// ToSerializedConstraintItem converts a ConstraintItem to a SerializedConstraintItem
// for efficient storage in Redis and easy consumption in Lua scripts.
// The config parameter is used to embed matching configuration limits directly into the constraint.
func (c ConstraintItem) ToSerializedConstraintItem(
	config ConstraintConfig,
	accountID uuid.UUID,
	envID uuid.UUID,
	functionID uuid.UUID,
	keyPrefix string,
) SerializedConstraintItem {
	serialized := SerializedConstraintItem{}

	// Convert ConstraintKind to integer
	switch c.Kind {
	case ConstraintKindRateLimit:
		serialized.Kind = 1
		if c.RateLimit != nil {
			rateLimitConstraint := &SerializedRateLimitConstraint{
				Scope:             int(c.RateLimit.Scope),
				KeyExpressionHash: c.RateLimit.KeyExpressionHash,
				EvaluatedKeyHash:  c.RateLimit.EvaluatedKeyHash,
			}

			// Find matching rate limit config
			for _, rlConfig := range config.RateLimit {
				if rlConfig.Scope == c.RateLimit.Scope && rlConfig.KeyExpressionHash == c.RateLimit.KeyExpressionHash {
					rateLimitConstraint.Limit = rlConfig.Limit
					// Ensure rate limiting period is encoded as nanoseconds
					rateLimitConstraint.Period = int((time.Duration(rlConfig.Period) * time.Second).Nanoseconds())
					break
				}
			}

			serialized.RateLimit = rateLimitConstraint
		}
	case ConstraintKindConcurrency:
		serialized.Kind = 2
		if c.Concurrency != nil {
			concurrencyConstraint := &SerializedConcurrencyConstraint{
				Mode:               int(c.Concurrency.Mode),
				Scope:              int(c.Concurrency.Scope),
				KeyExpressionHash:  c.Concurrency.KeyExpressionHash,
				EvaluatedKeyHash:   c.Concurrency.EvaluatedKeyHash,
				InProgressItemKey:  c.Concurrency.InProgressItemKey,
				InProgressLeaseKey: c.Concurrency.InProgressLeasesKey(keyPrefix, accountID, envID, functionID),
			}

			// Embed appropriate limit based on scope and mode
			if c.Concurrency.KeyExpressionHash != "" {
				// Custom concurrency key - find matching custom limit
				for _, customLimit := range config.Concurrency.CustomConcurrencyKeys {
					if customLimit.Mode == c.Concurrency.Mode &&
						customLimit.Scope == c.Concurrency.Scope &&
						customLimit.KeyExpressionHash == c.Concurrency.KeyExpressionHash {
						concurrencyConstraint.Limit = customLimit.Limit
						break
					}
				}
			} else {
				// Standard concurrency limits based on scope and mode
				switch c.Concurrency.Scope {
				case 0: // Function scope
					if c.Concurrency.Mode == 0 { // Step mode
						concurrencyConstraint.Limit = config.Concurrency.FunctionConcurrency
					} else { // Run mode
						concurrencyConstraint.Limit = config.Concurrency.FunctionRunConcurrency
					}
				case 2: // Account scope
					if c.Concurrency.Mode == 0 { // Step mode
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
		if c.Throttle != nil {
			throttleConstraint := &SerializedThrottleConstraint{
				Scope:             int(c.Throttle.Scope),
				KeyExpressionHash: c.Throttle.KeyExpressionHash,
				EvaluatedKeyHash:  c.Throttle.EvaluatedKeyHash,
			}

			// Find matching throttle config
			for _, tConfig := range config.Throttle {
				if tConfig.Scope == c.Throttle.Scope && tConfig.ThrottleKeyExpressionHash == c.Throttle.KeyExpressionHash {
					throttleConstraint.Limit = tConfig.Limit
					throttleConstraint.Burst = tConfig.Burst
					throttleConstraint.Period = tConfig.Period
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
