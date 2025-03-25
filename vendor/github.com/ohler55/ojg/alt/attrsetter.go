// Copyright (c) 2020, Peter Ohler, All rights reserved.

package alt

// AttrSetter interface is for objects that can set attributes using the
// SetAttr() function.
type AttrSetter interface {

	// SetAttr sets an attribute of the object associated with the path.
	SetAttr(attr string, val any) error
}
