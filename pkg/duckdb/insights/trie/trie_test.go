package trie

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestExactMatch(t *testing.T) {
	tr := New[[]string, string, string]()
	tr.Add([]string{"event", "id"}, "exact")

	v, ok := tr.Get([]string{"event", "id"})
	assert.True(t, ok)
	assert.Equal(t, "exact", v)
}

func TestExactMatchMissesUnregisteredPath(t *testing.T) {
	tr := New[[]string, string, string]()
	tr.Add([]string{"event", "id"}, "exact")

	_, ok := tr.Get([]string{"event", "name"})
	assert.False(t, ok)
}

func TestWildMatchesRegisteredPathAndEverythingBeneathIt(t *testing.T) {
	tr := New[[]string, string, string]()
	tr.AddWild([]string{"event", "data"}, "wild")

	for _, path := range [][]string{
		{"event", "data"},
		{"event", "data", "foo"},
		{"event", "data", "foo", "bar"},
	} {
		v, ok := tr.Get(path)
		assert.True(t, ok, "path %v", path)
		assert.Equal(t, "wild", v)
	}
}

func TestExactMatchWinsOverAnAncestorWildMatch(t *testing.T) {
	tr := New[[]string, string, string]()
	tr.AddWild([]string{"event"}, "wild")
	tr.Add([]string{"event", "id"}, "exact")

	v, ok := tr.Get([]string{"event", "id"})
	assert.True(t, ok)
	assert.Equal(t, "exact", v)
}

func TestLongestWildMatchWins(t *testing.T) {
	tr := New[[]string, string, string]()
	tr.AddWild([]string{"event"}, "shallow")
	tr.AddWild([]string{"event", "data"}, "deep")

	v, ok := tr.Get([]string{"event", "data", "foo"})
	assert.True(t, ok)
	assert.Equal(t, "deep", v)
}
