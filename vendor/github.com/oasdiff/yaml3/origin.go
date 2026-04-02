package yaml

import "fmt"

const originTag = "__origin__"

func isScalar(n *Node) bool {
	return n.Kind == ScalarNode
}

func isSequence(n *Node) bool {
	return n.Kind == SequenceNode
}

func addOriginInSeq(n *Node, file string) *Node {

	if n.Kind != MappingNode {
		return n
	}

	// in case of a sequence, we use the first element as the key
	return addOrigin(n.Content[0], n, file)
}

func addOriginInMap(key, n *Node, file string) *Node {

	if n.Kind != MappingNode {
		return n
	}

	return addOrigin(key, n, file)
}

func addOrigin(key, n *Node, file string) *Node {
	if isOrigin(key) {
		return n
	}

	content := getKeyLocation(key, file)
	content = append(content, getNamedMap("fields", getFieldLocations(n, file))...)
	content = append(content, getNamedMap("sequences", getSequenceLocations(n, file))...)
	n.Content = append(n.Content, getNamedMap(originTag, content)...)
	return n
}

func getFieldLocations(n *Node, file string) []*Node {

	l := len(n.Content)
	size := 0
	for i := 0; i < l; i += 2 {
		if isScalar(n.Content[i+1]) || isSequence(n.Content[i+1]) {
			size += 2
		}
	}

	nodes := make([]*Node, 0, size)
	for i := 0; i < l; i += 2 {
		if isScalar(n.Content[i+1]) || isSequence(n.Content[i+1]) {
			nodes = append(nodes, getNodeLocation(n.Content[i], file)...)
		}
	}
	return nodes
}

func getSequenceLocations(n *Node, file string) []*Node {
	l := len(n.Content)
	var nodes []*Node
	for i := 0; i < l; i += 2 {
		if isSequence(n.Content[i+1]) {
			nodes = append(nodes, getNamedSeq(n.Content[i].Value, n.Content[i+1], file)...)
		}
	}
	return nodes
}

func getNamedSeq(title string, seq *Node, file string) []*Node {
	var items []*Node
	for _, item := range seq.Content {
		if item.Kind == ScalarNode {
			items = append(items, getMap(getLocationObject(item, file)))
		}
	}
	if len(items) == 0 {
		return nil
	}
	return []*Node{
		{
			Kind:  ScalarNode,
			Tag:   "!!str",
			Value: title,
		},
		{
			Kind:    SequenceNode,
			Tag:     "!!seq",
			Content: items,
		},
	}
}

// isOrigin returns true if the key is an "origin" element
// the current implementation is not optimal, as it relies on the key's line number
// a better design would be to use a dedicated field in the Node struct
func isOrigin(key *Node) bool {
	return key.Line == 0
}

func getNodeLocation(n *Node, file string) []*Node {
	return getNamedMap(n.Value, getLocationObject(n, file))
}

func getKeyLocation(n *Node, file string) []*Node {
	return getNamedMap("key", getLocationObject(n, file))
}

func getNamedMap(title string, content []*Node) []*Node {
	if len(content) == 0 {
		return nil
	}

	return []*Node{
		{
			Kind:  ScalarNode,
			Tag:   "!!str",
			Value: title,
		},
		getMap(content),
	}
}

func getMap(content []*Node) *Node {
	return &Node{
		Kind:    MappingNode,
		Tag:     "!!map",
		Content: content,
	}
}

func getLocationObject(key *Node, file string) []*Node {
	return []*Node{
		{
			Kind:  ScalarNode,
			Tag:   "!!str",
			Value: "file",
		},
		{
			Kind:  ScalarNode,
			Tag:   "!!str",
			Value: file,
		},
		{
			Kind:  ScalarNode,
			Tag:   "!!str",
			Value: "line",
		},
		{
			Kind:  ScalarNode,
			Tag:   "!!int",
			Value: fmt.Sprintf("%d", key.Line),
		},
		{
			Kind:  ScalarNode,
			Tag:   "!!str",
			Value: "column",
		},
		{
			Kind:  ScalarNode,
			Tag:   "!!int",
			Value: fmt.Sprintf("%d", key.Column),
		},
		{
			Kind:  ScalarNode,
			Tag:   "!!str",
			Value: "name",
		},
		{
			Kind:  ScalarNode,
			Tag:   "!!string",
			Value: key.Value,
		},
	}
}
