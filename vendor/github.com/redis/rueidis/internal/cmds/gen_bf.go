// Code generated DO NOT EDIT

package cmds

import "strconv"

type BfAdd Completed

func (b Builder) BfAdd() (c BfAdd) {
	c = BfAdd{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "BF.ADD")
	return c
}

func (c BfAdd) Key(key string) BfAddKey {
	if c.ks&NoSlot == NoSlot {
		c.ks = NoSlot | slot(key)
	} else {
		c.ks = check(c.ks, slot(key))
	}
	c.cs.s = append(c.cs.s, key)
	return (BfAddKey)(c)
}

type BfAddItem Completed

func (c BfAddItem) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type BfAddKey Completed

func (c BfAddKey) Item(item string) BfAddItem {
	c.cs.s = append(c.cs.s, item)
	return (BfAddItem)(c)
}

type BfCard Completed

func (b Builder) BfCard() (c BfCard) {
	c = BfCard{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "BF.CARD")
	return c
}

func (c BfCard) Key(key string) BfCardKey {
	if c.ks&NoSlot == NoSlot {
		c.ks = NoSlot | slot(key)
	} else {
		c.ks = check(c.ks, slot(key))
	}
	c.cs.s = append(c.cs.s, key)
	return (BfCardKey)(c)
}

type BfCardKey Completed

func (c BfCardKey) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type BfExists Completed

func (b Builder) BfExists() (c BfExists) {
	c = BfExists{cs: get(), ks: b.ks, cf: readonly}
	c.cs.s = append(c.cs.s, "BF.EXISTS")
	return c
}

func (c BfExists) Key(key string) BfExistsKey {
	if c.ks&NoSlot == NoSlot {
		c.ks = NoSlot | slot(key)
	} else {
		c.ks = check(c.ks, slot(key))
	}
	c.cs.s = append(c.cs.s, key)
	return (BfExistsKey)(c)
}

type BfExistsItem Completed

func (c BfExistsItem) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

func (c BfExistsItem) Cache() Cacheable {
	c.cs.Build()
	return Cacheable(c)
}

type BfExistsKey Completed

func (c BfExistsKey) Item(item string) BfExistsItem {
	c.cs.s = append(c.cs.s, item)
	return (BfExistsItem)(c)
}

type BfInfo Completed

func (b Builder) BfInfo() (c BfInfo) {
	c = BfInfo{cs: get(), ks: b.ks, cf: readonly}
	c.cs.s = append(c.cs.s, "BF.INFO")
	return c
}

func (c BfInfo) Key(key string) BfInfoKey {
	if c.ks&NoSlot == NoSlot {
		c.ks = NoSlot | slot(key)
	} else {
		c.ks = check(c.ks, slot(key))
	}
	c.cs.s = append(c.cs.s, key)
	return (BfInfoKey)(c)
}

type BfInfoKey Completed

func (c BfInfoKey) Capacity() BfInfoSingleValueCapacity {
	c.cs.s = append(c.cs.s, "CAPACITY")
	return (BfInfoSingleValueCapacity)(c)
}

func (c BfInfoKey) Size() BfInfoSingleValueSize {
	c.cs.s = append(c.cs.s, "SIZE")
	return (BfInfoSingleValueSize)(c)
}

func (c BfInfoKey) Filters() BfInfoSingleValueFilters {
	c.cs.s = append(c.cs.s, "FILTERS")
	return (BfInfoSingleValueFilters)(c)
}

func (c BfInfoKey) Items() BfInfoSingleValueItems {
	c.cs.s = append(c.cs.s, "ITEMS")
	return (BfInfoSingleValueItems)(c)
}

func (c BfInfoKey) Expansion() BfInfoSingleValueExpansion {
	c.cs.s = append(c.cs.s, "EXPANSION")
	return (BfInfoSingleValueExpansion)(c)
}

func (c BfInfoKey) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

func (c BfInfoKey) Cache() Cacheable {
	c.cs.Build()
	return Cacheable(c)
}

type BfInfoSingleValueCapacity Completed

func (c BfInfoSingleValueCapacity) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

func (c BfInfoSingleValueCapacity) Cache() Cacheable {
	c.cs.Build()
	return Cacheable(c)
}

type BfInfoSingleValueExpansion Completed

func (c BfInfoSingleValueExpansion) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

func (c BfInfoSingleValueExpansion) Cache() Cacheable {
	c.cs.Build()
	return Cacheable(c)
}

type BfInfoSingleValueFilters Completed

func (c BfInfoSingleValueFilters) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

func (c BfInfoSingleValueFilters) Cache() Cacheable {
	c.cs.Build()
	return Cacheable(c)
}

type BfInfoSingleValueItems Completed

func (c BfInfoSingleValueItems) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

func (c BfInfoSingleValueItems) Cache() Cacheable {
	c.cs.Build()
	return Cacheable(c)
}

type BfInfoSingleValueSize Completed

func (c BfInfoSingleValueSize) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

func (c BfInfoSingleValueSize) Cache() Cacheable {
	c.cs.Build()
	return Cacheable(c)
}

type BfInsert Completed

func (b Builder) BfInsert() (c BfInsert) {
	c = BfInsert{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "BF.INSERT")
	return c
}

func (c BfInsert) Key(key string) BfInsertKey {
	if c.ks&NoSlot == NoSlot {
		c.ks = NoSlot | slot(key)
	} else {
		c.ks = check(c.ks, slot(key))
	}
	c.cs.s = append(c.cs.s, key)
	return (BfInsertKey)(c)
}

type BfInsertCapacity Completed

func (c BfInsertCapacity) Error(error float64) BfInsertError {
	c.cs.s = append(c.cs.s, "ERROR", strconv.FormatFloat(error, 'f', -1, 64))
	return (BfInsertError)(c)
}

func (c BfInsertCapacity) Expansion(expansion int64) BfInsertExpansion {
	c.cs.s = append(c.cs.s, "EXPANSION", strconv.FormatInt(expansion, 10))
	return (BfInsertExpansion)(c)
}

func (c BfInsertCapacity) Nocreate() BfInsertNocreate {
	c.cs.s = append(c.cs.s, "NOCREATE")
	return (BfInsertNocreate)(c)
}

func (c BfInsertCapacity) Nonscaling() BfInsertNonscaling {
	c.cs.s = append(c.cs.s, "NONSCALING")
	return (BfInsertNonscaling)(c)
}

func (c BfInsertCapacity) Items() BfInsertItems {
	c.cs.s = append(c.cs.s, "ITEMS")
	return (BfInsertItems)(c)
}

type BfInsertError Completed

func (c BfInsertError) Expansion(expansion int64) BfInsertExpansion {
	c.cs.s = append(c.cs.s, "EXPANSION", strconv.FormatInt(expansion, 10))
	return (BfInsertExpansion)(c)
}

func (c BfInsertError) Nocreate() BfInsertNocreate {
	c.cs.s = append(c.cs.s, "NOCREATE")
	return (BfInsertNocreate)(c)
}

func (c BfInsertError) Nonscaling() BfInsertNonscaling {
	c.cs.s = append(c.cs.s, "NONSCALING")
	return (BfInsertNonscaling)(c)
}

func (c BfInsertError) Items() BfInsertItems {
	c.cs.s = append(c.cs.s, "ITEMS")
	return (BfInsertItems)(c)
}

type BfInsertExpansion Completed

func (c BfInsertExpansion) Nocreate() BfInsertNocreate {
	c.cs.s = append(c.cs.s, "NOCREATE")
	return (BfInsertNocreate)(c)
}

func (c BfInsertExpansion) Nonscaling() BfInsertNonscaling {
	c.cs.s = append(c.cs.s, "NONSCALING")
	return (BfInsertNonscaling)(c)
}

func (c BfInsertExpansion) Items() BfInsertItems {
	c.cs.s = append(c.cs.s, "ITEMS")
	return (BfInsertItems)(c)
}

type BfInsertItem Completed

func (c BfInsertItem) Item(item ...string) BfInsertItem {
	c.cs.s = append(c.cs.s, item...)
	return c
}

func (c BfInsertItem) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type BfInsertItems Completed

func (c BfInsertItems) Item(item ...string) BfInsertItem {
	c.cs.s = append(c.cs.s, item...)
	return (BfInsertItem)(c)
}

type BfInsertKey Completed

func (c BfInsertKey) Capacity(capacity int64) BfInsertCapacity {
	c.cs.s = append(c.cs.s, "CAPACITY", strconv.FormatInt(capacity, 10))
	return (BfInsertCapacity)(c)
}

func (c BfInsertKey) Error(error float64) BfInsertError {
	c.cs.s = append(c.cs.s, "ERROR", strconv.FormatFloat(error, 'f', -1, 64))
	return (BfInsertError)(c)
}

func (c BfInsertKey) Expansion(expansion int64) BfInsertExpansion {
	c.cs.s = append(c.cs.s, "EXPANSION", strconv.FormatInt(expansion, 10))
	return (BfInsertExpansion)(c)
}

func (c BfInsertKey) Nocreate() BfInsertNocreate {
	c.cs.s = append(c.cs.s, "NOCREATE")
	return (BfInsertNocreate)(c)
}

func (c BfInsertKey) Nonscaling() BfInsertNonscaling {
	c.cs.s = append(c.cs.s, "NONSCALING")
	return (BfInsertNonscaling)(c)
}

func (c BfInsertKey) Items() BfInsertItems {
	c.cs.s = append(c.cs.s, "ITEMS")
	return (BfInsertItems)(c)
}

type BfInsertNocreate Completed

func (c BfInsertNocreate) Nonscaling() BfInsertNonscaling {
	c.cs.s = append(c.cs.s, "NONSCALING")
	return (BfInsertNonscaling)(c)
}

func (c BfInsertNocreate) Items() BfInsertItems {
	c.cs.s = append(c.cs.s, "ITEMS")
	return (BfInsertItems)(c)
}

type BfInsertNonscaling Completed

func (c BfInsertNonscaling) Items() BfInsertItems {
	c.cs.s = append(c.cs.s, "ITEMS")
	return (BfInsertItems)(c)
}

type BfLoadchunk Completed

func (b Builder) BfLoadchunk() (c BfLoadchunk) {
	c = BfLoadchunk{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "BF.LOADCHUNK")
	return c
}

func (c BfLoadchunk) Key(key string) BfLoadchunkKey {
	if c.ks&NoSlot == NoSlot {
		c.ks = NoSlot | slot(key)
	} else {
		c.ks = check(c.ks, slot(key))
	}
	c.cs.s = append(c.cs.s, key)
	return (BfLoadchunkKey)(c)
}

type BfLoadchunkData Completed

func (c BfLoadchunkData) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type BfLoadchunkIterator Completed

func (c BfLoadchunkIterator) Data(data string) BfLoadchunkData {
	c.cs.s = append(c.cs.s, data)
	return (BfLoadchunkData)(c)
}

type BfLoadchunkKey Completed

func (c BfLoadchunkKey) Iterator(iterator int64) BfLoadchunkIterator {
	c.cs.s = append(c.cs.s, strconv.FormatInt(iterator, 10))
	return (BfLoadchunkIterator)(c)
}

type BfMadd Completed

func (b Builder) BfMadd() (c BfMadd) {
	c = BfMadd{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "BF.MADD")
	return c
}

func (c BfMadd) Key(key string) BfMaddKey {
	if c.ks&NoSlot == NoSlot {
		c.ks = NoSlot | slot(key)
	} else {
		c.ks = check(c.ks, slot(key))
	}
	c.cs.s = append(c.cs.s, key)
	return (BfMaddKey)(c)
}

type BfMaddItem Completed

func (c BfMaddItem) Item(item ...string) BfMaddItem {
	c.cs.s = append(c.cs.s, item...)
	return c
}

func (c BfMaddItem) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type BfMaddKey Completed

func (c BfMaddKey) Item(item ...string) BfMaddItem {
	c.cs.s = append(c.cs.s, item...)
	return (BfMaddItem)(c)
}

type BfMexists Completed

func (b Builder) BfMexists() (c BfMexists) {
	c = BfMexists{cs: get(), ks: b.ks, cf: readonly}
	c.cs.s = append(c.cs.s, "BF.MEXISTS")
	return c
}

func (c BfMexists) Key(key string) BfMexistsKey {
	if c.ks&NoSlot == NoSlot {
		c.ks = NoSlot | slot(key)
	} else {
		c.ks = check(c.ks, slot(key))
	}
	c.cs.s = append(c.cs.s, key)
	return (BfMexistsKey)(c)
}

type BfMexistsItem Completed

func (c BfMexistsItem) Item(item ...string) BfMexistsItem {
	c.cs.s = append(c.cs.s, item...)
	return c
}

func (c BfMexistsItem) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type BfMexistsKey Completed

func (c BfMexistsKey) Item(item ...string) BfMexistsItem {
	c.cs.s = append(c.cs.s, item...)
	return (BfMexistsItem)(c)
}

type BfReserve Completed

func (b Builder) BfReserve() (c BfReserve) {
	c = BfReserve{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "BF.RESERVE")
	return c
}

func (c BfReserve) Key(key string) BfReserveKey {
	if c.ks&NoSlot == NoSlot {
		c.ks = NoSlot | slot(key)
	} else {
		c.ks = check(c.ks, slot(key))
	}
	c.cs.s = append(c.cs.s, key)
	return (BfReserveKey)(c)
}

type BfReserveCapacity Completed

func (c BfReserveCapacity) Expansion(expansion int64) BfReserveExpansion {
	c.cs.s = append(c.cs.s, "EXPANSION", strconv.FormatInt(expansion, 10))
	return (BfReserveExpansion)(c)
}

func (c BfReserveCapacity) Nonscaling() BfReserveNonscaling {
	c.cs.s = append(c.cs.s, "NONSCALING")
	return (BfReserveNonscaling)(c)
}

func (c BfReserveCapacity) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type BfReserveErrorRate Completed

func (c BfReserveErrorRate) Capacity(capacity int64) BfReserveCapacity {
	c.cs.s = append(c.cs.s, strconv.FormatInt(capacity, 10))
	return (BfReserveCapacity)(c)
}

type BfReserveExpansion Completed

func (c BfReserveExpansion) Nonscaling() BfReserveNonscaling {
	c.cs.s = append(c.cs.s, "NONSCALING")
	return (BfReserveNonscaling)(c)
}

func (c BfReserveExpansion) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type BfReserveKey Completed

func (c BfReserveKey) ErrorRate(errorRate float64) BfReserveErrorRate {
	c.cs.s = append(c.cs.s, strconv.FormatFloat(errorRate, 'f', -1, 64))
	return (BfReserveErrorRate)(c)
}

type BfReserveNonscaling Completed

func (c BfReserveNonscaling) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type BfScandump Completed

func (b Builder) BfScandump() (c BfScandump) {
	c = BfScandump{cs: get(), ks: b.ks, cf: readonly}
	c.cs.s = append(c.cs.s, "BF.SCANDUMP")
	return c
}

func (c BfScandump) Key(key string) BfScandumpKey {
	if c.ks&NoSlot == NoSlot {
		c.ks = NoSlot | slot(key)
	} else {
		c.ks = check(c.ks, slot(key))
	}
	c.cs.s = append(c.cs.s, key)
	return (BfScandumpKey)(c)
}

type BfScandumpIterator Completed

func (c BfScandumpIterator) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type BfScandumpKey Completed

func (c BfScandumpKey) Iterator(iterator int64) BfScandumpIterator {
	c.cs.s = append(c.cs.s, strconv.FormatInt(iterator, 10))
	return (BfScandumpIterator)(c)
}
