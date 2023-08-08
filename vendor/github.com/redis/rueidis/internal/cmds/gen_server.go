// Code generated DO NOT EDIT

package cmds

import "strconv"

type AclCat Completed

func (b Builder) AclCat() (c AclCat) {
	c = AclCat{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "ACL", "CAT")
	return c
}

func (c AclCat) Categoryname(categoryname string) AclCatCategoryname {
	c.cs.s = append(c.cs.s, categoryname)
	return (AclCatCategoryname)(c)
}

func (c AclCat) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type AclCatCategoryname Completed

func (c AclCatCategoryname) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type AclDeluser Completed

func (b Builder) AclDeluser() (c AclDeluser) {
	c = AclDeluser{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "ACL", "DELUSER")
	return c
}

func (c AclDeluser) Username(username ...string) AclDeluserUsername {
	c.cs.s = append(c.cs.s, username...)
	return (AclDeluserUsername)(c)
}

type AclDeluserUsername Completed

func (c AclDeluserUsername) Username(username ...string) AclDeluserUsername {
	c.cs.s = append(c.cs.s, username...)
	return c
}

func (c AclDeluserUsername) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type AclDryrun Completed

func (b Builder) AclDryrun() (c AclDryrun) {
	c = AclDryrun{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "ACL", "DRYRUN")
	return c
}

func (c AclDryrun) Username(username string) AclDryrunUsername {
	c.cs.s = append(c.cs.s, username)
	return (AclDryrunUsername)(c)
}

type AclDryrunArg Completed

func (c AclDryrunArg) Arg(arg ...string) AclDryrunArg {
	c.cs.s = append(c.cs.s, arg...)
	return c
}

func (c AclDryrunArg) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type AclDryrunCommand Completed

func (c AclDryrunCommand) Arg(arg ...string) AclDryrunArg {
	c.cs.s = append(c.cs.s, arg...)
	return (AclDryrunArg)(c)
}

func (c AclDryrunCommand) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type AclDryrunUsername Completed

func (c AclDryrunUsername) Command(command string) AclDryrunCommand {
	c.cs.s = append(c.cs.s, command)
	return (AclDryrunCommand)(c)
}

type AclGenpass Completed

func (b Builder) AclGenpass() (c AclGenpass) {
	c = AclGenpass{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "ACL", "GENPASS")
	return c
}

func (c AclGenpass) Bits(bits int64) AclGenpassBits {
	c.cs.s = append(c.cs.s, strconv.FormatInt(bits, 10))
	return (AclGenpassBits)(c)
}

func (c AclGenpass) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type AclGenpassBits Completed

func (c AclGenpassBits) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type AclGetuser Completed

func (b Builder) AclGetuser() (c AclGetuser) {
	c = AclGetuser{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "ACL", "GETUSER")
	return c
}

func (c AclGetuser) Username(username string) AclGetuserUsername {
	c.cs.s = append(c.cs.s, username)
	return (AclGetuserUsername)(c)
}

type AclGetuserUsername Completed

func (c AclGetuserUsername) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type AclHelp Completed

func (b Builder) AclHelp() (c AclHelp) {
	c = AclHelp{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "ACL", "HELP")
	return c
}

func (c AclHelp) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type AclList Completed

func (b Builder) AclList() (c AclList) {
	c = AclList{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "ACL", "LIST")
	return c
}

func (c AclList) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type AclLoad Completed

func (b Builder) AclLoad() (c AclLoad) {
	c = AclLoad{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "ACL", "LOAD")
	return c
}

func (c AclLoad) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type AclLog Completed

func (b Builder) AclLog() (c AclLog) {
	c = AclLog{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "ACL", "LOG")
	return c
}

func (c AclLog) Count(count int64) AclLogCountCount {
	c.cs.s = append(c.cs.s, strconv.FormatInt(count, 10))
	return (AclLogCountCount)(c)
}

func (c AclLog) Reset() AclLogCountReset {
	c.cs.s = append(c.cs.s, "RESET")
	return (AclLogCountReset)(c)
}

type AclLogCountCount Completed

func (c AclLogCountCount) Reset() AclLogCountReset {
	c.cs.s = append(c.cs.s, "RESET")
	return (AclLogCountReset)(c)
}

func (c AclLogCountCount) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type AclLogCountReset Completed

func (c AclLogCountReset) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type AclSave Completed

func (b Builder) AclSave() (c AclSave) {
	c = AclSave{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "ACL", "SAVE")
	return c
}

func (c AclSave) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type AclSetuser Completed

func (b Builder) AclSetuser() (c AclSetuser) {
	c = AclSetuser{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "ACL", "SETUSER")
	return c
}

func (c AclSetuser) Username(username string) AclSetuserUsername {
	c.cs.s = append(c.cs.s, username)
	return (AclSetuserUsername)(c)
}

type AclSetuserRule Completed

func (c AclSetuserRule) Rule(rule ...string) AclSetuserRule {
	c.cs.s = append(c.cs.s, rule...)
	return c
}

func (c AclSetuserRule) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type AclSetuserUsername Completed

func (c AclSetuserUsername) Rule(rule ...string) AclSetuserRule {
	c.cs.s = append(c.cs.s, rule...)
	return (AclSetuserRule)(c)
}

func (c AclSetuserUsername) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type AclUsers Completed

func (b Builder) AclUsers() (c AclUsers) {
	c = AclUsers{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "ACL", "USERS")
	return c
}

func (c AclUsers) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type AclWhoami Completed

func (b Builder) AclWhoami() (c AclWhoami) {
	c = AclWhoami{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "ACL", "WHOAMI")
	return c
}

func (c AclWhoami) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type Bgrewriteaof Completed

func (b Builder) Bgrewriteaof() (c Bgrewriteaof) {
	c = Bgrewriteaof{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "BGREWRITEAOF")
	return c
}

func (c Bgrewriteaof) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type Bgsave Completed

func (b Builder) Bgsave() (c Bgsave) {
	c = Bgsave{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "BGSAVE")
	return c
}

func (c Bgsave) Schedule() BgsaveSchedule {
	c.cs.s = append(c.cs.s, "SCHEDULE")
	return (BgsaveSchedule)(c)
}

func (c Bgsave) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type BgsaveSchedule Completed

func (c BgsaveSchedule) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type Command Completed

func (b Builder) Command() (c Command) {
	c = Command{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "COMMAND")
	return c
}

func (c Command) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type CommandCount Completed

func (b Builder) CommandCount() (c CommandCount) {
	c = CommandCount{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "COMMAND", "COUNT")
	return c
}

func (c CommandCount) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type CommandDocs Completed

func (b Builder) CommandDocs() (c CommandDocs) {
	c = CommandDocs{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "COMMAND", "DOCS")
	return c
}

func (c CommandDocs) CommandName(commandName ...string) CommandDocsCommandName {
	c.cs.s = append(c.cs.s, commandName...)
	return (CommandDocsCommandName)(c)
}

func (c CommandDocs) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type CommandDocsCommandName Completed

func (c CommandDocsCommandName) CommandName(commandName ...string) CommandDocsCommandName {
	c.cs.s = append(c.cs.s, commandName...)
	return c
}

func (c CommandDocsCommandName) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type CommandGetkeys Completed

func (b Builder) CommandGetkeys() (c CommandGetkeys) {
	c = CommandGetkeys{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "COMMAND", "GETKEYS")
	return c
}

func (c CommandGetkeys) Command(command string) CommandGetkeysCommand {
	c.cs.s = append(c.cs.s, command)
	return (CommandGetkeysCommand)(c)
}

type CommandGetkeysArg Completed

func (c CommandGetkeysArg) Arg(arg ...string) CommandGetkeysArg {
	c.cs.s = append(c.cs.s, arg...)
	return c
}

func (c CommandGetkeysArg) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type CommandGetkeysCommand Completed

func (c CommandGetkeysCommand) Arg(arg ...string) CommandGetkeysArg {
	c.cs.s = append(c.cs.s, arg...)
	return (CommandGetkeysArg)(c)
}

func (c CommandGetkeysCommand) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type CommandGetkeysandflags Completed

func (b Builder) CommandGetkeysandflags() (c CommandGetkeysandflags) {
	c = CommandGetkeysandflags{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "COMMAND", "GETKEYSANDFLAGS")
	return c
}

func (c CommandGetkeysandflags) Command(command string) CommandGetkeysandflagsCommand {
	c.cs.s = append(c.cs.s, command)
	return (CommandGetkeysandflagsCommand)(c)
}

type CommandGetkeysandflagsArg Completed

func (c CommandGetkeysandflagsArg) Arg(arg ...string) CommandGetkeysandflagsArg {
	c.cs.s = append(c.cs.s, arg...)
	return c
}

func (c CommandGetkeysandflagsArg) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type CommandGetkeysandflagsCommand Completed

func (c CommandGetkeysandflagsCommand) Arg(arg ...string) CommandGetkeysandflagsArg {
	c.cs.s = append(c.cs.s, arg...)
	return (CommandGetkeysandflagsArg)(c)
}

func (c CommandGetkeysandflagsCommand) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type CommandInfo Completed

func (b Builder) CommandInfo() (c CommandInfo) {
	c = CommandInfo{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "COMMAND", "INFO")
	return c
}

func (c CommandInfo) CommandName(commandName ...string) CommandInfoCommandName {
	c.cs.s = append(c.cs.s, commandName...)
	return (CommandInfoCommandName)(c)
}

func (c CommandInfo) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type CommandInfoCommandName Completed

func (c CommandInfoCommandName) CommandName(commandName ...string) CommandInfoCommandName {
	c.cs.s = append(c.cs.s, commandName...)
	return c
}

func (c CommandInfoCommandName) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type CommandList Completed

func (b Builder) CommandList() (c CommandList) {
	c = CommandList{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "COMMAND", "LIST")
	return c
}

func (c CommandList) FilterbyModuleName(name string) CommandListFilterbyModuleName {
	c.cs.s = append(c.cs.s, "FILTERBY", "MODULE", name)
	return (CommandListFilterbyModuleName)(c)
}

func (c CommandList) FilterbyAclcatCategory(category string) CommandListFilterbyAclcatCategory {
	c.cs.s = append(c.cs.s, "FILTERBY", "ACLCAT", category)
	return (CommandListFilterbyAclcatCategory)(c)
}

func (c CommandList) FilterbyPatternPattern(pattern string) CommandListFilterbyPatternPattern {
	c.cs.s = append(c.cs.s, "FILTERBY", "PATTERN", pattern)
	return (CommandListFilterbyPatternPattern)(c)
}

func (c CommandList) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type CommandListFilterbyAclcatCategory Completed

func (c CommandListFilterbyAclcatCategory) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type CommandListFilterbyModuleName Completed

func (c CommandListFilterbyModuleName) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type CommandListFilterbyPatternPattern Completed

func (c CommandListFilterbyPatternPattern) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type ConfigGet Completed

func (b Builder) ConfigGet() (c ConfigGet) {
	c = ConfigGet{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "CONFIG", "GET")
	return c
}

func (c ConfigGet) Parameter(parameter ...string) ConfigGetParameter {
	c.cs.s = append(c.cs.s, parameter...)
	return (ConfigGetParameter)(c)
}

type ConfigGetParameter Completed

func (c ConfigGetParameter) Parameter(parameter ...string) ConfigGetParameter {
	c.cs.s = append(c.cs.s, parameter...)
	return c
}

func (c ConfigGetParameter) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type ConfigResetstat Completed

func (b Builder) ConfigResetstat() (c ConfigResetstat) {
	c = ConfigResetstat{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "CONFIG", "RESETSTAT")
	return c
}

func (c ConfigResetstat) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type ConfigRewrite Completed

func (b Builder) ConfigRewrite() (c ConfigRewrite) {
	c = ConfigRewrite{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "CONFIG", "REWRITE")
	return c
}

func (c ConfigRewrite) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type ConfigSet Completed

func (b Builder) ConfigSet() (c ConfigSet) {
	c = ConfigSet{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "CONFIG", "SET")
	return c
}

func (c ConfigSet) ParameterValue() ConfigSetParameterValue {
	return (ConfigSetParameterValue)(c)
}

type ConfigSetParameterValue Completed

func (c ConfigSetParameterValue) ParameterValue(parameter string, value string) ConfigSetParameterValue {
	c.cs.s = append(c.cs.s, parameter, value)
	return c
}

func (c ConfigSetParameterValue) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type Dbsize Completed

func (b Builder) Dbsize() (c Dbsize) {
	c = Dbsize{cs: get(), ks: b.ks, cf: readonly}
	c.cs.s = append(c.cs.s, "DBSIZE")
	return c
}

func (c Dbsize) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type DebugObject Completed

func (b Builder) DebugObject() (c DebugObject) {
	c = DebugObject{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "DEBUG", "OBJECT")
	return c
}

func (c DebugObject) Key(key string) DebugObjectKey {
	if c.ks&NoSlot == NoSlot {
		c.ks = NoSlot | slot(key)
	} else {
		c.ks = check(c.ks, slot(key))
	}
	c.cs.s = append(c.cs.s, key)
	return (DebugObjectKey)(c)
}

type DebugObjectKey Completed

func (c DebugObjectKey) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type DebugSegfault Completed

func (b Builder) DebugSegfault() (c DebugSegfault) {
	c = DebugSegfault{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "DEBUG", "SEGFAULT")
	return c
}

func (c DebugSegfault) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type Failover Completed

func (b Builder) Failover() (c Failover) {
	c = Failover{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "FAILOVER")
	return c
}

func (c Failover) To() FailoverTargetTo {
	c.cs.s = append(c.cs.s, "TO")
	return (FailoverTargetTo)(c)
}

func (c Failover) Abort() FailoverAbort {
	c.cs.s = append(c.cs.s, "ABORT")
	return (FailoverAbort)(c)
}

func (c Failover) Timeout(milliseconds int64) FailoverTimeout {
	c.cs.s = append(c.cs.s, "TIMEOUT", strconv.FormatInt(milliseconds, 10))
	return (FailoverTimeout)(c)
}

func (c Failover) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type FailoverAbort Completed

func (c FailoverAbort) Timeout(milliseconds int64) FailoverTimeout {
	c.cs.s = append(c.cs.s, "TIMEOUT", strconv.FormatInt(milliseconds, 10))
	return (FailoverTimeout)(c)
}

func (c FailoverAbort) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type FailoverTargetForce Completed

func (c FailoverTargetForce) Abort() FailoverAbort {
	c.cs.s = append(c.cs.s, "ABORT")
	return (FailoverAbort)(c)
}

func (c FailoverTargetForce) Timeout(milliseconds int64) FailoverTimeout {
	c.cs.s = append(c.cs.s, "TIMEOUT", strconv.FormatInt(milliseconds, 10))
	return (FailoverTimeout)(c)
}

func (c FailoverTargetForce) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type FailoverTargetHost Completed

func (c FailoverTargetHost) Port(port int64) FailoverTargetPort {
	c.cs.s = append(c.cs.s, strconv.FormatInt(port, 10))
	return (FailoverTargetPort)(c)
}

type FailoverTargetPort Completed

func (c FailoverTargetPort) Force() FailoverTargetForce {
	c.cs.s = append(c.cs.s, "FORCE")
	return (FailoverTargetForce)(c)
}

func (c FailoverTargetPort) Abort() FailoverAbort {
	c.cs.s = append(c.cs.s, "ABORT")
	return (FailoverAbort)(c)
}

func (c FailoverTargetPort) Timeout(milliseconds int64) FailoverTimeout {
	c.cs.s = append(c.cs.s, "TIMEOUT", strconv.FormatInt(milliseconds, 10))
	return (FailoverTimeout)(c)
}

func (c FailoverTargetPort) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type FailoverTargetTo Completed

func (c FailoverTargetTo) Host(host string) FailoverTargetHost {
	c.cs.s = append(c.cs.s, host)
	return (FailoverTargetHost)(c)
}

type FailoverTimeout Completed

func (c FailoverTimeout) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type Flushall Completed

func (b Builder) Flushall() (c Flushall) {
	c = Flushall{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "FLUSHALL")
	return c
}

func (c Flushall) Async() FlushallAsync {
	c.cs.s = append(c.cs.s, "ASYNC")
	return (FlushallAsync)(c)
}

func (c Flushall) Sync() FlushallAsyncSync {
	c.cs.s = append(c.cs.s, "SYNC")
	return (FlushallAsyncSync)(c)
}

func (c Flushall) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type FlushallAsync Completed

func (c FlushallAsync) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type FlushallAsyncSync Completed

func (c FlushallAsyncSync) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type Flushdb Completed

func (b Builder) Flushdb() (c Flushdb) {
	c = Flushdb{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "FLUSHDB")
	return c
}

func (c Flushdb) Async() FlushdbAsync {
	c.cs.s = append(c.cs.s, "ASYNC")
	return (FlushdbAsync)(c)
}

func (c Flushdb) Sync() FlushdbAsyncSync {
	c.cs.s = append(c.cs.s, "SYNC")
	return (FlushdbAsyncSync)(c)
}

func (c Flushdb) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type FlushdbAsync Completed

func (c FlushdbAsync) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type FlushdbAsyncSync Completed

func (c FlushdbAsyncSync) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type Info Completed

func (b Builder) Info() (c Info) {
	c = Info{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "INFO")
	return c
}

func (c Info) Section(section ...string) InfoSection {
	c.cs.s = append(c.cs.s, section...)
	return (InfoSection)(c)
}

func (c Info) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type InfoSection Completed

func (c InfoSection) Section(section ...string) InfoSection {
	c.cs.s = append(c.cs.s, section...)
	return c
}

func (c InfoSection) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type Lastsave Completed

func (b Builder) Lastsave() (c Lastsave) {
	c = Lastsave{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "LASTSAVE")
	return c
}

func (c Lastsave) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type LatencyDoctor Completed

func (b Builder) LatencyDoctor() (c LatencyDoctor) {
	c = LatencyDoctor{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "LATENCY", "DOCTOR")
	return c
}

func (c LatencyDoctor) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type LatencyGraph Completed

func (b Builder) LatencyGraph() (c LatencyGraph) {
	c = LatencyGraph{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "LATENCY", "GRAPH")
	return c
}

func (c LatencyGraph) Event(event string) LatencyGraphEvent {
	c.cs.s = append(c.cs.s, event)
	return (LatencyGraphEvent)(c)
}

type LatencyGraphEvent Completed

func (c LatencyGraphEvent) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type LatencyHelp Completed

func (b Builder) LatencyHelp() (c LatencyHelp) {
	c = LatencyHelp{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "LATENCY", "HELP")
	return c
}

func (c LatencyHelp) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type LatencyHistogram Completed

func (b Builder) LatencyHistogram() (c LatencyHistogram) {
	c = LatencyHistogram{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "LATENCY", "HISTOGRAM")
	return c
}

func (c LatencyHistogram) Command(command ...string) LatencyHistogramCommand {
	c.cs.s = append(c.cs.s, command...)
	return (LatencyHistogramCommand)(c)
}

func (c LatencyHistogram) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type LatencyHistogramCommand Completed

func (c LatencyHistogramCommand) Command(command ...string) LatencyHistogramCommand {
	c.cs.s = append(c.cs.s, command...)
	return c
}

func (c LatencyHistogramCommand) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type LatencyHistory Completed

func (b Builder) LatencyHistory() (c LatencyHistory) {
	c = LatencyHistory{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "LATENCY", "HISTORY")
	return c
}

func (c LatencyHistory) Event(event string) LatencyHistoryEvent {
	c.cs.s = append(c.cs.s, event)
	return (LatencyHistoryEvent)(c)
}

type LatencyHistoryEvent Completed

func (c LatencyHistoryEvent) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type LatencyLatest Completed

func (b Builder) LatencyLatest() (c LatencyLatest) {
	c = LatencyLatest{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "LATENCY", "LATEST")
	return c
}

func (c LatencyLatest) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type LatencyReset Completed

func (b Builder) LatencyReset() (c LatencyReset) {
	c = LatencyReset{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "LATENCY", "RESET")
	return c
}

func (c LatencyReset) Event(event ...string) LatencyResetEvent {
	c.cs.s = append(c.cs.s, event...)
	return (LatencyResetEvent)(c)
}

func (c LatencyReset) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type LatencyResetEvent Completed

func (c LatencyResetEvent) Event(event ...string) LatencyResetEvent {
	c.cs.s = append(c.cs.s, event...)
	return c
}

func (c LatencyResetEvent) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type Lolwut Completed

func (b Builder) Lolwut() (c Lolwut) {
	c = Lolwut{cs: get(), ks: b.ks, cf: readonly}
	c.cs.s = append(c.cs.s, "LOLWUT")
	return c
}

func (c Lolwut) Version(version int64) LolwutVersion {
	c.cs.s = append(c.cs.s, "VERSION", strconv.FormatInt(version, 10))
	return (LolwutVersion)(c)
}

func (c Lolwut) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type LolwutVersion Completed

func (c LolwutVersion) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type MemoryDoctor Completed

func (b Builder) MemoryDoctor() (c MemoryDoctor) {
	c = MemoryDoctor{cs: get(), ks: b.ks, cf: readonly}
	c.cs.s = append(c.cs.s, "MEMORY", "DOCTOR")
	return c
}

func (c MemoryDoctor) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type MemoryHelp Completed

func (b Builder) MemoryHelp() (c MemoryHelp) {
	c = MemoryHelp{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "MEMORY", "HELP")
	return c
}

func (c MemoryHelp) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type MemoryMallocStats Completed

func (b Builder) MemoryMallocStats() (c MemoryMallocStats) {
	c = MemoryMallocStats{cs: get(), ks: b.ks, cf: readonly}
	c.cs.s = append(c.cs.s, "MEMORY", "MALLOC-STATS")
	return c
}

func (c MemoryMallocStats) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type MemoryPurge Completed

func (b Builder) MemoryPurge() (c MemoryPurge) {
	c = MemoryPurge{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "MEMORY", "PURGE")
	return c
}

func (c MemoryPurge) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type MemoryStats Completed

func (b Builder) MemoryStats() (c MemoryStats) {
	c = MemoryStats{cs: get(), ks: b.ks, cf: readonly}
	c.cs.s = append(c.cs.s, "MEMORY", "STATS")
	return c
}

func (c MemoryStats) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type MemoryUsage Completed

func (b Builder) MemoryUsage() (c MemoryUsage) {
	c = MemoryUsage{cs: get(), ks: b.ks, cf: readonly}
	c.cs.s = append(c.cs.s, "MEMORY", "USAGE")
	return c
}

func (c MemoryUsage) Key(key string) MemoryUsageKey {
	if c.ks&NoSlot == NoSlot {
		c.ks = NoSlot | slot(key)
	} else {
		c.ks = check(c.ks, slot(key))
	}
	c.cs.s = append(c.cs.s, key)
	return (MemoryUsageKey)(c)
}

type MemoryUsageKey Completed

func (c MemoryUsageKey) Samples(count int64) MemoryUsageSamples {
	c.cs.s = append(c.cs.s, "SAMPLES", strconv.FormatInt(count, 10))
	return (MemoryUsageSamples)(c)
}

func (c MemoryUsageKey) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type MemoryUsageSamples Completed

func (c MemoryUsageSamples) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type ModuleList Completed

func (b Builder) ModuleList() (c ModuleList) {
	c = ModuleList{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "MODULE", "LIST")
	return c
}

func (c ModuleList) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type ModuleLoad Completed

func (b Builder) ModuleLoad() (c ModuleLoad) {
	c = ModuleLoad{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "MODULE", "LOAD")
	return c
}

func (c ModuleLoad) Path(path string) ModuleLoadPath {
	c.cs.s = append(c.cs.s, path)
	return (ModuleLoadPath)(c)
}

type ModuleLoadArg Completed

func (c ModuleLoadArg) Arg(arg ...string) ModuleLoadArg {
	c.cs.s = append(c.cs.s, arg...)
	return c
}

func (c ModuleLoadArg) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type ModuleLoadPath Completed

func (c ModuleLoadPath) Arg(arg ...string) ModuleLoadArg {
	c.cs.s = append(c.cs.s, arg...)
	return (ModuleLoadArg)(c)
}

func (c ModuleLoadPath) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type ModuleLoadex Completed

func (b Builder) ModuleLoadex() (c ModuleLoadex) {
	c = ModuleLoadex{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "MODULE", "LOADEX")
	return c
}

func (c ModuleLoadex) Path(path string) ModuleLoadexPath {
	c.cs.s = append(c.cs.s, path)
	return (ModuleLoadexPath)(c)
}

type ModuleLoadexArgs Completed

func (c ModuleLoadexArgs) Args(args ...string) ModuleLoadexArgs {
	c.cs.s = append(c.cs.s, "ARGS")
	c.cs.s = append(c.cs.s, args...)
	return c
}

func (c ModuleLoadexArgs) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type ModuleLoadexConfig Completed

func (c ModuleLoadexConfig) Config(name string, value string) ModuleLoadexConfig {
	c.cs.s = append(c.cs.s, "CONFIG", name, value)
	return c
}

func (c ModuleLoadexConfig) Args(args ...string) ModuleLoadexArgs {
	c.cs.s = append(c.cs.s, "ARGS")
	c.cs.s = append(c.cs.s, args...)
	return (ModuleLoadexArgs)(c)
}

func (c ModuleLoadexConfig) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type ModuleLoadexPath Completed

func (c ModuleLoadexPath) Config() ModuleLoadexConfig {
	return (ModuleLoadexConfig)(c)
}

func (c ModuleLoadexPath) Args(args ...string) ModuleLoadexArgs {
	c.cs.s = append(c.cs.s, "ARGS")
	c.cs.s = append(c.cs.s, args...)
	return (ModuleLoadexArgs)(c)
}

func (c ModuleLoadexPath) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type ModuleUnload Completed

func (b Builder) ModuleUnload() (c ModuleUnload) {
	c = ModuleUnload{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "MODULE", "UNLOAD")
	return c
}

func (c ModuleUnload) Name(name string) ModuleUnloadName {
	c.cs.s = append(c.cs.s, name)
	return (ModuleUnloadName)(c)
}

type ModuleUnloadName Completed

func (c ModuleUnloadName) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type Monitor Completed

func (b Builder) Monitor() (c Monitor) {
	c = Monitor{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "MONITOR")
	return c
}

func (c Monitor) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type Psync Completed

func (b Builder) Psync() (c Psync) {
	c = Psync{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "PSYNC")
	return c
}

func (c Psync) Replicationid(replicationid string) PsyncReplicationid {
	c.cs.s = append(c.cs.s, replicationid)
	return (PsyncReplicationid)(c)
}

type PsyncOffset Completed

func (c PsyncOffset) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type PsyncReplicationid Completed

func (c PsyncReplicationid) Offset(offset int64) PsyncOffset {
	c.cs.s = append(c.cs.s, strconv.FormatInt(offset, 10))
	return (PsyncOffset)(c)
}

type Replicaof Completed

func (b Builder) Replicaof() (c Replicaof) {
	c = Replicaof{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "REPLICAOF")
	return c
}

func (c Replicaof) Host(host string) ReplicaofHost {
	c.cs.s = append(c.cs.s, host)
	return (ReplicaofHost)(c)
}

type ReplicaofHost Completed

func (c ReplicaofHost) Port(port int64) ReplicaofPort {
	c.cs.s = append(c.cs.s, strconv.FormatInt(port, 10))
	return (ReplicaofPort)(c)
}

type ReplicaofPort Completed

func (c ReplicaofPort) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type Role Completed

func (b Builder) Role() (c Role) {
	c = Role{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "ROLE")
	return c
}

func (c Role) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type Save Completed

func (b Builder) Save() (c Save) {
	c = Save{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "SAVE")
	return c
}

func (c Save) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type Shutdown Completed

func (b Builder) Shutdown() (c Shutdown) {
	c = Shutdown{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "SHUTDOWN")
	return c
}

func (c Shutdown) Nosave() ShutdownSaveModeNosave {
	c.cs.s = append(c.cs.s, "NOSAVE")
	return (ShutdownSaveModeNosave)(c)
}

func (c Shutdown) Save() ShutdownSaveModeSave {
	c.cs.s = append(c.cs.s, "SAVE")
	return (ShutdownSaveModeSave)(c)
}

func (c Shutdown) Now() ShutdownNow {
	c.cs.s = append(c.cs.s, "NOW")
	return (ShutdownNow)(c)
}

func (c Shutdown) Force() ShutdownForce {
	c.cs.s = append(c.cs.s, "FORCE")
	return (ShutdownForce)(c)
}

func (c Shutdown) Abort() ShutdownAbort {
	c.cs.s = append(c.cs.s, "ABORT")
	return (ShutdownAbort)(c)
}

func (c Shutdown) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type ShutdownAbort Completed

func (c ShutdownAbort) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type ShutdownForce Completed

func (c ShutdownForce) Abort() ShutdownAbort {
	c.cs.s = append(c.cs.s, "ABORT")
	return (ShutdownAbort)(c)
}

func (c ShutdownForce) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type ShutdownNow Completed

func (c ShutdownNow) Force() ShutdownForce {
	c.cs.s = append(c.cs.s, "FORCE")
	return (ShutdownForce)(c)
}

func (c ShutdownNow) Abort() ShutdownAbort {
	c.cs.s = append(c.cs.s, "ABORT")
	return (ShutdownAbort)(c)
}

func (c ShutdownNow) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type ShutdownSaveModeNosave Completed

func (c ShutdownSaveModeNosave) Now() ShutdownNow {
	c.cs.s = append(c.cs.s, "NOW")
	return (ShutdownNow)(c)
}

func (c ShutdownSaveModeNosave) Force() ShutdownForce {
	c.cs.s = append(c.cs.s, "FORCE")
	return (ShutdownForce)(c)
}

func (c ShutdownSaveModeNosave) Abort() ShutdownAbort {
	c.cs.s = append(c.cs.s, "ABORT")
	return (ShutdownAbort)(c)
}

func (c ShutdownSaveModeNosave) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type ShutdownSaveModeSave Completed

func (c ShutdownSaveModeSave) Now() ShutdownNow {
	c.cs.s = append(c.cs.s, "NOW")
	return (ShutdownNow)(c)
}

func (c ShutdownSaveModeSave) Force() ShutdownForce {
	c.cs.s = append(c.cs.s, "FORCE")
	return (ShutdownForce)(c)
}

func (c ShutdownSaveModeSave) Abort() ShutdownAbort {
	c.cs.s = append(c.cs.s, "ABORT")
	return (ShutdownAbort)(c)
}

func (c ShutdownSaveModeSave) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type Slaveof Completed

func (b Builder) Slaveof() (c Slaveof) {
	c = Slaveof{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "SLAVEOF")
	return c
}

func (c Slaveof) Host(host string) SlaveofHost {
	c.cs.s = append(c.cs.s, host)
	return (SlaveofHost)(c)
}

type SlaveofHost Completed

func (c SlaveofHost) Port(port int64) SlaveofPort {
	c.cs.s = append(c.cs.s, strconv.FormatInt(port, 10))
	return (SlaveofPort)(c)
}

type SlaveofPort Completed

func (c SlaveofPort) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type SlowlogGet Completed

func (b Builder) SlowlogGet() (c SlowlogGet) {
	c = SlowlogGet{cs: get(), ks: b.ks, cf: readonly}
	c.cs.s = append(c.cs.s, "SLOWLOG", "GET")
	return c
}

func (c SlowlogGet) Count(count int64) SlowlogGetCount {
	c.cs.s = append(c.cs.s, strconv.FormatInt(count, 10))
	return (SlowlogGetCount)(c)
}

func (c SlowlogGet) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type SlowlogGetCount Completed

func (c SlowlogGetCount) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type SlowlogHelp Completed

func (b Builder) SlowlogHelp() (c SlowlogHelp) {
	c = SlowlogHelp{cs: get(), ks: b.ks, cf: readonly}
	c.cs.s = append(c.cs.s, "SLOWLOG", "HELP")
	return c
}

func (c SlowlogHelp) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type SlowlogLen Completed

func (b Builder) SlowlogLen() (c SlowlogLen) {
	c = SlowlogLen{cs: get(), ks: b.ks, cf: readonly}
	c.cs.s = append(c.cs.s, "SLOWLOG", "LEN")
	return c
}

func (c SlowlogLen) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type SlowlogReset Completed

func (b Builder) SlowlogReset() (c SlowlogReset) {
	c = SlowlogReset{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "SLOWLOG", "RESET")
	return c
}

func (c SlowlogReset) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type Swapdb Completed

func (b Builder) Swapdb() (c Swapdb) {
	c = Swapdb{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "SWAPDB")
	return c
}

func (c Swapdb) Index1(index1 int64) SwapdbIndex1 {
	c.cs.s = append(c.cs.s, strconv.FormatInt(index1, 10))
	return (SwapdbIndex1)(c)
}

type SwapdbIndex1 Completed

func (c SwapdbIndex1) Index2(index2 int64) SwapdbIndex2 {
	c.cs.s = append(c.cs.s, strconv.FormatInt(index2, 10))
	return (SwapdbIndex2)(c)
}

type SwapdbIndex2 Completed

func (c SwapdbIndex2) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type Sync Completed

func (b Builder) Sync() (c Sync) {
	c = Sync{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "SYNC")
	return c
}

func (c Sync) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type Time Completed

func (b Builder) Time() (c Time) {
	c = Time{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "TIME")
	return c
}

func (c Time) Build() Completed {
	c.cs.Build()
	return Completed(c)
}
