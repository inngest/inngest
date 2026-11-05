package internal

import "reflect"

var ifaceType = reflect.TypeFor[any]()

type MapKey struct {
	iface any
}

func NewMapKey(is []any) MapKey {
	return MapKey{
		iface: newMapKey(is),
	}
}

func newMapKey(is []any) any {
	switch len(is) {
	case 1:
		ptr := new([1]any)
		copy((*ptr)[:], is)
		return *ptr
	case 2:
		ptr := new([2]any)
		copy((*ptr)[:], is)
		return *ptr
	case 3:
		ptr := new([3]any)
		copy((*ptr)[:], is)
		return *ptr
	case 4:
		ptr := new([4]any)
		copy((*ptr)[:], is)
		return *ptr
	case 5:
		ptr := new([5]any)
		copy((*ptr)[:], is)
		return *ptr
	case 6:
		ptr := new([6]any)
		copy((*ptr)[:], is)
		return *ptr
	case 7:
		ptr := new([7]any)
		copy((*ptr)[:], is)
		return *ptr
	case 8:
		ptr := new([8]any)
		copy((*ptr)[:], is)
		return *ptr
	case 9:
		ptr := new([9]any)
		copy((*ptr)[:], is)
		return *ptr
	case 10:
		ptr := new([10]any)
		copy((*ptr)[:], is)
		return *ptr
	default:
	}

	at := reflect.New(reflect.ArrayOf(len(is), ifaceType)).Elem()
	for i, v := range is {
		*(at.Index(i).Addr().Interface().(*any)) = v
	}
	return at.Interface()
}
