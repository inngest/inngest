// Copyright (c) 2021, Peter Ohler, All rights reserved.

package alt

import (
	"reflect"
	"strconv"
	"unsafe"
)

var uint32ValFuncs = [8]valFunc{
	valUint32,
	valUint32AsString,
	valUint32NotEmpty,
	valUint32NotEmptyAsString,
	ivalUint32,
	ivalUint32AsString,
	ivalUint32NotEmpty,
	ivalUint32NotEmptyAsString,
}

func valUint32(fi *finfo, rv reflect.Value, addr uintptr) (any, reflect.Value, bool) {
	return *(*uint32)(unsafe.Pointer(addr + fi.offset)), nilValue, false
}

func valUint32AsString(fi *finfo, rv reflect.Value, addr uintptr) (any, reflect.Value, bool) {
	return strconv.FormatUint(uint64(*(*uint32)(unsafe.Pointer(addr + fi.offset))), 10), nilValue, false
}

func valUint32NotEmpty(fi *finfo, rv reflect.Value, addr uintptr) (any, reflect.Value, bool) {
	v := *(*uint32)(unsafe.Pointer(addr + fi.offset))
	return v, nilValue, v == 0
}

func valUint32NotEmptyAsString(fi *finfo, rv reflect.Value, addr uintptr) (any, reflect.Value, bool) {
	v := *(*uint32)(unsafe.Pointer(addr + fi.offset))
	if v == 0 {
		return nil, nilValue, true
	}
	return strconv.FormatUint(uint64(v), 10), nilValue, false
}

func ivalUint32(fi *finfo, rv reflect.Value, addr uintptr) (any, reflect.Value, bool) {
	return rv.FieldByIndex(fi.index).Interface().(uint32), nilValue, false
}

func ivalUint32AsString(fi *finfo, rv reflect.Value, addr uintptr) (any, reflect.Value, bool) {
	return strconv.FormatUint(uint64(rv.FieldByIndex(fi.index).Interface().(uint32)), 10), nilValue, false
}

func ivalUint32NotEmpty(fi *finfo, rv reflect.Value, addr uintptr) (any, reflect.Value, bool) {
	v := rv.FieldByIndex(fi.index).Interface().(uint32)
	return v, nilValue, v == 0
}

func ivalUint32NotEmptyAsString(fi *finfo, rv reflect.Value, addr uintptr) (any, reflect.Value, bool) {
	v := rv.FieldByIndex(fi.index).Interface().(uint32)
	if v == 0 {
		return nil, nilValue, true
	}
	return strconv.FormatUint(uint64(v), 10), nilValue, false
}
