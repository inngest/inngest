// Code generated DO NOT EDIT

package cmds

import "strconv"

type JsonArrappend Completed

func (b Builder) JsonArrappend() (c JsonArrappend) {
	c = JsonArrappend{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "JSON.ARRAPPEND")
	return c
}

func (c JsonArrappend) Key(key string) JsonArrappendKey {
	if c.ks&NoSlot == NoSlot {
		c.ks = NoSlot | slot(key)
	} else {
		c.ks = check(c.ks, slot(key))
	}
	c.cs.s = append(c.cs.s, key)
	return (JsonArrappendKey)(c)
}

type JsonArrappendKey Completed

func (c JsonArrappendKey) Path(path string) JsonArrappendPath {
	c.cs.s = append(c.cs.s, path)
	return (JsonArrappendPath)(c)
}

func (c JsonArrappendKey) Value(value ...string) JsonArrappendValue {
	c.cs.s = append(c.cs.s, value...)
	return (JsonArrappendValue)(c)
}

type JsonArrappendPath Completed

func (c JsonArrappendPath) Value(value ...string) JsonArrappendValue {
	c.cs.s = append(c.cs.s, value...)
	return (JsonArrappendValue)(c)
}

type JsonArrappendValue Completed

func (c JsonArrappendValue) Value(value ...string) JsonArrappendValue {
	c.cs.s = append(c.cs.s, value...)
	return c
}

func (c JsonArrappendValue) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type JsonArrindex Completed

func (b Builder) JsonArrindex() (c JsonArrindex) {
	c = JsonArrindex{cs: get(), ks: b.ks, cf: readonly}
	c.cs.s = append(c.cs.s, "JSON.ARRINDEX")
	return c
}

func (c JsonArrindex) Key(key string) JsonArrindexKey {
	if c.ks&NoSlot == NoSlot {
		c.ks = NoSlot | slot(key)
	} else {
		c.ks = check(c.ks, slot(key))
	}
	c.cs.s = append(c.cs.s, key)
	return (JsonArrindexKey)(c)
}

type JsonArrindexKey Completed

func (c JsonArrindexKey) Path(path string) JsonArrindexPath {
	c.cs.s = append(c.cs.s, path)
	return (JsonArrindexPath)(c)
}

type JsonArrindexPath Completed

func (c JsonArrindexPath) Value(value string) JsonArrindexValue {
	c.cs.s = append(c.cs.s, value)
	return (JsonArrindexValue)(c)
}

type JsonArrindexStartStart Completed

func (c JsonArrindexStartStart) Stop(stop int64) JsonArrindexStartStop {
	c.cs.s = append(c.cs.s, strconv.FormatInt(stop, 10))
	return (JsonArrindexStartStop)(c)
}

func (c JsonArrindexStartStart) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

func (c JsonArrindexStartStart) Cache() Cacheable {
	c.cs.Build()
	return Cacheable(c)
}

type JsonArrindexStartStop Completed

func (c JsonArrindexStartStop) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

func (c JsonArrindexStartStop) Cache() Cacheable {
	c.cs.Build()
	return Cacheable(c)
}

type JsonArrindexValue Completed

func (c JsonArrindexValue) Start(start int64) JsonArrindexStartStart {
	c.cs.s = append(c.cs.s, strconv.FormatInt(start, 10))
	return (JsonArrindexStartStart)(c)
}

func (c JsonArrindexValue) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

func (c JsonArrindexValue) Cache() Cacheable {
	c.cs.Build()
	return Cacheable(c)
}

type JsonArrinsert Completed

func (b Builder) JsonArrinsert() (c JsonArrinsert) {
	c = JsonArrinsert{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "JSON.ARRINSERT")
	return c
}

func (c JsonArrinsert) Key(key string) JsonArrinsertKey {
	if c.ks&NoSlot == NoSlot {
		c.ks = NoSlot | slot(key)
	} else {
		c.ks = check(c.ks, slot(key))
	}
	c.cs.s = append(c.cs.s, key)
	return (JsonArrinsertKey)(c)
}

type JsonArrinsertIndex Completed

func (c JsonArrinsertIndex) Value(value ...string) JsonArrinsertValue {
	c.cs.s = append(c.cs.s, value...)
	return (JsonArrinsertValue)(c)
}

type JsonArrinsertKey Completed

func (c JsonArrinsertKey) Path(path string) JsonArrinsertPath {
	c.cs.s = append(c.cs.s, path)
	return (JsonArrinsertPath)(c)
}

type JsonArrinsertPath Completed

func (c JsonArrinsertPath) Index(index int64) JsonArrinsertIndex {
	c.cs.s = append(c.cs.s, strconv.FormatInt(index, 10))
	return (JsonArrinsertIndex)(c)
}

type JsonArrinsertValue Completed

func (c JsonArrinsertValue) Value(value ...string) JsonArrinsertValue {
	c.cs.s = append(c.cs.s, value...)
	return c
}

func (c JsonArrinsertValue) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type JsonArrlen Completed

func (b Builder) JsonArrlen() (c JsonArrlen) {
	c = JsonArrlen{cs: get(), ks: b.ks, cf: readonly}
	c.cs.s = append(c.cs.s, "JSON.ARRLEN")
	return c
}

func (c JsonArrlen) Key(key string) JsonArrlenKey {
	if c.ks&NoSlot == NoSlot {
		c.ks = NoSlot | slot(key)
	} else {
		c.ks = check(c.ks, slot(key))
	}
	c.cs.s = append(c.cs.s, key)
	return (JsonArrlenKey)(c)
}

type JsonArrlenKey Completed

func (c JsonArrlenKey) Path(path string) JsonArrlenPath {
	c.cs.s = append(c.cs.s, path)
	return (JsonArrlenPath)(c)
}

func (c JsonArrlenKey) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

func (c JsonArrlenKey) Cache() Cacheable {
	c.cs.Build()
	return Cacheable(c)
}

type JsonArrlenPath Completed

func (c JsonArrlenPath) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

func (c JsonArrlenPath) Cache() Cacheable {
	c.cs.Build()
	return Cacheable(c)
}

type JsonArrpop Completed

func (b Builder) JsonArrpop() (c JsonArrpop) {
	c = JsonArrpop{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "JSON.ARRPOP")
	return c
}

func (c JsonArrpop) Key(key string) JsonArrpopKey {
	if c.ks&NoSlot == NoSlot {
		c.ks = NoSlot | slot(key)
	} else {
		c.ks = check(c.ks, slot(key))
	}
	c.cs.s = append(c.cs.s, key)
	return (JsonArrpopKey)(c)
}

type JsonArrpopKey Completed

func (c JsonArrpopKey) Path(path string) JsonArrpopPathPath {
	c.cs.s = append(c.cs.s, path)
	return (JsonArrpopPathPath)(c)
}

func (c JsonArrpopKey) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type JsonArrpopPathIndex Completed

func (c JsonArrpopPathIndex) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type JsonArrpopPathPath Completed

func (c JsonArrpopPathPath) Index(index int64) JsonArrpopPathIndex {
	c.cs.s = append(c.cs.s, strconv.FormatInt(index, 10))
	return (JsonArrpopPathIndex)(c)
}

func (c JsonArrpopPathPath) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type JsonArrtrim Completed

func (b Builder) JsonArrtrim() (c JsonArrtrim) {
	c = JsonArrtrim{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "JSON.ARRTRIM")
	return c
}

func (c JsonArrtrim) Key(key string) JsonArrtrimKey {
	if c.ks&NoSlot == NoSlot {
		c.ks = NoSlot | slot(key)
	} else {
		c.ks = check(c.ks, slot(key))
	}
	c.cs.s = append(c.cs.s, key)
	return (JsonArrtrimKey)(c)
}

type JsonArrtrimKey Completed

func (c JsonArrtrimKey) Path(path string) JsonArrtrimPath {
	c.cs.s = append(c.cs.s, path)
	return (JsonArrtrimPath)(c)
}

type JsonArrtrimPath Completed

func (c JsonArrtrimPath) Start(start int64) JsonArrtrimStart {
	c.cs.s = append(c.cs.s, strconv.FormatInt(start, 10))
	return (JsonArrtrimStart)(c)
}

type JsonArrtrimStart Completed

func (c JsonArrtrimStart) Stop(stop int64) JsonArrtrimStop {
	c.cs.s = append(c.cs.s, strconv.FormatInt(stop, 10))
	return (JsonArrtrimStop)(c)
}

type JsonArrtrimStop Completed

func (c JsonArrtrimStop) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type JsonClear Completed

func (b Builder) JsonClear() (c JsonClear) {
	c = JsonClear{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "JSON.CLEAR")
	return c
}

func (c JsonClear) Key(key string) JsonClearKey {
	if c.ks&NoSlot == NoSlot {
		c.ks = NoSlot | slot(key)
	} else {
		c.ks = check(c.ks, slot(key))
	}
	c.cs.s = append(c.cs.s, key)
	return (JsonClearKey)(c)
}

type JsonClearKey Completed

func (c JsonClearKey) Path(path string) JsonClearPath {
	c.cs.s = append(c.cs.s, path)
	return (JsonClearPath)(c)
}

func (c JsonClearKey) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type JsonClearPath Completed

func (c JsonClearPath) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type JsonDebugHelp Completed

func (b Builder) JsonDebugHelp() (c JsonDebugHelp) {
	c = JsonDebugHelp{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "JSON.DEBUG", "HELP")
	return c
}

func (c JsonDebugHelp) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type JsonDebugMemory Completed

func (b Builder) JsonDebugMemory() (c JsonDebugMemory) {
	c = JsonDebugMemory{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "JSON.DEBUG", "MEMORY")
	return c
}

func (c JsonDebugMemory) Key(key string) JsonDebugMemoryKey {
	if c.ks&NoSlot == NoSlot {
		c.ks = NoSlot | slot(key)
	} else {
		c.ks = check(c.ks, slot(key))
	}
	c.cs.s = append(c.cs.s, key)
	return (JsonDebugMemoryKey)(c)
}

type JsonDebugMemoryKey Completed

func (c JsonDebugMemoryKey) Path(path string) JsonDebugMemoryPath {
	c.cs.s = append(c.cs.s, path)
	return (JsonDebugMemoryPath)(c)
}

func (c JsonDebugMemoryKey) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type JsonDebugMemoryPath Completed

func (c JsonDebugMemoryPath) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type JsonDel Completed

func (b Builder) JsonDel() (c JsonDel) {
	c = JsonDel{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "JSON.DEL")
	return c
}

func (c JsonDel) Key(key string) JsonDelKey {
	if c.ks&NoSlot == NoSlot {
		c.ks = NoSlot | slot(key)
	} else {
		c.ks = check(c.ks, slot(key))
	}
	c.cs.s = append(c.cs.s, key)
	return (JsonDelKey)(c)
}

type JsonDelKey Completed

func (c JsonDelKey) Path(path string) JsonDelPath {
	c.cs.s = append(c.cs.s, path)
	return (JsonDelPath)(c)
}

func (c JsonDelKey) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type JsonDelPath Completed

func (c JsonDelPath) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type JsonForget Completed

func (b Builder) JsonForget() (c JsonForget) {
	c = JsonForget{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "JSON.FORGET")
	return c
}

func (c JsonForget) Key(key string) JsonForgetKey {
	if c.ks&NoSlot == NoSlot {
		c.ks = NoSlot | slot(key)
	} else {
		c.ks = check(c.ks, slot(key))
	}
	c.cs.s = append(c.cs.s, key)
	return (JsonForgetKey)(c)
}

type JsonForgetKey Completed

func (c JsonForgetKey) Path(path string) JsonForgetPath {
	c.cs.s = append(c.cs.s, path)
	return (JsonForgetPath)(c)
}

func (c JsonForgetKey) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type JsonForgetPath Completed

func (c JsonForgetPath) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type JsonGet Completed

func (b Builder) JsonGet() (c JsonGet) {
	c = JsonGet{cs: get(), ks: b.ks, cf: readonly}
	c.cs.s = append(c.cs.s, "JSON.GET")
	return c
}

func (c JsonGet) Key(key string) JsonGetKey {
	if c.ks&NoSlot == NoSlot {
		c.ks = NoSlot | slot(key)
	} else {
		c.ks = check(c.ks, slot(key))
	}
	c.cs.s = append(c.cs.s, key)
	return (JsonGetKey)(c)
}

type JsonGetIndent Completed

func (c JsonGetIndent) Newline(newline string) JsonGetNewline {
	c.cs.s = append(c.cs.s, "NEWLINE", newline)
	return (JsonGetNewline)(c)
}

func (c JsonGetIndent) Space(space string) JsonGetSpace {
	c.cs.s = append(c.cs.s, "SPACE", space)
	return (JsonGetSpace)(c)
}

func (c JsonGetIndent) Path(path ...string) JsonGetPath {
	c.cs.s = append(c.cs.s, path...)
	return (JsonGetPath)(c)
}

func (c JsonGetIndent) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

func (c JsonGetIndent) Cache() Cacheable {
	c.cs.Build()
	return Cacheable(c)
}

type JsonGetKey Completed

func (c JsonGetKey) Indent(indent string) JsonGetIndent {
	c.cs.s = append(c.cs.s, "INDENT", indent)
	return (JsonGetIndent)(c)
}

func (c JsonGetKey) Newline(newline string) JsonGetNewline {
	c.cs.s = append(c.cs.s, "NEWLINE", newline)
	return (JsonGetNewline)(c)
}

func (c JsonGetKey) Space(space string) JsonGetSpace {
	c.cs.s = append(c.cs.s, "SPACE", space)
	return (JsonGetSpace)(c)
}

func (c JsonGetKey) Path(path ...string) JsonGetPath {
	c.cs.s = append(c.cs.s, path...)
	return (JsonGetPath)(c)
}

func (c JsonGetKey) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

func (c JsonGetKey) Cache() Cacheable {
	c.cs.Build()
	return Cacheable(c)
}

type JsonGetNewline Completed

func (c JsonGetNewline) Space(space string) JsonGetSpace {
	c.cs.s = append(c.cs.s, "SPACE", space)
	return (JsonGetSpace)(c)
}

func (c JsonGetNewline) Path(path ...string) JsonGetPath {
	c.cs.s = append(c.cs.s, path...)
	return (JsonGetPath)(c)
}

func (c JsonGetNewline) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

func (c JsonGetNewline) Cache() Cacheable {
	c.cs.Build()
	return Cacheable(c)
}

type JsonGetPath Completed

func (c JsonGetPath) Path(path ...string) JsonGetPath {
	c.cs.s = append(c.cs.s, path...)
	return c
}

func (c JsonGetPath) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

func (c JsonGetPath) Cache() Cacheable {
	c.cs.Build()
	return Cacheable(c)
}

type JsonGetSpace Completed

func (c JsonGetSpace) Path(path ...string) JsonGetPath {
	c.cs.s = append(c.cs.s, path...)
	return (JsonGetPath)(c)
}

func (c JsonGetSpace) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

func (c JsonGetSpace) Cache() Cacheable {
	c.cs.Build()
	return Cacheable(c)
}

type JsonMerge Completed

func (b Builder) JsonMerge() (c JsonMerge) {
	c = JsonMerge{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "JSON.MERGE")
	return c
}

func (c JsonMerge) Key(key string) JsonMergeKey {
	if c.ks&NoSlot == NoSlot {
		c.ks = NoSlot | slot(key)
	} else {
		c.ks = check(c.ks, slot(key))
	}
	c.cs.s = append(c.cs.s, key)
	return (JsonMergeKey)(c)
}

type JsonMergeKey Completed

func (c JsonMergeKey) Path(path string) JsonMergePath {
	c.cs.s = append(c.cs.s, path)
	return (JsonMergePath)(c)
}

type JsonMergePath Completed

func (c JsonMergePath) Value(value string) JsonMergeValue {
	c.cs.s = append(c.cs.s, value)
	return (JsonMergeValue)(c)
}

type JsonMergeValue Completed

func (c JsonMergeValue) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type JsonMget Completed

func (b Builder) JsonMget() (c JsonMget) {
	c = JsonMget{cs: get(), ks: b.ks, cf: mtGetTag}
	c.cs.s = append(c.cs.s, "JSON.MGET")
	return c
}

func (c JsonMget) Key(key ...string) JsonMgetKey {
	if c.ks&NoSlot == NoSlot {
		for _, k := range key {
			c.ks = NoSlot | slot(k)
			break
		}
	} else {
		for _, k := range key {
			c.ks = check(c.ks, slot(k))
		}
	}
	c.cs.s = append(c.cs.s, key...)
	return (JsonMgetKey)(c)
}

type JsonMgetKey Completed

func (c JsonMgetKey) Key(key ...string) JsonMgetKey {
	if c.ks&NoSlot == NoSlot {
		for _, k := range key {
			c.ks = NoSlot | slot(k)
			break
		}
	} else {
		for _, k := range key {
			c.ks = check(c.ks, slot(k))
		}
	}
	c.cs.s = append(c.cs.s, key...)
	return c
}

func (c JsonMgetKey) Path(path string) JsonMgetPath {
	c.cs.s = append(c.cs.s, path)
	return (JsonMgetPath)(c)
}

type JsonMgetPath Completed

func (c JsonMgetPath) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

func (c JsonMgetPath) Cache() Cacheable {
	c.cs.Build()
	return Cacheable(c)
}

type JsonMset Completed

func (b Builder) JsonMset() (c JsonMset) {
	c = JsonMset{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "JSON.MSET")
	return c
}

func (c JsonMset) Key(key string) JsonMsetTripletKey {
	if c.ks&NoSlot == NoSlot {
		c.ks = NoSlot | slot(key)
	} else {
		c.ks = check(c.ks, slot(key))
	}
	c.cs.s = append(c.cs.s, key)
	return (JsonMsetTripletKey)(c)
}

type JsonMsetTripletKey Completed

func (c JsonMsetTripletKey) Path(path string) JsonMsetTripletPath {
	c.cs.s = append(c.cs.s, path)
	return (JsonMsetTripletPath)(c)
}

type JsonMsetTripletPath Completed

func (c JsonMsetTripletPath) Value(value string) JsonMsetTripletValue {
	c.cs.s = append(c.cs.s, value)
	return (JsonMsetTripletValue)(c)
}

type JsonMsetTripletValue Completed

func (c JsonMsetTripletValue) Key(key string) JsonMsetTripletKey {
	if c.ks&NoSlot == NoSlot {
		c.ks = NoSlot | slot(key)
	} else {
		c.ks = check(c.ks, slot(key))
	}
	c.cs.s = append(c.cs.s, key)
	return (JsonMsetTripletKey)(c)
}

func (c JsonMsetTripletValue) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type JsonNumincrby Completed

func (b Builder) JsonNumincrby() (c JsonNumincrby) {
	c = JsonNumincrby{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "JSON.NUMINCRBY")
	return c
}

func (c JsonNumincrby) Key(key string) JsonNumincrbyKey {
	if c.ks&NoSlot == NoSlot {
		c.ks = NoSlot | slot(key)
	} else {
		c.ks = check(c.ks, slot(key))
	}
	c.cs.s = append(c.cs.s, key)
	return (JsonNumincrbyKey)(c)
}

type JsonNumincrbyKey Completed

func (c JsonNumincrbyKey) Path(path string) JsonNumincrbyPath {
	c.cs.s = append(c.cs.s, path)
	return (JsonNumincrbyPath)(c)
}

type JsonNumincrbyPath Completed

func (c JsonNumincrbyPath) Value(value float64) JsonNumincrbyValue {
	c.cs.s = append(c.cs.s, strconv.FormatFloat(value, 'f', -1, 64))
	return (JsonNumincrbyValue)(c)
}

type JsonNumincrbyValue Completed

func (c JsonNumincrbyValue) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type JsonNummultby Completed

func (b Builder) JsonNummultby() (c JsonNummultby) {
	c = JsonNummultby{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "JSON.NUMMULTBY")
	return c
}

func (c JsonNummultby) Key(key string) JsonNummultbyKey {
	if c.ks&NoSlot == NoSlot {
		c.ks = NoSlot | slot(key)
	} else {
		c.ks = check(c.ks, slot(key))
	}
	c.cs.s = append(c.cs.s, key)
	return (JsonNummultbyKey)(c)
}

type JsonNummultbyKey Completed

func (c JsonNummultbyKey) Path(path string) JsonNummultbyPath {
	c.cs.s = append(c.cs.s, path)
	return (JsonNummultbyPath)(c)
}

type JsonNummultbyPath Completed

func (c JsonNummultbyPath) Value(value float64) JsonNummultbyValue {
	c.cs.s = append(c.cs.s, strconv.FormatFloat(value, 'f', -1, 64))
	return (JsonNummultbyValue)(c)
}

type JsonNummultbyValue Completed

func (c JsonNummultbyValue) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type JsonObjkeys Completed

func (b Builder) JsonObjkeys() (c JsonObjkeys) {
	c = JsonObjkeys{cs: get(), ks: b.ks, cf: readonly}
	c.cs.s = append(c.cs.s, "JSON.OBJKEYS")
	return c
}

func (c JsonObjkeys) Key(key string) JsonObjkeysKey {
	if c.ks&NoSlot == NoSlot {
		c.ks = NoSlot | slot(key)
	} else {
		c.ks = check(c.ks, slot(key))
	}
	c.cs.s = append(c.cs.s, key)
	return (JsonObjkeysKey)(c)
}

type JsonObjkeysKey Completed

func (c JsonObjkeysKey) Path(path string) JsonObjkeysPath {
	c.cs.s = append(c.cs.s, path)
	return (JsonObjkeysPath)(c)
}

func (c JsonObjkeysKey) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

func (c JsonObjkeysKey) Cache() Cacheable {
	c.cs.Build()
	return Cacheable(c)
}

type JsonObjkeysPath Completed

func (c JsonObjkeysPath) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

func (c JsonObjkeysPath) Cache() Cacheable {
	c.cs.Build()
	return Cacheable(c)
}

type JsonObjlen Completed

func (b Builder) JsonObjlen() (c JsonObjlen) {
	c = JsonObjlen{cs: get(), ks: b.ks, cf: readonly}
	c.cs.s = append(c.cs.s, "JSON.OBJLEN")
	return c
}

func (c JsonObjlen) Key(key string) JsonObjlenKey {
	if c.ks&NoSlot == NoSlot {
		c.ks = NoSlot | slot(key)
	} else {
		c.ks = check(c.ks, slot(key))
	}
	c.cs.s = append(c.cs.s, key)
	return (JsonObjlenKey)(c)
}

type JsonObjlenKey Completed

func (c JsonObjlenKey) Path(path string) JsonObjlenPath {
	c.cs.s = append(c.cs.s, path)
	return (JsonObjlenPath)(c)
}

func (c JsonObjlenKey) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

func (c JsonObjlenKey) Cache() Cacheable {
	c.cs.Build()
	return Cacheable(c)
}

type JsonObjlenPath Completed

func (c JsonObjlenPath) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

func (c JsonObjlenPath) Cache() Cacheable {
	c.cs.Build()
	return Cacheable(c)
}

type JsonResp Completed

func (b Builder) JsonResp() (c JsonResp) {
	c = JsonResp{cs: get(), ks: b.ks, cf: readonly}
	c.cs.s = append(c.cs.s, "JSON.RESP")
	return c
}

func (c JsonResp) Key(key string) JsonRespKey {
	if c.ks&NoSlot == NoSlot {
		c.ks = NoSlot | slot(key)
	} else {
		c.ks = check(c.ks, slot(key))
	}
	c.cs.s = append(c.cs.s, key)
	return (JsonRespKey)(c)
}

type JsonRespKey Completed

func (c JsonRespKey) Path(path string) JsonRespPath {
	c.cs.s = append(c.cs.s, path)
	return (JsonRespPath)(c)
}

func (c JsonRespKey) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

func (c JsonRespKey) Cache() Cacheable {
	c.cs.Build()
	return Cacheable(c)
}

type JsonRespPath Completed

func (c JsonRespPath) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

func (c JsonRespPath) Cache() Cacheable {
	c.cs.Build()
	return Cacheable(c)
}

type JsonSet Completed

func (b Builder) JsonSet() (c JsonSet) {
	c = JsonSet{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "JSON.SET")
	return c
}

func (c JsonSet) Key(key string) JsonSetKey {
	if c.ks&NoSlot == NoSlot {
		c.ks = NoSlot | slot(key)
	} else {
		c.ks = check(c.ks, slot(key))
	}
	c.cs.s = append(c.cs.s, key)
	return (JsonSetKey)(c)
}

type JsonSetConditionNx Completed

func (c JsonSetConditionNx) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type JsonSetConditionXx Completed

func (c JsonSetConditionXx) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type JsonSetKey Completed

func (c JsonSetKey) Path(path string) JsonSetPath {
	c.cs.s = append(c.cs.s, path)
	return (JsonSetPath)(c)
}

type JsonSetPath Completed

func (c JsonSetPath) Value(value string) JsonSetValue {
	c.cs.s = append(c.cs.s, value)
	return (JsonSetValue)(c)
}

type JsonSetValue Completed

func (c JsonSetValue) Nx() JsonSetConditionNx {
	c.cs.s = append(c.cs.s, "NX")
	return (JsonSetConditionNx)(c)
}

func (c JsonSetValue) Xx() JsonSetConditionXx {
	c.cs.s = append(c.cs.s, "XX")
	return (JsonSetConditionXx)(c)
}

func (c JsonSetValue) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type JsonStrappend Completed

func (b Builder) JsonStrappend() (c JsonStrappend) {
	c = JsonStrappend{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "JSON.STRAPPEND")
	return c
}

func (c JsonStrappend) Key(key string) JsonStrappendKey {
	if c.ks&NoSlot == NoSlot {
		c.ks = NoSlot | slot(key)
	} else {
		c.ks = check(c.ks, slot(key))
	}
	c.cs.s = append(c.cs.s, key)
	return (JsonStrappendKey)(c)
}

type JsonStrappendKey Completed

func (c JsonStrappendKey) Path(path string) JsonStrappendPath {
	c.cs.s = append(c.cs.s, path)
	return (JsonStrappendPath)(c)
}

func (c JsonStrappendKey) Value(value string) JsonStrappendValue {
	c.cs.s = append(c.cs.s, value)
	return (JsonStrappendValue)(c)
}

type JsonStrappendPath Completed

func (c JsonStrappendPath) Value(value string) JsonStrappendValue {
	c.cs.s = append(c.cs.s, value)
	return (JsonStrappendValue)(c)
}

type JsonStrappendValue Completed

func (c JsonStrappendValue) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type JsonStrlen Completed

func (b Builder) JsonStrlen() (c JsonStrlen) {
	c = JsonStrlen{cs: get(), ks: b.ks, cf: readonly}
	c.cs.s = append(c.cs.s, "JSON.STRLEN")
	return c
}

func (c JsonStrlen) Key(key string) JsonStrlenKey {
	if c.ks&NoSlot == NoSlot {
		c.ks = NoSlot | slot(key)
	} else {
		c.ks = check(c.ks, slot(key))
	}
	c.cs.s = append(c.cs.s, key)
	return (JsonStrlenKey)(c)
}

type JsonStrlenKey Completed

func (c JsonStrlenKey) Path(path string) JsonStrlenPath {
	c.cs.s = append(c.cs.s, path)
	return (JsonStrlenPath)(c)
}

func (c JsonStrlenKey) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

func (c JsonStrlenKey) Cache() Cacheable {
	c.cs.Build()
	return Cacheable(c)
}

type JsonStrlenPath Completed

func (c JsonStrlenPath) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

func (c JsonStrlenPath) Cache() Cacheable {
	c.cs.Build()
	return Cacheable(c)
}

type JsonToggle Completed

func (b Builder) JsonToggle() (c JsonToggle) {
	c = JsonToggle{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "JSON.TOGGLE")
	return c
}

func (c JsonToggle) Key(key string) JsonToggleKey {
	if c.ks&NoSlot == NoSlot {
		c.ks = NoSlot | slot(key)
	} else {
		c.ks = check(c.ks, slot(key))
	}
	c.cs.s = append(c.cs.s, key)
	return (JsonToggleKey)(c)
}

type JsonToggleKey Completed

func (c JsonToggleKey) Path(path string) JsonTogglePath {
	c.cs.s = append(c.cs.s, path)
	return (JsonTogglePath)(c)
}

type JsonTogglePath Completed

func (c JsonTogglePath) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type JsonType Completed

func (b Builder) JsonType() (c JsonType) {
	c = JsonType{cs: get(), ks: b.ks, cf: readonly}
	c.cs.s = append(c.cs.s, "JSON.TYPE")
	return c
}

func (c JsonType) Key(key string) JsonTypeKey {
	if c.ks&NoSlot == NoSlot {
		c.ks = NoSlot | slot(key)
	} else {
		c.ks = check(c.ks, slot(key))
	}
	c.cs.s = append(c.cs.s, key)
	return (JsonTypeKey)(c)
}

type JsonTypeKey Completed

func (c JsonTypeKey) Path(path string) JsonTypePath {
	c.cs.s = append(c.cs.s, path)
	return (JsonTypePath)(c)
}

func (c JsonTypeKey) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

func (c JsonTypeKey) Cache() Cacheable {
	c.cs.Build()
	return Cacheable(c)
}

type JsonTypePath Completed

func (c JsonTypePath) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

func (c JsonTypePath) Cache() Cacheable {
	c.cs.Build()
	return Cacheable(c)
}
