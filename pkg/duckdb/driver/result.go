package driver

// columns is a result's column names in the query's own left-to-right
// order, plus an index from each name to its first position. Every row of
// one result shares a single *columns.
//
// Values live positionally in row.vals rather than in a map keyed by name,
// so a query that repeats a column name (SELECT a.x, b.x, or SELECT * over
// a join) keeps every value; by-name lookup (row.get) resolves to the first
// column of that name, which is all the driver's own internal lookups need.
type columns struct {
	names []string
	index map[string]int
}

func newColumns(names []string) *columns {
	index := make(map[string]int, len(names))
	for i, name := range names {
		if _, ok := index[name]; !ok {
			index[name] = i
		}
	}
	return &columns{names: names, index: index}
}

// row is one result row: vals[i] is the value of cols.names[i].
type row struct {
	cols *columns
	vals []any
}

// get returns the value of the first column named name, or nil if there is
// no such column.
func (r row) get(name string) any {
	if r.cols == nil {
		return nil
	}
	i, ok := r.cols.index[name]
	if !ok || i >= len(r.vals) {
		return nil
	}
	return r.vals[i]
}

// names returns the row's column names, for diagnostics.
func (r row) names() []string {
	if r.cols == nil {
		return nil
	}
	return r.cols.names
}
