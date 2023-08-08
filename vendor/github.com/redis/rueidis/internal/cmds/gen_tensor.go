// Code generated DO NOT EDIT

package cmds

import "strconv"

type AiTensorget Completed

func (b Builder) AiTensorget() (c AiTensorget) {
	c = AiTensorget{cs: get(), ks: b.ks, cf: readonly}
	c.cs.s = append(c.cs.s, "AI.TENSORGET")
	return c
}

func (c AiTensorget) Key(key string) AiTensorgetKey {
	if c.ks&NoSlot == NoSlot {
		c.ks = NoSlot | slot(key)
	} else {
		c.ks = check(c.ks, slot(key))
	}
	c.cs.s = append(c.cs.s, key)
	return (AiTensorgetKey)(c)
}

type AiTensorgetFormatBlob Completed

func (c AiTensorgetFormatBlob) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

func (c AiTensorgetFormatBlob) Cache() Cacheable {
	c.cs.Build()
	return Cacheable(c)
}

type AiTensorgetFormatValues Completed

func (c AiTensorgetFormatValues) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

func (c AiTensorgetFormatValues) Cache() Cacheable {
	c.cs.Build()
	return Cacheable(c)
}

type AiTensorgetKey Completed

func (c AiTensorgetKey) Meta() AiTensorgetMeta {
	c.cs.s = append(c.cs.s, "META")
	return (AiTensorgetMeta)(c)
}

type AiTensorgetMeta Completed

func (c AiTensorgetMeta) Blob() AiTensorgetFormatBlob {
	c.cs.s = append(c.cs.s, "BLOB")
	return (AiTensorgetFormatBlob)(c)
}

func (c AiTensorgetMeta) Values() AiTensorgetFormatValues {
	c.cs.s = append(c.cs.s, "VALUES")
	return (AiTensorgetFormatValues)(c)
}

func (c AiTensorgetMeta) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

func (c AiTensorgetMeta) Cache() Cacheable {
	c.cs.Build()
	return Cacheable(c)
}

type AiTensorset Completed

func (b Builder) AiTensorset() (c AiTensorset) {
	c = AiTensorset{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "AI.TENSORSET")
	return c
}

func (c AiTensorset) Key(key string) AiTensorsetKey {
	if c.ks&NoSlot == NoSlot {
		c.ks = NoSlot | slot(key)
	} else {
		c.ks = check(c.ks, slot(key))
	}
	c.cs.s = append(c.cs.s, key)
	return (AiTensorsetKey)(c)
}

type AiTensorsetBlob Completed

func (c AiTensorsetBlob) Values(value ...string) AiTensorsetValues {
	c.cs.s = append(c.cs.s, "VALUES")
	c.cs.s = append(c.cs.s, value...)
	return (AiTensorsetValues)(c)
}

func (c AiTensorsetBlob) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type AiTensorsetKey Completed

func (c AiTensorsetKey) Float() AiTensorsetTypeFloat {
	c.cs.s = append(c.cs.s, "FLOAT")
	return (AiTensorsetTypeFloat)(c)
}

func (c AiTensorsetKey) Double() AiTensorsetTypeDouble {
	c.cs.s = append(c.cs.s, "DOUBLE")
	return (AiTensorsetTypeDouble)(c)
}

func (c AiTensorsetKey) Int8() AiTensorsetTypeInt8 {
	c.cs.s = append(c.cs.s, "INT8")
	return (AiTensorsetTypeInt8)(c)
}

func (c AiTensorsetKey) Int16() AiTensorsetTypeInt16 {
	c.cs.s = append(c.cs.s, "INT16")
	return (AiTensorsetTypeInt16)(c)
}

func (c AiTensorsetKey) Int32() AiTensorsetTypeInt32 {
	c.cs.s = append(c.cs.s, "INT32")
	return (AiTensorsetTypeInt32)(c)
}

func (c AiTensorsetKey) Int64() AiTensorsetTypeInt64 {
	c.cs.s = append(c.cs.s, "INT64")
	return (AiTensorsetTypeInt64)(c)
}

func (c AiTensorsetKey) Uint8() AiTensorsetTypeUint8 {
	c.cs.s = append(c.cs.s, "UINT8")
	return (AiTensorsetTypeUint8)(c)
}

func (c AiTensorsetKey) Uint16() AiTensorsetTypeUint16 {
	c.cs.s = append(c.cs.s, "UINT16")
	return (AiTensorsetTypeUint16)(c)
}

func (c AiTensorsetKey) String() AiTensorsetTypeString {
	c.cs.s = append(c.cs.s, "STRING")
	return (AiTensorsetTypeString)(c)
}

func (c AiTensorsetKey) Bool() AiTensorsetTypeBool {
	c.cs.s = append(c.cs.s, "BOOL")
	return (AiTensorsetTypeBool)(c)
}

type AiTensorsetShape Completed

func (c AiTensorsetShape) Shape(shape ...int64) AiTensorsetShape {
	for _, n := range shape {
		c.cs.s = append(c.cs.s, strconv.FormatInt(n, 10))
	}
	return c
}

func (c AiTensorsetShape) Blob(blob string) AiTensorsetBlob {
	c.cs.s = append(c.cs.s, "BLOB", blob)
	return (AiTensorsetBlob)(c)
}

func (c AiTensorsetShape) Values(value ...string) AiTensorsetValues {
	c.cs.s = append(c.cs.s, "VALUES")
	c.cs.s = append(c.cs.s, value...)
	return (AiTensorsetValues)(c)
}

func (c AiTensorsetShape) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type AiTensorsetTypeBool Completed

func (c AiTensorsetTypeBool) Shape(shape ...int64) AiTensorsetShape {
	for _, n := range shape {
		c.cs.s = append(c.cs.s, strconv.FormatInt(n, 10))
	}
	return (AiTensorsetShape)(c)
}

type AiTensorsetTypeDouble Completed

func (c AiTensorsetTypeDouble) Shape(shape ...int64) AiTensorsetShape {
	for _, n := range shape {
		c.cs.s = append(c.cs.s, strconv.FormatInt(n, 10))
	}
	return (AiTensorsetShape)(c)
}

type AiTensorsetTypeFloat Completed

func (c AiTensorsetTypeFloat) Shape(shape ...int64) AiTensorsetShape {
	for _, n := range shape {
		c.cs.s = append(c.cs.s, strconv.FormatInt(n, 10))
	}
	return (AiTensorsetShape)(c)
}

type AiTensorsetTypeInt16 Completed

func (c AiTensorsetTypeInt16) Shape(shape ...int64) AiTensorsetShape {
	for _, n := range shape {
		c.cs.s = append(c.cs.s, strconv.FormatInt(n, 10))
	}
	return (AiTensorsetShape)(c)
}

type AiTensorsetTypeInt32 Completed

func (c AiTensorsetTypeInt32) Shape(shape ...int64) AiTensorsetShape {
	for _, n := range shape {
		c.cs.s = append(c.cs.s, strconv.FormatInt(n, 10))
	}
	return (AiTensorsetShape)(c)
}

type AiTensorsetTypeInt64 Completed

func (c AiTensorsetTypeInt64) Shape(shape ...int64) AiTensorsetShape {
	for _, n := range shape {
		c.cs.s = append(c.cs.s, strconv.FormatInt(n, 10))
	}
	return (AiTensorsetShape)(c)
}

type AiTensorsetTypeInt8 Completed

func (c AiTensorsetTypeInt8) Shape(shape ...int64) AiTensorsetShape {
	for _, n := range shape {
		c.cs.s = append(c.cs.s, strconv.FormatInt(n, 10))
	}
	return (AiTensorsetShape)(c)
}

type AiTensorsetTypeString Completed

func (c AiTensorsetTypeString) Shape(shape ...int64) AiTensorsetShape {
	for _, n := range shape {
		c.cs.s = append(c.cs.s, strconv.FormatInt(n, 10))
	}
	return (AiTensorsetShape)(c)
}

type AiTensorsetTypeUint16 Completed

func (c AiTensorsetTypeUint16) Shape(shape ...int64) AiTensorsetShape {
	for _, n := range shape {
		c.cs.s = append(c.cs.s, strconv.FormatInt(n, 10))
	}
	return (AiTensorsetShape)(c)
}

type AiTensorsetTypeUint8 Completed

func (c AiTensorsetTypeUint8) Shape(shape ...int64) AiTensorsetShape {
	for _, n := range shape {
		c.cs.s = append(c.cs.s, strconv.FormatInt(n, 10))
	}
	return (AiTensorsetShape)(c)
}

type AiTensorsetValues Completed

func (c AiTensorsetValues) Values(value ...string) AiTensorsetValues {
	c.cs.s = append(c.cs.s, "VALUES")
	c.cs.s = append(c.cs.s, value...)
	return c
}

func (c AiTensorsetValues) Build() Completed {
	c.cs.Build()
	return Completed(c)
}
