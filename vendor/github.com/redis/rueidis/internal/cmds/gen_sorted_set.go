// Code generated DO NOT EDIT

package cmds

import "strconv"

type Bzmpop Completed

func (b Builder) Bzmpop() (c Bzmpop) {
	c = Bzmpop{cs: get(), ks: b.ks, cf: blockTag}
	c.cs.s = append(c.cs.s, "BZMPOP")
	return c
}

func (c Bzmpop) Timeout(timeout float64) BzmpopTimeout {
	c.cs.s = append(c.cs.s, strconv.FormatFloat(timeout, 'f', -1, 64))
	return (BzmpopTimeout)(c)
}

type BzmpopCount Completed

func (c BzmpopCount) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type BzmpopKey Completed

func (c BzmpopKey) Key(key ...string) BzmpopKey {
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

func (c BzmpopKey) Min() BzmpopWhereMin {
	c.cs.s = append(c.cs.s, "MIN")
	return (BzmpopWhereMin)(c)
}

func (c BzmpopKey) Max() BzmpopWhereMax {
	c.cs.s = append(c.cs.s, "MAX")
	return (BzmpopWhereMax)(c)
}

type BzmpopNumkeys Completed

func (c BzmpopNumkeys) Key(key ...string) BzmpopKey {
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
	return (BzmpopKey)(c)
}

type BzmpopTimeout Completed

func (c BzmpopTimeout) Numkeys(numkeys int64) BzmpopNumkeys {
	c.cs.s = append(c.cs.s, strconv.FormatInt(numkeys, 10))
	return (BzmpopNumkeys)(c)
}

type BzmpopWhereMax Completed

func (c BzmpopWhereMax) Count(count int64) BzmpopCount {
	c.cs.s = append(c.cs.s, "COUNT", strconv.FormatInt(count, 10))
	return (BzmpopCount)(c)
}

func (c BzmpopWhereMax) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type BzmpopWhereMin Completed

func (c BzmpopWhereMin) Count(count int64) BzmpopCount {
	c.cs.s = append(c.cs.s, "COUNT", strconv.FormatInt(count, 10))
	return (BzmpopCount)(c)
}

func (c BzmpopWhereMin) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type Bzpopmax Completed

func (b Builder) Bzpopmax() (c Bzpopmax) {
	c = Bzpopmax{cs: get(), ks: b.ks, cf: blockTag}
	c.cs.s = append(c.cs.s, "BZPOPMAX")
	return c
}

func (c Bzpopmax) Key(key ...string) BzpopmaxKey {
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
	return (BzpopmaxKey)(c)
}

type BzpopmaxKey Completed

func (c BzpopmaxKey) Key(key ...string) BzpopmaxKey {
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

func (c BzpopmaxKey) Timeout(timeout float64) BzpopmaxTimeout {
	c.cs.s = append(c.cs.s, strconv.FormatFloat(timeout, 'f', -1, 64))
	return (BzpopmaxTimeout)(c)
}

type BzpopmaxTimeout Completed

func (c BzpopmaxTimeout) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type Bzpopmin Completed

func (b Builder) Bzpopmin() (c Bzpopmin) {
	c = Bzpopmin{cs: get(), ks: b.ks, cf: blockTag}
	c.cs.s = append(c.cs.s, "BZPOPMIN")
	return c
}

func (c Bzpopmin) Key(key ...string) BzpopminKey {
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
	return (BzpopminKey)(c)
}

type BzpopminKey Completed

func (c BzpopminKey) Key(key ...string) BzpopminKey {
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

func (c BzpopminKey) Timeout(timeout float64) BzpopminTimeout {
	c.cs.s = append(c.cs.s, strconv.FormatFloat(timeout, 'f', -1, 64))
	return (BzpopminTimeout)(c)
}

type BzpopminTimeout Completed

func (c BzpopminTimeout) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type Zadd Completed

func (b Builder) Zadd() (c Zadd) {
	c = Zadd{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "ZADD")
	return c
}

func (c Zadd) Key(key string) ZaddKey {
	if c.ks&NoSlot == NoSlot {
		c.ks = NoSlot | slot(key)
	} else {
		c.ks = check(c.ks, slot(key))
	}
	c.cs.s = append(c.cs.s, key)
	return (ZaddKey)(c)
}

type ZaddChangeCh Completed

func (c ZaddChangeCh) Incr() ZaddIncrementIncr {
	c.cs.s = append(c.cs.s, "INCR")
	return (ZaddIncrementIncr)(c)
}

func (c ZaddChangeCh) ScoreMember() ZaddScoreMember {
	return (ZaddScoreMember)(c)
}

type ZaddComparisonGt Completed

func (c ZaddComparisonGt) Ch() ZaddChangeCh {
	c.cs.s = append(c.cs.s, "CH")
	return (ZaddChangeCh)(c)
}

func (c ZaddComparisonGt) Incr() ZaddIncrementIncr {
	c.cs.s = append(c.cs.s, "INCR")
	return (ZaddIncrementIncr)(c)
}

func (c ZaddComparisonGt) ScoreMember() ZaddScoreMember {
	return (ZaddScoreMember)(c)
}

type ZaddComparisonLt Completed

func (c ZaddComparisonLt) Ch() ZaddChangeCh {
	c.cs.s = append(c.cs.s, "CH")
	return (ZaddChangeCh)(c)
}

func (c ZaddComparisonLt) Incr() ZaddIncrementIncr {
	c.cs.s = append(c.cs.s, "INCR")
	return (ZaddIncrementIncr)(c)
}

func (c ZaddComparisonLt) ScoreMember() ZaddScoreMember {
	return (ZaddScoreMember)(c)
}

type ZaddConditionNx Completed

func (c ZaddConditionNx) Gt() ZaddComparisonGt {
	c.cs.s = append(c.cs.s, "GT")
	return (ZaddComparisonGt)(c)
}

func (c ZaddConditionNx) Lt() ZaddComparisonLt {
	c.cs.s = append(c.cs.s, "LT")
	return (ZaddComparisonLt)(c)
}

func (c ZaddConditionNx) Ch() ZaddChangeCh {
	c.cs.s = append(c.cs.s, "CH")
	return (ZaddChangeCh)(c)
}

func (c ZaddConditionNx) Incr() ZaddIncrementIncr {
	c.cs.s = append(c.cs.s, "INCR")
	return (ZaddIncrementIncr)(c)
}

func (c ZaddConditionNx) ScoreMember() ZaddScoreMember {
	return (ZaddScoreMember)(c)
}

type ZaddConditionXx Completed

func (c ZaddConditionXx) Gt() ZaddComparisonGt {
	c.cs.s = append(c.cs.s, "GT")
	return (ZaddComparisonGt)(c)
}

func (c ZaddConditionXx) Lt() ZaddComparisonLt {
	c.cs.s = append(c.cs.s, "LT")
	return (ZaddComparisonLt)(c)
}

func (c ZaddConditionXx) Ch() ZaddChangeCh {
	c.cs.s = append(c.cs.s, "CH")
	return (ZaddChangeCh)(c)
}

func (c ZaddConditionXx) Incr() ZaddIncrementIncr {
	c.cs.s = append(c.cs.s, "INCR")
	return (ZaddIncrementIncr)(c)
}

func (c ZaddConditionXx) ScoreMember() ZaddScoreMember {
	return (ZaddScoreMember)(c)
}

type ZaddIncrementIncr Completed

func (c ZaddIncrementIncr) ScoreMember() ZaddScoreMember {
	return (ZaddScoreMember)(c)
}

type ZaddKey Completed

func (c ZaddKey) Nx() ZaddConditionNx {
	c.cs.s = append(c.cs.s, "NX")
	return (ZaddConditionNx)(c)
}

func (c ZaddKey) Xx() ZaddConditionXx {
	c.cs.s = append(c.cs.s, "XX")
	return (ZaddConditionXx)(c)
}

func (c ZaddKey) Gt() ZaddComparisonGt {
	c.cs.s = append(c.cs.s, "GT")
	return (ZaddComparisonGt)(c)
}

func (c ZaddKey) Lt() ZaddComparisonLt {
	c.cs.s = append(c.cs.s, "LT")
	return (ZaddComparisonLt)(c)
}

func (c ZaddKey) Ch() ZaddChangeCh {
	c.cs.s = append(c.cs.s, "CH")
	return (ZaddChangeCh)(c)
}

func (c ZaddKey) Incr() ZaddIncrementIncr {
	c.cs.s = append(c.cs.s, "INCR")
	return (ZaddIncrementIncr)(c)
}

func (c ZaddKey) ScoreMember() ZaddScoreMember {
	return (ZaddScoreMember)(c)
}

type ZaddScoreMember Completed

func (c ZaddScoreMember) ScoreMember(score float64, member string) ZaddScoreMember {
	c.cs.s = append(c.cs.s, strconv.FormatFloat(score, 'f', -1, 64), member)
	return c
}

func (c ZaddScoreMember) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type Zcard Completed

func (b Builder) Zcard() (c Zcard) {
	c = Zcard{cs: get(), ks: b.ks, cf: readonly}
	c.cs.s = append(c.cs.s, "ZCARD")
	return c
}

func (c Zcard) Key(key string) ZcardKey {
	if c.ks&NoSlot == NoSlot {
		c.ks = NoSlot | slot(key)
	} else {
		c.ks = check(c.ks, slot(key))
	}
	c.cs.s = append(c.cs.s, key)
	return (ZcardKey)(c)
}

type ZcardKey Completed

func (c ZcardKey) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

func (c ZcardKey) Cache() Cacheable {
	c.cs.Build()
	return Cacheable(c)
}

type Zcount Completed

func (b Builder) Zcount() (c Zcount) {
	c = Zcount{cs: get(), ks: b.ks, cf: readonly}
	c.cs.s = append(c.cs.s, "ZCOUNT")
	return c
}

func (c Zcount) Key(key string) ZcountKey {
	if c.ks&NoSlot == NoSlot {
		c.ks = NoSlot | slot(key)
	} else {
		c.ks = check(c.ks, slot(key))
	}
	c.cs.s = append(c.cs.s, key)
	return (ZcountKey)(c)
}

type ZcountKey Completed

func (c ZcountKey) Min(min string) ZcountMin {
	c.cs.s = append(c.cs.s, min)
	return (ZcountMin)(c)
}

type ZcountMax Completed

func (c ZcountMax) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

func (c ZcountMax) Cache() Cacheable {
	c.cs.Build()
	return Cacheable(c)
}

type ZcountMin Completed

func (c ZcountMin) Max(max string) ZcountMax {
	c.cs.s = append(c.cs.s, max)
	return (ZcountMax)(c)
}

type Zdiff Completed

func (b Builder) Zdiff() (c Zdiff) {
	c = Zdiff{cs: get(), ks: b.ks, cf: readonly}
	c.cs.s = append(c.cs.s, "ZDIFF")
	return c
}

func (c Zdiff) Numkeys(numkeys int64) ZdiffNumkeys {
	c.cs.s = append(c.cs.s, strconv.FormatInt(numkeys, 10))
	return (ZdiffNumkeys)(c)
}

type ZdiffKey Completed

func (c ZdiffKey) Key(key ...string) ZdiffKey {
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

func (c ZdiffKey) Withscores() ZdiffWithscores {
	c.cs.s = append(c.cs.s, "WITHSCORES")
	return (ZdiffWithscores)(c)
}

func (c ZdiffKey) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type ZdiffNumkeys Completed

func (c ZdiffNumkeys) Key(key ...string) ZdiffKey {
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
	return (ZdiffKey)(c)
}

type ZdiffWithscores Completed

func (c ZdiffWithscores) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type Zdiffstore Completed

func (b Builder) Zdiffstore() (c Zdiffstore) {
	c = Zdiffstore{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "ZDIFFSTORE")
	return c
}

func (c Zdiffstore) Destination(destination string) ZdiffstoreDestination {
	if c.ks&NoSlot == NoSlot {
		c.ks = NoSlot | slot(destination)
	} else {
		c.ks = check(c.ks, slot(destination))
	}
	c.cs.s = append(c.cs.s, destination)
	return (ZdiffstoreDestination)(c)
}

type ZdiffstoreDestination Completed

func (c ZdiffstoreDestination) Numkeys(numkeys int64) ZdiffstoreNumkeys {
	c.cs.s = append(c.cs.s, strconv.FormatInt(numkeys, 10))
	return (ZdiffstoreNumkeys)(c)
}

type ZdiffstoreKey Completed

func (c ZdiffstoreKey) Key(key ...string) ZdiffstoreKey {
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

func (c ZdiffstoreKey) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type ZdiffstoreNumkeys Completed

func (c ZdiffstoreNumkeys) Key(key ...string) ZdiffstoreKey {
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
	return (ZdiffstoreKey)(c)
}

type Zincrby Completed

func (b Builder) Zincrby() (c Zincrby) {
	c = Zincrby{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "ZINCRBY")
	return c
}

func (c Zincrby) Key(key string) ZincrbyKey {
	if c.ks&NoSlot == NoSlot {
		c.ks = NoSlot | slot(key)
	} else {
		c.ks = check(c.ks, slot(key))
	}
	c.cs.s = append(c.cs.s, key)
	return (ZincrbyKey)(c)
}

type ZincrbyIncrement Completed

func (c ZincrbyIncrement) Member(member string) ZincrbyMember {
	c.cs.s = append(c.cs.s, member)
	return (ZincrbyMember)(c)
}

type ZincrbyKey Completed

func (c ZincrbyKey) Increment(increment float64) ZincrbyIncrement {
	c.cs.s = append(c.cs.s, strconv.FormatFloat(increment, 'f', -1, 64))
	return (ZincrbyIncrement)(c)
}

type ZincrbyMember Completed

func (c ZincrbyMember) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type Zinter Completed

func (b Builder) Zinter() (c Zinter) {
	c = Zinter{cs: get(), ks: b.ks, cf: readonly}
	c.cs.s = append(c.cs.s, "ZINTER")
	return c
}

func (c Zinter) Numkeys(numkeys int64) ZinterNumkeys {
	c.cs.s = append(c.cs.s, strconv.FormatInt(numkeys, 10))
	return (ZinterNumkeys)(c)
}

type ZinterAggregateMax Completed

func (c ZinterAggregateMax) Withscores() ZinterWithscores {
	c.cs.s = append(c.cs.s, "WITHSCORES")
	return (ZinterWithscores)(c)
}

func (c ZinterAggregateMax) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type ZinterAggregateMin Completed

func (c ZinterAggregateMin) Withscores() ZinterWithscores {
	c.cs.s = append(c.cs.s, "WITHSCORES")
	return (ZinterWithscores)(c)
}

func (c ZinterAggregateMin) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type ZinterAggregateSum Completed

func (c ZinterAggregateSum) Withscores() ZinterWithscores {
	c.cs.s = append(c.cs.s, "WITHSCORES")
	return (ZinterWithscores)(c)
}

func (c ZinterAggregateSum) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type ZinterKey Completed

func (c ZinterKey) Key(key ...string) ZinterKey {
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

func (c ZinterKey) Weights(weight ...int64) ZinterWeights {
	c.cs.s = append(c.cs.s, "WEIGHTS")
	for _, n := range weight {
		c.cs.s = append(c.cs.s, strconv.FormatInt(n, 10))
	}
	return (ZinterWeights)(c)
}

func (c ZinterKey) AggregateSum() ZinterAggregateSum {
	c.cs.s = append(c.cs.s, "AGGREGATE", "SUM")
	return (ZinterAggregateSum)(c)
}

func (c ZinterKey) AggregateMin() ZinterAggregateMin {
	c.cs.s = append(c.cs.s, "AGGREGATE", "MIN")
	return (ZinterAggregateMin)(c)
}

func (c ZinterKey) AggregateMax() ZinterAggregateMax {
	c.cs.s = append(c.cs.s, "AGGREGATE", "MAX")
	return (ZinterAggregateMax)(c)
}

func (c ZinterKey) Withscores() ZinterWithscores {
	c.cs.s = append(c.cs.s, "WITHSCORES")
	return (ZinterWithscores)(c)
}

func (c ZinterKey) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type ZinterNumkeys Completed

func (c ZinterNumkeys) Key(key ...string) ZinterKey {
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
	return (ZinterKey)(c)
}

type ZinterWeights Completed

func (c ZinterWeights) Weights(weight ...int64) ZinterWeights {
	c.cs.s = append(c.cs.s, "WEIGHTS")
	for _, n := range weight {
		c.cs.s = append(c.cs.s, strconv.FormatInt(n, 10))
	}
	return c
}

func (c ZinterWeights) AggregateSum() ZinterAggregateSum {
	c.cs.s = append(c.cs.s, "AGGREGATE", "SUM")
	return (ZinterAggregateSum)(c)
}

func (c ZinterWeights) AggregateMin() ZinterAggregateMin {
	c.cs.s = append(c.cs.s, "AGGREGATE", "MIN")
	return (ZinterAggregateMin)(c)
}

func (c ZinterWeights) AggregateMax() ZinterAggregateMax {
	c.cs.s = append(c.cs.s, "AGGREGATE", "MAX")
	return (ZinterAggregateMax)(c)
}

func (c ZinterWeights) Withscores() ZinterWithscores {
	c.cs.s = append(c.cs.s, "WITHSCORES")
	return (ZinterWithscores)(c)
}

func (c ZinterWeights) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type ZinterWithscores Completed

func (c ZinterWithscores) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type Zintercard Completed

func (b Builder) Zintercard() (c Zintercard) {
	c = Zintercard{cs: get(), ks: b.ks, cf: readonly}
	c.cs.s = append(c.cs.s, "ZINTERCARD")
	return c
}

func (c Zintercard) Numkeys(numkeys int64) ZintercardNumkeys {
	c.cs.s = append(c.cs.s, strconv.FormatInt(numkeys, 10))
	return (ZintercardNumkeys)(c)
}

type ZintercardKey Completed

func (c ZintercardKey) Key(key ...string) ZintercardKey {
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

func (c ZintercardKey) Limit(limit int64) ZintercardLimit {
	c.cs.s = append(c.cs.s, "LIMIT", strconv.FormatInt(limit, 10))
	return (ZintercardLimit)(c)
}

func (c ZintercardKey) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type ZintercardLimit Completed

func (c ZintercardLimit) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type ZintercardNumkeys Completed

func (c ZintercardNumkeys) Key(key ...string) ZintercardKey {
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
	return (ZintercardKey)(c)
}

type Zinterstore Completed

func (b Builder) Zinterstore() (c Zinterstore) {
	c = Zinterstore{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "ZINTERSTORE")
	return c
}

func (c Zinterstore) Destination(destination string) ZinterstoreDestination {
	if c.ks&NoSlot == NoSlot {
		c.ks = NoSlot | slot(destination)
	} else {
		c.ks = check(c.ks, slot(destination))
	}
	c.cs.s = append(c.cs.s, destination)
	return (ZinterstoreDestination)(c)
}

type ZinterstoreAggregateMax Completed

func (c ZinterstoreAggregateMax) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type ZinterstoreAggregateMin Completed

func (c ZinterstoreAggregateMin) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type ZinterstoreAggregateSum Completed

func (c ZinterstoreAggregateSum) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type ZinterstoreDestination Completed

func (c ZinterstoreDestination) Numkeys(numkeys int64) ZinterstoreNumkeys {
	c.cs.s = append(c.cs.s, strconv.FormatInt(numkeys, 10))
	return (ZinterstoreNumkeys)(c)
}

type ZinterstoreKey Completed

func (c ZinterstoreKey) Key(key ...string) ZinterstoreKey {
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

func (c ZinterstoreKey) Weights(weight ...int64) ZinterstoreWeights {
	c.cs.s = append(c.cs.s, "WEIGHTS")
	for _, n := range weight {
		c.cs.s = append(c.cs.s, strconv.FormatInt(n, 10))
	}
	return (ZinterstoreWeights)(c)
}

func (c ZinterstoreKey) AggregateSum() ZinterstoreAggregateSum {
	c.cs.s = append(c.cs.s, "AGGREGATE", "SUM")
	return (ZinterstoreAggregateSum)(c)
}

func (c ZinterstoreKey) AggregateMin() ZinterstoreAggregateMin {
	c.cs.s = append(c.cs.s, "AGGREGATE", "MIN")
	return (ZinterstoreAggregateMin)(c)
}

func (c ZinterstoreKey) AggregateMax() ZinterstoreAggregateMax {
	c.cs.s = append(c.cs.s, "AGGREGATE", "MAX")
	return (ZinterstoreAggregateMax)(c)
}

func (c ZinterstoreKey) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type ZinterstoreNumkeys Completed

func (c ZinterstoreNumkeys) Key(key ...string) ZinterstoreKey {
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
	return (ZinterstoreKey)(c)
}

type ZinterstoreWeights Completed

func (c ZinterstoreWeights) Weights(weight ...int64) ZinterstoreWeights {
	c.cs.s = append(c.cs.s, "WEIGHTS")
	for _, n := range weight {
		c.cs.s = append(c.cs.s, strconv.FormatInt(n, 10))
	}
	return c
}

func (c ZinterstoreWeights) AggregateSum() ZinterstoreAggregateSum {
	c.cs.s = append(c.cs.s, "AGGREGATE", "SUM")
	return (ZinterstoreAggregateSum)(c)
}

func (c ZinterstoreWeights) AggregateMin() ZinterstoreAggregateMin {
	c.cs.s = append(c.cs.s, "AGGREGATE", "MIN")
	return (ZinterstoreAggregateMin)(c)
}

func (c ZinterstoreWeights) AggregateMax() ZinterstoreAggregateMax {
	c.cs.s = append(c.cs.s, "AGGREGATE", "MAX")
	return (ZinterstoreAggregateMax)(c)
}

func (c ZinterstoreWeights) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type Zlexcount Completed

func (b Builder) Zlexcount() (c Zlexcount) {
	c = Zlexcount{cs: get(), ks: b.ks, cf: readonly}
	c.cs.s = append(c.cs.s, "ZLEXCOUNT")
	return c
}

func (c Zlexcount) Key(key string) ZlexcountKey {
	if c.ks&NoSlot == NoSlot {
		c.ks = NoSlot | slot(key)
	} else {
		c.ks = check(c.ks, slot(key))
	}
	c.cs.s = append(c.cs.s, key)
	return (ZlexcountKey)(c)
}

type ZlexcountKey Completed

func (c ZlexcountKey) Min(min string) ZlexcountMin {
	c.cs.s = append(c.cs.s, min)
	return (ZlexcountMin)(c)
}

type ZlexcountMax Completed

func (c ZlexcountMax) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

func (c ZlexcountMax) Cache() Cacheable {
	c.cs.Build()
	return Cacheable(c)
}

type ZlexcountMin Completed

func (c ZlexcountMin) Max(max string) ZlexcountMax {
	c.cs.s = append(c.cs.s, max)
	return (ZlexcountMax)(c)
}

type Zmpop Completed

func (b Builder) Zmpop() (c Zmpop) {
	c = Zmpop{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "ZMPOP")
	return c
}

func (c Zmpop) Numkeys(numkeys int64) ZmpopNumkeys {
	c.cs.s = append(c.cs.s, strconv.FormatInt(numkeys, 10))
	return (ZmpopNumkeys)(c)
}

type ZmpopCount Completed

func (c ZmpopCount) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type ZmpopKey Completed

func (c ZmpopKey) Key(key ...string) ZmpopKey {
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

func (c ZmpopKey) Min() ZmpopWhereMin {
	c.cs.s = append(c.cs.s, "MIN")
	return (ZmpopWhereMin)(c)
}

func (c ZmpopKey) Max() ZmpopWhereMax {
	c.cs.s = append(c.cs.s, "MAX")
	return (ZmpopWhereMax)(c)
}

type ZmpopNumkeys Completed

func (c ZmpopNumkeys) Key(key ...string) ZmpopKey {
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
	return (ZmpopKey)(c)
}

type ZmpopWhereMax Completed

func (c ZmpopWhereMax) Count(count int64) ZmpopCount {
	c.cs.s = append(c.cs.s, "COUNT", strconv.FormatInt(count, 10))
	return (ZmpopCount)(c)
}

func (c ZmpopWhereMax) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type ZmpopWhereMin Completed

func (c ZmpopWhereMin) Count(count int64) ZmpopCount {
	c.cs.s = append(c.cs.s, "COUNT", strconv.FormatInt(count, 10))
	return (ZmpopCount)(c)
}

func (c ZmpopWhereMin) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type Zmscore Completed

func (b Builder) Zmscore() (c Zmscore) {
	c = Zmscore{cs: get(), ks: b.ks, cf: readonly}
	c.cs.s = append(c.cs.s, "ZMSCORE")
	return c
}

func (c Zmscore) Key(key string) ZmscoreKey {
	if c.ks&NoSlot == NoSlot {
		c.ks = NoSlot | slot(key)
	} else {
		c.ks = check(c.ks, slot(key))
	}
	c.cs.s = append(c.cs.s, key)
	return (ZmscoreKey)(c)
}

type ZmscoreKey Completed

func (c ZmscoreKey) Member(member ...string) ZmscoreMember {
	c.cs.s = append(c.cs.s, member...)
	return (ZmscoreMember)(c)
}

type ZmscoreMember Completed

func (c ZmscoreMember) Member(member ...string) ZmscoreMember {
	c.cs.s = append(c.cs.s, member...)
	return c
}

func (c ZmscoreMember) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

func (c ZmscoreMember) Cache() Cacheable {
	c.cs.Build()
	return Cacheable(c)
}

type Zpopmax Completed

func (b Builder) Zpopmax() (c Zpopmax) {
	c = Zpopmax{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "ZPOPMAX")
	return c
}

func (c Zpopmax) Key(key string) ZpopmaxKey {
	if c.ks&NoSlot == NoSlot {
		c.ks = NoSlot | slot(key)
	} else {
		c.ks = check(c.ks, slot(key))
	}
	c.cs.s = append(c.cs.s, key)
	return (ZpopmaxKey)(c)
}

type ZpopmaxCount Completed

func (c ZpopmaxCount) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type ZpopmaxKey Completed

func (c ZpopmaxKey) Count(count int64) ZpopmaxCount {
	c.cs.s = append(c.cs.s, strconv.FormatInt(count, 10))
	return (ZpopmaxCount)(c)
}

func (c ZpopmaxKey) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type Zpopmin Completed

func (b Builder) Zpopmin() (c Zpopmin) {
	c = Zpopmin{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "ZPOPMIN")
	return c
}

func (c Zpopmin) Key(key string) ZpopminKey {
	if c.ks&NoSlot == NoSlot {
		c.ks = NoSlot | slot(key)
	} else {
		c.ks = check(c.ks, slot(key))
	}
	c.cs.s = append(c.cs.s, key)
	return (ZpopminKey)(c)
}

type ZpopminCount Completed

func (c ZpopminCount) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type ZpopminKey Completed

func (c ZpopminKey) Count(count int64) ZpopminCount {
	c.cs.s = append(c.cs.s, strconv.FormatInt(count, 10))
	return (ZpopminCount)(c)
}

func (c ZpopminKey) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type Zrandmember Completed

func (b Builder) Zrandmember() (c Zrandmember) {
	c = Zrandmember{cs: get(), ks: b.ks, cf: readonly}
	c.cs.s = append(c.cs.s, "ZRANDMEMBER")
	return c
}

func (c Zrandmember) Key(key string) ZrandmemberKey {
	if c.ks&NoSlot == NoSlot {
		c.ks = NoSlot | slot(key)
	} else {
		c.ks = check(c.ks, slot(key))
	}
	c.cs.s = append(c.cs.s, key)
	return (ZrandmemberKey)(c)
}

type ZrandmemberKey Completed

func (c ZrandmemberKey) Count(count int64) ZrandmemberOptionsCount {
	c.cs.s = append(c.cs.s, strconv.FormatInt(count, 10))
	return (ZrandmemberOptionsCount)(c)
}

func (c ZrandmemberKey) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type ZrandmemberOptionsCount Completed

func (c ZrandmemberOptionsCount) Withscores() ZrandmemberOptionsWithscores {
	c.cs.s = append(c.cs.s, "WITHSCORES")
	return (ZrandmemberOptionsWithscores)(c)
}

func (c ZrandmemberOptionsCount) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type ZrandmemberOptionsWithscores Completed

func (c ZrandmemberOptionsWithscores) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type Zrange Completed

func (b Builder) Zrange() (c Zrange) {
	c = Zrange{cs: get(), ks: b.ks, cf: readonly}
	c.cs.s = append(c.cs.s, "ZRANGE")
	return c
}

func (c Zrange) Key(key string) ZrangeKey {
	if c.ks&NoSlot == NoSlot {
		c.ks = NoSlot | slot(key)
	} else {
		c.ks = check(c.ks, slot(key))
	}
	c.cs.s = append(c.cs.s, key)
	return (ZrangeKey)(c)
}

type ZrangeKey Completed

func (c ZrangeKey) Min(min string) ZrangeMin {
	c.cs.s = append(c.cs.s, min)
	return (ZrangeMin)(c)
}

type ZrangeLimit Completed

func (c ZrangeLimit) Withscores() ZrangeWithscores {
	c.cs.s = append(c.cs.s, "WITHSCORES")
	return (ZrangeWithscores)(c)
}

func (c ZrangeLimit) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

func (c ZrangeLimit) Cache() Cacheable {
	c.cs.Build()
	return Cacheable(c)
}

type ZrangeMax Completed

func (c ZrangeMax) Byscore() ZrangeSortbyByscore {
	c.cs.s = append(c.cs.s, "BYSCORE")
	return (ZrangeSortbyByscore)(c)
}

func (c ZrangeMax) Bylex() ZrangeSortbyBylex {
	c.cs.s = append(c.cs.s, "BYLEX")
	return (ZrangeSortbyBylex)(c)
}

func (c ZrangeMax) Rev() ZrangeRev {
	c.cs.s = append(c.cs.s, "REV")
	return (ZrangeRev)(c)
}

func (c ZrangeMax) Limit(offset int64, count int64) ZrangeLimit {
	c.cs.s = append(c.cs.s, "LIMIT", strconv.FormatInt(offset, 10), strconv.FormatInt(count, 10))
	return (ZrangeLimit)(c)
}

func (c ZrangeMax) Withscores() ZrangeWithscores {
	c.cs.s = append(c.cs.s, "WITHSCORES")
	return (ZrangeWithscores)(c)
}

func (c ZrangeMax) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

func (c ZrangeMax) Cache() Cacheable {
	c.cs.Build()
	return Cacheable(c)
}

type ZrangeMin Completed

func (c ZrangeMin) Max(max string) ZrangeMax {
	c.cs.s = append(c.cs.s, max)
	return (ZrangeMax)(c)
}

type ZrangeRev Completed

func (c ZrangeRev) Limit(offset int64, count int64) ZrangeLimit {
	c.cs.s = append(c.cs.s, "LIMIT", strconv.FormatInt(offset, 10), strconv.FormatInt(count, 10))
	return (ZrangeLimit)(c)
}

func (c ZrangeRev) Withscores() ZrangeWithscores {
	c.cs.s = append(c.cs.s, "WITHSCORES")
	return (ZrangeWithscores)(c)
}

func (c ZrangeRev) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

func (c ZrangeRev) Cache() Cacheable {
	c.cs.Build()
	return Cacheable(c)
}

type ZrangeSortbyBylex Completed

func (c ZrangeSortbyBylex) Rev() ZrangeRev {
	c.cs.s = append(c.cs.s, "REV")
	return (ZrangeRev)(c)
}

func (c ZrangeSortbyBylex) Limit(offset int64, count int64) ZrangeLimit {
	c.cs.s = append(c.cs.s, "LIMIT", strconv.FormatInt(offset, 10), strconv.FormatInt(count, 10))
	return (ZrangeLimit)(c)
}

func (c ZrangeSortbyBylex) Withscores() ZrangeWithscores {
	c.cs.s = append(c.cs.s, "WITHSCORES")
	return (ZrangeWithscores)(c)
}

func (c ZrangeSortbyBylex) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

func (c ZrangeSortbyBylex) Cache() Cacheable {
	c.cs.Build()
	return Cacheable(c)
}

type ZrangeSortbyByscore Completed

func (c ZrangeSortbyByscore) Rev() ZrangeRev {
	c.cs.s = append(c.cs.s, "REV")
	return (ZrangeRev)(c)
}

func (c ZrangeSortbyByscore) Limit(offset int64, count int64) ZrangeLimit {
	c.cs.s = append(c.cs.s, "LIMIT", strconv.FormatInt(offset, 10), strconv.FormatInt(count, 10))
	return (ZrangeLimit)(c)
}

func (c ZrangeSortbyByscore) Withscores() ZrangeWithscores {
	c.cs.s = append(c.cs.s, "WITHSCORES")
	return (ZrangeWithscores)(c)
}

func (c ZrangeSortbyByscore) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

func (c ZrangeSortbyByscore) Cache() Cacheable {
	c.cs.Build()
	return Cacheable(c)
}

type ZrangeWithscores Completed

func (c ZrangeWithscores) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

func (c ZrangeWithscores) Cache() Cacheable {
	c.cs.Build()
	return Cacheable(c)
}

type Zrangebylex Completed

func (b Builder) Zrangebylex() (c Zrangebylex) {
	c = Zrangebylex{cs: get(), ks: b.ks, cf: readonly}
	c.cs.s = append(c.cs.s, "ZRANGEBYLEX")
	return c
}

func (c Zrangebylex) Key(key string) ZrangebylexKey {
	if c.ks&NoSlot == NoSlot {
		c.ks = NoSlot | slot(key)
	} else {
		c.ks = check(c.ks, slot(key))
	}
	c.cs.s = append(c.cs.s, key)
	return (ZrangebylexKey)(c)
}

type ZrangebylexKey Completed

func (c ZrangebylexKey) Min(min string) ZrangebylexMin {
	c.cs.s = append(c.cs.s, min)
	return (ZrangebylexMin)(c)
}

type ZrangebylexLimit Completed

func (c ZrangebylexLimit) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

func (c ZrangebylexLimit) Cache() Cacheable {
	c.cs.Build()
	return Cacheable(c)
}

type ZrangebylexMax Completed

func (c ZrangebylexMax) Limit(offset int64, count int64) ZrangebylexLimit {
	c.cs.s = append(c.cs.s, "LIMIT", strconv.FormatInt(offset, 10), strconv.FormatInt(count, 10))
	return (ZrangebylexLimit)(c)
}

func (c ZrangebylexMax) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

func (c ZrangebylexMax) Cache() Cacheable {
	c.cs.Build()
	return Cacheable(c)
}

type ZrangebylexMin Completed

func (c ZrangebylexMin) Max(max string) ZrangebylexMax {
	c.cs.s = append(c.cs.s, max)
	return (ZrangebylexMax)(c)
}

type Zrangebyscore Completed

func (b Builder) Zrangebyscore() (c Zrangebyscore) {
	c = Zrangebyscore{cs: get(), ks: b.ks, cf: readonly}
	c.cs.s = append(c.cs.s, "ZRANGEBYSCORE")
	return c
}

func (c Zrangebyscore) Key(key string) ZrangebyscoreKey {
	if c.ks&NoSlot == NoSlot {
		c.ks = NoSlot | slot(key)
	} else {
		c.ks = check(c.ks, slot(key))
	}
	c.cs.s = append(c.cs.s, key)
	return (ZrangebyscoreKey)(c)
}

type ZrangebyscoreKey Completed

func (c ZrangebyscoreKey) Min(min string) ZrangebyscoreMin {
	c.cs.s = append(c.cs.s, min)
	return (ZrangebyscoreMin)(c)
}

type ZrangebyscoreLimit Completed

func (c ZrangebyscoreLimit) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

func (c ZrangebyscoreLimit) Cache() Cacheable {
	c.cs.Build()
	return Cacheable(c)
}

type ZrangebyscoreMax Completed

func (c ZrangebyscoreMax) Withscores() ZrangebyscoreWithscores {
	c.cs.s = append(c.cs.s, "WITHSCORES")
	return (ZrangebyscoreWithscores)(c)
}

func (c ZrangebyscoreMax) Limit(offset int64, count int64) ZrangebyscoreLimit {
	c.cs.s = append(c.cs.s, "LIMIT", strconv.FormatInt(offset, 10), strconv.FormatInt(count, 10))
	return (ZrangebyscoreLimit)(c)
}

func (c ZrangebyscoreMax) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

func (c ZrangebyscoreMax) Cache() Cacheable {
	c.cs.Build()
	return Cacheable(c)
}

type ZrangebyscoreMin Completed

func (c ZrangebyscoreMin) Max(max string) ZrangebyscoreMax {
	c.cs.s = append(c.cs.s, max)
	return (ZrangebyscoreMax)(c)
}

type ZrangebyscoreWithscores Completed

func (c ZrangebyscoreWithscores) Limit(offset int64, count int64) ZrangebyscoreLimit {
	c.cs.s = append(c.cs.s, "LIMIT", strconv.FormatInt(offset, 10), strconv.FormatInt(count, 10))
	return (ZrangebyscoreLimit)(c)
}

func (c ZrangebyscoreWithscores) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

func (c ZrangebyscoreWithscores) Cache() Cacheable {
	c.cs.Build()
	return Cacheable(c)
}

type Zrangestore Completed

func (b Builder) Zrangestore() (c Zrangestore) {
	c = Zrangestore{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "ZRANGESTORE")
	return c
}

func (c Zrangestore) Dst(dst string) ZrangestoreDst {
	if c.ks&NoSlot == NoSlot {
		c.ks = NoSlot | slot(dst)
	} else {
		c.ks = check(c.ks, slot(dst))
	}
	c.cs.s = append(c.cs.s, dst)
	return (ZrangestoreDst)(c)
}

type ZrangestoreDst Completed

func (c ZrangestoreDst) Src(src string) ZrangestoreSrc {
	if c.ks&NoSlot == NoSlot {
		c.ks = NoSlot | slot(src)
	} else {
		c.ks = check(c.ks, slot(src))
	}
	c.cs.s = append(c.cs.s, src)
	return (ZrangestoreSrc)(c)
}

type ZrangestoreLimit Completed

func (c ZrangestoreLimit) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type ZrangestoreMax Completed

func (c ZrangestoreMax) Byscore() ZrangestoreSortbyByscore {
	c.cs.s = append(c.cs.s, "BYSCORE")
	return (ZrangestoreSortbyByscore)(c)
}

func (c ZrangestoreMax) Bylex() ZrangestoreSortbyBylex {
	c.cs.s = append(c.cs.s, "BYLEX")
	return (ZrangestoreSortbyBylex)(c)
}

func (c ZrangestoreMax) Rev() ZrangestoreRev {
	c.cs.s = append(c.cs.s, "REV")
	return (ZrangestoreRev)(c)
}

func (c ZrangestoreMax) Limit(offset int64, count int64) ZrangestoreLimit {
	c.cs.s = append(c.cs.s, "LIMIT", strconv.FormatInt(offset, 10), strconv.FormatInt(count, 10))
	return (ZrangestoreLimit)(c)
}

func (c ZrangestoreMax) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type ZrangestoreMin Completed

func (c ZrangestoreMin) Max(max string) ZrangestoreMax {
	c.cs.s = append(c.cs.s, max)
	return (ZrangestoreMax)(c)
}

type ZrangestoreRev Completed

func (c ZrangestoreRev) Limit(offset int64, count int64) ZrangestoreLimit {
	c.cs.s = append(c.cs.s, "LIMIT", strconv.FormatInt(offset, 10), strconv.FormatInt(count, 10))
	return (ZrangestoreLimit)(c)
}

func (c ZrangestoreRev) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type ZrangestoreSortbyBylex Completed

func (c ZrangestoreSortbyBylex) Rev() ZrangestoreRev {
	c.cs.s = append(c.cs.s, "REV")
	return (ZrangestoreRev)(c)
}

func (c ZrangestoreSortbyBylex) Limit(offset int64, count int64) ZrangestoreLimit {
	c.cs.s = append(c.cs.s, "LIMIT", strconv.FormatInt(offset, 10), strconv.FormatInt(count, 10))
	return (ZrangestoreLimit)(c)
}

func (c ZrangestoreSortbyBylex) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type ZrangestoreSortbyByscore Completed

func (c ZrangestoreSortbyByscore) Rev() ZrangestoreRev {
	c.cs.s = append(c.cs.s, "REV")
	return (ZrangestoreRev)(c)
}

func (c ZrangestoreSortbyByscore) Limit(offset int64, count int64) ZrangestoreLimit {
	c.cs.s = append(c.cs.s, "LIMIT", strconv.FormatInt(offset, 10), strconv.FormatInt(count, 10))
	return (ZrangestoreLimit)(c)
}

func (c ZrangestoreSortbyByscore) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type ZrangestoreSrc Completed

func (c ZrangestoreSrc) Min(min string) ZrangestoreMin {
	c.cs.s = append(c.cs.s, min)
	return (ZrangestoreMin)(c)
}

type Zrank Completed

func (b Builder) Zrank() (c Zrank) {
	c = Zrank{cs: get(), ks: b.ks, cf: readonly}
	c.cs.s = append(c.cs.s, "ZRANK")
	return c
}

func (c Zrank) Key(key string) ZrankKey {
	if c.ks&NoSlot == NoSlot {
		c.ks = NoSlot | slot(key)
	} else {
		c.ks = check(c.ks, slot(key))
	}
	c.cs.s = append(c.cs.s, key)
	return (ZrankKey)(c)
}

type ZrankKey Completed

func (c ZrankKey) Member(member string) ZrankMember {
	c.cs.s = append(c.cs.s, member)
	return (ZrankMember)(c)
}

type ZrankMember Completed

func (c ZrankMember) Withscore() ZrankWithscore {
	c.cs.s = append(c.cs.s, "WITHSCORE")
	return (ZrankWithscore)(c)
}

func (c ZrankMember) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

func (c ZrankMember) Cache() Cacheable {
	c.cs.Build()
	return Cacheable(c)
}

type ZrankWithscore Completed

func (c ZrankWithscore) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

func (c ZrankWithscore) Cache() Cacheable {
	c.cs.Build()
	return Cacheable(c)
}

type Zrem Completed

func (b Builder) Zrem() (c Zrem) {
	c = Zrem{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "ZREM")
	return c
}

func (c Zrem) Key(key string) ZremKey {
	if c.ks&NoSlot == NoSlot {
		c.ks = NoSlot | slot(key)
	} else {
		c.ks = check(c.ks, slot(key))
	}
	c.cs.s = append(c.cs.s, key)
	return (ZremKey)(c)
}

type ZremKey Completed

func (c ZremKey) Member(member ...string) ZremMember {
	c.cs.s = append(c.cs.s, member...)
	return (ZremMember)(c)
}

type ZremMember Completed

func (c ZremMember) Member(member ...string) ZremMember {
	c.cs.s = append(c.cs.s, member...)
	return c
}

func (c ZremMember) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type Zremrangebylex Completed

func (b Builder) Zremrangebylex() (c Zremrangebylex) {
	c = Zremrangebylex{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "ZREMRANGEBYLEX")
	return c
}

func (c Zremrangebylex) Key(key string) ZremrangebylexKey {
	if c.ks&NoSlot == NoSlot {
		c.ks = NoSlot | slot(key)
	} else {
		c.ks = check(c.ks, slot(key))
	}
	c.cs.s = append(c.cs.s, key)
	return (ZremrangebylexKey)(c)
}

type ZremrangebylexKey Completed

func (c ZremrangebylexKey) Min(min string) ZremrangebylexMin {
	c.cs.s = append(c.cs.s, min)
	return (ZremrangebylexMin)(c)
}

type ZremrangebylexMax Completed

func (c ZremrangebylexMax) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type ZremrangebylexMin Completed

func (c ZremrangebylexMin) Max(max string) ZremrangebylexMax {
	c.cs.s = append(c.cs.s, max)
	return (ZremrangebylexMax)(c)
}

type Zremrangebyrank Completed

func (b Builder) Zremrangebyrank() (c Zremrangebyrank) {
	c = Zremrangebyrank{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "ZREMRANGEBYRANK")
	return c
}

func (c Zremrangebyrank) Key(key string) ZremrangebyrankKey {
	if c.ks&NoSlot == NoSlot {
		c.ks = NoSlot | slot(key)
	} else {
		c.ks = check(c.ks, slot(key))
	}
	c.cs.s = append(c.cs.s, key)
	return (ZremrangebyrankKey)(c)
}

type ZremrangebyrankKey Completed

func (c ZremrangebyrankKey) Start(start int64) ZremrangebyrankStart {
	c.cs.s = append(c.cs.s, strconv.FormatInt(start, 10))
	return (ZremrangebyrankStart)(c)
}

type ZremrangebyrankStart Completed

func (c ZremrangebyrankStart) Stop(stop int64) ZremrangebyrankStop {
	c.cs.s = append(c.cs.s, strconv.FormatInt(stop, 10))
	return (ZremrangebyrankStop)(c)
}

type ZremrangebyrankStop Completed

func (c ZremrangebyrankStop) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type Zremrangebyscore Completed

func (b Builder) Zremrangebyscore() (c Zremrangebyscore) {
	c = Zremrangebyscore{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "ZREMRANGEBYSCORE")
	return c
}

func (c Zremrangebyscore) Key(key string) ZremrangebyscoreKey {
	if c.ks&NoSlot == NoSlot {
		c.ks = NoSlot | slot(key)
	} else {
		c.ks = check(c.ks, slot(key))
	}
	c.cs.s = append(c.cs.s, key)
	return (ZremrangebyscoreKey)(c)
}

type ZremrangebyscoreKey Completed

func (c ZremrangebyscoreKey) Min(min string) ZremrangebyscoreMin {
	c.cs.s = append(c.cs.s, min)
	return (ZremrangebyscoreMin)(c)
}

type ZremrangebyscoreMax Completed

func (c ZremrangebyscoreMax) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type ZremrangebyscoreMin Completed

func (c ZremrangebyscoreMin) Max(max string) ZremrangebyscoreMax {
	c.cs.s = append(c.cs.s, max)
	return (ZremrangebyscoreMax)(c)
}

type Zrevrange Completed

func (b Builder) Zrevrange() (c Zrevrange) {
	c = Zrevrange{cs: get(), ks: b.ks, cf: readonly}
	c.cs.s = append(c.cs.s, "ZREVRANGE")
	return c
}

func (c Zrevrange) Key(key string) ZrevrangeKey {
	if c.ks&NoSlot == NoSlot {
		c.ks = NoSlot | slot(key)
	} else {
		c.ks = check(c.ks, slot(key))
	}
	c.cs.s = append(c.cs.s, key)
	return (ZrevrangeKey)(c)
}

type ZrevrangeKey Completed

func (c ZrevrangeKey) Start(start int64) ZrevrangeStart {
	c.cs.s = append(c.cs.s, strconv.FormatInt(start, 10))
	return (ZrevrangeStart)(c)
}

type ZrevrangeStart Completed

func (c ZrevrangeStart) Stop(stop int64) ZrevrangeStop {
	c.cs.s = append(c.cs.s, strconv.FormatInt(stop, 10))
	return (ZrevrangeStop)(c)
}

type ZrevrangeStop Completed

func (c ZrevrangeStop) Withscores() ZrevrangeWithscores {
	c.cs.s = append(c.cs.s, "WITHSCORES")
	return (ZrevrangeWithscores)(c)
}

func (c ZrevrangeStop) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

func (c ZrevrangeStop) Cache() Cacheable {
	c.cs.Build()
	return Cacheable(c)
}

type ZrevrangeWithscores Completed

func (c ZrevrangeWithscores) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

func (c ZrevrangeWithscores) Cache() Cacheable {
	c.cs.Build()
	return Cacheable(c)
}

type Zrevrangebylex Completed

func (b Builder) Zrevrangebylex() (c Zrevrangebylex) {
	c = Zrevrangebylex{cs: get(), ks: b.ks, cf: readonly}
	c.cs.s = append(c.cs.s, "ZREVRANGEBYLEX")
	return c
}

func (c Zrevrangebylex) Key(key string) ZrevrangebylexKey {
	if c.ks&NoSlot == NoSlot {
		c.ks = NoSlot | slot(key)
	} else {
		c.ks = check(c.ks, slot(key))
	}
	c.cs.s = append(c.cs.s, key)
	return (ZrevrangebylexKey)(c)
}

type ZrevrangebylexKey Completed

func (c ZrevrangebylexKey) Max(max string) ZrevrangebylexMax {
	c.cs.s = append(c.cs.s, max)
	return (ZrevrangebylexMax)(c)
}

type ZrevrangebylexLimit Completed

func (c ZrevrangebylexLimit) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

func (c ZrevrangebylexLimit) Cache() Cacheable {
	c.cs.Build()
	return Cacheable(c)
}

type ZrevrangebylexMax Completed

func (c ZrevrangebylexMax) Min(min string) ZrevrangebylexMin {
	c.cs.s = append(c.cs.s, min)
	return (ZrevrangebylexMin)(c)
}

type ZrevrangebylexMin Completed

func (c ZrevrangebylexMin) Limit(offset int64, count int64) ZrevrangebylexLimit {
	c.cs.s = append(c.cs.s, "LIMIT", strconv.FormatInt(offset, 10), strconv.FormatInt(count, 10))
	return (ZrevrangebylexLimit)(c)
}

func (c ZrevrangebylexMin) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

func (c ZrevrangebylexMin) Cache() Cacheable {
	c.cs.Build()
	return Cacheable(c)
}

type Zrevrangebyscore Completed

func (b Builder) Zrevrangebyscore() (c Zrevrangebyscore) {
	c = Zrevrangebyscore{cs: get(), ks: b.ks, cf: readonly}
	c.cs.s = append(c.cs.s, "ZREVRANGEBYSCORE")
	return c
}

func (c Zrevrangebyscore) Key(key string) ZrevrangebyscoreKey {
	if c.ks&NoSlot == NoSlot {
		c.ks = NoSlot | slot(key)
	} else {
		c.ks = check(c.ks, slot(key))
	}
	c.cs.s = append(c.cs.s, key)
	return (ZrevrangebyscoreKey)(c)
}

type ZrevrangebyscoreKey Completed

func (c ZrevrangebyscoreKey) Max(max string) ZrevrangebyscoreMax {
	c.cs.s = append(c.cs.s, max)
	return (ZrevrangebyscoreMax)(c)
}

type ZrevrangebyscoreLimit Completed

func (c ZrevrangebyscoreLimit) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

func (c ZrevrangebyscoreLimit) Cache() Cacheable {
	c.cs.Build()
	return Cacheable(c)
}

type ZrevrangebyscoreMax Completed

func (c ZrevrangebyscoreMax) Min(min string) ZrevrangebyscoreMin {
	c.cs.s = append(c.cs.s, min)
	return (ZrevrangebyscoreMin)(c)
}

type ZrevrangebyscoreMin Completed

func (c ZrevrangebyscoreMin) Withscores() ZrevrangebyscoreWithscores {
	c.cs.s = append(c.cs.s, "WITHSCORES")
	return (ZrevrangebyscoreWithscores)(c)
}

func (c ZrevrangebyscoreMin) Limit(offset int64, count int64) ZrevrangebyscoreLimit {
	c.cs.s = append(c.cs.s, "LIMIT", strconv.FormatInt(offset, 10), strconv.FormatInt(count, 10))
	return (ZrevrangebyscoreLimit)(c)
}

func (c ZrevrangebyscoreMin) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

func (c ZrevrangebyscoreMin) Cache() Cacheable {
	c.cs.Build()
	return Cacheable(c)
}

type ZrevrangebyscoreWithscores Completed

func (c ZrevrangebyscoreWithscores) Limit(offset int64, count int64) ZrevrangebyscoreLimit {
	c.cs.s = append(c.cs.s, "LIMIT", strconv.FormatInt(offset, 10), strconv.FormatInt(count, 10))
	return (ZrevrangebyscoreLimit)(c)
}

func (c ZrevrangebyscoreWithscores) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

func (c ZrevrangebyscoreWithscores) Cache() Cacheable {
	c.cs.Build()
	return Cacheable(c)
}

type Zrevrank Completed

func (b Builder) Zrevrank() (c Zrevrank) {
	c = Zrevrank{cs: get(), ks: b.ks, cf: readonly}
	c.cs.s = append(c.cs.s, "ZREVRANK")
	return c
}

func (c Zrevrank) Key(key string) ZrevrankKey {
	if c.ks&NoSlot == NoSlot {
		c.ks = NoSlot | slot(key)
	} else {
		c.ks = check(c.ks, slot(key))
	}
	c.cs.s = append(c.cs.s, key)
	return (ZrevrankKey)(c)
}

type ZrevrankKey Completed

func (c ZrevrankKey) Member(member string) ZrevrankMember {
	c.cs.s = append(c.cs.s, member)
	return (ZrevrankMember)(c)
}

type ZrevrankMember Completed

func (c ZrevrankMember) Withscore() ZrevrankWithscore {
	c.cs.s = append(c.cs.s, "WITHSCORE")
	return (ZrevrankWithscore)(c)
}

func (c ZrevrankMember) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

func (c ZrevrankMember) Cache() Cacheable {
	c.cs.Build()
	return Cacheable(c)
}

type ZrevrankWithscore Completed

func (c ZrevrankWithscore) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

func (c ZrevrankWithscore) Cache() Cacheable {
	c.cs.Build()
	return Cacheable(c)
}

type Zscan Completed

func (b Builder) Zscan() (c Zscan) {
	c = Zscan{cs: get(), ks: b.ks, cf: readonly}
	c.cs.s = append(c.cs.s, "ZSCAN")
	return c
}

func (c Zscan) Key(key string) ZscanKey {
	if c.ks&NoSlot == NoSlot {
		c.ks = NoSlot | slot(key)
	} else {
		c.ks = check(c.ks, slot(key))
	}
	c.cs.s = append(c.cs.s, key)
	return (ZscanKey)(c)
}

type ZscanCount Completed

func (c ZscanCount) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type ZscanCursor Completed

func (c ZscanCursor) Match(pattern string) ZscanMatch {
	c.cs.s = append(c.cs.s, "MATCH", pattern)
	return (ZscanMatch)(c)
}

func (c ZscanCursor) Count(count int64) ZscanCount {
	c.cs.s = append(c.cs.s, "COUNT", strconv.FormatInt(count, 10))
	return (ZscanCount)(c)
}

func (c ZscanCursor) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type ZscanKey Completed

func (c ZscanKey) Cursor(cursor uint64) ZscanCursor {
	c.cs.s = append(c.cs.s, strconv.FormatUint(cursor, 10))
	return (ZscanCursor)(c)
}

type ZscanMatch Completed

func (c ZscanMatch) Count(count int64) ZscanCount {
	c.cs.s = append(c.cs.s, "COUNT", strconv.FormatInt(count, 10))
	return (ZscanCount)(c)
}

func (c ZscanMatch) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type Zscore Completed

func (b Builder) Zscore() (c Zscore) {
	c = Zscore{cs: get(), ks: b.ks, cf: readonly}
	c.cs.s = append(c.cs.s, "ZSCORE")
	return c
}

func (c Zscore) Key(key string) ZscoreKey {
	if c.ks&NoSlot == NoSlot {
		c.ks = NoSlot | slot(key)
	} else {
		c.ks = check(c.ks, slot(key))
	}
	c.cs.s = append(c.cs.s, key)
	return (ZscoreKey)(c)
}

type ZscoreKey Completed

func (c ZscoreKey) Member(member string) ZscoreMember {
	c.cs.s = append(c.cs.s, member)
	return (ZscoreMember)(c)
}

type ZscoreMember Completed

func (c ZscoreMember) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

func (c ZscoreMember) Cache() Cacheable {
	c.cs.Build()
	return Cacheable(c)
}

type Zunion Completed

func (b Builder) Zunion() (c Zunion) {
	c = Zunion{cs: get(), ks: b.ks, cf: readonly}
	c.cs.s = append(c.cs.s, "ZUNION")
	return c
}

func (c Zunion) Numkeys(numkeys int64) ZunionNumkeys {
	c.cs.s = append(c.cs.s, strconv.FormatInt(numkeys, 10))
	return (ZunionNumkeys)(c)
}

type ZunionAggregateMax Completed

func (c ZunionAggregateMax) Withscores() ZunionWithscores {
	c.cs.s = append(c.cs.s, "WITHSCORES")
	return (ZunionWithscores)(c)
}

func (c ZunionAggregateMax) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type ZunionAggregateMin Completed

func (c ZunionAggregateMin) Withscores() ZunionWithscores {
	c.cs.s = append(c.cs.s, "WITHSCORES")
	return (ZunionWithscores)(c)
}

func (c ZunionAggregateMin) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type ZunionAggregateSum Completed

func (c ZunionAggregateSum) Withscores() ZunionWithscores {
	c.cs.s = append(c.cs.s, "WITHSCORES")
	return (ZunionWithscores)(c)
}

func (c ZunionAggregateSum) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type ZunionKey Completed

func (c ZunionKey) Key(key ...string) ZunionKey {
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

func (c ZunionKey) Weights(weight ...int64) ZunionWeights {
	c.cs.s = append(c.cs.s, "WEIGHTS")
	for _, n := range weight {
		c.cs.s = append(c.cs.s, strconv.FormatInt(n, 10))
	}
	return (ZunionWeights)(c)
}

func (c ZunionKey) AggregateSum() ZunionAggregateSum {
	c.cs.s = append(c.cs.s, "AGGREGATE", "SUM")
	return (ZunionAggregateSum)(c)
}

func (c ZunionKey) AggregateMin() ZunionAggregateMin {
	c.cs.s = append(c.cs.s, "AGGREGATE", "MIN")
	return (ZunionAggregateMin)(c)
}

func (c ZunionKey) AggregateMax() ZunionAggregateMax {
	c.cs.s = append(c.cs.s, "AGGREGATE", "MAX")
	return (ZunionAggregateMax)(c)
}

func (c ZunionKey) Withscores() ZunionWithscores {
	c.cs.s = append(c.cs.s, "WITHSCORES")
	return (ZunionWithscores)(c)
}

func (c ZunionKey) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type ZunionNumkeys Completed

func (c ZunionNumkeys) Key(key ...string) ZunionKey {
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
	return (ZunionKey)(c)
}

type ZunionWeights Completed

func (c ZunionWeights) Weights(weight ...int64) ZunionWeights {
	c.cs.s = append(c.cs.s, "WEIGHTS")
	for _, n := range weight {
		c.cs.s = append(c.cs.s, strconv.FormatInt(n, 10))
	}
	return c
}

func (c ZunionWeights) AggregateSum() ZunionAggregateSum {
	c.cs.s = append(c.cs.s, "AGGREGATE", "SUM")
	return (ZunionAggregateSum)(c)
}

func (c ZunionWeights) AggregateMin() ZunionAggregateMin {
	c.cs.s = append(c.cs.s, "AGGREGATE", "MIN")
	return (ZunionAggregateMin)(c)
}

func (c ZunionWeights) AggregateMax() ZunionAggregateMax {
	c.cs.s = append(c.cs.s, "AGGREGATE", "MAX")
	return (ZunionAggregateMax)(c)
}

func (c ZunionWeights) Withscores() ZunionWithscores {
	c.cs.s = append(c.cs.s, "WITHSCORES")
	return (ZunionWithscores)(c)
}

func (c ZunionWeights) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type ZunionWithscores Completed

func (c ZunionWithscores) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type Zunionstore Completed

func (b Builder) Zunionstore() (c Zunionstore) {
	c = Zunionstore{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "ZUNIONSTORE")
	return c
}

func (c Zunionstore) Destination(destination string) ZunionstoreDestination {
	if c.ks&NoSlot == NoSlot {
		c.ks = NoSlot | slot(destination)
	} else {
		c.ks = check(c.ks, slot(destination))
	}
	c.cs.s = append(c.cs.s, destination)
	return (ZunionstoreDestination)(c)
}

type ZunionstoreAggregateMax Completed

func (c ZunionstoreAggregateMax) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type ZunionstoreAggregateMin Completed

func (c ZunionstoreAggregateMin) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type ZunionstoreAggregateSum Completed

func (c ZunionstoreAggregateSum) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type ZunionstoreDestination Completed

func (c ZunionstoreDestination) Numkeys(numkeys int64) ZunionstoreNumkeys {
	c.cs.s = append(c.cs.s, strconv.FormatInt(numkeys, 10))
	return (ZunionstoreNumkeys)(c)
}

type ZunionstoreKey Completed

func (c ZunionstoreKey) Key(key ...string) ZunionstoreKey {
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

func (c ZunionstoreKey) Weights(weight ...int64) ZunionstoreWeights {
	c.cs.s = append(c.cs.s, "WEIGHTS")
	for _, n := range weight {
		c.cs.s = append(c.cs.s, strconv.FormatInt(n, 10))
	}
	return (ZunionstoreWeights)(c)
}

func (c ZunionstoreKey) AggregateSum() ZunionstoreAggregateSum {
	c.cs.s = append(c.cs.s, "AGGREGATE", "SUM")
	return (ZunionstoreAggregateSum)(c)
}

func (c ZunionstoreKey) AggregateMin() ZunionstoreAggregateMin {
	c.cs.s = append(c.cs.s, "AGGREGATE", "MIN")
	return (ZunionstoreAggregateMin)(c)
}

func (c ZunionstoreKey) AggregateMax() ZunionstoreAggregateMax {
	c.cs.s = append(c.cs.s, "AGGREGATE", "MAX")
	return (ZunionstoreAggregateMax)(c)
}

func (c ZunionstoreKey) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type ZunionstoreNumkeys Completed

func (c ZunionstoreNumkeys) Key(key ...string) ZunionstoreKey {
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
	return (ZunionstoreKey)(c)
}

type ZunionstoreWeights Completed

func (c ZunionstoreWeights) Weights(weight ...int64) ZunionstoreWeights {
	c.cs.s = append(c.cs.s, "WEIGHTS")
	for _, n := range weight {
		c.cs.s = append(c.cs.s, strconv.FormatInt(n, 10))
	}
	return c
}

func (c ZunionstoreWeights) AggregateSum() ZunionstoreAggregateSum {
	c.cs.s = append(c.cs.s, "AGGREGATE", "SUM")
	return (ZunionstoreAggregateSum)(c)
}

func (c ZunionstoreWeights) AggregateMin() ZunionstoreAggregateMin {
	c.cs.s = append(c.cs.s, "AGGREGATE", "MIN")
	return (ZunionstoreAggregateMin)(c)
}

func (c ZunionstoreWeights) AggregateMax() ZunionstoreAggregateMax {
	c.cs.s = append(c.cs.s, "AGGREGATE", "MAX")
	return (ZunionstoreAggregateMax)(c)
}

func (c ZunionstoreWeights) Build() Completed {
	c.cs.Build()
	return Completed(c)
}
