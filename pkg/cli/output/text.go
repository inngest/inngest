package output

import (
	"encoding/json"
	"fmt"
	"os"
	"reflect"
	"strings"
	"text/tabwriter"
)

func NewTextWriter() *TextWriter {
	return &TextWriter{
		w: newTabWriter(),
	}
}

func newTabWriter() *tabwriter.Writer {
	return tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
}

type TextWriter struct {
	indent int
	w      *tabwriter.Writer
}

type TextOpt func(o *txtOpt)

type txtOpt struct {
	leadSpace bool
}

func WithTextOptLeadSpace(space bool) TextOpt {
	return func(o *txtOpt) {
		o.leadSpace = space
	}
}

func (tw *TextWriter) WithIndent(indent int) *TextWriter {
	return &TextWriter{
		indent: indent,
		w:      newTabWriter(),
	}
}

func (tw *TextWriter) Write(data map[string]any, opts ...TextOpt) error {
	// Convert regular map to OrderedMap (key order will be non-deterministic)
	om := NewOrderedMap()
	for key, value := range data {
		om.Set(key, value)
	}
	return tw.WriteOrdered(om, opts...)
}

func (tw *TextWriter) WriteOrdered(data *OrderedMap, opts ...TextOpt) error {
	o := txtOpt{}
	for _, apply := range opts {
		apply(&o)
	}

	if o.leadSpace {
		fmt.Println()
	}

	indentStr := strings.Repeat(" ", tw.indent)
	for _, key := range data.Keys() {
		value, _ := data.Get(key)
		if tw.isNestedMap(value) {
			fmt.Fprintf(tw.w, "%s%s:\n", indentStr, key)

			nestedWriter := &TextWriter{
				indent: tw.indent + 2,
				w:      tw.w, // Use the same tabwriter
			}
			if err := nestedWriter.WriteOrdered(tw.convertToOrderedMap(value)); err != nil {
				return err
			}
		} else {
			fmt.Fprintf(tw.w, "%s%s:\t%s\n", indentStr, key, tw.valueToString(value))
		}
	}

	// Don't auto-flush to allow multiple Write calls to accumulate
	return nil
}

func (tw *TextWriter) Flush() error {
	return tw.w.Flush()
}

func (tw *TextWriter) isNestedMap(value any) bool {
	switch value.(type) {
	case *OrderedMap, map[string]any, map[string]string, map[string]int, map[string]int64, map[string]float64, map[string]bool:
		return true
	default:
		return false
	}
}

func (tw *TextWriter) convertToAnyMap(value any) map[string]any {
	result := make(map[string]any)

	switch v := value.(type) {
	case map[string]any:
		return v
	case map[string]string:
		for k, val := range v {
			result[k] = val
		}
	case map[string]int:
		for k, val := range v {
			result[k] = val
		}
	case map[string]int64:
		for k, val := range v {
			result[k] = val
		}
	case map[string]float64:
		for k, val := range v {
			result[k] = val
		}
	case map[string]bool:
		for k, val := range v {
			result[k] = val
		}
	}

	return result
}

func (tw *TextWriter) convertToOrderedMap(value any) *OrderedMap {
	om := NewOrderedMap()

	switch v := value.(type) {
	case *OrderedMap:
		return v
	case map[string]any:
		for key, val := range v {
			om.Set(key, val)
		}
	case map[string]string:
		for key, val := range v {
			om.Set(key, val)
		}
	case map[string]int:
		for key, val := range v {
			om.Set(key, val)
		}
	case map[string]int64:
		for key, val := range v {
			om.Set(key, val)
		}
	case map[string]float64:
		for key, val := range v {
			om.Set(key, val)
		}
	case map[string]bool:
		for key, val := range v {
			om.Set(key, val)
		}
	}

	return om
}

func (tw *TextWriter) valueToString(value any) string {
	if value == nil {
		return ""
	}

	switch v := value.(type) {
	case string:
		return v
	case int, int8, int16, int32, int64:
		return fmt.Sprintf("%d", v)
	case uint, uint8, uint16, uint32, uint64:
		return fmt.Sprintf("%d", v)
	case float32, float64:
		return fmt.Sprintf("%.2f", v)
	case bool:
		return fmt.Sprintf("%t", v)
	case fmt.Stringer:
		// Check for nil pointers that implement Stringer
		if v == nil {
			return "<nil>"
		}
		// Use reflection to check if it's a nil pointer
		rv := reflect.ValueOf(v)
		if rv.Kind() == reflect.Ptr && rv.IsNil() {
			return "<nil>"
		}

		return v.String()
	case error:
		return v.Error()
	default:
		// Try JSON serialization for complex types
		if jsonStr := tw.formatAsJSON(value); jsonStr != "" {
			return jsonStr
		}
		return fmt.Sprintf("%v", value)
	}
}

// formatAsJSON attempts to format a value as indented JSON with proper alignment
func (tw *TextWriter) formatAsJSON(value any) string {
	jsonBytes, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return ""
	}

	jsonStr := string(jsonBytes)
	// For multi-line JSON, indent continuation lines to align with value column
	lines := strings.Split(jsonStr, "\n")
	if len(lines) > 1 {
		indentStr := strings.Repeat(" ", tw.indent) + "\t"
		for i := 1; i < len(lines); i++ {
			lines[i] = indentStr + lines[i]
		}
		return strings.Join(lines, "\n")
	}
	return jsonStr
}

type Row struct {
	Key   string
	Value any
}

// OrderedMap preserves the order of key-value pairs
type OrderedMap struct {
	keys   []string
	values map[string]any
}

// NewOrderedMap creates a new ordered map
func NewOrderedMap() *OrderedMap {
	return &OrderedMap{
		keys:   make([]string, 0),
		values: make(map[string]any),
	}
}

// Set adds or updates a key-value pair
func (om *OrderedMap) Set(key string, value any) {
	if _, exists := om.values[key]; !exists {
		om.keys = append(om.keys, key)
	}
	om.values[key] = value
}

// Get retrieves a value by key
func (om *OrderedMap) Get(key string) (any, bool) {
	value, exists := om.values[key]
	return value, exists
}

// Keys returns all keys in order
func (om *OrderedMap) Keys() []string {
	return om.keys
}

// Len returns the number of key-value pairs
func (om *OrderedMap) Len() int {
	return len(om.keys)
}

// OrderedData creates an OrderedMap from a slice of key-value pairs
func OrderedData(pairs ...any) *OrderedMap {
	if len(pairs)%2 != 0 {
		panic("OrderedData requires an even number of arguments (key-value pairs)")
	}

	om := NewOrderedMap()
	for i := 0; i < len(pairs); i += 2 {
		key, ok := pairs[i].(string)
		if !ok {
			panic("OrderedData keys must be strings")
		}
		om.Set(key, pairs[i+1])
	}
	return om
}

func (r *Row) ToString() string {
	if r.Value == nil {
		return ""
	}

	switch v := r.Value.(type) {
	case string:
		return v
	case int, int8, int16, int32, int64:
		return fmt.Sprintf("%d", v)
	case uint, uint8, uint16, uint32, uint64:
		return fmt.Sprintf("%d", v)
	case float32, float64:
		return fmt.Sprintf("%.2f", v)
	case bool:
		return fmt.Sprintf("%t", v)
	case fmt.Stringer:
		// Check for nil pointers that implement Stringer
		if v == nil {
			return "<nil>"
		}
		// Use reflection to check if it's a nil pointer
		rv := reflect.ValueOf(v)
		if rv.Kind() == reflect.Ptr && rv.IsNil() {
			return "<nil>"
		}
		return v.String()
	case error:
		return v.Error()
	default:
		return fmt.Sprintf("%v", r.Value)
	}
}
