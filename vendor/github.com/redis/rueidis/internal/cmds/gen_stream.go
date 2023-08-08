// Code generated DO NOT EDIT

package cmds

import "strconv"

type Xack Completed

func (b Builder) Xack() (c Xack) {
	c = Xack{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "XACK")
	return c
}

func (c Xack) Key(key string) XackKey {
	if c.ks&NoSlot == NoSlot {
		c.ks = NoSlot | slot(key)
	} else {
		c.ks = check(c.ks, slot(key))
	}
	c.cs.s = append(c.cs.s, key)
	return (XackKey)(c)
}

type XackGroup Completed

func (c XackGroup) Id(id ...string) XackId {
	c.cs.s = append(c.cs.s, id...)
	return (XackId)(c)
}

type XackId Completed

func (c XackId) Id(id ...string) XackId {
	c.cs.s = append(c.cs.s, id...)
	return c
}

func (c XackId) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type XackKey Completed

func (c XackKey) Group(group string) XackGroup {
	c.cs.s = append(c.cs.s, group)
	return (XackGroup)(c)
}

type Xadd Completed

func (b Builder) Xadd() (c Xadd) {
	c = Xadd{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "XADD")
	return c
}

func (c Xadd) Key(key string) XaddKey {
	if c.ks&NoSlot == NoSlot {
		c.ks = NoSlot | slot(key)
	} else {
		c.ks = check(c.ks, slot(key))
	}
	c.cs.s = append(c.cs.s, key)
	return (XaddKey)(c)
}

type XaddFieldValue Completed

func (c XaddFieldValue) FieldValue(field string, value string) XaddFieldValue {
	c.cs.s = append(c.cs.s, field, value)
	return c
}

func (c XaddFieldValue) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type XaddId Completed

func (c XaddId) FieldValue() XaddFieldValue {
	return (XaddFieldValue)(c)
}

type XaddKey Completed

func (c XaddKey) Nomkstream() XaddNomkstream {
	c.cs.s = append(c.cs.s, "NOMKSTREAM")
	return (XaddNomkstream)(c)
}

func (c XaddKey) Maxlen() XaddTrimStrategyMaxlen {
	c.cs.s = append(c.cs.s, "MAXLEN")
	return (XaddTrimStrategyMaxlen)(c)
}

func (c XaddKey) Minid() XaddTrimStrategyMinid {
	c.cs.s = append(c.cs.s, "MINID")
	return (XaddTrimStrategyMinid)(c)
}

func (c XaddKey) Id(id string) XaddId {
	c.cs.s = append(c.cs.s, id)
	return (XaddId)(c)
}

type XaddNomkstream Completed

func (c XaddNomkstream) Maxlen() XaddTrimStrategyMaxlen {
	c.cs.s = append(c.cs.s, "MAXLEN")
	return (XaddTrimStrategyMaxlen)(c)
}

func (c XaddNomkstream) Minid() XaddTrimStrategyMinid {
	c.cs.s = append(c.cs.s, "MINID")
	return (XaddTrimStrategyMinid)(c)
}

func (c XaddNomkstream) Id(id string) XaddId {
	c.cs.s = append(c.cs.s, id)
	return (XaddId)(c)
}

type XaddTrimLimit Completed

func (c XaddTrimLimit) Id(id string) XaddId {
	c.cs.s = append(c.cs.s, id)
	return (XaddId)(c)
}

type XaddTrimOperatorAlmost Completed

func (c XaddTrimOperatorAlmost) Threshold(threshold string) XaddTrimThreshold {
	c.cs.s = append(c.cs.s, threshold)
	return (XaddTrimThreshold)(c)
}

type XaddTrimOperatorExact Completed

func (c XaddTrimOperatorExact) Threshold(threshold string) XaddTrimThreshold {
	c.cs.s = append(c.cs.s, threshold)
	return (XaddTrimThreshold)(c)
}

type XaddTrimStrategyMaxlen Completed

func (c XaddTrimStrategyMaxlen) Exact() XaddTrimOperatorExact {
	c.cs.s = append(c.cs.s, "=")
	return (XaddTrimOperatorExact)(c)
}

func (c XaddTrimStrategyMaxlen) Almost() XaddTrimOperatorAlmost {
	c.cs.s = append(c.cs.s, "~")
	return (XaddTrimOperatorAlmost)(c)
}

func (c XaddTrimStrategyMaxlen) Threshold(threshold string) XaddTrimThreshold {
	c.cs.s = append(c.cs.s, threshold)
	return (XaddTrimThreshold)(c)
}

type XaddTrimStrategyMinid Completed

func (c XaddTrimStrategyMinid) Exact() XaddTrimOperatorExact {
	c.cs.s = append(c.cs.s, "=")
	return (XaddTrimOperatorExact)(c)
}

func (c XaddTrimStrategyMinid) Almost() XaddTrimOperatorAlmost {
	c.cs.s = append(c.cs.s, "~")
	return (XaddTrimOperatorAlmost)(c)
}

func (c XaddTrimStrategyMinid) Threshold(threshold string) XaddTrimThreshold {
	c.cs.s = append(c.cs.s, threshold)
	return (XaddTrimThreshold)(c)
}

type XaddTrimThreshold Completed

func (c XaddTrimThreshold) Limit(count int64) XaddTrimLimit {
	c.cs.s = append(c.cs.s, "LIMIT", strconv.FormatInt(count, 10))
	return (XaddTrimLimit)(c)
}

func (c XaddTrimThreshold) Id(id string) XaddId {
	c.cs.s = append(c.cs.s, id)
	return (XaddId)(c)
}

type Xautoclaim Completed

func (b Builder) Xautoclaim() (c Xautoclaim) {
	c = Xautoclaim{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "XAUTOCLAIM")
	return c
}

func (c Xautoclaim) Key(key string) XautoclaimKey {
	if c.ks&NoSlot == NoSlot {
		c.ks = NoSlot | slot(key)
	} else {
		c.ks = check(c.ks, slot(key))
	}
	c.cs.s = append(c.cs.s, key)
	return (XautoclaimKey)(c)
}

type XautoclaimConsumer Completed

func (c XautoclaimConsumer) MinIdleTime(minIdleTime string) XautoclaimMinIdleTime {
	c.cs.s = append(c.cs.s, minIdleTime)
	return (XautoclaimMinIdleTime)(c)
}

type XautoclaimCount Completed

func (c XautoclaimCount) Justid() XautoclaimJustid {
	c.cs.s = append(c.cs.s, "JUSTID")
	return (XautoclaimJustid)(c)
}

func (c XautoclaimCount) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type XautoclaimGroup Completed

func (c XautoclaimGroup) Consumer(consumer string) XautoclaimConsumer {
	c.cs.s = append(c.cs.s, consumer)
	return (XautoclaimConsumer)(c)
}

type XautoclaimJustid Completed

func (c XautoclaimJustid) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type XautoclaimKey Completed

func (c XautoclaimKey) Group(group string) XautoclaimGroup {
	c.cs.s = append(c.cs.s, group)
	return (XautoclaimGroup)(c)
}

type XautoclaimMinIdleTime Completed

func (c XautoclaimMinIdleTime) Start(start string) XautoclaimStart {
	c.cs.s = append(c.cs.s, start)
	return (XautoclaimStart)(c)
}

type XautoclaimStart Completed

func (c XautoclaimStart) Count(count int64) XautoclaimCount {
	c.cs.s = append(c.cs.s, "COUNT", strconv.FormatInt(count, 10))
	return (XautoclaimCount)(c)
}

func (c XautoclaimStart) Justid() XautoclaimJustid {
	c.cs.s = append(c.cs.s, "JUSTID")
	return (XautoclaimJustid)(c)
}

func (c XautoclaimStart) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type Xclaim Completed

func (b Builder) Xclaim() (c Xclaim) {
	c = Xclaim{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "XCLAIM")
	return c
}

func (c Xclaim) Key(key string) XclaimKey {
	if c.ks&NoSlot == NoSlot {
		c.ks = NoSlot | slot(key)
	} else {
		c.ks = check(c.ks, slot(key))
	}
	c.cs.s = append(c.cs.s, key)
	return (XclaimKey)(c)
}

type XclaimConsumer Completed

func (c XclaimConsumer) MinIdleTime(minIdleTime string) XclaimMinIdleTime {
	c.cs.s = append(c.cs.s, minIdleTime)
	return (XclaimMinIdleTime)(c)
}

type XclaimForce Completed

func (c XclaimForce) Justid() XclaimJustid {
	c.cs.s = append(c.cs.s, "JUSTID")
	return (XclaimJustid)(c)
}

func (c XclaimForce) Lastid() XclaimLastid {
	c.cs.s = append(c.cs.s, "LASTID")
	return (XclaimLastid)(c)
}

func (c XclaimForce) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type XclaimGroup Completed

func (c XclaimGroup) Consumer(consumer string) XclaimConsumer {
	c.cs.s = append(c.cs.s, consumer)
	return (XclaimConsumer)(c)
}

type XclaimId Completed

func (c XclaimId) Id(id ...string) XclaimId {
	c.cs.s = append(c.cs.s, id...)
	return c
}

func (c XclaimId) Idle(ms int64) XclaimIdle {
	c.cs.s = append(c.cs.s, "IDLE", strconv.FormatInt(ms, 10))
	return (XclaimIdle)(c)
}

func (c XclaimId) Time(msUnixTime int64) XclaimTime {
	c.cs.s = append(c.cs.s, "TIME", strconv.FormatInt(msUnixTime, 10))
	return (XclaimTime)(c)
}

func (c XclaimId) Retrycount(count int64) XclaimRetrycount {
	c.cs.s = append(c.cs.s, "RETRYCOUNT", strconv.FormatInt(count, 10))
	return (XclaimRetrycount)(c)
}

func (c XclaimId) Force() XclaimForce {
	c.cs.s = append(c.cs.s, "FORCE")
	return (XclaimForce)(c)
}

func (c XclaimId) Justid() XclaimJustid {
	c.cs.s = append(c.cs.s, "JUSTID")
	return (XclaimJustid)(c)
}

func (c XclaimId) Lastid() XclaimLastid {
	c.cs.s = append(c.cs.s, "LASTID")
	return (XclaimLastid)(c)
}

func (c XclaimId) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type XclaimIdle Completed

func (c XclaimIdle) Time(msUnixTime int64) XclaimTime {
	c.cs.s = append(c.cs.s, "TIME", strconv.FormatInt(msUnixTime, 10))
	return (XclaimTime)(c)
}

func (c XclaimIdle) Retrycount(count int64) XclaimRetrycount {
	c.cs.s = append(c.cs.s, "RETRYCOUNT", strconv.FormatInt(count, 10))
	return (XclaimRetrycount)(c)
}

func (c XclaimIdle) Force() XclaimForce {
	c.cs.s = append(c.cs.s, "FORCE")
	return (XclaimForce)(c)
}

func (c XclaimIdle) Justid() XclaimJustid {
	c.cs.s = append(c.cs.s, "JUSTID")
	return (XclaimJustid)(c)
}

func (c XclaimIdle) Lastid() XclaimLastid {
	c.cs.s = append(c.cs.s, "LASTID")
	return (XclaimLastid)(c)
}

func (c XclaimIdle) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type XclaimJustid Completed

func (c XclaimJustid) Lastid() XclaimLastid {
	c.cs.s = append(c.cs.s, "LASTID")
	return (XclaimLastid)(c)
}

func (c XclaimJustid) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type XclaimKey Completed

func (c XclaimKey) Group(group string) XclaimGroup {
	c.cs.s = append(c.cs.s, group)
	return (XclaimGroup)(c)
}

type XclaimLastid Completed

func (c XclaimLastid) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type XclaimMinIdleTime Completed

func (c XclaimMinIdleTime) Id(id ...string) XclaimId {
	c.cs.s = append(c.cs.s, id...)
	return (XclaimId)(c)
}

type XclaimRetrycount Completed

func (c XclaimRetrycount) Force() XclaimForce {
	c.cs.s = append(c.cs.s, "FORCE")
	return (XclaimForce)(c)
}

func (c XclaimRetrycount) Justid() XclaimJustid {
	c.cs.s = append(c.cs.s, "JUSTID")
	return (XclaimJustid)(c)
}

func (c XclaimRetrycount) Lastid() XclaimLastid {
	c.cs.s = append(c.cs.s, "LASTID")
	return (XclaimLastid)(c)
}

func (c XclaimRetrycount) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type XclaimTime Completed

func (c XclaimTime) Retrycount(count int64) XclaimRetrycount {
	c.cs.s = append(c.cs.s, "RETRYCOUNT", strconv.FormatInt(count, 10))
	return (XclaimRetrycount)(c)
}

func (c XclaimTime) Force() XclaimForce {
	c.cs.s = append(c.cs.s, "FORCE")
	return (XclaimForce)(c)
}

func (c XclaimTime) Justid() XclaimJustid {
	c.cs.s = append(c.cs.s, "JUSTID")
	return (XclaimJustid)(c)
}

func (c XclaimTime) Lastid() XclaimLastid {
	c.cs.s = append(c.cs.s, "LASTID")
	return (XclaimLastid)(c)
}

func (c XclaimTime) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type Xdel Completed

func (b Builder) Xdel() (c Xdel) {
	c = Xdel{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "XDEL")
	return c
}

func (c Xdel) Key(key string) XdelKey {
	if c.ks&NoSlot == NoSlot {
		c.ks = NoSlot | slot(key)
	} else {
		c.ks = check(c.ks, slot(key))
	}
	c.cs.s = append(c.cs.s, key)
	return (XdelKey)(c)
}

type XdelId Completed

func (c XdelId) Id(id ...string) XdelId {
	c.cs.s = append(c.cs.s, id...)
	return c
}

func (c XdelId) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type XdelKey Completed

func (c XdelKey) Id(id ...string) XdelId {
	c.cs.s = append(c.cs.s, id...)
	return (XdelId)(c)
}

type XgroupCreate Completed

func (b Builder) XgroupCreate() (c XgroupCreate) {
	c = XgroupCreate{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "XGROUP", "CREATE")
	return c
}

func (c XgroupCreate) Key(key string) XgroupCreateKey {
	if c.ks&NoSlot == NoSlot {
		c.ks = NoSlot | slot(key)
	} else {
		c.ks = check(c.ks, slot(key))
	}
	c.cs.s = append(c.cs.s, key)
	return (XgroupCreateKey)(c)
}

type XgroupCreateEntriesread Completed

func (c XgroupCreateEntriesread) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type XgroupCreateGroup Completed

func (c XgroupCreateGroup) Id(id string) XgroupCreateId {
	c.cs.s = append(c.cs.s, id)
	return (XgroupCreateId)(c)
}

type XgroupCreateId Completed

func (c XgroupCreateId) Mkstream() XgroupCreateMkstream {
	c.cs.s = append(c.cs.s, "MKSTREAM")
	return (XgroupCreateMkstream)(c)
}

func (c XgroupCreateId) Entriesread(entriesRead int64) XgroupCreateEntriesread {
	c.cs.s = append(c.cs.s, "ENTRIESREAD", strconv.FormatInt(entriesRead, 10))
	return (XgroupCreateEntriesread)(c)
}

func (c XgroupCreateId) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type XgroupCreateKey Completed

func (c XgroupCreateKey) Group(group string) XgroupCreateGroup {
	c.cs.s = append(c.cs.s, group)
	return (XgroupCreateGroup)(c)
}

type XgroupCreateMkstream Completed

func (c XgroupCreateMkstream) Entriesread(entriesRead int64) XgroupCreateEntriesread {
	c.cs.s = append(c.cs.s, "ENTRIESREAD", strconv.FormatInt(entriesRead, 10))
	return (XgroupCreateEntriesread)(c)
}

func (c XgroupCreateMkstream) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type XgroupCreateconsumer Completed

func (b Builder) XgroupCreateconsumer() (c XgroupCreateconsumer) {
	c = XgroupCreateconsumer{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "XGROUP", "CREATECONSUMER")
	return c
}

func (c XgroupCreateconsumer) Key(key string) XgroupCreateconsumerKey {
	if c.ks&NoSlot == NoSlot {
		c.ks = NoSlot | slot(key)
	} else {
		c.ks = check(c.ks, slot(key))
	}
	c.cs.s = append(c.cs.s, key)
	return (XgroupCreateconsumerKey)(c)
}

type XgroupCreateconsumerConsumer Completed

func (c XgroupCreateconsumerConsumer) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type XgroupCreateconsumerGroup Completed

func (c XgroupCreateconsumerGroup) Consumer(consumer string) XgroupCreateconsumerConsumer {
	c.cs.s = append(c.cs.s, consumer)
	return (XgroupCreateconsumerConsumer)(c)
}

type XgroupCreateconsumerKey Completed

func (c XgroupCreateconsumerKey) Group(group string) XgroupCreateconsumerGroup {
	c.cs.s = append(c.cs.s, group)
	return (XgroupCreateconsumerGroup)(c)
}

type XgroupDelconsumer Completed

func (b Builder) XgroupDelconsumer() (c XgroupDelconsumer) {
	c = XgroupDelconsumer{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "XGROUP", "DELCONSUMER")
	return c
}

func (c XgroupDelconsumer) Key(key string) XgroupDelconsumerKey {
	if c.ks&NoSlot == NoSlot {
		c.ks = NoSlot | slot(key)
	} else {
		c.ks = check(c.ks, slot(key))
	}
	c.cs.s = append(c.cs.s, key)
	return (XgroupDelconsumerKey)(c)
}

type XgroupDelconsumerConsumername Completed

func (c XgroupDelconsumerConsumername) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type XgroupDelconsumerGroup Completed

func (c XgroupDelconsumerGroup) Consumername(consumername string) XgroupDelconsumerConsumername {
	c.cs.s = append(c.cs.s, consumername)
	return (XgroupDelconsumerConsumername)(c)
}

type XgroupDelconsumerKey Completed

func (c XgroupDelconsumerKey) Group(group string) XgroupDelconsumerGroup {
	c.cs.s = append(c.cs.s, group)
	return (XgroupDelconsumerGroup)(c)
}

type XgroupDestroy Completed

func (b Builder) XgroupDestroy() (c XgroupDestroy) {
	c = XgroupDestroy{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "XGROUP", "DESTROY")
	return c
}

func (c XgroupDestroy) Key(key string) XgroupDestroyKey {
	if c.ks&NoSlot == NoSlot {
		c.ks = NoSlot | slot(key)
	} else {
		c.ks = check(c.ks, slot(key))
	}
	c.cs.s = append(c.cs.s, key)
	return (XgroupDestroyKey)(c)
}

type XgroupDestroyGroup Completed

func (c XgroupDestroyGroup) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type XgroupDestroyKey Completed

func (c XgroupDestroyKey) Group(group string) XgroupDestroyGroup {
	c.cs.s = append(c.cs.s, group)
	return (XgroupDestroyGroup)(c)
}

type XgroupHelp Completed

func (b Builder) XgroupHelp() (c XgroupHelp) {
	c = XgroupHelp{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "XGROUP", "HELP")
	return c
}

func (c XgroupHelp) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type XgroupSetid Completed

func (b Builder) XgroupSetid() (c XgroupSetid) {
	c = XgroupSetid{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "XGROUP", "SETID")
	return c
}

func (c XgroupSetid) Key(key string) XgroupSetidKey {
	if c.ks&NoSlot == NoSlot {
		c.ks = NoSlot | slot(key)
	} else {
		c.ks = check(c.ks, slot(key))
	}
	c.cs.s = append(c.cs.s, key)
	return (XgroupSetidKey)(c)
}

type XgroupSetidEntriesread Completed

func (c XgroupSetidEntriesread) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type XgroupSetidGroup Completed

func (c XgroupSetidGroup) Id(id string) XgroupSetidId {
	c.cs.s = append(c.cs.s, id)
	return (XgroupSetidId)(c)
}

type XgroupSetidId Completed

func (c XgroupSetidId) Entriesread(entriesRead int64) XgroupSetidEntriesread {
	c.cs.s = append(c.cs.s, "ENTRIESREAD", strconv.FormatInt(entriesRead, 10))
	return (XgroupSetidEntriesread)(c)
}

func (c XgroupSetidId) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type XgroupSetidKey Completed

func (c XgroupSetidKey) Group(group string) XgroupSetidGroup {
	c.cs.s = append(c.cs.s, group)
	return (XgroupSetidGroup)(c)
}

type XinfoConsumers Completed

func (b Builder) XinfoConsumers() (c XinfoConsumers) {
	c = XinfoConsumers{cs: get(), ks: b.ks, cf: readonly}
	c.cs.s = append(c.cs.s, "XINFO", "CONSUMERS")
	return c
}

func (c XinfoConsumers) Key(key string) XinfoConsumersKey {
	if c.ks&NoSlot == NoSlot {
		c.ks = NoSlot | slot(key)
	} else {
		c.ks = check(c.ks, slot(key))
	}
	c.cs.s = append(c.cs.s, key)
	return (XinfoConsumersKey)(c)
}

type XinfoConsumersGroup Completed

func (c XinfoConsumersGroup) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type XinfoConsumersKey Completed

func (c XinfoConsumersKey) Group(group string) XinfoConsumersGroup {
	c.cs.s = append(c.cs.s, group)
	return (XinfoConsumersGroup)(c)
}

type XinfoGroups Completed

func (b Builder) XinfoGroups() (c XinfoGroups) {
	c = XinfoGroups{cs: get(), ks: b.ks, cf: readonly}
	c.cs.s = append(c.cs.s, "XINFO", "GROUPS")
	return c
}

func (c XinfoGroups) Key(key string) XinfoGroupsKey {
	if c.ks&NoSlot == NoSlot {
		c.ks = NoSlot | slot(key)
	} else {
		c.ks = check(c.ks, slot(key))
	}
	c.cs.s = append(c.cs.s, key)
	return (XinfoGroupsKey)(c)
}

type XinfoGroupsKey Completed

func (c XinfoGroupsKey) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type XinfoHelp Completed

func (b Builder) XinfoHelp() (c XinfoHelp) {
	c = XinfoHelp{cs: get(), ks: b.ks, cf: readonly}
	c.cs.s = append(c.cs.s, "XINFO", "HELP")
	return c
}

func (c XinfoHelp) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type XinfoStream Completed

func (b Builder) XinfoStream() (c XinfoStream) {
	c = XinfoStream{cs: get(), ks: b.ks, cf: readonly}
	c.cs.s = append(c.cs.s, "XINFO", "STREAM")
	return c
}

func (c XinfoStream) Key(key string) XinfoStreamKey {
	if c.ks&NoSlot == NoSlot {
		c.ks = NoSlot | slot(key)
	} else {
		c.ks = check(c.ks, slot(key))
	}
	c.cs.s = append(c.cs.s, key)
	return (XinfoStreamKey)(c)
}

type XinfoStreamFullCount Completed

func (c XinfoStreamFullCount) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type XinfoStreamFullFull Completed

func (c XinfoStreamFullFull) Count(count int64) XinfoStreamFullCount {
	c.cs.s = append(c.cs.s, "COUNT", strconv.FormatInt(count, 10))
	return (XinfoStreamFullCount)(c)
}

func (c XinfoStreamFullFull) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type XinfoStreamKey Completed

func (c XinfoStreamKey) Full() XinfoStreamFullFull {
	c.cs.s = append(c.cs.s, "FULL")
	return (XinfoStreamFullFull)(c)
}

func (c XinfoStreamKey) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type Xlen Completed

func (b Builder) Xlen() (c Xlen) {
	c = Xlen{cs: get(), ks: b.ks, cf: readonly}
	c.cs.s = append(c.cs.s, "XLEN")
	return c
}

func (c Xlen) Key(key string) XlenKey {
	if c.ks&NoSlot == NoSlot {
		c.ks = NoSlot | slot(key)
	} else {
		c.ks = check(c.ks, slot(key))
	}
	c.cs.s = append(c.cs.s, key)
	return (XlenKey)(c)
}

type XlenKey Completed

func (c XlenKey) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type Xpending Completed

func (b Builder) Xpending() (c Xpending) {
	c = Xpending{cs: get(), ks: b.ks, cf: readonly}
	c.cs.s = append(c.cs.s, "XPENDING")
	return c
}

func (c Xpending) Key(key string) XpendingKey {
	if c.ks&NoSlot == NoSlot {
		c.ks = NoSlot | slot(key)
	} else {
		c.ks = check(c.ks, slot(key))
	}
	c.cs.s = append(c.cs.s, key)
	return (XpendingKey)(c)
}

type XpendingFiltersConsumer Completed

func (c XpendingFiltersConsumer) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type XpendingFiltersCount Completed

func (c XpendingFiltersCount) Consumer(consumer string) XpendingFiltersConsumer {
	c.cs.s = append(c.cs.s, consumer)
	return (XpendingFiltersConsumer)(c)
}

func (c XpendingFiltersCount) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type XpendingFiltersEnd Completed

func (c XpendingFiltersEnd) Count(count int64) XpendingFiltersCount {
	c.cs.s = append(c.cs.s, strconv.FormatInt(count, 10))
	return (XpendingFiltersCount)(c)
}

type XpendingFiltersIdle Completed

func (c XpendingFiltersIdle) Start(start string) XpendingFiltersStart {
	c.cs.s = append(c.cs.s, start)
	return (XpendingFiltersStart)(c)
}

type XpendingFiltersStart Completed

func (c XpendingFiltersStart) End(end string) XpendingFiltersEnd {
	c.cs.s = append(c.cs.s, end)
	return (XpendingFiltersEnd)(c)
}

type XpendingGroup Completed

func (c XpendingGroup) Idle(minIdleTime int64) XpendingFiltersIdle {
	c.cs.s = append(c.cs.s, "IDLE", strconv.FormatInt(minIdleTime, 10))
	return (XpendingFiltersIdle)(c)
}

func (c XpendingGroup) Start(start string) XpendingFiltersStart {
	c.cs.s = append(c.cs.s, start)
	return (XpendingFiltersStart)(c)
}

func (c XpendingGroup) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type XpendingKey Completed

func (c XpendingKey) Group(group string) XpendingGroup {
	c.cs.s = append(c.cs.s, group)
	return (XpendingGroup)(c)
}

type Xrange Completed

func (b Builder) Xrange() (c Xrange) {
	c = Xrange{cs: get(), ks: b.ks, cf: readonly}
	c.cs.s = append(c.cs.s, "XRANGE")
	return c
}

func (c Xrange) Key(key string) XrangeKey {
	if c.ks&NoSlot == NoSlot {
		c.ks = NoSlot | slot(key)
	} else {
		c.ks = check(c.ks, slot(key))
	}
	c.cs.s = append(c.cs.s, key)
	return (XrangeKey)(c)
}

type XrangeCount Completed

func (c XrangeCount) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type XrangeEnd Completed

func (c XrangeEnd) Count(count int64) XrangeCount {
	c.cs.s = append(c.cs.s, "COUNT", strconv.FormatInt(count, 10))
	return (XrangeCount)(c)
}

func (c XrangeEnd) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type XrangeKey Completed

func (c XrangeKey) Start(start string) XrangeStart {
	c.cs.s = append(c.cs.s, start)
	return (XrangeStart)(c)
}

type XrangeStart Completed

func (c XrangeStart) End(end string) XrangeEnd {
	c.cs.s = append(c.cs.s, end)
	return (XrangeEnd)(c)
}

type Xread Completed

func (b Builder) Xread() (c Xread) {
	c = Xread{cs: get(), ks: b.ks, cf: readonly}
	c.cs.s = append(c.cs.s, "XREAD")
	return c
}

func (c Xread) Count(count int64) XreadCount {
	c.cs.s = append(c.cs.s, "COUNT", strconv.FormatInt(count, 10))
	return (XreadCount)(c)
}

func (c Xread) Block(milliseconds int64) XreadBlock {
	c.cf = blockTag
	c.cs.s = append(c.cs.s, "BLOCK", strconv.FormatInt(milliseconds, 10))
	return (XreadBlock)(c)
}

func (c Xread) Streams() XreadStreams {
	c.cs.s = append(c.cs.s, "STREAMS")
	return (XreadStreams)(c)
}

type XreadBlock Completed

func (c XreadBlock) Streams() XreadStreams {
	c.cs.s = append(c.cs.s, "STREAMS")
	return (XreadStreams)(c)
}

type XreadCount Completed

func (c XreadCount) Block(milliseconds int64) XreadBlock {
	c.cf = blockTag
	c.cs.s = append(c.cs.s, "BLOCK", strconv.FormatInt(milliseconds, 10))
	return (XreadBlock)(c)
}

func (c XreadCount) Streams() XreadStreams {
	c.cs.s = append(c.cs.s, "STREAMS")
	return (XreadStreams)(c)
}

type XreadId Completed

func (c XreadId) Id(id ...string) XreadId {
	c.cs.s = append(c.cs.s, id...)
	return c
}

func (c XreadId) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type XreadKey Completed

func (c XreadKey) Key(key ...string) XreadKey {
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

func (c XreadKey) Id(id ...string) XreadId {
	c.cs.s = append(c.cs.s, id...)
	return (XreadId)(c)
}

type XreadStreams Completed

func (c XreadStreams) Key(key ...string) XreadKey {
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
	return (XreadKey)(c)
}

type Xreadgroup Completed

func (b Builder) Xreadgroup() (c Xreadgroup) {
	c = Xreadgroup{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "XREADGROUP")
	return c
}

func (c Xreadgroup) Group(group string, consumer string) XreadgroupGroup {
	c.cs.s = append(c.cs.s, "GROUP", group, consumer)
	return (XreadgroupGroup)(c)
}

type XreadgroupBlock Completed

func (c XreadgroupBlock) Noack() XreadgroupNoack {
	c.cs.s = append(c.cs.s, "NOACK")
	return (XreadgroupNoack)(c)
}

func (c XreadgroupBlock) Streams() XreadgroupStreams {
	c.cs.s = append(c.cs.s, "STREAMS")
	return (XreadgroupStreams)(c)
}

type XreadgroupCount Completed

func (c XreadgroupCount) Block(milliseconds int64) XreadgroupBlock {
	c.cf = blockTag
	c.cs.s = append(c.cs.s, "BLOCK", strconv.FormatInt(milliseconds, 10))
	return (XreadgroupBlock)(c)
}

func (c XreadgroupCount) Noack() XreadgroupNoack {
	c.cs.s = append(c.cs.s, "NOACK")
	return (XreadgroupNoack)(c)
}

func (c XreadgroupCount) Streams() XreadgroupStreams {
	c.cs.s = append(c.cs.s, "STREAMS")
	return (XreadgroupStreams)(c)
}

type XreadgroupGroup Completed

func (c XreadgroupGroup) Count(count int64) XreadgroupCount {
	c.cs.s = append(c.cs.s, "COUNT", strconv.FormatInt(count, 10))
	return (XreadgroupCount)(c)
}

func (c XreadgroupGroup) Block(milliseconds int64) XreadgroupBlock {
	c.cf = blockTag
	c.cs.s = append(c.cs.s, "BLOCK", strconv.FormatInt(milliseconds, 10))
	return (XreadgroupBlock)(c)
}

func (c XreadgroupGroup) Noack() XreadgroupNoack {
	c.cs.s = append(c.cs.s, "NOACK")
	return (XreadgroupNoack)(c)
}

func (c XreadgroupGroup) Streams() XreadgroupStreams {
	c.cs.s = append(c.cs.s, "STREAMS")
	return (XreadgroupStreams)(c)
}

type XreadgroupId Completed

func (c XreadgroupId) Id(id ...string) XreadgroupId {
	c.cs.s = append(c.cs.s, id...)
	return c
}

func (c XreadgroupId) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type XreadgroupKey Completed

func (c XreadgroupKey) Key(key ...string) XreadgroupKey {
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

func (c XreadgroupKey) Id(id ...string) XreadgroupId {
	c.cs.s = append(c.cs.s, id...)
	return (XreadgroupId)(c)
}

type XreadgroupNoack Completed

func (c XreadgroupNoack) Streams() XreadgroupStreams {
	c.cs.s = append(c.cs.s, "STREAMS")
	return (XreadgroupStreams)(c)
}

type XreadgroupStreams Completed

func (c XreadgroupStreams) Key(key ...string) XreadgroupKey {
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
	return (XreadgroupKey)(c)
}

type Xrevrange Completed

func (b Builder) Xrevrange() (c Xrevrange) {
	c = Xrevrange{cs: get(), ks: b.ks, cf: readonly}
	c.cs.s = append(c.cs.s, "XREVRANGE")
	return c
}

func (c Xrevrange) Key(key string) XrevrangeKey {
	if c.ks&NoSlot == NoSlot {
		c.ks = NoSlot | slot(key)
	} else {
		c.ks = check(c.ks, slot(key))
	}
	c.cs.s = append(c.cs.s, key)
	return (XrevrangeKey)(c)
}

type XrevrangeCount Completed

func (c XrevrangeCount) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type XrevrangeEnd Completed

func (c XrevrangeEnd) Start(start string) XrevrangeStart {
	c.cs.s = append(c.cs.s, start)
	return (XrevrangeStart)(c)
}

type XrevrangeKey Completed

func (c XrevrangeKey) End(end string) XrevrangeEnd {
	c.cs.s = append(c.cs.s, end)
	return (XrevrangeEnd)(c)
}

type XrevrangeStart Completed

func (c XrevrangeStart) Count(count int64) XrevrangeCount {
	c.cs.s = append(c.cs.s, "COUNT", strconv.FormatInt(count, 10))
	return (XrevrangeCount)(c)
}

func (c XrevrangeStart) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type Xsetid Completed

func (b Builder) Xsetid() (c Xsetid) {
	c = Xsetid{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "XSETID")
	return c
}

func (c Xsetid) Key(key string) XsetidKey {
	if c.ks&NoSlot == NoSlot {
		c.ks = NoSlot | slot(key)
	} else {
		c.ks = check(c.ks, slot(key))
	}
	c.cs.s = append(c.cs.s, key)
	return (XsetidKey)(c)
}

type XsetidEntriesadded Completed

func (c XsetidEntriesadded) Maxdeletedid(maxDeletedEntryId string) XsetidMaxdeletedid {
	c.cs.s = append(c.cs.s, "MAXDELETEDID", maxDeletedEntryId)
	return (XsetidMaxdeletedid)(c)
}

func (c XsetidEntriesadded) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type XsetidKey Completed

func (c XsetidKey) LastId(lastId string) XsetidLastId {
	c.cs.s = append(c.cs.s, lastId)
	return (XsetidLastId)(c)
}

type XsetidLastId Completed

func (c XsetidLastId) Entriesadded(entriesAdded int64) XsetidEntriesadded {
	c.cs.s = append(c.cs.s, "ENTRIESADDED", strconv.FormatInt(entriesAdded, 10))
	return (XsetidEntriesadded)(c)
}

func (c XsetidLastId) Maxdeletedid(maxDeletedEntryId string) XsetidMaxdeletedid {
	c.cs.s = append(c.cs.s, "MAXDELETEDID", maxDeletedEntryId)
	return (XsetidMaxdeletedid)(c)
}

func (c XsetidLastId) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type XsetidMaxdeletedid Completed

func (c XsetidMaxdeletedid) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type Xtrim Completed

func (b Builder) Xtrim() (c Xtrim) {
	c = Xtrim{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "XTRIM")
	return c
}

func (c Xtrim) Key(key string) XtrimKey {
	if c.ks&NoSlot == NoSlot {
		c.ks = NoSlot | slot(key)
	} else {
		c.ks = check(c.ks, slot(key))
	}
	c.cs.s = append(c.cs.s, key)
	return (XtrimKey)(c)
}

type XtrimKey Completed

func (c XtrimKey) Maxlen() XtrimTrimStrategyMaxlen {
	c.cs.s = append(c.cs.s, "MAXLEN")
	return (XtrimTrimStrategyMaxlen)(c)
}

func (c XtrimKey) Minid() XtrimTrimStrategyMinid {
	c.cs.s = append(c.cs.s, "MINID")
	return (XtrimTrimStrategyMinid)(c)
}

type XtrimTrimLimit Completed

func (c XtrimTrimLimit) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type XtrimTrimOperatorAlmost Completed

func (c XtrimTrimOperatorAlmost) Threshold(threshold string) XtrimTrimThreshold {
	c.cs.s = append(c.cs.s, threshold)
	return (XtrimTrimThreshold)(c)
}

type XtrimTrimOperatorExact Completed

func (c XtrimTrimOperatorExact) Threshold(threshold string) XtrimTrimThreshold {
	c.cs.s = append(c.cs.s, threshold)
	return (XtrimTrimThreshold)(c)
}

type XtrimTrimStrategyMaxlen Completed

func (c XtrimTrimStrategyMaxlen) Exact() XtrimTrimOperatorExact {
	c.cs.s = append(c.cs.s, "=")
	return (XtrimTrimOperatorExact)(c)
}

func (c XtrimTrimStrategyMaxlen) Almost() XtrimTrimOperatorAlmost {
	c.cs.s = append(c.cs.s, "~")
	return (XtrimTrimOperatorAlmost)(c)
}

func (c XtrimTrimStrategyMaxlen) Threshold(threshold string) XtrimTrimThreshold {
	c.cs.s = append(c.cs.s, threshold)
	return (XtrimTrimThreshold)(c)
}

type XtrimTrimStrategyMinid Completed

func (c XtrimTrimStrategyMinid) Exact() XtrimTrimOperatorExact {
	c.cs.s = append(c.cs.s, "=")
	return (XtrimTrimOperatorExact)(c)
}

func (c XtrimTrimStrategyMinid) Almost() XtrimTrimOperatorAlmost {
	c.cs.s = append(c.cs.s, "~")
	return (XtrimTrimOperatorAlmost)(c)
}

func (c XtrimTrimStrategyMinid) Threshold(threshold string) XtrimTrimThreshold {
	c.cs.s = append(c.cs.s, threshold)
	return (XtrimTrimThreshold)(c)
}

type XtrimTrimThreshold Completed

func (c XtrimTrimThreshold) Limit(count int64) XtrimTrimLimit {
	c.cs.s = append(c.cs.s, "LIMIT", strconv.FormatInt(count, 10))
	return (XtrimTrimLimit)(c)
}

func (c XtrimTrimThreshold) Build() Completed {
	c.cs.Build()
	return Completed(c)
}
