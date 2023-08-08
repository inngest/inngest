// Code generated DO NOT EDIT

package cmds

import "strconv"

type Copy Completed

func (b Builder) Copy() (c Copy) {
	c = Copy{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "COPY")
	return c
}

func (c Copy) Source(source string) CopySource {
	if c.ks&NoSlot == NoSlot {
		c.ks = NoSlot | slot(source)
	} else {
		c.ks = check(c.ks, slot(source))
	}
	c.cs.s = append(c.cs.s, source)
	return (CopySource)(c)
}

type CopyDb Completed

func (c CopyDb) Replace() CopyReplace {
	c.cs.s = append(c.cs.s, "REPLACE")
	return (CopyReplace)(c)
}

func (c CopyDb) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type CopyDestination Completed

func (c CopyDestination) Db(destinationDb int64) CopyDb {
	c.cs.s = append(c.cs.s, "DB", strconv.FormatInt(destinationDb, 10))
	return (CopyDb)(c)
}

func (c CopyDestination) Replace() CopyReplace {
	c.cs.s = append(c.cs.s, "REPLACE")
	return (CopyReplace)(c)
}

func (c CopyDestination) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type CopyReplace Completed

func (c CopyReplace) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type CopySource Completed

func (c CopySource) Destination(destination string) CopyDestination {
	if c.ks&NoSlot == NoSlot {
		c.ks = NoSlot | slot(destination)
	} else {
		c.ks = check(c.ks, slot(destination))
	}
	c.cs.s = append(c.cs.s, destination)
	return (CopyDestination)(c)
}

type Del Completed

func (b Builder) Del() (c Del) {
	c = Del{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "DEL")
	return c
}

func (c Del) Key(key ...string) DelKey {
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
	return (DelKey)(c)
}

type DelKey Completed

func (c DelKey) Key(key ...string) DelKey {
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

func (c DelKey) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type Dump Completed

func (b Builder) Dump() (c Dump) {
	c = Dump{cs: get(), ks: b.ks, cf: readonly}
	c.cs.s = append(c.cs.s, "DUMP")
	return c
}

func (c Dump) Key(key string) DumpKey {
	if c.ks&NoSlot == NoSlot {
		c.ks = NoSlot | slot(key)
	} else {
		c.ks = check(c.ks, slot(key))
	}
	c.cs.s = append(c.cs.s, key)
	return (DumpKey)(c)
}

type DumpKey Completed

func (c DumpKey) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type Exists Completed

func (b Builder) Exists() (c Exists) {
	c = Exists{cs: get(), ks: b.ks, cf: readonly}
	c.cs.s = append(c.cs.s, "EXISTS")
	return c
}

func (c Exists) Key(key ...string) ExistsKey {
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
	return (ExistsKey)(c)
}

type ExistsKey Completed

func (c ExistsKey) Key(key ...string) ExistsKey {
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

func (c ExistsKey) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type Expire Completed

func (b Builder) Expire() (c Expire) {
	c = Expire{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "EXPIRE")
	return c
}

func (c Expire) Key(key string) ExpireKey {
	if c.ks&NoSlot == NoSlot {
		c.ks = NoSlot | slot(key)
	} else {
		c.ks = check(c.ks, slot(key))
	}
	c.cs.s = append(c.cs.s, key)
	return (ExpireKey)(c)
}

type ExpireConditionGt Completed

func (c ExpireConditionGt) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type ExpireConditionLt Completed

func (c ExpireConditionLt) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type ExpireConditionNx Completed

func (c ExpireConditionNx) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type ExpireConditionXx Completed

func (c ExpireConditionXx) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type ExpireKey Completed

func (c ExpireKey) Seconds(seconds int64) ExpireSeconds {
	c.cs.s = append(c.cs.s, strconv.FormatInt(seconds, 10))
	return (ExpireSeconds)(c)
}

type ExpireSeconds Completed

func (c ExpireSeconds) Nx() ExpireConditionNx {
	c.cs.s = append(c.cs.s, "NX")
	return (ExpireConditionNx)(c)
}

func (c ExpireSeconds) Xx() ExpireConditionXx {
	c.cs.s = append(c.cs.s, "XX")
	return (ExpireConditionXx)(c)
}

func (c ExpireSeconds) Gt() ExpireConditionGt {
	c.cs.s = append(c.cs.s, "GT")
	return (ExpireConditionGt)(c)
}

func (c ExpireSeconds) Lt() ExpireConditionLt {
	c.cs.s = append(c.cs.s, "LT")
	return (ExpireConditionLt)(c)
}

func (c ExpireSeconds) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type Expireat Completed

func (b Builder) Expireat() (c Expireat) {
	c = Expireat{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "EXPIREAT")
	return c
}

func (c Expireat) Key(key string) ExpireatKey {
	if c.ks&NoSlot == NoSlot {
		c.ks = NoSlot | slot(key)
	} else {
		c.ks = check(c.ks, slot(key))
	}
	c.cs.s = append(c.cs.s, key)
	return (ExpireatKey)(c)
}

type ExpireatConditionGt Completed

func (c ExpireatConditionGt) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type ExpireatConditionLt Completed

func (c ExpireatConditionLt) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type ExpireatConditionNx Completed

func (c ExpireatConditionNx) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type ExpireatConditionXx Completed

func (c ExpireatConditionXx) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type ExpireatKey Completed

func (c ExpireatKey) Timestamp(timestamp int64) ExpireatTimestamp {
	c.cs.s = append(c.cs.s, strconv.FormatInt(timestamp, 10))
	return (ExpireatTimestamp)(c)
}

type ExpireatTimestamp Completed

func (c ExpireatTimestamp) Nx() ExpireatConditionNx {
	c.cs.s = append(c.cs.s, "NX")
	return (ExpireatConditionNx)(c)
}

func (c ExpireatTimestamp) Xx() ExpireatConditionXx {
	c.cs.s = append(c.cs.s, "XX")
	return (ExpireatConditionXx)(c)
}

func (c ExpireatTimestamp) Gt() ExpireatConditionGt {
	c.cs.s = append(c.cs.s, "GT")
	return (ExpireatConditionGt)(c)
}

func (c ExpireatTimestamp) Lt() ExpireatConditionLt {
	c.cs.s = append(c.cs.s, "LT")
	return (ExpireatConditionLt)(c)
}

func (c ExpireatTimestamp) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type Expiretime Completed

func (b Builder) Expiretime() (c Expiretime) {
	c = Expiretime{cs: get(), ks: b.ks, cf: readonly}
	c.cs.s = append(c.cs.s, "EXPIRETIME")
	return c
}

func (c Expiretime) Key(key string) ExpiretimeKey {
	if c.ks&NoSlot == NoSlot {
		c.ks = NoSlot | slot(key)
	} else {
		c.ks = check(c.ks, slot(key))
	}
	c.cs.s = append(c.cs.s, key)
	return (ExpiretimeKey)(c)
}

type ExpiretimeKey Completed

func (c ExpiretimeKey) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

func (c ExpiretimeKey) Cache() Cacheable {
	c.cs.Build()
	return Cacheable(c)
}

type Keys Completed

func (b Builder) Keys() (c Keys) {
	c = Keys{cs: get(), ks: b.ks, cf: readonly}
	c.cs.s = append(c.cs.s, "KEYS")
	return c
}

func (c Keys) Pattern(pattern string) KeysPattern {
	c.cs.s = append(c.cs.s, pattern)
	return (KeysPattern)(c)
}

type KeysPattern Completed

func (c KeysPattern) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type Migrate Completed

func (b Builder) Migrate() (c Migrate) {
	c = Migrate{cs: get(), ks: b.ks, cf: blockTag}
	c.cs.s = append(c.cs.s, "MIGRATE")
	return c
}

func (c Migrate) Host(host string) MigrateHost {
	c.cs.s = append(c.cs.s, host)
	return (MigrateHost)(c)
}

type MigrateAuthAuth Completed

func (c MigrateAuthAuth) Auth2(username string, password string) MigrateAuthAuth2 {
	c.cs.s = append(c.cs.s, "AUTH2", username, password)
	return (MigrateAuthAuth2)(c)
}

func (c MigrateAuthAuth) Keys(key ...string) MigrateKeys {
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
	c.cs.s = append(c.cs.s, "KEYS")
	c.cs.s = append(c.cs.s, key...)
	return (MigrateKeys)(c)
}

func (c MigrateAuthAuth) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type MigrateAuthAuth2 Completed

func (c MigrateAuthAuth2) Keys(key ...string) MigrateKeys {
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
	c.cs.s = append(c.cs.s, "KEYS")
	c.cs.s = append(c.cs.s, key...)
	return (MigrateKeys)(c)
}

func (c MigrateAuthAuth2) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type MigrateCopy Completed

func (c MigrateCopy) Replace() MigrateReplace {
	c.cs.s = append(c.cs.s, "REPLACE")
	return (MigrateReplace)(c)
}

func (c MigrateCopy) Auth(password string) MigrateAuthAuth {
	c.cs.s = append(c.cs.s, "AUTH", password)
	return (MigrateAuthAuth)(c)
}

func (c MigrateCopy) Auth2(username string, password string) MigrateAuthAuth2 {
	c.cs.s = append(c.cs.s, "AUTH2", username, password)
	return (MigrateAuthAuth2)(c)
}

func (c MigrateCopy) Keys(key ...string) MigrateKeys {
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
	c.cs.s = append(c.cs.s, "KEYS")
	c.cs.s = append(c.cs.s, key...)
	return (MigrateKeys)(c)
}

func (c MigrateCopy) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type MigrateDestinationDb Completed

func (c MigrateDestinationDb) Timeout(timeout int64) MigrateTimeout {
	c.cs.s = append(c.cs.s, strconv.FormatInt(timeout, 10))
	return (MigrateTimeout)(c)
}

type MigrateHost Completed

func (c MigrateHost) Port(port int64) MigratePort {
	c.cs.s = append(c.cs.s, strconv.FormatInt(port, 10))
	return (MigratePort)(c)
}

type MigrateKey Completed

func (c MigrateKey) DestinationDb(destinationDb int64) MigrateDestinationDb {
	c.cs.s = append(c.cs.s, strconv.FormatInt(destinationDb, 10))
	return (MigrateDestinationDb)(c)
}

type MigrateKeys Completed

func (c MigrateKeys) Keys(key ...string) MigrateKeys {
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
	c.cs.s = append(c.cs.s, "KEYS")
	c.cs.s = append(c.cs.s, key...)
	return c
}

func (c MigrateKeys) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type MigratePort Completed

func (c MigratePort) Key(key string) MigrateKey {
	if c.ks&NoSlot == NoSlot {
		c.ks = NoSlot | slot(key)
	} else {
		c.ks = check(c.ks, slot(key))
	}
	c.cs.s = append(c.cs.s, key)
	return (MigrateKey)(c)
}

type MigrateReplace Completed

func (c MigrateReplace) Auth(password string) MigrateAuthAuth {
	c.cs.s = append(c.cs.s, "AUTH", password)
	return (MigrateAuthAuth)(c)
}

func (c MigrateReplace) Auth2(username string, password string) MigrateAuthAuth2 {
	c.cs.s = append(c.cs.s, "AUTH2", username, password)
	return (MigrateAuthAuth2)(c)
}

func (c MigrateReplace) Keys(key ...string) MigrateKeys {
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
	c.cs.s = append(c.cs.s, "KEYS")
	c.cs.s = append(c.cs.s, key...)
	return (MigrateKeys)(c)
}

func (c MigrateReplace) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type MigrateTimeout Completed

func (c MigrateTimeout) Copy() MigrateCopy {
	c.cs.s = append(c.cs.s, "COPY")
	return (MigrateCopy)(c)
}

func (c MigrateTimeout) Replace() MigrateReplace {
	c.cs.s = append(c.cs.s, "REPLACE")
	return (MigrateReplace)(c)
}

func (c MigrateTimeout) Auth(password string) MigrateAuthAuth {
	c.cs.s = append(c.cs.s, "AUTH", password)
	return (MigrateAuthAuth)(c)
}

func (c MigrateTimeout) Auth2(username string, password string) MigrateAuthAuth2 {
	c.cs.s = append(c.cs.s, "AUTH2", username, password)
	return (MigrateAuthAuth2)(c)
}

func (c MigrateTimeout) Keys(key ...string) MigrateKeys {
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
	c.cs.s = append(c.cs.s, "KEYS")
	c.cs.s = append(c.cs.s, key...)
	return (MigrateKeys)(c)
}

func (c MigrateTimeout) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type Move Completed

func (b Builder) Move() (c Move) {
	c = Move{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "MOVE")
	return c
}

func (c Move) Key(key string) MoveKey {
	if c.ks&NoSlot == NoSlot {
		c.ks = NoSlot | slot(key)
	} else {
		c.ks = check(c.ks, slot(key))
	}
	c.cs.s = append(c.cs.s, key)
	return (MoveKey)(c)
}

type MoveDb Completed

func (c MoveDb) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type MoveKey Completed

func (c MoveKey) Db(db int64) MoveDb {
	c.cs.s = append(c.cs.s, strconv.FormatInt(db, 10))
	return (MoveDb)(c)
}

type ObjectEncoding Completed

func (b Builder) ObjectEncoding() (c ObjectEncoding) {
	c = ObjectEncoding{cs: get(), ks: b.ks, cf: readonly}
	c.cs.s = append(c.cs.s, "OBJECT", "ENCODING")
	return c
}

func (c ObjectEncoding) Key(key string) ObjectEncodingKey {
	if c.ks&NoSlot == NoSlot {
		c.ks = NoSlot | slot(key)
	} else {
		c.ks = check(c.ks, slot(key))
	}
	c.cs.s = append(c.cs.s, key)
	return (ObjectEncodingKey)(c)
}

type ObjectEncodingKey Completed

func (c ObjectEncodingKey) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type ObjectFreq Completed

func (b Builder) ObjectFreq() (c ObjectFreq) {
	c = ObjectFreq{cs: get(), ks: b.ks, cf: readonly}
	c.cs.s = append(c.cs.s, "OBJECT", "FREQ")
	return c
}

func (c ObjectFreq) Key(key string) ObjectFreqKey {
	if c.ks&NoSlot == NoSlot {
		c.ks = NoSlot | slot(key)
	} else {
		c.ks = check(c.ks, slot(key))
	}
	c.cs.s = append(c.cs.s, key)
	return (ObjectFreqKey)(c)
}

type ObjectFreqKey Completed

func (c ObjectFreqKey) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type ObjectHelp Completed

func (b Builder) ObjectHelp() (c ObjectHelp) {
	c = ObjectHelp{cs: get(), ks: b.ks, cf: readonly}
	c.cs.s = append(c.cs.s, "OBJECT", "HELP")
	return c
}

func (c ObjectHelp) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type ObjectIdletime Completed

func (b Builder) ObjectIdletime() (c ObjectIdletime) {
	c = ObjectIdletime{cs: get(), ks: b.ks, cf: readonly}
	c.cs.s = append(c.cs.s, "OBJECT", "IDLETIME")
	return c
}

func (c ObjectIdletime) Key(key string) ObjectIdletimeKey {
	if c.ks&NoSlot == NoSlot {
		c.ks = NoSlot | slot(key)
	} else {
		c.ks = check(c.ks, slot(key))
	}
	c.cs.s = append(c.cs.s, key)
	return (ObjectIdletimeKey)(c)
}

type ObjectIdletimeKey Completed

func (c ObjectIdletimeKey) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type ObjectRefcount Completed

func (b Builder) ObjectRefcount() (c ObjectRefcount) {
	c = ObjectRefcount{cs: get(), ks: b.ks, cf: readonly}
	c.cs.s = append(c.cs.s, "OBJECT", "REFCOUNT")
	return c
}

func (c ObjectRefcount) Key(key string) ObjectRefcountKey {
	if c.ks&NoSlot == NoSlot {
		c.ks = NoSlot | slot(key)
	} else {
		c.ks = check(c.ks, slot(key))
	}
	c.cs.s = append(c.cs.s, key)
	return (ObjectRefcountKey)(c)
}

type ObjectRefcountKey Completed

func (c ObjectRefcountKey) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type Persist Completed

func (b Builder) Persist() (c Persist) {
	c = Persist{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "PERSIST")
	return c
}

func (c Persist) Key(key string) PersistKey {
	if c.ks&NoSlot == NoSlot {
		c.ks = NoSlot | slot(key)
	} else {
		c.ks = check(c.ks, slot(key))
	}
	c.cs.s = append(c.cs.s, key)
	return (PersistKey)(c)
}

type PersistKey Completed

func (c PersistKey) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type Pexpire Completed

func (b Builder) Pexpire() (c Pexpire) {
	c = Pexpire{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "PEXPIRE")
	return c
}

func (c Pexpire) Key(key string) PexpireKey {
	if c.ks&NoSlot == NoSlot {
		c.ks = NoSlot | slot(key)
	} else {
		c.ks = check(c.ks, slot(key))
	}
	c.cs.s = append(c.cs.s, key)
	return (PexpireKey)(c)
}

type PexpireConditionGt Completed

func (c PexpireConditionGt) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type PexpireConditionLt Completed

func (c PexpireConditionLt) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type PexpireConditionNx Completed

func (c PexpireConditionNx) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type PexpireConditionXx Completed

func (c PexpireConditionXx) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type PexpireKey Completed

func (c PexpireKey) Milliseconds(milliseconds int64) PexpireMilliseconds {
	c.cs.s = append(c.cs.s, strconv.FormatInt(milliseconds, 10))
	return (PexpireMilliseconds)(c)
}

type PexpireMilliseconds Completed

func (c PexpireMilliseconds) Nx() PexpireConditionNx {
	c.cs.s = append(c.cs.s, "NX")
	return (PexpireConditionNx)(c)
}

func (c PexpireMilliseconds) Xx() PexpireConditionXx {
	c.cs.s = append(c.cs.s, "XX")
	return (PexpireConditionXx)(c)
}

func (c PexpireMilliseconds) Gt() PexpireConditionGt {
	c.cs.s = append(c.cs.s, "GT")
	return (PexpireConditionGt)(c)
}

func (c PexpireMilliseconds) Lt() PexpireConditionLt {
	c.cs.s = append(c.cs.s, "LT")
	return (PexpireConditionLt)(c)
}

func (c PexpireMilliseconds) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type Pexpireat Completed

func (b Builder) Pexpireat() (c Pexpireat) {
	c = Pexpireat{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "PEXPIREAT")
	return c
}

func (c Pexpireat) Key(key string) PexpireatKey {
	if c.ks&NoSlot == NoSlot {
		c.ks = NoSlot | slot(key)
	} else {
		c.ks = check(c.ks, slot(key))
	}
	c.cs.s = append(c.cs.s, key)
	return (PexpireatKey)(c)
}

type PexpireatConditionGt Completed

func (c PexpireatConditionGt) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type PexpireatConditionLt Completed

func (c PexpireatConditionLt) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type PexpireatConditionNx Completed

func (c PexpireatConditionNx) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type PexpireatConditionXx Completed

func (c PexpireatConditionXx) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type PexpireatKey Completed

func (c PexpireatKey) MillisecondsTimestamp(millisecondsTimestamp int64) PexpireatMillisecondsTimestamp {
	c.cs.s = append(c.cs.s, strconv.FormatInt(millisecondsTimestamp, 10))
	return (PexpireatMillisecondsTimestamp)(c)
}

type PexpireatMillisecondsTimestamp Completed

func (c PexpireatMillisecondsTimestamp) Nx() PexpireatConditionNx {
	c.cs.s = append(c.cs.s, "NX")
	return (PexpireatConditionNx)(c)
}

func (c PexpireatMillisecondsTimestamp) Xx() PexpireatConditionXx {
	c.cs.s = append(c.cs.s, "XX")
	return (PexpireatConditionXx)(c)
}

func (c PexpireatMillisecondsTimestamp) Gt() PexpireatConditionGt {
	c.cs.s = append(c.cs.s, "GT")
	return (PexpireatConditionGt)(c)
}

func (c PexpireatMillisecondsTimestamp) Lt() PexpireatConditionLt {
	c.cs.s = append(c.cs.s, "LT")
	return (PexpireatConditionLt)(c)
}

func (c PexpireatMillisecondsTimestamp) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type Pexpiretime Completed

func (b Builder) Pexpiretime() (c Pexpiretime) {
	c = Pexpiretime{cs: get(), ks: b.ks, cf: readonly}
	c.cs.s = append(c.cs.s, "PEXPIRETIME")
	return c
}

func (c Pexpiretime) Key(key string) PexpiretimeKey {
	if c.ks&NoSlot == NoSlot {
		c.ks = NoSlot | slot(key)
	} else {
		c.ks = check(c.ks, slot(key))
	}
	c.cs.s = append(c.cs.s, key)
	return (PexpiretimeKey)(c)
}

type PexpiretimeKey Completed

func (c PexpiretimeKey) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

func (c PexpiretimeKey) Cache() Cacheable {
	c.cs.Build()
	return Cacheable(c)
}

type Pttl Completed

func (b Builder) Pttl() (c Pttl) {
	c = Pttl{cs: get(), ks: b.ks, cf: readonly}
	c.cs.s = append(c.cs.s, "PTTL")
	return c
}

func (c Pttl) Key(key string) PttlKey {
	if c.ks&NoSlot == NoSlot {
		c.ks = NoSlot | slot(key)
	} else {
		c.ks = check(c.ks, slot(key))
	}
	c.cs.s = append(c.cs.s, key)
	return (PttlKey)(c)
}

type PttlKey Completed

func (c PttlKey) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

func (c PttlKey) Cache() Cacheable {
	c.cs.Build()
	return Cacheable(c)
}

type Randomkey Completed

func (b Builder) Randomkey() (c Randomkey) {
	c = Randomkey{cs: get(), ks: b.ks, cf: readonly}
	c.cs.s = append(c.cs.s, "RANDOMKEY")
	return c
}

func (c Randomkey) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type Rename Completed

func (b Builder) Rename() (c Rename) {
	c = Rename{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "RENAME")
	return c
}

func (c Rename) Key(key string) RenameKey {
	if c.ks&NoSlot == NoSlot {
		c.ks = NoSlot | slot(key)
	} else {
		c.ks = check(c.ks, slot(key))
	}
	c.cs.s = append(c.cs.s, key)
	return (RenameKey)(c)
}

type RenameKey Completed

func (c RenameKey) Newkey(newkey string) RenameNewkey {
	if c.ks&NoSlot == NoSlot {
		c.ks = NoSlot | slot(newkey)
	} else {
		c.ks = check(c.ks, slot(newkey))
	}
	c.cs.s = append(c.cs.s, newkey)
	return (RenameNewkey)(c)
}

type RenameNewkey Completed

func (c RenameNewkey) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type Renamenx Completed

func (b Builder) Renamenx() (c Renamenx) {
	c = Renamenx{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "RENAMENX")
	return c
}

func (c Renamenx) Key(key string) RenamenxKey {
	if c.ks&NoSlot == NoSlot {
		c.ks = NoSlot | slot(key)
	} else {
		c.ks = check(c.ks, slot(key))
	}
	c.cs.s = append(c.cs.s, key)
	return (RenamenxKey)(c)
}

type RenamenxKey Completed

func (c RenamenxKey) Newkey(newkey string) RenamenxNewkey {
	if c.ks&NoSlot == NoSlot {
		c.ks = NoSlot | slot(newkey)
	} else {
		c.ks = check(c.ks, slot(newkey))
	}
	c.cs.s = append(c.cs.s, newkey)
	return (RenamenxNewkey)(c)
}

type RenamenxNewkey Completed

func (c RenamenxNewkey) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type Restore Completed

func (b Builder) Restore() (c Restore) {
	c = Restore{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "RESTORE")
	return c
}

func (c Restore) Key(key string) RestoreKey {
	if c.ks&NoSlot == NoSlot {
		c.ks = NoSlot | slot(key)
	} else {
		c.ks = check(c.ks, slot(key))
	}
	c.cs.s = append(c.cs.s, key)
	return (RestoreKey)(c)
}

type RestoreAbsttl Completed

func (c RestoreAbsttl) Idletime(seconds int64) RestoreIdletime {
	c.cs.s = append(c.cs.s, "IDLETIME", strconv.FormatInt(seconds, 10))
	return (RestoreIdletime)(c)
}

func (c RestoreAbsttl) Freq(frequency int64) RestoreFreq {
	c.cs.s = append(c.cs.s, "FREQ", strconv.FormatInt(frequency, 10))
	return (RestoreFreq)(c)
}

func (c RestoreAbsttl) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type RestoreFreq Completed

func (c RestoreFreq) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type RestoreIdletime Completed

func (c RestoreIdletime) Freq(frequency int64) RestoreFreq {
	c.cs.s = append(c.cs.s, "FREQ", strconv.FormatInt(frequency, 10))
	return (RestoreFreq)(c)
}

func (c RestoreIdletime) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type RestoreKey Completed

func (c RestoreKey) Ttl(ttl int64) RestoreTtl {
	c.cs.s = append(c.cs.s, strconv.FormatInt(ttl, 10))
	return (RestoreTtl)(c)
}

type RestoreReplace Completed

func (c RestoreReplace) Absttl() RestoreAbsttl {
	c.cs.s = append(c.cs.s, "ABSTTL")
	return (RestoreAbsttl)(c)
}

func (c RestoreReplace) Idletime(seconds int64) RestoreIdletime {
	c.cs.s = append(c.cs.s, "IDLETIME", strconv.FormatInt(seconds, 10))
	return (RestoreIdletime)(c)
}

func (c RestoreReplace) Freq(frequency int64) RestoreFreq {
	c.cs.s = append(c.cs.s, "FREQ", strconv.FormatInt(frequency, 10))
	return (RestoreFreq)(c)
}

func (c RestoreReplace) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type RestoreSerializedValue Completed

func (c RestoreSerializedValue) Replace() RestoreReplace {
	c.cs.s = append(c.cs.s, "REPLACE")
	return (RestoreReplace)(c)
}

func (c RestoreSerializedValue) Absttl() RestoreAbsttl {
	c.cs.s = append(c.cs.s, "ABSTTL")
	return (RestoreAbsttl)(c)
}

func (c RestoreSerializedValue) Idletime(seconds int64) RestoreIdletime {
	c.cs.s = append(c.cs.s, "IDLETIME", strconv.FormatInt(seconds, 10))
	return (RestoreIdletime)(c)
}

func (c RestoreSerializedValue) Freq(frequency int64) RestoreFreq {
	c.cs.s = append(c.cs.s, "FREQ", strconv.FormatInt(frequency, 10))
	return (RestoreFreq)(c)
}

func (c RestoreSerializedValue) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type RestoreTtl Completed

func (c RestoreTtl) SerializedValue(serializedValue string) RestoreSerializedValue {
	c.cs.s = append(c.cs.s, serializedValue)
	return (RestoreSerializedValue)(c)
}

type Scan Completed

func (b Builder) Scan() (c Scan) {
	c = Scan{cs: get(), ks: b.ks, cf: readonly}
	c.cs.s = append(c.cs.s, "SCAN")
	return c
}

func (c Scan) Cursor(cursor uint64) ScanCursor {
	c.cs.s = append(c.cs.s, strconv.FormatUint(cursor, 10))
	return (ScanCursor)(c)
}

type ScanCount Completed

func (c ScanCount) Type(typ string) ScanType {
	c.cs.s = append(c.cs.s, "TYPE", typ)
	return (ScanType)(c)
}

func (c ScanCount) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type ScanCursor Completed

func (c ScanCursor) Match(pattern string) ScanMatch {
	c.cs.s = append(c.cs.s, "MATCH", pattern)
	return (ScanMatch)(c)
}

func (c ScanCursor) Count(count int64) ScanCount {
	c.cs.s = append(c.cs.s, "COUNT", strconv.FormatInt(count, 10))
	return (ScanCount)(c)
}

func (c ScanCursor) Type(typ string) ScanType {
	c.cs.s = append(c.cs.s, "TYPE", typ)
	return (ScanType)(c)
}

func (c ScanCursor) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type ScanMatch Completed

func (c ScanMatch) Count(count int64) ScanCount {
	c.cs.s = append(c.cs.s, "COUNT", strconv.FormatInt(count, 10))
	return (ScanCount)(c)
}

func (c ScanMatch) Type(typ string) ScanType {
	c.cs.s = append(c.cs.s, "TYPE", typ)
	return (ScanType)(c)
}

func (c ScanMatch) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type ScanType Completed

func (c ScanType) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type Sort Completed

func (b Builder) Sort() (c Sort) {
	c = Sort{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "SORT")
	return c
}

func (c Sort) Key(key string) SortKey {
	if c.ks&NoSlot == NoSlot {
		c.ks = NoSlot | slot(key)
	} else {
		c.ks = check(c.ks, slot(key))
	}
	c.cs.s = append(c.cs.s, key)
	return (SortKey)(c)
}

type SortBy Completed

func (c SortBy) Limit(offset int64, count int64) SortLimit {
	c.cs.s = append(c.cs.s, "LIMIT", strconv.FormatInt(offset, 10), strconv.FormatInt(count, 10))
	return (SortLimit)(c)
}

func (c SortBy) Get() SortGet {
	return (SortGet)(c)
}

func (c SortBy) Asc() SortOrderAsc {
	c.cs.s = append(c.cs.s, "ASC")
	return (SortOrderAsc)(c)
}

func (c SortBy) Desc() SortOrderDesc {
	c.cs.s = append(c.cs.s, "DESC")
	return (SortOrderDesc)(c)
}

func (c SortBy) Alpha() SortSortingAlpha {
	c.cs.s = append(c.cs.s, "ALPHA")
	return (SortSortingAlpha)(c)
}

func (c SortBy) Store(destination string) SortStore {
	if c.ks&NoSlot == NoSlot {
		c.ks = NoSlot | slot(destination)
	} else {
		c.ks = check(c.ks, slot(destination))
	}
	c.cs.s = append(c.cs.s, "STORE", destination)
	return (SortStore)(c)
}

func (c SortBy) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type SortGet Completed

func (c SortGet) Get(pattern string) SortGet {
	c.cs.s = append(c.cs.s, "GET", pattern)
	return c
}

func (c SortGet) Asc() SortOrderAsc {
	c.cs.s = append(c.cs.s, "ASC")
	return (SortOrderAsc)(c)
}

func (c SortGet) Desc() SortOrderDesc {
	c.cs.s = append(c.cs.s, "DESC")
	return (SortOrderDesc)(c)
}

func (c SortGet) Alpha() SortSortingAlpha {
	c.cs.s = append(c.cs.s, "ALPHA")
	return (SortSortingAlpha)(c)
}

func (c SortGet) Store(destination string) SortStore {
	if c.ks&NoSlot == NoSlot {
		c.ks = NoSlot | slot(destination)
	} else {
		c.ks = check(c.ks, slot(destination))
	}
	c.cs.s = append(c.cs.s, "STORE", destination)
	return (SortStore)(c)
}

func (c SortGet) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type SortKey Completed

func (c SortKey) By(pattern string) SortBy {
	c.cs.s = append(c.cs.s, "BY", pattern)
	return (SortBy)(c)
}

func (c SortKey) Limit(offset int64, count int64) SortLimit {
	c.cs.s = append(c.cs.s, "LIMIT", strconv.FormatInt(offset, 10), strconv.FormatInt(count, 10))
	return (SortLimit)(c)
}

func (c SortKey) Get() SortGet {
	return (SortGet)(c)
}

func (c SortKey) Asc() SortOrderAsc {
	c.cs.s = append(c.cs.s, "ASC")
	return (SortOrderAsc)(c)
}

func (c SortKey) Desc() SortOrderDesc {
	c.cs.s = append(c.cs.s, "DESC")
	return (SortOrderDesc)(c)
}

func (c SortKey) Alpha() SortSortingAlpha {
	c.cs.s = append(c.cs.s, "ALPHA")
	return (SortSortingAlpha)(c)
}

func (c SortKey) Store(destination string) SortStore {
	if c.ks&NoSlot == NoSlot {
		c.ks = NoSlot | slot(destination)
	} else {
		c.ks = check(c.ks, slot(destination))
	}
	c.cs.s = append(c.cs.s, "STORE", destination)
	return (SortStore)(c)
}

func (c SortKey) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type SortLimit Completed

func (c SortLimit) Get() SortGet {
	return (SortGet)(c)
}

func (c SortLimit) Asc() SortOrderAsc {
	c.cs.s = append(c.cs.s, "ASC")
	return (SortOrderAsc)(c)
}

func (c SortLimit) Desc() SortOrderDesc {
	c.cs.s = append(c.cs.s, "DESC")
	return (SortOrderDesc)(c)
}

func (c SortLimit) Alpha() SortSortingAlpha {
	c.cs.s = append(c.cs.s, "ALPHA")
	return (SortSortingAlpha)(c)
}

func (c SortLimit) Store(destination string) SortStore {
	if c.ks&NoSlot == NoSlot {
		c.ks = NoSlot | slot(destination)
	} else {
		c.ks = check(c.ks, slot(destination))
	}
	c.cs.s = append(c.cs.s, "STORE", destination)
	return (SortStore)(c)
}

func (c SortLimit) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type SortOrderAsc Completed

func (c SortOrderAsc) Alpha() SortSortingAlpha {
	c.cs.s = append(c.cs.s, "ALPHA")
	return (SortSortingAlpha)(c)
}

func (c SortOrderAsc) Store(destination string) SortStore {
	if c.ks&NoSlot == NoSlot {
		c.ks = NoSlot | slot(destination)
	} else {
		c.ks = check(c.ks, slot(destination))
	}
	c.cs.s = append(c.cs.s, "STORE", destination)
	return (SortStore)(c)
}

func (c SortOrderAsc) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type SortOrderDesc Completed

func (c SortOrderDesc) Alpha() SortSortingAlpha {
	c.cs.s = append(c.cs.s, "ALPHA")
	return (SortSortingAlpha)(c)
}

func (c SortOrderDesc) Store(destination string) SortStore {
	if c.ks&NoSlot == NoSlot {
		c.ks = NoSlot | slot(destination)
	} else {
		c.ks = check(c.ks, slot(destination))
	}
	c.cs.s = append(c.cs.s, "STORE", destination)
	return (SortStore)(c)
}

func (c SortOrderDesc) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type SortRo Completed

func (b Builder) SortRo() (c SortRo) {
	c = SortRo{cs: get(), ks: b.ks, cf: readonly}
	c.cs.s = append(c.cs.s, "SORT_RO")
	return c
}

func (c SortRo) Key(key string) SortRoKey {
	if c.ks&NoSlot == NoSlot {
		c.ks = NoSlot | slot(key)
	} else {
		c.ks = check(c.ks, slot(key))
	}
	c.cs.s = append(c.cs.s, key)
	return (SortRoKey)(c)
}

type SortRoBy Completed

func (c SortRoBy) Limit(offset int64, count int64) SortRoLimit {
	c.cs.s = append(c.cs.s, "LIMIT", strconv.FormatInt(offset, 10), strconv.FormatInt(count, 10))
	return (SortRoLimit)(c)
}

func (c SortRoBy) Get() SortRoGet {
	return (SortRoGet)(c)
}

func (c SortRoBy) Asc() SortRoOrderAsc {
	c.cs.s = append(c.cs.s, "ASC")
	return (SortRoOrderAsc)(c)
}

func (c SortRoBy) Desc() SortRoOrderDesc {
	c.cs.s = append(c.cs.s, "DESC")
	return (SortRoOrderDesc)(c)
}

func (c SortRoBy) Alpha() SortRoSortingAlpha {
	c.cs.s = append(c.cs.s, "ALPHA")
	return (SortRoSortingAlpha)(c)
}

func (c SortRoBy) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

func (c SortRoBy) Cache() Cacheable {
	c.cs.Build()
	return Cacheable(c)
}

type SortRoGet Completed

func (c SortRoGet) Get(pattern string) SortRoGet {
	c.cs.s = append(c.cs.s, "GET", pattern)
	return c
}

func (c SortRoGet) Asc() SortRoOrderAsc {
	c.cs.s = append(c.cs.s, "ASC")
	return (SortRoOrderAsc)(c)
}

func (c SortRoGet) Desc() SortRoOrderDesc {
	c.cs.s = append(c.cs.s, "DESC")
	return (SortRoOrderDesc)(c)
}

func (c SortRoGet) Alpha() SortRoSortingAlpha {
	c.cs.s = append(c.cs.s, "ALPHA")
	return (SortRoSortingAlpha)(c)
}

func (c SortRoGet) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

func (c SortRoGet) Cache() Cacheable {
	c.cs.Build()
	return Cacheable(c)
}

type SortRoKey Completed

func (c SortRoKey) By(pattern string) SortRoBy {
	c.cs.s = append(c.cs.s, "BY", pattern)
	return (SortRoBy)(c)
}

func (c SortRoKey) Limit(offset int64, count int64) SortRoLimit {
	c.cs.s = append(c.cs.s, "LIMIT", strconv.FormatInt(offset, 10), strconv.FormatInt(count, 10))
	return (SortRoLimit)(c)
}

func (c SortRoKey) Get() SortRoGet {
	return (SortRoGet)(c)
}

func (c SortRoKey) Asc() SortRoOrderAsc {
	c.cs.s = append(c.cs.s, "ASC")
	return (SortRoOrderAsc)(c)
}

func (c SortRoKey) Desc() SortRoOrderDesc {
	c.cs.s = append(c.cs.s, "DESC")
	return (SortRoOrderDesc)(c)
}

func (c SortRoKey) Alpha() SortRoSortingAlpha {
	c.cs.s = append(c.cs.s, "ALPHA")
	return (SortRoSortingAlpha)(c)
}

func (c SortRoKey) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

func (c SortRoKey) Cache() Cacheable {
	c.cs.Build()
	return Cacheable(c)
}

type SortRoLimit Completed

func (c SortRoLimit) Get() SortRoGet {
	return (SortRoGet)(c)
}

func (c SortRoLimit) Asc() SortRoOrderAsc {
	c.cs.s = append(c.cs.s, "ASC")
	return (SortRoOrderAsc)(c)
}

func (c SortRoLimit) Desc() SortRoOrderDesc {
	c.cs.s = append(c.cs.s, "DESC")
	return (SortRoOrderDesc)(c)
}

func (c SortRoLimit) Alpha() SortRoSortingAlpha {
	c.cs.s = append(c.cs.s, "ALPHA")
	return (SortRoSortingAlpha)(c)
}

func (c SortRoLimit) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

func (c SortRoLimit) Cache() Cacheable {
	c.cs.Build()
	return Cacheable(c)
}

type SortRoOrderAsc Completed

func (c SortRoOrderAsc) Alpha() SortRoSortingAlpha {
	c.cs.s = append(c.cs.s, "ALPHA")
	return (SortRoSortingAlpha)(c)
}

func (c SortRoOrderAsc) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

func (c SortRoOrderAsc) Cache() Cacheable {
	c.cs.Build()
	return Cacheable(c)
}

type SortRoOrderDesc Completed

func (c SortRoOrderDesc) Alpha() SortRoSortingAlpha {
	c.cs.s = append(c.cs.s, "ALPHA")
	return (SortRoSortingAlpha)(c)
}

func (c SortRoOrderDesc) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

func (c SortRoOrderDesc) Cache() Cacheable {
	c.cs.Build()
	return Cacheable(c)
}

type SortRoSortingAlpha Completed

func (c SortRoSortingAlpha) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

func (c SortRoSortingAlpha) Cache() Cacheable {
	c.cs.Build()
	return Cacheable(c)
}

type SortSortingAlpha Completed

func (c SortSortingAlpha) Store(destination string) SortStore {
	if c.ks&NoSlot == NoSlot {
		c.ks = NoSlot | slot(destination)
	} else {
		c.ks = check(c.ks, slot(destination))
	}
	c.cs.s = append(c.cs.s, "STORE", destination)
	return (SortStore)(c)
}

func (c SortSortingAlpha) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type SortStore Completed

func (c SortStore) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type Touch Completed

func (b Builder) Touch() (c Touch) {
	c = Touch{cs: get(), ks: b.ks, cf: readonly}
	c.cs.s = append(c.cs.s, "TOUCH")
	return c
}

func (c Touch) Key(key ...string) TouchKey {
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
	return (TouchKey)(c)
}

type TouchKey Completed

func (c TouchKey) Key(key ...string) TouchKey {
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

func (c TouchKey) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type Ttl Completed

func (b Builder) Ttl() (c Ttl) {
	c = Ttl{cs: get(), ks: b.ks, cf: readonly}
	c.cs.s = append(c.cs.s, "TTL")
	return c
}

func (c Ttl) Key(key string) TtlKey {
	if c.ks&NoSlot == NoSlot {
		c.ks = NoSlot | slot(key)
	} else {
		c.ks = check(c.ks, slot(key))
	}
	c.cs.s = append(c.cs.s, key)
	return (TtlKey)(c)
}

type TtlKey Completed

func (c TtlKey) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

func (c TtlKey) Cache() Cacheable {
	c.cs.Build()
	return Cacheable(c)
}

type Type Completed

func (b Builder) Type() (c Type) {
	c = Type{cs: get(), ks: b.ks, cf: readonly}
	c.cs.s = append(c.cs.s, "TYPE")
	return c
}

func (c Type) Key(key string) TypeKey {
	if c.ks&NoSlot == NoSlot {
		c.ks = NoSlot | slot(key)
	} else {
		c.ks = check(c.ks, slot(key))
	}
	c.cs.s = append(c.cs.s, key)
	return (TypeKey)(c)
}

type TypeKey Completed

func (c TypeKey) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

func (c TypeKey) Cache() Cacheable {
	c.cs.Build()
	return Cacheable(c)
}

type Unlink Completed

func (b Builder) Unlink() (c Unlink) {
	c = Unlink{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "UNLINK")
	return c
}

func (c Unlink) Key(key ...string) UnlinkKey {
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
	return (UnlinkKey)(c)
}

type UnlinkKey Completed

func (c UnlinkKey) Key(key ...string) UnlinkKey {
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

func (c UnlinkKey) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type Wait Completed

func (b Builder) Wait() (c Wait) {
	c = Wait{cs: get(), ks: b.ks, cf: blockTag}
	c.cs.s = append(c.cs.s, "WAIT")
	return c
}

func (c Wait) Numreplicas(numreplicas int64) WaitNumreplicas {
	c.cs.s = append(c.cs.s, strconv.FormatInt(numreplicas, 10))
	return (WaitNumreplicas)(c)
}

type WaitNumreplicas Completed

func (c WaitNumreplicas) Timeout(timeout int64) WaitTimeout {
	c.cs.s = append(c.cs.s, strconv.FormatInt(timeout, 10))
	return (WaitTimeout)(c)
}

type WaitTimeout Completed

func (c WaitTimeout) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type Waitaof Completed

func (b Builder) Waitaof() (c Waitaof) {
	c = Waitaof{cs: get(), ks: b.ks, cf: blockTag}
	c.cs.s = append(c.cs.s, "WAITAOF")
	return c
}

func (c Waitaof) Numlocal(numlocal int64) WaitaofNumlocal {
	c.cs.s = append(c.cs.s, strconv.FormatInt(numlocal, 10))
	return (WaitaofNumlocal)(c)
}

type WaitaofNumlocal Completed

func (c WaitaofNumlocal) Numreplicas(numreplicas int64) WaitaofNumreplicas {
	c.cs.s = append(c.cs.s, strconv.FormatInt(numreplicas, 10))
	return (WaitaofNumreplicas)(c)
}

type WaitaofNumreplicas Completed

func (c WaitaofNumreplicas) Timeout(timeout int64) WaitaofTimeout {
	c.cs.s = append(c.cs.s, strconv.FormatInt(timeout, 10))
	return (WaitaofTimeout)(c)
}

type WaitaofTimeout Completed

func (c WaitaofTimeout) Build() Completed {
	c.cs.Build()
	return Completed(c)
}
