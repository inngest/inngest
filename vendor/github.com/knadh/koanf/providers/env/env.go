// Package env implements a koanf.Provider that reads environment
// variables as conf maps.
package env

import (
	"errors"
	"os"
	"strings"

	"github.com/knadh/koanf/maps"
)

// Env implements an environment variables provider.
type Env struct {
	prefix string
	delim  string
	cb     func(key string, value string) (string, interface{})
}

// Provider returns an environment variables provider that returns
// a nested map[string]interface{} of environment variable where the
// nesting hierarchy of keys is defined by delim. For instance, the
// delim "." will convert the key `parent.child.key: 1`
// to `{parent: {child: {key: 1}}}`.
//
// If prefix is specified (case-sensitive), only the env vars with
// the prefix are captured. cb is an optional callback that takes
// a string and returns a string (the env variable name) in case
// transformations have to be applied, for instance, to lowercase
// everything, strip prefixes and replace _ with . etc.
// If the callback returns an empty string, the variable will be
// ignored.
func Provider(prefix, delim string, cb func(s string) string) *Env {
	e := &Env{
		prefix: prefix,
		delim:  delim,
	}
	if cb != nil {
		e.cb = func(key string, value string) (string, interface{}) {
			return cb(key), value
		}
	}
	return e
}

// ProviderWithValue works exactly the same as Provider except the callback
// takes a (key, value) with the variable name and value and allows you
// to modify both. This is useful for cases where you may want to return
// other types like a string slice instead of just a string.
func ProviderWithValue(prefix, delim string, cb func(key string, value string) (string, interface{})) *Env {
	return &Env{
		prefix: prefix,
		delim:  delim,
		cb:     cb,
	}
}

// ReadBytes is not supported by the env provider.
func (e *Env) ReadBytes() ([]byte, error) {
	return nil, errors.New("env provider does not support this method")
}

// Read reads all available environment variables into a key:value map
// and returns it.
func (e *Env) Read() (map[string]interface{}, error) {
	// Collect the environment variable keys.
	var keys []string
	for _, k := range os.Environ() {
		if e.prefix != "" {
			if strings.HasPrefix(k, e.prefix) {
				keys = append(keys, k)
			}
		} else {
			keys = append(keys, k)
		}
	}

	mp := make(map[string]interface{})
	for _, k := range keys {
		parts := strings.SplitN(k, "=", 2)

		// If there's a transformation callback,
		// run it through every key/value.
		if e.cb != nil {
			key, value := e.cb(parts[0], parts[1])
			// If the callback blanked the key, it should be omitted
			if key == "" {
				continue
			}
			mp[key] = value
		} else {
			mp[parts[0]] = parts[1]
		}

	}

	if e.delim != "" {
		return maps.Unflatten(mp, e.delim), nil
	}

	return mp, nil
}
