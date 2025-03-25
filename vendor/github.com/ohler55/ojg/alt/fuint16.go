// Copyright (c) 2021, Peter Ohler, All rights reserved.

package alt

import (
	"reflect"
	"strconv"
	"unsafe"
)

var uint16ValFuncs = [8]valFunc{
	valUint16,
	valUint16AsString,
	valUint16NotEmpty,
	valUint16NotEmptyAsString,
	ivalUint16,
	ivalUint16AsString,
	ivalUint16NotEmpty,
	ivalUint16NotEmptyAsString,
}

func valUint16(fi *finfo, rv reflect.Value, addr uintptr) (any, reflect.Value, bool) {
	return *(*uint16)(unsafe.Pointer(addr + fi.offset)), nilValue, false
}

func valUint16AsString(fi *finfo, rv reflect.Value, addr uintptr) (any, reflect.Value, bool) {
	return strconv.FormatUint(uint64(*(*uint16)(unsafe.Pointer(addr + fi.offset))), 10), nilValue, false
}

func valUint16NotEmpty(fi *finfo, rv reflect.Value, addr uintptr) (any, reflect.Value, bool) {
	v := *(*uint16)(unsafe.Pointer(addr + fi.offset))
	return v, nilValue, v == 0
}

func valUint16NotEmptyAsString(fi *finfo, rv reflect.Value, addr uintptr) (any, reflect.Value, bool) {
	v := *(*uint16)(unsafe.Pointer(addr + fi.offset))
	if v == 0 {
		return nil, nilValue, true
	}
	return strconv.FormatUint(uint64(v), 10), nilValue, false
}

func ivalUint16(fi *finfo, rv reflect.Value, addr uintptr) (any, reflect.Value, bool) {
	return rv.FieldByIndex(fi.index).Interface().(uint16), nilValue, false
}

func ivalUint16AsString(fi *finfo, rv reflect.Value, addr uintptr) (any, reflect.Value, bool) {
	return strconv.FormatUint(uint64(rv.FieldByIndex(fi.index).Interface().(uint16)), 10), nilValue, false
}

func ivalUint16NotEmpty(fi *finfo, rv reflect.Value, addr uintptr) (any, reflect.Value, bool) {
	v := rv.FieldByIndex(fi.index).Interface().(uint16)
	return v, nilValue, v == 0
}

func ivalUint16NotEmptyAsString(fi *finfo, rv reflect.Value, addr uintptr) (any, reflect.Value, bool) {
	v := rv.FieldByIndex(fi.index).Interface().(uint16)
	if v == 0 {
		return nil, nilValue, true
	}
	return strconv.FormatUint(uint64(v), 10), nilValue, false
}
