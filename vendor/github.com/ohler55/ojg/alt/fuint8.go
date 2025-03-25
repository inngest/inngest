// Copyright (c) 2021, Peter Ohler, All rights reserved.

package alt

import (
	"reflect"
	"strconv"
	"unsafe"
)

var uint8ValFuncs = [8]valFunc{
	valUint8,
	valUint8AsString,
	valUint8NotEmpty,
	valUint8NotEmptyAsString,
	ivalUint8,
	ivalUint8AsString,
	ivalUint8NotEmpty,
	ivalUint8NotEmptyAsString,
}

func valUint8(fi *finfo, rv reflect.Value, addr uintptr) (any, reflect.Value, bool) {
	return *(*uint8)(unsafe.Pointer(addr + fi.offset)), nilValue, false
}

func valUint8AsString(fi *finfo, rv reflect.Value, addr uintptr) (any, reflect.Value, bool) {
	return strconv.FormatUint(uint64(*(*uint8)(unsafe.Pointer(addr + fi.offset))), 10), nilValue, false
}

func valUint8NotEmpty(fi *finfo, rv reflect.Value, addr uintptr) (any, reflect.Value, bool) {
	v := *(*uint8)(unsafe.Pointer(addr + fi.offset))
	return v, nilValue, v == 0
}

func valUint8NotEmptyAsString(fi *finfo, rv reflect.Value, addr uintptr) (any, reflect.Value, bool) {
	v := *(*uint8)(unsafe.Pointer(addr + fi.offset))
	if v == 0 {
		return nil, nilValue, true
	}
	return strconv.FormatUint(uint64(v), 10), nilValue, false
}

func ivalUint8(fi *finfo, rv reflect.Value, addr uintptr) (any, reflect.Value, bool) {
	return rv.FieldByIndex(fi.index).Interface().(uint8), nilValue, false
}

func ivalUint8AsString(fi *finfo, rv reflect.Value, addr uintptr) (any, reflect.Value, bool) {
	return strconv.FormatUint(uint64(rv.FieldByIndex(fi.index).Interface().(uint8)), 10), nilValue, false
}

func ivalUint8NotEmpty(fi *finfo, rv reflect.Value, addr uintptr) (any, reflect.Value, bool) {
	v := rv.FieldByIndex(fi.index).Interface().(uint8)
	return v, nilValue, v == 0
}

func ivalUint8NotEmptyAsString(fi *finfo, rv reflect.Value, addr uintptr) (any, reflect.Value, bool) {
	v := rv.FieldByIndex(fi.index).Interface().(uint8)
	if v == 0 {
		return nil, nilValue, true
	}
	return strconv.FormatUint(uint64(v), 10), nilValue, false
}
