package overlay

import (
	"bytes"
	"fmt"
	"log"
	"strings"

	"gopkg.in/yaml.v3"
)

// Compare compares input specifications from two files and returns an overlay
// that will convert the first into the second.
func Compare(title string, y1 *yaml.Node, y2 yaml.Node) (*Overlay, error) {
	actions, err := walkTreesAndCollectActions(simplePath{}, y1, y2)
	if err != nil {
		return nil, err
	}

	return &Overlay{
		Version: "1.0.0",
		Info: Info{
			Title:   title,
			Version: "0.0.0",
		},
		Actions: actions,
	}, nil
}

type simplePart struct {
	isKey bool
	key   string
	index int
}

func intPart(index int) simplePart {
	return simplePart{
		index: index,
	}
}

func keyPart(key string) simplePart {
	return simplePart{
		isKey: true,
		key:   key,
	}
}

func (p simplePart) String() string {
	if p.isKey {
		return fmt.Sprintf("[%q]", p.key)
	}
	return fmt.Sprintf("[%d]", p.index)
}

func (p simplePart) KeyString() string {
	if p.isKey {
		return p.key
	}
	panic("FIXME: Bug detected in overlay comparison algorithm: attempt to use non key part as key")
}

type simplePath []simplePart

func (p simplePath) WithIndex(index int) simplePath {
	return append(p, intPart(index))
}

func (p simplePath) WithKey(key string) simplePath {
	return append(p, keyPart(key))
}

func (p simplePath) ToJSONPath() string {
	out := &strings.Builder{}
	out.WriteString("$")
	for _, part := range p {
		out.WriteString(part.String())
	}
	return out.String()
}

func (p simplePath) Dir() simplePath {
	return p[:len(p)-1]
}

func (p simplePath) Base() simplePart {
	return p[len(p)-1]
}

func walkTreesAndCollectActions(path simplePath, y1 *yaml.Node, y2 yaml.Node) ([]Action, error) {
	if y1 == nil {
		return []Action{{
			Target: path.Dir().ToJSONPath(),
			Update: y2,
		}}, nil
	}

	if y2.IsZero() {
		return []Action{{
			Target: path.ToJSONPath(),
			Remove: true,
		}}, nil
	}
	if y1.Kind != y2.Kind {
		return []Action{{
			Target: path.ToJSONPath(),
			Update: y2,
		}}, nil
	}

	switch y1.Kind {
	case yaml.DocumentNode:
		return walkTreesAndCollectActions(path, y1.Content[0], *y2.Content[0])
	case yaml.SequenceNode:
		if len(y2.Content) == len(y1.Content) {
			return walkSequenceNode(path, y1, y2)
		}

		if len(y2.Content) == len(y1.Content)+1 &&
			yamlEquals(y2.Content[:len(y1.Content)], y1.Content) {
			return []Action{{
				Target: path.ToJSONPath(),
				Update: yaml.Node{
					Kind:    y1.Kind,
					Content: []*yaml.Node{y2.Content[len(y1.Content)]},
				},
			}}, nil
		}

		return []Action{{
			Target: path.ToJSONPath() + "[*]", // target all elements
			Remove: true,
		}, {
			Target: path.ToJSONPath(),
			Update: yaml.Node{
				Kind:    y1.Kind,
				Content: y2.Content,
			},
		}}, nil
	case yaml.MappingNode:
		return walkMappingNode(path, y1, y2)
	case yaml.ScalarNode:
		if y1.Value != y2.Value {
			return []Action{{
				Target: path.ToJSONPath(),
				Update: y2,
			}}, nil
		}
	case yaml.AliasNode:
		log.Println("YAML alias nodes are not yet supported for compare.")
	}
	return nil, nil
}

func yamlEquals(nodes []*yaml.Node, content []*yaml.Node) bool {
	for i := range nodes {
		bufA := &bytes.Buffer{}
		bufB := &bytes.Buffer{}
		decodeA := yaml.NewEncoder(bufA)
		decodeB := yaml.NewEncoder(bufB)
		err := decodeA.Encode(nodes[i])
		if err != nil {
			return false
		}
		err = decodeB.Encode(content[i])
		if err != nil {
			return false
		}

		if bufA.String() != bufB.String() {
			return false
		}
	}
	return true
}

func walkSequenceNode(path simplePath, y1 *yaml.Node, y2 yaml.Node) ([]Action, error) {
	nodeLen := max(len(y1.Content), len(y2.Content))
	var actions []Action
	for i := 0; i < nodeLen; i++ {
		var c1, c2 *yaml.Node
		if i < len(y1.Content) {
			c1 = y1.Content[i]
		}
		if i < len(y2.Content) {
			c2 = y2.Content[i]
		}

		newActions, err := walkTreesAndCollectActions(
			path.WithIndex(i),
			c1, *c2)
		if err != nil {
			return nil, err
		}

		actions = append(actions, newActions...)
	}

	return actions, nil
}

func walkMappingNode(path simplePath, y1 *yaml.Node, y2 yaml.Node) ([]Action, error) {
	var actions []Action
	foundKeys := map[string]struct{}{}

	// Add or update keys in y2 that differ/missing from y1
Outer:
	for i := 0; i < len(y2.Content); i += 2 {
		k2 := y2.Content[i]
		v2 := y2.Content[i+1]

		foundKeys[k2.Value] = struct{}{}

		// find keys in y1 to update
		for j := 0; j < len(y1.Content); j += 2 {
			k1 := y1.Content[j]
			v1 := y1.Content[j+1]

			if k1.Value == k2.Value {
				newActions, err := walkTreesAndCollectActions(
					path.WithKey(k2.Value),
					v1, *v2)
				if err != nil {
					return nil, err
				}
				actions = append(actions, newActions...)
				continue Outer
			}
		}

		// key not found in y1, so add it
		newActions, err := walkTreesAndCollectActions(
			path.WithKey(k2.Value),
			nil, yaml.Node{
				Kind:    y1.Kind,
				Content: []*yaml.Node{k2, v2},
			})
		if err != nil {
			return nil, err
		}

		actions = append(actions, newActions...)
	}

	// look for keys in y1 that are not in y2: remove them
	for i := 0; i < len(y1.Content); i += 2 {
		k1 := y1.Content[i]

		if _, alreadySeen := foundKeys[k1.Value]; alreadySeen {
			continue
		}

		actions = append(actions, Action{
			Target: path.WithKey(k1.Value).ToJSONPath(),
			Remove: true,
		})
	}

	return actions, nil
}
