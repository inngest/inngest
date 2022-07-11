// cuedefs provides cue definitions for configuring functions and events within Inngest.
//
// It also provides an embed.FS which contains the cue definitions for use within Go at
// runtime.
package cuedefs

import (
	"embed"
	"sync"
)

// FS embeds the cue module and definitions.
//
//go:embed cue.mod v1 config
var FS embed.FS

var (
	lock *sync.Mutex
)

func init() {
	lock = &sync.Mutex{}
}

// Unfortunately, cue is not thread safe.  We only parse cue when reading and validating
// configuration;  parsed functions and workflows are cached.  We add a mutex here
// to prevent concurrent access to Cue right now.
//
// Lock claims the mutex.
func Lock() {
	lock.Lock()
}

// Unlock releases the mutex.
func Unlock() {
	lock.Unlock()
}
