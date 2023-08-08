// Code generated DO NOT EDIT

package cmds

import "strconv"

type AiScriptdel Completed

func (b Builder) AiScriptdel() (c AiScriptdel) {
	c = AiScriptdel{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "AI.SCRIPTDEL")
	return c
}

func (c AiScriptdel) Key(key string) AiScriptdelKey {
	if c.ks&NoSlot == NoSlot {
		c.ks = NoSlot | slot(key)
	} else {
		c.ks = check(c.ks, slot(key))
	}
	c.cs.s = append(c.cs.s, key)
	return (AiScriptdelKey)(c)
}

type AiScriptdelKey Completed

func (c AiScriptdelKey) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type AiScriptget Completed

func (b Builder) AiScriptget() (c AiScriptget) {
	c = AiScriptget{cs: get(), ks: b.ks, cf: readonly}
	c.cs.s = append(c.cs.s, "AI.SCRIPTGET")
	return c
}

func (c AiScriptget) Key(key string) AiScriptgetKey {
	if c.ks&NoSlot == NoSlot {
		c.ks = NoSlot | slot(key)
	} else {
		c.ks = check(c.ks, slot(key))
	}
	c.cs.s = append(c.cs.s, key)
	return (AiScriptgetKey)(c)
}

type AiScriptgetKey Completed

func (c AiScriptgetKey) Meta() AiScriptgetMeta {
	c.cs.s = append(c.cs.s, "META")
	return (AiScriptgetMeta)(c)
}

func (c AiScriptgetKey) Source() AiScriptgetSource {
	c.cs.s = append(c.cs.s, "SOURCE")
	return (AiScriptgetSource)(c)
}

func (c AiScriptgetKey) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

func (c AiScriptgetKey) Cache() Cacheable {
	c.cs.Build()
	return Cacheable(c)
}

type AiScriptgetMeta Completed

func (c AiScriptgetMeta) Source() AiScriptgetSource {
	c.cs.s = append(c.cs.s, "SOURCE")
	return (AiScriptgetSource)(c)
}

func (c AiScriptgetMeta) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

func (c AiScriptgetMeta) Cache() Cacheable {
	c.cs.Build()
	return Cacheable(c)
}

type AiScriptgetSource Completed

func (c AiScriptgetSource) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

func (c AiScriptgetSource) Cache() Cacheable {
	c.cs.Build()
	return Cacheable(c)
}

type AiScriptstore Completed

func (b Builder) AiScriptstore() (c AiScriptstore) {
	c = AiScriptstore{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "AI.SCRIPTSTORE")
	return c
}

func (c AiScriptstore) Key(key string) AiScriptstoreKey {
	if c.ks&NoSlot == NoSlot {
		c.ks = NoSlot | slot(key)
	} else {
		c.ks = check(c.ks, slot(key))
	}
	c.cs.s = append(c.cs.s, key)
	return (AiScriptstoreKey)(c)
}

type AiScriptstoreDeviceCpu Completed

func (c AiScriptstoreDeviceCpu) Tag(tag string) AiScriptstoreTag {
	c.cs.s = append(c.cs.s, "TAG", tag)
	return (AiScriptstoreTag)(c)
}

func (c AiScriptstoreDeviceCpu) EntryPoints(entryPointCount int64) AiScriptstoreEntryPointsEntryPoints {
	c.cs.s = append(c.cs.s, "ENTRY_POINTS", strconv.FormatInt(entryPointCount, 10))
	return (AiScriptstoreEntryPointsEntryPoints)(c)
}

type AiScriptstoreDeviceGpu Completed

func (c AiScriptstoreDeviceGpu) Tag(tag string) AiScriptstoreTag {
	c.cs.s = append(c.cs.s, "TAG", tag)
	return (AiScriptstoreTag)(c)
}

func (c AiScriptstoreDeviceGpu) EntryPoints(entryPointCount int64) AiScriptstoreEntryPointsEntryPoints {
	c.cs.s = append(c.cs.s, "ENTRY_POINTS", strconv.FormatInt(entryPointCount, 10))
	return (AiScriptstoreEntryPointsEntryPoints)(c)
}

type AiScriptstoreEntryPointsEntryPoint Completed

func (c AiScriptstoreEntryPointsEntryPoint) EntryPoint(entryPoint ...string) AiScriptstoreEntryPointsEntryPoint {
	c.cs.s = append(c.cs.s, entryPoint...)
	return c
}

func (c AiScriptstoreEntryPointsEntryPoint) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type AiScriptstoreEntryPointsEntryPoints Completed

func (c AiScriptstoreEntryPointsEntryPoints) EntryPoint(entryPoint ...string) AiScriptstoreEntryPointsEntryPoint {
	c.cs.s = append(c.cs.s, entryPoint...)
	return (AiScriptstoreEntryPointsEntryPoint)(c)
}

type AiScriptstoreKey Completed

func (c AiScriptstoreKey) Cpu() AiScriptstoreDeviceCpu {
	c.cs.s = append(c.cs.s, "CPU")
	return (AiScriptstoreDeviceCpu)(c)
}

func (c AiScriptstoreKey) Gpu() AiScriptstoreDeviceGpu {
	c.cs.s = append(c.cs.s, "GPU")
	return (AiScriptstoreDeviceGpu)(c)
}

type AiScriptstoreTag Completed

func (c AiScriptstoreTag) EntryPoints(entryPointCount int64) AiScriptstoreEntryPointsEntryPoints {
	c.cs.s = append(c.cs.s, "ENTRY_POINTS", strconv.FormatInt(entryPointCount, 10))
	return (AiScriptstoreEntryPointsEntryPoints)(c)
}
