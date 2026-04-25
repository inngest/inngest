package openapi3

import (
	"reflect"
	"strings"

	"github.com/oasdiff/yaml"
)

const originKey = "__origin__"

var originPtrType = reflect.TypeFor[*Origin]()

// Origin contains the origin of a collection.
// Key is the location of the collection itself.
// Fields is a map of the location of each scalar field in the collection.
// Sequences is a map of the location of each item in sequence-valued fields.
type Origin struct {
	Key       *Location             `json:"key,omitempty" yaml:"key,omitempty"`
	Fields    map[string]Location   `json:"fields,omitempty" yaml:"fields,omitempty"`
	Sequences map[string][]Location `json:"sequences,omitempty" yaml:"sequences,omitempty"`
}

// Location is a struct that contains the location of a field.
type Location struct {
	File   string `json:"file,omitempty" yaml:"file,omitempty"`
	Line   int    `json:"line,omitempty" yaml:"line,omitempty"`
	Column int    `json:"column,omitempty" yaml:"column,omitempty"`
	Name   string `json:"name,omitempty" yaml:"name,omitempty"`
}

// originFromSeq parses the compact []any sequence produced by yaml3's addOrigin.
//
// Format: [file, key_name, key_line, key_col, nf, f1_name, f1_delta, f1_col, ..., ns, s1_name, s1_count, s1_l0_delta, s1_c0, ...]
func originFromSeq(s []any) *Origin {
	// Need at least: file, key_name, key_line, key_col, nf, ns
	if len(s) < 6 {
		return nil
	}
	file, _ := s[0].(string)
	keyName, _ := s[1].(string)
	keyLine := toInt(s[2])
	keyCol := toInt(s[3])

	o := &Origin{
		Key: &Location{
			File:   file,
			Line:   keyLine,
			Column: keyCol,
			Name:   keyName,
		},
	}

	idx := 4
	nf := toInt(s[idx])
	idx++
	if nf > 0 && idx+nf*3 <= len(s) {
		o.Fields = make(map[string]Location, nf)
		for range nf {
			fname, _ := s[idx].(string)
			delta := toInt(s[idx+1])
			col := toInt(s[idx+2])
			o.Fields[fname] = Location{
				File:   file,
				Line:   keyLine + delta,
				Column: col,
				Name:   fname,
			}
			idx += 3
		}
	}

	if idx >= len(s) {
		return o
	}
	ns := toInt(s[idx])
	idx++
	if ns > 0 {
		o.Sequences = make(map[string][]Location, ns)
		for range ns {
			if idx >= len(s) {
				break
			}
			sname, _ := s[idx].(string)
			idx++
			if idx >= len(s) {
				break
			}
			count := toInt(s[idx])
			idx++
			locs := make([]Location, count)
			for j := 0; j < count && idx+2 < len(s); j++ {
				name, _ := s[idx].(string)
				delta := toInt(s[idx+1])
				col := toInt(s[idx+2])
				locs[j] = Location{File: file, Line: keyLine + delta, Column: col, Name: name}
				idx += 3
			}
			o.Sequences[sname] = locs
		}
	}
	return o
}

// toInt converts numeric types to int. Handles int/uint64 from YAML decoding.
func toInt(v any) int {
	switch n := v.(type) {
	case int:
		return n
	case uint64:
		return int(n)
	}
	return 0
}

// applyOrigins walks a Go struct tree and a parallel OriginTree, setting
// Origin fields on each struct from the extracted origin data.
func applyOrigins(v any, tree *yaml.OriginTree) {
	if tree == nil {
		return
	}
	applyOriginsToValue(reflect.ValueOf(v), tree)
}

func applyOriginsToValue(val reflect.Value, tree *yaml.OriginTree) {
	// Keep track of the last pointer so we can pass it to struct handlers
	// (needed for calling methods like Map() on maplike types).
	var ptr reflect.Value
	for val.Kind() == reflect.Pointer || val.Kind() == reflect.Interface {
		if val.IsNil() {
			return
		}
		if val.Kind() == reflect.Pointer {
			ptr = val
		}
		val = val.Elem()
	}

	switch val.Kind() {
	case reflect.Struct:
		applyOriginsToStruct(val, ptr, tree)
	case reflect.Map:
		applyOriginsToMap(val, tree)
	case reflect.Slice:
		applyOriginsToSlice(val, tree)
	}
}

func applyOriginsToStruct(val reflect.Value, ptr reflect.Value, tree *yaml.OriginTree) {
	typ := val.Type()

	// Set Origin field for structs whose Origin field has a "-" json tag.
	if tree.Origin != nil {
		if sf, ok := typ.FieldByName("Origin"); ok && sf.Type == originPtrType {
			tag := sf.Tag.Get("json")
			if tag == "-" {
				if s, ok := tree.Origin.([]any); ok {
					val.FieldByName("Origin").Set(reflect.ValueOf(originFromSeq(s)))
				}
			}
		}
	}

	// Recurse into exported struct fields using json tags
	for i := range typ.NumField() {
		sf := typ.Field(i)
		if !sf.IsExported() {
			continue
		}
		tag := jsonTagName(sf)
		if tag == "" || tag == "-" {
			continue
		}
		childTree := tree.Fields[tag]
		if childTree != nil {
			applyOriginsToValue(val.Field(i), childTree)
		}
	}

	// Handle wrapper types whose inner struct has no json tag:
	// - *Ref types (e.g. SchemaRef, ResponseRef) have a "Value" field
	// - BoolSchema (AdditionalProperties, UnevaluatedProperties, UnevaluatedItems) has a "Schema" field
	// The origin tree data applies to the inner struct, not a sub-key.
	for _, fieldName := range []string{"Value", "Schema"} {
		vf := val.FieldByName(fieldName)
		if !vf.IsValid() || vf.Kind() != reflect.Pointer || vf.IsNil() {
			continue
		}
		sf, _ := typ.FieldByName(fieldName)
		if sf.Tag.Get("json") == "" {
			applyOriginsToValue(vf, tree)
		}
	}

	// Handle "maplike" types (Paths, Responses, Callback) whose items are
	// stored in an unexported map accessible via a Map() method.
	// Use the original pointer (if available) since dereferenced values
	// are not addressable.
	receiver := val
	if ptr.IsValid() {
		receiver = ptr
	} else if val.CanAddr() {
		receiver = val.Addr()
	}
	if receiver.Kind() == reflect.Pointer {
		if mapMethod := receiver.MethodByName("Map"); mapMethod.IsValid() {
			results := mapMethod.Call(nil)
			if len(results) == 1 {
				applyOriginsToMap(results[0], tree)
			}
		}
	}
}

func applyOriginsToMap(val reflect.Value, tree *yaml.OriginTree) {
	if tree.Fields == nil {
		return
	}
	for _, key := range val.MapKeys() {
		childTree := tree.Fields[key.String()]
		if childTree == nil {
			continue
		}
		elem := val.MapIndex(key)
		// Map values are not addressable. For pointer-typed values we can
		// recurse directly. For value types we must copy, apply, and set back.
		if elem.Kind() == reflect.Pointer || elem.Kind() == reflect.Interface {
			applyOriginsToValue(elem, childTree)
		} else if elem.Kind() == reflect.Struct {
			// Copy to a settable value
			cp := reflect.New(elem.Type()).Elem()
			cp.Set(elem)
			applyOriginsToStruct(cp, reflect.Value{}, childTree)
			val.SetMapIndex(key, cp)
		}
	}
}

func applyOriginsToSlice(val reflect.Value, tree *yaml.OriginTree) {
	for i := 0; i < val.Len() && i < len(tree.Items); i++ {
		if tree.Items[i] != nil {
			applyOriginsToValue(val.Index(i), tree.Items[i])
		}
	}
}

// jsonTagName returns the JSON field name from a struct field's json tag.
func jsonTagName(f reflect.StructField) string {
	tag := f.Tag.Get("json")
	if tag == "" {
		return ""
	}
	name, _, _ := strings.Cut(tag, ",")
	return name
}
