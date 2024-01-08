// Copyright (c) 2021, Peter Ohler, All rights reserved.

package alt

import (
	"reflect"
	"strconv"
	"unsafe"
)

var intValFuncs = [8]valFunc{
	valInt,
	valIntAsString,
	valIntNotEmpty,
	valIntNotEmptyAsString,
	ivalInt,
	ivalIntAsString,
	ivalIntNotEmpty,
	ivalIntNotEmptyAsString,
}

func valInt(fi *finfo, rv reflect.Value, addr uintptr) (any, reflect.Value, bool) {
	return *(*int)(unsafe.Pointer(addr + fi.offset)), nilValue, false
}

func valIntAsString(fi *finfo, rv reflect.Value, addr uintptr) (any, reflect.Value, bool) {
	return strconv.FormatInt(int64(*(*int)(unsafe.Pointer(addr + fi.offset))), 10), nilValue, false
}

func valIntNotEmpty(fi *finfo, rv reflect.Value, addr uintptr) (any, reflect.Value, bool) {
	v := *(*int)(unsafe.Pointer(addr + fi.offset))
	return v, nilValue, v == 0
}

func valIntNotEmptyAsString(fi *finfo, rv reflect.Value, addr uintptr) (any, reflect.Value, bool) {
	v := *(*int)(unsafe.Pointer(addr + fi.offset))
	if v == 0 {
		return nil, nilValue, true
	}
	return strconv.FormatInt(int64(v), 10), nilValue, false
}

func ivalInt(fi *finfo, rv reflect.Value, addr uintptr) (any, reflect.Value, bool) {
	return rv.FieldByIndex(fi.index).Interface().(int), nilValue, false
}

func ivalIntAsString(fi *finfo, rv reflect.Value, addr uintptr) (any, reflect.Value, bool) {
	return strconv.FormatInt(int64(rv.FieldByIndex(fi.index).Interface().(int)), 10), nilValue, false
}

func ivalIntNotEmpty(fi *finfo, rv reflect.Value, addr uintptr) (any, reflect.Value, bool) {
	v := rv.FieldByIndex(fi.index).Interface().(int)
	return v, nilValue, v == 0
}

func ivalIntNotEmptyAsString(fi *finfo, rv reflect.Value, addr uintptr) (any, reflect.Value, bool) {
	v := rv.FieldByIndex(fi.index).Interface().(int)
	if v == 0 {
		return nil, nilValue, true
	}
	return strconv.FormatInt(int64(v), 10), nilValue, false
}
