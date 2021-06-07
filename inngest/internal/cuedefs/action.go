package cuedefs

import "cuelang.org/go/cue"

const (
	actionConst     = "action"
	actionDefSuffix = "action: actions.#Action"
)

// ParseAction parses a cue configuration defining an action.  It returns the
// *cue.Value of a given valid action, or an error on failure.
func ParseAction(input string) (*cue.Value, error) {
	return parseDef(input, actionConst, actionDefSuffix)
}
