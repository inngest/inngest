package bun

import (
	"context"
	"fmt"
	"reflect"
	"strings"
)

func indirect(v reflect.Value) reflect.Value {
	switch v.Kind() {
	case reflect.Interface:
		return indirect(v.Elem())
	case reflect.Ptr:
		return v.Elem()
	default:
		return v
	}
}

func walk(v reflect.Value, index []int, fn func(reflect.Value)) {
	v = reflect.Indirect(v)
	switch v.Kind() {
	case reflect.Slice:
		sliceLen := v.Len()
		for i := 0; i < sliceLen; i++ {
			visitField(v.Index(i), index, fn)
		}
	default:
		visitField(v, index, fn)
	}
}

func visitField(v reflect.Value, index []int, fn func(reflect.Value)) {
	v = reflect.Indirect(v)
	if len(index) > 0 {
		v = v.Field(index[0])
		if v.Kind() == reflect.Ptr && v.IsNil() {
			return
		}
		walk(v, index[1:], fn)
	} else {
		fn(v)
	}
}

func typeByIndex(t reflect.Type, index []int) reflect.Type {
	for _, x := range index {
		switch t.Kind() {
		case reflect.Ptr:
			t = t.Elem()
		case reflect.Slice:
			t = indirectType(t.Elem())
		}
		t = t.Field(x).Type
	}
	return indirectType(t)
}

func indirectType(t reflect.Type) reflect.Type {
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	return t
}

func sliceElemType(v reflect.Value) reflect.Type {
	elemType := v.Type().Elem()
	if elemType.Kind() == reflect.Interface && v.Len() > 0 {
		return indirect(v.Index(0).Elem()).Type()
	}
	return indirectType(elemType)
}

// appendComment adds comment in the header of the query into buffer
func appendComment(b []byte, name string) []byte {
	if name == "" {
		return b
	}
	name = strings.Map(func(r rune) rune {
		if r == '\x00' {
			return -1
		}
		return r
	}, name)
	name = strings.ReplaceAll(name, `/*`, `/\*`)
	name = strings.ReplaceAll(name, `*/`, `*\/`)
	return append(b, fmt.Sprintf("/* %s */ ", name)...)
}

// queryCommentCtxKey is a context key for setting a query comment on a context instead of calling the Comment("...") API directly
type queryCommentCtxKey struct{}

// WithComment returns a context that includes a comment that may be included in a query for debugging
//
// If a context with an attached query is used, a comment set by the Comment("...") API will be overwritten.
func WithComment(ctx context.Context, comment string) context.Context {
	return context.WithValue(ctx, queryCommentCtxKey{}, comment)
}

// commenter describes the Comment interface implemented by all of the query types
type commenter[T any] interface {
	Comment(string) T
}

// setCommentFromContext sets the comment on the given query from the supplied context if one is set using the Comment(...) method.
func setCommentFromContext[T any](ctx context.Context, q commenter[T]) {
	s, _ := ctx.Value(queryCommentCtxKey{}).(string)
	if s != "" {
		q.Comment(s)
	}
}
