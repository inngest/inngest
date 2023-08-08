// Code generated DO NOT EDIT

package cmds

import "strconv"

type Eval Completed

func (b Builder) Eval() (c Eval) {
	c = Eval{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "EVAL")
	return c
}

func (c Eval) Script(script string) EvalScript {
	c.cs.s = append(c.cs.s, script)
	return (EvalScript)(c)
}

type EvalArg Completed

func (c EvalArg) Arg(arg ...string) EvalArg {
	c.cs.s = append(c.cs.s, arg...)
	return c
}

func (c EvalArg) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type EvalKey Completed

func (c EvalKey) Key(key ...string) EvalKey {
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

func (c EvalKey) Arg(arg ...string) EvalArg {
	c.cs.s = append(c.cs.s, arg...)
	return (EvalArg)(c)
}

func (c EvalKey) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type EvalNumkeys Completed

func (c EvalNumkeys) Key(key ...string) EvalKey {
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
	return (EvalKey)(c)
}

func (c EvalNumkeys) Arg(arg ...string) EvalArg {
	c.cs.s = append(c.cs.s, arg...)
	return (EvalArg)(c)
}

func (c EvalNumkeys) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type EvalRo Completed

func (b Builder) EvalRo() (c EvalRo) {
	c = EvalRo{cs: get(), ks: b.ks, cf: scrRoTag}
	c.cs.s = append(c.cs.s, "EVAL_RO")
	return c
}

func (c EvalRo) Script(script string) EvalRoScript {
	c.cs.s = append(c.cs.s, script)
	return (EvalRoScript)(c)
}

type EvalRoArg Completed

func (c EvalRoArg) Arg(arg ...string) EvalRoArg {
	c.cs.s = append(c.cs.s, arg...)
	return c
}

func (c EvalRoArg) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

func (c EvalRoArg) Cache() Cacheable {
	c.cs.Build()
	return Cacheable(c)
}

type EvalRoKey Completed

func (c EvalRoKey) Key(key ...string) EvalRoKey {
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

func (c EvalRoKey) Arg(arg ...string) EvalRoArg {
	c.cs.s = append(c.cs.s, arg...)
	return (EvalRoArg)(c)
}

func (c EvalRoKey) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

func (c EvalRoKey) Cache() Cacheable {
	c.cs.Build()
	return Cacheable(c)
}

type EvalRoNumkeys Completed

func (c EvalRoNumkeys) Key(key ...string) EvalRoKey {
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
	return (EvalRoKey)(c)
}

func (c EvalRoNumkeys) Arg(arg ...string) EvalRoArg {
	c.cs.s = append(c.cs.s, arg...)
	return (EvalRoArg)(c)
}

func (c EvalRoNumkeys) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

func (c EvalRoNumkeys) Cache() Cacheable {
	c.cs.Build()
	return Cacheable(c)
}

type EvalRoScript Completed

func (c EvalRoScript) Numkeys(numkeys int64) EvalRoNumkeys {
	c.cs.s = append(c.cs.s, strconv.FormatInt(numkeys, 10))
	return (EvalRoNumkeys)(c)
}

type EvalScript Completed

func (c EvalScript) Numkeys(numkeys int64) EvalNumkeys {
	c.cs.s = append(c.cs.s, strconv.FormatInt(numkeys, 10))
	return (EvalNumkeys)(c)
}

type Evalsha Completed

func (b Builder) Evalsha() (c Evalsha) {
	c = Evalsha{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "EVALSHA")
	return c
}

func (c Evalsha) Sha1(sha1 string) EvalshaSha1 {
	c.cs.s = append(c.cs.s, sha1)
	return (EvalshaSha1)(c)
}

type EvalshaArg Completed

func (c EvalshaArg) Arg(arg ...string) EvalshaArg {
	c.cs.s = append(c.cs.s, arg...)
	return c
}

func (c EvalshaArg) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type EvalshaKey Completed

func (c EvalshaKey) Key(key ...string) EvalshaKey {
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

func (c EvalshaKey) Arg(arg ...string) EvalshaArg {
	c.cs.s = append(c.cs.s, arg...)
	return (EvalshaArg)(c)
}

func (c EvalshaKey) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type EvalshaNumkeys Completed

func (c EvalshaNumkeys) Key(key ...string) EvalshaKey {
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
	return (EvalshaKey)(c)
}

func (c EvalshaNumkeys) Arg(arg ...string) EvalshaArg {
	c.cs.s = append(c.cs.s, arg...)
	return (EvalshaArg)(c)
}

func (c EvalshaNumkeys) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type EvalshaRo Completed

func (b Builder) EvalshaRo() (c EvalshaRo) {
	c = EvalshaRo{cs: get(), ks: b.ks, cf: scrRoTag}
	c.cs.s = append(c.cs.s, "EVALSHA_RO")
	return c
}

func (c EvalshaRo) Sha1(sha1 string) EvalshaRoSha1 {
	c.cs.s = append(c.cs.s, sha1)
	return (EvalshaRoSha1)(c)
}

type EvalshaRoArg Completed

func (c EvalshaRoArg) Arg(arg ...string) EvalshaRoArg {
	c.cs.s = append(c.cs.s, arg...)
	return c
}

func (c EvalshaRoArg) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

func (c EvalshaRoArg) Cache() Cacheable {
	c.cs.Build()
	return Cacheable(c)
}

type EvalshaRoKey Completed

func (c EvalshaRoKey) Key(key ...string) EvalshaRoKey {
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

func (c EvalshaRoKey) Arg(arg ...string) EvalshaRoArg {
	c.cs.s = append(c.cs.s, arg...)
	return (EvalshaRoArg)(c)
}

func (c EvalshaRoKey) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

func (c EvalshaRoKey) Cache() Cacheable {
	c.cs.Build()
	return Cacheable(c)
}

type EvalshaRoNumkeys Completed

func (c EvalshaRoNumkeys) Key(key ...string) EvalshaRoKey {
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
	return (EvalshaRoKey)(c)
}

func (c EvalshaRoNumkeys) Arg(arg ...string) EvalshaRoArg {
	c.cs.s = append(c.cs.s, arg...)
	return (EvalshaRoArg)(c)
}

func (c EvalshaRoNumkeys) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

func (c EvalshaRoNumkeys) Cache() Cacheable {
	c.cs.Build()
	return Cacheable(c)
}

type EvalshaRoSha1 Completed

func (c EvalshaRoSha1) Numkeys(numkeys int64) EvalshaRoNumkeys {
	c.cs.s = append(c.cs.s, strconv.FormatInt(numkeys, 10))
	return (EvalshaRoNumkeys)(c)
}

type EvalshaSha1 Completed

func (c EvalshaSha1) Numkeys(numkeys int64) EvalshaNumkeys {
	c.cs.s = append(c.cs.s, strconv.FormatInt(numkeys, 10))
	return (EvalshaNumkeys)(c)
}

type Fcall Completed

func (b Builder) Fcall() (c Fcall) {
	c = Fcall{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "FCALL")
	return c
}

func (c Fcall) Function(function string) FcallFunction {
	c.cs.s = append(c.cs.s, function)
	return (FcallFunction)(c)
}

type FcallArg Completed

func (c FcallArg) Arg(arg ...string) FcallArg {
	c.cs.s = append(c.cs.s, arg...)
	return c
}

func (c FcallArg) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type FcallFunction Completed

func (c FcallFunction) Numkeys(numkeys int64) FcallNumkeys {
	c.cs.s = append(c.cs.s, strconv.FormatInt(numkeys, 10))
	return (FcallNumkeys)(c)
}

type FcallKey Completed

func (c FcallKey) Key(key ...string) FcallKey {
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

func (c FcallKey) Arg(arg ...string) FcallArg {
	c.cs.s = append(c.cs.s, arg...)
	return (FcallArg)(c)
}

func (c FcallKey) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type FcallNumkeys Completed

func (c FcallNumkeys) Key(key ...string) FcallKey {
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
	return (FcallKey)(c)
}

func (c FcallNumkeys) Arg(arg ...string) FcallArg {
	c.cs.s = append(c.cs.s, arg...)
	return (FcallArg)(c)
}

func (c FcallNumkeys) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type FcallRo Completed

func (b Builder) FcallRo() (c FcallRo) {
	c = FcallRo{cs: get(), ks: b.ks, cf: scrRoTag}
	c.cs.s = append(c.cs.s, "FCALL_RO")
	return c
}

func (c FcallRo) Function(function string) FcallRoFunction {
	c.cs.s = append(c.cs.s, function)
	return (FcallRoFunction)(c)
}

type FcallRoArg Completed

func (c FcallRoArg) Arg(arg ...string) FcallRoArg {
	c.cs.s = append(c.cs.s, arg...)
	return c
}

func (c FcallRoArg) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

func (c FcallRoArg) Cache() Cacheable {
	c.cs.Build()
	return Cacheable(c)
}

type FcallRoFunction Completed

func (c FcallRoFunction) Numkeys(numkeys int64) FcallRoNumkeys {
	c.cs.s = append(c.cs.s, strconv.FormatInt(numkeys, 10))
	return (FcallRoNumkeys)(c)
}

type FcallRoKey Completed

func (c FcallRoKey) Key(key ...string) FcallRoKey {
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

func (c FcallRoKey) Arg(arg ...string) FcallRoArg {
	c.cs.s = append(c.cs.s, arg...)
	return (FcallRoArg)(c)
}

func (c FcallRoKey) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

func (c FcallRoKey) Cache() Cacheable {
	c.cs.Build()
	return Cacheable(c)
}

type FcallRoNumkeys Completed

func (c FcallRoNumkeys) Key(key ...string) FcallRoKey {
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
	return (FcallRoKey)(c)
}

func (c FcallRoNumkeys) Arg(arg ...string) FcallRoArg {
	c.cs.s = append(c.cs.s, arg...)
	return (FcallRoArg)(c)
}

func (c FcallRoNumkeys) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

func (c FcallRoNumkeys) Cache() Cacheable {
	c.cs.Build()
	return Cacheable(c)
}

type FunctionDelete Completed

func (b Builder) FunctionDelete() (c FunctionDelete) {
	c = FunctionDelete{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "FUNCTION", "DELETE")
	return c
}

func (c FunctionDelete) LibraryName(libraryName string) FunctionDeleteLibraryName {
	c.cs.s = append(c.cs.s, libraryName)
	return (FunctionDeleteLibraryName)(c)
}

type FunctionDeleteLibraryName Completed

func (c FunctionDeleteLibraryName) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type FunctionDump Completed

func (b Builder) FunctionDump() (c FunctionDump) {
	c = FunctionDump{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "FUNCTION", "DUMP")
	return c
}

func (c FunctionDump) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type FunctionFlush Completed

func (b Builder) FunctionFlush() (c FunctionFlush) {
	c = FunctionFlush{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "FUNCTION", "FLUSH")
	return c
}

func (c FunctionFlush) Async() FunctionFlushAsync {
	c.cs.s = append(c.cs.s, "ASYNC")
	return (FunctionFlushAsync)(c)
}

func (c FunctionFlush) Sync() FunctionFlushAsyncSync {
	c.cs.s = append(c.cs.s, "SYNC")
	return (FunctionFlushAsyncSync)(c)
}

func (c FunctionFlush) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type FunctionFlushAsync Completed

func (c FunctionFlushAsync) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type FunctionFlushAsyncSync Completed

func (c FunctionFlushAsyncSync) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type FunctionHelp Completed

func (b Builder) FunctionHelp() (c FunctionHelp) {
	c = FunctionHelp{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "FUNCTION", "HELP")
	return c
}

func (c FunctionHelp) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type FunctionKill Completed

func (b Builder) FunctionKill() (c FunctionKill) {
	c = FunctionKill{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "FUNCTION", "KILL")
	return c
}

func (c FunctionKill) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type FunctionList Completed

func (b Builder) FunctionList() (c FunctionList) {
	c = FunctionList{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "FUNCTION", "LIST")
	return c
}

func (c FunctionList) Libraryname(libraryNamePattern string) FunctionListLibraryname {
	c.cs.s = append(c.cs.s, "LIBRARYNAME", libraryNamePattern)
	return (FunctionListLibraryname)(c)
}

func (c FunctionList) Withcode() FunctionListWithcode {
	c.cs.s = append(c.cs.s, "WITHCODE")
	return (FunctionListWithcode)(c)
}

func (c FunctionList) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type FunctionListLibraryname Completed

func (c FunctionListLibraryname) Withcode() FunctionListWithcode {
	c.cs.s = append(c.cs.s, "WITHCODE")
	return (FunctionListWithcode)(c)
}

func (c FunctionListLibraryname) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type FunctionListWithcode Completed

func (c FunctionListWithcode) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type FunctionLoad Completed

func (b Builder) FunctionLoad() (c FunctionLoad) {
	c = FunctionLoad{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "FUNCTION", "LOAD")
	return c
}

func (c FunctionLoad) Replace() FunctionLoadReplace {
	c.cs.s = append(c.cs.s, "REPLACE")
	return (FunctionLoadReplace)(c)
}

func (c FunctionLoad) FunctionCode(functionCode string) FunctionLoadFunctionCode {
	c.cs.s = append(c.cs.s, functionCode)
	return (FunctionLoadFunctionCode)(c)
}

type FunctionLoadFunctionCode Completed

func (c FunctionLoadFunctionCode) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type FunctionLoadReplace Completed

func (c FunctionLoadReplace) FunctionCode(functionCode string) FunctionLoadFunctionCode {
	c.cs.s = append(c.cs.s, functionCode)
	return (FunctionLoadFunctionCode)(c)
}

type FunctionRestore Completed

func (b Builder) FunctionRestore() (c FunctionRestore) {
	c = FunctionRestore{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "FUNCTION", "RESTORE")
	return c
}

func (c FunctionRestore) SerializedValue(serializedValue string) FunctionRestoreSerializedValue {
	c.cs.s = append(c.cs.s, serializedValue)
	return (FunctionRestoreSerializedValue)(c)
}

type FunctionRestorePolicyAppend Completed

func (c FunctionRestorePolicyAppend) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type FunctionRestorePolicyFlush Completed

func (c FunctionRestorePolicyFlush) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type FunctionRestorePolicyReplace Completed

func (c FunctionRestorePolicyReplace) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type FunctionRestoreSerializedValue Completed

func (c FunctionRestoreSerializedValue) Flush() FunctionRestorePolicyFlush {
	c.cs.s = append(c.cs.s, "FLUSH")
	return (FunctionRestorePolicyFlush)(c)
}

func (c FunctionRestoreSerializedValue) Append() FunctionRestorePolicyAppend {
	c.cs.s = append(c.cs.s, "APPEND")
	return (FunctionRestorePolicyAppend)(c)
}

func (c FunctionRestoreSerializedValue) Replace() FunctionRestorePolicyReplace {
	c.cs.s = append(c.cs.s, "REPLACE")
	return (FunctionRestorePolicyReplace)(c)
}

func (c FunctionRestoreSerializedValue) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type FunctionStats Completed

func (b Builder) FunctionStats() (c FunctionStats) {
	c = FunctionStats{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "FUNCTION", "STATS")
	return c
}

func (c FunctionStats) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type ScriptDebug Completed

func (b Builder) ScriptDebug() (c ScriptDebug) {
	c = ScriptDebug{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "SCRIPT", "DEBUG")
	return c
}

func (c ScriptDebug) Yes() ScriptDebugModeYes {
	c.cs.s = append(c.cs.s, "YES")
	return (ScriptDebugModeYes)(c)
}

func (c ScriptDebug) Sync() ScriptDebugModeSync {
	c.cs.s = append(c.cs.s, "SYNC")
	return (ScriptDebugModeSync)(c)
}

func (c ScriptDebug) No() ScriptDebugModeNo {
	c.cs.s = append(c.cs.s, "NO")
	return (ScriptDebugModeNo)(c)
}

type ScriptDebugModeNo Completed

func (c ScriptDebugModeNo) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type ScriptDebugModeSync Completed

func (c ScriptDebugModeSync) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type ScriptDebugModeYes Completed

func (c ScriptDebugModeYes) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type ScriptExists Completed

func (b Builder) ScriptExists() (c ScriptExists) {
	c = ScriptExists{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "SCRIPT", "EXISTS")
	return c
}

func (c ScriptExists) Sha1(sha1 ...string) ScriptExistsSha1 {
	c.cs.s = append(c.cs.s, sha1...)
	return (ScriptExistsSha1)(c)
}

type ScriptExistsSha1 Completed

func (c ScriptExistsSha1) Sha1(sha1 ...string) ScriptExistsSha1 {
	c.cs.s = append(c.cs.s, sha1...)
	return c
}

func (c ScriptExistsSha1) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type ScriptFlush Completed

func (b Builder) ScriptFlush() (c ScriptFlush) {
	c = ScriptFlush{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "SCRIPT", "FLUSH")
	return c
}

func (c ScriptFlush) Async() ScriptFlushAsync {
	c.cs.s = append(c.cs.s, "ASYNC")
	return (ScriptFlushAsync)(c)
}

func (c ScriptFlush) Sync() ScriptFlushAsyncSync {
	c.cs.s = append(c.cs.s, "SYNC")
	return (ScriptFlushAsyncSync)(c)
}

func (c ScriptFlush) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type ScriptFlushAsync Completed

func (c ScriptFlushAsync) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type ScriptFlushAsyncSync Completed

func (c ScriptFlushAsyncSync) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type ScriptKill Completed

func (b Builder) ScriptKill() (c ScriptKill) {
	c = ScriptKill{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "SCRIPT", "KILL")
	return c
}

func (c ScriptKill) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type ScriptLoad Completed

func (b Builder) ScriptLoad() (c ScriptLoad) {
	c = ScriptLoad{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "SCRIPT", "LOAD")
	return c
}

func (c ScriptLoad) Script(script string) ScriptLoadScript {
	c.cs.s = append(c.cs.s, script)
	return (ScriptLoadScript)(c)
}

type ScriptLoadScript Completed

func (c ScriptLoadScript) Build() Completed {
	c.cs.Build()
	return Completed(c)
}
