package art

import (
	"bytes"
	"fmt"
)

const (
	printValuesAsChar = 1 << iota
	printValuesAsDecimal
	printValuesAsHex

	printValueDefault = printValuesAsChar
)

type depthStorage struct {
	childNum      int
	childrenTotal int
}

type treeStringer struct {
	storage []depthStorage
	buf     *bytes.Buffer
}

// String returns tree in the human readable format, see DumpNode for examples
func (t *tree) String() string {
	return DumpNode(t.root)
}

func (ts *treeStringer) generatePads(depth int, childNum int, childrenTotal int) (pad0, pad string) {
	ts.storage[depth] = depthStorage{childNum, childrenTotal}

	for d := 0; d <= depth; d++ {
		if d < depth {
			if ts.storage[d].childNum+1 < ts.storage[d].childrenTotal {
				pad0 += "│   "
			} else {
				pad0 += "    "
			}
		} else {
			if childrenTotal == 0 {
				pad0 += "─"
			} else if ts.storage[d].childNum+1 < ts.storage[d].childrenTotal {
				pad0 += "├"
			} else {
				pad0 += "└"
			}
			pad0 += "──"
		}

	}
	pad0 += " "

	for d := 0; d <= depth; d++ {
		if childNum+1 < childrenTotal && childrenTotal > 0 {
			if ts.storage[d].childNum+1 < ts.storage[d].childrenTotal {
				pad += "│   "
			} else {
				pad += "    "
			}
		} else if d < depth && ts.storage[d].childNum+1 < ts.storage[d].childrenTotal {
			pad += "│   "
		} else {
			pad += "    "
		}

	}

	return
}

func (ts *treeStringer) append(v interface{}, opts ...int) *treeStringer {
	options := 0
	for _, opt := range opts {
		options |= opt
	}

	if options == 0 {
		options = printValueDefault
	}

	switch v.(type) {

	case string:
		str, _ := v.(string)
		ts.buf.WriteString(str)

	case []byte:
		arr, _ := v.([]byte)
		ts.append("[")
		for i, b := range arr {
			if (options & printValuesAsChar) != 0 {
				if b > 0 {
					ts.append(fmt.Sprintf("%c", b))
				} else {
					ts.append("·")
				}

			} else if (options & printValuesAsDecimal) != 0 {
				ts.append(fmt.Sprintf("%d", b))
			}
			if (options&printValuesAsDecimal) != 0 && i+1 < len(arr) {
				ts.append(" ")
			}
		}
		ts.append("]")

	case Key:
		k, _ := v.(Key)
		ts.append([]byte(k))

	default:
		ts.append("[")
		ts.append(fmt.Sprintf("%#v", v))
		ts.append("]")
	}

	return ts
}

func (ts *treeStringer) appendKey(keys []byte, present []byte, opts ...int) *treeStringer {
	options := 0
	for _, opt := range opts {
		options |= opt
	}

	if options == 0 {
		options = printValueDefault
	}

	ts.append("[")
	for i, b := range keys {
		if (options & printValuesAsChar) != 0 {
			if present[i] != 0 {
				ts.append(fmt.Sprintf("%c", b))
			} else {
				ts.append("·")
			}

		} else if (options & printValuesAsDecimal) != 0 {
			if present[i] != 0 {
				ts.append(fmt.Sprintf("%2d", b))
			} else {
				ts.append("·")
			}
		} else if (options & printValuesAsHex) != 0 {
			if present[i] != 0 {
				ts.append(fmt.Sprintf("%2x", b))
			} else {
				ts.append("·")
			}
		}
		if (options&(printValuesAsDecimal|printValuesAsHex)) != 0 && i+1 < len(keys) {
			ts.append(" ")
		}
	}
	ts.append("]")

	return ts
}

func (ts *treeStringer) children(children []*artNode, numChildred uint16, depth int, zeroChild *artNode) {
	for i, child := range children {
		ts.baseNode(child, depth, i, len(children)+1)
	}

	ts.baseNode(zeroChild, depth, len(children)+1, len(children)+1)
}

func (ts *treeStringer) node(pad string, prefixLen uint32, prefix []byte, keys []byte, present []byte, children []*artNode, numChildren uint16, depth int, zeroChild *artNode) {
	if prefix != nil {
		ts.append(pad).append(fmt.Sprintf("prefix(%x): %v", prefixLen, prefix))
		ts.append(prefix).append("\n")
	}

	if keys != nil {
		ts.append(pad).append("keys: ").appendKey(keys, present, printValuesAsDecimal).append(" ")
		ts.appendKey(keys, present, printValuesAsChar).append("\n")
	}

	ts.append(pad).append(fmt.Sprintf("children(%v): %+v <%v>\n", numChildren, children, zeroChild))
	ts.children(children, numChildren, depth+1, zeroChild)
}

func (ts *treeStringer) baseNode(an *artNode, depth int, childNum int, childrenTotal int) {
	padHeader, pad := ts.generatePads(depth, childNum, childrenTotal)
	if an == nil {
		ts.append(padHeader).append("nil").append("\n")
		return
	}

	ts.append(padHeader)
	ts.append(fmt.Sprintf("%v (%p)\n", an.kind, an))
	switch an.kind {
	case Node4:
		nn := an.node4()

		ts.node(pad, nn.prefixLen, nn.prefix[:], nn.keys[:], nn.present[:], nn.children[:], nn.numChildren, depth, nn.zeroChild)

	case Node16:
		nn := an.node16()

		var present []byte
		for i := 0; i < len(nn.keys); i++ {
			if (nn.present & (1 << uint16(i))) != 0 {
				present = append(present, 1)
			} else {
				present = append(present, 0)
			}
		}

		ts.node(pad, nn.prefixLen, nn.prefix[:], nn.keys[:], present, nn.children[:], nn.numChildren, depth, nn.zeroChild)

	case Node48:
		nn := an.node48()

		var present []byte
		for i := 0; i < len(nn.keys); i++ {
			if (nn.present[uint16(i)>>n48s] & (1 << (uint16(i) % n48m))) != 0 {
				present = append(present, 1)
			} else {
				present = append(present, 0)
			}
		}

		ts.node(pad, nn.prefixLen, nn.prefix[:], nn.keys[:], present, nn.children[:], nn.numChildren, depth, nn.zeroChild)

	case Node256:
		nn := an.node256()
		ts.node(pad, nn.prefixLen, nn.prefix[:], nil, nil, nn.children[:], nn.numChildren, depth, nn.zeroChild)

	case Leaf:
		n := an.leaf()
		ts.append(pad).append(fmt.Sprintf("key(%d): %v ", len(n.key), n.key)).append(n.key[:]).append("\n")

		if s, ok := n.value.(string); ok {
			ts.append(pad).append(fmt.Sprintf("val: %v\n", s))
		} else if b, ok := n.value.([]byte); ok {
			ts.append(pad).append(fmt.Sprintf("val: %v\n", string(b)))
		} else {
			ts.append(pad).append(fmt.Sprintf("val: %v\n", n.value))
		}

	}

	ts.append(pad).append("\n")
}

func (ts *treeStringer) rootNode(an *artNode) {
	ts.baseNode(an, 0, 0, 0)
}

/*
DumpNode returns Tree in the human readable format:
 package main

 import (
	"fmt"
	"github.com/plar/go-adaptive-radix-tree"
 )

 func main() {
 	tree := art.New()
	terms := []string{"A", "a", "aa"}
	for _, term := range terms {
		tree.Insert(art.Key(term), term)
	}
	fmt.Println(tree)
 }

 Output:
 ─── Node4 (0xc00008a240)
     prefix(0): [0 0 0 0 0 0 0 0 0 0][··········]
     keys: [65 97 · ·] [Aa··]
     children(2): [0xc00008a210 0xc00008a270 <nil> <nil> <nil>]
     ├── Leaf (0xc00008a210)
     │   key(1): [65] [A]
     │   val: A
     │
     ├── Node4 (0xc00008a270)
     │   prefix(0): [0 0 0 0 0 0 0 0 0 0][··········]
     │   keys: [97 · · ·] [a···]
     │   children(1): [0xc00008a260 <nil> <nil> <nil> 0xc00008a230]
     │   ├── Leaf (0xc00008a260)
     │   │   key(2): [97 97] [aa]
     │   │   val: aa
     │   │
     │   ├── nil
     │   ├── nil
     │   ├── nil
     │   └── Leaf (0xc00008a230)
     │       key(1): [97] [a]
     │       val: a
     │
     │
     ├── nil
     ├── nil
     └── nil
*/
func DumpNode(root *artNode) string {
	ts := &treeStringer{make([]depthStorage, 4096), bytes.NewBufferString("")}
	ts.rootNode(root)

	return ts.buf.String()
}
