// Code generated DO NOT EDIT

package cmds

type Discard Completed

func (b Builder) Discard() (c Discard) {
	c = Discard{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "DISCARD")
	return c
}

func (c Discard) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type Exec Completed

func (b Builder) Exec() (c Exec) {
	c = Exec{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "EXEC")
	return c
}

func (c Exec) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type Multi Completed

func (b Builder) Multi() (c Multi) {
	c = Multi{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "MULTI")
	return c
}

func (c Multi) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type Unwatch Completed

func (b Builder) Unwatch() (c Unwatch) {
	c = Unwatch{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "UNWATCH")
	return c
}

func (c Unwatch) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type Watch Completed

func (b Builder) Watch() (c Watch) {
	c = Watch{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "WATCH")
	return c
}

func (c Watch) Key(key ...string) WatchKey {
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
	return (WatchKey)(c)
}

type WatchKey Completed

func (c WatchKey) Key(key ...string) WatchKey {
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

func (c WatchKey) Build() Completed {
	c.cs.Build()
	return Completed(c)
}
