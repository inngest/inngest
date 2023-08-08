// Code generated DO NOT EDIT

package cmds

import "strconv"

type TsAdd Completed

func (b Builder) TsAdd() (c TsAdd) {
	c = TsAdd{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "TS.ADD")
	return c
}

func (c TsAdd) Key(key string) TsAddKey {
	if c.ks&NoSlot == NoSlot {
		c.ks = NoSlot | slot(key)
	} else {
		c.ks = check(c.ks, slot(key))
	}
	c.cs.s = append(c.cs.s, key)
	return (TsAddKey)(c)
}

type TsAddChunkSize Completed

func (c TsAddChunkSize) OnDuplicateBlock() TsAddOnDuplicateBlock {
	c.cf = blockTag
	c.cs.s = append(c.cs.s, "ON_DUPLICATE", "BLOCK")
	return (TsAddOnDuplicateBlock)(c)
}

func (c TsAddChunkSize) OnDuplicateFirst() TsAddOnDuplicateFirst {
	c.cs.s = append(c.cs.s, "ON_DUPLICATE", "FIRST")
	return (TsAddOnDuplicateFirst)(c)
}

func (c TsAddChunkSize) OnDuplicateLast() TsAddOnDuplicateLast {
	c.cs.s = append(c.cs.s, "ON_DUPLICATE", "LAST")
	return (TsAddOnDuplicateLast)(c)
}

func (c TsAddChunkSize) OnDuplicateMin() TsAddOnDuplicateMin {
	c.cs.s = append(c.cs.s, "ON_DUPLICATE", "MIN")
	return (TsAddOnDuplicateMin)(c)
}

func (c TsAddChunkSize) OnDuplicateMax() TsAddOnDuplicateMax {
	c.cs.s = append(c.cs.s, "ON_DUPLICATE", "MAX")
	return (TsAddOnDuplicateMax)(c)
}

func (c TsAddChunkSize) OnDuplicateSum() TsAddOnDuplicateSum {
	c.cs.s = append(c.cs.s, "ON_DUPLICATE", "SUM")
	return (TsAddOnDuplicateSum)(c)
}

func (c TsAddChunkSize) Labels() TsAddLabels {
	c.cs.s = append(c.cs.s, "LABELS")
	return (TsAddLabels)(c)
}

func (c TsAddChunkSize) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type TsAddEncodingCompressed Completed

func (c TsAddEncodingCompressed) ChunkSize(size int64) TsAddChunkSize {
	c.cs.s = append(c.cs.s, "CHUNK_SIZE", strconv.FormatInt(size, 10))
	return (TsAddChunkSize)(c)
}

func (c TsAddEncodingCompressed) OnDuplicateBlock() TsAddOnDuplicateBlock {
	c.cf = blockTag
	c.cs.s = append(c.cs.s, "ON_DUPLICATE", "BLOCK")
	return (TsAddOnDuplicateBlock)(c)
}

func (c TsAddEncodingCompressed) OnDuplicateFirst() TsAddOnDuplicateFirst {
	c.cs.s = append(c.cs.s, "ON_DUPLICATE", "FIRST")
	return (TsAddOnDuplicateFirst)(c)
}

func (c TsAddEncodingCompressed) OnDuplicateLast() TsAddOnDuplicateLast {
	c.cs.s = append(c.cs.s, "ON_DUPLICATE", "LAST")
	return (TsAddOnDuplicateLast)(c)
}

func (c TsAddEncodingCompressed) OnDuplicateMin() TsAddOnDuplicateMin {
	c.cs.s = append(c.cs.s, "ON_DUPLICATE", "MIN")
	return (TsAddOnDuplicateMin)(c)
}

func (c TsAddEncodingCompressed) OnDuplicateMax() TsAddOnDuplicateMax {
	c.cs.s = append(c.cs.s, "ON_DUPLICATE", "MAX")
	return (TsAddOnDuplicateMax)(c)
}

func (c TsAddEncodingCompressed) OnDuplicateSum() TsAddOnDuplicateSum {
	c.cs.s = append(c.cs.s, "ON_DUPLICATE", "SUM")
	return (TsAddOnDuplicateSum)(c)
}

func (c TsAddEncodingCompressed) Labels() TsAddLabels {
	c.cs.s = append(c.cs.s, "LABELS")
	return (TsAddLabels)(c)
}

func (c TsAddEncodingCompressed) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type TsAddEncodingUncompressed Completed

func (c TsAddEncodingUncompressed) ChunkSize(size int64) TsAddChunkSize {
	c.cs.s = append(c.cs.s, "CHUNK_SIZE", strconv.FormatInt(size, 10))
	return (TsAddChunkSize)(c)
}

func (c TsAddEncodingUncompressed) OnDuplicateBlock() TsAddOnDuplicateBlock {
	c.cf = blockTag
	c.cs.s = append(c.cs.s, "ON_DUPLICATE", "BLOCK")
	return (TsAddOnDuplicateBlock)(c)
}

func (c TsAddEncodingUncompressed) OnDuplicateFirst() TsAddOnDuplicateFirst {
	c.cs.s = append(c.cs.s, "ON_DUPLICATE", "FIRST")
	return (TsAddOnDuplicateFirst)(c)
}

func (c TsAddEncodingUncompressed) OnDuplicateLast() TsAddOnDuplicateLast {
	c.cs.s = append(c.cs.s, "ON_DUPLICATE", "LAST")
	return (TsAddOnDuplicateLast)(c)
}

func (c TsAddEncodingUncompressed) OnDuplicateMin() TsAddOnDuplicateMin {
	c.cs.s = append(c.cs.s, "ON_DUPLICATE", "MIN")
	return (TsAddOnDuplicateMin)(c)
}

func (c TsAddEncodingUncompressed) OnDuplicateMax() TsAddOnDuplicateMax {
	c.cs.s = append(c.cs.s, "ON_DUPLICATE", "MAX")
	return (TsAddOnDuplicateMax)(c)
}

func (c TsAddEncodingUncompressed) OnDuplicateSum() TsAddOnDuplicateSum {
	c.cs.s = append(c.cs.s, "ON_DUPLICATE", "SUM")
	return (TsAddOnDuplicateSum)(c)
}

func (c TsAddEncodingUncompressed) Labels() TsAddLabels {
	c.cs.s = append(c.cs.s, "LABELS")
	return (TsAddLabels)(c)
}

func (c TsAddEncodingUncompressed) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type TsAddKey Completed

func (c TsAddKey) Timestamp(timestamp int64) TsAddTimestamp {
	c.cs.s = append(c.cs.s, strconv.FormatInt(timestamp, 10))
	return (TsAddTimestamp)(c)
}

type TsAddLabels Completed

func (c TsAddLabels) Labels(label string, value string) TsAddLabels {
	c.cs.s = append(c.cs.s, label, value)
	return c
}

func (c TsAddLabels) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type TsAddOnDuplicateBlock Completed

func (c TsAddOnDuplicateBlock) Labels() TsAddLabels {
	c.cs.s = append(c.cs.s, "LABELS")
	return (TsAddLabels)(c)
}

func (c TsAddOnDuplicateBlock) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type TsAddOnDuplicateFirst Completed

func (c TsAddOnDuplicateFirst) Labels() TsAddLabels {
	c.cs.s = append(c.cs.s, "LABELS")
	return (TsAddLabels)(c)
}

func (c TsAddOnDuplicateFirst) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type TsAddOnDuplicateLast Completed

func (c TsAddOnDuplicateLast) Labels() TsAddLabels {
	c.cs.s = append(c.cs.s, "LABELS")
	return (TsAddLabels)(c)
}

func (c TsAddOnDuplicateLast) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type TsAddOnDuplicateMax Completed

func (c TsAddOnDuplicateMax) Labels() TsAddLabels {
	c.cs.s = append(c.cs.s, "LABELS")
	return (TsAddLabels)(c)
}

func (c TsAddOnDuplicateMax) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type TsAddOnDuplicateMin Completed

func (c TsAddOnDuplicateMin) Labels() TsAddLabels {
	c.cs.s = append(c.cs.s, "LABELS")
	return (TsAddLabels)(c)
}

func (c TsAddOnDuplicateMin) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type TsAddOnDuplicateSum Completed

func (c TsAddOnDuplicateSum) Labels() TsAddLabels {
	c.cs.s = append(c.cs.s, "LABELS")
	return (TsAddLabels)(c)
}

func (c TsAddOnDuplicateSum) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type TsAddRetention Completed

func (c TsAddRetention) EncodingUncompressed() TsAddEncodingUncompressed {
	c.cs.s = append(c.cs.s, "ENCODING", "UNCOMPRESSED")
	return (TsAddEncodingUncompressed)(c)
}

func (c TsAddRetention) EncodingCompressed() TsAddEncodingCompressed {
	c.cs.s = append(c.cs.s, "ENCODING", "COMPRESSED")
	return (TsAddEncodingCompressed)(c)
}

func (c TsAddRetention) ChunkSize(size int64) TsAddChunkSize {
	c.cs.s = append(c.cs.s, "CHUNK_SIZE", strconv.FormatInt(size, 10))
	return (TsAddChunkSize)(c)
}

func (c TsAddRetention) OnDuplicateBlock() TsAddOnDuplicateBlock {
	c.cf = blockTag
	c.cs.s = append(c.cs.s, "ON_DUPLICATE", "BLOCK")
	return (TsAddOnDuplicateBlock)(c)
}

func (c TsAddRetention) OnDuplicateFirst() TsAddOnDuplicateFirst {
	c.cs.s = append(c.cs.s, "ON_DUPLICATE", "FIRST")
	return (TsAddOnDuplicateFirst)(c)
}

func (c TsAddRetention) OnDuplicateLast() TsAddOnDuplicateLast {
	c.cs.s = append(c.cs.s, "ON_DUPLICATE", "LAST")
	return (TsAddOnDuplicateLast)(c)
}

func (c TsAddRetention) OnDuplicateMin() TsAddOnDuplicateMin {
	c.cs.s = append(c.cs.s, "ON_DUPLICATE", "MIN")
	return (TsAddOnDuplicateMin)(c)
}

func (c TsAddRetention) OnDuplicateMax() TsAddOnDuplicateMax {
	c.cs.s = append(c.cs.s, "ON_DUPLICATE", "MAX")
	return (TsAddOnDuplicateMax)(c)
}

func (c TsAddRetention) OnDuplicateSum() TsAddOnDuplicateSum {
	c.cs.s = append(c.cs.s, "ON_DUPLICATE", "SUM")
	return (TsAddOnDuplicateSum)(c)
}

func (c TsAddRetention) Labels() TsAddLabels {
	c.cs.s = append(c.cs.s, "LABELS")
	return (TsAddLabels)(c)
}

func (c TsAddRetention) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type TsAddTimestamp Completed

func (c TsAddTimestamp) Value(value float64) TsAddValue {
	c.cs.s = append(c.cs.s, strconv.FormatFloat(value, 'f', -1, 64))
	return (TsAddValue)(c)
}

type TsAddValue Completed

func (c TsAddValue) Retention(retentionperiod int64) TsAddRetention {
	c.cs.s = append(c.cs.s, "RETENTION", strconv.FormatInt(retentionperiod, 10))
	return (TsAddRetention)(c)
}

func (c TsAddValue) EncodingUncompressed() TsAddEncodingUncompressed {
	c.cs.s = append(c.cs.s, "ENCODING", "UNCOMPRESSED")
	return (TsAddEncodingUncompressed)(c)
}

func (c TsAddValue) EncodingCompressed() TsAddEncodingCompressed {
	c.cs.s = append(c.cs.s, "ENCODING", "COMPRESSED")
	return (TsAddEncodingCompressed)(c)
}

func (c TsAddValue) ChunkSize(size int64) TsAddChunkSize {
	c.cs.s = append(c.cs.s, "CHUNK_SIZE", strconv.FormatInt(size, 10))
	return (TsAddChunkSize)(c)
}

func (c TsAddValue) OnDuplicateBlock() TsAddOnDuplicateBlock {
	c.cf = blockTag
	c.cs.s = append(c.cs.s, "ON_DUPLICATE", "BLOCK")
	return (TsAddOnDuplicateBlock)(c)
}

func (c TsAddValue) OnDuplicateFirst() TsAddOnDuplicateFirst {
	c.cs.s = append(c.cs.s, "ON_DUPLICATE", "FIRST")
	return (TsAddOnDuplicateFirst)(c)
}

func (c TsAddValue) OnDuplicateLast() TsAddOnDuplicateLast {
	c.cs.s = append(c.cs.s, "ON_DUPLICATE", "LAST")
	return (TsAddOnDuplicateLast)(c)
}

func (c TsAddValue) OnDuplicateMin() TsAddOnDuplicateMin {
	c.cs.s = append(c.cs.s, "ON_DUPLICATE", "MIN")
	return (TsAddOnDuplicateMin)(c)
}

func (c TsAddValue) OnDuplicateMax() TsAddOnDuplicateMax {
	c.cs.s = append(c.cs.s, "ON_DUPLICATE", "MAX")
	return (TsAddOnDuplicateMax)(c)
}

func (c TsAddValue) OnDuplicateSum() TsAddOnDuplicateSum {
	c.cs.s = append(c.cs.s, "ON_DUPLICATE", "SUM")
	return (TsAddOnDuplicateSum)(c)
}

func (c TsAddValue) Labels() TsAddLabels {
	c.cs.s = append(c.cs.s, "LABELS")
	return (TsAddLabels)(c)
}

func (c TsAddValue) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type TsAlter Completed

func (b Builder) TsAlter() (c TsAlter) {
	c = TsAlter{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "TS.ALTER")
	return c
}

func (c TsAlter) Key(key string) TsAlterKey {
	if c.ks&NoSlot == NoSlot {
		c.ks = NoSlot | slot(key)
	} else {
		c.ks = check(c.ks, slot(key))
	}
	c.cs.s = append(c.cs.s, key)
	return (TsAlterKey)(c)
}

type TsAlterChunkSize Completed

func (c TsAlterChunkSize) DuplicatePolicyBlock() TsAlterDuplicatePolicyBlock {
	c.cf = blockTag
	c.cs.s = append(c.cs.s, "DUPLICATE_POLICY", "BLOCK")
	return (TsAlterDuplicatePolicyBlock)(c)
}

func (c TsAlterChunkSize) DuplicatePolicyFirst() TsAlterDuplicatePolicyFirst {
	c.cs.s = append(c.cs.s, "DUPLICATE_POLICY", "FIRST")
	return (TsAlterDuplicatePolicyFirst)(c)
}

func (c TsAlterChunkSize) DuplicatePolicyLast() TsAlterDuplicatePolicyLast {
	c.cs.s = append(c.cs.s, "DUPLICATE_POLICY", "LAST")
	return (TsAlterDuplicatePolicyLast)(c)
}

func (c TsAlterChunkSize) DuplicatePolicyMin() TsAlterDuplicatePolicyMin {
	c.cs.s = append(c.cs.s, "DUPLICATE_POLICY", "MIN")
	return (TsAlterDuplicatePolicyMin)(c)
}

func (c TsAlterChunkSize) DuplicatePolicyMax() TsAlterDuplicatePolicyMax {
	c.cs.s = append(c.cs.s, "DUPLICATE_POLICY", "MAX")
	return (TsAlterDuplicatePolicyMax)(c)
}

func (c TsAlterChunkSize) DuplicatePolicySum() TsAlterDuplicatePolicySum {
	c.cs.s = append(c.cs.s, "DUPLICATE_POLICY", "SUM")
	return (TsAlterDuplicatePolicySum)(c)
}

func (c TsAlterChunkSize) Labels() TsAlterLabels {
	c.cs.s = append(c.cs.s, "LABELS")
	return (TsAlterLabels)(c)
}

func (c TsAlterChunkSize) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type TsAlterDuplicatePolicyBlock Completed

func (c TsAlterDuplicatePolicyBlock) Labels() TsAlterLabels {
	c.cs.s = append(c.cs.s, "LABELS")
	return (TsAlterLabels)(c)
}

func (c TsAlterDuplicatePolicyBlock) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type TsAlterDuplicatePolicyFirst Completed

func (c TsAlterDuplicatePolicyFirst) Labels() TsAlterLabels {
	c.cs.s = append(c.cs.s, "LABELS")
	return (TsAlterLabels)(c)
}

func (c TsAlterDuplicatePolicyFirst) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type TsAlterDuplicatePolicyLast Completed

func (c TsAlterDuplicatePolicyLast) Labels() TsAlterLabels {
	c.cs.s = append(c.cs.s, "LABELS")
	return (TsAlterLabels)(c)
}

func (c TsAlterDuplicatePolicyLast) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type TsAlterDuplicatePolicyMax Completed

func (c TsAlterDuplicatePolicyMax) Labels() TsAlterLabels {
	c.cs.s = append(c.cs.s, "LABELS")
	return (TsAlterLabels)(c)
}

func (c TsAlterDuplicatePolicyMax) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type TsAlterDuplicatePolicyMin Completed

func (c TsAlterDuplicatePolicyMin) Labels() TsAlterLabels {
	c.cs.s = append(c.cs.s, "LABELS")
	return (TsAlterLabels)(c)
}

func (c TsAlterDuplicatePolicyMin) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type TsAlterDuplicatePolicySum Completed

func (c TsAlterDuplicatePolicySum) Labels() TsAlterLabels {
	c.cs.s = append(c.cs.s, "LABELS")
	return (TsAlterLabels)(c)
}

func (c TsAlterDuplicatePolicySum) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type TsAlterKey Completed

func (c TsAlterKey) Retention(retentionperiod int64) TsAlterRetention {
	c.cs.s = append(c.cs.s, "RETENTION", strconv.FormatInt(retentionperiod, 10))
	return (TsAlterRetention)(c)
}

func (c TsAlterKey) ChunkSize(size int64) TsAlterChunkSize {
	c.cs.s = append(c.cs.s, "CHUNK_SIZE", strconv.FormatInt(size, 10))
	return (TsAlterChunkSize)(c)
}

func (c TsAlterKey) DuplicatePolicyBlock() TsAlterDuplicatePolicyBlock {
	c.cf = blockTag
	c.cs.s = append(c.cs.s, "DUPLICATE_POLICY", "BLOCK")
	return (TsAlterDuplicatePolicyBlock)(c)
}

func (c TsAlterKey) DuplicatePolicyFirst() TsAlterDuplicatePolicyFirst {
	c.cs.s = append(c.cs.s, "DUPLICATE_POLICY", "FIRST")
	return (TsAlterDuplicatePolicyFirst)(c)
}

func (c TsAlterKey) DuplicatePolicyLast() TsAlterDuplicatePolicyLast {
	c.cs.s = append(c.cs.s, "DUPLICATE_POLICY", "LAST")
	return (TsAlterDuplicatePolicyLast)(c)
}

func (c TsAlterKey) DuplicatePolicyMin() TsAlterDuplicatePolicyMin {
	c.cs.s = append(c.cs.s, "DUPLICATE_POLICY", "MIN")
	return (TsAlterDuplicatePolicyMin)(c)
}

func (c TsAlterKey) DuplicatePolicyMax() TsAlterDuplicatePolicyMax {
	c.cs.s = append(c.cs.s, "DUPLICATE_POLICY", "MAX")
	return (TsAlterDuplicatePolicyMax)(c)
}

func (c TsAlterKey) DuplicatePolicySum() TsAlterDuplicatePolicySum {
	c.cs.s = append(c.cs.s, "DUPLICATE_POLICY", "SUM")
	return (TsAlterDuplicatePolicySum)(c)
}

func (c TsAlterKey) Labels() TsAlterLabels {
	c.cs.s = append(c.cs.s, "LABELS")
	return (TsAlterLabels)(c)
}

func (c TsAlterKey) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type TsAlterLabels Completed

func (c TsAlterLabels) Labels(label string, value string) TsAlterLabels {
	c.cs.s = append(c.cs.s, label, value)
	return c
}

func (c TsAlterLabels) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type TsAlterRetention Completed

func (c TsAlterRetention) ChunkSize(size int64) TsAlterChunkSize {
	c.cs.s = append(c.cs.s, "CHUNK_SIZE", strconv.FormatInt(size, 10))
	return (TsAlterChunkSize)(c)
}

func (c TsAlterRetention) DuplicatePolicyBlock() TsAlterDuplicatePolicyBlock {
	c.cf = blockTag
	c.cs.s = append(c.cs.s, "DUPLICATE_POLICY", "BLOCK")
	return (TsAlterDuplicatePolicyBlock)(c)
}

func (c TsAlterRetention) DuplicatePolicyFirst() TsAlterDuplicatePolicyFirst {
	c.cs.s = append(c.cs.s, "DUPLICATE_POLICY", "FIRST")
	return (TsAlterDuplicatePolicyFirst)(c)
}

func (c TsAlterRetention) DuplicatePolicyLast() TsAlterDuplicatePolicyLast {
	c.cs.s = append(c.cs.s, "DUPLICATE_POLICY", "LAST")
	return (TsAlterDuplicatePolicyLast)(c)
}

func (c TsAlterRetention) DuplicatePolicyMin() TsAlterDuplicatePolicyMin {
	c.cs.s = append(c.cs.s, "DUPLICATE_POLICY", "MIN")
	return (TsAlterDuplicatePolicyMin)(c)
}

func (c TsAlterRetention) DuplicatePolicyMax() TsAlterDuplicatePolicyMax {
	c.cs.s = append(c.cs.s, "DUPLICATE_POLICY", "MAX")
	return (TsAlterDuplicatePolicyMax)(c)
}

func (c TsAlterRetention) DuplicatePolicySum() TsAlterDuplicatePolicySum {
	c.cs.s = append(c.cs.s, "DUPLICATE_POLICY", "SUM")
	return (TsAlterDuplicatePolicySum)(c)
}

func (c TsAlterRetention) Labels() TsAlterLabels {
	c.cs.s = append(c.cs.s, "LABELS")
	return (TsAlterLabels)(c)
}

func (c TsAlterRetention) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type TsCreate Completed

func (b Builder) TsCreate() (c TsCreate) {
	c = TsCreate{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "TS.CREATE")
	return c
}

func (c TsCreate) Key(key string) TsCreateKey {
	if c.ks&NoSlot == NoSlot {
		c.ks = NoSlot | slot(key)
	} else {
		c.ks = check(c.ks, slot(key))
	}
	c.cs.s = append(c.cs.s, key)
	return (TsCreateKey)(c)
}

type TsCreateChunkSize Completed

func (c TsCreateChunkSize) DuplicatePolicyBlock() TsCreateDuplicatePolicyBlock {
	c.cf = blockTag
	c.cs.s = append(c.cs.s, "DUPLICATE_POLICY", "BLOCK")
	return (TsCreateDuplicatePolicyBlock)(c)
}

func (c TsCreateChunkSize) DuplicatePolicyFirst() TsCreateDuplicatePolicyFirst {
	c.cs.s = append(c.cs.s, "DUPLICATE_POLICY", "FIRST")
	return (TsCreateDuplicatePolicyFirst)(c)
}

func (c TsCreateChunkSize) DuplicatePolicyLast() TsCreateDuplicatePolicyLast {
	c.cs.s = append(c.cs.s, "DUPLICATE_POLICY", "LAST")
	return (TsCreateDuplicatePolicyLast)(c)
}

func (c TsCreateChunkSize) DuplicatePolicyMin() TsCreateDuplicatePolicyMin {
	c.cs.s = append(c.cs.s, "DUPLICATE_POLICY", "MIN")
	return (TsCreateDuplicatePolicyMin)(c)
}

func (c TsCreateChunkSize) DuplicatePolicyMax() TsCreateDuplicatePolicyMax {
	c.cs.s = append(c.cs.s, "DUPLICATE_POLICY", "MAX")
	return (TsCreateDuplicatePolicyMax)(c)
}

func (c TsCreateChunkSize) DuplicatePolicySum() TsCreateDuplicatePolicySum {
	c.cs.s = append(c.cs.s, "DUPLICATE_POLICY", "SUM")
	return (TsCreateDuplicatePolicySum)(c)
}

func (c TsCreateChunkSize) Labels() TsCreateLabels {
	c.cs.s = append(c.cs.s, "LABELS")
	return (TsCreateLabels)(c)
}

func (c TsCreateChunkSize) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type TsCreateDuplicatePolicyBlock Completed

func (c TsCreateDuplicatePolicyBlock) Labels() TsCreateLabels {
	c.cs.s = append(c.cs.s, "LABELS")
	return (TsCreateLabels)(c)
}

func (c TsCreateDuplicatePolicyBlock) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type TsCreateDuplicatePolicyFirst Completed

func (c TsCreateDuplicatePolicyFirst) Labels() TsCreateLabels {
	c.cs.s = append(c.cs.s, "LABELS")
	return (TsCreateLabels)(c)
}

func (c TsCreateDuplicatePolicyFirst) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type TsCreateDuplicatePolicyLast Completed

func (c TsCreateDuplicatePolicyLast) Labels() TsCreateLabels {
	c.cs.s = append(c.cs.s, "LABELS")
	return (TsCreateLabels)(c)
}

func (c TsCreateDuplicatePolicyLast) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type TsCreateDuplicatePolicyMax Completed

func (c TsCreateDuplicatePolicyMax) Labels() TsCreateLabels {
	c.cs.s = append(c.cs.s, "LABELS")
	return (TsCreateLabels)(c)
}

func (c TsCreateDuplicatePolicyMax) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type TsCreateDuplicatePolicyMin Completed

func (c TsCreateDuplicatePolicyMin) Labels() TsCreateLabels {
	c.cs.s = append(c.cs.s, "LABELS")
	return (TsCreateLabels)(c)
}

func (c TsCreateDuplicatePolicyMin) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type TsCreateDuplicatePolicySum Completed

func (c TsCreateDuplicatePolicySum) Labels() TsCreateLabels {
	c.cs.s = append(c.cs.s, "LABELS")
	return (TsCreateLabels)(c)
}

func (c TsCreateDuplicatePolicySum) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type TsCreateEncodingCompressed Completed

func (c TsCreateEncodingCompressed) ChunkSize(size int64) TsCreateChunkSize {
	c.cs.s = append(c.cs.s, "CHUNK_SIZE", strconv.FormatInt(size, 10))
	return (TsCreateChunkSize)(c)
}

func (c TsCreateEncodingCompressed) DuplicatePolicyBlock() TsCreateDuplicatePolicyBlock {
	c.cf = blockTag
	c.cs.s = append(c.cs.s, "DUPLICATE_POLICY", "BLOCK")
	return (TsCreateDuplicatePolicyBlock)(c)
}

func (c TsCreateEncodingCompressed) DuplicatePolicyFirst() TsCreateDuplicatePolicyFirst {
	c.cs.s = append(c.cs.s, "DUPLICATE_POLICY", "FIRST")
	return (TsCreateDuplicatePolicyFirst)(c)
}

func (c TsCreateEncodingCompressed) DuplicatePolicyLast() TsCreateDuplicatePolicyLast {
	c.cs.s = append(c.cs.s, "DUPLICATE_POLICY", "LAST")
	return (TsCreateDuplicatePolicyLast)(c)
}

func (c TsCreateEncodingCompressed) DuplicatePolicyMin() TsCreateDuplicatePolicyMin {
	c.cs.s = append(c.cs.s, "DUPLICATE_POLICY", "MIN")
	return (TsCreateDuplicatePolicyMin)(c)
}

func (c TsCreateEncodingCompressed) DuplicatePolicyMax() TsCreateDuplicatePolicyMax {
	c.cs.s = append(c.cs.s, "DUPLICATE_POLICY", "MAX")
	return (TsCreateDuplicatePolicyMax)(c)
}

func (c TsCreateEncodingCompressed) DuplicatePolicySum() TsCreateDuplicatePolicySum {
	c.cs.s = append(c.cs.s, "DUPLICATE_POLICY", "SUM")
	return (TsCreateDuplicatePolicySum)(c)
}

func (c TsCreateEncodingCompressed) Labels() TsCreateLabels {
	c.cs.s = append(c.cs.s, "LABELS")
	return (TsCreateLabels)(c)
}

func (c TsCreateEncodingCompressed) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type TsCreateEncodingUncompressed Completed

func (c TsCreateEncodingUncompressed) ChunkSize(size int64) TsCreateChunkSize {
	c.cs.s = append(c.cs.s, "CHUNK_SIZE", strconv.FormatInt(size, 10))
	return (TsCreateChunkSize)(c)
}

func (c TsCreateEncodingUncompressed) DuplicatePolicyBlock() TsCreateDuplicatePolicyBlock {
	c.cf = blockTag
	c.cs.s = append(c.cs.s, "DUPLICATE_POLICY", "BLOCK")
	return (TsCreateDuplicatePolicyBlock)(c)
}

func (c TsCreateEncodingUncompressed) DuplicatePolicyFirst() TsCreateDuplicatePolicyFirst {
	c.cs.s = append(c.cs.s, "DUPLICATE_POLICY", "FIRST")
	return (TsCreateDuplicatePolicyFirst)(c)
}

func (c TsCreateEncodingUncompressed) DuplicatePolicyLast() TsCreateDuplicatePolicyLast {
	c.cs.s = append(c.cs.s, "DUPLICATE_POLICY", "LAST")
	return (TsCreateDuplicatePolicyLast)(c)
}

func (c TsCreateEncodingUncompressed) DuplicatePolicyMin() TsCreateDuplicatePolicyMin {
	c.cs.s = append(c.cs.s, "DUPLICATE_POLICY", "MIN")
	return (TsCreateDuplicatePolicyMin)(c)
}

func (c TsCreateEncodingUncompressed) DuplicatePolicyMax() TsCreateDuplicatePolicyMax {
	c.cs.s = append(c.cs.s, "DUPLICATE_POLICY", "MAX")
	return (TsCreateDuplicatePolicyMax)(c)
}

func (c TsCreateEncodingUncompressed) DuplicatePolicySum() TsCreateDuplicatePolicySum {
	c.cs.s = append(c.cs.s, "DUPLICATE_POLICY", "SUM")
	return (TsCreateDuplicatePolicySum)(c)
}

func (c TsCreateEncodingUncompressed) Labels() TsCreateLabels {
	c.cs.s = append(c.cs.s, "LABELS")
	return (TsCreateLabels)(c)
}

func (c TsCreateEncodingUncompressed) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type TsCreateKey Completed

func (c TsCreateKey) Retention(retentionperiod int64) TsCreateRetention {
	c.cs.s = append(c.cs.s, "RETENTION", strconv.FormatInt(retentionperiod, 10))
	return (TsCreateRetention)(c)
}

func (c TsCreateKey) EncodingUncompressed() TsCreateEncodingUncompressed {
	c.cs.s = append(c.cs.s, "ENCODING", "UNCOMPRESSED")
	return (TsCreateEncodingUncompressed)(c)
}

func (c TsCreateKey) EncodingCompressed() TsCreateEncodingCompressed {
	c.cs.s = append(c.cs.s, "ENCODING", "COMPRESSED")
	return (TsCreateEncodingCompressed)(c)
}

func (c TsCreateKey) ChunkSize(size int64) TsCreateChunkSize {
	c.cs.s = append(c.cs.s, "CHUNK_SIZE", strconv.FormatInt(size, 10))
	return (TsCreateChunkSize)(c)
}

func (c TsCreateKey) DuplicatePolicyBlock() TsCreateDuplicatePolicyBlock {
	c.cf = blockTag
	c.cs.s = append(c.cs.s, "DUPLICATE_POLICY", "BLOCK")
	return (TsCreateDuplicatePolicyBlock)(c)
}

func (c TsCreateKey) DuplicatePolicyFirst() TsCreateDuplicatePolicyFirst {
	c.cs.s = append(c.cs.s, "DUPLICATE_POLICY", "FIRST")
	return (TsCreateDuplicatePolicyFirst)(c)
}

func (c TsCreateKey) DuplicatePolicyLast() TsCreateDuplicatePolicyLast {
	c.cs.s = append(c.cs.s, "DUPLICATE_POLICY", "LAST")
	return (TsCreateDuplicatePolicyLast)(c)
}

func (c TsCreateKey) DuplicatePolicyMin() TsCreateDuplicatePolicyMin {
	c.cs.s = append(c.cs.s, "DUPLICATE_POLICY", "MIN")
	return (TsCreateDuplicatePolicyMin)(c)
}

func (c TsCreateKey) DuplicatePolicyMax() TsCreateDuplicatePolicyMax {
	c.cs.s = append(c.cs.s, "DUPLICATE_POLICY", "MAX")
	return (TsCreateDuplicatePolicyMax)(c)
}

func (c TsCreateKey) DuplicatePolicySum() TsCreateDuplicatePolicySum {
	c.cs.s = append(c.cs.s, "DUPLICATE_POLICY", "SUM")
	return (TsCreateDuplicatePolicySum)(c)
}

func (c TsCreateKey) Labels() TsCreateLabels {
	c.cs.s = append(c.cs.s, "LABELS")
	return (TsCreateLabels)(c)
}

func (c TsCreateKey) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type TsCreateLabels Completed

func (c TsCreateLabels) Labels(label string, value string) TsCreateLabels {
	c.cs.s = append(c.cs.s, label, value)
	return c
}

func (c TsCreateLabels) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type TsCreateRetention Completed

func (c TsCreateRetention) EncodingUncompressed() TsCreateEncodingUncompressed {
	c.cs.s = append(c.cs.s, "ENCODING", "UNCOMPRESSED")
	return (TsCreateEncodingUncompressed)(c)
}

func (c TsCreateRetention) EncodingCompressed() TsCreateEncodingCompressed {
	c.cs.s = append(c.cs.s, "ENCODING", "COMPRESSED")
	return (TsCreateEncodingCompressed)(c)
}

func (c TsCreateRetention) ChunkSize(size int64) TsCreateChunkSize {
	c.cs.s = append(c.cs.s, "CHUNK_SIZE", strconv.FormatInt(size, 10))
	return (TsCreateChunkSize)(c)
}

func (c TsCreateRetention) DuplicatePolicyBlock() TsCreateDuplicatePolicyBlock {
	c.cf = blockTag
	c.cs.s = append(c.cs.s, "DUPLICATE_POLICY", "BLOCK")
	return (TsCreateDuplicatePolicyBlock)(c)
}

func (c TsCreateRetention) DuplicatePolicyFirst() TsCreateDuplicatePolicyFirst {
	c.cs.s = append(c.cs.s, "DUPLICATE_POLICY", "FIRST")
	return (TsCreateDuplicatePolicyFirst)(c)
}

func (c TsCreateRetention) DuplicatePolicyLast() TsCreateDuplicatePolicyLast {
	c.cs.s = append(c.cs.s, "DUPLICATE_POLICY", "LAST")
	return (TsCreateDuplicatePolicyLast)(c)
}

func (c TsCreateRetention) DuplicatePolicyMin() TsCreateDuplicatePolicyMin {
	c.cs.s = append(c.cs.s, "DUPLICATE_POLICY", "MIN")
	return (TsCreateDuplicatePolicyMin)(c)
}

func (c TsCreateRetention) DuplicatePolicyMax() TsCreateDuplicatePolicyMax {
	c.cs.s = append(c.cs.s, "DUPLICATE_POLICY", "MAX")
	return (TsCreateDuplicatePolicyMax)(c)
}

func (c TsCreateRetention) DuplicatePolicySum() TsCreateDuplicatePolicySum {
	c.cs.s = append(c.cs.s, "DUPLICATE_POLICY", "SUM")
	return (TsCreateDuplicatePolicySum)(c)
}

func (c TsCreateRetention) Labels() TsCreateLabels {
	c.cs.s = append(c.cs.s, "LABELS")
	return (TsCreateLabels)(c)
}

func (c TsCreateRetention) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type TsCreaterule Completed

func (b Builder) TsCreaterule() (c TsCreaterule) {
	c = TsCreaterule{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "TS.CREATERULE")
	return c
}

func (c TsCreaterule) Sourcekey(sourcekey string) TsCreateruleSourcekey {
	if c.ks&NoSlot == NoSlot {
		c.ks = NoSlot | slot(sourcekey)
	} else {
		c.ks = check(c.ks, slot(sourcekey))
	}
	c.cs.s = append(c.cs.s, sourcekey)
	return (TsCreateruleSourcekey)(c)
}

type TsCreateruleAggregationAvg Completed

func (c TsCreateruleAggregationAvg) Bucketduration(bucketduration int64) TsCreateruleBucketduration {
	c.cs.s = append(c.cs.s, strconv.FormatInt(bucketduration, 10))
	return (TsCreateruleBucketduration)(c)
}

type TsCreateruleAggregationCount Completed

func (c TsCreateruleAggregationCount) Bucketduration(bucketduration int64) TsCreateruleBucketduration {
	c.cs.s = append(c.cs.s, strconv.FormatInt(bucketduration, 10))
	return (TsCreateruleBucketduration)(c)
}

type TsCreateruleAggregationFirst Completed

func (c TsCreateruleAggregationFirst) Bucketduration(bucketduration int64) TsCreateruleBucketduration {
	c.cs.s = append(c.cs.s, strconv.FormatInt(bucketduration, 10))
	return (TsCreateruleBucketduration)(c)
}

type TsCreateruleAggregationLast Completed

func (c TsCreateruleAggregationLast) Bucketduration(bucketduration int64) TsCreateruleBucketduration {
	c.cs.s = append(c.cs.s, strconv.FormatInt(bucketduration, 10))
	return (TsCreateruleBucketduration)(c)
}

type TsCreateruleAggregationMax Completed

func (c TsCreateruleAggregationMax) Bucketduration(bucketduration int64) TsCreateruleBucketduration {
	c.cs.s = append(c.cs.s, strconv.FormatInt(bucketduration, 10))
	return (TsCreateruleBucketduration)(c)
}

type TsCreateruleAggregationMin Completed

func (c TsCreateruleAggregationMin) Bucketduration(bucketduration int64) TsCreateruleBucketduration {
	c.cs.s = append(c.cs.s, strconv.FormatInt(bucketduration, 10))
	return (TsCreateruleBucketduration)(c)
}

type TsCreateruleAggregationRange Completed

func (c TsCreateruleAggregationRange) Bucketduration(bucketduration int64) TsCreateruleBucketduration {
	c.cs.s = append(c.cs.s, strconv.FormatInt(bucketduration, 10))
	return (TsCreateruleBucketduration)(c)
}

type TsCreateruleAggregationStdP Completed

func (c TsCreateruleAggregationStdP) Bucketduration(bucketduration int64) TsCreateruleBucketduration {
	c.cs.s = append(c.cs.s, strconv.FormatInt(bucketduration, 10))
	return (TsCreateruleBucketduration)(c)
}

type TsCreateruleAggregationStdS Completed

func (c TsCreateruleAggregationStdS) Bucketduration(bucketduration int64) TsCreateruleBucketduration {
	c.cs.s = append(c.cs.s, strconv.FormatInt(bucketduration, 10))
	return (TsCreateruleBucketduration)(c)
}

type TsCreateruleAggregationSum Completed

func (c TsCreateruleAggregationSum) Bucketduration(bucketduration int64) TsCreateruleBucketduration {
	c.cs.s = append(c.cs.s, strconv.FormatInt(bucketduration, 10))
	return (TsCreateruleBucketduration)(c)
}

type TsCreateruleAggregationTwa Completed

func (c TsCreateruleAggregationTwa) Bucketduration(bucketduration int64) TsCreateruleBucketduration {
	c.cs.s = append(c.cs.s, strconv.FormatInt(bucketduration, 10))
	return (TsCreateruleBucketduration)(c)
}

type TsCreateruleAggregationVarP Completed

func (c TsCreateruleAggregationVarP) Bucketduration(bucketduration int64) TsCreateruleBucketduration {
	c.cs.s = append(c.cs.s, strconv.FormatInt(bucketduration, 10))
	return (TsCreateruleBucketduration)(c)
}

type TsCreateruleAggregationVarS Completed

func (c TsCreateruleAggregationVarS) Bucketduration(bucketduration int64) TsCreateruleBucketduration {
	c.cs.s = append(c.cs.s, strconv.FormatInt(bucketduration, 10))
	return (TsCreateruleBucketduration)(c)
}

type TsCreateruleAligntimestamp Completed

func (c TsCreateruleAligntimestamp) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type TsCreateruleBucketduration Completed

func (c TsCreateruleBucketduration) Aligntimestamp(aligntimestamp int64) TsCreateruleAligntimestamp {
	c.cs.s = append(c.cs.s, strconv.FormatInt(aligntimestamp, 10))
	return (TsCreateruleAligntimestamp)(c)
}

func (c TsCreateruleBucketduration) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type TsCreateruleDestkey Completed

func (c TsCreateruleDestkey) AggregationAvg() TsCreateruleAggregationAvg {
	c.cs.s = append(c.cs.s, "AGGREGATION", "AVG")
	return (TsCreateruleAggregationAvg)(c)
}

func (c TsCreateruleDestkey) AggregationSum() TsCreateruleAggregationSum {
	c.cs.s = append(c.cs.s, "AGGREGATION", "SUM")
	return (TsCreateruleAggregationSum)(c)
}

func (c TsCreateruleDestkey) AggregationMin() TsCreateruleAggregationMin {
	c.cs.s = append(c.cs.s, "AGGREGATION", "MIN")
	return (TsCreateruleAggregationMin)(c)
}

func (c TsCreateruleDestkey) AggregationMax() TsCreateruleAggregationMax {
	c.cs.s = append(c.cs.s, "AGGREGATION", "MAX")
	return (TsCreateruleAggregationMax)(c)
}

func (c TsCreateruleDestkey) AggregationRange() TsCreateruleAggregationRange {
	c.cs.s = append(c.cs.s, "AGGREGATION", "RANGE")
	return (TsCreateruleAggregationRange)(c)
}

func (c TsCreateruleDestkey) AggregationCount() TsCreateruleAggregationCount {
	c.cs.s = append(c.cs.s, "AGGREGATION", "COUNT")
	return (TsCreateruleAggregationCount)(c)
}

func (c TsCreateruleDestkey) AggregationFirst() TsCreateruleAggregationFirst {
	c.cs.s = append(c.cs.s, "AGGREGATION", "FIRST")
	return (TsCreateruleAggregationFirst)(c)
}

func (c TsCreateruleDestkey) AggregationLast() TsCreateruleAggregationLast {
	c.cs.s = append(c.cs.s, "AGGREGATION", "LAST")
	return (TsCreateruleAggregationLast)(c)
}

func (c TsCreateruleDestkey) AggregationStdP() TsCreateruleAggregationStdP {
	c.cs.s = append(c.cs.s, "AGGREGATION", "STD.P")
	return (TsCreateruleAggregationStdP)(c)
}

func (c TsCreateruleDestkey) AggregationStdS() TsCreateruleAggregationStdS {
	c.cs.s = append(c.cs.s, "AGGREGATION", "STD.S")
	return (TsCreateruleAggregationStdS)(c)
}

func (c TsCreateruleDestkey) AggregationVarP() TsCreateruleAggregationVarP {
	c.cs.s = append(c.cs.s, "AGGREGATION", "VAR.P")
	return (TsCreateruleAggregationVarP)(c)
}

func (c TsCreateruleDestkey) AggregationVarS() TsCreateruleAggregationVarS {
	c.cs.s = append(c.cs.s, "AGGREGATION", "VAR.S")
	return (TsCreateruleAggregationVarS)(c)
}

func (c TsCreateruleDestkey) AggregationTwa() TsCreateruleAggregationTwa {
	c.cs.s = append(c.cs.s, "AGGREGATION", "TWA")
	return (TsCreateruleAggregationTwa)(c)
}

type TsCreateruleSourcekey Completed

func (c TsCreateruleSourcekey) Destkey(destkey string) TsCreateruleDestkey {
	if c.ks&NoSlot == NoSlot {
		c.ks = NoSlot | slot(destkey)
	} else {
		c.ks = check(c.ks, slot(destkey))
	}
	c.cs.s = append(c.cs.s, destkey)
	return (TsCreateruleDestkey)(c)
}

type TsDecrby Completed

func (b Builder) TsDecrby() (c TsDecrby) {
	c = TsDecrby{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "TS.DECRBY")
	return c
}

func (c TsDecrby) Key(key string) TsDecrbyKey {
	if c.ks&NoSlot == NoSlot {
		c.ks = NoSlot | slot(key)
	} else {
		c.ks = check(c.ks, slot(key))
	}
	c.cs.s = append(c.cs.s, key)
	return (TsDecrbyKey)(c)
}

type TsDecrbyChunkSize Completed

func (c TsDecrbyChunkSize) Labels() TsDecrbyLabels {
	c.cs.s = append(c.cs.s, "LABELS")
	return (TsDecrbyLabels)(c)
}

func (c TsDecrbyChunkSize) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type TsDecrbyKey Completed

func (c TsDecrbyKey) Value(value float64) TsDecrbyValue {
	c.cs.s = append(c.cs.s, strconv.FormatFloat(value, 'f', -1, 64))
	return (TsDecrbyValue)(c)
}

type TsDecrbyLabels Completed

func (c TsDecrbyLabels) Labels(label string, value string) TsDecrbyLabels {
	c.cs.s = append(c.cs.s, label, value)
	return c
}

func (c TsDecrbyLabels) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type TsDecrbyRetention Completed

func (c TsDecrbyRetention) Uncompressed() TsDecrbyUncompressed {
	c.cs.s = append(c.cs.s, "UNCOMPRESSED")
	return (TsDecrbyUncompressed)(c)
}

func (c TsDecrbyRetention) ChunkSize(size int64) TsDecrbyChunkSize {
	c.cs.s = append(c.cs.s, "CHUNK_SIZE", strconv.FormatInt(size, 10))
	return (TsDecrbyChunkSize)(c)
}

func (c TsDecrbyRetention) Labels() TsDecrbyLabels {
	c.cs.s = append(c.cs.s, "LABELS")
	return (TsDecrbyLabels)(c)
}

func (c TsDecrbyRetention) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type TsDecrbyTimestamp Completed

func (c TsDecrbyTimestamp) Retention(retentionperiod int64) TsDecrbyRetention {
	c.cs.s = append(c.cs.s, "RETENTION", strconv.FormatInt(retentionperiod, 10))
	return (TsDecrbyRetention)(c)
}

func (c TsDecrbyTimestamp) Uncompressed() TsDecrbyUncompressed {
	c.cs.s = append(c.cs.s, "UNCOMPRESSED")
	return (TsDecrbyUncompressed)(c)
}

func (c TsDecrbyTimestamp) ChunkSize(size int64) TsDecrbyChunkSize {
	c.cs.s = append(c.cs.s, "CHUNK_SIZE", strconv.FormatInt(size, 10))
	return (TsDecrbyChunkSize)(c)
}

func (c TsDecrbyTimestamp) Labels() TsDecrbyLabels {
	c.cs.s = append(c.cs.s, "LABELS")
	return (TsDecrbyLabels)(c)
}

func (c TsDecrbyTimestamp) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type TsDecrbyUncompressed Completed

func (c TsDecrbyUncompressed) ChunkSize(size int64) TsDecrbyChunkSize {
	c.cs.s = append(c.cs.s, "CHUNK_SIZE", strconv.FormatInt(size, 10))
	return (TsDecrbyChunkSize)(c)
}

func (c TsDecrbyUncompressed) Labels() TsDecrbyLabels {
	c.cs.s = append(c.cs.s, "LABELS")
	return (TsDecrbyLabels)(c)
}

func (c TsDecrbyUncompressed) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type TsDecrbyValue Completed

func (c TsDecrbyValue) Timestamp(timestamp int64) TsDecrbyTimestamp {
	c.cs.s = append(c.cs.s, "TIMESTAMP", strconv.FormatInt(timestamp, 10))
	return (TsDecrbyTimestamp)(c)
}

func (c TsDecrbyValue) Retention(retentionperiod int64) TsDecrbyRetention {
	c.cs.s = append(c.cs.s, "RETENTION", strconv.FormatInt(retentionperiod, 10))
	return (TsDecrbyRetention)(c)
}

func (c TsDecrbyValue) Uncompressed() TsDecrbyUncompressed {
	c.cs.s = append(c.cs.s, "UNCOMPRESSED")
	return (TsDecrbyUncompressed)(c)
}

func (c TsDecrbyValue) ChunkSize(size int64) TsDecrbyChunkSize {
	c.cs.s = append(c.cs.s, "CHUNK_SIZE", strconv.FormatInt(size, 10))
	return (TsDecrbyChunkSize)(c)
}

func (c TsDecrbyValue) Labels() TsDecrbyLabels {
	c.cs.s = append(c.cs.s, "LABELS")
	return (TsDecrbyLabels)(c)
}

func (c TsDecrbyValue) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type TsDel Completed

func (b Builder) TsDel() (c TsDel) {
	c = TsDel{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "TS.DEL")
	return c
}

func (c TsDel) Key(key string) TsDelKey {
	if c.ks&NoSlot == NoSlot {
		c.ks = NoSlot | slot(key)
	} else {
		c.ks = check(c.ks, slot(key))
	}
	c.cs.s = append(c.cs.s, key)
	return (TsDelKey)(c)
}

type TsDelFromTimestamp Completed

func (c TsDelFromTimestamp) ToTimestamp(toTimestamp int64) TsDelToTimestamp {
	c.cs.s = append(c.cs.s, strconv.FormatInt(toTimestamp, 10))
	return (TsDelToTimestamp)(c)
}

type TsDelKey Completed

func (c TsDelKey) FromTimestamp(fromTimestamp int64) TsDelFromTimestamp {
	c.cs.s = append(c.cs.s, strconv.FormatInt(fromTimestamp, 10))
	return (TsDelFromTimestamp)(c)
}

type TsDelToTimestamp Completed

func (c TsDelToTimestamp) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type TsDeleterule Completed

func (b Builder) TsDeleterule() (c TsDeleterule) {
	c = TsDeleterule{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "TS.DELETERULE")
	return c
}

func (c TsDeleterule) Sourcekey(sourcekey string) TsDeleteruleSourcekey {
	if c.ks&NoSlot == NoSlot {
		c.ks = NoSlot | slot(sourcekey)
	} else {
		c.ks = check(c.ks, slot(sourcekey))
	}
	c.cs.s = append(c.cs.s, sourcekey)
	return (TsDeleteruleSourcekey)(c)
}

type TsDeleteruleDestkey Completed

func (c TsDeleteruleDestkey) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type TsDeleteruleSourcekey Completed

func (c TsDeleteruleSourcekey) Destkey(destkey string) TsDeleteruleDestkey {
	if c.ks&NoSlot == NoSlot {
		c.ks = NoSlot | slot(destkey)
	} else {
		c.ks = check(c.ks, slot(destkey))
	}
	c.cs.s = append(c.cs.s, destkey)
	return (TsDeleteruleDestkey)(c)
}

type TsGet Completed

func (b Builder) TsGet() (c TsGet) {
	c = TsGet{cs: get(), ks: b.ks, cf: readonly}
	c.cs.s = append(c.cs.s, "TS.GET")
	return c
}

func (c TsGet) Key(key string) TsGetKey {
	if c.ks&NoSlot == NoSlot {
		c.ks = NoSlot | slot(key)
	} else {
		c.ks = check(c.ks, slot(key))
	}
	c.cs.s = append(c.cs.s, key)
	return (TsGetKey)(c)
}

type TsGetKey Completed

func (c TsGetKey) Latest() TsGetLatest {
	c.cs.s = append(c.cs.s, "LATEST")
	return (TsGetLatest)(c)
}

func (c TsGetKey) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type TsGetLatest Completed

func (c TsGetLatest) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type TsIncrby Completed

func (b Builder) TsIncrby() (c TsIncrby) {
	c = TsIncrby{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "TS.INCRBY")
	return c
}

func (c TsIncrby) Key(key string) TsIncrbyKey {
	if c.ks&NoSlot == NoSlot {
		c.ks = NoSlot | slot(key)
	} else {
		c.ks = check(c.ks, slot(key))
	}
	c.cs.s = append(c.cs.s, key)
	return (TsIncrbyKey)(c)
}

type TsIncrbyChunkSize Completed

func (c TsIncrbyChunkSize) Labels() TsIncrbyLabels {
	c.cs.s = append(c.cs.s, "LABELS")
	return (TsIncrbyLabels)(c)
}

func (c TsIncrbyChunkSize) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type TsIncrbyKey Completed

func (c TsIncrbyKey) Value(value float64) TsIncrbyValue {
	c.cs.s = append(c.cs.s, strconv.FormatFloat(value, 'f', -1, 64))
	return (TsIncrbyValue)(c)
}

type TsIncrbyLabels Completed

func (c TsIncrbyLabels) Labels(label string, value string) TsIncrbyLabels {
	c.cs.s = append(c.cs.s, label, value)
	return c
}

func (c TsIncrbyLabels) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type TsIncrbyRetention Completed

func (c TsIncrbyRetention) Uncompressed() TsIncrbyUncompressed {
	c.cs.s = append(c.cs.s, "UNCOMPRESSED")
	return (TsIncrbyUncompressed)(c)
}

func (c TsIncrbyRetention) ChunkSize(size int64) TsIncrbyChunkSize {
	c.cs.s = append(c.cs.s, "CHUNK_SIZE", strconv.FormatInt(size, 10))
	return (TsIncrbyChunkSize)(c)
}

func (c TsIncrbyRetention) Labels() TsIncrbyLabels {
	c.cs.s = append(c.cs.s, "LABELS")
	return (TsIncrbyLabels)(c)
}

func (c TsIncrbyRetention) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type TsIncrbyTimestamp Completed

func (c TsIncrbyTimestamp) Retention(retentionperiod int64) TsIncrbyRetention {
	c.cs.s = append(c.cs.s, "RETENTION", strconv.FormatInt(retentionperiod, 10))
	return (TsIncrbyRetention)(c)
}

func (c TsIncrbyTimestamp) Uncompressed() TsIncrbyUncompressed {
	c.cs.s = append(c.cs.s, "UNCOMPRESSED")
	return (TsIncrbyUncompressed)(c)
}

func (c TsIncrbyTimestamp) ChunkSize(size int64) TsIncrbyChunkSize {
	c.cs.s = append(c.cs.s, "CHUNK_SIZE", strconv.FormatInt(size, 10))
	return (TsIncrbyChunkSize)(c)
}

func (c TsIncrbyTimestamp) Labels() TsIncrbyLabels {
	c.cs.s = append(c.cs.s, "LABELS")
	return (TsIncrbyLabels)(c)
}

func (c TsIncrbyTimestamp) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type TsIncrbyUncompressed Completed

func (c TsIncrbyUncompressed) ChunkSize(size int64) TsIncrbyChunkSize {
	c.cs.s = append(c.cs.s, "CHUNK_SIZE", strconv.FormatInt(size, 10))
	return (TsIncrbyChunkSize)(c)
}

func (c TsIncrbyUncompressed) Labels() TsIncrbyLabels {
	c.cs.s = append(c.cs.s, "LABELS")
	return (TsIncrbyLabels)(c)
}

func (c TsIncrbyUncompressed) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type TsIncrbyValue Completed

func (c TsIncrbyValue) Timestamp(timestamp int64) TsIncrbyTimestamp {
	c.cs.s = append(c.cs.s, "TIMESTAMP", strconv.FormatInt(timestamp, 10))
	return (TsIncrbyTimestamp)(c)
}

func (c TsIncrbyValue) Retention(retentionperiod int64) TsIncrbyRetention {
	c.cs.s = append(c.cs.s, "RETENTION", strconv.FormatInt(retentionperiod, 10))
	return (TsIncrbyRetention)(c)
}

func (c TsIncrbyValue) Uncompressed() TsIncrbyUncompressed {
	c.cs.s = append(c.cs.s, "UNCOMPRESSED")
	return (TsIncrbyUncompressed)(c)
}

func (c TsIncrbyValue) ChunkSize(size int64) TsIncrbyChunkSize {
	c.cs.s = append(c.cs.s, "CHUNK_SIZE", strconv.FormatInt(size, 10))
	return (TsIncrbyChunkSize)(c)
}

func (c TsIncrbyValue) Labels() TsIncrbyLabels {
	c.cs.s = append(c.cs.s, "LABELS")
	return (TsIncrbyLabels)(c)
}

func (c TsIncrbyValue) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type TsInfo Completed

func (b Builder) TsInfo() (c TsInfo) {
	c = TsInfo{cs: get(), ks: b.ks, cf: readonly}
	c.cs.s = append(c.cs.s, "TS.INFO")
	return c
}

func (c TsInfo) Key(key string) TsInfoKey {
	if c.ks&NoSlot == NoSlot {
		c.ks = NoSlot | slot(key)
	} else {
		c.ks = check(c.ks, slot(key))
	}
	c.cs.s = append(c.cs.s, key)
	return (TsInfoKey)(c)
}

type TsInfoDebug Completed

func (c TsInfoDebug) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type TsInfoKey Completed

func (c TsInfoKey) Debug(debug string) TsInfoDebug {
	c.cs.s = append(c.cs.s, debug)
	return (TsInfoDebug)(c)
}

func (c TsInfoKey) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type TsMadd Completed

func (b Builder) TsMadd() (c TsMadd) {
	c = TsMadd{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "TS.MADD")
	return c
}

func (c TsMadd) KeyTimestampValue() TsMaddKeyTimestampValue {
	return (TsMaddKeyTimestampValue)(c)
}

type TsMaddKeyTimestampValue Completed

func (c TsMaddKeyTimestampValue) KeyTimestampValue(key string, timestamp int64, value float64) TsMaddKeyTimestampValue {
	if c.ks&NoSlot == NoSlot {
		c.ks = NoSlot | slot(key)
	} else {
		c.ks = check(c.ks, slot(key))
	}
	c.cs.s = append(c.cs.s, key, strconv.FormatInt(timestamp, 10), strconv.FormatFloat(value, 'f', -1, 64))
	return c
}

func (c TsMaddKeyTimestampValue) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type TsMget Completed

func (b Builder) TsMget() (c TsMget) {
	c = TsMget{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "TS.MGET")
	return c
}

func (c TsMget) Latest() TsMgetLatest {
	c.cs.s = append(c.cs.s, "LATEST")
	return (TsMgetLatest)(c)
}

func (c TsMget) Withlabels() TsMgetWithlabels {
	c.cs.s = append(c.cs.s, "WITHLABELS")
	return (TsMgetWithlabels)(c)
}

func (c TsMget) SelectedLabels(labels []string) TsMgetSelectedLabels {
	c.cs.s = append(c.cs.s, "SELECTED_LABELS")
	c.cs.s = append(c.cs.s, labels...)
	return (TsMgetSelectedLabels)(c)
}

func (c TsMget) Filter(filter ...string) TsMgetFilter {
	c.cs.s = append(c.cs.s, "FILTER")
	c.cs.s = append(c.cs.s, filter...)
	return (TsMgetFilter)(c)
}

type TsMgetFilter Completed

func (c TsMgetFilter) Filter(filter ...string) TsMgetFilter {
	c.cs.s = append(c.cs.s, "FILTER")
	c.cs.s = append(c.cs.s, filter...)
	return c
}

func (c TsMgetFilter) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type TsMgetLatest Completed

func (c TsMgetLatest) Withlabels() TsMgetWithlabels {
	c.cs.s = append(c.cs.s, "WITHLABELS")
	return (TsMgetWithlabels)(c)
}

func (c TsMgetLatest) SelectedLabels(labels []string) TsMgetSelectedLabels {
	c.cs.s = append(c.cs.s, "SELECTED_LABELS")
	c.cs.s = append(c.cs.s, labels...)
	return (TsMgetSelectedLabels)(c)
}

func (c TsMgetLatest) Filter(filter ...string) TsMgetFilter {
	c.cs.s = append(c.cs.s, "FILTER")
	c.cs.s = append(c.cs.s, filter...)
	return (TsMgetFilter)(c)
}

type TsMgetSelectedLabels Completed

func (c TsMgetSelectedLabels) Filter(filter ...string) TsMgetFilter {
	c.cs.s = append(c.cs.s, "FILTER")
	c.cs.s = append(c.cs.s, filter...)
	return (TsMgetFilter)(c)
}

type TsMgetWithlabels Completed

func (c TsMgetWithlabels) Filter(filter ...string) TsMgetFilter {
	c.cs.s = append(c.cs.s, "FILTER")
	c.cs.s = append(c.cs.s, filter...)
	return (TsMgetFilter)(c)
}

type TsMrange Completed

func (b Builder) TsMrange() (c TsMrange) {
	c = TsMrange{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "TS.MRANGE")
	return c
}

func (c TsMrange) Fromtimestamp(fromtimestamp int64) TsMrangeFromtimestamp {
	c.cs.s = append(c.cs.s, strconv.FormatInt(fromtimestamp, 10))
	return (TsMrangeFromtimestamp)(c)
}

type TsMrangeAggregationAggregationAvg Completed

func (c TsMrangeAggregationAggregationAvg) Bucketduration(bucketduration int64) TsMrangeAggregationBucketduration {
	c.cs.s = append(c.cs.s, strconv.FormatInt(bucketduration, 10))
	return (TsMrangeAggregationBucketduration)(c)
}

type TsMrangeAggregationAggregationCount Completed

func (c TsMrangeAggregationAggregationCount) Bucketduration(bucketduration int64) TsMrangeAggregationBucketduration {
	c.cs.s = append(c.cs.s, strconv.FormatInt(bucketduration, 10))
	return (TsMrangeAggregationBucketduration)(c)
}

type TsMrangeAggregationAggregationFirst Completed

func (c TsMrangeAggregationAggregationFirst) Bucketduration(bucketduration int64) TsMrangeAggregationBucketduration {
	c.cs.s = append(c.cs.s, strconv.FormatInt(bucketduration, 10))
	return (TsMrangeAggregationBucketduration)(c)
}

type TsMrangeAggregationAggregationLast Completed

func (c TsMrangeAggregationAggregationLast) Bucketduration(bucketduration int64) TsMrangeAggregationBucketduration {
	c.cs.s = append(c.cs.s, strconv.FormatInt(bucketduration, 10))
	return (TsMrangeAggregationBucketduration)(c)
}

type TsMrangeAggregationAggregationMax Completed

func (c TsMrangeAggregationAggregationMax) Bucketduration(bucketduration int64) TsMrangeAggregationBucketduration {
	c.cs.s = append(c.cs.s, strconv.FormatInt(bucketduration, 10))
	return (TsMrangeAggregationBucketduration)(c)
}

type TsMrangeAggregationAggregationMin Completed

func (c TsMrangeAggregationAggregationMin) Bucketduration(bucketduration int64) TsMrangeAggregationBucketduration {
	c.cs.s = append(c.cs.s, strconv.FormatInt(bucketduration, 10))
	return (TsMrangeAggregationBucketduration)(c)
}

type TsMrangeAggregationAggregationRange Completed

func (c TsMrangeAggregationAggregationRange) Bucketduration(bucketduration int64) TsMrangeAggregationBucketduration {
	c.cs.s = append(c.cs.s, strconv.FormatInt(bucketduration, 10))
	return (TsMrangeAggregationBucketduration)(c)
}

type TsMrangeAggregationAggregationStdP Completed

func (c TsMrangeAggregationAggregationStdP) Bucketduration(bucketduration int64) TsMrangeAggregationBucketduration {
	c.cs.s = append(c.cs.s, strconv.FormatInt(bucketduration, 10))
	return (TsMrangeAggregationBucketduration)(c)
}

type TsMrangeAggregationAggregationStdS Completed

func (c TsMrangeAggregationAggregationStdS) Bucketduration(bucketduration int64) TsMrangeAggregationBucketduration {
	c.cs.s = append(c.cs.s, strconv.FormatInt(bucketduration, 10))
	return (TsMrangeAggregationBucketduration)(c)
}

type TsMrangeAggregationAggregationSum Completed

func (c TsMrangeAggregationAggregationSum) Bucketduration(bucketduration int64) TsMrangeAggregationBucketduration {
	c.cs.s = append(c.cs.s, strconv.FormatInt(bucketduration, 10))
	return (TsMrangeAggregationBucketduration)(c)
}

type TsMrangeAggregationAggregationTwa Completed

func (c TsMrangeAggregationAggregationTwa) Bucketduration(bucketduration int64) TsMrangeAggregationBucketduration {
	c.cs.s = append(c.cs.s, strconv.FormatInt(bucketduration, 10))
	return (TsMrangeAggregationBucketduration)(c)
}

type TsMrangeAggregationAggregationVarP Completed

func (c TsMrangeAggregationAggregationVarP) Bucketduration(bucketduration int64) TsMrangeAggregationBucketduration {
	c.cs.s = append(c.cs.s, strconv.FormatInt(bucketduration, 10))
	return (TsMrangeAggregationBucketduration)(c)
}

type TsMrangeAggregationAggregationVarS Completed

func (c TsMrangeAggregationAggregationVarS) Bucketduration(bucketduration int64) TsMrangeAggregationBucketduration {
	c.cs.s = append(c.cs.s, strconv.FormatInt(bucketduration, 10))
	return (TsMrangeAggregationBucketduration)(c)
}

type TsMrangeAggregationBucketduration Completed

func (c TsMrangeAggregationBucketduration) Buckettimestamp(buckettimestamp string) TsMrangeAggregationBuckettimestamp {
	c.cs.s = append(c.cs.s, "BUCKETTIMESTAMP", buckettimestamp)
	return (TsMrangeAggregationBuckettimestamp)(c)
}

func (c TsMrangeAggregationBucketduration) Empty() TsMrangeAggregationEmpty {
	c.cs.s = append(c.cs.s, "EMPTY")
	return (TsMrangeAggregationEmpty)(c)
}

func (c TsMrangeAggregationBucketduration) Filter(filter ...string) TsMrangeFilter {
	c.cs.s = append(c.cs.s, "FILTER")
	c.cs.s = append(c.cs.s, filter...)
	return (TsMrangeFilter)(c)
}

type TsMrangeAggregationBuckettimestamp Completed

func (c TsMrangeAggregationBuckettimestamp) Empty() TsMrangeAggregationEmpty {
	c.cs.s = append(c.cs.s, "EMPTY")
	return (TsMrangeAggregationEmpty)(c)
}

func (c TsMrangeAggregationBuckettimestamp) Filter(filter ...string) TsMrangeFilter {
	c.cs.s = append(c.cs.s, "FILTER")
	c.cs.s = append(c.cs.s, filter...)
	return (TsMrangeFilter)(c)
}

type TsMrangeAggregationEmpty Completed

func (c TsMrangeAggregationEmpty) Filter(filter ...string) TsMrangeFilter {
	c.cs.s = append(c.cs.s, "FILTER")
	c.cs.s = append(c.cs.s, filter...)
	return (TsMrangeFilter)(c)
}

type TsMrangeAlign Completed

func (c TsMrangeAlign) AggregationAvg() TsMrangeAggregationAggregationAvg {
	c.cs.s = append(c.cs.s, "AGGREGATION", "AVG")
	return (TsMrangeAggregationAggregationAvg)(c)
}

func (c TsMrangeAlign) AggregationSum() TsMrangeAggregationAggregationSum {
	c.cs.s = append(c.cs.s, "AGGREGATION", "SUM")
	return (TsMrangeAggregationAggregationSum)(c)
}

func (c TsMrangeAlign) AggregationMin() TsMrangeAggregationAggregationMin {
	c.cs.s = append(c.cs.s, "AGGREGATION", "MIN")
	return (TsMrangeAggregationAggregationMin)(c)
}

func (c TsMrangeAlign) AggregationMax() TsMrangeAggregationAggregationMax {
	c.cs.s = append(c.cs.s, "AGGREGATION", "MAX")
	return (TsMrangeAggregationAggregationMax)(c)
}

func (c TsMrangeAlign) AggregationRange() TsMrangeAggregationAggregationRange {
	c.cs.s = append(c.cs.s, "AGGREGATION", "RANGE")
	return (TsMrangeAggregationAggregationRange)(c)
}

func (c TsMrangeAlign) AggregationCount() TsMrangeAggregationAggregationCount {
	c.cs.s = append(c.cs.s, "AGGREGATION", "COUNT")
	return (TsMrangeAggregationAggregationCount)(c)
}

func (c TsMrangeAlign) AggregationFirst() TsMrangeAggregationAggregationFirst {
	c.cs.s = append(c.cs.s, "AGGREGATION", "FIRST")
	return (TsMrangeAggregationAggregationFirst)(c)
}

func (c TsMrangeAlign) AggregationLast() TsMrangeAggregationAggregationLast {
	c.cs.s = append(c.cs.s, "AGGREGATION", "LAST")
	return (TsMrangeAggregationAggregationLast)(c)
}

func (c TsMrangeAlign) AggregationStdP() TsMrangeAggregationAggregationStdP {
	c.cs.s = append(c.cs.s, "AGGREGATION", "STD.P")
	return (TsMrangeAggregationAggregationStdP)(c)
}

func (c TsMrangeAlign) AggregationStdS() TsMrangeAggregationAggregationStdS {
	c.cs.s = append(c.cs.s, "AGGREGATION", "STD.S")
	return (TsMrangeAggregationAggregationStdS)(c)
}

func (c TsMrangeAlign) AggregationVarP() TsMrangeAggregationAggregationVarP {
	c.cs.s = append(c.cs.s, "AGGREGATION", "VAR.P")
	return (TsMrangeAggregationAggregationVarP)(c)
}

func (c TsMrangeAlign) AggregationVarS() TsMrangeAggregationAggregationVarS {
	c.cs.s = append(c.cs.s, "AGGREGATION", "VAR.S")
	return (TsMrangeAggregationAggregationVarS)(c)
}

func (c TsMrangeAlign) AggregationTwa() TsMrangeAggregationAggregationTwa {
	c.cs.s = append(c.cs.s, "AGGREGATION", "TWA")
	return (TsMrangeAggregationAggregationTwa)(c)
}

func (c TsMrangeAlign) Filter(filter ...string) TsMrangeFilter {
	c.cs.s = append(c.cs.s, "FILTER")
	c.cs.s = append(c.cs.s, filter...)
	return (TsMrangeFilter)(c)
}

type TsMrangeCount Completed

func (c TsMrangeCount) Align(value string) TsMrangeAlign {
	c.cs.s = append(c.cs.s, "ALIGN", value)
	return (TsMrangeAlign)(c)
}

func (c TsMrangeCount) AggregationAvg() TsMrangeAggregationAggregationAvg {
	c.cs.s = append(c.cs.s, "AGGREGATION", "AVG")
	return (TsMrangeAggregationAggregationAvg)(c)
}

func (c TsMrangeCount) AggregationSum() TsMrangeAggregationAggregationSum {
	c.cs.s = append(c.cs.s, "AGGREGATION", "SUM")
	return (TsMrangeAggregationAggregationSum)(c)
}

func (c TsMrangeCount) AggregationMin() TsMrangeAggregationAggregationMin {
	c.cs.s = append(c.cs.s, "AGGREGATION", "MIN")
	return (TsMrangeAggregationAggregationMin)(c)
}

func (c TsMrangeCount) AggregationMax() TsMrangeAggregationAggregationMax {
	c.cs.s = append(c.cs.s, "AGGREGATION", "MAX")
	return (TsMrangeAggregationAggregationMax)(c)
}

func (c TsMrangeCount) AggregationRange() TsMrangeAggregationAggregationRange {
	c.cs.s = append(c.cs.s, "AGGREGATION", "RANGE")
	return (TsMrangeAggregationAggregationRange)(c)
}

func (c TsMrangeCount) AggregationCount() TsMrangeAggregationAggregationCount {
	c.cs.s = append(c.cs.s, "AGGREGATION", "COUNT")
	return (TsMrangeAggregationAggregationCount)(c)
}

func (c TsMrangeCount) AggregationFirst() TsMrangeAggregationAggregationFirst {
	c.cs.s = append(c.cs.s, "AGGREGATION", "FIRST")
	return (TsMrangeAggregationAggregationFirst)(c)
}

func (c TsMrangeCount) AggregationLast() TsMrangeAggregationAggregationLast {
	c.cs.s = append(c.cs.s, "AGGREGATION", "LAST")
	return (TsMrangeAggregationAggregationLast)(c)
}

func (c TsMrangeCount) AggregationStdP() TsMrangeAggregationAggregationStdP {
	c.cs.s = append(c.cs.s, "AGGREGATION", "STD.P")
	return (TsMrangeAggregationAggregationStdP)(c)
}

func (c TsMrangeCount) AggregationStdS() TsMrangeAggregationAggregationStdS {
	c.cs.s = append(c.cs.s, "AGGREGATION", "STD.S")
	return (TsMrangeAggregationAggregationStdS)(c)
}

func (c TsMrangeCount) AggregationVarP() TsMrangeAggregationAggregationVarP {
	c.cs.s = append(c.cs.s, "AGGREGATION", "VAR.P")
	return (TsMrangeAggregationAggregationVarP)(c)
}

func (c TsMrangeCount) AggregationVarS() TsMrangeAggregationAggregationVarS {
	c.cs.s = append(c.cs.s, "AGGREGATION", "VAR.S")
	return (TsMrangeAggregationAggregationVarS)(c)
}

func (c TsMrangeCount) AggregationTwa() TsMrangeAggregationAggregationTwa {
	c.cs.s = append(c.cs.s, "AGGREGATION", "TWA")
	return (TsMrangeAggregationAggregationTwa)(c)
}

func (c TsMrangeCount) Filter(filter ...string) TsMrangeFilter {
	c.cs.s = append(c.cs.s, "FILTER")
	c.cs.s = append(c.cs.s, filter...)
	return (TsMrangeFilter)(c)
}

type TsMrangeFilter Completed

func (c TsMrangeFilter) Filter(filter ...string) TsMrangeFilter {
	c.cs.s = append(c.cs.s, "FILTER")
	c.cs.s = append(c.cs.s, filter...)
	return c
}

func (c TsMrangeFilter) Groupby(label string, reduce string, reducer string) TsMrangeGroupby {
	c.cs.s = append(c.cs.s, "GROUPBY", label, reduce, reducer)
	return (TsMrangeGroupby)(c)
}

func (c TsMrangeFilter) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type TsMrangeFilterByTs Completed

func (c TsMrangeFilterByTs) FilterByTs(timestamp ...int64) TsMrangeFilterByTs {
	c.cs.s = append(c.cs.s, "FILTER_BY_TS")
	for _, n := range timestamp {
		c.cs.s = append(c.cs.s, strconv.FormatInt(n, 10))
	}
	return c
}

func (c TsMrangeFilterByTs) FilterByValue(min float64, max float64) TsMrangeFilterByValue {
	c.cs.s = append(c.cs.s, "FILTER_BY_VALUE", strconv.FormatFloat(min, 'f', -1, 64), strconv.FormatFloat(max, 'f', -1, 64))
	return (TsMrangeFilterByValue)(c)
}

func (c TsMrangeFilterByTs) Withlabels() TsMrangeWithlabels {
	c.cs.s = append(c.cs.s, "WITHLABELS")
	return (TsMrangeWithlabels)(c)
}

func (c TsMrangeFilterByTs) SelectedLabels(labels []string) TsMrangeSelectedLabels {
	c.cs.s = append(c.cs.s, "SELECTED_LABELS")
	c.cs.s = append(c.cs.s, labels...)
	return (TsMrangeSelectedLabels)(c)
}

func (c TsMrangeFilterByTs) Count(count int64) TsMrangeCount {
	c.cs.s = append(c.cs.s, "COUNT", strconv.FormatInt(count, 10))
	return (TsMrangeCount)(c)
}

func (c TsMrangeFilterByTs) Align(value string) TsMrangeAlign {
	c.cs.s = append(c.cs.s, "ALIGN", value)
	return (TsMrangeAlign)(c)
}

func (c TsMrangeFilterByTs) AggregationAvg() TsMrangeAggregationAggregationAvg {
	c.cs.s = append(c.cs.s, "AGGREGATION", "AVG")
	return (TsMrangeAggregationAggregationAvg)(c)
}

func (c TsMrangeFilterByTs) AggregationSum() TsMrangeAggregationAggregationSum {
	c.cs.s = append(c.cs.s, "AGGREGATION", "SUM")
	return (TsMrangeAggregationAggregationSum)(c)
}

func (c TsMrangeFilterByTs) AggregationMin() TsMrangeAggregationAggregationMin {
	c.cs.s = append(c.cs.s, "AGGREGATION", "MIN")
	return (TsMrangeAggregationAggregationMin)(c)
}

func (c TsMrangeFilterByTs) AggregationMax() TsMrangeAggregationAggregationMax {
	c.cs.s = append(c.cs.s, "AGGREGATION", "MAX")
	return (TsMrangeAggregationAggregationMax)(c)
}

func (c TsMrangeFilterByTs) AggregationRange() TsMrangeAggregationAggregationRange {
	c.cs.s = append(c.cs.s, "AGGREGATION", "RANGE")
	return (TsMrangeAggregationAggregationRange)(c)
}

func (c TsMrangeFilterByTs) AggregationCount() TsMrangeAggregationAggregationCount {
	c.cs.s = append(c.cs.s, "AGGREGATION", "COUNT")
	return (TsMrangeAggregationAggregationCount)(c)
}

func (c TsMrangeFilterByTs) AggregationFirst() TsMrangeAggregationAggregationFirst {
	c.cs.s = append(c.cs.s, "AGGREGATION", "FIRST")
	return (TsMrangeAggregationAggregationFirst)(c)
}

func (c TsMrangeFilterByTs) AggregationLast() TsMrangeAggregationAggregationLast {
	c.cs.s = append(c.cs.s, "AGGREGATION", "LAST")
	return (TsMrangeAggregationAggregationLast)(c)
}

func (c TsMrangeFilterByTs) AggregationStdP() TsMrangeAggregationAggregationStdP {
	c.cs.s = append(c.cs.s, "AGGREGATION", "STD.P")
	return (TsMrangeAggregationAggregationStdP)(c)
}

func (c TsMrangeFilterByTs) AggregationStdS() TsMrangeAggregationAggregationStdS {
	c.cs.s = append(c.cs.s, "AGGREGATION", "STD.S")
	return (TsMrangeAggregationAggregationStdS)(c)
}

func (c TsMrangeFilterByTs) AggregationVarP() TsMrangeAggregationAggregationVarP {
	c.cs.s = append(c.cs.s, "AGGREGATION", "VAR.P")
	return (TsMrangeAggregationAggregationVarP)(c)
}

func (c TsMrangeFilterByTs) AggregationVarS() TsMrangeAggregationAggregationVarS {
	c.cs.s = append(c.cs.s, "AGGREGATION", "VAR.S")
	return (TsMrangeAggregationAggregationVarS)(c)
}

func (c TsMrangeFilterByTs) AggregationTwa() TsMrangeAggregationAggregationTwa {
	c.cs.s = append(c.cs.s, "AGGREGATION", "TWA")
	return (TsMrangeAggregationAggregationTwa)(c)
}

func (c TsMrangeFilterByTs) Filter(filter ...string) TsMrangeFilter {
	c.cs.s = append(c.cs.s, "FILTER")
	c.cs.s = append(c.cs.s, filter...)
	return (TsMrangeFilter)(c)
}

type TsMrangeFilterByValue Completed

func (c TsMrangeFilterByValue) Withlabels() TsMrangeWithlabels {
	c.cs.s = append(c.cs.s, "WITHLABELS")
	return (TsMrangeWithlabels)(c)
}

func (c TsMrangeFilterByValue) SelectedLabels(labels []string) TsMrangeSelectedLabels {
	c.cs.s = append(c.cs.s, "SELECTED_LABELS")
	c.cs.s = append(c.cs.s, labels...)
	return (TsMrangeSelectedLabels)(c)
}

func (c TsMrangeFilterByValue) Count(count int64) TsMrangeCount {
	c.cs.s = append(c.cs.s, "COUNT", strconv.FormatInt(count, 10))
	return (TsMrangeCount)(c)
}

func (c TsMrangeFilterByValue) Align(value string) TsMrangeAlign {
	c.cs.s = append(c.cs.s, "ALIGN", value)
	return (TsMrangeAlign)(c)
}

func (c TsMrangeFilterByValue) AggregationAvg() TsMrangeAggregationAggregationAvg {
	c.cs.s = append(c.cs.s, "AGGREGATION", "AVG")
	return (TsMrangeAggregationAggregationAvg)(c)
}

func (c TsMrangeFilterByValue) AggregationSum() TsMrangeAggregationAggregationSum {
	c.cs.s = append(c.cs.s, "AGGREGATION", "SUM")
	return (TsMrangeAggregationAggregationSum)(c)
}

func (c TsMrangeFilterByValue) AggregationMin() TsMrangeAggregationAggregationMin {
	c.cs.s = append(c.cs.s, "AGGREGATION", "MIN")
	return (TsMrangeAggregationAggregationMin)(c)
}

func (c TsMrangeFilterByValue) AggregationMax() TsMrangeAggregationAggregationMax {
	c.cs.s = append(c.cs.s, "AGGREGATION", "MAX")
	return (TsMrangeAggregationAggregationMax)(c)
}

func (c TsMrangeFilterByValue) AggregationRange() TsMrangeAggregationAggregationRange {
	c.cs.s = append(c.cs.s, "AGGREGATION", "RANGE")
	return (TsMrangeAggregationAggregationRange)(c)
}

func (c TsMrangeFilterByValue) AggregationCount() TsMrangeAggregationAggregationCount {
	c.cs.s = append(c.cs.s, "AGGREGATION", "COUNT")
	return (TsMrangeAggregationAggregationCount)(c)
}

func (c TsMrangeFilterByValue) AggregationFirst() TsMrangeAggregationAggregationFirst {
	c.cs.s = append(c.cs.s, "AGGREGATION", "FIRST")
	return (TsMrangeAggregationAggregationFirst)(c)
}

func (c TsMrangeFilterByValue) AggregationLast() TsMrangeAggregationAggregationLast {
	c.cs.s = append(c.cs.s, "AGGREGATION", "LAST")
	return (TsMrangeAggregationAggregationLast)(c)
}

func (c TsMrangeFilterByValue) AggregationStdP() TsMrangeAggregationAggregationStdP {
	c.cs.s = append(c.cs.s, "AGGREGATION", "STD.P")
	return (TsMrangeAggregationAggregationStdP)(c)
}

func (c TsMrangeFilterByValue) AggregationStdS() TsMrangeAggregationAggregationStdS {
	c.cs.s = append(c.cs.s, "AGGREGATION", "STD.S")
	return (TsMrangeAggregationAggregationStdS)(c)
}

func (c TsMrangeFilterByValue) AggregationVarP() TsMrangeAggregationAggregationVarP {
	c.cs.s = append(c.cs.s, "AGGREGATION", "VAR.P")
	return (TsMrangeAggregationAggregationVarP)(c)
}

func (c TsMrangeFilterByValue) AggregationVarS() TsMrangeAggregationAggregationVarS {
	c.cs.s = append(c.cs.s, "AGGREGATION", "VAR.S")
	return (TsMrangeAggregationAggregationVarS)(c)
}

func (c TsMrangeFilterByValue) AggregationTwa() TsMrangeAggregationAggregationTwa {
	c.cs.s = append(c.cs.s, "AGGREGATION", "TWA")
	return (TsMrangeAggregationAggregationTwa)(c)
}

func (c TsMrangeFilterByValue) Filter(filter ...string) TsMrangeFilter {
	c.cs.s = append(c.cs.s, "FILTER")
	c.cs.s = append(c.cs.s, filter...)
	return (TsMrangeFilter)(c)
}

type TsMrangeFromtimestamp Completed

func (c TsMrangeFromtimestamp) Totimestamp(totimestamp int64) TsMrangeTotimestamp {
	c.cs.s = append(c.cs.s, strconv.FormatInt(totimestamp, 10))
	return (TsMrangeTotimestamp)(c)
}

type TsMrangeGroupby Completed

func (c TsMrangeGroupby) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type TsMrangeLatest Completed

func (c TsMrangeLatest) FilterByTs(timestamp ...int64) TsMrangeFilterByTs {
	c.cs.s = append(c.cs.s, "FILTER_BY_TS")
	for _, n := range timestamp {
		c.cs.s = append(c.cs.s, strconv.FormatInt(n, 10))
	}
	return (TsMrangeFilterByTs)(c)
}

func (c TsMrangeLatest) FilterByValue(min float64, max float64) TsMrangeFilterByValue {
	c.cs.s = append(c.cs.s, "FILTER_BY_VALUE", strconv.FormatFloat(min, 'f', -1, 64), strconv.FormatFloat(max, 'f', -1, 64))
	return (TsMrangeFilterByValue)(c)
}

func (c TsMrangeLatest) Withlabels() TsMrangeWithlabels {
	c.cs.s = append(c.cs.s, "WITHLABELS")
	return (TsMrangeWithlabels)(c)
}

func (c TsMrangeLatest) SelectedLabels(labels []string) TsMrangeSelectedLabels {
	c.cs.s = append(c.cs.s, "SELECTED_LABELS")
	c.cs.s = append(c.cs.s, labels...)
	return (TsMrangeSelectedLabels)(c)
}

func (c TsMrangeLatest) Count(count int64) TsMrangeCount {
	c.cs.s = append(c.cs.s, "COUNT", strconv.FormatInt(count, 10))
	return (TsMrangeCount)(c)
}

func (c TsMrangeLatest) Align(value string) TsMrangeAlign {
	c.cs.s = append(c.cs.s, "ALIGN", value)
	return (TsMrangeAlign)(c)
}

func (c TsMrangeLatest) AggregationAvg() TsMrangeAggregationAggregationAvg {
	c.cs.s = append(c.cs.s, "AGGREGATION", "AVG")
	return (TsMrangeAggregationAggregationAvg)(c)
}

func (c TsMrangeLatest) AggregationSum() TsMrangeAggregationAggregationSum {
	c.cs.s = append(c.cs.s, "AGGREGATION", "SUM")
	return (TsMrangeAggregationAggregationSum)(c)
}

func (c TsMrangeLatest) AggregationMin() TsMrangeAggregationAggregationMin {
	c.cs.s = append(c.cs.s, "AGGREGATION", "MIN")
	return (TsMrangeAggregationAggregationMin)(c)
}

func (c TsMrangeLatest) AggregationMax() TsMrangeAggregationAggregationMax {
	c.cs.s = append(c.cs.s, "AGGREGATION", "MAX")
	return (TsMrangeAggregationAggregationMax)(c)
}

func (c TsMrangeLatest) AggregationRange() TsMrangeAggregationAggregationRange {
	c.cs.s = append(c.cs.s, "AGGREGATION", "RANGE")
	return (TsMrangeAggregationAggregationRange)(c)
}

func (c TsMrangeLatest) AggregationCount() TsMrangeAggregationAggregationCount {
	c.cs.s = append(c.cs.s, "AGGREGATION", "COUNT")
	return (TsMrangeAggregationAggregationCount)(c)
}

func (c TsMrangeLatest) AggregationFirst() TsMrangeAggregationAggregationFirst {
	c.cs.s = append(c.cs.s, "AGGREGATION", "FIRST")
	return (TsMrangeAggregationAggregationFirst)(c)
}

func (c TsMrangeLatest) AggregationLast() TsMrangeAggregationAggregationLast {
	c.cs.s = append(c.cs.s, "AGGREGATION", "LAST")
	return (TsMrangeAggregationAggregationLast)(c)
}

func (c TsMrangeLatest) AggregationStdP() TsMrangeAggregationAggregationStdP {
	c.cs.s = append(c.cs.s, "AGGREGATION", "STD.P")
	return (TsMrangeAggregationAggregationStdP)(c)
}

func (c TsMrangeLatest) AggregationStdS() TsMrangeAggregationAggregationStdS {
	c.cs.s = append(c.cs.s, "AGGREGATION", "STD.S")
	return (TsMrangeAggregationAggregationStdS)(c)
}

func (c TsMrangeLatest) AggregationVarP() TsMrangeAggregationAggregationVarP {
	c.cs.s = append(c.cs.s, "AGGREGATION", "VAR.P")
	return (TsMrangeAggregationAggregationVarP)(c)
}

func (c TsMrangeLatest) AggregationVarS() TsMrangeAggregationAggregationVarS {
	c.cs.s = append(c.cs.s, "AGGREGATION", "VAR.S")
	return (TsMrangeAggregationAggregationVarS)(c)
}

func (c TsMrangeLatest) AggregationTwa() TsMrangeAggregationAggregationTwa {
	c.cs.s = append(c.cs.s, "AGGREGATION", "TWA")
	return (TsMrangeAggregationAggregationTwa)(c)
}

func (c TsMrangeLatest) Filter(filter ...string) TsMrangeFilter {
	c.cs.s = append(c.cs.s, "FILTER")
	c.cs.s = append(c.cs.s, filter...)
	return (TsMrangeFilter)(c)
}

type TsMrangeSelectedLabels Completed

func (c TsMrangeSelectedLabels) Count(count int64) TsMrangeCount {
	c.cs.s = append(c.cs.s, "COUNT", strconv.FormatInt(count, 10))
	return (TsMrangeCount)(c)
}

func (c TsMrangeSelectedLabels) Align(value string) TsMrangeAlign {
	c.cs.s = append(c.cs.s, "ALIGN", value)
	return (TsMrangeAlign)(c)
}

func (c TsMrangeSelectedLabels) AggregationAvg() TsMrangeAggregationAggregationAvg {
	c.cs.s = append(c.cs.s, "AGGREGATION", "AVG")
	return (TsMrangeAggregationAggregationAvg)(c)
}

func (c TsMrangeSelectedLabels) AggregationSum() TsMrangeAggregationAggregationSum {
	c.cs.s = append(c.cs.s, "AGGREGATION", "SUM")
	return (TsMrangeAggregationAggregationSum)(c)
}

func (c TsMrangeSelectedLabels) AggregationMin() TsMrangeAggregationAggregationMin {
	c.cs.s = append(c.cs.s, "AGGREGATION", "MIN")
	return (TsMrangeAggregationAggregationMin)(c)
}

func (c TsMrangeSelectedLabels) AggregationMax() TsMrangeAggregationAggregationMax {
	c.cs.s = append(c.cs.s, "AGGREGATION", "MAX")
	return (TsMrangeAggregationAggregationMax)(c)
}

func (c TsMrangeSelectedLabels) AggregationRange() TsMrangeAggregationAggregationRange {
	c.cs.s = append(c.cs.s, "AGGREGATION", "RANGE")
	return (TsMrangeAggregationAggregationRange)(c)
}

func (c TsMrangeSelectedLabels) AggregationCount() TsMrangeAggregationAggregationCount {
	c.cs.s = append(c.cs.s, "AGGREGATION", "COUNT")
	return (TsMrangeAggregationAggregationCount)(c)
}

func (c TsMrangeSelectedLabels) AggregationFirst() TsMrangeAggregationAggregationFirst {
	c.cs.s = append(c.cs.s, "AGGREGATION", "FIRST")
	return (TsMrangeAggregationAggregationFirst)(c)
}

func (c TsMrangeSelectedLabels) AggregationLast() TsMrangeAggregationAggregationLast {
	c.cs.s = append(c.cs.s, "AGGREGATION", "LAST")
	return (TsMrangeAggregationAggregationLast)(c)
}

func (c TsMrangeSelectedLabels) AggregationStdP() TsMrangeAggregationAggregationStdP {
	c.cs.s = append(c.cs.s, "AGGREGATION", "STD.P")
	return (TsMrangeAggregationAggregationStdP)(c)
}

func (c TsMrangeSelectedLabels) AggregationStdS() TsMrangeAggregationAggregationStdS {
	c.cs.s = append(c.cs.s, "AGGREGATION", "STD.S")
	return (TsMrangeAggregationAggregationStdS)(c)
}

func (c TsMrangeSelectedLabels) AggregationVarP() TsMrangeAggregationAggregationVarP {
	c.cs.s = append(c.cs.s, "AGGREGATION", "VAR.P")
	return (TsMrangeAggregationAggregationVarP)(c)
}

func (c TsMrangeSelectedLabels) AggregationVarS() TsMrangeAggregationAggregationVarS {
	c.cs.s = append(c.cs.s, "AGGREGATION", "VAR.S")
	return (TsMrangeAggregationAggregationVarS)(c)
}

func (c TsMrangeSelectedLabels) AggregationTwa() TsMrangeAggregationAggregationTwa {
	c.cs.s = append(c.cs.s, "AGGREGATION", "TWA")
	return (TsMrangeAggregationAggregationTwa)(c)
}

func (c TsMrangeSelectedLabels) Filter(filter ...string) TsMrangeFilter {
	c.cs.s = append(c.cs.s, "FILTER")
	c.cs.s = append(c.cs.s, filter...)
	return (TsMrangeFilter)(c)
}

type TsMrangeTotimestamp Completed

func (c TsMrangeTotimestamp) Latest() TsMrangeLatest {
	c.cs.s = append(c.cs.s, "LATEST")
	return (TsMrangeLatest)(c)
}

func (c TsMrangeTotimestamp) FilterByTs(timestamp ...int64) TsMrangeFilterByTs {
	c.cs.s = append(c.cs.s, "FILTER_BY_TS")
	for _, n := range timestamp {
		c.cs.s = append(c.cs.s, strconv.FormatInt(n, 10))
	}
	return (TsMrangeFilterByTs)(c)
}

func (c TsMrangeTotimestamp) FilterByValue(min float64, max float64) TsMrangeFilterByValue {
	c.cs.s = append(c.cs.s, "FILTER_BY_VALUE", strconv.FormatFloat(min, 'f', -1, 64), strconv.FormatFloat(max, 'f', -1, 64))
	return (TsMrangeFilterByValue)(c)
}

func (c TsMrangeTotimestamp) Withlabels() TsMrangeWithlabels {
	c.cs.s = append(c.cs.s, "WITHLABELS")
	return (TsMrangeWithlabels)(c)
}

func (c TsMrangeTotimestamp) SelectedLabels(labels []string) TsMrangeSelectedLabels {
	c.cs.s = append(c.cs.s, "SELECTED_LABELS")
	c.cs.s = append(c.cs.s, labels...)
	return (TsMrangeSelectedLabels)(c)
}

func (c TsMrangeTotimestamp) Count(count int64) TsMrangeCount {
	c.cs.s = append(c.cs.s, "COUNT", strconv.FormatInt(count, 10))
	return (TsMrangeCount)(c)
}

func (c TsMrangeTotimestamp) Align(value string) TsMrangeAlign {
	c.cs.s = append(c.cs.s, "ALIGN", value)
	return (TsMrangeAlign)(c)
}

func (c TsMrangeTotimestamp) AggregationAvg() TsMrangeAggregationAggregationAvg {
	c.cs.s = append(c.cs.s, "AGGREGATION", "AVG")
	return (TsMrangeAggregationAggregationAvg)(c)
}

func (c TsMrangeTotimestamp) AggregationSum() TsMrangeAggregationAggregationSum {
	c.cs.s = append(c.cs.s, "AGGREGATION", "SUM")
	return (TsMrangeAggregationAggregationSum)(c)
}

func (c TsMrangeTotimestamp) AggregationMin() TsMrangeAggregationAggregationMin {
	c.cs.s = append(c.cs.s, "AGGREGATION", "MIN")
	return (TsMrangeAggregationAggregationMin)(c)
}

func (c TsMrangeTotimestamp) AggregationMax() TsMrangeAggregationAggregationMax {
	c.cs.s = append(c.cs.s, "AGGREGATION", "MAX")
	return (TsMrangeAggregationAggregationMax)(c)
}

func (c TsMrangeTotimestamp) AggregationRange() TsMrangeAggregationAggregationRange {
	c.cs.s = append(c.cs.s, "AGGREGATION", "RANGE")
	return (TsMrangeAggregationAggregationRange)(c)
}

func (c TsMrangeTotimestamp) AggregationCount() TsMrangeAggregationAggregationCount {
	c.cs.s = append(c.cs.s, "AGGREGATION", "COUNT")
	return (TsMrangeAggregationAggregationCount)(c)
}

func (c TsMrangeTotimestamp) AggregationFirst() TsMrangeAggregationAggregationFirst {
	c.cs.s = append(c.cs.s, "AGGREGATION", "FIRST")
	return (TsMrangeAggregationAggregationFirst)(c)
}

func (c TsMrangeTotimestamp) AggregationLast() TsMrangeAggregationAggregationLast {
	c.cs.s = append(c.cs.s, "AGGREGATION", "LAST")
	return (TsMrangeAggregationAggregationLast)(c)
}

func (c TsMrangeTotimestamp) AggregationStdP() TsMrangeAggregationAggregationStdP {
	c.cs.s = append(c.cs.s, "AGGREGATION", "STD.P")
	return (TsMrangeAggregationAggregationStdP)(c)
}

func (c TsMrangeTotimestamp) AggregationStdS() TsMrangeAggregationAggregationStdS {
	c.cs.s = append(c.cs.s, "AGGREGATION", "STD.S")
	return (TsMrangeAggregationAggregationStdS)(c)
}

func (c TsMrangeTotimestamp) AggregationVarP() TsMrangeAggregationAggregationVarP {
	c.cs.s = append(c.cs.s, "AGGREGATION", "VAR.P")
	return (TsMrangeAggregationAggregationVarP)(c)
}

func (c TsMrangeTotimestamp) AggregationVarS() TsMrangeAggregationAggregationVarS {
	c.cs.s = append(c.cs.s, "AGGREGATION", "VAR.S")
	return (TsMrangeAggregationAggregationVarS)(c)
}

func (c TsMrangeTotimestamp) AggregationTwa() TsMrangeAggregationAggregationTwa {
	c.cs.s = append(c.cs.s, "AGGREGATION", "TWA")
	return (TsMrangeAggregationAggregationTwa)(c)
}

func (c TsMrangeTotimestamp) Filter(filter ...string) TsMrangeFilter {
	c.cs.s = append(c.cs.s, "FILTER")
	c.cs.s = append(c.cs.s, filter...)
	return (TsMrangeFilter)(c)
}

type TsMrangeWithlabels Completed

func (c TsMrangeWithlabels) Count(count int64) TsMrangeCount {
	c.cs.s = append(c.cs.s, "COUNT", strconv.FormatInt(count, 10))
	return (TsMrangeCount)(c)
}

func (c TsMrangeWithlabels) Align(value string) TsMrangeAlign {
	c.cs.s = append(c.cs.s, "ALIGN", value)
	return (TsMrangeAlign)(c)
}

func (c TsMrangeWithlabels) AggregationAvg() TsMrangeAggregationAggregationAvg {
	c.cs.s = append(c.cs.s, "AGGREGATION", "AVG")
	return (TsMrangeAggregationAggregationAvg)(c)
}

func (c TsMrangeWithlabels) AggregationSum() TsMrangeAggregationAggregationSum {
	c.cs.s = append(c.cs.s, "AGGREGATION", "SUM")
	return (TsMrangeAggregationAggregationSum)(c)
}

func (c TsMrangeWithlabels) AggregationMin() TsMrangeAggregationAggregationMin {
	c.cs.s = append(c.cs.s, "AGGREGATION", "MIN")
	return (TsMrangeAggregationAggregationMin)(c)
}

func (c TsMrangeWithlabels) AggregationMax() TsMrangeAggregationAggregationMax {
	c.cs.s = append(c.cs.s, "AGGREGATION", "MAX")
	return (TsMrangeAggregationAggregationMax)(c)
}

func (c TsMrangeWithlabels) AggregationRange() TsMrangeAggregationAggregationRange {
	c.cs.s = append(c.cs.s, "AGGREGATION", "RANGE")
	return (TsMrangeAggregationAggregationRange)(c)
}

func (c TsMrangeWithlabels) AggregationCount() TsMrangeAggregationAggregationCount {
	c.cs.s = append(c.cs.s, "AGGREGATION", "COUNT")
	return (TsMrangeAggregationAggregationCount)(c)
}

func (c TsMrangeWithlabels) AggregationFirst() TsMrangeAggregationAggregationFirst {
	c.cs.s = append(c.cs.s, "AGGREGATION", "FIRST")
	return (TsMrangeAggregationAggregationFirst)(c)
}

func (c TsMrangeWithlabels) AggregationLast() TsMrangeAggregationAggregationLast {
	c.cs.s = append(c.cs.s, "AGGREGATION", "LAST")
	return (TsMrangeAggregationAggregationLast)(c)
}

func (c TsMrangeWithlabels) AggregationStdP() TsMrangeAggregationAggregationStdP {
	c.cs.s = append(c.cs.s, "AGGREGATION", "STD.P")
	return (TsMrangeAggregationAggregationStdP)(c)
}

func (c TsMrangeWithlabels) AggregationStdS() TsMrangeAggregationAggregationStdS {
	c.cs.s = append(c.cs.s, "AGGREGATION", "STD.S")
	return (TsMrangeAggregationAggregationStdS)(c)
}

func (c TsMrangeWithlabels) AggregationVarP() TsMrangeAggregationAggregationVarP {
	c.cs.s = append(c.cs.s, "AGGREGATION", "VAR.P")
	return (TsMrangeAggregationAggregationVarP)(c)
}

func (c TsMrangeWithlabels) AggregationVarS() TsMrangeAggregationAggregationVarS {
	c.cs.s = append(c.cs.s, "AGGREGATION", "VAR.S")
	return (TsMrangeAggregationAggregationVarS)(c)
}

func (c TsMrangeWithlabels) AggregationTwa() TsMrangeAggregationAggregationTwa {
	c.cs.s = append(c.cs.s, "AGGREGATION", "TWA")
	return (TsMrangeAggregationAggregationTwa)(c)
}

func (c TsMrangeWithlabels) Filter(filter ...string) TsMrangeFilter {
	c.cs.s = append(c.cs.s, "FILTER")
	c.cs.s = append(c.cs.s, filter...)
	return (TsMrangeFilter)(c)
}

type TsMrevrange Completed

func (b Builder) TsMrevrange() (c TsMrevrange) {
	c = TsMrevrange{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "TS.MREVRANGE")
	return c
}

func (c TsMrevrange) Fromtimestamp(fromtimestamp int64) TsMrevrangeFromtimestamp {
	c.cs.s = append(c.cs.s, strconv.FormatInt(fromtimestamp, 10))
	return (TsMrevrangeFromtimestamp)(c)
}

type TsMrevrangeAggregationAggregationAvg Completed

func (c TsMrevrangeAggregationAggregationAvg) Bucketduration(bucketduration int64) TsMrevrangeAggregationBucketduration {
	c.cs.s = append(c.cs.s, strconv.FormatInt(bucketduration, 10))
	return (TsMrevrangeAggregationBucketduration)(c)
}

type TsMrevrangeAggregationAggregationCount Completed

func (c TsMrevrangeAggregationAggregationCount) Bucketduration(bucketduration int64) TsMrevrangeAggregationBucketduration {
	c.cs.s = append(c.cs.s, strconv.FormatInt(bucketduration, 10))
	return (TsMrevrangeAggregationBucketduration)(c)
}

type TsMrevrangeAggregationAggregationFirst Completed

func (c TsMrevrangeAggregationAggregationFirst) Bucketduration(bucketduration int64) TsMrevrangeAggregationBucketduration {
	c.cs.s = append(c.cs.s, strconv.FormatInt(bucketduration, 10))
	return (TsMrevrangeAggregationBucketduration)(c)
}

type TsMrevrangeAggregationAggregationLast Completed

func (c TsMrevrangeAggregationAggregationLast) Bucketduration(bucketduration int64) TsMrevrangeAggregationBucketduration {
	c.cs.s = append(c.cs.s, strconv.FormatInt(bucketduration, 10))
	return (TsMrevrangeAggregationBucketduration)(c)
}

type TsMrevrangeAggregationAggregationMax Completed

func (c TsMrevrangeAggregationAggregationMax) Bucketduration(bucketduration int64) TsMrevrangeAggregationBucketduration {
	c.cs.s = append(c.cs.s, strconv.FormatInt(bucketduration, 10))
	return (TsMrevrangeAggregationBucketduration)(c)
}

type TsMrevrangeAggregationAggregationMin Completed

func (c TsMrevrangeAggregationAggregationMin) Bucketduration(bucketduration int64) TsMrevrangeAggregationBucketduration {
	c.cs.s = append(c.cs.s, strconv.FormatInt(bucketduration, 10))
	return (TsMrevrangeAggregationBucketduration)(c)
}

type TsMrevrangeAggregationAggregationRange Completed

func (c TsMrevrangeAggregationAggregationRange) Bucketduration(bucketduration int64) TsMrevrangeAggregationBucketduration {
	c.cs.s = append(c.cs.s, strconv.FormatInt(bucketduration, 10))
	return (TsMrevrangeAggregationBucketduration)(c)
}

type TsMrevrangeAggregationAggregationStdP Completed

func (c TsMrevrangeAggregationAggregationStdP) Bucketduration(bucketduration int64) TsMrevrangeAggregationBucketduration {
	c.cs.s = append(c.cs.s, strconv.FormatInt(bucketduration, 10))
	return (TsMrevrangeAggregationBucketduration)(c)
}

type TsMrevrangeAggregationAggregationStdS Completed

func (c TsMrevrangeAggregationAggregationStdS) Bucketduration(bucketduration int64) TsMrevrangeAggregationBucketduration {
	c.cs.s = append(c.cs.s, strconv.FormatInt(bucketduration, 10))
	return (TsMrevrangeAggregationBucketduration)(c)
}

type TsMrevrangeAggregationAggregationSum Completed

func (c TsMrevrangeAggregationAggregationSum) Bucketduration(bucketduration int64) TsMrevrangeAggregationBucketduration {
	c.cs.s = append(c.cs.s, strconv.FormatInt(bucketduration, 10))
	return (TsMrevrangeAggregationBucketduration)(c)
}

type TsMrevrangeAggregationAggregationTwa Completed

func (c TsMrevrangeAggregationAggregationTwa) Bucketduration(bucketduration int64) TsMrevrangeAggregationBucketduration {
	c.cs.s = append(c.cs.s, strconv.FormatInt(bucketduration, 10))
	return (TsMrevrangeAggregationBucketduration)(c)
}

type TsMrevrangeAggregationAggregationVarP Completed

func (c TsMrevrangeAggregationAggregationVarP) Bucketduration(bucketduration int64) TsMrevrangeAggregationBucketduration {
	c.cs.s = append(c.cs.s, strconv.FormatInt(bucketduration, 10))
	return (TsMrevrangeAggregationBucketduration)(c)
}

type TsMrevrangeAggregationAggregationVarS Completed

func (c TsMrevrangeAggregationAggregationVarS) Bucketduration(bucketduration int64) TsMrevrangeAggregationBucketduration {
	c.cs.s = append(c.cs.s, strconv.FormatInt(bucketduration, 10))
	return (TsMrevrangeAggregationBucketduration)(c)
}

type TsMrevrangeAggregationBucketduration Completed

func (c TsMrevrangeAggregationBucketduration) Buckettimestamp(buckettimestamp string) TsMrevrangeAggregationBuckettimestamp {
	c.cs.s = append(c.cs.s, "BUCKETTIMESTAMP", buckettimestamp)
	return (TsMrevrangeAggregationBuckettimestamp)(c)
}

func (c TsMrevrangeAggregationBucketduration) Empty() TsMrevrangeAggregationEmpty {
	c.cs.s = append(c.cs.s, "EMPTY")
	return (TsMrevrangeAggregationEmpty)(c)
}

func (c TsMrevrangeAggregationBucketduration) Filter(filter ...string) TsMrevrangeFilter {
	c.cs.s = append(c.cs.s, "FILTER")
	c.cs.s = append(c.cs.s, filter...)
	return (TsMrevrangeFilter)(c)
}

type TsMrevrangeAggregationBuckettimestamp Completed

func (c TsMrevrangeAggregationBuckettimestamp) Empty() TsMrevrangeAggregationEmpty {
	c.cs.s = append(c.cs.s, "EMPTY")
	return (TsMrevrangeAggregationEmpty)(c)
}

func (c TsMrevrangeAggregationBuckettimestamp) Filter(filter ...string) TsMrevrangeFilter {
	c.cs.s = append(c.cs.s, "FILTER")
	c.cs.s = append(c.cs.s, filter...)
	return (TsMrevrangeFilter)(c)
}

type TsMrevrangeAggregationEmpty Completed

func (c TsMrevrangeAggregationEmpty) Filter(filter ...string) TsMrevrangeFilter {
	c.cs.s = append(c.cs.s, "FILTER")
	c.cs.s = append(c.cs.s, filter...)
	return (TsMrevrangeFilter)(c)
}

type TsMrevrangeAlign Completed

func (c TsMrevrangeAlign) AggregationAvg() TsMrevrangeAggregationAggregationAvg {
	c.cs.s = append(c.cs.s, "AGGREGATION", "AVG")
	return (TsMrevrangeAggregationAggregationAvg)(c)
}

func (c TsMrevrangeAlign) AggregationSum() TsMrevrangeAggregationAggregationSum {
	c.cs.s = append(c.cs.s, "AGGREGATION", "SUM")
	return (TsMrevrangeAggregationAggregationSum)(c)
}

func (c TsMrevrangeAlign) AggregationMin() TsMrevrangeAggregationAggregationMin {
	c.cs.s = append(c.cs.s, "AGGREGATION", "MIN")
	return (TsMrevrangeAggregationAggregationMin)(c)
}

func (c TsMrevrangeAlign) AggregationMax() TsMrevrangeAggregationAggregationMax {
	c.cs.s = append(c.cs.s, "AGGREGATION", "MAX")
	return (TsMrevrangeAggregationAggregationMax)(c)
}

func (c TsMrevrangeAlign) AggregationRange() TsMrevrangeAggregationAggregationRange {
	c.cs.s = append(c.cs.s, "AGGREGATION", "RANGE")
	return (TsMrevrangeAggregationAggregationRange)(c)
}

func (c TsMrevrangeAlign) AggregationCount() TsMrevrangeAggregationAggregationCount {
	c.cs.s = append(c.cs.s, "AGGREGATION", "COUNT")
	return (TsMrevrangeAggregationAggregationCount)(c)
}

func (c TsMrevrangeAlign) AggregationFirst() TsMrevrangeAggregationAggregationFirst {
	c.cs.s = append(c.cs.s, "AGGREGATION", "FIRST")
	return (TsMrevrangeAggregationAggregationFirst)(c)
}

func (c TsMrevrangeAlign) AggregationLast() TsMrevrangeAggregationAggregationLast {
	c.cs.s = append(c.cs.s, "AGGREGATION", "LAST")
	return (TsMrevrangeAggregationAggregationLast)(c)
}

func (c TsMrevrangeAlign) AggregationStdP() TsMrevrangeAggregationAggregationStdP {
	c.cs.s = append(c.cs.s, "AGGREGATION", "STD.P")
	return (TsMrevrangeAggregationAggregationStdP)(c)
}

func (c TsMrevrangeAlign) AggregationStdS() TsMrevrangeAggregationAggregationStdS {
	c.cs.s = append(c.cs.s, "AGGREGATION", "STD.S")
	return (TsMrevrangeAggregationAggregationStdS)(c)
}

func (c TsMrevrangeAlign) AggregationVarP() TsMrevrangeAggregationAggregationVarP {
	c.cs.s = append(c.cs.s, "AGGREGATION", "VAR.P")
	return (TsMrevrangeAggregationAggregationVarP)(c)
}

func (c TsMrevrangeAlign) AggregationVarS() TsMrevrangeAggregationAggregationVarS {
	c.cs.s = append(c.cs.s, "AGGREGATION", "VAR.S")
	return (TsMrevrangeAggregationAggregationVarS)(c)
}

func (c TsMrevrangeAlign) AggregationTwa() TsMrevrangeAggregationAggregationTwa {
	c.cs.s = append(c.cs.s, "AGGREGATION", "TWA")
	return (TsMrevrangeAggregationAggregationTwa)(c)
}

func (c TsMrevrangeAlign) Filter(filter ...string) TsMrevrangeFilter {
	c.cs.s = append(c.cs.s, "FILTER")
	c.cs.s = append(c.cs.s, filter...)
	return (TsMrevrangeFilter)(c)
}

type TsMrevrangeCount Completed

func (c TsMrevrangeCount) Align(value string) TsMrevrangeAlign {
	c.cs.s = append(c.cs.s, "ALIGN", value)
	return (TsMrevrangeAlign)(c)
}

func (c TsMrevrangeCount) AggregationAvg() TsMrevrangeAggregationAggregationAvg {
	c.cs.s = append(c.cs.s, "AGGREGATION", "AVG")
	return (TsMrevrangeAggregationAggregationAvg)(c)
}

func (c TsMrevrangeCount) AggregationSum() TsMrevrangeAggregationAggregationSum {
	c.cs.s = append(c.cs.s, "AGGREGATION", "SUM")
	return (TsMrevrangeAggregationAggregationSum)(c)
}

func (c TsMrevrangeCount) AggregationMin() TsMrevrangeAggregationAggregationMin {
	c.cs.s = append(c.cs.s, "AGGREGATION", "MIN")
	return (TsMrevrangeAggregationAggregationMin)(c)
}

func (c TsMrevrangeCount) AggregationMax() TsMrevrangeAggregationAggregationMax {
	c.cs.s = append(c.cs.s, "AGGREGATION", "MAX")
	return (TsMrevrangeAggregationAggregationMax)(c)
}

func (c TsMrevrangeCount) AggregationRange() TsMrevrangeAggregationAggregationRange {
	c.cs.s = append(c.cs.s, "AGGREGATION", "RANGE")
	return (TsMrevrangeAggregationAggregationRange)(c)
}

func (c TsMrevrangeCount) AggregationCount() TsMrevrangeAggregationAggregationCount {
	c.cs.s = append(c.cs.s, "AGGREGATION", "COUNT")
	return (TsMrevrangeAggregationAggregationCount)(c)
}

func (c TsMrevrangeCount) AggregationFirst() TsMrevrangeAggregationAggregationFirst {
	c.cs.s = append(c.cs.s, "AGGREGATION", "FIRST")
	return (TsMrevrangeAggregationAggregationFirst)(c)
}

func (c TsMrevrangeCount) AggregationLast() TsMrevrangeAggregationAggregationLast {
	c.cs.s = append(c.cs.s, "AGGREGATION", "LAST")
	return (TsMrevrangeAggregationAggregationLast)(c)
}

func (c TsMrevrangeCount) AggregationStdP() TsMrevrangeAggregationAggregationStdP {
	c.cs.s = append(c.cs.s, "AGGREGATION", "STD.P")
	return (TsMrevrangeAggregationAggregationStdP)(c)
}

func (c TsMrevrangeCount) AggregationStdS() TsMrevrangeAggregationAggregationStdS {
	c.cs.s = append(c.cs.s, "AGGREGATION", "STD.S")
	return (TsMrevrangeAggregationAggregationStdS)(c)
}

func (c TsMrevrangeCount) AggregationVarP() TsMrevrangeAggregationAggregationVarP {
	c.cs.s = append(c.cs.s, "AGGREGATION", "VAR.P")
	return (TsMrevrangeAggregationAggregationVarP)(c)
}

func (c TsMrevrangeCount) AggregationVarS() TsMrevrangeAggregationAggregationVarS {
	c.cs.s = append(c.cs.s, "AGGREGATION", "VAR.S")
	return (TsMrevrangeAggregationAggregationVarS)(c)
}

func (c TsMrevrangeCount) AggregationTwa() TsMrevrangeAggregationAggregationTwa {
	c.cs.s = append(c.cs.s, "AGGREGATION", "TWA")
	return (TsMrevrangeAggregationAggregationTwa)(c)
}

func (c TsMrevrangeCount) Filter(filter ...string) TsMrevrangeFilter {
	c.cs.s = append(c.cs.s, "FILTER")
	c.cs.s = append(c.cs.s, filter...)
	return (TsMrevrangeFilter)(c)
}

type TsMrevrangeFilter Completed

func (c TsMrevrangeFilter) Filter(filter ...string) TsMrevrangeFilter {
	c.cs.s = append(c.cs.s, "FILTER")
	c.cs.s = append(c.cs.s, filter...)
	return c
}

func (c TsMrevrangeFilter) Groupby(label string, reduce string, reducer string) TsMrevrangeGroupby {
	c.cs.s = append(c.cs.s, "GROUPBY", label, reduce, reducer)
	return (TsMrevrangeGroupby)(c)
}

func (c TsMrevrangeFilter) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type TsMrevrangeFilterByTs Completed

func (c TsMrevrangeFilterByTs) FilterByTs(timestamp ...int64) TsMrevrangeFilterByTs {
	c.cs.s = append(c.cs.s, "FILTER_BY_TS")
	for _, n := range timestamp {
		c.cs.s = append(c.cs.s, strconv.FormatInt(n, 10))
	}
	return c
}

func (c TsMrevrangeFilterByTs) FilterByValue(min float64, max float64) TsMrevrangeFilterByValue {
	c.cs.s = append(c.cs.s, "FILTER_BY_VALUE", strconv.FormatFloat(min, 'f', -1, 64), strconv.FormatFloat(max, 'f', -1, 64))
	return (TsMrevrangeFilterByValue)(c)
}

func (c TsMrevrangeFilterByTs) Withlabels() TsMrevrangeWithlabels {
	c.cs.s = append(c.cs.s, "WITHLABELS")
	return (TsMrevrangeWithlabels)(c)
}

func (c TsMrevrangeFilterByTs) SelectedLabels(labels []string) TsMrevrangeSelectedLabels {
	c.cs.s = append(c.cs.s, "SELECTED_LABELS")
	c.cs.s = append(c.cs.s, labels...)
	return (TsMrevrangeSelectedLabels)(c)
}

func (c TsMrevrangeFilterByTs) Count(count int64) TsMrevrangeCount {
	c.cs.s = append(c.cs.s, "COUNT", strconv.FormatInt(count, 10))
	return (TsMrevrangeCount)(c)
}

func (c TsMrevrangeFilterByTs) Align(value string) TsMrevrangeAlign {
	c.cs.s = append(c.cs.s, "ALIGN", value)
	return (TsMrevrangeAlign)(c)
}

func (c TsMrevrangeFilterByTs) AggregationAvg() TsMrevrangeAggregationAggregationAvg {
	c.cs.s = append(c.cs.s, "AGGREGATION", "AVG")
	return (TsMrevrangeAggregationAggregationAvg)(c)
}

func (c TsMrevrangeFilterByTs) AggregationSum() TsMrevrangeAggregationAggregationSum {
	c.cs.s = append(c.cs.s, "AGGREGATION", "SUM")
	return (TsMrevrangeAggregationAggregationSum)(c)
}

func (c TsMrevrangeFilterByTs) AggregationMin() TsMrevrangeAggregationAggregationMin {
	c.cs.s = append(c.cs.s, "AGGREGATION", "MIN")
	return (TsMrevrangeAggregationAggregationMin)(c)
}

func (c TsMrevrangeFilterByTs) AggregationMax() TsMrevrangeAggregationAggregationMax {
	c.cs.s = append(c.cs.s, "AGGREGATION", "MAX")
	return (TsMrevrangeAggregationAggregationMax)(c)
}

func (c TsMrevrangeFilterByTs) AggregationRange() TsMrevrangeAggregationAggregationRange {
	c.cs.s = append(c.cs.s, "AGGREGATION", "RANGE")
	return (TsMrevrangeAggregationAggregationRange)(c)
}

func (c TsMrevrangeFilterByTs) AggregationCount() TsMrevrangeAggregationAggregationCount {
	c.cs.s = append(c.cs.s, "AGGREGATION", "COUNT")
	return (TsMrevrangeAggregationAggregationCount)(c)
}

func (c TsMrevrangeFilterByTs) AggregationFirst() TsMrevrangeAggregationAggregationFirst {
	c.cs.s = append(c.cs.s, "AGGREGATION", "FIRST")
	return (TsMrevrangeAggregationAggregationFirst)(c)
}

func (c TsMrevrangeFilterByTs) AggregationLast() TsMrevrangeAggregationAggregationLast {
	c.cs.s = append(c.cs.s, "AGGREGATION", "LAST")
	return (TsMrevrangeAggregationAggregationLast)(c)
}

func (c TsMrevrangeFilterByTs) AggregationStdP() TsMrevrangeAggregationAggregationStdP {
	c.cs.s = append(c.cs.s, "AGGREGATION", "STD.P")
	return (TsMrevrangeAggregationAggregationStdP)(c)
}

func (c TsMrevrangeFilterByTs) AggregationStdS() TsMrevrangeAggregationAggregationStdS {
	c.cs.s = append(c.cs.s, "AGGREGATION", "STD.S")
	return (TsMrevrangeAggregationAggregationStdS)(c)
}

func (c TsMrevrangeFilterByTs) AggregationVarP() TsMrevrangeAggregationAggregationVarP {
	c.cs.s = append(c.cs.s, "AGGREGATION", "VAR.P")
	return (TsMrevrangeAggregationAggregationVarP)(c)
}

func (c TsMrevrangeFilterByTs) AggregationVarS() TsMrevrangeAggregationAggregationVarS {
	c.cs.s = append(c.cs.s, "AGGREGATION", "VAR.S")
	return (TsMrevrangeAggregationAggregationVarS)(c)
}

func (c TsMrevrangeFilterByTs) AggregationTwa() TsMrevrangeAggregationAggregationTwa {
	c.cs.s = append(c.cs.s, "AGGREGATION", "TWA")
	return (TsMrevrangeAggregationAggregationTwa)(c)
}

func (c TsMrevrangeFilterByTs) Filter(filter ...string) TsMrevrangeFilter {
	c.cs.s = append(c.cs.s, "FILTER")
	c.cs.s = append(c.cs.s, filter...)
	return (TsMrevrangeFilter)(c)
}

type TsMrevrangeFilterByValue Completed

func (c TsMrevrangeFilterByValue) Withlabels() TsMrevrangeWithlabels {
	c.cs.s = append(c.cs.s, "WITHLABELS")
	return (TsMrevrangeWithlabels)(c)
}

func (c TsMrevrangeFilterByValue) SelectedLabels(labels []string) TsMrevrangeSelectedLabels {
	c.cs.s = append(c.cs.s, "SELECTED_LABELS")
	c.cs.s = append(c.cs.s, labels...)
	return (TsMrevrangeSelectedLabels)(c)
}

func (c TsMrevrangeFilterByValue) Count(count int64) TsMrevrangeCount {
	c.cs.s = append(c.cs.s, "COUNT", strconv.FormatInt(count, 10))
	return (TsMrevrangeCount)(c)
}

func (c TsMrevrangeFilterByValue) Align(value string) TsMrevrangeAlign {
	c.cs.s = append(c.cs.s, "ALIGN", value)
	return (TsMrevrangeAlign)(c)
}

func (c TsMrevrangeFilterByValue) AggregationAvg() TsMrevrangeAggregationAggregationAvg {
	c.cs.s = append(c.cs.s, "AGGREGATION", "AVG")
	return (TsMrevrangeAggregationAggregationAvg)(c)
}

func (c TsMrevrangeFilterByValue) AggregationSum() TsMrevrangeAggregationAggregationSum {
	c.cs.s = append(c.cs.s, "AGGREGATION", "SUM")
	return (TsMrevrangeAggregationAggregationSum)(c)
}

func (c TsMrevrangeFilterByValue) AggregationMin() TsMrevrangeAggregationAggregationMin {
	c.cs.s = append(c.cs.s, "AGGREGATION", "MIN")
	return (TsMrevrangeAggregationAggregationMin)(c)
}

func (c TsMrevrangeFilterByValue) AggregationMax() TsMrevrangeAggregationAggregationMax {
	c.cs.s = append(c.cs.s, "AGGREGATION", "MAX")
	return (TsMrevrangeAggregationAggregationMax)(c)
}

func (c TsMrevrangeFilterByValue) AggregationRange() TsMrevrangeAggregationAggregationRange {
	c.cs.s = append(c.cs.s, "AGGREGATION", "RANGE")
	return (TsMrevrangeAggregationAggregationRange)(c)
}

func (c TsMrevrangeFilterByValue) AggregationCount() TsMrevrangeAggregationAggregationCount {
	c.cs.s = append(c.cs.s, "AGGREGATION", "COUNT")
	return (TsMrevrangeAggregationAggregationCount)(c)
}

func (c TsMrevrangeFilterByValue) AggregationFirst() TsMrevrangeAggregationAggregationFirst {
	c.cs.s = append(c.cs.s, "AGGREGATION", "FIRST")
	return (TsMrevrangeAggregationAggregationFirst)(c)
}

func (c TsMrevrangeFilterByValue) AggregationLast() TsMrevrangeAggregationAggregationLast {
	c.cs.s = append(c.cs.s, "AGGREGATION", "LAST")
	return (TsMrevrangeAggregationAggregationLast)(c)
}

func (c TsMrevrangeFilterByValue) AggregationStdP() TsMrevrangeAggregationAggregationStdP {
	c.cs.s = append(c.cs.s, "AGGREGATION", "STD.P")
	return (TsMrevrangeAggregationAggregationStdP)(c)
}

func (c TsMrevrangeFilterByValue) AggregationStdS() TsMrevrangeAggregationAggregationStdS {
	c.cs.s = append(c.cs.s, "AGGREGATION", "STD.S")
	return (TsMrevrangeAggregationAggregationStdS)(c)
}

func (c TsMrevrangeFilterByValue) AggregationVarP() TsMrevrangeAggregationAggregationVarP {
	c.cs.s = append(c.cs.s, "AGGREGATION", "VAR.P")
	return (TsMrevrangeAggregationAggregationVarP)(c)
}

func (c TsMrevrangeFilterByValue) AggregationVarS() TsMrevrangeAggregationAggregationVarS {
	c.cs.s = append(c.cs.s, "AGGREGATION", "VAR.S")
	return (TsMrevrangeAggregationAggregationVarS)(c)
}

func (c TsMrevrangeFilterByValue) AggregationTwa() TsMrevrangeAggregationAggregationTwa {
	c.cs.s = append(c.cs.s, "AGGREGATION", "TWA")
	return (TsMrevrangeAggregationAggregationTwa)(c)
}

func (c TsMrevrangeFilterByValue) Filter(filter ...string) TsMrevrangeFilter {
	c.cs.s = append(c.cs.s, "FILTER")
	c.cs.s = append(c.cs.s, filter...)
	return (TsMrevrangeFilter)(c)
}

type TsMrevrangeFromtimestamp Completed

func (c TsMrevrangeFromtimestamp) Totimestamp(totimestamp int64) TsMrevrangeTotimestamp {
	c.cs.s = append(c.cs.s, strconv.FormatInt(totimestamp, 10))
	return (TsMrevrangeTotimestamp)(c)
}

type TsMrevrangeGroupby Completed

func (c TsMrevrangeGroupby) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type TsMrevrangeLatest Completed

func (c TsMrevrangeLatest) FilterByTs(timestamp ...int64) TsMrevrangeFilterByTs {
	c.cs.s = append(c.cs.s, "FILTER_BY_TS")
	for _, n := range timestamp {
		c.cs.s = append(c.cs.s, strconv.FormatInt(n, 10))
	}
	return (TsMrevrangeFilterByTs)(c)
}

func (c TsMrevrangeLatest) FilterByValue(min float64, max float64) TsMrevrangeFilterByValue {
	c.cs.s = append(c.cs.s, "FILTER_BY_VALUE", strconv.FormatFloat(min, 'f', -1, 64), strconv.FormatFloat(max, 'f', -1, 64))
	return (TsMrevrangeFilterByValue)(c)
}

func (c TsMrevrangeLatest) Withlabels() TsMrevrangeWithlabels {
	c.cs.s = append(c.cs.s, "WITHLABELS")
	return (TsMrevrangeWithlabels)(c)
}

func (c TsMrevrangeLatest) SelectedLabels(labels []string) TsMrevrangeSelectedLabels {
	c.cs.s = append(c.cs.s, "SELECTED_LABELS")
	c.cs.s = append(c.cs.s, labels...)
	return (TsMrevrangeSelectedLabels)(c)
}

func (c TsMrevrangeLatest) Count(count int64) TsMrevrangeCount {
	c.cs.s = append(c.cs.s, "COUNT", strconv.FormatInt(count, 10))
	return (TsMrevrangeCount)(c)
}

func (c TsMrevrangeLatest) Align(value string) TsMrevrangeAlign {
	c.cs.s = append(c.cs.s, "ALIGN", value)
	return (TsMrevrangeAlign)(c)
}

func (c TsMrevrangeLatest) AggregationAvg() TsMrevrangeAggregationAggregationAvg {
	c.cs.s = append(c.cs.s, "AGGREGATION", "AVG")
	return (TsMrevrangeAggregationAggregationAvg)(c)
}

func (c TsMrevrangeLatest) AggregationSum() TsMrevrangeAggregationAggregationSum {
	c.cs.s = append(c.cs.s, "AGGREGATION", "SUM")
	return (TsMrevrangeAggregationAggregationSum)(c)
}

func (c TsMrevrangeLatest) AggregationMin() TsMrevrangeAggregationAggregationMin {
	c.cs.s = append(c.cs.s, "AGGREGATION", "MIN")
	return (TsMrevrangeAggregationAggregationMin)(c)
}

func (c TsMrevrangeLatest) AggregationMax() TsMrevrangeAggregationAggregationMax {
	c.cs.s = append(c.cs.s, "AGGREGATION", "MAX")
	return (TsMrevrangeAggregationAggregationMax)(c)
}

func (c TsMrevrangeLatest) AggregationRange() TsMrevrangeAggregationAggregationRange {
	c.cs.s = append(c.cs.s, "AGGREGATION", "RANGE")
	return (TsMrevrangeAggregationAggregationRange)(c)
}

func (c TsMrevrangeLatest) AggregationCount() TsMrevrangeAggregationAggregationCount {
	c.cs.s = append(c.cs.s, "AGGREGATION", "COUNT")
	return (TsMrevrangeAggregationAggregationCount)(c)
}

func (c TsMrevrangeLatest) AggregationFirst() TsMrevrangeAggregationAggregationFirst {
	c.cs.s = append(c.cs.s, "AGGREGATION", "FIRST")
	return (TsMrevrangeAggregationAggregationFirst)(c)
}

func (c TsMrevrangeLatest) AggregationLast() TsMrevrangeAggregationAggregationLast {
	c.cs.s = append(c.cs.s, "AGGREGATION", "LAST")
	return (TsMrevrangeAggregationAggregationLast)(c)
}

func (c TsMrevrangeLatest) AggregationStdP() TsMrevrangeAggregationAggregationStdP {
	c.cs.s = append(c.cs.s, "AGGREGATION", "STD.P")
	return (TsMrevrangeAggregationAggregationStdP)(c)
}

func (c TsMrevrangeLatest) AggregationStdS() TsMrevrangeAggregationAggregationStdS {
	c.cs.s = append(c.cs.s, "AGGREGATION", "STD.S")
	return (TsMrevrangeAggregationAggregationStdS)(c)
}

func (c TsMrevrangeLatest) AggregationVarP() TsMrevrangeAggregationAggregationVarP {
	c.cs.s = append(c.cs.s, "AGGREGATION", "VAR.P")
	return (TsMrevrangeAggregationAggregationVarP)(c)
}

func (c TsMrevrangeLatest) AggregationVarS() TsMrevrangeAggregationAggregationVarS {
	c.cs.s = append(c.cs.s, "AGGREGATION", "VAR.S")
	return (TsMrevrangeAggregationAggregationVarS)(c)
}

func (c TsMrevrangeLatest) AggregationTwa() TsMrevrangeAggregationAggregationTwa {
	c.cs.s = append(c.cs.s, "AGGREGATION", "TWA")
	return (TsMrevrangeAggregationAggregationTwa)(c)
}

func (c TsMrevrangeLatest) Filter(filter ...string) TsMrevrangeFilter {
	c.cs.s = append(c.cs.s, "FILTER")
	c.cs.s = append(c.cs.s, filter...)
	return (TsMrevrangeFilter)(c)
}

type TsMrevrangeSelectedLabels Completed

func (c TsMrevrangeSelectedLabels) Count(count int64) TsMrevrangeCount {
	c.cs.s = append(c.cs.s, "COUNT", strconv.FormatInt(count, 10))
	return (TsMrevrangeCount)(c)
}

func (c TsMrevrangeSelectedLabels) Align(value string) TsMrevrangeAlign {
	c.cs.s = append(c.cs.s, "ALIGN", value)
	return (TsMrevrangeAlign)(c)
}

func (c TsMrevrangeSelectedLabels) AggregationAvg() TsMrevrangeAggregationAggregationAvg {
	c.cs.s = append(c.cs.s, "AGGREGATION", "AVG")
	return (TsMrevrangeAggregationAggregationAvg)(c)
}

func (c TsMrevrangeSelectedLabels) AggregationSum() TsMrevrangeAggregationAggregationSum {
	c.cs.s = append(c.cs.s, "AGGREGATION", "SUM")
	return (TsMrevrangeAggregationAggregationSum)(c)
}

func (c TsMrevrangeSelectedLabels) AggregationMin() TsMrevrangeAggregationAggregationMin {
	c.cs.s = append(c.cs.s, "AGGREGATION", "MIN")
	return (TsMrevrangeAggregationAggregationMin)(c)
}

func (c TsMrevrangeSelectedLabels) AggregationMax() TsMrevrangeAggregationAggregationMax {
	c.cs.s = append(c.cs.s, "AGGREGATION", "MAX")
	return (TsMrevrangeAggregationAggregationMax)(c)
}

func (c TsMrevrangeSelectedLabels) AggregationRange() TsMrevrangeAggregationAggregationRange {
	c.cs.s = append(c.cs.s, "AGGREGATION", "RANGE")
	return (TsMrevrangeAggregationAggregationRange)(c)
}

func (c TsMrevrangeSelectedLabels) AggregationCount() TsMrevrangeAggregationAggregationCount {
	c.cs.s = append(c.cs.s, "AGGREGATION", "COUNT")
	return (TsMrevrangeAggregationAggregationCount)(c)
}

func (c TsMrevrangeSelectedLabels) AggregationFirst() TsMrevrangeAggregationAggregationFirst {
	c.cs.s = append(c.cs.s, "AGGREGATION", "FIRST")
	return (TsMrevrangeAggregationAggregationFirst)(c)
}

func (c TsMrevrangeSelectedLabels) AggregationLast() TsMrevrangeAggregationAggregationLast {
	c.cs.s = append(c.cs.s, "AGGREGATION", "LAST")
	return (TsMrevrangeAggregationAggregationLast)(c)
}

func (c TsMrevrangeSelectedLabels) AggregationStdP() TsMrevrangeAggregationAggregationStdP {
	c.cs.s = append(c.cs.s, "AGGREGATION", "STD.P")
	return (TsMrevrangeAggregationAggregationStdP)(c)
}

func (c TsMrevrangeSelectedLabels) AggregationStdS() TsMrevrangeAggregationAggregationStdS {
	c.cs.s = append(c.cs.s, "AGGREGATION", "STD.S")
	return (TsMrevrangeAggregationAggregationStdS)(c)
}

func (c TsMrevrangeSelectedLabels) AggregationVarP() TsMrevrangeAggregationAggregationVarP {
	c.cs.s = append(c.cs.s, "AGGREGATION", "VAR.P")
	return (TsMrevrangeAggregationAggregationVarP)(c)
}

func (c TsMrevrangeSelectedLabels) AggregationVarS() TsMrevrangeAggregationAggregationVarS {
	c.cs.s = append(c.cs.s, "AGGREGATION", "VAR.S")
	return (TsMrevrangeAggregationAggregationVarS)(c)
}

func (c TsMrevrangeSelectedLabels) AggregationTwa() TsMrevrangeAggregationAggregationTwa {
	c.cs.s = append(c.cs.s, "AGGREGATION", "TWA")
	return (TsMrevrangeAggregationAggregationTwa)(c)
}

func (c TsMrevrangeSelectedLabels) Filter(filter ...string) TsMrevrangeFilter {
	c.cs.s = append(c.cs.s, "FILTER")
	c.cs.s = append(c.cs.s, filter...)
	return (TsMrevrangeFilter)(c)
}

type TsMrevrangeTotimestamp Completed

func (c TsMrevrangeTotimestamp) Latest() TsMrevrangeLatest {
	c.cs.s = append(c.cs.s, "LATEST")
	return (TsMrevrangeLatest)(c)
}

func (c TsMrevrangeTotimestamp) FilterByTs(timestamp ...int64) TsMrevrangeFilterByTs {
	c.cs.s = append(c.cs.s, "FILTER_BY_TS")
	for _, n := range timestamp {
		c.cs.s = append(c.cs.s, strconv.FormatInt(n, 10))
	}
	return (TsMrevrangeFilterByTs)(c)
}

func (c TsMrevrangeTotimestamp) FilterByValue(min float64, max float64) TsMrevrangeFilterByValue {
	c.cs.s = append(c.cs.s, "FILTER_BY_VALUE", strconv.FormatFloat(min, 'f', -1, 64), strconv.FormatFloat(max, 'f', -1, 64))
	return (TsMrevrangeFilterByValue)(c)
}

func (c TsMrevrangeTotimestamp) Withlabels() TsMrevrangeWithlabels {
	c.cs.s = append(c.cs.s, "WITHLABELS")
	return (TsMrevrangeWithlabels)(c)
}

func (c TsMrevrangeTotimestamp) SelectedLabels(labels []string) TsMrevrangeSelectedLabels {
	c.cs.s = append(c.cs.s, "SELECTED_LABELS")
	c.cs.s = append(c.cs.s, labels...)
	return (TsMrevrangeSelectedLabels)(c)
}

func (c TsMrevrangeTotimestamp) Count(count int64) TsMrevrangeCount {
	c.cs.s = append(c.cs.s, "COUNT", strconv.FormatInt(count, 10))
	return (TsMrevrangeCount)(c)
}

func (c TsMrevrangeTotimestamp) Align(value string) TsMrevrangeAlign {
	c.cs.s = append(c.cs.s, "ALIGN", value)
	return (TsMrevrangeAlign)(c)
}

func (c TsMrevrangeTotimestamp) AggregationAvg() TsMrevrangeAggregationAggregationAvg {
	c.cs.s = append(c.cs.s, "AGGREGATION", "AVG")
	return (TsMrevrangeAggregationAggregationAvg)(c)
}

func (c TsMrevrangeTotimestamp) AggregationSum() TsMrevrangeAggregationAggregationSum {
	c.cs.s = append(c.cs.s, "AGGREGATION", "SUM")
	return (TsMrevrangeAggregationAggregationSum)(c)
}

func (c TsMrevrangeTotimestamp) AggregationMin() TsMrevrangeAggregationAggregationMin {
	c.cs.s = append(c.cs.s, "AGGREGATION", "MIN")
	return (TsMrevrangeAggregationAggregationMin)(c)
}

func (c TsMrevrangeTotimestamp) AggregationMax() TsMrevrangeAggregationAggregationMax {
	c.cs.s = append(c.cs.s, "AGGREGATION", "MAX")
	return (TsMrevrangeAggregationAggregationMax)(c)
}

func (c TsMrevrangeTotimestamp) AggregationRange() TsMrevrangeAggregationAggregationRange {
	c.cs.s = append(c.cs.s, "AGGREGATION", "RANGE")
	return (TsMrevrangeAggregationAggregationRange)(c)
}

func (c TsMrevrangeTotimestamp) AggregationCount() TsMrevrangeAggregationAggregationCount {
	c.cs.s = append(c.cs.s, "AGGREGATION", "COUNT")
	return (TsMrevrangeAggregationAggregationCount)(c)
}

func (c TsMrevrangeTotimestamp) AggregationFirst() TsMrevrangeAggregationAggregationFirst {
	c.cs.s = append(c.cs.s, "AGGREGATION", "FIRST")
	return (TsMrevrangeAggregationAggregationFirst)(c)
}

func (c TsMrevrangeTotimestamp) AggregationLast() TsMrevrangeAggregationAggregationLast {
	c.cs.s = append(c.cs.s, "AGGREGATION", "LAST")
	return (TsMrevrangeAggregationAggregationLast)(c)
}

func (c TsMrevrangeTotimestamp) AggregationStdP() TsMrevrangeAggregationAggregationStdP {
	c.cs.s = append(c.cs.s, "AGGREGATION", "STD.P")
	return (TsMrevrangeAggregationAggregationStdP)(c)
}

func (c TsMrevrangeTotimestamp) AggregationStdS() TsMrevrangeAggregationAggregationStdS {
	c.cs.s = append(c.cs.s, "AGGREGATION", "STD.S")
	return (TsMrevrangeAggregationAggregationStdS)(c)
}

func (c TsMrevrangeTotimestamp) AggregationVarP() TsMrevrangeAggregationAggregationVarP {
	c.cs.s = append(c.cs.s, "AGGREGATION", "VAR.P")
	return (TsMrevrangeAggregationAggregationVarP)(c)
}

func (c TsMrevrangeTotimestamp) AggregationVarS() TsMrevrangeAggregationAggregationVarS {
	c.cs.s = append(c.cs.s, "AGGREGATION", "VAR.S")
	return (TsMrevrangeAggregationAggregationVarS)(c)
}

func (c TsMrevrangeTotimestamp) AggregationTwa() TsMrevrangeAggregationAggregationTwa {
	c.cs.s = append(c.cs.s, "AGGREGATION", "TWA")
	return (TsMrevrangeAggregationAggregationTwa)(c)
}

func (c TsMrevrangeTotimestamp) Filter(filter ...string) TsMrevrangeFilter {
	c.cs.s = append(c.cs.s, "FILTER")
	c.cs.s = append(c.cs.s, filter...)
	return (TsMrevrangeFilter)(c)
}

type TsMrevrangeWithlabels Completed

func (c TsMrevrangeWithlabels) Count(count int64) TsMrevrangeCount {
	c.cs.s = append(c.cs.s, "COUNT", strconv.FormatInt(count, 10))
	return (TsMrevrangeCount)(c)
}

func (c TsMrevrangeWithlabels) Align(value string) TsMrevrangeAlign {
	c.cs.s = append(c.cs.s, "ALIGN", value)
	return (TsMrevrangeAlign)(c)
}

func (c TsMrevrangeWithlabels) AggregationAvg() TsMrevrangeAggregationAggregationAvg {
	c.cs.s = append(c.cs.s, "AGGREGATION", "AVG")
	return (TsMrevrangeAggregationAggregationAvg)(c)
}

func (c TsMrevrangeWithlabels) AggregationSum() TsMrevrangeAggregationAggregationSum {
	c.cs.s = append(c.cs.s, "AGGREGATION", "SUM")
	return (TsMrevrangeAggregationAggregationSum)(c)
}

func (c TsMrevrangeWithlabels) AggregationMin() TsMrevrangeAggregationAggregationMin {
	c.cs.s = append(c.cs.s, "AGGREGATION", "MIN")
	return (TsMrevrangeAggregationAggregationMin)(c)
}

func (c TsMrevrangeWithlabels) AggregationMax() TsMrevrangeAggregationAggregationMax {
	c.cs.s = append(c.cs.s, "AGGREGATION", "MAX")
	return (TsMrevrangeAggregationAggregationMax)(c)
}

func (c TsMrevrangeWithlabels) AggregationRange() TsMrevrangeAggregationAggregationRange {
	c.cs.s = append(c.cs.s, "AGGREGATION", "RANGE")
	return (TsMrevrangeAggregationAggregationRange)(c)
}

func (c TsMrevrangeWithlabels) AggregationCount() TsMrevrangeAggregationAggregationCount {
	c.cs.s = append(c.cs.s, "AGGREGATION", "COUNT")
	return (TsMrevrangeAggregationAggregationCount)(c)
}

func (c TsMrevrangeWithlabels) AggregationFirst() TsMrevrangeAggregationAggregationFirst {
	c.cs.s = append(c.cs.s, "AGGREGATION", "FIRST")
	return (TsMrevrangeAggregationAggregationFirst)(c)
}

func (c TsMrevrangeWithlabels) AggregationLast() TsMrevrangeAggregationAggregationLast {
	c.cs.s = append(c.cs.s, "AGGREGATION", "LAST")
	return (TsMrevrangeAggregationAggregationLast)(c)
}

func (c TsMrevrangeWithlabels) AggregationStdP() TsMrevrangeAggregationAggregationStdP {
	c.cs.s = append(c.cs.s, "AGGREGATION", "STD.P")
	return (TsMrevrangeAggregationAggregationStdP)(c)
}

func (c TsMrevrangeWithlabels) AggregationStdS() TsMrevrangeAggregationAggregationStdS {
	c.cs.s = append(c.cs.s, "AGGREGATION", "STD.S")
	return (TsMrevrangeAggregationAggregationStdS)(c)
}

func (c TsMrevrangeWithlabels) AggregationVarP() TsMrevrangeAggregationAggregationVarP {
	c.cs.s = append(c.cs.s, "AGGREGATION", "VAR.P")
	return (TsMrevrangeAggregationAggregationVarP)(c)
}

func (c TsMrevrangeWithlabels) AggregationVarS() TsMrevrangeAggregationAggregationVarS {
	c.cs.s = append(c.cs.s, "AGGREGATION", "VAR.S")
	return (TsMrevrangeAggregationAggregationVarS)(c)
}

func (c TsMrevrangeWithlabels) AggregationTwa() TsMrevrangeAggregationAggregationTwa {
	c.cs.s = append(c.cs.s, "AGGREGATION", "TWA")
	return (TsMrevrangeAggregationAggregationTwa)(c)
}

func (c TsMrevrangeWithlabels) Filter(filter ...string) TsMrevrangeFilter {
	c.cs.s = append(c.cs.s, "FILTER")
	c.cs.s = append(c.cs.s, filter...)
	return (TsMrevrangeFilter)(c)
}

type TsQueryindex Completed

func (b Builder) TsQueryindex() (c TsQueryindex) {
	c = TsQueryindex{cs: get(), ks: b.ks, cf: readonly}
	c.cs.s = append(c.cs.s, "TS.QUERYINDEX")
	return c
}

func (c TsQueryindex) Filter(filter ...string) TsQueryindexFilter {
	c.cs.s = append(c.cs.s, filter...)
	return (TsQueryindexFilter)(c)
}

type TsQueryindexFilter Completed

func (c TsQueryindexFilter) Filter(filter ...string) TsQueryindexFilter {
	c.cs.s = append(c.cs.s, filter...)
	return c
}

func (c TsQueryindexFilter) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type TsRange Completed

func (b Builder) TsRange() (c TsRange) {
	c = TsRange{cs: get(), ks: b.ks, cf: readonly}
	c.cs.s = append(c.cs.s, "TS.RANGE")
	return c
}

func (c TsRange) Key(key string) TsRangeKey {
	if c.ks&NoSlot == NoSlot {
		c.ks = NoSlot | slot(key)
	} else {
		c.ks = check(c.ks, slot(key))
	}
	c.cs.s = append(c.cs.s, key)
	return (TsRangeKey)(c)
}

type TsRangeAggregationAggregationAvg Completed

func (c TsRangeAggregationAggregationAvg) Bucketduration(bucketduration int64) TsRangeAggregationBucketduration {
	c.cs.s = append(c.cs.s, strconv.FormatInt(bucketduration, 10))
	return (TsRangeAggregationBucketduration)(c)
}

type TsRangeAggregationAggregationCount Completed

func (c TsRangeAggregationAggregationCount) Bucketduration(bucketduration int64) TsRangeAggregationBucketduration {
	c.cs.s = append(c.cs.s, strconv.FormatInt(bucketduration, 10))
	return (TsRangeAggregationBucketduration)(c)
}

type TsRangeAggregationAggregationFirst Completed

func (c TsRangeAggregationAggregationFirst) Bucketduration(bucketduration int64) TsRangeAggregationBucketduration {
	c.cs.s = append(c.cs.s, strconv.FormatInt(bucketduration, 10))
	return (TsRangeAggregationBucketduration)(c)
}

type TsRangeAggregationAggregationLast Completed

func (c TsRangeAggregationAggregationLast) Bucketduration(bucketduration int64) TsRangeAggregationBucketduration {
	c.cs.s = append(c.cs.s, strconv.FormatInt(bucketduration, 10))
	return (TsRangeAggregationBucketduration)(c)
}

type TsRangeAggregationAggregationMax Completed

func (c TsRangeAggregationAggregationMax) Bucketduration(bucketduration int64) TsRangeAggregationBucketduration {
	c.cs.s = append(c.cs.s, strconv.FormatInt(bucketduration, 10))
	return (TsRangeAggregationBucketduration)(c)
}

type TsRangeAggregationAggregationMin Completed

func (c TsRangeAggregationAggregationMin) Bucketduration(bucketduration int64) TsRangeAggregationBucketduration {
	c.cs.s = append(c.cs.s, strconv.FormatInt(bucketduration, 10))
	return (TsRangeAggregationBucketduration)(c)
}

type TsRangeAggregationAggregationRange Completed

func (c TsRangeAggregationAggregationRange) Bucketduration(bucketduration int64) TsRangeAggregationBucketduration {
	c.cs.s = append(c.cs.s, strconv.FormatInt(bucketduration, 10))
	return (TsRangeAggregationBucketduration)(c)
}

type TsRangeAggregationAggregationStdP Completed

func (c TsRangeAggregationAggregationStdP) Bucketduration(bucketduration int64) TsRangeAggregationBucketduration {
	c.cs.s = append(c.cs.s, strconv.FormatInt(bucketduration, 10))
	return (TsRangeAggregationBucketduration)(c)
}

type TsRangeAggregationAggregationStdS Completed

func (c TsRangeAggregationAggregationStdS) Bucketduration(bucketduration int64) TsRangeAggregationBucketduration {
	c.cs.s = append(c.cs.s, strconv.FormatInt(bucketduration, 10))
	return (TsRangeAggregationBucketduration)(c)
}

type TsRangeAggregationAggregationSum Completed

func (c TsRangeAggregationAggregationSum) Bucketduration(bucketduration int64) TsRangeAggregationBucketduration {
	c.cs.s = append(c.cs.s, strconv.FormatInt(bucketduration, 10))
	return (TsRangeAggregationBucketduration)(c)
}

type TsRangeAggregationAggregationTwa Completed

func (c TsRangeAggregationAggregationTwa) Bucketduration(bucketduration int64) TsRangeAggregationBucketduration {
	c.cs.s = append(c.cs.s, strconv.FormatInt(bucketduration, 10))
	return (TsRangeAggregationBucketduration)(c)
}

type TsRangeAggregationAggregationVarP Completed

func (c TsRangeAggregationAggregationVarP) Bucketduration(bucketduration int64) TsRangeAggregationBucketduration {
	c.cs.s = append(c.cs.s, strconv.FormatInt(bucketduration, 10))
	return (TsRangeAggregationBucketduration)(c)
}

type TsRangeAggregationAggregationVarS Completed

func (c TsRangeAggregationAggregationVarS) Bucketduration(bucketduration int64) TsRangeAggregationBucketduration {
	c.cs.s = append(c.cs.s, strconv.FormatInt(bucketduration, 10))
	return (TsRangeAggregationBucketduration)(c)
}

type TsRangeAggregationBucketduration Completed

func (c TsRangeAggregationBucketduration) Buckettimestamp(buckettimestamp string) TsRangeAggregationBuckettimestamp {
	c.cs.s = append(c.cs.s, "BUCKETTIMESTAMP", buckettimestamp)
	return (TsRangeAggregationBuckettimestamp)(c)
}

func (c TsRangeAggregationBucketduration) Empty() TsRangeAggregationEmpty {
	c.cs.s = append(c.cs.s, "EMPTY")
	return (TsRangeAggregationEmpty)(c)
}

func (c TsRangeAggregationBucketduration) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type TsRangeAggregationBuckettimestamp Completed

func (c TsRangeAggregationBuckettimestamp) Empty() TsRangeAggregationEmpty {
	c.cs.s = append(c.cs.s, "EMPTY")
	return (TsRangeAggregationEmpty)(c)
}

func (c TsRangeAggregationBuckettimestamp) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type TsRangeAggregationEmpty Completed

func (c TsRangeAggregationEmpty) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type TsRangeAlign Completed

func (c TsRangeAlign) AggregationAvg() TsRangeAggregationAggregationAvg {
	c.cs.s = append(c.cs.s, "AGGREGATION", "AVG")
	return (TsRangeAggregationAggregationAvg)(c)
}

func (c TsRangeAlign) AggregationSum() TsRangeAggregationAggregationSum {
	c.cs.s = append(c.cs.s, "AGGREGATION", "SUM")
	return (TsRangeAggregationAggregationSum)(c)
}

func (c TsRangeAlign) AggregationMin() TsRangeAggregationAggregationMin {
	c.cs.s = append(c.cs.s, "AGGREGATION", "MIN")
	return (TsRangeAggregationAggregationMin)(c)
}

func (c TsRangeAlign) AggregationMax() TsRangeAggregationAggregationMax {
	c.cs.s = append(c.cs.s, "AGGREGATION", "MAX")
	return (TsRangeAggregationAggregationMax)(c)
}

func (c TsRangeAlign) AggregationRange() TsRangeAggregationAggregationRange {
	c.cs.s = append(c.cs.s, "AGGREGATION", "RANGE")
	return (TsRangeAggregationAggregationRange)(c)
}

func (c TsRangeAlign) AggregationCount() TsRangeAggregationAggregationCount {
	c.cs.s = append(c.cs.s, "AGGREGATION", "COUNT")
	return (TsRangeAggregationAggregationCount)(c)
}

func (c TsRangeAlign) AggregationFirst() TsRangeAggregationAggregationFirst {
	c.cs.s = append(c.cs.s, "AGGREGATION", "FIRST")
	return (TsRangeAggregationAggregationFirst)(c)
}

func (c TsRangeAlign) AggregationLast() TsRangeAggregationAggregationLast {
	c.cs.s = append(c.cs.s, "AGGREGATION", "LAST")
	return (TsRangeAggregationAggregationLast)(c)
}

func (c TsRangeAlign) AggregationStdP() TsRangeAggregationAggregationStdP {
	c.cs.s = append(c.cs.s, "AGGREGATION", "STD.P")
	return (TsRangeAggregationAggregationStdP)(c)
}

func (c TsRangeAlign) AggregationStdS() TsRangeAggregationAggregationStdS {
	c.cs.s = append(c.cs.s, "AGGREGATION", "STD.S")
	return (TsRangeAggregationAggregationStdS)(c)
}

func (c TsRangeAlign) AggregationVarP() TsRangeAggregationAggregationVarP {
	c.cs.s = append(c.cs.s, "AGGREGATION", "VAR.P")
	return (TsRangeAggregationAggregationVarP)(c)
}

func (c TsRangeAlign) AggregationVarS() TsRangeAggregationAggregationVarS {
	c.cs.s = append(c.cs.s, "AGGREGATION", "VAR.S")
	return (TsRangeAggregationAggregationVarS)(c)
}

func (c TsRangeAlign) AggregationTwa() TsRangeAggregationAggregationTwa {
	c.cs.s = append(c.cs.s, "AGGREGATION", "TWA")
	return (TsRangeAggregationAggregationTwa)(c)
}

func (c TsRangeAlign) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type TsRangeCount Completed

func (c TsRangeCount) Align(value string) TsRangeAlign {
	c.cs.s = append(c.cs.s, "ALIGN", value)
	return (TsRangeAlign)(c)
}

func (c TsRangeCount) AggregationAvg() TsRangeAggregationAggregationAvg {
	c.cs.s = append(c.cs.s, "AGGREGATION", "AVG")
	return (TsRangeAggregationAggregationAvg)(c)
}

func (c TsRangeCount) AggregationSum() TsRangeAggregationAggregationSum {
	c.cs.s = append(c.cs.s, "AGGREGATION", "SUM")
	return (TsRangeAggregationAggregationSum)(c)
}

func (c TsRangeCount) AggregationMin() TsRangeAggregationAggregationMin {
	c.cs.s = append(c.cs.s, "AGGREGATION", "MIN")
	return (TsRangeAggregationAggregationMin)(c)
}

func (c TsRangeCount) AggregationMax() TsRangeAggregationAggregationMax {
	c.cs.s = append(c.cs.s, "AGGREGATION", "MAX")
	return (TsRangeAggregationAggregationMax)(c)
}

func (c TsRangeCount) AggregationRange() TsRangeAggregationAggregationRange {
	c.cs.s = append(c.cs.s, "AGGREGATION", "RANGE")
	return (TsRangeAggregationAggregationRange)(c)
}

func (c TsRangeCount) AggregationCount() TsRangeAggregationAggregationCount {
	c.cs.s = append(c.cs.s, "AGGREGATION", "COUNT")
	return (TsRangeAggregationAggregationCount)(c)
}

func (c TsRangeCount) AggregationFirst() TsRangeAggregationAggregationFirst {
	c.cs.s = append(c.cs.s, "AGGREGATION", "FIRST")
	return (TsRangeAggregationAggregationFirst)(c)
}

func (c TsRangeCount) AggregationLast() TsRangeAggregationAggregationLast {
	c.cs.s = append(c.cs.s, "AGGREGATION", "LAST")
	return (TsRangeAggregationAggregationLast)(c)
}

func (c TsRangeCount) AggregationStdP() TsRangeAggregationAggregationStdP {
	c.cs.s = append(c.cs.s, "AGGREGATION", "STD.P")
	return (TsRangeAggregationAggregationStdP)(c)
}

func (c TsRangeCount) AggregationStdS() TsRangeAggregationAggregationStdS {
	c.cs.s = append(c.cs.s, "AGGREGATION", "STD.S")
	return (TsRangeAggregationAggregationStdS)(c)
}

func (c TsRangeCount) AggregationVarP() TsRangeAggregationAggregationVarP {
	c.cs.s = append(c.cs.s, "AGGREGATION", "VAR.P")
	return (TsRangeAggregationAggregationVarP)(c)
}

func (c TsRangeCount) AggregationVarS() TsRangeAggregationAggregationVarS {
	c.cs.s = append(c.cs.s, "AGGREGATION", "VAR.S")
	return (TsRangeAggregationAggregationVarS)(c)
}

func (c TsRangeCount) AggregationTwa() TsRangeAggregationAggregationTwa {
	c.cs.s = append(c.cs.s, "AGGREGATION", "TWA")
	return (TsRangeAggregationAggregationTwa)(c)
}

func (c TsRangeCount) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type TsRangeFilterByTs Completed

func (c TsRangeFilterByTs) FilterByTs(timestamp ...int64) TsRangeFilterByTs {
	c.cs.s = append(c.cs.s, "FILTER_BY_TS")
	for _, n := range timestamp {
		c.cs.s = append(c.cs.s, strconv.FormatInt(n, 10))
	}
	return c
}

func (c TsRangeFilterByTs) FilterByValue(min float64, max float64) TsRangeFilterByValue {
	c.cs.s = append(c.cs.s, "FILTER_BY_VALUE", strconv.FormatFloat(min, 'f', -1, 64), strconv.FormatFloat(max, 'f', -1, 64))
	return (TsRangeFilterByValue)(c)
}

func (c TsRangeFilterByTs) Count(count int64) TsRangeCount {
	c.cs.s = append(c.cs.s, "COUNT", strconv.FormatInt(count, 10))
	return (TsRangeCount)(c)
}

func (c TsRangeFilterByTs) Align(value string) TsRangeAlign {
	c.cs.s = append(c.cs.s, "ALIGN", value)
	return (TsRangeAlign)(c)
}

func (c TsRangeFilterByTs) AggregationAvg() TsRangeAggregationAggregationAvg {
	c.cs.s = append(c.cs.s, "AGGREGATION", "AVG")
	return (TsRangeAggregationAggregationAvg)(c)
}

func (c TsRangeFilterByTs) AggregationSum() TsRangeAggregationAggregationSum {
	c.cs.s = append(c.cs.s, "AGGREGATION", "SUM")
	return (TsRangeAggregationAggregationSum)(c)
}

func (c TsRangeFilterByTs) AggregationMin() TsRangeAggregationAggregationMin {
	c.cs.s = append(c.cs.s, "AGGREGATION", "MIN")
	return (TsRangeAggregationAggregationMin)(c)
}

func (c TsRangeFilterByTs) AggregationMax() TsRangeAggregationAggregationMax {
	c.cs.s = append(c.cs.s, "AGGREGATION", "MAX")
	return (TsRangeAggregationAggregationMax)(c)
}

func (c TsRangeFilterByTs) AggregationRange() TsRangeAggregationAggregationRange {
	c.cs.s = append(c.cs.s, "AGGREGATION", "RANGE")
	return (TsRangeAggregationAggregationRange)(c)
}

func (c TsRangeFilterByTs) AggregationCount() TsRangeAggregationAggregationCount {
	c.cs.s = append(c.cs.s, "AGGREGATION", "COUNT")
	return (TsRangeAggregationAggregationCount)(c)
}

func (c TsRangeFilterByTs) AggregationFirst() TsRangeAggregationAggregationFirst {
	c.cs.s = append(c.cs.s, "AGGREGATION", "FIRST")
	return (TsRangeAggregationAggregationFirst)(c)
}

func (c TsRangeFilterByTs) AggregationLast() TsRangeAggregationAggregationLast {
	c.cs.s = append(c.cs.s, "AGGREGATION", "LAST")
	return (TsRangeAggregationAggregationLast)(c)
}

func (c TsRangeFilterByTs) AggregationStdP() TsRangeAggregationAggregationStdP {
	c.cs.s = append(c.cs.s, "AGGREGATION", "STD.P")
	return (TsRangeAggregationAggregationStdP)(c)
}

func (c TsRangeFilterByTs) AggregationStdS() TsRangeAggregationAggregationStdS {
	c.cs.s = append(c.cs.s, "AGGREGATION", "STD.S")
	return (TsRangeAggregationAggregationStdS)(c)
}

func (c TsRangeFilterByTs) AggregationVarP() TsRangeAggregationAggregationVarP {
	c.cs.s = append(c.cs.s, "AGGREGATION", "VAR.P")
	return (TsRangeAggregationAggregationVarP)(c)
}

func (c TsRangeFilterByTs) AggregationVarS() TsRangeAggregationAggregationVarS {
	c.cs.s = append(c.cs.s, "AGGREGATION", "VAR.S")
	return (TsRangeAggregationAggregationVarS)(c)
}

func (c TsRangeFilterByTs) AggregationTwa() TsRangeAggregationAggregationTwa {
	c.cs.s = append(c.cs.s, "AGGREGATION", "TWA")
	return (TsRangeAggregationAggregationTwa)(c)
}

func (c TsRangeFilterByTs) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type TsRangeFilterByValue Completed

func (c TsRangeFilterByValue) Count(count int64) TsRangeCount {
	c.cs.s = append(c.cs.s, "COUNT", strconv.FormatInt(count, 10))
	return (TsRangeCount)(c)
}

func (c TsRangeFilterByValue) Align(value string) TsRangeAlign {
	c.cs.s = append(c.cs.s, "ALIGN", value)
	return (TsRangeAlign)(c)
}

func (c TsRangeFilterByValue) AggregationAvg() TsRangeAggregationAggregationAvg {
	c.cs.s = append(c.cs.s, "AGGREGATION", "AVG")
	return (TsRangeAggregationAggregationAvg)(c)
}

func (c TsRangeFilterByValue) AggregationSum() TsRangeAggregationAggregationSum {
	c.cs.s = append(c.cs.s, "AGGREGATION", "SUM")
	return (TsRangeAggregationAggregationSum)(c)
}

func (c TsRangeFilterByValue) AggregationMin() TsRangeAggregationAggregationMin {
	c.cs.s = append(c.cs.s, "AGGREGATION", "MIN")
	return (TsRangeAggregationAggregationMin)(c)
}

func (c TsRangeFilterByValue) AggregationMax() TsRangeAggregationAggregationMax {
	c.cs.s = append(c.cs.s, "AGGREGATION", "MAX")
	return (TsRangeAggregationAggregationMax)(c)
}

func (c TsRangeFilterByValue) AggregationRange() TsRangeAggregationAggregationRange {
	c.cs.s = append(c.cs.s, "AGGREGATION", "RANGE")
	return (TsRangeAggregationAggregationRange)(c)
}

func (c TsRangeFilterByValue) AggregationCount() TsRangeAggregationAggregationCount {
	c.cs.s = append(c.cs.s, "AGGREGATION", "COUNT")
	return (TsRangeAggregationAggregationCount)(c)
}

func (c TsRangeFilterByValue) AggregationFirst() TsRangeAggregationAggregationFirst {
	c.cs.s = append(c.cs.s, "AGGREGATION", "FIRST")
	return (TsRangeAggregationAggregationFirst)(c)
}

func (c TsRangeFilterByValue) AggregationLast() TsRangeAggregationAggregationLast {
	c.cs.s = append(c.cs.s, "AGGREGATION", "LAST")
	return (TsRangeAggregationAggregationLast)(c)
}

func (c TsRangeFilterByValue) AggregationStdP() TsRangeAggregationAggregationStdP {
	c.cs.s = append(c.cs.s, "AGGREGATION", "STD.P")
	return (TsRangeAggregationAggregationStdP)(c)
}

func (c TsRangeFilterByValue) AggregationStdS() TsRangeAggregationAggregationStdS {
	c.cs.s = append(c.cs.s, "AGGREGATION", "STD.S")
	return (TsRangeAggregationAggregationStdS)(c)
}

func (c TsRangeFilterByValue) AggregationVarP() TsRangeAggregationAggregationVarP {
	c.cs.s = append(c.cs.s, "AGGREGATION", "VAR.P")
	return (TsRangeAggregationAggregationVarP)(c)
}

func (c TsRangeFilterByValue) AggregationVarS() TsRangeAggregationAggregationVarS {
	c.cs.s = append(c.cs.s, "AGGREGATION", "VAR.S")
	return (TsRangeAggregationAggregationVarS)(c)
}

func (c TsRangeFilterByValue) AggregationTwa() TsRangeAggregationAggregationTwa {
	c.cs.s = append(c.cs.s, "AGGREGATION", "TWA")
	return (TsRangeAggregationAggregationTwa)(c)
}

func (c TsRangeFilterByValue) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type TsRangeFromtimestamp Completed

func (c TsRangeFromtimestamp) Totimestamp(totimestamp int64) TsRangeTotimestamp {
	c.cs.s = append(c.cs.s, strconv.FormatInt(totimestamp, 10))
	return (TsRangeTotimestamp)(c)
}

type TsRangeKey Completed

func (c TsRangeKey) Fromtimestamp(fromtimestamp int64) TsRangeFromtimestamp {
	c.cs.s = append(c.cs.s, strconv.FormatInt(fromtimestamp, 10))
	return (TsRangeFromtimestamp)(c)
}

type TsRangeLatest Completed

func (c TsRangeLatest) FilterByTs(timestamp ...int64) TsRangeFilterByTs {
	c.cs.s = append(c.cs.s, "FILTER_BY_TS")
	for _, n := range timestamp {
		c.cs.s = append(c.cs.s, strconv.FormatInt(n, 10))
	}
	return (TsRangeFilterByTs)(c)
}

func (c TsRangeLatest) FilterByValue(min float64, max float64) TsRangeFilterByValue {
	c.cs.s = append(c.cs.s, "FILTER_BY_VALUE", strconv.FormatFloat(min, 'f', -1, 64), strconv.FormatFloat(max, 'f', -1, 64))
	return (TsRangeFilterByValue)(c)
}

func (c TsRangeLatest) Count(count int64) TsRangeCount {
	c.cs.s = append(c.cs.s, "COUNT", strconv.FormatInt(count, 10))
	return (TsRangeCount)(c)
}

func (c TsRangeLatest) Align(value string) TsRangeAlign {
	c.cs.s = append(c.cs.s, "ALIGN", value)
	return (TsRangeAlign)(c)
}

func (c TsRangeLatest) AggregationAvg() TsRangeAggregationAggregationAvg {
	c.cs.s = append(c.cs.s, "AGGREGATION", "AVG")
	return (TsRangeAggregationAggregationAvg)(c)
}

func (c TsRangeLatest) AggregationSum() TsRangeAggregationAggregationSum {
	c.cs.s = append(c.cs.s, "AGGREGATION", "SUM")
	return (TsRangeAggregationAggregationSum)(c)
}

func (c TsRangeLatest) AggregationMin() TsRangeAggregationAggregationMin {
	c.cs.s = append(c.cs.s, "AGGREGATION", "MIN")
	return (TsRangeAggregationAggregationMin)(c)
}

func (c TsRangeLatest) AggregationMax() TsRangeAggregationAggregationMax {
	c.cs.s = append(c.cs.s, "AGGREGATION", "MAX")
	return (TsRangeAggregationAggregationMax)(c)
}

func (c TsRangeLatest) AggregationRange() TsRangeAggregationAggregationRange {
	c.cs.s = append(c.cs.s, "AGGREGATION", "RANGE")
	return (TsRangeAggregationAggregationRange)(c)
}

func (c TsRangeLatest) AggregationCount() TsRangeAggregationAggregationCount {
	c.cs.s = append(c.cs.s, "AGGREGATION", "COUNT")
	return (TsRangeAggregationAggregationCount)(c)
}

func (c TsRangeLatest) AggregationFirst() TsRangeAggregationAggregationFirst {
	c.cs.s = append(c.cs.s, "AGGREGATION", "FIRST")
	return (TsRangeAggregationAggregationFirst)(c)
}

func (c TsRangeLatest) AggregationLast() TsRangeAggregationAggregationLast {
	c.cs.s = append(c.cs.s, "AGGREGATION", "LAST")
	return (TsRangeAggregationAggregationLast)(c)
}

func (c TsRangeLatest) AggregationStdP() TsRangeAggregationAggregationStdP {
	c.cs.s = append(c.cs.s, "AGGREGATION", "STD.P")
	return (TsRangeAggregationAggregationStdP)(c)
}

func (c TsRangeLatest) AggregationStdS() TsRangeAggregationAggregationStdS {
	c.cs.s = append(c.cs.s, "AGGREGATION", "STD.S")
	return (TsRangeAggregationAggregationStdS)(c)
}

func (c TsRangeLatest) AggregationVarP() TsRangeAggregationAggregationVarP {
	c.cs.s = append(c.cs.s, "AGGREGATION", "VAR.P")
	return (TsRangeAggregationAggregationVarP)(c)
}

func (c TsRangeLatest) AggregationVarS() TsRangeAggregationAggregationVarS {
	c.cs.s = append(c.cs.s, "AGGREGATION", "VAR.S")
	return (TsRangeAggregationAggregationVarS)(c)
}

func (c TsRangeLatest) AggregationTwa() TsRangeAggregationAggregationTwa {
	c.cs.s = append(c.cs.s, "AGGREGATION", "TWA")
	return (TsRangeAggregationAggregationTwa)(c)
}

func (c TsRangeLatest) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type TsRangeTotimestamp Completed

func (c TsRangeTotimestamp) Latest() TsRangeLatest {
	c.cs.s = append(c.cs.s, "LATEST")
	return (TsRangeLatest)(c)
}

func (c TsRangeTotimestamp) FilterByTs(timestamp ...int64) TsRangeFilterByTs {
	c.cs.s = append(c.cs.s, "FILTER_BY_TS")
	for _, n := range timestamp {
		c.cs.s = append(c.cs.s, strconv.FormatInt(n, 10))
	}
	return (TsRangeFilterByTs)(c)
}

func (c TsRangeTotimestamp) FilterByValue(min float64, max float64) TsRangeFilterByValue {
	c.cs.s = append(c.cs.s, "FILTER_BY_VALUE", strconv.FormatFloat(min, 'f', -1, 64), strconv.FormatFloat(max, 'f', -1, 64))
	return (TsRangeFilterByValue)(c)
}

func (c TsRangeTotimestamp) Count(count int64) TsRangeCount {
	c.cs.s = append(c.cs.s, "COUNT", strconv.FormatInt(count, 10))
	return (TsRangeCount)(c)
}

func (c TsRangeTotimestamp) Align(value string) TsRangeAlign {
	c.cs.s = append(c.cs.s, "ALIGN", value)
	return (TsRangeAlign)(c)
}

func (c TsRangeTotimestamp) AggregationAvg() TsRangeAggregationAggregationAvg {
	c.cs.s = append(c.cs.s, "AGGREGATION", "AVG")
	return (TsRangeAggregationAggregationAvg)(c)
}

func (c TsRangeTotimestamp) AggregationSum() TsRangeAggregationAggregationSum {
	c.cs.s = append(c.cs.s, "AGGREGATION", "SUM")
	return (TsRangeAggregationAggregationSum)(c)
}

func (c TsRangeTotimestamp) AggregationMin() TsRangeAggregationAggregationMin {
	c.cs.s = append(c.cs.s, "AGGREGATION", "MIN")
	return (TsRangeAggregationAggregationMin)(c)
}

func (c TsRangeTotimestamp) AggregationMax() TsRangeAggregationAggregationMax {
	c.cs.s = append(c.cs.s, "AGGREGATION", "MAX")
	return (TsRangeAggregationAggregationMax)(c)
}

func (c TsRangeTotimestamp) AggregationRange() TsRangeAggregationAggregationRange {
	c.cs.s = append(c.cs.s, "AGGREGATION", "RANGE")
	return (TsRangeAggregationAggregationRange)(c)
}

func (c TsRangeTotimestamp) AggregationCount() TsRangeAggregationAggregationCount {
	c.cs.s = append(c.cs.s, "AGGREGATION", "COUNT")
	return (TsRangeAggregationAggregationCount)(c)
}

func (c TsRangeTotimestamp) AggregationFirst() TsRangeAggregationAggregationFirst {
	c.cs.s = append(c.cs.s, "AGGREGATION", "FIRST")
	return (TsRangeAggregationAggregationFirst)(c)
}

func (c TsRangeTotimestamp) AggregationLast() TsRangeAggregationAggregationLast {
	c.cs.s = append(c.cs.s, "AGGREGATION", "LAST")
	return (TsRangeAggregationAggregationLast)(c)
}

func (c TsRangeTotimestamp) AggregationStdP() TsRangeAggregationAggregationStdP {
	c.cs.s = append(c.cs.s, "AGGREGATION", "STD.P")
	return (TsRangeAggregationAggregationStdP)(c)
}

func (c TsRangeTotimestamp) AggregationStdS() TsRangeAggregationAggregationStdS {
	c.cs.s = append(c.cs.s, "AGGREGATION", "STD.S")
	return (TsRangeAggregationAggregationStdS)(c)
}

func (c TsRangeTotimestamp) AggregationVarP() TsRangeAggregationAggregationVarP {
	c.cs.s = append(c.cs.s, "AGGREGATION", "VAR.P")
	return (TsRangeAggregationAggregationVarP)(c)
}

func (c TsRangeTotimestamp) AggregationVarS() TsRangeAggregationAggregationVarS {
	c.cs.s = append(c.cs.s, "AGGREGATION", "VAR.S")
	return (TsRangeAggregationAggregationVarS)(c)
}

func (c TsRangeTotimestamp) AggregationTwa() TsRangeAggregationAggregationTwa {
	c.cs.s = append(c.cs.s, "AGGREGATION", "TWA")
	return (TsRangeAggregationAggregationTwa)(c)
}

func (c TsRangeTotimestamp) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type TsRevrange Completed

func (b Builder) TsRevrange() (c TsRevrange) {
	c = TsRevrange{cs: get(), ks: b.ks, cf: readonly}
	c.cs.s = append(c.cs.s, "TS.REVRANGE")
	return c
}

func (c TsRevrange) Key(key string) TsRevrangeKey {
	if c.ks&NoSlot == NoSlot {
		c.ks = NoSlot | slot(key)
	} else {
		c.ks = check(c.ks, slot(key))
	}
	c.cs.s = append(c.cs.s, key)
	return (TsRevrangeKey)(c)
}

type TsRevrangeAggregationAggregationAvg Completed

func (c TsRevrangeAggregationAggregationAvg) Bucketduration(bucketduration int64) TsRevrangeAggregationBucketduration {
	c.cs.s = append(c.cs.s, strconv.FormatInt(bucketduration, 10))
	return (TsRevrangeAggregationBucketduration)(c)
}

type TsRevrangeAggregationAggregationCount Completed

func (c TsRevrangeAggregationAggregationCount) Bucketduration(bucketduration int64) TsRevrangeAggregationBucketduration {
	c.cs.s = append(c.cs.s, strconv.FormatInt(bucketduration, 10))
	return (TsRevrangeAggregationBucketduration)(c)
}

type TsRevrangeAggregationAggregationFirst Completed

func (c TsRevrangeAggregationAggregationFirst) Bucketduration(bucketduration int64) TsRevrangeAggregationBucketduration {
	c.cs.s = append(c.cs.s, strconv.FormatInt(bucketduration, 10))
	return (TsRevrangeAggregationBucketduration)(c)
}

type TsRevrangeAggregationAggregationLast Completed

func (c TsRevrangeAggregationAggregationLast) Bucketduration(bucketduration int64) TsRevrangeAggregationBucketduration {
	c.cs.s = append(c.cs.s, strconv.FormatInt(bucketduration, 10))
	return (TsRevrangeAggregationBucketduration)(c)
}

type TsRevrangeAggregationAggregationMax Completed

func (c TsRevrangeAggregationAggregationMax) Bucketduration(bucketduration int64) TsRevrangeAggregationBucketduration {
	c.cs.s = append(c.cs.s, strconv.FormatInt(bucketduration, 10))
	return (TsRevrangeAggregationBucketduration)(c)
}

type TsRevrangeAggregationAggregationMin Completed

func (c TsRevrangeAggregationAggregationMin) Bucketduration(bucketduration int64) TsRevrangeAggregationBucketduration {
	c.cs.s = append(c.cs.s, strconv.FormatInt(bucketduration, 10))
	return (TsRevrangeAggregationBucketduration)(c)
}

type TsRevrangeAggregationAggregationRange Completed

func (c TsRevrangeAggregationAggregationRange) Bucketduration(bucketduration int64) TsRevrangeAggregationBucketduration {
	c.cs.s = append(c.cs.s, strconv.FormatInt(bucketduration, 10))
	return (TsRevrangeAggregationBucketduration)(c)
}

type TsRevrangeAggregationAggregationStdP Completed

func (c TsRevrangeAggregationAggregationStdP) Bucketduration(bucketduration int64) TsRevrangeAggregationBucketduration {
	c.cs.s = append(c.cs.s, strconv.FormatInt(bucketduration, 10))
	return (TsRevrangeAggregationBucketduration)(c)
}

type TsRevrangeAggregationAggregationStdS Completed

func (c TsRevrangeAggregationAggregationStdS) Bucketduration(bucketduration int64) TsRevrangeAggregationBucketduration {
	c.cs.s = append(c.cs.s, strconv.FormatInt(bucketduration, 10))
	return (TsRevrangeAggregationBucketduration)(c)
}

type TsRevrangeAggregationAggregationSum Completed

func (c TsRevrangeAggregationAggregationSum) Bucketduration(bucketduration int64) TsRevrangeAggregationBucketduration {
	c.cs.s = append(c.cs.s, strconv.FormatInt(bucketduration, 10))
	return (TsRevrangeAggregationBucketduration)(c)
}

type TsRevrangeAggregationAggregationTwa Completed

func (c TsRevrangeAggregationAggregationTwa) Bucketduration(bucketduration int64) TsRevrangeAggregationBucketduration {
	c.cs.s = append(c.cs.s, strconv.FormatInt(bucketduration, 10))
	return (TsRevrangeAggregationBucketduration)(c)
}

type TsRevrangeAggregationAggregationVarP Completed

func (c TsRevrangeAggregationAggregationVarP) Bucketduration(bucketduration int64) TsRevrangeAggregationBucketduration {
	c.cs.s = append(c.cs.s, strconv.FormatInt(bucketduration, 10))
	return (TsRevrangeAggregationBucketduration)(c)
}

type TsRevrangeAggregationAggregationVarS Completed

func (c TsRevrangeAggregationAggregationVarS) Bucketduration(bucketduration int64) TsRevrangeAggregationBucketduration {
	c.cs.s = append(c.cs.s, strconv.FormatInt(bucketduration, 10))
	return (TsRevrangeAggregationBucketduration)(c)
}

type TsRevrangeAggregationBucketduration Completed

func (c TsRevrangeAggregationBucketduration) Buckettimestamp(buckettimestamp string) TsRevrangeAggregationBuckettimestamp {
	c.cs.s = append(c.cs.s, "BUCKETTIMESTAMP", buckettimestamp)
	return (TsRevrangeAggregationBuckettimestamp)(c)
}

func (c TsRevrangeAggregationBucketduration) Empty() TsRevrangeAggregationEmpty {
	c.cs.s = append(c.cs.s, "EMPTY")
	return (TsRevrangeAggregationEmpty)(c)
}

func (c TsRevrangeAggregationBucketduration) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type TsRevrangeAggregationBuckettimestamp Completed

func (c TsRevrangeAggregationBuckettimestamp) Empty() TsRevrangeAggregationEmpty {
	c.cs.s = append(c.cs.s, "EMPTY")
	return (TsRevrangeAggregationEmpty)(c)
}

func (c TsRevrangeAggregationBuckettimestamp) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type TsRevrangeAggregationEmpty Completed

func (c TsRevrangeAggregationEmpty) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type TsRevrangeAlign Completed

func (c TsRevrangeAlign) AggregationAvg() TsRevrangeAggregationAggregationAvg {
	c.cs.s = append(c.cs.s, "AGGREGATION", "AVG")
	return (TsRevrangeAggregationAggregationAvg)(c)
}

func (c TsRevrangeAlign) AggregationSum() TsRevrangeAggregationAggregationSum {
	c.cs.s = append(c.cs.s, "AGGREGATION", "SUM")
	return (TsRevrangeAggregationAggregationSum)(c)
}

func (c TsRevrangeAlign) AggregationMin() TsRevrangeAggregationAggregationMin {
	c.cs.s = append(c.cs.s, "AGGREGATION", "MIN")
	return (TsRevrangeAggregationAggregationMin)(c)
}

func (c TsRevrangeAlign) AggregationMax() TsRevrangeAggregationAggregationMax {
	c.cs.s = append(c.cs.s, "AGGREGATION", "MAX")
	return (TsRevrangeAggregationAggregationMax)(c)
}

func (c TsRevrangeAlign) AggregationRange() TsRevrangeAggregationAggregationRange {
	c.cs.s = append(c.cs.s, "AGGREGATION", "RANGE")
	return (TsRevrangeAggregationAggregationRange)(c)
}

func (c TsRevrangeAlign) AggregationCount() TsRevrangeAggregationAggregationCount {
	c.cs.s = append(c.cs.s, "AGGREGATION", "COUNT")
	return (TsRevrangeAggregationAggregationCount)(c)
}

func (c TsRevrangeAlign) AggregationFirst() TsRevrangeAggregationAggregationFirst {
	c.cs.s = append(c.cs.s, "AGGREGATION", "FIRST")
	return (TsRevrangeAggregationAggregationFirst)(c)
}

func (c TsRevrangeAlign) AggregationLast() TsRevrangeAggregationAggregationLast {
	c.cs.s = append(c.cs.s, "AGGREGATION", "LAST")
	return (TsRevrangeAggregationAggregationLast)(c)
}

func (c TsRevrangeAlign) AggregationStdP() TsRevrangeAggregationAggregationStdP {
	c.cs.s = append(c.cs.s, "AGGREGATION", "STD.P")
	return (TsRevrangeAggregationAggregationStdP)(c)
}

func (c TsRevrangeAlign) AggregationStdS() TsRevrangeAggregationAggregationStdS {
	c.cs.s = append(c.cs.s, "AGGREGATION", "STD.S")
	return (TsRevrangeAggregationAggregationStdS)(c)
}

func (c TsRevrangeAlign) AggregationVarP() TsRevrangeAggregationAggregationVarP {
	c.cs.s = append(c.cs.s, "AGGREGATION", "VAR.P")
	return (TsRevrangeAggregationAggregationVarP)(c)
}

func (c TsRevrangeAlign) AggregationVarS() TsRevrangeAggregationAggregationVarS {
	c.cs.s = append(c.cs.s, "AGGREGATION", "VAR.S")
	return (TsRevrangeAggregationAggregationVarS)(c)
}

func (c TsRevrangeAlign) AggregationTwa() TsRevrangeAggregationAggregationTwa {
	c.cs.s = append(c.cs.s, "AGGREGATION", "TWA")
	return (TsRevrangeAggregationAggregationTwa)(c)
}

func (c TsRevrangeAlign) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type TsRevrangeCount Completed

func (c TsRevrangeCount) Align(value string) TsRevrangeAlign {
	c.cs.s = append(c.cs.s, "ALIGN", value)
	return (TsRevrangeAlign)(c)
}

func (c TsRevrangeCount) AggregationAvg() TsRevrangeAggregationAggregationAvg {
	c.cs.s = append(c.cs.s, "AGGREGATION", "AVG")
	return (TsRevrangeAggregationAggregationAvg)(c)
}

func (c TsRevrangeCount) AggregationSum() TsRevrangeAggregationAggregationSum {
	c.cs.s = append(c.cs.s, "AGGREGATION", "SUM")
	return (TsRevrangeAggregationAggregationSum)(c)
}

func (c TsRevrangeCount) AggregationMin() TsRevrangeAggregationAggregationMin {
	c.cs.s = append(c.cs.s, "AGGREGATION", "MIN")
	return (TsRevrangeAggregationAggregationMin)(c)
}

func (c TsRevrangeCount) AggregationMax() TsRevrangeAggregationAggregationMax {
	c.cs.s = append(c.cs.s, "AGGREGATION", "MAX")
	return (TsRevrangeAggregationAggregationMax)(c)
}

func (c TsRevrangeCount) AggregationRange() TsRevrangeAggregationAggregationRange {
	c.cs.s = append(c.cs.s, "AGGREGATION", "RANGE")
	return (TsRevrangeAggregationAggregationRange)(c)
}

func (c TsRevrangeCount) AggregationCount() TsRevrangeAggregationAggregationCount {
	c.cs.s = append(c.cs.s, "AGGREGATION", "COUNT")
	return (TsRevrangeAggregationAggregationCount)(c)
}

func (c TsRevrangeCount) AggregationFirst() TsRevrangeAggregationAggregationFirst {
	c.cs.s = append(c.cs.s, "AGGREGATION", "FIRST")
	return (TsRevrangeAggregationAggregationFirst)(c)
}

func (c TsRevrangeCount) AggregationLast() TsRevrangeAggregationAggregationLast {
	c.cs.s = append(c.cs.s, "AGGREGATION", "LAST")
	return (TsRevrangeAggregationAggregationLast)(c)
}

func (c TsRevrangeCount) AggregationStdP() TsRevrangeAggregationAggregationStdP {
	c.cs.s = append(c.cs.s, "AGGREGATION", "STD.P")
	return (TsRevrangeAggregationAggregationStdP)(c)
}

func (c TsRevrangeCount) AggregationStdS() TsRevrangeAggregationAggregationStdS {
	c.cs.s = append(c.cs.s, "AGGREGATION", "STD.S")
	return (TsRevrangeAggregationAggregationStdS)(c)
}

func (c TsRevrangeCount) AggregationVarP() TsRevrangeAggregationAggregationVarP {
	c.cs.s = append(c.cs.s, "AGGREGATION", "VAR.P")
	return (TsRevrangeAggregationAggregationVarP)(c)
}

func (c TsRevrangeCount) AggregationVarS() TsRevrangeAggregationAggregationVarS {
	c.cs.s = append(c.cs.s, "AGGREGATION", "VAR.S")
	return (TsRevrangeAggregationAggregationVarS)(c)
}

func (c TsRevrangeCount) AggregationTwa() TsRevrangeAggregationAggregationTwa {
	c.cs.s = append(c.cs.s, "AGGREGATION", "TWA")
	return (TsRevrangeAggregationAggregationTwa)(c)
}

func (c TsRevrangeCount) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type TsRevrangeFilterByTs Completed

func (c TsRevrangeFilterByTs) FilterByTs(timestamp ...int64) TsRevrangeFilterByTs {
	c.cs.s = append(c.cs.s, "FILTER_BY_TS")
	for _, n := range timestamp {
		c.cs.s = append(c.cs.s, strconv.FormatInt(n, 10))
	}
	return c
}

func (c TsRevrangeFilterByTs) FilterByValue(min float64, max float64) TsRevrangeFilterByValue {
	c.cs.s = append(c.cs.s, "FILTER_BY_VALUE", strconv.FormatFloat(min, 'f', -1, 64), strconv.FormatFloat(max, 'f', -1, 64))
	return (TsRevrangeFilterByValue)(c)
}

func (c TsRevrangeFilterByTs) Count(count int64) TsRevrangeCount {
	c.cs.s = append(c.cs.s, "COUNT", strconv.FormatInt(count, 10))
	return (TsRevrangeCount)(c)
}

func (c TsRevrangeFilterByTs) Align(value string) TsRevrangeAlign {
	c.cs.s = append(c.cs.s, "ALIGN", value)
	return (TsRevrangeAlign)(c)
}

func (c TsRevrangeFilterByTs) AggregationAvg() TsRevrangeAggregationAggregationAvg {
	c.cs.s = append(c.cs.s, "AGGREGATION", "AVG")
	return (TsRevrangeAggregationAggregationAvg)(c)
}

func (c TsRevrangeFilterByTs) AggregationSum() TsRevrangeAggregationAggregationSum {
	c.cs.s = append(c.cs.s, "AGGREGATION", "SUM")
	return (TsRevrangeAggregationAggregationSum)(c)
}

func (c TsRevrangeFilterByTs) AggregationMin() TsRevrangeAggregationAggregationMin {
	c.cs.s = append(c.cs.s, "AGGREGATION", "MIN")
	return (TsRevrangeAggregationAggregationMin)(c)
}

func (c TsRevrangeFilterByTs) AggregationMax() TsRevrangeAggregationAggregationMax {
	c.cs.s = append(c.cs.s, "AGGREGATION", "MAX")
	return (TsRevrangeAggregationAggregationMax)(c)
}

func (c TsRevrangeFilterByTs) AggregationRange() TsRevrangeAggregationAggregationRange {
	c.cs.s = append(c.cs.s, "AGGREGATION", "RANGE")
	return (TsRevrangeAggregationAggregationRange)(c)
}

func (c TsRevrangeFilterByTs) AggregationCount() TsRevrangeAggregationAggregationCount {
	c.cs.s = append(c.cs.s, "AGGREGATION", "COUNT")
	return (TsRevrangeAggregationAggregationCount)(c)
}

func (c TsRevrangeFilterByTs) AggregationFirst() TsRevrangeAggregationAggregationFirst {
	c.cs.s = append(c.cs.s, "AGGREGATION", "FIRST")
	return (TsRevrangeAggregationAggregationFirst)(c)
}

func (c TsRevrangeFilterByTs) AggregationLast() TsRevrangeAggregationAggregationLast {
	c.cs.s = append(c.cs.s, "AGGREGATION", "LAST")
	return (TsRevrangeAggregationAggregationLast)(c)
}

func (c TsRevrangeFilterByTs) AggregationStdP() TsRevrangeAggregationAggregationStdP {
	c.cs.s = append(c.cs.s, "AGGREGATION", "STD.P")
	return (TsRevrangeAggregationAggregationStdP)(c)
}

func (c TsRevrangeFilterByTs) AggregationStdS() TsRevrangeAggregationAggregationStdS {
	c.cs.s = append(c.cs.s, "AGGREGATION", "STD.S")
	return (TsRevrangeAggregationAggregationStdS)(c)
}

func (c TsRevrangeFilterByTs) AggregationVarP() TsRevrangeAggregationAggregationVarP {
	c.cs.s = append(c.cs.s, "AGGREGATION", "VAR.P")
	return (TsRevrangeAggregationAggregationVarP)(c)
}

func (c TsRevrangeFilterByTs) AggregationVarS() TsRevrangeAggregationAggregationVarS {
	c.cs.s = append(c.cs.s, "AGGREGATION", "VAR.S")
	return (TsRevrangeAggregationAggregationVarS)(c)
}

func (c TsRevrangeFilterByTs) AggregationTwa() TsRevrangeAggregationAggregationTwa {
	c.cs.s = append(c.cs.s, "AGGREGATION", "TWA")
	return (TsRevrangeAggregationAggregationTwa)(c)
}

func (c TsRevrangeFilterByTs) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type TsRevrangeFilterByValue Completed

func (c TsRevrangeFilterByValue) Count(count int64) TsRevrangeCount {
	c.cs.s = append(c.cs.s, "COUNT", strconv.FormatInt(count, 10))
	return (TsRevrangeCount)(c)
}

func (c TsRevrangeFilterByValue) Align(value string) TsRevrangeAlign {
	c.cs.s = append(c.cs.s, "ALIGN", value)
	return (TsRevrangeAlign)(c)
}

func (c TsRevrangeFilterByValue) AggregationAvg() TsRevrangeAggregationAggregationAvg {
	c.cs.s = append(c.cs.s, "AGGREGATION", "AVG")
	return (TsRevrangeAggregationAggregationAvg)(c)
}

func (c TsRevrangeFilterByValue) AggregationSum() TsRevrangeAggregationAggregationSum {
	c.cs.s = append(c.cs.s, "AGGREGATION", "SUM")
	return (TsRevrangeAggregationAggregationSum)(c)
}

func (c TsRevrangeFilterByValue) AggregationMin() TsRevrangeAggregationAggregationMin {
	c.cs.s = append(c.cs.s, "AGGREGATION", "MIN")
	return (TsRevrangeAggregationAggregationMin)(c)
}

func (c TsRevrangeFilterByValue) AggregationMax() TsRevrangeAggregationAggregationMax {
	c.cs.s = append(c.cs.s, "AGGREGATION", "MAX")
	return (TsRevrangeAggregationAggregationMax)(c)
}

func (c TsRevrangeFilterByValue) AggregationRange() TsRevrangeAggregationAggregationRange {
	c.cs.s = append(c.cs.s, "AGGREGATION", "RANGE")
	return (TsRevrangeAggregationAggregationRange)(c)
}

func (c TsRevrangeFilterByValue) AggregationCount() TsRevrangeAggregationAggregationCount {
	c.cs.s = append(c.cs.s, "AGGREGATION", "COUNT")
	return (TsRevrangeAggregationAggregationCount)(c)
}

func (c TsRevrangeFilterByValue) AggregationFirst() TsRevrangeAggregationAggregationFirst {
	c.cs.s = append(c.cs.s, "AGGREGATION", "FIRST")
	return (TsRevrangeAggregationAggregationFirst)(c)
}

func (c TsRevrangeFilterByValue) AggregationLast() TsRevrangeAggregationAggregationLast {
	c.cs.s = append(c.cs.s, "AGGREGATION", "LAST")
	return (TsRevrangeAggregationAggregationLast)(c)
}

func (c TsRevrangeFilterByValue) AggregationStdP() TsRevrangeAggregationAggregationStdP {
	c.cs.s = append(c.cs.s, "AGGREGATION", "STD.P")
	return (TsRevrangeAggregationAggregationStdP)(c)
}

func (c TsRevrangeFilterByValue) AggregationStdS() TsRevrangeAggregationAggregationStdS {
	c.cs.s = append(c.cs.s, "AGGREGATION", "STD.S")
	return (TsRevrangeAggregationAggregationStdS)(c)
}

func (c TsRevrangeFilterByValue) AggregationVarP() TsRevrangeAggregationAggregationVarP {
	c.cs.s = append(c.cs.s, "AGGREGATION", "VAR.P")
	return (TsRevrangeAggregationAggregationVarP)(c)
}

func (c TsRevrangeFilterByValue) AggregationVarS() TsRevrangeAggregationAggregationVarS {
	c.cs.s = append(c.cs.s, "AGGREGATION", "VAR.S")
	return (TsRevrangeAggregationAggregationVarS)(c)
}

func (c TsRevrangeFilterByValue) AggregationTwa() TsRevrangeAggregationAggregationTwa {
	c.cs.s = append(c.cs.s, "AGGREGATION", "TWA")
	return (TsRevrangeAggregationAggregationTwa)(c)
}

func (c TsRevrangeFilterByValue) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type TsRevrangeFromtimestamp Completed

func (c TsRevrangeFromtimestamp) Totimestamp(totimestamp int64) TsRevrangeTotimestamp {
	c.cs.s = append(c.cs.s, strconv.FormatInt(totimestamp, 10))
	return (TsRevrangeTotimestamp)(c)
}

type TsRevrangeKey Completed

func (c TsRevrangeKey) Fromtimestamp(fromtimestamp int64) TsRevrangeFromtimestamp {
	c.cs.s = append(c.cs.s, strconv.FormatInt(fromtimestamp, 10))
	return (TsRevrangeFromtimestamp)(c)
}

type TsRevrangeLatest Completed

func (c TsRevrangeLatest) FilterByTs(timestamp ...int64) TsRevrangeFilterByTs {
	c.cs.s = append(c.cs.s, "FILTER_BY_TS")
	for _, n := range timestamp {
		c.cs.s = append(c.cs.s, strconv.FormatInt(n, 10))
	}
	return (TsRevrangeFilterByTs)(c)
}

func (c TsRevrangeLatest) FilterByValue(min float64, max float64) TsRevrangeFilterByValue {
	c.cs.s = append(c.cs.s, "FILTER_BY_VALUE", strconv.FormatFloat(min, 'f', -1, 64), strconv.FormatFloat(max, 'f', -1, 64))
	return (TsRevrangeFilterByValue)(c)
}

func (c TsRevrangeLatest) Count(count int64) TsRevrangeCount {
	c.cs.s = append(c.cs.s, "COUNT", strconv.FormatInt(count, 10))
	return (TsRevrangeCount)(c)
}

func (c TsRevrangeLatest) Align(value string) TsRevrangeAlign {
	c.cs.s = append(c.cs.s, "ALIGN", value)
	return (TsRevrangeAlign)(c)
}

func (c TsRevrangeLatest) AggregationAvg() TsRevrangeAggregationAggregationAvg {
	c.cs.s = append(c.cs.s, "AGGREGATION", "AVG")
	return (TsRevrangeAggregationAggregationAvg)(c)
}

func (c TsRevrangeLatest) AggregationSum() TsRevrangeAggregationAggregationSum {
	c.cs.s = append(c.cs.s, "AGGREGATION", "SUM")
	return (TsRevrangeAggregationAggregationSum)(c)
}

func (c TsRevrangeLatest) AggregationMin() TsRevrangeAggregationAggregationMin {
	c.cs.s = append(c.cs.s, "AGGREGATION", "MIN")
	return (TsRevrangeAggregationAggregationMin)(c)
}

func (c TsRevrangeLatest) AggregationMax() TsRevrangeAggregationAggregationMax {
	c.cs.s = append(c.cs.s, "AGGREGATION", "MAX")
	return (TsRevrangeAggregationAggregationMax)(c)
}

func (c TsRevrangeLatest) AggregationRange() TsRevrangeAggregationAggregationRange {
	c.cs.s = append(c.cs.s, "AGGREGATION", "RANGE")
	return (TsRevrangeAggregationAggregationRange)(c)
}

func (c TsRevrangeLatest) AggregationCount() TsRevrangeAggregationAggregationCount {
	c.cs.s = append(c.cs.s, "AGGREGATION", "COUNT")
	return (TsRevrangeAggregationAggregationCount)(c)
}

func (c TsRevrangeLatest) AggregationFirst() TsRevrangeAggregationAggregationFirst {
	c.cs.s = append(c.cs.s, "AGGREGATION", "FIRST")
	return (TsRevrangeAggregationAggregationFirst)(c)
}

func (c TsRevrangeLatest) AggregationLast() TsRevrangeAggregationAggregationLast {
	c.cs.s = append(c.cs.s, "AGGREGATION", "LAST")
	return (TsRevrangeAggregationAggregationLast)(c)
}

func (c TsRevrangeLatest) AggregationStdP() TsRevrangeAggregationAggregationStdP {
	c.cs.s = append(c.cs.s, "AGGREGATION", "STD.P")
	return (TsRevrangeAggregationAggregationStdP)(c)
}

func (c TsRevrangeLatest) AggregationStdS() TsRevrangeAggregationAggregationStdS {
	c.cs.s = append(c.cs.s, "AGGREGATION", "STD.S")
	return (TsRevrangeAggregationAggregationStdS)(c)
}

func (c TsRevrangeLatest) AggregationVarP() TsRevrangeAggregationAggregationVarP {
	c.cs.s = append(c.cs.s, "AGGREGATION", "VAR.P")
	return (TsRevrangeAggregationAggregationVarP)(c)
}

func (c TsRevrangeLatest) AggregationVarS() TsRevrangeAggregationAggregationVarS {
	c.cs.s = append(c.cs.s, "AGGREGATION", "VAR.S")
	return (TsRevrangeAggregationAggregationVarS)(c)
}

func (c TsRevrangeLatest) AggregationTwa() TsRevrangeAggregationAggregationTwa {
	c.cs.s = append(c.cs.s, "AGGREGATION", "TWA")
	return (TsRevrangeAggregationAggregationTwa)(c)
}

func (c TsRevrangeLatest) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type TsRevrangeTotimestamp Completed

func (c TsRevrangeTotimestamp) Latest() TsRevrangeLatest {
	c.cs.s = append(c.cs.s, "LATEST")
	return (TsRevrangeLatest)(c)
}

func (c TsRevrangeTotimestamp) FilterByTs(timestamp ...int64) TsRevrangeFilterByTs {
	c.cs.s = append(c.cs.s, "FILTER_BY_TS")
	for _, n := range timestamp {
		c.cs.s = append(c.cs.s, strconv.FormatInt(n, 10))
	}
	return (TsRevrangeFilterByTs)(c)
}

func (c TsRevrangeTotimestamp) FilterByValue(min float64, max float64) TsRevrangeFilterByValue {
	c.cs.s = append(c.cs.s, "FILTER_BY_VALUE", strconv.FormatFloat(min, 'f', -1, 64), strconv.FormatFloat(max, 'f', -1, 64))
	return (TsRevrangeFilterByValue)(c)
}

func (c TsRevrangeTotimestamp) Count(count int64) TsRevrangeCount {
	c.cs.s = append(c.cs.s, "COUNT", strconv.FormatInt(count, 10))
	return (TsRevrangeCount)(c)
}

func (c TsRevrangeTotimestamp) Align(value string) TsRevrangeAlign {
	c.cs.s = append(c.cs.s, "ALIGN", value)
	return (TsRevrangeAlign)(c)
}

func (c TsRevrangeTotimestamp) AggregationAvg() TsRevrangeAggregationAggregationAvg {
	c.cs.s = append(c.cs.s, "AGGREGATION", "AVG")
	return (TsRevrangeAggregationAggregationAvg)(c)
}

func (c TsRevrangeTotimestamp) AggregationSum() TsRevrangeAggregationAggregationSum {
	c.cs.s = append(c.cs.s, "AGGREGATION", "SUM")
	return (TsRevrangeAggregationAggregationSum)(c)
}

func (c TsRevrangeTotimestamp) AggregationMin() TsRevrangeAggregationAggregationMin {
	c.cs.s = append(c.cs.s, "AGGREGATION", "MIN")
	return (TsRevrangeAggregationAggregationMin)(c)
}

func (c TsRevrangeTotimestamp) AggregationMax() TsRevrangeAggregationAggregationMax {
	c.cs.s = append(c.cs.s, "AGGREGATION", "MAX")
	return (TsRevrangeAggregationAggregationMax)(c)
}

func (c TsRevrangeTotimestamp) AggregationRange() TsRevrangeAggregationAggregationRange {
	c.cs.s = append(c.cs.s, "AGGREGATION", "RANGE")
	return (TsRevrangeAggregationAggregationRange)(c)
}

func (c TsRevrangeTotimestamp) AggregationCount() TsRevrangeAggregationAggregationCount {
	c.cs.s = append(c.cs.s, "AGGREGATION", "COUNT")
	return (TsRevrangeAggregationAggregationCount)(c)
}

func (c TsRevrangeTotimestamp) AggregationFirst() TsRevrangeAggregationAggregationFirst {
	c.cs.s = append(c.cs.s, "AGGREGATION", "FIRST")
	return (TsRevrangeAggregationAggregationFirst)(c)
}

func (c TsRevrangeTotimestamp) AggregationLast() TsRevrangeAggregationAggregationLast {
	c.cs.s = append(c.cs.s, "AGGREGATION", "LAST")
	return (TsRevrangeAggregationAggregationLast)(c)
}

func (c TsRevrangeTotimestamp) AggregationStdP() TsRevrangeAggregationAggregationStdP {
	c.cs.s = append(c.cs.s, "AGGREGATION", "STD.P")
	return (TsRevrangeAggregationAggregationStdP)(c)
}

func (c TsRevrangeTotimestamp) AggregationStdS() TsRevrangeAggregationAggregationStdS {
	c.cs.s = append(c.cs.s, "AGGREGATION", "STD.S")
	return (TsRevrangeAggregationAggregationStdS)(c)
}

func (c TsRevrangeTotimestamp) AggregationVarP() TsRevrangeAggregationAggregationVarP {
	c.cs.s = append(c.cs.s, "AGGREGATION", "VAR.P")
	return (TsRevrangeAggregationAggregationVarP)(c)
}

func (c TsRevrangeTotimestamp) AggregationVarS() TsRevrangeAggregationAggregationVarS {
	c.cs.s = append(c.cs.s, "AGGREGATION", "VAR.S")
	return (TsRevrangeAggregationAggregationVarS)(c)
}

func (c TsRevrangeTotimestamp) AggregationTwa() TsRevrangeAggregationAggregationTwa {
	c.cs.s = append(c.cs.s, "AGGREGATION", "TWA")
	return (TsRevrangeAggregationAggregationTwa)(c)
}

func (c TsRevrangeTotimestamp) Build() Completed {
	c.cs.Build()
	return Completed(c)
}
