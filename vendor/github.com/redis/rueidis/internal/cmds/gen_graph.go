// Code generated DO NOT EDIT

package cmds

import "strconv"

type GraphConfigGet Completed

func (b Builder) GraphConfigGet() (c GraphConfigGet) {
	c = GraphConfigGet{cs: get(), ks: b.ks, cf: readonly}
	c.cs.s = append(c.cs.s, "GRAPH.CONFIG", "GET")
	return c
}

func (c GraphConfigGet) Name(name string) GraphConfigGetName {
	c.cs.s = append(c.cs.s, name)
	return (GraphConfigGetName)(c)
}

type GraphConfigGetName Completed

func (c GraphConfigGetName) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type GraphConfigSet Completed

func (b Builder) GraphConfigSet() (c GraphConfigSet) {
	c = GraphConfigSet{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "GRAPH.CONFIG", "SET")
	return c
}

func (c GraphConfigSet) Name(name string) GraphConfigSetName {
	c.cs.s = append(c.cs.s, name)
	return (GraphConfigSetName)(c)
}

type GraphConfigSetName Completed

func (c GraphConfigSetName) Value(value string) GraphConfigSetValue {
	c.cs.s = append(c.cs.s, value)
	return (GraphConfigSetValue)(c)
}

type GraphConfigSetValue Completed

func (c GraphConfigSetValue) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type GraphConstraintCreate Completed

func (b Builder) GraphConstraintCreate() (c GraphConstraintCreate) {
	c = GraphConstraintCreate{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "GRAPH.CONSTRAINT", "CREATE")
	return c
}

func (c GraphConstraintCreate) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type GraphConstraintDrop Completed

func (b Builder) GraphConstraintDrop() (c GraphConstraintDrop) {
	c = GraphConstraintDrop{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "GRAPH.CONSTRAINT", "DROP")
	return c
}

func (c GraphConstraintDrop) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type GraphDelete Completed

func (b Builder) GraphDelete() (c GraphDelete) {
	c = GraphDelete{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "GRAPH.DELETE")
	return c
}

func (c GraphDelete) Graph(graph string) GraphDeleteGraph {
	if c.ks&NoSlot == NoSlot {
		c.ks = NoSlot | slot(graph)
	} else {
		c.ks = check(c.ks, slot(graph))
	}
	c.cs.s = append(c.cs.s, graph)
	return (GraphDeleteGraph)(c)
}

type GraphDeleteGraph Completed

func (c GraphDeleteGraph) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type GraphExplain Completed

func (b Builder) GraphExplain() (c GraphExplain) {
	c = GraphExplain{cs: get(), ks: b.ks, cf: readonly}
	c.cs.s = append(c.cs.s, "GRAPH.EXPLAIN")
	return c
}

func (c GraphExplain) Graph(graph string) GraphExplainGraph {
	if c.ks&NoSlot == NoSlot {
		c.ks = NoSlot | slot(graph)
	} else {
		c.ks = check(c.ks, slot(graph))
	}
	c.cs.s = append(c.cs.s, graph)
	return (GraphExplainGraph)(c)
}

type GraphExplainGraph Completed

func (c GraphExplainGraph) Query(query string) GraphExplainQuery {
	c.cs.s = append(c.cs.s, query)
	return (GraphExplainQuery)(c)
}

type GraphExplainQuery Completed

func (c GraphExplainQuery) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type GraphList Completed

func (b Builder) GraphList() (c GraphList) {
	c = GraphList{cs: get(), ks: b.ks, cf: readonly}
	c.cs.s = append(c.cs.s, "GRAPH.LIST")
	return c
}

func (c GraphList) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type GraphProfile Completed

func (b Builder) GraphProfile() (c GraphProfile) {
	c = GraphProfile{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "GRAPH.PROFILE")
	return c
}

func (c GraphProfile) Graph(graph string) GraphProfileGraph {
	if c.ks&NoSlot == NoSlot {
		c.ks = NoSlot | slot(graph)
	} else {
		c.ks = check(c.ks, slot(graph))
	}
	c.cs.s = append(c.cs.s, graph)
	return (GraphProfileGraph)(c)
}

type GraphProfileGraph Completed

func (c GraphProfileGraph) Query(query string) GraphProfileQuery {
	c.cs.s = append(c.cs.s, query)
	return (GraphProfileQuery)(c)
}

type GraphProfileQuery Completed

func (c GraphProfileQuery) Timeout(timeout int64) GraphProfileTimeout {
	c.cs.s = append(c.cs.s, "TIMEOUT", strconv.FormatInt(timeout, 10))
	return (GraphProfileTimeout)(c)
}

func (c GraphProfileQuery) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type GraphProfileTimeout Completed

func (c GraphProfileTimeout) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type GraphQuery Completed

func (b Builder) GraphQuery() (c GraphQuery) {
	c = GraphQuery{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "GRAPH.QUERY")
	return c
}

func (c GraphQuery) Graph(graph string) GraphQueryGraph {
	if c.ks&NoSlot == NoSlot {
		c.ks = NoSlot | slot(graph)
	} else {
		c.ks = check(c.ks, slot(graph))
	}
	c.cs.s = append(c.cs.s, graph)
	return (GraphQueryGraph)(c)
}

type GraphQueryGraph Completed

func (c GraphQueryGraph) Query(query string) GraphQueryQuery {
	c.cs.s = append(c.cs.s, query)
	return (GraphQueryQuery)(c)
}

type GraphQueryQuery Completed

func (c GraphQueryQuery) Timeout(timeout int64) GraphQueryTimeout {
	c.cs.s = append(c.cs.s, "TIMEOUT", strconv.FormatInt(timeout, 10))
	return (GraphQueryTimeout)(c)
}

func (c GraphQueryQuery) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type GraphQueryTimeout Completed

func (c GraphQueryTimeout) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type GraphRoQuery Completed

func (b Builder) GraphRoQuery() (c GraphRoQuery) {
	c = GraphRoQuery{cs: get(), ks: b.ks, cf: readonly}
	c.cs.s = append(c.cs.s, "GRAPH.RO_QUERY")
	return c
}

func (c GraphRoQuery) Graph(graph string) GraphRoQueryGraph {
	if c.ks&NoSlot == NoSlot {
		c.ks = NoSlot | slot(graph)
	} else {
		c.ks = check(c.ks, slot(graph))
	}
	c.cs.s = append(c.cs.s, graph)
	return (GraphRoQueryGraph)(c)
}

type GraphRoQueryGraph Completed

func (c GraphRoQueryGraph) Query(query string) GraphRoQueryQuery {
	c.cs.s = append(c.cs.s, query)
	return (GraphRoQueryQuery)(c)
}

type GraphRoQueryQuery Completed

func (c GraphRoQueryQuery) Timeout(timeout int64) GraphRoQueryTimeout {
	c.cs.s = append(c.cs.s, "TIMEOUT", strconv.FormatInt(timeout, 10))
	return (GraphRoQueryTimeout)(c)
}

func (c GraphRoQueryQuery) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

func (c GraphRoQueryQuery) Cache() Cacheable {
	c.cs.Build()
	return Cacheable(c)
}

type GraphRoQueryTimeout Completed

func (c GraphRoQueryTimeout) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

func (c GraphRoQueryTimeout) Cache() Cacheable {
	c.cs.Build()
	return Cacheable(c)
}

type GraphSlowlog Completed

func (b Builder) GraphSlowlog() (c GraphSlowlog) {
	c = GraphSlowlog{cs: get(), ks: b.ks, cf: readonly}
	c.cs.s = append(c.cs.s, "GRAPH.SLOWLOG")
	return c
}

func (c GraphSlowlog) Graph(graph string) GraphSlowlogGraph {
	if c.ks&NoSlot == NoSlot {
		c.ks = NoSlot | slot(graph)
	} else {
		c.ks = check(c.ks, slot(graph))
	}
	c.cs.s = append(c.cs.s, graph)
	return (GraphSlowlogGraph)(c)
}

type GraphSlowlogGraph Completed

func (c GraphSlowlogGraph) Build() Completed {
	c.cs.Build()
	return Completed(c)
}
