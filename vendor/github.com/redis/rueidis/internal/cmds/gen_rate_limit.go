// Code generated DO NOT EDIT

package cmds

import "strconv"

type Gcra Incomplete

func (b Builder) Gcra() (c Gcra) {
	c = Gcra{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "GCRA")
	return c
}

func (c Gcra) Key(key string) GcraKey {
	if c.ks&NoSlot == NoSlot {
		c.ks = NoSlot | slot(key)
	} else {
		c.ks = check(c.ks, slot(key))
	}
	c.cs.s = append(c.cs.s, key)
	return (GcraKey)(c)
}

type GcraCount Incomplete

func (c GcraCount) Build() Completed {
	c.cs.Build()
	return Completed{cs: c.cs, cf: uint16(c.cf), ks: c.ks}
}

type GcraKey Incomplete

func (c GcraKey) MaxBurst(maxBurst int64) GcraMaxBurst {
	c.cs.s = append(c.cs.s, strconv.FormatInt(maxBurst, 10))
	return (GcraMaxBurst)(c)
}

type GcraMaxBurst Incomplete

func (c GcraMaxBurst) TokensPerPeriod(tokensPerPeriod int64) GcraTokensPerPeriod {
	c.cs.s = append(c.cs.s, strconv.FormatInt(tokensPerPeriod, 10))
	return (GcraTokensPerPeriod)(c)
}

type GcraPeriod Incomplete

func (c GcraPeriod) Count(count int64) GcraCount {
	c.cs.s = append(c.cs.s, "TOKENS", strconv.FormatInt(count, 10))
	return (GcraCount)(c)
}

func (c GcraPeriod) Build() Completed {
	c.cs.Build()
	return Completed{cs: c.cs, cf: uint16(c.cf), ks: c.ks}
}

type GcraTokensPerPeriod Incomplete

func (c GcraTokensPerPeriod) Period(period float64) GcraPeriod {
	c.cs.s = append(c.cs.s, strconv.FormatFloat(period, 'f', -1, 64))
	return (GcraPeriod)(c)
}
