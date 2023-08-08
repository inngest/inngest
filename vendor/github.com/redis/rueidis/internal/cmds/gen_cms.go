// Code generated DO NOT EDIT

package cmds

import "strconv"

type CmsIncrby Completed

func (b Builder) CmsIncrby() (c CmsIncrby) {
	c = CmsIncrby{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "CMS.INCRBY")
	return c
}

func (c CmsIncrby) Key(key string) CmsIncrbyKey {
	if c.ks&NoSlot == NoSlot {
		c.ks = NoSlot | slot(key)
	} else {
		c.ks = check(c.ks, slot(key))
	}
	c.cs.s = append(c.cs.s, key)
	return (CmsIncrbyKey)(c)
}

type CmsIncrbyItemsIncrement Completed

func (c CmsIncrbyItemsIncrement) Item(item string) CmsIncrbyItemsItem {
	c.cs.s = append(c.cs.s, item)
	return (CmsIncrbyItemsItem)(c)
}

func (c CmsIncrbyItemsIncrement) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type CmsIncrbyItemsItem Completed

func (c CmsIncrbyItemsItem) Increment(increment int64) CmsIncrbyItemsIncrement {
	c.cs.s = append(c.cs.s, strconv.FormatInt(increment, 10))
	return (CmsIncrbyItemsIncrement)(c)
}

type CmsIncrbyKey Completed

func (c CmsIncrbyKey) Item(item string) CmsIncrbyItemsItem {
	c.cs.s = append(c.cs.s, item)
	return (CmsIncrbyItemsItem)(c)
}

type CmsInfo Completed

func (b Builder) CmsInfo() (c CmsInfo) {
	c = CmsInfo{cs: get(), ks: b.ks, cf: readonly}
	c.cs.s = append(c.cs.s, "CMS.INFO")
	return c
}

func (c CmsInfo) Key(key string) CmsInfoKey {
	if c.ks&NoSlot == NoSlot {
		c.ks = NoSlot | slot(key)
	} else {
		c.ks = check(c.ks, slot(key))
	}
	c.cs.s = append(c.cs.s, key)
	return (CmsInfoKey)(c)
}

type CmsInfoKey Completed

func (c CmsInfoKey) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

func (c CmsInfoKey) Cache() Cacheable {
	c.cs.Build()
	return Cacheable(c)
}

type CmsInitbydim Completed

func (b Builder) CmsInitbydim() (c CmsInitbydim) {
	c = CmsInitbydim{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "CMS.INITBYDIM")
	return c
}

func (c CmsInitbydim) Key(key string) CmsInitbydimKey {
	if c.ks&NoSlot == NoSlot {
		c.ks = NoSlot | slot(key)
	} else {
		c.ks = check(c.ks, slot(key))
	}
	c.cs.s = append(c.cs.s, key)
	return (CmsInitbydimKey)(c)
}

type CmsInitbydimDepth Completed

func (c CmsInitbydimDepth) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type CmsInitbydimKey Completed

func (c CmsInitbydimKey) Width(width int64) CmsInitbydimWidth {
	c.cs.s = append(c.cs.s, strconv.FormatInt(width, 10))
	return (CmsInitbydimWidth)(c)
}

type CmsInitbydimWidth Completed

func (c CmsInitbydimWidth) Depth(depth int64) CmsInitbydimDepth {
	c.cs.s = append(c.cs.s, strconv.FormatInt(depth, 10))
	return (CmsInitbydimDepth)(c)
}

type CmsInitbyprob Completed

func (b Builder) CmsInitbyprob() (c CmsInitbyprob) {
	c = CmsInitbyprob{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "CMS.INITBYPROB")
	return c
}

func (c CmsInitbyprob) Key(key string) CmsInitbyprobKey {
	if c.ks&NoSlot == NoSlot {
		c.ks = NoSlot | slot(key)
	} else {
		c.ks = check(c.ks, slot(key))
	}
	c.cs.s = append(c.cs.s, key)
	return (CmsInitbyprobKey)(c)
}

type CmsInitbyprobError Completed

func (c CmsInitbyprobError) Probability(probability float64) CmsInitbyprobProbability {
	c.cs.s = append(c.cs.s, strconv.FormatFloat(probability, 'f', -1, 64))
	return (CmsInitbyprobProbability)(c)
}

type CmsInitbyprobKey Completed

func (c CmsInitbyprobKey) Error(error float64) CmsInitbyprobError {
	c.cs.s = append(c.cs.s, strconv.FormatFloat(error, 'f', -1, 64))
	return (CmsInitbyprobError)(c)
}

type CmsInitbyprobProbability Completed

func (c CmsInitbyprobProbability) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type CmsMerge Completed

func (b Builder) CmsMerge() (c CmsMerge) {
	c = CmsMerge{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "CMS.MERGE")
	return c
}

func (c CmsMerge) Destination(destination string) CmsMergeDestination {
	if c.ks&NoSlot == NoSlot {
		c.ks = NoSlot | slot(destination)
	} else {
		c.ks = check(c.ks, slot(destination))
	}
	c.cs.s = append(c.cs.s, destination)
	return (CmsMergeDestination)(c)
}

type CmsMergeDestination Completed

func (c CmsMergeDestination) Numkeys(numkeys int64) CmsMergeNumkeys {
	c.cs.s = append(c.cs.s, strconv.FormatInt(numkeys, 10))
	return (CmsMergeNumkeys)(c)
}

type CmsMergeNumkeys Completed

func (c CmsMergeNumkeys) Source(source ...string) CmsMergeSource {
	if c.ks&NoSlot == NoSlot {
		for _, k := range source {
			c.ks = NoSlot | slot(k)
			break
		}
	} else {
		for _, k := range source {
			c.ks = check(c.ks, slot(k))
		}
	}
	c.cs.s = append(c.cs.s, source...)
	return (CmsMergeSource)(c)
}

type CmsMergeSource Completed

func (c CmsMergeSource) Source(source ...string) CmsMergeSource {
	if c.ks&NoSlot == NoSlot {
		for _, k := range source {
			c.ks = NoSlot | slot(k)
			break
		}
	} else {
		for _, k := range source {
			c.ks = check(c.ks, slot(k))
		}
	}
	c.cs.s = append(c.cs.s, source...)
	return c
}

func (c CmsMergeSource) Weights() CmsMergeWeightWeights {
	c.cs.s = append(c.cs.s, "WEIGHTS")
	return (CmsMergeWeightWeights)(c)
}

func (c CmsMergeSource) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type CmsMergeWeightWeight Completed

func (c CmsMergeWeightWeight) Weight(weight ...float64) CmsMergeWeightWeight {
	for _, n := range weight {
		c.cs.s = append(c.cs.s, strconv.FormatFloat(n, 'f', -1, 64))
	}
	return c
}

func (c CmsMergeWeightWeight) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type CmsMergeWeightWeights Completed

func (c CmsMergeWeightWeights) Weight(weight ...float64) CmsMergeWeightWeight {
	for _, n := range weight {
		c.cs.s = append(c.cs.s, strconv.FormatFloat(n, 'f', -1, 64))
	}
	return (CmsMergeWeightWeight)(c)
}

type CmsQuery Completed

func (b Builder) CmsQuery() (c CmsQuery) {
	c = CmsQuery{cs: get(), ks: b.ks, cf: readonly}
	c.cs.s = append(c.cs.s, "CMS.QUERY")
	return c
}

func (c CmsQuery) Key(key string) CmsQueryKey {
	if c.ks&NoSlot == NoSlot {
		c.ks = NoSlot | slot(key)
	} else {
		c.ks = check(c.ks, slot(key))
	}
	c.cs.s = append(c.cs.s, key)
	return (CmsQueryKey)(c)
}

type CmsQueryItem Completed

func (c CmsQueryItem) Item(item ...string) CmsQueryItem {
	c.cs.s = append(c.cs.s, item...)
	return c
}

func (c CmsQueryItem) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

func (c CmsQueryItem) Cache() Cacheable {
	c.cs.Build()
	return Cacheable(c)
}

type CmsQueryKey Completed

func (c CmsQueryKey) Item(item ...string) CmsQueryItem {
	c.cs.s = append(c.cs.s, item...)
	return (CmsQueryItem)(c)
}
