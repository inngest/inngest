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
		fmt.Fprintf(tw.w, "%s%s:\t%s\n", indentStr, key, tw.valueToString(value))
	}

	return tw.w.Flush()
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
