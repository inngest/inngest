package constraintapi

import (
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"regexp"
	"strconv"
	"strings"

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
}

// SerializedThrottleConstraint represents a minimal version of ThrottleConstraint
type SerializedThrottleConstraint struct {
	// s = Scope as integer: 0=Fn, 1=Env, 2=Account
	Scope int `json:"s,omitempty"`

	// h = KeyExpressionHash
	KeyExpressionHash string `json:"h,omitempty"`

	// eh = EvaluatedKeyHash
	EvaluatedKeyHash string `json:"eh,omitempty"`
}

// SerializedRateLimitConstraint represents a minimal version of RateLimitConstraint
type SerializedRateLimitConstraint struct {
	// s = Scope as integer: 0=Fn, 1=Env, 2=Account
	Scope int `json:"s,omitempty"`

	// h = KeyExpressionHash
	KeyExpressionHash string `json:"h,omitempty"`

	// eh = EvaluatedKeyHash
	EvaluatedKeyHash string `json:"eh,omitempty"`
}

// ToSerializedConstraintItem converts a ConstraintItem to a SerializedConstraintItem
// for efficient storage in Redis and easy consumption in Lua scripts.
func (c ConstraintItem) ToSerializedConstraintItem() SerializedConstraintItem {
	serialized := SerializedConstraintItem{}

	// Convert ConstraintKind to integer
	switch c.Kind {
	case ConstraintKindRateLimit:
		serialized.Kind = 1
		if c.RateLimit != nil {
			serialized.RateLimit = &SerializedRateLimitConstraint{
				Scope:             int(c.RateLimit.Scope),
				KeyExpressionHash: c.RateLimit.KeyExpressionHash,
				EvaluatedKeyHash:  c.RateLimit.EvaluatedKeyHash,
			}
		}
	case ConstraintKindConcurrency:
		serialized.Kind = 2
		if c.Concurrency != nil {
			serialized.Concurrency = &SerializedConcurrencyConstraint{
				Mode:              int(c.Concurrency.Mode),
				Scope:             int(c.Concurrency.Scope),
				KeyExpressionHash: c.Concurrency.KeyExpressionHash,
				EvaluatedKeyHash:  c.Concurrency.EvaluatedKeyHash,
			}
		}
	case ConstraintKindThrottle:
		serialized.Kind = 3
		if c.Throttle != nil {
			serialized.Throttle = &SerializedThrottleConstraint{
				Scope:             int(c.Throttle.Scope),
				KeyExpressionHash: c.Throttle.KeyExpressionHash,
				EvaluatedKeyHash:  c.Throttle.EvaluatedKeyHash,
			}
		}
	}

	return serialized
}

// SerializedConstraintConfig represents a minimal, Lua-friendly version of ConstraintConfig
// with short JSON field names and integer enums to reduce Redis storage size.
type SerializedConstraintConfig struct {
	// v = FunctionVersion
	FunctionVersion int `json:"v,omitempty"`

	// r = RateLimit configs
	RateLimit []SerializedRateLimitConfig `json:"r,omitempty"`

	// c = Concurrency config
	Concurrency SerializedConcurrencyConfig `json:"c,omitempty"`

	// t = Throttle configs
	Throttle []SerializedThrottleConfig `json:"t,omitempty"`
}

// SerializedRateLimitConfig represents a minimal version of RateLimitConfig
type SerializedRateLimitConfig struct {
	// s = Scope as integer: 0=Fn, 1=Env, 2=Account
	Scope int `json:"s,omitempty"`

	// l = Limit
	Limit int `json:"l,omitempty"`

	// p = Period
	Period string `json:"p,omitempty"`

	// h = KeyExpressionHash
	KeyExpressionHash string `json:"h,omitempty"`
}

// SerializedConcurrencyConfig represents a minimal version of ConcurrencyConfig
type SerializedConcurrencyConfig struct {
	// ac = AccountConcurrency
	AccountConcurrency int `json:"ac,omitempty"`

	// fc = FunctionConcurrency
	FunctionConcurrency int `json:"fc,omitempty"`

	// arc = AccountRunConcurrency
	AccountRunConcurrency int `json:"arc,omitempty"`

	// frc = FunctionRunConcurrency
	FunctionRunConcurrency int `json:"frc,omitempty"`

	// cck = CustomConcurrencyKeys
	CustomConcurrencyKeys []SerializedCustomConcurrencyLimit `json:"cck,omitempty"`
}

// SerializedCustomConcurrencyLimit represents a minimal version of CustomConcurrencyLimit
type SerializedCustomConcurrencyLimit struct {
	// m = Mode as integer: 0=Step, 1=Run
	Mode int `json:"m,omitempty"`

	// s = Scope as integer: 0=Fn, 1=Env, 2=Account
	Scope int `json:"s,omitempty"`

	// l = Limit
	Limit int `json:"l,omitempty"`

	// h = KeyExpressionHash
	KeyExpressionHash string `json:"h,omitempty"`
}

// SerializedThrottleConfig represents a minimal version of ThrottleConfig
type SerializedThrottleConfig struct {
	// s = Scope as integer: 0=Fn, 1=Env, 2=Account
	Scope int `json:"s,omitempty"`

	// l = Limit
	Limit int `json:"l,omitempty"`

	// b = Burst
	Burst int `json:"b,omitempty"`

	// p = Period
	Period int `json:"p,omitempty"`

	// h = ThrottleKeyExpressionHash
	ThrottleKeyExpressionHash string `json:"h,omitempty"`
}

// ToSerializedConstraintConfig converts a ConstraintConfig to a SerializedConstraintConfig
// for efficient storage in Redis and easy consumption in Lua scripts.
func (c ConstraintConfig) ToSerializedConstraintConfig() SerializedConstraintConfig {
	serialized := SerializedConstraintConfig{
		FunctionVersion: c.FunctionVersion,
	}

	// Convert RateLimit configs
	if len(c.RateLimit) > 0 {
		serialized.RateLimit = make([]SerializedRateLimitConfig, len(c.RateLimit))
		for i, rl := range c.RateLimit {
			serialized.RateLimit[i] = SerializedRateLimitConfig{
				Scope:             int(rl.Scope),
				Limit:             rl.Limit,
				Period:            rl.Period,
				KeyExpressionHash: rl.KeyExpressionHash,
			}
		}
	}

	// Convert Concurrency config
	serialized.Concurrency = SerializedConcurrencyConfig{
		AccountConcurrency:     c.Concurrency.AccountConcurrency,
		FunctionConcurrency:    c.Concurrency.FunctionConcurrency,
		AccountRunConcurrency:  c.Concurrency.AccountRunConcurrency,
		FunctionRunConcurrency: c.Concurrency.FunctionRunConcurrency,
	}

	// Convert CustomConcurrencyKeys
	if len(c.Concurrency.CustomConcurrencyKeys) > 0 {
		serialized.Concurrency.CustomConcurrencyKeys = make([]SerializedCustomConcurrencyLimit, len(c.Concurrency.CustomConcurrencyKeys))
		for i, cck := range c.Concurrency.CustomConcurrencyKeys {
			serialized.Concurrency.CustomConcurrencyKeys[i] = SerializedCustomConcurrencyLimit{
				Mode:              int(cck.Mode),
				Scope:             int(cck.Scope),
				Limit:             cck.Limit,
				KeyExpressionHash: cck.KeyExpressionHash,
			}
		}
	}

	// Convert Throttle configs
	if len(c.Throttle) > 0 {
		serialized.Throttle = make([]SerializedThrottleConfig, len(c.Throttle))
		for i, t := range c.Throttle {
			serialized.Throttle[i] = SerializedThrottleConfig{
				Scope:                     int(t.Scope),
				Limit:                     t.Limit,
				Burst:                     t.Burst,
				Period:                    t.Period,
				ThrottleKeyExpressionHash: t.ThrottleKeyExpressionHash,
			}
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
