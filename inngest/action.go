package inngest

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/inngest/inngest/pkg/consts"
)

// ActionVersion represents a version of an action defined via its cue configuration.
type ActionVersion struct {
	// DSN represents the immutable identifier for the action.
	DSN string `json:"dsn"`

	// Name represents the name of this action
	Name string `json:"name"`

	// Version defines the current action version.  Each action version can have
	// an updated configuration.
	Version *VersionInfo `json:"version,omitempty"`

	// WorkflowMetadata defines workflow-specific configuration for the action.  For example,
	// the "wait" action is uniquely configured within each workflow to wait for some specific
	// amount of time.
	WorkflowMetadata MetadataMap `json:"workflowMetadata"`

	// Response defines the response type for this action.  This allows us to show UI-specific
	// information around the "stack" or "baggage" that is built up around your workflow as
	// actions run.
	Response map[string]Response `json:"response"`

	// Scopes defines the permissions that this action requires to execute.
	Scopes []string `json:"scopes"`

	// Edges define predetermined edges based off of responses for this action.  For example,
	// the webhook action can define some success and error edges for the response.
	Edges map[string]Edge `json:"edges"`

	// Runtime specifies which language/runtime is being used for this action.  This is decoded
	// via the GetRuntime() function call, as we need a specific decoder to
	Runtime RuntimeWrapper `json:"runtime"`

	Retries *RetryOptions `json:"retries,omitempty"`
}

func (av ActionVersion) RetryAttempts() int {
	if av.Retries != nil && av.Retries.Attempts != nil {
		return *av.Retries.Attempts
	}
	return consts.DefaultRetryCount
}

type VersionInfo struct {
	Major uint `json:"major"`
	Minor uint `json:"minor"`
}

func (v VersionInfo) String() string {
	return fmt.Sprintf("%d.%d", v.Major, v.Minor)
}

func (v VersionInfo) Tag() string {
	return fmt.Sprintf("%d-%d", v.Major, v.Minor)
}

// Response represents a value that is returned from the action
type Response struct {
	Name     string `json:"name"`
	Type     string `json:"type"`
	Optional bool   `json:"optional"`
}

type MetadataMap map[string]Metadata

type Metadata struct {
	Name       string      `json:"name"`
	Expression *string     `json:"expression,omitempty"`
	Required   bool        `json:"required"`
	Default    interface{} `json:"default,omitempty"`
	// Type represents the datatype for this particular entry.
	Type string `json:"type"`
	Form Form   `json:"form"`
}

// Form represents form-specific data.  It shares two fields common to each
// form type, and then embedded structs depending on the cue type chosen.
type Form struct {
	Title string `json:"title"`
	Type  string `json:"type"`

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
	Templating bool `json:"templating,omitempty"`
}

type FormInput struct {
	Templating bool `json:"templating,omitempty"`
}

type FormTextarea struct {
	Templating bool `json:"templating,omitempty"`
}

type FormSelect struct {
	Choices []Choice `json:"choices"`
	Eval    *string  `json:"eval"`
}

type Choice struct {
	Name  string      `json:"name"`
	Value interface{} `json:"value"`
}
