package overlay

import "gopkg.in/yaml.v3"

type parentIndex map[*yaml.Node]*yaml.Node

// newParentIndex returns a new parentIndex, populated for the given root node.
func newParentIndex(root *yaml.Node) parentIndex {
	index := parentIndex{}
	index.indexNodeRecursively(root)
	return index
}

func (index parentIndex) indexNodeRecursively(parent *yaml.Node) {
	for _, child := range parent.Content {
		index[child] = parent
		index.indexNodeRecursively(child)
	}
}

func (index parentIndex) getParent(child *yaml.Node) *yaml.Node {
	return index[child]
}
