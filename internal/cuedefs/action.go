package cuedefs

import (
	"bytes"
	"fmt"

	"cuelang.org/go/cue"
	cueerrors "cuelang.org/go/cue/errors"
	"github.com/inngest/inngest/inngest"
)

const (
	actionConst     = "action"
	actionDefSuffix = "action: actions.#Action"
)

func ParseAction(input string) (*inngest.ActionVersion, error) {
	val, err := ReadAction(input)
	if err != nil {
		return nil, err
	}
	a := &inngest.ActionVersion{}
	if err := val.Decode(a); err != nil {
		buf := &bytes.Buffer{}
		cueerrors.Print(buf, err, nil)
		return nil, fmt.Errorf("error parsing config: %s", buf.String())
	}
	return a, nil
}

// ReadAction parses a cue configuration defining an action.  It returns
// the cue.Value of the given action.
func ReadAction(input string) (*cue.Value, error) {
	return parseDef(input, actionConst, actionDefSuffix)
}

func FormatAction(a inngest.ActionVersion) (string, error) {
	def, err := FormatDef(a)
	if err != nil {
		return "", err
	}
	// XXX: Inspect cue and implement packages.
	return fmt.Sprintf(packageTpl, def), nil
}

const packageTpl = `package main

import (
	"inngest.com/actions"
)

action: actions.#Action
action: %s`
