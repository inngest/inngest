// Package schemes maps step URI schemes to the drivers that run them.  a
// process registers its schemes at startup, before it syncs, stores, or runs
// a function.  a process that has not registered a scheme rejects functions
// that use it.
package schemes

import (
	"fmt"
	"net/url"
	"sync"
)

// DriverName returns the name of the driver that runs a step URI.
type DriverName func(u *url.URL) string

var (
	mu         sync.RWMutex
	registered = map[string]DriverName{}
)

// Register adds a step URI scheme.  inngest.Driver returns name(u) for a URI
// with the scheme, function validation accepts it, and app URL normalization
// keeps it, because it is not a network address.  a second Register of one
// scheme panics, because two drivers would claim the same steps.
func Register(scheme string, name DriverName) {
	mu.Lock()
	defer mu.Unlock()
	if _, ok := registered[scheme]; ok {
		panic(fmt.Sprintf("schemes: %q is already registered", scheme))
	}
	registered[scheme] = name
}

// Driver returns the driver name for u.  it returns false when no driver
// registered u's scheme.
func Driver(u *url.URL) (string, bool) {
	mu.RLock()
	name, ok := registered[u.Scheme]
	mu.RUnlock()
	if !ok {
		return "", false
	}
	return name(u), true
}

// Registered reports whether a driver registered scheme.
func Registered(scheme string) bool {
	mu.RLock()
	defer mu.RUnlock()
	_, ok := registered[scheme]
	return ok
}
