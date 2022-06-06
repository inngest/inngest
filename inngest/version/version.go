package version

import "fmt"

var (
	Version = "dev"
	Hash    = ""
)

func Print() string {
	return fmt.Sprintf("%s-%s", Version, Hash)
}
