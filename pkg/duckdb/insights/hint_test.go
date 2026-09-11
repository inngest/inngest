package insights

import "testing"

func TestColumnHintZeroValueIsNone(t *testing.T) {
	var h ColumnHint
	if h != HintNone {
		t.Errorf("zero value of ColumnHint = %v, want HintNone", h)
	}
}

func TestColumnHintsAreDistinct(t *testing.T) {
	seen := map[ColumnHint]bool{}
	for _, h := range []ColumnHint{HintNone, HintAppID, HintFunctionID, HintRunID, HintEventID} {
		if seen[h] {
			t.Fatalf("duplicate ColumnHint value %v", h)
		}
		seen[h] = true
	}
}
