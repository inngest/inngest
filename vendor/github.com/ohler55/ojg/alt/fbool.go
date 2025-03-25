// Copyright (c) 2021, Peter Ohler, All rights reserved.

package alt

import (
	"reflect"
	"unsafe"
)

var boolValFuncs = [8]valFunc{
	valBool,
	valBoolAsString,
	valBoolNotEmpty,
	valBoolNotEmptyAsString,
	ivalBool,
	ivalBoolAsString,
	ivalBoolNotEmpty,
	ivalBoolNotEmptyAsString,
}

func valBool(fi *finfo, rv reflect.Value, addr uintptr) (any, reflect.Value, bool) {
	return *(*bool)(unsafe.Pointer(addr + fi.offset)), nilValue, false
}

func valBoolAsString(fi *finfo, rv reflect.Value, addr uintptr) (any, reflect.Value, bool) {
	if *(*bool)(unsafe.Pointer(addr + fi.offset)) {
		return "true", nilValue, false
	}
	return "false", nilValue, false
}

func valBoolNotEmpty(fi *finfo, rv reflect.Value, addr uintptr) (any, reflect.Value, bool) {
	v := *(*bool)(unsafe.Pointer(addr + fi.offset))
	return v, nilValue, !v
}

func valBoolNotEmptyAsString(fi *finfo, rv reflect.Value, addr uintptr) (any, reflect.Value, bool) {
	if *(*bool)(unsafe.Pointer(addr + fi.offset)) {
		return "true", nilValue, false
	}
	return "false", nilValue, true
}

func ivalBool(fi *finfo, rv reflect.Value, addr uintptr) (any, reflect.Value, bool) {
	return rv.FieldByIndex(fi.index).Interface(), nilValue, false
}

func ivalBoolAsString(fi *finfo, rv reflect.Value, addr uintptr) (any, reflect.Value, bool) {
	if rv.FieldByIndex(fi.index).Interface().(bool) {
		return "true", nilValue, false
	}
	return "false", nilValue, false
}

func ivalBoolNotEmpty(fi *finfo, rv reflect.Value, addr uintptr) (any, reflect.Value, bool) {
	v := rv.FieldByIndex(fi.index).Interface().(bool)
	return v, nilValue, !v
}

func ivalBoolNotEmptyAsString(fi *finfo, rv reflect.Value, addr uintptr) (any, reflect.Value, bool) {
	if rv.FieldByIndex(fi.index).Interface().(bool) {
		return "true", nilValue, false
	}
	return "false", nilValue, true
}
