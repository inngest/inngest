package config

import (
	"os"
	"strings"
)

func IsEnvVarFalsy(key string) bool {
	val := strings.ToLower(os.Getenv(key))
	return val == "false" || val == "0"
}
