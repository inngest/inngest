// Package dnscache is a Go package for caching DNS lookup results in memory.
// It asynchronously lookups DNS and refreshes results. The main motivation of this package
// is to avoid too much DNS lookups for every request (DNS lookup sometimes makes request
// really slow and causes error). This can be mainly used for the targets which are
// running on non-dynamic environment where IP does not change often.
package dnscache // import "go.mercari.io/go-dnscache"
