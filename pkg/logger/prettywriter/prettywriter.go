package prettywriter

import "io"

type prettywriter struct {
	Out io.Writer

	NoColor bool
}
