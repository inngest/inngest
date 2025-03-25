// Copyright (c) 2020, Peter Ohler, All rights reserved.

package jp

import (
	"time"

	"github.com/ohler55/ojg/alt"
	"github.com/ohler55/ojg/gen"
)

// Walk data and call the cb callback for each node in the data. The path is
// reused in each call so if the path needs to be save it should be copied.
func Walk(data any, cb func(path Expr, value any), justLeaves ...bool) {
	path := Expr{Root('$')}
	walk(path, data, cb, 0 < len(justLeaves) && justLeaves[0])
}

func walk(path Expr, data any, cb func(path Expr, value any), justLeaves bool) {
top:
	switch td := data.(type) {
	case nil, bool, int64, float64, string,
		int, int8, int16, int32, uint, uint8, uint16, uint32, uint64, float32,
		[]byte, time.Time:
		// leaf node
		cb(path, data)
	case []any:
		if !justLeaves {
			cb(path, data)
		}
		pi := len(path)
		path = append(path, nil)
		for i, v := range td {
			path[pi] = Nth(i)
			walk(path, v, cb, justLeaves)
		}
	case map[string]any:
		if !justLeaves {
			cb(path, data)
		}
		pi := len(path)
		path = append(path, nil)
		for k, v := range td {
			path[pi] = Child(k)
			walk(path, v, cb, justLeaves)
		}
	case gen.Array:
		if !justLeaves {
			cb(path, data)
		}
		pi := len(path)
		path = append(path, nil)
		for i, v := range td {
			path[pi] = Nth(i)
			walk(path, v, cb, justLeaves)
		}
	case gen.Object:
		if !justLeaves {
			cb(path, data)
		}
		pi := len(path)
		path = append(path, nil)
		for k, v := range td {
			path[pi] = Child(k)
			walk(path, v, cb, justLeaves)
		}
	case alt.Simplifier:
		data = td.Simplify()
		goto top
	default:
		cb(path, data)
	}
}
