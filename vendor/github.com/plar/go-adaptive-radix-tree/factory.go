package art

import (
	"unsafe"
)

type nodeFactory interface {
	newNode4() *artNode
	newNode16() *artNode
	newNode48() *artNode
	newNode256() *artNode
	newLeaf(key Key, value interface{}) *artNode
}

// make sure that objFactory implements all methods of nodeFactory interface
var _ nodeFactory = &objFactory{}

var factory = newObjFactory()

func newTree() *tree {
	return &tree{}
}

type objFactory struct{}

func newObjFactory() nodeFactory {
	return &objFactory{}
}

// Simple obj factory implementation
func (f *objFactory) newNode4() *artNode {
	return &artNode{kind: Node4, ref: unsafe.Pointer(new(node4))}
}

func (f *objFactory) newNode16() *artNode {
	return &artNode{kind: Node16, ref: unsafe.Pointer(&node16{})}
}

func (f *objFactory) newNode48() *artNode {
	return &artNode{kind: Node48, ref: unsafe.Pointer(&node48{})}
}

func (f *objFactory) newNode256() *artNode {
	return &artNode{kind: Node256, ref: unsafe.Pointer(&node256{})}
}

func (f *objFactory) newLeaf(key Key, value interface{}) *artNode {
	clonedKey := make(Key, len(key))
	copy(clonedKey, key)
	return &artNode{
		kind: Leaf,
		ref:  unsafe.Pointer(&leaf{key: clonedKey, value: value}),
	}
}
