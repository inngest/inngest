// cuedefs provides cue definitions for configuring functions and events within Inngest.
//
// It also provides an embed.FS which contains the cue definitions for use within Go at
// runtime.
package cuedefs

import "embed"

// FS embeds the cue module and definitions.
//
//go:embed cue.mod v1
var FS embed.FS
