// Copyright (c) 2020, Peter Ohler, All rights reserved.

package alt

import (
	"reflect"
	"strings"
)

type composer struct {
	fun     RecomposeFunc
	any     RecomposeAnyFunc
	short   string
	full    string
	rtype   reflect.Type
	indexes map[string]reflect.StructField
}

func indexType(rt reflect.Type) (im map[string]reflect.StructField) {
	i := rt.NumField()
	if 0 < i {
		im = map[string]reflect.StructField{}
		for i--; 0 <= i; i-- {
			f := rt.Field(i)
			if 0 < len(f.PkgPath) {
				continue
			}
			if f.Anonymous {
				fim := indexType(f.Type)
				// prepend index and add to im
				for k := range fim {
					ff := fim[k]
					ff.Index = append([]int{i}, ff.Index...)
					im[k] = ff
				}
			} else if k, _ := f.Tag.Lookup("json"); 0 < len(k) {
				parts := strings.Split(k, ",")
				switch parts[0] {
				case "":
					k = strings.ToLower(f.Name)
				case "-":
					if 1 < len(parts) {
						k = "-"
					} else {
						continue
					}
				default:
					k = parts[0]
				}
				im[k] = f
			} else {
				im[f.Name] = f
			}
		}
	}
	return
}
