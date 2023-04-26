package actions

import (
	"fmt"
	"strings"

	"github.com/inngest/inngest/internal/cuedefs"
	"github.com/inngest/inngest/pkg/inngest"
)

// Parse parses an action.  This differs from inngest.ParseAction as we automatically
// prefix the action DSN with the given account identifier if not present.
func Parse(accountPrefix, input string) (version *inngest.ActionVersion, formatted string, err error) {
	version, err = cuedefs.ParseAction(input)

	if err != nil || strings.Contains(version.DSN, "/") {
		return version, input, err
	}

	// We have an action, but there was an error parsing.  There's
	// one situation in which we allow the local CLI to parse invalid
	// actions:  creating a .cue config with no account prefix in the
	// action DSN.
	//
	// We don't want you to have to put your account prefix in the DSN:
	// "http" instead of "funky-albatross-81236/http".  This makes it easy
	// for people to clone & push actions without having to change the DSN.
	//
	// Here, check to see if the action DSN has a slash in it.  If not,
	// add our account prefix and re-serialize the action cue.
	//
	// XXX: this is a code smell.  in v2, reformat the account cue config
	//      to make this nicer (remove account identifiers / separate, etc).
	if !strings.Contains(version.DSN, "/") {
		version.DSN = fmt.Sprintf("%s/%s", accountPrefix, version.DSN)
	}

	// Re-format the cue config and return the newly formatted data.
	formatted, err = cuedefs.FormatAction(*version)
	return version, formatted, err
}
