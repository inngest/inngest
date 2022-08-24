package env

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/inngest/inngest/pkg/function"
	"github.com/joho/godotenv"
)

var (
	ErrConflict = fmt.Errorf("conflicting env value")
)

// EnvReader returns a map of key <> values from parsed .env files for each function.
type EnvReader interface {
	// Read returns env variables for a specific function.  These variables
	// are plaintext, read from a .env file - useful for local debugging.
	Read(ctx context.Context, fnID string) map[string]string
}

// EnvReader returns a reader which returns parsed env files for
// each function.
type reader struct {
	fns map[string]map[string]string
}

func (e reader) Read(ctx context.Context, id string) map[string]string {
	if e.fns == nil {
		return nil
	}
	env := e.fns[id]
	return env
}

func NewReader(fns []function.Function) (EnvReader, error) {
	r := reader{
		fns: make(map[string]map[string]string),
	}

	for _, fn := range fns {
		var err error
		r.fns[fn.ID], err = ParseDotEnv(fn)
		if err != nil {
			return nil, err
		}
	}

	return r, nil
}

// ParseDotEnvAll parses .env files for each function specfied.
func ParseDotEnvAll(fns []function.Function) (map[string]string, error) {
	env := map[string]string{}
	for _, f := range fns {
		// TODO: Add dotenv-vault as an integration here.
		parsed, err := ParseDotEnv(f)
		if err != nil {
			return nil, err
		}
		for k, v := range parsed {
			if existing, ok := env[k]; ok && existing != v {
				return nil, ErrConflict
			}
			env[k] = v
		}
	}

	return env, nil
}

// ParseDotEnv parses the env file for a specific function.
func ParseDotEnv(f function.Function) (map[string]string, error) {
	parsed, err := godotenv.Read(filepath.Join(f.Dir(), ".env"))
	if os.IsNotExist(err) {
		return nil, nil
	}
	return parsed, err
}
