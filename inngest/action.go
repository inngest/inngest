package inngest

import (
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strings"

	"github.com/inngest/inngestctl/inngest/internal/cuedefs"
)

// ParseAction parses a cue configuration defining an action.
func ParseAction(input string) (*ActionVersion, error) {
	val, err := cuedefs.ParseAction(input)
	if err != nil {
		return nil, fmt.Errorf("error parsing action definition: %w", err)
	}
	a := &ActionVersion{}
	if err := val.Decode(&a); err != nil {
		return nil, fmt.Errorf("error deserializing action version: %w", err)
	}
	return a, nil
}

func FormatAction(a ActionVersion) (string, error) {
	def, err := cuedefs.FormatDef(a)
	if err != nil {
		return "", err
	}
	// XXX: Inspect cue and implement packages.
	return fmt.Sprintf(packageTpl, def), nil
}

// ActionVersion represents a version of an action defined via its cue configuration.
type ActionVersion struct {
	// DSN represents the immutable identifier for the action.
	DSN string
	// Name represents the name of this action
	Name string

	// Version defines the current action version.  Each action version can have
	// an updated configuration.
	Version VersionInfo

	// WorkflowMetadata defines workflow-specific configuration for the action.  For example,
	// the "wait" action is uniquely configured within each workflow to wait for some specific
	// amount of time.
	WorkflowMetadata MetadataMap

	// Response defines the response type for this action.  This allows us to show UI-specific
	// information around the "stack" or "baggage" that is built up around your workflow as
	// actions run.
	Response map[string]Response

	// Edges define predetermined edges based off of responses for this action.  For example,
	// the webhook action can define some success and error edges for the response.
	Edges map[string]Edge

	// Runtime specifies which language/runtime is being used for this action.  This is decoded
	// via the GetRuntime() function call, as we need a specific decoder to
	Runtime RuntimeWrapper
}

type RuntimeWrapper struct {
	Runtime
}

func (r *RuntimeWrapper) UnmarshalJSON(b []byte) error {
	// XXX: This is wasteful, as we decode the runtime twice.  We can implement a custom decoder
	// which decodes and fills in one pass.
	interim := map[string]interface{}{}
	if err := json.Unmarshal(b, &interim); err != nil {
		return err
	}
	typ, ok := interim["type"]
	if !ok {
		return errors.New("unknown type")
	}

	switch typ {
	case "docker":
		docker := RuntimeDocker{}
		if err := json.Unmarshal(b, &docker); err != nil {
			return err
		}
		*&r.Runtime = docker
		return nil
	}

	return nil
}

type Runtime interface {
	RuntimeType() string
}

type RuntimeDocker struct {
	Image      string
	Entrypoint *string
}

type VersionInfo struct {
	Major int
	Minor int
}

// Response represents a value that is returned from the action
type Response struct {
	Name     string `json:"name"`
	Type     string `json:"type"`
	Optional bool   `json:"optional"`
}

func (RuntimeDocker) RuntimeType() string {
	return "docker"
}

type Edge struct {
	Type string `json:"type"`
	Name string `json:"name"`
	If   string `json:"if"`

	// AsyncEdge must be specified if Type is enums.EdgeTypeAsync.
	*AsyncEdge `json:"async,omitempty"`
}

func (e Edge) IsZero() bool {
	return e == Edge{}
}

type AsyncEdge struct {
	TTL string `json:"ttl"`

	// Event specifies the event name to listen for, which can coninue this workflow.
	Event string `json:"event"`

	// Match represents the optional expression to use when matching the event.
	// If specified, the event name must match and this expression must evaluate
	// to true for the workflow to continue.  This allows you to filter events
	// to eg. the same user.
	Match *string `json:"match"`
}

type MetadataMap map[string]Metadata

type Metadata struct {
	Name       string
	Expression *string
	Required   bool
	Default    interface{}
	// Type represents the datatype for this particular entry.
	Type string
	Form Form
}

// Form represents form-specific data.  It shares two fields common to each
// form type, and then embedded structs depending on the cue type chosen.
type Form struct {
	Title string
	Type  string

	// By embedding each form type we can leverage the builtin decoder to
	// properly initialize the correct Form struct.
	//
	// TODO (tonyhb): use a decoder and make Form an interface, then decode
	// the form values using UnmarshalJSON to remove this.
	*FormInput    `json:",omitempty"`
	*FormDateTime `json:",omitempty"`
	*FormTextarea `json:",omitempty"`
	*FormSelect   `json:",omitempty"`
}

// Map returns the form information as a map for GraphQL
func (f Form) Map() map[string]interface{} {
	ret := map[string]interface{}{}
	v := reflect.ValueOf(f)
	t := v.Type()

	for i := 0; i < v.NumField(); i++ {
		value := v.Field(i)

		if value.Kind() == reflect.Ptr && value.IsNil() {
			continue
		}

		field := t.Field(i)
		ret[strings.ToLower(field.Name)] = value.Interface()
	}

	return ret
}

type FormDateTime struct {
	Templating bool
}

type FormInput struct {
	Templating bool
}

type FormTextarea struct {
	Templating bool
}

type FormSelect struct {
	Choices []Choice `json:"choices"`
	Eval    *string  `json:"eval"`
}

type Choice struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

const packageTpl = `package main

import (
	"inngest.com/actions"
)

%s
`
