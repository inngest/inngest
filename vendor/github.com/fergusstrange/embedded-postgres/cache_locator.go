package embeddedpostgres

import (
	"fmt"
	"os"
	"path/filepath"
)

// CacheLocator retrieves the location of the Postgres binary cache returning it to location.
// The result of whether this cache is present will be returned to exists.
type CacheLocator func() (location string, exists bool)

func defaultCacheLocator(versionStrategy VersionStrategy) CacheLocator {
	return func() (string, bool) {
		cacheDirectory := ".embedded-postgres-go"
		if userHome, err := os.UserHomeDir(); err == nil {
			cacheDirectory = filepath.Join(userHome, ".embedded-postgres-go")
		}

		operatingSystem, architecture, version := versionStrategy()
		cacheLocation := filepath.Join(cacheDirectory,
			fmt.Sprintf("embedded-postgres-binaries-%s-%s-%s.txz",
				operatingSystem,
				architecture,
				version))

		info, err := os.Stat(cacheLocation)

		if err != nil {
			return cacheLocation, os.IsExist(err) && !info.IsDir()
		}

		return cacheLocation, !info.IsDir()
	}
}
