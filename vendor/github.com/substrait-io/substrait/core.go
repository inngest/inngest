// Package substrait provides access to Substrait artifacts via embed.FS.
// Use substrait.GetSubstraitFS() to retrieve the embed.FS object.
package substrait

import "embed"

//go:embed extensions/*
var substraitExtensionsFS embed.FS

func GetSubstraitFS() embed.FS {
	return substraitExtensionsFS
}

func GetSubstraitExtensionsFS() embed.FS {
	return substraitExtensionsFS
}

//go:embed tests/cases/*/*.test
var substraitTestsFS embed.FS

func GetSubstraitTestsFS() embed.FS {
	return substraitTestsFS
}
