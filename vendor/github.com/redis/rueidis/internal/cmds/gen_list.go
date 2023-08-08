// Code generated DO NOT EDIT

package cmds

import "strconv"

type Blmove Completed

func (b Builder) Blmove() (c Blmove) {
	c = Blmove{cs: get(), ks: b.ks, cf: blockTag}
	c.cs.s = append(c.cs.s, "BLMOVE")
	return c
}

func (c Blmove) Source(source string) BlmoveSource {
	if c.ks&NoSlot == NoSlot {
		c.ks = NoSlot | slot(source)
	} else {
		c.ks = check(c.ks, slot(source))
	}
	c.cs.s = append(c.cs.s, source)
	return (BlmoveSource)(c)
}

type BlmoveDestination Completed

func (c BlmoveDestination) Left() BlmoveWherefromLeft {
	c.cs.s = append(c.cs.s, "LEFT")
	return (BlmoveWherefromLeft)(c)
}

func (c BlmoveDestination) Right() BlmoveWherefromRight {
	c.cs.s = append(c.cs.s, "RIGHT")
	return (BlmoveWherefromRight)(c)
}

type BlmoveSource Completed

func (c BlmoveSource) Destination(destination string) BlmoveDestination {
	if c.ks&NoSlot == NoSlot {
		c.ks = NoSlot | slot(destination)
	} else {
		c.ks = check(c.ks, slot(destination))
	}
	c.cs.s = append(c.cs.s, destination)
	return (BlmoveDestination)(c)
}

type BlmoveTimeout Completed

func (c BlmoveTimeout) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type BlmoveWherefromLeft Completed

func (c BlmoveWherefromLeft) Left() BlmoveWheretoLeft {
	c.cs.s = append(c.cs.s, "LEFT")
	return (BlmoveWheretoLeft)(c)
}

func (c BlmoveWherefromLeft) Right() BlmoveWheretoRight {
	c.cs.s = append(c.cs.s, "RIGHT")
	return (BlmoveWheretoRight)(c)
}

type BlmoveWherefromRight Completed

func (c BlmoveWherefromRight) Left() BlmoveWheretoLeft {
	c.cs.s = append(c.cs.s, "LEFT")
	return (BlmoveWheretoLeft)(c)
}

func (c BlmoveWherefromRight) Right() BlmoveWheretoRight {
	c.cs.s = append(c.cs.s, "RIGHT")
	return (BlmoveWheretoRight)(c)
}

type BlmoveWheretoLeft Completed

func (c BlmoveWheretoLeft) Timeout(timeout float64) BlmoveTimeout {
	c.cs.s = append(c.cs.s, strconv.FormatFloat(timeout, 'f', -1, 64))
	return (BlmoveTimeout)(c)
}

type BlmoveWheretoRight Completed

func (c BlmoveWheretoRight) Timeout(timeout float64) BlmoveTimeout {
	c.cs.s = append(c.cs.s, strconv.FormatFloat(timeout, 'f', -1, 64))
	return (BlmoveTimeout)(c)
}

type Blmpop Completed

func (b Builder) Blmpop() (c Blmpop) {
	c = Blmpop{cs: get(), ks: b.ks, cf: blockTag}
	c.cs.s = append(c.cs.s, "BLMPOP")
	return c
}

func (c Blmpop) Timeout(timeout float64) BlmpopTimeout {
	c.cs.s = append(c.cs.s, strconv.FormatFloat(timeout, 'f', -1, 64))
	return (BlmpopTimeout)(c)
}

type BlmpopCount Completed

func (c BlmpopCount) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type BlmpopKey Completed

func (c BlmpopKey) Key(key ...string) BlmpopKey {
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

func (c BlmpopKey) Left() BlmpopWhereLeft {
	c.cs.s = append(c.cs.s, "LEFT")
	return (BlmpopWhereLeft)(c)
}

func (c BlmpopKey) Right() BlmpopWhereRight {
	c.cs.s = append(c.cs.s, "RIGHT")
	return (BlmpopWhereRight)(c)
}

type BlmpopNumkeys Completed

func (c BlmpopNumkeys) Key(key ...string) BlmpopKey {
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
	return (BlmpopKey)(c)
}

type BlmpopTimeout Completed

func (c BlmpopTimeout) Numkeys(numkeys int64) BlmpopNumkeys {
	c.cs.s = append(c.cs.s, strconv.FormatInt(numkeys, 10))
	return (BlmpopNumkeys)(c)
}

type BlmpopWhereLeft Completed

func (c BlmpopWhereLeft) Count(count int64) BlmpopCount {
	c.cs.s = append(c.cs.s, "COUNT", strconv.FormatInt(count, 10))
	return (BlmpopCount)(c)
}

func (c BlmpopWhereLeft) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type BlmpopWhereRight Completed

func (c BlmpopWhereRight) Count(count int64) BlmpopCount {
	c.cs.s = append(c.cs.s, "COUNT", strconv.FormatInt(count, 10))
	return (BlmpopCount)(c)
}

func (c BlmpopWhereRight) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type Blpop Completed

func (b Builder) Blpop() (c Blpop) {
	c = Blpop{cs: get(), ks: b.ks, cf: blockTag}
	c.cs.s = append(c.cs.s, "BLPOP")
	return c
}

func (c Blpop) Key(key ...string) BlpopKey {
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
	return (BlpopKey)(c)
}

type BlpopKey Completed

func (c BlpopKey) Key(key ...string) BlpopKey {
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

func (c BlpopKey) Timeout(timeout float64) BlpopTimeout {
	c.cs.s = append(c.cs.s, strconv.FormatFloat(timeout, 'f', -1, 64))
	return (BlpopTimeout)(c)
}

type BlpopTimeout Completed

func (c BlpopTimeout) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type Brpop Completed

func (b Builder) Brpop() (c Brpop) {
	c = Brpop{cs: get(), ks: b.ks, cf: blockTag}
	c.cs.s = append(c.cs.s, "BRPOP")
	return c
}

func (c Brpop) Key(key ...string) BrpopKey {
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
	return (BrpopKey)(c)
}

type BrpopKey Completed

func (c BrpopKey) Key(key ...string) BrpopKey {
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

func (c BrpopKey) Timeout(timeout float64) BrpopTimeout {
	c.cs.s = append(c.cs.s, strconv.FormatFloat(timeout, 'f', -1, 64))
	return (BrpopTimeout)(c)
}

type BrpopTimeout Completed

func (c BrpopTimeout) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type Brpoplpush Completed

func (b Builder) Brpoplpush() (c Brpoplpush) {
	c = Brpoplpush{cs: get(), ks: b.ks, cf: blockTag}
	c.cs.s = append(c.cs.s, "BRPOPLPUSH")
	return c
}

func (c Brpoplpush) Source(source string) BrpoplpushSource {
	if c.ks&NoSlot == NoSlot {
		c.ks = NoSlot | slot(source)
	} else {
		c.ks = check(c.ks, slot(source))
	}
	c.cs.s = append(c.cs.s, source)
	return (BrpoplpushSource)(c)
}

type BrpoplpushDestination Completed

func (c BrpoplpushDestination) Timeout(timeout float64) BrpoplpushTimeout {
	c.cs.s = append(c.cs.s, strconv.FormatFloat(timeout, 'f', -1, 64))
	return (BrpoplpushTimeout)(c)
}

type BrpoplpushSource Completed

func (c BrpoplpushSource) Destination(destination string) BrpoplpushDestination {
	if c.ks&NoSlot == NoSlot {
		c.ks = NoSlot | slot(destination)
	} else {
		c.ks = check(c.ks, slot(destination))
	}
	c.cs.s = append(c.cs.s, destination)
	return (BrpoplpushDestination)(c)
}

type BrpoplpushTimeout Completed

func (c BrpoplpushTimeout) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type Lindex Completed

func (b Builder) Lindex() (c Lindex) {
	c = Lindex{cs: get(), ks: b.ks, cf: readonly}
	c.cs.s = append(c.cs.s, "LINDEX")
	return c
}

func (c Lindex) Key(key string) LindexKey {
	if c.ks&NoSlot == NoSlot {
		c.ks = NoSlot | slot(key)
	} else {
		c.ks = check(c.ks, slot(key))
	}
	c.cs.s = append(c.cs.s, key)
	return (LindexKey)(c)
}

type LindexIndex Completed

func (c LindexIndex) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

func (c LindexIndex) Cache() Cacheable {
	c.cs.Build()
	return Cacheable(c)
}

type LindexKey Completed

func (c LindexKey) Index(index int64) LindexIndex {
	c.cs.s = append(c.cs.s, strconv.FormatInt(index, 10))
	return (LindexIndex)(c)
}

type Linsert Completed

func (b Builder) Linsert() (c Linsert) {
	c = Linsert{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "LINSERT")
	return c
}

func (c Linsert) Key(key string) LinsertKey {
	if c.ks&NoSlot == NoSlot {
		c.ks = NoSlot | slot(key)
	} else {
		c.ks = check(c.ks, slot(key))
	}
	c.cs.s = append(c.cs.s, key)
	return (LinsertKey)(c)
}

type LinsertElement Completed

func (c LinsertElement) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type LinsertKey Completed

func (c LinsertKey) Before() LinsertWhereBefore {
	c.cs.s = append(c.cs.s, "BEFORE")
	return (LinsertWhereBefore)(c)
}

func (c LinsertKey) After() LinsertWhereAfter {
	c.cs.s = append(c.cs.s, "AFTER")
	return (LinsertWhereAfter)(c)
}

type LinsertPivot Completed

func (c LinsertPivot) Element(element string) LinsertElement {
	c.cs.s = append(c.cs.s, element)
	return (LinsertElement)(c)
}

type LinsertWhereAfter Completed

func (c LinsertWhereAfter) Pivot(pivot string) LinsertPivot {
	c.cs.s = append(c.cs.s, pivot)
	return (LinsertPivot)(c)
}

type LinsertWhereBefore Completed

func (c LinsertWhereBefore) Pivot(pivot string) LinsertPivot {
	c.cs.s = append(c.cs.s, pivot)
	return (LinsertPivot)(c)
}

type Llen Completed

func (b Builder) Llen() (c Llen) {
	c = Llen{cs: get(), ks: b.ks, cf: readonly}
	c.cs.s = append(c.cs.s, "LLEN")
	return c
}

func (c Llen) Key(key string) LlenKey {
	if c.ks&NoSlot == NoSlot {
		c.ks = NoSlot | slot(key)
	} else {
		c.ks = check(c.ks, slot(key))
	}
	c.cs.s = append(c.cs.s, key)
	return (LlenKey)(c)
}

type LlenKey Completed

func (c LlenKey) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

func (c LlenKey) Cache() Cacheable {
	c.cs.Build()
	return Cacheable(c)
}

type Lmove Completed

func (b Builder) Lmove() (c Lmove) {
	c = Lmove{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "LMOVE")
	return c
}

func (c Lmove) Source(source string) LmoveSource {
	if c.ks&NoSlot == NoSlot {
		c.ks = NoSlot | slot(source)
	} else {
		c.ks = check(c.ks, slot(source))
	}
	c.cs.s = append(c.cs.s, source)
	return (LmoveSource)(c)
}

type LmoveDestination Completed

func (c LmoveDestination) Left() LmoveWherefromLeft {
	c.cs.s = append(c.cs.s, "LEFT")
	return (LmoveWherefromLeft)(c)
}

func (c LmoveDestination) Right() LmoveWherefromRight {
	c.cs.s = append(c.cs.s, "RIGHT")
	return (LmoveWherefromRight)(c)
}

type LmoveSource Completed

func (c LmoveSource) Destination(destination string) LmoveDestination {
	if c.ks&NoSlot == NoSlot {
		c.ks = NoSlot | slot(destination)
	} else {
		c.ks = check(c.ks, slot(destination))
	}
	c.cs.s = append(c.cs.s, destination)
	return (LmoveDestination)(c)
}

type LmoveWherefromLeft Completed

func (c LmoveWherefromLeft) Left() LmoveWheretoLeft {
	c.cs.s = append(c.cs.s, "LEFT")
	return (LmoveWheretoLeft)(c)
}

func (c LmoveWherefromLeft) Right() LmoveWheretoRight {
	c.cs.s = append(c.cs.s, "RIGHT")
	return (LmoveWheretoRight)(c)
}

type LmoveWherefromRight Completed

func (c LmoveWherefromRight) Left() LmoveWheretoLeft {
	c.cs.s = append(c.cs.s, "LEFT")
	return (LmoveWheretoLeft)(c)
}

func (c LmoveWherefromRight) Right() LmoveWheretoRight {
	c.cs.s = append(c.cs.s, "RIGHT")
	return (LmoveWheretoRight)(c)
}

type LmoveWheretoLeft Completed

func (c LmoveWheretoLeft) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type LmoveWheretoRight Completed

func (c LmoveWheretoRight) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type Lmpop Completed

func (b Builder) Lmpop() (c Lmpop) {
	c = Lmpop{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "LMPOP")
	return c
}

func (c Lmpop) Numkeys(numkeys int64) LmpopNumkeys {
	c.cs.s = append(c.cs.s, strconv.FormatInt(numkeys, 10))
	return (LmpopNumkeys)(c)
}

type LmpopCount Completed

func (c LmpopCount) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type LmpopKey Completed

func (c LmpopKey) Key(key ...string) LmpopKey {
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

func (c LmpopKey) Left() LmpopWhereLeft {
	c.cs.s = append(c.cs.s, "LEFT")
	return (LmpopWhereLeft)(c)
}

func (c LmpopKey) Right() LmpopWhereRight {
	c.cs.s = append(c.cs.s, "RIGHT")
	return (LmpopWhereRight)(c)
}

type LmpopNumkeys Completed

func (c LmpopNumkeys) Key(key ...string) LmpopKey {
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
	return (LmpopKey)(c)
}

type LmpopWhereLeft Completed

func (c LmpopWhereLeft) Count(count int64) LmpopCount {
	c.cs.s = append(c.cs.s, "COUNT", strconv.FormatInt(count, 10))
	return (LmpopCount)(c)
}

func (c LmpopWhereLeft) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type LmpopWhereRight Completed

func (c LmpopWhereRight) Count(count int64) LmpopCount {
	c.cs.s = append(c.cs.s, "COUNT", strconv.FormatInt(count, 10))
	return (LmpopCount)(c)
}

func (c LmpopWhereRight) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type Lpop Completed

func (b Builder) Lpop() (c Lpop) {
	c = Lpop{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "LPOP")
	return c
}

func (c Lpop) Key(key string) LpopKey {
	if c.ks&NoSlot == NoSlot {
		c.ks = NoSlot | slot(key)
	} else {
		c.ks = check(c.ks, slot(key))
	}
	c.cs.s = append(c.cs.s, key)
	return (LpopKey)(c)
}

type LpopCount Completed

func (c LpopCount) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type LpopKey Completed

func (c LpopKey) Count(count int64) LpopCount {
	c.cs.s = append(c.cs.s, strconv.FormatInt(count, 10))
	return (LpopCount)(c)
}

func (c LpopKey) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type Lpos Completed

func (b Builder) Lpos() (c Lpos) {
	c = Lpos{cs: get(), ks: b.ks, cf: readonly}
	c.cs.s = append(c.cs.s, "LPOS")
	return c
}

func (c Lpos) Key(key string) LposKey {
	if c.ks&NoSlot == NoSlot {
		c.ks = NoSlot | slot(key)
	} else {
		c.ks = check(c.ks, slot(key))
	}
	c.cs.s = append(c.cs.s, key)
	return (LposKey)(c)
}

type LposCount Completed

func (c LposCount) Maxlen(len int64) LposMaxlen {
	c.cs.s = append(c.cs.s, "MAXLEN", strconv.FormatInt(len, 10))
	return (LposMaxlen)(c)
}

func (c LposCount) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

func (c LposCount) Cache() Cacheable {
	c.cs.Build()
	return Cacheable(c)
}

type LposElement Completed

func (c LposElement) Rank(rank int64) LposRank {
	c.cs.s = append(c.cs.s, "RANK", strconv.FormatInt(rank, 10))
	return (LposRank)(c)
}

func (c LposElement) Count(numMatches int64) LposCount {
	c.cs.s = append(c.cs.s, "COUNT", strconv.FormatInt(numMatches, 10))
	return (LposCount)(c)
}

func (c LposElement) Maxlen(len int64) LposMaxlen {
	c.cs.s = append(c.cs.s, "MAXLEN", strconv.FormatInt(len, 10))
	return (LposMaxlen)(c)
}

func (c LposElement) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

func (c LposElement) Cache() Cacheable {
	c.cs.Build()
	return Cacheable(c)
}

type LposKey Completed

func (c LposKey) Element(element string) LposElement {
	c.cs.s = append(c.cs.s, element)
	return (LposElement)(c)
}

type LposMaxlen Completed

func (c LposMaxlen) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

func (c LposMaxlen) Cache() Cacheable {
	c.cs.Build()
	return Cacheable(c)
}

type LposRank Completed

func (c LposRank) Count(numMatches int64) LposCount {
	c.cs.s = append(c.cs.s, "COUNT", strconv.FormatInt(numMatches, 10))
	return (LposCount)(c)
}

func (c LposRank) Maxlen(len int64) LposMaxlen {
	c.cs.s = append(c.cs.s, "MAXLEN", strconv.FormatInt(len, 10))
	return (LposMaxlen)(c)
}

func (c LposRank) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

func (c LposRank) Cache() Cacheable {
	c.cs.Build()
	return Cacheable(c)
}

type Lpush Completed

func (b Builder) Lpush() (c Lpush) {
	c = Lpush{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "LPUSH")
	return c
}

func (c Lpush) Key(key string) LpushKey {
	if c.ks&NoSlot == NoSlot {
		c.ks = NoSlot | slot(key)
	} else {
		c.ks = check(c.ks, slot(key))
	}
	c.cs.s = append(c.cs.s, key)
	return (LpushKey)(c)
}

type LpushElement Completed

func (c LpushElement) Element(element ...string) LpushElement {
	c.cs.s = append(c.cs.s, element...)
	return c
}

func (c LpushElement) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type LpushKey Completed

func (c LpushKey) Element(element ...string) LpushElement {
	c.cs.s = append(c.cs.s, element...)
	return (LpushElement)(c)
}

type Lpushx Completed

func (b Builder) Lpushx() (c Lpushx) {
	c = Lpushx{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "LPUSHX")
	return c
}

func (c Lpushx) Key(key string) LpushxKey {
	if c.ks&NoSlot == NoSlot {
		c.ks = NoSlot | slot(key)
	} else {
		c.ks = check(c.ks, slot(key))
	}
	c.cs.s = append(c.cs.s, key)
	return (LpushxKey)(c)
}

type LpushxElement Completed

func (c LpushxElement) Element(element ...string) LpushxElement {
	c.cs.s = append(c.cs.s, element...)
	return c
}

func (c LpushxElement) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type LpushxKey Completed

func (c LpushxKey) Element(element ...string) LpushxElement {
	c.cs.s = append(c.cs.s, element...)
	return (LpushxElement)(c)
}

type Lrange Completed

func (b Builder) Lrange() (c Lrange) {
	c = Lrange{cs: get(), ks: b.ks, cf: readonly}
	c.cs.s = append(c.cs.s, "LRANGE")
	return c
}

func (c Lrange) Key(key string) LrangeKey {
	if c.ks&NoSlot == NoSlot {
		c.ks = NoSlot | slot(key)
	} else {
		c.ks = check(c.ks, slot(key))
	}
	c.cs.s = append(c.cs.s, key)
	return (LrangeKey)(c)
}

type LrangeKey Completed

func (c LrangeKey) Start(start int64) LrangeStart {
	c.cs.s = append(c.cs.s, strconv.FormatInt(start, 10))
	return (LrangeStart)(c)
}

type LrangeStart Completed

func (c LrangeStart) Stop(stop int64) LrangeStop {
	c.cs.s = append(c.cs.s, strconv.FormatInt(stop, 10))
	return (LrangeStop)(c)
}

type LrangeStop Completed

func (c LrangeStop) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

func (c LrangeStop) Cache() Cacheable {
	c.cs.Build()
	return Cacheable(c)
}

type Lrem Completed

func (b Builder) Lrem() (c Lrem) {
	c = Lrem{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "LREM")
	return c
}

func (c Lrem) Key(key string) LremKey {
	if c.ks&NoSlot == NoSlot {
		c.ks = NoSlot | slot(key)
	} else {
		c.ks = check(c.ks, slot(key))
	}
	c.cs.s = append(c.cs.s, key)
	return (LremKey)(c)
}

type LremCount Completed

func (c LremCount) Element(element string) LremElement {
	c.cs.s = append(c.cs.s, element)
	return (LremElement)(c)
}

type LremElement Completed

func (c LremElement) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type LremKey Completed

func (c LremKey) Count(count int64) LremCount {
	c.cs.s = append(c.cs.s, strconv.FormatInt(count, 10))
	return (LremCount)(c)
}

type Lset Completed

func (b Builder) Lset() (c Lset) {
	c = Lset{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "LSET")
	return c
}

func (c Lset) Key(key string) LsetKey {
	if c.ks&NoSlot == NoSlot {
		c.ks = NoSlot | slot(key)
	} else {
		c.ks = check(c.ks, slot(key))
	}
	c.cs.s = append(c.cs.s, key)
	return (LsetKey)(c)
}

type LsetElement Completed

func (c LsetElement) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type LsetIndex Completed

func (c LsetIndex) Element(element string) LsetElement {
	c.cs.s = append(c.cs.s, element)
	return (LsetElement)(c)
}

type LsetKey Completed

func (c LsetKey) Index(index int64) LsetIndex {
	c.cs.s = append(c.cs.s, strconv.FormatInt(index, 10))
	return (LsetIndex)(c)
}

type Ltrim Completed

func (b Builder) Ltrim() (c Ltrim) {
	c = Ltrim{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "LTRIM")
	return c
}

func (c Ltrim) Key(key string) LtrimKey {
	if c.ks&NoSlot == NoSlot {
		c.ks = NoSlot | slot(key)
	} else {
		c.ks = check(c.ks, slot(key))
	}
	c.cs.s = append(c.cs.s, key)
	return (LtrimKey)(c)
}

type LtrimKey Completed

func (c LtrimKey) Start(start int64) LtrimStart {
	c.cs.s = append(c.cs.s, strconv.FormatInt(start, 10))
	return (LtrimStart)(c)
}

type LtrimStart Completed

func (c LtrimStart) Stop(stop int64) LtrimStop {
	c.cs.s = append(c.cs.s, strconv.FormatInt(stop, 10))
	return (LtrimStop)(c)
}

type LtrimStop Completed

func (c LtrimStop) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type Rpop Completed

func (b Builder) Rpop() (c Rpop) {
	c = Rpop{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "RPOP")
	return c
}

func (c Rpop) Key(key string) RpopKey {
	if c.ks&NoSlot == NoSlot {
		c.ks = NoSlot | slot(key)
	} else {
		c.ks = check(c.ks, slot(key))
	}
	c.cs.s = append(c.cs.s, key)
	return (RpopKey)(c)
}

type RpopCount Completed

func (c RpopCount) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type RpopKey Completed

func (c RpopKey) Count(count int64) RpopCount {
	c.cs.s = append(c.cs.s, strconv.FormatInt(count, 10))
	return (RpopCount)(c)
}

func (c RpopKey) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type Rpoplpush Completed

func (b Builder) Rpoplpush() (c Rpoplpush) {
	c = Rpoplpush{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "RPOPLPUSH")
	return c
}

func (c Rpoplpush) Source(source string) RpoplpushSource {
	if c.ks&NoSlot == NoSlot {
		c.ks = NoSlot | slot(source)
	} else {
		c.ks = check(c.ks, slot(source))
	}
	c.cs.s = append(c.cs.s, source)
	return (RpoplpushSource)(c)
}

type RpoplpushDestination Completed

func (c RpoplpushDestination) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type RpoplpushSource Completed

func (c RpoplpushSource) Destination(destination string) RpoplpushDestination {
	if c.ks&NoSlot == NoSlot {
		c.ks = NoSlot | slot(destination)
	} else {
		c.ks = check(c.ks, slot(destination))
	}
	c.cs.s = append(c.cs.s, destination)
	return (RpoplpushDestination)(c)
}

type Rpush Completed

func (b Builder) Rpush() (c Rpush) {
	c = Rpush{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "RPUSH")
	return c
}

func (c Rpush) Key(key string) RpushKey {
	if c.ks&NoSlot == NoSlot {
		c.ks = NoSlot | slot(key)
	} else {
		c.ks = check(c.ks, slot(key))
	}
	c.cs.s = append(c.cs.s, key)
	return (RpushKey)(c)
}

type RpushElement Completed

func (c RpushElement) Element(element ...string) RpushElement {
	c.cs.s = append(c.cs.s, element...)
	return c
}

func (c RpushElement) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type RpushKey Completed

func (c RpushKey) Element(element ...string) RpushElement {
	c.cs.s = append(c.cs.s, element...)
	return (RpushElement)(c)
}

type Rpushx Completed

func (b Builder) Rpushx() (c Rpushx) {
	c = Rpushx{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "RPUSHX")
	return c
}

func (c Rpushx) Key(key string) RpushxKey {
	if c.ks&NoSlot == NoSlot {
		c.ks = NoSlot | slot(key)
	} else {
		c.ks = check(c.ks, slot(key))
	}
	c.cs.s = append(c.cs.s, key)
	return (RpushxKey)(c)
}

type RpushxElement Completed

func (c RpushxElement) Element(element ...string) RpushxElement {
	c.cs.s = append(c.cs.s, element...)
	return c
}

func (c RpushxElement) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type RpushxKey Completed

func (c RpushxKey) Element(element ...string) RpushxElement {
	c.cs.s = append(c.cs.s, element...)
	return (RpushxElement)(c)
}
