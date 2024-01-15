package expressions

import (
	"context"
	"encoding/json"
	"reflect"

	"github.com/google/cel-go/cel"
	"github.com/google/cel-go/interpreter"
)

// NewData returns data ready for use within an evaluation.  This formats all
// values of the incoming map correctly for evaluation.
func NewData(data map[string]interface{}) *Data {
	return &Data{data: mapify(data)}
}

// Data represents data that will be used as variables when evaluating an
// expression.  Initializing a new Data instance parses and formats the
// data, copying values into a new map.
//
// Data is safe to use from multiple goroutines.
type Data struct {
	// data represents the data being passed into an expression
	data map[string]interface{}
}

// Map returns the data as a map.
func (d Data) Clone() *Data {
	return &Data{
		data: mapify(d.data),
	}

}

// Map returns the data as a map.
func (d Data) Map() map[string]interface{} {
	return d.data
}

// MarshalJSON returns the data as JSON.
func (d Data) MarshalJSON() ([]byte, error) {
	return json.Marshal(d.data)
}

// Add adds data to be evaluated, overwriting existing data on conflicts.
func (d *Data) Add(m map[string]interface{}) {
	merge(d.data, m)
}

// Partial returns a PartialActivation for CEL.  This inspects the given expression to determine
// each field being resolved, then inspects the data given.  For each missing field in data,
// we add an AttributePattern for the missing element such that CEL returns unknown and skips
// the data if possible.
//
// This allows us to evaluate arbitrary maps with missing data without errors.
//
// See https://github.com/google/cel-go/issues/409 for why we have to inspect - partial attribute
// patterns override any data passed and always return unknowns.
func (d *Data) Partial(ctx context.Context, attrs UsedAttributes) (interpreter.PartialActivation, error) {
	patterns := []*interpreter.AttributePattern{}
	for _, path := range attrs.FullPaths() {
		// path represents a series of keys within data.
		if d.PathExists(ctx, path) {
			continue
		}

		pattern := cel.AttributePattern(path[0])
		for n, piece := range path[1:] {
			if n == len(path)-1 {
				pattern = pattern.QualString(piece)
				break
			}
			pattern = pattern.QualString(piece)
		}

		patterns = append(patterns, pattern)
	}

	// Add the mapped data and the "patterns" which allow us to use empty data
	// within CEL.
	return interpreter.NewPartialActivation(d.data, patterns...)
}

// Get returns the data from the given path, with a boolean indicating
// whether the path exists within data.
func (d *Data) Get(ctx context.Context, path []string) (interface{}, bool) {
	data := d.data
	if data == nil {
		data = map[string]interface{}{}
	}

	for n, key := range path {
		val, ok := data[key]
		if !ok {
			return nil, false
		}

		// This is the last item and exists, so we don't need to attempt to
		// recurse.
		if n == len(path)-1 {
			return val, true
		}

		switch v := val.(type) {
		case map[string]interface{}:
			if v == nil {
				// We're not on the last element but this is a nil map.
				// This path cannot exist.
				return nil, false
			}
			// Recurse into the map.
			data = v
		default:
			// This isn't the last path segment, but the type we're nesting
			// into isn't a map.  In this case the path cannot exist.
			if n < len(path)-1 {
				return nil, false
			}
		}
	}

	return nil, false
}

// PathExists returns whether the given path exists within data.
func (d *Data) PathExists(ctx context.Context, path []string) bool {
	_, ok := d.Get(ctx, path)
	return ok
}

// mapify takes a map[string]interface{} which may contain different types
// and ensures they're all map[string]interface{} for cel.
func mapify(data map[string]interface{}) map[string]interface{} {
	copied := map[string]interface{}{}

	// Iterate through each piece of data.
	for key, val := range data {
		switch v := val.(type) {
		case map[string]interface{}:
			val = mapify(v)
		case nil, string, int, int64, int32, float64, float32, bool:
			// Do nothing.  For nil, this prevents reflects on nil-values which
			// is not supported.  For other datatypes, this is a minor perf
			// optimization to prevent unnecessary reflect calls.
		default:
			// XXX: slices are not handled, but are not added to our system as non-maps
			// internally.
			if reflect.TypeOf(v).Kind() == reflect.Struct || reflect.TypeOf(v).Kind() == reflect.Map {
				// Convert to JSON and back again, as we want to use the JSON encoding
				// tags for the map names.
				m := map[string]interface{}{}
				byt, _ := json.Marshal(v)
				_ = json.Unmarshal(byt, &m)
				val = m
			}
		}
		copied[key] = val
	}

	return copied
}

// merge recursively merges map[string]interface{} elements together.
func merge(to, from map[string]interface{}) {
	for key, val := range from {
		oldVal, ok := to[key]
		if !ok {
			to[key] = val
			continue
		}

		_, fromMap := val.(map[string]interface{})
		_, isPrevMap := oldVal.(map[string]interface{})

		if fromMap && isPrevMap {
			merge(to[key].(map[string]interface{}), val.(map[string]interface{}))
			continue
		}

		to[key] = val
	}
}
