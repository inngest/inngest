package function

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/inngest/cuetypescript"
	"github.com/inngest/event-schemas/events/marshalling/jsonschema"
)

// DefinitionFormat specifies how the event is typed.
type DefinitionFormat string

const (
	// FormatCue specifies that the event type is written using Cue.
	FormatCue DefinitionFormat = "cue"
	// FormatJSONSchema specifies that the event type is valid JSON Schema
	FormatJSONSchema = "json-schema"

	// eventIdentifier is the identifier used within cue to declare the event's type.
	eventIdentifier = "InngestEvent"
)

// EventDefinition represents the type information for an event trigger.  The
// type information is stored locally within the function for offline usage and
// for working with unsynced events.
type EventDefinition struct {
	// Format represents the format for the event definition.  This may be
	// either "cue" or "string
	Format DefinitionFormat `json:"format"`

	// Synced represents whether this is synced via the event registry or
	// if this is a new event and is the source of truth itself.
	Synced bool `json:"synced"`

	// Def represents the type definition.  This may be the JSON schema
	// definition, the cue syntax, or (soon) the avro syntax, etc.
	//
	// This may be the event definition itself, or it may be a path to
	// a file which contains the event definition.
	Def string `json:"def"`

	// cueType is canonical cue definition for the event.  We use cue as our
	// source of truth; to generate other event types we convert Def to cue,
	// then cue to the desired output.
	//
	// The cue schema should be stored within the "InngestEvent" identifier.
	cueType string
}

// Validate attempts to parse the event definition and reports any errors.
func (ed *EventDefinition) Validate(ctx context.Context) error {
	if err := ed.createCueType(ctx); err != nil {
		return err
	}
	return nil
}

func (ed *EventDefinition) readDefinition(ctx context.Context) (string, error) {
	file, _ := PathName(ctx, ed.Def)
	if file == "" {
		return ed.Def, nil
	}
	// The event definition is stored within a file.
	file, err := filepath.Abs(file)
	if err != nil {
		return "", fmt.Errorf("error finding event definition: %w", err)
	}
	byt, err := os.ReadFile(file)
	if err != nil {
		return "", fmt.Errorf("error reading event definition '%s': %w", file, err)
	}
	return string(byt), nil
}

// createCueType converts the Def input into Cue.
func (ed *EventDefinition) createCueType(ctx context.Context) error {
	if ed.cueType != "" {
		// This has already been completed.
		return nil
	}

	def, err := ed.readDefinition(ctx)
	if err != nil {
		return nil
	}

	switch ed.Format {
	case FormatCue:
		ed.cueType = string(def)
		return nil
	case FormatJSONSchema:
		// Convert JSON schema to cue, then store our canonical cue representation.
		cue, err := jsonschema.UnmarshalString(string(def))
		if err != nil {
			return fmt.Errorf("error converting json-schema definition: %w", err)
		}
		ed.cueType = cue
		return nil
	default:
		return fmt.Errorf("unknown event definition format: %s", ed.Format)
	}
}

// Cue returns the Cue type definition of the event.
func (ed *EventDefinition) Cue(ctx context.Context) (string, error) {
	err := ed.createCueType(ctx)
	return ed.cueType, err
}

// Typescript returns the Typescript definition of the event.
func (ed *EventDefinition) Typescript(ctx context.Context) (string, error) {
	if err := ed.createCueType(ctx); err != nil {
		return "", err
	}

	// Ensure we have an identifier so that this isn't broken into event components.
	def := ed.cueType
	if strings.TrimSpace(ed.cueType)[0] == '{' {
		def = "#Event: " + def
	}

	return cuetypescript.MarshalString(def)
}

// JSONSchema returns the JSON Schema for the event.
func (ed *EventDefinition) JSONSchema(ctx context.Context) (map[string]interface{}, error) {
	// If the original event definition is JSON-schema, don't convert:
	// straight up return it.  Cue has _somewhat_ lossy support for JSON-schema:
	// it doesn't support the "additionalProperties" field for object
	// definitions, and there may be other unsupported fields supported.
	if ed.Format == FormatJSONSchema {
		def, err := ed.readDefinition(ctx)
		if err != nil {
			return nil, err
		}
		data := map[string]interface{}{}
		err = json.Unmarshal([]byte(def), &data)
		return data, err
	}

	if err := ed.createCueType(ctx); err != nil {
		return nil, err
	}

	// Prepend the cue type, which must be a root object with no identifiers,
	// with our InngestEvent definition.  THis allows us to return the concrete
	// definition, as jsonschema can return multiple definitions per cue file.
	schemas, err := jsonschema.MarshalString(fmt.Sprintf("#%s: %s", eventIdentifier, ed.cueType))
	if err != nil {
		return nil, err
	}

	return schemas.Find("InngestEvent"), nil
}

// evtDefinition is a blank event definition
const evtDefinition = `{
  name: %s
  data: {
    // Your event data should go here.
  },
  user: {
    // Any user information for audit trails, eg. email, external_id, should go here.
  },
  v: "1", // A sortable version
}`
