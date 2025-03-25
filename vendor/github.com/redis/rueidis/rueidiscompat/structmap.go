// Copyright (c) 2013 The github.com/go-redis/redis Authors.
// All rights reserved.
//
// Redistribution and use in source and binary forms, with or without
// modification, are permitted provided that the following conditions are
// met:
//
// * Redistributions of source code must retain the above copyright
// notice, this list of conditions and the following disclaimer.
// * Redistributions in binary form must reproduce the above
// copyright notice, this list of conditions and the following disclaimer
// in the documentation and/or other materials provided with the
// distribution.
//
// THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS
// "AS IS" AND ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT
// LIMITED TO, THE IMPLIED WARRANTIES OF MERCHANTABILITY AND FITNESS FOR
// A PARTICULAR PURPOSE ARE DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT
// OWNER OR CONTRIBUTORS BE LIABLE FOR ANY DIRECT, INDIRECT, INCIDENTAL,
// SPECIAL, EXEMPLARY, OR CONSEQUENTIAL DAMAGES (INCLUDING, BUT NOT
// LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR SERVICES; LOSS OF USE,
// DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER CAUSED AND ON ANY
// THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY, OR TORT
// (INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE
// OF THIS SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.

package rueidiscompat

import (
	"encoding"
	"fmt"
	"reflect"
	"strings"
	"sync"
)

// structMap contains the map of struct fields for target structs
// indexed by the struct type.
type structMap struct {
	m sync.Map
}

func newStructMap() *structMap {
	return new(structMap)
}

func (s *structMap) get(t reflect.Type) *structSpec {
	if v, ok := s.m.Load(t); ok {
		return v.(*structSpec)
	}

	spec := newStructSpec(t, "redis")
	s.m.Store(t, spec)
	return spec
}

//------------------------------------------------------------------------------

// structSpec contains the list of all fields in a target struct.
type structSpec struct {
	m map[string]*structField
}

func (s *structSpec) set(tag string, sf *structField) {
	s.m[tag] = sf
}

func newStructSpec(t reflect.Type, fieldTag string) *structSpec {
	numField := t.NumField()
	out := &structSpec{
		m: make(map[string]*structField, numField),
	}

	for i := 0; i < numField; i++ {
		f := t.Field(i)

		tag := f.Tag.Get(fieldTag)
		if tag == "" || tag == "-" {
			continue
		}

		tag = strings.Split(tag, ",")[0]
		if tag == "" {
			continue
		}

		// Added a check for Pointer here. If it's a Pointer use the built-in decoder of the element.
		// This works, because in Scan() #129-131
		// if isPtr && v.IsNil() {
		//     v.Set(reflect.New(v.Type().Elem()))
		// }
		// A new value is set
		if f.Type.Kind() == reflect.Pointer {
			out.set(tag, &structField{index: i, fn: decoders[f.Type.Elem().Kind()]})
		} else {
			// Use the built-in decoder.
			out.set(tag, &structField{index: i, fn: decoders[f.Type.Kind()]})
		}
	}

	return out
}

//------------------------------------------------------------------------------

// structField represents a single field in a target struct.
type structField struct {
	fn    decoderFunc
	index int
}

//------------------------------------------------------------------------------

type StructValue struct {
	spec  *structSpec
	value reflect.Value
}

func (s StructValue) Scan(key string, value string) error {
	field, ok := s.spec.m[key]
	if !ok {
		return nil
	}

	v := s.value.Field(field.index)
	isPtr := v.Kind() == reflect.Ptr

	if isPtr && v.IsNil() {
		v.Set(reflect.New(v.Type().Elem()))
	}
	if !isPtr && v.Type().Name() != "" && v.CanAddr() {
		v = v.Addr()
		isPtr = true
	}

	if isPtr && v.Type().NumMethod() > 0 && v.CanInterface() {
		switch scan := v.Interface().(type) {
		case Scanner:
			return scan.ScanRedis(value)
		case encoding.TextUnmarshaler:
			return scan.UnmarshalText([]byte(value))
		}
	}

	if isPtr {
		v = v.Elem()
	}

	if err := field.fn(v, value); err != nil {
		t := s.value.Type()
		return fmt.Errorf("cannot scan redis.result %s into struct field %s.%s of type %s, error-%s",
			value, t.Name(), t.Field(field.index).Name, t.Field(field.index).Type, err.Error())
	}
	return nil
}
