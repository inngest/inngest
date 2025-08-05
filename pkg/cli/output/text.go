package output

import (
	"fmt"
	"os"
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

func (tw *TextWriter) WithIndent(indent int) *TextWriter {
	return &TextWriter{
		indent: tw.indent,
		w:      newTabWriter(),
	}
}

func (tw *TextWriter) Write(rows []Row) error {
	for _, r := range rows {
		fmt.Fprintf(tw.w, "%s:\t%s\n", r.Key, r.ToString())
	}

	return tw.w.Flush()
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
