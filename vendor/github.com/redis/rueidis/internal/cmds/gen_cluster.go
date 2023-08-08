// Code generated DO NOT EDIT

package cmds

import "strconv"

type Asking Completed

func (b Builder) Asking() (c Asking) {
	c = Asking{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "ASKING")
	return c
}

func (c Asking) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type ClusterAddslots Completed

func (b Builder) ClusterAddslots() (c ClusterAddslots) {
	c = ClusterAddslots{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "CLUSTER", "ADDSLOTS")
	return c
}

func (c ClusterAddslots) Slot(slot ...int64) ClusterAddslotsSlot {
	for _, n := range slot {
		c.cs.s = append(c.cs.s, strconv.FormatInt(n, 10))
	}
	return (ClusterAddslotsSlot)(c)
}

type ClusterAddslotsSlot Completed

func (c ClusterAddslotsSlot) Slot(slot ...int64) ClusterAddslotsSlot {
	for _, n := range slot {
		c.cs.s = append(c.cs.s, strconv.FormatInt(n, 10))
	}
	return c
}

func (c ClusterAddslotsSlot) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type ClusterAddslotsrange Completed

func (b Builder) ClusterAddslotsrange() (c ClusterAddslotsrange) {
	c = ClusterAddslotsrange{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "CLUSTER", "ADDSLOTSRANGE")
	return c
}

func (c ClusterAddslotsrange) StartSlotEndSlot() ClusterAddslotsrangeStartSlotEndSlot {
	return (ClusterAddslotsrangeStartSlotEndSlot)(c)
}

type ClusterAddslotsrangeStartSlotEndSlot Completed

func (c ClusterAddslotsrangeStartSlotEndSlot) StartSlotEndSlot(startSlot int64, endSlot int64) ClusterAddslotsrangeStartSlotEndSlot {
	c.cs.s = append(c.cs.s, strconv.FormatInt(startSlot, 10), strconv.FormatInt(endSlot, 10))
	return c
}

func (c ClusterAddslotsrangeStartSlotEndSlot) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type ClusterBumpepoch Completed

func (b Builder) ClusterBumpepoch() (c ClusterBumpepoch) {
	c = ClusterBumpepoch{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "CLUSTER", "BUMPEPOCH")
	return c
}

func (c ClusterBumpepoch) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type ClusterCountFailureReports Completed

func (b Builder) ClusterCountFailureReports() (c ClusterCountFailureReports) {
	c = ClusterCountFailureReports{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "CLUSTER", "COUNT-FAILURE-REPORTS")
	return c
}

func (c ClusterCountFailureReports) NodeId(nodeId string) ClusterCountFailureReportsNodeId {
	c.cs.s = append(c.cs.s, nodeId)
	return (ClusterCountFailureReportsNodeId)(c)
}

type ClusterCountFailureReportsNodeId Completed

func (c ClusterCountFailureReportsNodeId) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type ClusterCountkeysinslot Completed

func (b Builder) ClusterCountkeysinslot() (c ClusterCountkeysinslot) {
	c = ClusterCountkeysinslot{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "CLUSTER", "COUNTKEYSINSLOT")
	return c
}

func (c ClusterCountkeysinslot) Slot(slot int64) ClusterCountkeysinslotSlot {
	c.cs.s = append(c.cs.s, strconv.FormatInt(slot, 10))
	return (ClusterCountkeysinslotSlot)(c)
}

type ClusterCountkeysinslotSlot Completed

func (c ClusterCountkeysinslotSlot) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type ClusterDelslots Completed

func (b Builder) ClusterDelslots() (c ClusterDelslots) {
	c = ClusterDelslots{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "CLUSTER", "DELSLOTS")
	return c
}

func (c ClusterDelslots) Slot(slot ...int64) ClusterDelslotsSlot {
	for _, n := range slot {
		c.cs.s = append(c.cs.s, strconv.FormatInt(n, 10))
	}
	return (ClusterDelslotsSlot)(c)
}

type ClusterDelslotsSlot Completed

func (c ClusterDelslotsSlot) Slot(slot ...int64) ClusterDelslotsSlot {
	for _, n := range slot {
		c.cs.s = append(c.cs.s, strconv.FormatInt(n, 10))
	}
	return c
}

func (c ClusterDelslotsSlot) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type ClusterDelslotsrange Completed

func (b Builder) ClusterDelslotsrange() (c ClusterDelslotsrange) {
	c = ClusterDelslotsrange{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "CLUSTER", "DELSLOTSRANGE")
	return c
}

func (c ClusterDelslotsrange) StartSlotEndSlot() ClusterDelslotsrangeStartSlotEndSlot {
	return (ClusterDelslotsrangeStartSlotEndSlot)(c)
}

type ClusterDelslotsrangeStartSlotEndSlot Completed

func (c ClusterDelslotsrangeStartSlotEndSlot) StartSlotEndSlot(startSlot int64, endSlot int64) ClusterDelslotsrangeStartSlotEndSlot {
	c.cs.s = append(c.cs.s, strconv.FormatInt(startSlot, 10), strconv.FormatInt(endSlot, 10))
	return c
}

func (c ClusterDelslotsrangeStartSlotEndSlot) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type ClusterFailover Completed

func (b Builder) ClusterFailover() (c ClusterFailover) {
	c = ClusterFailover{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "CLUSTER", "FAILOVER")
	return c
}

func (c ClusterFailover) Force() ClusterFailoverOptionsForce {
	c.cs.s = append(c.cs.s, "FORCE")
	return (ClusterFailoverOptionsForce)(c)
}

func (c ClusterFailover) Takeover() ClusterFailoverOptionsTakeover {
	c.cs.s = append(c.cs.s, "TAKEOVER")
	return (ClusterFailoverOptionsTakeover)(c)
}

func (c ClusterFailover) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type ClusterFailoverOptionsForce Completed

func (c ClusterFailoverOptionsForce) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type ClusterFailoverOptionsTakeover Completed

func (c ClusterFailoverOptionsTakeover) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type ClusterFlushslots Completed

func (b Builder) ClusterFlushslots() (c ClusterFlushslots) {
	c = ClusterFlushslots{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "CLUSTER", "FLUSHSLOTS")
	return c
}

func (c ClusterFlushslots) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type ClusterForget Completed

func (b Builder) ClusterForget() (c ClusterForget) {
	c = ClusterForget{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "CLUSTER", "FORGET")
	return c
}

func (c ClusterForget) NodeId(nodeId string) ClusterForgetNodeId {
	c.cs.s = append(c.cs.s, nodeId)
	return (ClusterForgetNodeId)(c)
}

type ClusterForgetNodeId Completed

func (c ClusterForgetNodeId) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type ClusterGetkeysinslot Completed

func (b Builder) ClusterGetkeysinslot() (c ClusterGetkeysinslot) {
	c = ClusterGetkeysinslot{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "CLUSTER", "GETKEYSINSLOT")
	return c
}

func (c ClusterGetkeysinslot) Slot(slot int64) ClusterGetkeysinslotSlot {
	c.cs.s = append(c.cs.s, strconv.FormatInt(slot, 10))
	return (ClusterGetkeysinslotSlot)(c)
}

type ClusterGetkeysinslotCount Completed

func (c ClusterGetkeysinslotCount) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type ClusterGetkeysinslotSlot Completed

func (c ClusterGetkeysinslotSlot) Count(count int64) ClusterGetkeysinslotCount {
	c.cs.s = append(c.cs.s, strconv.FormatInt(count, 10))
	return (ClusterGetkeysinslotCount)(c)
}

type ClusterInfo Completed

func (b Builder) ClusterInfo() (c ClusterInfo) {
	c = ClusterInfo{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "CLUSTER", "INFO")
	return c
}

func (c ClusterInfo) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type ClusterKeyslot Completed

func (b Builder) ClusterKeyslot() (c ClusterKeyslot) {
	c = ClusterKeyslot{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "CLUSTER", "KEYSLOT")
	return c
}

func (c ClusterKeyslot) Key(key string) ClusterKeyslotKey {
	c.cs.s = append(c.cs.s, key)
	return (ClusterKeyslotKey)(c)
}

type ClusterKeyslotKey Completed

func (c ClusterKeyslotKey) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type ClusterLinks Completed

func (b Builder) ClusterLinks() (c ClusterLinks) {
	c = ClusterLinks{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "CLUSTER", "LINKS")
	return c
}

func (c ClusterLinks) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type ClusterMeet Completed

func (b Builder) ClusterMeet() (c ClusterMeet) {
	c = ClusterMeet{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "CLUSTER", "MEET")
	return c
}

func (c ClusterMeet) Ip(ip string) ClusterMeetIp {
	c.cs.s = append(c.cs.s, ip)
	return (ClusterMeetIp)(c)
}

type ClusterMeetClusterBusPort Completed

func (c ClusterMeetClusterBusPort) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type ClusterMeetIp Completed

func (c ClusterMeetIp) Port(port int64) ClusterMeetPort {
	c.cs.s = append(c.cs.s, strconv.FormatInt(port, 10))
	return (ClusterMeetPort)(c)
}

type ClusterMeetPort Completed

func (c ClusterMeetPort) ClusterBusPort(clusterBusPort int64) ClusterMeetClusterBusPort {
	c.cs.s = append(c.cs.s, strconv.FormatInt(clusterBusPort, 10))
	return (ClusterMeetClusterBusPort)(c)
}

func (c ClusterMeetPort) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type ClusterMyid Completed

func (b Builder) ClusterMyid() (c ClusterMyid) {
	c = ClusterMyid{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "CLUSTER", "MYID")
	return c
}

func (c ClusterMyid) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type ClusterMyshardid Completed

func (b Builder) ClusterMyshardid() (c ClusterMyshardid) {
	c = ClusterMyshardid{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "CLUSTER", "MYSHARDID")
	return c
}

func (c ClusterMyshardid) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type ClusterNodes Completed

func (b Builder) ClusterNodes() (c ClusterNodes) {
	c = ClusterNodes{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "CLUSTER", "NODES")
	return c
}

func (c ClusterNodes) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type ClusterReplicas Completed

func (b Builder) ClusterReplicas() (c ClusterReplicas) {
	c = ClusterReplicas{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "CLUSTER", "REPLICAS")
	return c
}

func (c ClusterReplicas) NodeId(nodeId string) ClusterReplicasNodeId {
	c.cs.s = append(c.cs.s, nodeId)
	return (ClusterReplicasNodeId)(c)
}

type ClusterReplicasNodeId Completed

func (c ClusterReplicasNodeId) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type ClusterReplicate Completed

func (b Builder) ClusterReplicate() (c ClusterReplicate) {
	c = ClusterReplicate{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "CLUSTER", "REPLICATE")
	return c
}

func (c ClusterReplicate) NodeId(nodeId string) ClusterReplicateNodeId {
	c.cs.s = append(c.cs.s, nodeId)
	return (ClusterReplicateNodeId)(c)
}

type ClusterReplicateNodeId Completed

func (c ClusterReplicateNodeId) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type ClusterReset Completed

func (b Builder) ClusterReset() (c ClusterReset) {
	c = ClusterReset{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "CLUSTER", "RESET")
	return c
}

func (c ClusterReset) Hard() ClusterResetResetTypeHard {
	c.cs.s = append(c.cs.s, "HARD")
	return (ClusterResetResetTypeHard)(c)
}

func (c ClusterReset) Soft() ClusterResetResetTypeSoft {
	c.cs.s = append(c.cs.s, "SOFT")
	return (ClusterResetResetTypeSoft)(c)
}

func (c ClusterReset) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type ClusterResetResetTypeHard Completed

func (c ClusterResetResetTypeHard) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type ClusterResetResetTypeSoft Completed

func (c ClusterResetResetTypeSoft) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type ClusterSaveconfig Completed

func (b Builder) ClusterSaveconfig() (c ClusterSaveconfig) {
	c = ClusterSaveconfig{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "CLUSTER", "SAVECONFIG")
	return c
}

func (c ClusterSaveconfig) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type ClusterSetConfigEpoch Completed

func (b Builder) ClusterSetConfigEpoch() (c ClusterSetConfigEpoch) {
	c = ClusterSetConfigEpoch{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "CLUSTER", "SET-CONFIG-EPOCH")
	return c
}

func (c ClusterSetConfigEpoch) ConfigEpoch(configEpoch int64) ClusterSetConfigEpochConfigEpoch {
	c.cs.s = append(c.cs.s, strconv.FormatInt(configEpoch, 10))
	return (ClusterSetConfigEpochConfigEpoch)(c)
}

type ClusterSetConfigEpochConfigEpoch Completed

func (c ClusterSetConfigEpochConfigEpoch) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type ClusterSetslot Completed

func (b Builder) ClusterSetslot() (c ClusterSetslot) {
	c = ClusterSetslot{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "CLUSTER", "SETSLOT")
	return c
}

func (c ClusterSetslot) Slot(slot int64) ClusterSetslotSlot {
	c.cs.s = append(c.cs.s, strconv.FormatInt(slot, 10))
	return (ClusterSetslotSlot)(c)
}

type ClusterSetslotNodeId Completed

func (c ClusterSetslotNodeId) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type ClusterSetslotSlot Completed

func (c ClusterSetslotSlot) Importing() ClusterSetslotSubcommandImporting {
	c.cs.s = append(c.cs.s, "IMPORTING")
	return (ClusterSetslotSubcommandImporting)(c)
}

func (c ClusterSetslotSlot) Migrating() ClusterSetslotSubcommandMigrating {
	c.cs.s = append(c.cs.s, "MIGRATING")
	return (ClusterSetslotSubcommandMigrating)(c)
}

func (c ClusterSetslotSlot) Stable() ClusterSetslotSubcommandStable {
	c.cs.s = append(c.cs.s, "STABLE")
	return (ClusterSetslotSubcommandStable)(c)
}

func (c ClusterSetslotSlot) Node() ClusterSetslotSubcommandNode {
	c.cs.s = append(c.cs.s, "NODE")
	return (ClusterSetslotSubcommandNode)(c)
}

type ClusterSetslotSubcommandImporting Completed

func (c ClusterSetslotSubcommandImporting) NodeId(nodeId string) ClusterSetslotNodeId {
	c.cs.s = append(c.cs.s, nodeId)
	return (ClusterSetslotNodeId)(c)
}

func (c ClusterSetslotSubcommandImporting) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type ClusterSetslotSubcommandMigrating Completed

func (c ClusterSetslotSubcommandMigrating) NodeId(nodeId string) ClusterSetslotNodeId {
	c.cs.s = append(c.cs.s, nodeId)
	return (ClusterSetslotNodeId)(c)
}

func (c ClusterSetslotSubcommandMigrating) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type ClusterSetslotSubcommandNode Completed

func (c ClusterSetslotSubcommandNode) NodeId(nodeId string) ClusterSetslotNodeId {
	c.cs.s = append(c.cs.s, nodeId)
	return (ClusterSetslotNodeId)(c)
}

func (c ClusterSetslotSubcommandNode) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type ClusterSetslotSubcommandStable Completed

func (c ClusterSetslotSubcommandStable) NodeId(nodeId string) ClusterSetslotNodeId {
	c.cs.s = append(c.cs.s, nodeId)
	return (ClusterSetslotNodeId)(c)
}

func (c ClusterSetslotSubcommandStable) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type ClusterShards Completed

func (b Builder) ClusterShards() (c ClusterShards) {
	c = ClusterShards{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "CLUSTER", "SHARDS")
	return c
}

func (c ClusterShards) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type ClusterSlaves Completed

func (b Builder) ClusterSlaves() (c ClusterSlaves) {
	c = ClusterSlaves{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "CLUSTER", "SLAVES")
	return c
}

func (c ClusterSlaves) NodeId(nodeId string) ClusterSlavesNodeId {
	c.cs.s = append(c.cs.s, nodeId)
	return (ClusterSlavesNodeId)(c)
}

type ClusterSlavesNodeId Completed

func (c ClusterSlavesNodeId) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type ClusterSlots Completed

func (b Builder) ClusterSlots() (c ClusterSlots) {
	c = ClusterSlots{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "CLUSTER", "SLOTS")
	return c
}

func (c ClusterSlots) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type Readonly Completed

func (b Builder) Readonly() (c Readonly) {
	c = Readonly{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "READONLY")
	return c
}

func (c Readonly) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type Readwrite Completed

func (b Builder) Readwrite() (c Readwrite) {
	c = Readwrite{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "READWRITE")
	return c
}

func (c Readwrite) Build() Completed {
	c.cs.Build()
	return Completed(c)
}
