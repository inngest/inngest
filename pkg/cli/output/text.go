package output

import (
	"fmt"
	"os"
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
	o := txtOpt{}
	for _, apply := range opts {
		apply(&o)
	}

	if o.leadSpace {
		fmt.Println()
	}

	indentStr := strings.Repeat(" ", tw.indent)
	for key, value := range data {
		if tw.isNestedMap(value) {
			fmt.Fprintf(tw.w, "%s%s:\n", indentStr, key)
			tw.w.Flush()
			
			nestedWriter := tw.WithIndent(tw.indent + 2)
			nestedWriter.Write(tw.convertToAnyMap(value))
		} else {
			fmt.Fprintf(tw.w, "%s%s:\t%s\n", indentStr, key, tw.valueToString(value))
		}
	}

	return tw.w.Flush()
}

func (tw *TextWriter) isNestedMap(value any) bool {
	switch value.(type) {
	case map[string]any, map[string]string, map[string]int, map[string]int64, map[string]float64, map[string]bool:
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
		return v.String()
	case error:
		return v.Error()
	default:
		return fmt.Sprintf("%v", value)
	}
}

type Row struct {
	Key   string
	Value any
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
		return v.String()
	case error:
		return v.Error()
	default:
		return fmt.Sprintf("%v", r.Value)
	}
}
