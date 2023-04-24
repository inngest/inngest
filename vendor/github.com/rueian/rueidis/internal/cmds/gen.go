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

type AiModeldel Completed

func (b Builder) AiModeldel() (c AiModeldel) {
	c = AiModeldel{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "AI.MODELDEL")
	return c
}

func (c AiModeldel) Key(key string) AiModeldelKey {
	if c.ks&NoSlot == NoSlot {
		c.ks = NoSlot | slot(key)
	} else {
		c.ks = check(c.ks, slot(key))
	}
	c.cs.s = append(c.cs.s, key)
	return (AiModeldelKey)(c)
}

type AiModeldelKey Completed

func (c AiModeldelKey) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type AiModelexecute Completed

func (b Builder) AiModelexecute() (c AiModelexecute) {
	c = AiModelexecute{cs: get(), ks: b.ks, cf: readonly}
	c.cs.s = append(c.cs.s, "AI.MODELEXECUTE")
	return c
}

func (c AiModelexecute) Key(key string) AiModelexecuteKey {
	if c.ks&NoSlot == NoSlot {
		c.ks = NoSlot | slot(key)
	} else {
		c.ks = check(c.ks, slot(key))
	}
	c.cs.s = append(c.cs.s, key)
	return (AiModelexecuteKey)(c)
}

type AiModelexecuteInputsInput Completed

func (c AiModelexecuteInputsInput) Input(input ...string) AiModelexecuteInputsInput {
	c.cs.s = append(c.cs.s, input...)
	return c
}

func (c AiModelexecuteInputsInput) Outputs(outputCount int64) AiModelexecuteOutputsOutputs {
	c.cs.s = append(c.cs.s, "OUTPUTS", strconv.FormatInt(outputCount, 10))
	return (AiModelexecuteOutputsOutputs)(c)
}

type AiModelexecuteInputsInputs Completed

func (c AiModelexecuteInputsInputs) Input(input ...string) AiModelexecuteInputsInput {
	c.cs.s = append(c.cs.s, input...)
	return (AiModelexecuteInputsInput)(c)
}

type AiModelexecuteKey Completed

func (c AiModelexecuteKey) Inputs(inputCount int64) AiModelexecuteInputsInputs {
	c.cs.s = append(c.cs.s, "INPUTS", strconv.FormatInt(inputCount, 10))
	return (AiModelexecuteInputsInputs)(c)
}

type AiModelexecuteOutputsOutput Completed

func (c AiModelexecuteOutputsOutput) Output(output ...string) AiModelexecuteOutputsOutput {
	c.cs.s = append(c.cs.s, output...)
	return c
}

func (c AiModelexecuteOutputsOutput) Timeout(timeout int64) AiModelexecuteTimeout {
	c.cs.s = append(c.cs.s, "TIMEOUT", strconv.FormatInt(timeout, 10))
	return (AiModelexecuteTimeout)(c)
}

func (c AiModelexecuteOutputsOutput) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

func (c AiModelexecuteOutputsOutput) Cache() Cacheable {
	c.cs.Build()
	return Cacheable(c)
}

type AiModelexecuteOutputsOutputs Completed

func (c AiModelexecuteOutputsOutputs) Output(output ...string) AiModelexecuteOutputsOutput {
	c.cs.s = append(c.cs.s, output...)
	return (AiModelexecuteOutputsOutput)(c)
}

type AiModelexecuteTimeout Completed

func (c AiModelexecuteTimeout) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

func (c AiModelexecuteTimeout) Cache() Cacheable {
	c.cs.Build()
	return Cacheable(c)
}

type AiModelget Completed

func (b Builder) AiModelget() (c AiModelget) {
	c = AiModelget{cs: get(), ks: b.ks, cf: readonly}
	c.cs.s = append(c.cs.s, "AI.MODELGET")
	return c
}

func (c AiModelget) Key(key string) AiModelgetKey {
	if c.ks&NoSlot == NoSlot {
		c.ks = NoSlot | slot(key)
	} else {
		c.ks = check(c.ks, slot(key))
	}
	c.cs.s = append(c.cs.s, key)
	return (AiModelgetKey)(c)
}

type AiModelgetBlob Completed

func (c AiModelgetBlob) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

func (c AiModelgetBlob) Cache() Cacheable {
	c.cs.Build()
	return Cacheable(c)
}

type AiModelgetKey Completed

func (c AiModelgetKey) Meta() AiModelgetMeta {
	c.cs.s = append(c.cs.s, "META")
	return (AiModelgetMeta)(c)
}

func (c AiModelgetKey) Blob() AiModelgetBlob {
	c.cs.s = append(c.cs.s, "BLOB")
	return (AiModelgetBlob)(c)
}

func (c AiModelgetKey) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

func (c AiModelgetKey) Cache() Cacheable {
	c.cs.Build()
	return Cacheable(c)
}

type AiModelgetMeta Completed

func (c AiModelgetMeta) Blob() AiModelgetBlob {
	c.cs.s = append(c.cs.s, "BLOB")
	return (AiModelgetBlob)(c)
}

func (c AiModelgetMeta) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

func (c AiModelgetMeta) Cache() Cacheable {
	c.cs.Build()
	return Cacheable(c)
}

type AiModelstore Completed

func (b Builder) AiModelstore() (c AiModelstore) {
	c = AiModelstore{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "AI.MODELSTORE")
	return c
}

func (c AiModelstore) Key(key string) AiModelstoreKey {
	if c.ks&NoSlot == NoSlot {
		c.ks = NoSlot | slot(key)
	} else {
		c.ks = check(c.ks, slot(key))
	}
	c.cs.s = append(c.cs.s, key)
	return (AiModelstoreKey)(c)
}

type AiModelstoreBackendOnnx Completed

func (c AiModelstoreBackendOnnx) Cpu() AiModelstoreDeviceCpu {
	c.cs.s = append(c.cs.s, "CPU")
	return (AiModelstoreDeviceCpu)(c)
}

func (c AiModelstoreBackendOnnx) Gpu() AiModelstoreDeviceGpu {
	c.cs.s = append(c.cs.s, "GPU")
	return (AiModelstoreDeviceGpu)(c)
}

type AiModelstoreBackendTf Completed

func (c AiModelstoreBackendTf) Cpu() AiModelstoreDeviceCpu {
	c.cs.s = append(c.cs.s, "CPU")
	return (AiModelstoreDeviceCpu)(c)
}

func (c AiModelstoreBackendTf) Gpu() AiModelstoreDeviceGpu {
	c.cs.s = append(c.cs.s, "GPU")
	return (AiModelstoreDeviceGpu)(c)
}

type AiModelstoreBackendTorch Completed

func (c AiModelstoreBackendTorch) Cpu() AiModelstoreDeviceCpu {
	c.cs.s = append(c.cs.s, "CPU")
	return (AiModelstoreDeviceCpu)(c)
}

func (c AiModelstoreBackendTorch) Gpu() AiModelstoreDeviceGpu {
	c.cs.s = append(c.cs.s, "GPU")
	return (AiModelstoreDeviceGpu)(c)
}

type AiModelstoreBatchsize Completed

func (c AiModelstoreBatchsize) Minbatchsize(minbatchsize int64) AiModelstoreMinbatchsize {
	c.cs.s = append(c.cs.s, "MINBATCHSIZE", strconv.FormatInt(minbatchsize, 10))
	return (AiModelstoreMinbatchsize)(c)
}

func (c AiModelstoreBatchsize) Minbatchtimeout(minbatchtimeout int64) AiModelstoreMinbatchtimeout {
	c.cs.s = append(c.cs.s, "MINBATCHTIMEOUT", strconv.FormatInt(minbatchtimeout, 10))
	return (AiModelstoreMinbatchtimeout)(c)
}

func (c AiModelstoreBatchsize) Inputs(inputCount int64) AiModelstoreInputsInputs {
	c.cs.s = append(c.cs.s, "INPUTS", strconv.FormatInt(inputCount, 10))
	return (AiModelstoreInputsInputs)(c)
}

func (c AiModelstoreBatchsize) Outputs(outputCount int64) AiModelstoreOutputsOutputs {
	c.cs.s = append(c.cs.s, "OUTPUTS", strconv.FormatInt(outputCount, 10))
	return (AiModelstoreOutputsOutputs)(c)
}

func (c AiModelstoreBatchsize) Blob(blob string) AiModelstoreBlob {
	c.cs.s = append(c.cs.s, "BLOB", blob)
	return (AiModelstoreBlob)(c)
}

func (c AiModelstoreBatchsize) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type AiModelstoreBlob Completed

func (c AiModelstoreBlob) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type AiModelstoreDeviceCpu Completed

func (c AiModelstoreDeviceCpu) Tag(tag string) AiModelstoreTag {
	c.cs.s = append(c.cs.s, "TAG", tag)
	return (AiModelstoreTag)(c)
}

func (c AiModelstoreDeviceCpu) Batchsize(batchsize int64) AiModelstoreBatchsize {
	c.cs.s = append(c.cs.s, "BATCHSIZE", strconv.FormatInt(batchsize, 10))
	return (AiModelstoreBatchsize)(c)
}

func (c AiModelstoreDeviceCpu) Minbatchsize(minbatchsize int64) AiModelstoreMinbatchsize {
	c.cs.s = append(c.cs.s, "MINBATCHSIZE", strconv.FormatInt(minbatchsize, 10))
	return (AiModelstoreMinbatchsize)(c)
}

func (c AiModelstoreDeviceCpu) Minbatchtimeout(minbatchtimeout int64) AiModelstoreMinbatchtimeout {
	c.cs.s = append(c.cs.s, "MINBATCHTIMEOUT", strconv.FormatInt(minbatchtimeout, 10))
	return (AiModelstoreMinbatchtimeout)(c)
}

func (c AiModelstoreDeviceCpu) Inputs(inputCount int64) AiModelstoreInputsInputs {
	c.cs.s = append(c.cs.s, "INPUTS", strconv.FormatInt(inputCount, 10))
	return (AiModelstoreInputsInputs)(c)
}

func (c AiModelstoreDeviceCpu) Outputs(outputCount int64) AiModelstoreOutputsOutputs {
	c.cs.s = append(c.cs.s, "OUTPUTS", strconv.FormatInt(outputCount, 10))
	return (AiModelstoreOutputsOutputs)(c)
}

func (c AiModelstoreDeviceCpu) Blob(blob string) AiModelstoreBlob {
	c.cs.s = append(c.cs.s, "BLOB", blob)
	return (AiModelstoreBlob)(c)
}

func (c AiModelstoreDeviceCpu) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type AiModelstoreDeviceGpu Completed

func (c AiModelstoreDeviceGpu) Tag(tag string) AiModelstoreTag {
	c.cs.s = append(c.cs.s, "TAG", tag)
	return (AiModelstoreTag)(c)
}

func (c AiModelstoreDeviceGpu) Batchsize(batchsize int64) AiModelstoreBatchsize {
	c.cs.s = append(c.cs.s, "BATCHSIZE", strconv.FormatInt(batchsize, 10))
	return (AiModelstoreBatchsize)(c)
}

func (c AiModelstoreDeviceGpu) Minbatchsize(minbatchsize int64) AiModelstoreMinbatchsize {
	c.cs.s = append(c.cs.s, "MINBATCHSIZE", strconv.FormatInt(minbatchsize, 10))
	return (AiModelstoreMinbatchsize)(c)
}

func (c AiModelstoreDeviceGpu) Minbatchtimeout(minbatchtimeout int64) AiModelstoreMinbatchtimeout {
	c.cs.s = append(c.cs.s, "MINBATCHTIMEOUT", strconv.FormatInt(minbatchtimeout, 10))
	return (AiModelstoreMinbatchtimeout)(c)
}

func (c AiModelstoreDeviceGpu) Inputs(inputCount int64) AiModelstoreInputsInputs {
	c.cs.s = append(c.cs.s, "INPUTS", strconv.FormatInt(inputCount, 10))
	return (AiModelstoreInputsInputs)(c)
}

func (c AiModelstoreDeviceGpu) Outputs(outputCount int64) AiModelstoreOutputsOutputs {
	c.cs.s = append(c.cs.s, "OUTPUTS", strconv.FormatInt(outputCount, 10))
	return (AiModelstoreOutputsOutputs)(c)
}

func (c AiModelstoreDeviceGpu) Blob(blob string) AiModelstoreBlob {
	c.cs.s = append(c.cs.s, "BLOB", blob)
	return (AiModelstoreBlob)(c)
}

func (c AiModelstoreDeviceGpu) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type AiModelstoreInputsInput Completed

func (c AiModelstoreInputsInput) Input(input ...string) AiModelstoreInputsInput {
	c.cs.s = append(c.cs.s, input...)
	return c
}

func (c AiModelstoreInputsInput) Outputs(outputCount int64) AiModelstoreOutputsOutputs {
	c.cs.s = append(c.cs.s, "OUTPUTS", strconv.FormatInt(outputCount, 10))
	return (AiModelstoreOutputsOutputs)(c)
}

func (c AiModelstoreInputsInput) Blob(blob string) AiModelstoreBlob {
	c.cs.s = append(c.cs.s, "BLOB", blob)
	return (AiModelstoreBlob)(c)
}

func (c AiModelstoreInputsInput) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type AiModelstoreInputsInputs Completed

func (c AiModelstoreInputsInputs) Input(input ...string) AiModelstoreInputsInput {
	c.cs.s = append(c.cs.s, input...)
	return (AiModelstoreInputsInput)(c)
}

type AiModelstoreKey Completed

func (c AiModelstoreKey) Tf() AiModelstoreBackendTf {
	c.cs.s = append(c.cs.s, "TF")
	return (AiModelstoreBackendTf)(c)
}

func (c AiModelstoreKey) Torch() AiModelstoreBackendTorch {
	c.cs.s = append(c.cs.s, "TORCH")
	return (AiModelstoreBackendTorch)(c)
}

func (c AiModelstoreKey) Onnx() AiModelstoreBackendOnnx {
	c.cs.s = append(c.cs.s, "ONNX")
	return (AiModelstoreBackendOnnx)(c)
}

type AiModelstoreMinbatchsize Completed

func (c AiModelstoreMinbatchsize) Minbatchtimeout(minbatchtimeout int64) AiModelstoreMinbatchtimeout {
	c.cs.s = append(c.cs.s, "MINBATCHTIMEOUT", strconv.FormatInt(minbatchtimeout, 10))
	return (AiModelstoreMinbatchtimeout)(c)
}

func (c AiModelstoreMinbatchsize) Inputs(inputCount int64) AiModelstoreInputsInputs {
	c.cs.s = append(c.cs.s, "INPUTS", strconv.FormatInt(inputCount, 10))
	return (AiModelstoreInputsInputs)(c)
}

func (c AiModelstoreMinbatchsize) Outputs(outputCount int64) AiModelstoreOutputsOutputs {
	c.cs.s = append(c.cs.s, "OUTPUTS", strconv.FormatInt(outputCount, 10))
	return (AiModelstoreOutputsOutputs)(c)
}

func (c AiModelstoreMinbatchsize) Blob(blob string) AiModelstoreBlob {
	c.cs.s = append(c.cs.s, "BLOB", blob)
	return (AiModelstoreBlob)(c)
}

func (c AiModelstoreMinbatchsize) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type AiModelstoreMinbatchtimeout Completed

func (c AiModelstoreMinbatchtimeout) Inputs(inputCount int64) AiModelstoreInputsInputs {
	c.cs.s = append(c.cs.s, "INPUTS", strconv.FormatInt(inputCount, 10))
	return (AiModelstoreInputsInputs)(c)
}

func (c AiModelstoreMinbatchtimeout) Outputs(outputCount int64) AiModelstoreOutputsOutputs {
	c.cs.s = append(c.cs.s, "OUTPUTS", strconv.FormatInt(outputCount, 10))
	return (AiModelstoreOutputsOutputs)(c)
}

func (c AiModelstoreMinbatchtimeout) Blob(blob string) AiModelstoreBlob {
	c.cs.s = append(c.cs.s, "BLOB", blob)
	return (AiModelstoreBlob)(c)
}

func (c AiModelstoreMinbatchtimeout) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type AiModelstoreOutputsOutput Completed

func (c AiModelstoreOutputsOutput) Output(output ...string) AiModelstoreOutputsOutput {
	c.cs.s = append(c.cs.s, output...)
	return c
}

func (c AiModelstoreOutputsOutput) Blob(blob string) AiModelstoreBlob {
	c.cs.s = append(c.cs.s, "BLOB", blob)
	return (AiModelstoreBlob)(c)
}

func (c AiModelstoreOutputsOutput) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type AiModelstoreOutputsOutputs Completed

func (c AiModelstoreOutputsOutputs) Output(output ...string) AiModelstoreOutputsOutput {
	c.cs.s = append(c.cs.s, output...)
	return (AiModelstoreOutputsOutput)(c)
}

type AiModelstoreTag Completed

func (c AiModelstoreTag) Batchsize(batchsize int64) AiModelstoreBatchsize {
	c.cs.s = append(c.cs.s, "BATCHSIZE", strconv.FormatInt(batchsize, 10))
	return (AiModelstoreBatchsize)(c)
}

func (c AiModelstoreTag) Minbatchsize(minbatchsize int64) AiModelstoreMinbatchsize {
	c.cs.s = append(c.cs.s, "MINBATCHSIZE", strconv.FormatInt(minbatchsize, 10))
	return (AiModelstoreMinbatchsize)(c)
}

func (c AiModelstoreTag) Minbatchtimeout(minbatchtimeout int64) AiModelstoreMinbatchtimeout {
	c.cs.s = append(c.cs.s, "MINBATCHTIMEOUT", strconv.FormatInt(minbatchtimeout, 10))
	return (AiModelstoreMinbatchtimeout)(c)
}

func (c AiModelstoreTag) Inputs(inputCount int64) AiModelstoreInputsInputs {
	c.cs.s = append(c.cs.s, "INPUTS", strconv.FormatInt(inputCount, 10))
	return (AiModelstoreInputsInputs)(c)
}

func (c AiModelstoreTag) Outputs(outputCount int64) AiModelstoreOutputsOutputs {
	c.cs.s = append(c.cs.s, "OUTPUTS", strconv.FormatInt(outputCount, 10))
	return (AiModelstoreOutputsOutputs)(c)
}

func (c AiModelstoreTag) Blob(blob string) AiModelstoreBlob {
	c.cs.s = append(c.cs.s, "BLOB", blob)
	return (AiModelstoreBlob)(c)
}

func (c AiModelstoreTag) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

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

type AiScriptexecute Completed

func (b Builder) AiScriptexecute() (c AiScriptexecute) {
	c = AiScriptexecute{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "AI.SCRIPTEXECUTE")
	return c
}

func (c AiScriptexecute) Key(key string) AiScriptexecuteKey {
	if c.ks&NoSlot == NoSlot {
		c.ks = NoSlot | slot(key)
	} else {
		c.ks = check(c.ks, slot(key))
	}
	c.cs.s = append(c.cs.s, key)
	return (AiScriptexecuteKey)(c)
}

type AiScriptexecuteArgsArg Completed

func (c AiScriptexecuteArgsArg) Arg(arg ...string) AiScriptexecuteArgsArg {
	c.cs.s = append(c.cs.s, arg...)
	return c
}

func (c AiScriptexecuteArgsArg) Outputs(outputCount int64) AiScriptexecuteOutputsOutputs {
	c.cs.s = append(c.cs.s, "OUTPUTS", strconv.FormatInt(outputCount, 10))
	return (AiScriptexecuteOutputsOutputs)(c)
}

func (c AiScriptexecuteArgsArg) Timeout(timeout int64) AiScriptexecuteTimeout {
	c.cs.s = append(c.cs.s, "TIMEOUT", strconv.FormatInt(timeout, 10))
	return (AiScriptexecuteTimeout)(c)
}

func (c AiScriptexecuteArgsArg) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type AiScriptexecuteArgsArgs Completed

func (c AiScriptexecuteArgsArgs) Arg(arg ...string) AiScriptexecuteArgsArg {
	c.cs.s = append(c.cs.s, arg...)
	return (AiScriptexecuteArgsArg)(c)
}

type AiScriptexecuteFunction Completed

func (c AiScriptexecuteFunction) Keys(keyCount int64) AiScriptexecuteKeysKeys {
	c.cs.s = append(c.cs.s, "KEYS", strconv.FormatInt(keyCount, 10))
	return (AiScriptexecuteKeysKeys)(c)
}

func (c AiScriptexecuteFunction) Inputs(inputCount int64) AiScriptexecuteInputsInputs {
	c.cs.s = append(c.cs.s, "INPUTS", strconv.FormatInt(inputCount, 10))
	return (AiScriptexecuteInputsInputs)(c)
}

func (c AiScriptexecuteFunction) Args(argCount int64) AiScriptexecuteArgsArgs {
	c.cs.s = append(c.cs.s, "ARGS", strconv.FormatInt(argCount, 10))
	return (AiScriptexecuteArgsArgs)(c)
}

func (c AiScriptexecuteFunction) Outputs(outputCount int64) AiScriptexecuteOutputsOutputs {
	c.cs.s = append(c.cs.s, "OUTPUTS", strconv.FormatInt(outputCount, 10))
	return (AiScriptexecuteOutputsOutputs)(c)
}

func (c AiScriptexecuteFunction) Timeout(timeout int64) AiScriptexecuteTimeout {
	c.cs.s = append(c.cs.s, "TIMEOUT", strconv.FormatInt(timeout, 10))
	return (AiScriptexecuteTimeout)(c)
}

func (c AiScriptexecuteFunction) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type AiScriptexecuteInputsInput Completed

func (c AiScriptexecuteInputsInput) Input(input ...string) AiScriptexecuteInputsInput {
	c.cs.s = append(c.cs.s, input...)
	return c
}

func (c AiScriptexecuteInputsInput) Args(argCount int64) AiScriptexecuteArgsArgs {
	c.cs.s = append(c.cs.s, "ARGS", strconv.FormatInt(argCount, 10))
	return (AiScriptexecuteArgsArgs)(c)
}

func (c AiScriptexecuteInputsInput) Outputs(outputCount int64) AiScriptexecuteOutputsOutputs {
	c.cs.s = append(c.cs.s, "OUTPUTS", strconv.FormatInt(outputCount, 10))
	return (AiScriptexecuteOutputsOutputs)(c)
}

func (c AiScriptexecuteInputsInput) Timeout(timeout int64) AiScriptexecuteTimeout {
	c.cs.s = append(c.cs.s, "TIMEOUT", strconv.FormatInt(timeout, 10))
	return (AiScriptexecuteTimeout)(c)
}

func (c AiScriptexecuteInputsInput) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type AiScriptexecuteInputsInputs Completed

func (c AiScriptexecuteInputsInputs) Input(input ...string) AiScriptexecuteInputsInput {
	c.cs.s = append(c.cs.s, input...)
	return (AiScriptexecuteInputsInput)(c)
}

type AiScriptexecuteKey Completed

func (c AiScriptexecuteKey) Function(function string) AiScriptexecuteFunction {
	c.cs.s = append(c.cs.s, function)
	return (AiScriptexecuteFunction)(c)
}

type AiScriptexecuteKeysKey Completed

func (c AiScriptexecuteKeysKey) Key(key ...string) AiScriptexecuteKeysKey {
	c.cs.s = append(c.cs.s, key...)
	return c
}

func (c AiScriptexecuteKeysKey) Inputs(inputCount int64) AiScriptexecuteInputsInputs {
	c.cs.s = append(c.cs.s, "INPUTS", strconv.FormatInt(inputCount, 10))
	return (AiScriptexecuteInputsInputs)(c)
}

func (c AiScriptexecuteKeysKey) Args(argCount int64) AiScriptexecuteArgsArgs {
	c.cs.s = append(c.cs.s, "ARGS", strconv.FormatInt(argCount, 10))
	return (AiScriptexecuteArgsArgs)(c)
}

func (c AiScriptexecuteKeysKey) Outputs(outputCount int64) AiScriptexecuteOutputsOutputs {
	c.cs.s = append(c.cs.s, "OUTPUTS", strconv.FormatInt(outputCount, 10))
	return (AiScriptexecuteOutputsOutputs)(c)
}

func (c AiScriptexecuteKeysKey) Timeout(timeout int64) AiScriptexecuteTimeout {
	c.cs.s = append(c.cs.s, "TIMEOUT", strconv.FormatInt(timeout, 10))
	return (AiScriptexecuteTimeout)(c)
}

func (c AiScriptexecuteKeysKey) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type AiScriptexecuteKeysKeys Completed

func (c AiScriptexecuteKeysKeys) Key(key ...string) AiScriptexecuteKeysKey {
	c.cs.s = append(c.cs.s, key...)
	return (AiScriptexecuteKeysKey)(c)
}

type AiScriptexecuteOutputsOutput Completed

func (c AiScriptexecuteOutputsOutput) Output(output ...string) AiScriptexecuteOutputsOutput {
	c.cs.s = append(c.cs.s, output...)
	return c
}

func (c AiScriptexecuteOutputsOutput) Timeout(timeout int64) AiScriptexecuteTimeout {
	c.cs.s = append(c.cs.s, "TIMEOUT", strconv.FormatInt(timeout, 10))
	return (AiScriptexecuteTimeout)(c)
}

func (c AiScriptexecuteOutputsOutput) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type AiScriptexecuteOutputsOutputs Completed

func (c AiScriptexecuteOutputsOutputs) Output(output ...string) AiScriptexecuteOutputsOutput {
	c.cs.s = append(c.cs.s, output...)
	return (AiScriptexecuteOutputsOutput)(c)
}

type AiScriptexecuteTimeout Completed

func (c AiScriptexecuteTimeout) Build() Completed {
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

type Append Completed

func (b Builder) Append() (c Append) {
	c = Append{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "APPEND")
	return c
}

func (c Append) Key(key string) AppendKey {
	if c.ks&NoSlot == NoSlot {
		c.ks = NoSlot | slot(key)
	} else {
		c.ks = check(c.ks, slot(key))
	}
	c.cs.s = append(c.cs.s, key)
	return (AppendKey)(c)
}

type AppendKey Completed

func (c AppendKey) Value(value string) AppendValue {
	c.cs.s = append(c.cs.s, value)
	return (AppendValue)(c)
}

type AppendValue Completed

func (c AppendValue) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

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

type Auth Completed

func (b Builder) Auth() (c Auth) {
	c = Auth{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "AUTH")
	return c
}

func (c Auth) Username(username string) AuthUsername {
	c.cs.s = append(c.cs.s, username)
	return (AuthUsername)(c)
}

func (c Auth) Password(password string) AuthPassword {
	c.cs.s = append(c.cs.s, password)
	return (AuthPassword)(c)
}

type AuthPassword Completed

func (c AuthPassword) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type AuthUsername Completed

func (c AuthUsername) Password(password string) AuthPassword {
	c.cs.s = append(c.cs.s, password)
	return (AuthPassword)(c)
}

type BfAdd Completed

func (b Builder) BfAdd() (c BfAdd) {
	c = BfAdd{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "BF.ADD")
	return c
}

func (c BfAdd) Key(key string) BfAddKey {
	if c.ks&NoSlot == NoSlot {
		c.ks = NoSlot | slot(key)
	} else {
		c.ks = check(c.ks, slot(key))
	}
	c.cs.s = append(c.cs.s, key)
	return (BfAddKey)(c)
}

type BfAddItem Completed

func (c BfAddItem) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type BfAddKey Completed

func (c BfAddKey) Item(item string) BfAddItem {
	c.cs.s = append(c.cs.s, item)
	return (BfAddItem)(c)
}

type BfCard Completed

func (b Builder) BfCard() (c BfCard) {
	c = BfCard{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "BF.CARD")
	return c
}

func (c BfCard) Key(key string) BfCardKey {
	if c.ks&NoSlot == NoSlot {
		c.ks = NoSlot | slot(key)
	} else {
		c.ks = check(c.ks, slot(key))
	}
	c.cs.s = append(c.cs.s, key)
	return (BfCardKey)(c)
}

type BfCardKey Completed

func (c BfCardKey) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type BfExists Completed

func (b Builder) BfExists() (c BfExists) {
	c = BfExists{cs: get(), ks: b.ks, cf: readonly}
	c.cs.s = append(c.cs.s, "BF.EXISTS")
	return c
}

func (c BfExists) Key(key string) BfExistsKey {
	if c.ks&NoSlot == NoSlot {
		c.ks = NoSlot | slot(key)
	} else {
		c.ks = check(c.ks, slot(key))
	}
	c.cs.s = append(c.cs.s, key)
	return (BfExistsKey)(c)
}

type BfExistsItem Completed

func (c BfExistsItem) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

func (c BfExistsItem) Cache() Cacheable {
	c.cs.Build()
	return Cacheable(c)
}

type BfExistsKey Completed

func (c BfExistsKey) Item(item string) BfExistsItem {
	c.cs.s = append(c.cs.s, item)
	return (BfExistsItem)(c)
}

type BfInfo Completed

func (b Builder) BfInfo() (c BfInfo) {
	c = BfInfo{cs: get(), ks: b.ks, cf: readonly}
	c.cs.s = append(c.cs.s, "BF.INFO")
	return c
}

func (c BfInfo) Key(key string) BfInfoKey {
	if c.ks&NoSlot == NoSlot {
		c.ks = NoSlot | slot(key)
	} else {
		c.ks = check(c.ks, slot(key))
	}
	c.cs.s = append(c.cs.s, key)
	return (BfInfoKey)(c)
}

type BfInfoKey Completed

func (c BfInfoKey) Capacity() BfInfoSingleValueCapacity {
	c.cs.s = append(c.cs.s, "CAPACITY")
	return (BfInfoSingleValueCapacity)(c)
}

func (c BfInfoKey) Size() BfInfoSingleValueSize {
	c.cs.s = append(c.cs.s, "SIZE")
	return (BfInfoSingleValueSize)(c)
}

func (c BfInfoKey) Filters() BfInfoSingleValueFilters {
	c.cs.s = append(c.cs.s, "FILTERS")
	return (BfInfoSingleValueFilters)(c)
}

func (c BfInfoKey) Items() BfInfoSingleValueItems {
	c.cs.s = append(c.cs.s, "ITEMS")
	return (BfInfoSingleValueItems)(c)
}

func (c BfInfoKey) Expansion() BfInfoSingleValueExpansion {
	c.cs.s = append(c.cs.s, "EXPANSION")
	return (BfInfoSingleValueExpansion)(c)
}

func (c BfInfoKey) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

func (c BfInfoKey) Cache() Cacheable {
	c.cs.Build()
	return Cacheable(c)
}

type BfInfoSingleValueCapacity Completed

func (c BfInfoSingleValueCapacity) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

func (c BfInfoSingleValueCapacity) Cache() Cacheable {
	c.cs.Build()
	return Cacheable(c)
}

type BfInfoSingleValueExpansion Completed

func (c BfInfoSingleValueExpansion) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

func (c BfInfoSingleValueExpansion) Cache() Cacheable {
	c.cs.Build()
	return Cacheable(c)
}

type BfInfoSingleValueFilters Completed

func (c BfInfoSingleValueFilters) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

func (c BfInfoSingleValueFilters) Cache() Cacheable {
	c.cs.Build()
	return Cacheable(c)
}

type BfInfoSingleValueItems Completed

func (c BfInfoSingleValueItems) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

func (c BfInfoSingleValueItems) Cache() Cacheable {
	c.cs.Build()
	return Cacheable(c)
}

type BfInfoSingleValueSize Completed

func (c BfInfoSingleValueSize) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

func (c BfInfoSingleValueSize) Cache() Cacheable {
	c.cs.Build()
	return Cacheable(c)
}

type BfInsert Completed

func (b Builder) BfInsert() (c BfInsert) {
	c = BfInsert{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "BF.INSERT")
	return c
}

func (c BfInsert) Key(key string) BfInsertKey {
	if c.ks&NoSlot == NoSlot {
		c.ks = NoSlot | slot(key)
	} else {
		c.ks = check(c.ks, slot(key))
	}
	c.cs.s = append(c.cs.s, key)
	return (BfInsertKey)(c)
}

type BfInsertCapacity Completed

func (c BfInsertCapacity) Error(error float64) BfInsertError {
	c.cs.s = append(c.cs.s, "ERROR", strconv.FormatFloat(error, 'f', -1, 64))
	return (BfInsertError)(c)
}

func (c BfInsertCapacity) Expansion(expansion int64) BfInsertExpansion {
	c.cs.s = append(c.cs.s, "EXPANSION", strconv.FormatInt(expansion, 10))
	return (BfInsertExpansion)(c)
}

func (c BfInsertCapacity) Nocreate() BfInsertNocreate {
	c.cs.s = append(c.cs.s, "NOCREATE")
	return (BfInsertNocreate)(c)
}

func (c BfInsertCapacity) Nonscaling() BfInsertNonscaling {
	c.cs.s = append(c.cs.s, "NONSCALING")
	return (BfInsertNonscaling)(c)
}

func (c BfInsertCapacity) Items() BfInsertItems {
	c.cs.s = append(c.cs.s, "ITEMS")
	return (BfInsertItems)(c)
}

type BfInsertError Completed

func (c BfInsertError) Expansion(expansion int64) BfInsertExpansion {
	c.cs.s = append(c.cs.s, "EXPANSION", strconv.FormatInt(expansion, 10))
	return (BfInsertExpansion)(c)
}

func (c BfInsertError) Nocreate() BfInsertNocreate {
	c.cs.s = append(c.cs.s, "NOCREATE")
	return (BfInsertNocreate)(c)
}

func (c BfInsertError) Nonscaling() BfInsertNonscaling {
	c.cs.s = append(c.cs.s, "NONSCALING")
	return (BfInsertNonscaling)(c)
}

func (c BfInsertError) Items() BfInsertItems {
	c.cs.s = append(c.cs.s, "ITEMS")
	return (BfInsertItems)(c)
}

type BfInsertExpansion Completed

func (c BfInsertExpansion) Nocreate() BfInsertNocreate {
	c.cs.s = append(c.cs.s, "NOCREATE")
	return (BfInsertNocreate)(c)
}

func (c BfInsertExpansion) Nonscaling() BfInsertNonscaling {
	c.cs.s = append(c.cs.s, "NONSCALING")
	return (BfInsertNonscaling)(c)
}

func (c BfInsertExpansion) Items() BfInsertItems {
	c.cs.s = append(c.cs.s, "ITEMS")
	return (BfInsertItems)(c)
}

type BfInsertItem Completed

func (c BfInsertItem) Item(item ...string) BfInsertItem {
	c.cs.s = append(c.cs.s, item...)
	return c
}

func (c BfInsertItem) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type BfInsertItems Completed

func (c BfInsertItems) Item(item ...string) BfInsertItem {
	c.cs.s = append(c.cs.s, item...)
	return (BfInsertItem)(c)
}

type BfInsertKey Completed

func (c BfInsertKey) Capacity(capacity int64) BfInsertCapacity {
	c.cs.s = append(c.cs.s, "CAPACITY", strconv.FormatInt(capacity, 10))
	return (BfInsertCapacity)(c)
}

func (c BfInsertKey) Error(error float64) BfInsertError {
	c.cs.s = append(c.cs.s, "ERROR", strconv.FormatFloat(error, 'f', -1, 64))
	return (BfInsertError)(c)
}

func (c BfInsertKey) Expansion(expansion int64) BfInsertExpansion {
	c.cs.s = append(c.cs.s, "EXPANSION", strconv.FormatInt(expansion, 10))
	return (BfInsertExpansion)(c)
}

func (c BfInsertKey) Nocreate() BfInsertNocreate {
	c.cs.s = append(c.cs.s, "NOCREATE")
	return (BfInsertNocreate)(c)
}

func (c BfInsertKey) Nonscaling() BfInsertNonscaling {
	c.cs.s = append(c.cs.s, "NONSCALING")
	return (BfInsertNonscaling)(c)
}

func (c BfInsertKey) Items() BfInsertItems {
	c.cs.s = append(c.cs.s, "ITEMS")
	return (BfInsertItems)(c)
}

type BfInsertNocreate Completed

func (c BfInsertNocreate) Nonscaling() BfInsertNonscaling {
	c.cs.s = append(c.cs.s, "NONSCALING")
	return (BfInsertNonscaling)(c)
}

func (c BfInsertNocreate) Items() BfInsertItems {
	c.cs.s = append(c.cs.s, "ITEMS")
	return (BfInsertItems)(c)
}

type BfInsertNonscaling Completed

func (c BfInsertNonscaling) Items() BfInsertItems {
	c.cs.s = append(c.cs.s, "ITEMS")
	return (BfInsertItems)(c)
}

type BfLoadchunk Completed

func (b Builder) BfLoadchunk() (c BfLoadchunk) {
	c = BfLoadchunk{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "BF.LOADCHUNK")
	return c
}

func (c BfLoadchunk) Key(key string) BfLoadchunkKey {
	if c.ks&NoSlot == NoSlot {
		c.ks = NoSlot | slot(key)
	} else {
		c.ks = check(c.ks, slot(key))
	}
	c.cs.s = append(c.cs.s, key)
	return (BfLoadchunkKey)(c)
}

type BfLoadchunkData Completed

func (c BfLoadchunkData) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type BfLoadchunkIterator Completed

func (c BfLoadchunkIterator) Data(data string) BfLoadchunkData {
	c.cs.s = append(c.cs.s, data)
	return (BfLoadchunkData)(c)
}

type BfLoadchunkKey Completed

func (c BfLoadchunkKey) Iterator(iterator int64) BfLoadchunkIterator {
	c.cs.s = append(c.cs.s, strconv.FormatInt(iterator, 10))
	return (BfLoadchunkIterator)(c)
}

type BfMadd Completed

func (b Builder) BfMadd() (c BfMadd) {
	c = BfMadd{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "BF.MADD")
	return c
}

func (c BfMadd) Key(key string) BfMaddKey {
	if c.ks&NoSlot == NoSlot {
		c.ks = NoSlot | slot(key)
	} else {
		c.ks = check(c.ks, slot(key))
	}
	c.cs.s = append(c.cs.s, key)
	return (BfMaddKey)(c)
}

type BfMaddItem Completed

func (c BfMaddItem) Item(item ...string) BfMaddItem {
	c.cs.s = append(c.cs.s, item...)
	return c
}

func (c BfMaddItem) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type BfMaddKey Completed

func (c BfMaddKey) Item(item ...string) BfMaddItem {
	c.cs.s = append(c.cs.s, item...)
	return (BfMaddItem)(c)
}

type BfMexists Completed

func (b Builder) BfMexists() (c BfMexists) {
	c = BfMexists{cs: get(), ks: b.ks, cf: readonly}
	c.cs.s = append(c.cs.s, "BF.MEXISTS")
	return c
}

func (c BfMexists) Key(key string) BfMexistsKey {
	if c.ks&NoSlot == NoSlot {
		c.ks = NoSlot | slot(key)
	} else {
		c.ks = check(c.ks, slot(key))
	}
	c.cs.s = append(c.cs.s, key)
	return (BfMexistsKey)(c)
}

type BfMexistsItem Completed

func (c BfMexistsItem) Item(item ...string) BfMexistsItem {
	c.cs.s = append(c.cs.s, item...)
	return c
}

func (c BfMexistsItem) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type BfMexistsKey Completed

func (c BfMexistsKey) Item(item ...string) BfMexistsItem {
	c.cs.s = append(c.cs.s, item...)
	return (BfMexistsItem)(c)
}

type BfReserve Completed

func (b Builder) BfReserve() (c BfReserve) {
	c = BfReserve{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "BF.RESERVE")
	return c
}

func (c BfReserve) Key(key string) BfReserveKey {
	if c.ks&NoSlot == NoSlot {
		c.ks = NoSlot | slot(key)
	} else {
		c.ks = check(c.ks, slot(key))
	}
	c.cs.s = append(c.cs.s, key)
	return (BfReserveKey)(c)
}

type BfReserveCapacity Completed

func (c BfReserveCapacity) Expansion(expansion int64) BfReserveExpansion {
	c.cs.s = append(c.cs.s, "EXPANSION", strconv.FormatInt(expansion, 10))
	return (BfReserveExpansion)(c)
}

func (c BfReserveCapacity) Nonscaling() BfReserveNonscaling {
	c.cs.s = append(c.cs.s, "NONSCALING")
	return (BfReserveNonscaling)(c)
}

func (c BfReserveCapacity) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type BfReserveErrorRate Completed

func (c BfReserveErrorRate) Capacity(capacity int64) BfReserveCapacity {
	c.cs.s = append(c.cs.s, strconv.FormatInt(capacity, 10))
	return (BfReserveCapacity)(c)
}

type BfReserveExpansion Completed

func (c BfReserveExpansion) Nonscaling() BfReserveNonscaling {
	c.cs.s = append(c.cs.s, "NONSCALING")
	return (BfReserveNonscaling)(c)
}

func (c BfReserveExpansion) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type BfReserveKey Completed

func (c BfReserveKey) ErrorRate(errorRate float64) BfReserveErrorRate {
	c.cs.s = append(c.cs.s, strconv.FormatFloat(errorRate, 'f', -1, 64))
	return (BfReserveErrorRate)(c)
}

type BfReserveNonscaling Completed

func (c BfReserveNonscaling) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type BfScandump Completed

func (b Builder) BfScandump() (c BfScandump) {
	c = BfScandump{cs: get(), ks: b.ks, cf: readonly}
	c.cs.s = append(c.cs.s, "BF.SCANDUMP")
	return c
}

func (c BfScandump) Key(key string) BfScandumpKey {
	if c.ks&NoSlot == NoSlot {
		c.ks = NoSlot | slot(key)
	} else {
		c.ks = check(c.ks, slot(key))
	}
	c.cs.s = append(c.cs.s, key)
	return (BfScandumpKey)(c)
}

type BfScandumpIterator Completed

func (c BfScandumpIterator) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type BfScandumpKey Completed

func (c BfScandumpKey) Iterator(iterator int64) BfScandumpIterator {
	c.cs.s = append(c.cs.s, strconv.FormatInt(iterator, 10))
	return (BfScandumpIterator)(c)
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

type Bitcount Completed

func (b Builder) Bitcount() (c Bitcount) {
	c = Bitcount{cs: get(), ks: b.ks, cf: readonly}
	c.cs.s = append(c.cs.s, "BITCOUNT")
	return c
}

func (c Bitcount) Key(key string) BitcountKey {
	if c.ks&NoSlot == NoSlot {
		c.ks = NoSlot | slot(key)
	} else {
		c.ks = check(c.ks, slot(key))
	}
	c.cs.s = append(c.cs.s, key)
	return (BitcountKey)(c)
}

type BitcountIndexEnd Completed

func (c BitcountIndexEnd) Byte() BitcountIndexIndexUnitByte {
	c.cs.s = append(c.cs.s, "BYTE")
	return (BitcountIndexIndexUnitByte)(c)
}

func (c BitcountIndexEnd) Bit() BitcountIndexIndexUnitBit {
	c.cs.s = append(c.cs.s, "BIT")
	return (BitcountIndexIndexUnitBit)(c)
}

func (c BitcountIndexEnd) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

func (c BitcountIndexEnd) Cache() Cacheable {
	c.cs.Build()
	return Cacheable(c)
}

type BitcountIndexIndexUnitBit Completed

func (c BitcountIndexIndexUnitBit) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

func (c BitcountIndexIndexUnitBit) Cache() Cacheable {
	c.cs.Build()
	return Cacheable(c)
}

type BitcountIndexIndexUnitByte Completed

func (c BitcountIndexIndexUnitByte) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

func (c BitcountIndexIndexUnitByte) Cache() Cacheable {
	c.cs.Build()
	return Cacheable(c)
}

type BitcountIndexStart Completed

func (c BitcountIndexStart) End(end int64) BitcountIndexEnd {
	c.cs.s = append(c.cs.s, strconv.FormatInt(end, 10))
	return (BitcountIndexEnd)(c)
}

type BitcountKey Completed

func (c BitcountKey) Start(start int64) BitcountIndexStart {
	c.cs.s = append(c.cs.s, strconv.FormatInt(start, 10))
	return (BitcountIndexStart)(c)
}

func (c BitcountKey) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

func (c BitcountKey) Cache() Cacheable {
	c.cs.Build()
	return Cacheable(c)
}

type Bitfield Completed

func (b Builder) Bitfield() (c Bitfield) {
	c = Bitfield{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "BITFIELD")
	return c
}

func (c Bitfield) Key(key string) BitfieldKey {
	if c.ks&NoSlot == NoSlot {
		c.ks = NoSlot | slot(key)
	} else {
		c.ks = check(c.ks, slot(key))
	}
	c.cs.s = append(c.cs.s, key)
	return (BitfieldKey)(c)
}

type BitfieldKey Completed

func (c BitfieldKey) Get(encoding string, offset int64) BitfieldOperationGet {
	c.cs.s = append(c.cs.s, "GET", encoding, strconv.FormatInt(offset, 10))
	return (BitfieldOperationGet)(c)
}

func (c BitfieldKey) OverflowWrap() BitfieldOperationWriteOverflowWrap {
	c.cs.s = append(c.cs.s, "OVERFLOW", "WRAP")
	return (BitfieldOperationWriteOverflowWrap)(c)
}

func (c BitfieldKey) OverflowSat() BitfieldOperationWriteOverflowSat {
	c.cs.s = append(c.cs.s, "OVERFLOW", "SAT")
	return (BitfieldOperationWriteOverflowSat)(c)
}

func (c BitfieldKey) OverflowFail() BitfieldOperationWriteOverflowFail {
	c.cs.s = append(c.cs.s, "OVERFLOW", "FAIL")
	return (BitfieldOperationWriteOverflowFail)(c)
}

func (c BitfieldKey) Set(encoding string, offset int64, value int64) BitfieldOperationWriteSetSet {
	c.cs.s = append(c.cs.s, "SET", encoding, strconv.FormatInt(offset, 10), strconv.FormatInt(value, 10))
	return (BitfieldOperationWriteSetSet)(c)
}

func (c BitfieldKey) Incrby(encoding string, offset int64, increment int64) BitfieldOperationWriteSetIncrby {
	c.cs.s = append(c.cs.s, "INCRBY", encoding, strconv.FormatInt(offset, 10), strconv.FormatInt(increment, 10))
	return (BitfieldOperationWriteSetIncrby)(c)
}

func (c BitfieldKey) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type BitfieldOperationGet Completed

func (c BitfieldOperationGet) OverflowWrap() BitfieldOperationWriteOverflowWrap {
	c.cs.s = append(c.cs.s, "OVERFLOW", "WRAP")
	return (BitfieldOperationWriteOverflowWrap)(c)
}

func (c BitfieldOperationGet) OverflowSat() BitfieldOperationWriteOverflowSat {
	c.cs.s = append(c.cs.s, "OVERFLOW", "SAT")
	return (BitfieldOperationWriteOverflowSat)(c)
}

func (c BitfieldOperationGet) OverflowFail() BitfieldOperationWriteOverflowFail {
	c.cs.s = append(c.cs.s, "OVERFLOW", "FAIL")
	return (BitfieldOperationWriteOverflowFail)(c)
}

func (c BitfieldOperationGet) Set(encoding string, offset int64, value int64) BitfieldOperationWriteSetSet {
	c.cs.s = append(c.cs.s, "SET", encoding, strconv.FormatInt(offset, 10), strconv.FormatInt(value, 10))
	return (BitfieldOperationWriteSetSet)(c)
}

func (c BitfieldOperationGet) Incrby(encoding string, offset int64, increment int64) BitfieldOperationWriteSetIncrby {
	c.cs.s = append(c.cs.s, "INCRBY", encoding, strconv.FormatInt(offset, 10), strconv.FormatInt(increment, 10))
	return (BitfieldOperationWriteSetIncrby)(c)
}

type BitfieldOperationWriteOverflowFail Completed

func (c BitfieldOperationWriteOverflowFail) Set(encoding string, offset int64, value int64) BitfieldOperationWriteSetSet {
	c.cs.s = append(c.cs.s, "SET", encoding, strconv.FormatInt(offset, 10), strconv.FormatInt(value, 10))
	return (BitfieldOperationWriteSetSet)(c)
}

func (c BitfieldOperationWriteOverflowFail) Incrby(encoding string, offset int64, increment int64) BitfieldOperationWriteSetIncrby {
	c.cs.s = append(c.cs.s, "INCRBY", encoding, strconv.FormatInt(offset, 10), strconv.FormatInt(increment, 10))
	return (BitfieldOperationWriteSetIncrby)(c)
}

type BitfieldOperationWriteOverflowSat Completed

func (c BitfieldOperationWriteOverflowSat) Set(encoding string, offset int64, value int64) BitfieldOperationWriteSetSet {
	c.cs.s = append(c.cs.s, "SET", encoding, strconv.FormatInt(offset, 10), strconv.FormatInt(value, 10))
	return (BitfieldOperationWriteSetSet)(c)
}

func (c BitfieldOperationWriteOverflowSat) Incrby(encoding string, offset int64, increment int64) BitfieldOperationWriteSetIncrby {
	c.cs.s = append(c.cs.s, "INCRBY", encoding, strconv.FormatInt(offset, 10), strconv.FormatInt(increment, 10))
	return (BitfieldOperationWriteSetIncrby)(c)
}

type BitfieldOperationWriteOverflowWrap Completed

func (c BitfieldOperationWriteOverflowWrap) Set(encoding string, offset int64, value int64) BitfieldOperationWriteSetSet {
	c.cs.s = append(c.cs.s, "SET", encoding, strconv.FormatInt(offset, 10), strconv.FormatInt(value, 10))
	return (BitfieldOperationWriteSetSet)(c)
}

func (c BitfieldOperationWriteOverflowWrap) Incrby(encoding string, offset int64, increment int64) BitfieldOperationWriteSetIncrby {
	c.cs.s = append(c.cs.s, "INCRBY", encoding, strconv.FormatInt(offset, 10), strconv.FormatInt(increment, 10))
	return (BitfieldOperationWriteSetIncrby)(c)
}

type BitfieldOperationWriteSetIncrby Completed

func (c BitfieldOperationWriteSetIncrby) Get(encoding string, offset int64) BitfieldOperationGet {
	c.cs.s = append(c.cs.s, "GET", encoding, strconv.FormatInt(offset, 10))
	return (BitfieldOperationGet)(c)
}

func (c BitfieldOperationWriteSetIncrby) OverflowWrap() BitfieldOperationWriteOverflowWrap {
	c.cs.s = append(c.cs.s, "OVERFLOW", "WRAP")
	return (BitfieldOperationWriteOverflowWrap)(c)
}

func (c BitfieldOperationWriteSetIncrby) OverflowSat() BitfieldOperationWriteOverflowSat {
	c.cs.s = append(c.cs.s, "OVERFLOW", "SAT")
	return (BitfieldOperationWriteOverflowSat)(c)
}

func (c BitfieldOperationWriteSetIncrby) OverflowFail() BitfieldOperationWriteOverflowFail {
	c.cs.s = append(c.cs.s, "OVERFLOW", "FAIL")
	return (BitfieldOperationWriteOverflowFail)(c)
}

func (c BitfieldOperationWriteSetIncrby) Set(encoding string, offset int64, value int64) BitfieldOperationWriteSetSet {
	c.cs.s = append(c.cs.s, "SET", encoding, strconv.FormatInt(offset, 10), strconv.FormatInt(value, 10))
	return (BitfieldOperationWriteSetSet)(c)
}

func (c BitfieldOperationWriteSetIncrby) Incrby(encoding string, offset int64, increment int64) BitfieldOperationWriteSetIncrby {
	c.cs.s = append(c.cs.s, "INCRBY", encoding, strconv.FormatInt(offset, 10), strconv.FormatInt(increment, 10))
	return c
}

func (c BitfieldOperationWriteSetIncrby) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type BitfieldOperationWriteSetSet Completed

func (c BitfieldOperationWriteSetSet) Incrby(encoding string, offset int64, increment int64) BitfieldOperationWriteSetIncrby {
	c.cs.s = append(c.cs.s, "INCRBY", encoding, strconv.FormatInt(offset, 10), strconv.FormatInt(increment, 10))
	return (BitfieldOperationWriteSetIncrby)(c)
}

func (c BitfieldOperationWriteSetSet) Get(encoding string, offset int64) BitfieldOperationGet {
	c.cs.s = append(c.cs.s, "GET", encoding, strconv.FormatInt(offset, 10))
	return (BitfieldOperationGet)(c)
}

func (c BitfieldOperationWriteSetSet) OverflowWrap() BitfieldOperationWriteOverflowWrap {
	c.cs.s = append(c.cs.s, "OVERFLOW", "WRAP")
	return (BitfieldOperationWriteOverflowWrap)(c)
}

func (c BitfieldOperationWriteSetSet) OverflowSat() BitfieldOperationWriteOverflowSat {
	c.cs.s = append(c.cs.s, "OVERFLOW", "SAT")
	return (BitfieldOperationWriteOverflowSat)(c)
}

func (c BitfieldOperationWriteSetSet) OverflowFail() BitfieldOperationWriteOverflowFail {
	c.cs.s = append(c.cs.s, "OVERFLOW", "FAIL")
	return (BitfieldOperationWriteOverflowFail)(c)
}

func (c BitfieldOperationWriteSetSet) Set(encoding string, offset int64, value int64) BitfieldOperationWriteSetSet {
	c.cs.s = append(c.cs.s, "SET", encoding, strconv.FormatInt(offset, 10), strconv.FormatInt(value, 10))
	return c
}

func (c BitfieldOperationWriteSetSet) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type BitfieldRo Completed

func (b Builder) BitfieldRo() (c BitfieldRo) {
	c = BitfieldRo{cs: get(), ks: b.ks, cf: readonly}
	c.cs.s = append(c.cs.s, "BITFIELD_RO")
	return c
}

func (c BitfieldRo) Key(key string) BitfieldRoKey {
	if c.ks&NoSlot == NoSlot {
		c.ks = NoSlot | slot(key)
	} else {
		c.ks = check(c.ks, slot(key))
	}
	c.cs.s = append(c.cs.s, key)
	return (BitfieldRoKey)(c)
}

type BitfieldRoGet Completed

func (c BitfieldRoGet) Get(encoding string, offset int64) BitfieldRoGet {
	c.cs.s = append(c.cs.s, "GET", encoding, strconv.FormatInt(offset, 10))
	return c
}

func (c BitfieldRoGet) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

func (c BitfieldRoGet) Cache() Cacheable {
	c.cs.Build()
	return Cacheable(c)
}

type BitfieldRoKey Completed

func (c BitfieldRoKey) Get() BitfieldRoGet {
	return (BitfieldRoGet)(c)
}

func (c BitfieldRoKey) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

func (c BitfieldRoKey) Cache() Cacheable {
	c.cs.Build()
	return Cacheable(c)
}

type Bitop Completed

func (b Builder) Bitop() (c Bitop) {
	c = Bitop{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "BITOP")
	return c
}

func (c Bitop) And() BitopOperationAnd {
	c.cs.s = append(c.cs.s, "AND")
	return (BitopOperationAnd)(c)
}

func (c Bitop) Or() BitopOperationOr {
	c.cs.s = append(c.cs.s, "OR")
	return (BitopOperationOr)(c)
}

func (c Bitop) Xor() BitopOperationXor {
	c.cs.s = append(c.cs.s, "XOR")
	return (BitopOperationXor)(c)
}

func (c Bitop) Not() BitopOperationNot {
	c.cs.s = append(c.cs.s, "NOT")
	return (BitopOperationNot)(c)
}

type BitopDestkey Completed

func (c BitopDestkey) Key(key ...string) BitopKey {
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
	return (BitopKey)(c)
}

type BitopKey Completed

func (c BitopKey) Key(key ...string) BitopKey {
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

func (c BitopKey) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type BitopOperationAnd Completed

func (c BitopOperationAnd) Destkey(destkey string) BitopDestkey {
	if c.ks&NoSlot == NoSlot {
		c.ks = NoSlot | slot(destkey)
	} else {
		c.ks = check(c.ks, slot(destkey))
	}
	c.cs.s = append(c.cs.s, destkey)
	return (BitopDestkey)(c)
}

type BitopOperationNot Completed

func (c BitopOperationNot) Destkey(destkey string) BitopDestkey {
	if c.ks&NoSlot == NoSlot {
		c.ks = NoSlot | slot(destkey)
	} else {
		c.ks = check(c.ks, slot(destkey))
	}
	c.cs.s = append(c.cs.s, destkey)
	return (BitopDestkey)(c)
}

type BitopOperationOr Completed

func (c BitopOperationOr) Destkey(destkey string) BitopDestkey {
	if c.ks&NoSlot == NoSlot {
		c.ks = NoSlot | slot(destkey)
	} else {
		c.ks = check(c.ks, slot(destkey))
	}
	c.cs.s = append(c.cs.s, destkey)
	return (BitopDestkey)(c)
}

type BitopOperationXor Completed

func (c BitopOperationXor) Destkey(destkey string) BitopDestkey {
	if c.ks&NoSlot == NoSlot {
		c.ks = NoSlot | slot(destkey)
	} else {
		c.ks = check(c.ks, slot(destkey))
	}
	c.cs.s = append(c.cs.s, destkey)
	return (BitopDestkey)(c)
}

type Bitpos Completed

func (b Builder) Bitpos() (c Bitpos) {
	c = Bitpos{cs: get(), ks: b.ks, cf: readonly}
	c.cs.s = append(c.cs.s, "BITPOS")
	return c
}

func (c Bitpos) Key(key string) BitposKey {
	if c.ks&NoSlot == NoSlot {
		c.ks = NoSlot | slot(key)
	} else {
		c.ks = check(c.ks, slot(key))
	}
	c.cs.s = append(c.cs.s, key)
	return (BitposKey)(c)
}

type BitposBit Completed

func (c BitposBit) Start(start int64) BitposIndexStart {
	c.cs.s = append(c.cs.s, strconv.FormatInt(start, 10))
	return (BitposIndexStart)(c)
}

func (c BitposBit) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

func (c BitposBit) Cache() Cacheable {
	c.cs.Build()
	return Cacheable(c)
}

type BitposIndexEndIndexEnd Completed

func (c BitposIndexEndIndexEnd) Byte() BitposIndexEndIndexIndexUnitByte {
	c.cs.s = append(c.cs.s, "BYTE")
	return (BitposIndexEndIndexIndexUnitByte)(c)
}

func (c BitposIndexEndIndexEnd) Bit() BitposIndexEndIndexIndexUnitBit {
	c.cs.s = append(c.cs.s, "BIT")
	return (BitposIndexEndIndexIndexUnitBit)(c)
}

func (c BitposIndexEndIndexEnd) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

func (c BitposIndexEndIndexEnd) Cache() Cacheable {
	c.cs.Build()
	return Cacheable(c)
}

type BitposIndexEndIndexIndexUnitBit Completed

func (c BitposIndexEndIndexIndexUnitBit) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

func (c BitposIndexEndIndexIndexUnitBit) Cache() Cacheable {
	c.cs.Build()
	return Cacheable(c)
}

type BitposIndexEndIndexIndexUnitByte Completed

func (c BitposIndexEndIndexIndexUnitByte) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

func (c BitposIndexEndIndexIndexUnitByte) Cache() Cacheable {
	c.cs.Build()
	return Cacheable(c)
}

type BitposIndexStart Completed

func (c BitposIndexStart) End(end int64) BitposIndexEndIndexEnd {
	c.cs.s = append(c.cs.s, strconv.FormatInt(end, 10))
	return (BitposIndexEndIndexEnd)(c)
}

func (c BitposIndexStart) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

func (c BitposIndexStart) Cache() Cacheable {
	c.cs.Build()
	return Cacheable(c)
}

type BitposKey Completed

func (c BitposKey) Bit(bit int64) BitposBit {
	c.cs.s = append(c.cs.s, strconv.FormatInt(bit, 10))
	return (BitposBit)(c)
}

type Blmove Completed

func (b Builder) Blmove() (c Blmove) {
	c = Blmove{cs: get(), ks: b.ks, cf: blockTag}
	c.cs.s = append(c.cs.s, "BLMOVE")
	return c
}

func (c Blmove) Source(source string) BlmoveSource {
	if c.ks&NoSlot == NoSlot {
		c.ks = NoSlot | slot(source)
	} else {
		c.ks = check(c.ks, slot(source))
	}
	c.cs.s = append(c.cs.s, source)
	return (BlmoveSource)(c)
}

type BlmoveDestination Completed

func (c BlmoveDestination) Left() BlmoveWherefromLeft {
	c.cs.s = append(c.cs.s, "LEFT")
	return (BlmoveWherefromLeft)(c)
}

func (c BlmoveDestination) Right() BlmoveWherefromRight {
	c.cs.s = append(c.cs.s, "RIGHT")
	return (BlmoveWherefromRight)(c)
}

type BlmoveSource Completed

func (c BlmoveSource) Destination(destination string) BlmoveDestination {
	if c.ks&NoSlot == NoSlot {
		c.ks = NoSlot | slot(destination)
	} else {
		c.ks = check(c.ks, slot(destination))
	}
	c.cs.s = append(c.cs.s, destination)
	return (BlmoveDestination)(c)
}

type BlmoveTimeout Completed

func (c BlmoveTimeout) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type BlmoveWherefromLeft Completed

func (c BlmoveWherefromLeft) Left() BlmoveWheretoLeft {
	c.cs.s = append(c.cs.s, "LEFT")
	return (BlmoveWheretoLeft)(c)
}

func (c BlmoveWherefromLeft) Right() BlmoveWheretoRight {
	c.cs.s = append(c.cs.s, "RIGHT")
	return (BlmoveWheretoRight)(c)
}

type BlmoveWherefromRight Completed

func (c BlmoveWherefromRight) Left() BlmoveWheretoLeft {
	c.cs.s = append(c.cs.s, "LEFT")
	return (BlmoveWheretoLeft)(c)
}

func (c BlmoveWherefromRight) Right() BlmoveWheretoRight {
	c.cs.s = append(c.cs.s, "RIGHT")
	return (BlmoveWheretoRight)(c)
}

type BlmoveWheretoLeft Completed

func (c BlmoveWheretoLeft) Timeout(timeout float64) BlmoveTimeout {
	c.cs.s = append(c.cs.s, strconv.FormatFloat(timeout, 'f', -1, 64))
	return (BlmoveTimeout)(c)
}

type BlmoveWheretoRight Completed

func (c BlmoveWheretoRight) Timeout(timeout float64) BlmoveTimeout {
	c.cs.s = append(c.cs.s, strconv.FormatFloat(timeout, 'f', -1, 64))
	return (BlmoveTimeout)(c)
}

type Blmpop Completed

func (b Builder) Blmpop() (c Blmpop) {
	c = Blmpop{cs: get(), ks: b.ks, cf: blockTag}
	c.cs.s = append(c.cs.s, "BLMPOP")
	return c
}

func (c Blmpop) Timeout(timeout float64) BlmpopTimeout {
	c.cs.s = append(c.cs.s, strconv.FormatFloat(timeout, 'f', -1, 64))
	return (BlmpopTimeout)(c)
}

type BlmpopCount Completed

func (c BlmpopCount) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type BlmpopKey Completed

func (c BlmpopKey) Key(key ...string) BlmpopKey {
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

func (c BlmpopKey) Left() BlmpopWhereLeft {
	c.cs.s = append(c.cs.s, "LEFT")
	return (BlmpopWhereLeft)(c)
}

func (c BlmpopKey) Right() BlmpopWhereRight {
	c.cs.s = append(c.cs.s, "RIGHT")
	return (BlmpopWhereRight)(c)
}

type BlmpopNumkeys Completed

func (c BlmpopNumkeys) Key(key ...string) BlmpopKey {
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
	return (BlmpopKey)(c)
}

type BlmpopTimeout Completed

func (c BlmpopTimeout) Numkeys(numkeys int64) BlmpopNumkeys {
	c.cs.s = append(c.cs.s, strconv.FormatInt(numkeys, 10))
	return (BlmpopNumkeys)(c)
}

type BlmpopWhereLeft Completed

func (c BlmpopWhereLeft) Count(count int64) BlmpopCount {
	c.cs.s = append(c.cs.s, "COUNT", strconv.FormatInt(count, 10))
	return (BlmpopCount)(c)
}

func (c BlmpopWhereLeft) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type BlmpopWhereRight Completed

func (c BlmpopWhereRight) Count(count int64) BlmpopCount {
	c.cs.s = append(c.cs.s, "COUNT", strconv.FormatInt(count, 10))
	return (BlmpopCount)(c)
}

func (c BlmpopWhereRight) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type Blpop Completed

func (b Builder) Blpop() (c Blpop) {
	c = Blpop{cs: get(), ks: b.ks, cf: blockTag}
	c.cs.s = append(c.cs.s, "BLPOP")
	return c
}

func (c Blpop) Key(key ...string) BlpopKey {
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
	return (BlpopKey)(c)
}

type BlpopKey Completed

func (c BlpopKey) Key(key ...string) BlpopKey {
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

func (c BlpopKey) Timeout(timeout float64) BlpopTimeout {
	c.cs.s = append(c.cs.s, strconv.FormatFloat(timeout, 'f', -1, 64))
	return (BlpopTimeout)(c)
}

type BlpopTimeout Completed

func (c BlpopTimeout) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type Brpop Completed

func (b Builder) Brpop() (c Brpop) {
	c = Brpop{cs: get(), ks: b.ks, cf: blockTag}
	c.cs.s = append(c.cs.s, "BRPOP")
	return c
}

func (c Brpop) Key(key ...string) BrpopKey {
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
	return (BrpopKey)(c)
}

type BrpopKey Completed

func (c BrpopKey) Key(key ...string) BrpopKey {
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

func (c BrpopKey) Timeout(timeout float64) BrpopTimeout {
	c.cs.s = append(c.cs.s, strconv.FormatFloat(timeout, 'f', -1, 64))
	return (BrpopTimeout)(c)
}

type BrpopTimeout Completed

func (c BrpopTimeout) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type Brpoplpush Completed

func (b Builder) Brpoplpush() (c Brpoplpush) {
	c = Brpoplpush{cs: get(), ks: b.ks, cf: blockTag}
	c.cs.s = append(c.cs.s, "BRPOPLPUSH")
	return c
}

func (c Brpoplpush) Source(source string) BrpoplpushSource {
	if c.ks&NoSlot == NoSlot {
		c.ks = NoSlot | slot(source)
	} else {
		c.ks = check(c.ks, slot(source))
	}
	c.cs.s = append(c.cs.s, source)
	return (BrpoplpushSource)(c)
}

type BrpoplpushDestination Completed

func (c BrpoplpushDestination) Timeout(timeout float64) BrpoplpushTimeout {
	c.cs.s = append(c.cs.s, strconv.FormatFloat(timeout, 'f', -1, 64))
	return (BrpoplpushTimeout)(c)
}

type BrpoplpushSource Completed

func (c BrpoplpushSource) Destination(destination string) BrpoplpushDestination {
	if c.ks&NoSlot == NoSlot {
		c.ks = NoSlot | slot(destination)
	} else {
		c.ks = check(c.ks, slot(destination))
	}
	c.cs.s = append(c.cs.s, destination)
	return (BrpoplpushDestination)(c)
}

type BrpoplpushTimeout Completed

func (c BrpoplpushTimeout) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

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

type CfAdd Completed

func (b Builder) CfAdd() (c CfAdd) {
	c = CfAdd{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "CF.ADD")
	return c
}

func (c CfAdd) Key(key string) CfAddKey {
	if c.ks&NoSlot == NoSlot {
		c.ks = NoSlot | slot(key)
	} else {
		c.ks = check(c.ks, slot(key))
	}
	c.cs.s = append(c.cs.s, key)
	return (CfAddKey)(c)
}

type CfAddItem Completed

func (c CfAddItem) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type CfAddKey Completed

func (c CfAddKey) Item(item string) CfAddItem {
	c.cs.s = append(c.cs.s, item)
	return (CfAddItem)(c)
}

type CfAddnx Completed

func (b Builder) CfAddnx() (c CfAddnx) {
	c = CfAddnx{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "CF.ADDNX")
	return c
}

func (c CfAddnx) Key(key string) CfAddnxKey {
	if c.ks&NoSlot == NoSlot {
		c.ks = NoSlot | slot(key)
	} else {
		c.ks = check(c.ks, slot(key))
	}
	c.cs.s = append(c.cs.s, key)
	return (CfAddnxKey)(c)
}

type CfAddnxItem Completed

func (c CfAddnxItem) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type CfAddnxKey Completed

func (c CfAddnxKey) Item(item string) CfAddnxItem {
	c.cs.s = append(c.cs.s, item)
	return (CfAddnxItem)(c)
}

type CfCount Completed

func (b Builder) CfCount() (c CfCount) {
	c = CfCount{cs: get(), ks: b.ks, cf: readonly}
	c.cs.s = append(c.cs.s, "CF.COUNT")
	return c
}

func (c CfCount) Key(key string) CfCountKey {
	if c.ks&NoSlot == NoSlot {
		c.ks = NoSlot | slot(key)
	} else {
		c.ks = check(c.ks, slot(key))
	}
	c.cs.s = append(c.cs.s, key)
	return (CfCountKey)(c)
}

type CfCountItem Completed

func (c CfCountItem) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

func (c CfCountItem) Cache() Cacheable {
	c.cs.Build()
	return Cacheable(c)
}

type CfCountKey Completed

func (c CfCountKey) Item(item string) CfCountItem {
	c.cs.s = append(c.cs.s, item)
	return (CfCountItem)(c)
}

type CfDel Completed

func (b Builder) CfDel() (c CfDel) {
	c = CfDel{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "CF.DEL")
	return c
}

func (c CfDel) Key(key string) CfDelKey {
	if c.ks&NoSlot == NoSlot {
		c.ks = NoSlot | slot(key)
	} else {
		c.ks = check(c.ks, slot(key))
	}
	c.cs.s = append(c.cs.s, key)
	return (CfDelKey)(c)
}

type CfDelItem Completed

func (c CfDelItem) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type CfDelKey Completed

func (c CfDelKey) Item(item string) CfDelItem {
	c.cs.s = append(c.cs.s, item)
	return (CfDelItem)(c)
}

type CfExists Completed

func (b Builder) CfExists() (c CfExists) {
	c = CfExists{cs: get(), ks: b.ks, cf: readonly}
	c.cs.s = append(c.cs.s, "CF.EXISTS")
	return c
}

func (c CfExists) Key(key string) CfExistsKey {
	if c.ks&NoSlot == NoSlot {
		c.ks = NoSlot | slot(key)
	} else {
		c.ks = check(c.ks, slot(key))
	}
	c.cs.s = append(c.cs.s, key)
	return (CfExistsKey)(c)
}

type CfExistsItem Completed

func (c CfExistsItem) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

func (c CfExistsItem) Cache() Cacheable {
	c.cs.Build()
	return Cacheable(c)
}

type CfExistsKey Completed

func (c CfExistsKey) Item(item string) CfExistsItem {
	c.cs.s = append(c.cs.s, item)
	return (CfExistsItem)(c)
}

type CfInfo Completed

func (b Builder) CfInfo() (c CfInfo) {
	c = CfInfo{cs: get(), ks: b.ks, cf: readonly}
	c.cs.s = append(c.cs.s, "CF.INFO")
	return c
}

func (c CfInfo) Key(key string) CfInfoKey {
	if c.ks&NoSlot == NoSlot {
		c.ks = NoSlot | slot(key)
	} else {
		c.ks = check(c.ks, slot(key))
	}
	c.cs.s = append(c.cs.s, key)
	return (CfInfoKey)(c)
}

type CfInfoKey Completed

func (c CfInfoKey) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

func (c CfInfoKey) Cache() Cacheable {
	c.cs.Build()
	return Cacheable(c)
}

type CfInsert Completed

func (b Builder) CfInsert() (c CfInsert) {
	c = CfInsert{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "CF.INSERT")
	return c
}

func (c CfInsert) Key(key string) CfInsertKey {
	if c.ks&NoSlot == NoSlot {
		c.ks = NoSlot | slot(key)
	} else {
		c.ks = check(c.ks, slot(key))
	}
	c.cs.s = append(c.cs.s, key)
	return (CfInsertKey)(c)
}

type CfInsertCapacity Completed

func (c CfInsertCapacity) Nocreate() CfInsertNocreate {
	c.cs.s = append(c.cs.s, "NOCREATE")
	return (CfInsertNocreate)(c)
}

func (c CfInsertCapacity) Items() CfInsertItems {
	c.cs.s = append(c.cs.s, "ITEMS")
	return (CfInsertItems)(c)
}

type CfInsertItem Completed

func (c CfInsertItem) Item(item ...string) CfInsertItem {
	c.cs.s = append(c.cs.s, item...)
	return c
}

func (c CfInsertItem) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type CfInsertItems Completed

func (c CfInsertItems) Item(item ...string) CfInsertItem {
	c.cs.s = append(c.cs.s, item...)
	return (CfInsertItem)(c)
}

type CfInsertKey Completed

func (c CfInsertKey) Capacity(capacity int64) CfInsertCapacity {
	c.cs.s = append(c.cs.s, "CAPACITY", strconv.FormatInt(capacity, 10))
	return (CfInsertCapacity)(c)
}

func (c CfInsertKey) Nocreate() CfInsertNocreate {
	c.cs.s = append(c.cs.s, "NOCREATE")
	return (CfInsertNocreate)(c)
}

func (c CfInsertKey) Items() CfInsertItems {
	c.cs.s = append(c.cs.s, "ITEMS")
	return (CfInsertItems)(c)
}

type CfInsertNocreate Completed

func (c CfInsertNocreate) Items() CfInsertItems {
	c.cs.s = append(c.cs.s, "ITEMS")
	return (CfInsertItems)(c)
}

type CfInsertnx Completed

func (b Builder) CfInsertnx() (c CfInsertnx) {
	c = CfInsertnx{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "CF.INSERTNX")
	return c
}

func (c CfInsertnx) Key(key string) CfInsertnxKey {
	if c.ks&NoSlot == NoSlot {
		c.ks = NoSlot | slot(key)
	} else {
		c.ks = check(c.ks, slot(key))
	}
	c.cs.s = append(c.cs.s, key)
	return (CfInsertnxKey)(c)
}

type CfInsertnxCapacity Completed

func (c CfInsertnxCapacity) Nocreate() CfInsertnxNocreate {
	c.cs.s = append(c.cs.s, "NOCREATE")
	return (CfInsertnxNocreate)(c)
}

func (c CfInsertnxCapacity) Items() CfInsertnxItems {
	c.cs.s = append(c.cs.s, "ITEMS")
	return (CfInsertnxItems)(c)
}

type CfInsertnxItem Completed

func (c CfInsertnxItem) Item(item ...string) CfInsertnxItem {
	c.cs.s = append(c.cs.s, item...)
	return c
}

func (c CfInsertnxItem) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type CfInsertnxItems Completed

func (c CfInsertnxItems) Item(item ...string) CfInsertnxItem {
	c.cs.s = append(c.cs.s, item...)
	return (CfInsertnxItem)(c)
}

type CfInsertnxKey Completed

func (c CfInsertnxKey) Capacity(capacity int64) CfInsertnxCapacity {
	c.cs.s = append(c.cs.s, "CAPACITY", strconv.FormatInt(capacity, 10))
	return (CfInsertnxCapacity)(c)
}

func (c CfInsertnxKey) Nocreate() CfInsertnxNocreate {
	c.cs.s = append(c.cs.s, "NOCREATE")
	return (CfInsertnxNocreate)(c)
}

func (c CfInsertnxKey) Items() CfInsertnxItems {
	c.cs.s = append(c.cs.s, "ITEMS")
	return (CfInsertnxItems)(c)
}

type CfInsertnxNocreate Completed

func (c CfInsertnxNocreate) Items() CfInsertnxItems {
	c.cs.s = append(c.cs.s, "ITEMS")
	return (CfInsertnxItems)(c)
}

type CfLoadchunk Completed

func (b Builder) CfLoadchunk() (c CfLoadchunk) {
	c = CfLoadchunk{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "CF.LOADCHUNK")
	return c
}

func (c CfLoadchunk) Key(key string) CfLoadchunkKey {
	if c.ks&NoSlot == NoSlot {
		c.ks = NoSlot | slot(key)
	} else {
		c.ks = check(c.ks, slot(key))
	}
	c.cs.s = append(c.cs.s, key)
	return (CfLoadchunkKey)(c)
}

type CfLoadchunkData Completed

func (c CfLoadchunkData) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type CfLoadchunkIterator Completed

func (c CfLoadchunkIterator) Data(data string) CfLoadchunkData {
	c.cs.s = append(c.cs.s, data)
	return (CfLoadchunkData)(c)
}

type CfLoadchunkKey Completed

func (c CfLoadchunkKey) Iterator(iterator int64) CfLoadchunkIterator {
	c.cs.s = append(c.cs.s, strconv.FormatInt(iterator, 10))
	return (CfLoadchunkIterator)(c)
}

type CfMexists Completed

func (b Builder) CfMexists() (c CfMexists) {
	c = CfMexists{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "CF.MEXISTS")
	return c
}

func (c CfMexists) Key(key string) CfMexistsKey {
	if c.ks&NoSlot == NoSlot {
		c.ks = NoSlot | slot(key)
	} else {
		c.ks = check(c.ks, slot(key))
	}
	c.cs.s = append(c.cs.s, key)
	return (CfMexistsKey)(c)
}

type CfMexistsItem Completed

func (c CfMexistsItem) Item(item ...string) CfMexistsItem {
	c.cs.s = append(c.cs.s, item...)
	return c
}

func (c CfMexistsItem) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type CfMexistsKey Completed

func (c CfMexistsKey) Item(item ...string) CfMexistsItem {
	c.cs.s = append(c.cs.s, item...)
	return (CfMexistsItem)(c)
}

type CfReserve Completed

func (b Builder) CfReserve() (c CfReserve) {
	c = CfReserve{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "CF.RESERVE")
	return c
}

func (c CfReserve) Key(key string) CfReserveKey {
	if c.ks&NoSlot == NoSlot {
		c.ks = NoSlot | slot(key)
	} else {
		c.ks = check(c.ks, slot(key))
	}
	c.cs.s = append(c.cs.s, key)
	return (CfReserveKey)(c)
}

type CfReserveBucketsize Completed

func (c CfReserveBucketsize) Maxiterations(maxiterations int64) CfReserveMaxiterations {
	c.cs.s = append(c.cs.s, "MAXITERATIONS", strconv.FormatInt(maxiterations, 10))
	return (CfReserveMaxiterations)(c)
}

func (c CfReserveBucketsize) Expansion(expansion int64) CfReserveExpansion {
	c.cs.s = append(c.cs.s, "EXPANSION", strconv.FormatInt(expansion, 10))
	return (CfReserveExpansion)(c)
}

func (c CfReserveBucketsize) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type CfReserveCapacity Completed

func (c CfReserveCapacity) Bucketsize(bucketsize int64) CfReserveBucketsize {
	c.cs.s = append(c.cs.s, "BUCKETSIZE", strconv.FormatInt(bucketsize, 10))
	return (CfReserveBucketsize)(c)
}

func (c CfReserveCapacity) Maxiterations(maxiterations int64) CfReserveMaxiterations {
	c.cs.s = append(c.cs.s, "MAXITERATIONS", strconv.FormatInt(maxiterations, 10))
	return (CfReserveMaxiterations)(c)
}

func (c CfReserveCapacity) Expansion(expansion int64) CfReserveExpansion {
	c.cs.s = append(c.cs.s, "EXPANSION", strconv.FormatInt(expansion, 10))
	return (CfReserveExpansion)(c)
}

func (c CfReserveCapacity) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type CfReserveExpansion Completed

func (c CfReserveExpansion) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type CfReserveKey Completed

func (c CfReserveKey) Capacity(capacity int64) CfReserveCapacity {
	c.cs.s = append(c.cs.s, strconv.FormatInt(capacity, 10))
	return (CfReserveCapacity)(c)
}

type CfReserveMaxiterations Completed

func (c CfReserveMaxiterations) Expansion(expansion int64) CfReserveExpansion {
	c.cs.s = append(c.cs.s, "EXPANSION", strconv.FormatInt(expansion, 10))
	return (CfReserveExpansion)(c)
}

func (c CfReserveMaxiterations) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type CfScandump Completed

func (b Builder) CfScandump() (c CfScandump) {
	c = CfScandump{cs: get(), ks: b.ks, cf: readonly}
	c.cs.s = append(c.cs.s, "CF.SCANDUMP")
	return c
}

func (c CfScandump) Key(key string) CfScandumpKey {
	if c.ks&NoSlot == NoSlot {
		c.ks = NoSlot | slot(key)
	} else {
		c.ks = check(c.ks, slot(key))
	}
	c.cs.s = append(c.cs.s, key)
	return (CfScandumpKey)(c)
}

type CfScandumpIterator Completed

func (c CfScandumpIterator) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type CfScandumpKey Completed

func (c CfScandumpKey) Iterator(iterator int64) CfScandumpIterator {
	c.cs.s = append(c.cs.s, strconv.FormatInt(iterator, 10))
	return (CfScandumpIterator)(c)
}

type ClientCaching Completed

func (b Builder) ClientCaching() (c ClientCaching) {
	c = ClientCaching{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "CLIENT", "CACHING")
	return c
}

func (c ClientCaching) Yes() ClientCachingModeYes {
	c.cs.s = append(c.cs.s, "YES")
	return (ClientCachingModeYes)(c)
}

func (c ClientCaching) No() ClientCachingModeNo {
	c.cs.s = append(c.cs.s, "NO")
	return (ClientCachingModeNo)(c)
}

type ClientCachingModeNo Completed

func (c ClientCachingModeNo) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type ClientCachingModeYes Completed

func (c ClientCachingModeYes) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type ClientGetname Completed

func (b Builder) ClientGetname() (c ClientGetname) {
	c = ClientGetname{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "CLIENT", "GETNAME")
	return c
}

func (c ClientGetname) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type ClientGetredir Completed

func (b Builder) ClientGetredir() (c ClientGetredir) {
	c = ClientGetredir{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "CLIENT", "GETREDIR")
	return c
}

func (c ClientGetredir) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type ClientId Completed

func (b Builder) ClientId() (c ClientId) {
	c = ClientId{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "CLIENT", "ID")
	return c
}

func (c ClientId) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type ClientInfo Completed

func (b Builder) ClientInfo() (c ClientInfo) {
	c = ClientInfo{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "CLIENT", "INFO")
	return c
}

func (c ClientInfo) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type ClientKill Completed

func (b Builder) ClientKill() (c ClientKill) {
	c = ClientKill{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "CLIENT", "KILL")
	return c
}

func (c ClientKill) IpPort(ipPort string) ClientKillIpPort {
	c.cs.s = append(c.cs.s, ipPort)
	return (ClientKillIpPort)(c)
}

func (c ClientKill) Id(clientId int64) ClientKillId {
	c.cs.s = append(c.cs.s, "ID", strconv.FormatInt(clientId, 10))
	return (ClientKillId)(c)
}

func (c ClientKill) TypeNormal() ClientKillTypeNormal {
	c.cs.s = append(c.cs.s, "TYPE", "NORMAL")
	return (ClientKillTypeNormal)(c)
}

func (c ClientKill) TypeMaster() ClientKillTypeMaster {
	c.cs.s = append(c.cs.s, "TYPE", "MASTER")
	return (ClientKillTypeMaster)(c)
}

func (c ClientKill) TypeReplica() ClientKillTypeReplica {
	c.cs.s = append(c.cs.s, "TYPE", "REPLICA")
	return (ClientKillTypeReplica)(c)
}

func (c ClientKill) TypePubsub() ClientKillTypePubsub {
	c.cs.s = append(c.cs.s, "TYPE", "PUBSUB")
	return (ClientKillTypePubsub)(c)
}

func (c ClientKill) User(username string) ClientKillUser {
	c.cs.s = append(c.cs.s, "USER", username)
	return (ClientKillUser)(c)
}

func (c ClientKill) Addr(ipPort string) ClientKillAddr {
	c.cs.s = append(c.cs.s, "ADDR", ipPort)
	return (ClientKillAddr)(c)
}

func (c ClientKill) Laddr(ipPort string) ClientKillLaddr {
	c.cs.s = append(c.cs.s, "LADDR", ipPort)
	return (ClientKillLaddr)(c)
}

func (c ClientKill) SkipmeYes() ClientKillSkipmeYes {
	c.cs.s = append(c.cs.s, "SKIPME", "YES")
	return (ClientKillSkipmeYes)(c)
}

func (c ClientKill) SkipmeNo() ClientKillSkipmeNo {
	c.cs.s = append(c.cs.s, "SKIPME", "NO")
	return (ClientKillSkipmeNo)(c)
}

func (c ClientKill) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type ClientKillAddr Completed

func (c ClientKillAddr) Laddr(ipPort string) ClientKillLaddr {
	c.cs.s = append(c.cs.s, "LADDR", ipPort)
	return (ClientKillLaddr)(c)
}

func (c ClientKillAddr) SkipmeYes() ClientKillSkipmeYes {
	c.cs.s = append(c.cs.s, "SKIPME", "YES")
	return (ClientKillSkipmeYes)(c)
}

func (c ClientKillAddr) SkipmeNo() ClientKillSkipmeNo {
	c.cs.s = append(c.cs.s, "SKIPME", "NO")
	return (ClientKillSkipmeNo)(c)
}

func (c ClientKillAddr) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type ClientKillId Completed

func (c ClientKillId) TypeNormal() ClientKillTypeNormal {
	c.cs.s = append(c.cs.s, "TYPE", "NORMAL")
	return (ClientKillTypeNormal)(c)
}

func (c ClientKillId) TypeMaster() ClientKillTypeMaster {
	c.cs.s = append(c.cs.s, "TYPE", "MASTER")
	return (ClientKillTypeMaster)(c)
}

func (c ClientKillId) TypeReplica() ClientKillTypeReplica {
	c.cs.s = append(c.cs.s, "TYPE", "REPLICA")
	return (ClientKillTypeReplica)(c)
}

func (c ClientKillId) TypePubsub() ClientKillTypePubsub {
	c.cs.s = append(c.cs.s, "TYPE", "PUBSUB")
	return (ClientKillTypePubsub)(c)
}

func (c ClientKillId) User(username string) ClientKillUser {
	c.cs.s = append(c.cs.s, "USER", username)
	return (ClientKillUser)(c)
}

func (c ClientKillId) Addr(ipPort string) ClientKillAddr {
	c.cs.s = append(c.cs.s, "ADDR", ipPort)
	return (ClientKillAddr)(c)
}

func (c ClientKillId) Laddr(ipPort string) ClientKillLaddr {
	c.cs.s = append(c.cs.s, "LADDR", ipPort)
	return (ClientKillLaddr)(c)
}

func (c ClientKillId) SkipmeYes() ClientKillSkipmeYes {
	c.cs.s = append(c.cs.s, "SKIPME", "YES")
	return (ClientKillSkipmeYes)(c)
}

func (c ClientKillId) SkipmeNo() ClientKillSkipmeNo {
	c.cs.s = append(c.cs.s, "SKIPME", "NO")
	return (ClientKillSkipmeNo)(c)
}

func (c ClientKillId) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type ClientKillIpPort Completed

func (c ClientKillIpPort) Id(clientId int64) ClientKillId {
	c.cs.s = append(c.cs.s, "ID", strconv.FormatInt(clientId, 10))
	return (ClientKillId)(c)
}

func (c ClientKillIpPort) TypeNormal() ClientKillTypeNormal {
	c.cs.s = append(c.cs.s, "TYPE", "NORMAL")
	return (ClientKillTypeNormal)(c)
}

func (c ClientKillIpPort) TypeMaster() ClientKillTypeMaster {
	c.cs.s = append(c.cs.s, "TYPE", "MASTER")
	return (ClientKillTypeMaster)(c)
}

func (c ClientKillIpPort) TypeReplica() ClientKillTypeReplica {
	c.cs.s = append(c.cs.s, "TYPE", "REPLICA")
	return (ClientKillTypeReplica)(c)
}

func (c ClientKillIpPort) TypePubsub() ClientKillTypePubsub {
	c.cs.s = append(c.cs.s, "TYPE", "PUBSUB")
	return (ClientKillTypePubsub)(c)
}

func (c ClientKillIpPort) User(username string) ClientKillUser {
	c.cs.s = append(c.cs.s, "USER", username)
	return (ClientKillUser)(c)
}

func (c ClientKillIpPort) Addr(ipPort string) ClientKillAddr {
	c.cs.s = append(c.cs.s, "ADDR", ipPort)
	return (ClientKillAddr)(c)
}

func (c ClientKillIpPort) Laddr(ipPort string) ClientKillLaddr {
	c.cs.s = append(c.cs.s, "LADDR", ipPort)
	return (ClientKillLaddr)(c)
}

func (c ClientKillIpPort) SkipmeYes() ClientKillSkipmeYes {
	c.cs.s = append(c.cs.s, "SKIPME", "YES")
	return (ClientKillSkipmeYes)(c)
}

func (c ClientKillIpPort) SkipmeNo() ClientKillSkipmeNo {
	c.cs.s = append(c.cs.s, "SKIPME", "NO")
	return (ClientKillSkipmeNo)(c)
}

func (c ClientKillIpPort) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type ClientKillLaddr Completed

func (c ClientKillLaddr) SkipmeYes() ClientKillSkipmeYes {
	c.cs.s = append(c.cs.s, "SKIPME", "YES")
	return (ClientKillSkipmeYes)(c)
}

func (c ClientKillLaddr) SkipmeNo() ClientKillSkipmeNo {
	c.cs.s = append(c.cs.s, "SKIPME", "NO")
	return (ClientKillSkipmeNo)(c)
}

func (c ClientKillLaddr) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type ClientKillSkipmeNo Completed

func (c ClientKillSkipmeNo) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type ClientKillSkipmeYes Completed

func (c ClientKillSkipmeYes) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type ClientKillTypeMaster Completed

func (c ClientKillTypeMaster) User(username string) ClientKillUser {
	c.cs.s = append(c.cs.s, "USER", username)
	return (ClientKillUser)(c)
}

func (c ClientKillTypeMaster) Addr(ipPort string) ClientKillAddr {
	c.cs.s = append(c.cs.s, "ADDR", ipPort)
	return (ClientKillAddr)(c)
}

func (c ClientKillTypeMaster) Laddr(ipPort string) ClientKillLaddr {
	c.cs.s = append(c.cs.s, "LADDR", ipPort)
	return (ClientKillLaddr)(c)
}

func (c ClientKillTypeMaster) SkipmeYes() ClientKillSkipmeYes {
	c.cs.s = append(c.cs.s, "SKIPME", "YES")
	return (ClientKillSkipmeYes)(c)
}

func (c ClientKillTypeMaster) SkipmeNo() ClientKillSkipmeNo {
	c.cs.s = append(c.cs.s, "SKIPME", "NO")
	return (ClientKillSkipmeNo)(c)
}

func (c ClientKillTypeMaster) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type ClientKillTypeNormal Completed

func (c ClientKillTypeNormal) User(username string) ClientKillUser {
	c.cs.s = append(c.cs.s, "USER", username)
	return (ClientKillUser)(c)
}

func (c ClientKillTypeNormal) Addr(ipPort string) ClientKillAddr {
	c.cs.s = append(c.cs.s, "ADDR", ipPort)
	return (ClientKillAddr)(c)
}

func (c ClientKillTypeNormal) Laddr(ipPort string) ClientKillLaddr {
	c.cs.s = append(c.cs.s, "LADDR", ipPort)
	return (ClientKillLaddr)(c)
}

func (c ClientKillTypeNormal) SkipmeYes() ClientKillSkipmeYes {
	c.cs.s = append(c.cs.s, "SKIPME", "YES")
	return (ClientKillSkipmeYes)(c)
}

func (c ClientKillTypeNormal) SkipmeNo() ClientKillSkipmeNo {
	c.cs.s = append(c.cs.s, "SKIPME", "NO")
	return (ClientKillSkipmeNo)(c)
}

func (c ClientKillTypeNormal) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type ClientKillTypePubsub Completed

func (c ClientKillTypePubsub) User(username string) ClientKillUser {
	c.cs.s = append(c.cs.s, "USER", username)
	return (ClientKillUser)(c)
}

func (c ClientKillTypePubsub) Addr(ipPort string) ClientKillAddr {
	c.cs.s = append(c.cs.s, "ADDR", ipPort)
	return (ClientKillAddr)(c)
}

func (c ClientKillTypePubsub) Laddr(ipPort string) ClientKillLaddr {
	c.cs.s = append(c.cs.s, "LADDR", ipPort)
	return (ClientKillLaddr)(c)
}

func (c ClientKillTypePubsub) SkipmeYes() ClientKillSkipmeYes {
	c.cs.s = append(c.cs.s, "SKIPME", "YES")
	return (ClientKillSkipmeYes)(c)
}

func (c ClientKillTypePubsub) SkipmeNo() ClientKillSkipmeNo {
	c.cs.s = append(c.cs.s, "SKIPME", "NO")
	return (ClientKillSkipmeNo)(c)
}

func (c ClientKillTypePubsub) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type ClientKillTypeReplica Completed

func (c ClientKillTypeReplica) User(username string) ClientKillUser {
	c.cs.s = append(c.cs.s, "USER", username)
	return (ClientKillUser)(c)
}

func (c ClientKillTypeReplica) Addr(ipPort string) ClientKillAddr {
	c.cs.s = append(c.cs.s, "ADDR", ipPort)
	return (ClientKillAddr)(c)
}

func (c ClientKillTypeReplica) Laddr(ipPort string) ClientKillLaddr {
	c.cs.s = append(c.cs.s, "LADDR", ipPort)
	return (ClientKillLaddr)(c)
}

func (c ClientKillTypeReplica) SkipmeYes() ClientKillSkipmeYes {
	c.cs.s = append(c.cs.s, "SKIPME", "YES")
	return (ClientKillSkipmeYes)(c)
}

func (c ClientKillTypeReplica) SkipmeNo() ClientKillSkipmeNo {
	c.cs.s = append(c.cs.s, "SKIPME", "NO")
	return (ClientKillSkipmeNo)(c)
}

func (c ClientKillTypeReplica) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type ClientKillUser Completed

func (c ClientKillUser) Addr(ipPort string) ClientKillAddr {
	c.cs.s = append(c.cs.s, "ADDR", ipPort)
	return (ClientKillAddr)(c)
}

func (c ClientKillUser) Laddr(ipPort string) ClientKillLaddr {
	c.cs.s = append(c.cs.s, "LADDR", ipPort)
	return (ClientKillLaddr)(c)
}

func (c ClientKillUser) SkipmeYes() ClientKillSkipmeYes {
	c.cs.s = append(c.cs.s, "SKIPME", "YES")
	return (ClientKillSkipmeYes)(c)
}

func (c ClientKillUser) SkipmeNo() ClientKillSkipmeNo {
	c.cs.s = append(c.cs.s, "SKIPME", "NO")
	return (ClientKillSkipmeNo)(c)
}

func (c ClientKillUser) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type ClientList Completed

func (b Builder) ClientList() (c ClientList) {
	c = ClientList{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "CLIENT", "LIST")
	return c
}

func (c ClientList) TypeNormal() ClientListTypeNormal {
	c.cs.s = append(c.cs.s, "TYPE", "NORMAL")
	return (ClientListTypeNormal)(c)
}

func (c ClientList) TypeMaster() ClientListTypeMaster {
	c.cs.s = append(c.cs.s, "TYPE", "MASTER")
	return (ClientListTypeMaster)(c)
}

func (c ClientList) TypeReplica() ClientListTypeReplica {
	c.cs.s = append(c.cs.s, "TYPE", "REPLICA")
	return (ClientListTypeReplica)(c)
}

func (c ClientList) TypePubsub() ClientListTypePubsub {
	c.cs.s = append(c.cs.s, "TYPE", "PUBSUB")
	return (ClientListTypePubsub)(c)
}

func (c ClientList) Id() ClientListIdId {
	c.cs.s = append(c.cs.s, "ID")
	return (ClientListIdId)(c)
}

func (c ClientList) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type ClientListIdClientId Completed

func (c ClientListIdClientId) ClientId(clientId ...int64) ClientListIdClientId {
	for _, n := range clientId {
		c.cs.s = append(c.cs.s, strconv.FormatInt(n, 10))
	}
	return c
}

func (c ClientListIdClientId) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type ClientListIdId Completed

func (c ClientListIdId) ClientId(clientId ...int64) ClientListIdClientId {
	for _, n := range clientId {
		c.cs.s = append(c.cs.s, strconv.FormatInt(n, 10))
	}
	return (ClientListIdClientId)(c)
}

type ClientListTypeMaster Completed

func (c ClientListTypeMaster) Id() ClientListIdId {
	c.cs.s = append(c.cs.s, "ID")
	return (ClientListIdId)(c)
}

func (c ClientListTypeMaster) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type ClientListTypeNormal Completed

func (c ClientListTypeNormal) Id() ClientListIdId {
	c.cs.s = append(c.cs.s, "ID")
	return (ClientListIdId)(c)
}

func (c ClientListTypeNormal) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type ClientListTypePubsub Completed

func (c ClientListTypePubsub) Id() ClientListIdId {
	c.cs.s = append(c.cs.s, "ID")
	return (ClientListIdId)(c)
}

func (c ClientListTypePubsub) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type ClientListTypeReplica Completed

func (c ClientListTypeReplica) Id() ClientListIdId {
	c.cs.s = append(c.cs.s, "ID")
	return (ClientListIdId)(c)
}

func (c ClientListTypeReplica) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type ClientNoEvict Completed

func (b Builder) ClientNoEvict() (c ClientNoEvict) {
	c = ClientNoEvict{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "CLIENT", "NO-EVICT")
	return c
}

func (c ClientNoEvict) On() ClientNoEvictEnabledOn {
	c.cs.s = append(c.cs.s, "ON")
	return (ClientNoEvictEnabledOn)(c)
}

func (c ClientNoEvict) Off() ClientNoEvictEnabledOff {
	c.cs.s = append(c.cs.s, "OFF")
	return (ClientNoEvictEnabledOff)(c)
}

type ClientNoEvictEnabledOff Completed

func (c ClientNoEvictEnabledOff) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type ClientNoEvictEnabledOn Completed

func (c ClientNoEvictEnabledOn) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type ClientNoTouch Completed

func (b Builder) ClientNoTouch() (c ClientNoTouch) {
	c = ClientNoTouch{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "CLIENT", "NO-TOUCH")
	return c
}

func (c ClientNoTouch) On() ClientNoTouchEnabledOn {
	c.cs.s = append(c.cs.s, "ON")
	return (ClientNoTouchEnabledOn)(c)
}

func (c ClientNoTouch) Off() ClientNoTouchEnabledOff {
	c.cs.s = append(c.cs.s, "OFF")
	return (ClientNoTouchEnabledOff)(c)
}

type ClientNoTouchEnabledOff Completed

func (c ClientNoTouchEnabledOff) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type ClientNoTouchEnabledOn Completed

func (c ClientNoTouchEnabledOn) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type ClientPause Completed

func (b Builder) ClientPause() (c ClientPause) {
	c = ClientPause{cs: get(), ks: b.ks, cf: blockTag}
	c.cs.s = append(c.cs.s, "CLIENT", "PAUSE")
	return c
}

func (c ClientPause) Timeout(timeout int64) ClientPauseTimeout {
	c.cs.s = append(c.cs.s, strconv.FormatInt(timeout, 10))
	return (ClientPauseTimeout)(c)
}

type ClientPauseModeAll Completed

func (c ClientPauseModeAll) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type ClientPauseModeWrite Completed

func (c ClientPauseModeWrite) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type ClientPauseTimeout Completed

func (c ClientPauseTimeout) Write() ClientPauseModeWrite {
	c.cs.s = append(c.cs.s, "WRITE")
	return (ClientPauseModeWrite)(c)
}

func (c ClientPauseTimeout) All() ClientPauseModeAll {
	c.cs.s = append(c.cs.s, "ALL")
	return (ClientPauseModeAll)(c)
}

func (c ClientPauseTimeout) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type ClientReply Completed

func (b Builder) ClientReply() (c ClientReply) {
	c = ClientReply{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "CLIENT", "REPLY")
	return c
}

func (c ClientReply) On() ClientReplyReplyModeOn {
	c.cs.s = append(c.cs.s, "ON")
	return (ClientReplyReplyModeOn)(c)
}

func (c ClientReply) Off() ClientReplyReplyModeOff {
	c.cs.s = append(c.cs.s, "OFF")
	return (ClientReplyReplyModeOff)(c)
}

func (c ClientReply) Skip() ClientReplyReplyModeSkip {
	c.cs.s = append(c.cs.s, "SKIP")
	return (ClientReplyReplyModeSkip)(c)
}

type ClientReplyReplyModeOff Completed

func (c ClientReplyReplyModeOff) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type ClientReplyReplyModeOn Completed

func (c ClientReplyReplyModeOn) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type ClientReplyReplyModeSkip Completed

func (c ClientReplyReplyModeSkip) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type ClientSetinfo Completed

func (b Builder) ClientSetinfo() (c ClientSetinfo) {
	c = ClientSetinfo{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "CLIENT", "SETINFO")
	return c
}

func (c ClientSetinfo) Libname(libname string) ClientSetinfoAttrLibname {
	c.cs.s = append(c.cs.s, libname)
	return (ClientSetinfoAttrLibname)(c)
}

type ClientSetinfoAttrLibname Completed

func (c ClientSetinfoAttrLibname) Libver(libver string) ClientSetinfoAttrLibver {
	c.cs.s = append(c.cs.s, libver)
	return (ClientSetinfoAttrLibver)(c)
}

type ClientSetinfoAttrLibver Completed

func (c ClientSetinfoAttrLibver) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type ClientSetname Completed

func (b Builder) ClientSetname() (c ClientSetname) {
	c = ClientSetname{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "CLIENT", "SETNAME")
	return c
}

func (c ClientSetname) ConnectionName(connectionName string) ClientSetnameConnectionName {
	c.cs.s = append(c.cs.s, connectionName)
	return (ClientSetnameConnectionName)(c)
}

type ClientSetnameConnectionName Completed

func (c ClientSetnameConnectionName) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type ClientTracking Completed

func (b Builder) ClientTracking() (c ClientTracking) {
	c = ClientTracking{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "CLIENT", "TRACKING")
	return c
}

func (c ClientTracking) On() ClientTrackingStatusOn {
	c.cs.s = append(c.cs.s, "ON")
	return (ClientTrackingStatusOn)(c)
}

func (c ClientTracking) Off() ClientTrackingStatusOff {
	c.cs.s = append(c.cs.s, "OFF")
	return (ClientTrackingStatusOff)(c)
}

type ClientTrackingBcast Completed

func (c ClientTrackingBcast) Optin() ClientTrackingOptin {
	c.cs.s = append(c.cs.s, "OPTIN")
	return (ClientTrackingOptin)(c)
}

func (c ClientTrackingBcast) Optout() ClientTrackingOptout {
	c.cs.s = append(c.cs.s, "OPTOUT")
	return (ClientTrackingOptout)(c)
}

func (c ClientTrackingBcast) Noloop() ClientTrackingNoloop {
	c.cs.s = append(c.cs.s, "NOLOOP")
	return (ClientTrackingNoloop)(c)
}

func (c ClientTrackingBcast) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type ClientTrackingNoloop Completed

func (c ClientTrackingNoloop) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type ClientTrackingOptin Completed

func (c ClientTrackingOptin) Optout() ClientTrackingOptout {
	c.cs.s = append(c.cs.s, "OPTOUT")
	return (ClientTrackingOptout)(c)
}

func (c ClientTrackingOptin) Noloop() ClientTrackingNoloop {
	c.cs.s = append(c.cs.s, "NOLOOP")
	return (ClientTrackingNoloop)(c)
}

func (c ClientTrackingOptin) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type ClientTrackingOptout Completed

func (c ClientTrackingOptout) Noloop() ClientTrackingNoloop {
	c.cs.s = append(c.cs.s, "NOLOOP")
	return (ClientTrackingNoloop)(c)
}

func (c ClientTrackingOptout) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type ClientTrackingPrefix Completed

func (c ClientTrackingPrefix) Prefix(prefix string) ClientTrackingPrefix {
	c.cs.s = append(c.cs.s, "PREFIX", prefix)
	return c
}

func (c ClientTrackingPrefix) Bcast() ClientTrackingBcast {
	c.cs.s = append(c.cs.s, "BCAST")
	return (ClientTrackingBcast)(c)
}

func (c ClientTrackingPrefix) Optin() ClientTrackingOptin {
	c.cs.s = append(c.cs.s, "OPTIN")
	return (ClientTrackingOptin)(c)
}

func (c ClientTrackingPrefix) Optout() ClientTrackingOptout {
	c.cs.s = append(c.cs.s, "OPTOUT")
	return (ClientTrackingOptout)(c)
}

func (c ClientTrackingPrefix) Noloop() ClientTrackingNoloop {
	c.cs.s = append(c.cs.s, "NOLOOP")
	return (ClientTrackingNoloop)(c)
}

func (c ClientTrackingPrefix) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type ClientTrackingRedirect Completed

func (c ClientTrackingRedirect) Prefix() ClientTrackingPrefix {
	return (ClientTrackingPrefix)(c)
}

func (c ClientTrackingRedirect) Bcast() ClientTrackingBcast {
	c.cs.s = append(c.cs.s, "BCAST")
	return (ClientTrackingBcast)(c)
}

func (c ClientTrackingRedirect) Optin() ClientTrackingOptin {
	c.cs.s = append(c.cs.s, "OPTIN")
	return (ClientTrackingOptin)(c)
}

func (c ClientTrackingRedirect) Optout() ClientTrackingOptout {
	c.cs.s = append(c.cs.s, "OPTOUT")
	return (ClientTrackingOptout)(c)
}

func (c ClientTrackingRedirect) Noloop() ClientTrackingNoloop {
	c.cs.s = append(c.cs.s, "NOLOOP")
	return (ClientTrackingNoloop)(c)
}

func (c ClientTrackingRedirect) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type ClientTrackingStatusOff Completed

func (c ClientTrackingStatusOff) Redirect(clientId int64) ClientTrackingRedirect {
	c.cs.s = append(c.cs.s, "REDIRECT", strconv.FormatInt(clientId, 10))
	return (ClientTrackingRedirect)(c)
}

func (c ClientTrackingStatusOff) Prefix() ClientTrackingPrefix {
	return (ClientTrackingPrefix)(c)
}

func (c ClientTrackingStatusOff) Bcast() ClientTrackingBcast {
	c.cs.s = append(c.cs.s, "BCAST")
	return (ClientTrackingBcast)(c)
}

func (c ClientTrackingStatusOff) Optin() ClientTrackingOptin {
	c.cs.s = append(c.cs.s, "OPTIN")
	return (ClientTrackingOptin)(c)
}

func (c ClientTrackingStatusOff) Optout() ClientTrackingOptout {
	c.cs.s = append(c.cs.s, "OPTOUT")
	return (ClientTrackingOptout)(c)
}

func (c ClientTrackingStatusOff) Noloop() ClientTrackingNoloop {
	c.cs.s = append(c.cs.s, "NOLOOP")
	return (ClientTrackingNoloop)(c)
}

func (c ClientTrackingStatusOff) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type ClientTrackingStatusOn Completed

func (c ClientTrackingStatusOn) Redirect(clientId int64) ClientTrackingRedirect {
	c.cs.s = append(c.cs.s, "REDIRECT", strconv.FormatInt(clientId, 10))
	return (ClientTrackingRedirect)(c)
}

func (c ClientTrackingStatusOn) Prefix() ClientTrackingPrefix {
	return (ClientTrackingPrefix)(c)
}

func (c ClientTrackingStatusOn) Bcast() ClientTrackingBcast {
	c.cs.s = append(c.cs.s, "BCAST")
	return (ClientTrackingBcast)(c)
}

func (c ClientTrackingStatusOn) Optin() ClientTrackingOptin {
	c.cs.s = append(c.cs.s, "OPTIN")
	return (ClientTrackingOptin)(c)
}

func (c ClientTrackingStatusOn) Optout() ClientTrackingOptout {
	c.cs.s = append(c.cs.s, "OPTOUT")
	return (ClientTrackingOptout)(c)
}

func (c ClientTrackingStatusOn) Noloop() ClientTrackingNoloop {
	c.cs.s = append(c.cs.s, "NOLOOP")
	return (ClientTrackingNoloop)(c)
}

func (c ClientTrackingStatusOn) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type ClientTrackinginfo Completed

func (b Builder) ClientTrackinginfo() (c ClientTrackinginfo) {
	c = ClientTrackinginfo{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "CLIENT", "TRACKINGINFO")
	return c
}

func (c ClientTrackinginfo) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type ClientUnblock Completed

func (b Builder) ClientUnblock() (c ClientUnblock) {
	c = ClientUnblock{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "CLIENT", "UNBLOCK")
	return c
}

func (c ClientUnblock) ClientId(clientId int64) ClientUnblockClientId {
	c.cs.s = append(c.cs.s, strconv.FormatInt(clientId, 10))
	return (ClientUnblockClientId)(c)
}

type ClientUnblockClientId Completed

func (c ClientUnblockClientId) Timeout() ClientUnblockUnblockTypeTimeout {
	c.cs.s = append(c.cs.s, "TIMEOUT")
	return (ClientUnblockUnblockTypeTimeout)(c)
}

func (c ClientUnblockClientId) Error() ClientUnblockUnblockTypeError {
	c.cs.s = append(c.cs.s, "ERROR")
	return (ClientUnblockUnblockTypeError)(c)
}

func (c ClientUnblockClientId) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type ClientUnblockUnblockTypeError Completed

func (c ClientUnblockUnblockTypeError) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type ClientUnblockUnblockTypeTimeout Completed

func (c ClientUnblockUnblockTypeTimeout) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type ClientUnpause Completed

func (b Builder) ClientUnpause() (c ClientUnpause) {
	c = ClientUnpause{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "CLIENT", "UNPAUSE")
	return c
}

func (c ClientUnpause) Build() Completed {
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

type Decr Completed

func (b Builder) Decr() (c Decr) {
	c = Decr{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "DECR")
	return c
}

func (c Decr) Key(key string) DecrKey {
	if c.ks&NoSlot == NoSlot {
		c.ks = NoSlot | slot(key)
	} else {
		c.ks = check(c.ks, slot(key))
	}
	c.cs.s = append(c.cs.s, key)
	return (DecrKey)(c)
}

type DecrKey Completed

func (c DecrKey) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type Decrby Completed

func (b Builder) Decrby() (c Decrby) {
	c = Decrby{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "DECRBY")
	return c
}

func (c Decrby) Key(key string) DecrbyKey {
	if c.ks&NoSlot == NoSlot {
		c.ks = NoSlot | slot(key)
	} else {
		c.ks = check(c.ks, slot(key))
	}
	c.cs.s = append(c.cs.s, key)
	return (DecrbyKey)(c)
}

type DecrbyDecrement Completed

func (c DecrbyDecrement) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type DecrbyKey Completed

func (c DecrbyKey) Decrement(decrement int64) DecrbyDecrement {
	c.cs.s = append(c.cs.s, strconv.FormatInt(decrement, 10))
	return (DecrbyDecrement)(c)
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

type Echo Completed

func (b Builder) Echo() (c Echo) {
	c = Echo{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "ECHO")
	return c
}

func (c Echo) Message(message string) EchoMessage {
	c.cs.s = append(c.cs.s, message)
	return (EchoMessage)(c)
}

type EchoMessage Completed

func (c EchoMessage) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

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
	c = EvalRo{cs: get(), ks: b.ks, cf: readonly}
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
	c = EvalshaRo{cs: get(), ks: b.ks, cf: readonly}
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
	c = FcallRo{cs: get(), ks: b.ks, cf: readonly}
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

type FtAggregate Completed

func (b Builder) FtAggregate() (c FtAggregate) {
	c = FtAggregate{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "FT.AGGREGATE")
	return c
}

func (c FtAggregate) Index(index string) FtAggregateIndex {
	c.cs.s = append(c.cs.s, index)
	return (FtAggregateIndex)(c)
}

type FtAggregateCursorCount Completed

func (c FtAggregateCursorCount) Maxidle(idleTime int64) FtAggregateCursorMaxidle {
	c.cs.s = append(c.cs.s, "MAXIDLE", strconv.FormatInt(idleTime, 10))
	return (FtAggregateCursorMaxidle)(c)
}

func (c FtAggregateCursorCount) Params() FtAggregateParamsParams {
	c.cs.s = append(c.cs.s, "PARAMS")
	return (FtAggregateParamsParams)(c)
}

func (c FtAggregateCursorCount) Dialect(dialect int64) FtAggregateDialect {
	c.cs.s = append(c.cs.s, "DIALECT", strconv.FormatInt(dialect, 10))
	return (FtAggregateDialect)(c)
}

func (c FtAggregateCursorCount) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type FtAggregateCursorMaxidle Completed

func (c FtAggregateCursorMaxidle) Params() FtAggregateParamsParams {
	c.cs.s = append(c.cs.s, "PARAMS")
	return (FtAggregateParamsParams)(c)
}

func (c FtAggregateCursorMaxidle) Dialect(dialect int64) FtAggregateDialect {
	c.cs.s = append(c.cs.s, "DIALECT", strconv.FormatInt(dialect, 10))
	return (FtAggregateDialect)(c)
}

func (c FtAggregateCursorMaxidle) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type FtAggregateCursorWithcursor Completed

func (c FtAggregateCursorWithcursor) Count(readSize int64) FtAggregateCursorCount {
	c.cs.s = append(c.cs.s, "COUNT", strconv.FormatInt(readSize, 10))
	return (FtAggregateCursorCount)(c)
}

func (c FtAggregateCursorWithcursor) Maxidle(idleTime int64) FtAggregateCursorMaxidle {
	c.cs.s = append(c.cs.s, "MAXIDLE", strconv.FormatInt(idleTime, 10))
	return (FtAggregateCursorMaxidle)(c)
}

func (c FtAggregateCursorWithcursor) Params() FtAggregateParamsParams {
	c.cs.s = append(c.cs.s, "PARAMS")
	return (FtAggregateParamsParams)(c)
}

func (c FtAggregateCursorWithcursor) Dialect(dialect int64) FtAggregateDialect {
	c.cs.s = append(c.cs.s, "DIALECT", strconv.FormatInt(dialect, 10))
	return (FtAggregateDialect)(c)
}

func (c FtAggregateCursorWithcursor) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type FtAggregateDialect Completed

func (c FtAggregateDialect) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type FtAggregateIndex Completed

func (c FtAggregateIndex) Query(query string) FtAggregateQuery {
	c.cs.s = append(c.cs.s, query)
	return (FtAggregateQuery)(c)
}

type FtAggregateOpApplyApply Completed

func (c FtAggregateOpApplyApply) As(name string) FtAggregateOpApplyAs {
	c.cs.s = append(c.cs.s, "AS", name)
	return (FtAggregateOpApplyAs)(c)
}

type FtAggregateOpApplyAs Completed

func (c FtAggregateOpApplyAs) Apply(expression string) FtAggregateOpApplyApply {
	c.cs.s = append(c.cs.s, "APPLY", expression)
	return (FtAggregateOpApplyApply)(c)
}

func (c FtAggregateOpApplyAs) Limit() FtAggregateOpLimitLimit {
	c.cs.s = append(c.cs.s, "LIMIT")
	return (FtAggregateOpLimitLimit)(c)
}

func (c FtAggregateOpApplyAs) Filter(filter string) FtAggregateOpFilter {
	c.cs.s = append(c.cs.s, "FILTER", filter)
	return (FtAggregateOpFilter)(c)
}

func (c FtAggregateOpApplyAs) LoadAll() FtAggregateOpLoadallLoadAll {
	c.cs.s = append(c.cs.s, "LOAD", "*")
	return (FtAggregateOpLoadallLoadAll)(c)
}

func (c FtAggregateOpApplyAs) Load(count int64) FtAggregateOpLoadLoad {
	c.cs.s = append(c.cs.s, "LOAD", strconv.FormatInt(count, 10))
	return (FtAggregateOpLoadLoad)(c)
}

func (c FtAggregateOpApplyAs) Groupby(nargs int64) FtAggregateOpGroupbyGroupby {
	c.cs.s = append(c.cs.s, "GROUPBY", strconv.FormatInt(nargs, 10))
	return (FtAggregateOpGroupbyGroupby)(c)
}

func (c FtAggregateOpApplyAs) Sortby(nargs int64) FtAggregateOpSortbySortby {
	c.cs.s = append(c.cs.s, "SORTBY", strconv.FormatInt(nargs, 10))
	return (FtAggregateOpSortbySortby)(c)
}

func (c FtAggregateOpApplyAs) Withcursor() FtAggregateCursorWithcursor {
	c.cs.s = append(c.cs.s, "WITHCURSOR")
	return (FtAggregateCursorWithcursor)(c)
}

func (c FtAggregateOpApplyAs) Params() FtAggregateParamsParams {
	c.cs.s = append(c.cs.s, "PARAMS")
	return (FtAggregateParamsParams)(c)
}

func (c FtAggregateOpApplyAs) Dialect(dialect int64) FtAggregateDialect {
	c.cs.s = append(c.cs.s, "DIALECT", strconv.FormatInt(dialect, 10))
	return (FtAggregateDialect)(c)
}

func (c FtAggregateOpApplyAs) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type FtAggregateOpFilter Completed

func (c FtAggregateOpFilter) LoadAll() FtAggregateOpLoadallLoadAll {
	c.cs.s = append(c.cs.s, "LOAD", "*")
	return (FtAggregateOpLoadallLoadAll)(c)
}

func (c FtAggregateOpFilter) Load(count int64) FtAggregateOpLoadLoad {
	c.cs.s = append(c.cs.s, "LOAD", strconv.FormatInt(count, 10))
	return (FtAggregateOpLoadLoad)(c)
}

func (c FtAggregateOpFilter) Groupby(nargs int64) FtAggregateOpGroupbyGroupby {
	c.cs.s = append(c.cs.s, "GROUPBY", strconv.FormatInt(nargs, 10))
	return (FtAggregateOpGroupbyGroupby)(c)
}

func (c FtAggregateOpFilter) Sortby(nargs int64) FtAggregateOpSortbySortby {
	c.cs.s = append(c.cs.s, "SORTBY", strconv.FormatInt(nargs, 10))
	return (FtAggregateOpSortbySortby)(c)
}

func (c FtAggregateOpFilter) Apply(expression string) FtAggregateOpApplyApply {
	c.cs.s = append(c.cs.s, "APPLY", expression)
	return (FtAggregateOpApplyApply)(c)
}

func (c FtAggregateOpFilter) Limit() FtAggregateOpLimitLimit {
	c.cs.s = append(c.cs.s, "LIMIT")
	return (FtAggregateOpLimitLimit)(c)
}

func (c FtAggregateOpFilter) Filter(filter string) FtAggregateOpFilter {
	c.cs.s = append(c.cs.s, "FILTER", filter)
	return c
}

func (c FtAggregateOpFilter) Withcursor() FtAggregateCursorWithcursor {
	c.cs.s = append(c.cs.s, "WITHCURSOR")
	return (FtAggregateCursorWithcursor)(c)
}

func (c FtAggregateOpFilter) Params() FtAggregateParamsParams {
	c.cs.s = append(c.cs.s, "PARAMS")
	return (FtAggregateParamsParams)(c)
}

func (c FtAggregateOpFilter) Dialect(dialect int64) FtAggregateDialect {
	c.cs.s = append(c.cs.s, "DIALECT", strconv.FormatInt(dialect, 10))
	return (FtAggregateDialect)(c)
}

func (c FtAggregateOpFilter) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type FtAggregateOpGroupbyGroupby Completed

func (c FtAggregateOpGroupbyGroupby) Property(property ...string) FtAggregateOpGroupbyProperty {
	c.cs.s = append(c.cs.s, property...)
	return (FtAggregateOpGroupbyProperty)(c)
}

func (c FtAggregateOpGroupbyGroupby) Reduce(function string) FtAggregateOpGroupbyReduceReduce {
	c.cs.s = append(c.cs.s, "REDUCE", function)
	return (FtAggregateOpGroupbyReduceReduce)(c)
}

func (c FtAggregateOpGroupbyGroupby) Groupby(nargs int64) FtAggregateOpGroupbyGroupby {
	c.cs.s = append(c.cs.s, "GROUPBY", strconv.FormatInt(nargs, 10))
	return c
}

func (c FtAggregateOpGroupbyGroupby) Sortby(nargs int64) FtAggregateOpSortbySortby {
	c.cs.s = append(c.cs.s, "SORTBY", strconv.FormatInt(nargs, 10))
	return (FtAggregateOpSortbySortby)(c)
}

func (c FtAggregateOpGroupbyGroupby) Apply(expression string) FtAggregateOpApplyApply {
	c.cs.s = append(c.cs.s, "APPLY", expression)
	return (FtAggregateOpApplyApply)(c)
}

func (c FtAggregateOpGroupbyGroupby) Limit() FtAggregateOpLimitLimit {
	c.cs.s = append(c.cs.s, "LIMIT")
	return (FtAggregateOpLimitLimit)(c)
}

func (c FtAggregateOpGroupbyGroupby) Filter(filter string) FtAggregateOpFilter {
	c.cs.s = append(c.cs.s, "FILTER", filter)
	return (FtAggregateOpFilter)(c)
}

func (c FtAggregateOpGroupbyGroupby) LoadAll() FtAggregateOpLoadallLoadAll {
	c.cs.s = append(c.cs.s, "LOAD", "*")
	return (FtAggregateOpLoadallLoadAll)(c)
}

func (c FtAggregateOpGroupbyGroupby) Load(count int64) FtAggregateOpLoadLoad {
	c.cs.s = append(c.cs.s, "LOAD", strconv.FormatInt(count, 10))
	return (FtAggregateOpLoadLoad)(c)
}

func (c FtAggregateOpGroupbyGroupby) Withcursor() FtAggregateCursorWithcursor {
	c.cs.s = append(c.cs.s, "WITHCURSOR")
	return (FtAggregateCursorWithcursor)(c)
}

func (c FtAggregateOpGroupbyGroupby) Params() FtAggregateParamsParams {
	c.cs.s = append(c.cs.s, "PARAMS")
	return (FtAggregateParamsParams)(c)
}

func (c FtAggregateOpGroupbyGroupby) Dialect(dialect int64) FtAggregateDialect {
	c.cs.s = append(c.cs.s, "DIALECT", strconv.FormatInt(dialect, 10))
	return (FtAggregateDialect)(c)
}

func (c FtAggregateOpGroupbyGroupby) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type FtAggregateOpGroupbyProperty Completed

func (c FtAggregateOpGroupbyProperty) Property(property ...string) FtAggregateOpGroupbyProperty {
	c.cs.s = append(c.cs.s, property...)
	return c
}

func (c FtAggregateOpGroupbyProperty) Reduce(function string) FtAggregateOpGroupbyReduceReduce {
	c.cs.s = append(c.cs.s, "REDUCE", function)
	return (FtAggregateOpGroupbyReduceReduce)(c)
}

func (c FtAggregateOpGroupbyProperty) Groupby(nargs int64) FtAggregateOpGroupbyGroupby {
	c.cs.s = append(c.cs.s, "GROUPBY", strconv.FormatInt(nargs, 10))
	return (FtAggregateOpGroupbyGroupby)(c)
}

func (c FtAggregateOpGroupbyProperty) Sortby(nargs int64) FtAggregateOpSortbySortby {
	c.cs.s = append(c.cs.s, "SORTBY", strconv.FormatInt(nargs, 10))
	return (FtAggregateOpSortbySortby)(c)
}

func (c FtAggregateOpGroupbyProperty) Apply(expression string) FtAggregateOpApplyApply {
	c.cs.s = append(c.cs.s, "APPLY", expression)
	return (FtAggregateOpApplyApply)(c)
}

func (c FtAggregateOpGroupbyProperty) Limit() FtAggregateOpLimitLimit {
	c.cs.s = append(c.cs.s, "LIMIT")
	return (FtAggregateOpLimitLimit)(c)
}

func (c FtAggregateOpGroupbyProperty) Filter(filter string) FtAggregateOpFilter {
	c.cs.s = append(c.cs.s, "FILTER", filter)
	return (FtAggregateOpFilter)(c)
}

func (c FtAggregateOpGroupbyProperty) LoadAll() FtAggregateOpLoadallLoadAll {
	c.cs.s = append(c.cs.s, "LOAD", "*")
	return (FtAggregateOpLoadallLoadAll)(c)
}

func (c FtAggregateOpGroupbyProperty) Load(count int64) FtAggregateOpLoadLoad {
	c.cs.s = append(c.cs.s, "LOAD", strconv.FormatInt(count, 10))
	return (FtAggregateOpLoadLoad)(c)
}

func (c FtAggregateOpGroupbyProperty) Withcursor() FtAggregateCursorWithcursor {
	c.cs.s = append(c.cs.s, "WITHCURSOR")
	return (FtAggregateCursorWithcursor)(c)
}

func (c FtAggregateOpGroupbyProperty) Params() FtAggregateParamsParams {
	c.cs.s = append(c.cs.s, "PARAMS")
	return (FtAggregateParamsParams)(c)
}

func (c FtAggregateOpGroupbyProperty) Dialect(dialect int64) FtAggregateDialect {
	c.cs.s = append(c.cs.s, "DIALECT", strconv.FormatInt(dialect, 10))
	return (FtAggregateDialect)(c)
}

func (c FtAggregateOpGroupbyProperty) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type FtAggregateOpGroupbyReduceArg Completed

func (c FtAggregateOpGroupbyReduceArg) Arg(arg ...string) FtAggregateOpGroupbyReduceArg {
	c.cs.s = append(c.cs.s, arg...)
	return c
}

func (c FtAggregateOpGroupbyReduceArg) As(name string) FtAggregateOpGroupbyReduceAs {
	c.cs.s = append(c.cs.s, "AS", name)
	return (FtAggregateOpGroupbyReduceAs)(c)
}

func (c FtAggregateOpGroupbyReduceArg) By(by string) FtAggregateOpGroupbyReduceBy {
	c.cs.s = append(c.cs.s, "BY", by)
	return (FtAggregateOpGroupbyReduceBy)(c)
}

func (c FtAggregateOpGroupbyReduceArg) Asc() FtAggregateOpGroupbyReduceOrderAsc {
	c.cs.s = append(c.cs.s, "ASC")
	return (FtAggregateOpGroupbyReduceOrderAsc)(c)
}

func (c FtAggregateOpGroupbyReduceArg) Desc() FtAggregateOpGroupbyReduceOrderDesc {
	c.cs.s = append(c.cs.s, "DESC")
	return (FtAggregateOpGroupbyReduceOrderDesc)(c)
}

func (c FtAggregateOpGroupbyReduceArg) Reduce(function string) FtAggregateOpGroupbyReduceReduce {
	c.cs.s = append(c.cs.s, "REDUCE", function)
	return (FtAggregateOpGroupbyReduceReduce)(c)
}

func (c FtAggregateOpGroupbyReduceArg) Groupby(nargs int64) FtAggregateOpGroupbyGroupby {
	c.cs.s = append(c.cs.s, "GROUPBY", strconv.FormatInt(nargs, 10))
	return (FtAggregateOpGroupbyGroupby)(c)
}

func (c FtAggregateOpGroupbyReduceArg) Sortby(nargs int64) FtAggregateOpSortbySortby {
	c.cs.s = append(c.cs.s, "SORTBY", strconv.FormatInt(nargs, 10))
	return (FtAggregateOpSortbySortby)(c)
}

func (c FtAggregateOpGroupbyReduceArg) Apply(expression string) FtAggregateOpApplyApply {
	c.cs.s = append(c.cs.s, "APPLY", expression)
	return (FtAggregateOpApplyApply)(c)
}

func (c FtAggregateOpGroupbyReduceArg) Limit() FtAggregateOpLimitLimit {
	c.cs.s = append(c.cs.s, "LIMIT")
	return (FtAggregateOpLimitLimit)(c)
}

func (c FtAggregateOpGroupbyReduceArg) Filter(filter string) FtAggregateOpFilter {
	c.cs.s = append(c.cs.s, "FILTER", filter)
	return (FtAggregateOpFilter)(c)
}

func (c FtAggregateOpGroupbyReduceArg) LoadAll() FtAggregateOpLoadallLoadAll {
	c.cs.s = append(c.cs.s, "LOAD", "*")
	return (FtAggregateOpLoadallLoadAll)(c)
}

func (c FtAggregateOpGroupbyReduceArg) Load(count int64) FtAggregateOpLoadLoad {
	c.cs.s = append(c.cs.s, "LOAD", strconv.FormatInt(count, 10))
	return (FtAggregateOpLoadLoad)(c)
}

func (c FtAggregateOpGroupbyReduceArg) Withcursor() FtAggregateCursorWithcursor {
	c.cs.s = append(c.cs.s, "WITHCURSOR")
	return (FtAggregateCursorWithcursor)(c)
}

func (c FtAggregateOpGroupbyReduceArg) Params() FtAggregateParamsParams {
	c.cs.s = append(c.cs.s, "PARAMS")
	return (FtAggregateParamsParams)(c)
}

func (c FtAggregateOpGroupbyReduceArg) Dialect(dialect int64) FtAggregateDialect {
	c.cs.s = append(c.cs.s, "DIALECT", strconv.FormatInt(dialect, 10))
	return (FtAggregateDialect)(c)
}

func (c FtAggregateOpGroupbyReduceArg) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type FtAggregateOpGroupbyReduceAs Completed

func (c FtAggregateOpGroupbyReduceAs) By(by string) FtAggregateOpGroupbyReduceBy {
	c.cs.s = append(c.cs.s, "BY", by)
	return (FtAggregateOpGroupbyReduceBy)(c)
}

func (c FtAggregateOpGroupbyReduceAs) Asc() FtAggregateOpGroupbyReduceOrderAsc {
	c.cs.s = append(c.cs.s, "ASC")
	return (FtAggregateOpGroupbyReduceOrderAsc)(c)
}

func (c FtAggregateOpGroupbyReduceAs) Desc() FtAggregateOpGroupbyReduceOrderDesc {
	c.cs.s = append(c.cs.s, "DESC")
	return (FtAggregateOpGroupbyReduceOrderDesc)(c)
}

func (c FtAggregateOpGroupbyReduceAs) Reduce(function string) FtAggregateOpGroupbyReduceReduce {
	c.cs.s = append(c.cs.s, "REDUCE", function)
	return (FtAggregateOpGroupbyReduceReduce)(c)
}

func (c FtAggregateOpGroupbyReduceAs) Groupby(nargs int64) FtAggregateOpGroupbyGroupby {
	c.cs.s = append(c.cs.s, "GROUPBY", strconv.FormatInt(nargs, 10))
	return (FtAggregateOpGroupbyGroupby)(c)
}

func (c FtAggregateOpGroupbyReduceAs) Sortby(nargs int64) FtAggregateOpSortbySortby {
	c.cs.s = append(c.cs.s, "SORTBY", strconv.FormatInt(nargs, 10))
	return (FtAggregateOpSortbySortby)(c)
}

func (c FtAggregateOpGroupbyReduceAs) Apply(expression string) FtAggregateOpApplyApply {
	c.cs.s = append(c.cs.s, "APPLY", expression)
	return (FtAggregateOpApplyApply)(c)
}

func (c FtAggregateOpGroupbyReduceAs) Limit() FtAggregateOpLimitLimit {
	c.cs.s = append(c.cs.s, "LIMIT")
	return (FtAggregateOpLimitLimit)(c)
}

func (c FtAggregateOpGroupbyReduceAs) Filter(filter string) FtAggregateOpFilter {
	c.cs.s = append(c.cs.s, "FILTER", filter)
	return (FtAggregateOpFilter)(c)
}

func (c FtAggregateOpGroupbyReduceAs) LoadAll() FtAggregateOpLoadallLoadAll {
	c.cs.s = append(c.cs.s, "LOAD", "*")
	return (FtAggregateOpLoadallLoadAll)(c)
}

func (c FtAggregateOpGroupbyReduceAs) Load(count int64) FtAggregateOpLoadLoad {
	c.cs.s = append(c.cs.s, "LOAD", strconv.FormatInt(count, 10))
	return (FtAggregateOpLoadLoad)(c)
}

func (c FtAggregateOpGroupbyReduceAs) Withcursor() FtAggregateCursorWithcursor {
	c.cs.s = append(c.cs.s, "WITHCURSOR")
	return (FtAggregateCursorWithcursor)(c)
}

func (c FtAggregateOpGroupbyReduceAs) Params() FtAggregateParamsParams {
	c.cs.s = append(c.cs.s, "PARAMS")
	return (FtAggregateParamsParams)(c)
}

func (c FtAggregateOpGroupbyReduceAs) Dialect(dialect int64) FtAggregateDialect {
	c.cs.s = append(c.cs.s, "DIALECT", strconv.FormatInt(dialect, 10))
	return (FtAggregateDialect)(c)
}

func (c FtAggregateOpGroupbyReduceAs) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type FtAggregateOpGroupbyReduceBy Completed

func (c FtAggregateOpGroupbyReduceBy) Asc() FtAggregateOpGroupbyReduceOrderAsc {
	c.cs.s = append(c.cs.s, "ASC")
	return (FtAggregateOpGroupbyReduceOrderAsc)(c)
}

func (c FtAggregateOpGroupbyReduceBy) Desc() FtAggregateOpGroupbyReduceOrderDesc {
	c.cs.s = append(c.cs.s, "DESC")
	return (FtAggregateOpGroupbyReduceOrderDesc)(c)
}

func (c FtAggregateOpGroupbyReduceBy) Reduce(function string) FtAggregateOpGroupbyReduceReduce {
	c.cs.s = append(c.cs.s, "REDUCE", function)
	return (FtAggregateOpGroupbyReduceReduce)(c)
}

func (c FtAggregateOpGroupbyReduceBy) Groupby(nargs int64) FtAggregateOpGroupbyGroupby {
	c.cs.s = append(c.cs.s, "GROUPBY", strconv.FormatInt(nargs, 10))
	return (FtAggregateOpGroupbyGroupby)(c)
}

func (c FtAggregateOpGroupbyReduceBy) Sortby(nargs int64) FtAggregateOpSortbySortby {
	c.cs.s = append(c.cs.s, "SORTBY", strconv.FormatInt(nargs, 10))
	return (FtAggregateOpSortbySortby)(c)
}

func (c FtAggregateOpGroupbyReduceBy) Apply(expression string) FtAggregateOpApplyApply {
	c.cs.s = append(c.cs.s, "APPLY", expression)
	return (FtAggregateOpApplyApply)(c)
}

func (c FtAggregateOpGroupbyReduceBy) Limit() FtAggregateOpLimitLimit {
	c.cs.s = append(c.cs.s, "LIMIT")
	return (FtAggregateOpLimitLimit)(c)
}

func (c FtAggregateOpGroupbyReduceBy) Filter(filter string) FtAggregateOpFilter {
	c.cs.s = append(c.cs.s, "FILTER", filter)
	return (FtAggregateOpFilter)(c)
}

func (c FtAggregateOpGroupbyReduceBy) LoadAll() FtAggregateOpLoadallLoadAll {
	c.cs.s = append(c.cs.s, "LOAD", "*")
	return (FtAggregateOpLoadallLoadAll)(c)
}

func (c FtAggregateOpGroupbyReduceBy) Load(count int64) FtAggregateOpLoadLoad {
	c.cs.s = append(c.cs.s, "LOAD", strconv.FormatInt(count, 10))
	return (FtAggregateOpLoadLoad)(c)
}

func (c FtAggregateOpGroupbyReduceBy) Withcursor() FtAggregateCursorWithcursor {
	c.cs.s = append(c.cs.s, "WITHCURSOR")
	return (FtAggregateCursorWithcursor)(c)
}

func (c FtAggregateOpGroupbyReduceBy) Params() FtAggregateParamsParams {
	c.cs.s = append(c.cs.s, "PARAMS")
	return (FtAggregateParamsParams)(c)
}

func (c FtAggregateOpGroupbyReduceBy) Dialect(dialect int64) FtAggregateDialect {
	c.cs.s = append(c.cs.s, "DIALECT", strconv.FormatInt(dialect, 10))
	return (FtAggregateDialect)(c)
}

func (c FtAggregateOpGroupbyReduceBy) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type FtAggregateOpGroupbyReduceNargs Completed

func (c FtAggregateOpGroupbyReduceNargs) Arg(arg ...string) FtAggregateOpGroupbyReduceArg {
	c.cs.s = append(c.cs.s, arg...)
	return (FtAggregateOpGroupbyReduceArg)(c)
}

func (c FtAggregateOpGroupbyReduceNargs) As(name string) FtAggregateOpGroupbyReduceAs {
	c.cs.s = append(c.cs.s, "AS", name)
	return (FtAggregateOpGroupbyReduceAs)(c)
}

func (c FtAggregateOpGroupbyReduceNargs) By(by string) FtAggregateOpGroupbyReduceBy {
	c.cs.s = append(c.cs.s, "BY", by)
	return (FtAggregateOpGroupbyReduceBy)(c)
}

func (c FtAggregateOpGroupbyReduceNargs) Asc() FtAggregateOpGroupbyReduceOrderAsc {
	c.cs.s = append(c.cs.s, "ASC")
	return (FtAggregateOpGroupbyReduceOrderAsc)(c)
}

func (c FtAggregateOpGroupbyReduceNargs) Desc() FtAggregateOpGroupbyReduceOrderDesc {
	c.cs.s = append(c.cs.s, "DESC")
	return (FtAggregateOpGroupbyReduceOrderDesc)(c)
}

func (c FtAggregateOpGroupbyReduceNargs) Reduce(function string) FtAggregateOpGroupbyReduceReduce {
	c.cs.s = append(c.cs.s, "REDUCE", function)
	return (FtAggregateOpGroupbyReduceReduce)(c)
}

func (c FtAggregateOpGroupbyReduceNargs) Groupby(nargs int64) FtAggregateOpGroupbyGroupby {
	c.cs.s = append(c.cs.s, "GROUPBY", strconv.FormatInt(nargs, 10))
	return (FtAggregateOpGroupbyGroupby)(c)
}

func (c FtAggregateOpGroupbyReduceNargs) Sortby(nargs int64) FtAggregateOpSortbySortby {
	c.cs.s = append(c.cs.s, "SORTBY", strconv.FormatInt(nargs, 10))
	return (FtAggregateOpSortbySortby)(c)
}

func (c FtAggregateOpGroupbyReduceNargs) Apply(expression string) FtAggregateOpApplyApply {
	c.cs.s = append(c.cs.s, "APPLY", expression)
	return (FtAggregateOpApplyApply)(c)
}

func (c FtAggregateOpGroupbyReduceNargs) Limit() FtAggregateOpLimitLimit {
	c.cs.s = append(c.cs.s, "LIMIT")
	return (FtAggregateOpLimitLimit)(c)
}

func (c FtAggregateOpGroupbyReduceNargs) Filter(filter string) FtAggregateOpFilter {
	c.cs.s = append(c.cs.s, "FILTER", filter)
	return (FtAggregateOpFilter)(c)
}

func (c FtAggregateOpGroupbyReduceNargs) LoadAll() FtAggregateOpLoadallLoadAll {
	c.cs.s = append(c.cs.s, "LOAD", "*")
	return (FtAggregateOpLoadallLoadAll)(c)
}

func (c FtAggregateOpGroupbyReduceNargs) Load(count int64) FtAggregateOpLoadLoad {
	c.cs.s = append(c.cs.s, "LOAD", strconv.FormatInt(count, 10))
	return (FtAggregateOpLoadLoad)(c)
}

func (c FtAggregateOpGroupbyReduceNargs) Withcursor() FtAggregateCursorWithcursor {
	c.cs.s = append(c.cs.s, "WITHCURSOR")
	return (FtAggregateCursorWithcursor)(c)
}

func (c FtAggregateOpGroupbyReduceNargs) Params() FtAggregateParamsParams {
	c.cs.s = append(c.cs.s, "PARAMS")
	return (FtAggregateParamsParams)(c)
}

func (c FtAggregateOpGroupbyReduceNargs) Dialect(dialect int64) FtAggregateDialect {
	c.cs.s = append(c.cs.s, "DIALECT", strconv.FormatInt(dialect, 10))
	return (FtAggregateDialect)(c)
}

func (c FtAggregateOpGroupbyReduceNargs) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type FtAggregateOpGroupbyReduceOrderAsc Completed

func (c FtAggregateOpGroupbyReduceOrderAsc) Reduce(function string) FtAggregateOpGroupbyReduceReduce {
	c.cs.s = append(c.cs.s, "REDUCE", function)
	return (FtAggregateOpGroupbyReduceReduce)(c)
}

func (c FtAggregateOpGroupbyReduceOrderAsc) Groupby(nargs int64) FtAggregateOpGroupbyGroupby {
	c.cs.s = append(c.cs.s, "GROUPBY", strconv.FormatInt(nargs, 10))
	return (FtAggregateOpGroupbyGroupby)(c)
}

func (c FtAggregateOpGroupbyReduceOrderAsc) Sortby(nargs int64) FtAggregateOpSortbySortby {
	c.cs.s = append(c.cs.s, "SORTBY", strconv.FormatInt(nargs, 10))
	return (FtAggregateOpSortbySortby)(c)
}

func (c FtAggregateOpGroupbyReduceOrderAsc) Apply(expression string) FtAggregateOpApplyApply {
	c.cs.s = append(c.cs.s, "APPLY", expression)
	return (FtAggregateOpApplyApply)(c)
}

func (c FtAggregateOpGroupbyReduceOrderAsc) Limit() FtAggregateOpLimitLimit {
	c.cs.s = append(c.cs.s, "LIMIT")
	return (FtAggregateOpLimitLimit)(c)
}

func (c FtAggregateOpGroupbyReduceOrderAsc) Filter(filter string) FtAggregateOpFilter {
	c.cs.s = append(c.cs.s, "FILTER", filter)
	return (FtAggregateOpFilter)(c)
}

func (c FtAggregateOpGroupbyReduceOrderAsc) LoadAll() FtAggregateOpLoadallLoadAll {
	c.cs.s = append(c.cs.s, "LOAD", "*")
	return (FtAggregateOpLoadallLoadAll)(c)
}

func (c FtAggregateOpGroupbyReduceOrderAsc) Load(count int64) FtAggregateOpLoadLoad {
	c.cs.s = append(c.cs.s, "LOAD", strconv.FormatInt(count, 10))
	return (FtAggregateOpLoadLoad)(c)
}

func (c FtAggregateOpGroupbyReduceOrderAsc) Withcursor() FtAggregateCursorWithcursor {
	c.cs.s = append(c.cs.s, "WITHCURSOR")
	return (FtAggregateCursorWithcursor)(c)
}

func (c FtAggregateOpGroupbyReduceOrderAsc) Params() FtAggregateParamsParams {
	c.cs.s = append(c.cs.s, "PARAMS")
	return (FtAggregateParamsParams)(c)
}

func (c FtAggregateOpGroupbyReduceOrderAsc) Dialect(dialect int64) FtAggregateDialect {
	c.cs.s = append(c.cs.s, "DIALECT", strconv.FormatInt(dialect, 10))
	return (FtAggregateDialect)(c)
}

func (c FtAggregateOpGroupbyReduceOrderAsc) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type FtAggregateOpGroupbyReduceOrderDesc Completed

func (c FtAggregateOpGroupbyReduceOrderDesc) Reduce(function string) FtAggregateOpGroupbyReduceReduce {
	c.cs.s = append(c.cs.s, "REDUCE", function)
	return (FtAggregateOpGroupbyReduceReduce)(c)
}

func (c FtAggregateOpGroupbyReduceOrderDesc) Groupby(nargs int64) FtAggregateOpGroupbyGroupby {
	c.cs.s = append(c.cs.s, "GROUPBY", strconv.FormatInt(nargs, 10))
	return (FtAggregateOpGroupbyGroupby)(c)
}

func (c FtAggregateOpGroupbyReduceOrderDesc) Sortby(nargs int64) FtAggregateOpSortbySortby {
	c.cs.s = append(c.cs.s, "SORTBY", strconv.FormatInt(nargs, 10))
	return (FtAggregateOpSortbySortby)(c)
}

func (c FtAggregateOpGroupbyReduceOrderDesc) Apply(expression string) FtAggregateOpApplyApply {
	c.cs.s = append(c.cs.s, "APPLY", expression)
	return (FtAggregateOpApplyApply)(c)
}

func (c FtAggregateOpGroupbyReduceOrderDesc) Limit() FtAggregateOpLimitLimit {
	c.cs.s = append(c.cs.s, "LIMIT")
	return (FtAggregateOpLimitLimit)(c)
}

func (c FtAggregateOpGroupbyReduceOrderDesc) Filter(filter string) FtAggregateOpFilter {
	c.cs.s = append(c.cs.s, "FILTER", filter)
	return (FtAggregateOpFilter)(c)
}

func (c FtAggregateOpGroupbyReduceOrderDesc) LoadAll() FtAggregateOpLoadallLoadAll {
	c.cs.s = append(c.cs.s, "LOAD", "*")
	return (FtAggregateOpLoadallLoadAll)(c)
}

func (c FtAggregateOpGroupbyReduceOrderDesc) Load(count int64) FtAggregateOpLoadLoad {
	c.cs.s = append(c.cs.s, "LOAD", strconv.FormatInt(count, 10))
	return (FtAggregateOpLoadLoad)(c)
}

func (c FtAggregateOpGroupbyReduceOrderDesc) Withcursor() FtAggregateCursorWithcursor {
	c.cs.s = append(c.cs.s, "WITHCURSOR")
	return (FtAggregateCursorWithcursor)(c)
}

func (c FtAggregateOpGroupbyReduceOrderDesc) Params() FtAggregateParamsParams {
	c.cs.s = append(c.cs.s, "PARAMS")
	return (FtAggregateParamsParams)(c)
}

func (c FtAggregateOpGroupbyReduceOrderDesc) Dialect(dialect int64) FtAggregateDialect {
	c.cs.s = append(c.cs.s, "DIALECT", strconv.FormatInt(dialect, 10))
	return (FtAggregateDialect)(c)
}

func (c FtAggregateOpGroupbyReduceOrderDesc) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type FtAggregateOpGroupbyReduceReduce Completed

func (c FtAggregateOpGroupbyReduceReduce) Nargs(nargs int64) FtAggregateOpGroupbyReduceNargs {
	c.cs.s = append(c.cs.s, strconv.FormatInt(nargs, 10))
	return (FtAggregateOpGroupbyReduceNargs)(c)
}

type FtAggregateOpLimitLimit Completed

func (c FtAggregateOpLimitLimit) OffsetNum(offset int64, num int64) FtAggregateOpLimitOffsetNum {
	c.cs.s = append(c.cs.s, strconv.FormatInt(offset, 10), strconv.FormatInt(num, 10))
	return (FtAggregateOpLimitOffsetNum)(c)
}

type FtAggregateOpLimitOffsetNum Completed

func (c FtAggregateOpLimitOffsetNum) Filter(filter string) FtAggregateOpFilter {
	c.cs.s = append(c.cs.s, "FILTER", filter)
	return (FtAggregateOpFilter)(c)
}

func (c FtAggregateOpLimitOffsetNum) LoadAll() FtAggregateOpLoadallLoadAll {
	c.cs.s = append(c.cs.s, "LOAD", "*")
	return (FtAggregateOpLoadallLoadAll)(c)
}

func (c FtAggregateOpLimitOffsetNum) Load(count int64) FtAggregateOpLoadLoad {
	c.cs.s = append(c.cs.s, "LOAD", strconv.FormatInt(count, 10))
	return (FtAggregateOpLoadLoad)(c)
}

func (c FtAggregateOpLimitOffsetNum) Groupby(nargs int64) FtAggregateOpGroupbyGroupby {
	c.cs.s = append(c.cs.s, "GROUPBY", strconv.FormatInt(nargs, 10))
	return (FtAggregateOpGroupbyGroupby)(c)
}

func (c FtAggregateOpLimitOffsetNum) Sortby(nargs int64) FtAggregateOpSortbySortby {
	c.cs.s = append(c.cs.s, "SORTBY", strconv.FormatInt(nargs, 10))
	return (FtAggregateOpSortbySortby)(c)
}

func (c FtAggregateOpLimitOffsetNum) Apply(expression string) FtAggregateOpApplyApply {
	c.cs.s = append(c.cs.s, "APPLY", expression)
	return (FtAggregateOpApplyApply)(c)
}

func (c FtAggregateOpLimitOffsetNum) Limit() FtAggregateOpLimitLimit {
	c.cs.s = append(c.cs.s, "LIMIT")
	return (FtAggregateOpLimitLimit)(c)
}

func (c FtAggregateOpLimitOffsetNum) Withcursor() FtAggregateCursorWithcursor {
	c.cs.s = append(c.cs.s, "WITHCURSOR")
	return (FtAggregateCursorWithcursor)(c)
}

func (c FtAggregateOpLimitOffsetNum) Params() FtAggregateParamsParams {
	c.cs.s = append(c.cs.s, "PARAMS")
	return (FtAggregateParamsParams)(c)
}

func (c FtAggregateOpLimitOffsetNum) Dialect(dialect int64) FtAggregateDialect {
	c.cs.s = append(c.cs.s, "DIALECT", strconv.FormatInt(dialect, 10))
	return (FtAggregateDialect)(c)
}

func (c FtAggregateOpLimitOffsetNum) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type FtAggregateOpLoadField Completed

func (c FtAggregateOpLoadField) Field(field ...string) FtAggregateOpLoadField {
	c.cs.s = append(c.cs.s, field...)
	return c
}

func (c FtAggregateOpLoadField) Groupby(nargs int64) FtAggregateOpGroupbyGroupby {
	c.cs.s = append(c.cs.s, "GROUPBY", strconv.FormatInt(nargs, 10))
	return (FtAggregateOpGroupbyGroupby)(c)
}

func (c FtAggregateOpLoadField) Sortby(nargs int64) FtAggregateOpSortbySortby {
	c.cs.s = append(c.cs.s, "SORTBY", strconv.FormatInt(nargs, 10))
	return (FtAggregateOpSortbySortby)(c)
}

func (c FtAggregateOpLoadField) Apply(expression string) FtAggregateOpApplyApply {
	c.cs.s = append(c.cs.s, "APPLY", expression)
	return (FtAggregateOpApplyApply)(c)
}

func (c FtAggregateOpLoadField) Limit() FtAggregateOpLimitLimit {
	c.cs.s = append(c.cs.s, "LIMIT")
	return (FtAggregateOpLimitLimit)(c)
}

func (c FtAggregateOpLoadField) Filter(filter string) FtAggregateOpFilter {
	c.cs.s = append(c.cs.s, "FILTER", filter)
	return (FtAggregateOpFilter)(c)
}

func (c FtAggregateOpLoadField) LoadAll() FtAggregateOpLoadallLoadAll {
	c.cs.s = append(c.cs.s, "LOAD", "*")
	return (FtAggregateOpLoadallLoadAll)(c)
}

func (c FtAggregateOpLoadField) Load(count int64) FtAggregateOpLoadLoad {
	c.cs.s = append(c.cs.s, "LOAD", strconv.FormatInt(count, 10))
	return (FtAggregateOpLoadLoad)(c)
}

func (c FtAggregateOpLoadField) Withcursor() FtAggregateCursorWithcursor {
	c.cs.s = append(c.cs.s, "WITHCURSOR")
	return (FtAggregateCursorWithcursor)(c)
}

func (c FtAggregateOpLoadField) Params() FtAggregateParamsParams {
	c.cs.s = append(c.cs.s, "PARAMS")
	return (FtAggregateParamsParams)(c)
}

func (c FtAggregateOpLoadField) Dialect(dialect int64) FtAggregateDialect {
	c.cs.s = append(c.cs.s, "DIALECT", strconv.FormatInt(dialect, 10))
	return (FtAggregateDialect)(c)
}

func (c FtAggregateOpLoadField) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type FtAggregateOpLoadLoad Completed

func (c FtAggregateOpLoadLoad) Field(field ...string) FtAggregateOpLoadField {
	c.cs.s = append(c.cs.s, field...)
	return (FtAggregateOpLoadField)(c)
}

type FtAggregateOpLoadallLoadAll Completed

func (c FtAggregateOpLoadallLoadAll) Load(count int64) FtAggregateOpLoadLoad {
	c.cs.s = append(c.cs.s, "LOAD", strconv.FormatInt(count, 10))
	return (FtAggregateOpLoadLoad)(c)
}

func (c FtAggregateOpLoadallLoadAll) Groupby(nargs int64) FtAggregateOpGroupbyGroupby {
	c.cs.s = append(c.cs.s, "GROUPBY", strconv.FormatInt(nargs, 10))
	return (FtAggregateOpGroupbyGroupby)(c)
}

func (c FtAggregateOpLoadallLoadAll) Sortby(nargs int64) FtAggregateOpSortbySortby {
	c.cs.s = append(c.cs.s, "SORTBY", strconv.FormatInt(nargs, 10))
	return (FtAggregateOpSortbySortby)(c)
}

func (c FtAggregateOpLoadallLoadAll) Apply(expression string) FtAggregateOpApplyApply {
	c.cs.s = append(c.cs.s, "APPLY", expression)
	return (FtAggregateOpApplyApply)(c)
}

func (c FtAggregateOpLoadallLoadAll) Limit() FtAggregateOpLimitLimit {
	c.cs.s = append(c.cs.s, "LIMIT")
	return (FtAggregateOpLimitLimit)(c)
}

func (c FtAggregateOpLoadallLoadAll) Filter(filter string) FtAggregateOpFilter {
	c.cs.s = append(c.cs.s, "FILTER", filter)
	return (FtAggregateOpFilter)(c)
}

func (c FtAggregateOpLoadallLoadAll) LoadAll() FtAggregateOpLoadallLoadAll {
	c.cs.s = append(c.cs.s, "LOAD", "*")
	return c
}

func (c FtAggregateOpLoadallLoadAll) Withcursor() FtAggregateCursorWithcursor {
	c.cs.s = append(c.cs.s, "WITHCURSOR")
	return (FtAggregateCursorWithcursor)(c)
}

func (c FtAggregateOpLoadallLoadAll) Params() FtAggregateParamsParams {
	c.cs.s = append(c.cs.s, "PARAMS")
	return (FtAggregateParamsParams)(c)
}

func (c FtAggregateOpLoadallLoadAll) Dialect(dialect int64) FtAggregateDialect {
	c.cs.s = append(c.cs.s, "DIALECT", strconv.FormatInt(dialect, 10))
	return (FtAggregateDialect)(c)
}

func (c FtAggregateOpLoadallLoadAll) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type FtAggregateOpSortbyFieldsOrderAsc Completed

func (c FtAggregateOpSortbyFieldsOrderAsc) Property(property string) FtAggregateOpSortbyFieldsProperty {
	c.cs.s = append(c.cs.s, property)
	return (FtAggregateOpSortbyFieldsProperty)(c)
}

func (c FtAggregateOpSortbyFieldsOrderAsc) Max(num int64) FtAggregateOpSortbyMax {
	c.cs.s = append(c.cs.s, "MAX", strconv.FormatInt(num, 10))
	return (FtAggregateOpSortbyMax)(c)
}

func (c FtAggregateOpSortbyFieldsOrderAsc) Apply(expression string) FtAggregateOpApplyApply {
	c.cs.s = append(c.cs.s, "APPLY", expression)
	return (FtAggregateOpApplyApply)(c)
}

func (c FtAggregateOpSortbyFieldsOrderAsc) Limit() FtAggregateOpLimitLimit {
	c.cs.s = append(c.cs.s, "LIMIT")
	return (FtAggregateOpLimitLimit)(c)
}

func (c FtAggregateOpSortbyFieldsOrderAsc) Filter(filter string) FtAggregateOpFilter {
	c.cs.s = append(c.cs.s, "FILTER", filter)
	return (FtAggregateOpFilter)(c)
}

func (c FtAggregateOpSortbyFieldsOrderAsc) LoadAll() FtAggregateOpLoadallLoadAll {
	c.cs.s = append(c.cs.s, "LOAD", "*")
	return (FtAggregateOpLoadallLoadAll)(c)
}

func (c FtAggregateOpSortbyFieldsOrderAsc) Load(count int64) FtAggregateOpLoadLoad {
	c.cs.s = append(c.cs.s, "LOAD", strconv.FormatInt(count, 10))
	return (FtAggregateOpLoadLoad)(c)
}

func (c FtAggregateOpSortbyFieldsOrderAsc) Groupby(nargs int64) FtAggregateOpGroupbyGroupby {
	c.cs.s = append(c.cs.s, "GROUPBY", strconv.FormatInt(nargs, 10))
	return (FtAggregateOpGroupbyGroupby)(c)
}

func (c FtAggregateOpSortbyFieldsOrderAsc) Sortby(nargs int64) FtAggregateOpSortbySortby {
	c.cs.s = append(c.cs.s, "SORTBY", strconv.FormatInt(nargs, 10))
	return (FtAggregateOpSortbySortby)(c)
}

func (c FtAggregateOpSortbyFieldsOrderAsc) Withcursor() FtAggregateCursorWithcursor {
	c.cs.s = append(c.cs.s, "WITHCURSOR")
	return (FtAggregateCursorWithcursor)(c)
}

func (c FtAggregateOpSortbyFieldsOrderAsc) Params() FtAggregateParamsParams {
	c.cs.s = append(c.cs.s, "PARAMS")
	return (FtAggregateParamsParams)(c)
}

func (c FtAggregateOpSortbyFieldsOrderAsc) Dialect(dialect int64) FtAggregateDialect {
	c.cs.s = append(c.cs.s, "DIALECT", strconv.FormatInt(dialect, 10))
	return (FtAggregateDialect)(c)
}

func (c FtAggregateOpSortbyFieldsOrderAsc) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type FtAggregateOpSortbyFieldsOrderDesc Completed

func (c FtAggregateOpSortbyFieldsOrderDesc) Property(property string) FtAggregateOpSortbyFieldsProperty {
	c.cs.s = append(c.cs.s, property)
	return (FtAggregateOpSortbyFieldsProperty)(c)
}

func (c FtAggregateOpSortbyFieldsOrderDesc) Max(num int64) FtAggregateOpSortbyMax {
	c.cs.s = append(c.cs.s, "MAX", strconv.FormatInt(num, 10))
	return (FtAggregateOpSortbyMax)(c)
}

func (c FtAggregateOpSortbyFieldsOrderDesc) Apply(expression string) FtAggregateOpApplyApply {
	c.cs.s = append(c.cs.s, "APPLY", expression)
	return (FtAggregateOpApplyApply)(c)
}

func (c FtAggregateOpSortbyFieldsOrderDesc) Limit() FtAggregateOpLimitLimit {
	c.cs.s = append(c.cs.s, "LIMIT")
	return (FtAggregateOpLimitLimit)(c)
}

func (c FtAggregateOpSortbyFieldsOrderDesc) Filter(filter string) FtAggregateOpFilter {
	c.cs.s = append(c.cs.s, "FILTER", filter)
	return (FtAggregateOpFilter)(c)
}

func (c FtAggregateOpSortbyFieldsOrderDesc) LoadAll() FtAggregateOpLoadallLoadAll {
	c.cs.s = append(c.cs.s, "LOAD", "*")
	return (FtAggregateOpLoadallLoadAll)(c)
}

func (c FtAggregateOpSortbyFieldsOrderDesc) Load(count int64) FtAggregateOpLoadLoad {
	c.cs.s = append(c.cs.s, "LOAD", strconv.FormatInt(count, 10))
	return (FtAggregateOpLoadLoad)(c)
}

func (c FtAggregateOpSortbyFieldsOrderDesc) Groupby(nargs int64) FtAggregateOpGroupbyGroupby {
	c.cs.s = append(c.cs.s, "GROUPBY", strconv.FormatInt(nargs, 10))
	return (FtAggregateOpGroupbyGroupby)(c)
}

func (c FtAggregateOpSortbyFieldsOrderDesc) Sortby(nargs int64) FtAggregateOpSortbySortby {
	c.cs.s = append(c.cs.s, "SORTBY", strconv.FormatInt(nargs, 10))
	return (FtAggregateOpSortbySortby)(c)
}

func (c FtAggregateOpSortbyFieldsOrderDesc) Withcursor() FtAggregateCursorWithcursor {
	c.cs.s = append(c.cs.s, "WITHCURSOR")
	return (FtAggregateCursorWithcursor)(c)
}

func (c FtAggregateOpSortbyFieldsOrderDesc) Params() FtAggregateParamsParams {
	c.cs.s = append(c.cs.s, "PARAMS")
	return (FtAggregateParamsParams)(c)
}

func (c FtAggregateOpSortbyFieldsOrderDesc) Dialect(dialect int64) FtAggregateDialect {
	c.cs.s = append(c.cs.s, "DIALECT", strconv.FormatInt(dialect, 10))
	return (FtAggregateDialect)(c)
}

func (c FtAggregateOpSortbyFieldsOrderDesc) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type FtAggregateOpSortbyFieldsProperty Completed

func (c FtAggregateOpSortbyFieldsProperty) Asc() FtAggregateOpSortbyFieldsOrderAsc {
	c.cs.s = append(c.cs.s, "ASC")
	return (FtAggregateOpSortbyFieldsOrderAsc)(c)
}

func (c FtAggregateOpSortbyFieldsProperty) Desc() FtAggregateOpSortbyFieldsOrderDesc {
	c.cs.s = append(c.cs.s, "DESC")
	return (FtAggregateOpSortbyFieldsOrderDesc)(c)
}

func (c FtAggregateOpSortbyFieldsProperty) Property(property string) FtAggregateOpSortbyFieldsProperty {
	c.cs.s = append(c.cs.s, property)
	return c
}

func (c FtAggregateOpSortbyFieldsProperty) Max(num int64) FtAggregateOpSortbyMax {
	c.cs.s = append(c.cs.s, "MAX", strconv.FormatInt(num, 10))
	return (FtAggregateOpSortbyMax)(c)
}

func (c FtAggregateOpSortbyFieldsProperty) Apply(expression string) FtAggregateOpApplyApply {
	c.cs.s = append(c.cs.s, "APPLY", expression)
	return (FtAggregateOpApplyApply)(c)
}

func (c FtAggregateOpSortbyFieldsProperty) Limit() FtAggregateOpLimitLimit {
	c.cs.s = append(c.cs.s, "LIMIT")
	return (FtAggregateOpLimitLimit)(c)
}

func (c FtAggregateOpSortbyFieldsProperty) Filter(filter string) FtAggregateOpFilter {
	c.cs.s = append(c.cs.s, "FILTER", filter)
	return (FtAggregateOpFilter)(c)
}

func (c FtAggregateOpSortbyFieldsProperty) LoadAll() FtAggregateOpLoadallLoadAll {
	c.cs.s = append(c.cs.s, "LOAD", "*")
	return (FtAggregateOpLoadallLoadAll)(c)
}

func (c FtAggregateOpSortbyFieldsProperty) Load(count int64) FtAggregateOpLoadLoad {
	c.cs.s = append(c.cs.s, "LOAD", strconv.FormatInt(count, 10))
	return (FtAggregateOpLoadLoad)(c)
}

func (c FtAggregateOpSortbyFieldsProperty) Groupby(nargs int64) FtAggregateOpGroupbyGroupby {
	c.cs.s = append(c.cs.s, "GROUPBY", strconv.FormatInt(nargs, 10))
	return (FtAggregateOpGroupbyGroupby)(c)
}

func (c FtAggregateOpSortbyFieldsProperty) Sortby(nargs int64) FtAggregateOpSortbySortby {
	c.cs.s = append(c.cs.s, "SORTBY", strconv.FormatInt(nargs, 10))
	return (FtAggregateOpSortbySortby)(c)
}

func (c FtAggregateOpSortbyFieldsProperty) Withcursor() FtAggregateCursorWithcursor {
	c.cs.s = append(c.cs.s, "WITHCURSOR")
	return (FtAggregateCursorWithcursor)(c)
}

func (c FtAggregateOpSortbyFieldsProperty) Params() FtAggregateParamsParams {
	c.cs.s = append(c.cs.s, "PARAMS")
	return (FtAggregateParamsParams)(c)
}

func (c FtAggregateOpSortbyFieldsProperty) Dialect(dialect int64) FtAggregateDialect {
	c.cs.s = append(c.cs.s, "DIALECT", strconv.FormatInt(dialect, 10))
	return (FtAggregateDialect)(c)
}

func (c FtAggregateOpSortbyFieldsProperty) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type FtAggregateOpSortbyMax Completed

func (c FtAggregateOpSortbyMax) Apply(expression string) FtAggregateOpApplyApply {
	c.cs.s = append(c.cs.s, "APPLY", expression)
	return (FtAggregateOpApplyApply)(c)
}

func (c FtAggregateOpSortbyMax) Limit() FtAggregateOpLimitLimit {
	c.cs.s = append(c.cs.s, "LIMIT")
	return (FtAggregateOpLimitLimit)(c)
}

func (c FtAggregateOpSortbyMax) Filter(filter string) FtAggregateOpFilter {
	c.cs.s = append(c.cs.s, "FILTER", filter)
	return (FtAggregateOpFilter)(c)
}

func (c FtAggregateOpSortbyMax) LoadAll() FtAggregateOpLoadallLoadAll {
	c.cs.s = append(c.cs.s, "LOAD", "*")
	return (FtAggregateOpLoadallLoadAll)(c)
}

func (c FtAggregateOpSortbyMax) Load(count int64) FtAggregateOpLoadLoad {
	c.cs.s = append(c.cs.s, "LOAD", strconv.FormatInt(count, 10))
	return (FtAggregateOpLoadLoad)(c)
}

func (c FtAggregateOpSortbyMax) Groupby(nargs int64) FtAggregateOpGroupbyGroupby {
	c.cs.s = append(c.cs.s, "GROUPBY", strconv.FormatInt(nargs, 10))
	return (FtAggregateOpGroupbyGroupby)(c)
}

func (c FtAggregateOpSortbyMax) Sortby(nargs int64) FtAggregateOpSortbySortby {
	c.cs.s = append(c.cs.s, "SORTBY", strconv.FormatInt(nargs, 10))
	return (FtAggregateOpSortbySortby)(c)
}

func (c FtAggregateOpSortbyMax) Withcursor() FtAggregateCursorWithcursor {
	c.cs.s = append(c.cs.s, "WITHCURSOR")
	return (FtAggregateCursorWithcursor)(c)
}

func (c FtAggregateOpSortbyMax) Params() FtAggregateParamsParams {
	c.cs.s = append(c.cs.s, "PARAMS")
	return (FtAggregateParamsParams)(c)
}

func (c FtAggregateOpSortbyMax) Dialect(dialect int64) FtAggregateDialect {
	c.cs.s = append(c.cs.s, "DIALECT", strconv.FormatInt(dialect, 10))
	return (FtAggregateDialect)(c)
}

func (c FtAggregateOpSortbyMax) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type FtAggregateOpSortbySortby Completed

func (c FtAggregateOpSortbySortby) Property(property string) FtAggregateOpSortbyFieldsProperty {
	c.cs.s = append(c.cs.s, property)
	return (FtAggregateOpSortbyFieldsProperty)(c)
}

func (c FtAggregateOpSortbySortby) Max(num int64) FtAggregateOpSortbyMax {
	c.cs.s = append(c.cs.s, "MAX", strconv.FormatInt(num, 10))
	return (FtAggregateOpSortbyMax)(c)
}

func (c FtAggregateOpSortbySortby) Apply(expression string) FtAggregateOpApplyApply {
	c.cs.s = append(c.cs.s, "APPLY", expression)
	return (FtAggregateOpApplyApply)(c)
}

func (c FtAggregateOpSortbySortby) Limit() FtAggregateOpLimitLimit {
	c.cs.s = append(c.cs.s, "LIMIT")
	return (FtAggregateOpLimitLimit)(c)
}

func (c FtAggregateOpSortbySortby) Filter(filter string) FtAggregateOpFilter {
	c.cs.s = append(c.cs.s, "FILTER", filter)
	return (FtAggregateOpFilter)(c)
}

func (c FtAggregateOpSortbySortby) LoadAll() FtAggregateOpLoadallLoadAll {
	c.cs.s = append(c.cs.s, "LOAD", "*")
	return (FtAggregateOpLoadallLoadAll)(c)
}

func (c FtAggregateOpSortbySortby) Load(count int64) FtAggregateOpLoadLoad {
	c.cs.s = append(c.cs.s, "LOAD", strconv.FormatInt(count, 10))
	return (FtAggregateOpLoadLoad)(c)
}

func (c FtAggregateOpSortbySortby) Groupby(nargs int64) FtAggregateOpGroupbyGroupby {
	c.cs.s = append(c.cs.s, "GROUPBY", strconv.FormatInt(nargs, 10))
	return (FtAggregateOpGroupbyGroupby)(c)
}

func (c FtAggregateOpSortbySortby) Sortby(nargs int64) FtAggregateOpSortbySortby {
	c.cs.s = append(c.cs.s, "SORTBY", strconv.FormatInt(nargs, 10))
	return c
}

func (c FtAggregateOpSortbySortby) Withcursor() FtAggregateCursorWithcursor {
	c.cs.s = append(c.cs.s, "WITHCURSOR")
	return (FtAggregateCursorWithcursor)(c)
}

func (c FtAggregateOpSortbySortby) Params() FtAggregateParamsParams {
	c.cs.s = append(c.cs.s, "PARAMS")
	return (FtAggregateParamsParams)(c)
}

func (c FtAggregateOpSortbySortby) Dialect(dialect int64) FtAggregateDialect {
	c.cs.s = append(c.cs.s, "DIALECT", strconv.FormatInt(dialect, 10))
	return (FtAggregateDialect)(c)
}

func (c FtAggregateOpSortbySortby) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type FtAggregateParamsNameValue Completed

func (c FtAggregateParamsNameValue) NameValue(name string, value string) FtAggregateParamsNameValue {
	c.cs.s = append(c.cs.s, name, value)
	return c
}

func (c FtAggregateParamsNameValue) Dialect(dialect int64) FtAggregateDialect {
	c.cs.s = append(c.cs.s, "DIALECT", strconv.FormatInt(dialect, 10))
	return (FtAggregateDialect)(c)
}

func (c FtAggregateParamsNameValue) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type FtAggregateParamsNargs Completed

func (c FtAggregateParamsNargs) NameValue() FtAggregateParamsNameValue {
	return (FtAggregateParamsNameValue)(c)
}

type FtAggregateParamsParams Completed

func (c FtAggregateParamsParams) Nargs(nargs int64) FtAggregateParamsNargs {
	c.cs.s = append(c.cs.s, strconv.FormatInt(nargs, 10))
	return (FtAggregateParamsNargs)(c)
}

type FtAggregateQuery Completed

func (c FtAggregateQuery) Verbatim() FtAggregateVerbatim {
	c.cs.s = append(c.cs.s, "VERBATIM")
	return (FtAggregateVerbatim)(c)
}

func (c FtAggregateQuery) Timeout(timeout int64) FtAggregateTimeout {
	c.cs.s = append(c.cs.s, "TIMEOUT", strconv.FormatInt(timeout, 10))
	return (FtAggregateTimeout)(c)
}

func (c FtAggregateQuery) LoadAll() FtAggregateOpLoadallLoadAll {
	c.cs.s = append(c.cs.s, "LOAD", "*")
	return (FtAggregateOpLoadallLoadAll)(c)
}

func (c FtAggregateQuery) Load(count int64) FtAggregateOpLoadLoad {
	c.cs.s = append(c.cs.s, "LOAD", strconv.FormatInt(count, 10))
	return (FtAggregateOpLoadLoad)(c)
}

func (c FtAggregateQuery) Groupby(nargs int64) FtAggregateOpGroupbyGroupby {
	c.cs.s = append(c.cs.s, "GROUPBY", strconv.FormatInt(nargs, 10))
	return (FtAggregateOpGroupbyGroupby)(c)
}

func (c FtAggregateQuery) Sortby(nargs int64) FtAggregateOpSortbySortby {
	c.cs.s = append(c.cs.s, "SORTBY", strconv.FormatInt(nargs, 10))
	return (FtAggregateOpSortbySortby)(c)
}

func (c FtAggregateQuery) Apply(expression string) FtAggregateOpApplyApply {
	c.cs.s = append(c.cs.s, "APPLY", expression)
	return (FtAggregateOpApplyApply)(c)
}

func (c FtAggregateQuery) Limit() FtAggregateOpLimitLimit {
	c.cs.s = append(c.cs.s, "LIMIT")
	return (FtAggregateOpLimitLimit)(c)
}

func (c FtAggregateQuery) Filter(filter string) FtAggregateOpFilter {
	c.cs.s = append(c.cs.s, "FILTER", filter)
	return (FtAggregateOpFilter)(c)
}

func (c FtAggregateQuery) Withcursor() FtAggregateCursorWithcursor {
	c.cs.s = append(c.cs.s, "WITHCURSOR")
	return (FtAggregateCursorWithcursor)(c)
}

func (c FtAggregateQuery) Params() FtAggregateParamsParams {
	c.cs.s = append(c.cs.s, "PARAMS")
	return (FtAggregateParamsParams)(c)
}

func (c FtAggregateQuery) Dialect(dialect int64) FtAggregateDialect {
	c.cs.s = append(c.cs.s, "DIALECT", strconv.FormatInt(dialect, 10))
	return (FtAggregateDialect)(c)
}

func (c FtAggregateQuery) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type FtAggregateTimeout Completed

func (c FtAggregateTimeout) LoadAll() FtAggregateOpLoadallLoadAll {
	c.cs.s = append(c.cs.s, "LOAD", "*")
	return (FtAggregateOpLoadallLoadAll)(c)
}

func (c FtAggregateTimeout) Load(count int64) FtAggregateOpLoadLoad {
	c.cs.s = append(c.cs.s, "LOAD", strconv.FormatInt(count, 10))
	return (FtAggregateOpLoadLoad)(c)
}

func (c FtAggregateTimeout) Groupby(nargs int64) FtAggregateOpGroupbyGroupby {
	c.cs.s = append(c.cs.s, "GROUPBY", strconv.FormatInt(nargs, 10))
	return (FtAggregateOpGroupbyGroupby)(c)
}

func (c FtAggregateTimeout) Sortby(nargs int64) FtAggregateOpSortbySortby {
	c.cs.s = append(c.cs.s, "SORTBY", strconv.FormatInt(nargs, 10))
	return (FtAggregateOpSortbySortby)(c)
}

func (c FtAggregateTimeout) Apply(expression string) FtAggregateOpApplyApply {
	c.cs.s = append(c.cs.s, "APPLY", expression)
	return (FtAggregateOpApplyApply)(c)
}

func (c FtAggregateTimeout) Limit() FtAggregateOpLimitLimit {
	c.cs.s = append(c.cs.s, "LIMIT")
	return (FtAggregateOpLimitLimit)(c)
}

func (c FtAggregateTimeout) Filter(filter string) FtAggregateOpFilter {
	c.cs.s = append(c.cs.s, "FILTER", filter)
	return (FtAggregateOpFilter)(c)
}

func (c FtAggregateTimeout) Withcursor() FtAggregateCursorWithcursor {
	c.cs.s = append(c.cs.s, "WITHCURSOR")
	return (FtAggregateCursorWithcursor)(c)
}

func (c FtAggregateTimeout) Params() FtAggregateParamsParams {
	c.cs.s = append(c.cs.s, "PARAMS")
	return (FtAggregateParamsParams)(c)
}

func (c FtAggregateTimeout) Dialect(dialect int64) FtAggregateDialect {
	c.cs.s = append(c.cs.s, "DIALECT", strconv.FormatInt(dialect, 10))
	return (FtAggregateDialect)(c)
}

func (c FtAggregateTimeout) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type FtAggregateVerbatim Completed

func (c FtAggregateVerbatim) Timeout(timeout int64) FtAggregateTimeout {
	c.cs.s = append(c.cs.s, "TIMEOUT", strconv.FormatInt(timeout, 10))
	return (FtAggregateTimeout)(c)
}

func (c FtAggregateVerbatim) LoadAll() FtAggregateOpLoadallLoadAll {
	c.cs.s = append(c.cs.s, "LOAD", "*")
	return (FtAggregateOpLoadallLoadAll)(c)
}

func (c FtAggregateVerbatim) Load(count int64) FtAggregateOpLoadLoad {
	c.cs.s = append(c.cs.s, "LOAD", strconv.FormatInt(count, 10))
	return (FtAggregateOpLoadLoad)(c)
}

func (c FtAggregateVerbatim) Groupby(nargs int64) FtAggregateOpGroupbyGroupby {
	c.cs.s = append(c.cs.s, "GROUPBY", strconv.FormatInt(nargs, 10))
	return (FtAggregateOpGroupbyGroupby)(c)
}

func (c FtAggregateVerbatim) Sortby(nargs int64) FtAggregateOpSortbySortby {
	c.cs.s = append(c.cs.s, "SORTBY", strconv.FormatInt(nargs, 10))
	return (FtAggregateOpSortbySortby)(c)
}

func (c FtAggregateVerbatim) Apply(expression string) FtAggregateOpApplyApply {
	c.cs.s = append(c.cs.s, "APPLY", expression)
	return (FtAggregateOpApplyApply)(c)
}

func (c FtAggregateVerbatim) Limit() FtAggregateOpLimitLimit {
	c.cs.s = append(c.cs.s, "LIMIT")
	return (FtAggregateOpLimitLimit)(c)
}

func (c FtAggregateVerbatim) Filter(filter string) FtAggregateOpFilter {
	c.cs.s = append(c.cs.s, "FILTER", filter)
	return (FtAggregateOpFilter)(c)
}

func (c FtAggregateVerbatim) Withcursor() FtAggregateCursorWithcursor {
	c.cs.s = append(c.cs.s, "WITHCURSOR")
	return (FtAggregateCursorWithcursor)(c)
}

func (c FtAggregateVerbatim) Params() FtAggregateParamsParams {
	c.cs.s = append(c.cs.s, "PARAMS")
	return (FtAggregateParamsParams)(c)
}

func (c FtAggregateVerbatim) Dialect(dialect int64) FtAggregateDialect {
	c.cs.s = append(c.cs.s, "DIALECT", strconv.FormatInt(dialect, 10))
	return (FtAggregateDialect)(c)
}

func (c FtAggregateVerbatim) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type FtAliasadd Completed

func (b Builder) FtAliasadd() (c FtAliasadd) {
	c = FtAliasadd{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "FT.ALIASADD")
	return c
}

func (c FtAliasadd) Alias(alias string) FtAliasaddAlias {
	c.cs.s = append(c.cs.s, alias)
	return (FtAliasaddAlias)(c)
}

type FtAliasaddAlias Completed

func (c FtAliasaddAlias) Index(index string) FtAliasaddIndex {
	c.cs.s = append(c.cs.s, index)
	return (FtAliasaddIndex)(c)
}

type FtAliasaddIndex Completed

func (c FtAliasaddIndex) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type FtAliasdel Completed

func (b Builder) FtAliasdel() (c FtAliasdel) {
	c = FtAliasdel{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "FT.ALIASDEL")
	return c
}

func (c FtAliasdel) Alias(alias string) FtAliasdelAlias {
	c.cs.s = append(c.cs.s, alias)
	return (FtAliasdelAlias)(c)
}

type FtAliasdelAlias Completed

func (c FtAliasdelAlias) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type FtAliasupdate Completed

func (b Builder) FtAliasupdate() (c FtAliasupdate) {
	c = FtAliasupdate{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "FT.ALIASUPDATE")
	return c
}

func (c FtAliasupdate) Alias(alias string) FtAliasupdateAlias {
	c.cs.s = append(c.cs.s, alias)
	return (FtAliasupdateAlias)(c)
}

type FtAliasupdateAlias Completed

func (c FtAliasupdateAlias) Index(index string) FtAliasupdateIndex {
	c.cs.s = append(c.cs.s, index)
	return (FtAliasupdateIndex)(c)
}

type FtAliasupdateIndex Completed

func (c FtAliasupdateIndex) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type FtAlter Completed

func (b Builder) FtAlter() (c FtAlter) {
	c = FtAlter{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "FT.ALTER")
	return c
}

func (c FtAlter) Index(index string) FtAlterIndex {
	c.cs.s = append(c.cs.s, index)
	return (FtAlterIndex)(c)
}

type FtAlterAdd Completed

func (c FtAlterAdd) Field(field string) FtAlterField {
	c.cs.s = append(c.cs.s, field)
	return (FtAlterField)(c)
}

type FtAlterField Completed

func (c FtAlterField) Options(options string) FtAlterOptions {
	c.cs.s = append(c.cs.s, options)
	return (FtAlterOptions)(c)
}

type FtAlterIndex Completed

func (c FtAlterIndex) Skipinitialscan() FtAlterSkipinitialscan {
	c.cs.s = append(c.cs.s, "SKIPINITIALSCAN")
	return (FtAlterSkipinitialscan)(c)
}

func (c FtAlterIndex) Schema() FtAlterSchema {
	c.cs.s = append(c.cs.s, "SCHEMA")
	return (FtAlterSchema)(c)
}

type FtAlterOptions Completed

func (c FtAlterOptions) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type FtAlterSchema Completed

func (c FtAlterSchema) Add() FtAlterAdd {
	c.cs.s = append(c.cs.s, "ADD")
	return (FtAlterAdd)(c)
}

type FtAlterSkipinitialscan Completed

func (c FtAlterSkipinitialscan) Schema() FtAlterSchema {
	c.cs.s = append(c.cs.s, "SCHEMA")
	return (FtAlterSchema)(c)
}

type FtConfigGet Completed

func (b Builder) FtConfigGet() (c FtConfigGet) {
	c = FtConfigGet{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "FT.CONFIG", "GET")
	return c
}

func (c FtConfigGet) Option(option string) FtConfigGetOption {
	c.cs.s = append(c.cs.s, option)
	return (FtConfigGetOption)(c)
}

type FtConfigGetOption Completed

func (c FtConfigGetOption) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type FtConfigHelp Completed

func (b Builder) FtConfigHelp() (c FtConfigHelp) {
	c = FtConfigHelp{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "FT.CONFIG", "HELP")
	return c
}

func (c FtConfigHelp) Option(option string) FtConfigHelpOption {
	c.cs.s = append(c.cs.s, option)
	return (FtConfigHelpOption)(c)
}

type FtConfigHelpOption Completed

func (c FtConfigHelpOption) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type FtConfigSet Completed

func (b Builder) FtConfigSet() (c FtConfigSet) {
	c = FtConfigSet{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "FT.CONFIG", "SET")
	return c
}

func (c FtConfigSet) Option(option string) FtConfigSetOption {
	c.cs.s = append(c.cs.s, option)
	return (FtConfigSetOption)(c)
}

type FtConfigSetOption Completed

func (c FtConfigSetOption) Value(value string) FtConfigSetValue {
	c.cs.s = append(c.cs.s, value)
	return (FtConfigSetValue)(c)
}

type FtConfigSetValue Completed

func (c FtConfigSetValue) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type FtCreate Completed

func (b Builder) FtCreate() (c FtCreate) {
	c = FtCreate{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "FT.CREATE")
	return c
}

func (c FtCreate) Index(index string) FtCreateIndex {
	c.cs.s = append(c.cs.s, index)
	return (FtCreateIndex)(c)
}

type FtCreateFieldAs Completed

func (c FtCreateFieldAs) Text() FtCreateFieldFieldTypeText {
	c.cs.s = append(c.cs.s, "TEXT")
	return (FtCreateFieldFieldTypeText)(c)
}

func (c FtCreateFieldAs) Tag() FtCreateFieldFieldTypeTag {
	c.cs.s = append(c.cs.s, "TAG")
	return (FtCreateFieldFieldTypeTag)(c)
}

func (c FtCreateFieldAs) Numeric() FtCreateFieldFieldTypeNumeric {
	c.cs.s = append(c.cs.s, "NUMERIC")
	return (FtCreateFieldFieldTypeNumeric)(c)
}

func (c FtCreateFieldAs) Geo() FtCreateFieldFieldTypeGeo {
	c.cs.s = append(c.cs.s, "GEO")
	return (FtCreateFieldFieldTypeGeo)(c)
}

func (c FtCreateFieldAs) Vector(algo string, nargs int64, args ...string) FtCreateFieldFieldTypeVector {
	c.cs.s = append(c.cs.s, "VECTOR", algo, strconv.FormatInt(nargs, 10))
	c.cs.s = append(c.cs.s, args...)
	return (FtCreateFieldFieldTypeVector)(c)
}

type FtCreateFieldFieldName Completed

func (c FtCreateFieldFieldName) As(alias string) FtCreateFieldAs {
	c.cs.s = append(c.cs.s, "AS", alias)
	return (FtCreateFieldAs)(c)
}

func (c FtCreateFieldFieldName) Text() FtCreateFieldFieldTypeText {
	c.cs.s = append(c.cs.s, "TEXT")
	return (FtCreateFieldFieldTypeText)(c)
}

func (c FtCreateFieldFieldName) Tag() FtCreateFieldFieldTypeTag {
	c.cs.s = append(c.cs.s, "TAG")
	return (FtCreateFieldFieldTypeTag)(c)
}

func (c FtCreateFieldFieldName) Numeric() FtCreateFieldFieldTypeNumeric {
	c.cs.s = append(c.cs.s, "NUMERIC")
	return (FtCreateFieldFieldTypeNumeric)(c)
}

func (c FtCreateFieldFieldName) Geo() FtCreateFieldFieldTypeGeo {
	c.cs.s = append(c.cs.s, "GEO")
	return (FtCreateFieldFieldTypeGeo)(c)
}

func (c FtCreateFieldFieldName) Vector(algo string, nargs int64, args ...string) FtCreateFieldFieldTypeVector {
	c.cs.s = append(c.cs.s, "VECTOR", algo, strconv.FormatInt(nargs, 10))
	c.cs.s = append(c.cs.s, args...)
	return (FtCreateFieldFieldTypeVector)(c)
}

type FtCreateFieldFieldTypeGeo Completed

func (c FtCreateFieldFieldTypeGeo) Withsuffixtrie() FtCreateFieldOptionWithsuffixtrie {
	c.cs.s = append(c.cs.s, "WITHSUFFIXTRIE")
	return (FtCreateFieldOptionWithsuffixtrie)(c)
}

func (c FtCreateFieldFieldTypeGeo) Sortable() FtCreateFieldOptionSortableSortable {
	c.cs.s = append(c.cs.s, "SORTABLE")
	return (FtCreateFieldOptionSortableSortable)(c)
}

func (c FtCreateFieldFieldTypeGeo) Noindex() FtCreateFieldOptionNoindex {
	c.cs.s = append(c.cs.s, "NOINDEX")
	return (FtCreateFieldOptionNoindex)(c)
}

func (c FtCreateFieldFieldTypeGeo) Nostem() FtCreateFieldOptionNostem {
	c.cs.s = append(c.cs.s, "NOSTEM")
	return (FtCreateFieldOptionNostem)(c)
}

func (c FtCreateFieldFieldTypeGeo) Phonetic(phonetic string) FtCreateFieldOptionPhonetic {
	c.cs.s = append(c.cs.s, "PHONETIC", phonetic)
	return (FtCreateFieldOptionPhonetic)(c)
}

func (c FtCreateFieldFieldTypeGeo) Weight(weight float64) FtCreateFieldOptionWeight {
	c.cs.s = append(c.cs.s, "WEIGHT", strconv.FormatFloat(weight, 'f', -1, 64))
	return (FtCreateFieldOptionWeight)(c)
}

func (c FtCreateFieldFieldTypeGeo) Separator(separator string) FtCreateFieldOptionSeparator {
	c.cs.s = append(c.cs.s, "SEPARATOR", separator)
	return (FtCreateFieldOptionSeparator)(c)
}

func (c FtCreateFieldFieldTypeGeo) Casesensitive() FtCreateFieldOptionCasesensitive {
	c.cs.s = append(c.cs.s, "CASESENSITIVE")
	return (FtCreateFieldOptionCasesensitive)(c)
}

func (c FtCreateFieldFieldTypeGeo) FieldName(fieldName string) FtCreateFieldFieldName {
	c.cs.s = append(c.cs.s, fieldName)
	return (FtCreateFieldFieldName)(c)
}

func (c FtCreateFieldFieldTypeGeo) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type FtCreateFieldFieldTypeNumeric Completed

func (c FtCreateFieldFieldTypeNumeric) Withsuffixtrie() FtCreateFieldOptionWithsuffixtrie {
	c.cs.s = append(c.cs.s, "WITHSUFFIXTRIE")
	return (FtCreateFieldOptionWithsuffixtrie)(c)
}

func (c FtCreateFieldFieldTypeNumeric) Sortable() FtCreateFieldOptionSortableSortable {
	c.cs.s = append(c.cs.s, "SORTABLE")
	return (FtCreateFieldOptionSortableSortable)(c)
}

func (c FtCreateFieldFieldTypeNumeric) Noindex() FtCreateFieldOptionNoindex {
	c.cs.s = append(c.cs.s, "NOINDEX")
	return (FtCreateFieldOptionNoindex)(c)
}

func (c FtCreateFieldFieldTypeNumeric) Nostem() FtCreateFieldOptionNostem {
	c.cs.s = append(c.cs.s, "NOSTEM")
	return (FtCreateFieldOptionNostem)(c)
}

func (c FtCreateFieldFieldTypeNumeric) Phonetic(phonetic string) FtCreateFieldOptionPhonetic {
	c.cs.s = append(c.cs.s, "PHONETIC", phonetic)
	return (FtCreateFieldOptionPhonetic)(c)
}

func (c FtCreateFieldFieldTypeNumeric) Weight(weight float64) FtCreateFieldOptionWeight {
	c.cs.s = append(c.cs.s, "WEIGHT", strconv.FormatFloat(weight, 'f', -1, 64))
	return (FtCreateFieldOptionWeight)(c)
}

func (c FtCreateFieldFieldTypeNumeric) Separator(separator string) FtCreateFieldOptionSeparator {
	c.cs.s = append(c.cs.s, "SEPARATOR", separator)
	return (FtCreateFieldOptionSeparator)(c)
}

func (c FtCreateFieldFieldTypeNumeric) Casesensitive() FtCreateFieldOptionCasesensitive {
	c.cs.s = append(c.cs.s, "CASESENSITIVE")
	return (FtCreateFieldOptionCasesensitive)(c)
}

func (c FtCreateFieldFieldTypeNumeric) FieldName(fieldName string) FtCreateFieldFieldName {
	c.cs.s = append(c.cs.s, fieldName)
	return (FtCreateFieldFieldName)(c)
}

func (c FtCreateFieldFieldTypeNumeric) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type FtCreateFieldFieldTypeTag Completed

func (c FtCreateFieldFieldTypeTag) Withsuffixtrie() FtCreateFieldOptionWithsuffixtrie {
	c.cs.s = append(c.cs.s, "WITHSUFFIXTRIE")
	return (FtCreateFieldOptionWithsuffixtrie)(c)
}

func (c FtCreateFieldFieldTypeTag) Sortable() FtCreateFieldOptionSortableSortable {
	c.cs.s = append(c.cs.s, "SORTABLE")
	return (FtCreateFieldOptionSortableSortable)(c)
}

func (c FtCreateFieldFieldTypeTag) Noindex() FtCreateFieldOptionNoindex {
	c.cs.s = append(c.cs.s, "NOINDEX")
	return (FtCreateFieldOptionNoindex)(c)
}

func (c FtCreateFieldFieldTypeTag) Nostem() FtCreateFieldOptionNostem {
	c.cs.s = append(c.cs.s, "NOSTEM")
	return (FtCreateFieldOptionNostem)(c)
}

func (c FtCreateFieldFieldTypeTag) Phonetic(phonetic string) FtCreateFieldOptionPhonetic {
	c.cs.s = append(c.cs.s, "PHONETIC", phonetic)
	return (FtCreateFieldOptionPhonetic)(c)
}

func (c FtCreateFieldFieldTypeTag) Weight(weight float64) FtCreateFieldOptionWeight {
	c.cs.s = append(c.cs.s, "WEIGHT", strconv.FormatFloat(weight, 'f', -1, 64))
	return (FtCreateFieldOptionWeight)(c)
}

func (c FtCreateFieldFieldTypeTag) Separator(separator string) FtCreateFieldOptionSeparator {
	c.cs.s = append(c.cs.s, "SEPARATOR", separator)
	return (FtCreateFieldOptionSeparator)(c)
}

func (c FtCreateFieldFieldTypeTag) Casesensitive() FtCreateFieldOptionCasesensitive {
	c.cs.s = append(c.cs.s, "CASESENSITIVE")
	return (FtCreateFieldOptionCasesensitive)(c)
}

func (c FtCreateFieldFieldTypeTag) FieldName(fieldName string) FtCreateFieldFieldName {
	c.cs.s = append(c.cs.s, fieldName)
	return (FtCreateFieldFieldName)(c)
}

func (c FtCreateFieldFieldTypeTag) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type FtCreateFieldFieldTypeText Completed

func (c FtCreateFieldFieldTypeText) Withsuffixtrie() FtCreateFieldOptionWithsuffixtrie {
	c.cs.s = append(c.cs.s, "WITHSUFFIXTRIE")
	return (FtCreateFieldOptionWithsuffixtrie)(c)
}

func (c FtCreateFieldFieldTypeText) Sortable() FtCreateFieldOptionSortableSortable {
	c.cs.s = append(c.cs.s, "SORTABLE")
	return (FtCreateFieldOptionSortableSortable)(c)
}

func (c FtCreateFieldFieldTypeText) Noindex() FtCreateFieldOptionNoindex {
	c.cs.s = append(c.cs.s, "NOINDEX")
	return (FtCreateFieldOptionNoindex)(c)
}

func (c FtCreateFieldFieldTypeText) Nostem() FtCreateFieldOptionNostem {
	c.cs.s = append(c.cs.s, "NOSTEM")
	return (FtCreateFieldOptionNostem)(c)
}

func (c FtCreateFieldFieldTypeText) Phonetic(phonetic string) FtCreateFieldOptionPhonetic {
	c.cs.s = append(c.cs.s, "PHONETIC", phonetic)
	return (FtCreateFieldOptionPhonetic)(c)
}

func (c FtCreateFieldFieldTypeText) Weight(weight float64) FtCreateFieldOptionWeight {
	c.cs.s = append(c.cs.s, "WEIGHT", strconv.FormatFloat(weight, 'f', -1, 64))
	return (FtCreateFieldOptionWeight)(c)
}

func (c FtCreateFieldFieldTypeText) Separator(separator string) FtCreateFieldOptionSeparator {
	c.cs.s = append(c.cs.s, "SEPARATOR", separator)
	return (FtCreateFieldOptionSeparator)(c)
}

func (c FtCreateFieldFieldTypeText) Casesensitive() FtCreateFieldOptionCasesensitive {
	c.cs.s = append(c.cs.s, "CASESENSITIVE")
	return (FtCreateFieldOptionCasesensitive)(c)
}

func (c FtCreateFieldFieldTypeText) FieldName(fieldName string) FtCreateFieldFieldName {
	c.cs.s = append(c.cs.s, fieldName)
	return (FtCreateFieldFieldName)(c)
}

func (c FtCreateFieldFieldTypeText) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type FtCreateFieldFieldTypeVector Completed

func (c FtCreateFieldFieldTypeVector) Withsuffixtrie() FtCreateFieldOptionWithsuffixtrie {
	c.cs.s = append(c.cs.s, "WITHSUFFIXTRIE")
	return (FtCreateFieldOptionWithsuffixtrie)(c)
}

func (c FtCreateFieldFieldTypeVector) Sortable() FtCreateFieldOptionSortableSortable {
	c.cs.s = append(c.cs.s, "SORTABLE")
	return (FtCreateFieldOptionSortableSortable)(c)
}

func (c FtCreateFieldFieldTypeVector) Noindex() FtCreateFieldOptionNoindex {
	c.cs.s = append(c.cs.s, "NOINDEX")
	return (FtCreateFieldOptionNoindex)(c)
}

func (c FtCreateFieldFieldTypeVector) Nostem() FtCreateFieldOptionNostem {
	c.cs.s = append(c.cs.s, "NOSTEM")
	return (FtCreateFieldOptionNostem)(c)
}

func (c FtCreateFieldFieldTypeVector) Phonetic(phonetic string) FtCreateFieldOptionPhonetic {
	c.cs.s = append(c.cs.s, "PHONETIC", phonetic)
	return (FtCreateFieldOptionPhonetic)(c)
}

func (c FtCreateFieldFieldTypeVector) Weight(weight float64) FtCreateFieldOptionWeight {
	c.cs.s = append(c.cs.s, "WEIGHT", strconv.FormatFloat(weight, 'f', -1, 64))
	return (FtCreateFieldOptionWeight)(c)
}

func (c FtCreateFieldFieldTypeVector) Separator(separator string) FtCreateFieldOptionSeparator {
	c.cs.s = append(c.cs.s, "SEPARATOR", separator)
	return (FtCreateFieldOptionSeparator)(c)
}

func (c FtCreateFieldFieldTypeVector) Casesensitive() FtCreateFieldOptionCasesensitive {
	c.cs.s = append(c.cs.s, "CASESENSITIVE")
	return (FtCreateFieldOptionCasesensitive)(c)
}

func (c FtCreateFieldFieldTypeVector) FieldName(fieldName string) FtCreateFieldFieldName {
	c.cs.s = append(c.cs.s, fieldName)
	return (FtCreateFieldFieldName)(c)
}

func (c FtCreateFieldFieldTypeVector) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type FtCreateFieldOptionCasesensitive Completed

func (c FtCreateFieldOptionCasesensitive) Withsuffixtrie() FtCreateFieldOptionWithsuffixtrie {
	c.cs.s = append(c.cs.s, "WITHSUFFIXTRIE")
	return (FtCreateFieldOptionWithsuffixtrie)(c)
}

func (c FtCreateFieldOptionCasesensitive) Sortable() FtCreateFieldOptionSortableSortable {
	c.cs.s = append(c.cs.s, "SORTABLE")
	return (FtCreateFieldOptionSortableSortable)(c)
}

func (c FtCreateFieldOptionCasesensitive) Noindex() FtCreateFieldOptionNoindex {
	c.cs.s = append(c.cs.s, "NOINDEX")
	return (FtCreateFieldOptionNoindex)(c)
}

func (c FtCreateFieldOptionCasesensitive) Nostem() FtCreateFieldOptionNostem {
	c.cs.s = append(c.cs.s, "NOSTEM")
	return (FtCreateFieldOptionNostem)(c)
}

func (c FtCreateFieldOptionCasesensitive) Phonetic(phonetic string) FtCreateFieldOptionPhonetic {
	c.cs.s = append(c.cs.s, "PHONETIC", phonetic)
	return (FtCreateFieldOptionPhonetic)(c)
}

func (c FtCreateFieldOptionCasesensitive) Weight(weight float64) FtCreateFieldOptionWeight {
	c.cs.s = append(c.cs.s, "WEIGHT", strconv.FormatFloat(weight, 'f', -1, 64))
	return (FtCreateFieldOptionWeight)(c)
}

func (c FtCreateFieldOptionCasesensitive) Separator(separator string) FtCreateFieldOptionSeparator {
	c.cs.s = append(c.cs.s, "SEPARATOR", separator)
	return (FtCreateFieldOptionSeparator)(c)
}

func (c FtCreateFieldOptionCasesensitive) Casesensitive() FtCreateFieldOptionCasesensitive {
	c.cs.s = append(c.cs.s, "CASESENSITIVE")
	return c
}

func (c FtCreateFieldOptionCasesensitive) FieldName(fieldName string) FtCreateFieldFieldName {
	c.cs.s = append(c.cs.s, fieldName)
	return (FtCreateFieldFieldName)(c)
}

func (c FtCreateFieldOptionCasesensitive) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type FtCreateFieldOptionNoindex Completed

func (c FtCreateFieldOptionNoindex) Nostem() FtCreateFieldOptionNostem {
	c.cs.s = append(c.cs.s, "NOSTEM")
	return (FtCreateFieldOptionNostem)(c)
}

func (c FtCreateFieldOptionNoindex) Phonetic(phonetic string) FtCreateFieldOptionPhonetic {
	c.cs.s = append(c.cs.s, "PHONETIC", phonetic)
	return (FtCreateFieldOptionPhonetic)(c)
}

func (c FtCreateFieldOptionNoindex) Weight(weight float64) FtCreateFieldOptionWeight {
	c.cs.s = append(c.cs.s, "WEIGHT", strconv.FormatFloat(weight, 'f', -1, 64))
	return (FtCreateFieldOptionWeight)(c)
}

func (c FtCreateFieldOptionNoindex) Separator(separator string) FtCreateFieldOptionSeparator {
	c.cs.s = append(c.cs.s, "SEPARATOR", separator)
	return (FtCreateFieldOptionSeparator)(c)
}

func (c FtCreateFieldOptionNoindex) Casesensitive() FtCreateFieldOptionCasesensitive {
	c.cs.s = append(c.cs.s, "CASESENSITIVE")
	return (FtCreateFieldOptionCasesensitive)(c)
}

func (c FtCreateFieldOptionNoindex) Withsuffixtrie() FtCreateFieldOptionWithsuffixtrie {
	c.cs.s = append(c.cs.s, "WITHSUFFIXTRIE")
	return (FtCreateFieldOptionWithsuffixtrie)(c)
}

func (c FtCreateFieldOptionNoindex) Sortable() FtCreateFieldOptionSortableSortable {
	c.cs.s = append(c.cs.s, "SORTABLE")
	return (FtCreateFieldOptionSortableSortable)(c)
}

func (c FtCreateFieldOptionNoindex) Noindex() FtCreateFieldOptionNoindex {
	c.cs.s = append(c.cs.s, "NOINDEX")
	return c
}

func (c FtCreateFieldOptionNoindex) FieldName(fieldName string) FtCreateFieldFieldName {
	c.cs.s = append(c.cs.s, fieldName)
	return (FtCreateFieldFieldName)(c)
}

func (c FtCreateFieldOptionNoindex) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type FtCreateFieldOptionNostem Completed

func (c FtCreateFieldOptionNostem) Phonetic(phonetic string) FtCreateFieldOptionPhonetic {
	c.cs.s = append(c.cs.s, "PHONETIC", phonetic)
	return (FtCreateFieldOptionPhonetic)(c)
}

func (c FtCreateFieldOptionNostem) Weight(weight float64) FtCreateFieldOptionWeight {
	c.cs.s = append(c.cs.s, "WEIGHT", strconv.FormatFloat(weight, 'f', -1, 64))
	return (FtCreateFieldOptionWeight)(c)
}

func (c FtCreateFieldOptionNostem) Separator(separator string) FtCreateFieldOptionSeparator {
	c.cs.s = append(c.cs.s, "SEPARATOR", separator)
	return (FtCreateFieldOptionSeparator)(c)
}

func (c FtCreateFieldOptionNostem) Casesensitive() FtCreateFieldOptionCasesensitive {
	c.cs.s = append(c.cs.s, "CASESENSITIVE")
	return (FtCreateFieldOptionCasesensitive)(c)
}

func (c FtCreateFieldOptionNostem) Withsuffixtrie() FtCreateFieldOptionWithsuffixtrie {
	c.cs.s = append(c.cs.s, "WITHSUFFIXTRIE")
	return (FtCreateFieldOptionWithsuffixtrie)(c)
}

func (c FtCreateFieldOptionNostem) Sortable() FtCreateFieldOptionSortableSortable {
	c.cs.s = append(c.cs.s, "SORTABLE")
	return (FtCreateFieldOptionSortableSortable)(c)
}

func (c FtCreateFieldOptionNostem) Noindex() FtCreateFieldOptionNoindex {
	c.cs.s = append(c.cs.s, "NOINDEX")
	return (FtCreateFieldOptionNoindex)(c)
}

func (c FtCreateFieldOptionNostem) Nostem() FtCreateFieldOptionNostem {
	c.cs.s = append(c.cs.s, "NOSTEM")
	return c
}

func (c FtCreateFieldOptionNostem) FieldName(fieldName string) FtCreateFieldFieldName {
	c.cs.s = append(c.cs.s, fieldName)
	return (FtCreateFieldFieldName)(c)
}

func (c FtCreateFieldOptionNostem) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type FtCreateFieldOptionPhonetic Completed

func (c FtCreateFieldOptionPhonetic) Weight(weight float64) FtCreateFieldOptionWeight {
	c.cs.s = append(c.cs.s, "WEIGHT", strconv.FormatFloat(weight, 'f', -1, 64))
	return (FtCreateFieldOptionWeight)(c)
}

func (c FtCreateFieldOptionPhonetic) Separator(separator string) FtCreateFieldOptionSeparator {
	c.cs.s = append(c.cs.s, "SEPARATOR", separator)
	return (FtCreateFieldOptionSeparator)(c)
}

func (c FtCreateFieldOptionPhonetic) Casesensitive() FtCreateFieldOptionCasesensitive {
	c.cs.s = append(c.cs.s, "CASESENSITIVE")
	return (FtCreateFieldOptionCasesensitive)(c)
}

func (c FtCreateFieldOptionPhonetic) Withsuffixtrie() FtCreateFieldOptionWithsuffixtrie {
	c.cs.s = append(c.cs.s, "WITHSUFFIXTRIE")
	return (FtCreateFieldOptionWithsuffixtrie)(c)
}

func (c FtCreateFieldOptionPhonetic) Sortable() FtCreateFieldOptionSortableSortable {
	c.cs.s = append(c.cs.s, "SORTABLE")
	return (FtCreateFieldOptionSortableSortable)(c)
}

func (c FtCreateFieldOptionPhonetic) Noindex() FtCreateFieldOptionNoindex {
	c.cs.s = append(c.cs.s, "NOINDEX")
	return (FtCreateFieldOptionNoindex)(c)
}

func (c FtCreateFieldOptionPhonetic) Nostem() FtCreateFieldOptionNostem {
	c.cs.s = append(c.cs.s, "NOSTEM")
	return (FtCreateFieldOptionNostem)(c)
}

func (c FtCreateFieldOptionPhonetic) Phonetic(phonetic string) FtCreateFieldOptionPhonetic {
	c.cs.s = append(c.cs.s, "PHONETIC", phonetic)
	return c
}

func (c FtCreateFieldOptionPhonetic) FieldName(fieldName string) FtCreateFieldFieldName {
	c.cs.s = append(c.cs.s, fieldName)
	return (FtCreateFieldFieldName)(c)
}

func (c FtCreateFieldOptionPhonetic) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type FtCreateFieldOptionSeparator Completed

func (c FtCreateFieldOptionSeparator) Casesensitive() FtCreateFieldOptionCasesensitive {
	c.cs.s = append(c.cs.s, "CASESENSITIVE")
	return (FtCreateFieldOptionCasesensitive)(c)
}

func (c FtCreateFieldOptionSeparator) Withsuffixtrie() FtCreateFieldOptionWithsuffixtrie {
	c.cs.s = append(c.cs.s, "WITHSUFFIXTRIE")
	return (FtCreateFieldOptionWithsuffixtrie)(c)
}

func (c FtCreateFieldOptionSeparator) Sortable() FtCreateFieldOptionSortableSortable {
	c.cs.s = append(c.cs.s, "SORTABLE")
	return (FtCreateFieldOptionSortableSortable)(c)
}

func (c FtCreateFieldOptionSeparator) Noindex() FtCreateFieldOptionNoindex {
	c.cs.s = append(c.cs.s, "NOINDEX")
	return (FtCreateFieldOptionNoindex)(c)
}

func (c FtCreateFieldOptionSeparator) Nostem() FtCreateFieldOptionNostem {
	c.cs.s = append(c.cs.s, "NOSTEM")
	return (FtCreateFieldOptionNostem)(c)
}

func (c FtCreateFieldOptionSeparator) Phonetic(phonetic string) FtCreateFieldOptionPhonetic {
	c.cs.s = append(c.cs.s, "PHONETIC", phonetic)
	return (FtCreateFieldOptionPhonetic)(c)
}

func (c FtCreateFieldOptionSeparator) Weight(weight float64) FtCreateFieldOptionWeight {
	c.cs.s = append(c.cs.s, "WEIGHT", strconv.FormatFloat(weight, 'f', -1, 64))
	return (FtCreateFieldOptionWeight)(c)
}

func (c FtCreateFieldOptionSeparator) Separator(separator string) FtCreateFieldOptionSeparator {
	c.cs.s = append(c.cs.s, "SEPARATOR", separator)
	return c
}

func (c FtCreateFieldOptionSeparator) FieldName(fieldName string) FtCreateFieldFieldName {
	c.cs.s = append(c.cs.s, fieldName)
	return (FtCreateFieldFieldName)(c)
}

func (c FtCreateFieldOptionSeparator) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type FtCreateFieldOptionSortableSortable Completed

func (c FtCreateFieldOptionSortableSortable) Unf() FtCreateFieldOptionSortableUnf {
	c.cs.s = append(c.cs.s, "UNF")
	return (FtCreateFieldOptionSortableUnf)(c)
}

func (c FtCreateFieldOptionSortableSortable) Noindex() FtCreateFieldOptionNoindex {
	c.cs.s = append(c.cs.s, "NOINDEX")
	return (FtCreateFieldOptionNoindex)(c)
}

func (c FtCreateFieldOptionSortableSortable) Nostem() FtCreateFieldOptionNostem {
	c.cs.s = append(c.cs.s, "NOSTEM")
	return (FtCreateFieldOptionNostem)(c)
}

func (c FtCreateFieldOptionSortableSortable) Phonetic(phonetic string) FtCreateFieldOptionPhonetic {
	c.cs.s = append(c.cs.s, "PHONETIC", phonetic)
	return (FtCreateFieldOptionPhonetic)(c)
}

func (c FtCreateFieldOptionSortableSortable) Weight(weight float64) FtCreateFieldOptionWeight {
	c.cs.s = append(c.cs.s, "WEIGHT", strconv.FormatFloat(weight, 'f', -1, 64))
	return (FtCreateFieldOptionWeight)(c)
}

func (c FtCreateFieldOptionSortableSortable) Separator(separator string) FtCreateFieldOptionSeparator {
	c.cs.s = append(c.cs.s, "SEPARATOR", separator)
	return (FtCreateFieldOptionSeparator)(c)
}

func (c FtCreateFieldOptionSortableSortable) Casesensitive() FtCreateFieldOptionCasesensitive {
	c.cs.s = append(c.cs.s, "CASESENSITIVE")
	return (FtCreateFieldOptionCasesensitive)(c)
}

func (c FtCreateFieldOptionSortableSortable) Withsuffixtrie() FtCreateFieldOptionWithsuffixtrie {
	c.cs.s = append(c.cs.s, "WITHSUFFIXTRIE")
	return (FtCreateFieldOptionWithsuffixtrie)(c)
}

func (c FtCreateFieldOptionSortableSortable) Sortable() FtCreateFieldOptionSortableSortable {
	c.cs.s = append(c.cs.s, "SORTABLE")
	return c
}

func (c FtCreateFieldOptionSortableSortable) FieldName(fieldName string) FtCreateFieldFieldName {
	c.cs.s = append(c.cs.s, fieldName)
	return (FtCreateFieldFieldName)(c)
}

func (c FtCreateFieldOptionSortableSortable) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type FtCreateFieldOptionSortableUnf Completed

func (c FtCreateFieldOptionSortableUnf) Noindex() FtCreateFieldOptionNoindex {
	c.cs.s = append(c.cs.s, "NOINDEX")
	return (FtCreateFieldOptionNoindex)(c)
}

func (c FtCreateFieldOptionSortableUnf) Nostem() FtCreateFieldOptionNostem {
	c.cs.s = append(c.cs.s, "NOSTEM")
	return (FtCreateFieldOptionNostem)(c)
}

func (c FtCreateFieldOptionSortableUnf) Phonetic(phonetic string) FtCreateFieldOptionPhonetic {
	c.cs.s = append(c.cs.s, "PHONETIC", phonetic)
	return (FtCreateFieldOptionPhonetic)(c)
}

func (c FtCreateFieldOptionSortableUnf) Weight(weight float64) FtCreateFieldOptionWeight {
	c.cs.s = append(c.cs.s, "WEIGHT", strconv.FormatFloat(weight, 'f', -1, 64))
	return (FtCreateFieldOptionWeight)(c)
}

func (c FtCreateFieldOptionSortableUnf) Separator(separator string) FtCreateFieldOptionSeparator {
	c.cs.s = append(c.cs.s, "SEPARATOR", separator)
	return (FtCreateFieldOptionSeparator)(c)
}

func (c FtCreateFieldOptionSortableUnf) Casesensitive() FtCreateFieldOptionCasesensitive {
	c.cs.s = append(c.cs.s, "CASESENSITIVE")
	return (FtCreateFieldOptionCasesensitive)(c)
}

func (c FtCreateFieldOptionSortableUnf) Withsuffixtrie() FtCreateFieldOptionWithsuffixtrie {
	c.cs.s = append(c.cs.s, "WITHSUFFIXTRIE")
	return (FtCreateFieldOptionWithsuffixtrie)(c)
}

func (c FtCreateFieldOptionSortableUnf) Sortable() FtCreateFieldOptionSortableSortable {
	c.cs.s = append(c.cs.s, "SORTABLE")
	return (FtCreateFieldOptionSortableSortable)(c)
}

func (c FtCreateFieldOptionSortableUnf) FieldName(fieldName string) FtCreateFieldFieldName {
	c.cs.s = append(c.cs.s, fieldName)
	return (FtCreateFieldFieldName)(c)
}

func (c FtCreateFieldOptionSortableUnf) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type FtCreateFieldOptionWeight Completed

func (c FtCreateFieldOptionWeight) Separator(separator string) FtCreateFieldOptionSeparator {
	c.cs.s = append(c.cs.s, "SEPARATOR", separator)
	return (FtCreateFieldOptionSeparator)(c)
}

func (c FtCreateFieldOptionWeight) Casesensitive() FtCreateFieldOptionCasesensitive {
	c.cs.s = append(c.cs.s, "CASESENSITIVE")
	return (FtCreateFieldOptionCasesensitive)(c)
}

func (c FtCreateFieldOptionWeight) Withsuffixtrie() FtCreateFieldOptionWithsuffixtrie {
	c.cs.s = append(c.cs.s, "WITHSUFFIXTRIE")
	return (FtCreateFieldOptionWithsuffixtrie)(c)
}

func (c FtCreateFieldOptionWeight) Sortable() FtCreateFieldOptionSortableSortable {
	c.cs.s = append(c.cs.s, "SORTABLE")
	return (FtCreateFieldOptionSortableSortable)(c)
}

func (c FtCreateFieldOptionWeight) Noindex() FtCreateFieldOptionNoindex {
	c.cs.s = append(c.cs.s, "NOINDEX")
	return (FtCreateFieldOptionNoindex)(c)
}

func (c FtCreateFieldOptionWeight) Nostem() FtCreateFieldOptionNostem {
	c.cs.s = append(c.cs.s, "NOSTEM")
	return (FtCreateFieldOptionNostem)(c)
}

func (c FtCreateFieldOptionWeight) Phonetic(phonetic string) FtCreateFieldOptionPhonetic {
	c.cs.s = append(c.cs.s, "PHONETIC", phonetic)
	return (FtCreateFieldOptionPhonetic)(c)
}

func (c FtCreateFieldOptionWeight) Weight(weight float64) FtCreateFieldOptionWeight {
	c.cs.s = append(c.cs.s, "WEIGHT", strconv.FormatFloat(weight, 'f', -1, 64))
	return c
}

func (c FtCreateFieldOptionWeight) FieldName(fieldName string) FtCreateFieldFieldName {
	c.cs.s = append(c.cs.s, fieldName)
	return (FtCreateFieldFieldName)(c)
}

func (c FtCreateFieldOptionWeight) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type FtCreateFieldOptionWithsuffixtrie Completed

func (c FtCreateFieldOptionWithsuffixtrie) Sortable() FtCreateFieldOptionSortableSortable {
	c.cs.s = append(c.cs.s, "SORTABLE")
	return (FtCreateFieldOptionSortableSortable)(c)
}

func (c FtCreateFieldOptionWithsuffixtrie) Noindex() FtCreateFieldOptionNoindex {
	c.cs.s = append(c.cs.s, "NOINDEX")
	return (FtCreateFieldOptionNoindex)(c)
}

func (c FtCreateFieldOptionWithsuffixtrie) Nostem() FtCreateFieldOptionNostem {
	c.cs.s = append(c.cs.s, "NOSTEM")
	return (FtCreateFieldOptionNostem)(c)
}

func (c FtCreateFieldOptionWithsuffixtrie) Phonetic(phonetic string) FtCreateFieldOptionPhonetic {
	c.cs.s = append(c.cs.s, "PHONETIC", phonetic)
	return (FtCreateFieldOptionPhonetic)(c)
}

func (c FtCreateFieldOptionWithsuffixtrie) Weight(weight float64) FtCreateFieldOptionWeight {
	c.cs.s = append(c.cs.s, "WEIGHT", strconv.FormatFloat(weight, 'f', -1, 64))
	return (FtCreateFieldOptionWeight)(c)
}

func (c FtCreateFieldOptionWithsuffixtrie) Separator(separator string) FtCreateFieldOptionSeparator {
	c.cs.s = append(c.cs.s, "SEPARATOR", separator)
	return (FtCreateFieldOptionSeparator)(c)
}

func (c FtCreateFieldOptionWithsuffixtrie) Casesensitive() FtCreateFieldOptionCasesensitive {
	c.cs.s = append(c.cs.s, "CASESENSITIVE")
	return (FtCreateFieldOptionCasesensitive)(c)
}

func (c FtCreateFieldOptionWithsuffixtrie) Withsuffixtrie() FtCreateFieldOptionWithsuffixtrie {
	c.cs.s = append(c.cs.s, "WITHSUFFIXTRIE")
	return c
}

func (c FtCreateFieldOptionWithsuffixtrie) FieldName(fieldName string) FtCreateFieldFieldName {
	c.cs.s = append(c.cs.s, fieldName)
	return (FtCreateFieldFieldName)(c)
}

func (c FtCreateFieldOptionWithsuffixtrie) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type FtCreateFilter Completed

func (c FtCreateFilter) Language(defaultLang string) FtCreateLanguage {
	c.cs.s = append(c.cs.s, "LANGUAGE", defaultLang)
	return (FtCreateLanguage)(c)
}

func (c FtCreateFilter) LanguageField(langAttribute string) FtCreateLanguageField {
	c.cs.s = append(c.cs.s, "LANGUAGE_FIELD", langAttribute)
	return (FtCreateLanguageField)(c)
}

func (c FtCreateFilter) Score(defaultScore float64) FtCreateScore {
	c.cs.s = append(c.cs.s, "SCORE", strconv.FormatFloat(defaultScore, 'f', -1, 64))
	return (FtCreateScore)(c)
}

func (c FtCreateFilter) ScoreField(scoreAttribute string) FtCreateScoreField {
	c.cs.s = append(c.cs.s, "SCORE_FIELD", scoreAttribute)
	return (FtCreateScoreField)(c)
}

func (c FtCreateFilter) PayloadField(payloadAttribute string) FtCreatePayloadField {
	c.cs.s = append(c.cs.s, "PAYLOAD_FIELD", payloadAttribute)
	return (FtCreatePayloadField)(c)
}

func (c FtCreateFilter) Maxtextfields() FtCreateMaxtextfields {
	c.cs.s = append(c.cs.s, "MAXTEXTFIELDS")
	return (FtCreateMaxtextfields)(c)
}

func (c FtCreateFilter) Temporary(seconds float64) FtCreateTemporary {
	c.cs.s = append(c.cs.s, "TEMPORARY", strconv.FormatFloat(seconds, 'f', -1, 64))
	return (FtCreateTemporary)(c)
}

func (c FtCreateFilter) Nooffsets() FtCreateNooffsets {
	c.cs.s = append(c.cs.s, "NOOFFSETS")
	return (FtCreateNooffsets)(c)
}

func (c FtCreateFilter) Nohl() FtCreateNohl {
	c.cs.s = append(c.cs.s, "NOHL")
	return (FtCreateNohl)(c)
}

func (c FtCreateFilter) Nofields() FtCreateNofields {
	c.cs.s = append(c.cs.s, "NOFIELDS")
	return (FtCreateNofields)(c)
}

func (c FtCreateFilter) Nofreqs() FtCreateNofreqs {
	c.cs.s = append(c.cs.s, "NOFREQS")
	return (FtCreateNofreqs)(c)
}

func (c FtCreateFilter) Stopwords(count int64) FtCreateStopwordsStopwords {
	c.cs.s = append(c.cs.s, "STOPWORDS", strconv.FormatInt(count, 10))
	return (FtCreateStopwordsStopwords)(c)
}

func (c FtCreateFilter) Skipinitialscan() FtCreateSkipinitialscan {
	c.cs.s = append(c.cs.s, "SKIPINITIALSCAN")
	return (FtCreateSkipinitialscan)(c)
}

func (c FtCreateFilter) Schema() FtCreateSchema {
	c.cs.s = append(c.cs.s, "SCHEMA")
	return (FtCreateSchema)(c)
}

type FtCreateIndex Completed

func (c FtCreateIndex) OnHash() FtCreateOnHash {
	c.cs.s = append(c.cs.s, "ON", "HASH")
	return (FtCreateOnHash)(c)
}

func (c FtCreateIndex) OnJson() FtCreateOnJson {
	c.cs.s = append(c.cs.s, "ON", "JSON")
	return (FtCreateOnJson)(c)
}

func (c FtCreateIndex) Prefix(count int64) FtCreatePrefixCount {
	c.cs.s = append(c.cs.s, "PREFIX", strconv.FormatInt(count, 10))
	return (FtCreatePrefixCount)(c)
}

func (c FtCreateIndex) Filter(filter string) FtCreateFilter {
	c.cs.s = append(c.cs.s, "FILTER", filter)
	return (FtCreateFilter)(c)
}

func (c FtCreateIndex) Language(defaultLang string) FtCreateLanguage {
	c.cs.s = append(c.cs.s, "LANGUAGE", defaultLang)
	return (FtCreateLanguage)(c)
}

func (c FtCreateIndex) LanguageField(langAttribute string) FtCreateLanguageField {
	c.cs.s = append(c.cs.s, "LANGUAGE_FIELD", langAttribute)
	return (FtCreateLanguageField)(c)
}

func (c FtCreateIndex) Score(defaultScore float64) FtCreateScore {
	c.cs.s = append(c.cs.s, "SCORE", strconv.FormatFloat(defaultScore, 'f', -1, 64))
	return (FtCreateScore)(c)
}

func (c FtCreateIndex) ScoreField(scoreAttribute string) FtCreateScoreField {
	c.cs.s = append(c.cs.s, "SCORE_FIELD", scoreAttribute)
	return (FtCreateScoreField)(c)
}

func (c FtCreateIndex) PayloadField(payloadAttribute string) FtCreatePayloadField {
	c.cs.s = append(c.cs.s, "PAYLOAD_FIELD", payloadAttribute)
	return (FtCreatePayloadField)(c)
}

func (c FtCreateIndex) Maxtextfields() FtCreateMaxtextfields {
	c.cs.s = append(c.cs.s, "MAXTEXTFIELDS")
	return (FtCreateMaxtextfields)(c)
}

func (c FtCreateIndex) Temporary(seconds float64) FtCreateTemporary {
	c.cs.s = append(c.cs.s, "TEMPORARY", strconv.FormatFloat(seconds, 'f', -1, 64))
	return (FtCreateTemporary)(c)
}

func (c FtCreateIndex) Nooffsets() FtCreateNooffsets {
	c.cs.s = append(c.cs.s, "NOOFFSETS")
	return (FtCreateNooffsets)(c)
}

func (c FtCreateIndex) Nohl() FtCreateNohl {
	c.cs.s = append(c.cs.s, "NOHL")
	return (FtCreateNohl)(c)
}

func (c FtCreateIndex) Nofields() FtCreateNofields {
	c.cs.s = append(c.cs.s, "NOFIELDS")
	return (FtCreateNofields)(c)
}

func (c FtCreateIndex) Nofreqs() FtCreateNofreqs {
	c.cs.s = append(c.cs.s, "NOFREQS")
	return (FtCreateNofreqs)(c)
}

func (c FtCreateIndex) Stopwords(count int64) FtCreateStopwordsStopwords {
	c.cs.s = append(c.cs.s, "STOPWORDS", strconv.FormatInt(count, 10))
	return (FtCreateStopwordsStopwords)(c)
}

func (c FtCreateIndex) Skipinitialscan() FtCreateSkipinitialscan {
	c.cs.s = append(c.cs.s, "SKIPINITIALSCAN")
	return (FtCreateSkipinitialscan)(c)
}

func (c FtCreateIndex) Schema() FtCreateSchema {
	c.cs.s = append(c.cs.s, "SCHEMA")
	return (FtCreateSchema)(c)
}

type FtCreateLanguage Completed

func (c FtCreateLanguage) LanguageField(langAttribute string) FtCreateLanguageField {
	c.cs.s = append(c.cs.s, "LANGUAGE_FIELD", langAttribute)
	return (FtCreateLanguageField)(c)
}

func (c FtCreateLanguage) Score(defaultScore float64) FtCreateScore {
	c.cs.s = append(c.cs.s, "SCORE", strconv.FormatFloat(defaultScore, 'f', -1, 64))
	return (FtCreateScore)(c)
}

func (c FtCreateLanguage) ScoreField(scoreAttribute string) FtCreateScoreField {
	c.cs.s = append(c.cs.s, "SCORE_FIELD", scoreAttribute)
	return (FtCreateScoreField)(c)
}

func (c FtCreateLanguage) PayloadField(payloadAttribute string) FtCreatePayloadField {
	c.cs.s = append(c.cs.s, "PAYLOAD_FIELD", payloadAttribute)
	return (FtCreatePayloadField)(c)
}

func (c FtCreateLanguage) Maxtextfields() FtCreateMaxtextfields {
	c.cs.s = append(c.cs.s, "MAXTEXTFIELDS")
	return (FtCreateMaxtextfields)(c)
}

func (c FtCreateLanguage) Temporary(seconds float64) FtCreateTemporary {
	c.cs.s = append(c.cs.s, "TEMPORARY", strconv.FormatFloat(seconds, 'f', -1, 64))
	return (FtCreateTemporary)(c)
}

func (c FtCreateLanguage) Nooffsets() FtCreateNooffsets {
	c.cs.s = append(c.cs.s, "NOOFFSETS")
	return (FtCreateNooffsets)(c)
}

func (c FtCreateLanguage) Nohl() FtCreateNohl {
	c.cs.s = append(c.cs.s, "NOHL")
	return (FtCreateNohl)(c)
}

func (c FtCreateLanguage) Nofields() FtCreateNofields {
	c.cs.s = append(c.cs.s, "NOFIELDS")
	return (FtCreateNofields)(c)
}

func (c FtCreateLanguage) Nofreqs() FtCreateNofreqs {
	c.cs.s = append(c.cs.s, "NOFREQS")
	return (FtCreateNofreqs)(c)
}

func (c FtCreateLanguage) Stopwords(count int64) FtCreateStopwordsStopwords {
	c.cs.s = append(c.cs.s, "STOPWORDS", strconv.FormatInt(count, 10))
	return (FtCreateStopwordsStopwords)(c)
}

func (c FtCreateLanguage) Skipinitialscan() FtCreateSkipinitialscan {
	c.cs.s = append(c.cs.s, "SKIPINITIALSCAN")
	return (FtCreateSkipinitialscan)(c)
}

func (c FtCreateLanguage) Schema() FtCreateSchema {
	c.cs.s = append(c.cs.s, "SCHEMA")
	return (FtCreateSchema)(c)
}

type FtCreateLanguageField Completed

func (c FtCreateLanguageField) Score(defaultScore float64) FtCreateScore {
	c.cs.s = append(c.cs.s, "SCORE", strconv.FormatFloat(defaultScore, 'f', -1, 64))
	return (FtCreateScore)(c)
}

func (c FtCreateLanguageField) ScoreField(scoreAttribute string) FtCreateScoreField {
	c.cs.s = append(c.cs.s, "SCORE_FIELD", scoreAttribute)
	return (FtCreateScoreField)(c)
}

func (c FtCreateLanguageField) PayloadField(payloadAttribute string) FtCreatePayloadField {
	c.cs.s = append(c.cs.s, "PAYLOAD_FIELD", payloadAttribute)
	return (FtCreatePayloadField)(c)
}

func (c FtCreateLanguageField) Maxtextfields() FtCreateMaxtextfields {
	c.cs.s = append(c.cs.s, "MAXTEXTFIELDS")
	return (FtCreateMaxtextfields)(c)
}

func (c FtCreateLanguageField) Temporary(seconds float64) FtCreateTemporary {
	c.cs.s = append(c.cs.s, "TEMPORARY", strconv.FormatFloat(seconds, 'f', -1, 64))
	return (FtCreateTemporary)(c)
}

func (c FtCreateLanguageField) Nooffsets() FtCreateNooffsets {
	c.cs.s = append(c.cs.s, "NOOFFSETS")
	return (FtCreateNooffsets)(c)
}

func (c FtCreateLanguageField) Nohl() FtCreateNohl {
	c.cs.s = append(c.cs.s, "NOHL")
	return (FtCreateNohl)(c)
}

func (c FtCreateLanguageField) Nofields() FtCreateNofields {
	c.cs.s = append(c.cs.s, "NOFIELDS")
	return (FtCreateNofields)(c)
}

func (c FtCreateLanguageField) Nofreqs() FtCreateNofreqs {
	c.cs.s = append(c.cs.s, "NOFREQS")
	return (FtCreateNofreqs)(c)
}

func (c FtCreateLanguageField) Stopwords(count int64) FtCreateStopwordsStopwords {
	c.cs.s = append(c.cs.s, "STOPWORDS", strconv.FormatInt(count, 10))
	return (FtCreateStopwordsStopwords)(c)
}

func (c FtCreateLanguageField) Skipinitialscan() FtCreateSkipinitialscan {
	c.cs.s = append(c.cs.s, "SKIPINITIALSCAN")
	return (FtCreateSkipinitialscan)(c)
}

func (c FtCreateLanguageField) Schema() FtCreateSchema {
	c.cs.s = append(c.cs.s, "SCHEMA")
	return (FtCreateSchema)(c)
}

type FtCreateMaxtextfields Completed

func (c FtCreateMaxtextfields) Temporary(seconds float64) FtCreateTemporary {
	c.cs.s = append(c.cs.s, "TEMPORARY", strconv.FormatFloat(seconds, 'f', -1, 64))
	return (FtCreateTemporary)(c)
}

func (c FtCreateMaxtextfields) Nooffsets() FtCreateNooffsets {
	c.cs.s = append(c.cs.s, "NOOFFSETS")
	return (FtCreateNooffsets)(c)
}

func (c FtCreateMaxtextfields) Nohl() FtCreateNohl {
	c.cs.s = append(c.cs.s, "NOHL")
	return (FtCreateNohl)(c)
}

func (c FtCreateMaxtextfields) Nofields() FtCreateNofields {
	c.cs.s = append(c.cs.s, "NOFIELDS")
	return (FtCreateNofields)(c)
}

func (c FtCreateMaxtextfields) Nofreqs() FtCreateNofreqs {
	c.cs.s = append(c.cs.s, "NOFREQS")
	return (FtCreateNofreqs)(c)
}

func (c FtCreateMaxtextfields) Stopwords(count int64) FtCreateStopwordsStopwords {
	c.cs.s = append(c.cs.s, "STOPWORDS", strconv.FormatInt(count, 10))
	return (FtCreateStopwordsStopwords)(c)
}

func (c FtCreateMaxtextfields) Skipinitialscan() FtCreateSkipinitialscan {
	c.cs.s = append(c.cs.s, "SKIPINITIALSCAN")
	return (FtCreateSkipinitialscan)(c)
}

func (c FtCreateMaxtextfields) Schema() FtCreateSchema {
	c.cs.s = append(c.cs.s, "SCHEMA")
	return (FtCreateSchema)(c)
}

type FtCreateNofields Completed

func (c FtCreateNofields) Nofreqs() FtCreateNofreqs {
	c.cs.s = append(c.cs.s, "NOFREQS")
	return (FtCreateNofreqs)(c)
}

func (c FtCreateNofields) Stopwords(count int64) FtCreateStopwordsStopwords {
	c.cs.s = append(c.cs.s, "STOPWORDS", strconv.FormatInt(count, 10))
	return (FtCreateStopwordsStopwords)(c)
}

func (c FtCreateNofields) Skipinitialscan() FtCreateSkipinitialscan {
	c.cs.s = append(c.cs.s, "SKIPINITIALSCAN")
	return (FtCreateSkipinitialscan)(c)
}

func (c FtCreateNofields) Schema() FtCreateSchema {
	c.cs.s = append(c.cs.s, "SCHEMA")
	return (FtCreateSchema)(c)
}

type FtCreateNofreqs Completed

func (c FtCreateNofreqs) Stopwords(count int64) FtCreateStopwordsStopwords {
	c.cs.s = append(c.cs.s, "STOPWORDS", strconv.FormatInt(count, 10))
	return (FtCreateStopwordsStopwords)(c)
}

func (c FtCreateNofreqs) Skipinitialscan() FtCreateSkipinitialscan {
	c.cs.s = append(c.cs.s, "SKIPINITIALSCAN")
	return (FtCreateSkipinitialscan)(c)
}

func (c FtCreateNofreqs) Schema() FtCreateSchema {
	c.cs.s = append(c.cs.s, "SCHEMA")
	return (FtCreateSchema)(c)
}

type FtCreateNohl Completed

func (c FtCreateNohl) Nofields() FtCreateNofields {
	c.cs.s = append(c.cs.s, "NOFIELDS")
	return (FtCreateNofields)(c)
}

func (c FtCreateNohl) Nofreqs() FtCreateNofreqs {
	c.cs.s = append(c.cs.s, "NOFREQS")
	return (FtCreateNofreqs)(c)
}

func (c FtCreateNohl) Stopwords(count int64) FtCreateStopwordsStopwords {
	c.cs.s = append(c.cs.s, "STOPWORDS", strconv.FormatInt(count, 10))
	return (FtCreateStopwordsStopwords)(c)
}

func (c FtCreateNohl) Skipinitialscan() FtCreateSkipinitialscan {
	c.cs.s = append(c.cs.s, "SKIPINITIALSCAN")
	return (FtCreateSkipinitialscan)(c)
}

func (c FtCreateNohl) Schema() FtCreateSchema {
	c.cs.s = append(c.cs.s, "SCHEMA")
	return (FtCreateSchema)(c)
}

type FtCreateNooffsets Completed

func (c FtCreateNooffsets) Nohl() FtCreateNohl {
	c.cs.s = append(c.cs.s, "NOHL")
	return (FtCreateNohl)(c)
}

func (c FtCreateNooffsets) Nofields() FtCreateNofields {
	c.cs.s = append(c.cs.s, "NOFIELDS")
	return (FtCreateNofields)(c)
}

func (c FtCreateNooffsets) Nofreqs() FtCreateNofreqs {
	c.cs.s = append(c.cs.s, "NOFREQS")
	return (FtCreateNofreqs)(c)
}

func (c FtCreateNooffsets) Stopwords(count int64) FtCreateStopwordsStopwords {
	c.cs.s = append(c.cs.s, "STOPWORDS", strconv.FormatInt(count, 10))
	return (FtCreateStopwordsStopwords)(c)
}

func (c FtCreateNooffsets) Skipinitialscan() FtCreateSkipinitialscan {
	c.cs.s = append(c.cs.s, "SKIPINITIALSCAN")
	return (FtCreateSkipinitialscan)(c)
}

func (c FtCreateNooffsets) Schema() FtCreateSchema {
	c.cs.s = append(c.cs.s, "SCHEMA")
	return (FtCreateSchema)(c)
}

type FtCreateOnHash Completed

func (c FtCreateOnHash) Prefix(count int64) FtCreatePrefixCount {
	c.cs.s = append(c.cs.s, "PREFIX", strconv.FormatInt(count, 10))
	return (FtCreatePrefixCount)(c)
}

func (c FtCreateOnHash) Filter(filter string) FtCreateFilter {
	c.cs.s = append(c.cs.s, "FILTER", filter)
	return (FtCreateFilter)(c)
}

func (c FtCreateOnHash) Language(defaultLang string) FtCreateLanguage {
	c.cs.s = append(c.cs.s, "LANGUAGE", defaultLang)
	return (FtCreateLanguage)(c)
}

func (c FtCreateOnHash) LanguageField(langAttribute string) FtCreateLanguageField {
	c.cs.s = append(c.cs.s, "LANGUAGE_FIELD", langAttribute)
	return (FtCreateLanguageField)(c)
}

func (c FtCreateOnHash) Score(defaultScore float64) FtCreateScore {
	c.cs.s = append(c.cs.s, "SCORE", strconv.FormatFloat(defaultScore, 'f', -1, 64))
	return (FtCreateScore)(c)
}

func (c FtCreateOnHash) ScoreField(scoreAttribute string) FtCreateScoreField {
	c.cs.s = append(c.cs.s, "SCORE_FIELD", scoreAttribute)
	return (FtCreateScoreField)(c)
}

func (c FtCreateOnHash) PayloadField(payloadAttribute string) FtCreatePayloadField {
	c.cs.s = append(c.cs.s, "PAYLOAD_FIELD", payloadAttribute)
	return (FtCreatePayloadField)(c)
}

func (c FtCreateOnHash) Maxtextfields() FtCreateMaxtextfields {
	c.cs.s = append(c.cs.s, "MAXTEXTFIELDS")
	return (FtCreateMaxtextfields)(c)
}

func (c FtCreateOnHash) Temporary(seconds float64) FtCreateTemporary {
	c.cs.s = append(c.cs.s, "TEMPORARY", strconv.FormatFloat(seconds, 'f', -1, 64))
	return (FtCreateTemporary)(c)
}

func (c FtCreateOnHash) Nooffsets() FtCreateNooffsets {
	c.cs.s = append(c.cs.s, "NOOFFSETS")
	return (FtCreateNooffsets)(c)
}

func (c FtCreateOnHash) Nohl() FtCreateNohl {
	c.cs.s = append(c.cs.s, "NOHL")
	return (FtCreateNohl)(c)
}

func (c FtCreateOnHash) Nofields() FtCreateNofields {
	c.cs.s = append(c.cs.s, "NOFIELDS")
	return (FtCreateNofields)(c)
}

func (c FtCreateOnHash) Nofreqs() FtCreateNofreqs {
	c.cs.s = append(c.cs.s, "NOFREQS")
	return (FtCreateNofreqs)(c)
}

func (c FtCreateOnHash) Stopwords(count int64) FtCreateStopwordsStopwords {
	c.cs.s = append(c.cs.s, "STOPWORDS", strconv.FormatInt(count, 10))
	return (FtCreateStopwordsStopwords)(c)
}

func (c FtCreateOnHash) Skipinitialscan() FtCreateSkipinitialscan {
	c.cs.s = append(c.cs.s, "SKIPINITIALSCAN")
	return (FtCreateSkipinitialscan)(c)
}

func (c FtCreateOnHash) Schema() FtCreateSchema {
	c.cs.s = append(c.cs.s, "SCHEMA")
	return (FtCreateSchema)(c)
}

type FtCreateOnJson Completed

func (c FtCreateOnJson) Prefix(count int64) FtCreatePrefixCount {
	c.cs.s = append(c.cs.s, "PREFIX", strconv.FormatInt(count, 10))
	return (FtCreatePrefixCount)(c)
}

func (c FtCreateOnJson) Filter(filter string) FtCreateFilter {
	c.cs.s = append(c.cs.s, "FILTER", filter)
	return (FtCreateFilter)(c)
}

func (c FtCreateOnJson) Language(defaultLang string) FtCreateLanguage {
	c.cs.s = append(c.cs.s, "LANGUAGE", defaultLang)
	return (FtCreateLanguage)(c)
}

func (c FtCreateOnJson) LanguageField(langAttribute string) FtCreateLanguageField {
	c.cs.s = append(c.cs.s, "LANGUAGE_FIELD", langAttribute)
	return (FtCreateLanguageField)(c)
}

func (c FtCreateOnJson) Score(defaultScore float64) FtCreateScore {
	c.cs.s = append(c.cs.s, "SCORE", strconv.FormatFloat(defaultScore, 'f', -1, 64))
	return (FtCreateScore)(c)
}

func (c FtCreateOnJson) ScoreField(scoreAttribute string) FtCreateScoreField {
	c.cs.s = append(c.cs.s, "SCORE_FIELD", scoreAttribute)
	return (FtCreateScoreField)(c)
}

func (c FtCreateOnJson) PayloadField(payloadAttribute string) FtCreatePayloadField {
	c.cs.s = append(c.cs.s, "PAYLOAD_FIELD", payloadAttribute)
	return (FtCreatePayloadField)(c)
}

func (c FtCreateOnJson) Maxtextfields() FtCreateMaxtextfields {
	c.cs.s = append(c.cs.s, "MAXTEXTFIELDS")
	return (FtCreateMaxtextfields)(c)
}

func (c FtCreateOnJson) Temporary(seconds float64) FtCreateTemporary {
	c.cs.s = append(c.cs.s, "TEMPORARY", strconv.FormatFloat(seconds, 'f', -1, 64))
	return (FtCreateTemporary)(c)
}

func (c FtCreateOnJson) Nooffsets() FtCreateNooffsets {
	c.cs.s = append(c.cs.s, "NOOFFSETS")
	return (FtCreateNooffsets)(c)
}

func (c FtCreateOnJson) Nohl() FtCreateNohl {
	c.cs.s = append(c.cs.s, "NOHL")
	return (FtCreateNohl)(c)
}

func (c FtCreateOnJson) Nofields() FtCreateNofields {
	c.cs.s = append(c.cs.s, "NOFIELDS")
	return (FtCreateNofields)(c)
}

func (c FtCreateOnJson) Nofreqs() FtCreateNofreqs {
	c.cs.s = append(c.cs.s, "NOFREQS")
	return (FtCreateNofreqs)(c)
}

func (c FtCreateOnJson) Stopwords(count int64) FtCreateStopwordsStopwords {
	c.cs.s = append(c.cs.s, "STOPWORDS", strconv.FormatInt(count, 10))
	return (FtCreateStopwordsStopwords)(c)
}

func (c FtCreateOnJson) Skipinitialscan() FtCreateSkipinitialscan {
	c.cs.s = append(c.cs.s, "SKIPINITIALSCAN")
	return (FtCreateSkipinitialscan)(c)
}

func (c FtCreateOnJson) Schema() FtCreateSchema {
	c.cs.s = append(c.cs.s, "SCHEMA")
	return (FtCreateSchema)(c)
}

type FtCreatePayloadField Completed

func (c FtCreatePayloadField) Maxtextfields() FtCreateMaxtextfields {
	c.cs.s = append(c.cs.s, "MAXTEXTFIELDS")
	return (FtCreateMaxtextfields)(c)
}

func (c FtCreatePayloadField) Temporary(seconds float64) FtCreateTemporary {
	c.cs.s = append(c.cs.s, "TEMPORARY", strconv.FormatFloat(seconds, 'f', -1, 64))
	return (FtCreateTemporary)(c)
}

func (c FtCreatePayloadField) Nooffsets() FtCreateNooffsets {
	c.cs.s = append(c.cs.s, "NOOFFSETS")
	return (FtCreateNooffsets)(c)
}

func (c FtCreatePayloadField) Nohl() FtCreateNohl {
	c.cs.s = append(c.cs.s, "NOHL")
	return (FtCreateNohl)(c)
}

func (c FtCreatePayloadField) Nofields() FtCreateNofields {
	c.cs.s = append(c.cs.s, "NOFIELDS")
	return (FtCreateNofields)(c)
}

func (c FtCreatePayloadField) Nofreqs() FtCreateNofreqs {
	c.cs.s = append(c.cs.s, "NOFREQS")
	return (FtCreateNofreqs)(c)
}

func (c FtCreatePayloadField) Stopwords(count int64) FtCreateStopwordsStopwords {
	c.cs.s = append(c.cs.s, "STOPWORDS", strconv.FormatInt(count, 10))
	return (FtCreateStopwordsStopwords)(c)
}

func (c FtCreatePayloadField) Skipinitialscan() FtCreateSkipinitialscan {
	c.cs.s = append(c.cs.s, "SKIPINITIALSCAN")
	return (FtCreateSkipinitialscan)(c)
}

func (c FtCreatePayloadField) Schema() FtCreateSchema {
	c.cs.s = append(c.cs.s, "SCHEMA")
	return (FtCreateSchema)(c)
}

type FtCreatePrefixCount Completed

func (c FtCreatePrefixCount) Prefix(prefix ...string) FtCreatePrefixPrefix {
	c.cs.s = append(c.cs.s, prefix...)
	return (FtCreatePrefixPrefix)(c)
}

type FtCreatePrefixPrefix Completed

func (c FtCreatePrefixPrefix) Prefix(prefix ...string) FtCreatePrefixPrefix {
	c.cs.s = append(c.cs.s, prefix...)
	return c
}

func (c FtCreatePrefixPrefix) Filter(filter string) FtCreateFilter {
	c.cs.s = append(c.cs.s, "FILTER", filter)
	return (FtCreateFilter)(c)
}

func (c FtCreatePrefixPrefix) Language(defaultLang string) FtCreateLanguage {
	c.cs.s = append(c.cs.s, "LANGUAGE", defaultLang)
	return (FtCreateLanguage)(c)
}

func (c FtCreatePrefixPrefix) LanguageField(langAttribute string) FtCreateLanguageField {
	c.cs.s = append(c.cs.s, "LANGUAGE_FIELD", langAttribute)
	return (FtCreateLanguageField)(c)
}

func (c FtCreatePrefixPrefix) Score(defaultScore float64) FtCreateScore {
	c.cs.s = append(c.cs.s, "SCORE", strconv.FormatFloat(defaultScore, 'f', -1, 64))
	return (FtCreateScore)(c)
}

func (c FtCreatePrefixPrefix) ScoreField(scoreAttribute string) FtCreateScoreField {
	c.cs.s = append(c.cs.s, "SCORE_FIELD", scoreAttribute)
	return (FtCreateScoreField)(c)
}

func (c FtCreatePrefixPrefix) PayloadField(payloadAttribute string) FtCreatePayloadField {
	c.cs.s = append(c.cs.s, "PAYLOAD_FIELD", payloadAttribute)
	return (FtCreatePayloadField)(c)
}

func (c FtCreatePrefixPrefix) Maxtextfields() FtCreateMaxtextfields {
	c.cs.s = append(c.cs.s, "MAXTEXTFIELDS")
	return (FtCreateMaxtextfields)(c)
}

func (c FtCreatePrefixPrefix) Temporary(seconds float64) FtCreateTemporary {
	c.cs.s = append(c.cs.s, "TEMPORARY", strconv.FormatFloat(seconds, 'f', -1, 64))
	return (FtCreateTemporary)(c)
}

func (c FtCreatePrefixPrefix) Nooffsets() FtCreateNooffsets {
	c.cs.s = append(c.cs.s, "NOOFFSETS")
	return (FtCreateNooffsets)(c)
}

func (c FtCreatePrefixPrefix) Nohl() FtCreateNohl {
	c.cs.s = append(c.cs.s, "NOHL")
	return (FtCreateNohl)(c)
}

func (c FtCreatePrefixPrefix) Nofields() FtCreateNofields {
	c.cs.s = append(c.cs.s, "NOFIELDS")
	return (FtCreateNofields)(c)
}

func (c FtCreatePrefixPrefix) Nofreqs() FtCreateNofreqs {
	c.cs.s = append(c.cs.s, "NOFREQS")
	return (FtCreateNofreqs)(c)
}

func (c FtCreatePrefixPrefix) Stopwords(count int64) FtCreateStopwordsStopwords {
	c.cs.s = append(c.cs.s, "STOPWORDS", strconv.FormatInt(count, 10))
	return (FtCreateStopwordsStopwords)(c)
}

func (c FtCreatePrefixPrefix) Skipinitialscan() FtCreateSkipinitialscan {
	c.cs.s = append(c.cs.s, "SKIPINITIALSCAN")
	return (FtCreateSkipinitialscan)(c)
}

func (c FtCreatePrefixPrefix) Schema() FtCreateSchema {
	c.cs.s = append(c.cs.s, "SCHEMA")
	return (FtCreateSchema)(c)
}

type FtCreateSchema Completed

func (c FtCreateSchema) FieldName(fieldName string) FtCreateFieldFieldName {
	c.cs.s = append(c.cs.s, fieldName)
	return (FtCreateFieldFieldName)(c)
}

type FtCreateScore Completed

func (c FtCreateScore) ScoreField(scoreAttribute string) FtCreateScoreField {
	c.cs.s = append(c.cs.s, "SCORE_FIELD", scoreAttribute)
	return (FtCreateScoreField)(c)
}

func (c FtCreateScore) PayloadField(payloadAttribute string) FtCreatePayloadField {
	c.cs.s = append(c.cs.s, "PAYLOAD_FIELD", payloadAttribute)
	return (FtCreatePayloadField)(c)
}

func (c FtCreateScore) Maxtextfields() FtCreateMaxtextfields {
	c.cs.s = append(c.cs.s, "MAXTEXTFIELDS")
	return (FtCreateMaxtextfields)(c)
}

func (c FtCreateScore) Temporary(seconds float64) FtCreateTemporary {
	c.cs.s = append(c.cs.s, "TEMPORARY", strconv.FormatFloat(seconds, 'f', -1, 64))
	return (FtCreateTemporary)(c)
}

func (c FtCreateScore) Nooffsets() FtCreateNooffsets {
	c.cs.s = append(c.cs.s, "NOOFFSETS")
	return (FtCreateNooffsets)(c)
}

func (c FtCreateScore) Nohl() FtCreateNohl {
	c.cs.s = append(c.cs.s, "NOHL")
	return (FtCreateNohl)(c)
}

func (c FtCreateScore) Nofields() FtCreateNofields {
	c.cs.s = append(c.cs.s, "NOFIELDS")
	return (FtCreateNofields)(c)
}

func (c FtCreateScore) Nofreqs() FtCreateNofreqs {
	c.cs.s = append(c.cs.s, "NOFREQS")
	return (FtCreateNofreqs)(c)
}

func (c FtCreateScore) Stopwords(count int64) FtCreateStopwordsStopwords {
	c.cs.s = append(c.cs.s, "STOPWORDS", strconv.FormatInt(count, 10))
	return (FtCreateStopwordsStopwords)(c)
}

func (c FtCreateScore) Skipinitialscan() FtCreateSkipinitialscan {
	c.cs.s = append(c.cs.s, "SKIPINITIALSCAN")
	return (FtCreateSkipinitialscan)(c)
}

func (c FtCreateScore) Schema() FtCreateSchema {
	c.cs.s = append(c.cs.s, "SCHEMA")
	return (FtCreateSchema)(c)
}

type FtCreateScoreField Completed

func (c FtCreateScoreField) PayloadField(payloadAttribute string) FtCreatePayloadField {
	c.cs.s = append(c.cs.s, "PAYLOAD_FIELD", payloadAttribute)
	return (FtCreatePayloadField)(c)
}

func (c FtCreateScoreField) Maxtextfields() FtCreateMaxtextfields {
	c.cs.s = append(c.cs.s, "MAXTEXTFIELDS")
	return (FtCreateMaxtextfields)(c)
}

func (c FtCreateScoreField) Temporary(seconds float64) FtCreateTemporary {
	c.cs.s = append(c.cs.s, "TEMPORARY", strconv.FormatFloat(seconds, 'f', -1, 64))
	return (FtCreateTemporary)(c)
}

func (c FtCreateScoreField) Nooffsets() FtCreateNooffsets {
	c.cs.s = append(c.cs.s, "NOOFFSETS")
	return (FtCreateNooffsets)(c)
}

func (c FtCreateScoreField) Nohl() FtCreateNohl {
	c.cs.s = append(c.cs.s, "NOHL")
	return (FtCreateNohl)(c)
}

func (c FtCreateScoreField) Nofields() FtCreateNofields {
	c.cs.s = append(c.cs.s, "NOFIELDS")
	return (FtCreateNofields)(c)
}

func (c FtCreateScoreField) Nofreqs() FtCreateNofreqs {
	c.cs.s = append(c.cs.s, "NOFREQS")
	return (FtCreateNofreqs)(c)
}

func (c FtCreateScoreField) Stopwords(count int64) FtCreateStopwordsStopwords {
	c.cs.s = append(c.cs.s, "STOPWORDS", strconv.FormatInt(count, 10))
	return (FtCreateStopwordsStopwords)(c)
}

func (c FtCreateScoreField) Skipinitialscan() FtCreateSkipinitialscan {
	c.cs.s = append(c.cs.s, "SKIPINITIALSCAN")
	return (FtCreateSkipinitialscan)(c)
}

func (c FtCreateScoreField) Schema() FtCreateSchema {
	c.cs.s = append(c.cs.s, "SCHEMA")
	return (FtCreateSchema)(c)
}

type FtCreateSkipinitialscan Completed

func (c FtCreateSkipinitialscan) Schema() FtCreateSchema {
	c.cs.s = append(c.cs.s, "SCHEMA")
	return (FtCreateSchema)(c)
}

type FtCreateStopwordsStopword Completed

func (c FtCreateStopwordsStopword) Stopword(stopword ...string) FtCreateStopwordsStopword {
	c.cs.s = append(c.cs.s, stopword...)
	return c
}

func (c FtCreateStopwordsStopword) Skipinitialscan() FtCreateSkipinitialscan {
	c.cs.s = append(c.cs.s, "SKIPINITIALSCAN")
	return (FtCreateSkipinitialscan)(c)
}

func (c FtCreateStopwordsStopword) Schema() FtCreateSchema {
	c.cs.s = append(c.cs.s, "SCHEMA")
	return (FtCreateSchema)(c)
}

type FtCreateStopwordsStopwords Completed

func (c FtCreateStopwordsStopwords) Stopword(stopword ...string) FtCreateStopwordsStopword {
	c.cs.s = append(c.cs.s, stopword...)
	return (FtCreateStopwordsStopword)(c)
}

func (c FtCreateStopwordsStopwords) Skipinitialscan() FtCreateSkipinitialscan {
	c.cs.s = append(c.cs.s, "SKIPINITIALSCAN")
	return (FtCreateSkipinitialscan)(c)
}

func (c FtCreateStopwordsStopwords) Schema() FtCreateSchema {
	c.cs.s = append(c.cs.s, "SCHEMA")
	return (FtCreateSchema)(c)
}

type FtCreateTemporary Completed

func (c FtCreateTemporary) Nooffsets() FtCreateNooffsets {
	c.cs.s = append(c.cs.s, "NOOFFSETS")
	return (FtCreateNooffsets)(c)
}

func (c FtCreateTemporary) Nohl() FtCreateNohl {
	c.cs.s = append(c.cs.s, "NOHL")
	return (FtCreateNohl)(c)
}

func (c FtCreateTemporary) Nofields() FtCreateNofields {
	c.cs.s = append(c.cs.s, "NOFIELDS")
	return (FtCreateNofields)(c)
}

func (c FtCreateTemporary) Nofreqs() FtCreateNofreqs {
	c.cs.s = append(c.cs.s, "NOFREQS")
	return (FtCreateNofreqs)(c)
}

func (c FtCreateTemporary) Stopwords(count int64) FtCreateStopwordsStopwords {
	c.cs.s = append(c.cs.s, "STOPWORDS", strconv.FormatInt(count, 10))
	return (FtCreateStopwordsStopwords)(c)
}

func (c FtCreateTemporary) Skipinitialscan() FtCreateSkipinitialscan {
	c.cs.s = append(c.cs.s, "SKIPINITIALSCAN")
	return (FtCreateSkipinitialscan)(c)
}

func (c FtCreateTemporary) Schema() FtCreateSchema {
	c.cs.s = append(c.cs.s, "SCHEMA")
	return (FtCreateSchema)(c)
}

type FtCursorDel Completed

func (b Builder) FtCursorDel() (c FtCursorDel) {
	c = FtCursorDel{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "FT.CURSOR", "DEL")
	return c
}

func (c FtCursorDel) Index(index string) FtCursorDelIndex {
	c.cs.s = append(c.cs.s, index)
	return (FtCursorDelIndex)(c)
}

type FtCursorDelCursorId Completed

func (c FtCursorDelCursorId) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type FtCursorDelIndex Completed

func (c FtCursorDelIndex) CursorId(cursorId int64) FtCursorDelCursorId {
	c.cs.s = append(c.cs.s, strconv.FormatInt(cursorId, 10))
	return (FtCursorDelCursorId)(c)
}

type FtCursorRead Completed

func (b Builder) FtCursorRead() (c FtCursorRead) {
	c = FtCursorRead{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "FT.CURSOR", "READ")
	return c
}

func (c FtCursorRead) Index(index string) FtCursorReadIndex {
	c.cs.s = append(c.cs.s, index)
	return (FtCursorReadIndex)(c)
}

type FtCursorReadCount Completed

func (c FtCursorReadCount) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type FtCursorReadCursorId Completed

func (c FtCursorReadCursorId) Count(readSize int64) FtCursorReadCount {
	c.cs.s = append(c.cs.s, "COUNT", strconv.FormatInt(readSize, 10))
	return (FtCursorReadCount)(c)
}

func (c FtCursorReadCursorId) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type FtCursorReadIndex Completed

func (c FtCursorReadIndex) CursorId(cursorId int64) FtCursorReadCursorId {
	c.cs.s = append(c.cs.s, strconv.FormatInt(cursorId, 10))
	return (FtCursorReadCursorId)(c)
}

type FtDictadd Completed

func (b Builder) FtDictadd() (c FtDictadd) {
	c = FtDictadd{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "FT.DICTADD")
	return c
}

func (c FtDictadd) Dict(dict string) FtDictaddDict {
	c.cs.s = append(c.cs.s, dict)
	return (FtDictaddDict)(c)
}

type FtDictaddDict Completed

func (c FtDictaddDict) Term(term ...string) FtDictaddTerm {
	c.cs.s = append(c.cs.s, term...)
	return (FtDictaddTerm)(c)
}

type FtDictaddTerm Completed

func (c FtDictaddTerm) Term(term ...string) FtDictaddTerm {
	c.cs.s = append(c.cs.s, term...)
	return c
}

func (c FtDictaddTerm) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type FtDictdel Completed

func (b Builder) FtDictdel() (c FtDictdel) {
	c = FtDictdel{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "FT.DICTDEL")
	return c
}

func (c FtDictdel) Dict(dict string) FtDictdelDict {
	c.cs.s = append(c.cs.s, dict)
	return (FtDictdelDict)(c)
}

type FtDictdelDict Completed

func (c FtDictdelDict) Term(term ...string) FtDictdelTerm {
	c.cs.s = append(c.cs.s, term...)
	return (FtDictdelTerm)(c)
}

type FtDictdelTerm Completed

func (c FtDictdelTerm) Term(term ...string) FtDictdelTerm {
	c.cs.s = append(c.cs.s, term...)
	return c
}

func (c FtDictdelTerm) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type FtDictdump Completed

func (b Builder) FtDictdump() (c FtDictdump) {
	c = FtDictdump{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "FT.DICTDUMP")
	return c
}

func (c FtDictdump) Dict(dict string) FtDictdumpDict {
	c.cs.s = append(c.cs.s, dict)
	return (FtDictdumpDict)(c)
}

type FtDictdumpDict Completed

func (c FtDictdumpDict) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type FtDropindex Completed

func (b Builder) FtDropindex() (c FtDropindex) {
	c = FtDropindex{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "FT.DROPINDEX")
	return c
}

func (c FtDropindex) Index(index string) FtDropindexIndex {
	c.cs.s = append(c.cs.s, index)
	return (FtDropindexIndex)(c)
}

type FtDropindexDeleteDocsDd Completed

func (c FtDropindexDeleteDocsDd) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type FtDropindexIndex Completed

func (c FtDropindexIndex) Dd() FtDropindexDeleteDocsDd {
	c.cs.s = append(c.cs.s, "DD")
	return (FtDropindexDeleteDocsDd)(c)
}

func (c FtDropindexIndex) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type FtExplain Completed

func (b Builder) FtExplain() (c FtExplain) {
	c = FtExplain{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "FT.EXPLAIN")
	return c
}

func (c FtExplain) Index(index string) FtExplainIndex {
	c.cs.s = append(c.cs.s, index)
	return (FtExplainIndex)(c)
}

type FtExplainDialect Completed

func (c FtExplainDialect) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type FtExplainIndex Completed

func (c FtExplainIndex) Query(query string) FtExplainQuery {
	c.cs.s = append(c.cs.s, query)
	return (FtExplainQuery)(c)
}

type FtExplainQuery Completed

func (c FtExplainQuery) Dialect(dialect int64) FtExplainDialect {
	c.cs.s = append(c.cs.s, "DIALECT", strconv.FormatInt(dialect, 10))
	return (FtExplainDialect)(c)
}

func (c FtExplainQuery) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type FtExplaincli Completed

func (b Builder) FtExplaincli() (c FtExplaincli) {
	c = FtExplaincli{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "FT.EXPLAINCLI")
	return c
}

func (c FtExplaincli) Index(index string) FtExplaincliIndex {
	c.cs.s = append(c.cs.s, index)
	return (FtExplaincliIndex)(c)
}

type FtExplaincliDialect Completed

func (c FtExplaincliDialect) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type FtExplaincliIndex Completed

func (c FtExplaincliIndex) Query(query string) FtExplaincliQuery {
	c.cs.s = append(c.cs.s, query)
	return (FtExplaincliQuery)(c)
}

type FtExplaincliQuery Completed

func (c FtExplaincliQuery) Dialect(dialect int64) FtExplaincliDialect {
	c.cs.s = append(c.cs.s, "DIALECT", strconv.FormatInt(dialect, 10))
	return (FtExplaincliDialect)(c)
}

func (c FtExplaincliQuery) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type FtInfo Completed

func (b Builder) FtInfo() (c FtInfo) {
	c = FtInfo{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "FT.INFO")
	return c
}

func (c FtInfo) Index(index string) FtInfoIndex {
	c.cs.s = append(c.cs.s, index)
	return (FtInfoIndex)(c)
}

type FtInfoIndex Completed

func (c FtInfoIndex) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type FtList Completed

func (b Builder) FtList() (c FtList) {
	c = FtList{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "FT._LIST")
	return c
}

func (c FtList) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type FtProfile Completed

func (b Builder) FtProfile() (c FtProfile) {
	c = FtProfile{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "FT.PROFILE")
	return c
}

func (c FtProfile) Index(index string) FtProfileIndex {
	c.cs.s = append(c.cs.s, index)
	return (FtProfileIndex)(c)
}

type FtProfileIndex Completed

func (c FtProfileIndex) Search() FtProfileQuerytypeSearch {
	c.cs.s = append(c.cs.s, "SEARCH")
	return (FtProfileQuerytypeSearch)(c)
}

func (c FtProfileIndex) Aggregate() FtProfileQuerytypeAggregate {
	c.cs.s = append(c.cs.s, "AGGREGATE")
	return (FtProfileQuerytypeAggregate)(c)
}

type FtProfileLimited Completed

func (c FtProfileLimited) Query(query string) FtProfileQuery {
	c.cs.s = append(c.cs.s, "QUERY", query)
	return (FtProfileQuery)(c)
}

type FtProfileQuery Completed

func (c FtProfileQuery) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type FtProfileQuerytypeAggregate Completed

func (c FtProfileQuerytypeAggregate) Limited() FtProfileLimited {
	c.cs.s = append(c.cs.s, "LIMITED")
	return (FtProfileLimited)(c)
}

func (c FtProfileQuerytypeAggregate) Query(query string) FtProfileQuery {
	c.cs.s = append(c.cs.s, "QUERY", query)
	return (FtProfileQuery)(c)
}

type FtProfileQuerytypeSearch Completed

func (c FtProfileQuerytypeSearch) Limited() FtProfileLimited {
	c.cs.s = append(c.cs.s, "LIMITED")
	return (FtProfileLimited)(c)
}

func (c FtProfileQuerytypeSearch) Query(query string) FtProfileQuery {
	c.cs.s = append(c.cs.s, "QUERY", query)
	return (FtProfileQuery)(c)
}

type FtSearch Completed

func (b Builder) FtSearch() (c FtSearch) {
	c = FtSearch{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "FT.SEARCH")
	return c
}

func (c FtSearch) Index(index string) FtSearchIndex {
	c.cs.s = append(c.cs.s, index)
	return (FtSearchIndex)(c)
}

type FtSearchDialect Completed

func (c FtSearchDialect) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type FtSearchExpander Completed

func (c FtSearchExpander) Scorer(scorer string) FtSearchScorer {
	c.cs.s = append(c.cs.s, "SCORER", scorer)
	return (FtSearchScorer)(c)
}

func (c FtSearchExpander) Explainscore() FtSearchExplainscore {
	c.cs.s = append(c.cs.s, "EXPLAINSCORE")
	return (FtSearchExplainscore)(c)
}

func (c FtSearchExpander) Payload(payload string) FtSearchPayload {
	c.cs.s = append(c.cs.s, "PAYLOAD", payload)
	return (FtSearchPayload)(c)
}

func (c FtSearchExpander) Sortby(sortby string) FtSearchSortbySortby {
	c.cs.s = append(c.cs.s, "SORTBY", sortby)
	return (FtSearchSortbySortby)(c)
}

func (c FtSearchExpander) Limit() FtSearchLimitLimit {
	c.cs.s = append(c.cs.s, "LIMIT")
	return (FtSearchLimitLimit)(c)
}

func (c FtSearchExpander) Params() FtSearchParamsParams {
	c.cs.s = append(c.cs.s, "PARAMS")
	return (FtSearchParamsParams)(c)
}

func (c FtSearchExpander) Dialect(dialect int64) FtSearchDialect {
	c.cs.s = append(c.cs.s, "DIALECT", strconv.FormatInt(dialect, 10))
	return (FtSearchDialect)(c)
}

func (c FtSearchExpander) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type FtSearchExplainscore Completed

func (c FtSearchExplainscore) Payload(payload string) FtSearchPayload {
	c.cs.s = append(c.cs.s, "PAYLOAD", payload)
	return (FtSearchPayload)(c)
}

func (c FtSearchExplainscore) Sortby(sortby string) FtSearchSortbySortby {
	c.cs.s = append(c.cs.s, "SORTBY", sortby)
	return (FtSearchSortbySortby)(c)
}

func (c FtSearchExplainscore) Limit() FtSearchLimitLimit {
	c.cs.s = append(c.cs.s, "LIMIT")
	return (FtSearchLimitLimit)(c)
}

func (c FtSearchExplainscore) Params() FtSearchParamsParams {
	c.cs.s = append(c.cs.s, "PARAMS")
	return (FtSearchParamsParams)(c)
}

func (c FtSearchExplainscore) Dialect(dialect int64) FtSearchDialect {
	c.cs.s = append(c.cs.s, "DIALECT", strconv.FormatInt(dialect, 10))
	return (FtSearchDialect)(c)
}

func (c FtSearchExplainscore) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type FtSearchFilterFilter Completed

func (c FtSearchFilterFilter) Min(min float64) FtSearchFilterMin {
	c.cs.s = append(c.cs.s, strconv.FormatFloat(min, 'f', -1, 64))
	return (FtSearchFilterMin)(c)
}

type FtSearchFilterMax Completed

func (c FtSearchFilterMax) Filter(numericField string) FtSearchFilterFilter {
	c.cs.s = append(c.cs.s, "FILTER", numericField)
	return (FtSearchFilterFilter)(c)
}

func (c FtSearchFilterMax) Geofilter(geoField string) FtSearchGeoFilterGeofilter {
	c.cs.s = append(c.cs.s, "GEOFILTER", geoField)
	return (FtSearchGeoFilterGeofilter)(c)
}

func (c FtSearchFilterMax) Inkeys(count string) FtSearchInKeysInkeys {
	c.cs.s = append(c.cs.s, "INKEYS", count)
	return (FtSearchInKeysInkeys)(c)
}

func (c FtSearchFilterMax) Infields(count string) FtSearchInFieldsInfields {
	c.cs.s = append(c.cs.s, "INFIELDS", count)
	return (FtSearchInFieldsInfields)(c)
}

func (c FtSearchFilterMax) Return(count string) FtSearchReturnReturn {
	c.cs.s = append(c.cs.s, "RETURN", count)
	return (FtSearchReturnReturn)(c)
}

func (c FtSearchFilterMax) Summarize() FtSearchSummarizeSummarize {
	c.cs.s = append(c.cs.s, "SUMMARIZE")
	return (FtSearchSummarizeSummarize)(c)
}

func (c FtSearchFilterMax) Highlight() FtSearchHighlightHighlight {
	c.cs.s = append(c.cs.s, "HIGHLIGHT")
	return (FtSearchHighlightHighlight)(c)
}

func (c FtSearchFilterMax) Slop(slop int64) FtSearchSlop {
	c.cs.s = append(c.cs.s, "SLOP", strconv.FormatInt(slop, 10))
	return (FtSearchSlop)(c)
}

func (c FtSearchFilterMax) Timeout(timeout int64) FtSearchTimeout {
	c.cs.s = append(c.cs.s, "TIMEOUT", strconv.FormatInt(timeout, 10))
	return (FtSearchTimeout)(c)
}

func (c FtSearchFilterMax) Inorder() FtSearchTagsInorder {
	c.cs.s = append(c.cs.s, "INORDER")
	return (FtSearchTagsInorder)(c)
}

func (c FtSearchFilterMax) Language(language string) FtSearchLanguage {
	c.cs.s = append(c.cs.s, "LANGUAGE", language)
	return (FtSearchLanguage)(c)
}

func (c FtSearchFilterMax) Expander(expander string) FtSearchExpander {
	c.cs.s = append(c.cs.s, "EXPANDER", expander)
	return (FtSearchExpander)(c)
}

func (c FtSearchFilterMax) Scorer(scorer string) FtSearchScorer {
	c.cs.s = append(c.cs.s, "SCORER", scorer)
	return (FtSearchScorer)(c)
}

func (c FtSearchFilterMax) Explainscore() FtSearchExplainscore {
	c.cs.s = append(c.cs.s, "EXPLAINSCORE")
	return (FtSearchExplainscore)(c)
}

func (c FtSearchFilterMax) Payload(payload string) FtSearchPayload {
	c.cs.s = append(c.cs.s, "PAYLOAD", payload)
	return (FtSearchPayload)(c)
}

func (c FtSearchFilterMax) Sortby(sortby string) FtSearchSortbySortby {
	c.cs.s = append(c.cs.s, "SORTBY", sortby)
	return (FtSearchSortbySortby)(c)
}

func (c FtSearchFilterMax) Limit() FtSearchLimitLimit {
	c.cs.s = append(c.cs.s, "LIMIT")
	return (FtSearchLimitLimit)(c)
}

func (c FtSearchFilterMax) Params() FtSearchParamsParams {
	c.cs.s = append(c.cs.s, "PARAMS")
	return (FtSearchParamsParams)(c)
}

func (c FtSearchFilterMax) Dialect(dialect int64) FtSearchDialect {
	c.cs.s = append(c.cs.s, "DIALECT", strconv.FormatInt(dialect, 10))
	return (FtSearchDialect)(c)
}

func (c FtSearchFilterMax) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type FtSearchFilterMin Completed

func (c FtSearchFilterMin) Max(max float64) FtSearchFilterMax {
	c.cs.s = append(c.cs.s, strconv.FormatFloat(max, 'f', -1, 64))
	return (FtSearchFilterMax)(c)
}

type FtSearchGeoFilterGeofilter Completed

func (c FtSearchGeoFilterGeofilter) Lon(lon float64) FtSearchGeoFilterLon {
	c.cs.s = append(c.cs.s, strconv.FormatFloat(lon, 'f', -1, 64))
	return (FtSearchGeoFilterLon)(c)
}

type FtSearchGeoFilterLat Completed

func (c FtSearchGeoFilterLat) Radius(radius float64) FtSearchGeoFilterRadius {
	c.cs.s = append(c.cs.s, strconv.FormatFloat(radius, 'f', -1, 64))
	return (FtSearchGeoFilterRadius)(c)
}

type FtSearchGeoFilterLon Completed

func (c FtSearchGeoFilterLon) Lat(lat float64) FtSearchGeoFilterLat {
	c.cs.s = append(c.cs.s, strconv.FormatFloat(lat, 'f', -1, 64))
	return (FtSearchGeoFilterLat)(c)
}

type FtSearchGeoFilterRadius Completed

func (c FtSearchGeoFilterRadius) M() FtSearchGeoFilterRadiusTypeM {
	c.cs.s = append(c.cs.s, "m")
	return (FtSearchGeoFilterRadiusTypeM)(c)
}

func (c FtSearchGeoFilterRadius) Km() FtSearchGeoFilterRadiusTypeKm {
	c.cs.s = append(c.cs.s, "km")
	return (FtSearchGeoFilterRadiusTypeKm)(c)
}

func (c FtSearchGeoFilterRadius) Mi() FtSearchGeoFilterRadiusTypeMi {
	c.cs.s = append(c.cs.s, "mi")
	return (FtSearchGeoFilterRadiusTypeMi)(c)
}

func (c FtSearchGeoFilterRadius) Ft() FtSearchGeoFilterRadiusTypeFt {
	c.cs.s = append(c.cs.s, "ft")
	return (FtSearchGeoFilterRadiusTypeFt)(c)
}

type FtSearchGeoFilterRadiusTypeFt Completed

func (c FtSearchGeoFilterRadiusTypeFt) Geofilter(geoField string) FtSearchGeoFilterGeofilter {
	c.cs.s = append(c.cs.s, "GEOFILTER", geoField)
	return (FtSearchGeoFilterGeofilter)(c)
}

func (c FtSearchGeoFilterRadiusTypeFt) Inkeys(count string) FtSearchInKeysInkeys {
	c.cs.s = append(c.cs.s, "INKEYS", count)
	return (FtSearchInKeysInkeys)(c)
}

func (c FtSearchGeoFilterRadiusTypeFt) Infields(count string) FtSearchInFieldsInfields {
	c.cs.s = append(c.cs.s, "INFIELDS", count)
	return (FtSearchInFieldsInfields)(c)
}

func (c FtSearchGeoFilterRadiusTypeFt) Return(count string) FtSearchReturnReturn {
	c.cs.s = append(c.cs.s, "RETURN", count)
	return (FtSearchReturnReturn)(c)
}

func (c FtSearchGeoFilterRadiusTypeFt) Summarize() FtSearchSummarizeSummarize {
	c.cs.s = append(c.cs.s, "SUMMARIZE")
	return (FtSearchSummarizeSummarize)(c)
}

func (c FtSearchGeoFilterRadiusTypeFt) Highlight() FtSearchHighlightHighlight {
	c.cs.s = append(c.cs.s, "HIGHLIGHT")
	return (FtSearchHighlightHighlight)(c)
}

func (c FtSearchGeoFilterRadiusTypeFt) Slop(slop int64) FtSearchSlop {
	c.cs.s = append(c.cs.s, "SLOP", strconv.FormatInt(slop, 10))
	return (FtSearchSlop)(c)
}

func (c FtSearchGeoFilterRadiusTypeFt) Timeout(timeout int64) FtSearchTimeout {
	c.cs.s = append(c.cs.s, "TIMEOUT", strconv.FormatInt(timeout, 10))
	return (FtSearchTimeout)(c)
}

func (c FtSearchGeoFilterRadiusTypeFt) Inorder() FtSearchTagsInorder {
	c.cs.s = append(c.cs.s, "INORDER")
	return (FtSearchTagsInorder)(c)
}

func (c FtSearchGeoFilterRadiusTypeFt) Language(language string) FtSearchLanguage {
	c.cs.s = append(c.cs.s, "LANGUAGE", language)
	return (FtSearchLanguage)(c)
}

func (c FtSearchGeoFilterRadiusTypeFt) Expander(expander string) FtSearchExpander {
	c.cs.s = append(c.cs.s, "EXPANDER", expander)
	return (FtSearchExpander)(c)
}

func (c FtSearchGeoFilterRadiusTypeFt) Scorer(scorer string) FtSearchScorer {
	c.cs.s = append(c.cs.s, "SCORER", scorer)
	return (FtSearchScorer)(c)
}

func (c FtSearchGeoFilterRadiusTypeFt) Explainscore() FtSearchExplainscore {
	c.cs.s = append(c.cs.s, "EXPLAINSCORE")
	return (FtSearchExplainscore)(c)
}

func (c FtSearchGeoFilterRadiusTypeFt) Payload(payload string) FtSearchPayload {
	c.cs.s = append(c.cs.s, "PAYLOAD", payload)
	return (FtSearchPayload)(c)
}

func (c FtSearchGeoFilterRadiusTypeFt) Sortby(sortby string) FtSearchSortbySortby {
	c.cs.s = append(c.cs.s, "SORTBY", sortby)
	return (FtSearchSortbySortby)(c)
}

func (c FtSearchGeoFilterRadiusTypeFt) Limit() FtSearchLimitLimit {
	c.cs.s = append(c.cs.s, "LIMIT")
	return (FtSearchLimitLimit)(c)
}

func (c FtSearchGeoFilterRadiusTypeFt) Params() FtSearchParamsParams {
	c.cs.s = append(c.cs.s, "PARAMS")
	return (FtSearchParamsParams)(c)
}

func (c FtSearchGeoFilterRadiusTypeFt) Dialect(dialect int64) FtSearchDialect {
	c.cs.s = append(c.cs.s, "DIALECT", strconv.FormatInt(dialect, 10))
	return (FtSearchDialect)(c)
}

func (c FtSearchGeoFilterRadiusTypeFt) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type FtSearchGeoFilterRadiusTypeKm Completed

func (c FtSearchGeoFilterRadiusTypeKm) Geofilter(geoField string) FtSearchGeoFilterGeofilter {
	c.cs.s = append(c.cs.s, "GEOFILTER", geoField)
	return (FtSearchGeoFilterGeofilter)(c)
}

func (c FtSearchGeoFilterRadiusTypeKm) Inkeys(count string) FtSearchInKeysInkeys {
	c.cs.s = append(c.cs.s, "INKEYS", count)
	return (FtSearchInKeysInkeys)(c)
}

func (c FtSearchGeoFilterRadiusTypeKm) Infields(count string) FtSearchInFieldsInfields {
	c.cs.s = append(c.cs.s, "INFIELDS", count)
	return (FtSearchInFieldsInfields)(c)
}

func (c FtSearchGeoFilterRadiusTypeKm) Return(count string) FtSearchReturnReturn {
	c.cs.s = append(c.cs.s, "RETURN", count)
	return (FtSearchReturnReturn)(c)
}

func (c FtSearchGeoFilterRadiusTypeKm) Summarize() FtSearchSummarizeSummarize {
	c.cs.s = append(c.cs.s, "SUMMARIZE")
	return (FtSearchSummarizeSummarize)(c)
}

func (c FtSearchGeoFilterRadiusTypeKm) Highlight() FtSearchHighlightHighlight {
	c.cs.s = append(c.cs.s, "HIGHLIGHT")
	return (FtSearchHighlightHighlight)(c)
}

func (c FtSearchGeoFilterRadiusTypeKm) Slop(slop int64) FtSearchSlop {
	c.cs.s = append(c.cs.s, "SLOP", strconv.FormatInt(slop, 10))
	return (FtSearchSlop)(c)
}

func (c FtSearchGeoFilterRadiusTypeKm) Timeout(timeout int64) FtSearchTimeout {
	c.cs.s = append(c.cs.s, "TIMEOUT", strconv.FormatInt(timeout, 10))
	return (FtSearchTimeout)(c)
}

func (c FtSearchGeoFilterRadiusTypeKm) Inorder() FtSearchTagsInorder {
	c.cs.s = append(c.cs.s, "INORDER")
	return (FtSearchTagsInorder)(c)
}

func (c FtSearchGeoFilterRadiusTypeKm) Language(language string) FtSearchLanguage {
	c.cs.s = append(c.cs.s, "LANGUAGE", language)
	return (FtSearchLanguage)(c)
}

func (c FtSearchGeoFilterRadiusTypeKm) Expander(expander string) FtSearchExpander {
	c.cs.s = append(c.cs.s, "EXPANDER", expander)
	return (FtSearchExpander)(c)
}

func (c FtSearchGeoFilterRadiusTypeKm) Scorer(scorer string) FtSearchScorer {
	c.cs.s = append(c.cs.s, "SCORER", scorer)
	return (FtSearchScorer)(c)
}

func (c FtSearchGeoFilterRadiusTypeKm) Explainscore() FtSearchExplainscore {
	c.cs.s = append(c.cs.s, "EXPLAINSCORE")
	return (FtSearchExplainscore)(c)
}

func (c FtSearchGeoFilterRadiusTypeKm) Payload(payload string) FtSearchPayload {
	c.cs.s = append(c.cs.s, "PAYLOAD", payload)
	return (FtSearchPayload)(c)
}

func (c FtSearchGeoFilterRadiusTypeKm) Sortby(sortby string) FtSearchSortbySortby {
	c.cs.s = append(c.cs.s, "SORTBY", sortby)
	return (FtSearchSortbySortby)(c)
}

func (c FtSearchGeoFilterRadiusTypeKm) Limit() FtSearchLimitLimit {
	c.cs.s = append(c.cs.s, "LIMIT")
	return (FtSearchLimitLimit)(c)
}

func (c FtSearchGeoFilterRadiusTypeKm) Params() FtSearchParamsParams {
	c.cs.s = append(c.cs.s, "PARAMS")
	return (FtSearchParamsParams)(c)
}

func (c FtSearchGeoFilterRadiusTypeKm) Dialect(dialect int64) FtSearchDialect {
	c.cs.s = append(c.cs.s, "DIALECT", strconv.FormatInt(dialect, 10))
	return (FtSearchDialect)(c)
}

func (c FtSearchGeoFilterRadiusTypeKm) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type FtSearchGeoFilterRadiusTypeM Completed

func (c FtSearchGeoFilterRadiusTypeM) Geofilter(geoField string) FtSearchGeoFilterGeofilter {
	c.cs.s = append(c.cs.s, "GEOFILTER", geoField)
	return (FtSearchGeoFilterGeofilter)(c)
}

func (c FtSearchGeoFilterRadiusTypeM) Inkeys(count string) FtSearchInKeysInkeys {
	c.cs.s = append(c.cs.s, "INKEYS", count)
	return (FtSearchInKeysInkeys)(c)
}

func (c FtSearchGeoFilterRadiusTypeM) Infields(count string) FtSearchInFieldsInfields {
	c.cs.s = append(c.cs.s, "INFIELDS", count)
	return (FtSearchInFieldsInfields)(c)
}

func (c FtSearchGeoFilterRadiusTypeM) Return(count string) FtSearchReturnReturn {
	c.cs.s = append(c.cs.s, "RETURN", count)
	return (FtSearchReturnReturn)(c)
}

func (c FtSearchGeoFilterRadiusTypeM) Summarize() FtSearchSummarizeSummarize {
	c.cs.s = append(c.cs.s, "SUMMARIZE")
	return (FtSearchSummarizeSummarize)(c)
}

func (c FtSearchGeoFilterRadiusTypeM) Highlight() FtSearchHighlightHighlight {
	c.cs.s = append(c.cs.s, "HIGHLIGHT")
	return (FtSearchHighlightHighlight)(c)
}

func (c FtSearchGeoFilterRadiusTypeM) Slop(slop int64) FtSearchSlop {
	c.cs.s = append(c.cs.s, "SLOP", strconv.FormatInt(slop, 10))
	return (FtSearchSlop)(c)
}

func (c FtSearchGeoFilterRadiusTypeM) Timeout(timeout int64) FtSearchTimeout {
	c.cs.s = append(c.cs.s, "TIMEOUT", strconv.FormatInt(timeout, 10))
	return (FtSearchTimeout)(c)
}

func (c FtSearchGeoFilterRadiusTypeM) Inorder() FtSearchTagsInorder {
	c.cs.s = append(c.cs.s, "INORDER")
	return (FtSearchTagsInorder)(c)
}

func (c FtSearchGeoFilterRadiusTypeM) Language(language string) FtSearchLanguage {
	c.cs.s = append(c.cs.s, "LANGUAGE", language)
	return (FtSearchLanguage)(c)
}

func (c FtSearchGeoFilterRadiusTypeM) Expander(expander string) FtSearchExpander {
	c.cs.s = append(c.cs.s, "EXPANDER", expander)
	return (FtSearchExpander)(c)
}

func (c FtSearchGeoFilterRadiusTypeM) Scorer(scorer string) FtSearchScorer {
	c.cs.s = append(c.cs.s, "SCORER", scorer)
	return (FtSearchScorer)(c)
}

func (c FtSearchGeoFilterRadiusTypeM) Explainscore() FtSearchExplainscore {
	c.cs.s = append(c.cs.s, "EXPLAINSCORE")
	return (FtSearchExplainscore)(c)
}

func (c FtSearchGeoFilterRadiusTypeM) Payload(payload string) FtSearchPayload {
	c.cs.s = append(c.cs.s, "PAYLOAD", payload)
	return (FtSearchPayload)(c)
}

func (c FtSearchGeoFilterRadiusTypeM) Sortby(sortby string) FtSearchSortbySortby {
	c.cs.s = append(c.cs.s, "SORTBY", sortby)
	return (FtSearchSortbySortby)(c)
}

func (c FtSearchGeoFilterRadiusTypeM) Limit() FtSearchLimitLimit {
	c.cs.s = append(c.cs.s, "LIMIT")
	return (FtSearchLimitLimit)(c)
}

func (c FtSearchGeoFilterRadiusTypeM) Params() FtSearchParamsParams {
	c.cs.s = append(c.cs.s, "PARAMS")
	return (FtSearchParamsParams)(c)
}

func (c FtSearchGeoFilterRadiusTypeM) Dialect(dialect int64) FtSearchDialect {
	c.cs.s = append(c.cs.s, "DIALECT", strconv.FormatInt(dialect, 10))
	return (FtSearchDialect)(c)
}

func (c FtSearchGeoFilterRadiusTypeM) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type FtSearchGeoFilterRadiusTypeMi Completed

func (c FtSearchGeoFilterRadiusTypeMi) Geofilter(geoField string) FtSearchGeoFilterGeofilter {
	c.cs.s = append(c.cs.s, "GEOFILTER", geoField)
	return (FtSearchGeoFilterGeofilter)(c)
}

func (c FtSearchGeoFilterRadiusTypeMi) Inkeys(count string) FtSearchInKeysInkeys {
	c.cs.s = append(c.cs.s, "INKEYS", count)
	return (FtSearchInKeysInkeys)(c)
}

func (c FtSearchGeoFilterRadiusTypeMi) Infields(count string) FtSearchInFieldsInfields {
	c.cs.s = append(c.cs.s, "INFIELDS", count)
	return (FtSearchInFieldsInfields)(c)
}

func (c FtSearchGeoFilterRadiusTypeMi) Return(count string) FtSearchReturnReturn {
	c.cs.s = append(c.cs.s, "RETURN", count)
	return (FtSearchReturnReturn)(c)
}

func (c FtSearchGeoFilterRadiusTypeMi) Summarize() FtSearchSummarizeSummarize {
	c.cs.s = append(c.cs.s, "SUMMARIZE")
	return (FtSearchSummarizeSummarize)(c)
}

func (c FtSearchGeoFilterRadiusTypeMi) Highlight() FtSearchHighlightHighlight {
	c.cs.s = append(c.cs.s, "HIGHLIGHT")
	return (FtSearchHighlightHighlight)(c)
}

func (c FtSearchGeoFilterRadiusTypeMi) Slop(slop int64) FtSearchSlop {
	c.cs.s = append(c.cs.s, "SLOP", strconv.FormatInt(slop, 10))
	return (FtSearchSlop)(c)
}

func (c FtSearchGeoFilterRadiusTypeMi) Timeout(timeout int64) FtSearchTimeout {
	c.cs.s = append(c.cs.s, "TIMEOUT", strconv.FormatInt(timeout, 10))
	return (FtSearchTimeout)(c)
}

func (c FtSearchGeoFilterRadiusTypeMi) Inorder() FtSearchTagsInorder {
	c.cs.s = append(c.cs.s, "INORDER")
	return (FtSearchTagsInorder)(c)
}

func (c FtSearchGeoFilterRadiusTypeMi) Language(language string) FtSearchLanguage {
	c.cs.s = append(c.cs.s, "LANGUAGE", language)
	return (FtSearchLanguage)(c)
}

func (c FtSearchGeoFilterRadiusTypeMi) Expander(expander string) FtSearchExpander {
	c.cs.s = append(c.cs.s, "EXPANDER", expander)
	return (FtSearchExpander)(c)
}

func (c FtSearchGeoFilterRadiusTypeMi) Scorer(scorer string) FtSearchScorer {
	c.cs.s = append(c.cs.s, "SCORER", scorer)
	return (FtSearchScorer)(c)
}

func (c FtSearchGeoFilterRadiusTypeMi) Explainscore() FtSearchExplainscore {
	c.cs.s = append(c.cs.s, "EXPLAINSCORE")
	return (FtSearchExplainscore)(c)
}

func (c FtSearchGeoFilterRadiusTypeMi) Payload(payload string) FtSearchPayload {
	c.cs.s = append(c.cs.s, "PAYLOAD", payload)
	return (FtSearchPayload)(c)
}

func (c FtSearchGeoFilterRadiusTypeMi) Sortby(sortby string) FtSearchSortbySortby {
	c.cs.s = append(c.cs.s, "SORTBY", sortby)
	return (FtSearchSortbySortby)(c)
}

func (c FtSearchGeoFilterRadiusTypeMi) Limit() FtSearchLimitLimit {
	c.cs.s = append(c.cs.s, "LIMIT")
	return (FtSearchLimitLimit)(c)
}

func (c FtSearchGeoFilterRadiusTypeMi) Params() FtSearchParamsParams {
	c.cs.s = append(c.cs.s, "PARAMS")
	return (FtSearchParamsParams)(c)
}

func (c FtSearchGeoFilterRadiusTypeMi) Dialect(dialect int64) FtSearchDialect {
	c.cs.s = append(c.cs.s, "DIALECT", strconv.FormatInt(dialect, 10))
	return (FtSearchDialect)(c)
}

func (c FtSearchGeoFilterRadiusTypeMi) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type FtSearchHighlightFieldsField Completed

func (c FtSearchHighlightFieldsField) Field(field ...string) FtSearchHighlightFieldsField {
	c.cs.s = append(c.cs.s, field...)
	return c
}

func (c FtSearchHighlightFieldsField) Tags() FtSearchHighlightTagsTags {
	c.cs.s = append(c.cs.s, "TAGS")
	return (FtSearchHighlightTagsTags)(c)
}

func (c FtSearchHighlightFieldsField) Slop(slop int64) FtSearchSlop {
	c.cs.s = append(c.cs.s, "SLOP", strconv.FormatInt(slop, 10))
	return (FtSearchSlop)(c)
}

func (c FtSearchHighlightFieldsField) Timeout(timeout int64) FtSearchTimeout {
	c.cs.s = append(c.cs.s, "TIMEOUT", strconv.FormatInt(timeout, 10))
	return (FtSearchTimeout)(c)
}

func (c FtSearchHighlightFieldsField) Inorder() FtSearchTagsInorder {
	c.cs.s = append(c.cs.s, "INORDER")
	return (FtSearchTagsInorder)(c)
}

func (c FtSearchHighlightFieldsField) Language(language string) FtSearchLanguage {
	c.cs.s = append(c.cs.s, "LANGUAGE", language)
	return (FtSearchLanguage)(c)
}

func (c FtSearchHighlightFieldsField) Expander(expander string) FtSearchExpander {
	c.cs.s = append(c.cs.s, "EXPANDER", expander)
	return (FtSearchExpander)(c)
}

func (c FtSearchHighlightFieldsField) Scorer(scorer string) FtSearchScorer {
	c.cs.s = append(c.cs.s, "SCORER", scorer)
	return (FtSearchScorer)(c)
}

func (c FtSearchHighlightFieldsField) Explainscore() FtSearchExplainscore {
	c.cs.s = append(c.cs.s, "EXPLAINSCORE")
	return (FtSearchExplainscore)(c)
}

func (c FtSearchHighlightFieldsField) Payload(payload string) FtSearchPayload {
	c.cs.s = append(c.cs.s, "PAYLOAD", payload)
	return (FtSearchPayload)(c)
}

func (c FtSearchHighlightFieldsField) Sortby(sortby string) FtSearchSortbySortby {
	c.cs.s = append(c.cs.s, "SORTBY", sortby)
	return (FtSearchSortbySortby)(c)
}

func (c FtSearchHighlightFieldsField) Limit() FtSearchLimitLimit {
	c.cs.s = append(c.cs.s, "LIMIT")
	return (FtSearchLimitLimit)(c)
}

func (c FtSearchHighlightFieldsField) Params() FtSearchParamsParams {
	c.cs.s = append(c.cs.s, "PARAMS")
	return (FtSearchParamsParams)(c)
}

func (c FtSearchHighlightFieldsField) Dialect(dialect int64) FtSearchDialect {
	c.cs.s = append(c.cs.s, "DIALECT", strconv.FormatInt(dialect, 10))
	return (FtSearchDialect)(c)
}

func (c FtSearchHighlightFieldsField) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type FtSearchHighlightFieldsFields Completed

func (c FtSearchHighlightFieldsFields) Field(field ...string) FtSearchHighlightFieldsField {
	c.cs.s = append(c.cs.s, field...)
	return (FtSearchHighlightFieldsField)(c)
}

type FtSearchHighlightHighlight Completed

func (c FtSearchHighlightHighlight) Fields(count string) FtSearchHighlightFieldsFields {
	c.cs.s = append(c.cs.s, "FIELDS", count)
	return (FtSearchHighlightFieldsFields)(c)
}

func (c FtSearchHighlightHighlight) Tags() FtSearchHighlightTagsTags {
	c.cs.s = append(c.cs.s, "TAGS")
	return (FtSearchHighlightTagsTags)(c)
}

func (c FtSearchHighlightHighlight) Slop(slop int64) FtSearchSlop {
	c.cs.s = append(c.cs.s, "SLOP", strconv.FormatInt(slop, 10))
	return (FtSearchSlop)(c)
}

func (c FtSearchHighlightHighlight) Timeout(timeout int64) FtSearchTimeout {
	c.cs.s = append(c.cs.s, "TIMEOUT", strconv.FormatInt(timeout, 10))
	return (FtSearchTimeout)(c)
}

func (c FtSearchHighlightHighlight) Inorder() FtSearchTagsInorder {
	c.cs.s = append(c.cs.s, "INORDER")
	return (FtSearchTagsInorder)(c)
}

func (c FtSearchHighlightHighlight) Language(language string) FtSearchLanguage {
	c.cs.s = append(c.cs.s, "LANGUAGE", language)
	return (FtSearchLanguage)(c)
}

func (c FtSearchHighlightHighlight) Expander(expander string) FtSearchExpander {
	c.cs.s = append(c.cs.s, "EXPANDER", expander)
	return (FtSearchExpander)(c)
}

func (c FtSearchHighlightHighlight) Scorer(scorer string) FtSearchScorer {
	c.cs.s = append(c.cs.s, "SCORER", scorer)
	return (FtSearchScorer)(c)
}

func (c FtSearchHighlightHighlight) Explainscore() FtSearchExplainscore {
	c.cs.s = append(c.cs.s, "EXPLAINSCORE")
	return (FtSearchExplainscore)(c)
}

func (c FtSearchHighlightHighlight) Payload(payload string) FtSearchPayload {
	c.cs.s = append(c.cs.s, "PAYLOAD", payload)
	return (FtSearchPayload)(c)
}

func (c FtSearchHighlightHighlight) Sortby(sortby string) FtSearchSortbySortby {
	c.cs.s = append(c.cs.s, "SORTBY", sortby)
	return (FtSearchSortbySortby)(c)
}

func (c FtSearchHighlightHighlight) Limit() FtSearchLimitLimit {
	c.cs.s = append(c.cs.s, "LIMIT")
	return (FtSearchLimitLimit)(c)
}

func (c FtSearchHighlightHighlight) Params() FtSearchParamsParams {
	c.cs.s = append(c.cs.s, "PARAMS")
	return (FtSearchParamsParams)(c)
}

func (c FtSearchHighlightHighlight) Dialect(dialect int64) FtSearchDialect {
	c.cs.s = append(c.cs.s, "DIALECT", strconv.FormatInt(dialect, 10))
	return (FtSearchDialect)(c)
}

func (c FtSearchHighlightHighlight) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type FtSearchHighlightTagsOpenClose Completed

func (c FtSearchHighlightTagsOpenClose) Slop(slop int64) FtSearchSlop {
	c.cs.s = append(c.cs.s, "SLOP", strconv.FormatInt(slop, 10))
	return (FtSearchSlop)(c)
}

func (c FtSearchHighlightTagsOpenClose) Timeout(timeout int64) FtSearchTimeout {
	c.cs.s = append(c.cs.s, "TIMEOUT", strconv.FormatInt(timeout, 10))
	return (FtSearchTimeout)(c)
}

func (c FtSearchHighlightTagsOpenClose) Inorder() FtSearchTagsInorder {
	c.cs.s = append(c.cs.s, "INORDER")
	return (FtSearchTagsInorder)(c)
}

func (c FtSearchHighlightTagsOpenClose) Language(language string) FtSearchLanguage {
	c.cs.s = append(c.cs.s, "LANGUAGE", language)
	return (FtSearchLanguage)(c)
}

func (c FtSearchHighlightTagsOpenClose) Expander(expander string) FtSearchExpander {
	c.cs.s = append(c.cs.s, "EXPANDER", expander)
	return (FtSearchExpander)(c)
}

func (c FtSearchHighlightTagsOpenClose) Scorer(scorer string) FtSearchScorer {
	c.cs.s = append(c.cs.s, "SCORER", scorer)
	return (FtSearchScorer)(c)
}

func (c FtSearchHighlightTagsOpenClose) Explainscore() FtSearchExplainscore {
	c.cs.s = append(c.cs.s, "EXPLAINSCORE")
	return (FtSearchExplainscore)(c)
}

func (c FtSearchHighlightTagsOpenClose) Payload(payload string) FtSearchPayload {
	c.cs.s = append(c.cs.s, "PAYLOAD", payload)
	return (FtSearchPayload)(c)
}

func (c FtSearchHighlightTagsOpenClose) Sortby(sortby string) FtSearchSortbySortby {
	c.cs.s = append(c.cs.s, "SORTBY", sortby)
	return (FtSearchSortbySortby)(c)
}

func (c FtSearchHighlightTagsOpenClose) Limit() FtSearchLimitLimit {
	c.cs.s = append(c.cs.s, "LIMIT")
	return (FtSearchLimitLimit)(c)
}

func (c FtSearchHighlightTagsOpenClose) Params() FtSearchParamsParams {
	c.cs.s = append(c.cs.s, "PARAMS")
	return (FtSearchParamsParams)(c)
}

func (c FtSearchHighlightTagsOpenClose) Dialect(dialect int64) FtSearchDialect {
	c.cs.s = append(c.cs.s, "DIALECT", strconv.FormatInt(dialect, 10))
	return (FtSearchDialect)(c)
}

func (c FtSearchHighlightTagsOpenClose) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type FtSearchHighlightTagsTags Completed

func (c FtSearchHighlightTagsTags) OpenClose(open string, close string) FtSearchHighlightTagsOpenClose {
	c.cs.s = append(c.cs.s, open, close)
	return (FtSearchHighlightTagsOpenClose)(c)
}

type FtSearchInFieldsField Completed

func (c FtSearchInFieldsField) Field(field ...string) FtSearchInFieldsField {
	c.cs.s = append(c.cs.s, field...)
	return c
}

func (c FtSearchInFieldsField) Return(count string) FtSearchReturnReturn {
	c.cs.s = append(c.cs.s, "RETURN", count)
	return (FtSearchReturnReturn)(c)
}

func (c FtSearchInFieldsField) Summarize() FtSearchSummarizeSummarize {
	c.cs.s = append(c.cs.s, "SUMMARIZE")
	return (FtSearchSummarizeSummarize)(c)
}

func (c FtSearchInFieldsField) Highlight() FtSearchHighlightHighlight {
	c.cs.s = append(c.cs.s, "HIGHLIGHT")
	return (FtSearchHighlightHighlight)(c)
}

func (c FtSearchInFieldsField) Slop(slop int64) FtSearchSlop {
	c.cs.s = append(c.cs.s, "SLOP", strconv.FormatInt(slop, 10))
	return (FtSearchSlop)(c)
}

func (c FtSearchInFieldsField) Timeout(timeout int64) FtSearchTimeout {
	c.cs.s = append(c.cs.s, "TIMEOUT", strconv.FormatInt(timeout, 10))
	return (FtSearchTimeout)(c)
}

func (c FtSearchInFieldsField) Inorder() FtSearchTagsInorder {
	c.cs.s = append(c.cs.s, "INORDER")
	return (FtSearchTagsInorder)(c)
}

func (c FtSearchInFieldsField) Language(language string) FtSearchLanguage {
	c.cs.s = append(c.cs.s, "LANGUAGE", language)
	return (FtSearchLanguage)(c)
}

func (c FtSearchInFieldsField) Expander(expander string) FtSearchExpander {
	c.cs.s = append(c.cs.s, "EXPANDER", expander)
	return (FtSearchExpander)(c)
}

func (c FtSearchInFieldsField) Scorer(scorer string) FtSearchScorer {
	c.cs.s = append(c.cs.s, "SCORER", scorer)
	return (FtSearchScorer)(c)
}

func (c FtSearchInFieldsField) Explainscore() FtSearchExplainscore {
	c.cs.s = append(c.cs.s, "EXPLAINSCORE")
	return (FtSearchExplainscore)(c)
}

func (c FtSearchInFieldsField) Payload(payload string) FtSearchPayload {
	c.cs.s = append(c.cs.s, "PAYLOAD", payload)
	return (FtSearchPayload)(c)
}

func (c FtSearchInFieldsField) Sortby(sortby string) FtSearchSortbySortby {
	c.cs.s = append(c.cs.s, "SORTBY", sortby)
	return (FtSearchSortbySortby)(c)
}

func (c FtSearchInFieldsField) Limit() FtSearchLimitLimit {
	c.cs.s = append(c.cs.s, "LIMIT")
	return (FtSearchLimitLimit)(c)
}

func (c FtSearchInFieldsField) Params() FtSearchParamsParams {
	c.cs.s = append(c.cs.s, "PARAMS")
	return (FtSearchParamsParams)(c)
}

func (c FtSearchInFieldsField) Dialect(dialect int64) FtSearchDialect {
	c.cs.s = append(c.cs.s, "DIALECT", strconv.FormatInt(dialect, 10))
	return (FtSearchDialect)(c)
}

func (c FtSearchInFieldsField) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type FtSearchInFieldsInfields Completed

func (c FtSearchInFieldsInfields) Field(field ...string) FtSearchInFieldsField {
	c.cs.s = append(c.cs.s, field...)
	return (FtSearchInFieldsField)(c)
}

type FtSearchInKeysInkeys Completed

func (c FtSearchInKeysInkeys) Key(key ...string) FtSearchInKeysKey {
	c.cs.s = append(c.cs.s, key...)
	return (FtSearchInKeysKey)(c)
}

type FtSearchInKeysKey Completed

func (c FtSearchInKeysKey) Key(key ...string) FtSearchInKeysKey {
	c.cs.s = append(c.cs.s, key...)
	return c
}

func (c FtSearchInKeysKey) Infields(count string) FtSearchInFieldsInfields {
	c.cs.s = append(c.cs.s, "INFIELDS", count)
	return (FtSearchInFieldsInfields)(c)
}

func (c FtSearchInKeysKey) Return(count string) FtSearchReturnReturn {
	c.cs.s = append(c.cs.s, "RETURN", count)
	return (FtSearchReturnReturn)(c)
}

func (c FtSearchInKeysKey) Summarize() FtSearchSummarizeSummarize {
	c.cs.s = append(c.cs.s, "SUMMARIZE")
	return (FtSearchSummarizeSummarize)(c)
}

func (c FtSearchInKeysKey) Highlight() FtSearchHighlightHighlight {
	c.cs.s = append(c.cs.s, "HIGHLIGHT")
	return (FtSearchHighlightHighlight)(c)
}

func (c FtSearchInKeysKey) Slop(slop int64) FtSearchSlop {
	c.cs.s = append(c.cs.s, "SLOP", strconv.FormatInt(slop, 10))
	return (FtSearchSlop)(c)
}

func (c FtSearchInKeysKey) Timeout(timeout int64) FtSearchTimeout {
	c.cs.s = append(c.cs.s, "TIMEOUT", strconv.FormatInt(timeout, 10))
	return (FtSearchTimeout)(c)
}

func (c FtSearchInKeysKey) Inorder() FtSearchTagsInorder {
	c.cs.s = append(c.cs.s, "INORDER")
	return (FtSearchTagsInorder)(c)
}

func (c FtSearchInKeysKey) Language(language string) FtSearchLanguage {
	c.cs.s = append(c.cs.s, "LANGUAGE", language)
	return (FtSearchLanguage)(c)
}

func (c FtSearchInKeysKey) Expander(expander string) FtSearchExpander {
	c.cs.s = append(c.cs.s, "EXPANDER", expander)
	return (FtSearchExpander)(c)
}

func (c FtSearchInKeysKey) Scorer(scorer string) FtSearchScorer {
	c.cs.s = append(c.cs.s, "SCORER", scorer)
	return (FtSearchScorer)(c)
}

func (c FtSearchInKeysKey) Explainscore() FtSearchExplainscore {
	c.cs.s = append(c.cs.s, "EXPLAINSCORE")
	return (FtSearchExplainscore)(c)
}

func (c FtSearchInKeysKey) Payload(payload string) FtSearchPayload {
	c.cs.s = append(c.cs.s, "PAYLOAD", payload)
	return (FtSearchPayload)(c)
}

func (c FtSearchInKeysKey) Sortby(sortby string) FtSearchSortbySortby {
	c.cs.s = append(c.cs.s, "SORTBY", sortby)
	return (FtSearchSortbySortby)(c)
}

func (c FtSearchInKeysKey) Limit() FtSearchLimitLimit {
	c.cs.s = append(c.cs.s, "LIMIT")
	return (FtSearchLimitLimit)(c)
}

func (c FtSearchInKeysKey) Params() FtSearchParamsParams {
	c.cs.s = append(c.cs.s, "PARAMS")
	return (FtSearchParamsParams)(c)
}

func (c FtSearchInKeysKey) Dialect(dialect int64) FtSearchDialect {
	c.cs.s = append(c.cs.s, "DIALECT", strconv.FormatInt(dialect, 10))
	return (FtSearchDialect)(c)
}

func (c FtSearchInKeysKey) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type FtSearchIndex Completed

func (c FtSearchIndex) Query(query string) FtSearchQuery {
	c.cs.s = append(c.cs.s, query)
	return (FtSearchQuery)(c)
}

type FtSearchLanguage Completed

func (c FtSearchLanguage) Expander(expander string) FtSearchExpander {
	c.cs.s = append(c.cs.s, "EXPANDER", expander)
	return (FtSearchExpander)(c)
}

func (c FtSearchLanguage) Scorer(scorer string) FtSearchScorer {
	c.cs.s = append(c.cs.s, "SCORER", scorer)
	return (FtSearchScorer)(c)
}

func (c FtSearchLanguage) Explainscore() FtSearchExplainscore {
	c.cs.s = append(c.cs.s, "EXPLAINSCORE")
	return (FtSearchExplainscore)(c)
}

func (c FtSearchLanguage) Payload(payload string) FtSearchPayload {
	c.cs.s = append(c.cs.s, "PAYLOAD", payload)
	return (FtSearchPayload)(c)
}

func (c FtSearchLanguage) Sortby(sortby string) FtSearchSortbySortby {
	c.cs.s = append(c.cs.s, "SORTBY", sortby)
	return (FtSearchSortbySortby)(c)
}

func (c FtSearchLanguage) Limit() FtSearchLimitLimit {
	c.cs.s = append(c.cs.s, "LIMIT")
	return (FtSearchLimitLimit)(c)
}

func (c FtSearchLanguage) Params() FtSearchParamsParams {
	c.cs.s = append(c.cs.s, "PARAMS")
	return (FtSearchParamsParams)(c)
}

func (c FtSearchLanguage) Dialect(dialect int64) FtSearchDialect {
	c.cs.s = append(c.cs.s, "DIALECT", strconv.FormatInt(dialect, 10))
	return (FtSearchDialect)(c)
}

func (c FtSearchLanguage) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type FtSearchLimitLimit Completed

func (c FtSearchLimitLimit) OffsetNum(offset int64, num int64) FtSearchLimitOffsetNum {
	c.cs.s = append(c.cs.s, strconv.FormatInt(offset, 10), strconv.FormatInt(num, 10))
	return (FtSearchLimitOffsetNum)(c)
}

type FtSearchLimitOffsetNum Completed

func (c FtSearchLimitOffsetNum) Params() FtSearchParamsParams {
	c.cs.s = append(c.cs.s, "PARAMS")
	return (FtSearchParamsParams)(c)
}

func (c FtSearchLimitOffsetNum) Dialect(dialect int64) FtSearchDialect {
	c.cs.s = append(c.cs.s, "DIALECT", strconv.FormatInt(dialect, 10))
	return (FtSearchDialect)(c)
}

func (c FtSearchLimitOffsetNum) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type FtSearchNocontent Completed

func (c FtSearchNocontent) Verbatim() FtSearchVerbatim {
	c.cs.s = append(c.cs.s, "VERBATIM")
	return (FtSearchVerbatim)(c)
}

func (c FtSearchNocontent) Nostopwords() FtSearchNostopwords {
	c.cs.s = append(c.cs.s, "NOSTOPWORDS")
	return (FtSearchNostopwords)(c)
}

func (c FtSearchNocontent) Withscores() FtSearchWithscores {
	c.cs.s = append(c.cs.s, "WITHSCORES")
	return (FtSearchWithscores)(c)
}

func (c FtSearchNocontent) Withpayloads() FtSearchWithpayloads {
	c.cs.s = append(c.cs.s, "WITHPAYLOADS")
	return (FtSearchWithpayloads)(c)
}

func (c FtSearchNocontent) Withsortkeys() FtSearchWithsortkeys {
	c.cs.s = append(c.cs.s, "WITHSORTKEYS")
	return (FtSearchWithsortkeys)(c)
}

func (c FtSearchNocontent) Filter(numericField string) FtSearchFilterFilter {
	c.cs.s = append(c.cs.s, "FILTER", numericField)
	return (FtSearchFilterFilter)(c)
}

func (c FtSearchNocontent) Geofilter(geoField string) FtSearchGeoFilterGeofilter {
	c.cs.s = append(c.cs.s, "GEOFILTER", geoField)
	return (FtSearchGeoFilterGeofilter)(c)
}

func (c FtSearchNocontent) Inkeys(count string) FtSearchInKeysInkeys {
	c.cs.s = append(c.cs.s, "INKEYS", count)
	return (FtSearchInKeysInkeys)(c)
}

func (c FtSearchNocontent) Infields(count string) FtSearchInFieldsInfields {
	c.cs.s = append(c.cs.s, "INFIELDS", count)
	return (FtSearchInFieldsInfields)(c)
}

func (c FtSearchNocontent) Return(count string) FtSearchReturnReturn {
	c.cs.s = append(c.cs.s, "RETURN", count)
	return (FtSearchReturnReturn)(c)
}

func (c FtSearchNocontent) Summarize() FtSearchSummarizeSummarize {
	c.cs.s = append(c.cs.s, "SUMMARIZE")
	return (FtSearchSummarizeSummarize)(c)
}

func (c FtSearchNocontent) Highlight() FtSearchHighlightHighlight {
	c.cs.s = append(c.cs.s, "HIGHLIGHT")
	return (FtSearchHighlightHighlight)(c)
}

func (c FtSearchNocontent) Slop(slop int64) FtSearchSlop {
	c.cs.s = append(c.cs.s, "SLOP", strconv.FormatInt(slop, 10))
	return (FtSearchSlop)(c)
}

func (c FtSearchNocontent) Timeout(timeout int64) FtSearchTimeout {
	c.cs.s = append(c.cs.s, "TIMEOUT", strconv.FormatInt(timeout, 10))
	return (FtSearchTimeout)(c)
}

func (c FtSearchNocontent) Inorder() FtSearchTagsInorder {
	c.cs.s = append(c.cs.s, "INORDER")
	return (FtSearchTagsInorder)(c)
}

func (c FtSearchNocontent) Language(language string) FtSearchLanguage {
	c.cs.s = append(c.cs.s, "LANGUAGE", language)
	return (FtSearchLanguage)(c)
}

func (c FtSearchNocontent) Expander(expander string) FtSearchExpander {
	c.cs.s = append(c.cs.s, "EXPANDER", expander)
	return (FtSearchExpander)(c)
}

func (c FtSearchNocontent) Scorer(scorer string) FtSearchScorer {
	c.cs.s = append(c.cs.s, "SCORER", scorer)
	return (FtSearchScorer)(c)
}

func (c FtSearchNocontent) Explainscore() FtSearchExplainscore {
	c.cs.s = append(c.cs.s, "EXPLAINSCORE")
	return (FtSearchExplainscore)(c)
}

func (c FtSearchNocontent) Payload(payload string) FtSearchPayload {
	c.cs.s = append(c.cs.s, "PAYLOAD", payload)
	return (FtSearchPayload)(c)
}

func (c FtSearchNocontent) Sortby(sortby string) FtSearchSortbySortby {
	c.cs.s = append(c.cs.s, "SORTBY", sortby)
	return (FtSearchSortbySortby)(c)
}

func (c FtSearchNocontent) Limit() FtSearchLimitLimit {
	c.cs.s = append(c.cs.s, "LIMIT")
	return (FtSearchLimitLimit)(c)
}

func (c FtSearchNocontent) Params() FtSearchParamsParams {
	c.cs.s = append(c.cs.s, "PARAMS")
	return (FtSearchParamsParams)(c)
}

func (c FtSearchNocontent) Dialect(dialect int64) FtSearchDialect {
	c.cs.s = append(c.cs.s, "DIALECT", strconv.FormatInt(dialect, 10))
	return (FtSearchDialect)(c)
}

func (c FtSearchNocontent) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type FtSearchNostopwords Completed

func (c FtSearchNostopwords) Withscores() FtSearchWithscores {
	c.cs.s = append(c.cs.s, "WITHSCORES")
	return (FtSearchWithscores)(c)
}

func (c FtSearchNostopwords) Withpayloads() FtSearchWithpayloads {
	c.cs.s = append(c.cs.s, "WITHPAYLOADS")
	return (FtSearchWithpayloads)(c)
}

func (c FtSearchNostopwords) Withsortkeys() FtSearchWithsortkeys {
	c.cs.s = append(c.cs.s, "WITHSORTKEYS")
	return (FtSearchWithsortkeys)(c)
}

func (c FtSearchNostopwords) Filter(numericField string) FtSearchFilterFilter {
	c.cs.s = append(c.cs.s, "FILTER", numericField)
	return (FtSearchFilterFilter)(c)
}

func (c FtSearchNostopwords) Geofilter(geoField string) FtSearchGeoFilterGeofilter {
	c.cs.s = append(c.cs.s, "GEOFILTER", geoField)
	return (FtSearchGeoFilterGeofilter)(c)
}

func (c FtSearchNostopwords) Inkeys(count string) FtSearchInKeysInkeys {
	c.cs.s = append(c.cs.s, "INKEYS", count)
	return (FtSearchInKeysInkeys)(c)
}

func (c FtSearchNostopwords) Infields(count string) FtSearchInFieldsInfields {
	c.cs.s = append(c.cs.s, "INFIELDS", count)
	return (FtSearchInFieldsInfields)(c)
}

func (c FtSearchNostopwords) Return(count string) FtSearchReturnReturn {
	c.cs.s = append(c.cs.s, "RETURN", count)
	return (FtSearchReturnReturn)(c)
}

func (c FtSearchNostopwords) Summarize() FtSearchSummarizeSummarize {
	c.cs.s = append(c.cs.s, "SUMMARIZE")
	return (FtSearchSummarizeSummarize)(c)
}

func (c FtSearchNostopwords) Highlight() FtSearchHighlightHighlight {
	c.cs.s = append(c.cs.s, "HIGHLIGHT")
	return (FtSearchHighlightHighlight)(c)
}

func (c FtSearchNostopwords) Slop(slop int64) FtSearchSlop {
	c.cs.s = append(c.cs.s, "SLOP", strconv.FormatInt(slop, 10))
	return (FtSearchSlop)(c)
}

func (c FtSearchNostopwords) Timeout(timeout int64) FtSearchTimeout {
	c.cs.s = append(c.cs.s, "TIMEOUT", strconv.FormatInt(timeout, 10))
	return (FtSearchTimeout)(c)
}

func (c FtSearchNostopwords) Inorder() FtSearchTagsInorder {
	c.cs.s = append(c.cs.s, "INORDER")
	return (FtSearchTagsInorder)(c)
}

func (c FtSearchNostopwords) Language(language string) FtSearchLanguage {
	c.cs.s = append(c.cs.s, "LANGUAGE", language)
	return (FtSearchLanguage)(c)
}

func (c FtSearchNostopwords) Expander(expander string) FtSearchExpander {
	c.cs.s = append(c.cs.s, "EXPANDER", expander)
	return (FtSearchExpander)(c)
}

func (c FtSearchNostopwords) Scorer(scorer string) FtSearchScorer {
	c.cs.s = append(c.cs.s, "SCORER", scorer)
	return (FtSearchScorer)(c)
}

func (c FtSearchNostopwords) Explainscore() FtSearchExplainscore {
	c.cs.s = append(c.cs.s, "EXPLAINSCORE")
	return (FtSearchExplainscore)(c)
}

func (c FtSearchNostopwords) Payload(payload string) FtSearchPayload {
	c.cs.s = append(c.cs.s, "PAYLOAD", payload)
	return (FtSearchPayload)(c)
}

func (c FtSearchNostopwords) Sortby(sortby string) FtSearchSortbySortby {
	c.cs.s = append(c.cs.s, "SORTBY", sortby)
	return (FtSearchSortbySortby)(c)
}

func (c FtSearchNostopwords) Limit() FtSearchLimitLimit {
	c.cs.s = append(c.cs.s, "LIMIT")
	return (FtSearchLimitLimit)(c)
}

func (c FtSearchNostopwords) Params() FtSearchParamsParams {
	c.cs.s = append(c.cs.s, "PARAMS")
	return (FtSearchParamsParams)(c)
}

func (c FtSearchNostopwords) Dialect(dialect int64) FtSearchDialect {
	c.cs.s = append(c.cs.s, "DIALECT", strconv.FormatInt(dialect, 10))
	return (FtSearchDialect)(c)
}

func (c FtSearchNostopwords) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type FtSearchParamsNameValue Completed

func (c FtSearchParamsNameValue) NameValue(name string, value string) FtSearchParamsNameValue {
	c.cs.s = append(c.cs.s, name, value)
	return c
}

func (c FtSearchParamsNameValue) Dialect(dialect int64) FtSearchDialect {
	c.cs.s = append(c.cs.s, "DIALECT", strconv.FormatInt(dialect, 10))
	return (FtSearchDialect)(c)
}

func (c FtSearchParamsNameValue) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type FtSearchParamsNargs Completed

func (c FtSearchParamsNargs) NameValue() FtSearchParamsNameValue {
	return (FtSearchParamsNameValue)(c)
}

type FtSearchParamsParams Completed

func (c FtSearchParamsParams) Nargs(nargs int64) FtSearchParamsNargs {
	c.cs.s = append(c.cs.s, strconv.FormatInt(nargs, 10))
	return (FtSearchParamsNargs)(c)
}

type FtSearchPayload Completed

func (c FtSearchPayload) Sortby(sortby string) FtSearchSortbySortby {
	c.cs.s = append(c.cs.s, "SORTBY", sortby)
	return (FtSearchSortbySortby)(c)
}

func (c FtSearchPayload) Limit() FtSearchLimitLimit {
	c.cs.s = append(c.cs.s, "LIMIT")
	return (FtSearchLimitLimit)(c)
}

func (c FtSearchPayload) Params() FtSearchParamsParams {
	c.cs.s = append(c.cs.s, "PARAMS")
	return (FtSearchParamsParams)(c)
}

func (c FtSearchPayload) Dialect(dialect int64) FtSearchDialect {
	c.cs.s = append(c.cs.s, "DIALECT", strconv.FormatInt(dialect, 10))
	return (FtSearchDialect)(c)
}

func (c FtSearchPayload) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type FtSearchQuery Completed

func (c FtSearchQuery) Nocontent() FtSearchNocontent {
	c.cs.s = append(c.cs.s, "NOCONTENT")
	return (FtSearchNocontent)(c)
}

func (c FtSearchQuery) Verbatim() FtSearchVerbatim {
	c.cs.s = append(c.cs.s, "VERBATIM")
	return (FtSearchVerbatim)(c)
}

func (c FtSearchQuery) Nostopwords() FtSearchNostopwords {
	c.cs.s = append(c.cs.s, "NOSTOPWORDS")
	return (FtSearchNostopwords)(c)
}

func (c FtSearchQuery) Withscores() FtSearchWithscores {
	c.cs.s = append(c.cs.s, "WITHSCORES")
	return (FtSearchWithscores)(c)
}

func (c FtSearchQuery) Withpayloads() FtSearchWithpayloads {
	c.cs.s = append(c.cs.s, "WITHPAYLOADS")
	return (FtSearchWithpayloads)(c)
}

func (c FtSearchQuery) Withsortkeys() FtSearchWithsortkeys {
	c.cs.s = append(c.cs.s, "WITHSORTKEYS")
	return (FtSearchWithsortkeys)(c)
}

func (c FtSearchQuery) Filter(numericField string) FtSearchFilterFilter {
	c.cs.s = append(c.cs.s, "FILTER", numericField)
	return (FtSearchFilterFilter)(c)
}

func (c FtSearchQuery) Geofilter(geoField string) FtSearchGeoFilterGeofilter {
	c.cs.s = append(c.cs.s, "GEOFILTER", geoField)
	return (FtSearchGeoFilterGeofilter)(c)
}

func (c FtSearchQuery) Inkeys(count string) FtSearchInKeysInkeys {
	c.cs.s = append(c.cs.s, "INKEYS", count)
	return (FtSearchInKeysInkeys)(c)
}

func (c FtSearchQuery) Infields(count string) FtSearchInFieldsInfields {
	c.cs.s = append(c.cs.s, "INFIELDS", count)
	return (FtSearchInFieldsInfields)(c)
}

func (c FtSearchQuery) Return(count string) FtSearchReturnReturn {
	c.cs.s = append(c.cs.s, "RETURN", count)
	return (FtSearchReturnReturn)(c)
}

func (c FtSearchQuery) Summarize() FtSearchSummarizeSummarize {
	c.cs.s = append(c.cs.s, "SUMMARIZE")
	return (FtSearchSummarizeSummarize)(c)
}

func (c FtSearchQuery) Highlight() FtSearchHighlightHighlight {
	c.cs.s = append(c.cs.s, "HIGHLIGHT")
	return (FtSearchHighlightHighlight)(c)
}

func (c FtSearchQuery) Slop(slop int64) FtSearchSlop {
	c.cs.s = append(c.cs.s, "SLOP", strconv.FormatInt(slop, 10))
	return (FtSearchSlop)(c)
}

func (c FtSearchQuery) Timeout(timeout int64) FtSearchTimeout {
	c.cs.s = append(c.cs.s, "TIMEOUT", strconv.FormatInt(timeout, 10))
	return (FtSearchTimeout)(c)
}

func (c FtSearchQuery) Inorder() FtSearchTagsInorder {
	c.cs.s = append(c.cs.s, "INORDER")
	return (FtSearchTagsInorder)(c)
}

func (c FtSearchQuery) Language(language string) FtSearchLanguage {
	c.cs.s = append(c.cs.s, "LANGUAGE", language)
	return (FtSearchLanguage)(c)
}

func (c FtSearchQuery) Expander(expander string) FtSearchExpander {
	c.cs.s = append(c.cs.s, "EXPANDER", expander)
	return (FtSearchExpander)(c)
}

func (c FtSearchQuery) Scorer(scorer string) FtSearchScorer {
	c.cs.s = append(c.cs.s, "SCORER", scorer)
	return (FtSearchScorer)(c)
}

func (c FtSearchQuery) Explainscore() FtSearchExplainscore {
	c.cs.s = append(c.cs.s, "EXPLAINSCORE")
	return (FtSearchExplainscore)(c)
}

func (c FtSearchQuery) Payload(payload string) FtSearchPayload {
	c.cs.s = append(c.cs.s, "PAYLOAD", payload)
	return (FtSearchPayload)(c)
}

func (c FtSearchQuery) Sortby(sortby string) FtSearchSortbySortby {
	c.cs.s = append(c.cs.s, "SORTBY", sortby)
	return (FtSearchSortbySortby)(c)
}

func (c FtSearchQuery) Limit() FtSearchLimitLimit {
	c.cs.s = append(c.cs.s, "LIMIT")
	return (FtSearchLimitLimit)(c)
}

func (c FtSearchQuery) Params() FtSearchParamsParams {
	c.cs.s = append(c.cs.s, "PARAMS")
	return (FtSearchParamsParams)(c)
}

func (c FtSearchQuery) Dialect(dialect int64) FtSearchDialect {
	c.cs.s = append(c.cs.s, "DIALECT", strconv.FormatInt(dialect, 10))
	return (FtSearchDialect)(c)
}

func (c FtSearchQuery) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type FtSearchReturnIdentifiersAs Completed

func (c FtSearchReturnIdentifiersAs) Identifier(identifier string) FtSearchReturnIdentifiersIdentifier {
	c.cs.s = append(c.cs.s, identifier)
	return (FtSearchReturnIdentifiersIdentifier)(c)
}

func (c FtSearchReturnIdentifiersAs) Summarize() FtSearchSummarizeSummarize {
	c.cs.s = append(c.cs.s, "SUMMARIZE")
	return (FtSearchSummarizeSummarize)(c)
}

func (c FtSearchReturnIdentifiersAs) Highlight() FtSearchHighlightHighlight {
	c.cs.s = append(c.cs.s, "HIGHLIGHT")
	return (FtSearchHighlightHighlight)(c)
}

func (c FtSearchReturnIdentifiersAs) Slop(slop int64) FtSearchSlop {
	c.cs.s = append(c.cs.s, "SLOP", strconv.FormatInt(slop, 10))
	return (FtSearchSlop)(c)
}

func (c FtSearchReturnIdentifiersAs) Timeout(timeout int64) FtSearchTimeout {
	c.cs.s = append(c.cs.s, "TIMEOUT", strconv.FormatInt(timeout, 10))
	return (FtSearchTimeout)(c)
}

func (c FtSearchReturnIdentifiersAs) Inorder() FtSearchTagsInorder {
	c.cs.s = append(c.cs.s, "INORDER")
	return (FtSearchTagsInorder)(c)
}

func (c FtSearchReturnIdentifiersAs) Language(language string) FtSearchLanguage {
	c.cs.s = append(c.cs.s, "LANGUAGE", language)
	return (FtSearchLanguage)(c)
}

func (c FtSearchReturnIdentifiersAs) Expander(expander string) FtSearchExpander {
	c.cs.s = append(c.cs.s, "EXPANDER", expander)
	return (FtSearchExpander)(c)
}

func (c FtSearchReturnIdentifiersAs) Scorer(scorer string) FtSearchScorer {
	c.cs.s = append(c.cs.s, "SCORER", scorer)
	return (FtSearchScorer)(c)
}

func (c FtSearchReturnIdentifiersAs) Explainscore() FtSearchExplainscore {
	c.cs.s = append(c.cs.s, "EXPLAINSCORE")
	return (FtSearchExplainscore)(c)
}

func (c FtSearchReturnIdentifiersAs) Payload(payload string) FtSearchPayload {
	c.cs.s = append(c.cs.s, "PAYLOAD", payload)
	return (FtSearchPayload)(c)
}

func (c FtSearchReturnIdentifiersAs) Sortby(sortby string) FtSearchSortbySortby {
	c.cs.s = append(c.cs.s, "SORTBY", sortby)
	return (FtSearchSortbySortby)(c)
}

func (c FtSearchReturnIdentifiersAs) Limit() FtSearchLimitLimit {
	c.cs.s = append(c.cs.s, "LIMIT")
	return (FtSearchLimitLimit)(c)
}

func (c FtSearchReturnIdentifiersAs) Params() FtSearchParamsParams {
	c.cs.s = append(c.cs.s, "PARAMS")
	return (FtSearchParamsParams)(c)
}

func (c FtSearchReturnIdentifiersAs) Dialect(dialect int64) FtSearchDialect {
	c.cs.s = append(c.cs.s, "DIALECT", strconv.FormatInt(dialect, 10))
	return (FtSearchDialect)(c)
}

func (c FtSearchReturnIdentifiersAs) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type FtSearchReturnIdentifiersIdentifier Completed

func (c FtSearchReturnIdentifiersIdentifier) As(property string) FtSearchReturnIdentifiersAs {
	c.cs.s = append(c.cs.s, "AS", property)
	return (FtSearchReturnIdentifiersAs)(c)
}

func (c FtSearchReturnIdentifiersIdentifier) Identifier(identifier string) FtSearchReturnIdentifiersIdentifier {
	c.cs.s = append(c.cs.s, identifier)
	return c
}

func (c FtSearchReturnIdentifiersIdentifier) Summarize() FtSearchSummarizeSummarize {
	c.cs.s = append(c.cs.s, "SUMMARIZE")
	return (FtSearchSummarizeSummarize)(c)
}

func (c FtSearchReturnIdentifiersIdentifier) Highlight() FtSearchHighlightHighlight {
	c.cs.s = append(c.cs.s, "HIGHLIGHT")
	return (FtSearchHighlightHighlight)(c)
}

func (c FtSearchReturnIdentifiersIdentifier) Slop(slop int64) FtSearchSlop {
	c.cs.s = append(c.cs.s, "SLOP", strconv.FormatInt(slop, 10))
	return (FtSearchSlop)(c)
}

func (c FtSearchReturnIdentifiersIdentifier) Timeout(timeout int64) FtSearchTimeout {
	c.cs.s = append(c.cs.s, "TIMEOUT", strconv.FormatInt(timeout, 10))
	return (FtSearchTimeout)(c)
}

func (c FtSearchReturnIdentifiersIdentifier) Inorder() FtSearchTagsInorder {
	c.cs.s = append(c.cs.s, "INORDER")
	return (FtSearchTagsInorder)(c)
}

func (c FtSearchReturnIdentifiersIdentifier) Language(language string) FtSearchLanguage {
	c.cs.s = append(c.cs.s, "LANGUAGE", language)
	return (FtSearchLanguage)(c)
}

func (c FtSearchReturnIdentifiersIdentifier) Expander(expander string) FtSearchExpander {
	c.cs.s = append(c.cs.s, "EXPANDER", expander)
	return (FtSearchExpander)(c)
}

func (c FtSearchReturnIdentifiersIdentifier) Scorer(scorer string) FtSearchScorer {
	c.cs.s = append(c.cs.s, "SCORER", scorer)
	return (FtSearchScorer)(c)
}

func (c FtSearchReturnIdentifiersIdentifier) Explainscore() FtSearchExplainscore {
	c.cs.s = append(c.cs.s, "EXPLAINSCORE")
	return (FtSearchExplainscore)(c)
}

func (c FtSearchReturnIdentifiersIdentifier) Payload(payload string) FtSearchPayload {
	c.cs.s = append(c.cs.s, "PAYLOAD", payload)
	return (FtSearchPayload)(c)
}

func (c FtSearchReturnIdentifiersIdentifier) Sortby(sortby string) FtSearchSortbySortby {
	c.cs.s = append(c.cs.s, "SORTBY", sortby)
	return (FtSearchSortbySortby)(c)
}

func (c FtSearchReturnIdentifiersIdentifier) Limit() FtSearchLimitLimit {
	c.cs.s = append(c.cs.s, "LIMIT")
	return (FtSearchLimitLimit)(c)
}

func (c FtSearchReturnIdentifiersIdentifier) Params() FtSearchParamsParams {
	c.cs.s = append(c.cs.s, "PARAMS")
	return (FtSearchParamsParams)(c)
}

func (c FtSearchReturnIdentifiersIdentifier) Dialect(dialect int64) FtSearchDialect {
	c.cs.s = append(c.cs.s, "DIALECT", strconv.FormatInt(dialect, 10))
	return (FtSearchDialect)(c)
}

func (c FtSearchReturnIdentifiersIdentifier) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type FtSearchReturnReturn Completed

func (c FtSearchReturnReturn) Identifier(identifier string) FtSearchReturnIdentifiersIdentifier {
	c.cs.s = append(c.cs.s, identifier)
	return (FtSearchReturnIdentifiersIdentifier)(c)
}

type FtSearchScorer Completed

func (c FtSearchScorer) Explainscore() FtSearchExplainscore {
	c.cs.s = append(c.cs.s, "EXPLAINSCORE")
	return (FtSearchExplainscore)(c)
}

func (c FtSearchScorer) Payload(payload string) FtSearchPayload {
	c.cs.s = append(c.cs.s, "PAYLOAD", payload)
	return (FtSearchPayload)(c)
}

func (c FtSearchScorer) Sortby(sortby string) FtSearchSortbySortby {
	c.cs.s = append(c.cs.s, "SORTBY", sortby)
	return (FtSearchSortbySortby)(c)
}

func (c FtSearchScorer) Limit() FtSearchLimitLimit {
	c.cs.s = append(c.cs.s, "LIMIT")
	return (FtSearchLimitLimit)(c)
}

func (c FtSearchScorer) Params() FtSearchParamsParams {
	c.cs.s = append(c.cs.s, "PARAMS")
	return (FtSearchParamsParams)(c)
}

func (c FtSearchScorer) Dialect(dialect int64) FtSearchDialect {
	c.cs.s = append(c.cs.s, "DIALECT", strconv.FormatInt(dialect, 10))
	return (FtSearchDialect)(c)
}

func (c FtSearchScorer) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type FtSearchSlop Completed

func (c FtSearchSlop) Timeout(timeout int64) FtSearchTimeout {
	c.cs.s = append(c.cs.s, "TIMEOUT", strconv.FormatInt(timeout, 10))
	return (FtSearchTimeout)(c)
}

func (c FtSearchSlop) Inorder() FtSearchTagsInorder {
	c.cs.s = append(c.cs.s, "INORDER")
	return (FtSearchTagsInorder)(c)
}

func (c FtSearchSlop) Language(language string) FtSearchLanguage {
	c.cs.s = append(c.cs.s, "LANGUAGE", language)
	return (FtSearchLanguage)(c)
}

func (c FtSearchSlop) Expander(expander string) FtSearchExpander {
	c.cs.s = append(c.cs.s, "EXPANDER", expander)
	return (FtSearchExpander)(c)
}

func (c FtSearchSlop) Scorer(scorer string) FtSearchScorer {
	c.cs.s = append(c.cs.s, "SCORER", scorer)
	return (FtSearchScorer)(c)
}

func (c FtSearchSlop) Explainscore() FtSearchExplainscore {
	c.cs.s = append(c.cs.s, "EXPLAINSCORE")
	return (FtSearchExplainscore)(c)
}

func (c FtSearchSlop) Payload(payload string) FtSearchPayload {
	c.cs.s = append(c.cs.s, "PAYLOAD", payload)
	return (FtSearchPayload)(c)
}

func (c FtSearchSlop) Sortby(sortby string) FtSearchSortbySortby {
	c.cs.s = append(c.cs.s, "SORTBY", sortby)
	return (FtSearchSortbySortby)(c)
}

func (c FtSearchSlop) Limit() FtSearchLimitLimit {
	c.cs.s = append(c.cs.s, "LIMIT")
	return (FtSearchLimitLimit)(c)
}

func (c FtSearchSlop) Params() FtSearchParamsParams {
	c.cs.s = append(c.cs.s, "PARAMS")
	return (FtSearchParamsParams)(c)
}

func (c FtSearchSlop) Dialect(dialect int64) FtSearchDialect {
	c.cs.s = append(c.cs.s, "DIALECT", strconv.FormatInt(dialect, 10))
	return (FtSearchDialect)(c)
}

func (c FtSearchSlop) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type FtSearchSortbyOrderAsc Completed

func (c FtSearchSortbyOrderAsc) Limit() FtSearchLimitLimit {
	c.cs.s = append(c.cs.s, "LIMIT")
	return (FtSearchLimitLimit)(c)
}

func (c FtSearchSortbyOrderAsc) Params() FtSearchParamsParams {
	c.cs.s = append(c.cs.s, "PARAMS")
	return (FtSearchParamsParams)(c)
}

func (c FtSearchSortbyOrderAsc) Dialect(dialect int64) FtSearchDialect {
	c.cs.s = append(c.cs.s, "DIALECT", strconv.FormatInt(dialect, 10))
	return (FtSearchDialect)(c)
}

func (c FtSearchSortbyOrderAsc) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type FtSearchSortbyOrderDesc Completed

func (c FtSearchSortbyOrderDesc) Limit() FtSearchLimitLimit {
	c.cs.s = append(c.cs.s, "LIMIT")
	return (FtSearchLimitLimit)(c)
}

func (c FtSearchSortbyOrderDesc) Params() FtSearchParamsParams {
	c.cs.s = append(c.cs.s, "PARAMS")
	return (FtSearchParamsParams)(c)
}

func (c FtSearchSortbyOrderDesc) Dialect(dialect int64) FtSearchDialect {
	c.cs.s = append(c.cs.s, "DIALECT", strconv.FormatInt(dialect, 10))
	return (FtSearchDialect)(c)
}

func (c FtSearchSortbyOrderDesc) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type FtSearchSortbySortby Completed

func (c FtSearchSortbySortby) Asc() FtSearchSortbyOrderAsc {
	c.cs.s = append(c.cs.s, "ASC")
	return (FtSearchSortbyOrderAsc)(c)
}

func (c FtSearchSortbySortby) Desc() FtSearchSortbyOrderDesc {
	c.cs.s = append(c.cs.s, "DESC")
	return (FtSearchSortbyOrderDesc)(c)
}

func (c FtSearchSortbySortby) Limit() FtSearchLimitLimit {
	c.cs.s = append(c.cs.s, "LIMIT")
	return (FtSearchLimitLimit)(c)
}

func (c FtSearchSortbySortby) Params() FtSearchParamsParams {
	c.cs.s = append(c.cs.s, "PARAMS")
	return (FtSearchParamsParams)(c)
}

func (c FtSearchSortbySortby) Dialect(dialect int64) FtSearchDialect {
	c.cs.s = append(c.cs.s, "DIALECT", strconv.FormatInt(dialect, 10))
	return (FtSearchDialect)(c)
}

func (c FtSearchSortbySortby) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type FtSearchSummarizeFieldsField Completed

func (c FtSearchSummarizeFieldsField) Field(field ...string) FtSearchSummarizeFieldsField {
	c.cs.s = append(c.cs.s, field...)
	return c
}

func (c FtSearchSummarizeFieldsField) Frags(num int64) FtSearchSummarizeFrags {
	c.cs.s = append(c.cs.s, "FRAGS", strconv.FormatInt(num, 10))
	return (FtSearchSummarizeFrags)(c)
}

func (c FtSearchSummarizeFieldsField) Len(fragsize int64) FtSearchSummarizeLen {
	c.cs.s = append(c.cs.s, "LEN", strconv.FormatInt(fragsize, 10))
	return (FtSearchSummarizeLen)(c)
}

func (c FtSearchSummarizeFieldsField) Separator(separator string) FtSearchSummarizeSeparator {
	c.cs.s = append(c.cs.s, "SEPARATOR", separator)
	return (FtSearchSummarizeSeparator)(c)
}

func (c FtSearchSummarizeFieldsField) Highlight() FtSearchHighlightHighlight {
	c.cs.s = append(c.cs.s, "HIGHLIGHT")
	return (FtSearchHighlightHighlight)(c)
}

func (c FtSearchSummarizeFieldsField) Slop(slop int64) FtSearchSlop {
	c.cs.s = append(c.cs.s, "SLOP", strconv.FormatInt(slop, 10))
	return (FtSearchSlop)(c)
}

func (c FtSearchSummarizeFieldsField) Timeout(timeout int64) FtSearchTimeout {
	c.cs.s = append(c.cs.s, "TIMEOUT", strconv.FormatInt(timeout, 10))
	return (FtSearchTimeout)(c)
}

func (c FtSearchSummarizeFieldsField) Inorder() FtSearchTagsInorder {
	c.cs.s = append(c.cs.s, "INORDER")
	return (FtSearchTagsInorder)(c)
}

func (c FtSearchSummarizeFieldsField) Language(language string) FtSearchLanguage {
	c.cs.s = append(c.cs.s, "LANGUAGE", language)
	return (FtSearchLanguage)(c)
}

func (c FtSearchSummarizeFieldsField) Expander(expander string) FtSearchExpander {
	c.cs.s = append(c.cs.s, "EXPANDER", expander)
	return (FtSearchExpander)(c)
}

func (c FtSearchSummarizeFieldsField) Scorer(scorer string) FtSearchScorer {
	c.cs.s = append(c.cs.s, "SCORER", scorer)
	return (FtSearchScorer)(c)
}

func (c FtSearchSummarizeFieldsField) Explainscore() FtSearchExplainscore {
	c.cs.s = append(c.cs.s, "EXPLAINSCORE")
	return (FtSearchExplainscore)(c)
}

func (c FtSearchSummarizeFieldsField) Payload(payload string) FtSearchPayload {
	c.cs.s = append(c.cs.s, "PAYLOAD", payload)
	return (FtSearchPayload)(c)
}

func (c FtSearchSummarizeFieldsField) Sortby(sortby string) FtSearchSortbySortby {
	c.cs.s = append(c.cs.s, "SORTBY", sortby)
	return (FtSearchSortbySortby)(c)
}

func (c FtSearchSummarizeFieldsField) Limit() FtSearchLimitLimit {
	c.cs.s = append(c.cs.s, "LIMIT")
	return (FtSearchLimitLimit)(c)
}

func (c FtSearchSummarizeFieldsField) Params() FtSearchParamsParams {
	c.cs.s = append(c.cs.s, "PARAMS")
	return (FtSearchParamsParams)(c)
}

func (c FtSearchSummarizeFieldsField) Dialect(dialect int64) FtSearchDialect {
	c.cs.s = append(c.cs.s, "DIALECT", strconv.FormatInt(dialect, 10))
	return (FtSearchDialect)(c)
}

func (c FtSearchSummarizeFieldsField) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type FtSearchSummarizeFieldsFields Completed

func (c FtSearchSummarizeFieldsFields) Field(field ...string) FtSearchSummarizeFieldsField {
	c.cs.s = append(c.cs.s, field...)
	return (FtSearchSummarizeFieldsField)(c)
}

type FtSearchSummarizeFrags Completed

func (c FtSearchSummarizeFrags) Len(fragsize int64) FtSearchSummarizeLen {
	c.cs.s = append(c.cs.s, "LEN", strconv.FormatInt(fragsize, 10))
	return (FtSearchSummarizeLen)(c)
}

func (c FtSearchSummarizeFrags) Separator(separator string) FtSearchSummarizeSeparator {
	c.cs.s = append(c.cs.s, "SEPARATOR", separator)
	return (FtSearchSummarizeSeparator)(c)
}

func (c FtSearchSummarizeFrags) Highlight() FtSearchHighlightHighlight {
	c.cs.s = append(c.cs.s, "HIGHLIGHT")
	return (FtSearchHighlightHighlight)(c)
}

func (c FtSearchSummarizeFrags) Slop(slop int64) FtSearchSlop {
	c.cs.s = append(c.cs.s, "SLOP", strconv.FormatInt(slop, 10))
	return (FtSearchSlop)(c)
}

func (c FtSearchSummarizeFrags) Timeout(timeout int64) FtSearchTimeout {
	c.cs.s = append(c.cs.s, "TIMEOUT", strconv.FormatInt(timeout, 10))
	return (FtSearchTimeout)(c)
}

func (c FtSearchSummarizeFrags) Inorder() FtSearchTagsInorder {
	c.cs.s = append(c.cs.s, "INORDER")
	return (FtSearchTagsInorder)(c)
}

func (c FtSearchSummarizeFrags) Language(language string) FtSearchLanguage {
	c.cs.s = append(c.cs.s, "LANGUAGE", language)
	return (FtSearchLanguage)(c)
}

func (c FtSearchSummarizeFrags) Expander(expander string) FtSearchExpander {
	c.cs.s = append(c.cs.s, "EXPANDER", expander)
	return (FtSearchExpander)(c)
}

func (c FtSearchSummarizeFrags) Scorer(scorer string) FtSearchScorer {
	c.cs.s = append(c.cs.s, "SCORER", scorer)
	return (FtSearchScorer)(c)
}

func (c FtSearchSummarizeFrags) Explainscore() FtSearchExplainscore {
	c.cs.s = append(c.cs.s, "EXPLAINSCORE")
	return (FtSearchExplainscore)(c)
}

func (c FtSearchSummarizeFrags) Payload(payload string) FtSearchPayload {
	c.cs.s = append(c.cs.s, "PAYLOAD", payload)
	return (FtSearchPayload)(c)
}

func (c FtSearchSummarizeFrags) Sortby(sortby string) FtSearchSortbySortby {
	c.cs.s = append(c.cs.s, "SORTBY", sortby)
	return (FtSearchSortbySortby)(c)
}

func (c FtSearchSummarizeFrags) Limit() FtSearchLimitLimit {
	c.cs.s = append(c.cs.s, "LIMIT")
	return (FtSearchLimitLimit)(c)
}

func (c FtSearchSummarizeFrags) Params() FtSearchParamsParams {
	c.cs.s = append(c.cs.s, "PARAMS")
	return (FtSearchParamsParams)(c)
}

func (c FtSearchSummarizeFrags) Dialect(dialect int64) FtSearchDialect {
	c.cs.s = append(c.cs.s, "DIALECT", strconv.FormatInt(dialect, 10))
	return (FtSearchDialect)(c)
}

func (c FtSearchSummarizeFrags) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type FtSearchSummarizeLen Completed

func (c FtSearchSummarizeLen) Separator(separator string) FtSearchSummarizeSeparator {
	c.cs.s = append(c.cs.s, "SEPARATOR", separator)
	return (FtSearchSummarizeSeparator)(c)
}

func (c FtSearchSummarizeLen) Highlight() FtSearchHighlightHighlight {
	c.cs.s = append(c.cs.s, "HIGHLIGHT")
	return (FtSearchHighlightHighlight)(c)
}

func (c FtSearchSummarizeLen) Slop(slop int64) FtSearchSlop {
	c.cs.s = append(c.cs.s, "SLOP", strconv.FormatInt(slop, 10))
	return (FtSearchSlop)(c)
}

func (c FtSearchSummarizeLen) Timeout(timeout int64) FtSearchTimeout {
	c.cs.s = append(c.cs.s, "TIMEOUT", strconv.FormatInt(timeout, 10))
	return (FtSearchTimeout)(c)
}

func (c FtSearchSummarizeLen) Inorder() FtSearchTagsInorder {
	c.cs.s = append(c.cs.s, "INORDER")
	return (FtSearchTagsInorder)(c)
}

func (c FtSearchSummarizeLen) Language(language string) FtSearchLanguage {
	c.cs.s = append(c.cs.s, "LANGUAGE", language)
	return (FtSearchLanguage)(c)
}

func (c FtSearchSummarizeLen) Expander(expander string) FtSearchExpander {
	c.cs.s = append(c.cs.s, "EXPANDER", expander)
	return (FtSearchExpander)(c)
}

func (c FtSearchSummarizeLen) Scorer(scorer string) FtSearchScorer {
	c.cs.s = append(c.cs.s, "SCORER", scorer)
	return (FtSearchScorer)(c)
}

func (c FtSearchSummarizeLen) Explainscore() FtSearchExplainscore {
	c.cs.s = append(c.cs.s, "EXPLAINSCORE")
	return (FtSearchExplainscore)(c)
}

func (c FtSearchSummarizeLen) Payload(payload string) FtSearchPayload {
	c.cs.s = append(c.cs.s, "PAYLOAD", payload)
	return (FtSearchPayload)(c)
}

func (c FtSearchSummarizeLen) Sortby(sortby string) FtSearchSortbySortby {
	c.cs.s = append(c.cs.s, "SORTBY", sortby)
	return (FtSearchSortbySortby)(c)
}

func (c FtSearchSummarizeLen) Limit() FtSearchLimitLimit {
	c.cs.s = append(c.cs.s, "LIMIT")
	return (FtSearchLimitLimit)(c)
}

func (c FtSearchSummarizeLen) Params() FtSearchParamsParams {
	c.cs.s = append(c.cs.s, "PARAMS")
	return (FtSearchParamsParams)(c)
}

func (c FtSearchSummarizeLen) Dialect(dialect int64) FtSearchDialect {
	c.cs.s = append(c.cs.s, "DIALECT", strconv.FormatInt(dialect, 10))
	return (FtSearchDialect)(c)
}

func (c FtSearchSummarizeLen) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type FtSearchSummarizeSeparator Completed

func (c FtSearchSummarizeSeparator) Highlight() FtSearchHighlightHighlight {
	c.cs.s = append(c.cs.s, "HIGHLIGHT")
	return (FtSearchHighlightHighlight)(c)
}

func (c FtSearchSummarizeSeparator) Slop(slop int64) FtSearchSlop {
	c.cs.s = append(c.cs.s, "SLOP", strconv.FormatInt(slop, 10))
	return (FtSearchSlop)(c)
}

func (c FtSearchSummarizeSeparator) Timeout(timeout int64) FtSearchTimeout {
	c.cs.s = append(c.cs.s, "TIMEOUT", strconv.FormatInt(timeout, 10))
	return (FtSearchTimeout)(c)
}

func (c FtSearchSummarizeSeparator) Inorder() FtSearchTagsInorder {
	c.cs.s = append(c.cs.s, "INORDER")
	return (FtSearchTagsInorder)(c)
}

func (c FtSearchSummarizeSeparator) Language(language string) FtSearchLanguage {
	c.cs.s = append(c.cs.s, "LANGUAGE", language)
	return (FtSearchLanguage)(c)
}

func (c FtSearchSummarizeSeparator) Expander(expander string) FtSearchExpander {
	c.cs.s = append(c.cs.s, "EXPANDER", expander)
	return (FtSearchExpander)(c)
}

func (c FtSearchSummarizeSeparator) Scorer(scorer string) FtSearchScorer {
	c.cs.s = append(c.cs.s, "SCORER", scorer)
	return (FtSearchScorer)(c)
}

func (c FtSearchSummarizeSeparator) Explainscore() FtSearchExplainscore {
	c.cs.s = append(c.cs.s, "EXPLAINSCORE")
	return (FtSearchExplainscore)(c)
}

func (c FtSearchSummarizeSeparator) Payload(payload string) FtSearchPayload {
	c.cs.s = append(c.cs.s, "PAYLOAD", payload)
	return (FtSearchPayload)(c)
}

func (c FtSearchSummarizeSeparator) Sortby(sortby string) FtSearchSortbySortby {
	c.cs.s = append(c.cs.s, "SORTBY", sortby)
	return (FtSearchSortbySortby)(c)
}

func (c FtSearchSummarizeSeparator) Limit() FtSearchLimitLimit {
	c.cs.s = append(c.cs.s, "LIMIT")
	return (FtSearchLimitLimit)(c)
}

func (c FtSearchSummarizeSeparator) Params() FtSearchParamsParams {
	c.cs.s = append(c.cs.s, "PARAMS")
	return (FtSearchParamsParams)(c)
}

func (c FtSearchSummarizeSeparator) Dialect(dialect int64) FtSearchDialect {
	c.cs.s = append(c.cs.s, "DIALECT", strconv.FormatInt(dialect, 10))
	return (FtSearchDialect)(c)
}

func (c FtSearchSummarizeSeparator) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type FtSearchSummarizeSummarize Completed

func (c FtSearchSummarizeSummarize) Fields(count string) FtSearchSummarizeFieldsFields {
	c.cs.s = append(c.cs.s, "FIELDS", count)
	return (FtSearchSummarizeFieldsFields)(c)
}

func (c FtSearchSummarizeSummarize) Frags(num int64) FtSearchSummarizeFrags {
	c.cs.s = append(c.cs.s, "FRAGS", strconv.FormatInt(num, 10))
	return (FtSearchSummarizeFrags)(c)
}

func (c FtSearchSummarizeSummarize) Len(fragsize int64) FtSearchSummarizeLen {
	c.cs.s = append(c.cs.s, "LEN", strconv.FormatInt(fragsize, 10))
	return (FtSearchSummarizeLen)(c)
}

func (c FtSearchSummarizeSummarize) Separator(separator string) FtSearchSummarizeSeparator {
	c.cs.s = append(c.cs.s, "SEPARATOR", separator)
	return (FtSearchSummarizeSeparator)(c)
}

func (c FtSearchSummarizeSummarize) Highlight() FtSearchHighlightHighlight {
	c.cs.s = append(c.cs.s, "HIGHLIGHT")
	return (FtSearchHighlightHighlight)(c)
}

func (c FtSearchSummarizeSummarize) Slop(slop int64) FtSearchSlop {
	c.cs.s = append(c.cs.s, "SLOP", strconv.FormatInt(slop, 10))
	return (FtSearchSlop)(c)
}

func (c FtSearchSummarizeSummarize) Timeout(timeout int64) FtSearchTimeout {
	c.cs.s = append(c.cs.s, "TIMEOUT", strconv.FormatInt(timeout, 10))
	return (FtSearchTimeout)(c)
}

func (c FtSearchSummarizeSummarize) Inorder() FtSearchTagsInorder {
	c.cs.s = append(c.cs.s, "INORDER")
	return (FtSearchTagsInorder)(c)
}

func (c FtSearchSummarizeSummarize) Language(language string) FtSearchLanguage {
	c.cs.s = append(c.cs.s, "LANGUAGE", language)
	return (FtSearchLanguage)(c)
}

func (c FtSearchSummarizeSummarize) Expander(expander string) FtSearchExpander {
	c.cs.s = append(c.cs.s, "EXPANDER", expander)
	return (FtSearchExpander)(c)
}

func (c FtSearchSummarizeSummarize) Scorer(scorer string) FtSearchScorer {
	c.cs.s = append(c.cs.s, "SCORER", scorer)
	return (FtSearchScorer)(c)
}

func (c FtSearchSummarizeSummarize) Explainscore() FtSearchExplainscore {
	c.cs.s = append(c.cs.s, "EXPLAINSCORE")
	return (FtSearchExplainscore)(c)
}

func (c FtSearchSummarizeSummarize) Payload(payload string) FtSearchPayload {
	c.cs.s = append(c.cs.s, "PAYLOAD", payload)
	return (FtSearchPayload)(c)
}

func (c FtSearchSummarizeSummarize) Sortby(sortby string) FtSearchSortbySortby {
	c.cs.s = append(c.cs.s, "SORTBY", sortby)
	return (FtSearchSortbySortby)(c)
}

func (c FtSearchSummarizeSummarize) Limit() FtSearchLimitLimit {
	c.cs.s = append(c.cs.s, "LIMIT")
	return (FtSearchLimitLimit)(c)
}

func (c FtSearchSummarizeSummarize) Params() FtSearchParamsParams {
	c.cs.s = append(c.cs.s, "PARAMS")
	return (FtSearchParamsParams)(c)
}

func (c FtSearchSummarizeSummarize) Dialect(dialect int64) FtSearchDialect {
	c.cs.s = append(c.cs.s, "DIALECT", strconv.FormatInt(dialect, 10))
	return (FtSearchDialect)(c)
}

func (c FtSearchSummarizeSummarize) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type FtSearchTagsInorder Completed

func (c FtSearchTagsInorder) Language(language string) FtSearchLanguage {
	c.cs.s = append(c.cs.s, "LANGUAGE", language)
	return (FtSearchLanguage)(c)
}

func (c FtSearchTagsInorder) Expander(expander string) FtSearchExpander {
	c.cs.s = append(c.cs.s, "EXPANDER", expander)
	return (FtSearchExpander)(c)
}

func (c FtSearchTagsInorder) Scorer(scorer string) FtSearchScorer {
	c.cs.s = append(c.cs.s, "SCORER", scorer)
	return (FtSearchScorer)(c)
}

func (c FtSearchTagsInorder) Explainscore() FtSearchExplainscore {
	c.cs.s = append(c.cs.s, "EXPLAINSCORE")
	return (FtSearchExplainscore)(c)
}

func (c FtSearchTagsInorder) Payload(payload string) FtSearchPayload {
	c.cs.s = append(c.cs.s, "PAYLOAD", payload)
	return (FtSearchPayload)(c)
}

func (c FtSearchTagsInorder) Sortby(sortby string) FtSearchSortbySortby {
	c.cs.s = append(c.cs.s, "SORTBY", sortby)
	return (FtSearchSortbySortby)(c)
}

func (c FtSearchTagsInorder) Limit() FtSearchLimitLimit {
	c.cs.s = append(c.cs.s, "LIMIT")
	return (FtSearchLimitLimit)(c)
}

func (c FtSearchTagsInorder) Params() FtSearchParamsParams {
	c.cs.s = append(c.cs.s, "PARAMS")
	return (FtSearchParamsParams)(c)
}

func (c FtSearchTagsInorder) Dialect(dialect int64) FtSearchDialect {
	c.cs.s = append(c.cs.s, "DIALECT", strconv.FormatInt(dialect, 10))
	return (FtSearchDialect)(c)
}

func (c FtSearchTagsInorder) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type FtSearchTimeout Completed

func (c FtSearchTimeout) Inorder() FtSearchTagsInorder {
	c.cs.s = append(c.cs.s, "INORDER")
	return (FtSearchTagsInorder)(c)
}

func (c FtSearchTimeout) Language(language string) FtSearchLanguage {
	c.cs.s = append(c.cs.s, "LANGUAGE", language)
	return (FtSearchLanguage)(c)
}

func (c FtSearchTimeout) Expander(expander string) FtSearchExpander {
	c.cs.s = append(c.cs.s, "EXPANDER", expander)
	return (FtSearchExpander)(c)
}

func (c FtSearchTimeout) Scorer(scorer string) FtSearchScorer {
	c.cs.s = append(c.cs.s, "SCORER", scorer)
	return (FtSearchScorer)(c)
}

func (c FtSearchTimeout) Explainscore() FtSearchExplainscore {
	c.cs.s = append(c.cs.s, "EXPLAINSCORE")
	return (FtSearchExplainscore)(c)
}

func (c FtSearchTimeout) Payload(payload string) FtSearchPayload {
	c.cs.s = append(c.cs.s, "PAYLOAD", payload)
	return (FtSearchPayload)(c)
}

func (c FtSearchTimeout) Sortby(sortby string) FtSearchSortbySortby {
	c.cs.s = append(c.cs.s, "SORTBY", sortby)
	return (FtSearchSortbySortby)(c)
}

func (c FtSearchTimeout) Limit() FtSearchLimitLimit {
	c.cs.s = append(c.cs.s, "LIMIT")
	return (FtSearchLimitLimit)(c)
}

func (c FtSearchTimeout) Params() FtSearchParamsParams {
	c.cs.s = append(c.cs.s, "PARAMS")
	return (FtSearchParamsParams)(c)
}

func (c FtSearchTimeout) Dialect(dialect int64) FtSearchDialect {
	c.cs.s = append(c.cs.s, "DIALECT", strconv.FormatInt(dialect, 10))
	return (FtSearchDialect)(c)
}

func (c FtSearchTimeout) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type FtSearchVerbatim Completed

func (c FtSearchVerbatim) Nostopwords() FtSearchNostopwords {
	c.cs.s = append(c.cs.s, "NOSTOPWORDS")
	return (FtSearchNostopwords)(c)
}

func (c FtSearchVerbatim) Withscores() FtSearchWithscores {
	c.cs.s = append(c.cs.s, "WITHSCORES")
	return (FtSearchWithscores)(c)
}

func (c FtSearchVerbatim) Withpayloads() FtSearchWithpayloads {
	c.cs.s = append(c.cs.s, "WITHPAYLOADS")
	return (FtSearchWithpayloads)(c)
}

func (c FtSearchVerbatim) Withsortkeys() FtSearchWithsortkeys {
	c.cs.s = append(c.cs.s, "WITHSORTKEYS")
	return (FtSearchWithsortkeys)(c)
}

func (c FtSearchVerbatim) Filter(numericField string) FtSearchFilterFilter {
	c.cs.s = append(c.cs.s, "FILTER", numericField)
	return (FtSearchFilterFilter)(c)
}

func (c FtSearchVerbatim) Geofilter(geoField string) FtSearchGeoFilterGeofilter {
	c.cs.s = append(c.cs.s, "GEOFILTER", geoField)
	return (FtSearchGeoFilterGeofilter)(c)
}

func (c FtSearchVerbatim) Inkeys(count string) FtSearchInKeysInkeys {
	c.cs.s = append(c.cs.s, "INKEYS", count)
	return (FtSearchInKeysInkeys)(c)
}

func (c FtSearchVerbatim) Infields(count string) FtSearchInFieldsInfields {
	c.cs.s = append(c.cs.s, "INFIELDS", count)
	return (FtSearchInFieldsInfields)(c)
}

func (c FtSearchVerbatim) Return(count string) FtSearchReturnReturn {
	c.cs.s = append(c.cs.s, "RETURN", count)
	return (FtSearchReturnReturn)(c)
}

func (c FtSearchVerbatim) Summarize() FtSearchSummarizeSummarize {
	c.cs.s = append(c.cs.s, "SUMMARIZE")
	return (FtSearchSummarizeSummarize)(c)
}

func (c FtSearchVerbatim) Highlight() FtSearchHighlightHighlight {
	c.cs.s = append(c.cs.s, "HIGHLIGHT")
	return (FtSearchHighlightHighlight)(c)
}

func (c FtSearchVerbatim) Slop(slop int64) FtSearchSlop {
	c.cs.s = append(c.cs.s, "SLOP", strconv.FormatInt(slop, 10))
	return (FtSearchSlop)(c)
}

func (c FtSearchVerbatim) Timeout(timeout int64) FtSearchTimeout {
	c.cs.s = append(c.cs.s, "TIMEOUT", strconv.FormatInt(timeout, 10))
	return (FtSearchTimeout)(c)
}

func (c FtSearchVerbatim) Inorder() FtSearchTagsInorder {
	c.cs.s = append(c.cs.s, "INORDER")
	return (FtSearchTagsInorder)(c)
}

func (c FtSearchVerbatim) Language(language string) FtSearchLanguage {
	c.cs.s = append(c.cs.s, "LANGUAGE", language)
	return (FtSearchLanguage)(c)
}

func (c FtSearchVerbatim) Expander(expander string) FtSearchExpander {
	c.cs.s = append(c.cs.s, "EXPANDER", expander)
	return (FtSearchExpander)(c)
}

func (c FtSearchVerbatim) Scorer(scorer string) FtSearchScorer {
	c.cs.s = append(c.cs.s, "SCORER", scorer)
	return (FtSearchScorer)(c)
}

func (c FtSearchVerbatim) Explainscore() FtSearchExplainscore {
	c.cs.s = append(c.cs.s, "EXPLAINSCORE")
	return (FtSearchExplainscore)(c)
}

func (c FtSearchVerbatim) Payload(payload string) FtSearchPayload {
	c.cs.s = append(c.cs.s, "PAYLOAD", payload)
	return (FtSearchPayload)(c)
}

func (c FtSearchVerbatim) Sortby(sortby string) FtSearchSortbySortby {
	c.cs.s = append(c.cs.s, "SORTBY", sortby)
	return (FtSearchSortbySortby)(c)
}

func (c FtSearchVerbatim) Limit() FtSearchLimitLimit {
	c.cs.s = append(c.cs.s, "LIMIT")
	return (FtSearchLimitLimit)(c)
}

func (c FtSearchVerbatim) Params() FtSearchParamsParams {
	c.cs.s = append(c.cs.s, "PARAMS")
	return (FtSearchParamsParams)(c)
}

func (c FtSearchVerbatim) Dialect(dialect int64) FtSearchDialect {
	c.cs.s = append(c.cs.s, "DIALECT", strconv.FormatInt(dialect, 10))
	return (FtSearchDialect)(c)
}

func (c FtSearchVerbatim) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type FtSearchWithpayloads Completed

func (c FtSearchWithpayloads) Withsortkeys() FtSearchWithsortkeys {
	c.cs.s = append(c.cs.s, "WITHSORTKEYS")
	return (FtSearchWithsortkeys)(c)
}

func (c FtSearchWithpayloads) Filter(numericField string) FtSearchFilterFilter {
	c.cs.s = append(c.cs.s, "FILTER", numericField)
	return (FtSearchFilterFilter)(c)
}

func (c FtSearchWithpayloads) Geofilter(geoField string) FtSearchGeoFilterGeofilter {
	c.cs.s = append(c.cs.s, "GEOFILTER", geoField)
	return (FtSearchGeoFilterGeofilter)(c)
}

func (c FtSearchWithpayloads) Inkeys(count string) FtSearchInKeysInkeys {
	c.cs.s = append(c.cs.s, "INKEYS", count)
	return (FtSearchInKeysInkeys)(c)
}

func (c FtSearchWithpayloads) Infields(count string) FtSearchInFieldsInfields {
	c.cs.s = append(c.cs.s, "INFIELDS", count)
	return (FtSearchInFieldsInfields)(c)
}

func (c FtSearchWithpayloads) Return(count string) FtSearchReturnReturn {
	c.cs.s = append(c.cs.s, "RETURN", count)
	return (FtSearchReturnReturn)(c)
}

func (c FtSearchWithpayloads) Summarize() FtSearchSummarizeSummarize {
	c.cs.s = append(c.cs.s, "SUMMARIZE")
	return (FtSearchSummarizeSummarize)(c)
}

func (c FtSearchWithpayloads) Highlight() FtSearchHighlightHighlight {
	c.cs.s = append(c.cs.s, "HIGHLIGHT")
	return (FtSearchHighlightHighlight)(c)
}

func (c FtSearchWithpayloads) Slop(slop int64) FtSearchSlop {
	c.cs.s = append(c.cs.s, "SLOP", strconv.FormatInt(slop, 10))
	return (FtSearchSlop)(c)
}

func (c FtSearchWithpayloads) Timeout(timeout int64) FtSearchTimeout {
	c.cs.s = append(c.cs.s, "TIMEOUT", strconv.FormatInt(timeout, 10))
	return (FtSearchTimeout)(c)
}

func (c FtSearchWithpayloads) Inorder() FtSearchTagsInorder {
	c.cs.s = append(c.cs.s, "INORDER")
	return (FtSearchTagsInorder)(c)
}

func (c FtSearchWithpayloads) Language(language string) FtSearchLanguage {
	c.cs.s = append(c.cs.s, "LANGUAGE", language)
	return (FtSearchLanguage)(c)
}

func (c FtSearchWithpayloads) Expander(expander string) FtSearchExpander {
	c.cs.s = append(c.cs.s, "EXPANDER", expander)
	return (FtSearchExpander)(c)
}

func (c FtSearchWithpayloads) Scorer(scorer string) FtSearchScorer {
	c.cs.s = append(c.cs.s, "SCORER", scorer)
	return (FtSearchScorer)(c)
}

func (c FtSearchWithpayloads) Explainscore() FtSearchExplainscore {
	c.cs.s = append(c.cs.s, "EXPLAINSCORE")
	return (FtSearchExplainscore)(c)
}

func (c FtSearchWithpayloads) Payload(payload string) FtSearchPayload {
	c.cs.s = append(c.cs.s, "PAYLOAD", payload)
	return (FtSearchPayload)(c)
}

func (c FtSearchWithpayloads) Sortby(sortby string) FtSearchSortbySortby {
	c.cs.s = append(c.cs.s, "SORTBY", sortby)
	return (FtSearchSortbySortby)(c)
}

func (c FtSearchWithpayloads) Limit() FtSearchLimitLimit {
	c.cs.s = append(c.cs.s, "LIMIT")
	return (FtSearchLimitLimit)(c)
}

func (c FtSearchWithpayloads) Params() FtSearchParamsParams {
	c.cs.s = append(c.cs.s, "PARAMS")
	return (FtSearchParamsParams)(c)
}

func (c FtSearchWithpayloads) Dialect(dialect int64) FtSearchDialect {
	c.cs.s = append(c.cs.s, "DIALECT", strconv.FormatInt(dialect, 10))
	return (FtSearchDialect)(c)
}

func (c FtSearchWithpayloads) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type FtSearchWithscores Completed

func (c FtSearchWithscores) Withpayloads() FtSearchWithpayloads {
	c.cs.s = append(c.cs.s, "WITHPAYLOADS")
	return (FtSearchWithpayloads)(c)
}

func (c FtSearchWithscores) Withsortkeys() FtSearchWithsortkeys {
	c.cs.s = append(c.cs.s, "WITHSORTKEYS")
	return (FtSearchWithsortkeys)(c)
}

func (c FtSearchWithscores) Filter(numericField string) FtSearchFilterFilter {
	c.cs.s = append(c.cs.s, "FILTER", numericField)
	return (FtSearchFilterFilter)(c)
}

func (c FtSearchWithscores) Geofilter(geoField string) FtSearchGeoFilterGeofilter {
	c.cs.s = append(c.cs.s, "GEOFILTER", geoField)
	return (FtSearchGeoFilterGeofilter)(c)
}

func (c FtSearchWithscores) Inkeys(count string) FtSearchInKeysInkeys {
	c.cs.s = append(c.cs.s, "INKEYS", count)
	return (FtSearchInKeysInkeys)(c)
}

func (c FtSearchWithscores) Infields(count string) FtSearchInFieldsInfields {
	c.cs.s = append(c.cs.s, "INFIELDS", count)
	return (FtSearchInFieldsInfields)(c)
}

func (c FtSearchWithscores) Return(count string) FtSearchReturnReturn {
	c.cs.s = append(c.cs.s, "RETURN", count)
	return (FtSearchReturnReturn)(c)
}

func (c FtSearchWithscores) Summarize() FtSearchSummarizeSummarize {
	c.cs.s = append(c.cs.s, "SUMMARIZE")
	return (FtSearchSummarizeSummarize)(c)
}

func (c FtSearchWithscores) Highlight() FtSearchHighlightHighlight {
	c.cs.s = append(c.cs.s, "HIGHLIGHT")
	return (FtSearchHighlightHighlight)(c)
}

func (c FtSearchWithscores) Slop(slop int64) FtSearchSlop {
	c.cs.s = append(c.cs.s, "SLOP", strconv.FormatInt(slop, 10))
	return (FtSearchSlop)(c)
}

func (c FtSearchWithscores) Timeout(timeout int64) FtSearchTimeout {
	c.cs.s = append(c.cs.s, "TIMEOUT", strconv.FormatInt(timeout, 10))
	return (FtSearchTimeout)(c)
}

func (c FtSearchWithscores) Inorder() FtSearchTagsInorder {
	c.cs.s = append(c.cs.s, "INORDER")
	return (FtSearchTagsInorder)(c)
}

func (c FtSearchWithscores) Language(language string) FtSearchLanguage {
	c.cs.s = append(c.cs.s, "LANGUAGE", language)
	return (FtSearchLanguage)(c)
}

func (c FtSearchWithscores) Expander(expander string) FtSearchExpander {
	c.cs.s = append(c.cs.s, "EXPANDER", expander)
	return (FtSearchExpander)(c)
}

func (c FtSearchWithscores) Scorer(scorer string) FtSearchScorer {
	c.cs.s = append(c.cs.s, "SCORER", scorer)
	return (FtSearchScorer)(c)
}

func (c FtSearchWithscores) Explainscore() FtSearchExplainscore {
	c.cs.s = append(c.cs.s, "EXPLAINSCORE")
	return (FtSearchExplainscore)(c)
}

func (c FtSearchWithscores) Payload(payload string) FtSearchPayload {
	c.cs.s = append(c.cs.s, "PAYLOAD", payload)
	return (FtSearchPayload)(c)
}

func (c FtSearchWithscores) Sortby(sortby string) FtSearchSortbySortby {
	c.cs.s = append(c.cs.s, "SORTBY", sortby)
	return (FtSearchSortbySortby)(c)
}

func (c FtSearchWithscores) Limit() FtSearchLimitLimit {
	c.cs.s = append(c.cs.s, "LIMIT")
	return (FtSearchLimitLimit)(c)
}

func (c FtSearchWithscores) Params() FtSearchParamsParams {
	c.cs.s = append(c.cs.s, "PARAMS")
	return (FtSearchParamsParams)(c)
}

func (c FtSearchWithscores) Dialect(dialect int64) FtSearchDialect {
	c.cs.s = append(c.cs.s, "DIALECT", strconv.FormatInt(dialect, 10))
	return (FtSearchDialect)(c)
}

func (c FtSearchWithscores) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type FtSearchWithsortkeys Completed

func (c FtSearchWithsortkeys) Filter(numericField string) FtSearchFilterFilter {
	c.cs.s = append(c.cs.s, "FILTER", numericField)
	return (FtSearchFilterFilter)(c)
}

func (c FtSearchWithsortkeys) Geofilter(geoField string) FtSearchGeoFilterGeofilter {
	c.cs.s = append(c.cs.s, "GEOFILTER", geoField)
	return (FtSearchGeoFilterGeofilter)(c)
}

func (c FtSearchWithsortkeys) Inkeys(count string) FtSearchInKeysInkeys {
	c.cs.s = append(c.cs.s, "INKEYS", count)
	return (FtSearchInKeysInkeys)(c)
}

func (c FtSearchWithsortkeys) Infields(count string) FtSearchInFieldsInfields {
	c.cs.s = append(c.cs.s, "INFIELDS", count)
	return (FtSearchInFieldsInfields)(c)
}

func (c FtSearchWithsortkeys) Return(count string) FtSearchReturnReturn {
	c.cs.s = append(c.cs.s, "RETURN", count)
	return (FtSearchReturnReturn)(c)
}

func (c FtSearchWithsortkeys) Summarize() FtSearchSummarizeSummarize {
	c.cs.s = append(c.cs.s, "SUMMARIZE")
	return (FtSearchSummarizeSummarize)(c)
}

func (c FtSearchWithsortkeys) Highlight() FtSearchHighlightHighlight {
	c.cs.s = append(c.cs.s, "HIGHLIGHT")
	return (FtSearchHighlightHighlight)(c)
}

func (c FtSearchWithsortkeys) Slop(slop int64) FtSearchSlop {
	c.cs.s = append(c.cs.s, "SLOP", strconv.FormatInt(slop, 10))
	return (FtSearchSlop)(c)
}

func (c FtSearchWithsortkeys) Timeout(timeout int64) FtSearchTimeout {
	c.cs.s = append(c.cs.s, "TIMEOUT", strconv.FormatInt(timeout, 10))
	return (FtSearchTimeout)(c)
}

func (c FtSearchWithsortkeys) Inorder() FtSearchTagsInorder {
	c.cs.s = append(c.cs.s, "INORDER")
	return (FtSearchTagsInorder)(c)
}

func (c FtSearchWithsortkeys) Language(language string) FtSearchLanguage {
	c.cs.s = append(c.cs.s, "LANGUAGE", language)
	return (FtSearchLanguage)(c)
}

func (c FtSearchWithsortkeys) Expander(expander string) FtSearchExpander {
	c.cs.s = append(c.cs.s, "EXPANDER", expander)
	return (FtSearchExpander)(c)
}

func (c FtSearchWithsortkeys) Scorer(scorer string) FtSearchScorer {
	c.cs.s = append(c.cs.s, "SCORER", scorer)
	return (FtSearchScorer)(c)
}

func (c FtSearchWithsortkeys) Explainscore() FtSearchExplainscore {
	c.cs.s = append(c.cs.s, "EXPLAINSCORE")
	return (FtSearchExplainscore)(c)
}

func (c FtSearchWithsortkeys) Payload(payload string) FtSearchPayload {
	c.cs.s = append(c.cs.s, "PAYLOAD", payload)
	return (FtSearchPayload)(c)
}

func (c FtSearchWithsortkeys) Sortby(sortby string) FtSearchSortbySortby {
	c.cs.s = append(c.cs.s, "SORTBY", sortby)
	return (FtSearchSortbySortby)(c)
}

func (c FtSearchWithsortkeys) Limit() FtSearchLimitLimit {
	c.cs.s = append(c.cs.s, "LIMIT")
	return (FtSearchLimitLimit)(c)
}

func (c FtSearchWithsortkeys) Params() FtSearchParamsParams {
	c.cs.s = append(c.cs.s, "PARAMS")
	return (FtSearchParamsParams)(c)
}

func (c FtSearchWithsortkeys) Dialect(dialect int64) FtSearchDialect {
	c.cs.s = append(c.cs.s, "DIALECT", strconv.FormatInt(dialect, 10))
	return (FtSearchDialect)(c)
}

func (c FtSearchWithsortkeys) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type FtSpellcheck Completed

func (b Builder) FtSpellcheck() (c FtSpellcheck) {
	c = FtSpellcheck{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "FT.SPELLCHECK")
	return c
}

func (c FtSpellcheck) Index(index string) FtSpellcheckIndex {
	c.cs.s = append(c.cs.s, index)
	return (FtSpellcheckIndex)(c)
}

type FtSpellcheckDialect Completed

func (c FtSpellcheckDialect) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type FtSpellcheckDistance Completed

func (c FtSpellcheckDistance) TermsInclude() FtSpellcheckTermsTermsInclude {
	c.cs.s = append(c.cs.s, "TERMS", "INCLUDE")
	return (FtSpellcheckTermsTermsInclude)(c)
}

func (c FtSpellcheckDistance) TermsExclude() FtSpellcheckTermsTermsExclude {
	c.cs.s = append(c.cs.s, "TERMS", "EXCLUDE")
	return (FtSpellcheckTermsTermsExclude)(c)
}

func (c FtSpellcheckDistance) Dialect(dialect int64) FtSpellcheckDialect {
	c.cs.s = append(c.cs.s, "DIALECT", strconv.FormatInt(dialect, 10))
	return (FtSpellcheckDialect)(c)
}

func (c FtSpellcheckDistance) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type FtSpellcheckIndex Completed

func (c FtSpellcheckIndex) Query(query string) FtSpellcheckQuery {
	c.cs.s = append(c.cs.s, query)
	return (FtSpellcheckQuery)(c)
}

type FtSpellcheckQuery Completed

func (c FtSpellcheckQuery) Distance(distance int64) FtSpellcheckDistance {
	c.cs.s = append(c.cs.s, "DISTANCE", strconv.FormatInt(distance, 10))
	return (FtSpellcheckDistance)(c)
}

func (c FtSpellcheckQuery) TermsInclude() FtSpellcheckTermsTermsInclude {
	c.cs.s = append(c.cs.s, "TERMS", "INCLUDE")
	return (FtSpellcheckTermsTermsInclude)(c)
}

func (c FtSpellcheckQuery) TermsExclude() FtSpellcheckTermsTermsExclude {
	c.cs.s = append(c.cs.s, "TERMS", "EXCLUDE")
	return (FtSpellcheckTermsTermsExclude)(c)
}

func (c FtSpellcheckQuery) Dialect(dialect int64) FtSpellcheckDialect {
	c.cs.s = append(c.cs.s, "DIALECT", strconv.FormatInt(dialect, 10))
	return (FtSpellcheckDialect)(c)
}

func (c FtSpellcheckQuery) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type FtSpellcheckTermsDictionary Completed

func (c FtSpellcheckTermsDictionary) Terms(terms ...string) FtSpellcheckTermsTerms {
	c.cs.s = append(c.cs.s, terms...)
	return (FtSpellcheckTermsTerms)(c)
}

func (c FtSpellcheckTermsDictionary) Dialect(dialect int64) FtSpellcheckDialect {
	c.cs.s = append(c.cs.s, "DIALECT", strconv.FormatInt(dialect, 10))
	return (FtSpellcheckDialect)(c)
}

func (c FtSpellcheckTermsDictionary) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type FtSpellcheckTermsTerms Completed

func (c FtSpellcheckTermsTerms) Terms(terms ...string) FtSpellcheckTermsTerms {
	c.cs.s = append(c.cs.s, terms...)
	return c
}

func (c FtSpellcheckTermsTerms) Dialect(dialect int64) FtSpellcheckDialect {
	c.cs.s = append(c.cs.s, "DIALECT", strconv.FormatInt(dialect, 10))
	return (FtSpellcheckDialect)(c)
}

func (c FtSpellcheckTermsTerms) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type FtSpellcheckTermsTermsExclude Completed

func (c FtSpellcheckTermsTermsExclude) Dictionary(dictionary string) FtSpellcheckTermsDictionary {
	c.cs.s = append(c.cs.s, dictionary)
	return (FtSpellcheckTermsDictionary)(c)
}

type FtSpellcheckTermsTermsInclude Completed

func (c FtSpellcheckTermsTermsInclude) Dictionary(dictionary string) FtSpellcheckTermsDictionary {
	c.cs.s = append(c.cs.s, dictionary)
	return (FtSpellcheckTermsDictionary)(c)
}

type FtSugadd Completed

func (b Builder) FtSugadd() (c FtSugadd) {
	c = FtSugadd{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "FT.SUGADD")
	return c
}

func (c FtSugadd) Key(key string) FtSugaddKey {
	c.cs.s = append(c.cs.s, key)
	return (FtSugaddKey)(c)
}

type FtSugaddIncrementScoreIncr Completed

func (c FtSugaddIncrementScoreIncr) Payload(payload string) FtSugaddPayload {
	c.cs.s = append(c.cs.s, "PAYLOAD", payload)
	return (FtSugaddPayload)(c)
}

func (c FtSugaddIncrementScoreIncr) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type FtSugaddKey Completed

func (c FtSugaddKey) String(string string) FtSugaddString {
	c.cs.s = append(c.cs.s, string)
	return (FtSugaddString)(c)
}

type FtSugaddPayload Completed

func (c FtSugaddPayload) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type FtSugaddScore Completed

func (c FtSugaddScore) Incr() FtSugaddIncrementScoreIncr {
	c.cs.s = append(c.cs.s, "INCR")
	return (FtSugaddIncrementScoreIncr)(c)
}

func (c FtSugaddScore) Payload(payload string) FtSugaddPayload {
	c.cs.s = append(c.cs.s, "PAYLOAD", payload)
	return (FtSugaddPayload)(c)
}

func (c FtSugaddScore) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type FtSugaddString Completed

func (c FtSugaddString) Score(score float64) FtSugaddScore {
	c.cs.s = append(c.cs.s, strconv.FormatFloat(score, 'f', -1, 64))
	return (FtSugaddScore)(c)
}

type FtSugdel Completed

func (b Builder) FtSugdel() (c FtSugdel) {
	c = FtSugdel{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "FT.SUGDEL")
	return c
}

func (c FtSugdel) Key(key string) FtSugdelKey {
	c.cs.s = append(c.cs.s, key)
	return (FtSugdelKey)(c)
}

type FtSugdelKey Completed

func (c FtSugdelKey) String(string string) FtSugdelString {
	c.cs.s = append(c.cs.s, string)
	return (FtSugdelString)(c)
}

type FtSugdelString Completed

func (c FtSugdelString) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type FtSugget Completed

func (b Builder) FtSugget() (c FtSugget) {
	c = FtSugget{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "FT.SUGGET")
	return c
}

func (c FtSugget) Key(key string) FtSuggetKey {
	c.cs.s = append(c.cs.s, key)
	return (FtSuggetKey)(c)
}

type FtSuggetFuzzy Completed

func (c FtSuggetFuzzy) Withscores() FtSuggetWithscores {
	c.cs.s = append(c.cs.s, "WITHSCORES")
	return (FtSuggetWithscores)(c)
}

func (c FtSuggetFuzzy) Withpayloads() FtSuggetWithpayloads {
	c.cs.s = append(c.cs.s, "WITHPAYLOADS")
	return (FtSuggetWithpayloads)(c)
}

func (c FtSuggetFuzzy) Max(max int64) FtSuggetMax {
	c.cs.s = append(c.cs.s, "MAX", strconv.FormatInt(max, 10))
	return (FtSuggetMax)(c)
}

func (c FtSuggetFuzzy) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type FtSuggetKey Completed

func (c FtSuggetKey) Prefix(prefix string) FtSuggetPrefix {
	c.cs.s = append(c.cs.s, prefix)
	return (FtSuggetPrefix)(c)
}

type FtSuggetMax Completed

func (c FtSuggetMax) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type FtSuggetPrefix Completed

func (c FtSuggetPrefix) Fuzzy() FtSuggetFuzzy {
	c.cs.s = append(c.cs.s, "FUZZY")
	return (FtSuggetFuzzy)(c)
}

func (c FtSuggetPrefix) Withscores() FtSuggetWithscores {
	c.cs.s = append(c.cs.s, "WITHSCORES")
	return (FtSuggetWithscores)(c)
}

func (c FtSuggetPrefix) Withpayloads() FtSuggetWithpayloads {
	c.cs.s = append(c.cs.s, "WITHPAYLOADS")
	return (FtSuggetWithpayloads)(c)
}

func (c FtSuggetPrefix) Max(max int64) FtSuggetMax {
	c.cs.s = append(c.cs.s, "MAX", strconv.FormatInt(max, 10))
	return (FtSuggetMax)(c)
}

func (c FtSuggetPrefix) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type FtSuggetWithpayloads Completed

func (c FtSuggetWithpayloads) Max(max int64) FtSuggetMax {
	c.cs.s = append(c.cs.s, "MAX", strconv.FormatInt(max, 10))
	return (FtSuggetMax)(c)
}

func (c FtSuggetWithpayloads) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type FtSuggetWithscores Completed

func (c FtSuggetWithscores) Withpayloads() FtSuggetWithpayloads {
	c.cs.s = append(c.cs.s, "WITHPAYLOADS")
	return (FtSuggetWithpayloads)(c)
}

func (c FtSuggetWithscores) Max(max int64) FtSuggetMax {
	c.cs.s = append(c.cs.s, "MAX", strconv.FormatInt(max, 10))
	return (FtSuggetMax)(c)
}

func (c FtSuggetWithscores) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type FtSuglen Completed

func (b Builder) FtSuglen() (c FtSuglen) {
	c = FtSuglen{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "FT.SUGLEN")
	return c
}

func (c FtSuglen) Key(key string) FtSuglenKey {
	c.cs.s = append(c.cs.s, key)
	return (FtSuglenKey)(c)
}

type FtSuglenKey Completed

func (c FtSuglenKey) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type FtSyndump Completed

func (b Builder) FtSyndump() (c FtSyndump) {
	c = FtSyndump{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "FT.SYNDUMP")
	return c
}

func (c FtSyndump) Index(index string) FtSyndumpIndex {
	c.cs.s = append(c.cs.s, index)
	return (FtSyndumpIndex)(c)
}

type FtSyndumpIndex Completed

func (c FtSyndumpIndex) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type FtSynupdate Completed

func (b Builder) FtSynupdate() (c FtSynupdate) {
	c = FtSynupdate{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "FT.SYNUPDATE")
	return c
}

func (c FtSynupdate) Index(index string) FtSynupdateIndex {
	c.cs.s = append(c.cs.s, index)
	return (FtSynupdateIndex)(c)
}

type FtSynupdateIndex Completed

func (c FtSynupdateIndex) SynonymGroupId(synonymGroupId string) FtSynupdateSynonymGroupId {
	c.cs.s = append(c.cs.s, synonymGroupId)
	return (FtSynupdateSynonymGroupId)(c)
}

type FtSynupdateSkipinitialscan Completed

func (c FtSynupdateSkipinitialscan) Term(term ...string) FtSynupdateTerm {
	c.cs.s = append(c.cs.s, term...)
	return (FtSynupdateTerm)(c)
}

type FtSynupdateSynonymGroupId Completed

func (c FtSynupdateSynonymGroupId) Skipinitialscan() FtSynupdateSkipinitialscan {
	c.cs.s = append(c.cs.s, "SKIPINITIALSCAN")
	return (FtSynupdateSkipinitialscan)(c)
}

func (c FtSynupdateSynonymGroupId) Term(term ...string) FtSynupdateTerm {
	c.cs.s = append(c.cs.s, term...)
	return (FtSynupdateTerm)(c)
}

type FtSynupdateTerm Completed

func (c FtSynupdateTerm) Term(term ...string) FtSynupdateTerm {
	c.cs.s = append(c.cs.s, term...)
	return c
}

func (c FtSynupdateTerm) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type FtTagvals Completed

func (b Builder) FtTagvals() (c FtTagvals) {
	c = FtTagvals{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "FT.TAGVALS")
	return c
}

func (c FtTagvals) Index(index string) FtTagvalsIndex {
	c.cs.s = append(c.cs.s, index)
	return (FtTagvalsIndex)(c)
}

type FtTagvalsFieldName Completed

func (c FtTagvalsFieldName) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type FtTagvalsIndex Completed

func (c FtTagvalsIndex) FieldName(fieldName string) FtTagvalsFieldName {
	c.cs.s = append(c.cs.s, fieldName)
	return (FtTagvalsFieldName)(c)
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

type Geoadd Completed

func (b Builder) Geoadd() (c Geoadd) {
	c = Geoadd{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "GEOADD")
	return c
}

func (c Geoadd) Key(key string) GeoaddKey {
	if c.ks&NoSlot == NoSlot {
		c.ks = NoSlot | slot(key)
	} else {
		c.ks = check(c.ks, slot(key))
	}
	c.cs.s = append(c.cs.s, key)
	return (GeoaddKey)(c)
}

type GeoaddChangeCh Completed

func (c GeoaddChangeCh) LongitudeLatitudeMember() GeoaddLongitudeLatitudeMember {
	return (GeoaddLongitudeLatitudeMember)(c)
}

type GeoaddConditionNx Completed

func (c GeoaddConditionNx) Ch() GeoaddChangeCh {
	c.cs.s = append(c.cs.s, "CH")
	return (GeoaddChangeCh)(c)
}

func (c GeoaddConditionNx) LongitudeLatitudeMember() GeoaddLongitudeLatitudeMember {
	return (GeoaddLongitudeLatitudeMember)(c)
}

type GeoaddConditionXx Completed

func (c GeoaddConditionXx) Ch() GeoaddChangeCh {
	c.cs.s = append(c.cs.s, "CH")
	return (GeoaddChangeCh)(c)
}

func (c GeoaddConditionXx) LongitudeLatitudeMember() GeoaddLongitudeLatitudeMember {
	return (GeoaddLongitudeLatitudeMember)(c)
}

type GeoaddKey Completed

func (c GeoaddKey) Nx() GeoaddConditionNx {
	c.cs.s = append(c.cs.s, "NX")
	return (GeoaddConditionNx)(c)
}

func (c GeoaddKey) Xx() GeoaddConditionXx {
	c.cs.s = append(c.cs.s, "XX")
	return (GeoaddConditionXx)(c)
}

func (c GeoaddKey) Ch() GeoaddChangeCh {
	c.cs.s = append(c.cs.s, "CH")
	return (GeoaddChangeCh)(c)
}

func (c GeoaddKey) LongitudeLatitudeMember() GeoaddLongitudeLatitudeMember {
	return (GeoaddLongitudeLatitudeMember)(c)
}

type GeoaddLongitudeLatitudeMember Completed

func (c GeoaddLongitudeLatitudeMember) LongitudeLatitudeMember(longitude float64, latitude float64, member string) GeoaddLongitudeLatitudeMember {
	c.cs.s = append(c.cs.s, strconv.FormatFloat(longitude, 'f', -1, 64), strconv.FormatFloat(latitude, 'f', -1, 64), member)
	return c
}

func (c GeoaddLongitudeLatitudeMember) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type Geodist Completed

func (b Builder) Geodist() (c Geodist) {
	c = Geodist{cs: get(), ks: b.ks, cf: readonly}
	c.cs.s = append(c.cs.s, "GEODIST")
	return c
}

func (c Geodist) Key(key string) GeodistKey {
	if c.ks&NoSlot == NoSlot {
		c.ks = NoSlot | slot(key)
	} else {
		c.ks = check(c.ks, slot(key))
	}
	c.cs.s = append(c.cs.s, key)
	return (GeodistKey)(c)
}

type GeodistKey Completed

func (c GeodistKey) Member1(member1 string) GeodistMember1 {
	c.cs.s = append(c.cs.s, member1)
	return (GeodistMember1)(c)
}

type GeodistMember1 Completed

func (c GeodistMember1) Member2(member2 string) GeodistMember2 {
	c.cs.s = append(c.cs.s, member2)
	return (GeodistMember2)(c)
}

type GeodistMember2 Completed

func (c GeodistMember2) M() GeodistUnitM {
	c.cs.s = append(c.cs.s, "m")
	return (GeodistUnitM)(c)
}

func (c GeodistMember2) Km() GeodistUnitKm {
	c.cs.s = append(c.cs.s, "km")
	return (GeodistUnitKm)(c)
}

func (c GeodistMember2) Ft() GeodistUnitFt {
	c.cs.s = append(c.cs.s, "ft")
	return (GeodistUnitFt)(c)
}

func (c GeodistMember2) Mi() GeodistUnitMi {
	c.cs.s = append(c.cs.s, "mi")
	return (GeodistUnitMi)(c)
}

func (c GeodistMember2) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

func (c GeodistMember2) Cache() Cacheable {
	c.cs.Build()
	return Cacheable(c)
}

type GeodistUnitFt Completed

func (c GeodistUnitFt) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

func (c GeodistUnitFt) Cache() Cacheable {
	c.cs.Build()
	return Cacheable(c)
}

type GeodistUnitKm Completed

func (c GeodistUnitKm) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

func (c GeodistUnitKm) Cache() Cacheable {
	c.cs.Build()
	return Cacheable(c)
}

type GeodistUnitM Completed

func (c GeodistUnitM) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

func (c GeodistUnitM) Cache() Cacheable {
	c.cs.Build()
	return Cacheable(c)
}

type GeodistUnitMi Completed

func (c GeodistUnitMi) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

func (c GeodistUnitMi) Cache() Cacheable {
	c.cs.Build()
	return Cacheable(c)
}

type Geohash Completed

func (b Builder) Geohash() (c Geohash) {
	c = Geohash{cs: get(), ks: b.ks, cf: readonly}
	c.cs.s = append(c.cs.s, "GEOHASH")
	return c
}

func (c Geohash) Key(key string) GeohashKey {
	if c.ks&NoSlot == NoSlot {
		c.ks = NoSlot | slot(key)
	} else {
		c.ks = check(c.ks, slot(key))
	}
	c.cs.s = append(c.cs.s, key)
	return (GeohashKey)(c)
}

type GeohashKey Completed

func (c GeohashKey) Member(member ...string) GeohashMember {
	c.cs.s = append(c.cs.s, member...)
	return (GeohashMember)(c)
}

func (c GeohashKey) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

func (c GeohashKey) Cache() Cacheable {
	c.cs.Build()
	return Cacheable(c)
}

type GeohashMember Completed

func (c GeohashMember) Member(member ...string) GeohashMember {
	c.cs.s = append(c.cs.s, member...)
	return c
}

func (c GeohashMember) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

func (c GeohashMember) Cache() Cacheable {
	c.cs.Build()
	return Cacheable(c)
}

type Geopos Completed

func (b Builder) Geopos() (c Geopos) {
	c = Geopos{cs: get(), ks: b.ks, cf: readonly}
	c.cs.s = append(c.cs.s, "GEOPOS")
	return c
}

func (c Geopos) Key(key string) GeoposKey {
	if c.ks&NoSlot == NoSlot {
		c.ks = NoSlot | slot(key)
	} else {
		c.ks = check(c.ks, slot(key))
	}
	c.cs.s = append(c.cs.s, key)
	return (GeoposKey)(c)
}

type GeoposKey Completed

func (c GeoposKey) Member(member ...string) GeoposMember {
	c.cs.s = append(c.cs.s, member...)
	return (GeoposMember)(c)
}

func (c GeoposKey) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

func (c GeoposKey) Cache() Cacheable {
	c.cs.Build()
	return Cacheable(c)
}

type GeoposMember Completed

func (c GeoposMember) Member(member ...string) GeoposMember {
	c.cs.s = append(c.cs.s, member...)
	return c
}

func (c GeoposMember) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

func (c GeoposMember) Cache() Cacheable {
	c.cs.Build()
	return Cacheable(c)
}

type Georadius Completed

func (b Builder) Georadius() (c Georadius) {
	c = Georadius{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "GEORADIUS")
	return c
}

func (c Georadius) Key(key string) GeoradiusKey {
	if c.ks&NoSlot == NoSlot {
		c.ks = NoSlot | slot(key)
	} else {
		c.ks = check(c.ks, slot(key))
	}
	c.cs.s = append(c.cs.s, key)
	return (GeoradiusKey)(c)
}

type GeoradiusCountAny Completed

func (c GeoradiusCountAny) Asc() GeoradiusOrderAsc {
	c.cs.s = append(c.cs.s, "ASC")
	return (GeoradiusOrderAsc)(c)
}

func (c GeoradiusCountAny) Desc() GeoradiusOrderDesc {
	c.cs.s = append(c.cs.s, "DESC")
	return (GeoradiusOrderDesc)(c)
}

func (c GeoradiusCountAny) Store(key string) GeoradiusStore {
	if c.ks&NoSlot == NoSlot {
		c.ks = NoSlot | slot(key)
	} else {
		c.ks = check(c.ks, slot(key))
	}
	c.cs.s = append(c.cs.s, "STORE", key)
	return (GeoradiusStore)(c)
}

func (c GeoradiusCountAny) Storedist(key string) GeoradiusStoredist {
	if c.ks&NoSlot == NoSlot {
		c.ks = NoSlot | slot(key)
	} else {
		c.ks = check(c.ks, slot(key))
	}
	c.cs.s = append(c.cs.s, "STOREDIST", key)
	return (GeoradiusStoredist)(c)
}

func (c GeoradiusCountAny) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type GeoradiusCountCount Completed

func (c GeoradiusCountCount) Any() GeoradiusCountAny {
	c.cs.s = append(c.cs.s, "ANY")
	return (GeoradiusCountAny)(c)
}

func (c GeoradiusCountCount) Asc() GeoradiusOrderAsc {
	c.cs.s = append(c.cs.s, "ASC")
	return (GeoradiusOrderAsc)(c)
}

func (c GeoradiusCountCount) Desc() GeoradiusOrderDesc {
	c.cs.s = append(c.cs.s, "DESC")
	return (GeoradiusOrderDesc)(c)
}

func (c GeoradiusCountCount) Store(key string) GeoradiusStore {
	if c.ks&NoSlot == NoSlot {
		c.ks = NoSlot | slot(key)
	} else {
		c.ks = check(c.ks, slot(key))
	}
	c.cs.s = append(c.cs.s, "STORE", key)
	return (GeoradiusStore)(c)
}

func (c GeoradiusCountCount) Storedist(key string) GeoradiusStoredist {
	if c.ks&NoSlot == NoSlot {
		c.ks = NoSlot | slot(key)
	} else {
		c.ks = check(c.ks, slot(key))
	}
	c.cs.s = append(c.cs.s, "STOREDIST", key)
	return (GeoradiusStoredist)(c)
}

func (c GeoradiusCountCount) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type GeoradiusKey Completed

func (c GeoradiusKey) Longitude(longitude float64) GeoradiusLongitude {
	c.cs.s = append(c.cs.s, strconv.FormatFloat(longitude, 'f', -1, 64))
	return (GeoradiusLongitude)(c)
}

type GeoradiusLatitude Completed

func (c GeoradiusLatitude) Radius(radius float64) GeoradiusRadius {
	c.cs.s = append(c.cs.s, strconv.FormatFloat(radius, 'f', -1, 64))
	return (GeoradiusRadius)(c)
}

type GeoradiusLongitude Completed

func (c GeoradiusLongitude) Latitude(latitude float64) GeoradiusLatitude {
	c.cs.s = append(c.cs.s, strconv.FormatFloat(latitude, 'f', -1, 64))
	return (GeoradiusLatitude)(c)
}

type GeoradiusOrderAsc Completed

func (c GeoradiusOrderAsc) Store(key string) GeoradiusStore {
	if c.ks&NoSlot == NoSlot {
		c.ks = NoSlot | slot(key)
	} else {
		c.ks = check(c.ks, slot(key))
	}
	c.cs.s = append(c.cs.s, "STORE", key)
	return (GeoradiusStore)(c)
}

func (c GeoradiusOrderAsc) Storedist(key string) GeoradiusStoredist {
	if c.ks&NoSlot == NoSlot {
		c.ks = NoSlot | slot(key)
	} else {
		c.ks = check(c.ks, slot(key))
	}
	c.cs.s = append(c.cs.s, "STOREDIST", key)
	return (GeoradiusStoredist)(c)
}

func (c GeoradiusOrderAsc) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type GeoradiusOrderDesc Completed

func (c GeoradiusOrderDesc) Store(key string) GeoradiusStore {
	if c.ks&NoSlot == NoSlot {
		c.ks = NoSlot | slot(key)
	} else {
		c.ks = check(c.ks, slot(key))
	}
	c.cs.s = append(c.cs.s, "STORE", key)
	return (GeoradiusStore)(c)
}

func (c GeoradiusOrderDesc) Storedist(key string) GeoradiusStoredist {
	if c.ks&NoSlot == NoSlot {
		c.ks = NoSlot | slot(key)
	} else {
		c.ks = check(c.ks, slot(key))
	}
	c.cs.s = append(c.cs.s, "STOREDIST", key)
	return (GeoradiusStoredist)(c)
}

func (c GeoradiusOrderDesc) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type GeoradiusRadius Completed

func (c GeoradiusRadius) M() GeoradiusUnitM {
	c.cs.s = append(c.cs.s, "m")
	return (GeoradiusUnitM)(c)
}

func (c GeoradiusRadius) Km() GeoradiusUnitKm {
	c.cs.s = append(c.cs.s, "km")
	return (GeoradiusUnitKm)(c)
}

func (c GeoradiusRadius) Ft() GeoradiusUnitFt {
	c.cs.s = append(c.cs.s, "ft")
	return (GeoradiusUnitFt)(c)
}

func (c GeoradiusRadius) Mi() GeoradiusUnitMi {
	c.cs.s = append(c.cs.s, "mi")
	return (GeoradiusUnitMi)(c)
}

type GeoradiusRo Completed

func (b Builder) GeoradiusRo() (c GeoradiusRo) {
	c = GeoradiusRo{cs: get(), ks: b.ks, cf: readonly}
	c.cs.s = append(c.cs.s, "GEORADIUS_RO")
	return c
}

func (c GeoradiusRo) Key(key string) GeoradiusRoKey {
	if c.ks&NoSlot == NoSlot {
		c.ks = NoSlot | slot(key)
	} else {
		c.ks = check(c.ks, slot(key))
	}
	c.cs.s = append(c.cs.s, key)
	return (GeoradiusRoKey)(c)
}

type GeoradiusRoCountAny Completed

func (c GeoradiusRoCountAny) Asc() GeoradiusRoOrderAsc {
	c.cs.s = append(c.cs.s, "ASC")
	return (GeoradiusRoOrderAsc)(c)
}

func (c GeoradiusRoCountAny) Desc() GeoradiusRoOrderDesc {
	c.cs.s = append(c.cs.s, "DESC")
	return (GeoradiusRoOrderDesc)(c)
}

func (c GeoradiusRoCountAny) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

func (c GeoradiusRoCountAny) Cache() Cacheable {
	c.cs.Build()
	return Cacheable(c)
}

type GeoradiusRoCountCount Completed

func (c GeoradiusRoCountCount) Any() GeoradiusRoCountAny {
	c.cs.s = append(c.cs.s, "ANY")
	return (GeoradiusRoCountAny)(c)
}

func (c GeoradiusRoCountCount) Asc() GeoradiusRoOrderAsc {
	c.cs.s = append(c.cs.s, "ASC")
	return (GeoradiusRoOrderAsc)(c)
}

func (c GeoradiusRoCountCount) Desc() GeoradiusRoOrderDesc {
	c.cs.s = append(c.cs.s, "DESC")
	return (GeoradiusRoOrderDesc)(c)
}

func (c GeoradiusRoCountCount) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

func (c GeoradiusRoCountCount) Cache() Cacheable {
	c.cs.Build()
	return Cacheable(c)
}

type GeoradiusRoKey Completed

func (c GeoradiusRoKey) Longitude(longitude float64) GeoradiusRoLongitude {
	c.cs.s = append(c.cs.s, strconv.FormatFloat(longitude, 'f', -1, 64))
	return (GeoradiusRoLongitude)(c)
}

type GeoradiusRoLatitude Completed

func (c GeoradiusRoLatitude) Radius(radius float64) GeoradiusRoRadius {
	c.cs.s = append(c.cs.s, strconv.FormatFloat(radius, 'f', -1, 64))
	return (GeoradiusRoRadius)(c)
}

type GeoradiusRoLongitude Completed

func (c GeoradiusRoLongitude) Latitude(latitude float64) GeoradiusRoLatitude {
	c.cs.s = append(c.cs.s, strconv.FormatFloat(latitude, 'f', -1, 64))
	return (GeoradiusRoLatitude)(c)
}

type GeoradiusRoOrderAsc Completed

func (c GeoradiusRoOrderAsc) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

func (c GeoradiusRoOrderAsc) Cache() Cacheable {
	c.cs.Build()
	return Cacheable(c)
}

type GeoradiusRoOrderDesc Completed

func (c GeoradiusRoOrderDesc) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

func (c GeoradiusRoOrderDesc) Cache() Cacheable {
	c.cs.Build()
	return Cacheable(c)
}

type GeoradiusRoRadius Completed

func (c GeoradiusRoRadius) M() GeoradiusRoUnitM {
	c.cs.s = append(c.cs.s, "m")
	return (GeoradiusRoUnitM)(c)
}

func (c GeoradiusRoRadius) Km() GeoradiusRoUnitKm {
	c.cs.s = append(c.cs.s, "km")
	return (GeoradiusRoUnitKm)(c)
}

func (c GeoradiusRoRadius) Ft() GeoradiusRoUnitFt {
	c.cs.s = append(c.cs.s, "ft")
	return (GeoradiusRoUnitFt)(c)
}

func (c GeoradiusRoRadius) Mi() GeoradiusRoUnitMi {
	c.cs.s = append(c.cs.s, "mi")
	return (GeoradiusRoUnitMi)(c)
}

type GeoradiusRoUnitFt Completed

func (c GeoradiusRoUnitFt) Withcoord() GeoradiusRoWithcoord {
	c.cs.s = append(c.cs.s, "WITHCOORD")
	return (GeoradiusRoWithcoord)(c)
}

func (c GeoradiusRoUnitFt) Withdist() GeoradiusRoWithdist {
	c.cs.s = append(c.cs.s, "WITHDIST")
	return (GeoradiusRoWithdist)(c)
}

func (c GeoradiusRoUnitFt) Withhash() GeoradiusRoWithhash {
	c.cs.s = append(c.cs.s, "WITHHASH")
	return (GeoradiusRoWithhash)(c)
}

func (c GeoradiusRoUnitFt) Count(count int64) GeoradiusRoCountCount {
	c.cs.s = append(c.cs.s, "COUNT", strconv.FormatInt(count, 10))
	return (GeoradiusRoCountCount)(c)
}

func (c GeoradiusRoUnitFt) Asc() GeoradiusRoOrderAsc {
	c.cs.s = append(c.cs.s, "ASC")
	return (GeoradiusRoOrderAsc)(c)
}

func (c GeoradiusRoUnitFt) Desc() GeoradiusRoOrderDesc {
	c.cs.s = append(c.cs.s, "DESC")
	return (GeoradiusRoOrderDesc)(c)
}

func (c GeoradiusRoUnitFt) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

func (c GeoradiusRoUnitFt) Cache() Cacheable {
	c.cs.Build()
	return Cacheable(c)
}

type GeoradiusRoUnitKm Completed

func (c GeoradiusRoUnitKm) Withcoord() GeoradiusRoWithcoord {
	c.cs.s = append(c.cs.s, "WITHCOORD")
	return (GeoradiusRoWithcoord)(c)
}

func (c GeoradiusRoUnitKm) Withdist() GeoradiusRoWithdist {
	c.cs.s = append(c.cs.s, "WITHDIST")
	return (GeoradiusRoWithdist)(c)
}

func (c GeoradiusRoUnitKm) Withhash() GeoradiusRoWithhash {
	c.cs.s = append(c.cs.s, "WITHHASH")
	return (GeoradiusRoWithhash)(c)
}

func (c GeoradiusRoUnitKm) Count(count int64) GeoradiusRoCountCount {
	c.cs.s = append(c.cs.s, "COUNT", strconv.FormatInt(count, 10))
	return (GeoradiusRoCountCount)(c)
}

func (c GeoradiusRoUnitKm) Asc() GeoradiusRoOrderAsc {
	c.cs.s = append(c.cs.s, "ASC")
	return (GeoradiusRoOrderAsc)(c)
}

func (c GeoradiusRoUnitKm) Desc() GeoradiusRoOrderDesc {
	c.cs.s = append(c.cs.s, "DESC")
	return (GeoradiusRoOrderDesc)(c)
}

func (c GeoradiusRoUnitKm) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

func (c GeoradiusRoUnitKm) Cache() Cacheable {
	c.cs.Build()
	return Cacheable(c)
}

type GeoradiusRoUnitM Completed

func (c GeoradiusRoUnitM) Withcoord() GeoradiusRoWithcoord {
	c.cs.s = append(c.cs.s, "WITHCOORD")
	return (GeoradiusRoWithcoord)(c)
}

func (c GeoradiusRoUnitM) Withdist() GeoradiusRoWithdist {
	c.cs.s = append(c.cs.s, "WITHDIST")
	return (GeoradiusRoWithdist)(c)
}

func (c GeoradiusRoUnitM) Withhash() GeoradiusRoWithhash {
	c.cs.s = append(c.cs.s, "WITHHASH")
	return (GeoradiusRoWithhash)(c)
}

func (c GeoradiusRoUnitM) Count(count int64) GeoradiusRoCountCount {
	c.cs.s = append(c.cs.s, "COUNT", strconv.FormatInt(count, 10))
	return (GeoradiusRoCountCount)(c)
}

func (c GeoradiusRoUnitM) Asc() GeoradiusRoOrderAsc {
	c.cs.s = append(c.cs.s, "ASC")
	return (GeoradiusRoOrderAsc)(c)
}

func (c GeoradiusRoUnitM) Desc() GeoradiusRoOrderDesc {
	c.cs.s = append(c.cs.s, "DESC")
	return (GeoradiusRoOrderDesc)(c)
}

func (c GeoradiusRoUnitM) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

func (c GeoradiusRoUnitM) Cache() Cacheable {
	c.cs.Build()
	return Cacheable(c)
}

type GeoradiusRoUnitMi Completed

func (c GeoradiusRoUnitMi) Withcoord() GeoradiusRoWithcoord {
	c.cs.s = append(c.cs.s, "WITHCOORD")
	return (GeoradiusRoWithcoord)(c)
}

func (c GeoradiusRoUnitMi) Withdist() GeoradiusRoWithdist {
	c.cs.s = append(c.cs.s, "WITHDIST")
	return (GeoradiusRoWithdist)(c)
}

func (c GeoradiusRoUnitMi) Withhash() GeoradiusRoWithhash {
	c.cs.s = append(c.cs.s, "WITHHASH")
	return (GeoradiusRoWithhash)(c)
}

func (c GeoradiusRoUnitMi) Count(count int64) GeoradiusRoCountCount {
	c.cs.s = append(c.cs.s, "COUNT", strconv.FormatInt(count, 10))
	return (GeoradiusRoCountCount)(c)
}

func (c GeoradiusRoUnitMi) Asc() GeoradiusRoOrderAsc {
	c.cs.s = append(c.cs.s, "ASC")
	return (GeoradiusRoOrderAsc)(c)
}

func (c GeoradiusRoUnitMi) Desc() GeoradiusRoOrderDesc {
	c.cs.s = append(c.cs.s, "DESC")
	return (GeoradiusRoOrderDesc)(c)
}

func (c GeoradiusRoUnitMi) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

func (c GeoradiusRoUnitMi) Cache() Cacheable {
	c.cs.Build()
	return Cacheable(c)
}

type GeoradiusRoWithcoord Completed

func (c GeoradiusRoWithcoord) Withdist() GeoradiusRoWithdist {
	c.cs.s = append(c.cs.s, "WITHDIST")
	return (GeoradiusRoWithdist)(c)
}

func (c GeoradiusRoWithcoord) Withhash() GeoradiusRoWithhash {
	c.cs.s = append(c.cs.s, "WITHHASH")
	return (GeoradiusRoWithhash)(c)
}

func (c GeoradiusRoWithcoord) Count(count int64) GeoradiusRoCountCount {
	c.cs.s = append(c.cs.s, "COUNT", strconv.FormatInt(count, 10))
	return (GeoradiusRoCountCount)(c)
}

func (c GeoradiusRoWithcoord) Asc() GeoradiusRoOrderAsc {
	c.cs.s = append(c.cs.s, "ASC")
	return (GeoradiusRoOrderAsc)(c)
}

func (c GeoradiusRoWithcoord) Desc() GeoradiusRoOrderDesc {
	c.cs.s = append(c.cs.s, "DESC")
	return (GeoradiusRoOrderDesc)(c)
}

func (c GeoradiusRoWithcoord) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

func (c GeoradiusRoWithcoord) Cache() Cacheable {
	c.cs.Build()
	return Cacheable(c)
}

type GeoradiusRoWithdist Completed

func (c GeoradiusRoWithdist) Withhash() GeoradiusRoWithhash {
	c.cs.s = append(c.cs.s, "WITHHASH")
	return (GeoradiusRoWithhash)(c)
}

func (c GeoradiusRoWithdist) Count(count int64) GeoradiusRoCountCount {
	c.cs.s = append(c.cs.s, "COUNT", strconv.FormatInt(count, 10))
	return (GeoradiusRoCountCount)(c)
}

func (c GeoradiusRoWithdist) Asc() GeoradiusRoOrderAsc {
	c.cs.s = append(c.cs.s, "ASC")
	return (GeoradiusRoOrderAsc)(c)
}

func (c GeoradiusRoWithdist) Desc() GeoradiusRoOrderDesc {
	c.cs.s = append(c.cs.s, "DESC")
	return (GeoradiusRoOrderDesc)(c)
}

func (c GeoradiusRoWithdist) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

func (c GeoradiusRoWithdist) Cache() Cacheable {
	c.cs.Build()
	return Cacheable(c)
}

type GeoradiusRoWithhash Completed

func (c GeoradiusRoWithhash) Count(count int64) GeoradiusRoCountCount {
	c.cs.s = append(c.cs.s, "COUNT", strconv.FormatInt(count, 10))
	return (GeoradiusRoCountCount)(c)
}

func (c GeoradiusRoWithhash) Asc() GeoradiusRoOrderAsc {
	c.cs.s = append(c.cs.s, "ASC")
	return (GeoradiusRoOrderAsc)(c)
}

func (c GeoradiusRoWithhash) Desc() GeoradiusRoOrderDesc {
	c.cs.s = append(c.cs.s, "DESC")
	return (GeoradiusRoOrderDesc)(c)
}

func (c GeoradiusRoWithhash) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

func (c GeoradiusRoWithhash) Cache() Cacheable {
	c.cs.Build()
	return Cacheable(c)
}

type GeoradiusStore Completed

func (c GeoradiusStore) Storedist(key string) GeoradiusStoredist {
	if c.ks&NoSlot == NoSlot {
		c.ks = NoSlot | slot(key)
	} else {
		c.ks = check(c.ks, slot(key))
	}
	c.cs.s = append(c.cs.s, "STOREDIST", key)
	return (GeoradiusStoredist)(c)
}

func (c GeoradiusStore) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type GeoradiusStoredist Completed

func (c GeoradiusStoredist) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type GeoradiusUnitFt Completed

func (c GeoradiusUnitFt) Withcoord() GeoradiusWithcoord {
	c.cs.s = append(c.cs.s, "WITHCOORD")
	return (GeoradiusWithcoord)(c)
}

func (c GeoradiusUnitFt) Withdist() GeoradiusWithdist {
	c.cs.s = append(c.cs.s, "WITHDIST")
	return (GeoradiusWithdist)(c)
}

func (c GeoradiusUnitFt) Withhash() GeoradiusWithhash {
	c.cs.s = append(c.cs.s, "WITHHASH")
	return (GeoradiusWithhash)(c)
}

func (c GeoradiusUnitFt) Count(count int64) GeoradiusCountCount {
	c.cs.s = append(c.cs.s, "COUNT", strconv.FormatInt(count, 10))
	return (GeoradiusCountCount)(c)
}

func (c GeoradiusUnitFt) Asc() GeoradiusOrderAsc {
	c.cs.s = append(c.cs.s, "ASC")
	return (GeoradiusOrderAsc)(c)
}

func (c GeoradiusUnitFt) Desc() GeoradiusOrderDesc {
	c.cs.s = append(c.cs.s, "DESC")
	return (GeoradiusOrderDesc)(c)
}

func (c GeoradiusUnitFt) Store(key string) GeoradiusStore {
	if c.ks&NoSlot == NoSlot {
		c.ks = NoSlot | slot(key)
	} else {
		c.ks = check(c.ks, slot(key))
	}
	c.cs.s = append(c.cs.s, "STORE", key)
	return (GeoradiusStore)(c)
}

func (c GeoradiusUnitFt) Storedist(key string) GeoradiusStoredist {
	if c.ks&NoSlot == NoSlot {
		c.ks = NoSlot | slot(key)
	} else {
		c.ks = check(c.ks, slot(key))
	}
	c.cs.s = append(c.cs.s, "STOREDIST", key)
	return (GeoradiusStoredist)(c)
}

func (c GeoradiusUnitFt) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type GeoradiusUnitKm Completed

func (c GeoradiusUnitKm) Withcoord() GeoradiusWithcoord {
	c.cs.s = append(c.cs.s, "WITHCOORD")
	return (GeoradiusWithcoord)(c)
}

func (c GeoradiusUnitKm) Withdist() GeoradiusWithdist {
	c.cs.s = append(c.cs.s, "WITHDIST")
	return (GeoradiusWithdist)(c)
}

func (c GeoradiusUnitKm) Withhash() GeoradiusWithhash {
	c.cs.s = append(c.cs.s, "WITHHASH")
	return (GeoradiusWithhash)(c)
}

func (c GeoradiusUnitKm) Count(count int64) GeoradiusCountCount {
	c.cs.s = append(c.cs.s, "COUNT", strconv.FormatInt(count, 10))
	return (GeoradiusCountCount)(c)
}

func (c GeoradiusUnitKm) Asc() GeoradiusOrderAsc {
	c.cs.s = append(c.cs.s, "ASC")
	return (GeoradiusOrderAsc)(c)
}

func (c GeoradiusUnitKm) Desc() GeoradiusOrderDesc {
	c.cs.s = append(c.cs.s, "DESC")
	return (GeoradiusOrderDesc)(c)
}

func (c GeoradiusUnitKm) Store(key string) GeoradiusStore {
	if c.ks&NoSlot == NoSlot {
		c.ks = NoSlot | slot(key)
	} else {
		c.ks = check(c.ks, slot(key))
	}
	c.cs.s = append(c.cs.s, "STORE", key)
	return (GeoradiusStore)(c)
}

func (c GeoradiusUnitKm) Storedist(key string) GeoradiusStoredist {
	if c.ks&NoSlot == NoSlot {
		c.ks = NoSlot | slot(key)
	} else {
		c.ks = check(c.ks, slot(key))
	}
	c.cs.s = append(c.cs.s, "STOREDIST", key)
	return (GeoradiusStoredist)(c)
}

func (c GeoradiusUnitKm) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type GeoradiusUnitM Completed

func (c GeoradiusUnitM) Withcoord() GeoradiusWithcoord {
	c.cs.s = append(c.cs.s, "WITHCOORD")
	return (GeoradiusWithcoord)(c)
}

func (c GeoradiusUnitM) Withdist() GeoradiusWithdist {
	c.cs.s = append(c.cs.s, "WITHDIST")
	return (GeoradiusWithdist)(c)
}

func (c GeoradiusUnitM) Withhash() GeoradiusWithhash {
	c.cs.s = append(c.cs.s, "WITHHASH")
	return (GeoradiusWithhash)(c)
}

func (c GeoradiusUnitM) Count(count int64) GeoradiusCountCount {
	c.cs.s = append(c.cs.s, "COUNT", strconv.FormatInt(count, 10))
	return (GeoradiusCountCount)(c)
}

func (c GeoradiusUnitM) Asc() GeoradiusOrderAsc {
	c.cs.s = append(c.cs.s, "ASC")
	return (GeoradiusOrderAsc)(c)
}

func (c GeoradiusUnitM) Desc() GeoradiusOrderDesc {
	c.cs.s = append(c.cs.s, "DESC")
	return (GeoradiusOrderDesc)(c)
}

func (c GeoradiusUnitM) Store(key string) GeoradiusStore {
	if c.ks&NoSlot == NoSlot {
		c.ks = NoSlot | slot(key)
	} else {
		c.ks = check(c.ks, slot(key))
	}
	c.cs.s = append(c.cs.s, "STORE", key)
	return (GeoradiusStore)(c)
}

func (c GeoradiusUnitM) Storedist(key string) GeoradiusStoredist {
	if c.ks&NoSlot == NoSlot {
		c.ks = NoSlot | slot(key)
	} else {
		c.ks = check(c.ks, slot(key))
	}
	c.cs.s = append(c.cs.s, "STOREDIST", key)
	return (GeoradiusStoredist)(c)
}

func (c GeoradiusUnitM) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type GeoradiusUnitMi Completed

func (c GeoradiusUnitMi) Withcoord() GeoradiusWithcoord {
	c.cs.s = append(c.cs.s, "WITHCOORD")
	return (GeoradiusWithcoord)(c)
}

func (c GeoradiusUnitMi) Withdist() GeoradiusWithdist {
	c.cs.s = append(c.cs.s, "WITHDIST")
	return (GeoradiusWithdist)(c)
}

func (c GeoradiusUnitMi) Withhash() GeoradiusWithhash {
	c.cs.s = append(c.cs.s, "WITHHASH")
	return (GeoradiusWithhash)(c)
}

func (c GeoradiusUnitMi) Count(count int64) GeoradiusCountCount {
	c.cs.s = append(c.cs.s, "COUNT", strconv.FormatInt(count, 10))
	return (GeoradiusCountCount)(c)
}

func (c GeoradiusUnitMi) Asc() GeoradiusOrderAsc {
	c.cs.s = append(c.cs.s, "ASC")
	return (GeoradiusOrderAsc)(c)
}

func (c GeoradiusUnitMi) Desc() GeoradiusOrderDesc {
	c.cs.s = append(c.cs.s, "DESC")
	return (GeoradiusOrderDesc)(c)
}

func (c GeoradiusUnitMi) Store(key string) GeoradiusStore {
	if c.ks&NoSlot == NoSlot {
		c.ks = NoSlot | slot(key)
	} else {
		c.ks = check(c.ks, slot(key))
	}
	c.cs.s = append(c.cs.s, "STORE", key)
	return (GeoradiusStore)(c)
}

func (c GeoradiusUnitMi) Storedist(key string) GeoradiusStoredist {
	if c.ks&NoSlot == NoSlot {
		c.ks = NoSlot | slot(key)
	} else {
		c.ks = check(c.ks, slot(key))
	}
	c.cs.s = append(c.cs.s, "STOREDIST", key)
	return (GeoradiusStoredist)(c)
}

func (c GeoradiusUnitMi) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type GeoradiusWithcoord Completed

func (c GeoradiusWithcoord) Withdist() GeoradiusWithdist {
	c.cs.s = append(c.cs.s, "WITHDIST")
	return (GeoradiusWithdist)(c)
}

func (c GeoradiusWithcoord) Withhash() GeoradiusWithhash {
	c.cs.s = append(c.cs.s, "WITHHASH")
	return (GeoradiusWithhash)(c)
}

func (c GeoradiusWithcoord) Count(count int64) GeoradiusCountCount {
	c.cs.s = append(c.cs.s, "COUNT", strconv.FormatInt(count, 10))
	return (GeoradiusCountCount)(c)
}

func (c GeoradiusWithcoord) Asc() GeoradiusOrderAsc {
	c.cs.s = append(c.cs.s, "ASC")
	return (GeoradiusOrderAsc)(c)
}

func (c GeoradiusWithcoord) Desc() GeoradiusOrderDesc {
	c.cs.s = append(c.cs.s, "DESC")
	return (GeoradiusOrderDesc)(c)
}

func (c GeoradiusWithcoord) Store(key string) GeoradiusStore {
	if c.ks&NoSlot == NoSlot {
		c.ks = NoSlot | slot(key)
	} else {
		c.ks = check(c.ks, slot(key))
	}
	c.cs.s = append(c.cs.s, "STORE", key)
	return (GeoradiusStore)(c)
}

func (c GeoradiusWithcoord) Storedist(key string) GeoradiusStoredist {
	if c.ks&NoSlot == NoSlot {
		c.ks = NoSlot | slot(key)
	} else {
		c.ks = check(c.ks, slot(key))
	}
	c.cs.s = append(c.cs.s, "STOREDIST", key)
	return (GeoradiusStoredist)(c)
}

func (c GeoradiusWithcoord) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type GeoradiusWithdist Completed

func (c GeoradiusWithdist) Withhash() GeoradiusWithhash {
	c.cs.s = append(c.cs.s, "WITHHASH")
	return (GeoradiusWithhash)(c)
}

func (c GeoradiusWithdist) Count(count int64) GeoradiusCountCount {
	c.cs.s = append(c.cs.s, "COUNT", strconv.FormatInt(count, 10))
	return (GeoradiusCountCount)(c)
}

func (c GeoradiusWithdist) Asc() GeoradiusOrderAsc {
	c.cs.s = append(c.cs.s, "ASC")
	return (GeoradiusOrderAsc)(c)
}

func (c GeoradiusWithdist) Desc() GeoradiusOrderDesc {
	c.cs.s = append(c.cs.s, "DESC")
	return (GeoradiusOrderDesc)(c)
}

func (c GeoradiusWithdist) Store(key string) GeoradiusStore {
	if c.ks&NoSlot == NoSlot {
		c.ks = NoSlot | slot(key)
	} else {
		c.ks = check(c.ks, slot(key))
	}
	c.cs.s = append(c.cs.s, "STORE", key)
	return (GeoradiusStore)(c)
}

func (c GeoradiusWithdist) Storedist(key string) GeoradiusStoredist {
	if c.ks&NoSlot == NoSlot {
		c.ks = NoSlot | slot(key)
	} else {
		c.ks = check(c.ks, slot(key))
	}
	c.cs.s = append(c.cs.s, "STOREDIST", key)
	return (GeoradiusStoredist)(c)
}

func (c GeoradiusWithdist) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type GeoradiusWithhash Completed

func (c GeoradiusWithhash) Count(count int64) GeoradiusCountCount {
	c.cs.s = append(c.cs.s, "COUNT", strconv.FormatInt(count, 10))
	return (GeoradiusCountCount)(c)
}

func (c GeoradiusWithhash) Asc() GeoradiusOrderAsc {
	c.cs.s = append(c.cs.s, "ASC")
	return (GeoradiusOrderAsc)(c)
}

func (c GeoradiusWithhash) Desc() GeoradiusOrderDesc {
	c.cs.s = append(c.cs.s, "DESC")
	return (GeoradiusOrderDesc)(c)
}

func (c GeoradiusWithhash) Store(key string) GeoradiusStore {
	if c.ks&NoSlot == NoSlot {
		c.ks = NoSlot | slot(key)
	} else {
		c.ks = check(c.ks, slot(key))
	}
	c.cs.s = append(c.cs.s, "STORE", key)
	return (GeoradiusStore)(c)
}

func (c GeoradiusWithhash) Storedist(key string) GeoradiusStoredist {
	if c.ks&NoSlot == NoSlot {
		c.ks = NoSlot | slot(key)
	} else {
		c.ks = check(c.ks, slot(key))
	}
	c.cs.s = append(c.cs.s, "STOREDIST", key)
	return (GeoradiusStoredist)(c)
}

func (c GeoradiusWithhash) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type Georadiusbymember Completed

func (b Builder) Georadiusbymember() (c Georadiusbymember) {
	c = Georadiusbymember{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "GEORADIUSBYMEMBER")
	return c
}

func (c Georadiusbymember) Key(key string) GeoradiusbymemberKey {
	if c.ks&NoSlot == NoSlot {
		c.ks = NoSlot | slot(key)
	} else {
		c.ks = check(c.ks, slot(key))
	}
	c.cs.s = append(c.cs.s, key)
	return (GeoradiusbymemberKey)(c)
}

type GeoradiusbymemberCountAny Completed

func (c GeoradiusbymemberCountAny) Asc() GeoradiusbymemberOrderAsc {
	c.cs.s = append(c.cs.s, "ASC")
	return (GeoradiusbymemberOrderAsc)(c)
}

func (c GeoradiusbymemberCountAny) Desc() GeoradiusbymemberOrderDesc {
	c.cs.s = append(c.cs.s, "DESC")
	return (GeoradiusbymemberOrderDesc)(c)
}

func (c GeoradiusbymemberCountAny) Store(key string) GeoradiusbymemberStore {
	if c.ks&NoSlot == NoSlot {
		c.ks = NoSlot | slot(key)
	} else {
		c.ks = check(c.ks, slot(key))
	}
	c.cs.s = append(c.cs.s, "STORE", key)
	return (GeoradiusbymemberStore)(c)
}

func (c GeoradiusbymemberCountAny) Storedist(key string) GeoradiusbymemberStoredist {
	if c.ks&NoSlot == NoSlot {
		c.ks = NoSlot | slot(key)
	} else {
		c.ks = check(c.ks, slot(key))
	}
	c.cs.s = append(c.cs.s, "STOREDIST", key)
	return (GeoradiusbymemberStoredist)(c)
}

func (c GeoradiusbymemberCountAny) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type GeoradiusbymemberCountCount Completed

func (c GeoradiusbymemberCountCount) Any() GeoradiusbymemberCountAny {
	c.cs.s = append(c.cs.s, "ANY")
	return (GeoradiusbymemberCountAny)(c)
}

func (c GeoradiusbymemberCountCount) Asc() GeoradiusbymemberOrderAsc {
	c.cs.s = append(c.cs.s, "ASC")
	return (GeoradiusbymemberOrderAsc)(c)
}

func (c GeoradiusbymemberCountCount) Desc() GeoradiusbymemberOrderDesc {
	c.cs.s = append(c.cs.s, "DESC")
	return (GeoradiusbymemberOrderDesc)(c)
}

func (c GeoradiusbymemberCountCount) Store(key string) GeoradiusbymemberStore {
	if c.ks&NoSlot == NoSlot {
		c.ks = NoSlot | slot(key)
	} else {
		c.ks = check(c.ks, slot(key))
	}
	c.cs.s = append(c.cs.s, "STORE", key)
	return (GeoradiusbymemberStore)(c)
}

func (c GeoradiusbymemberCountCount) Storedist(key string) GeoradiusbymemberStoredist {
	if c.ks&NoSlot == NoSlot {
		c.ks = NoSlot | slot(key)
	} else {
		c.ks = check(c.ks, slot(key))
	}
	c.cs.s = append(c.cs.s, "STOREDIST", key)
	return (GeoradiusbymemberStoredist)(c)
}

func (c GeoradiusbymemberCountCount) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type GeoradiusbymemberKey Completed

func (c GeoradiusbymemberKey) Member(member string) GeoradiusbymemberMember {
	c.cs.s = append(c.cs.s, member)
	return (GeoradiusbymemberMember)(c)
}

type GeoradiusbymemberMember Completed

func (c GeoradiusbymemberMember) Radius(radius float64) GeoradiusbymemberRadius {
	c.cs.s = append(c.cs.s, strconv.FormatFloat(radius, 'f', -1, 64))
	return (GeoradiusbymemberRadius)(c)
}

type GeoradiusbymemberOrderAsc Completed

func (c GeoradiusbymemberOrderAsc) Store(key string) GeoradiusbymemberStore {
	if c.ks&NoSlot == NoSlot {
		c.ks = NoSlot | slot(key)
	} else {
		c.ks = check(c.ks, slot(key))
	}
	c.cs.s = append(c.cs.s, "STORE", key)
	return (GeoradiusbymemberStore)(c)
}

func (c GeoradiusbymemberOrderAsc) Storedist(key string) GeoradiusbymemberStoredist {
	if c.ks&NoSlot == NoSlot {
		c.ks = NoSlot | slot(key)
	} else {
		c.ks = check(c.ks, slot(key))
	}
	c.cs.s = append(c.cs.s, "STOREDIST", key)
	return (GeoradiusbymemberStoredist)(c)
}

func (c GeoradiusbymemberOrderAsc) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type GeoradiusbymemberOrderDesc Completed

func (c GeoradiusbymemberOrderDesc) Store(key string) GeoradiusbymemberStore {
	if c.ks&NoSlot == NoSlot {
		c.ks = NoSlot | slot(key)
	} else {
		c.ks = check(c.ks, slot(key))
	}
	c.cs.s = append(c.cs.s, "STORE", key)
	return (GeoradiusbymemberStore)(c)
}

func (c GeoradiusbymemberOrderDesc) Storedist(key string) GeoradiusbymemberStoredist {
	if c.ks&NoSlot == NoSlot {
		c.ks = NoSlot | slot(key)
	} else {
		c.ks = check(c.ks, slot(key))
	}
	c.cs.s = append(c.cs.s, "STOREDIST", key)
	return (GeoradiusbymemberStoredist)(c)
}

func (c GeoradiusbymemberOrderDesc) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type GeoradiusbymemberRadius Completed

func (c GeoradiusbymemberRadius) M() GeoradiusbymemberUnitM {
	c.cs.s = append(c.cs.s, "m")
	return (GeoradiusbymemberUnitM)(c)
}

func (c GeoradiusbymemberRadius) Km() GeoradiusbymemberUnitKm {
	c.cs.s = append(c.cs.s, "km")
	return (GeoradiusbymemberUnitKm)(c)
}

func (c GeoradiusbymemberRadius) Ft() GeoradiusbymemberUnitFt {
	c.cs.s = append(c.cs.s, "ft")
	return (GeoradiusbymemberUnitFt)(c)
}

func (c GeoradiusbymemberRadius) Mi() GeoradiusbymemberUnitMi {
	c.cs.s = append(c.cs.s, "mi")
	return (GeoradiusbymemberUnitMi)(c)
}

type GeoradiusbymemberRo Completed

func (b Builder) GeoradiusbymemberRo() (c GeoradiusbymemberRo) {
	c = GeoradiusbymemberRo{cs: get(), ks: b.ks, cf: readonly}
	c.cs.s = append(c.cs.s, "GEORADIUSBYMEMBER_RO")
	return c
}

func (c GeoradiusbymemberRo) Key(key string) GeoradiusbymemberRoKey {
	if c.ks&NoSlot == NoSlot {
		c.ks = NoSlot | slot(key)
	} else {
		c.ks = check(c.ks, slot(key))
	}
	c.cs.s = append(c.cs.s, key)
	return (GeoradiusbymemberRoKey)(c)
}

type GeoradiusbymemberRoCountAny Completed

func (c GeoradiusbymemberRoCountAny) Asc() GeoradiusbymemberRoOrderAsc {
	c.cs.s = append(c.cs.s, "ASC")
	return (GeoradiusbymemberRoOrderAsc)(c)
}

func (c GeoradiusbymemberRoCountAny) Desc() GeoradiusbymemberRoOrderDesc {
	c.cs.s = append(c.cs.s, "DESC")
	return (GeoradiusbymemberRoOrderDesc)(c)
}

func (c GeoradiusbymemberRoCountAny) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

func (c GeoradiusbymemberRoCountAny) Cache() Cacheable {
	c.cs.Build()
	return Cacheable(c)
}

type GeoradiusbymemberRoCountCount Completed

func (c GeoradiusbymemberRoCountCount) Any() GeoradiusbymemberRoCountAny {
	c.cs.s = append(c.cs.s, "ANY")
	return (GeoradiusbymemberRoCountAny)(c)
}

func (c GeoradiusbymemberRoCountCount) Asc() GeoradiusbymemberRoOrderAsc {
	c.cs.s = append(c.cs.s, "ASC")
	return (GeoradiusbymemberRoOrderAsc)(c)
}

func (c GeoradiusbymemberRoCountCount) Desc() GeoradiusbymemberRoOrderDesc {
	c.cs.s = append(c.cs.s, "DESC")
	return (GeoradiusbymemberRoOrderDesc)(c)
}

func (c GeoradiusbymemberRoCountCount) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

func (c GeoradiusbymemberRoCountCount) Cache() Cacheable {
	c.cs.Build()
	return Cacheable(c)
}

type GeoradiusbymemberRoKey Completed

func (c GeoradiusbymemberRoKey) Member(member string) GeoradiusbymemberRoMember {
	c.cs.s = append(c.cs.s, member)
	return (GeoradiusbymemberRoMember)(c)
}

type GeoradiusbymemberRoMember Completed

func (c GeoradiusbymemberRoMember) Radius(radius float64) GeoradiusbymemberRoRadius {
	c.cs.s = append(c.cs.s, strconv.FormatFloat(radius, 'f', -1, 64))
	return (GeoradiusbymemberRoRadius)(c)
}

type GeoradiusbymemberRoOrderAsc Completed

func (c GeoradiusbymemberRoOrderAsc) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

func (c GeoradiusbymemberRoOrderAsc) Cache() Cacheable {
	c.cs.Build()
	return Cacheable(c)
}

type GeoradiusbymemberRoOrderDesc Completed

func (c GeoradiusbymemberRoOrderDesc) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

func (c GeoradiusbymemberRoOrderDesc) Cache() Cacheable {
	c.cs.Build()
	return Cacheable(c)
}

type GeoradiusbymemberRoRadius Completed

func (c GeoradiusbymemberRoRadius) M() GeoradiusbymemberRoUnitM {
	c.cs.s = append(c.cs.s, "m")
	return (GeoradiusbymemberRoUnitM)(c)
}

func (c GeoradiusbymemberRoRadius) Km() GeoradiusbymemberRoUnitKm {
	c.cs.s = append(c.cs.s, "km")
	return (GeoradiusbymemberRoUnitKm)(c)
}

func (c GeoradiusbymemberRoRadius) Ft() GeoradiusbymemberRoUnitFt {
	c.cs.s = append(c.cs.s, "ft")
	return (GeoradiusbymemberRoUnitFt)(c)
}

func (c GeoradiusbymemberRoRadius) Mi() GeoradiusbymemberRoUnitMi {
	c.cs.s = append(c.cs.s, "mi")
	return (GeoradiusbymemberRoUnitMi)(c)
}

type GeoradiusbymemberRoUnitFt Completed

func (c GeoradiusbymemberRoUnitFt) Withcoord() GeoradiusbymemberRoWithcoord {
	c.cs.s = append(c.cs.s, "WITHCOORD")
	return (GeoradiusbymemberRoWithcoord)(c)
}

func (c GeoradiusbymemberRoUnitFt) Withdist() GeoradiusbymemberRoWithdist {
	c.cs.s = append(c.cs.s, "WITHDIST")
	return (GeoradiusbymemberRoWithdist)(c)
}

func (c GeoradiusbymemberRoUnitFt) Withhash() GeoradiusbymemberRoWithhash {
	c.cs.s = append(c.cs.s, "WITHHASH")
	return (GeoradiusbymemberRoWithhash)(c)
}

func (c GeoradiusbymemberRoUnitFt) Count(count int64) GeoradiusbymemberRoCountCount {
	c.cs.s = append(c.cs.s, "COUNT", strconv.FormatInt(count, 10))
	return (GeoradiusbymemberRoCountCount)(c)
}

func (c GeoradiusbymemberRoUnitFt) Asc() GeoradiusbymemberRoOrderAsc {
	c.cs.s = append(c.cs.s, "ASC")
	return (GeoradiusbymemberRoOrderAsc)(c)
}

func (c GeoradiusbymemberRoUnitFt) Desc() GeoradiusbymemberRoOrderDesc {
	c.cs.s = append(c.cs.s, "DESC")
	return (GeoradiusbymemberRoOrderDesc)(c)
}

func (c GeoradiusbymemberRoUnitFt) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

func (c GeoradiusbymemberRoUnitFt) Cache() Cacheable {
	c.cs.Build()
	return Cacheable(c)
}

type GeoradiusbymemberRoUnitKm Completed

func (c GeoradiusbymemberRoUnitKm) Withcoord() GeoradiusbymemberRoWithcoord {
	c.cs.s = append(c.cs.s, "WITHCOORD")
	return (GeoradiusbymemberRoWithcoord)(c)
}

func (c GeoradiusbymemberRoUnitKm) Withdist() GeoradiusbymemberRoWithdist {
	c.cs.s = append(c.cs.s, "WITHDIST")
	return (GeoradiusbymemberRoWithdist)(c)
}

func (c GeoradiusbymemberRoUnitKm) Withhash() GeoradiusbymemberRoWithhash {
	c.cs.s = append(c.cs.s, "WITHHASH")
	return (GeoradiusbymemberRoWithhash)(c)
}

func (c GeoradiusbymemberRoUnitKm) Count(count int64) GeoradiusbymemberRoCountCount {
	c.cs.s = append(c.cs.s, "COUNT", strconv.FormatInt(count, 10))
	return (GeoradiusbymemberRoCountCount)(c)
}

func (c GeoradiusbymemberRoUnitKm) Asc() GeoradiusbymemberRoOrderAsc {
	c.cs.s = append(c.cs.s, "ASC")
	return (GeoradiusbymemberRoOrderAsc)(c)
}

func (c GeoradiusbymemberRoUnitKm) Desc() GeoradiusbymemberRoOrderDesc {
	c.cs.s = append(c.cs.s, "DESC")
	return (GeoradiusbymemberRoOrderDesc)(c)
}

func (c GeoradiusbymemberRoUnitKm) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

func (c GeoradiusbymemberRoUnitKm) Cache() Cacheable {
	c.cs.Build()
	return Cacheable(c)
}

type GeoradiusbymemberRoUnitM Completed

func (c GeoradiusbymemberRoUnitM) Withcoord() GeoradiusbymemberRoWithcoord {
	c.cs.s = append(c.cs.s, "WITHCOORD")
	return (GeoradiusbymemberRoWithcoord)(c)
}

func (c GeoradiusbymemberRoUnitM) Withdist() GeoradiusbymemberRoWithdist {
	c.cs.s = append(c.cs.s, "WITHDIST")
	return (GeoradiusbymemberRoWithdist)(c)
}

func (c GeoradiusbymemberRoUnitM) Withhash() GeoradiusbymemberRoWithhash {
	c.cs.s = append(c.cs.s, "WITHHASH")
	return (GeoradiusbymemberRoWithhash)(c)
}

func (c GeoradiusbymemberRoUnitM) Count(count int64) GeoradiusbymemberRoCountCount {
	c.cs.s = append(c.cs.s, "COUNT", strconv.FormatInt(count, 10))
	return (GeoradiusbymemberRoCountCount)(c)
}

func (c GeoradiusbymemberRoUnitM) Asc() GeoradiusbymemberRoOrderAsc {
	c.cs.s = append(c.cs.s, "ASC")
	return (GeoradiusbymemberRoOrderAsc)(c)
}

func (c GeoradiusbymemberRoUnitM) Desc() GeoradiusbymemberRoOrderDesc {
	c.cs.s = append(c.cs.s, "DESC")
	return (GeoradiusbymemberRoOrderDesc)(c)
}

func (c GeoradiusbymemberRoUnitM) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

func (c GeoradiusbymemberRoUnitM) Cache() Cacheable {
	c.cs.Build()
	return Cacheable(c)
}

type GeoradiusbymemberRoUnitMi Completed

func (c GeoradiusbymemberRoUnitMi) Withcoord() GeoradiusbymemberRoWithcoord {
	c.cs.s = append(c.cs.s, "WITHCOORD")
	return (GeoradiusbymemberRoWithcoord)(c)
}

func (c GeoradiusbymemberRoUnitMi) Withdist() GeoradiusbymemberRoWithdist {
	c.cs.s = append(c.cs.s, "WITHDIST")
	return (GeoradiusbymemberRoWithdist)(c)
}

func (c GeoradiusbymemberRoUnitMi) Withhash() GeoradiusbymemberRoWithhash {
	c.cs.s = append(c.cs.s, "WITHHASH")
	return (GeoradiusbymemberRoWithhash)(c)
}

func (c GeoradiusbymemberRoUnitMi) Count(count int64) GeoradiusbymemberRoCountCount {
	c.cs.s = append(c.cs.s, "COUNT", strconv.FormatInt(count, 10))
	return (GeoradiusbymemberRoCountCount)(c)
}

func (c GeoradiusbymemberRoUnitMi) Asc() GeoradiusbymemberRoOrderAsc {
	c.cs.s = append(c.cs.s, "ASC")
	return (GeoradiusbymemberRoOrderAsc)(c)
}

func (c GeoradiusbymemberRoUnitMi) Desc() GeoradiusbymemberRoOrderDesc {
	c.cs.s = append(c.cs.s, "DESC")
	return (GeoradiusbymemberRoOrderDesc)(c)
}

func (c GeoradiusbymemberRoUnitMi) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

func (c GeoradiusbymemberRoUnitMi) Cache() Cacheable {
	c.cs.Build()
	return Cacheable(c)
}

type GeoradiusbymemberRoWithcoord Completed

func (c GeoradiusbymemberRoWithcoord) Withdist() GeoradiusbymemberRoWithdist {
	c.cs.s = append(c.cs.s, "WITHDIST")
	return (GeoradiusbymemberRoWithdist)(c)
}

func (c GeoradiusbymemberRoWithcoord) Withhash() GeoradiusbymemberRoWithhash {
	c.cs.s = append(c.cs.s, "WITHHASH")
	return (GeoradiusbymemberRoWithhash)(c)
}

func (c GeoradiusbymemberRoWithcoord) Count(count int64) GeoradiusbymemberRoCountCount {
	c.cs.s = append(c.cs.s, "COUNT", strconv.FormatInt(count, 10))
	return (GeoradiusbymemberRoCountCount)(c)
}

func (c GeoradiusbymemberRoWithcoord) Asc() GeoradiusbymemberRoOrderAsc {
	c.cs.s = append(c.cs.s, "ASC")
	return (GeoradiusbymemberRoOrderAsc)(c)
}

func (c GeoradiusbymemberRoWithcoord) Desc() GeoradiusbymemberRoOrderDesc {
	c.cs.s = append(c.cs.s, "DESC")
	return (GeoradiusbymemberRoOrderDesc)(c)
}

func (c GeoradiusbymemberRoWithcoord) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

func (c GeoradiusbymemberRoWithcoord) Cache() Cacheable {
	c.cs.Build()
	return Cacheable(c)
}

type GeoradiusbymemberRoWithdist Completed

func (c GeoradiusbymemberRoWithdist) Withhash() GeoradiusbymemberRoWithhash {
	c.cs.s = append(c.cs.s, "WITHHASH")
	return (GeoradiusbymemberRoWithhash)(c)
}

func (c GeoradiusbymemberRoWithdist) Count(count int64) GeoradiusbymemberRoCountCount {
	c.cs.s = append(c.cs.s, "COUNT", strconv.FormatInt(count, 10))
	return (GeoradiusbymemberRoCountCount)(c)
}

func (c GeoradiusbymemberRoWithdist) Asc() GeoradiusbymemberRoOrderAsc {
	c.cs.s = append(c.cs.s, "ASC")
	return (GeoradiusbymemberRoOrderAsc)(c)
}

func (c GeoradiusbymemberRoWithdist) Desc() GeoradiusbymemberRoOrderDesc {
	c.cs.s = append(c.cs.s, "DESC")
	return (GeoradiusbymemberRoOrderDesc)(c)
}

func (c GeoradiusbymemberRoWithdist) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

func (c GeoradiusbymemberRoWithdist) Cache() Cacheable {
	c.cs.Build()
	return Cacheable(c)
}

type GeoradiusbymemberRoWithhash Completed

func (c GeoradiusbymemberRoWithhash) Count(count int64) GeoradiusbymemberRoCountCount {
	c.cs.s = append(c.cs.s, "COUNT", strconv.FormatInt(count, 10))
	return (GeoradiusbymemberRoCountCount)(c)
}

func (c GeoradiusbymemberRoWithhash) Asc() GeoradiusbymemberRoOrderAsc {
	c.cs.s = append(c.cs.s, "ASC")
	return (GeoradiusbymemberRoOrderAsc)(c)
}

func (c GeoradiusbymemberRoWithhash) Desc() GeoradiusbymemberRoOrderDesc {
	c.cs.s = append(c.cs.s, "DESC")
	return (GeoradiusbymemberRoOrderDesc)(c)
}

func (c GeoradiusbymemberRoWithhash) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

func (c GeoradiusbymemberRoWithhash) Cache() Cacheable {
	c.cs.Build()
	return Cacheable(c)
}

type GeoradiusbymemberStore Completed

func (c GeoradiusbymemberStore) Storedist(key string) GeoradiusbymemberStoredist {
	if c.ks&NoSlot == NoSlot {
		c.ks = NoSlot | slot(key)
	} else {
		c.ks = check(c.ks, slot(key))
	}
	c.cs.s = append(c.cs.s, "STOREDIST", key)
	return (GeoradiusbymemberStoredist)(c)
}

func (c GeoradiusbymemberStore) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type GeoradiusbymemberStoredist Completed

func (c GeoradiusbymemberStoredist) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type GeoradiusbymemberUnitFt Completed

func (c GeoradiusbymemberUnitFt) Withcoord() GeoradiusbymemberWithcoord {
	c.cs.s = append(c.cs.s, "WITHCOORD")
	return (GeoradiusbymemberWithcoord)(c)
}

func (c GeoradiusbymemberUnitFt) Withdist() GeoradiusbymemberWithdist {
	c.cs.s = append(c.cs.s, "WITHDIST")
	return (GeoradiusbymemberWithdist)(c)
}

func (c GeoradiusbymemberUnitFt) Withhash() GeoradiusbymemberWithhash {
	c.cs.s = append(c.cs.s, "WITHHASH")
	return (GeoradiusbymemberWithhash)(c)
}

func (c GeoradiusbymemberUnitFt) Count(count int64) GeoradiusbymemberCountCount {
	c.cs.s = append(c.cs.s, "COUNT", strconv.FormatInt(count, 10))
	return (GeoradiusbymemberCountCount)(c)
}

func (c GeoradiusbymemberUnitFt) Asc() GeoradiusbymemberOrderAsc {
	c.cs.s = append(c.cs.s, "ASC")
	return (GeoradiusbymemberOrderAsc)(c)
}

func (c GeoradiusbymemberUnitFt) Desc() GeoradiusbymemberOrderDesc {
	c.cs.s = append(c.cs.s, "DESC")
	return (GeoradiusbymemberOrderDesc)(c)
}

func (c GeoradiusbymemberUnitFt) Store(key string) GeoradiusbymemberStore {
	if c.ks&NoSlot == NoSlot {
		c.ks = NoSlot | slot(key)
	} else {
		c.ks = check(c.ks, slot(key))
	}
	c.cs.s = append(c.cs.s, "STORE", key)
	return (GeoradiusbymemberStore)(c)
}

func (c GeoradiusbymemberUnitFt) Storedist(key string) GeoradiusbymemberStoredist {
	if c.ks&NoSlot == NoSlot {
		c.ks = NoSlot | slot(key)
	} else {
		c.ks = check(c.ks, slot(key))
	}
	c.cs.s = append(c.cs.s, "STOREDIST", key)
	return (GeoradiusbymemberStoredist)(c)
}

func (c GeoradiusbymemberUnitFt) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type GeoradiusbymemberUnitKm Completed

func (c GeoradiusbymemberUnitKm) Withcoord() GeoradiusbymemberWithcoord {
	c.cs.s = append(c.cs.s, "WITHCOORD")
	return (GeoradiusbymemberWithcoord)(c)
}

func (c GeoradiusbymemberUnitKm) Withdist() GeoradiusbymemberWithdist {
	c.cs.s = append(c.cs.s, "WITHDIST")
	return (GeoradiusbymemberWithdist)(c)
}

func (c GeoradiusbymemberUnitKm) Withhash() GeoradiusbymemberWithhash {
	c.cs.s = append(c.cs.s, "WITHHASH")
	return (GeoradiusbymemberWithhash)(c)
}

func (c GeoradiusbymemberUnitKm) Count(count int64) GeoradiusbymemberCountCount {
	c.cs.s = append(c.cs.s, "COUNT", strconv.FormatInt(count, 10))
	return (GeoradiusbymemberCountCount)(c)
}

func (c GeoradiusbymemberUnitKm) Asc() GeoradiusbymemberOrderAsc {
	c.cs.s = append(c.cs.s, "ASC")
	return (GeoradiusbymemberOrderAsc)(c)
}

func (c GeoradiusbymemberUnitKm) Desc() GeoradiusbymemberOrderDesc {
	c.cs.s = append(c.cs.s, "DESC")
	return (GeoradiusbymemberOrderDesc)(c)
}

func (c GeoradiusbymemberUnitKm) Store(key string) GeoradiusbymemberStore {
	if c.ks&NoSlot == NoSlot {
		c.ks = NoSlot | slot(key)
	} else {
		c.ks = check(c.ks, slot(key))
	}
	c.cs.s = append(c.cs.s, "STORE", key)
	return (GeoradiusbymemberStore)(c)
}

func (c GeoradiusbymemberUnitKm) Storedist(key string) GeoradiusbymemberStoredist {
	if c.ks&NoSlot == NoSlot {
		c.ks = NoSlot | slot(key)
	} else {
		c.ks = check(c.ks, slot(key))
	}
	c.cs.s = append(c.cs.s, "STOREDIST", key)
	return (GeoradiusbymemberStoredist)(c)
}

func (c GeoradiusbymemberUnitKm) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type GeoradiusbymemberUnitM Completed

func (c GeoradiusbymemberUnitM) Withcoord() GeoradiusbymemberWithcoord {
	c.cs.s = append(c.cs.s, "WITHCOORD")
	return (GeoradiusbymemberWithcoord)(c)
}

func (c GeoradiusbymemberUnitM) Withdist() GeoradiusbymemberWithdist {
	c.cs.s = append(c.cs.s, "WITHDIST")
	return (GeoradiusbymemberWithdist)(c)
}

func (c GeoradiusbymemberUnitM) Withhash() GeoradiusbymemberWithhash {
	c.cs.s = append(c.cs.s, "WITHHASH")
	return (GeoradiusbymemberWithhash)(c)
}

func (c GeoradiusbymemberUnitM) Count(count int64) GeoradiusbymemberCountCount {
	c.cs.s = append(c.cs.s, "COUNT", strconv.FormatInt(count, 10))
	return (GeoradiusbymemberCountCount)(c)
}

func (c GeoradiusbymemberUnitM) Asc() GeoradiusbymemberOrderAsc {
	c.cs.s = append(c.cs.s, "ASC")
	return (GeoradiusbymemberOrderAsc)(c)
}

func (c GeoradiusbymemberUnitM) Desc() GeoradiusbymemberOrderDesc {
	c.cs.s = append(c.cs.s, "DESC")
	return (GeoradiusbymemberOrderDesc)(c)
}

func (c GeoradiusbymemberUnitM) Store(key string) GeoradiusbymemberStore {
	if c.ks&NoSlot == NoSlot {
		c.ks = NoSlot | slot(key)
	} else {
		c.ks = check(c.ks, slot(key))
	}
	c.cs.s = append(c.cs.s, "STORE", key)
	return (GeoradiusbymemberStore)(c)
}

func (c GeoradiusbymemberUnitM) Storedist(key string) GeoradiusbymemberStoredist {
	if c.ks&NoSlot == NoSlot {
		c.ks = NoSlot | slot(key)
	} else {
		c.ks = check(c.ks, slot(key))
	}
	c.cs.s = append(c.cs.s, "STOREDIST", key)
	return (GeoradiusbymemberStoredist)(c)
}

func (c GeoradiusbymemberUnitM) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type GeoradiusbymemberUnitMi Completed

func (c GeoradiusbymemberUnitMi) Withcoord() GeoradiusbymemberWithcoord {
	c.cs.s = append(c.cs.s, "WITHCOORD")
	return (GeoradiusbymemberWithcoord)(c)
}

func (c GeoradiusbymemberUnitMi) Withdist() GeoradiusbymemberWithdist {
	c.cs.s = append(c.cs.s, "WITHDIST")
	return (GeoradiusbymemberWithdist)(c)
}

func (c GeoradiusbymemberUnitMi) Withhash() GeoradiusbymemberWithhash {
	c.cs.s = append(c.cs.s, "WITHHASH")
	return (GeoradiusbymemberWithhash)(c)
}

func (c GeoradiusbymemberUnitMi) Count(count int64) GeoradiusbymemberCountCount {
	c.cs.s = append(c.cs.s, "COUNT", strconv.FormatInt(count, 10))
	return (GeoradiusbymemberCountCount)(c)
}

func (c GeoradiusbymemberUnitMi) Asc() GeoradiusbymemberOrderAsc {
	c.cs.s = append(c.cs.s, "ASC")
	return (GeoradiusbymemberOrderAsc)(c)
}

func (c GeoradiusbymemberUnitMi) Desc() GeoradiusbymemberOrderDesc {
	c.cs.s = append(c.cs.s, "DESC")
	return (GeoradiusbymemberOrderDesc)(c)
}

func (c GeoradiusbymemberUnitMi) Store(key string) GeoradiusbymemberStore {
	if c.ks&NoSlot == NoSlot {
		c.ks = NoSlot | slot(key)
	} else {
		c.ks = check(c.ks, slot(key))
	}
	c.cs.s = append(c.cs.s, "STORE", key)
	return (GeoradiusbymemberStore)(c)
}

func (c GeoradiusbymemberUnitMi) Storedist(key string) GeoradiusbymemberStoredist {
	if c.ks&NoSlot == NoSlot {
		c.ks = NoSlot | slot(key)
	} else {
		c.ks = check(c.ks, slot(key))
	}
	c.cs.s = append(c.cs.s, "STOREDIST", key)
	return (GeoradiusbymemberStoredist)(c)
}

func (c GeoradiusbymemberUnitMi) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type GeoradiusbymemberWithcoord Completed

func (c GeoradiusbymemberWithcoord) Withdist() GeoradiusbymemberWithdist {
	c.cs.s = append(c.cs.s, "WITHDIST")
	return (GeoradiusbymemberWithdist)(c)
}

func (c GeoradiusbymemberWithcoord) Withhash() GeoradiusbymemberWithhash {
	c.cs.s = append(c.cs.s, "WITHHASH")
	return (GeoradiusbymemberWithhash)(c)
}

func (c GeoradiusbymemberWithcoord) Count(count int64) GeoradiusbymemberCountCount {
	c.cs.s = append(c.cs.s, "COUNT", strconv.FormatInt(count, 10))
	return (GeoradiusbymemberCountCount)(c)
}

func (c GeoradiusbymemberWithcoord) Asc() GeoradiusbymemberOrderAsc {
	c.cs.s = append(c.cs.s, "ASC")
	return (GeoradiusbymemberOrderAsc)(c)
}

func (c GeoradiusbymemberWithcoord) Desc() GeoradiusbymemberOrderDesc {
	c.cs.s = append(c.cs.s, "DESC")
	return (GeoradiusbymemberOrderDesc)(c)
}

func (c GeoradiusbymemberWithcoord) Store(key string) GeoradiusbymemberStore {
	if c.ks&NoSlot == NoSlot {
		c.ks = NoSlot | slot(key)
	} else {
		c.ks = check(c.ks, slot(key))
	}
	c.cs.s = append(c.cs.s, "STORE", key)
	return (GeoradiusbymemberStore)(c)
}

func (c GeoradiusbymemberWithcoord) Storedist(key string) GeoradiusbymemberStoredist {
	if c.ks&NoSlot == NoSlot {
		c.ks = NoSlot | slot(key)
	} else {
		c.ks = check(c.ks, slot(key))
	}
	c.cs.s = append(c.cs.s, "STOREDIST", key)
	return (GeoradiusbymemberStoredist)(c)
}

func (c GeoradiusbymemberWithcoord) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type GeoradiusbymemberWithdist Completed

func (c GeoradiusbymemberWithdist) Withhash() GeoradiusbymemberWithhash {
	c.cs.s = append(c.cs.s, "WITHHASH")
	return (GeoradiusbymemberWithhash)(c)
}

func (c GeoradiusbymemberWithdist) Count(count int64) GeoradiusbymemberCountCount {
	c.cs.s = append(c.cs.s, "COUNT", strconv.FormatInt(count, 10))
	return (GeoradiusbymemberCountCount)(c)
}

func (c GeoradiusbymemberWithdist) Asc() GeoradiusbymemberOrderAsc {
	c.cs.s = append(c.cs.s, "ASC")
	return (GeoradiusbymemberOrderAsc)(c)
}

func (c GeoradiusbymemberWithdist) Desc() GeoradiusbymemberOrderDesc {
	c.cs.s = append(c.cs.s, "DESC")
	return (GeoradiusbymemberOrderDesc)(c)
}

func (c GeoradiusbymemberWithdist) Store(key string) GeoradiusbymemberStore {
	if c.ks&NoSlot == NoSlot {
		c.ks = NoSlot | slot(key)
	} else {
		c.ks = check(c.ks, slot(key))
	}
	c.cs.s = append(c.cs.s, "STORE", key)
	return (GeoradiusbymemberStore)(c)
}

func (c GeoradiusbymemberWithdist) Storedist(key string) GeoradiusbymemberStoredist {
	if c.ks&NoSlot == NoSlot {
		c.ks = NoSlot | slot(key)
	} else {
		c.ks = check(c.ks, slot(key))
	}
	c.cs.s = append(c.cs.s, "STOREDIST", key)
	return (GeoradiusbymemberStoredist)(c)
}

func (c GeoradiusbymemberWithdist) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type GeoradiusbymemberWithhash Completed

func (c GeoradiusbymemberWithhash) Count(count int64) GeoradiusbymemberCountCount {
	c.cs.s = append(c.cs.s, "COUNT", strconv.FormatInt(count, 10))
	return (GeoradiusbymemberCountCount)(c)
}

func (c GeoradiusbymemberWithhash) Asc() GeoradiusbymemberOrderAsc {
	c.cs.s = append(c.cs.s, "ASC")
	return (GeoradiusbymemberOrderAsc)(c)
}

func (c GeoradiusbymemberWithhash) Desc() GeoradiusbymemberOrderDesc {
	c.cs.s = append(c.cs.s, "DESC")
	return (GeoradiusbymemberOrderDesc)(c)
}

func (c GeoradiusbymemberWithhash) Store(key string) GeoradiusbymemberStore {
	if c.ks&NoSlot == NoSlot {
		c.ks = NoSlot | slot(key)
	} else {
		c.ks = check(c.ks, slot(key))
	}
	c.cs.s = append(c.cs.s, "STORE", key)
	return (GeoradiusbymemberStore)(c)
}

func (c GeoradiusbymemberWithhash) Storedist(key string) GeoradiusbymemberStoredist {
	if c.ks&NoSlot == NoSlot {
		c.ks = NoSlot | slot(key)
	} else {
		c.ks = check(c.ks, slot(key))
	}
	c.cs.s = append(c.cs.s, "STOREDIST", key)
	return (GeoradiusbymemberStoredist)(c)
}

func (c GeoradiusbymemberWithhash) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type Geosearch Completed

func (b Builder) Geosearch() (c Geosearch) {
	c = Geosearch{cs: get(), ks: b.ks, cf: readonly}
	c.cs.s = append(c.cs.s, "GEOSEARCH")
	return c
}

func (c Geosearch) Key(key string) GeosearchKey {
	if c.ks&NoSlot == NoSlot {
		c.ks = NoSlot | slot(key)
	} else {
		c.ks = check(c.ks, slot(key))
	}
	c.cs.s = append(c.cs.s, key)
	return (GeosearchKey)(c)
}

type GeosearchCircleBoxBybox Completed

func (c GeosearchCircleBoxBybox) Height(height float64) GeosearchCircleBoxHeight {
	c.cs.s = append(c.cs.s, strconv.FormatFloat(height, 'f', -1, 64))
	return (GeosearchCircleBoxHeight)(c)
}

type GeosearchCircleBoxHeight Completed

func (c GeosearchCircleBoxHeight) M() GeosearchCircleBoxUnitM {
	c.cs.s = append(c.cs.s, "m")
	return (GeosearchCircleBoxUnitM)(c)
}

func (c GeosearchCircleBoxHeight) Km() GeosearchCircleBoxUnitKm {
	c.cs.s = append(c.cs.s, "km")
	return (GeosearchCircleBoxUnitKm)(c)
}

func (c GeosearchCircleBoxHeight) Ft() GeosearchCircleBoxUnitFt {
	c.cs.s = append(c.cs.s, "ft")
	return (GeosearchCircleBoxUnitFt)(c)
}

func (c GeosearchCircleBoxHeight) Mi() GeosearchCircleBoxUnitMi {
	c.cs.s = append(c.cs.s, "mi")
	return (GeosearchCircleBoxUnitMi)(c)
}

type GeosearchCircleBoxUnitFt Completed

func (c GeosearchCircleBoxUnitFt) Asc() GeosearchOrderAsc {
	c.cs.s = append(c.cs.s, "ASC")
	return (GeosearchOrderAsc)(c)
}

func (c GeosearchCircleBoxUnitFt) Desc() GeosearchOrderDesc {
	c.cs.s = append(c.cs.s, "DESC")
	return (GeosearchOrderDesc)(c)
}

func (c GeosearchCircleBoxUnitFt) Count(count int64) GeosearchCountCount {
	c.cs.s = append(c.cs.s, "COUNT", strconv.FormatInt(count, 10))
	return (GeosearchCountCount)(c)
}

func (c GeosearchCircleBoxUnitFt) Withcoord() GeosearchWithcoord {
	c.cs.s = append(c.cs.s, "WITHCOORD")
	return (GeosearchWithcoord)(c)
}

func (c GeosearchCircleBoxUnitFt) Withdist() GeosearchWithdist {
	c.cs.s = append(c.cs.s, "WITHDIST")
	return (GeosearchWithdist)(c)
}

func (c GeosearchCircleBoxUnitFt) Withhash() GeosearchWithhash {
	c.cs.s = append(c.cs.s, "WITHHASH")
	return (GeosearchWithhash)(c)
}

func (c GeosearchCircleBoxUnitFt) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

func (c GeosearchCircleBoxUnitFt) Cache() Cacheable {
	c.cs.Build()
	return Cacheable(c)
}

type GeosearchCircleBoxUnitKm Completed

func (c GeosearchCircleBoxUnitKm) Asc() GeosearchOrderAsc {
	c.cs.s = append(c.cs.s, "ASC")
	return (GeosearchOrderAsc)(c)
}

func (c GeosearchCircleBoxUnitKm) Desc() GeosearchOrderDesc {
	c.cs.s = append(c.cs.s, "DESC")
	return (GeosearchOrderDesc)(c)
}

func (c GeosearchCircleBoxUnitKm) Count(count int64) GeosearchCountCount {
	c.cs.s = append(c.cs.s, "COUNT", strconv.FormatInt(count, 10))
	return (GeosearchCountCount)(c)
}

func (c GeosearchCircleBoxUnitKm) Withcoord() GeosearchWithcoord {
	c.cs.s = append(c.cs.s, "WITHCOORD")
	return (GeosearchWithcoord)(c)
}

func (c GeosearchCircleBoxUnitKm) Withdist() GeosearchWithdist {
	c.cs.s = append(c.cs.s, "WITHDIST")
	return (GeosearchWithdist)(c)
}

func (c GeosearchCircleBoxUnitKm) Withhash() GeosearchWithhash {
	c.cs.s = append(c.cs.s, "WITHHASH")
	return (GeosearchWithhash)(c)
}

func (c GeosearchCircleBoxUnitKm) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

func (c GeosearchCircleBoxUnitKm) Cache() Cacheable {
	c.cs.Build()
	return Cacheable(c)
}

type GeosearchCircleBoxUnitM Completed

func (c GeosearchCircleBoxUnitM) Asc() GeosearchOrderAsc {
	c.cs.s = append(c.cs.s, "ASC")
	return (GeosearchOrderAsc)(c)
}

func (c GeosearchCircleBoxUnitM) Desc() GeosearchOrderDesc {
	c.cs.s = append(c.cs.s, "DESC")
	return (GeosearchOrderDesc)(c)
}

func (c GeosearchCircleBoxUnitM) Count(count int64) GeosearchCountCount {
	c.cs.s = append(c.cs.s, "COUNT", strconv.FormatInt(count, 10))
	return (GeosearchCountCount)(c)
}

func (c GeosearchCircleBoxUnitM) Withcoord() GeosearchWithcoord {
	c.cs.s = append(c.cs.s, "WITHCOORD")
	return (GeosearchWithcoord)(c)
}

func (c GeosearchCircleBoxUnitM) Withdist() GeosearchWithdist {
	c.cs.s = append(c.cs.s, "WITHDIST")
	return (GeosearchWithdist)(c)
}

func (c GeosearchCircleBoxUnitM) Withhash() GeosearchWithhash {
	c.cs.s = append(c.cs.s, "WITHHASH")
	return (GeosearchWithhash)(c)
}

func (c GeosearchCircleBoxUnitM) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

func (c GeosearchCircleBoxUnitM) Cache() Cacheable {
	c.cs.Build()
	return Cacheable(c)
}

type GeosearchCircleBoxUnitMi Completed

func (c GeosearchCircleBoxUnitMi) Asc() GeosearchOrderAsc {
	c.cs.s = append(c.cs.s, "ASC")
	return (GeosearchOrderAsc)(c)
}

func (c GeosearchCircleBoxUnitMi) Desc() GeosearchOrderDesc {
	c.cs.s = append(c.cs.s, "DESC")
	return (GeosearchOrderDesc)(c)
}

func (c GeosearchCircleBoxUnitMi) Count(count int64) GeosearchCountCount {
	c.cs.s = append(c.cs.s, "COUNT", strconv.FormatInt(count, 10))
	return (GeosearchCountCount)(c)
}

func (c GeosearchCircleBoxUnitMi) Withcoord() GeosearchWithcoord {
	c.cs.s = append(c.cs.s, "WITHCOORD")
	return (GeosearchWithcoord)(c)
}

func (c GeosearchCircleBoxUnitMi) Withdist() GeosearchWithdist {
	c.cs.s = append(c.cs.s, "WITHDIST")
	return (GeosearchWithdist)(c)
}

func (c GeosearchCircleBoxUnitMi) Withhash() GeosearchWithhash {
	c.cs.s = append(c.cs.s, "WITHHASH")
	return (GeosearchWithhash)(c)
}

func (c GeosearchCircleBoxUnitMi) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

func (c GeosearchCircleBoxUnitMi) Cache() Cacheable {
	c.cs.Build()
	return Cacheable(c)
}

type GeosearchCircleCircleByradius Completed

func (c GeosearchCircleCircleByradius) M() GeosearchCircleCircleUnitM {
	c.cs.s = append(c.cs.s, "m")
	return (GeosearchCircleCircleUnitM)(c)
}

func (c GeosearchCircleCircleByradius) Km() GeosearchCircleCircleUnitKm {
	c.cs.s = append(c.cs.s, "km")
	return (GeosearchCircleCircleUnitKm)(c)
}

func (c GeosearchCircleCircleByradius) Ft() GeosearchCircleCircleUnitFt {
	c.cs.s = append(c.cs.s, "ft")
	return (GeosearchCircleCircleUnitFt)(c)
}

func (c GeosearchCircleCircleByradius) Mi() GeosearchCircleCircleUnitMi {
	c.cs.s = append(c.cs.s, "mi")
	return (GeosearchCircleCircleUnitMi)(c)
}

type GeosearchCircleCircleUnitFt Completed

func (c GeosearchCircleCircleUnitFt) Bybox(width float64) GeosearchCircleBoxBybox {
	c.cs.s = append(c.cs.s, "BYBOX", strconv.FormatFloat(width, 'f', -1, 64))
	return (GeosearchCircleBoxBybox)(c)
}

func (c GeosearchCircleCircleUnitFt) Asc() GeosearchOrderAsc {
	c.cs.s = append(c.cs.s, "ASC")
	return (GeosearchOrderAsc)(c)
}

func (c GeosearchCircleCircleUnitFt) Desc() GeosearchOrderDesc {
	c.cs.s = append(c.cs.s, "DESC")
	return (GeosearchOrderDesc)(c)
}

func (c GeosearchCircleCircleUnitFt) Count(count int64) GeosearchCountCount {
	c.cs.s = append(c.cs.s, "COUNT", strconv.FormatInt(count, 10))
	return (GeosearchCountCount)(c)
}

func (c GeosearchCircleCircleUnitFt) Withcoord() GeosearchWithcoord {
	c.cs.s = append(c.cs.s, "WITHCOORD")
	return (GeosearchWithcoord)(c)
}

func (c GeosearchCircleCircleUnitFt) Withdist() GeosearchWithdist {
	c.cs.s = append(c.cs.s, "WITHDIST")
	return (GeosearchWithdist)(c)
}

func (c GeosearchCircleCircleUnitFt) Withhash() GeosearchWithhash {
	c.cs.s = append(c.cs.s, "WITHHASH")
	return (GeosearchWithhash)(c)
}

func (c GeosearchCircleCircleUnitFt) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

func (c GeosearchCircleCircleUnitFt) Cache() Cacheable {
	c.cs.Build()
	return Cacheable(c)
}

type GeosearchCircleCircleUnitKm Completed

func (c GeosearchCircleCircleUnitKm) Bybox(width float64) GeosearchCircleBoxBybox {
	c.cs.s = append(c.cs.s, "BYBOX", strconv.FormatFloat(width, 'f', -1, 64))
	return (GeosearchCircleBoxBybox)(c)
}

func (c GeosearchCircleCircleUnitKm) Asc() GeosearchOrderAsc {
	c.cs.s = append(c.cs.s, "ASC")
	return (GeosearchOrderAsc)(c)
}

func (c GeosearchCircleCircleUnitKm) Desc() GeosearchOrderDesc {
	c.cs.s = append(c.cs.s, "DESC")
	return (GeosearchOrderDesc)(c)
}

func (c GeosearchCircleCircleUnitKm) Count(count int64) GeosearchCountCount {
	c.cs.s = append(c.cs.s, "COUNT", strconv.FormatInt(count, 10))
	return (GeosearchCountCount)(c)
}

func (c GeosearchCircleCircleUnitKm) Withcoord() GeosearchWithcoord {
	c.cs.s = append(c.cs.s, "WITHCOORD")
	return (GeosearchWithcoord)(c)
}

func (c GeosearchCircleCircleUnitKm) Withdist() GeosearchWithdist {
	c.cs.s = append(c.cs.s, "WITHDIST")
	return (GeosearchWithdist)(c)
}

func (c GeosearchCircleCircleUnitKm) Withhash() GeosearchWithhash {
	c.cs.s = append(c.cs.s, "WITHHASH")
	return (GeosearchWithhash)(c)
}

func (c GeosearchCircleCircleUnitKm) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

func (c GeosearchCircleCircleUnitKm) Cache() Cacheable {
	c.cs.Build()
	return Cacheable(c)
}

type GeosearchCircleCircleUnitM Completed

func (c GeosearchCircleCircleUnitM) Bybox(width float64) GeosearchCircleBoxBybox {
	c.cs.s = append(c.cs.s, "BYBOX", strconv.FormatFloat(width, 'f', -1, 64))
	return (GeosearchCircleBoxBybox)(c)
}

func (c GeosearchCircleCircleUnitM) Asc() GeosearchOrderAsc {
	c.cs.s = append(c.cs.s, "ASC")
	return (GeosearchOrderAsc)(c)
}

func (c GeosearchCircleCircleUnitM) Desc() GeosearchOrderDesc {
	c.cs.s = append(c.cs.s, "DESC")
	return (GeosearchOrderDesc)(c)
}

func (c GeosearchCircleCircleUnitM) Count(count int64) GeosearchCountCount {
	c.cs.s = append(c.cs.s, "COUNT", strconv.FormatInt(count, 10))
	return (GeosearchCountCount)(c)
}

func (c GeosearchCircleCircleUnitM) Withcoord() GeosearchWithcoord {
	c.cs.s = append(c.cs.s, "WITHCOORD")
	return (GeosearchWithcoord)(c)
}

func (c GeosearchCircleCircleUnitM) Withdist() GeosearchWithdist {
	c.cs.s = append(c.cs.s, "WITHDIST")
	return (GeosearchWithdist)(c)
}

func (c GeosearchCircleCircleUnitM) Withhash() GeosearchWithhash {
	c.cs.s = append(c.cs.s, "WITHHASH")
	return (GeosearchWithhash)(c)
}

func (c GeosearchCircleCircleUnitM) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

func (c GeosearchCircleCircleUnitM) Cache() Cacheable {
	c.cs.Build()
	return Cacheable(c)
}

type GeosearchCircleCircleUnitMi Completed

func (c GeosearchCircleCircleUnitMi) Bybox(width float64) GeosearchCircleBoxBybox {
	c.cs.s = append(c.cs.s, "BYBOX", strconv.FormatFloat(width, 'f', -1, 64))
	return (GeosearchCircleBoxBybox)(c)
}

func (c GeosearchCircleCircleUnitMi) Asc() GeosearchOrderAsc {
	c.cs.s = append(c.cs.s, "ASC")
	return (GeosearchOrderAsc)(c)
}

func (c GeosearchCircleCircleUnitMi) Desc() GeosearchOrderDesc {
	c.cs.s = append(c.cs.s, "DESC")
	return (GeosearchOrderDesc)(c)
}

func (c GeosearchCircleCircleUnitMi) Count(count int64) GeosearchCountCount {
	c.cs.s = append(c.cs.s, "COUNT", strconv.FormatInt(count, 10))
	return (GeosearchCountCount)(c)
}

func (c GeosearchCircleCircleUnitMi) Withcoord() GeosearchWithcoord {
	c.cs.s = append(c.cs.s, "WITHCOORD")
	return (GeosearchWithcoord)(c)
}

func (c GeosearchCircleCircleUnitMi) Withdist() GeosearchWithdist {
	c.cs.s = append(c.cs.s, "WITHDIST")
	return (GeosearchWithdist)(c)
}

func (c GeosearchCircleCircleUnitMi) Withhash() GeosearchWithhash {
	c.cs.s = append(c.cs.s, "WITHHASH")
	return (GeosearchWithhash)(c)
}

func (c GeosearchCircleCircleUnitMi) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

func (c GeosearchCircleCircleUnitMi) Cache() Cacheable {
	c.cs.Build()
	return Cacheable(c)
}

type GeosearchCountAny Completed

func (c GeosearchCountAny) Withcoord() GeosearchWithcoord {
	c.cs.s = append(c.cs.s, "WITHCOORD")
	return (GeosearchWithcoord)(c)
}

func (c GeosearchCountAny) Withdist() GeosearchWithdist {
	c.cs.s = append(c.cs.s, "WITHDIST")
	return (GeosearchWithdist)(c)
}

func (c GeosearchCountAny) Withhash() GeosearchWithhash {
	c.cs.s = append(c.cs.s, "WITHHASH")
	return (GeosearchWithhash)(c)
}

func (c GeosearchCountAny) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

func (c GeosearchCountAny) Cache() Cacheable {
	c.cs.Build()
	return Cacheable(c)
}

type GeosearchCountCount Completed

func (c GeosearchCountCount) Any() GeosearchCountAny {
	c.cs.s = append(c.cs.s, "ANY")
	return (GeosearchCountAny)(c)
}

func (c GeosearchCountCount) Withcoord() GeosearchWithcoord {
	c.cs.s = append(c.cs.s, "WITHCOORD")
	return (GeosearchWithcoord)(c)
}

func (c GeosearchCountCount) Withdist() GeosearchWithdist {
	c.cs.s = append(c.cs.s, "WITHDIST")
	return (GeosearchWithdist)(c)
}

func (c GeosearchCountCount) Withhash() GeosearchWithhash {
	c.cs.s = append(c.cs.s, "WITHHASH")
	return (GeosearchWithhash)(c)
}

func (c GeosearchCountCount) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

func (c GeosearchCountCount) Cache() Cacheable {
	c.cs.Build()
	return Cacheable(c)
}

type GeosearchFrommemberFromlonlat Completed

func (c GeosearchFrommemberFromlonlat) Byradius(radius float64) GeosearchCircleCircleByradius {
	c.cs.s = append(c.cs.s, "BYRADIUS", strconv.FormatFloat(radius, 'f', -1, 64))
	return (GeosearchCircleCircleByradius)(c)
}

func (c GeosearchFrommemberFromlonlat) Bybox(width float64) GeosearchCircleBoxBybox {
	c.cs.s = append(c.cs.s, "BYBOX", strconv.FormatFloat(width, 'f', -1, 64))
	return (GeosearchCircleBoxBybox)(c)
}

type GeosearchFrommemberFrommember Completed

func (c GeosearchFrommemberFrommember) Fromlonlat(longitude float64, latitude float64) GeosearchFrommemberFromlonlat {
	c.cs.s = append(c.cs.s, "FROMLONLAT", strconv.FormatFloat(longitude, 'f', -1, 64), strconv.FormatFloat(latitude, 'f', -1, 64))
	return (GeosearchFrommemberFromlonlat)(c)
}

func (c GeosearchFrommemberFrommember) Byradius(radius float64) GeosearchCircleCircleByradius {
	c.cs.s = append(c.cs.s, "BYRADIUS", strconv.FormatFloat(radius, 'f', -1, 64))
	return (GeosearchCircleCircleByradius)(c)
}

func (c GeosearchFrommemberFrommember) Bybox(width float64) GeosearchCircleBoxBybox {
	c.cs.s = append(c.cs.s, "BYBOX", strconv.FormatFloat(width, 'f', -1, 64))
	return (GeosearchCircleBoxBybox)(c)
}

type GeosearchKey Completed

func (c GeosearchKey) Frommember(member string) GeosearchFrommemberFrommember {
	c.cs.s = append(c.cs.s, "FROMMEMBER", member)
	return (GeosearchFrommemberFrommember)(c)
}

func (c GeosearchKey) Fromlonlat(longitude float64, latitude float64) GeosearchFrommemberFromlonlat {
	c.cs.s = append(c.cs.s, "FROMLONLAT", strconv.FormatFloat(longitude, 'f', -1, 64), strconv.FormatFloat(latitude, 'f', -1, 64))
	return (GeosearchFrommemberFromlonlat)(c)
}

type GeosearchOrderAsc Completed

func (c GeosearchOrderAsc) Count(count int64) GeosearchCountCount {
	c.cs.s = append(c.cs.s, "COUNT", strconv.FormatInt(count, 10))
	return (GeosearchCountCount)(c)
}

func (c GeosearchOrderAsc) Withcoord() GeosearchWithcoord {
	c.cs.s = append(c.cs.s, "WITHCOORD")
	return (GeosearchWithcoord)(c)
}

func (c GeosearchOrderAsc) Withdist() GeosearchWithdist {
	c.cs.s = append(c.cs.s, "WITHDIST")
	return (GeosearchWithdist)(c)
}

func (c GeosearchOrderAsc) Withhash() GeosearchWithhash {
	c.cs.s = append(c.cs.s, "WITHHASH")
	return (GeosearchWithhash)(c)
}

func (c GeosearchOrderAsc) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

func (c GeosearchOrderAsc) Cache() Cacheable {
	c.cs.Build()
	return Cacheable(c)
}

type GeosearchOrderDesc Completed

func (c GeosearchOrderDesc) Count(count int64) GeosearchCountCount {
	c.cs.s = append(c.cs.s, "COUNT", strconv.FormatInt(count, 10))
	return (GeosearchCountCount)(c)
}

func (c GeosearchOrderDesc) Withcoord() GeosearchWithcoord {
	c.cs.s = append(c.cs.s, "WITHCOORD")
	return (GeosearchWithcoord)(c)
}

func (c GeosearchOrderDesc) Withdist() GeosearchWithdist {
	c.cs.s = append(c.cs.s, "WITHDIST")
	return (GeosearchWithdist)(c)
}

func (c GeosearchOrderDesc) Withhash() GeosearchWithhash {
	c.cs.s = append(c.cs.s, "WITHHASH")
	return (GeosearchWithhash)(c)
}

func (c GeosearchOrderDesc) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

func (c GeosearchOrderDesc) Cache() Cacheable {
	c.cs.Build()
	return Cacheable(c)
}

type GeosearchWithcoord Completed

func (c GeosearchWithcoord) Withdist() GeosearchWithdist {
	c.cs.s = append(c.cs.s, "WITHDIST")
	return (GeosearchWithdist)(c)
}

func (c GeosearchWithcoord) Withhash() GeosearchWithhash {
	c.cs.s = append(c.cs.s, "WITHHASH")
	return (GeosearchWithhash)(c)
}

func (c GeosearchWithcoord) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

func (c GeosearchWithcoord) Cache() Cacheable {
	c.cs.Build()
	return Cacheable(c)
}

type GeosearchWithdist Completed

func (c GeosearchWithdist) Withhash() GeosearchWithhash {
	c.cs.s = append(c.cs.s, "WITHHASH")
	return (GeosearchWithhash)(c)
}

func (c GeosearchWithdist) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

func (c GeosearchWithdist) Cache() Cacheable {
	c.cs.Build()
	return Cacheable(c)
}

type GeosearchWithhash Completed

func (c GeosearchWithhash) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

func (c GeosearchWithhash) Cache() Cacheable {
	c.cs.Build()
	return Cacheable(c)
}

type Geosearchstore Completed

func (b Builder) Geosearchstore() (c Geosearchstore) {
	c = Geosearchstore{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "GEOSEARCHSTORE")
	return c
}

func (c Geosearchstore) Destination(destination string) GeosearchstoreDestination {
	if c.ks&NoSlot == NoSlot {
		c.ks = NoSlot | slot(destination)
	} else {
		c.ks = check(c.ks, slot(destination))
	}
	c.cs.s = append(c.cs.s, destination)
	return (GeosearchstoreDestination)(c)
}

type GeosearchstoreCircleBoxBybox Completed

func (c GeosearchstoreCircleBoxBybox) Height(height float64) GeosearchstoreCircleBoxHeight {
	c.cs.s = append(c.cs.s, strconv.FormatFloat(height, 'f', -1, 64))
	return (GeosearchstoreCircleBoxHeight)(c)
}

type GeosearchstoreCircleBoxHeight Completed

func (c GeosearchstoreCircleBoxHeight) M() GeosearchstoreCircleBoxUnitM {
	c.cs.s = append(c.cs.s, "m")
	return (GeosearchstoreCircleBoxUnitM)(c)
}

func (c GeosearchstoreCircleBoxHeight) Km() GeosearchstoreCircleBoxUnitKm {
	c.cs.s = append(c.cs.s, "km")
	return (GeosearchstoreCircleBoxUnitKm)(c)
}

func (c GeosearchstoreCircleBoxHeight) Ft() GeosearchstoreCircleBoxUnitFt {
	c.cs.s = append(c.cs.s, "ft")
	return (GeosearchstoreCircleBoxUnitFt)(c)
}

func (c GeosearchstoreCircleBoxHeight) Mi() GeosearchstoreCircleBoxUnitMi {
	c.cs.s = append(c.cs.s, "mi")
	return (GeosearchstoreCircleBoxUnitMi)(c)
}

type GeosearchstoreCircleBoxUnitFt Completed

func (c GeosearchstoreCircleBoxUnitFt) Asc() GeosearchstoreOrderAsc {
	c.cs.s = append(c.cs.s, "ASC")
	return (GeosearchstoreOrderAsc)(c)
}

func (c GeosearchstoreCircleBoxUnitFt) Desc() GeosearchstoreOrderDesc {
	c.cs.s = append(c.cs.s, "DESC")
	return (GeosearchstoreOrderDesc)(c)
}

func (c GeosearchstoreCircleBoxUnitFt) Count(count int64) GeosearchstoreCountCount {
	c.cs.s = append(c.cs.s, "COUNT", strconv.FormatInt(count, 10))
	return (GeosearchstoreCountCount)(c)
}

func (c GeosearchstoreCircleBoxUnitFt) Storedist() GeosearchstoreStoredist {
	c.cs.s = append(c.cs.s, "STOREDIST")
	return (GeosearchstoreStoredist)(c)
}

func (c GeosearchstoreCircleBoxUnitFt) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type GeosearchstoreCircleBoxUnitKm Completed

func (c GeosearchstoreCircleBoxUnitKm) Asc() GeosearchstoreOrderAsc {
	c.cs.s = append(c.cs.s, "ASC")
	return (GeosearchstoreOrderAsc)(c)
}

func (c GeosearchstoreCircleBoxUnitKm) Desc() GeosearchstoreOrderDesc {
	c.cs.s = append(c.cs.s, "DESC")
	return (GeosearchstoreOrderDesc)(c)
}

func (c GeosearchstoreCircleBoxUnitKm) Count(count int64) GeosearchstoreCountCount {
	c.cs.s = append(c.cs.s, "COUNT", strconv.FormatInt(count, 10))
	return (GeosearchstoreCountCount)(c)
}

func (c GeosearchstoreCircleBoxUnitKm) Storedist() GeosearchstoreStoredist {
	c.cs.s = append(c.cs.s, "STOREDIST")
	return (GeosearchstoreStoredist)(c)
}

func (c GeosearchstoreCircleBoxUnitKm) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type GeosearchstoreCircleBoxUnitM Completed

func (c GeosearchstoreCircleBoxUnitM) Asc() GeosearchstoreOrderAsc {
	c.cs.s = append(c.cs.s, "ASC")
	return (GeosearchstoreOrderAsc)(c)
}

func (c GeosearchstoreCircleBoxUnitM) Desc() GeosearchstoreOrderDesc {
	c.cs.s = append(c.cs.s, "DESC")
	return (GeosearchstoreOrderDesc)(c)
}

func (c GeosearchstoreCircleBoxUnitM) Count(count int64) GeosearchstoreCountCount {
	c.cs.s = append(c.cs.s, "COUNT", strconv.FormatInt(count, 10))
	return (GeosearchstoreCountCount)(c)
}

func (c GeosearchstoreCircleBoxUnitM) Storedist() GeosearchstoreStoredist {
	c.cs.s = append(c.cs.s, "STOREDIST")
	return (GeosearchstoreStoredist)(c)
}

func (c GeosearchstoreCircleBoxUnitM) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type GeosearchstoreCircleBoxUnitMi Completed

func (c GeosearchstoreCircleBoxUnitMi) Asc() GeosearchstoreOrderAsc {
	c.cs.s = append(c.cs.s, "ASC")
	return (GeosearchstoreOrderAsc)(c)
}

func (c GeosearchstoreCircleBoxUnitMi) Desc() GeosearchstoreOrderDesc {
	c.cs.s = append(c.cs.s, "DESC")
	return (GeosearchstoreOrderDesc)(c)
}

func (c GeosearchstoreCircleBoxUnitMi) Count(count int64) GeosearchstoreCountCount {
	c.cs.s = append(c.cs.s, "COUNT", strconv.FormatInt(count, 10))
	return (GeosearchstoreCountCount)(c)
}

func (c GeosearchstoreCircleBoxUnitMi) Storedist() GeosearchstoreStoredist {
	c.cs.s = append(c.cs.s, "STOREDIST")
	return (GeosearchstoreStoredist)(c)
}

func (c GeosearchstoreCircleBoxUnitMi) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type GeosearchstoreCircleCircleByradius Completed

func (c GeosearchstoreCircleCircleByradius) M() GeosearchstoreCircleCircleUnitM {
	c.cs.s = append(c.cs.s, "m")
	return (GeosearchstoreCircleCircleUnitM)(c)
}

func (c GeosearchstoreCircleCircleByradius) Km() GeosearchstoreCircleCircleUnitKm {
	c.cs.s = append(c.cs.s, "km")
	return (GeosearchstoreCircleCircleUnitKm)(c)
}

func (c GeosearchstoreCircleCircleByradius) Ft() GeosearchstoreCircleCircleUnitFt {
	c.cs.s = append(c.cs.s, "ft")
	return (GeosearchstoreCircleCircleUnitFt)(c)
}

func (c GeosearchstoreCircleCircleByradius) Mi() GeosearchstoreCircleCircleUnitMi {
	c.cs.s = append(c.cs.s, "mi")
	return (GeosearchstoreCircleCircleUnitMi)(c)
}

type GeosearchstoreCircleCircleUnitFt Completed

func (c GeosearchstoreCircleCircleUnitFt) Bybox(width float64) GeosearchstoreCircleBoxBybox {
	c.cs.s = append(c.cs.s, "BYBOX", strconv.FormatFloat(width, 'f', -1, 64))
	return (GeosearchstoreCircleBoxBybox)(c)
}

func (c GeosearchstoreCircleCircleUnitFt) Asc() GeosearchstoreOrderAsc {
	c.cs.s = append(c.cs.s, "ASC")
	return (GeosearchstoreOrderAsc)(c)
}

func (c GeosearchstoreCircleCircleUnitFt) Desc() GeosearchstoreOrderDesc {
	c.cs.s = append(c.cs.s, "DESC")
	return (GeosearchstoreOrderDesc)(c)
}

func (c GeosearchstoreCircleCircleUnitFt) Count(count int64) GeosearchstoreCountCount {
	c.cs.s = append(c.cs.s, "COUNT", strconv.FormatInt(count, 10))
	return (GeosearchstoreCountCount)(c)
}

func (c GeosearchstoreCircleCircleUnitFt) Storedist() GeosearchstoreStoredist {
	c.cs.s = append(c.cs.s, "STOREDIST")
	return (GeosearchstoreStoredist)(c)
}

func (c GeosearchstoreCircleCircleUnitFt) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type GeosearchstoreCircleCircleUnitKm Completed

func (c GeosearchstoreCircleCircleUnitKm) Bybox(width float64) GeosearchstoreCircleBoxBybox {
	c.cs.s = append(c.cs.s, "BYBOX", strconv.FormatFloat(width, 'f', -1, 64))
	return (GeosearchstoreCircleBoxBybox)(c)
}

func (c GeosearchstoreCircleCircleUnitKm) Asc() GeosearchstoreOrderAsc {
	c.cs.s = append(c.cs.s, "ASC")
	return (GeosearchstoreOrderAsc)(c)
}

func (c GeosearchstoreCircleCircleUnitKm) Desc() GeosearchstoreOrderDesc {
	c.cs.s = append(c.cs.s, "DESC")
	return (GeosearchstoreOrderDesc)(c)
}

func (c GeosearchstoreCircleCircleUnitKm) Count(count int64) GeosearchstoreCountCount {
	c.cs.s = append(c.cs.s, "COUNT", strconv.FormatInt(count, 10))
	return (GeosearchstoreCountCount)(c)
}

func (c GeosearchstoreCircleCircleUnitKm) Storedist() GeosearchstoreStoredist {
	c.cs.s = append(c.cs.s, "STOREDIST")
	return (GeosearchstoreStoredist)(c)
}

func (c GeosearchstoreCircleCircleUnitKm) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type GeosearchstoreCircleCircleUnitM Completed

func (c GeosearchstoreCircleCircleUnitM) Bybox(width float64) GeosearchstoreCircleBoxBybox {
	c.cs.s = append(c.cs.s, "BYBOX", strconv.FormatFloat(width, 'f', -1, 64))
	return (GeosearchstoreCircleBoxBybox)(c)
}

func (c GeosearchstoreCircleCircleUnitM) Asc() GeosearchstoreOrderAsc {
	c.cs.s = append(c.cs.s, "ASC")
	return (GeosearchstoreOrderAsc)(c)
}

func (c GeosearchstoreCircleCircleUnitM) Desc() GeosearchstoreOrderDesc {
	c.cs.s = append(c.cs.s, "DESC")
	return (GeosearchstoreOrderDesc)(c)
}

func (c GeosearchstoreCircleCircleUnitM) Count(count int64) GeosearchstoreCountCount {
	c.cs.s = append(c.cs.s, "COUNT", strconv.FormatInt(count, 10))
	return (GeosearchstoreCountCount)(c)
}

func (c GeosearchstoreCircleCircleUnitM) Storedist() GeosearchstoreStoredist {
	c.cs.s = append(c.cs.s, "STOREDIST")
	return (GeosearchstoreStoredist)(c)
}

func (c GeosearchstoreCircleCircleUnitM) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type GeosearchstoreCircleCircleUnitMi Completed

func (c GeosearchstoreCircleCircleUnitMi) Bybox(width float64) GeosearchstoreCircleBoxBybox {
	c.cs.s = append(c.cs.s, "BYBOX", strconv.FormatFloat(width, 'f', -1, 64))
	return (GeosearchstoreCircleBoxBybox)(c)
}

func (c GeosearchstoreCircleCircleUnitMi) Asc() GeosearchstoreOrderAsc {
	c.cs.s = append(c.cs.s, "ASC")
	return (GeosearchstoreOrderAsc)(c)
}

func (c GeosearchstoreCircleCircleUnitMi) Desc() GeosearchstoreOrderDesc {
	c.cs.s = append(c.cs.s, "DESC")
	return (GeosearchstoreOrderDesc)(c)
}

func (c GeosearchstoreCircleCircleUnitMi) Count(count int64) GeosearchstoreCountCount {
	c.cs.s = append(c.cs.s, "COUNT", strconv.FormatInt(count, 10))
	return (GeosearchstoreCountCount)(c)
}

func (c GeosearchstoreCircleCircleUnitMi) Storedist() GeosearchstoreStoredist {
	c.cs.s = append(c.cs.s, "STOREDIST")
	return (GeosearchstoreStoredist)(c)
}

func (c GeosearchstoreCircleCircleUnitMi) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type GeosearchstoreCountAny Completed

func (c GeosearchstoreCountAny) Storedist() GeosearchstoreStoredist {
	c.cs.s = append(c.cs.s, "STOREDIST")
	return (GeosearchstoreStoredist)(c)
}

func (c GeosearchstoreCountAny) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type GeosearchstoreCountCount Completed

func (c GeosearchstoreCountCount) Any() GeosearchstoreCountAny {
	c.cs.s = append(c.cs.s, "ANY")
	return (GeosearchstoreCountAny)(c)
}

func (c GeosearchstoreCountCount) Storedist() GeosearchstoreStoredist {
	c.cs.s = append(c.cs.s, "STOREDIST")
	return (GeosearchstoreStoredist)(c)
}

func (c GeosearchstoreCountCount) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type GeosearchstoreDestination Completed

func (c GeosearchstoreDestination) Source(source string) GeosearchstoreSource {
	if c.ks&NoSlot == NoSlot {
		c.ks = NoSlot | slot(source)
	} else {
		c.ks = check(c.ks, slot(source))
	}
	c.cs.s = append(c.cs.s, source)
	return (GeosearchstoreSource)(c)
}

type GeosearchstoreFrommemberFromlonlat Completed

func (c GeosearchstoreFrommemberFromlonlat) Byradius(radius float64) GeosearchstoreCircleCircleByradius {
	c.cs.s = append(c.cs.s, "BYRADIUS", strconv.FormatFloat(radius, 'f', -1, 64))
	return (GeosearchstoreCircleCircleByradius)(c)
}

func (c GeosearchstoreFrommemberFromlonlat) Bybox(width float64) GeosearchstoreCircleBoxBybox {
	c.cs.s = append(c.cs.s, "BYBOX", strconv.FormatFloat(width, 'f', -1, 64))
	return (GeosearchstoreCircleBoxBybox)(c)
}

type GeosearchstoreFrommemberFrommember Completed

func (c GeosearchstoreFrommemberFrommember) Fromlonlat(longitude float64, latitude float64) GeosearchstoreFrommemberFromlonlat {
	c.cs.s = append(c.cs.s, "FROMLONLAT", strconv.FormatFloat(longitude, 'f', -1, 64), strconv.FormatFloat(latitude, 'f', -1, 64))
	return (GeosearchstoreFrommemberFromlonlat)(c)
}

func (c GeosearchstoreFrommemberFrommember) Byradius(radius float64) GeosearchstoreCircleCircleByradius {
	c.cs.s = append(c.cs.s, "BYRADIUS", strconv.FormatFloat(radius, 'f', -1, 64))
	return (GeosearchstoreCircleCircleByradius)(c)
}

func (c GeosearchstoreFrommemberFrommember) Bybox(width float64) GeosearchstoreCircleBoxBybox {
	c.cs.s = append(c.cs.s, "BYBOX", strconv.FormatFloat(width, 'f', -1, 64))
	return (GeosearchstoreCircleBoxBybox)(c)
}

type GeosearchstoreOrderAsc Completed

func (c GeosearchstoreOrderAsc) Count(count int64) GeosearchstoreCountCount {
	c.cs.s = append(c.cs.s, "COUNT", strconv.FormatInt(count, 10))
	return (GeosearchstoreCountCount)(c)
}

func (c GeosearchstoreOrderAsc) Storedist() GeosearchstoreStoredist {
	c.cs.s = append(c.cs.s, "STOREDIST")
	return (GeosearchstoreStoredist)(c)
}

func (c GeosearchstoreOrderAsc) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type GeosearchstoreOrderDesc Completed

func (c GeosearchstoreOrderDesc) Count(count int64) GeosearchstoreCountCount {
	c.cs.s = append(c.cs.s, "COUNT", strconv.FormatInt(count, 10))
	return (GeosearchstoreCountCount)(c)
}

func (c GeosearchstoreOrderDesc) Storedist() GeosearchstoreStoredist {
	c.cs.s = append(c.cs.s, "STOREDIST")
	return (GeosearchstoreStoredist)(c)
}

func (c GeosearchstoreOrderDesc) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type GeosearchstoreSource Completed

func (c GeosearchstoreSource) Frommember(member string) GeosearchstoreFrommemberFrommember {
	c.cs.s = append(c.cs.s, "FROMMEMBER", member)
	return (GeosearchstoreFrommemberFrommember)(c)
}

func (c GeosearchstoreSource) Fromlonlat(longitude float64, latitude float64) GeosearchstoreFrommemberFromlonlat {
	c.cs.s = append(c.cs.s, "FROMLONLAT", strconv.FormatFloat(longitude, 'f', -1, 64), strconv.FormatFloat(latitude, 'f', -1, 64))
	return (GeosearchstoreFrommemberFromlonlat)(c)
}

type GeosearchstoreStoredist Completed

func (c GeosearchstoreStoredist) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type Get Completed

func (b Builder) Get() (c Get) {
	c = Get{cs: get(), ks: b.ks, cf: readonly}
	c.cs.s = append(c.cs.s, "GET")
	return c
}

func (c Get) Key(key string) GetKey {
	if c.ks&NoSlot == NoSlot {
		c.ks = NoSlot | slot(key)
	} else {
		c.ks = check(c.ks, slot(key))
	}
	c.cs.s = append(c.cs.s, key)
	return (GetKey)(c)
}

type GetKey Completed

func (c GetKey) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

func (c GetKey) Cache() Cacheable {
	c.cs.Build()
	return Cacheable(c)
}

type Getbit Completed

func (b Builder) Getbit() (c Getbit) {
	c = Getbit{cs: get(), ks: b.ks, cf: readonly}
	c.cs.s = append(c.cs.s, "GETBIT")
	return c
}

func (c Getbit) Key(key string) GetbitKey {
	if c.ks&NoSlot == NoSlot {
		c.ks = NoSlot | slot(key)
	} else {
		c.ks = check(c.ks, slot(key))
	}
	c.cs.s = append(c.cs.s, key)
	return (GetbitKey)(c)
}

type GetbitKey Completed

func (c GetbitKey) Offset(offset int64) GetbitOffset {
	c.cs.s = append(c.cs.s, strconv.FormatInt(offset, 10))
	return (GetbitOffset)(c)
}

type GetbitOffset Completed

func (c GetbitOffset) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

func (c GetbitOffset) Cache() Cacheable {
	c.cs.Build()
	return Cacheable(c)
}

type Getdel Completed

func (b Builder) Getdel() (c Getdel) {
	c = Getdel{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "GETDEL")
	return c
}

func (c Getdel) Key(key string) GetdelKey {
	if c.ks&NoSlot == NoSlot {
		c.ks = NoSlot | slot(key)
	} else {
		c.ks = check(c.ks, slot(key))
	}
	c.cs.s = append(c.cs.s, key)
	return (GetdelKey)(c)
}

type GetdelKey Completed

func (c GetdelKey) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type Getex Completed

func (b Builder) Getex() (c Getex) {
	c = Getex{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "GETEX")
	return c
}

func (c Getex) Key(key string) GetexKey {
	if c.ks&NoSlot == NoSlot {
		c.ks = NoSlot | slot(key)
	} else {
		c.ks = check(c.ks, slot(key))
	}
	c.cs.s = append(c.cs.s, key)
	return (GetexKey)(c)
}

type GetexExpirationExSeconds Completed

func (c GetexExpirationExSeconds) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type GetexExpirationExatTimestamp Completed

func (c GetexExpirationExatTimestamp) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type GetexExpirationPersist Completed

func (c GetexExpirationPersist) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type GetexExpirationPxMilliseconds Completed

func (c GetexExpirationPxMilliseconds) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type GetexExpirationPxatMillisecondsTimestamp Completed

func (c GetexExpirationPxatMillisecondsTimestamp) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type GetexKey Completed

func (c GetexKey) ExSeconds(seconds int64) GetexExpirationExSeconds {
	c.cs.s = append(c.cs.s, "EX", strconv.FormatInt(seconds, 10))
	return (GetexExpirationExSeconds)(c)
}

func (c GetexKey) PxMilliseconds(milliseconds int64) GetexExpirationPxMilliseconds {
	c.cs.s = append(c.cs.s, "PX", strconv.FormatInt(milliseconds, 10))
	return (GetexExpirationPxMilliseconds)(c)
}

func (c GetexKey) ExatTimestamp(timestamp int64) GetexExpirationExatTimestamp {
	c.cs.s = append(c.cs.s, "EXAT", strconv.FormatInt(timestamp, 10))
	return (GetexExpirationExatTimestamp)(c)
}

func (c GetexKey) PxatMillisecondsTimestamp(millisecondsTimestamp int64) GetexExpirationPxatMillisecondsTimestamp {
	c.cs.s = append(c.cs.s, "PXAT", strconv.FormatInt(millisecondsTimestamp, 10))
	return (GetexExpirationPxatMillisecondsTimestamp)(c)
}

func (c GetexKey) Persist() GetexExpirationPersist {
	c.cs.s = append(c.cs.s, "PERSIST")
	return (GetexExpirationPersist)(c)
}

func (c GetexKey) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type Getrange Completed

func (b Builder) Getrange() (c Getrange) {
	c = Getrange{cs: get(), ks: b.ks, cf: readonly}
	c.cs.s = append(c.cs.s, "GETRANGE")
	return c
}

func (c Getrange) Key(key string) GetrangeKey {
	if c.ks&NoSlot == NoSlot {
		c.ks = NoSlot | slot(key)
	} else {
		c.ks = check(c.ks, slot(key))
	}
	c.cs.s = append(c.cs.s, key)
	return (GetrangeKey)(c)
}

type GetrangeEnd Completed

func (c GetrangeEnd) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

func (c GetrangeEnd) Cache() Cacheable {
	c.cs.Build()
	return Cacheable(c)
}

type GetrangeKey Completed

func (c GetrangeKey) Start(start int64) GetrangeStart {
	c.cs.s = append(c.cs.s, strconv.FormatInt(start, 10))
	return (GetrangeStart)(c)
}

type GetrangeStart Completed

func (c GetrangeStart) End(end int64) GetrangeEnd {
	c.cs.s = append(c.cs.s, strconv.FormatInt(end, 10))
	return (GetrangeEnd)(c)
}

type Getset Completed

func (b Builder) Getset() (c Getset) {
	c = Getset{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "GETSET")
	return c
}

func (c Getset) Key(key string) GetsetKey {
	if c.ks&NoSlot == NoSlot {
		c.ks = NoSlot | slot(key)
	} else {
		c.ks = check(c.ks, slot(key))
	}
	c.cs.s = append(c.cs.s, key)
	return (GetsetKey)(c)
}

type GetsetKey Completed

func (c GetsetKey) Value(value string) GetsetValue {
	c.cs.s = append(c.cs.s, value)
	return (GetsetValue)(c)
}

type GetsetValue Completed

func (c GetsetValue) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type GraphConfigGet Completed

func (b Builder) GraphConfigGet() (c GraphConfigGet) {
	c = GraphConfigGet{cs: get(), ks: b.ks, cf: readonly}
	c.cs.s = append(c.cs.s, "GRAPH.CONFIG", "GET")
	return c
}

func (c GraphConfigGet) Name(name string) GraphConfigGetName {
	c.cs.s = append(c.cs.s, name)
	return (GraphConfigGetName)(c)
}

type GraphConfigGetName Completed

func (c GraphConfigGetName) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type GraphConfigSet Completed

func (b Builder) GraphConfigSet() (c GraphConfigSet) {
	c = GraphConfigSet{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "GRAPH.CONFIG", "SET")
	return c
}

func (c GraphConfigSet) Name(name string) GraphConfigSetName {
	c.cs.s = append(c.cs.s, name)
	return (GraphConfigSetName)(c)
}

type GraphConfigSetName Completed

func (c GraphConfigSetName) Value(value string) GraphConfigSetValue {
	c.cs.s = append(c.cs.s, value)
	return (GraphConfigSetValue)(c)
}

type GraphConfigSetValue Completed

func (c GraphConfigSetValue) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type GraphDelete Completed

func (b Builder) GraphDelete() (c GraphDelete) {
	c = GraphDelete{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "GRAPH.DELETE")
	return c
}

func (c GraphDelete) Graph(graph string) GraphDeleteGraph {
	if c.ks&NoSlot == NoSlot {
		c.ks = NoSlot | slot(graph)
	} else {
		c.ks = check(c.ks, slot(graph))
	}
	c.cs.s = append(c.cs.s, graph)
	return (GraphDeleteGraph)(c)
}

type GraphDeleteGraph Completed

func (c GraphDeleteGraph) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type GraphExplain Completed

func (b Builder) GraphExplain() (c GraphExplain) {
	c = GraphExplain{cs: get(), ks: b.ks, cf: readonly}
	c.cs.s = append(c.cs.s, "GRAPH.EXPLAIN")
	return c
}

func (c GraphExplain) Graph(graph string) GraphExplainGraph {
	if c.ks&NoSlot == NoSlot {
		c.ks = NoSlot | slot(graph)
	} else {
		c.ks = check(c.ks, slot(graph))
	}
	c.cs.s = append(c.cs.s, graph)
	return (GraphExplainGraph)(c)
}

type GraphExplainGraph Completed

func (c GraphExplainGraph) Query(query string) GraphExplainQuery {
	c.cs.s = append(c.cs.s, query)
	return (GraphExplainQuery)(c)
}

type GraphExplainQuery Completed

func (c GraphExplainQuery) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type GraphList Completed

func (b Builder) GraphList() (c GraphList) {
	c = GraphList{cs: get(), ks: b.ks, cf: readonly}
	c.cs.s = append(c.cs.s, "GRAPH.LIST")
	return c
}

func (c GraphList) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type GraphProfile Completed

func (b Builder) GraphProfile() (c GraphProfile) {
	c = GraphProfile{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "GRAPH.PROFILE")
	return c
}

func (c GraphProfile) Graph(graph string) GraphProfileGraph {
	if c.ks&NoSlot == NoSlot {
		c.ks = NoSlot | slot(graph)
	} else {
		c.ks = check(c.ks, slot(graph))
	}
	c.cs.s = append(c.cs.s, graph)
	return (GraphProfileGraph)(c)
}

type GraphProfileGraph Completed

func (c GraphProfileGraph) Query(query string) GraphProfileQuery {
	c.cs.s = append(c.cs.s, query)
	return (GraphProfileQuery)(c)
}

type GraphProfileQuery Completed

func (c GraphProfileQuery) Timeout(timeout int64) GraphProfileTimeout {
	c.cs.s = append(c.cs.s, "TIMEOUT", strconv.FormatInt(timeout, 10))
	return (GraphProfileTimeout)(c)
}

func (c GraphProfileQuery) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type GraphProfileTimeout Completed

func (c GraphProfileTimeout) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type GraphQuery Completed

func (b Builder) GraphQuery() (c GraphQuery) {
	c = GraphQuery{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "GRAPH.QUERY")
	return c
}

func (c GraphQuery) Graph(graph string) GraphQueryGraph {
	if c.ks&NoSlot == NoSlot {
		c.ks = NoSlot | slot(graph)
	} else {
		c.ks = check(c.ks, slot(graph))
	}
	c.cs.s = append(c.cs.s, graph)
	return (GraphQueryGraph)(c)
}

type GraphQueryGraph Completed

func (c GraphQueryGraph) Query(query string) GraphQueryQuery {
	c.cs.s = append(c.cs.s, query)
	return (GraphQueryQuery)(c)
}

type GraphQueryQuery Completed

func (c GraphQueryQuery) Timeout(timeout int64) GraphQueryTimeout {
	c.cs.s = append(c.cs.s, "TIMEOUT", strconv.FormatInt(timeout, 10))
	return (GraphQueryTimeout)(c)
}

func (c GraphQueryQuery) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type GraphQueryTimeout Completed

func (c GraphQueryTimeout) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type GraphRoQuery Completed

func (b Builder) GraphRoQuery() (c GraphRoQuery) {
	c = GraphRoQuery{cs: get(), ks: b.ks, cf: readonly}
	c.cs.s = append(c.cs.s, "GRAPH.RO_QUERY")
	return c
}

func (c GraphRoQuery) Graph(graph string) GraphRoQueryGraph {
	if c.ks&NoSlot == NoSlot {
		c.ks = NoSlot | slot(graph)
	} else {
		c.ks = check(c.ks, slot(graph))
	}
	c.cs.s = append(c.cs.s, graph)
	return (GraphRoQueryGraph)(c)
}

type GraphRoQueryGraph Completed

func (c GraphRoQueryGraph) Query(query string) GraphRoQueryQuery {
	c.cs.s = append(c.cs.s, query)
	return (GraphRoQueryQuery)(c)
}

type GraphRoQueryQuery Completed

func (c GraphRoQueryQuery) Timeout(timeout int64) GraphRoQueryTimeout {
	c.cs.s = append(c.cs.s, "TIMEOUT", strconv.FormatInt(timeout, 10))
	return (GraphRoQueryTimeout)(c)
}

func (c GraphRoQueryQuery) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

func (c GraphRoQueryQuery) Cache() Cacheable {
	c.cs.Build()
	return Cacheable(c)
}

type GraphRoQueryTimeout Completed

func (c GraphRoQueryTimeout) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

func (c GraphRoQueryTimeout) Cache() Cacheable {
	c.cs.Build()
	return Cacheable(c)
}

type GraphSlowlog Completed

func (b Builder) GraphSlowlog() (c GraphSlowlog) {
	c = GraphSlowlog{cs: get(), ks: b.ks, cf: readonly}
	c.cs.s = append(c.cs.s, "GRAPH.SLOWLOG")
	return c
}

func (c GraphSlowlog) Graph(graph string) GraphSlowlogGraph {
	if c.ks&NoSlot == NoSlot {
		c.ks = NoSlot | slot(graph)
	} else {
		c.ks = check(c.ks, slot(graph))
	}
	c.cs.s = append(c.cs.s, graph)
	return (GraphSlowlogGraph)(c)
}

type GraphSlowlogGraph Completed

func (c GraphSlowlogGraph) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type Hdel Completed

func (b Builder) Hdel() (c Hdel) {
	c = Hdel{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "HDEL")
	return c
}

func (c Hdel) Key(key string) HdelKey {
	if c.ks&NoSlot == NoSlot {
		c.ks = NoSlot | slot(key)
	} else {
		c.ks = check(c.ks, slot(key))
	}
	c.cs.s = append(c.cs.s, key)
	return (HdelKey)(c)
}

type HdelField Completed

func (c HdelField) Field(field ...string) HdelField {
	c.cs.s = append(c.cs.s, field...)
	return c
}

func (c HdelField) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type HdelKey Completed

func (c HdelKey) Field(field ...string) HdelField {
	c.cs.s = append(c.cs.s, field...)
	return (HdelField)(c)
}

type Hello Completed

func (b Builder) Hello() (c Hello) {
	c = Hello{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "HELLO")
	return c
}

func (c Hello) Protover(protover int64) HelloArgumentsProtover {
	c.cs.s = append(c.cs.s, strconv.FormatInt(protover, 10))
	return (HelloArgumentsProtover)(c)
}

func (c Hello) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type HelloArgumentsAuth Completed

func (c HelloArgumentsAuth) Setname(clientname string) HelloArgumentsSetname {
	c.cs.s = append(c.cs.s, "SETNAME", clientname)
	return (HelloArgumentsSetname)(c)
}

func (c HelloArgumentsAuth) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type HelloArgumentsProtover Completed

func (c HelloArgumentsProtover) Auth(username string, password string) HelloArgumentsAuth {
	c.cs.s = append(c.cs.s, "AUTH", username, password)
	return (HelloArgumentsAuth)(c)
}

func (c HelloArgumentsProtover) Setname(clientname string) HelloArgumentsSetname {
	c.cs.s = append(c.cs.s, "SETNAME", clientname)
	return (HelloArgumentsSetname)(c)
}

func (c HelloArgumentsProtover) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type HelloArgumentsSetname Completed

func (c HelloArgumentsSetname) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type Hexists Completed

func (b Builder) Hexists() (c Hexists) {
	c = Hexists{cs: get(), ks: b.ks, cf: readonly}
	c.cs.s = append(c.cs.s, "HEXISTS")
	return c
}

func (c Hexists) Key(key string) HexistsKey {
	if c.ks&NoSlot == NoSlot {
		c.ks = NoSlot | slot(key)
	} else {
		c.ks = check(c.ks, slot(key))
	}
	c.cs.s = append(c.cs.s, key)
	return (HexistsKey)(c)
}

type HexistsField Completed

func (c HexistsField) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

func (c HexistsField) Cache() Cacheable {
	c.cs.Build()
	return Cacheable(c)
}

type HexistsKey Completed

func (c HexistsKey) Field(field string) HexistsField {
	c.cs.s = append(c.cs.s, field)
	return (HexistsField)(c)
}

type Hget Completed

func (b Builder) Hget() (c Hget) {
	c = Hget{cs: get(), ks: b.ks, cf: readonly}
	c.cs.s = append(c.cs.s, "HGET")
	return c
}

func (c Hget) Key(key string) HgetKey {
	if c.ks&NoSlot == NoSlot {
		c.ks = NoSlot | slot(key)
	} else {
		c.ks = check(c.ks, slot(key))
	}
	c.cs.s = append(c.cs.s, key)
	return (HgetKey)(c)
}

type HgetField Completed

func (c HgetField) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

func (c HgetField) Cache() Cacheable {
	c.cs.Build()
	return Cacheable(c)
}

type HgetKey Completed

func (c HgetKey) Field(field string) HgetField {
	c.cs.s = append(c.cs.s, field)
	return (HgetField)(c)
}

type Hgetall Completed

func (b Builder) Hgetall() (c Hgetall) {
	c = Hgetall{cs: get(), ks: b.ks, cf: readonly}
	c.cs.s = append(c.cs.s, "HGETALL")
	return c
}

func (c Hgetall) Key(key string) HgetallKey {
	if c.ks&NoSlot == NoSlot {
		c.ks = NoSlot | slot(key)
	} else {
		c.ks = check(c.ks, slot(key))
	}
	c.cs.s = append(c.cs.s, key)
	return (HgetallKey)(c)
}

type HgetallKey Completed

func (c HgetallKey) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

func (c HgetallKey) Cache() Cacheable {
	c.cs.Build()
	return Cacheable(c)
}

type Hincrby Completed

func (b Builder) Hincrby() (c Hincrby) {
	c = Hincrby{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "HINCRBY")
	return c
}

func (c Hincrby) Key(key string) HincrbyKey {
	if c.ks&NoSlot == NoSlot {
		c.ks = NoSlot | slot(key)
	} else {
		c.ks = check(c.ks, slot(key))
	}
	c.cs.s = append(c.cs.s, key)
	return (HincrbyKey)(c)
}

type HincrbyField Completed

func (c HincrbyField) Increment(increment int64) HincrbyIncrement {
	c.cs.s = append(c.cs.s, strconv.FormatInt(increment, 10))
	return (HincrbyIncrement)(c)
}

type HincrbyIncrement Completed

func (c HincrbyIncrement) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type HincrbyKey Completed

func (c HincrbyKey) Field(field string) HincrbyField {
	c.cs.s = append(c.cs.s, field)
	return (HincrbyField)(c)
}

type Hincrbyfloat Completed

func (b Builder) Hincrbyfloat() (c Hincrbyfloat) {
	c = Hincrbyfloat{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "HINCRBYFLOAT")
	return c
}

func (c Hincrbyfloat) Key(key string) HincrbyfloatKey {
	if c.ks&NoSlot == NoSlot {
		c.ks = NoSlot | slot(key)
	} else {
		c.ks = check(c.ks, slot(key))
	}
	c.cs.s = append(c.cs.s, key)
	return (HincrbyfloatKey)(c)
}

type HincrbyfloatField Completed

func (c HincrbyfloatField) Increment(increment float64) HincrbyfloatIncrement {
	c.cs.s = append(c.cs.s, strconv.FormatFloat(increment, 'f', -1, 64))
	return (HincrbyfloatIncrement)(c)
}

type HincrbyfloatIncrement Completed

func (c HincrbyfloatIncrement) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type HincrbyfloatKey Completed

func (c HincrbyfloatKey) Field(field string) HincrbyfloatField {
	c.cs.s = append(c.cs.s, field)
	return (HincrbyfloatField)(c)
}

type Hkeys Completed

func (b Builder) Hkeys() (c Hkeys) {
	c = Hkeys{cs: get(), ks: b.ks, cf: readonly}
	c.cs.s = append(c.cs.s, "HKEYS")
	return c
}

func (c Hkeys) Key(key string) HkeysKey {
	if c.ks&NoSlot == NoSlot {
		c.ks = NoSlot | slot(key)
	} else {
		c.ks = check(c.ks, slot(key))
	}
	c.cs.s = append(c.cs.s, key)
	return (HkeysKey)(c)
}

type HkeysKey Completed

func (c HkeysKey) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

func (c HkeysKey) Cache() Cacheable {
	c.cs.Build()
	return Cacheable(c)
}

type Hlen Completed

func (b Builder) Hlen() (c Hlen) {
	c = Hlen{cs: get(), ks: b.ks, cf: readonly}
	c.cs.s = append(c.cs.s, "HLEN")
	return c
}

func (c Hlen) Key(key string) HlenKey {
	if c.ks&NoSlot == NoSlot {
		c.ks = NoSlot | slot(key)
	} else {
		c.ks = check(c.ks, slot(key))
	}
	c.cs.s = append(c.cs.s, key)
	return (HlenKey)(c)
}

type HlenKey Completed

func (c HlenKey) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

func (c HlenKey) Cache() Cacheable {
	c.cs.Build()
	return Cacheable(c)
}

type Hmget Completed

func (b Builder) Hmget() (c Hmget) {
	c = Hmget{cs: get(), ks: b.ks, cf: readonly}
	c.cs.s = append(c.cs.s, "HMGET")
	return c
}

func (c Hmget) Key(key string) HmgetKey {
	if c.ks&NoSlot == NoSlot {
		c.ks = NoSlot | slot(key)
	} else {
		c.ks = check(c.ks, slot(key))
	}
	c.cs.s = append(c.cs.s, key)
	return (HmgetKey)(c)
}

type HmgetField Completed

func (c HmgetField) Field(field ...string) HmgetField {
	c.cs.s = append(c.cs.s, field...)
	return c
}

func (c HmgetField) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

func (c HmgetField) Cache() Cacheable {
	c.cs.Build()
	return Cacheable(c)
}

type HmgetKey Completed

func (c HmgetKey) Field(field ...string) HmgetField {
	c.cs.s = append(c.cs.s, field...)
	return (HmgetField)(c)
}

type Hmset Completed

func (b Builder) Hmset() (c Hmset) {
	c = Hmset{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "HMSET")
	return c
}

func (c Hmset) Key(key string) HmsetKey {
	if c.ks&NoSlot == NoSlot {
		c.ks = NoSlot | slot(key)
	} else {
		c.ks = check(c.ks, slot(key))
	}
	c.cs.s = append(c.cs.s, key)
	return (HmsetKey)(c)
}

type HmsetFieldValue Completed

func (c HmsetFieldValue) FieldValue(field string, value string) HmsetFieldValue {
	c.cs.s = append(c.cs.s, field, value)
	return c
}

func (c HmsetFieldValue) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type HmsetKey Completed

func (c HmsetKey) FieldValue() HmsetFieldValue {
	return (HmsetFieldValue)(c)
}

type Hrandfield Completed

func (b Builder) Hrandfield() (c Hrandfield) {
	c = Hrandfield{cs: get(), ks: b.ks, cf: readonly}
	c.cs.s = append(c.cs.s, "HRANDFIELD")
	return c
}

func (c Hrandfield) Key(key string) HrandfieldKey {
	if c.ks&NoSlot == NoSlot {
		c.ks = NoSlot | slot(key)
	} else {
		c.ks = check(c.ks, slot(key))
	}
	c.cs.s = append(c.cs.s, key)
	return (HrandfieldKey)(c)
}

type HrandfieldKey Completed

func (c HrandfieldKey) Count(count int64) HrandfieldOptionsCount {
	c.cs.s = append(c.cs.s, strconv.FormatInt(count, 10))
	return (HrandfieldOptionsCount)(c)
}

func (c HrandfieldKey) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type HrandfieldOptionsCount Completed

func (c HrandfieldOptionsCount) Withvalues() HrandfieldOptionsWithvalues {
	c.cs.s = append(c.cs.s, "WITHVALUES")
	return (HrandfieldOptionsWithvalues)(c)
}

func (c HrandfieldOptionsCount) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type HrandfieldOptionsWithvalues Completed

func (c HrandfieldOptionsWithvalues) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type Hscan Completed

func (b Builder) Hscan() (c Hscan) {
	c = Hscan{cs: get(), ks: b.ks, cf: readonly}
	c.cs.s = append(c.cs.s, "HSCAN")
	return c
}

func (c Hscan) Key(key string) HscanKey {
	if c.ks&NoSlot == NoSlot {
		c.ks = NoSlot | slot(key)
	} else {
		c.ks = check(c.ks, slot(key))
	}
	c.cs.s = append(c.cs.s, key)
	return (HscanKey)(c)
}

type HscanCount Completed

func (c HscanCount) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type HscanCursor Completed

func (c HscanCursor) Match(pattern string) HscanMatch {
	c.cs.s = append(c.cs.s, "MATCH", pattern)
	return (HscanMatch)(c)
}

func (c HscanCursor) Count(count int64) HscanCount {
	c.cs.s = append(c.cs.s, "COUNT", strconv.FormatInt(count, 10))
	return (HscanCount)(c)
}

func (c HscanCursor) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type HscanKey Completed

func (c HscanKey) Cursor(cursor uint64) HscanCursor {
	c.cs.s = append(c.cs.s, strconv.FormatUint(cursor, 10))
	return (HscanCursor)(c)
}

type HscanMatch Completed

func (c HscanMatch) Count(count int64) HscanCount {
	c.cs.s = append(c.cs.s, "COUNT", strconv.FormatInt(count, 10))
	return (HscanCount)(c)
}

func (c HscanMatch) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type Hset Completed

func (b Builder) Hset() (c Hset) {
	c = Hset{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "HSET")
	return c
}

func (c Hset) Key(key string) HsetKey {
	if c.ks&NoSlot == NoSlot {
		c.ks = NoSlot | slot(key)
	} else {
		c.ks = check(c.ks, slot(key))
	}
	c.cs.s = append(c.cs.s, key)
	return (HsetKey)(c)
}

type HsetFieldValue Completed

func (c HsetFieldValue) FieldValue(field string, value string) HsetFieldValue {
	c.cs.s = append(c.cs.s, field, value)
	return c
}

func (c HsetFieldValue) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type HsetKey Completed

func (c HsetKey) FieldValue() HsetFieldValue {
	return (HsetFieldValue)(c)
}

type Hsetnx Completed

func (b Builder) Hsetnx() (c Hsetnx) {
	c = Hsetnx{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "HSETNX")
	return c
}

func (c Hsetnx) Key(key string) HsetnxKey {
	if c.ks&NoSlot == NoSlot {
		c.ks = NoSlot | slot(key)
	} else {
		c.ks = check(c.ks, slot(key))
	}
	c.cs.s = append(c.cs.s, key)
	return (HsetnxKey)(c)
}

type HsetnxField Completed

func (c HsetnxField) Value(value string) HsetnxValue {
	c.cs.s = append(c.cs.s, value)
	return (HsetnxValue)(c)
}

type HsetnxKey Completed

func (c HsetnxKey) Field(field string) HsetnxField {
	c.cs.s = append(c.cs.s, field)
	return (HsetnxField)(c)
}

type HsetnxValue Completed

func (c HsetnxValue) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type Hstrlen Completed

func (b Builder) Hstrlen() (c Hstrlen) {
	c = Hstrlen{cs: get(), ks: b.ks, cf: readonly}
	c.cs.s = append(c.cs.s, "HSTRLEN")
	return c
}

func (c Hstrlen) Key(key string) HstrlenKey {
	if c.ks&NoSlot == NoSlot {
		c.ks = NoSlot | slot(key)
	} else {
		c.ks = check(c.ks, slot(key))
	}
	c.cs.s = append(c.cs.s, key)
	return (HstrlenKey)(c)
}

type HstrlenField Completed

func (c HstrlenField) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

func (c HstrlenField) Cache() Cacheable {
	c.cs.Build()
	return Cacheable(c)
}

type HstrlenKey Completed

func (c HstrlenKey) Field(field string) HstrlenField {
	c.cs.s = append(c.cs.s, field)
	return (HstrlenField)(c)
}

type Hvals Completed

func (b Builder) Hvals() (c Hvals) {
	c = Hvals{cs: get(), ks: b.ks, cf: readonly}
	c.cs.s = append(c.cs.s, "HVALS")
	return c
}

func (c Hvals) Key(key string) HvalsKey {
	if c.ks&NoSlot == NoSlot {
		c.ks = NoSlot | slot(key)
	} else {
		c.ks = check(c.ks, slot(key))
	}
	c.cs.s = append(c.cs.s, key)
	return (HvalsKey)(c)
}

type HvalsKey Completed

func (c HvalsKey) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

func (c HvalsKey) Cache() Cacheable {
	c.cs.Build()
	return Cacheable(c)
}

type Incr Completed

func (b Builder) Incr() (c Incr) {
	c = Incr{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "INCR")
	return c
}

func (c Incr) Key(key string) IncrKey {
	if c.ks&NoSlot == NoSlot {
		c.ks = NoSlot | slot(key)
	} else {
		c.ks = check(c.ks, slot(key))
	}
	c.cs.s = append(c.cs.s, key)
	return (IncrKey)(c)
}

type IncrKey Completed

func (c IncrKey) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type Incrby Completed

func (b Builder) Incrby() (c Incrby) {
	c = Incrby{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "INCRBY")
	return c
}

func (c Incrby) Key(key string) IncrbyKey {
	if c.ks&NoSlot == NoSlot {
		c.ks = NoSlot | slot(key)
	} else {
		c.ks = check(c.ks, slot(key))
	}
	c.cs.s = append(c.cs.s, key)
	return (IncrbyKey)(c)
}

type IncrbyIncrement Completed

func (c IncrbyIncrement) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type IncrbyKey Completed

func (c IncrbyKey) Increment(increment int64) IncrbyIncrement {
	c.cs.s = append(c.cs.s, strconv.FormatInt(increment, 10))
	return (IncrbyIncrement)(c)
}

type Incrbyfloat Completed

func (b Builder) Incrbyfloat() (c Incrbyfloat) {
	c = Incrbyfloat{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "INCRBYFLOAT")
	return c
}

func (c Incrbyfloat) Key(key string) IncrbyfloatKey {
	if c.ks&NoSlot == NoSlot {
		c.ks = NoSlot | slot(key)
	} else {
		c.ks = check(c.ks, slot(key))
	}
	c.cs.s = append(c.cs.s, key)
	return (IncrbyfloatKey)(c)
}

type IncrbyfloatIncrement Completed

func (c IncrbyfloatIncrement) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type IncrbyfloatKey Completed

func (c IncrbyfloatKey) Increment(increment float64) IncrbyfloatIncrement {
	c.cs.s = append(c.cs.s, strconv.FormatFloat(increment, 'f', -1, 64))
	return (IncrbyfloatIncrement)(c)
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

type JsonArrappend Completed

func (b Builder) JsonArrappend() (c JsonArrappend) {
	c = JsonArrappend{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "JSON.ARRAPPEND")
	return c
}

func (c JsonArrappend) Key(key string) JsonArrappendKey {
	if c.ks&NoSlot == NoSlot {
		c.ks = NoSlot | slot(key)
	} else {
		c.ks = check(c.ks, slot(key))
	}
	c.cs.s = append(c.cs.s, key)
	return (JsonArrappendKey)(c)
}

type JsonArrappendKey Completed

func (c JsonArrappendKey) Path(path string) JsonArrappendPath {
	c.cs.s = append(c.cs.s, path)
	return (JsonArrappendPath)(c)
}

func (c JsonArrappendKey) Value(value ...string) JsonArrappendValue {
	c.cs.s = append(c.cs.s, value...)
	return (JsonArrappendValue)(c)
}

type JsonArrappendPath Completed

func (c JsonArrappendPath) Value(value ...string) JsonArrappendValue {
	c.cs.s = append(c.cs.s, value...)
	return (JsonArrappendValue)(c)
}

type JsonArrappendValue Completed

func (c JsonArrappendValue) Value(value ...string) JsonArrappendValue {
	c.cs.s = append(c.cs.s, value...)
	return c
}

func (c JsonArrappendValue) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type JsonArrindex Completed

func (b Builder) JsonArrindex() (c JsonArrindex) {
	c = JsonArrindex{cs: get(), ks: b.ks, cf: readonly}
	c.cs.s = append(c.cs.s, "JSON.ARRINDEX")
	return c
}

func (c JsonArrindex) Key(key string) JsonArrindexKey {
	if c.ks&NoSlot == NoSlot {
		c.ks = NoSlot | slot(key)
	} else {
		c.ks = check(c.ks, slot(key))
	}
	c.cs.s = append(c.cs.s, key)
	return (JsonArrindexKey)(c)
}

type JsonArrindexKey Completed

func (c JsonArrindexKey) Path(path string) JsonArrindexPath {
	c.cs.s = append(c.cs.s, path)
	return (JsonArrindexPath)(c)
}

type JsonArrindexPath Completed

func (c JsonArrindexPath) Value(value string) JsonArrindexValue {
	c.cs.s = append(c.cs.s, value)
	return (JsonArrindexValue)(c)
}

type JsonArrindexStartStart Completed

func (c JsonArrindexStartStart) Stop(stop int64) JsonArrindexStartStop {
	c.cs.s = append(c.cs.s, strconv.FormatInt(stop, 10))
	return (JsonArrindexStartStop)(c)
}

func (c JsonArrindexStartStart) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

func (c JsonArrindexStartStart) Cache() Cacheable {
	c.cs.Build()
	return Cacheable(c)
}

type JsonArrindexStartStop Completed

func (c JsonArrindexStartStop) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

func (c JsonArrindexStartStop) Cache() Cacheable {
	c.cs.Build()
	return Cacheable(c)
}

type JsonArrindexValue Completed

func (c JsonArrindexValue) Start(start int64) JsonArrindexStartStart {
	c.cs.s = append(c.cs.s, strconv.FormatInt(start, 10))
	return (JsonArrindexStartStart)(c)
}

func (c JsonArrindexValue) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

func (c JsonArrindexValue) Cache() Cacheable {
	c.cs.Build()
	return Cacheable(c)
}

type JsonArrinsert Completed

func (b Builder) JsonArrinsert() (c JsonArrinsert) {
	c = JsonArrinsert{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "JSON.ARRINSERT")
	return c
}

func (c JsonArrinsert) Key(key string) JsonArrinsertKey {
	if c.ks&NoSlot == NoSlot {
		c.ks = NoSlot | slot(key)
	} else {
		c.ks = check(c.ks, slot(key))
	}
	c.cs.s = append(c.cs.s, key)
	return (JsonArrinsertKey)(c)
}

type JsonArrinsertIndex Completed

func (c JsonArrinsertIndex) Value(value ...string) JsonArrinsertValue {
	c.cs.s = append(c.cs.s, value...)
	return (JsonArrinsertValue)(c)
}

type JsonArrinsertKey Completed

func (c JsonArrinsertKey) Path(path string) JsonArrinsertPath {
	c.cs.s = append(c.cs.s, path)
	return (JsonArrinsertPath)(c)
}

type JsonArrinsertPath Completed

func (c JsonArrinsertPath) Index(index int64) JsonArrinsertIndex {
	c.cs.s = append(c.cs.s, strconv.FormatInt(index, 10))
	return (JsonArrinsertIndex)(c)
}

type JsonArrinsertValue Completed

func (c JsonArrinsertValue) Value(value ...string) JsonArrinsertValue {
	c.cs.s = append(c.cs.s, value...)
	return c
}

func (c JsonArrinsertValue) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type JsonArrlen Completed

func (b Builder) JsonArrlen() (c JsonArrlen) {
	c = JsonArrlen{cs: get(), ks: b.ks, cf: readonly}
	c.cs.s = append(c.cs.s, "JSON.ARRLEN")
	return c
}

func (c JsonArrlen) Key(key string) JsonArrlenKey {
	if c.ks&NoSlot == NoSlot {
		c.ks = NoSlot | slot(key)
	} else {
		c.ks = check(c.ks, slot(key))
	}
	c.cs.s = append(c.cs.s, key)
	return (JsonArrlenKey)(c)
}

type JsonArrlenKey Completed

func (c JsonArrlenKey) Path(path string) JsonArrlenPath {
	c.cs.s = append(c.cs.s, path)
	return (JsonArrlenPath)(c)
}

func (c JsonArrlenKey) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

func (c JsonArrlenKey) Cache() Cacheable {
	c.cs.Build()
	return Cacheable(c)
}

type JsonArrlenPath Completed

func (c JsonArrlenPath) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

func (c JsonArrlenPath) Cache() Cacheable {
	c.cs.Build()
	return Cacheable(c)
}

type JsonArrpop Completed

func (b Builder) JsonArrpop() (c JsonArrpop) {
	c = JsonArrpop{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "JSON.ARRPOP")
	return c
}

func (c JsonArrpop) Key(key string) JsonArrpopKey {
	if c.ks&NoSlot == NoSlot {
		c.ks = NoSlot | slot(key)
	} else {
		c.ks = check(c.ks, slot(key))
	}
	c.cs.s = append(c.cs.s, key)
	return (JsonArrpopKey)(c)
}

type JsonArrpopKey Completed

func (c JsonArrpopKey) Path(path string) JsonArrpopPathPath {
	c.cs.s = append(c.cs.s, path)
	return (JsonArrpopPathPath)(c)
}

func (c JsonArrpopKey) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type JsonArrpopPathIndex Completed

func (c JsonArrpopPathIndex) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type JsonArrpopPathPath Completed

func (c JsonArrpopPathPath) Index(index int64) JsonArrpopPathIndex {
	c.cs.s = append(c.cs.s, strconv.FormatInt(index, 10))
	return (JsonArrpopPathIndex)(c)
}

func (c JsonArrpopPathPath) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type JsonArrtrim Completed

func (b Builder) JsonArrtrim() (c JsonArrtrim) {
	c = JsonArrtrim{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "JSON.ARRTRIM")
	return c
}

func (c JsonArrtrim) Key(key string) JsonArrtrimKey {
	if c.ks&NoSlot == NoSlot {
		c.ks = NoSlot | slot(key)
	} else {
		c.ks = check(c.ks, slot(key))
	}
	c.cs.s = append(c.cs.s, key)
	return (JsonArrtrimKey)(c)
}

type JsonArrtrimKey Completed

func (c JsonArrtrimKey) Path(path string) JsonArrtrimPath {
	c.cs.s = append(c.cs.s, path)
	return (JsonArrtrimPath)(c)
}

type JsonArrtrimPath Completed

func (c JsonArrtrimPath) Start(start int64) JsonArrtrimStart {
	c.cs.s = append(c.cs.s, strconv.FormatInt(start, 10))
	return (JsonArrtrimStart)(c)
}

type JsonArrtrimStart Completed

func (c JsonArrtrimStart) Stop(stop int64) JsonArrtrimStop {
	c.cs.s = append(c.cs.s, strconv.FormatInt(stop, 10))
	return (JsonArrtrimStop)(c)
}

type JsonArrtrimStop Completed

func (c JsonArrtrimStop) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type JsonClear Completed

func (b Builder) JsonClear() (c JsonClear) {
	c = JsonClear{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "JSON.CLEAR")
	return c
}

func (c JsonClear) Key(key string) JsonClearKey {
	if c.ks&NoSlot == NoSlot {
		c.ks = NoSlot | slot(key)
	} else {
		c.ks = check(c.ks, slot(key))
	}
	c.cs.s = append(c.cs.s, key)
	return (JsonClearKey)(c)
}

type JsonClearKey Completed

func (c JsonClearKey) Path(path string) JsonClearPath {
	c.cs.s = append(c.cs.s, path)
	return (JsonClearPath)(c)
}

func (c JsonClearKey) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type JsonClearPath Completed

func (c JsonClearPath) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type JsonDebugHelp Completed

func (b Builder) JsonDebugHelp() (c JsonDebugHelp) {
	c = JsonDebugHelp{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "JSON.DEBUG", "HELP")
	return c
}

func (c JsonDebugHelp) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type JsonDebugMemory Completed

func (b Builder) JsonDebugMemory() (c JsonDebugMemory) {
	c = JsonDebugMemory{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "JSON.DEBUG", "MEMORY")
	return c
}

func (c JsonDebugMemory) Key(key string) JsonDebugMemoryKey {
	if c.ks&NoSlot == NoSlot {
		c.ks = NoSlot | slot(key)
	} else {
		c.ks = check(c.ks, slot(key))
	}
	c.cs.s = append(c.cs.s, key)
	return (JsonDebugMemoryKey)(c)
}

type JsonDebugMemoryKey Completed

func (c JsonDebugMemoryKey) Path(path string) JsonDebugMemoryPath {
	c.cs.s = append(c.cs.s, path)
	return (JsonDebugMemoryPath)(c)
}

func (c JsonDebugMemoryKey) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type JsonDebugMemoryPath Completed

func (c JsonDebugMemoryPath) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type JsonDel Completed

func (b Builder) JsonDel() (c JsonDel) {
	c = JsonDel{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "JSON.DEL")
	return c
}

func (c JsonDel) Key(key string) JsonDelKey {
	if c.ks&NoSlot == NoSlot {
		c.ks = NoSlot | slot(key)
	} else {
		c.ks = check(c.ks, slot(key))
	}
	c.cs.s = append(c.cs.s, key)
	return (JsonDelKey)(c)
}

type JsonDelKey Completed

func (c JsonDelKey) Path(path string) JsonDelPath {
	c.cs.s = append(c.cs.s, path)
	return (JsonDelPath)(c)
}

func (c JsonDelKey) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type JsonDelPath Completed

func (c JsonDelPath) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type JsonForget Completed

func (b Builder) JsonForget() (c JsonForget) {
	c = JsonForget{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "JSON.FORGET")
	return c
}

func (c JsonForget) Key(key string) JsonForgetKey {
	if c.ks&NoSlot == NoSlot {
		c.ks = NoSlot | slot(key)
	} else {
		c.ks = check(c.ks, slot(key))
	}
	c.cs.s = append(c.cs.s, key)
	return (JsonForgetKey)(c)
}

type JsonForgetKey Completed

func (c JsonForgetKey) Path(path string) JsonForgetPath {
	c.cs.s = append(c.cs.s, path)
	return (JsonForgetPath)(c)
}

func (c JsonForgetKey) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type JsonForgetPath Completed

func (c JsonForgetPath) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type JsonGet Completed

func (b Builder) JsonGet() (c JsonGet) {
	c = JsonGet{cs: get(), ks: b.ks, cf: readonly}
	c.cs.s = append(c.cs.s, "JSON.GET")
	return c
}

func (c JsonGet) Key(key string) JsonGetKey {
	if c.ks&NoSlot == NoSlot {
		c.ks = NoSlot | slot(key)
	} else {
		c.ks = check(c.ks, slot(key))
	}
	c.cs.s = append(c.cs.s, key)
	return (JsonGetKey)(c)
}

type JsonGetIndent Completed

func (c JsonGetIndent) Newline(newline string) JsonGetNewline {
	c.cs.s = append(c.cs.s, "NEWLINE", newline)
	return (JsonGetNewline)(c)
}

func (c JsonGetIndent) Space(space string) JsonGetSpace {
	c.cs.s = append(c.cs.s, "SPACE", space)
	return (JsonGetSpace)(c)
}

func (c JsonGetIndent) Path(path ...string) JsonGetPath {
	c.cs.s = append(c.cs.s, path...)
	return (JsonGetPath)(c)
}

func (c JsonGetIndent) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

func (c JsonGetIndent) Cache() Cacheable {
	c.cs.Build()
	return Cacheable(c)
}

type JsonGetKey Completed

func (c JsonGetKey) Indent(indent string) JsonGetIndent {
	c.cs.s = append(c.cs.s, "INDENT", indent)
	return (JsonGetIndent)(c)
}

func (c JsonGetKey) Newline(newline string) JsonGetNewline {
	c.cs.s = append(c.cs.s, "NEWLINE", newline)
	return (JsonGetNewline)(c)
}

func (c JsonGetKey) Space(space string) JsonGetSpace {
	c.cs.s = append(c.cs.s, "SPACE", space)
	return (JsonGetSpace)(c)
}

func (c JsonGetKey) Path(path ...string) JsonGetPath {
	c.cs.s = append(c.cs.s, path...)
	return (JsonGetPath)(c)
}

func (c JsonGetKey) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

func (c JsonGetKey) Cache() Cacheable {
	c.cs.Build()
	return Cacheable(c)
}

type JsonGetNewline Completed

func (c JsonGetNewline) Space(space string) JsonGetSpace {
	c.cs.s = append(c.cs.s, "SPACE", space)
	return (JsonGetSpace)(c)
}

func (c JsonGetNewline) Path(path ...string) JsonGetPath {
	c.cs.s = append(c.cs.s, path...)
	return (JsonGetPath)(c)
}

func (c JsonGetNewline) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

func (c JsonGetNewline) Cache() Cacheable {
	c.cs.Build()
	return Cacheable(c)
}

type JsonGetPath Completed

func (c JsonGetPath) Path(path ...string) JsonGetPath {
	c.cs.s = append(c.cs.s, path...)
	return c
}

func (c JsonGetPath) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

func (c JsonGetPath) Cache() Cacheable {
	c.cs.Build()
	return Cacheable(c)
}

type JsonGetSpace Completed

func (c JsonGetSpace) Path(path ...string) JsonGetPath {
	c.cs.s = append(c.cs.s, path...)
	return (JsonGetPath)(c)
}

func (c JsonGetSpace) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

func (c JsonGetSpace) Cache() Cacheable {
	c.cs.Build()
	return Cacheable(c)
}

type JsonMget Completed

func (b Builder) JsonMget() (c JsonMget) {
	c = JsonMget{cs: get(), ks: b.ks, cf: mtGetTag}
	c.cs.s = append(c.cs.s, "JSON.MGET")
	return c
}

func (c JsonMget) Key(key ...string) JsonMgetKey {
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
	return (JsonMgetKey)(c)
}

type JsonMgetKey Completed

func (c JsonMgetKey) Key(key ...string) JsonMgetKey {
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

func (c JsonMgetKey) Path(path string) JsonMgetPath {
	c.cs.s = append(c.cs.s, path)
	return (JsonMgetPath)(c)
}

type JsonMgetPath Completed

func (c JsonMgetPath) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

func (c JsonMgetPath) Cache() Cacheable {
	c.cs.Build()
	return Cacheable(c)
}

type JsonNumincrby Completed

func (b Builder) JsonNumincrby() (c JsonNumincrby) {
	c = JsonNumincrby{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "JSON.NUMINCRBY")
	return c
}

func (c JsonNumincrby) Key(key string) JsonNumincrbyKey {
	if c.ks&NoSlot == NoSlot {
		c.ks = NoSlot | slot(key)
	} else {
		c.ks = check(c.ks, slot(key))
	}
	c.cs.s = append(c.cs.s, key)
	return (JsonNumincrbyKey)(c)
}

type JsonNumincrbyKey Completed

func (c JsonNumincrbyKey) Path(path string) JsonNumincrbyPath {
	c.cs.s = append(c.cs.s, path)
	return (JsonNumincrbyPath)(c)
}

type JsonNumincrbyPath Completed

func (c JsonNumincrbyPath) Value(value float64) JsonNumincrbyValue {
	c.cs.s = append(c.cs.s, strconv.FormatFloat(value, 'f', -1, 64))
	return (JsonNumincrbyValue)(c)
}

type JsonNumincrbyValue Completed

func (c JsonNumincrbyValue) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type JsonNummultby Completed

func (b Builder) JsonNummultby() (c JsonNummultby) {
	c = JsonNummultby{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "JSON.NUMMULTBY")
	return c
}

func (c JsonNummultby) Key(key string) JsonNummultbyKey {
	if c.ks&NoSlot == NoSlot {
		c.ks = NoSlot | slot(key)
	} else {
		c.ks = check(c.ks, slot(key))
	}
	c.cs.s = append(c.cs.s, key)
	return (JsonNummultbyKey)(c)
}

type JsonNummultbyKey Completed

func (c JsonNummultbyKey) Path(path string) JsonNummultbyPath {
	c.cs.s = append(c.cs.s, path)
	return (JsonNummultbyPath)(c)
}

type JsonNummultbyPath Completed

func (c JsonNummultbyPath) Value(value float64) JsonNummultbyValue {
	c.cs.s = append(c.cs.s, strconv.FormatFloat(value, 'f', -1, 64))
	return (JsonNummultbyValue)(c)
}

type JsonNummultbyValue Completed

func (c JsonNummultbyValue) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type JsonObjkeys Completed

func (b Builder) JsonObjkeys() (c JsonObjkeys) {
	c = JsonObjkeys{cs: get(), ks: b.ks, cf: readonly}
	c.cs.s = append(c.cs.s, "JSON.OBJKEYS")
	return c
}

func (c JsonObjkeys) Key(key string) JsonObjkeysKey {
	if c.ks&NoSlot == NoSlot {
		c.ks = NoSlot | slot(key)
	} else {
		c.ks = check(c.ks, slot(key))
	}
	c.cs.s = append(c.cs.s, key)
	return (JsonObjkeysKey)(c)
}

type JsonObjkeysKey Completed

func (c JsonObjkeysKey) Path(path string) JsonObjkeysPath {
	c.cs.s = append(c.cs.s, path)
	return (JsonObjkeysPath)(c)
}

func (c JsonObjkeysKey) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

func (c JsonObjkeysKey) Cache() Cacheable {
	c.cs.Build()
	return Cacheable(c)
}

type JsonObjkeysPath Completed

func (c JsonObjkeysPath) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

func (c JsonObjkeysPath) Cache() Cacheable {
	c.cs.Build()
	return Cacheable(c)
}

type JsonObjlen Completed

func (b Builder) JsonObjlen() (c JsonObjlen) {
	c = JsonObjlen{cs: get(), ks: b.ks, cf: readonly}
	c.cs.s = append(c.cs.s, "JSON.OBJLEN")
	return c
}

func (c JsonObjlen) Key(key string) JsonObjlenKey {
	if c.ks&NoSlot == NoSlot {
		c.ks = NoSlot | slot(key)
	} else {
		c.ks = check(c.ks, slot(key))
	}
	c.cs.s = append(c.cs.s, key)
	return (JsonObjlenKey)(c)
}

type JsonObjlenKey Completed

func (c JsonObjlenKey) Path(path string) JsonObjlenPath {
	c.cs.s = append(c.cs.s, path)
	return (JsonObjlenPath)(c)
}

func (c JsonObjlenKey) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

func (c JsonObjlenKey) Cache() Cacheable {
	c.cs.Build()
	return Cacheable(c)
}

type JsonObjlenPath Completed

func (c JsonObjlenPath) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

func (c JsonObjlenPath) Cache() Cacheable {
	c.cs.Build()
	return Cacheable(c)
}

type JsonResp Completed

func (b Builder) JsonResp() (c JsonResp) {
	c = JsonResp{cs: get(), ks: b.ks, cf: readonly}
	c.cs.s = append(c.cs.s, "JSON.RESP")
	return c
}

func (c JsonResp) Key(key string) JsonRespKey {
	if c.ks&NoSlot == NoSlot {
		c.ks = NoSlot | slot(key)
	} else {
		c.ks = check(c.ks, slot(key))
	}
	c.cs.s = append(c.cs.s, key)
	return (JsonRespKey)(c)
}

type JsonRespKey Completed

func (c JsonRespKey) Path(path string) JsonRespPath {
	c.cs.s = append(c.cs.s, path)
	return (JsonRespPath)(c)
}

func (c JsonRespKey) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

func (c JsonRespKey) Cache() Cacheable {
	c.cs.Build()
	return Cacheable(c)
}

type JsonRespPath Completed

func (c JsonRespPath) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

func (c JsonRespPath) Cache() Cacheable {
	c.cs.Build()
	return Cacheable(c)
}

type JsonSet Completed

func (b Builder) JsonSet() (c JsonSet) {
	c = JsonSet{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "JSON.SET")
	return c
}

func (c JsonSet) Key(key string) JsonSetKey {
	if c.ks&NoSlot == NoSlot {
		c.ks = NoSlot | slot(key)
	} else {
		c.ks = check(c.ks, slot(key))
	}
	c.cs.s = append(c.cs.s, key)
	return (JsonSetKey)(c)
}

type JsonSetConditionNx Completed

func (c JsonSetConditionNx) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type JsonSetConditionXx Completed

func (c JsonSetConditionXx) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type JsonSetKey Completed

func (c JsonSetKey) Path(path string) JsonSetPath {
	c.cs.s = append(c.cs.s, path)
	return (JsonSetPath)(c)
}

type JsonSetPath Completed

func (c JsonSetPath) Value(value string) JsonSetValue {
	c.cs.s = append(c.cs.s, value)
	return (JsonSetValue)(c)
}

type JsonSetValue Completed

func (c JsonSetValue) Nx() JsonSetConditionNx {
	c.cs.s = append(c.cs.s, "NX")
	return (JsonSetConditionNx)(c)
}

func (c JsonSetValue) Xx() JsonSetConditionXx {
	c.cs.s = append(c.cs.s, "XX")
	return (JsonSetConditionXx)(c)
}

func (c JsonSetValue) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type JsonStrappend Completed

func (b Builder) JsonStrappend() (c JsonStrappend) {
	c = JsonStrappend{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "JSON.STRAPPEND")
	return c
}

func (c JsonStrappend) Key(key string) JsonStrappendKey {
	if c.ks&NoSlot == NoSlot {
		c.ks = NoSlot | slot(key)
	} else {
		c.ks = check(c.ks, slot(key))
	}
	c.cs.s = append(c.cs.s, key)
	return (JsonStrappendKey)(c)
}

type JsonStrappendKey Completed

func (c JsonStrappendKey) Path(path string) JsonStrappendPath {
	c.cs.s = append(c.cs.s, path)
	return (JsonStrappendPath)(c)
}

func (c JsonStrappendKey) Value(value string) JsonStrappendValue {
	c.cs.s = append(c.cs.s, value)
	return (JsonStrappendValue)(c)
}

type JsonStrappendPath Completed

func (c JsonStrappendPath) Value(value string) JsonStrappendValue {
	c.cs.s = append(c.cs.s, value)
	return (JsonStrappendValue)(c)
}

type JsonStrappendValue Completed

func (c JsonStrappendValue) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type JsonStrlen Completed

func (b Builder) JsonStrlen() (c JsonStrlen) {
	c = JsonStrlen{cs: get(), ks: b.ks, cf: readonly}
	c.cs.s = append(c.cs.s, "JSON.STRLEN")
	return c
}

func (c JsonStrlen) Key(key string) JsonStrlenKey {
	if c.ks&NoSlot == NoSlot {
		c.ks = NoSlot | slot(key)
	} else {
		c.ks = check(c.ks, slot(key))
	}
	c.cs.s = append(c.cs.s, key)
	return (JsonStrlenKey)(c)
}

type JsonStrlenKey Completed

func (c JsonStrlenKey) Path(path string) JsonStrlenPath {
	c.cs.s = append(c.cs.s, path)
	return (JsonStrlenPath)(c)
}

func (c JsonStrlenKey) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

func (c JsonStrlenKey) Cache() Cacheable {
	c.cs.Build()
	return Cacheable(c)
}

type JsonStrlenPath Completed

func (c JsonStrlenPath) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

func (c JsonStrlenPath) Cache() Cacheable {
	c.cs.Build()
	return Cacheable(c)
}

type JsonToggle Completed

func (b Builder) JsonToggle() (c JsonToggle) {
	c = JsonToggle{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "JSON.TOGGLE")
	return c
}

func (c JsonToggle) Key(key string) JsonToggleKey {
	if c.ks&NoSlot == NoSlot {
		c.ks = NoSlot | slot(key)
	} else {
		c.ks = check(c.ks, slot(key))
	}
	c.cs.s = append(c.cs.s, key)
	return (JsonToggleKey)(c)
}

type JsonToggleKey Completed

func (c JsonToggleKey) Path(path string) JsonTogglePath {
	c.cs.s = append(c.cs.s, path)
	return (JsonTogglePath)(c)
}

type JsonTogglePath Completed

func (c JsonTogglePath) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type JsonType Completed

func (b Builder) JsonType() (c JsonType) {
	c = JsonType{cs: get(), ks: b.ks, cf: readonly}
	c.cs.s = append(c.cs.s, "JSON.TYPE")
	return c
}

func (c JsonType) Key(key string) JsonTypeKey {
	if c.ks&NoSlot == NoSlot {
		c.ks = NoSlot | slot(key)
	} else {
		c.ks = check(c.ks, slot(key))
	}
	c.cs.s = append(c.cs.s, key)
	return (JsonTypeKey)(c)
}

type JsonTypeKey Completed

func (c JsonTypeKey) Path(path string) JsonTypePath {
	c.cs.s = append(c.cs.s, path)
	return (JsonTypePath)(c)
}

func (c JsonTypeKey) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

func (c JsonTypeKey) Cache() Cacheable {
	c.cs.Build()
	return Cacheable(c)
}

type JsonTypePath Completed

func (c JsonTypePath) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

func (c JsonTypePath) Cache() Cacheable {
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

type Lcs Completed

func (b Builder) Lcs() (c Lcs) {
	c = Lcs{cs: get(), ks: b.ks, cf: readonly}
	c.cs.s = append(c.cs.s, "LCS")
	return c
}

func (c Lcs) Key1(key1 string) LcsKey1 {
	if c.ks&NoSlot == NoSlot {
		c.ks = NoSlot | slot(key1)
	} else {
		c.ks = check(c.ks, slot(key1))
	}
	c.cs.s = append(c.cs.s, key1)
	return (LcsKey1)(c)
}

type LcsIdx Completed

func (c LcsIdx) Minmatchlen(len int64) LcsMinmatchlen {
	c.cs.s = append(c.cs.s, "MINMATCHLEN", strconv.FormatInt(len, 10))
	return (LcsMinmatchlen)(c)
}

func (c LcsIdx) Withmatchlen() LcsWithmatchlen {
	c.cs.s = append(c.cs.s, "WITHMATCHLEN")
	return (LcsWithmatchlen)(c)
}

func (c LcsIdx) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type LcsKey1 Completed

func (c LcsKey1) Key2(key2 string) LcsKey2 {
	if c.ks&NoSlot == NoSlot {
		c.ks = NoSlot | slot(key2)
	} else {
		c.ks = check(c.ks, slot(key2))
	}
	c.cs.s = append(c.cs.s, key2)
	return (LcsKey2)(c)
}

type LcsKey2 Completed

func (c LcsKey2) Len() LcsLen {
	c.cs.s = append(c.cs.s, "LEN")
	return (LcsLen)(c)
}

func (c LcsKey2) Idx() LcsIdx {
	c.cs.s = append(c.cs.s, "IDX")
	return (LcsIdx)(c)
}

func (c LcsKey2) Minmatchlen(len int64) LcsMinmatchlen {
	c.cs.s = append(c.cs.s, "MINMATCHLEN", strconv.FormatInt(len, 10))
	return (LcsMinmatchlen)(c)
}

func (c LcsKey2) Withmatchlen() LcsWithmatchlen {
	c.cs.s = append(c.cs.s, "WITHMATCHLEN")
	return (LcsWithmatchlen)(c)
}

func (c LcsKey2) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type LcsLen Completed

func (c LcsLen) Idx() LcsIdx {
	c.cs.s = append(c.cs.s, "IDX")
	return (LcsIdx)(c)
}

func (c LcsLen) Minmatchlen(len int64) LcsMinmatchlen {
	c.cs.s = append(c.cs.s, "MINMATCHLEN", strconv.FormatInt(len, 10))
	return (LcsMinmatchlen)(c)
}

func (c LcsLen) Withmatchlen() LcsWithmatchlen {
	c.cs.s = append(c.cs.s, "WITHMATCHLEN")
	return (LcsWithmatchlen)(c)
}

func (c LcsLen) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type LcsMinmatchlen Completed

func (c LcsMinmatchlen) Withmatchlen() LcsWithmatchlen {
	c.cs.s = append(c.cs.s, "WITHMATCHLEN")
	return (LcsWithmatchlen)(c)
}

func (c LcsMinmatchlen) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type LcsWithmatchlen Completed

func (c LcsWithmatchlen) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type Lindex Completed

func (b Builder) Lindex() (c Lindex) {
	c = Lindex{cs: get(), ks: b.ks, cf: readonly}
	c.cs.s = append(c.cs.s, "LINDEX")
	return c
}

func (c Lindex) Key(key string) LindexKey {
	if c.ks&NoSlot == NoSlot {
		c.ks = NoSlot | slot(key)
	} else {
		c.ks = check(c.ks, slot(key))
	}
	c.cs.s = append(c.cs.s, key)
	return (LindexKey)(c)
}

type LindexIndex Completed

func (c LindexIndex) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

func (c LindexIndex) Cache() Cacheable {
	c.cs.Build()
	return Cacheable(c)
}

type LindexKey Completed

func (c LindexKey) Index(index int64) LindexIndex {
	c.cs.s = append(c.cs.s, strconv.FormatInt(index, 10))
	return (LindexIndex)(c)
}

type Linsert Completed

func (b Builder) Linsert() (c Linsert) {
	c = Linsert{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "LINSERT")
	return c
}

func (c Linsert) Key(key string) LinsertKey {
	if c.ks&NoSlot == NoSlot {
		c.ks = NoSlot | slot(key)
	} else {
		c.ks = check(c.ks, slot(key))
	}
	c.cs.s = append(c.cs.s, key)
	return (LinsertKey)(c)
}

type LinsertElement Completed

func (c LinsertElement) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type LinsertKey Completed

func (c LinsertKey) Before() LinsertWhereBefore {
	c.cs.s = append(c.cs.s, "BEFORE")
	return (LinsertWhereBefore)(c)
}

func (c LinsertKey) After() LinsertWhereAfter {
	c.cs.s = append(c.cs.s, "AFTER")
	return (LinsertWhereAfter)(c)
}

type LinsertPivot Completed

func (c LinsertPivot) Element(element string) LinsertElement {
	c.cs.s = append(c.cs.s, element)
	return (LinsertElement)(c)
}

type LinsertWhereAfter Completed

func (c LinsertWhereAfter) Pivot(pivot string) LinsertPivot {
	c.cs.s = append(c.cs.s, pivot)
	return (LinsertPivot)(c)
}

type LinsertWhereBefore Completed

func (c LinsertWhereBefore) Pivot(pivot string) LinsertPivot {
	c.cs.s = append(c.cs.s, pivot)
	return (LinsertPivot)(c)
}

type Llen Completed

func (b Builder) Llen() (c Llen) {
	c = Llen{cs: get(), ks: b.ks, cf: readonly}
	c.cs.s = append(c.cs.s, "LLEN")
	return c
}

func (c Llen) Key(key string) LlenKey {
	if c.ks&NoSlot == NoSlot {
		c.ks = NoSlot | slot(key)
	} else {
		c.ks = check(c.ks, slot(key))
	}
	c.cs.s = append(c.cs.s, key)
	return (LlenKey)(c)
}

type LlenKey Completed

func (c LlenKey) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

func (c LlenKey) Cache() Cacheable {
	c.cs.Build()
	return Cacheable(c)
}

type Lmove Completed

func (b Builder) Lmove() (c Lmove) {
	c = Lmove{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "LMOVE")
	return c
}

func (c Lmove) Source(source string) LmoveSource {
	if c.ks&NoSlot == NoSlot {
		c.ks = NoSlot | slot(source)
	} else {
		c.ks = check(c.ks, slot(source))
	}
	c.cs.s = append(c.cs.s, source)
	return (LmoveSource)(c)
}

type LmoveDestination Completed

func (c LmoveDestination) Left() LmoveWherefromLeft {
	c.cs.s = append(c.cs.s, "LEFT")
	return (LmoveWherefromLeft)(c)
}

func (c LmoveDestination) Right() LmoveWherefromRight {
	c.cs.s = append(c.cs.s, "RIGHT")
	return (LmoveWherefromRight)(c)
}

type LmoveSource Completed

func (c LmoveSource) Destination(destination string) LmoveDestination {
	if c.ks&NoSlot == NoSlot {
		c.ks = NoSlot | slot(destination)
	} else {
		c.ks = check(c.ks, slot(destination))
	}
	c.cs.s = append(c.cs.s, destination)
	return (LmoveDestination)(c)
}

type LmoveWherefromLeft Completed

func (c LmoveWherefromLeft) Left() LmoveWheretoLeft {
	c.cs.s = append(c.cs.s, "LEFT")
	return (LmoveWheretoLeft)(c)
}

func (c LmoveWherefromLeft) Right() LmoveWheretoRight {
	c.cs.s = append(c.cs.s, "RIGHT")
	return (LmoveWheretoRight)(c)
}

type LmoveWherefromRight Completed

func (c LmoveWherefromRight) Left() LmoveWheretoLeft {
	c.cs.s = append(c.cs.s, "LEFT")
	return (LmoveWheretoLeft)(c)
}

func (c LmoveWherefromRight) Right() LmoveWheretoRight {
	c.cs.s = append(c.cs.s, "RIGHT")
	return (LmoveWheretoRight)(c)
}

type LmoveWheretoLeft Completed

func (c LmoveWheretoLeft) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type LmoveWheretoRight Completed

func (c LmoveWheretoRight) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type Lmpop Completed

func (b Builder) Lmpop() (c Lmpop) {
	c = Lmpop{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "LMPOP")
	return c
}

func (c Lmpop) Numkeys(numkeys int64) LmpopNumkeys {
	c.cs.s = append(c.cs.s, strconv.FormatInt(numkeys, 10))
	return (LmpopNumkeys)(c)
}

type LmpopCount Completed

func (c LmpopCount) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type LmpopKey Completed

func (c LmpopKey) Key(key ...string) LmpopKey {
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

func (c LmpopKey) Left() LmpopWhereLeft {
	c.cs.s = append(c.cs.s, "LEFT")
	return (LmpopWhereLeft)(c)
}

func (c LmpopKey) Right() LmpopWhereRight {
	c.cs.s = append(c.cs.s, "RIGHT")
	return (LmpopWhereRight)(c)
}

type LmpopNumkeys Completed

func (c LmpopNumkeys) Key(key ...string) LmpopKey {
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
	return (LmpopKey)(c)
}

type LmpopWhereLeft Completed

func (c LmpopWhereLeft) Count(count int64) LmpopCount {
	c.cs.s = append(c.cs.s, "COUNT", strconv.FormatInt(count, 10))
	return (LmpopCount)(c)
}

func (c LmpopWhereLeft) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type LmpopWhereRight Completed

func (c LmpopWhereRight) Count(count int64) LmpopCount {
	c.cs.s = append(c.cs.s, "COUNT", strconv.FormatInt(count, 10))
	return (LmpopCount)(c)
}

func (c LmpopWhereRight) Build() Completed {
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

type Lpop Completed

func (b Builder) Lpop() (c Lpop) {
	c = Lpop{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "LPOP")
	return c
}

func (c Lpop) Key(key string) LpopKey {
	if c.ks&NoSlot == NoSlot {
		c.ks = NoSlot | slot(key)
	} else {
		c.ks = check(c.ks, slot(key))
	}
	c.cs.s = append(c.cs.s, key)
	return (LpopKey)(c)
}

type LpopCount Completed

func (c LpopCount) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type LpopKey Completed

func (c LpopKey) Count(count int64) LpopCount {
	c.cs.s = append(c.cs.s, strconv.FormatInt(count, 10))
	return (LpopCount)(c)
}

func (c LpopKey) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type Lpos Completed

func (b Builder) Lpos() (c Lpos) {
	c = Lpos{cs: get(), ks: b.ks, cf: readonly}
	c.cs.s = append(c.cs.s, "LPOS")
	return c
}

func (c Lpos) Key(key string) LposKey {
	if c.ks&NoSlot == NoSlot {
		c.ks = NoSlot | slot(key)
	} else {
		c.ks = check(c.ks, slot(key))
	}
	c.cs.s = append(c.cs.s, key)
	return (LposKey)(c)
}

type LposCount Completed

func (c LposCount) Maxlen(len int64) LposMaxlen {
	c.cs.s = append(c.cs.s, "MAXLEN", strconv.FormatInt(len, 10))
	return (LposMaxlen)(c)
}

func (c LposCount) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

func (c LposCount) Cache() Cacheable {
	c.cs.Build()
	return Cacheable(c)
}

type LposElement Completed

func (c LposElement) Rank(rank int64) LposRank {
	c.cs.s = append(c.cs.s, "RANK", strconv.FormatInt(rank, 10))
	return (LposRank)(c)
}

func (c LposElement) Count(numMatches int64) LposCount {
	c.cs.s = append(c.cs.s, "COUNT", strconv.FormatInt(numMatches, 10))
	return (LposCount)(c)
}

func (c LposElement) Maxlen(len int64) LposMaxlen {
	c.cs.s = append(c.cs.s, "MAXLEN", strconv.FormatInt(len, 10))
	return (LposMaxlen)(c)
}

func (c LposElement) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

func (c LposElement) Cache() Cacheable {
	c.cs.Build()
	return Cacheable(c)
}

type LposKey Completed

func (c LposKey) Element(element string) LposElement {
	c.cs.s = append(c.cs.s, element)
	return (LposElement)(c)
}

type LposMaxlen Completed

func (c LposMaxlen) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

func (c LposMaxlen) Cache() Cacheable {
	c.cs.Build()
	return Cacheable(c)
}

type LposRank Completed

func (c LposRank) Count(numMatches int64) LposCount {
	c.cs.s = append(c.cs.s, "COUNT", strconv.FormatInt(numMatches, 10))
	return (LposCount)(c)
}

func (c LposRank) Maxlen(len int64) LposMaxlen {
	c.cs.s = append(c.cs.s, "MAXLEN", strconv.FormatInt(len, 10))
	return (LposMaxlen)(c)
}

func (c LposRank) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

func (c LposRank) Cache() Cacheable {
	c.cs.Build()
	return Cacheable(c)
}

type Lpush Completed

func (b Builder) Lpush() (c Lpush) {
	c = Lpush{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "LPUSH")
	return c
}

func (c Lpush) Key(key string) LpushKey {
	if c.ks&NoSlot == NoSlot {
		c.ks = NoSlot | slot(key)
	} else {
		c.ks = check(c.ks, slot(key))
	}
	c.cs.s = append(c.cs.s, key)
	return (LpushKey)(c)
}

type LpushElement Completed

func (c LpushElement) Element(element ...string) LpushElement {
	c.cs.s = append(c.cs.s, element...)
	return c
}

func (c LpushElement) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type LpushKey Completed

func (c LpushKey) Element(element ...string) LpushElement {
	c.cs.s = append(c.cs.s, element...)
	return (LpushElement)(c)
}

type Lpushx Completed

func (b Builder) Lpushx() (c Lpushx) {
	c = Lpushx{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "LPUSHX")
	return c
}

func (c Lpushx) Key(key string) LpushxKey {
	if c.ks&NoSlot == NoSlot {
		c.ks = NoSlot | slot(key)
	} else {
		c.ks = check(c.ks, slot(key))
	}
	c.cs.s = append(c.cs.s, key)
	return (LpushxKey)(c)
}

type LpushxElement Completed

func (c LpushxElement) Element(element ...string) LpushxElement {
	c.cs.s = append(c.cs.s, element...)
	return c
}

func (c LpushxElement) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type LpushxKey Completed

func (c LpushxKey) Element(element ...string) LpushxElement {
	c.cs.s = append(c.cs.s, element...)
	return (LpushxElement)(c)
}

type Lrange Completed

func (b Builder) Lrange() (c Lrange) {
	c = Lrange{cs: get(), ks: b.ks, cf: readonly}
	c.cs.s = append(c.cs.s, "LRANGE")
	return c
}

func (c Lrange) Key(key string) LrangeKey {
	if c.ks&NoSlot == NoSlot {
		c.ks = NoSlot | slot(key)
	} else {
		c.ks = check(c.ks, slot(key))
	}
	c.cs.s = append(c.cs.s, key)
	return (LrangeKey)(c)
}

type LrangeKey Completed

func (c LrangeKey) Start(start int64) LrangeStart {
	c.cs.s = append(c.cs.s, strconv.FormatInt(start, 10))
	return (LrangeStart)(c)
}

type LrangeStart Completed

func (c LrangeStart) Stop(stop int64) LrangeStop {
	c.cs.s = append(c.cs.s, strconv.FormatInt(stop, 10))
	return (LrangeStop)(c)
}

type LrangeStop Completed

func (c LrangeStop) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

func (c LrangeStop) Cache() Cacheable {
	c.cs.Build()
	return Cacheable(c)
}

type Lrem Completed

func (b Builder) Lrem() (c Lrem) {
	c = Lrem{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "LREM")
	return c
}

func (c Lrem) Key(key string) LremKey {
	if c.ks&NoSlot == NoSlot {
		c.ks = NoSlot | slot(key)
	} else {
		c.ks = check(c.ks, slot(key))
	}
	c.cs.s = append(c.cs.s, key)
	return (LremKey)(c)
}

type LremCount Completed

func (c LremCount) Element(element string) LremElement {
	c.cs.s = append(c.cs.s, element)
	return (LremElement)(c)
}

type LremElement Completed

func (c LremElement) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type LremKey Completed

func (c LremKey) Count(count int64) LremCount {
	c.cs.s = append(c.cs.s, strconv.FormatInt(count, 10))
	return (LremCount)(c)
}

type Lset Completed

func (b Builder) Lset() (c Lset) {
	c = Lset{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "LSET")
	return c
}

func (c Lset) Key(key string) LsetKey {
	if c.ks&NoSlot == NoSlot {
		c.ks = NoSlot | slot(key)
	} else {
		c.ks = check(c.ks, slot(key))
	}
	c.cs.s = append(c.cs.s, key)
	return (LsetKey)(c)
}

type LsetElement Completed

func (c LsetElement) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type LsetIndex Completed

func (c LsetIndex) Element(element string) LsetElement {
	c.cs.s = append(c.cs.s, element)
	return (LsetElement)(c)
}

type LsetKey Completed

func (c LsetKey) Index(index int64) LsetIndex {
	c.cs.s = append(c.cs.s, strconv.FormatInt(index, 10))
	return (LsetIndex)(c)
}

type Ltrim Completed

func (b Builder) Ltrim() (c Ltrim) {
	c = Ltrim{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "LTRIM")
	return c
}

func (c Ltrim) Key(key string) LtrimKey {
	if c.ks&NoSlot == NoSlot {
		c.ks = NoSlot | slot(key)
	} else {
		c.ks = check(c.ks, slot(key))
	}
	c.cs.s = append(c.cs.s, key)
	return (LtrimKey)(c)
}

type LtrimKey Completed

func (c LtrimKey) Start(start int64) LtrimStart {
	c.cs.s = append(c.cs.s, strconv.FormatInt(start, 10))
	return (LtrimStart)(c)
}

type LtrimStart Completed

func (c LtrimStart) Stop(stop int64) LtrimStop {
	c.cs.s = append(c.cs.s, strconv.FormatInt(stop, 10))
	return (LtrimStop)(c)
}

type LtrimStop Completed

func (c LtrimStop) Build() Completed {
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

type Mget Completed

func (b Builder) Mget() (c Mget) {
	c = Mget{cs: get(), ks: b.ks, cf: mtGetTag}
	c.cs.s = append(c.cs.s, "MGET")
	return c
}

func (c Mget) Key(key ...string) MgetKey {
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
	return (MgetKey)(c)
}

type MgetKey Completed

func (c MgetKey) Key(key ...string) MgetKey {
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

func (c MgetKey) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

func (c MgetKey) Cache() Cacheable {
	c.cs.Build()
	return Cacheable(c)
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

type Mset Completed

func (b Builder) Mset() (c Mset) {
	c = Mset{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "MSET")
	return c
}

func (c Mset) KeyValue() MsetKeyValue {
	return (MsetKeyValue)(c)
}

type MsetKeyValue Completed

func (c MsetKeyValue) KeyValue(key string, value string) MsetKeyValue {
	if c.ks&NoSlot == NoSlot {
		c.ks = NoSlot | slot(key)
	} else {
		c.ks = check(c.ks, slot(key))
	}
	c.cs.s = append(c.cs.s, key, value)
	return c
}

func (c MsetKeyValue) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type Msetnx Completed

func (b Builder) Msetnx() (c Msetnx) {
	c = Msetnx{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "MSETNX")
	return c
}

func (c Msetnx) KeyValue() MsetnxKeyValue {
	return (MsetnxKeyValue)(c)
}

type MsetnxKeyValue Completed

func (c MsetnxKeyValue) KeyValue(key string, value string) MsetnxKeyValue {
	if c.ks&NoSlot == NoSlot {
		c.ks = NoSlot | slot(key)
	} else {
		c.ks = check(c.ks, slot(key))
	}
	c.cs.s = append(c.cs.s, key, value)
	return c
}

func (c MsetnxKeyValue) Build() Completed {
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

type Pfadd Completed

func (b Builder) Pfadd() (c Pfadd) {
	c = Pfadd{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "PFADD")
	return c
}

func (c Pfadd) Key(key string) PfaddKey {
	if c.ks&NoSlot == NoSlot {
		c.ks = NoSlot | slot(key)
	} else {
		c.ks = check(c.ks, slot(key))
	}
	c.cs.s = append(c.cs.s, key)
	return (PfaddKey)(c)
}

type PfaddElement Completed

func (c PfaddElement) Element(element ...string) PfaddElement {
	c.cs.s = append(c.cs.s, element...)
	return c
}

func (c PfaddElement) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type PfaddKey Completed

func (c PfaddKey) Element(element ...string) PfaddElement {
	c.cs.s = append(c.cs.s, element...)
	return (PfaddElement)(c)
}

func (c PfaddKey) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type Pfcount Completed

func (b Builder) Pfcount() (c Pfcount) {
	c = Pfcount{cs: get(), ks: b.ks, cf: readonly}
	c.cs.s = append(c.cs.s, "PFCOUNT")
	return c
}

func (c Pfcount) Key(key ...string) PfcountKey {
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
	return (PfcountKey)(c)
}

type PfcountKey Completed

func (c PfcountKey) Key(key ...string) PfcountKey {
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

func (c PfcountKey) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type Pfmerge Completed

func (b Builder) Pfmerge() (c Pfmerge) {
	c = Pfmerge{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "PFMERGE")
	return c
}

func (c Pfmerge) Destkey(destkey string) PfmergeDestkey {
	if c.ks&NoSlot == NoSlot {
		c.ks = NoSlot | slot(destkey)
	} else {
		c.ks = check(c.ks, slot(destkey))
	}
	c.cs.s = append(c.cs.s, destkey)
	return (PfmergeDestkey)(c)
}

type PfmergeDestkey Completed

func (c PfmergeDestkey) Sourcekey(sourcekey ...string) PfmergeSourcekey {
	if c.ks&NoSlot == NoSlot {
		for _, k := range sourcekey {
			c.ks = NoSlot | slot(k)
			break
		}
	} else {
		for _, k := range sourcekey {
			c.ks = check(c.ks, slot(k))
		}
	}
	c.cs.s = append(c.cs.s, sourcekey...)
	return (PfmergeSourcekey)(c)
}

func (c PfmergeDestkey) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type PfmergeSourcekey Completed

func (c PfmergeSourcekey) Sourcekey(sourcekey ...string) PfmergeSourcekey {
	if c.ks&NoSlot == NoSlot {
		for _, k := range sourcekey {
			c.ks = NoSlot | slot(k)
			break
		}
	} else {
		for _, k := range sourcekey {
			c.ks = check(c.ks, slot(k))
		}
	}
	c.cs.s = append(c.cs.s, sourcekey...)
	return c
}

func (c PfmergeSourcekey) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type Ping Completed

func (b Builder) Ping() (c Ping) {
	c = Ping{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "PING")
	return c
}

func (c Ping) Message(message string) PingMessage {
	c.cs.s = append(c.cs.s, message)
	return (PingMessage)(c)
}

func (c Ping) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type PingMessage Completed

func (c PingMessage) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type Psetex Completed

func (b Builder) Psetex() (c Psetex) {
	c = Psetex{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "PSETEX")
	return c
}

func (c Psetex) Key(key string) PsetexKey {
	if c.ks&NoSlot == NoSlot {
		c.ks = NoSlot | slot(key)
	} else {
		c.ks = check(c.ks, slot(key))
	}
	c.cs.s = append(c.cs.s, key)
	return (PsetexKey)(c)
}

type PsetexKey Completed

func (c PsetexKey) Milliseconds(milliseconds int64) PsetexMilliseconds {
	c.cs.s = append(c.cs.s, strconv.FormatInt(milliseconds, 10))
	return (PsetexMilliseconds)(c)
}

type PsetexMilliseconds Completed

func (c PsetexMilliseconds) Value(value string) PsetexValue {
	c.cs.s = append(c.cs.s, value)
	return (PsetexValue)(c)
}

type PsetexValue Completed

func (c PsetexValue) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type Psubscribe Completed

func (b Builder) Psubscribe() (c Psubscribe) {
	c = Psubscribe{cs: get(), ks: b.ks, cf: noRetTag}
	c.cs.s = append(c.cs.s, "PSUBSCRIBE")
	return c
}

func (c Psubscribe) Pattern(pattern ...string) PsubscribePattern {
	c.cs.s = append(c.cs.s, pattern...)
	return (PsubscribePattern)(c)
}

type PsubscribePattern Completed

func (c PsubscribePattern) Pattern(pattern ...string) PsubscribePattern {
	c.cs.s = append(c.cs.s, pattern...)
	return c
}

func (c PsubscribePattern) Build() Completed {
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

type Publish Completed

func (b Builder) Publish() (c Publish) {
	c = Publish{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "PUBLISH")
	return c
}

func (c Publish) Channel(channel string) PublishChannel {
	c.cs.s = append(c.cs.s, channel)
	return (PublishChannel)(c)
}

type PublishChannel Completed

func (c PublishChannel) Message(message string) PublishMessage {
	c.cs.s = append(c.cs.s, message)
	return (PublishMessage)(c)
}

type PublishMessage Completed

func (c PublishMessage) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type PubsubChannels Completed

func (b Builder) PubsubChannels() (c PubsubChannels) {
	c = PubsubChannels{cs: get(), ks: b.ks, cf: readonly}
	c.cs.s = append(c.cs.s, "PUBSUB", "CHANNELS")
	return c
}

func (c PubsubChannels) Pattern(pattern string) PubsubChannelsPattern {
	c.cs.s = append(c.cs.s, pattern)
	return (PubsubChannelsPattern)(c)
}

func (c PubsubChannels) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type PubsubChannelsPattern Completed

func (c PubsubChannelsPattern) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type PubsubHelp Completed

func (b Builder) PubsubHelp() (c PubsubHelp) {
	c = PubsubHelp{cs: get(), ks: b.ks, cf: readonly}
	c.cs.s = append(c.cs.s, "PUBSUB", "HELP")
	return c
}

func (c PubsubHelp) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type PubsubNumpat Completed

func (b Builder) PubsubNumpat() (c PubsubNumpat) {
	c = PubsubNumpat{cs: get(), ks: b.ks, cf: readonly}
	c.cs.s = append(c.cs.s, "PUBSUB", "NUMPAT")
	return c
}

func (c PubsubNumpat) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type PubsubNumsub Completed

func (b Builder) PubsubNumsub() (c PubsubNumsub) {
	c = PubsubNumsub{cs: get(), ks: b.ks, cf: readonly}
	c.cs.s = append(c.cs.s, "PUBSUB", "NUMSUB")
	return c
}

func (c PubsubNumsub) Channel(channel ...string) PubsubNumsubChannel {
	c.cs.s = append(c.cs.s, channel...)
	return (PubsubNumsubChannel)(c)
}

func (c PubsubNumsub) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type PubsubNumsubChannel Completed

func (c PubsubNumsubChannel) Channel(channel ...string) PubsubNumsubChannel {
	c.cs.s = append(c.cs.s, channel...)
	return c
}

func (c PubsubNumsubChannel) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type PubsubShardchannels Completed

func (b Builder) PubsubShardchannels() (c PubsubShardchannels) {
	c = PubsubShardchannels{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "PUBSUB", "SHARDCHANNELS")
	return c
}

func (c PubsubShardchannels) Pattern(pattern string) PubsubShardchannelsPattern {
	c.cs.s = append(c.cs.s, pattern)
	return (PubsubShardchannelsPattern)(c)
}

func (c PubsubShardchannels) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type PubsubShardchannelsPattern Completed

func (c PubsubShardchannelsPattern) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type PubsubShardnumsub Completed

func (b Builder) PubsubShardnumsub() (c PubsubShardnumsub) {
	c = PubsubShardnumsub{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "PUBSUB", "SHARDNUMSUB")
	return c
}

func (c PubsubShardnumsub) Channel(channel ...string) PubsubShardnumsubChannel {
	c.cs.s = append(c.cs.s, channel...)
	return (PubsubShardnumsubChannel)(c)
}

func (c PubsubShardnumsub) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type PubsubShardnumsubChannel Completed

func (c PubsubShardnumsubChannel) Channel(channel ...string) PubsubShardnumsubChannel {
	c.cs.s = append(c.cs.s, channel...)
	return c
}

func (c PubsubShardnumsubChannel) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type Punsubscribe Completed

func (b Builder) Punsubscribe() (c Punsubscribe) {
	c = Punsubscribe{cs: get(), ks: b.ks, cf: noRetTag}
	c.cs.s = append(c.cs.s, "PUNSUBSCRIBE")
	return c
}

func (c Punsubscribe) Pattern(pattern ...string) PunsubscribePattern {
	c.cs.s = append(c.cs.s, pattern...)
	return (PunsubscribePattern)(c)
}

func (c Punsubscribe) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type PunsubscribePattern Completed

func (c PunsubscribePattern) Pattern(pattern ...string) PunsubscribePattern {
	c.cs.s = append(c.cs.s, pattern...)
	return c
}

func (c PunsubscribePattern) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type Quit Completed

func (b Builder) Quit() (c Quit) {
	c = Quit{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "QUIT")
	return c
}

func (c Quit) Build() Completed {
	c.cs.Build()
	return Completed(c)
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

type Reset Completed

func (b Builder) Reset() (c Reset) {
	c = Reset{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "RESET")
	return c
}

func (c Reset) Build() Completed {
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

type RgAbortexecution Completed

func (b Builder) RgAbortexecution() (c RgAbortexecution) {
	c = RgAbortexecution{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "RG.ABORTEXECUTION")
	return c
}

func (c RgAbortexecution) Id(id string) RgAbortexecutionId {
	c.cs.s = append(c.cs.s, id)
	return (RgAbortexecutionId)(c)
}

type RgAbortexecutionId Completed

func (c RgAbortexecutionId) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type RgConfigget Completed

func (b Builder) RgConfigget() (c RgConfigget) {
	c = RgConfigget{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "RG.CONFIGGET")
	return c
}

func (c RgConfigget) Key(key ...string) RgConfiggetKey {
	c.cs.s = append(c.cs.s, key...)
	return (RgConfiggetKey)(c)
}

type RgConfiggetKey Completed

func (c RgConfiggetKey) Key(key ...string) RgConfiggetKey {
	c.cs.s = append(c.cs.s, key...)
	return c
}

func (c RgConfiggetKey) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type RgConfigset Completed

func (b Builder) RgConfigset() (c RgConfigset) {
	c = RgConfigset{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "RG.CONFIGSET")
	return c
}

func (c RgConfigset) KeyValue() RgConfigsetKeyValue {
	return (RgConfigsetKeyValue)(c)
}

type RgConfigsetKeyValue Completed

func (c RgConfigsetKeyValue) KeyValue(key string, value string) RgConfigsetKeyValue {
	c.cs.s = append(c.cs.s, key, value)
	return c
}

func (c RgConfigsetKeyValue) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type RgDropexecution Completed

func (b Builder) RgDropexecution() (c RgDropexecution) {
	c = RgDropexecution{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "RG.DROPEXECUTION")
	return c
}

func (c RgDropexecution) Id(id string) RgDropexecutionId {
	c.cs.s = append(c.cs.s, id)
	return (RgDropexecutionId)(c)
}

type RgDropexecutionId Completed

func (c RgDropexecutionId) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type RgDumpexecutions Completed

func (b Builder) RgDumpexecutions() (c RgDumpexecutions) {
	c = RgDumpexecutions{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "RG.DUMPEXECUTIONS")
	return c
}

func (c RgDumpexecutions) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type RgDumpregistrations Completed

func (b Builder) RgDumpregistrations() (c RgDumpregistrations) {
	c = RgDumpregistrations{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "RG.DUMPREGISTRATIONS")
	return c
}

func (c RgDumpregistrations) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type RgGetexecution Completed

func (b Builder) RgGetexecution() (c RgGetexecution) {
	c = RgGetexecution{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "RG.GETEXECUTION")
	return c
}

func (c RgGetexecution) Id(id string) RgGetexecutionId {
	c.cs.s = append(c.cs.s, id)
	return (RgGetexecutionId)(c)
}

type RgGetexecutionId Completed

func (c RgGetexecutionId) Shard() RgGetexecutionModeShard {
	c.cs.s = append(c.cs.s, "SHARD")
	return (RgGetexecutionModeShard)(c)
}

func (c RgGetexecutionId) Cluster() RgGetexecutionModeCluster {
	c.cs.s = append(c.cs.s, "CLUSTER")
	return (RgGetexecutionModeCluster)(c)
}

func (c RgGetexecutionId) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type RgGetexecutionModeCluster Completed

func (c RgGetexecutionModeCluster) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type RgGetexecutionModeShard Completed

func (c RgGetexecutionModeShard) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type RgGetresults Completed

func (b Builder) RgGetresults() (c RgGetresults) {
	c = RgGetresults{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "RG.GETRESULTS")
	return c
}

func (c RgGetresults) Id(id string) RgGetresultsId {
	c.cs.s = append(c.cs.s, id)
	return (RgGetresultsId)(c)
}

type RgGetresultsId Completed

func (c RgGetresultsId) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type RgGetresultsblocking Completed

func (b Builder) RgGetresultsblocking() (c RgGetresultsblocking) {
	c = RgGetresultsblocking{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "RG.GETRESULTSBLOCKING")
	return c
}

func (c RgGetresultsblocking) Id(id string) RgGetresultsblockingId {
	c.cs.s = append(c.cs.s, id)
	return (RgGetresultsblockingId)(c)
}

type RgGetresultsblockingId Completed

func (c RgGetresultsblockingId) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type RgInfocluster Completed

func (b Builder) RgInfocluster() (c RgInfocluster) {
	c = RgInfocluster{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "RG.INFOCLUSTER")
	return c
}

func (c RgInfocluster) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type RgPydumpreqs Completed

func (b Builder) RgPydumpreqs() (c RgPydumpreqs) {
	c = RgPydumpreqs{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "RG.PYDUMPREQS")
	return c
}

func (c RgPydumpreqs) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type RgPyexecute Completed

func (b Builder) RgPyexecute() (c RgPyexecute) {
	c = RgPyexecute{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "RG.PYEXECUTE")
	return c
}

func (c RgPyexecute) Function(function string) RgPyexecuteFunction {
	c.cs.s = append(c.cs.s, function)
	return (RgPyexecuteFunction)(c)
}

type RgPyexecuteDescription Completed

func (c RgPyexecuteDescription) Upgrade() RgPyexecuteUpgrade {
	c.cs.s = append(c.cs.s, "UPGRADE")
	return (RgPyexecuteUpgrade)(c)
}

func (c RgPyexecuteDescription) ReplaceWith(replaceWith string) RgPyexecuteReplaceWith {
	c.cs.s = append(c.cs.s, "REPLACE_WITH", replaceWith)
	return (RgPyexecuteReplaceWith)(c)
}

func (c RgPyexecuteDescription) Requirements(requirement ...string) RgPyexecuteRequirementsRequirements {
	c.cs.s = append(c.cs.s, "REQUIREMENTS")
	c.cs.s = append(c.cs.s, requirement...)
	return (RgPyexecuteRequirementsRequirements)(c)
}

func (c RgPyexecuteDescription) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type RgPyexecuteFunction Completed

func (c RgPyexecuteFunction) Unblocking() RgPyexecuteUnblocking {
	c.cs.s = append(c.cs.s, "UNBLOCKING")
	return (RgPyexecuteUnblocking)(c)
}

func (c RgPyexecuteFunction) Id(id string) RgPyexecuteId {
	c.cs.s = append(c.cs.s, "ID", id)
	return (RgPyexecuteId)(c)
}

func (c RgPyexecuteFunction) Description(description string) RgPyexecuteDescription {
	c.cs.s = append(c.cs.s, "DESCRIPTION", description)
	return (RgPyexecuteDescription)(c)
}

func (c RgPyexecuteFunction) Upgrade() RgPyexecuteUpgrade {
	c.cs.s = append(c.cs.s, "UPGRADE")
	return (RgPyexecuteUpgrade)(c)
}

func (c RgPyexecuteFunction) ReplaceWith(replaceWith string) RgPyexecuteReplaceWith {
	c.cs.s = append(c.cs.s, "REPLACE_WITH", replaceWith)
	return (RgPyexecuteReplaceWith)(c)
}

func (c RgPyexecuteFunction) Requirements(requirement ...string) RgPyexecuteRequirementsRequirements {
	c.cs.s = append(c.cs.s, "REQUIREMENTS")
	c.cs.s = append(c.cs.s, requirement...)
	return (RgPyexecuteRequirementsRequirements)(c)
}

func (c RgPyexecuteFunction) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type RgPyexecuteId Completed

func (c RgPyexecuteId) Description(description string) RgPyexecuteDescription {
	c.cs.s = append(c.cs.s, "DESCRIPTION", description)
	return (RgPyexecuteDescription)(c)
}

func (c RgPyexecuteId) Upgrade() RgPyexecuteUpgrade {
	c.cs.s = append(c.cs.s, "UPGRADE")
	return (RgPyexecuteUpgrade)(c)
}

func (c RgPyexecuteId) ReplaceWith(replaceWith string) RgPyexecuteReplaceWith {
	c.cs.s = append(c.cs.s, "REPLACE_WITH", replaceWith)
	return (RgPyexecuteReplaceWith)(c)
}

func (c RgPyexecuteId) Requirements(requirement ...string) RgPyexecuteRequirementsRequirements {
	c.cs.s = append(c.cs.s, "REQUIREMENTS")
	c.cs.s = append(c.cs.s, requirement...)
	return (RgPyexecuteRequirementsRequirements)(c)
}

func (c RgPyexecuteId) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type RgPyexecuteReplaceWith Completed

func (c RgPyexecuteReplaceWith) Requirements(requirement ...string) RgPyexecuteRequirementsRequirements {
	c.cs.s = append(c.cs.s, "REQUIREMENTS")
	c.cs.s = append(c.cs.s, requirement...)
	return (RgPyexecuteRequirementsRequirements)(c)
}

func (c RgPyexecuteReplaceWith) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type RgPyexecuteRequirementsRequirements Completed

func (c RgPyexecuteRequirementsRequirements) Requirements(requirement ...string) RgPyexecuteRequirementsRequirements {
	c.cs.s = append(c.cs.s, "REQUIREMENTS")
	c.cs.s = append(c.cs.s, requirement...)
	return c
}

func (c RgPyexecuteRequirementsRequirements) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type RgPyexecuteUnblocking Completed

func (c RgPyexecuteUnblocking) Id(id string) RgPyexecuteId {
	c.cs.s = append(c.cs.s, "ID", id)
	return (RgPyexecuteId)(c)
}

func (c RgPyexecuteUnblocking) Description(description string) RgPyexecuteDescription {
	c.cs.s = append(c.cs.s, "DESCRIPTION", description)
	return (RgPyexecuteDescription)(c)
}

func (c RgPyexecuteUnblocking) Upgrade() RgPyexecuteUpgrade {
	c.cs.s = append(c.cs.s, "UPGRADE")
	return (RgPyexecuteUpgrade)(c)
}

func (c RgPyexecuteUnblocking) ReplaceWith(replaceWith string) RgPyexecuteReplaceWith {
	c.cs.s = append(c.cs.s, "REPLACE_WITH", replaceWith)
	return (RgPyexecuteReplaceWith)(c)
}

func (c RgPyexecuteUnblocking) Requirements(requirement ...string) RgPyexecuteRequirementsRequirements {
	c.cs.s = append(c.cs.s, "REQUIREMENTS")
	c.cs.s = append(c.cs.s, requirement...)
	return (RgPyexecuteRequirementsRequirements)(c)
}

func (c RgPyexecuteUnblocking) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type RgPyexecuteUpgrade Completed

func (c RgPyexecuteUpgrade) ReplaceWith(replaceWith string) RgPyexecuteReplaceWith {
	c.cs.s = append(c.cs.s, "REPLACE_WITH", replaceWith)
	return (RgPyexecuteReplaceWith)(c)
}

func (c RgPyexecuteUpgrade) Requirements(requirement ...string) RgPyexecuteRequirementsRequirements {
	c.cs.s = append(c.cs.s, "REQUIREMENTS")
	c.cs.s = append(c.cs.s, requirement...)
	return (RgPyexecuteRequirementsRequirements)(c)
}

func (c RgPyexecuteUpgrade) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type RgPystats Completed

func (b Builder) RgPystats() (c RgPystats) {
	c = RgPystats{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "RG.PYSTATS")
	return c
}

func (c RgPystats) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type RgRefreshcluster Completed

func (b Builder) RgRefreshcluster() (c RgRefreshcluster) {
	c = RgRefreshcluster{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "RG.REFRESHCLUSTER")
	return c
}

func (c RgRefreshcluster) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type RgTrigger Completed

func (b Builder) RgTrigger() (c RgTrigger) {
	c = RgTrigger{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "RG.TRIGGER")
	return c
}

func (c RgTrigger) Trigger(trigger string) RgTriggerTrigger {
	c.cs.s = append(c.cs.s, trigger)
	return (RgTriggerTrigger)(c)
}

type RgTriggerArgument Completed

func (c RgTriggerArgument) Argument(argument ...string) RgTriggerArgument {
	c.cs.s = append(c.cs.s, argument...)
	return c
}

func (c RgTriggerArgument) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type RgTriggerTrigger Completed

func (c RgTriggerTrigger) Argument(argument ...string) RgTriggerArgument {
	c.cs.s = append(c.cs.s, argument...)
	return (RgTriggerArgument)(c)
}

type RgUnregister Completed

func (b Builder) RgUnregister() (c RgUnregister) {
	c = RgUnregister{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "RG.UNREGISTER")
	return c
}

func (c RgUnregister) Id(id string) RgUnregisterId {
	c.cs.s = append(c.cs.s, id)
	return (RgUnregisterId)(c)
}

type RgUnregisterId Completed

func (c RgUnregisterId) Build() Completed {
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

type Rpop Completed

func (b Builder) Rpop() (c Rpop) {
	c = Rpop{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "RPOP")
	return c
}

func (c Rpop) Key(key string) RpopKey {
	if c.ks&NoSlot == NoSlot {
		c.ks = NoSlot | slot(key)
	} else {
		c.ks = check(c.ks, slot(key))
	}
	c.cs.s = append(c.cs.s, key)
	return (RpopKey)(c)
}

type RpopCount Completed

func (c RpopCount) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type RpopKey Completed

func (c RpopKey) Count(count int64) RpopCount {
	c.cs.s = append(c.cs.s, strconv.FormatInt(count, 10))
	return (RpopCount)(c)
}

func (c RpopKey) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type Rpoplpush Completed

func (b Builder) Rpoplpush() (c Rpoplpush) {
	c = Rpoplpush{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "RPOPLPUSH")
	return c
}

func (c Rpoplpush) Source(source string) RpoplpushSource {
	if c.ks&NoSlot == NoSlot {
		c.ks = NoSlot | slot(source)
	} else {
		c.ks = check(c.ks, slot(source))
	}
	c.cs.s = append(c.cs.s, source)
	return (RpoplpushSource)(c)
}

type RpoplpushDestination Completed

func (c RpoplpushDestination) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type RpoplpushSource Completed

func (c RpoplpushSource) Destination(destination string) RpoplpushDestination {
	if c.ks&NoSlot == NoSlot {
		c.ks = NoSlot | slot(destination)
	} else {
		c.ks = check(c.ks, slot(destination))
	}
	c.cs.s = append(c.cs.s, destination)
	return (RpoplpushDestination)(c)
}

type Rpush Completed

func (b Builder) Rpush() (c Rpush) {
	c = Rpush{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "RPUSH")
	return c
}

func (c Rpush) Key(key string) RpushKey {
	if c.ks&NoSlot == NoSlot {
		c.ks = NoSlot | slot(key)
	} else {
		c.ks = check(c.ks, slot(key))
	}
	c.cs.s = append(c.cs.s, key)
	return (RpushKey)(c)
}

type RpushElement Completed

func (c RpushElement) Element(element ...string) RpushElement {
	c.cs.s = append(c.cs.s, element...)
	return c
}

func (c RpushElement) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type RpushKey Completed

func (c RpushKey) Element(element ...string) RpushElement {
	c.cs.s = append(c.cs.s, element...)
	return (RpushElement)(c)
}

type Rpushx Completed

func (b Builder) Rpushx() (c Rpushx) {
	c = Rpushx{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "RPUSHX")
	return c
}

func (c Rpushx) Key(key string) RpushxKey {
	if c.ks&NoSlot == NoSlot {
		c.ks = NoSlot | slot(key)
	} else {
		c.ks = check(c.ks, slot(key))
	}
	c.cs.s = append(c.cs.s, key)
	return (RpushxKey)(c)
}

type RpushxElement Completed

func (c RpushxElement) Element(element ...string) RpushxElement {
	c.cs.s = append(c.cs.s, element...)
	return c
}

func (c RpushxElement) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type RpushxKey Completed

func (c RpushxKey) Element(element ...string) RpushxElement {
	c.cs.s = append(c.cs.s, element...)
	return (RpushxElement)(c)
}

type Sadd Completed

func (b Builder) Sadd() (c Sadd) {
	c = Sadd{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "SADD")
	return c
}

func (c Sadd) Key(key string) SaddKey {
	if c.ks&NoSlot == NoSlot {
		c.ks = NoSlot | slot(key)
	} else {
		c.ks = check(c.ks, slot(key))
	}
	c.cs.s = append(c.cs.s, key)
	return (SaddKey)(c)
}

type SaddKey Completed

func (c SaddKey) Member(member ...string) SaddMember {
	c.cs.s = append(c.cs.s, member...)
	return (SaddMember)(c)
}

type SaddMember Completed

func (c SaddMember) Member(member ...string) SaddMember {
	c.cs.s = append(c.cs.s, member...)
	return c
}

func (c SaddMember) Build() Completed {
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

type Scard Completed

func (b Builder) Scard() (c Scard) {
	c = Scard{cs: get(), ks: b.ks, cf: readonly}
	c.cs.s = append(c.cs.s, "SCARD")
	return c
}

func (c Scard) Key(key string) ScardKey {
	if c.ks&NoSlot == NoSlot {
		c.ks = NoSlot | slot(key)
	} else {
		c.ks = check(c.ks, slot(key))
	}
	c.cs.s = append(c.cs.s, key)
	return (ScardKey)(c)
}

type ScardKey Completed

func (c ScardKey) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

func (c ScardKey) Cache() Cacheable {
	c.cs.Build()
	return Cacheable(c)
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

type Sdiff Completed

func (b Builder) Sdiff() (c Sdiff) {
	c = Sdiff{cs: get(), ks: b.ks, cf: readonly}
	c.cs.s = append(c.cs.s, "SDIFF")
	return c
}

func (c Sdiff) Key(key ...string) SdiffKey {
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
	return (SdiffKey)(c)
}

type SdiffKey Completed

func (c SdiffKey) Key(key ...string) SdiffKey {
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

func (c SdiffKey) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type Sdiffstore Completed

func (b Builder) Sdiffstore() (c Sdiffstore) {
	c = Sdiffstore{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "SDIFFSTORE")
	return c
}

func (c Sdiffstore) Destination(destination string) SdiffstoreDestination {
	if c.ks&NoSlot == NoSlot {
		c.ks = NoSlot | slot(destination)
	} else {
		c.ks = check(c.ks, slot(destination))
	}
	c.cs.s = append(c.cs.s, destination)
	return (SdiffstoreDestination)(c)
}

type SdiffstoreDestination Completed

func (c SdiffstoreDestination) Key(key ...string) SdiffstoreKey {
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
	return (SdiffstoreKey)(c)
}

type SdiffstoreKey Completed

func (c SdiffstoreKey) Key(key ...string) SdiffstoreKey {
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

func (c SdiffstoreKey) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type Select Completed

func (b Builder) Select() (c Select) {
	c = Select{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "SELECT")
	return c
}

func (c Select) Index(index int64) SelectIndex {
	c.cs.s = append(c.cs.s, strconv.FormatInt(index, 10))
	return (SelectIndex)(c)
}

type SelectIndex Completed

func (c SelectIndex) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type SentinelFailover Completed

func (b Builder) SentinelFailover() (c SentinelFailover) {
	c = SentinelFailover{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "SENTINEL", "FAILOVER")
	return c
}

func (c SentinelFailover) Master(master string) SentinelFailoverMaster {
	c.cs.s = append(c.cs.s, master)
	return (SentinelFailoverMaster)(c)
}

type SentinelFailoverMaster Completed

func (c SentinelFailoverMaster) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type SentinelGetMasterAddrByName Completed

func (b Builder) SentinelGetMasterAddrByName() (c SentinelGetMasterAddrByName) {
	c = SentinelGetMasterAddrByName{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "SENTINEL", "GET-MASTER-ADDR-BY-NAME")
	return c
}

func (c SentinelGetMasterAddrByName) Master(master string) SentinelGetMasterAddrByNameMaster {
	c.cs.s = append(c.cs.s, master)
	return (SentinelGetMasterAddrByNameMaster)(c)
}

type SentinelGetMasterAddrByNameMaster Completed

func (c SentinelGetMasterAddrByNameMaster) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type SentinelSentinels Completed

func (b Builder) SentinelSentinels() (c SentinelSentinels) {
	c = SentinelSentinels{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "SENTINEL", "SENTINELS")
	return c
}

func (c SentinelSentinels) Master(master string) SentinelSentinelsMaster {
	c.cs.s = append(c.cs.s, master)
	return (SentinelSentinelsMaster)(c)
}

type SentinelSentinelsMaster Completed

func (c SentinelSentinelsMaster) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type Set Completed

func (b Builder) Set() (c Set) {
	c = Set{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "SET")
	return c
}

func (c Set) Key(key string) SetKey {
	if c.ks&NoSlot == NoSlot {
		c.ks = NoSlot | slot(key)
	} else {
		c.ks = check(c.ks, slot(key))
	}
	c.cs.s = append(c.cs.s, key)
	return (SetKey)(c)
}

type SetConditionNx Completed

func (c SetConditionNx) Get() SetGet {
	c.cs.s = append(c.cs.s, "GET")
	return (SetGet)(c)
}

func (c SetConditionNx) ExSeconds(seconds int64) SetExpirationExSeconds {
	c.cs.s = append(c.cs.s, "EX", strconv.FormatInt(seconds, 10))
	return (SetExpirationExSeconds)(c)
}

func (c SetConditionNx) PxMilliseconds(milliseconds int64) SetExpirationPxMilliseconds {
	c.cs.s = append(c.cs.s, "PX", strconv.FormatInt(milliseconds, 10))
	return (SetExpirationPxMilliseconds)(c)
}

func (c SetConditionNx) ExatTimestamp(timestamp int64) SetExpirationExatTimestamp {
	c.cs.s = append(c.cs.s, "EXAT", strconv.FormatInt(timestamp, 10))
	return (SetExpirationExatTimestamp)(c)
}

func (c SetConditionNx) PxatMillisecondsTimestamp(millisecondsTimestamp int64) SetExpirationPxatMillisecondsTimestamp {
	c.cs.s = append(c.cs.s, "PXAT", strconv.FormatInt(millisecondsTimestamp, 10))
	return (SetExpirationPxatMillisecondsTimestamp)(c)
}

func (c SetConditionNx) Keepttl() SetExpirationKeepttl {
	c.cs.s = append(c.cs.s, "KEEPTTL")
	return (SetExpirationKeepttl)(c)
}

func (c SetConditionNx) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type SetConditionXx Completed

func (c SetConditionXx) Get() SetGet {
	c.cs.s = append(c.cs.s, "GET")
	return (SetGet)(c)
}

func (c SetConditionXx) ExSeconds(seconds int64) SetExpirationExSeconds {
	c.cs.s = append(c.cs.s, "EX", strconv.FormatInt(seconds, 10))
	return (SetExpirationExSeconds)(c)
}

func (c SetConditionXx) PxMilliseconds(milliseconds int64) SetExpirationPxMilliseconds {
	c.cs.s = append(c.cs.s, "PX", strconv.FormatInt(milliseconds, 10))
	return (SetExpirationPxMilliseconds)(c)
}

func (c SetConditionXx) ExatTimestamp(timestamp int64) SetExpirationExatTimestamp {
	c.cs.s = append(c.cs.s, "EXAT", strconv.FormatInt(timestamp, 10))
	return (SetExpirationExatTimestamp)(c)
}

func (c SetConditionXx) PxatMillisecondsTimestamp(millisecondsTimestamp int64) SetExpirationPxatMillisecondsTimestamp {
	c.cs.s = append(c.cs.s, "PXAT", strconv.FormatInt(millisecondsTimestamp, 10))
	return (SetExpirationPxatMillisecondsTimestamp)(c)
}

func (c SetConditionXx) Keepttl() SetExpirationKeepttl {
	c.cs.s = append(c.cs.s, "KEEPTTL")
	return (SetExpirationKeepttl)(c)
}

func (c SetConditionXx) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type SetExpirationExSeconds Completed

func (c SetExpirationExSeconds) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type SetExpirationExatTimestamp Completed

func (c SetExpirationExatTimestamp) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type SetExpirationKeepttl Completed

func (c SetExpirationKeepttl) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type SetExpirationPxMilliseconds Completed

func (c SetExpirationPxMilliseconds) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type SetExpirationPxatMillisecondsTimestamp Completed

func (c SetExpirationPxatMillisecondsTimestamp) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type SetGet Completed

func (c SetGet) ExSeconds(seconds int64) SetExpirationExSeconds {
	c.cs.s = append(c.cs.s, "EX", strconv.FormatInt(seconds, 10))
	return (SetExpirationExSeconds)(c)
}

func (c SetGet) PxMilliseconds(milliseconds int64) SetExpirationPxMilliseconds {
	c.cs.s = append(c.cs.s, "PX", strconv.FormatInt(milliseconds, 10))
	return (SetExpirationPxMilliseconds)(c)
}

func (c SetGet) ExatTimestamp(timestamp int64) SetExpirationExatTimestamp {
	c.cs.s = append(c.cs.s, "EXAT", strconv.FormatInt(timestamp, 10))
	return (SetExpirationExatTimestamp)(c)
}

func (c SetGet) PxatMillisecondsTimestamp(millisecondsTimestamp int64) SetExpirationPxatMillisecondsTimestamp {
	c.cs.s = append(c.cs.s, "PXAT", strconv.FormatInt(millisecondsTimestamp, 10))
	return (SetExpirationPxatMillisecondsTimestamp)(c)
}

func (c SetGet) Keepttl() SetExpirationKeepttl {
	c.cs.s = append(c.cs.s, "KEEPTTL")
	return (SetExpirationKeepttl)(c)
}

func (c SetGet) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type SetKey Completed

func (c SetKey) Value(value string) SetValue {
	c.cs.s = append(c.cs.s, value)
	return (SetValue)(c)
}

type SetValue Completed

func (c SetValue) Nx() SetConditionNx {
	c.cs.s = append(c.cs.s, "NX")
	return (SetConditionNx)(c)
}

func (c SetValue) Xx() SetConditionXx {
	c.cs.s = append(c.cs.s, "XX")
	return (SetConditionXx)(c)
}

func (c SetValue) Get() SetGet {
	c.cs.s = append(c.cs.s, "GET")
	return (SetGet)(c)
}

func (c SetValue) ExSeconds(seconds int64) SetExpirationExSeconds {
	c.cs.s = append(c.cs.s, "EX", strconv.FormatInt(seconds, 10))
	return (SetExpirationExSeconds)(c)
}

func (c SetValue) PxMilliseconds(milliseconds int64) SetExpirationPxMilliseconds {
	c.cs.s = append(c.cs.s, "PX", strconv.FormatInt(milliseconds, 10))
	return (SetExpirationPxMilliseconds)(c)
}

func (c SetValue) ExatTimestamp(timestamp int64) SetExpirationExatTimestamp {
	c.cs.s = append(c.cs.s, "EXAT", strconv.FormatInt(timestamp, 10))
	return (SetExpirationExatTimestamp)(c)
}

func (c SetValue) PxatMillisecondsTimestamp(millisecondsTimestamp int64) SetExpirationPxatMillisecondsTimestamp {
	c.cs.s = append(c.cs.s, "PXAT", strconv.FormatInt(millisecondsTimestamp, 10))
	return (SetExpirationPxatMillisecondsTimestamp)(c)
}

func (c SetValue) Keepttl() SetExpirationKeepttl {
	c.cs.s = append(c.cs.s, "KEEPTTL")
	return (SetExpirationKeepttl)(c)
}

func (c SetValue) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type Setbit Completed

func (b Builder) Setbit() (c Setbit) {
	c = Setbit{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "SETBIT")
	return c
}

func (c Setbit) Key(key string) SetbitKey {
	if c.ks&NoSlot == NoSlot {
		c.ks = NoSlot | slot(key)
	} else {
		c.ks = check(c.ks, slot(key))
	}
	c.cs.s = append(c.cs.s, key)
	return (SetbitKey)(c)
}

type SetbitKey Completed

func (c SetbitKey) Offset(offset int64) SetbitOffset {
	c.cs.s = append(c.cs.s, strconv.FormatInt(offset, 10))
	return (SetbitOffset)(c)
}

type SetbitOffset Completed

func (c SetbitOffset) Value(value int64) SetbitValue {
	c.cs.s = append(c.cs.s, strconv.FormatInt(value, 10))
	return (SetbitValue)(c)
}

type SetbitValue Completed

func (c SetbitValue) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type Setex Completed

func (b Builder) Setex() (c Setex) {
	c = Setex{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "SETEX")
	return c
}

func (c Setex) Key(key string) SetexKey {
	if c.ks&NoSlot == NoSlot {
		c.ks = NoSlot | slot(key)
	} else {
		c.ks = check(c.ks, slot(key))
	}
	c.cs.s = append(c.cs.s, key)
	return (SetexKey)(c)
}

type SetexKey Completed

func (c SetexKey) Seconds(seconds int64) SetexSeconds {
	c.cs.s = append(c.cs.s, strconv.FormatInt(seconds, 10))
	return (SetexSeconds)(c)
}

type SetexSeconds Completed

func (c SetexSeconds) Value(value string) SetexValue {
	c.cs.s = append(c.cs.s, value)
	return (SetexValue)(c)
}

type SetexValue Completed

func (c SetexValue) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type Setnx Completed

func (b Builder) Setnx() (c Setnx) {
	c = Setnx{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "SETNX")
	return c
}

func (c Setnx) Key(key string) SetnxKey {
	if c.ks&NoSlot == NoSlot {
		c.ks = NoSlot | slot(key)
	} else {
		c.ks = check(c.ks, slot(key))
	}
	c.cs.s = append(c.cs.s, key)
	return (SetnxKey)(c)
}

type SetnxKey Completed

func (c SetnxKey) Value(value string) SetnxValue {
	c.cs.s = append(c.cs.s, value)
	return (SetnxValue)(c)
}

type SetnxValue Completed

func (c SetnxValue) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type Setrange Completed

func (b Builder) Setrange() (c Setrange) {
	c = Setrange{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "SETRANGE")
	return c
}

func (c Setrange) Key(key string) SetrangeKey {
	if c.ks&NoSlot == NoSlot {
		c.ks = NoSlot | slot(key)
	} else {
		c.ks = check(c.ks, slot(key))
	}
	c.cs.s = append(c.cs.s, key)
	return (SetrangeKey)(c)
}

type SetrangeKey Completed

func (c SetrangeKey) Offset(offset int64) SetrangeOffset {
	c.cs.s = append(c.cs.s, strconv.FormatInt(offset, 10))
	return (SetrangeOffset)(c)
}

type SetrangeOffset Completed

func (c SetrangeOffset) Value(value string) SetrangeValue {
	c.cs.s = append(c.cs.s, value)
	return (SetrangeValue)(c)
}

type SetrangeValue Completed

func (c SetrangeValue) Build() Completed {
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

type Sinter Completed

func (b Builder) Sinter() (c Sinter) {
	c = Sinter{cs: get(), ks: b.ks, cf: readonly}
	c.cs.s = append(c.cs.s, "SINTER")
	return c
}

func (c Sinter) Key(key ...string) SinterKey {
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
	return (SinterKey)(c)
}

type SinterKey Completed

func (c SinterKey) Key(key ...string) SinterKey {
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

func (c SinterKey) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type Sintercard Completed

func (b Builder) Sintercard() (c Sintercard) {
	c = Sintercard{cs: get(), ks: b.ks, cf: readonly}
	c.cs.s = append(c.cs.s, "SINTERCARD")
	return c
}

func (c Sintercard) Numkeys(numkeys int64) SintercardNumkeys {
	c.cs.s = append(c.cs.s, strconv.FormatInt(numkeys, 10))
	return (SintercardNumkeys)(c)
}

type SintercardKey Completed

func (c SintercardKey) Key(key ...string) SintercardKey {
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

func (c SintercardKey) Limit(limit int64) SintercardLimit {
	c.cs.s = append(c.cs.s, "LIMIT", strconv.FormatInt(limit, 10))
	return (SintercardLimit)(c)
}

func (c SintercardKey) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type SintercardLimit Completed

func (c SintercardLimit) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type SintercardNumkeys Completed

func (c SintercardNumkeys) Key(key ...string) SintercardKey {
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
	return (SintercardKey)(c)
}

type Sinterstore Completed

func (b Builder) Sinterstore() (c Sinterstore) {
	c = Sinterstore{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "SINTERSTORE")
	return c
}

func (c Sinterstore) Destination(destination string) SinterstoreDestination {
	if c.ks&NoSlot == NoSlot {
		c.ks = NoSlot | slot(destination)
	} else {
		c.ks = check(c.ks, slot(destination))
	}
	c.cs.s = append(c.cs.s, destination)
	return (SinterstoreDestination)(c)
}

type SinterstoreDestination Completed

func (c SinterstoreDestination) Key(key ...string) SinterstoreKey {
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
	return (SinterstoreKey)(c)
}

type SinterstoreKey Completed

func (c SinterstoreKey) Key(key ...string) SinterstoreKey {
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

func (c SinterstoreKey) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type Sismember Completed

func (b Builder) Sismember() (c Sismember) {
	c = Sismember{cs: get(), ks: b.ks, cf: readonly}
	c.cs.s = append(c.cs.s, "SISMEMBER")
	return c
}

func (c Sismember) Key(key string) SismemberKey {
	if c.ks&NoSlot == NoSlot {
		c.ks = NoSlot | slot(key)
	} else {
		c.ks = check(c.ks, slot(key))
	}
	c.cs.s = append(c.cs.s, key)
	return (SismemberKey)(c)
}

type SismemberKey Completed

func (c SismemberKey) Member(member string) SismemberMember {
	c.cs.s = append(c.cs.s, member)
	return (SismemberMember)(c)
}

type SismemberMember Completed

func (c SismemberMember) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

func (c SismemberMember) Cache() Cacheable {
	c.cs.Build()
	return Cacheable(c)
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

type Smembers Completed

func (b Builder) Smembers() (c Smembers) {
	c = Smembers{cs: get(), ks: b.ks, cf: readonly}
	c.cs.s = append(c.cs.s, "SMEMBERS")
	return c
}

func (c Smembers) Key(key string) SmembersKey {
	if c.ks&NoSlot == NoSlot {
		c.ks = NoSlot | slot(key)
	} else {
		c.ks = check(c.ks, slot(key))
	}
	c.cs.s = append(c.cs.s, key)
	return (SmembersKey)(c)
}

type SmembersKey Completed

func (c SmembersKey) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

func (c SmembersKey) Cache() Cacheable {
	c.cs.Build()
	return Cacheable(c)
}

type Smismember Completed

func (b Builder) Smismember() (c Smismember) {
	c = Smismember{cs: get(), ks: b.ks, cf: readonly}
	c.cs.s = append(c.cs.s, "SMISMEMBER")
	return c
}

func (c Smismember) Key(key string) SmismemberKey {
	if c.ks&NoSlot == NoSlot {
		c.ks = NoSlot | slot(key)
	} else {
		c.ks = check(c.ks, slot(key))
	}
	c.cs.s = append(c.cs.s, key)
	return (SmismemberKey)(c)
}

type SmismemberKey Completed

func (c SmismemberKey) Member(member ...string) SmismemberMember {
	c.cs.s = append(c.cs.s, member...)
	return (SmismemberMember)(c)
}

type SmismemberMember Completed

func (c SmismemberMember) Member(member ...string) SmismemberMember {
	c.cs.s = append(c.cs.s, member...)
	return c
}

func (c SmismemberMember) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

func (c SmismemberMember) Cache() Cacheable {
	c.cs.Build()
	return Cacheable(c)
}

type Smove Completed

func (b Builder) Smove() (c Smove) {
	c = Smove{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "SMOVE")
	return c
}

func (c Smove) Source(source string) SmoveSource {
	if c.ks&NoSlot == NoSlot {
		c.ks = NoSlot | slot(source)
	} else {
		c.ks = check(c.ks, slot(source))
	}
	c.cs.s = append(c.cs.s, source)
	return (SmoveSource)(c)
}

type SmoveDestination Completed

func (c SmoveDestination) Member(member string) SmoveMember {
	c.cs.s = append(c.cs.s, member)
	return (SmoveMember)(c)
}

type SmoveMember Completed

func (c SmoveMember) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type SmoveSource Completed

func (c SmoveSource) Destination(destination string) SmoveDestination {
	if c.ks&NoSlot == NoSlot {
		c.ks = NoSlot | slot(destination)
	} else {
		c.ks = check(c.ks, slot(destination))
	}
	c.cs.s = append(c.cs.s, destination)
	return (SmoveDestination)(c)
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

type Spop Completed

func (b Builder) Spop() (c Spop) {
	c = Spop{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "SPOP")
	return c
}

func (c Spop) Key(key string) SpopKey {
	if c.ks&NoSlot == NoSlot {
		c.ks = NoSlot | slot(key)
	} else {
		c.ks = check(c.ks, slot(key))
	}
	c.cs.s = append(c.cs.s, key)
	return (SpopKey)(c)
}

type SpopCount Completed

func (c SpopCount) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type SpopKey Completed

func (c SpopKey) Count(count int64) SpopCount {
	c.cs.s = append(c.cs.s, strconv.FormatInt(count, 10))
	return (SpopCount)(c)
}

func (c SpopKey) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type Spublish Completed

func (b Builder) Spublish() (c Spublish) {
	c = Spublish{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "SPUBLISH")
	return c
}

func (c Spublish) Channel(channel string) SpublishChannel {
	if c.ks&NoSlot == NoSlot {
		c.ks = NoSlot | slot(channel)
	} else {
		c.ks = check(c.ks, slot(channel))
	}
	c.cs.s = append(c.cs.s, channel)
	return (SpublishChannel)(c)
}

type SpublishChannel Completed

func (c SpublishChannel) Message(message string) SpublishMessage {
	c.cs.s = append(c.cs.s, message)
	return (SpublishMessage)(c)
}

type SpublishMessage Completed

func (c SpublishMessage) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type Srandmember Completed

func (b Builder) Srandmember() (c Srandmember) {
	c = Srandmember{cs: get(), ks: b.ks, cf: readonly}
	c.cs.s = append(c.cs.s, "SRANDMEMBER")
	return c
}

func (c Srandmember) Key(key string) SrandmemberKey {
	if c.ks&NoSlot == NoSlot {
		c.ks = NoSlot | slot(key)
	} else {
		c.ks = check(c.ks, slot(key))
	}
	c.cs.s = append(c.cs.s, key)
	return (SrandmemberKey)(c)
}

type SrandmemberCount Completed

func (c SrandmemberCount) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type SrandmemberKey Completed

func (c SrandmemberKey) Count(count int64) SrandmemberCount {
	c.cs.s = append(c.cs.s, strconv.FormatInt(count, 10))
	return (SrandmemberCount)(c)
}

func (c SrandmemberKey) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type Srem Completed

func (b Builder) Srem() (c Srem) {
	c = Srem{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "SREM")
	return c
}

func (c Srem) Key(key string) SremKey {
	if c.ks&NoSlot == NoSlot {
		c.ks = NoSlot | slot(key)
	} else {
		c.ks = check(c.ks, slot(key))
	}
	c.cs.s = append(c.cs.s, key)
	return (SremKey)(c)
}

type SremKey Completed

func (c SremKey) Member(member ...string) SremMember {
	c.cs.s = append(c.cs.s, member...)
	return (SremMember)(c)
}

type SremMember Completed

func (c SremMember) Member(member ...string) SremMember {
	c.cs.s = append(c.cs.s, member...)
	return c
}

func (c SremMember) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type Sscan Completed

func (b Builder) Sscan() (c Sscan) {
	c = Sscan{cs: get(), ks: b.ks, cf: readonly}
	c.cs.s = append(c.cs.s, "SSCAN")
	return c
}

func (c Sscan) Key(key string) SscanKey {
	if c.ks&NoSlot == NoSlot {
		c.ks = NoSlot | slot(key)
	} else {
		c.ks = check(c.ks, slot(key))
	}
	c.cs.s = append(c.cs.s, key)
	return (SscanKey)(c)
}

type SscanCount Completed

func (c SscanCount) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type SscanCursor Completed

func (c SscanCursor) Match(pattern string) SscanMatch {
	c.cs.s = append(c.cs.s, "MATCH", pattern)
	return (SscanMatch)(c)
}

func (c SscanCursor) Count(count int64) SscanCount {
	c.cs.s = append(c.cs.s, "COUNT", strconv.FormatInt(count, 10))
	return (SscanCount)(c)
}

func (c SscanCursor) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type SscanKey Completed

func (c SscanKey) Cursor(cursor uint64) SscanCursor {
	c.cs.s = append(c.cs.s, strconv.FormatUint(cursor, 10))
	return (SscanCursor)(c)
}

type SscanMatch Completed

func (c SscanMatch) Count(count int64) SscanCount {
	c.cs.s = append(c.cs.s, "COUNT", strconv.FormatInt(count, 10))
	return (SscanCount)(c)
}

func (c SscanMatch) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type Ssubscribe Completed

func (b Builder) Ssubscribe() (c Ssubscribe) {
	c = Ssubscribe{cs: get(), ks: b.ks, cf: noRetTag}
	c.cs.s = append(c.cs.s, "SSUBSCRIBE")
	return c
}

func (c Ssubscribe) Channel(channel ...string) SsubscribeChannel {
	if c.ks&NoSlot == NoSlot {
		for _, k := range channel {
			c.ks = NoSlot | slot(k)
			break
		}
	} else {
		for _, k := range channel {
			c.ks = check(c.ks, slot(k))
		}
	}
	c.cs.s = append(c.cs.s, channel...)
	return (SsubscribeChannel)(c)
}

type SsubscribeChannel Completed

func (c SsubscribeChannel) Channel(channel ...string) SsubscribeChannel {
	if c.ks&NoSlot == NoSlot {
		for _, k := range channel {
			c.ks = NoSlot | slot(k)
			break
		}
	} else {
		for _, k := range channel {
			c.ks = check(c.ks, slot(k))
		}
	}
	c.cs.s = append(c.cs.s, channel...)
	return c
}

func (c SsubscribeChannel) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type Strlen Completed

func (b Builder) Strlen() (c Strlen) {
	c = Strlen{cs: get(), ks: b.ks, cf: readonly}
	c.cs.s = append(c.cs.s, "STRLEN")
	return c
}

func (c Strlen) Key(key string) StrlenKey {
	if c.ks&NoSlot == NoSlot {
		c.ks = NoSlot | slot(key)
	} else {
		c.ks = check(c.ks, slot(key))
	}
	c.cs.s = append(c.cs.s, key)
	return (StrlenKey)(c)
}

type StrlenKey Completed

func (c StrlenKey) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

func (c StrlenKey) Cache() Cacheable {
	c.cs.Build()
	return Cacheable(c)
}

type Subscribe Completed

func (b Builder) Subscribe() (c Subscribe) {
	c = Subscribe{cs: get(), ks: b.ks, cf: noRetTag}
	c.cs.s = append(c.cs.s, "SUBSCRIBE")
	return c
}

func (c Subscribe) Channel(channel ...string) SubscribeChannel {
	c.cs.s = append(c.cs.s, channel...)
	return (SubscribeChannel)(c)
}

type SubscribeChannel Completed

func (c SubscribeChannel) Channel(channel ...string) SubscribeChannel {
	c.cs.s = append(c.cs.s, channel...)
	return c
}

func (c SubscribeChannel) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type Sunion Completed

func (b Builder) Sunion() (c Sunion) {
	c = Sunion{cs: get(), ks: b.ks, cf: readonly}
	c.cs.s = append(c.cs.s, "SUNION")
	return c
}

func (c Sunion) Key(key ...string) SunionKey {
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
	return (SunionKey)(c)
}

type SunionKey Completed

func (c SunionKey) Key(key ...string) SunionKey {
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

func (c SunionKey) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type Sunionstore Completed

func (b Builder) Sunionstore() (c Sunionstore) {
	c = Sunionstore{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "SUNIONSTORE")
	return c
}

func (c Sunionstore) Destination(destination string) SunionstoreDestination {
	if c.ks&NoSlot == NoSlot {
		c.ks = NoSlot | slot(destination)
	} else {
		c.ks = check(c.ks, slot(destination))
	}
	c.cs.s = append(c.cs.s, destination)
	return (SunionstoreDestination)(c)
}

type SunionstoreDestination Completed

func (c SunionstoreDestination) Key(key ...string) SunionstoreKey {
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
	return (SunionstoreKey)(c)
}

type SunionstoreKey Completed

func (c SunionstoreKey) Key(key ...string) SunionstoreKey {
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

func (c SunionstoreKey) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type Sunsubscribe Completed

func (b Builder) Sunsubscribe() (c Sunsubscribe) {
	c = Sunsubscribe{cs: get(), ks: b.ks, cf: noRetTag}
	c.cs.s = append(c.cs.s, "SUNSUBSCRIBE")
	return c
}

func (c Sunsubscribe) Channel(channel ...string) SunsubscribeChannel {
	if c.ks&NoSlot == NoSlot {
		for _, k := range channel {
			c.ks = NoSlot | slot(k)
			break
		}
	} else {
		for _, k := range channel {
			c.ks = check(c.ks, slot(k))
		}
	}
	c.cs.s = append(c.cs.s, channel...)
	return (SunsubscribeChannel)(c)
}

func (c Sunsubscribe) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type SunsubscribeChannel Completed

func (c SunsubscribeChannel) Channel(channel ...string) SunsubscribeChannel {
	if c.ks&NoSlot == NoSlot {
		for _, k := range channel {
			c.ks = NoSlot | slot(k)
			break
		}
	} else {
		for _, k := range channel {
			c.ks = check(c.ks, slot(k))
		}
	}
	c.cs.s = append(c.cs.s, channel...)
	return c
}

func (c SunsubscribeChannel) Build() Completed {
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

type TdigestAdd Completed

func (b Builder) TdigestAdd() (c TdigestAdd) {
	c = TdigestAdd{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "TDIGEST.ADD")
	return c
}

func (c TdigestAdd) Key(key string) TdigestAddKey {
	if c.ks&NoSlot == NoSlot {
		c.ks = NoSlot | slot(key)
	} else {
		c.ks = check(c.ks, slot(key))
	}
	c.cs.s = append(c.cs.s, key)
	return (TdigestAddKey)(c)
}

type TdigestAddKey Completed

func (c TdigestAddKey) Value(value float64) TdigestAddValuesValue {
	c.cs.s = append(c.cs.s, strconv.FormatFloat(value, 'f', -1, 64))
	return (TdigestAddValuesValue)(c)
}

type TdigestAddValuesValue Completed

func (c TdigestAddValuesValue) Value(value float64) TdigestAddValuesValue {
	c.cs.s = append(c.cs.s, strconv.FormatFloat(value, 'f', -1, 64))
	return c
}

func (c TdigestAddValuesValue) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type TdigestByrank Completed

func (b Builder) TdigestByrank() (c TdigestByrank) {
	c = TdigestByrank{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "TDIGEST.BYRANK")
	return c
}

func (c TdigestByrank) Key(key string) TdigestByrankKey {
	if c.ks&NoSlot == NoSlot {
		c.ks = NoSlot | slot(key)
	} else {
		c.ks = check(c.ks, slot(key))
	}
	c.cs.s = append(c.cs.s, key)
	return (TdigestByrankKey)(c)
}

type TdigestByrankKey Completed

func (c TdigestByrankKey) Rank(rank ...float64) TdigestByrankRank {
	for _, n := range rank {
		c.cs.s = append(c.cs.s, strconv.FormatFloat(n, 'f', -1, 64))
	}
	return (TdigestByrankRank)(c)
}

type TdigestByrankRank Completed

func (c TdigestByrankRank) Rank(rank ...float64) TdigestByrankRank {
	for _, n := range rank {
		c.cs.s = append(c.cs.s, strconv.FormatFloat(n, 'f', -1, 64))
	}
	return c
}

func (c TdigestByrankRank) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type TdigestByrevrank Completed

func (b Builder) TdigestByrevrank() (c TdigestByrevrank) {
	c = TdigestByrevrank{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "TDIGEST.BYREVRANK")
	return c
}

func (c TdigestByrevrank) Key(key string) TdigestByrevrankKey {
	if c.ks&NoSlot == NoSlot {
		c.ks = NoSlot | slot(key)
	} else {
		c.ks = check(c.ks, slot(key))
	}
	c.cs.s = append(c.cs.s, key)
	return (TdigestByrevrankKey)(c)
}

type TdigestByrevrankKey Completed

func (c TdigestByrevrankKey) ReverseRank(reverseRank ...float64) TdigestByrevrankReverseRank {
	for _, n := range reverseRank {
		c.cs.s = append(c.cs.s, strconv.FormatFloat(n, 'f', -1, 64))
	}
	return (TdigestByrevrankReverseRank)(c)
}

type TdigestByrevrankReverseRank Completed

func (c TdigestByrevrankReverseRank) ReverseRank(reverseRank ...float64) TdigestByrevrankReverseRank {
	for _, n := range reverseRank {
		c.cs.s = append(c.cs.s, strconv.FormatFloat(n, 'f', -1, 64))
	}
	return c
}

func (c TdigestByrevrankReverseRank) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type TdigestCdf Completed

func (b Builder) TdigestCdf() (c TdigestCdf) {
	c = TdigestCdf{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "TDIGEST.CDF")
	return c
}

func (c TdigestCdf) Key(key string) TdigestCdfKey {
	if c.ks&NoSlot == NoSlot {
		c.ks = NoSlot | slot(key)
	} else {
		c.ks = check(c.ks, slot(key))
	}
	c.cs.s = append(c.cs.s, key)
	return (TdigestCdfKey)(c)
}

type TdigestCdfKey Completed

func (c TdigestCdfKey) Value(value ...float64) TdigestCdfValue {
	for _, n := range value {
		c.cs.s = append(c.cs.s, strconv.FormatFloat(n, 'f', -1, 64))
	}
	return (TdigestCdfValue)(c)
}

type TdigestCdfValue Completed

func (c TdigestCdfValue) Value(value ...float64) TdigestCdfValue {
	for _, n := range value {
		c.cs.s = append(c.cs.s, strconv.FormatFloat(n, 'f', -1, 64))
	}
	return c
}

func (c TdigestCdfValue) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type TdigestCreate Completed

func (b Builder) TdigestCreate() (c TdigestCreate) {
	c = TdigestCreate{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "TDIGEST.CREATE")
	return c
}

func (c TdigestCreate) Key(key string) TdigestCreateKey {
	if c.ks&NoSlot == NoSlot {
		c.ks = NoSlot | slot(key)
	} else {
		c.ks = check(c.ks, slot(key))
	}
	c.cs.s = append(c.cs.s, key)
	return (TdigestCreateKey)(c)
}

type TdigestCreateCompression Completed

func (c TdigestCreateCompression) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type TdigestCreateKey Completed

func (c TdigestCreateKey) Compression(compression int64) TdigestCreateCompression {
	c.cs.s = append(c.cs.s, "COMPRESSION", strconv.FormatInt(compression, 10))
	return (TdigestCreateCompression)(c)
}

func (c TdigestCreateKey) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type TdigestInfo Completed

func (b Builder) TdigestInfo() (c TdigestInfo) {
	c = TdigestInfo{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "TDIGEST.INFO")
	return c
}

func (c TdigestInfo) Key(key string) TdigestInfoKey {
	if c.ks&NoSlot == NoSlot {
		c.ks = NoSlot | slot(key)
	} else {
		c.ks = check(c.ks, slot(key))
	}
	c.cs.s = append(c.cs.s, key)
	return (TdigestInfoKey)(c)
}

type TdigestInfoKey Completed

func (c TdigestInfoKey) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type TdigestMax Completed

func (b Builder) TdigestMax() (c TdigestMax) {
	c = TdigestMax{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "TDIGEST.MAX")
	return c
}

func (c TdigestMax) Key(key string) TdigestMaxKey {
	if c.ks&NoSlot == NoSlot {
		c.ks = NoSlot | slot(key)
	} else {
		c.ks = check(c.ks, slot(key))
	}
	c.cs.s = append(c.cs.s, key)
	return (TdigestMaxKey)(c)
}

type TdigestMaxKey Completed

func (c TdigestMaxKey) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type TdigestMerge Completed

func (b Builder) TdigestMerge() (c TdigestMerge) {
	c = TdigestMerge{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "TDIGEST.MERGE")
	return c
}

func (c TdigestMerge) DestinationKey(destinationKey string) TdigestMergeDestinationKey {
	if c.ks&NoSlot == NoSlot {
		c.ks = NoSlot | slot(destinationKey)
	} else {
		c.ks = check(c.ks, slot(destinationKey))
	}
	c.cs.s = append(c.cs.s, destinationKey)
	return (TdigestMergeDestinationKey)(c)
}

type TdigestMergeConfigCompression Completed

func (c TdigestMergeConfigCompression) Override() TdigestMergeOverride {
	c.cs.s = append(c.cs.s, "OVERRIDE")
	return (TdigestMergeOverride)(c)
}

func (c TdigestMergeConfigCompression) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type TdigestMergeDestinationKey Completed

func (c TdigestMergeDestinationKey) Numkeys(numkeys int64) TdigestMergeNumkeys {
	c.cs.s = append(c.cs.s, strconv.FormatInt(numkeys, 10))
	return (TdigestMergeNumkeys)(c)
}

type TdigestMergeNumkeys Completed

func (c TdigestMergeNumkeys) SourceKey(sourceKey ...string) TdigestMergeSourceKey {
	if c.ks&NoSlot == NoSlot {
		for _, k := range sourceKey {
			c.ks = NoSlot | slot(k)
			break
		}
	} else {
		for _, k := range sourceKey {
			c.ks = check(c.ks, slot(k))
		}
	}
	c.cs.s = append(c.cs.s, sourceKey...)
	return (TdigestMergeSourceKey)(c)
}

type TdigestMergeOverride Completed

func (c TdigestMergeOverride) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type TdigestMergeSourceKey Completed

func (c TdigestMergeSourceKey) SourceKey(sourceKey ...string) TdigestMergeSourceKey {
	if c.ks&NoSlot == NoSlot {
		for _, k := range sourceKey {
			c.ks = NoSlot | slot(k)
			break
		}
	} else {
		for _, k := range sourceKey {
			c.ks = check(c.ks, slot(k))
		}
	}
	c.cs.s = append(c.cs.s, sourceKey...)
	return c
}

func (c TdigestMergeSourceKey) Compression(compression int64) TdigestMergeConfigCompression {
	c.cs.s = append(c.cs.s, "COMPRESSION", strconv.FormatInt(compression, 10))
	return (TdigestMergeConfigCompression)(c)
}

func (c TdigestMergeSourceKey) Override() TdigestMergeOverride {
	c.cs.s = append(c.cs.s, "OVERRIDE")
	return (TdigestMergeOverride)(c)
}

func (c TdigestMergeSourceKey) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type TdigestMin Completed

func (b Builder) TdigestMin() (c TdigestMin) {
	c = TdigestMin{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "TDIGEST.MIN")
	return c
}

func (c TdigestMin) Key(key string) TdigestMinKey {
	if c.ks&NoSlot == NoSlot {
		c.ks = NoSlot | slot(key)
	} else {
		c.ks = check(c.ks, slot(key))
	}
	c.cs.s = append(c.cs.s, key)
	return (TdigestMinKey)(c)
}

type TdigestMinKey Completed

func (c TdigestMinKey) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type TdigestQuantile Completed

func (b Builder) TdigestQuantile() (c TdigestQuantile) {
	c = TdigestQuantile{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "TDIGEST.QUANTILE")
	return c
}

func (c TdigestQuantile) Key(key string) TdigestQuantileKey {
	if c.ks&NoSlot == NoSlot {
		c.ks = NoSlot | slot(key)
	} else {
		c.ks = check(c.ks, slot(key))
	}
	c.cs.s = append(c.cs.s, key)
	return (TdigestQuantileKey)(c)
}

type TdigestQuantileKey Completed

func (c TdigestQuantileKey) Quantile(quantile ...float64) TdigestQuantileQuantile {
	for _, n := range quantile {
		c.cs.s = append(c.cs.s, strconv.FormatFloat(n, 'f', -1, 64))
	}
	return (TdigestQuantileQuantile)(c)
}

type TdigestQuantileQuantile Completed

func (c TdigestQuantileQuantile) Quantile(quantile ...float64) TdigestQuantileQuantile {
	for _, n := range quantile {
		c.cs.s = append(c.cs.s, strconv.FormatFloat(n, 'f', -1, 64))
	}
	return c
}

func (c TdigestQuantileQuantile) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type TdigestRank Completed

func (b Builder) TdigestRank() (c TdigestRank) {
	c = TdigestRank{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "TDIGEST.RANK")
	return c
}

func (c TdigestRank) Key(key string) TdigestRankKey {
	if c.ks&NoSlot == NoSlot {
		c.ks = NoSlot | slot(key)
	} else {
		c.ks = check(c.ks, slot(key))
	}
	c.cs.s = append(c.cs.s, key)
	return (TdigestRankKey)(c)
}

type TdigestRankKey Completed

func (c TdigestRankKey) Value(value ...float64) TdigestRankValue {
	for _, n := range value {
		c.cs.s = append(c.cs.s, strconv.FormatFloat(n, 'f', -1, 64))
	}
	return (TdigestRankValue)(c)
}

type TdigestRankValue Completed

func (c TdigestRankValue) Value(value ...float64) TdigestRankValue {
	for _, n := range value {
		c.cs.s = append(c.cs.s, strconv.FormatFloat(n, 'f', -1, 64))
	}
	return c
}

func (c TdigestRankValue) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type TdigestReset Completed

func (b Builder) TdigestReset() (c TdigestReset) {
	c = TdigestReset{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "TDIGEST.RESET")
	return c
}

func (c TdigestReset) Key(key string) TdigestResetKey {
	if c.ks&NoSlot == NoSlot {
		c.ks = NoSlot | slot(key)
	} else {
		c.ks = check(c.ks, slot(key))
	}
	c.cs.s = append(c.cs.s, key)
	return (TdigestResetKey)(c)
}

type TdigestResetKey Completed

func (c TdigestResetKey) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type TdigestRevrank Completed

func (b Builder) TdigestRevrank() (c TdigestRevrank) {
	c = TdigestRevrank{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "TDIGEST.REVRANK")
	return c
}

func (c TdigestRevrank) Key(key string) TdigestRevrankKey {
	if c.ks&NoSlot == NoSlot {
		c.ks = NoSlot | slot(key)
	} else {
		c.ks = check(c.ks, slot(key))
	}
	c.cs.s = append(c.cs.s, key)
	return (TdigestRevrankKey)(c)
}

type TdigestRevrankKey Completed

func (c TdigestRevrankKey) Value(value ...float64) TdigestRevrankValue {
	for _, n := range value {
		c.cs.s = append(c.cs.s, strconv.FormatFloat(n, 'f', -1, 64))
	}
	return (TdigestRevrankValue)(c)
}

type TdigestRevrankValue Completed

func (c TdigestRevrankValue) Value(value ...float64) TdigestRevrankValue {
	for _, n := range value {
		c.cs.s = append(c.cs.s, strconv.FormatFloat(n, 'f', -1, 64))
	}
	return c
}

func (c TdigestRevrankValue) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type TdigestTrimmedMean Completed

func (b Builder) TdigestTrimmedMean() (c TdigestTrimmedMean) {
	c = TdigestTrimmedMean{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "TDIGEST.TRIMMED_MEAN")
	return c
}

func (c TdigestTrimmedMean) Key(key string) TdigestTrimmedMeanKey {
	if c.ks&NoSlot == NoSlot {
		c.ks = NoSlot | slot(key)
	} else {
		c.ks = check(c.ks, slot(key))
	}
	c.cs.s = append(c.cs.s, key)
	return (TdigestTrimmedMeanKey)(c)
}

type TdigestTrimmedMeanHighCutQuantile Completed

func (c TdigestTrimmedMeanHighCutQuantile) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type TdigestTrimmedMeanKey Completed

func (c TdigestTrimmedMeanKey) LowCutQuantile(lowCutQuantile float64) TdigestTrimmedMeanLowCutQuantile {
	c.cs.s = append(c.cs.s, strconv.FormatFloat(lowCutQuantile, 'f', -1, 64))
	return (TdigestTrimmedMeanLowCutQuantile)(c)
}

type TdigestTrimmedMeanLowCutQuantile Completed

func (c TdigestTrimmedMeanLowCutQuantile) HighCutQuantile(highCutQuantile float64) TdigestTrimmedMeanHighCutQuantile {
	c.cs.s = append(c.cs.s, strconv.FormatFloat(highCutQuantile, 'f', -1, 64))
	return (TdigestTrimmedMeanHighCutQuantile)(c)
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

type TopkAdd Completed

func (b Builder) TopkAdd() (c TopkAdd) {
	c = TopkAdd{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "TOPK.ADD")
	return c
}

func (c TopkAdd) Key(key string) TopkAddKey {
	if c.ks&NoSlot == NoSlot {
		c.ks = NoSlot | slot(key)
	} else {
		c.ks = check(c.ks, slot(key))
	}
	c.cs.s = append(c.cs.s, key)
	return (TopkAddKey)(c)
}

type TopkAddItems Completed

func (c TopkAddItems) Items(items ...string) TopkAddItems {
	c.cs.s = append(c.cs.s, items...)
	return c
}

func (c TopkAddItems) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type TopkAddKey Completed

func (c TopkAddKey) Items(items ...string) TopkAddItems {
	c.cs.s = append(c.cs.s, items...)
	return (TopkAddItems)(c)
}

type TopkCount Completed

func (b Builder) TopkCount() (c TopkCount) {
	c = TopkCount{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "TOPK.COUNT")
	return c
}

func (c TopkCount) Key(key string) TopkCountKey {
	if c.ks&NoSlot == NoSlot {
		c.ks = NoSlot | slot(key)
	} else {
		c.ks = check(c.ks, slot(key))
	}
	c.cs.s = append(c.cs.s, key)
	return (TopkCountKey)(c)
}

type TopkCountItem Completed

func (c TopkCountItem) Item(item ...string) TopkCountItem {
	c.cs.s = append(c.cs.s, item...)
	return c
}

func (c TopkCountItem) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type TopkCountKey Completed

func (c TopkCountKey) Item(item ...string) TopkCountItem {
	c.cs.s = append(c.cs.s, item...)
	return (TopkCountItem)(c)
}

type TopkIncrby Completed

func (b Builder) TopkIncrby() (c TopkIncrby) {
	c = TopkIncrby{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "TOPK.INCRBY")
	return c
}

func (c TopkIncrby) Key(key string) TopkIncrbyKey {
	if c.ks&NoSlot == NoSlot {
		c.ks = NoSlot | slot(key)
	} else {
		c.ks = check(c.ks, slot(key))
	}
	c.cs.s = append(c.cs.s, key)
	return (TopkIncrbyKey)(c)
}

type TopkIncrbyItemsIncrement Completed

func (c TopkIncrbyItemsIncrement) Item(item string) TopkIncrbyItemsItem {
	c.cs.s = append(c.cs.s, item)
	return (TopkIncrbyItemsItem)(c)
}

func (c TopkIncrbyItemsIncrement) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type TopkIncrbyItemsItem Completed

func (c TopkIncrbyItemsItem) Increment(increment int64) TopkIncrbyItemsIncrement {
	c.cs.s = append(c.cs.s, strconv.FormatInt(increment, 10))
	return (TopkIncrbyItemsIncrement)(c)
}

type TopkIncrbyKey Completed

func (c TopkIncrbyKey) Item(item string) TopkIncrbyItemsItem {
	c.cs.s = append(c.cs.s, item)
	return (TopkIncrbyItemsItem)(c)
}

type TopkInfo Completed

func (b Builder) TopkInfo() (c TopkInfo) {
	c = TopkInfo{cs: get(), ks: b.ks, cf: readonly}
	c.cs.s = append(c.cs.s, "TOPK.INFO")
	return c
}

func (c TopkInfo) Key(key string) TopkInfoKey {
	if c.ks&NoSlot == NoSlot {
		c.ks = NoSlot | slot(key)
	} else {
		c.ks = check(c.ks, slot(key))
	}
	c.cs.s = append(c.cs.s, key)
	return (TopkInfoKey)(c)
}

type TopkInfoKey Completed

func (c TopkInfoKey) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

func (c TopkInfoKey) Cache() Cacheable {
	c.cs.Build()
	return Cacheable(c)
}

type TopkList Completed

func (b Builder) TopkList() (c TopkList) {
	c = TopkList{cs: get(), ks: b.ks, cf: readonly}
	c.cs.s = append(c.cs.s, "TOPK.LIST")
	return c
}

func (c TopkList) Key(key string) TopkListKey {
	if c.ks&NoSlot == NoSlot {
		c.ks = NoSlot | slot(key)
	} else {
		c.ks = check(c.ks, slot(key))
	}
	c.cs.s = append(c.cs.s, key)
	return (TopkListKey)(c)
}

type TopkListKey Completed

func (c TopkListKey) Withcount() TopkListWithcount {
	c.cs.s = append(c.cs.s, "WITHCOUNT")
	return (TopkListWithcount)(c)
}

func (c TopkListKey) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

func (c TopkListKey) Cache() Cacheable {
	c.cs.Build()
	return Cacheable(c)
}

type TopkListWithcount Completed

func (c TopkListWithcount) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

func (c TopkListWithcount) Cache() Cacheable {
	c.cs.Build()
	return Cacheable(c)
}

type TopkQuery Completed

func (b Builder) TopkQuery() (c TopkQuery) {
	c = TopkQuery{cs: get(), ks: b.ks, cf: readonly}
	c.cs.s = append(c.cs.s, "TOPK.QUERY")
	return c
}

func (c TopkQuery) Key(key string) TopkQueryKey {
	if c.ks&NoSlot == NoSlot {
		c.ks = NoSlot | slot(key)
	} else {
		c.ks = check(c.ks, slot(key))
	}
	c.cs.s = append(c.cs.s, key)
	return (TopkQueryKey)(c)
}

type TopkQueryItem Completed

func (c TopkQueryItem) Item(item ...string) TopkQueryItem {
	c.cs.s = append(c.cs.s, item...)
	return c
}

func (c TopkQueryItem) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

func (c TopkQueryItem) Cache() Cacheable {
	c.cs.Build()
	return Cacheable(c)
}

type TopkQueryKey Completed

func (c TopkQueryKey) Item(item ...string) TopkQueryItem {
	c.cs.s = append(c.cs.s, item...)
	return (TopkQueryItem)(c)
}

type TopkReserve Completed

func (b Builder) TopkReserve() (c TopkReserve) {
	c = TopkReserve{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "TOPK.RESERVE")
	return c
}

func (c TopkReserve) Key(key string) TopkReserveKey {
	if c.ks&NoSlot == NoSlot {
		c.ks = NoSlot | slot(key)
	} else {
		c.ks = check(c.ks, slot(key))
	}
	c.cs.s = append(c.cs.s, key)
	return (TopkReserveKey)(c)
}

type TopkReserveKey Completed

func (c TopkReserveKey) Topk(topk int64) TopkReserveTopk {
	c.cs.s = append(c.cs.s, strconv.FormatInt(topk, 10))
	return (TopkReserveTopk)(c)
}

type TopkReserveParamsDecay Completed

func (c TopkReserveParamsDecay) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type TopkReserveParamsDepth Completed

func (c TopkReserveParamsDepth) Decay(decay float64) TopkReserveParamsDecay {
	c.cs.s = append(c.cs.s, strconv.FormatFloat(decay, 'f', -1, 64))
	return (TopkReserveParamsDecay)(c)
}

type TopkReserveParamsWidth Completed

func (c TopkReserveParamsWidth) Depth(depth int64) TopkReserveParamsDepth {
	c.cs.s = append(c.cs.s, strconv.FormatInt(depth, 10))
	return (TopkReserveParamsDepth)(c)
}

type TopkReserveTopk Completed

func (c TopkReserveTopk) Width(width int64) TopkReserveParamsWidth {
	c.cs.s = append(c.cs.s, strconv.FormatInt(width, 10))
	return (TopkReserveParamsWidth)(c)
}

func (c TopkReserveTopk) Build() Completed {
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

type TsAdd Completed

func (b Builder) TsAdd() (c TsAdd) {
	c = TsAdd{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "TS.ADD")
	return c
}

func (c TsAdd) Key(key string) TsAddKey {
	if c.ks&NoSlot == NoSlot {
		c.ks = NoSlot | slot(key)
	} else {
		c.ks = check(c.ks, slot(key))
	}
	c.cs.s = append(c.cs.s, key)
	return (TsAddKey)(c)
}

type TsAddChunkSize Completed

func (c TsAddChunkSize) OnDuplicateBlock() TsAddOnDuplicateBlock {
	c.cf = blockTag
	c.cs.s = append(c.cs.s, "ON_DUPLICATE", "BLOCK")
	return (TsAddOnDuplicateBlock)(c)
}

func (c TsAddChunkSize) OnDuplicateFirst() TsAddOnDuplicateFirst {
	c.cs.s = append(c.cs.s, "ON_DUPLICATE", "FIRST")
	return (TsAddOnDuplicateFirst)(c)
}

func (c TsAddChunkSize) OnDuplicateLast() TsAddOnDuplicateLast {
	c.cs.s = append(c.cs.s, "ON_DUPLICATE", "LAST")
	return (TsAddOnDuplicateLast)(c)
}

func (c TsAddChunkSize) OnDuplicateMin() TsAddOnDuplicateMin {
	c.cs.s = append(c.cs.s, "ON_DUPLICATE", "MIN")
	return (TsAddOnDuplicateMin)(c)
}

func (c TsAddChunkSize) OnDuplicateMax() TsAddOnDuplicateMax {
	c.cs.s = append(c.cs.s, "ON_DUPLICATE", "MAX")
	return (TsAddOnDuplicateMax)(c)
}

func (c TsAddChunkSize) OnDuplicateSum() TsAddOnDuplicateSum {
	c.cs.s = append(c.cs.s, "ON_DUPLICATE", "SUM")
	return (TsAddOnDuplicateSum)(c)
}

func (c TsAddChunkSize) Labels() TsAddLabels {
	c.cs.s = append(c.cs.s, "LABELS")
	return (TsAddLabels)(c)
}

func (c TsAddChunkSize) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type TsAddEncodingCompressed Completed

func (c TsAddEncodingCompressed) ChunkSize(size int64) TsAddChunkSize {
	c.cs.s = append(c.cs.s, "CHUNK_SIZE", strconv.FormatInt(size, 10))
	return (TsAddChunkSize)(c)
}

func (c TsAddEncodingCompressed) OnDuplicateBlock() TsAddOnDuplicateBlock {
	c.cf = blockTag
	c.cs.s = append(c.cs.s, "ON_DUPLICATE", "BLOCK")
	return (TsAddOnDuplicateBlock)(c)
}

func (c TsAddEncodingCompressed) OnDuplicateFirst() TsAddOnDuplicateFirst {
	c.cs.s = append(c.cs.s, "ON_DUPLICATE", "FIRST")
	return (TsAddOnDuplicateFirst)(c)
}

func (c TsAddEncodingCompressed) OnDuplicateLast() TsAddOnDuplicateLast {
	c.cs.s = append(c.cs.s, "ON_DUPLICATE", "LAST")
	return (TsAddOnDuplicateLast)(c)
}

func (c TsAddEncodingCompressed) OnDuplicateMin() TsAddOnDuplicateMin {
	c.cs.s = append(c.cs.s, "ON_DUPLICATE", "MIN")
	return (TsAddOnDuplicateMin)(c)
}

func (c TsAddEncodingCompressed) OnDuplicateMax() TsAddOnDuplicateMax {
	c.cs.s = append(c.cs.s, "ON_DUPLICATE", "MAX")
	return (TsAddOnDuplicateMax)(c)
}

func (c TsAddEncodingCompressed) OnDuplicateSum() TsAddOnDuplicateSum {
	c.cs.s = append(c.cs.s, "ON_DUPLICATE", "SUM")
	return (TsAddOnDuplicateSum)(c)
}

func (c TsAddEncodingCompressed) Labels() TsAddLabels {
	c.cs.s = append(c.cs.s, "LABELS")
	return (TsAddLabels)(c)
}

func (c TsAddEncodingCompressed) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type TsAddEncodingUncompressed Completed

func (c TsAddEncodingUncompressed) ChunkSize(size int64) TsAddChunkSize {
	c.cs.s = append(c.cs.s, "CHUNK_SIZE", strconv.FormatInt(size, 10))
	return (TsAddChunkSize)(c)
}

func (c TsAddEncodingUncompressed) OnDuplicateBlock() TsAddOnDuplicateBlock {
	c.cf = blockTag
	c.cs.s = append(c.cs.s, "ON_DUPLICATE", "BLOCK")
	return (TsAddOnDuplicateBlock)(c)
}

func (c TsAddEncodingUncompressed) OnDuplicateFirst() TsAddOnDuplicateFirst {
	c.cs.s = append(c.cs.s, "ON_DUPLICATE", "FIRST")
	return (TsAddOnDuplicateFirst)(c)
}

func (c TsAddEncodingUncompressed) OnDuplicateLast() TsAddOnDuplicateLast {
	c.cs.s = append(c.cs.s, "ON_DUPLICATE", "LAST")
	return (TsAddOnDuplicateLast)(c)
}

func (c TsAddEncodingUncompressed) OnDuplicateMin() TsAddOnDuplicateMin {
	c.cs.s = append(c.cs.s, "ON_DUPLICATE", "MIN")
	return (TsAddOnDuplicateMin)(c)
}

func (c TsAddEncodingUncompressed) OnDuplicateMax() TsAddOnDuplicateMax {
	c.cs.s = append(c.cs.s, "ON_DUPLICATE", "MAX")
	return (TsAddOnDuplicateMax)(c)
}

func (c TsAddEncodingUncompressed) OnDuplicateSum() TsAddOnDuplicateSum {
	c.cs.s = append(c.cs.s, "ON_DUPLICATE", "SUM")
	return (TsAddOnDuplicateSum)(c)
}

func (c TsAddEncodingUncompressed) Labels() TsAddLabels {
	c.cs.s = append(c.cs.s, "LABELS")
	return (TsAddLabels)(c)
}

func (c TsAddEncodingUncompressed) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type TsAddKey Completed

func (c TsAddKey) Timestamp(timestamp int64) TsAddTimestamp {
	c.cs.s = append(c.cs.s, strconv.FormatInt(timestamp, 10))
	return (TsAddTimestamp)(c)
}

type TsAddLabels Completed

func (c TsAddLabels) Labels(label string, value string) TsAddLabels {
	c.cs.s = append(c.cs.s, label, value)
	return c
}

func (c TsAddLabels) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type TsAddOnDuplicateBlock Completed

func (c TsAddOnDuplicateBlock) Labels() TsAddLabels {
	c.cs.s = append(c.cs.s, "LABELS")
	return (TsAddLabels)(c)
}

func (c TsAddOnDuplicateBlock) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type TsAddOnDuplicateFirst Completed

func (c TsAddOnDuplicateFirst) Labels() TsAddLabels {
	c.cs.s = append(c.cs.s, "LABELS")
	return (TsAddLabels)(c)
}

func (c TsAddOnDuplicateFirst) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type TsAddOnDuplicateLast Completed

func (c TsAddOnDuplicateLast) Labels() TsAddLabels {
	c.cs.s = append(c.cs.s, "LABELS")
	return (TsAddLabels)(c)
}

func (c TsAddOnDuplicateLast) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type TsAddOnDuplicateMax Completed

func (c TsAddOnDuplicateMax) Labels() TsAddLabels {
	c.cs.s = append(c.cs.s, "LABELS")
	return (TsAddLabels)(c)
}

func (c TsAddOnDuplicateMax) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type TsAddOnDuplicateMin Completed

func (c TsAddOnDuplicateMin) Labels() TsAddLabels {
	c.cs.s = append(c.cs.s, "LABELS")
	return (TsAddLabels)(c)
}

func (c TsAddOnDuplicateMin) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type TsAddOnDuplicateSum Completed

func (c TsAddOnDuplicateSum) Labels() TsAddLabels {
	c.cs.s = append(c.cs.s, "LABELS")
	return (TsAddLabels)(c)
}

func (c TsAddOnDuplicateSum) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type TsAddRetention Completed

func (c TsAddRetention) EncodingUncompressed() TsAddEncodingUncompressed {
	c.cs.s = append(c.cs.s, "ENCODING", "UNCOMPRESSED")
	return (TsAddEncodingUncompressed)(c)
}

func (c TsAddRetention) EncodingCompressed() TsAddEncodingCompressed {
	c.cs.s = append(c.cs.s, "ENCODING", "COMPRESSED")
	return (TsAddEncodingCompressed)(c)
}

func (c TsAddRetention) ChunkSize(size int64) TsAddChunkSize {
	c.cs.s = append(c.cs.s, "CHUNK_SIZE", strconv.FormatInt(size, 10))
	return (TsAddChunkSize)(c)
}

func (c TsAddRetention) OnDuplicateBlock() TsAddOnDuplicateBlock {
	c.cf = blockTag
	c.cs.s = append(c.cs.s, "ON_DUPLICATE", "BLOCK")
	return (TsAddOnDuplicateBlock)(c)
}

func (c TsAddRetention) OnDuplicateFirst() TsAddOnDuplicateFirst {
	c.cs.s = append(c.cs.s, "ON_DUPLICATE", "FIRST")
	return (TsAddOnDuplicateFirst)(c)
}

func (c TsAddRetention) OnDuplicateLast() TsAddOnDuplicateLast {
	c.cs.s = append(c.cs.s, "ON_DUPLICATE", "LAST")
	return (TsAddOnDuplicateLast)(c)
}

func (c TsAddRetention) OnDuplicateMin() TsAddOnDuplicateMin {
	c.cs.s = append(c.cs.s, "ON_DUPLICATE", "MIN")
	return (TsAddOnDuplicateMin)(c)
}

func (c TsAddRetention) OnDuplicateMax() TsAddOnDuplicateMax {
	c.cs.s = append(c.cs.s, "ON_DUPLICATE", "MAX")
	return (TsAddOnDuplicateMax)(c)
}

func (c TsAddRetention) OnDuplicateSum() TsAddOnDuplicateSum {
	c.cs.s = append(c.cs.s, "ON_DUPLICATE", "SUM")
	return (TsAddOnDuplicateSum)(c)
}

func (c TsAddRetention) Labels() TsAddLabels {
	c.cs.s = append(c.cs.s, "LABELS")
	return (TsAddLabels)(c)
}

func (c TsAddRetention) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type TsAddTimestamp Completed

func (c TsAddTimestamp) Value(value float64) TsAddValue {
	c.cs.s = append(c.cs.s, strconv.FormatFloat(value, 'f', -1, 64))
	return (TsAddValue)(c)
}

type TsAddValue Completed

func (c TsAddValue) Retention(retentionperiod int64) TsAddRetention {
	c.cs.s = append(c.cs.s, "RETENTION", strconv.FormatInt(retentionperiod, 10))
	return (TsAddRetention)(c)
}

func (c TsAddValue) EncodingUncompressed() TsAddEncodingUncompressed {
	c.cs.s = append(c.cs.s, "ENCODING", "UNCOMPRESSED")
	return (TsAddEncodingUncompressed)(c)
}

func (c TsAddValue) EncodingCompressed() TsAddEncodingCompressed {
	c.cs.s = append(c.cs.s, "ENCODING", "COMPRESSED")
	return (TsAddEncodingCompressed)(c)
}

func (c TsAddValue) ChunkSize(size int64) TsAddChunkSize {
	c.cs.s = append(c.cs.s, "CHUNK_SIZE", strconv.FormatInt(size, 10))
	return (TsAddChunkSize)(c)
}

func (c TsAddValue) OnDuplicateBlock() TsAddOnDuplicateBlock {
	c.cf = blockTag
	c.cs.s = append(c.cs.s, "ON_DUPLICATE", "BLOCK")
	return (TsAddOnDuplicateBlock)(c)
}

func (c TsAddValue) OnDuplicateFirst() TsAddOnDuplicateFirst {
	c.cs.s = append(c.cs.s, "ON_DUPLICATE", "FIRST")
	return (TsAddOnDuplicateFirst)(c)
}

func (c TsAddValue) OnDuplicateLast() TsAddOnDuplicateLast {
	c.cs.s = append(c.cs.s, "ON_DUPLICATE", "LAST")
	return (TsAddOnDuplicateLast)(c)
}

func (c TsAddValue) OnDuplicateMin() TsAddOnDuplicateMin {
	c.cs.s = append(c.cs.s, "ON_DUPLICATE", "MIN")
	return (TsAddOnDuplicateMin)(c)
}

func (c TsAddValue) OnDuplicateMax() TsAddOnDuplicateMax {
	c.cs.s = append(c.cs.s, "ON_DUPLICATE", "MAX")
	return (TsAddOnDuplicateMax)(c)
}

func (c TsAddValue) OnDuplicateSum() TsAddOnDuplicateSum {
	c.cs.s = append(c.cs.s, "ON_DUPLICATE", "SUM")
	return (TsAddOnDuplicateSum)(c)
}

func (c TsAddValue) Labels() TsAddLabels {
	c.cs.s = append(c.cs.s, "LABELS")
	return (TsAddLabels)(c)
}

func (c TsAddValue) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type TsAlter Completed

func (b Builder) TsAlter() (c TsAlter) {
	c = TsAlter{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "TS.ALTER")
	return c
}

func (c TsAlter) Key(key string) TsAlterKey {
	if c.ks&NoSlot == NoSlot {
		c.ks = NoSlot | slot(key)
	} else {
		c.ks = check(c.ks, slot(key))
	}
	c.cs.s = append(c.cs.s, key)
	return (TsAlterKey)(c)
}

type TsAlterChunkSize Completed

func (c TsAlterChunkSize) DuplicatePolicyBlock() TsAlterDuplicatePolicyBlock {
	c.cf = blockTag
	c.cs.s = append(c.cs.s, "DUPLICATE_POLICY", "BLOCK")
	return (TsAlterDuplicatePolicyBlock)(c)
}

func (c TsAlterChunkSize) DuplicatePolicyFirst() TsAlterDuplicatePolicyFirst {
	c.cs.s = append(c.cs.s, "DUPLICATE_POLICY", "FIRST")
	return (TsAlterDuplicatePolicyFirst)(c)
}

func (c TsAlterChunkSize) DuplicatePolicyLast() TsAlterDuplicatePolicyLast {
	c.cs.s = append(c.cs.s, "DUPLICATE_POLICY", "LAST")
	return (TsAlterDuplicatePolicyLast)(c)
}

func (c TsAlterChunkSize) DuplicatePolicyMin() TsAlterDuplicatePolicyMin {
	c.cs.s = append(c.cs.s, "DUPLICATE_POLICY", "MIN")
	return (TsAlterDuplicatePolicyMin)(c)
}

func (c TsAlterChunkSize) DuplicatePolicyMax() TsAlterDuplicatePolicyMax {
	c.cs.s = append(c.cs.s, "DUPLICATE_POLICY", "MAX")
	return (TsAlterDuplicatePolicyMax)(c)
}

func (c TsAlterChunkSize) DuplicatePolicySum() TsAlterDuplicatePolicySum {
	c.cs.s = append(c.cs.s, "DUPLICATE_POLICY", "SUM")
	return (TsAlterDuplicatePolicySum)(c)
}

func (c TsAlterChunkSize) Labels() TsAlterLabels {
	c.cs.s = append(c.cs.s, "LABELS")
	return (TsAlterLabels)(c)
}

func (c TsAlterChunkSize) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type TsAlterDuplicatePolicyBlock Completed

func (c TsAlterDuplicatePolicyBlock) Labels() TsAlterLabels {
	c.cs.s = append(c.cs.s, "LABELS")
	return (TsAlterLabels)(c)
}

func (c TsAlterDuplicatePolicyBlock) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type TsAlterDuplicatePolicyFirst Completed

func (c TsAlterDuplicatePolicyFirst) Labels() TsAlterLabels {
	c.cs.s = append(c.cs.s, "LABELS")
	return (TsAlterLabels)(c)
}

func (c TsAlterDuplicatePolicyFirst) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type TsAlterDuplicatePolicyLast Completed

func (c TsAlterDuplicatePolicyLast) Labels() TsAlterLabels {
	c.cs.s = append(c.cs.s, "LABELS")
	return (TsAlterLabels)(c)
}

func (c TsAlterDuplicatePolicyLast) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type TsAlterDuplicatePolicyMax Completed

func (c TsAlterDuplicatePolicyMax) Labels() TsAlterLabels {
	c.cs.s = append(c.cs.s, "LABELS")
	return (TsAlterLabels)(c)
}

func (c TsAlterDuplicatePolicyMax) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type TsAlterDuplicatePolicyMin Completed

func (c TsAlterDuplicatePolicyMin) Labels() TsAlterLabels {
	c.cs.s = append(c.cs.s, "LABELS")
	return (TsAlterLabels)(c)
}

func (c TsAlterDuplicatePolicyMin) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type TsAlterDuplicatePolicySum Completed

func (c TsAlterDuplicatePolicySum) Labels() TsAlterLabels {
	c.cs.s = append(c.cs.s, "LABELS")
	return (TsAlterLabels)(c)
}

func (c TsAlterDuplicatePolicySum) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type TsAlterKey Completed

func (c TsAlterKey) Retention(retentionperiod int64) TsAlterRetention {
	c.cs.s = append(c.cs.s, "RETENTION", strconv.FormatInt(retentionperiod, 10))
	return (TsAlterRetention)(c)
}

func (c TsAlterKey) ChunkSize(size int64) TsAlterChunkSize {
	c.cs.s = append(c.cs.s, "CHUNK_SIZE", strconv.FormatInt(size, 10))
	return (TsAlterChunkSize)(c)
}

func (c TsAlterKey) DuplicatePolicyBlock() TsAlterDuplicatePolicyBlock {
	c.cf = blockTag
	c.cs.s = append(c.cs.s, "DUPLICATE_POLICY", "BLOCK")
	return (TsAlterDuplicatePolicyBlock)(c)
}

func (c TsAlterKey) DuplicatePolicyFirst() TsAlterDuplicatePolicyFirst {
	c.cs.s = append(c.cs.s, "DUPLICATE_POLICY", "FIRST")
	return (TsAlterDuplicatePolicyFirst)(c)
}

func (c TsAlterKey) DuplicatePolicyLast() TsAlterDuplicatePolicyLast {
	c.cs.s = append(c.cs.s, "DUPLICATE_POLICY", "LAST")
	return (TsAlterDuplicatePolicyLast)(c)
}

func (c TsAlterKey) DuplicatePolicyMin() TsAlterDuplicatePolicyMin {
	c.cs.s = append(c.cs.s, "DUPLICATE_POLICY", "MIN")
	return (TsAlterDuplicatePolicyMin)(c)
}

func (c TsAlterKey) DuplicatePolicyMax() TsAlterDuplicatePolicyMax {
	c.cs.s = append(c.cs.s, "DUPLICATE_POLICY", "MAX")
	return (TsAlterDuplicatePolicyMax)(c)
}

func (c TsAlterKey) DuplicatePolicySum() TsAlterDuplicatePolicySum {
	c.cs.s = append(c.cs.s, "DUPLICATE_POLICY", "SUM")
	return (TsAlterDuplicatePolicySum)(c)
}

func (c TsAlterKey) Labels() TsAlterLabels {
	c.cs.s = append(c.cs.s, "LABELS")
	return (TsAlterLabels)(c)
}

func (c TsAlterKey) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type TsAlterLabels Completed

func (c TsAlterLabels) Labels(label string, value string) TsAlterLabels {
	c.cs.s = append(c.cs.s, label, value)
	return c
}

func (c TsAlterLabels) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type TsAlterRetention Completed

func (c TsAlterRetention) ChunkSize(size int64) TsAlterChunkSize {
	c.cs.s = append(c.cs.s, "CHUNK_SIZE", strconv.FormatInt(size, 10))
	return (TsAlterChunkSize)(c)
}

func (c TsAlterRetention) DuplicatePolicyBlock() TsAlterDuplicatePolicyBlock {
	c.cf = blockTag
	c.cs.s = append(c.cs.s, "DUPLICATE_POLICY", "BLOCK")
	return (TsAlterDuplicatePolicyBlock)(c)
}

func (c TsAlterRetention) DuplicatePolicyFirst() TsAlterDuplicatePolicyFirst {
	c.cs.s = append(c.cs.s, "DUPLICATE_POLICY", "FIRST")
	return (TsAlterDuplicatePolicyFirst)(c)
}

func (c TsAlterRetention) DuplicatePolicyLast() TsAlterDuplicatePolicyLast {
	c.cs.s = append(c.cs.s, "DUPLICATE_POLICY", "LAST")
	return (TsAlterDuplicatePolicyLast)(c)
}

func (c TsAlterRetention) DuplicatePolicyMin() TsAlterDuplicatePolicyMin {
	c.cs.s = append(c.cs.s, "DUPLICATE_POLICY", "MIN")
	return (TsAlterDuplicatePolicyMin)(c)
}

func (c TsAlterRetention) DuplicatePolicyMax() TsAlterDuplicatePolicyMax {
	c.cs.s = append(c.cs.s, "DUPLICATE_POLICY", "MAX")
	return (TsAlterDuplicatePolicyMax)(c)
}

func (c TsAlterRetention) DuplicatePolicySum() TsAlterDuplicatePolicySum {
	c.cs.s = append(c.cs.s, "DUPLICATE_POLICY", "SUM")
	return (TsAlterDuplicatePolicySum)(c)
}

func (c TsAlterRetention) Labels() TsAlterLabels {
	c.cs.s = append(c.cs.s, "LABELS")
	return (TsAlterLabels)(c)
}

func (c TsAlterRetention) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type TsCreate Completed

func (b Builder) TsCreate() (c TsCreate) {
	c = TsCreate{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "TS.CREATE")
	return c
}

func (c TsCreate) Key(key string) TsCreateKey {
	if c.ks&NoSlot == NoSlot {
		c.ks = NoSlot | slot(key)
	} else {
		c.ks = check(c.ks, slot(key))
	}
	c.cs.s = append(c.cs.s, key)
	return (TsCreateKey)(c)
}

type TsCreateChunkSize Completed

func (c TsCreateChunkSize) DuplicatePolicyBlock() TsCreateDuplicatePolicyBlock {
	c.cf = blockTag
	c.cs.s = append(c.cs.s, "DUPLICATE_POLICY", "BLOCK")
	return (TsCreateDuplicatePolicyBlock)(c)
}

func (c TsCreateChunkSize) DuplicatePolicyFirst() TsCreateDuplicatePolicyFirst {
	c.cs.s = append(c.cs.s, "DUPLICATE_POLICY", "FIRST")
	return (TsCreateDuplicatePolicyFirst)(c)
}

func (c TsCreateChunkSize) DuplicatePolicyLast() TsCreateDuplicatePolicyLast {
	c.cs.s = append(c.cs.s, "DUPLICATE_POLICY", "LAST")
	return (TsCreateDuplicatePolicyLast)(c)
}

func (c TsCreateChunkSize) DuplicatePolicyMin() TsCreateDuplicatePolicyMin {
	c.cs.s = append(c.cs.s, "DUPLICATE_POLICY", "MIN")
	return (TsCreateDuplicatePolicyMin)(c)
}

func (c TsCreateChunkSize) DuplicatePolicyMax() TsCreateDuplicatePolicyMax {
	c.cs.s = append(c.cs.s, "DUPLICATE_POLICY", "MAX")
	return (TsCreateDuplicatePolicyMax)(c)
}

func (c TsCreateChunkSize) DuplicatePolicySum() TsCreateDuplicatePolicySum {
	c.cs.s = append(c.cs.s, "DUPLICATE_POLICY", "SUM")
	return (TsCreateDuplicatePolicySum)(c)
}

func (c TsCreateChunkSize) Labels() TsCreateLabels {
	c.cs.s = append(c.cs.s, "LABELS")
	return (TsCreateLabels)(c)
}

func (c TsCreateChunkSize) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type TsCreateDuplicatePolicyBlock Completed

func (c TsCreateDuplicatePolicyBlock) Labels() TsCreateLabels {
	c.cs.s = append(c.cs.s, "LABELS")
	return (TsCreateLabels)(c)
}

func (c TsCreateDuplicatePolicyBlock) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type TsCreateDuplicatePolicyFirst Completed

func (c TsCreateDuplicatePolicyFirst) Labels() TsCreateLabels {
	c.cs.s = append(c.cs.s, "LABELS")
	return (TsCreateLabels)(c)
}

func (c TsCreateDuplicatePolicyFirst) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type TsCreateDuplicatePolicyLast Completed

func (c TsCreateDuplicatePolicyLast) Labels() TsCreateLabels {
	c.cs.s = append(c.cs.s, "LABELS")
	return (TsCreateLabels)(c)
}

func (c TsCreateDuplicatePolicyLast) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type TsCreateDuplicatePolicyMax Completed

func (c TsCreateDuplicatePolicyMax) Labels() TsCreateLabels {
	c.cs.s = append(c.cs.s, "LABELS")
	return (TsCreateLabels)(c)
}

func (c TsCreateDuplicatePolicyMax) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type TsCreateDuplicatePolicyMin Completed

func (c TsCreateDuplicatePolicyMin) Labels() TsCreateLabels {
	c.cs.s = append(c.cs.s, "LABELS")
	return (TsCreateLabels)(c)
}

func (c TsCreateDuplicatePolicyMin) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type TsCreateDuplicatePolicySum Completed

func (c TsCreateDuplicatePolicySum) Labels() TsCreateLabels {
	c.cs.s = append(c.cs.s, "LABELS")
	return (TsCreateLabels)(c)
}

func (c TsCreateDuplicatePolicySum) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type TsCreateEncodingCompressed Completed

func (c TsCreateEncodingCompressed) ChunkSize(size int64) TsCreateChunkSize {
	c.cs.s = append(c.cs.s, "CHUNK_SIZE", strconv.FormatInt(size, 10))
	return (TsCreateChunkSize)(c)
}

func (c TsCreateEncodingCompressed) DuplicatePolicyBlock() TsCreateDuplicatePolicyBlock {
	c.cf = blockTag
	c.cs.s = append(c.cs.s, "DUPLICATE_POLICY", "BLOCK")
	return (TsCreateDuplicatePolicyBlock)(c)
}

func (c TsCreateEncodingCompressed) DuplicatePolicyFirst() TsCreateDuplicatePolicyFirst {
	c.cs.s = append(c.cs.s, "DUPLICATE_POLICY", "FIRST")
	return (TsCreateDuplicatePolicyFirst)(c)
}

func (c TsCreateEncodingCompressed) DuplicatePolicyLast() TsCreateDuplicatePolicyLast {
	c.cs.s = append(c.cs.s, "DUPLICATE_POLICY", "LAST")
	return (TsCreateDuplicatePolicyLast)(c)
}

func (c TsCreateEncodingCompressed) DuplicatePolicyMin() TsCreateDuplicatePolicyMin {
	c.cs.s = append(c.cs.s, "DUPLICATE_POLICY", "MIN")
	return (TsCreateDuplicatePolicyMin)(c)
}

func (c TsCreateEncodingCompressed) DuplicatePolicyMax() TsCreateDuplicatePolicyMax {
	c.cs.s = append(c.cs.s, "DUPLICATE_POLICY", "MAX")
	return (TsCreateDuplicatePolicyMax)(c)
}

func (c TsCreateEncodingCompressed) DuplicatePolicySum() TsCreateDuplicatePolicySum {
	c.cs.s = append(c.cs.s, "DUPLICATE_POLICY", "SUM")
	return (TsCreateDuplicatePolicySum)(c)
}

func (c TsCreateEncodingCompressed) Labels() TsCreateLabels {
	c.cs.s = append(c.cs.s, "LABELS")
	return (TsCreateLabels)(c)
}

func (c TsCreateEncodingCompressed) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type TsCreateEncodingUncompressed Completed

func (c TsCreateEncodingUncompressed) ChunkSize(size int64) TsCreateChunkSize {
	c.cs.s = append(c.cs.s, "CHUNK_SIZE", strconv.FormatInt(size, 10))
	return (TsCreateChunkSize)(c)
}

func (c TsCreateEncodingUncompressed) DuplicatePolicyBlock() TsCreateDuplicatePolicyBlock {
	c.cf = blockTag
	c.cs.s = append(c.cs.s, "DUPLICATE_POLICY", "BLOCK")
	return (TsCreateDuplicatePolicyBlock)(c)
}

func (c TsCreateEncodingUncompressed) DuplicatePolicyFirst() TsCreateDuplicatePolicyFirst {
	c.cs.s = append(c.cs.s, "DUPLICATE_POLICY", "FIRST")
	return (TsCreateDuplicatePolicyFirst)(c)
}

func (c TsCreateEncodingUncompressed) DuplicatePolicyLast() TsCreateDuplicatePolicyLast {
	c.cs.s = append(c.cs.s, "DUPLICATE_POLICY", "LAST")
	return (TsCreateDuplicatePolicyLast)(c)
}

func (c TsCreateEncodingUncompressed) DuplicatePolicyMin() TsCreateDuplicatePolicyMin {
	c.cs.s = append(c.cs.s, "DUPLICATE_POLICY", "MIN")
	return (TsCreateDuplicatePolicyMin)(c)
}

func (c TsCreateEncodingUncompressed) DuplicatePolicyMax() TsCreateDuplicatePolicyMax {
	c.cs.s = append(c.cs.s, "DUPLICATE_POLICY", "MAX")
	return (TsCreateDuplicatePolicyMax)(c)
}

func (c TsCreateEncodingUncompressed) DuplicatePolicySum() TsCreateDuplicatePolicySum {
	c.cs.s = append(c.cs.s, "DUPLICATE_POLICY", "SUM")
	return (TsCreateDuplicatePolicySum)(c)
}

func (c TsCreateEncodingUncompressed) Labels() TsCreateLabels {
	c.cs.s = append(c.cs.s, "LABELS")
	return (TsCreateLabels)(c)
}

func (c TsCreateEncodingUncompressed) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type TsCreateKey Completed

func (c TsCreateKey) Retention(retentionperiod int64) TsCreateRetention {
	c.cs.s = append(c.cs.s, "RETENTION", strconv.FormatInt(retentionperiod, 10))
	return (TsCreateRetention)(c)
}

func (c TsCreateKey) EncodingUncompressed() TsCreateEncodingUncompressed {
	c.cs.s = append(c.cs.s, "ENCODING", "UNCOMPRESSED")
	return (TsCreateEncodingUncompressed)(c)
}

func (c TsCreateKey) EncodingCompressed() TsCreateEncodingCompressed {
	c.cs.s = append(c.cs.s, "ENCODING", "COMPRESSED")
	return (TsCreateEncodingCompressed)(c)
}

func (c TsCreateKey) ChunkSize(size int64) TsCreateChunkSize {
	c.cs.s = append(c.cs.s, "CHUNK_SIZE", strconv.FormatInt(size, 10))
	return (TsCreateChunkSize)(c)
}

func (c TsCreateKey) DuplicatePolicyBlock() TsCreateDuplicatePolicyBlock {
	c.cf = blockTag
	c.cs.s = append(c.cs.s, "DUPLICATE_POLICY", "BLOCK")
	return (TsCreateDuplicatePolicyBlock)(c)
}

func (c TsCreateKey) DuplicatePolicyFirst() TsCreateDuplicatePolicyFirst {
	c.cs.s = append(c.cs.s, "DUPLICATE_POLICY", "FIRST")
	return (TsCreateDuplicatePolicyFirst)(c)
}

func (c TsCreateKey) DuplicatePolicyLast() TsCreateDuplicatePolicyLast {
	c.cs.s = append(c.cs.s, "DUPLICATE_POLICY", "LAST")
	return (TsCreateDuplicatePolicyLast)(c)
}

func (c TsCreateKey) DuplicatePolicyMin() TsCreateDuplicatePolicyMin {
	c.cs.s = append(c.cs.s, "DUPLICATE_POLICY", "MIN")
	return (TsCreateDuplicatePolicyMin)(c)
}

func (c TsCreateKey) DuplicatePolicyMax() TsCreateDuplicatePolicyMax {
	c.cs.s = append(c.cs.s, "DUPLICATE_POLICY", "MAX")
	return (TsCreateDuplicatePolicyMax)(c)
}

func (c TsCreateKey) DuplicatePolicySum() TsCreateDuplicatePolicySum {
	c.cs.s = append(c.cs.s, "DUPLICATE_POLICY", "SUM")
	return (TsCreateDuplicatePolicySum)(c)
}

func (c TsCreateKey) Labels() TsCreateLabels {
	c.cs.s = append(c.cs.s, "LABELS")
	return (TsCreateLabels)(c)
}

func (c TsCreateKey) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type TsCreateLabels Completed

func (c TsCreateLabels) Labels(label string, value string) TsCreateLabels {
	c.cs.s = append(c.cs.s, label, value)
	return c
}

func (c TsCreateLabels) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type TsCreateRetention Completed

func (c TsCreateRetention) EncodingUncompressed() TsCreateEncodingUncompressed {
	c.cs.s = append(c.cs.s, "ENCODING", "UNCOMPRESSED")
	return (TsCreateEncodingUncompressed)(c)
}

func (c TsCreateRetention) EncodingCompressed() TsCreateEncodingCompressed {
	c.cs.s = append(c.cs.s, "ENCODING", "COMPRESSED")
	return (TsCreateEncodingCompressed)(c)
}

func (c TsCreateRetention) ChunkSize(size int64) TsCreateChunkSize {
	c.cs.s = append(c.cs.s, "CHUNK_SIZE", strconv.FormatInt(size, 10))
	return (TsCreateChunkSize)(c)
}

func (c TsCreateRetention) DuplicatePolicyBlock() TsCreateDuplicatePolicyBlock {
	c.cf = blockTag
	c.cs.s = append(c.cs.s, "DUPLICATE_POLICY", "BLOCK")
	return (TsCreateDuplicatePolicyBlock)(c)
}

func (c TsCreateRetention) DuplicatePolicyFirst() TsCreateDuplicatePolicyFirst {
	c.cs.s = append(c.cs.s, "DUPLICATE_POLICY", "FIRST")
	return (TsCreateDuplicatePolicyFirst)(c)
}

func (c TsCreateRetention) DuplicatePolicyLast() TsCreateDuplicatePolicyLast {
	c.cs.s = append(c.cs.s, "DUPLICATE_POLICY", "LAST")
	return (TsCreateDuplicatePolicyLast)(c)
}

func (c TsCreateRetention) DuplicatePolicyMin() TsCreateDuplicatePolicyMin {
	c.cs.s = append(c.cs.s, "DUPLICATE_POLICY", "MIN")
	return (TsCreateDuplicatePolicyMin)(c)
}

func (c TsCreateRetention) DuplicatePolicyMax() TsCreateDuplicatePolicyMax {
	c.cs.s = append(c.cs.s, "DUPLICATE_POLICY", "MAX")
	return (TsCreateDuplicatePolicyMax)(c)
}

func (c TsCreateRetention) DuplicatePolicySum() TsCreateDuplicatePolicySum {
	c.cs.s = append(c.cs.s, "DUPLICATE_POLICY", "SUM")
	return (TsCreateDuplicatePolicySum)(c)
}

func (c TsCreateRetention) Labels() TsCreateLabels {
	c.cs.s = append(c.cs.s, "LABELS")
	return (TsCreateLabels)(c)
}

func (c TsCreateRetention) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type TsCreaterule Completed

func (b Builder) TsCreaterule() (c TsCreaterule) {
	c = TsCreaterule{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "TS.CREATERULE")
	return c
}

func (c TsCreaterule) Sourcekey(sourcekey string) TsCreateruleSourcekey {
	if c.ks&NoSlot == NoSlot {
		c.ks = NoSlot | slot(sourcekey)
	} else {
		c.ks = check(c.ks, slot(sourcekey))
	}
	c.cs.s = append(c.cs.s, sourcekey)
	return (TsCreateruleSourcekey)(c)
}

type TsCreateruleAggregationAvg Completed

func (c TsCreateruleAggregationAvg) Bucketduration(bucketduration int64) TsCreateruleBucketduration {
	c.cs.s = append(c.cs.s, strconv.FormatInt(bucketduration, 10))
	return (TsCreateruleBucketduration)(c)
}

type TsCreateruleAggregationCount Completed

func (c TsCreateruleAggregationCount) Bucketduration(bucketduration int64) TsCreateruleBucketduration {
	c.cs.s = append(c.cs.s, strconv.FormatInt(bucketduration, 10))
	return (TsCreateruleBucketduration)(c)
}

type TsCreateruleAggregationFirst Completed

func (c TsCreateruleAggregationFirst) Bucketduration(bucketduration int64) TsCreateruleBucketduration {
	c.cs.s = append(c.cs.s, strconv.FormatInt(bucketduration, 10))
	return (TsCreateruleBucketduration)(c)
}

type TsCreateruleAggregationLast Completed

func (c TsCreateruleAggregationLast) Bucketduration(bucketduration int64) TsCreateruleBucketduration {
	c.cs.s = append(c.cs.s, strconv.FormatInt(bucketduration, 10))
	return (TsCreateruleBucketduration)(c)
}

type TsCreateruleAggregationMax Completed

func (c TsCreateruleAggregationMax) Bucketduration(bucketduration int64) TsCreateruleBucketduration {
	c.cs.s = append(c.cs.s, strconv.FormatInt(bucketduration, 10))
	return (TsCreateruleBucketduration)(c)
}

type TsCreateruleAggregationMin Completed

func (c TsCreateruleAggregationMin) Bucketduration(bucketduration int64) TsCreateruleBucketduration {
	c.cs.s = append(c.cs.s, strconv.FormatInt(bucketduration, 10))
	return (TsCreateruleBucketduration)(c)
}

type TsCreateruleAggregationRange Completed

func (c TsCreateruleAggregationRange) Bucketduration(bucketduration int64) TsCreateruleBucketduration {
	c.cs.s = append(c.cs.s, strconv.FormatInt(bucketduration, 10))
	return (TsCreateruleBucketduration)(c)
}

type TsCreateruleAggregationStdP Completed

func (c TsCreateruleAggregationStdP) Bucketduration(bucketduration int64) TsCreateruleBucketduration {
	c.cs.s = append(c.cs.s, strconv.FormatInt(bucketduration, 10))
	return (TsCreateruleBucketduration)(c)
}

type TsCreateruleAggregationStdS Completed

func (c TsCreateruleAggregationStdS) Bucketduration(bucketduration int64) TsCreateruleBucketduration {
	c.cs.s = append(c.cs.s, strconv.FormatInt(bucketduration, 10))
	return (TsCreateruleBucketduration)(c)
}

type TsCreateruleAggregationSum Completed

func (c TsCreateruleAggregationSum) Bucketduration(bucketduration int64) TsCreateruleBucketduration {
	c.cs.s = append(c.cs.s, strconv.FormatInt(bucketduration, 10))
	return (TsCreateruleBucketduration)(c)
}

type TsCreateruleAggregationTwa Completed

func (c TsCreateruleAggregationTwa) Bucketduration(bucketduration int64) TsCreateruleBucketduration {
	c.cs.s = append(c.cs.s, strconv.FormatInt(bucketduration, 10))
	return (TsCreateruleBucketduration)(c)
}

type TsCreateruleAggregationVarP Completed

func (c TsCreateruleAggregationVarP) Bucketduration(bucketduration int64) TsCreateruleBucketduration {
	c.cs.s = append(c.cs.s, strconv.FormatInt(bucketduration, 10))
	return (TsCreateruleBucketduration)(c)
}

type TsCreateruleAggregationVarS Completed

func (c TsCreateruleAggregationVarS) Bucketduration(bucketduration int64) TsCreateruleBucketduration {
	c.cs.s = append(c.cs.s, strconv.FormatInt(bucketduration, 10))
	return (TsCreateruleBucketduration)(c)
}

type TsCreateruleAligntimestamp Completed

func (c TsCreateruleAligntimestamp) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type TsCreateruleBucketduration Completed

func (c TsCreateruleBucketduration) Aligntimestamp(aligntimestamp int64) TsCreateruleAligntimestamp {
	c.cs.s = append(c.cs.s, strconv.FormatInt(aligntimestamp, 10))
	return (TsCreateruleAligntimestamp)(c)
}

func (c TsCreateruleBucketduration) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type TsCreateruleDestkey Completed

func (c TsCreateruleDestkey) AggregationAvg() TsCreateruleAggregationAvg {
	c.cs.s = append(c.cs.s, "AGGREGATION", "AVG")
	return (TsCreateruleAggregationAvg)(c)
}

func (c TsCreateruleDestkey) AggregationSum() TsCreateruleAggregationSum {
	c.cs.s = append(c.cs.s, "AGGREGATION", "SUM")
	return (TsCreateruleAggregationSum)(c)
}

func (c TsCreateruleDestkey) AggregationMin() TsCreateruleAggregationMin {
	c.cs.s = append(c.cs.s, "AGGREGATION", "MIN")
	return (TsCreateruleAggregationMin)(c)
}

func (c TsCreateruleDestkey) AggregationMax() TsCreateruleAggregationMax {
	c.cs.s = append(c.cs.s, "AGGREGATION", "MAX")
	return (TsCreateruleAggregationMax)(c)
}

func (c TsCreateruleDestkey) AggregationRange() TsCreateruleAggregationRange {
	c.cs.s = append(c.cs.s, "AGGREGATION", "RANGE")
	return (TsCreateruleAggregationRange)(c)
}

func (c TsCreateruleDestkey) AggregationCount() TsCreateruleAggregationCount {
	c.cs.s = append(c.cs.s, "AGGREGATION", "COUNT")
	return (TsCreateruleAggregationCount)(c)
}

func (c TsCreateruleDestkey) AggregationFirst() TsCreateruleAggregationFirst {
	c.cs.s = append(c.cs.s, "AGGREGATION", "FIRST")
	return (TsCreateruleAggregationFirst)(c)
}

func (c TsCreateruleDestkey) AggregationLast() TsCreateruleAggregationLast {
	c.cs.s = append(c.cs.s, "AGGREGATION", "LAST")
	return (TsCreateruleAggregationLast)(c)
}

func (c TsCreateruleDestkey) AggregationStdP() TsCreateruleAggregationStdP {
	c.cs.s = append(c.cs.s, "AGGREGATION", "STD.P")
	return (TsCreateruleAggregationStdP)(c)
}

func (c TsCreateruleDestkey) AggregationStdS() TsCreateruleAggregationStdS {
	c.cs.s = append(c.cs.s, "AGGREGATION", "STD.S")
	return (TsCreateruleAggregationStdS)(c)
}

func (c TsCreateruleDestkey) AggregationVarP() TsCreateruleAggregationVarP {
	c.cs.s = append(c.cs.s, "AGGREGATION", "VAR.P")
	return (TsCreateruleAggregationVarP)(c)
}

func (c TsCreateruleDestkey) AggregationVarS() TsCreateruleAggregationVarS {
	c.cs.s = append(c.cs.s, "AGGREGATION", "VAR.S")
	return (TsCreateruleAggregationVarS)(c)
}

func (c TsCreateruleDestkey) AggregationTwa() TsCreateruleAggregationTwa {
	c.cs.s = append(c.cs.s, "AGGREGATION", "TWA")
	return (TsCreateruleAggregationTwa)(c)
}

type TsCreateruleSourcekey Completed

func (c TsCreateruleSourcekey) Destkey(destkey string) TsCreateruleDestkey {
	if c.ks&NoSlot == NoSlot {
		c.ks = NoSlot | slot(destkey)
	} else {
		c.ks = check(c.ks, slot(destkey))
	}
	c.cs.s = append(c.cs.s, destkey)
	return (TsCreateruleDestkey)(c)
}

type TsDecrby Completed

func (b Builder) TsDecrby() (c TsDecrby) {
	c = TsDecrby{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "TS.DECRBY")
	return c
}

func (c TsDecrby) Key(key string) TsDecrbyKey {
	if c.ks&NoSlot == NoSlot {
		c.ks = NoSlot | slot(key)
	} else {
		c.ks = check(c.ks, slot(key))
	}
	c.cs.s = append(c.cs.s, key)
	return (TsDecrbyKey)(c)
}

type TsDecrbyChunkSize Completed

func (c TsDecrbyChunkSize) Labels() TsDecrbyLabels {
	c.cs.s = append(c.cs.s, "LABELS")
	return (TsDecrbyLabels)(c)
}

func (c TsDecrbyChunkSize) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type TsDecrbyKey Completed

func (c TsDecrbyKey) Value(value float64) TsDecrbyValue {
	c.cs.s = append(c.cs.s, strconv.FormatFloat(value, 'f', -1, 64))
	return (TsDecrbyValue)(c)
}

type TsDecrbyLabels Completed

func (c TsDecrbyLabels) Labels(label string, value string) TsDecrbyLabels {
	c.cs.s = append(c.cs.s, label, value)
	return c
}

func (c TsDecrbyLabels) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type TsDecrbyRetention Completed

func (c TsDecrbyRetention) Uncompressed() TsDecrbyUncompressed {
	c.cs.s = append(c.cs.s, "UNCOMPRESSED")
	return (TsDecrbyUncompressed)(c)
}

func (c TsDecrbyRetention) ChunkSize(size int64) TsDecrbyChunkSize {
	c.cs.s = append(c.cs.s, "CHUNK_SIZE", strconv.FormatInt(size, 10))
	return (TsDecrbyChunkSize)(c)
}

func (c TsDecrbyRetention) Labels() TsDecrbyLabels {
	c.cs.s = append(c.cs.s, "LABELS")
	return (TsDecrbyLabels)(c)
}

func (c TsDecrbyRetention) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type TsDecrbyTimestamp Completed

func (c TsDecrbyTimestamp) Retention(retentionperiod int64) TsDecrbyRetention {
	c.cs.s = append(c.cs.s, "RETENTION", strconv.FormatInt(retentionperiod, 10))
	return (TsDecrbyRetention)(c)
}

func (c TsDecrbyTimestamp) Uncompressed() TsDecrbyUncompressed {
	c.cs.s = append(c.cs.s, "UNCOMPRESSED")
	return (TsDecrbyUncompressed)(c)
}

func (c TsDecrbyTimestamp) ChunkSize(size int64) TsDecrbyChunkSize {
	c.cs.s = append(c.cs.s, "CHUNK_SIZE", strconv.FormatInt(size, 10))
	return (TsDecrbyChunkSize)(c)
}

func (c TsDecrbyTimestamp) Labels() TsDecrbyLabels {
	c.cs.s = append(c.cs.s, "LABELS")
	return (TsDecrbyLabels)(c)
}

func (c TsDecrbyTimestamp) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type TsDecrbyUncompressed Completed

func (c TsDecrbyUncompressed) ChunkSize(size int64) TsDecrbyChunkSize {
	c.cs.s = append(c.cs.s, "CHUNK_SIZE", strconv.FormatInt(size, 10))
	return (TsDecrbyChunkSize)(c)
}

func (c TsDecrbyUncompressed) Labels() TsDecrbyLabels {
	c.cs.s = append(c.cs.s, "LABELS")
	return (TsDecrbyLabels)(c)
}

func (c TsDecrbyUncompressed) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type TsDecrbyValue Completed

func (c TsDecrbyValue) Timestamp(timestamp int64) TsDecrbyTimestamp {
	c.cs.s = append(c.cs.s, "TIMESTAMP", strconv.FormatInt(timestamp, 10))
	return (TsDecrbyTimestamp)(c)
}

func (c TsDecrbyValue) Retention(retentionperiod int64) TsDecrbyRetention {
	c.cs.s = append(c.cs.s, "RETENTION", strconv.FormatInt(retentionperiod, 10))
	return (TsDecrbyRetention)(c)
}

func (c TsDecrbyValue) Uncompressed() TsDecrbyUncompressed {
	c.cs.s = append(c.cs.s, "UNCOMPRESSED")
	return (TsDecrbyUncompressed)(c)
}

func (c TsDecrbyValue) ChunkSize(size int64) TsDecrbyChunkSize {
	c.cs.s = append(c.cs.s, "CHUNK_SIZE", strconv.FormatInt(size, 10))
	return (TsDecrbyChunkSize)(c)
}

func (c TsDecrbyValue) Labels() TsDecrbyLabels {
	c.cs.s = append(c.cs.s, "LABELS")
	return (TsDecrbyLabels)(c)
}

func (c TsDecrbyValue) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type TsDel Completed

func (b Builder) TsDel() (c TsDel) {
	c = TsDel{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "TS.DEL")
	return c
}

func (c TsDel) Key(key string) TsDelKey {
	if c.ks&NoSlot == NoSlot {
		c.ks = NoSlot | slot(key)
	} else {
		c.ks = check(c.ks, slot(key))
	}
	c.cs.s = append(c.cs.s, key)
	return (TsDelKey)(c)
}

type TsDelFromTimestamp Completed

func (c TsDelFromTimestamp) ToTimestamp(toTimestamp int64) TsDelToTimestamp {
	c.cs.s = append(c.cs.s, strconv.FormatInt(toTimestamp, 10))
	return (TsDelToTimestamp)(c)
}

type TsDelKey Completed

func (c TsDelKey) FromTimestamp(fromTimestamp int64) TsDelFromTimestamp {
	c.cs.s = append(c.cs.s, strconv.FormatInt(fromTimestamp, 10))
	return (TsDelFromTimestamp)(c)
}

type TsDelToTimestamp Completed

func (c TsDelToTimestamp) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type TsDeleterule Completed

func (b Builder) TsDeleterule() (c TsDeleterule) {
	c = TsDeleterule{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "TS.DELETERULE")
	return c
}

func (c TsDeleterule) Sourcekey(sourcekey string) TsDeleteruleSourcekey {
	if c.ks&NoSlot == NoSlot {
		c.ks = NoSlot | slot(sourcekey)
	} else {
		c.ks = check(c.ks, slot(sourcekey))
	}
	c.cs.s = append(c.cs.s, sourcekey)
	return (TsDeleteruleSourcekey)(c)
}

type TsDeleteruleDestkey Completed

func (c TsDeleteruleDestkey) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type TsDeleteruleSourcekey Completed

func (c TsDeleteruleSourcekey) Destkey(destkey string) TsDeleteruleDestkey {
	if c.ks&NoSlot == NoSlot {
		c.ks = NoSlot | slot(destkey)
	} else {
		c.ks = check(c.ks, slot(destkey))
	}
	c.cs.s = append(c.cs.s, destkey)
	return (TsDeleteruleDestkey)(c)
}

type TsGet Completed

func (b Builder) TsGet() (c TsGet) {
	c = TsGet{cs: get(), ks: b.ks, cf: readonly}
	c.cs.s = append(c.cs.s, "TS.GET")
	return c
}

func (c TsGet) Key(key string) TsGetKey {
	if c.ks&NoSlot == NoSlot {
		c.ks = NoSlot | slot(key)
	} else {
		c.ks = check(c.ks, slot(key))
	}
	c.cs.s = append(c.cs.s, key)
	return (TsGetKey)(c)
}

type TsGetKey Completed

func (c TsGetKey) Latest() TsGetLatest {
	c.cs.s = append(c.cs.s, "LATEST")
	return (TsGetLatest)(c)
}

func (c TsGetKey) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type TsGetLatest Completed

func (c TsGetLatest) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type TsIncrby Completed

func (b Builder) TsIncrby() (c TsIncrby) {
	c = TsIncrby{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "TS.INCRBY")
	return c
}

func (c TsIncrby) Key(key string) TsIncrbyKey {
	if c.ks&NoSlot == NoSlot {
		c.ks = NoSlot | slot(key)
	} else {
		c.ks = check(c.ks, slot(key))
	}
	c.cs.s = append(c.cs.s, key)
	return (TsIncrbyKey)(c)
}

type TsIncrbyChunkSize Completed

func (c TsIncrbyChunkSize) Labels() TsIncrbyLabels {
	c.cs.s = append(c.cs.s, "LABELS")
	return (TsIncrbyLabels)(c)
}

func (c TsIncrbyChunkSize) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type TsIncrbyKey Completed

func (c TsIncrbyKey) Value(value float64) TsIncrbyValue {
	c.cs.s = append(c.cs.s, strconv.FormatFloat(value, 'f', -1, 64))
	return (TsIncrbyValue)(c)
}

type TsIncrbyLabels Completed

func (c TsIncrbyLabels) Labels(label string, value string) TsIncrbyLabels {
	c.cs.s = append(c.cs.s, label, value)
	return c
}

func (c TsIncrbyLabels) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type TsIncrbyRetention Completed

func (c TsIncrbyRetention) Uncompressed() TsIncrbyUncompressed {
	c.cs.s = append(c.cs.s, "UNCOMPRESSED")
	return (TsIncrbyUncompressed)(c)
}

func (c TsIncrbyRetention) ChunkSize(size int64) TsIncrbyChunkSize {
	c.cs.s = append(c.cs.s, "CHUNK_SIZE", strconv.FormatInt(size, 10))
	return (TsIncrbyChunkSize)(c)
}

func (c TsIncrbyRetention) Labels() TsIncrbyLabels {
	c.cs.s = append(c.cs.s, "LABELS")
	return (TsIncrbyLabels)(c)
}

func (c TsIncrbyRetention) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type TsIncrbyTimestamp Completed

func (c TsIncrbyTimestamp) Retention(retentionperiod int64) TsIncrbyRetention {
	c.cs.s = append(c.cs.s, "RETENTION", strconv.FormatInt(retentionperiod, 10))
	return (TsIncrbyRetention)(c)
}

func (c TsIncrbyTimestamp) Uncompressed() TsIncrbyUncompressed {
	c.cs.s = append(c.cs.s, "UNCOMPRESSED")
	return (TsIncrbyUncompressed)(c)
}

func (c TsIncrbyTimestamp) ChunkSize(size int64) TsIncrbyChunkSize {
	c.cs.s = append(c.cs.s, "CHUNK_SIZE", strconv.FormatInt(size, 10))
	return (TsIncrbyChunkSize)(c)
}

func (c TsIncrbyTimestamp) Labels() TsIncrbyLabels {
	c.cs.s = append(c.cs.s, "LABELS")
	return (TsIncrbyLabels)(c)
}

func (c TsIncrbyTimestamp) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type TsIncrbyUncompressed Completed

func (c TsIncrbyUncompressed) ChunkSize(size int64) TsIncrbyChunkSize {
	c.cs.s = append(c.cs.s, "CHUNK_SIZE", strconv.FormatInt(size, 10))
	return (TsIncrbyChunkSize)(c)
}

func (c TsIncrbyUncompressed) Labels() TsIncrbyLabels {
	c.cs.s = append(c.cs.s, "LABELS")
	return (TsIncrbyLabels)(c)
}

func (c TsIncrbyUncompressed) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type TsIncrbyValue Completed

func (c TsIncrbyValue) Timestamp(timestamp int64) TsIncrbyTimestamp {
	c.cs.s = append(c.cs.s, "TIMESTAMP", strconv.FormatInt(timestamp, 10))
	return (TsIncrbyTimestamp)(c)
}

func (c TsIncrbyValue) Retention(retentionperiod int64) TsIncrbyRetention {
	c.cs.s = append(c.cs.s, "RETENTION", strconv.FormatInt(retentionperiod, 10))
	return (TsIncrbyRetention)(c)
}

func (c TsIncrbyValue) Uncompressed() TsIncrbyUncompressed {
	c.cs.s = append(c.cs.s, "UNCOMPRESSED")
	return (TsIncrbyUncompressed)(c)
}

func (c TsIncrbyValue) ChunkSize(size int64) TsIncrbyChunkSize {
	c.cs.s = append(c.cs.s, "CHUNK_SIZE", strconv.FormatInt(size, 10))
	return (TsIncrbyChunkSize)(c)
}

func (c TsIncrbyValue) Labels() TsIncrbyLabels {
	c.cs.s = append(c.cs.s, "LABELS")
	return (TsIncrbyLabels)(c)
}

func (c TsIncrbyValue) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type TsInfo Completed

func (b Builder) TsInfo() (c TsInfo) {
	c = TsInfo{cs: get(), ks: b.ks, cf: readonly}
	c.cs.s = append(c.cs.s, "TS.INFO")
	return c
}

func (c TsInfo) Key(key string) TsInfoKey {
	if c.ks&NoSlot == NoSlot {
		c.ks = NoSlot | slot(key)
	} else {
		c.ks = check(c.ks, slot(key))
	}
	c.cs.s = append(c.cs.s, key)
	return (TsInfoKey)(c)
}

type TsInfoDebug Completed

func (c TsInfoDebug) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type TsInfoKey Completed

func (c TsInfoKey) Debug(debug string) TsInfoDebug {
	c.cs.s = append(c.cs.s, debug)
	return (TsInfoDebug)(c)
}

func (c TsInfoKey) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type TsMadd Completed

func (b Builder) TsMadd() (c TsMadd) {
	c = TsMadd{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "TS.MADD")
	return c
}

func (c TsMadd) KeyTimestampValue() TsMaddKeyTimestampValue {
	return (TsMaddKeyTimestampValue)(c)
}

type TsMaddKeyTimestampValue Completed

func (c TsMaddKeyTimestampValue) KeyTimestampValue(key string, timestamp int64, value float64) TsMaddKeyTimestampValue {
	if c.ks&NoSlot == NoSlot {
		c.ks = NoSlot | slot(key)
	} else {
		c.ks = check(c.ks, slot(key))
	}
	c.cs.s = append(c.cs.s, key, strconv.FormatInt(timestamp, 10), strconv.FormatFloat(value, 'f', -1, 64))
	return c
}

func (c TsMaddKeyTimestampValue) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type TsMget Completed

func (b Builder) TsMget() (c TsMget) {
	c = TsMget{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "TS.MGET")
	return c
}

func (c TsMget) Latest() TsMgetLatest {
	c.cs.s = append(c.cs.s, "LATEST")
	return (TsMgetLatest)(c)
}

func (c TsMget) Withlabels() TsMgetWithlabels {
	c.cs.s = append(c.cs.s, "WITHLABELS")
	return (TsMgetWithlabels)(c)
}

func (c TsMget) SelectedLabels(labels []string) TsMgetSelectedLabels {
	c.cs.s = append(c.cs.s, "SELECTED_LABELS")
	c.cs.s = append(c.cs.s, labels...)
	return (TsMgetSelectedLabels)(c)
}

func (c TsMget) Filter(filter ...string) TsMgetFilter {
	c.cs.s = append(c.cs.s, "FILTER")
	c.cs.s = append(c.cs.s, filter...)
	return (TsMgetFilter)(c)
}

type TsMgetFilter Completed

func (c TsMgetFilter) Filter(filter ...string) TsMgetFilter {
	c.cs.s = append(c.cs.s, "FILTER")
	c.cs.s = append(c.cs.s, filter...)
	return c
}

func (c TsMgetFilter) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type TsMgetLatest Completed

func (c TsMgetLatest) Withlabels() TsMgetWithlabels {
	c.cs.s = append(c.cs.s, "WITHLABELS")
	return (TsMgetWithlabels)(c)
}

func (c TsMgetLatest) SelectedLabels(labels []string) TsMgetSelectedLabels {
	c.cs.s = append(c.cs.s, "SELECTED_LABELS")
	c.cs.s = append(c.cs.s, labels...)
	return (TsMgetSelectedLabels)(c)
}

func (c TsMgetLatest) Filter(filter ...string) TsMgetFilter {
	c.cs.s = append(c.cs.s, "FILTER")
	c.cs.s = append(c.cs.s, filter...)
	return (TsMgetFilter)(c)
}

type TsMgetSelectedLabels Completed

func (c TsMgetSelectedLabels) Filter(filter ...string) TsMgetFilter {
	c.cs.s = append(c.cs.s, "FILTER")
	c.cs.s = append(c.cs.s, filter...)
	return (TsMgetFilter)(c)
}

type TsMgetWithlabels Completed

func (c TsMgetWithlabels) Filter(filter ...string) TsMgetFilter {
	c.cs.s = append(c.cs.s, "FILTER")
	c.cs.s = append(c.cs.s, filter...)
	return (TsMgetFilter)(c)
}

type TsMrange Completed

func (b Builder) TsMrange() (c TsMrange) {
	c = TsMrange{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "TS.MRANGE")
	return c
}

func (c TsMrange) Fromtimestamp(fromtimestamp int64) TsMrangeFromtimestamp {
	c.cs.s = append(c.cs.s, strconv.FormatInt(fromtimestamp, 10))
	return (TsMrangeFromtimestamp)(c)
}

type TsMrangeAggregationAggregationAvg Completed

func (c TsMrangeAggregationAggregationAvg) Bucketduration(bucketduration int64) TsMrangeAggregationBucketduration {
	c.cs.s = append(c.cs.s, strconv.FormatInt(bucketduration, 10))
	return (TsMrangeAggregationBucketduration)(c)
}

type TsMrangeAggregationAggregationCount Completed

func (c TsMrangeAggregationAggregationCount) Bucketduration(bucketduration int64) TsMrangeAggregationBucketduration {
	c.cs.s = append(c.cs.s, strconv.FormatInt(bucketduration, 10))
	return (TsMrangeAggregationBucketduration)(c)
}

type TsMrangeAggregationAggregationFirst Completed

func (c TsMrangeAggregationAggregationFirst) Bucketduration(bucketduration int64) TsMrangeAggregationBucketduration {
	c.cs.s = append(c.cs.s, strconv.FormatInt(bucketduration, 10))
	return (TsMrangeAggregationBucketduration)(c)
}

type TsMrangeAggregationAggregationLast Completed

func (c TsMrangeAggregationAggregationLast) Bucketduration(bucketduration int64) TsMrangeAggregationBucketduration {
	c.cs.s = append(c.cs.s, strconv.FormatInt(bucketduration, 10))
	return (TsMrangeAggregationBucketduration)(c)
}

type TsMrangeAggregationAggregationMax Completed

func (c TsMrangeAggregationAggregationMax) Bucketduration(bucketduration int64) TsMrangeAggregationBucketduration {
	c.cs.s = append(c.cs.s, strconv.FormatInt(bucketduration, 10))
	return (TsMrangeAggregationBucketduration)(c)
}

type TsMrangeAggregationAggregationMin Completed

func (c TsMrangeAggregationAggregationMin) Bucketduration(bucketduration int64) TsMrangeAggregationBucketduration {
	c.cs.s = append(c.cs.s, strconv.FormatInt(bucketduration, 10))
	return (TsMrangeAggregationBucketduration)(c)
}

type TsMrangeAggregationAggregationRange Completed

func (c TsMrangeAggregationAggregationRange) Bucketduration(bucketduration int64) TsMrangeAggregationBucketduration {
	c.cs.s = append(c.cs.s, strconv.FormatInt(bucketduration, 10))
	return (TsMrangeAggregationBucketduration)(c)
}

type TsMrangeAggregationAggregationStdP Completed

func (c TsMrangeAggregationAggregationStdP) Bucketduration(bucketduration int64) TsMrangeAggregationBucketduration {
	c.cs.s = append(c.cs.s, strconv.FormatInt(bucketduration, 10))
	return (TsMrangeAggregationBucketduration)(c)
}

type TsMrangeAggregationAggregationStdS Completed

func (c TsMrangeAggregationAggregationStdS) Bucketduration(bucketduration int64) TsMrangeAggregationBucketduration {
	c.cs.s = append(c.cs.s, strconv.FormatInt(bucketduration, 10))
	return (TsMrangeAggregationBucketduration)(c)
}

type TsMrangeAggregationAggregationSum Completed

func (c TsMrangeAggregationAggregationSum) Bucketduration(bucketduration int64) TsMrangeAggregationBucketduration {
	c.cs.s = append(c.cs.s, strconv.FormatInt(bucketduration, 10))
	return (TsMrangeAggregationBucketduration)(c)
}

type TsMrangeAggregationAggregationTwa Completed

func (c TsMrangeAggregationAggregationTwa) Bucketduration(bucketduration int64) TsMrangeAggregationBucketduration {
	c.cs.s = append(c.cs.s, strconv.FormatInt(bucketduration, 10))
	return (TsMrangeAggregationBucketduration)(c)
}

type TsMrangeAggregationAggregationVarP Completed

func (c TsMrangeAggregationAggregationVarP) Bucketduration(bucketduration int64) TsMrangeAggregationBucketduration {
	c.cs.s = append(c.cs.s, strconv.FormatInt(bucketduration, 10))
	return (TsMrangeAggregationBucketduration)(c)
}

type TsMrangeAggregationAggregationVarS Completed

func (c TsMrangeAggregationAggregationVarS) Bucketduration(bucketduration int64) TsMrangeAggregationBucketduration {
	c.cs.s = append(c.cs.s, strconv.FormatInt(bucketduration, 10))
	return (TsMrangeAggregationBucketduration)(c)
}

type TsMrangeAggregationBucketduration Completed

func (c TsMrangeAggregationBucketduration) Buckettimestamp(buckettimestamp string) TsMrangeAggregationBuckettimestamp {
	c.cs.s = append(c.cs.s, "BUCKETTIMESTAMP", buckettimestamp)
	return (TsMrangeAggregationBuckettimestamp)(c)
}

func (c TsMrangeAggregationBucketduration) Empty() TsMrangeAggregationEmpty {
	c.cs.s = append(c.cs.s, "EMPTY")
	return (TsMrangeAggregationEmpty)(c)
}

func (c TsMrangeAggregationBucketduration) Filter(filter ...string) TsMrangeFilter {
	c.cs.s = append(c.cs.s, "FILTER")
	c.cs.s = append(c.cs.s, filter...)
	return (TsMrangeFilter)(c)
}

type TsMrangeAggregationBuckettimestamp Completed

func (c TsMrangeAggregationBuckettimestamp) Empty() TsMrangeAggregationEmpty {
	c.cs.s = append(c.cs.s, "EMPTY")
	return (TsMrangeAggregationEmpty)(c)
}

func (c TsMrangeAggregationBuckettimestamp) Filter(filter ...string) TsMrangeFilter {
	c.cs.s = append(c.cs.s, "FILTER")
	c.cs.s = append(c.cs.s, filter...)
	return (TsMrangeFilter)(c)
}

type TsMrangeAggregationEmpty Completed

func (c TsMrangeAggregationEmpty) Filter(filter ...string) TsMrangeFilter {
	c.cs.s = append(c.cs.s, "FILTER")
	c.cs.s = append(c.cs.s, filter...)
	return (TsMrangeFilter)(c)
}

type TsMrangeAlign Completed

func (c TsMrangeAlign) AggregationAvg() TsMrangeAggregationAggregationAvg {
	c.cs.s = append(c.cs.s, "AGGREGATION", "AVG")
	return (TsMrangeAggregationAggregationAvg)(c)
}

func (c TsMrangeAlign) AggregationSum() TsMrangeAggregationAggregationSum {
	c.cs.s = append(c.cs.s, "AGGREGATION", "SUM")
	return (TsMrangeAggregationAggregationSum)(c)
}

func (c TsMrangeAlign) AggregationMin() TsMrangeAggregationAggregationMin {
	c.cs.s = append(c.cs.s, "AGGREGATION", "MIN")
	return (TsMrangeAggregationAggregationMin)(c)
}

func (c TsMrangeAlign) AggregationMax() TsMrangeAggregationAggregationMax {
	c.cs.s = append(c.cs.s, "AGGREGATION", "MAX")
	return (TsMrangeAggregationAggregationMax)(c)
}

func (c TsMrangeAlign) AggregationRange() TsMrangeAggregationAggregationRange {
	c.cs.s = append(c.cs.s, "AGGREGATION", "RANGE")
	return (TsMrangeAggregationAggregationRange)(c)
}

func (c TsMrangeAlign) AggregationCount() TsMrangeAggregationAggregationCount {
	c.cs.s = append(c.cs.s, "AGGREGATION", "COUNT")
	return (TsMrangeAggregationAggregationCount)(c)
}

func (c TsMrangeAlign) AggregationFirst() TsMrangeAggregationAggregationFirst {
	c.cs.s = append(c.cs.s, "AGGREGATION", "FIRST")
	return (TsMrangeAggregationAggregationFirst)(c)
}

func (c TsMrangeAlign) AggregationLast() TsMrangeAggregationAggregationLast {
	c.cs.s = append(c.cs.s, "AGGREGATION", "LAST")
	return (TsMrangeAggregationAggregationLast)(c)
}

func (c TsMrangeAlign) AggregationStdP() TsMrangeAggregationAggregationStdP {
	c.cs.s = append(c.cs.s, "AGGREGATION", "STD.P")
	return (TsMrangeAggregationAggregationStdP)(c)
}

func (c TsMrangeAlign) AggregationStdS() TsMrangeAggregationAggregationStdS {
	c.cs.s = append(c.cs.s, "AGGREGATION", "STD.S")
	return (TsMrangeAggregationAggregationStdS)(c)
}

func (c TsMrangeAlign) AggregationVarP() TsMrangeAggregationAggregationVarP {
	c.cs.s = append(c.cs.s, "AGGREGATION", "VAR.P")
	return (TsMrangeAggregationAggregationVarP)(c)
}

func (c TsMrangeAlign) AggregationVarS() TsMrangeAggregationAggregationVarS {
	c.cs.s = append(c.cs.s, "AGGREGATION", "VAR.S")
	return (TsMrangeAggregationAggregationVarS)(c)
}

func (c TsMrangeAlign) AggregationTwa() TsMrangeAggregationAggregationTwa {
	c.cs.s = append(c.cs.s, "AGGREGATION", "TWA")
	return (TsMrangeAggregationAggregationTwa)(c)
}

func (c TsMrangeAlign) Filter(filter ...string) TsMrangeFilter {
	c.cs.s = append(c.cs.s, "FILTER")
	c.cs.s = append(c.cs.s, filter...)
	return (TsMrangeFilter)(c)
}

type TsMrangeCount Completed

func (c TsMrangeCount) Align(value string) TsMrangeAlign {
	c.cs.s = append(c.cs.s, "ALIGN", value)
	return (TsMrangeAlign)(c)
}

func (c TsMrangeCount) AggregationAvg() TsMrangeAggregationAggregationAvg {
	c.cs.s = append(c.cs.s, "AGGREGATION", "AVG")
	return (TsMrangeAggregationAggregationAvg)(c)
}

func (c TsMrangeCount) AggregationSum() TsMrangeAggregationAggregationSum {
	c.cs.s = append(c.cs.s, "AGGREGATION", "SUM")
	return (TsMrangeAggregationAggregationSum)(c)
}

func (c TsMrangeCount) AggregationMin() TsMrangeAggregationAggregationMin {
	c.cs.s = append(c.cs.s, "AGGREGATION", "MIN")
	return (TsMrangeAggregationAggregationMin)(c)
}

func (c TsMrangeCount) AggregationMax() TsMrangeAggregationAggregationMax {
	c.cs.s = append(c.cs.s, "AGGREGATION", "MAX")
	return (TsMrangeAggregationAggregationMax)(c)
}

func (c TsMrangeCount) AggregationRange() TsMrangeAggregationAggregationRange {
	c.cs.s = append(c.cs.s, "AGGREGATION", "RANGE")
	return (TsMrangeAggregationAggregationRange)(c)
}

func (c TsMrangeCount) AggregationCount() TsMrangeAggregationAggregationCount {
	c.cs.s = append(c.cs.s, "AGGREGATION", "COUNT")
	return (TsMrangeAggregationAggregationCount)(c)
}

func (c TsMrangeCount) AggregationFirst() TsMrangeAggregationAggregationFirst {
	c.cs.s = append(c.cs.s, "AGGREGATION", "FIRST")
	return (TsMrangeAggregationAggregationFirst)(c)
}

func (c TsMrangeCount) AggregationLast() TsMrangeAggregationAggregationLast {
	c.cs.s = append(c.cs.s, "AGGREGATION", "LAST")
	return (TsMrangeAggregationAggregationLast)(c)
}

func (c TsMrangeCount) AggregationStdP() TsMrangeAggregationAggregationStdP {
	c.cs.s = append(c.cs.s, "AGGREGATION", "STD.P")
	return (TsMrangeAggregationAggregationStdP)(c)
}

func (c TsMrangeCount) AggregationStdS() TsMrangeAggregationAggregationStdS {
	c.cs.s = append(c.cs.s, "AGGREGATION", "STD.S")
	return (TsMrangeAggregationAggregationStdS)(c)
}

func (c TsMrangeCount) AggregationVarP() TsMrangeAggregationAggregationVarP {
	c.cs.s = append(c.cs.s, "AGGREGATION", "VAR.P")
	return (TsMrangeAggregationAggregationVarP)(c)
}

func (c TsMrangeCount) AggregationVarS() TsMrangeAggregationAggregationVarS {
	c.cs.s = append(c.cs.s, "AGGREGATION", "VAR.S")
	return (TsMrangeAggregationAggregationVarS)(c)
}

func (c TsMrangeCount) AggregationTwa() TsMrangeAggregationAggregationTwa {
	c.cs.s = append(c.cs.s, "AGGREGATION", "TWA")
	return (TsMrangeAggregationAggregationTwa)(c)
}

func (c TsMrangeCount) Filter(filter ...string) TsMrangeFilter {
	c.cs.s = append(c.cs.s, "FILTER")
	c.cs.s = append(c.cs.s, filter...)
	return (TsMrangeFilter)(c)
}

type TsMrangeFilter Completed

func (c TsMrangeFilter) Filter(filter ...string) TsMrangeFilter {
	c.cs.s = append(c.cs.s, "FILTER")
	c.cs.s = append(c.cs.s, filter...)
	return c
}

func (c TsMrangeFilter) Groupby(label string, reduce string, reducer string) TsMrangeGroupby {
	c.cs.s = append(c.cs.s, "GROUPBY", label, reduce, reducer)
	return (TsMrangeGroupby)(c)
}

func (c TsMrangeFilter) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type TsMrangeFilterByTs Completed

func (c TsMrangeFilterByTs) FilterByTs(timestamp ...int64) TsMrangeFilterByTs {
	c.cs.s = append(c.cs.s, "FILTER_BY_TS")
	for _, n := range timestamp {
		c.cs.s = append(c.cs.s, strconv.FormatInt(n, 10))
	}
	return c
}

func (c TsMrangeFilterByTs) FilterByValue(min float64, max float64) TsMrangeFilterByValue {
	c.cs.s = append(c.cs.s, "FILTER_BY_VALUE", strconv.FormatFloat(min, 'f', -1, 64), strconv.FormatFloat(max, 'f', -1, 64))
	return (TsMrangeFilterByValue)(c)
}

func (c TsMrangeFilterByTs) Withlabels() TsMrangeWithlabels {
	c.cs.s = append(c.cs.s, "WITHLABELS")
	return (TsMrangeWithlabels)(c)
}

func (c TsMrangeFilterByTs) SelectedLabels(labels []string) TsMrangeSelectedLabels {
	c.cs.s = append(c.cs.s, "SELECTED_LABELS")
	c.cs.s = append(c.cs.s, labels...)
	return (TsMrangeSelectedLabels)(c)
}

func (c TsMrangeFilterByTs) Count(count int64) TsMrangeCount {
	c.cs.s = append(c.cs.s, "COUNT", strconv.FormatInt(count, 10))
	return (TsMrangeCount)(c)
}

func (c TsMrangeFilterByTs) Align(value string) TsMrangeAlign {
	c.cs.s = append(c.cs.s, "ALIGN", value)
	return (TsMrangeAlign)(c)
}

func (c TsMrangeFilterByTs) AggregationAvg() TsMrangeAggregationAggregationAvg {
	c.cs.s = append(c.cs.s, "AGGREGATION", "AVG")
	return (TsMrangeAggregationAggregationAvg)(c)
}

func (c TsMrangeFilterByTs) AggregationSum() TsMrangeAggregationAggregationSum {
	c.cs.s = append(c.cs.s, "AGGREGATION", "SUM")
	return (TsMrangeAggregationAggregationSum)(c)
}

func (c TsMrangeFilterByTs) AggregationMin() TsMrangeAggregationAggregationMin {
	c.cs.s = append(c.cs.s, "AGGREGATION", "MIN")
	return (TsMrangeAggregationAggregationMin)(c)
}

func (c TsMrangeFilterByTs) AggregationMax() TsMrangeAggregationAggregationMax {
	c.cs.s = append(c.cs.s, "AGGREGATION", "MAX")
	return (TsMrangeAggregationAggregationMax)(c)
}

func (c TsMrangeFilterByTs) AggregationRange() TsMrangeAggregationAggregationRange {
	c.cs.s = append(c.cs.s, "AGGREGATION", "RANGE")
	return (TsMrangeAggregationAggregationRange)(c)
}

func (c TsMrangeFilterByTs) AggregationCount() TsMrangeAggregationAggregationCount {
	c.cs.s = append(c.cs.s, "AGGREGATION", "COUNT")
	return (TsMrangeAggregationAggregationCount)(c)
}

func (c TsMrangeFilterByTs) AggregationFirst() TsMrangeAggregationAggregationFirst {
	c.cs.s = append(c.cs.s, "AGGREGATION", "FIRST")
	return (TsMrangeAggregationAggregationFirst)(c)
}

func (c TsMrangeFilterByTs) AggregationLast() TsMrangeAggregationAggregationLast {
	c.cs.s = append(c.cs.s, "AGGREGATION", "LAST")
	return (TsMrangeAggregationAggregationLast)(c)
}

func (c TsMrangeFilterByTs) AggregationStdP() TsMrangeAggregationAggregationStdP {
	c.cs.s = append(c.cs.s, "AGGREGATION", "STD.P")
	return (TsMrangeAggregationAggregationStdP)(c)
}

func (c TsMrangeFilterByTs) AggregationStdS() TsMrangeAggregationAggregationStdS {
	c.cs.s = append(c.cs.s, "AGGREGATION", "STD.S")
	return (TsMrangeAggregationAggregationStdS)(c)
}

func (c TsMrangeFilterByTs) AggregationVarP() TsMrangeAggregationAggregationVarP {
	c.cs.s = append(c.cs.s, "AGGREGATION", "VAR.P")
	return (TsMrangeAggregationAggregationVarP)(c)
}

func (c TsMrangeFilterByTs) AggregationVarS() TsMrangeAggregationAggregationVarS {
	c.cs.s = append(c.cs.s, "AGGREGATION", "VAR.S")
	return (TsMrangeAggregationAggregationVarS)(c)
}

func (c TsMrangeFilterByTs) AggregationTwa() TsMrangeAggregationAggregationTwa {
	c.cs.s = append(c.cs.s, "AGGREGATION", "TWA")
	return (TsMrangeAggregationAggregationTwa)(c)
}

func (c TsMrangeFilterByTs) Filter(filter ...string) TsMrangeFilter {
	c.cs.s = append(c.cs.s, "FILTER")
	c.cs.s = append(c.cs.s, filter...)
	return (TsMrangeFilter)(c)
}

type TsMrangeFilterByValue Completed

func (c TsMrangeFilterByValue) Withlabels() TsMrangeWithlabels {
	c.cs.s = append(c.cs.s, "WITHLABELS")
	return (TsMrangeWithlabels)(c)
}

func (c TsMrangeFilterByValue) SelectedLabels(labels []string) TsMrangeSelectedLabels {
	c.cs.s = append(c.cs.s, "SELECTED_LABELS")
	c.cs.s = append(c.cs.s, labels...)
	return (TsMrangeSelectedLabels)(c)
}

func (c TsMrangeFilterByValue) Count(count int64) TsMrangeCount {
	c.cs.s = append(c.cs.s, "COUNT", strconv.FormatInt(count, 10))
	return (TsMrangeCount)(c)
}

func (c TsMrangeFilterByValue) Align(value string) TsMrangeAlign {
	c.cs.s = append(c.cs.s, "ALIGN", value)
	return (TsMrangeAlign)(c)
}

func (c TsMrangeFilterByValue) AggregationAvg() TsMrangeAggregationAggregationAvg {
	c.cs.s = append(c.cs.s, "AGGREGATION", "AVG")
	return (TsMrangeAggregationAggregationAvg)(c)
}

func (c TsMrangeFilterByValue) AggregationSum() TsMrangeAggregationAggregationSum {
	c.cs.s = append(c.cs.s, "AGGREGATION", "SUM")
	return (TsMrangeAggregationAggregationSum)(c)
}

func (c TsMrangeFilterByValue) AggregationMin() TsMrangeAggregationAggregationMin {
	c.cs.s = append(c.cs.s, "AGGREGATION", "MIN")
	return (TsMrangeAggregationAggregationMin)(c)
}

func (c TsMrangeFilterByValue) AggregationMax() TsMrangeAggregationAggregationMax {
	c.cs.s = append(c.cs.s, "AGGREGATION", "MAX")
	return (TsMrangeAggregationAggregationMax)(c)
}

func (c TsMrangeFilterByValue) AggregationRange() TsMrangeAggregationAggregationRange {
	c.cs.s = append(c.cs.s, "AGGREGATION", "RANGE")
	return (TsMrangeAggregationAggregationRange)(c)
}

func (c TsMrangeFilterByValue) AggregationCount() TsMrangeAggregationAggregationCount {
	c.cs.s = append(c.cs.s, "AGGREGATION", "COUNT")
	return (TsMrangeAggregationAggregationCount)(c)
}

func (c TsMrangeFilterByValue) AggregationFirst() TsMrangeAggregationAggregationFirst {
	c.cs.s = append(c.cs.s, "AGGREGATION", "FIRST")
	return (TsMrangeAggregationAggregationFirst)(c)
}

func (c TsMrangeFilterByValue) AggregationLast() TsMrangeAggregationAggregationLast {
	c.cs.s = append(c.cs.s, "AGGREGATION", "LAST")
	return (TsMrangeAggregationAggregationLast)(c)
}

func (c TsMrangeFilterByValue) AggregationStdP() TsMrangeAggregationAggregationStdP {
	c.cs.s = append(c.cs.s, "AGGREGATION", "STD.P")
	return (TsMrangeAggregationAggregationStdP)(c)
}

func (c TsMrangeFilterByValue) AggregationStdS() TsMrangeAggregationAggregationStdS {
	c.cs.s = append(c.cs.s, "AGGREGATION", "STD.S")
	return (TsMrangeAggregationAggregationStdS)(c)
}

func (c TsMrangeFilterByValue) AggregationVarP() TsMrangeAggregationAggregationVarP {
	c.cs.s = append(c.cs.s, "AGGREGATION", "VAR.P")
	return (TsMrangeAggregationAggregationVarP)(c)
}

func (c TsMrangeFilterByValue) AggregationVarS() TsMrangeAggregationAggregationVarS {
	c.cs.s = append(c.cs.s, "AGGREGATION", "VAR.S")
	return (TsMrangeAggregationAggregationVarS)(c)
}

func (c TsMrangeFilterByValue) AggregationTwa() TsMrangeAggregationAggregationTwa {
	c.cs.s = append(c.cs.s, "AGGREGATION", "TWA")
	return (TsMrangeAggregationAggregationTwa)(c)
}

func (c TsMrangeFilterByValue) Filter(filter ...string) TsMrangeFilter {
	c.cs.s = append(c.cs.s, "FILTER")
	c.cs.s = append(c.cs.s, filter...)
	return (TsMrangeFilter)(c)
}

type TsMrangeFromtimestamp Completed

func (c TsMrangeFromtimestamp) Totimestamp(totimestamp int64) TsMrangeTotimestamp {
	c.cs.s = append(c.cs.s, strconv.FormatInt(totimestamp, 10))
	return (TsMrangeTotimestamp)(c)
}

type TsMrangeGroupby Completed

func (c TsMrangeGroupby) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type TsMrangeLatest Completed

func (c TsMrangeLatest) FilterByTs(timestamp ...int64) TsMrangeFilterByTs {
	c.cs.s = append(c.cs.s, "FILTER_BY_TS")
	for _, n := range timestamp {
		c.cs.s = append(c.cs.s, strconv.FormatInt(n, 10))
	}
	return (TsMrangeFilterByTs)(c)
}

func (c TsMrangeLatest) FilterByValue(min float64, max float64) TsMrangeFilterByValue {
	c.cs.s = append(c.cs.s, "FILTER_BY_VALUE", strconv.FormatFloat(min, 'f', -1, 64), strconv.FormatFloat(max, 'f', -1, 64))
	return (TsMrangeFilterByValue)(c)
}

func (c TsMrangeLatest) Withlabels() TsMrangeWithlabels {
	c.cs.s = append(c.cs.s, "WITHLABELS")
	return (TsMrangeWithlabels)(c)
}

func (c TsMrangeLatest) SelectedLabels(labels []string) TsMrangeSelectedLabels {
	c.cs.s = append(c.cs.s, "SELECTED_LABELS")
	c.cs.s = append(c.cs.s, labels...)
	return (TsMrangeSelectedLabels)(c)
}

func (c TsMrangeLatest) Count(count int64) TsMrangeCount {
	c.cs.s = append(c.cs.s, "COUNT", strconv.FormatInt(count, 10))
	return (TsMrangeCount)(c)
}

func (c TsMrangeLatest) Align(value string) TsMrangeAlign {
	c.cs.s = append(c.cs.s, "ALIGN", value)
	return (TsMrangeAlign)(c)
}

func (c TsMrangeLatest) AggregationAvg() TsMrangeAggregationAggregationAvg {
	c.cs.s = append(c.cs.s, "AGGREGATION", "AVG")
	return (TsMrangeAggregationAggregationAvg)(c)
}

func (c TsMrangeLatest) AggregationSum() TsMrangeAggregationAggregationSum {
	c.cs.s = append(c.cs.s, "AGGREGATION", "SUM")
	return (TsMrangeAggregationAggregationSum)(c)
}

func (c TsMrangeLatest) AggregationMin() TsMrangeAggregationAggregationMin {
	c.cs.s = append(c.cs.s, "AGGREGATION", "MIN")
	return (TsMrangeAggregationAggregationMin)(c)
}

func (c TsMrangeLatest) AggregationMax() TsMrangeAggregationAggregationMax {
	c.cs.s = append(c.cs.s, "AGGREGATION", "MAX")
	return (TsMrangeAggregationAggregationMax)(c)
}

func (c TsMrangeLatest) AggregationRange() TsMrangeAggregationAggregationRange {
	c.cs.s = append(c.cs.s, "AGGREGATION", "RANGE")
	return (TsMrangeAggregationAggregationRange)(c)
}

func (c TsMrangeLatest) AggregationCount() TsMrangeAggregationAggregationCount {
	c.cs.s = append(c.cs.s, "AGGREGATION", "COUNT")
	return (TsMrangeAggregationAggregationCount)(c)
}

func (c TsMrangeLatest) AggregationFirst() TsMrangeAggregationAggregationFirst {
	c.cs.s = append(c.cs.s, "AGGREGATION", "FIRST")
	return (TsMrangeAggregationAggregationFirst)(c)
}

func (c TsMrangeLatest) AggregationLast() TsMrangeAggregationAggregationLast {
	c.cs.s = append(c.cs.s, "AGGREGATION", "LAST")
	return (TsMrangeAggregationAggregationLast)(c)
}

func (c TsMrangeLatest) AggregationStdP() TsMrangeAggregationAggregationStdP {
	c.cs.s = append(c.cs.s, "AGGREGATION", "STD.P")
	return (TsMrangeAggregationAggregationStdP)(c)
}

func (c TsMrangeLatest) AggregationStdS() TsMrangeAggregationAggregationStdS {
	c.cs.s = append(c.cs.s, "AGGREGATION", "STD.S")
	return (TsMrangeAggregationAggregationStdS)(c)
}

func (c TsMrangeLatest) AggregationVarP() TsMrangeAggregationAggregationVarP {
	c.cs.s = append(c.cs.s, "AGGREGATION", "VAR.P")
	return (TsMrangeAggregationAggregationVarP)(c)
}

func (c TsMrangeLatest) AggregationVarS() TsMrangeAggregationAggregationVarS {
	c.cs.s = append(c.cs.s, "AGGREGATION", "VAR.S")
	return (TsMrangeAggregationAggregationVarS)(c)
}

func (c TsMrangeLatest) AggregationTwa() TsMrangeAggregationAggregationTwa {
	c.cs.s = append(c.cs.s, "AGGREGATION", "TWA")
	return (TsMrangeAggregationAggregationTwa)(c)
}

func (c TsMrangeLatest) Filter(filter ...string) TsMrangeFilter {
	c.cs.s = append(c.cs.s, "FILTER")
	c.cs.s = append(c.cs.s, filter...)
	return (TsMrangeFilter)(c)
}

type TsMrangeSelectedLabels Completed

func (c TsMrangeSelectedLabels) Count(count int64) TsMrangeCount {
	c.cs.s = append(c.cs.s, "COUNT", strconv.FormatInt(count, 10))
	return (TsMrangeCount)(c)
}

func (c TsMrangeSelectedLabels) Align(value string) TsMrangeAlign {
	c.cs.s = append(c.cs.s, "ALIGN", value)
	return (TsMrangeAlign)(c)
}

func (c TsMrangeSelectedLabels) AggregationAvg() TsMrangeAggregationAggregationAvg {
	c.cs.s = append(c.cs.s, "AGGREGATION", "AVG")
	return (TsMrangeAggregationAggregationAvg)(c)
}

func (c TsMrangeSelectedLabels) AggregationSum() TsMrangeAggregationAggregationSum {
	c.cs.s = append(c.cs.s, "AGGREGATION", "SUM")
	return (TsMrangeAggregationAggregationSum)(c)
}

func (c TsMrangeSelectedLabels) AggregationMin() TsMrangeAggregationAggregationMin {
	c.cs.s = append(c.cs.s, "AGGREGATION", "MIN")
	return (TsMrangeAggregationAggregationMin)(c)
}

func (c TsMrangeSelectedLabels) AggregationMax() TsMrangeAggregationAggregationMax {
	c.cs.s = append(c.cs.s, "AGGREGATION", "MAX")
	return (TsMrangeAggregationAggregationMax)(c)
}

func (c TsMrangeSelectedLabels) AggregationRange() TsMrangeAggregationAggregationRange {
	c.cs.s = append(c.cs.s, "AGGREGATION", "RANGE")
	return (TsMrangeAggregationAggregationRange)(c)
}

func (c TsMrangeSelectedLabels) AggregationCount() TsMrangeAggregationAggregationCount {
	c.cs.s = append(c.cs.s, "AGGREGATION", "COUNT")
	return (TsMrangeAggregationAggregationCount)(c)
}

func (c TsMrangeSelectedLabels) AggregationFirst() TsMrangeAggregationAggregationFirst {
	c.cs.s = append(c.cs.s, "AGGREGATION", "FIRST")
	return (TsMrangeAggregationAggregationFirst)(c)
}

func (c TsMrangeSelectedLabels) AggregationLast() TsMrangeAggregationAggregationLast {
	c.cs.s = append(c.cs.s, "AGGREGATION", "LAST")
	return (TsMrangeAggregationAggregationLast)(c)
}

func (c TsMrangeSelectedLabels) AggregationStdP() TsMrangeAggregationAggregationStdP {
	c.cs.s = append(c.cs.s, "AGGREGATION", "STD.P")
	return (TsMrangeAggregationAggregationStdP)(c)
}

func (c TsMrangeSelectedLabels) AggregationStdS() TsMrangeAggregationAggregationStdS {
	c.cs.s = append(c.cs.s, "AGGREGATION", "STD.S")
	return (TsMrangeAggregationAggregationStdS)(c)
}

func (c TsMrangeSelectedLabels) AggregationVarP() TsMrangeAggregationAggregationVarP {
	c.cs.s = append(c.cs.s, "AGGREGATION", "VAR.P")
	return (TsMrangeAggregationAggregationVarP)(c)
}

func (c TsMrangeSelectedLabels) AggregationVarS() TsMrangeAggregationAggregationVarS {
	c.cs.s = append(c.cs.s, "AGGREGATION", "VAR.S")
	return (TsMrangeAggregationAggregationVarS)(c)
}

func (c TsMrangeSelectedLabels) AggregationTwa() TsMrangeAggregationAggregationTwa {
	c.cs.s = append(c.cs.s, "AGGREGATION", "TWA")
	return (TsMrangeAggregationAggregationTwa)(c)
}

func (c TsMrangeSelectedLabels) Filter(filter ...string) TsMrangeFilter {
	c.cs.s = append(c.cs.s, "FILTER")
	c.cs.s = append(c.cs.s, filter...)
	return (TsMrangeFilter)(c)
}

type TsMrangeTotimestamp Completed

func (c TsMrangeTotimestamp) Latest() TsMrangeLatest {
	c.cs.s = append(c.cs.s, "LATEST")
	return (TsMrangeLatest)(c)
}

func (c TsMrangeTotimestamp) FilterByTs(timestamp ...int64) TsMrangeFilterByTs {
	c.cs.s = append(c.cs.s, "FILTER_BY_TS")
	for _, n := range timestamp {
		c.cs.s = append(c.cs.s, strconv.FormatInt(n, 10))
	}
	return (TsMrangeFilterByTs)(c)
}

func (c TsMrangeTotimestamp) FilterByValue(min float64, max float64) TsMrangeFilterByValue {
	c.cs.s = append(c.cs.s, "FILTER_BY_VALUE", strconv.FormatFloat(min, 'f', -1, 64), strconv.FormatFloat(max, 'f', -1, 64))
	return (TsMrangeFilterByValue)(c)
}

func (c TsMrangeTotimestamp) Withlabels() TsMrangeWithlabels {
	c.cs.s = append(c.cs.s, "WITHLABELS")
	return (TsMrangeWithlabels)(c)
}

func (c TsMrangeTotimestamp) SelectedLabels(labels []string) TsMrangeSelectedLabels {
	c.cs.s = append(c.cs.s, "SELECTED_LABELS")
	c.cs.s = append(c.cs.s, labels...)
	return (TsMrangeSelectedLabels)(c)
}

func (c TsMrangeTotimestamp) Count(count int64) TsMrangeCount {
	c.cs.s = append(c.cs.s, "COUNT", strconv.FormatInt(count, 10))
	return (TsMrangeCount)(c)
}

func (c TsMrangeTotimestamp) Align(value string) TsMrangeAlign {
	c.cs.s = append(c.cs.s, "ALIGN", value)
	return (TsMrangeAlign)(c)
}

func (c TsMrangeTotimestamp) AggregationAvg() TsMrangeAggregationAggregationAvg {
	c.cs.s = append(c.cs.s, "AGGREGATION", "AVG")
	return (TsMrangeAggregationAggregationAvg)(c)
}

func (c TsMrangeTotimestamp) AggregationSum() TsMrangeAggregationAggregationSum {
	c.cs.s = append(c.cs.s, "AGGREGATION", "SUM")
	return (TsMrangeAggregationAggregationSum)(c)
}

func (c TsMrangeTotimestamp) AggregationMin() TsMrangeAggregationAggregationMin {
	c.cs.s = append(c.cs.s, "AGGREGATION", "MIN")
	return (TsMrangeAggregationAggregationMin)(c)
}

func (c TsMrangeTotimestamp) AggregationMax() TsMrangeAggregationAggregationMax {
	c.cs.s = append(c.cs.s, "AGGREGATION", "MAX")
	return (TsMrangeAggregationAggregationMax)(c)
}

func (c TsMrangeTotimestamp) AggregationRange() TsMrangeAggregationAggregationRange {
	c.cs.s = append(c.cs.s, "AGGREGATION", "RANGE")
	return (TsMrangeAggregationAggregationRange)(c)
}

func (c TsMrangeTotimestamp) AggregationCount() TsMrangeAggregationAggregationCount {
	c.cs.s = append(c.cs.s, "AGGREGATION", "COUNT")
	return (TsMrangeAggregationAggregationCount)(c)
}

func (c TsMrangeTotimestamp) AggregationFirst() TsMrangeAggregationAggregationFirst {
	c.cs.s = append(c.cs.s, "AGGREGATION", "FIRST")
	return (TsMrangeAggregationAggregationFirst)(c)
}

func (c TsMrangeTotimestamp) AggregationLast() TsMrangeAggregationAggregationLast {
	c.cs.s = append(c.cs.s, "AGGREGATION", "LAST")
	return (TsMrangeAggregationAggregationLast)(c)
}

func (c TsMrangeTotimestamp) AggregationStdP() TsMrangeAggregationAggregationStdP {
	c.cs.s = append(c.cs.s, "AGGREGATION", "STD.P")
	return (TsMrangeAggregationAggregationStdP)(c)
}

func (c TsMrangeTotimestamp) AggregationStdS() TsMrangeAggregationAggregationStdS {
	c.cs.s = append(c.cs.s, "AGGREGATION", "STD.S")
	return (TsMrangeAggregationAggregationStdS)(c)
}

func (c TsMrangeTotimestamp) AggregationVarP() TsMrangeAggregationAggregationVarP {
	c.cs.s = append(c.cs.s, "AGGREGATION", "VAR.P")
	return (TsMrangeAggregationAggregationVarP)(c)
}

func (c TsMrangeTotimestamp) AggregationVarS() TsMrangeAggregationAggregationVarS {
	c.cs.s = append(c.cs.s, "AGGREGATION", "VAR.S")
	return (TsMrangeAggregationAggregationVarS)(c)
}

func (c TsMrangeTotimestamp) AggregationTwa() TsMrangeAggregationAggregationTwa {
	c.cs.s = append(c.cs.s, "AGGREGATION", "TWA")
	return (TsMrangeAggregationAggregationTwa)(c)
}

func (c TsMrangeTotimestamp) Filter(filter ...string) TsMrangeFilter {
	c.cs.s = append(c.cs.s, "FILTER")
	c.cs.s = append(c.cs.s, filter...)
	return (TsMrangeFilter)(c)
}

type TsMrangeWithlabels Completed

func (c TsMrangeWithlabels) Count(count int64) TsMrangeCount {
	c.cs.s = append(c.cs.s, "COUNT", strconv.FormatInt(count, 10))
	return (TsMrangeCount)(c)
}

func (c TsMrangeWithlabels) Align(value string) TsMrangeAlign {
	c.cs.s = append(c.cs.s, "ALIGN", value)
	return (TsMrangeAlign)(c)
}

func (c TsMrangeWithlabels) AggregationAvg() TsMrangeAggregationAggregationAvg {
	c.cs.s = append(c.cs.s, "AGGREGATION", "AVG")
	return (TsMrangeAggregationAggregationAvg)(c)
}

func (c TsMrangeWithlabels) AggregationSum() TsMrangeAggregationAggregationSum {
	c.cs.s = append(c.cs.s, "AGGREGATION", "SUM")
	return (TsMrangeAggregationAggregationSum)(c)
}

func (c TsMrangeWithlabels) AggregationMin() TsMrangeAggregationAggregationMin {
	c.cs.s = append(c.cs.s, "AGGREGATION", "MIN")
	return (TsMrangeAggregationAggregationMin)(c)
}

func (c TsMrangeWithlabels) AggregationMax() TsMrangeAggregationAggregationMax {
	c.cs.s = append(c.cs.s, "AGGREGATION", "MAX")
	return (TsMrangeAggregationAggregationMax)(c)
}

func (c TsMrangeWithlabels) AggregationRange() TsMrangeAggregationAggregationRange {
	c.cs.s = append(c.cs.s, "AGGREGATION", "RANGE")
	return (TsMrangeAggregationAggregationRange)(c)
}

func (c TsMrangeWithlabels) AggregationCount() TsMrangeAggregationAggregationCount {
	c.cs.s = append(c.cs.s, "AGGREGATION", "COUNT")
	return (TsMrangeAggregationAggregationCount)(c)
}

func (c TsMrangeWithlabels) AggregationFirst() TsMrangeAggregationAggregationFirst {
	c.cs.s = append(c.cs.s, "AGGREGATION", "FIRST")
	return (TsMrangeAggregationAggregationFirst)(c)
}

func (c TsMrangeWithlabels) AggregationLast() TsMrangeAggregationAggregationLast {
	c.cs.s = append(c.cs.s, "AGGREGATION", "LAST")
	return (TsMrangeAggregationAggregationLast)(c)
}

func (c TsMrangeWithlabels) AggregationStdP() TsMrangeAggregationAggregationStdP {
	c.cs.s = append(c.cs.s, "AGGREGATION", "STD.P")
	return (TsMrangeAggregationAggregationStdP)(c)
}

func (c TsMrangeWithlabels) AggregationStdS() TsMrangeAggregationAggregationStdS {
	c.cs.s = append(c.cs.s, "AGGREGATION", "STD.S")
	return (TsMrangeAggregationAggregationStdS)(c)
}

func (c TsMrangeWithlabels) AggregationVarP() TsMrangeAggregationAggregationVarP {
	c.cs.s = append(c.cs.s, "AGGREGATION", "VAR.P")
	return (TsMrangeAggregationAggregationVarP)(c)
}

func (c TsMrangeWithlabels) AggregationVarS() TsMrangeAggregationAggregationVarS {
	c.cs.s = append(c.cs.s, "AGGREGATION", "VAR.S")
	return (TsMrangeAggregationAggregationVarS)(c)
}

func (c TsMrangeWithlabels) AggregationTwa() TsMrangeAggregationAggregationTwa {
	c.cs.s = append(c.cs.s, "AGGREGATION", "TWA")
	return (TsMrangeAggregationAggregationTwa)(c)
}

func (c TsMrangeWithlabels) Filter(filter ...string) TsMrangeFilter {
	c.cs.s = append(c.cs.s, "FILTER")
	c.cs.s = append(c.cs.s, filter...)
	return (TsMrangeFilter)(c)
}

type TsMrevrange Completed

func (b Builder) TsMrevrange() (c TsMrevrange) {
	c = TsMrevrange{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "TS.MREVRANGE")
	return c
}

func (c TsMrevrange) Fromtimestamp(fromtimestamp int64) TsMrevrangeFromtimestamp {
	c.cs.s = append(c.cs.s, strconv.FormatInt(fromtimestamp, 10))
	return (TsMrevrangeFromtimestamp)(c)
}

type TsMrevrangeAggregationAggregationAvg Completed

func (c TsMrevrangeAggregationAggregationAvg) Bucketduration(bucketduration int64) TsMrevrangeAggregationBucketduration {
	c.cs.s = append(c.cs.s, strconv.FormatInt(bucketduration, 10))
	return (TsMrevrangeAggregationBucketduration)(c)
}

type TsMrevrangeAggregationAggregationCount Completed

func (c TsMrevrangeAggregationAggregationCount) Bucketduration(bucketduration int64) TsMrevrangeAggregationBucketduration {
	c.cs.s = append(c.cs.s, strconv.FormatInt(bucketduration, 10))
	return (TsMrevrangeAggregationBucketduration)(c)
}

type TsMrevrangeAggregationAggregationFirst Completed

func (c TsMrevrangeAggregationAggregationFirst) Bucketduration(bucketduration int64) TsMrevrangeAggregationBucketduration {
	c.cs.s = append(c.cs.s, strconv.FormatInt(bucketduration, 10))
	return (TsMrevrangeAggregationBucketduration)(c)
}

type TsMrevrangeAggregationAggregationLast Completed

func (c TsMrevrangeAggregationAggregationLast) Bucketduration(bucketduration int64) TsMrevrangeAggregationBucketduration {
	c.cs.s = append(c.cs.s, strconv.FormatInt(bucketduration, 10))
	return (TsMrevrangeAggregationBucketduration)(c)
}

type TsMrevrangeAggregationAggregationMax Completed

func (c TsMrevrangeAggregationAggregationMax) Bucketduration(bucketduration int64) TsMrevrangeAggregationBucketduration {
	c.cs.s = append(c.cs.s, strconv.FormatInt(bucketduration, 10))
	return (TsMrevrangeAggregationBucketduration)(c)
}

type TsMrevrangeAggregationAggregationMin Completed

func (c TsMrevrangeAggregationAggregationMin) Bucketduration(bucketduration int64) TsMrevrangeAggregationBucketduration {
	c.cs.s = append(c.cs.s, strconv.FormatInt(bucketduration, 10))
	return (TsMrevrangeAggregationBucketduration)(c)
}

type TsMrevrangeAggregationAggregationRange Completed

func (c TsMrevrangeAggregationAggregationRange) Bucketduration(bucketduration int64) TsMrevrangeAggregationBucketduration {
	c.cs.s = append(c.cs.s, strconv.FormatInt(bucketduration, 10))
	return (TsMrevrangeAggregationBucketduration)(c)
}

type TsMrevrangeAggregationAggregationStdP Completed

func (c TsMrevrangeAggregationAggregationStdP) Bucketduration(bucketduration int64) TsMrevrangeAggregationBucketduration {
	c.cs.s = append(c.cs.s, strconv.FormatInt(bucketduration, 10))
	return (TsMrevrangeAggregationBucketduration)(c)
}

type TsMrevrangeAggregationAggregationStdS Completed

func (c TsMrevrangeAggregationAggregationStdS) Bucketduration(bucketduration int64) TsMrevrangeAggregationBucketduration {
	c.cs.s = append(c.cs.s, strconv.FormatInt(bucketduration, 10))
	return (TsMrevrangeAggregationBucketduration)(c)
}

type TsMrevrangeAggregationAggregationSum Completed

func (c TsMrevrangeAggregationAggregationSum) Bucketduration(bucketduration int64) TsMrevrangeAggregationBucketduration {
	c.cs.s = append(c.cs.s, strconv.FormatInt(bucketduration, 10))
	return (TsMrevrangeAggregationBucketduration)(c)
}

type TsMrevrangeAggregationAggregationTwa Completed

func (c TsMrevrangeAggregationAggregationTwa) Bucketduration(bucketduration int64) TsMrevrangeAggregationBucketduration {
	c.cs.s = append(c.cs.s, strconv.FormatInt(bucketduration, 10))
	return (TsMrevrangeAggregationBucketduration)(c)
}

type TsMrevrangeAggregationAggregationVarP Completed

func (c TsMrevrangeAggregationAggregationVarP) Bucketduration(bucketduration int64) TsMrevrangeAggregationBucketduration {
	c.cs.s = append(c.cs.s, strconv.FormatInt(bucketduration, 10))
	return (TsMrevrangeAggregationBucketduration)(c)
}

type TsMrevrangeAggregationAggregationVarS Completed

func (c TsMrevrangeAggregationAggregationVarS) Bucketduration(bucketduration int64) TsMrevrangeAggregationBucketduration {
	c.cs.s = append(c.cs.s, strconv.FormatInt(bucketduration, 10))
	return (TsMrevrangeAggregationBucketduration)(c)
}

type TsMrevrangeAggregationBucketduration Completed

func (c TsMrevrangeAggregationBucketduration) Buckettimestamp(buckettimestamp string) TsMrevrangeAggregationBuckettimestamp {
	c.cs.s = append(c.cs.s, "BUCKETTIMESTAMP", buckettimestamp)
	return (TsMrevrangeAggregationBuckettimestamp)(c)
}

func (c TsMrevrangeAggregationBucketduration) Empty() TsMrevrangeAggregationEmpty {
	c.cs.s = append(c.cs.s, "EMPTY")
	return (TsMrevrangeAggregationEmpty)(c)
}

func (c TsMrevrangeAggregationBucketduration) Filter(filter ...string) TsMrevrangeFilter {
	c.cs.s = append(c.cs.s, "FILTER")
	c.cs.s = append(c.cs.s, filter...)
	return (TsMrevrangeFilter)(c)
}

type TsMrevrangeAggregationBuckettimestamp Completed

func (c TsMrevrangeAggregationBuckettimestamp) Empty() TsMrevrangeAggregationEmpty {
	c.cs.s = append(c.cs.s, "EMPTY")
	return (TsMrevrangeAggregationEmpty)(c)
}

func (c TsMrevrangeAggregationBuckettimestamp) Filter(filter ...string) TsMrevrangeFilter {
	c.cs.s = append(c.cs.s, "FILTER")
	c.cs.s = append(c.cs.s, filter...)
	return (TsMrevrangeFilter)(c)
}

type TsMrevrangeAggregationEmpty Completed

func (c TsMrevrangeAggregationEmpty) Filter(filter ...string) TsMrevrangeFilter {
	c.cs.s = append(c.cs.s, "FILTER")
	c.cs.s = append(c.cs.s, filter...)
	return (TsMrevrangeFilter)(c)
}

type TsMrevrangeAlign Completed

func (c TsMrevrangeAlign) AggregationAvg() TsMrevrangeAggregationAggregationAvg {
	c.cs.s = append(c.cs.s, "AGGREGATION", "AVG")
	return (TsMrevrangeAggregationAggregationAvg)(c)
}

func (c TsMrevrangeAlign) AggregationSum() TsMrevrangeAggregationAggregationSum {
	c.cs.s = append(c.cs.s, "AGGREGATION", "SUM")
	return (TsMrevrangeAggregationAggregationSum)(c)
}

func (c TsMrevrangeAlign) AggregationMin() TsMrevrangeAggregationAggregationMin {
	c.cs.s = append(c.cs.s, "AGGREGATION", "MIN")
	return (TsMrevrangeAggregationAggregationMin)(c)
}

func (c TsMrevrangeAlign) AggregationMax() TsMrevrangeAggregationAggregationMax {
	c.cs.s = append(c.cs.s, "AGGREGATION", "MAX")
	return (TsMrevrangeAggregationAggregationMax)(c)
}

func (c TsMrevrangeAlign) AggregationRange() TsMrevrangeAggregationAggregationRange {
	c.cs.s = append(c.cs.s, "AGGREGATION", "RANGE")
	return (TsMrevrangeAggregationAggregationRange)(c)
}

func (c TsMrevrangeAlign) AggregationCount() TsMrevrangeAggregationAggregationCount {
	c.cs.s = append(c.cs.s, "AGGREGATION", "COUNT")
	return (TsMrevrangeAggregationAggregationCount)(c)
}

func (c TsMrevrangeAlign) AggregationFirst() TsMrevrangeAggregationAggregationFirst {
	c.cs.s = append(c.cs.s, "AGGREGATION", "FIRST")
	return (TsMrevrangeAggregationAggregationFirst)(c)
}

func (c TsMrevrangeAlign) AggregationLast() TsMrevrangeAggregationAggregationLast {
	c.cs.s = append(c.cs.s, "AGGREGATION", "LAST")
	return (TsMrevrangeAggregationAggregationLast)(c)
}

func (c TsMrevrangeAlign) AggregationStdP() TsMrevrangeAggregationAggregationStdP {
	c.cs.s = append(c.cs.s, "AGGREGATION", "STD.P")
	return (TsMrevrangeAggregationAggregationStdP)(c)
}

func (c TsMrevrangeAlign) AggregationStdS() TsMrevrangeAggregationAggregationStdS {
	c.cs.s = append(c.cs.s, "AGGREGATION", "STD.S")
	return (TsMrevrangeAggregationAggregationStdS)(c)
}

func (c TsMrevrangeAlign) AggregationVarP() TsMrevrangeAggregationAggregationVarP {
	c.cs.s = append(c.cs.s, "AGGREGATION", "VAR.P")
	return (TsMrevrangeAggregationAggregationVarP)(c)
}

func (c TsMrevrangeAlign) AggregationVarS() TsMrevrangeAggregationAggregationVarS {
	c.cs.s = append(c.cs.s, "AGGREGATION", "VAR.S")
	return (TsMrevrangeAggregationAggregationVarS)(c)
}

func (c TsMrevrangeAlign) AggregationTwa() TsMrevrangeAggregationAggregationTwa {
	c.cs.s = append(c.cs.s, "AGGREGATION", "TWA")
	return (TsMrevrangeAggregationAggregationTwa)(c)
}

func (c TsMrevrangeAlign) Filter(filter ...string) TsMrevrangeFilter {
	c.cs.s = append(c.cs.s, "FILTER")
	c.cs.s = append(c.cs.s, filter...)
	return (TsMrevrangeFilter)(c)
}

type TsMrevrangeCount Completed

func (c TsMrevrangeCount) Align(value string) TsMrevrangeAlign {
	c.cs.s = append(c.cs.s, "ALIGN", value)
	return (TsMrevrangeAlign)(c)
}

func (c TsMrevrangeCount) AggregationAvg() TsMrevrangeAggregationAggregationAvg {
	c.cs.s = append(c.cs.s, "AGGREGATION", "AVG")
	return (TsMrevrangeAggregationAggregationAvg)(c)
}

func (c TsMrevrangeCount) AggregationSum() TsMrevrangeAggregationAggregationSum {
	c.cs.s = append(c.cs.s, "AGGREGATION", "SUM")
	return (TsMrevrangeAggregationAggregationSum)(c)
}

func (c TsMrevrangeCount) AggregationMin() TsMrevrangeAggregationAggregationMin {
	c.cs.s = append(c.cs.s, "AGGREGATION", "MIN")
	return (TsMrevrangeAggregationAggregationMin)(c)
}

func (c TsMrevrangeCount) AggregationMax() TsMrevrangeAggregationAggregationMax {
	c.cs.s = append(c.cs.s, "AGGREGATION", "MAX")
	return (TsMrevrangeAggregationAggregationMax)(c)
}

func (c TsMrevrangeCount) AggregationRange() TsMrevrangeAggregationAggregationRange {
	c.cs.s = append(c.cs.s, "AGGREGATION", "RANGE")
	return (TsMrevrangeAggregationAggregationRange)(c)
}

func (c TsMrevrangeCount) AggregationCount() TsMrevrangeAggregationAggregationCount {
	c.cs.s = append(c.cs.s, "AGGREGATION", "COUNT")
	return (TsMrevrangeAggregationAggregationCount)(c)
}

func (c TsMrevrangeCount) AggregationFirst() TsMrevrangeAggregationAggregationFirst {
	c.cs.s = append(c.cs.s, "AGGREGATION", "FIRST")
	return (TsMrevrangeAggregationAggregationFirst)(c)
}

func (c TsMrevrangeCount) AggregationLast() TsMrevrangeAggregationAggregationLast {
	c.cs.s = append(c.cs.s, "AGGREGATION", "LAST")
	return (TsMrevrangeAggregationAggregationLast)(c)
}

func (c TsMrevrangeCount) AggregationStdP() TsMrevrangeAggregationAggregationStdP {
	c.cs.s = append(c.cs.s, "AGGREGATION", "STD.P")
	return (TsMrevrangeAggregationAggregationStdP)(c)
}

func (c TsMrevrangeCount) AggregationStdS() TsMrevrangeAggregationAggregationStdS {
	c.cs.s = append(c.cs.s, "AGGREGATION", "STD.S")
	return (TsMrevrangeAggregationAggregationStdS)(c)
}

func (c TsMrevrangeCount) AggregationVarP() TsMrevrangeAggregationAggregationVarP {
	c.cs.s = append(c.cs.s, "AGGREGATION", "VAR.P")
	return (TsMrevrangeAggregationAggregationVarP)(c)
}

func (c TsMrevrangeCount) AggregationVarS() TsMrevrangeAggregationAggregationVarS {
	c.cs.s = append(c.cs.s, "AGGREGATION", "VAR.S")
	return (TsMrevrangeAggregationAggregationVarS)(c)
}

func (c TsMrevrangeCount) AggregationTwa() TsMrevrangeAggregationAggregationTwa {
	c.cs.s = append(c.cs.s, "AGGREGATION", "TWA")
	return (TsMrevrangeAggregationAggregationTwa)(c)
}

func (c TsMrevrangeCount) Filter(filter ...string) TsMrevrangeFilter {
	c.cs.s = append(c.cs.s, "FILTER")
	c.cs.s = append(c.cs.s, filter...)
	return (TsMrevrangeFilter)(c)
}

type TsMrevrangeFilter Completed

func (c TsMrevrangeFilter) Filter(filter ...string) TsMrevrangeFilter {
	c.cs.s = append(c.cs.s, "FILTER")
	c.cs.s = append(c.cs.s, filter...)
	return c
}

func (c TsMrevrangeFilter) Groupby(label string, reduce string, reducer string) TsMrevrangeGroupby {
	c.cs.s = append(c.cs.s, "GROUPBY", label, reduce, reducer)
	return (TsMrevrangeGroupby)(c)
}

func (c TsMrevrangeFilter) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type TsMrevrangeFilterByTs Completed

func (c TsMrevrangeFilterByTs) FilterByTs(timestamp ...int64) TsMrevrangeFilterByTs {
	c.cs.s = append(c.cs.s, "FILTER_BY_TS")
	for _, n := range timestamp {
		c.cs.s = append(c.cs.s, strconv.FormatInt(n, 10))
	}
	return c
}

func (c TsMrevrangeFilterByTs) FilterByValue(min float64, max float64) TsMrevrangeFilterByValue {
	c.cs.s = append(c.cs.s, "FILTER_BY_VALUE", strconv.FormatFloat(min, 'f', -1, 64), strconv.FormatFloat(max, 'f', -1, 64))
	return (TsMrevrangeFilterByValue)(c)
}

func (c TsMrevrangeFilterByTs) Withlabels() TsMrevrangeWithlabels {
	c.cs.s = append(c.cs.s, "WITHLABELS")
	return (TsMrevrangeWithlabels)(c)
}

func (c TsMrevrangeFilterByTs) SelectedLabels(labels []string) TsMrevrangeSelectedLabels {
	c.cs.s = append(c.cs.s, "SELECTED_LABELS")
	c.cs.s = append(c.cs.s, labels...)
	return (TsMrevrangeSelectedLabels)(c)
}

func (c TsMrevrangeFilterByTs) Count(count int64) TsMrevrangeCount {
	c.cs.s = append(c.cs.s, "COUNT", strconv.FormatInt(count, 10))
	return (TsMrevrangeCount)(c)
}

func (c TsMrevrangeFilterByTs) Align(value string) TsMrevrangeAlign {
	c.cs.s = append(c.cs.s, "ALIGN", value)
	return (TsMrevrangeAlign)(c)
}

func (c TsMrevrangeFilterByTs) AggregationAvg() TsMrevrangeAggregationAggregationAvg {
	c.cs.s = append(c.cs.s, "AGGREGATION", "AVG")
	return (TsMrevrangeAggregationAggregationAvg)(c)
}

func (c TsMrevrangeFilterByTs) AggregationSum() TsMrevrangeAggregationAggregationSum {
	c.cs.s = append(c.cs.s, "AGGREGATION", "SUM")
	return (TsMrevrangeAggregationAggregationSum)(c)
}

func (c TsMrevrangeFilterByTs) AggregationMin() TsMrevrangeAggregationAggregationMin {
	c.cs.s = append(c.cs.s, "AGGREGATION", "MIN")
	return (TsMrevrangeAggregationAggregationMin)(c)
}

func (c TsMrevrangeFilterByTs) AggregationMax() TsMrevrangeAggregationAggregationMax {
	c.cs.s = append(c.cs.s, "AGGREGATION", "MAX")
	return (TsMrevrangeAggregationAggregationMax)(c)
}

func (c TsMrevrangeFilterByTs) AggregationRange() TsMrevrangeAggregationAggregationRange {
	c.cs.s = append(c.cs.s, "AGGREGATION", "RANGE")
	return (TsMrevrangeAggregationAggregationRange)(c)
}

func (c TsMrevrangeFilterByTs) AggregationCount() TsMrevrangeAggregationAggregationCount {
	c.cs.s = append(c.cs.s, "AGGREGATION", "COUNT")
	return (TsMrevrangeAggregationAggregationCount)(c)
}

func (c TsMrevrangeFilterByTs) AggregationFirst() TsMrevrangeAggregationAggregationFirst {
	c.cs.s = append(c.cs.s, "AGGREGATION", "FIRST")
	return (TsMrevrangeAggregationAggregationFirst)(c)
}

func (c TsMrevrangeFilterByTs) AggregationLast() TsMrevrangeAggregationAggregationLast {
	c.cs.s = append(c.cs.s, "AGGREGATION", "LAST")
	return (TsMrevrangeAggregationAggregationLast)(c)
}

func (c TsMrevrangeFilterByTs) AggregationStdP() TsMrevrangeAggregationAggregationStdP {
	c.cs.s = append(c.cs.s, "AGGREGATION", "STD.P")
	return (TsMrevrangeAggregationAggregationStdP)(c)
}

func (c TsMrevrangeFilterByTs) AggregationStdS() TsMrevrangeAggregationAggregationStdS {
	c.cs.s = append(c.cs.s, "AGGREGATION", "STD.S")
	return (TsMrevrangeAggregationAggregationStdS)(c)
}

func (c TsMrevrangeFilterByTs) AggregationVarP() TsMrevrangeAggregationAggregationVarP {
	c.cs.s = append(c.cs.s, "AGGREGATION", "VAR.P")
	return (TsMrevrangeAggregationAggregationVarP)(c)
}

func (c TsMrevrangeFilterByTs) AggregationVarS() TsMrevrangeAggregationAggregationVarS {
	c.cs.s = append(c.cs.s, "AGGREGATION", "VAR.S")
	return (TsMrevrangeAggregationAggregationVarS)(c)
}

func (c TsMrevrangeFilterByTs) AggregationTwa() TsMrevrangeAggregationAggregationTwa {
	c.cs.s = append(c.cs.s, "AGGREGATION", "TWA")
	return (TsMrevrangeAggregationAggregationTwa)(c)
}

func (c TsMrevrangeFilterByTs) Filter(filter ...string) TsMrevrangeFilter {
	c.cs.s = append(c.cs.s, "FILTER")
	c.cs.s = append(c.cs.s, filter...)
	return (TsMrevrangeFilter)(c)
}

type TsMrevrangeFilterByValue Completed

func (c TsMrevrangeFilterByValue) Withlabels() TsMrevrangeWithlabels {
	c.cs.s = append(c.cs.s, "WITHLABELS")
	return (TsMrevrangeWithlabels)(c)
}

func (c TsMrevrangeFilterByValue) SelectedLabels(labels []string) TsMrevrangeSelectedLabels {
	c.cs.s = append(c.cs.s, "SELECTED_LABELS")
	c.cs.s = append(c.cs.s, labels...)
	return (TsMrevrangeSelectedLabels)(c)
}

func (c TsMrevrangeFilterByValue) Count(count int64) TsMrevrangeCount {
	c.cs.s = append(c.cs.s, "COUNT", strconv.FormatInt(count, 10))
	return (TsMrevrangeCount)(c)
}

func (c TsMrevrangeFilterByValue) Align(value string) TsMrevrangeAlign {
	c.cs.s = append(c.cs.s, "ALIGN", value)
	return (TsMrevrangeAlign)(c)
}

func (c TsMrevrangeFilterByValue) AggregationAvg() TsMrevrangeAggregationAggregationAvg {
	c.cs.s = append(c.cs.s, "AGGREGATION", "AVG")
	return (TsMrevrangeAggregationAggregationAvg)(c)
}

func (c TsMrevrangeFilterByValue) AggregationSum() TsMrevrangeAggregationAggregationSum {
	c.cs.s = append(c.cs.s, "AGGREGATION", "SUM")
	return (TsMrevrangeAggregationAggregationSum)(c)
}

func (c TsMrevrangeFilterByValue) AggregationMin() TsMrevrangeAggregationAggregationMin {
	c.cs.s = append(c.cs.s, "AGGREGATION", "MIN")
	return (TsMrevrangeAggregationAggregationMin)(c)
}

func (c TsMrevrangeFilterByValue) AggregationMax() TsMrevrangeAggregationAggregationMax {
	c.cs.s = append(c.cs.s, "AGGREGATION", "MAX")
	return (TsMrevrangeAggregationAggregationMax)(c)
}

func (c TsMrevrangeFilterByValue) AggregationRange() TsMrevrangeAggregationAggregationRange {
	c.cs.s = append(c.cs.s, "AGGREGATION", "RANGE")
	return (TsMrevrangeAggregationAggregationRange)(c)
}

func (c TsMrevrangeFilterByValue) AggregationCount() TsMrevrangeAggregationAggregationCount {
	c.cs.s = append(c.cs.s, "AGGREGATION", "COUNT")
	return (TsMrevrangeAggregationAggregationCount)(c)
}

func (c TsMrevrangeFilterByValue) AggregationFirst() TsMrevrangeAggregationAggregationFirst {
	c.cs.s = append(c.cs.s, "AGGREGATION", "FIRST")
	return (TsMrevrangeAggregationAggregationFirst)(c)
}

func (c TsMrevrangeFilterByValue) AggregationLast() TsMrevrangeAggregationAggregationLast {
	c.cs.s = append(c.cs.s, "AGGREGATION", "LAST")
	return (TsMrevrangeAggregationAggregationLast)(c)
}

func (c TsMrevrangeFilterByValue) AggregationStdP() TsMrevrangeAggregationAggregationStdP {
	c.cs.s = append(c.cs.s, "AGGREGATION", "STD.P")
	return (TsMrevrangeAggregationAggregationStdP)(c)
}

func (c TsMrevrangeFilterByValue) AggregationStdS() TsMrevrangeAggregationAggregationStdS {
	c.cs.s = append(c.cs.s, "AGGREGATION", "STD.S")
	return (TsMrevrangeAggregationAggregationStdS)(c)
}

func (c TsMrevrangeFilterByValue) AggregationVarP() TsMrevrangeAggregationAggregationVarP {
	c.cs.s = append(c.cs.s, "AGGREGATION", "VAR.P")
	return (TsMrevrangeAggregationAggregationVarP)(c)
}

func (c TsMrevrangeFilterByValue) AggregationVarS() TsMrevrangeAggregationAggregationVarS {
	c.cs.s = append(c.cs.s, "AGGREGATION", "VAR.S")
	return (TsMrevrangeAggregationAggregationVarS)(c)
}

func (c TsMrevrangeFilterByValue) AggregationTwa() TsMrevrangeAggregationAggregationTwa {
	c.cs.s = append(c.cs.s, "AGGREGATION", "TWA")
	return (TsMrevrangeAggregationAggregationTwa)(c)
}

func (c TsMrevrangeFilterByValue) Filter(filter ...string) TsMrevrangeFilter {
	c.cs.s = append(c.cs.s, "FILTER")
	c.cs.s = append(c.cs.s, filter...)
	return (TsMrevrangeFilter)(c)
}

type TsMrevrangeFromtimestamp Completed

func (c TsMrevrangeFromtimestamp) Totimestamp(totimestamp int64) TsMrevrangeTotimestamp {
	c.cs.s = append(c.cs.s, strconv.FormatInt(totimestamp, 10))
	return (TsMrevrangeTotimestamp)(c)
}

type TsMrevrangeGroupby Completed

func (c TsMrevrangeGroupby) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type TsMrevrangeLatest Completed

func (c TsMrevrangeLatest) FilterByTs(timestamp ...int64) TsMrevrangeFilterByTs {
	c.cs.s = append(c.cs.s, "FILTER_BY_TS")
	for _, n := range timestamp {
		c.cs.s = append(c.cs.s, strconv.FormatInt(n, 10))
	}
	return (TsMrevrangeFilterByTs)(c)
}

func (c TsMrevrangeLatest) FilterByValue(min float64, max float64) TsMrevrangeFilterByValue {
	c.cs.s = append(c.cs.s, "FILTER_BY_VALUE", strconv.FormatFloat(min, 'f', -1, 64), strconv.FormatFloat(max, 'f', -1, 64))
	return (TsMrevrangeFilterByValue)(c)
}

func (c TsMrevrangeLatest) Withlabels() TsMrevrangeWithlabels {
	c.cs.s = append(c.cs.s, "WITHLABELS")
	return (TsMrevrangeWithlabels)(c)
}

func (c TsMrevrangeLatest) SelectedLabels(labels []string) TsMrevrangeSelectedLabels {
	c.cs.s = append(c.cs.s, "SELECTED_LABELS")
	c.cs.s = append(c.cs.s, labels...)
	return (TsMrevrangeSelectedLabels)(c)
}

func (c TsMrevrangeLatest) Count(count int64) TsMrevrangeCount {
	c.cs.s = append(c.cs.s, "COUNT", strconv.FormatInt(count, 10))
	return (TsMrevrangeCount)(c)
}

func (c TsMrevrangeLatest) Align(value string) TsMrevrangeAlign {
	c.cs.s = append(c.cs.s, "ALIGN", value)
	return (TsMrevrangeAlign)(c)
}

func (c TsMrevrangeLatest) AggregationAvg() TsMrevrangeAggregationAggregationAvg {
	c.cs.s = append(c.cs.s, "AGGREGATION", "AVG")
	return (TsMrevrangeAggregationAggregationAvg)(c)
}

func (c TsMrevrangeLatest) AggregationSum() TsMrevrangeAggregationAggregationSum {
	c.cs.s = append(c.cs.s, "AGGREGATION", "SUM")
	return (TsMrevrangeAggregationAggregationSum)(c)
}

func (c TsMrevrangeLatest) AggregationMin() TsMrevrangeAggregationAggregationMin {
	c.cs.s = append(c.cs.s, "AGGREGATION", "MIN")
	return (TsMrevrangeAggregationAggregationMin)(c)
}

func (c TsMrevrangeLatest) AggregationMax() TsMrevrangeAggregationAggregationMax {
	c.cs.s = append(c.cs.s, "AGGREGATION", "MAX")
	return (TsMrevrangeAggregationAggregationMax)(c)
}

func (c TsMrevrangeLatest) AggregationRange() TsMrevrangeAggregationAggregationRange {
	c.cs.s = append(c.cs.s, "AGGREGATION", "RANGE")
	return (TsMrevrangeAggregationAggregationRange)(c)
}

func (c TsMrevrangeLatest) AggregationCount() TsMrevrangeAggregationAggregationCount {
	c.cs.s = append(c.cs.s, "AGGREGATION", "COUNT")
	return (TsMrevrangeAggregationAggregationCount)(c)
}

func (c TsMrevrangeLatest) AggregationFirst() TsMrevrangeAggregationAggregationFirst {
	c.cs.s = append(c.cs.s, "AGGREGATION", "FIRST")
	return (TsMrevrangeAggregationAggregationFirst)(c)
}

func (c TsMrevrangeLatest) AggregationLast() TsMrevrangeAggregationAggregationLast {
	c.cs.s = append(c.cs.s, "AGGREGATION", "LAST")
	return (TsMrevrangeAggregationAggregationLast)(c)
}

func (c TsMrevrangeLatest) AggregationStdP() TsMrevrangeAggregationAggregationStdP {
	c.cs.s = append(c.cs.s, "AGGREGATION", "STD.P")
	return (TsMrevrangeAggregationAggregationStdP)(c)
}

func (c TsMrevrangeLatest) AggregationStdS() TsMrevrangeAggregationAggregationStdS {
	c.cs.s = append(c.cs.s, "AGGREGATION", "STD.S")
	return (TsMrevrangeAggregationAggregationStdS)(c)
}

func (c TsMrevrangeLatest) AggregationVarP() TsMrevrangeAggregationAggregationVarP {
	c.cs.s = append(c.cs.s, "AGGREGATION", "VAR.P")
	return (TsMrevrangeAggregationAggregationVarP)(c)
}

func (c TsMrevrangeLatest) AggregationVarS() TsMrevrangeAggregationAggregationVarS {
	c.cs.s = append(c.cs.s, "AGGREGATION", "VAR.S")
	return (TsMrevrangeAggregationAggregationVarS)(c)
}

func (c TsMrevrangeLatest) AggregationTwa() TsMrevrangeAggregationAggregationTwa {
	c.cs.s = append(c.cs.s, "AGGREGATION", "TWA")
	return (TsMrevrangeAggregationAggregationTwa)(c)
}

func (c TsMrevrangeLatest) Filter(filter ...string) TsMrevrangeFilter {
	c.cs.s = append(c.cs.s, "FILTER")
	c.cs.s = append(c.cs.s, filter...)
	return (TsMrevrangeFilter)(c)
}

type TsMrevrangeSelectedLabels Completed

func (c TsMrevrangeSelectedLabels) Count(count int64) TsMrevrangeCount {
	c.cs.s = append(c.cs.s, "COUNT", strconv.FormatInt(count, 10))
	return (TsMrevrangeCount)(c)
}

func (c TsMrevrangeSelectedLabels) Align(value string) TsMrevrangeAlign {
	c.cs.s = append(c.cs.s, "ALIGN", value)
	return (TsMrevrangeAlign)(c)
}

func (c TsMrevrangeSelectedLabels) AggregationAvg() TsMrevrangeAggregationAggregationAvg {
	c.cs.s = append(c.cs.s, "AGGREGATION", "AVG")
	return (TsMrevrangeAggregationAggregationAvg)(c)
}

func (c TsMrevrangeSelectedLabels) AggregationSum() TsMrevrangeAggregationAggregationSum {
	c.cs.s = append(c.cs.s, "AGGREGATION", "SUM")
	return (TsMrevrangeAggregationAggregationSum)(c)
}

func (c TsMrevrangeSelectedLabels) AggregationMin() TsMrevrangeAggregationAggregationMin {
	c.cs.s = append(c.cs.s, "AGGREGATION", "MIN")
	return (TsMrevrangeAggregationAggregationMin)(c)
}

func (c TsMrevrangeSelectedLabels) AggregationMax() TsMrevrangeAggregationAggregationMax {
	c.cs.s = append(c.cs.s, "AGGREGATION", "MAX")
	return (TsMrevrangeAggregationAggregationMax)(c)
}

func (c TsMrevrangeSelectedLabels) AggregationRange() TsMrevrangeAggregationAggregationRange {
	c.cs.s = append(c.cs.s, "AGGREGATION", "RANGE")
	return (TsMrevrangeAggregationAggregationRange)(c)
}

func (c TsMrevrangeSelectedLabels) AggregationCount() TsMrevrangeAggregationAggregationCount {
	c.cs.s = append(c.cs.s, "AGGREGATION", "COUNT")
	return (TsMrevrangeAggregationAggregationCount)(c)
}

func (c TsMrevrangeSelectedLabels) AggregationFirst() TsMrevrangeAggregationAggregationFirst {
	c.cs.s = append(c.cs.s, "AGGREGATION", "FIRST")
	return (TsMrevrangeAggregationAggregationFirst)(c)
}

func (c TsMrevrangeSelectedLabels) AggregationLast() TsMrevrangeAggregationAggregationLast {
	c.cs.s = append(c.cs.s, "AGGREGATION", "LAST")
	return (TsMrevrangeAggregationAggregationLast)(c)
}

func (c TsMrevrangeSelectedLabels) AggregationStdP() TsMrevrangeAggregationAggregationStdP {
	c.cs.s = append(c.cs.s, "AGGREGATION", "STD.P")
	return (TsMrevrangeAggregationAggregationStdP)(c)
}

func (c TsMrevrangeSelectedLabels) AggregationStdS() TsMrevrangeAggregationAggregationStdS {
	c.cs.s = append(c.cs.s, "AGGREGATION", "STD.S")
	return (TsMrevrangeAggregationAggregationStdS)(c)
}

func (c TsMrevrangeSelectedLabels) AggregationVarP() TsMrevrangeAggregationAggregationVarP {
	c.cs.s = append(c.cs.s, "AGGREGATION", "VAR.P")
	return (TsMrevrangeAggregationAggregationVarP)(c)
}

func (c TsMrevrangeSelectedLabels) AggregationVarS() TsMrevrangeAggregationAggregationVarS {
	c.cs.s = append(c.cs.s, "AGGREGATION", "VAR.S")
	return (TsMrevrangeAggregationAggregationVarS)(c)
}

func (c TsMrevrangeSelectedLabels) AggregationTwa() TsMrevrangeAggregationAggregationTwa {
	c.cs.s = append(c.cs.s, "AGGREGATION", "TWA")
	return (TsMrevrangeAggregationAggregationTwa)(c)
}

func (c TsMrevrangeSelectedLabels) Filter(filter ...string) TsMrevrangeFilter {
	c.cs.s = append(c.cs.s, "FILTER")
	c.cs.s = append(c.cs.s, filter...)
	return (TsMrevrangeFilter)(c)
}

type TsMrevrangeTotimestamp Completed

func (c TsMrevrangeTotimestamp) Latest() TsMrevrangeLatest {
	c.cs.s = append(c.cs.s, "LATEST")
	return (TsMrevrangeLatest)(c)
}

func (c TsMrevrangeTotimestamp) FilterByTs(timestamp ...int64) TsMrevrangeFilterByTs {
	c.cs.s = append(c.cs.s, "FILTER_BY_TS")
	for _, n := range timestamp {
		c.cs.s = append(c.cs.s, strconv.FormatInt(n, 10))
	}
	return (TsMrevrangeFilterByTs)(c)
}

func (c TsMrevrangeTotimestamp) FilterByValue(min float64, max float64) TsMrevrangeFilterByValue {
	c.cs.s = append(c.cs.s, "FILTER_BY_VALUE", strconv.FormatFloat(min, 'f', -1, 64), strconv.FormatFloat(max, 'f', -1, 64))
	return (TsMrevrangeFilterByValue)(c)
}

func (c TsMrevrangeTotimestamp) Withlabels() TsMrevrangeWithlabels {
	c.cs.s = append(c.cs.s, "WITHLABELS")
	return (TsMrevrangeWithlabels)(c)
}

func (c TsMrevrangeTotimestamp) SelectedLabels(labels []string) TsMrevrangeSelectedLabels {
	c.cs.s = append(c.cs.s, "SELECTED_LABELS")
	c.cs.s = append(c.cs.s, labels...)
	return (TsMrevrangeSelectedLabels)(c)
}

func (c TsMrevrangeTotimestamp) Count(count int64) TsMrevrangeCount {
	c.cs.s = append(c.cs.s, "COUNT", strconv.FormatInt(count, 10))
	return (TsMrevrangeCount)(c)
}

func (c TsMrevrangeTotimestamp) Align(value string) TsMrevrangeAlign {
	c.cs.s = append(c.cs.s, "ALIGN", value)
	return (TsMrevrangeAlign)(c)
}

func (c TsMrevrangeTotimestamp) AggregationAvg() TsMrevrangeAggregationAggregationAvg {
	c.cs.s = append(c.cs.s, "AGGREGATION", "AVG")
	return (TsMrevrangeAggregationAggregationAvg)(c)
}

func (c TsMrevrangeTotimestamp) AggregationSum() TsMrevrangeAggregationAggregationSum {
	c.cs.s = append(c.cs.s, "AGGREGATION", "SUM")
	return (TsMrevrangeAggregationAggregationSum)(c)
}

func (c TsMrevrangeTotimestamp) AggregationMin() TsMrevrangeAggregationAggregationMin {
	c.cs.s = append(c.cs.s, "AGGREGATION", "MIN")
	return (TsMrevrangeAggregationAggregationMin)(c)
}

func (c TsMrevrangeTotimestamp) AggregationMax() TsMrevrangeAggregationAggregationMax {
	c.cs.s = append(c.cs.s, "AGGREGATION", "MAX")
	return (TsMrevrangeAggregationAggregationMax)(c)
}

func (c TsMrevrangeTotimestamp) AggregationRange() TsMrevrangeAggregationAggregationRange {
	c.cs.s = append(c.cs.s, "AGGREGATION", "RANGE")
	return (TsMrevrangeAggregationAggregationRange)(c)
}

func (c TsMrevrangeTotimestamp) AggregationCount() TsMrevrangeAggregationAggregationCount {
	c.cs.s = append(c.cs.s, "AGGREGATION", "COUNT")
	return (TsMrevrangeAggregationAggregationCount)(c)
}

func (c TsMrevrangeTotimestamp) AggregationFirst() TsMrevrangeAggregationAggregationFirst {
	c.cs.s = append(c.cs.s, "AGGREGATION", "FIRST")
	return (TsMrevrangeAggregationAggregationFirst)(c)
}

func (c TsMrevrangeTotimestamp) AggregationLast() TsMrevrangeAggregationAggregationLast {
	c.cs.s = append(c.cs.s, "AGGREGATION", "LAST")
	return (TsMrevrangeAggregationAggregationLast)(c)
}

func (c TsMrevrangeTotimestamp) AggregationStdP() TsMrevrangeAggregationAggregationStdP {
	c.cs.s = append(c.cs.s, "AGGREGATION", "STD.P")
	return (TsMrevrangeAggregationAggregationStdP)(c)
}

func (c TsMrevrangeTotimestamp) AggregationStdS() TsMrevrangeAggregationAggregationStdS {
	c.cs.s = append(c.cs.s, "AGGREGATION", "STD.S")
	return (TsMrevrangeAggregationAggregationStdS)(c)
}

func (c TsMrevrangeTotimestamp) AggregationVarP() TsMrevrangeAggregationAggregationVarP {
	c.cs.s = append(c.cs.s, "AGGREGATION", "VAR.P")
	return (TsMrevrangeAggregationAggregationVarP)(c)
}

func (c TsMrevrangeTotimestamp) AggregationVarS() TsMrevrangeAggregationAggregationVarS {
	c.cs.s = append(c.cs.s, "AGGREGATION", "VAR.S")
	return (TsMrevrangeAggregationAggregationVarS)(c)
}

func (c TsMrevrangeTotimestamp) AggregationTwa() TsMrevrangeAggregationAggregationTwa {
	c.cs.s = append(c.cs.s, "AGGREGATION", "TWA")
	return (TsMrevrangeAggregationAggregationTwa)(c)
}

func (c TsMrevrangeTotimestamp) Filter(filter ...string) TsMrevrangeFilter {
	c.cs.s = append(c.cs.s, "FILTER")
	c.cs.s = append(c.cs.s, filter...)
	return (TsMrevrangeFilter)(c)
}

type TsMrevrangeWithlabels Completed

func (c TsMrevrangeWithlabels) Count(count int64) TsMrevrangeCount {
	c.cs.s = append(c.cs.s, "COUNT", strconv.FormatInt(count, 10))
	return (TsMrevrangeCount)(c)
}

func (c TsMrevrangeWithlabels) Align(value string) TsMrevrangeAlign {
	c.cs.s = append(c.cs.s, "ALIGN", value)
	return (TsMrevrangeAlign)(c)
}

func (c TsMrevrangeWithlabels) AggregationAvg() TsMrevrangeAggregationAggregationAvg {
	c.cs.s = append(c.cs.s, "AGGREGATION", "AVG")
	return (TsMrevrangeAggregationAggregationAvg)(c)
}

func (c TsMrevrangeWithlabels) AggregationSum() TsMrevrangeAggregationAggregationSum {
	c.cs.s = append(c.cs.s, "AGGREGATION", "SUM")
	return (TsMrevrangeAggregationAggregationSum)(c)
}

func (c TsMrevrangeWithlabels) AggregationMin() TsMrevrangeAggregationAggregationMin {
	c.cs.s = append(c.cs.s, "AGGREGATION", "MIN")
	return (TsMrevrangeAggregationAggregationMin)(c)
}

func (c TsMrevrangeWithlabels) AggregationMax() TsMrevrangeAggregationAggregationMax {
	c.cs.s = append(c.cs.s, "AGGREGATION", "MAX")
	return (TsMrevrangeAggregationAggregationMax)(c)
}

func (c TsMrevrangeWithlabels) AggregationRange() TsMrevrangeAggregationAggregationRange {
	c.cs.s = append(c.cs.s, "AGGREGATION", "RANGE")
	return (TsMrevrangeAggregationAggregationRange)(c)
}

func (c TsMrevrangeWithlabels) AggregationCount() TsMrevrangeAggregationAggregationCount {
	c.cs.s = append(c.cs.s, "AGGREGATION", "COUNT")
	return (TsMrevrangeAggregationAggregationCount)(c)
}

func (c TsMrevrangeWithlabels) AggregationFirst() TsMrevrangeAggregationAggregationFirst {
	c.cs.s = append(c.cs.s, "AGGREGATION", "FIRST")
	return (TsMrevrangeAggregationAggregationFirst)(c)
}

func (c TsMrevrangeWithlabels) AggregationLast() TsMrevrangeAggregationAggregationLast {
	c.cs.s = append(c.cs.s, "AGGREGATION", "LAST")
	return (TsMrevrangeAggregationAggregationLast)(c)
}

func (c TsMrevrangeWithlabels) AggregationStdP() TsMrevrangeAggregationAggregationStdP {
	c.cs.s = append(c.cs.s, "AGGREGATION", "STD.P")
	return (TsMrevrangeAggregationAggregationStdP)(c)
}

func (c TsMrevrangeWithlabels) AggregationStdS() TsMrevrangeAggregationAggregationStdS {
	c.cs.s = append(c.cs.s, "AGGREGATION", "STD.S")
	return (TsMrevrangeAggregationAggregationStdS)(c)
}

func (c TsMrevrangeWithlabels) AggregationVarP() TsMrevrangeAggregationAggregationVarP {
	c.cs.s = append(c.cs.s, "AGGREGATION", "VAR.P")
	return (TsMrevrangeAggregationAggregationVarP)(c)
}

func (c TsMrevrangeWithlabels) AggregationVarS() TsMrevrangeAggregationAggregationVarS {
	c.cs.s = append(c.cs.s, "AGGREGATION", "VAR.S")
	return (TsMrevrangeAggregationAggregationVarS)(c)
}

func (c TsMrevrangeWithlabels) AggregationTwa() TsMrevrangeAggregationAggregationTwa {
	c.cs.s = append(c.cs.s, "AGGREGATION", "TWA")
	return (TsMrevrangeAggregationAggregationTwa)(c)
}

func (c TsMrevrangeWithlabels) Filter(filter ...string) TsMrevrangeFilter {
	c.cs.s = append(c.cs.s, "FILTER")
	c.cs.s = append(c.cs.s, filter...)
	return (TsMrevrangeFilter)(c)
}

type TsQueryindex Completed

func (b Builder) TsQueryindex() (c TsQueryindex) {
	c = TsQueryindex{cs: get(), ks: b.ks, cf: readonly}
	c.cs.s = append(c.cs.s, "TS.QUERYINDEX")
	return c
}

func (c TsQueryindex) Filter(filter ...string) TsQueryindexFilter {
	c.cs.s = append(c.cs.s, filter...)
	return (TsQueryindexFilter)(c)
}

type TsQueryindexFilter Completed

func (c TsQueryindexFilter) Filter(filter ...string) TsQueryindexFilter {
	c.cs.s = append(c.cs.s, filter...)
	return c
}

func (c TsQueryindexFilter) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type TsRange Completed

func (b Builder) TsRange() (c TsRange) {
	c = TsRange{cs: get(), ks: b.ks, cf: readonly}
	c.cs.s = append(c.cs.s, "TS.RANGE")
	return c
}

func (c TsRange) Key(key string) TsRangeKey {
	if c.ks&NoSlot == NoSlot {
		c.ks = NoSlot | slot(key)
	} else {
		c.ks = check(c.ks, slot(key))
	}
	c.cs.s = append(c.cs.s, key)
	return (TsRangeKey)(c)
}

type TsRangeAggregationAggregationAvg Completed

func (c TsRangeAggregationAggregationAvg) Bucketduration(bucketduration int64) TsRangeAggregationBucketduration {
	c.cs.s = append(c.cs.s, strconv.FormatInt(bucketduration, 10))
	return (TsRangeAggregationBucketduration)(c)
}

type TsRangeAggregationAggregationCount Completed

func (c TsRangeAggregationAggregationCount) Bucketduration(bucketduration int64) TsRangeAggregationBucketduration {
	c.cs.s = append(c.cs.s, strconv.FormatInt(bucketduration, 10))
	return (TsRangeAggregationBucketduration)(c)
}

type TsRangeAggregationAggregationFirst Completed

func (c TsRangeAggregationAggregationFirst) Bucketduration(bucketduration int64) TsRangeAggregationBucketduration {
	c.cs.s = append(c.cs.s, strconv.FormatInt(bucketduration, 10))
	return (TsRangeAggregationBucketduration)(c)
}

type TsRangeAggregationAggregationLast Completed

func (c TsRangeAggregationAggregationLast) Bucketduration(bucketduration int64) TsRangeAggregationBucketduration {
	c.cs.s = append(c.cs.s, strconv.FormatInt(bucketduration, 10))
	return (TsRangeAggregationBucketduration)(c)
}

type TsRangeAggregationAggregationMax Completed

func (c TsRangeAggregationAggregationMax) Bucketduration(bucketduration int64) TsRangeAggregationBucketduration {
	c.cs.s = append(c.cs.s, strconv.FormatInt(bucketduration, 10))
	return (TsRangeAggregationBucketduration)(c)
}

type TsRangeAggregationAggregationMin Completed

func (c TsRangeAggregationAggregationMin) Bucketduration(bucketduration int64) TsRangeAggregationBucketduration {
	c.cs.s = append(c.cs.s, strconv.FormatInt(bucketduration, 10))
	return (TsRangeAggregationBucketduration)(c)
}

type TsRangeAggregationAggregationRange Completed

func (c TsRangeAggregationAggregationRange) Bucketduration(bucketduration int64) TsRangeAggregationBucketduration {
	c.cs.s = append(c.cs.s, strconv.FormatInt(bucketduration, 10))
	return (TsRangeAggregationBucketduration)(c)
}

type TsRangeAggregationAggregationStdP Completed

func (c TsRangeAggregationAggregationStdP) Bucketduration(bucketduration int64) TsRangeAggregationBucketduration {
	c.cs.s = append(c.cs.s, strconv.FormatInt(bucketduration, 10))
	return (TsRangeAggregationBucketduration)(c)
}

type TsRangeAggregationAggregationStdS Completed

func (c TsRangeAggregationAggregationStdS) Bucketduration(bucketduration int64) TsRangeAggregationBucketduration {
	c.cs.s = append(c.cs.s, strconv.FormatInt(bucketduration, 10))
	return (TsRangeAggregationBucketduration)(c)
}

type TsRangeAggregationAggregationSum Completed

func (c TsRangeAggregationAggregationSum) Bucketduration(bucketduration int64) TsRangeAggregationBucketduration {
	c.cs.s = append(c.cs.s, strconv.FormatInt(bucketduration, 10))
	return (TsRangeAggregationBucketduration)(c)
}

type TsRangeAggregationAggregationTwa Completed

func (c TsRangeAggregationAggregationTwa) Bucketduration(bucketduration int64) TsRangeAggregationBucketduration {
	c.cs.s = append(c.cs.s, strconv.FormatInt(bucketduration, 10))
	return (TsRangeAggregationBucketduration)(c)
}

type TsRangeAggregationAggregationVarP Completed

func (c TsRangeAggregationAggregationVarP) Bucketduration(bucketduration int64) TsRangeAggregationBucketduration {
	c.cs.s = append(c.cs.s, strconv.FormatInt(bucketduration, 10))
	return (TsRangeAggregationBucketduration)(c)
}

type TsRangeAggregationAggregationVarS Completed

func (c TsRangeAggregationAggregationVarS) Bucketduration(bucketduration int64) TsRangeAggregationBucketduration {
	c.cs.s = append(c.cs.s, strconv.FormatInt(bucketduration, 10))
	return (TsRangeAggregationBucketduration)(c)
}

type TsRangeAggregationBucketduration Completed

func (c TsRangeAggregationBucketduration) Buckettimestamp(buckettimestamp string) TsRangeAggregationBuckettimestamp {
	c.cs.s = append(c.cs.s, "BUCKETTIMESTAMP", buckettimestamp)
	return (TsRangeAggregationBuckettimestamp)(c)
}

func (c TsRangeAggregationBucketduration) Empty() TsRangeAggregationEmpty {
	c.cs.s = append(c.cs.s, "EMPTY")
	return (TsRangeAggregationEmpty)(c)
}

func (c TsRangeAggregationBucketduration) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type TsRangeAggregationBuckettimestamp Completed

func (c TsRangeAggregationBuckettimestamp) Empty() TsRangeAggregationEmpty {
	c.cs.s = append(c.cs.s, "EMPTY")
	return (TsRangeAggregationEmpty)(c)
}

func (c TsRangeAggregationBuckettimestamp) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type TsRangeAggregationEmpty Completed

func (c TsRangeAggregationEmpty) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type TsRangeAlign Completed

func (c TsRangeAlign) AggregationAvg() TsRangeAggregationAggregationAvg {
	c.cs.s = append(c.cs.s, "AGGREGATION", "AVG")
	return (TsRangeAggregationAggregationAvg)(c)
}

func (c TsRangeAlign) AggregationSum() TsRangeAggregationAggregationSum {
	c.cs.s = append(c.cs.s, "AGGREGATION", "SUM")
	return (TsRangeAggregationAggregationSum)(c)
}

func (c TsRangeAlign) AggregationMin() TsRangeAggregationAggregationMin {
	c.cs.s = append(c.cs.s, "AGGREGATION", "MIN")
	return (TsRangeAggregationAggregationMin)(c)
}

func (c TsRangeAlign) AggregationMax() TsRangeAggregationAggregationMax {
	c.cs.s = append(c.cs.s, "AGGREGATION", "MAX")
	return (TsRangeAggregationAggregationMax)(c)
}

func (c TsRangeAlign) AggregationRange() TsRangeAggregationAggregationRange {
	c.cs.s = append(c.cs.s, "AGGREGATION", "RANGE")
	return (TsRangeAggregationAggregationRange)(c)
}

func (c TsRangeAlign) AggregationCount() TsRangeAggregationAggregationCount {
	c.cs.s = append(c.cs.s, "AGGREGATION", "COUNT")
	return (TsRangeAggregationAggregationCount)(c)
}

func (c TsRangeAlign) AggregationFirst() TsRangeAggregationAggregationFirst {
	c.cs.s = append(c.cs.s, "AGGREGATION", "FIRST")
	return (TsRangeAggregationAggregationFirst)(c)
}

func (c TsRangeAlign) AggregationLast() TsRangeAggregationAggregationLast {
	c.cs.s = append(c.cs.s, "AGGREGATION", "LAST")
	return (TsRangeAggregationAggregationLast)(c)
}

func (c TsRangeAlign) AggregationStdP() TsRangeAggregationAggregationStdP {
	c.cs.s = append(c.cs.s, "AGGREGATION", "STD.P")
	return (TsRangeAggregationAggregationStdP)(c)
}

func (c TsRangeAlign) AggregationStdS() TsRangeAggregationAggregationStdS {
	c.cs.s = append(c.cs.s, "AGGREGATION", "STD.S")
	return (TsRangeAggregationAggregationStdS)(c)
}

func (c TsRangeAlign) AggregationVarP() TsRangeAggregationAggregationVarP {
	c.cs.s = append(c.cs.s, "AGGREGATION", "VAR.P")
	return (TsRangeAggregationAggregationVarP)(c)
}

func (c TsRangeAlign) AggregationVarS() TsRangeAggregationAggregationVarS {
	c.cs.s = append(c.cs.s, "AGGREGATION", "VAR.S")
	return (TsRangeAggregationAggregationVarS)(c)
}

func (c TsRangeAlign) AggregationTwa() TsRangeAggregationAggregationTwa {
	c.cs.s = append(c.cs.s, "AGGREGATION", "TWA")
	return (TsRangeAggregationAggregationTwa)(c)
}

func (c TsRangeAlign) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type TsRangeCount Completed

func (c TsRangeCount) Align(value string) TsRangeAlign {
	c.cs.s = append(c.cs.s, "ALIGN", value)
	return (TsRangeAlign)(c)
}

func (c TsRangeCount) AggregationAvg() TsRangeAggregationAggregationAvg {
	c.cs.s = append(c.cs.s, "AGGREGATION", "AVG")
	return (TsRangeAggregationAggregationAvg)(c)
}

func (c TsRangeCount) AggregationSum() TsRangeAggregationAggregationSum {
	c.cs.s = append(c.cs.s, "AGGREGATION", "SUM")
	return (TsRangeAggregationAggregationSum)(c)
}

func (c TsRangeCount) AggregationMin() TsRangeAggregationAggregationMin {
	c.cs.s = append(c.cs.s, "AGGREGATION", "MIN")
	return (TsRangeAggregationAggregationMin)(c)
}

func (c TsRangeCount) AggregationMax() TsRangeAggregationAggregationMax {
	c.cs.s = append(c.cs.s, "AGGREGATION", "MAX")
	return (TsRangeAggregationAggregationMax)(c)
}

func (c TsRangeCount) AggregationRange() TsRangeAggregationAggregationRange {
	c.cs.s = append(c.cs.s, "AGGREGATION", "RANGE")
	return (TsRangeAggregationAggregationRange)(c)
}

func (c TsRangeCount) AggregationCount() TsRangeAggregationAggregationCount {
	c.cs.s = append(c.cs.s, "AGGREGATION", "COUNT")
	return (TsRangeAggregationAggregationCount)(c)
}

func (c TsRangeCount) AggregationFirst() TsRangeAggregationAggregationFirst {
	c.cs.s = append(c.cs.s, "AGGREGATION", "FIRST")
	return (TsRangeAggregationAggregationFirst)(c)
}

func (c TsRangeCount) AggregationLast() TsRangeAggregationAggregationLast {
	c.cs.s = append(c.cs.s, "AGGREGATION", "LAST")
	return (TsRangeAggregationAggregationLast)(c)
}

func (c TsRangeCount) AggregationStdP() TsRangeAggregationAggregationStdP {
	c.cs.s = append(c.cs.s, "AGGREGATION", "STD.P")
	return (TsRangeAggregationAggregationStdP)(c)
}

func (c TsRangeCount) AggregationStdS() TsRangeAggregationAggregationStdS {
	c.cs.s = append(c.cs.s, "AGGREGATION", "STD.S")
	return (TsRangeAggregationAggregationStdS)(c)
}

func (c TsRangeCount) AggregationVarP() TsRangeAggregationAggregationVarP {
	c.cs.s = append(c.cs.s, "AGGREGATION", "VAR.P")
	return (TsRangeAggregationAggregationVarP)(c)
}

func (c TsRangeCount) AggregationVarS() TsRangeAggregationAggregationVarS {
	c.cs.s = append(c.cs.s, "AGGREGATION", "VAR.S")
	return (TsRangeAggregationAggregationVarS)(c)
}

func (c TsRangeCount) AggregationTwa() TsRangeAggregationAggregationTwa {
	c.cs.s = append(c.cs.s, "AGGREGATION", "TWA")
	return (TsRangeAggregationAggregationTwa)(c)
}

func (c TsRangeCount) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type TsRangeFilterByTs Completed

func (c TsRangeFilterByTs) FilterByTs(timestamp ...int64) TsRangeFilterByTs {
	c.cs.s = append(c.cs.s, "FILTER_BY_TS")
	for _, n := range timestamp {
		c.cs.s = append(c.cs.s, strconv.FormatInt(n, 10))
	}
	return c
}

func (c TsRangeFilterByTs) FilterByValue(min float64, max float64) TsRangeFilterByValue {
	c.cs.s = append(c.cs.s, "FILTER_BY_VALUE", strconv.FormatFloat(min, 'f', -1, 64), strconv.FormatFloat(max, 'f', -1, 64))
	return (TsRangeFilterByValue)(c)
}

func (c TsRangeFilterByTs) Count(count int64) TsRangeCount {
	c.cs.s = append(c.cs.s, "COUNT", strconv.FormatInt(count, 10))
	return (TsRangeCount)(c)
}

func (c TsRangeFilterByTs) Align(value string) TsRangeAlign {
	c.cs.s = append(c.cs.s, "ALIGN", value)
	return (TsRangeAlign)(c)
}

func (c TsRangeFilterByTs) AggregationAvg() TsRangeAggregationAggregationAvg {
	c.cs.s = append(c.cs.s, "AGGREGATION", "AVG")
	return (TsRangeAggregationAggregationAvg)(c)
}

func (c TsRangeFilterByTs) AggregationSum() TsRangeAggregationAggregationSum {
	c.cs.s = append(c.cs.s, "AGGREGATION", "SUM")
	return (TsRangeAggregationAggregationSum)(c)
}

func (c TsRangeFilterByTs) AggregationMin() TsRangeAggregationAggregationMin {
	c.cs.s = append(c.cs.s, "AGGREGATION", "MIN")
	return (TsRangeAggregationAggregationMin)(c)
}

func (c TsRangeFilterByTs) AggregationMax() TsRangeAggregationAggregationMax {
	c.cs.s = append(c.cs.s, "AGGREGATION", "MAX")
	return (TsRangeAggregationAggregationMax)(c)
}

func (c TsRangeFilterByTs) AggregationRange() TsRangeAggregationAggregationRange {
	c.cs.s = append(c.cs.s, "AGGREGATION", "RANGE")
	return (TsRangeAggregationAggregationRange)(c)
}

func (c TsRangeFilterByTs) AggregationCount() TsRangeAggregationAggregationCount {
	c.cs.s = append(c.cs.s, "AGGREGATION", "COUNT")
	return (TsRangeAggregationAggregationCount)(c)
}

func (c TsRangeFilterByTs) AggregationFirst() TsRangeAggregationAggregationFirst {
	c.cs.s = append(c.cs.s, "AGGREGATION", "FIRST")
	return (TsRangeAggregationAggregationFirst)(c)
}

func (c TsRangeFilterByTs) AggregationLast() TsRangeAggregationAggregationLast {
	c.cs.s = append(c.cs.s, "AGGREGATION", "LAST")
	return (TsRangeAggregationAggregationLast)(c)
}

func (c TsRangeFilterByTs) AggregationStdP() TsRangeAggregationAggregationStdP {
	c.cs.s = append(c.cs.s, "AGGREGATION", "STD.P")
	return (TsRangeAggregationAggregationStdP)(c)
}

func (c TsRangeFilterByTs) AggregationStdS() TsRangeAggregationAggregationStdS {
	c.cs.s = append(c.cs.s, "AGGREGATION", "STD.S")
	return (TsRangeAggregationAggregationStdS)(c)
}

func (c TsRangeFilterByTs) AggregationVarP() TsRangeAggregationAggregationVarP {
	c.cs.s = append(c.cs.s, "AGGREGATION", "VAR.P")
	return (TsRangeAggregationAggregationVarP)(c)
}

func (c TsRangeFilterByTs) AggregationVarS() TsRangeAggregationAggregationVarS {
	c.cs.s = append(c.cs.s, "AGGREGATION", "VAR.S")
	return (TsRangeAggregationAggregationVarS)(c)
}

func (c TsRangeFilterByTs) AggregationTwa() TsRangeAggregationAggregationTwa {
	c.cs.s = append(c.cs.s, "AGGREGATION", "TWA")
	return (TsRangeAggregationAggregationTwa)(c)
}

func (c TsRangeFilterByTs) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type TsRangeFilterByValue Completed

func (c TsRangeFilterByValue) Count(count int64) TsRangeCount {
	c.cs.s = append(c.cs.s, "COUNT", strconv.FormatInt(count, 10))
	return (TsRangeCount)(c)
}

func (c TsRangeFilterByValue) Align(value string) TsRangeAlign {
	c.cs.s = append(c.cs.s, "ALIGN", value)
	return (TsRangeAlign)(c)
}

func (c TsRangeFilterByValue) AggregationAvg() TsRangeAggregationAggregationAvg {
	c.cs.s = append(c.cs.s, "AGGREGATION", "AVG")
	return (TsRangeAggregationAggregationAvg)(c)
}

func (c TsRangeFilterByValue) AggregationSum() TsRangeAggregationAggregationSum {
	c.cs.s = append(c.cs.s, "AGGREGATION", "SUM")
	return (TsRangeAggregationAggregationSum)(c)
}

func (c TsRangeFilterByValue) AggregationMin() TsRangeAggregationAggregationMin {
	c.cs.s = append(c.cs.s, "AGGREGATION", "MIN")
	return (TsRangeAggregationAggregationMin)(c)
}

func (c TsRangeFilterByValue) AggregationMax() TsRangeAggregationAggregationMax {
	c.cs.s = append(c.cs.s, "AGGREGATION", "MAX")
	return (TsRangeAggregationAggregationMax)(c)
}

func (c TsRangeFilterByValue) AggregationRange() TsRangeAggregationAggregationRange {
	c.cs.s = append(c.cs.s, "AGGREGATION", "RANGE")
	return (TsRangeAggregationAggregationRange)(c)
}

func (c TsRangeFilterByValue) AggregationCount() TsRangeAggregationAggregationCount {
	c.cs.s = append(c.cs.s, "AGGREGATION", "COUNT")
	return (TsRangeAggregationAggregationCount)(c)
}

func (c TsRangeFilterByValue) AggregationFirst() TsRangeAggregationAggregationFirst {
	c.cs.s = append(c.cs.s, "AGGREGATION", "FIRST")
	return (TsRangeAggregationAggregationFirst)(c)
}

func (c TsRangeFilterByValue) AggregationLast() TsRangeAggregationAggregationLast {
	c.cs.s = append(c.cs.s, "AGGREGATION", "LAST")
	return (TsRangeAggregationAggregationLast)(c)
}

func (c TsRangeFilterByValue) AggregationStdP() TsRangeAggregationAggregationStdP {
	c.cs.s = append(c.cs.s, "AGGREGATION", "STD.P")
	return (TsRangeAggregationAggregationStdP)(c)
}

func (c TsRangeFilterByValue) AggregationStdS() TsRangeAggregationAggregationStdS {
	c.cs.s = append(c.cs.s, "AGGREGATION", "STD.S")
	return (TsRangeAggregationAggregationStdS)(c)
}

func (c TsRangeFilterByValue) AggregationVarP() TsRangeAggregationAggregationVarP {
	c.cs.s = append(c.cs.s, "AGGREGATION", "VAR.P")
	return (TsRangeAggregationAggregationVarP)(c)
}

func (c TsRangeFilterByValue) AggregationVarS() TsRangeAggregationAggregationVarS {
	c.cs.s = append(c.cs.s, "AGGREGATION", "VAR.S")
	return (TsRangeAggregationAggregationVarS)(c)
}

func (c TsRangeFilterByValue) AggregationTwa() TsRangeAggregationAggregationTwa {
	c.cs.s = append(c.cs.s, "AGGREGATION", "TWA")
	return (TsRangeAggregationAggregationTwa)(c)
}

func (c TsRangeFilterByValue) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type TsRangeFromtimestamp Completed

func (c TsRangeFromtimestamp) Totimestamp(totimestamp int64) TsRangeTotimestamp {
	c.cs.s = append(c.cs.s, strconv.FormatInt(totimestamp, 10))
	return (TsRangeTotimestamp)(c)
}

type TsRangeKey Completed

func (c TsRangeKey) Fromtimestamp(fromtimestamp int64) TsRangeFromtimestamp {
	c.cs.s = append(c.cs.s, strconv.FormatInt(fromtimestamp, 10))
	return (TsRangeFromtimestamp)(c)
}

type TsRangeLatest Completed

func (c TsRangeLatest) FilterByTs(timestamp ...int64) TsRangeFilterByTs {
	c.cs.s = append(c.cs.s, "FILTER_BY_TS")
	for _, n := range timestamp {
		c.cs.s = append(c.cs.s, strconv.FormatInt(n, 10))
	}
	return (TsRangeFilterByTs)(c)
}

func (c TsRangeLatest) FilterByValue(min float64, max float64) TsRangeFilterByValue {
	c.cs.s = append(c.cs.s, "FILTER_BY_VALUE", strconv.FormatFloat(min, 'f', -1, 64), strconv.FormatFloat(max, 'f', -1, 64))
	return (TsRangeFilterByValue)(c)
}

func (c TsRangeLatest) Count(count int64) TsRangeCount {
	c.cs.s = append(c.cs.s, "COUNT", strconv.FormatInt(count, 10))
	return (TsRangeCount)(c)
}

func (c TsRangeLatest) Align(value string) TsRangeAlign {
	c.cs.s = append(c.cs.s, "ALIGN", value)
	return (TsRangeAlign)(c)
}

func (c TsRangeLatest) AggregationAvg() TsRangeAggregationAggregationAvg {
	c.cs.s = append(c.cs.s, "AGGREGATION", "AVG")
	return (TsRangeAggregationAggregationAvg)(c)
}

func (c TsRangeLatest) AggregationSum() TsRangeAggregationAggregationSum {
	c.cs.s = append(c.cs.s, "AGGREGATION", "SUM")
	return (TsRangeAggregationAggregationSum)(c)
}

func (c TsRangeLatest) AggregationMin() TsRangeAggregationAggregationMin {
	c.cs.s = append(c.cs.s, "AGGREGATION", "MIN")
	return (TsRangeAggregationAggregationMin)(c)
}

func (c TsRangeLatest) AggregationMax() TsRangeAggregationAggregationMax {
	c.cs.s = append(c.cs.s, "AGGREGATION", "MAX")
	return (TsRangeAggregationAggregationMax)(c)
}

func (c TsRangeLatest) AggregationRange() TsRangeAggregationAggregationRange {
	c.cs.s = append(c.cs.s, "AGGREGATION", "RANGE")
	return (TsRangeAggregationAggregationRange)(c)
}

func (c TsRangeLatest) AggregationCount() TsRangeAggregationAggregationCount {
	c.cs.s = append(c.cs.s, "AGGREGATION", "COUNT")
	return (TsRangeAggregationAggregationCount)(c)
}

func (c TsRangeLatest) AggregationFirst() TsRangeAggregationAggregationFirst {
	c.cs.s = append(c.cs.s, "AGGREGATION", "FIRST")
	return (TsRangeAggregationAggregationFirst)(c)
}

func (c TsRangeLatest) AggregationLast() TsRangeAggregationAggregationLast {
	c.cs.s = append(c.cs.s, "AGGREGATION", "LAST")
	return (TsRangeAggregationAggregationLast)(c)
}

func (c TsRangeLatest) AggregationStdP() TsRangeAggregationAggregationStdP {
	c.cs.s = append(c.cs.s, "AGGREGATION", "STD.P")
	return (TsRangeAggregationAggregationStdP)(c)
}

func (c TsRangeLatest) AggregationStdS() TsRangeAggregationAggregationStdS {
	c.cs.s = append(c.cs.s, "AGGREGATION", "STD.S")
	return (TsRangeAggregationAggregationStdS)(c)
}

func (c TsRangeLatest) AggregationVarP() TsRangeAggregationAggregationVarP {
	c.cs.s = append(c.cs.s, "AGGREGATION", "VAR.P")
	return (TsRangeAggregationAggregationVarP)(c)
}

func (c TsRangeLatest) AggregationVarS() TsRangeAggregationAggregationVarS {
	c.cs.s = append(c.cs.s, "AGGREGATION", "VAR.S")
	return (TsRangeAggregationAggregationVarS)(c)
}

func (c TsRangeLatest) AggregationTwa() TsRangeAggregationAggregationTwa {
	c.cs.s = append(c.cs.s, "AGGREGATION", "TWA")
	return (TsRangeAggregationAggregationTwa)(c)
}

func (c TsRangeLatest) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type TsRangeTotimestamp Completed

func (c TsRangeTotimestamp) Latest() TsRangeLatest {
	c.cs.s = append(c.cs.s, "LATEST")
	return (TsRangeLatest)(c)
}

func (c TsRangeTotimestamp) FilterByTs(timestamp ...int64) TsRangeFilterByTs {
	c.cs.s = append(c.cs.s, "FILTER_BY_TS")
	for _, n := range timestamp {
		c.cs.s = append(c.cs.s, strconv.FormatInt(n, 10))
	}
	return (TsRangeFilterByTs)(c)
}

func (c TsRangeTotimestamp) FilterByValue(min float64, max float64) TsRangeFilterByValue {
	c.cs.s = append(c.cs.s, "FILTER_BY_VALUE", strconv.FormatFloat(min, 'f', -1, 64), strconv.FormatFloat(max, 'f', -1, 64))
	return (TsRangeFilterByValue)(c)
}

func (c TsRangeTotimestamp) Count(count int64) TsRangeCount {
	c.cs.s = append(c.cs.s, "COUNT", strconv.FormatInt(count, 10))
	return (TsRangeCount)(c)
}

func (c TsRangeTotimestamp) Align(value string) TsRangeAlign {
	c.cs.s = append(c.cs.s, "ALIGN", value)
	return (TsRangeAlign)(c)
}

func (c TsRangeTotimestamp) AggregationAvg() TsRangeAggregationAggregationAvg {
	c.cs.s = append(c.cs.s, "AGGREGATION", "AVG")
	return (TsRangeAggregationAggregationAvg)(c)
}

func (c TsRangeTotimestamp) AggregationSum() TsRangeAggregationAggregationSum {
	c.cs.s = append(c.cs.s, "AGGREGATION", "SUM")
	return (TsRangeAggregationAggregationSum)(c)
}

func (c TsRangeTotimestamp) AggregationMin() TsRangeAggregationAggregationMin {
	c.cs.s = append(c.cs.s, "AGGREGATION", "MIN")
	return (TsRangeAggregationAggregationMin)(c)
}

func (c TsRangeTotimestamp) AggregationMax() TsRangeAggregationAggregationMax {
	c.cs.s = append(c.cs.s, "AGGREGATION", "MAX")
	return (TsRangeAggregationAggregationMax)(c)
}

func (c TsRangeTotimestamp) AggregationRange() TsRangeAggregationAggregationRange {
	c.cs.s = append(c.cs.s, "AGGREGATION", "RANGE")
	return (TsRangeAggregationAggregationRange)(c)
}

func (c TsRangeTotimestamp) AggregationCount() TsRangeAggregationAggregationCount {
	c.cs.s = append(c.cs.s, "AGGREGATION", "COUNT")
	return (TsRangeAggregationAggregationCount)(c)
}

func (c TsRangeTotimestamp) AggregationFirst() TsRangeAggregationAggregationFirst {
	c.cs.s = append(c.cs.s, "AGGREGATION", "FIRST")
	return (TsRangeAggregationAggregationFirst)(c)
}

func (c TsRangeTotimestamp) AggregationLast() TsRangeAggregationAggregationLast {
	c.cs.s = append(c.cs.s, "AGGREGATION", "LAST")
	return (TsRangeAggregationAggregationLast)(c)
}

func (c TsRangeTotimestamp) AggregationStdP() TsRangeAggregationAggregationStdP {
	c.cs.s = append(c.cs.s, "AGGREGATION", "STD.P")
	return (TsRangeAggregationAggregationStdP)(c)
}

func (c TsRangeTotimestamp) AggregationStdS() TsRangeAggregationAggregationStdS {
	c.cs.s = append(c.cs.s, "AGGREGATION", "STD.S")
	return (TsRangeAggregationAggregationStdS)(c)
}

func (c TsRangeTotimestamp) AggregationVarP() TsRangeAggregationAggregationVarP {
	c.cs.s = append(c.cs.s, "AGGREGATION", "VAR.P")
	return (TsRangeAggregationAggregationVarP)(c)
}

func (c TsRangeTotimestamp) AggregationVarS() TsRangeAggregationAggregationVarS {
	c.cs.s = append(c.cs.s, "AGGREGATION", "VAR.S")
	return (TsRangeAggregationAggregationVarS)(c)
}

func (c TsRangeTotimestamp) AggregationTwa() TsRangeAggregationAggregationTwa {
	c.cs.s = append(c.cs.s, "AGGREGATION", "TWA")
	return (TsRangeAggregationAggregationTwa)(c)
}

func (c TsRangeTotimestamp) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type TsRevrange Completed

func (b Builder) TsRevrange() (c TsRevrange) {
	c = TsRevrange{cs: get(), ks: b.ks, cf: readonly}
	c.cs.s = append(c.cs.s, "TS.REVRANGE")
	return c
}

func (c TsRevrange) Key(key string) TsRevrangeKey {
	if c.ks&NoSlot == NoSlot {
		c.ks = NoSlot | slot(key)
	} else {
		c.ks = check(c.ks, slot(key))
	}
	c.cs.s = append(c.cs.s, key)
	return (TsRevrangeKey)(c)
}

type TsRevrangeAggregationAggregationAvg Completed

func (c TsRevrangeAggregationAggregationAvg) Bucketduration(bucketduration int64) TsRevrangeAggregationBucketduration {
	c.cs.s = append(c.cs.s, strconv.FormatInt(bucketduration, 10))
	return (TsRevrangeAggregationBucketduration)(c)
}

type TsRevrangeAggregationAggregationCount Completed

func (c TsRevrangeAggregationAggregationCount) Bucketduration(bucketduration int64) TsRevrangeAggregationBucketduration {
	c.cs.s = append(c.cs.s, strconv.FormatInt(bucketduration, 10))
	return (TsRevrangeAggregationBucketduration)(c)
}

type TsRevrangeAggregationAggregationFirst Completed

func (c TsRevrangeAggregationAggregationFirst) Bucketduration(bucketduration int64) TsRevrangeAggregationBucketduration {
	c.cs.s = append(c.cs.s, strconv.FormatInt(bucketduration, 10))
	return (TsRevrangeAggregationBucketduration)(c)
}

type TsRevrangeAggregationAggregationLast Completed

func (c TsRevrangeAggregationAggregationLast) Bucketduration(bucketduration int64) TsRevrangeAggregationBucketduration {
	c.cs.s = append(c.cs.s, strconv.FormatInt(bucketduration, 10))
	return (TsRevrangeAggregationBucketduration)(c)
}

type TsRevrangeAggregationAggregationMax Completed

func (c TsRevrangeAggregationAggregationMax) Bucketduration(bucketduration int64) TsRevrangeAggregationBucketduration {
	c.cs.s = append(c.cs.s, strconv.FormatInt(bucketduration, 10))
	return (TsRevrangeAggregationBucketduration)(c)
}

type TsRevrangeAggregationAggregationMin Completed

func (c TsRevrangeAggregationAggregationMin) Bucketduration(bucketduration int64) TsRevrangeAggregationBucketduration {
	c.cs.s = append(c.cs.s, strconv.FormatInt(bucketduration, 10))
	return (TsRevrangeAggregationBucketduration)(c)
}

type TsRevrangeAggregationAggregationRange Completed

func (c TsRevrangeAggregationAggregationRange) Bucketduration(bucketduration int64) TsRevrangeAggregationBucketduration {
	c.cs.s = append(c.cs.s, strconv.FormatInt(bucketduration, 10))
	return (TsRevrangeAggregationBucketduration)(c)
}

type TsRevrangeAggregationAggregationStdP Completed

func (c TsRevrangeAggregationAggregationStdP) Bucketduration(bucketduration int64) TsRevrangeAggregationBucketduration {
	c.cs.s = append(c.cs.s, strconv.FormatInt(bucketduration, 10))
	return (TsRevrangeAggregationBucketduration)(c)
}

type TsRevrangeAggregationAggregationStdS Completed

func (c TsRevrangeAggregationAggregationStdS) Bucketduration(bucketduration int64) TsRevrangeAggregationBucketduration {
	c.cs.s = append(c.cs.s, strconv.FormatInt(bucketduration, 10))
	return (TsRevrangeAggregationBucketduration)(c)
}

type TsRevrangeAggregationAggregationSum Completed

func (c TsRevrangeAggregationAggregationSum) Bucketduration(bucketduration int64) TsRevrangeAggregationBucketduration {
	c.cs.s = append(c.cs.s, strconv.FormatInt(bucketduration, 10))
	return (TsRevrangeAggregationBucketduration)(c)
}

type TsRevrangeAggregationAggregationTwa Completed

func (c TsRevrangeAggregationAggregationTwa) Bucketduration(bucketduration int64) TsRevrangeAggregationBucketduration {
	c.cs.s = append(c.cs.s, strconv.FormatInt(bucketduration, 10))
	return (TsRevrangeAggregationBucketduration)(c)
}

type TsRevrangeAggregationAggregationVarP Completed

func (c TsRevrangeAggregationAggregationVarP) Bucketduration(bucketduration int64) TsRevrangeAggregationBucketduration {
	c.cs.s = append(c.cs.s, strconv.FormatInt(bucketduration, 10))
	return (TsRevrangeAggregationBucketduration)(c)
}

type TsRevrangeAggregationAggregationVarS Completed

func (c TsRevrangeAggregationAggregationVarS) Bucketduration(bucketduration int64) TsRevrangeAggregationBucketduration {
	c.cs.s = append(c.cs.s, strconv.FormatInt(bucketduration, 10))
	return (TsRevrangeAggregationBucketduration)(c)
}

type TsRevrangeAggregationBucketduration Completed

func (c TsRevrangeAggregationBucketduration) Buckettimestamp(buckettimestamp string) TsRevrangeAggregationBuckettimestamp {
	c.cs.s = append(c.cs.s, "BUCKETTIMESTAMP", buckettimestamp)
	return (TsRevrangeAggregationBuckettimestamp)(c)
}

func (c TsRevrangeAggregationBucketduration) Empty() TsRevrangeAggregationEmpty {
	c.cs.s = append(c.cs.s, "EMPTY")
	return (TsRevrangeAggregationEmpty)(c)
}

func (c TsRevrangeAggregationBucketduration) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type TsRevrangeAggregationBuckettimestamp Completed

func (c TsRevrangeAggregationBuckettimestamp) Empty() TsRevrangeAggregationEmpty {
	c.cs.s = append(c.cs.s, "EMPTY")
	return (TsRevrangeAggregationEmpty)(c)
}

func (c TsRevrangeAggregationBuckettimestamp) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type TsRevrangeAggregationEmpty Completed

func (c TsRevrangeAggregationEmpty) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type TsRevrangeAlign Completed

func (c TsRevrangeAlign) AggregationAvg() TsRevrangeAggregationAggregationAvg {
	c.cs.s = append(c.cs.s, "AGGREGATION", "AVG")
	return (TsRevrangeAggregationAggregationAvg)(c)
}

func (c TsRevrangeAlign) AggregationSum() TsRevrangeAggregationAggregationSum {
	c.cs.s = append(c.cs.s, "AGGREGATION", "SUM")
	return (TsRevrangeAggregationAggregationSum)(c)
}

func (c TsRevrangeAlign) AggregationMin() TsRevrangeAggregationAggregationMin {
	c.cs.s = append(c.cs.s, "AGGREGATION", "MIN")
	return (TsRevrangeAggregationAggregationMin)(c)
}

func (c TsRevrangeAlign) AggregationMax() TsRevrangeAggregationAggregationMax {
	c.cs.s = append(c.cs.s, "AGGREGATION", "MAX")
	return (TsRevrangeAggregationAggregationMax)(c)
}

func (c TsRevrangeAlign) AggregationRange() TsRevrangeAggregationAggregationRange {
	c.cs.s = append(c.cs.s, "AGGREGATION", "RANGE")
	return (TsRevrangeAggregationAggregationRange)(c)
}

func (c TsRevrangeAlign) AggregationCount() TsRevrangeAggregationAggregationCount {
	c.cs.s = append(c.cs.s, "AGGREGATION", "COUNT")
	return (TsRevrangeAggregationAggregationCount)(c)
}

func (c TsRevrangeAlign) AggregationFirst() TsRevrangeAggregationAggregationFirst {
	c.cs.s = append(c.cs.s, "AGGREGATION", "FIRST")
	return (TsRevrangeAggregationAggregationFirst)(c)
}

func (c TsRevrangeAlign) AggregationLast() TsRevrangeAggregationAggregationLast {
	c.cs.s = append(c.cs.s, "AGGREGATION", "LAST")
	return (TsRevrangeAggregationAggregationLast)(c)
}

func (c TsRevrangeAlign) AggregationStdP() TsRevrangeAggregationAggregationStdP {
	c.cs.s = append(c.cs.s, "AGGREGATION", "STD.P")
	return (TsRevrangeAggregationAggregationStdP)(c)
}

func (c TsRevrangeAlign) AggregationStdS() TsRevrangeAggregationAggregationStdS {
	c.cs.s = append(c.cs.s, "AGGREGATION", "STD.S")
	return (TsRevrangeAggregationAggregationStdS)(c)
}

func (c TsRevrangeAlign) AggregationVarP() TsRevrangeAggregationAggregationVarP {
	c.cs.s = append(c.cs.s, "AGGREGATION", "VAR.P")
	return (TsRevrangeAggregationAggregationVarP)(c)
}

func (c TsRevrangeAlign) AggregationVarS() TsRevrangeAggregationAggregationVarS {
	c.cs.s = append(c.cs.s, "AGGREGATION", "VAR.S")
	return (TsRevrangeAggregationAggregationVarS)(c)
}

func (c TsRevrangeAlign) AggregationTwa() TsRevrangeAggregationAggregationTwa {
	c.cs.s = append(c.cs.s, "AGGREGATION", "TWA")
	return (TsRevrangeAggregationAggregationTwa)(c)
}

func (c TsRevrangeAlign) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type TsRevrangeCount Completed

func (c TsRevrangeCount) Align(value string) TsRevrangeAlign {
	c.cs.s = append(c.cs.s, "ALIGN", value)
	return (TsRevrangeAlign)(c)
}

func (c TsRevrangeCount) AggregationAvg() TsRevrangeAggregationAggregationAvg {
	c.cs.s = append(c.cs.s, "AGGREGATION", "AVG")
	return (TsRevrangeAggregationAggregationAvg)(c)
}

func (c TsRevrangeCount) AggregationSum() TsRevrangeAggregationAggregationSum {
	c.cs.s = append(c.cs.s, "AGGREGATION", "SUM")
	return (TsRevrangeAggregationAggregationSum)(c)
}

func (c TsRevrangeCount) AggregationMin() TsRevrangeAggregationAggregationMin {
	c.cs.s = append(c.cs.s, "AGGREGATION", "MIN")
	return (TsRevrangeAggregationAggregationMin)(c)
}

func (c TsRevrangeCount) AggregationMax() TsRevrangeAggregationAggregationMax {
	c.cs.s = append(c.cs.s, "AGGREGATION", "MAX")
	return (TsRevrangeAggregationAggregationMax)(c)
}

func (c TsRevrangeCount) AggregationRange() TsRevrangeAggregationAggregationRange {
	c.cs.s = append(c.cs.s, "AGGREGATION", "RANGE")
	return (TsRevrangeAggregationAggregationRange)(c)
}

func (c TsRevrangeCount) AggregationCount() TsRevrangeAggregationAggregationCount {
	c.cs.s = append(c.cs.s, "AGGREGATION", "COUNT")
	return (TsRevrangeAggregationAggregationCount)(c)
}

func (c TsRevrangeCount) AggregationFirst() TsRevrangeAggregationAggregationFirst {
	c.cs.s = append(c.cs.s, "AGGREGATION", "FIRST")
	return (TsRevrangeAggregationAggregationFirst)(c)
}

func (c TsRevrangeCount) AggregationLast() TsRevrangeAggregationAggregationLast {
	c.cs.s = append(c.cs.s, "AGGREGATION", "LAST")
	return (TsRevrangeAggregationAggregationLast)(c)
}

func (c TsRevrangeCount) AggregationStdP() TsRevrangeAggregationAggregationStdP {
	c.cs.s = append(c.cs.s, "AGGREGATION", "STD.P")
	return (TsRevrangeAggregationAggregationStdP)(c)
}

func (c TsRevrangeCount) AggregationStdS() TsRevrangeAggregationAggregationStdS {
	c.cs.s = append(c.cs.s, "AGGREGATION", "STD.S")
	return (TsRevrangeAggregationAggregationStdS)(c)
}

func (c TsRevrangeCount) AggregationVarP() TsRevrangeAggregationAggregationVarP {
	c.cs.s = append(c.cs.s, "AGGREGATION", "VAR.P")
	return (TsRevrangeAggregationAggregationVarP)(c)
}

func (c TsRevrangeCount) AggregationVarS() TsRevrangeAggregationAggregationVarS {
	c.cs.s = append(c.cs.s, "AGGREGATION", "VAR.S")
	return (TsRevrangeAggregationAggregationVarS)(c)
}

func (c TsRevrangeCount) AggregationTwa() TsRevrangeAggregationAggregationTwa {
	c.cs.s = append(c.cs.s, "AGGREGATION", "TWA")
	return (TsRevrangeAggregationAggregationTwa)(c)
}

func (c TsRevrangeCount) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type TsRevrangeFilterByTs Completed

func (c TsRevrangeFilterByTs) FilterByTs(timestamp ...int64) TsRevrangeFilterByTs {
	c.cs.s = append(c.cs.s, "FILTER_BY_TS")
	for _, n := range timestamp {
		c.cs.s = append(c.cs.s, strconv.FormatInt(n, 10))
	}
	return c
}

func (c TsRevrangeFilterByTs) FilterByValue(min float64, max float64) TsRevrangeFilterByValue {
	c.cs.s = append(c.cs.s, "FILTER_BY_VALUE", strconv.FormatFloat(min, 'f', -1, 64), strconv.FormatFloat(max, 'f', -1, 64))
	return (TsRevrangeFilterByValue)(c)
}

func (c TsRevrangeFilterByTs) Count(count int64) TsRevrangeCount {
	c.cs.s = append(c.cs.s, "COUNT", strconv.FormatInt(count, 10))
	return (TsRevrangeCount)(c)
}

func (c TsRevrangeFilterByTs) Align(value string) TsRevrangeAlign {
	c.cs.s = append(c.cs.s, "ALIGN", value)
	return (TsRevrangeAlign)(c)
}

func (c TsRevrangeFilterByTs) AggregationAvg() TsRevrangeAggregationAggregationAvg {
	c.cs.s = append(c.cs.s, "AGGREGATION", "AVG")
	return (TsRevrangeAggregationAggregationAvg)(c)
}

func (c TsRevrangeFilterByTs) AggregationSum() TsRevrangeAggregationAggregationSum {
	c.cs.s = append(c.cs.s, "AGGREGATION", "SUM")
	return (TsRevrangeAggregationAggregationSum)(c)
}

func (c TsRevrangeFilterByTs) AggregationMin() TsRevrangeAggregationAggregationMin {
	c.cs.s = append(c.cs.s, "AGGREGATION", "MIN")
	return (TsRevrangeAggregationAggregationMin)(c)
}

func (c TsRevrangeFilterByTs) AggregationMax() TsRevrangeAggregationAggregationMax {
	c.cs.s = append(c.cs.s, "AGGREGATION", "MAX")
	return (TsRevrangeAggregationAggregationMax)(c)
}

func (c TsRevrangeFilterByTs) AggregationRange() TsRevrangeAggregationAggregationRange {
	c.cs.s = append(c.cs.s, "AGGREGATION", "RANGE")
	return (TsRevrangeAggregationAggregationRange)(c)
}

func (c TsRevrangeFilterByTs) AggregationCount() TsRevrangeAggregationAggregationCount {
	c.cs.s = append(c.cs.s, "AGGREGATION", "COUNT")
	return (TsRevrangeAggregationAggregationCount)(c)
}

func (c TsRevrangeFilterByTs) AggregationFirst() TsRevrangeAggregationAggregationFirst {
	c.cs.s = append(c.cs.s, "AGGREGATION", "FIRST")
	return (TsRevrangeAggregationAggregationFirst)(c)
}

func (c TsRevrangeFilterByTs) AggregationLast() TsRevrangeAggregationAggregationLast {
	c.cs.s = append(c.cs.s, "AGGREGATION", "LAST")
	return (TsRevrangeAggregationAggregationLast)(c)
}

func (c TsRevrangeFilterByTs) AggregationStdP() TsRevrangeAggregationAggregationStdP {
	c.cs.s = append(c.cs.s, "AGGREGATION", "STD.P")
	return (TsRevrangeAggregationAggregationStdP)(c)
}

func (c TsRevrangeFilterByTs) AggregationStdS() TsRevrangeAggregationAggregationStdS {
	c.cs.s = append(c.cs.s, "AGGREGATION", "STD.S")
	return (TsRevrangeAggregationAggregationStdS)(c)
}

func (c TsRevrangeFilterByTs) AggregationVarP() TsRevrangeAggregationAggregationVarP {
	c.cs.s = append(c.cs.s, "AGGREGATION", "VAR.P")
	return (TsRevrangeAggregationAggregationVarP)(c)
}

func (c TsRevrangeFilterByTs) AggregationVarS() TsRevrangeAggregationAggregationVarS {
	c.cs.s = append(c.cs.s, "AGGREGATION", "VAR.S")
	return (TsRevrangeAggregationAggregationVarS)(c)
}

func (c TsRevrangeFilterByTs) AggregationTwa() TsRevrangeAggregationAggregationTwa {
	c.cs.s = append(c.cs.s, "AGGREGATION", "TWA")
	return (TsRevrangeAggregationAggregationTwa)(c)
}

func (c TsRevrangeFilterByTs) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type TsRevrangeFilterByValue Completed

func (c TsRevrangeFilterByValue) Count(count int64) TsRevrangeCount {
	c.cs.s = append(c.cs.s, "COUNT", strconv.FormatInt(count, 10))
	return (TsRevrangeCount)(c)
}

func (c TsRevrangeFilterByValue) Align(value string) TsRevrangeAlign {
	c.cs.s = append(c.cs.s, "ALIGN", value)
	return (TsRevrangeAlign)(c)
}

func (c TsRevrangeFilterByValue) AggregationAvg() TsRevrangeAggregationAggregationAvg {
	c.cs.s = append(c.cs.s, "AGGREGATION", "AVG")
	return (TsRevrangeAggregationAggregationAvg)(c)
}

func (c TsRevrangeFilterByValue) AggregationSum() TsRevrangeAggregationAggregationSum {
	c.cs.s = append(c.cs.s, "AGGREGATION", "SUM")
	return (TsRevrangeAggregationAggregationSum)(c)
}

func (c TsRevrangeFilterByValue) AggregationMin() TsRevrangeAggregationAggregationMin {
	c.cs.s = append(c.cs.s, "AGGREGATION", "MIN")
	return (TsRevrangeAggregationAggregationMin)(c)
}

func (c TsRevrangeFilterByValue) AggregationMax() TsRevrangeAggregationAggregationMax {
	c.cs.s = append(c.cs.s, "AGGREGATION", "MAX")
	return (TsRevrangeAggregationAggregationMax)(c)
}

func (c TsRevrangeFilterByValue) AggregationRange() TsRevrangeAggregationAggregationRange {
	c.cs.s = append(c.cs.s, "AGGREGATION", "RANGE")
	return (TsRevrangeAggregationAggregationRange)(c)
}

func (c TsRevrangeFilterByValue) AggregationCount() TsRevrangeAggregationAggregationCount {
	c.cs.s = append(c.cs.s, "AGGREGATION", "COUNT")
	return (TsRevrangeAggregationAggregationCount)(c)
}

func (c TsRevrangeFilterByValue) AggregationFirst() TsRevrangeAggregationAggregationFirst {
	c.cs.s = append(c.cs.s, "AGGREGATION", "FIRST")
	return (TsRevrangeAggregationAggregationFirst)(c)
}

func (c TsRevrangeFilterByValue) AggregationLast() TsRevrangeAggregationAggregationLast {
	c.cs.s = append(c.cs.s, "AGGREGATION", "LAST")
	return (TsRevrangeAggregationAggregationLast)(c)
}

func (c TsRevrangeFilterByValue) AggregationStdP() TsRevrangeAggregationAggregationStdP {
	c.cs.s = append(c.cs.s, "AGGREGATION", "STD.P")
	return (TsRevrangeAggregationAggregationStdP)(c)
}

func (c TsRevrangeFilterByValue) AggregationStdS() TsRevrangeAggregationAggregationStdS {
	c.cs.s = append(c.cs.s, "AGGREGATION", "STD.S")
	return (TsRevrangeAggregationAggregationStdS)(c)
}

func (c TsRevrangeFilterByValue) AggregationVarP() TsRevrangeAggregationAggregationVarP {
	c.cs.s = append(c.cs.s, "AGGREGATION", "VAR.P")
	return (TsRevrangeAggregationAggregationVarP)(c)
}

func (c TsRevrangeFilterByValue) AggregationVarS() TsRevrangeAggregationAggregationVarS {
	c.cs.s = append(c.cs.s, "AGGREGATION", "VAR.S")
	return (TsRevrangeAggregationAggregationVarS)(c)
}

func (c TsRevrangeFilterByValue) AggregationTwa() TsRevrangeAggregationAggregationTwa {
	c.cs.s = append(c.cs.s, "AGGREGATION", "TWA")
	return (TsRevrangeAggregationAggregationTwa)(c)
}

func (c TsRevrangeFilterByValue) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type TsRevrangeFromtimestamp Completed

func (c TsRevrangeFromtimestamp) Totimestamp(totimestamp int64) TsRevrangeTotimestamp {
	c.cs.s = append(c.cs.s, strconv.FormatInt(totimestamp, 10))
	return (TsRevrangeTotimestamp)(c)
}

type TsRevrangeKey Completed

func (c TsRevrangeKey) Fromtimestamp(fromtimestamp int64) TsRevrangeFromtimestamp {
	c.cs.s = append(c.cs.s, strconv.FormatInt(fromtimestamp, 10))
	return (TsRevrangeFromtimestamp)(c)
}

type TsRevrangeLatest Completed

func (c TsRevrangeLatest) FilterByTs(timestamp ...int64) TsRevrangeFilterByTs {
	c.cs.s = append(c.cs.s, "FILTER_BY_TS")
	for _, n := range timestamp {
		c.cs.s = append(c.cs.s, strconv.FormatInt(n, 10))
	}
	return (TsRevrangeFilterByTs)(c)
}

func (c TsRevrangeLatest) FilterByValue(min float64, max float64) TsRevrangeFilterByValue {
	c.cs.s = append(c.cs.s, "FILTER_BY_VALUE", strconv.FormatFloat(min, 'f', -1, 64), strconv.FormatFloat(max, 'f', -1, 64))
	return (TsRevrangeFilterByValue)(c)
}

func (c TsRevrangeLatest) Count(count int64) TsRevrangeCount {
	c.cs.s = append(c.cs.s, "COUNT", strconv.FormatInt(count, 10))
	return (TsRevrangeCount)(c)
}

func (c TsRevrangeLatest) Align(value string) TsRevrangeAlign {
	c.cs.s = append(c.cs.s, "ALIGN", value)
	return (TsRevrangeAlign)(c)
}

func (c TsRevrangeLatest) AggregationAvg() TsRevrangeAggregationAggregationAvg {
	c.cs.s = append(c.cs.s, "AGGREGATION", "AVG")
	return (TsRevrangeAggregationAggregationAvg)(c)
}

func (c TsRevrangeLatest) AggregationSum() TsRevrangeAggregationAggregationSum {
	c.cs.s = append(c.cs.s, "AGGREGATION", "SUM")
	return (TsRevrangeAggregationAggregationSum)(c)
}

func (c TsRevrangeLatest) AggregationMin() TsRevrangeAggregationAggregationMin {
	c.cs.s = append(c.cs.s, "AGGREGATION", "MIN")
	return (TsRevrangeAggregationAggregationMin)(c)
}

func (c TsRevrangeLatest) AggregationMax() TsRevrangeAggregationAggregationMax {
	c.cs.s = append(c.cs.s, "AGGREGATION", "MAX")
	return (TsRevrangeAggregationAggregationMax)(c)
}

func (c TsRevrangeLatest) AggregationRange() TsRevrangeAggregationAggregationRange {
	c.cs.s = append(c.cs.s, "AGGREGATION", "RANGE")
	return (TsRevrangeAggregationAggregationRange)(c)
}

func (c TsRevrangeLatest) AggregationCount() TsRevrangeAggregationAggregationCount {
	c.cs.s = append(c.cs.s, "AGGREGATION", "COUNT")
	return (TsRevrangeAggregationAggregationCount)(c)
}

func (c TsRevrangeLatest) AggregationFirst() TsRevrangeAggregationAggregationFirst {
	c.cs.s = append(c.cs.s, "AGGREGATION", "FIRST")
	return (TsRevrangeAggregationAggregationFirst)(c)
}

func (c TsRevrangeLatest) AggregationLast() TsRevrangeAggregationAggregationLast {
	c.cs.s = append(c.cs.s, "AGGREGATION", "LAST")
	return (TsRevrangeAggregationAggregationLast)(c)
}

func (c TsRevrangeLatest) AggregationStdP() TsRevrangeAggregationAggregationStdP {
	c.cs.s = append(c.cs.s, "AGGREGATION", "STD.P")
	return (TsRevrangeAggregationAggregationStdP)(c)
}

func (c TsRevrangeLatest) AggregationStdS() TsRevrangeAggregationAggregationStdS {
	c.cs.s = append(c.cs.s, "AGGREGATION", "STD.S")
	return (TsRevrangeAggregationAggregationStdS)(c)
}

func (c TsRevrangeLatest) AggregationVarP() TsRevrangeAggregationAggregationVarP {
	c.cs.s = append(c.cs.s, "AGGREGATION", "VAR.P")
	return (TsRevrangeAggregationAggregationVarP)(c)
}

func (c TsRevrangeLatest) AggregationVarS() TsRevrangeAggregationAggregationVarS {
	c.cs.s = append(c.cs.s, "AGGREGATION", "VAR.S")
	return (TsRevrangeAggregationAggregationVarS)(c)
}

func (c TsRevrangeLatest) AggregationTwa() TsRevrangeAggregationAggregationTwa {
	c.cs.s = append(c.cs.s, "AGGREGATION", "TWA")
	return (TsRevrangeAggregationAggregationTwa)(c)
}

func (c TsRevrangeLatest) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type TsRevrangeTotimestamp Completed

func (c TsRevrangeTotimestamp) Latest() TsRevrangeLatest {
	c.cs.s = append(c.cs.s, "LATEST")
	return (TsRevrangeLatest)(c)
}

func (c TsRevrangeTotimestamp) FilterByTs(timestamp ...int64) TsRevrangeFilterByTs {
	c.cs.s = append(c.cs.s, "FILTER_BY_TS")
	for _, n := range timestamp {
		c.cs.s = append(c.cs.s, strconv.FormatInt(n, 10))
	}
	return (TsRevrangeFilterByTs)(c)
}

func (c TsRevrangeTotimestamp) FilterByValue(min float64, max float64) TsRevrangeFilterByValue {
	c.cs.s = append(c.cs.s, "FILTER_BY_VALUE", strconv.FormatFloat(min, 'f', -1, 64), strconv.FormatFloat(max, 'f', -1, 64))
	return (TsRevrangeFilterByValue)(c)
}

func (c TsRevrangeTotimestamp) Count(count int64) TsRevrangeCount {
	c.cs.s = append(c.cs.s, "COUNT", strconv.FormatInt(count, 10))
	return (TsRevrangeCount)(c)
}

func (c TsRevrangeTotimestamp) Align(value string) TsRevrangeAlign {
	c.cs.s = append(c.cs.s, "ALIGN", value)
	return (TsRevrangeAlign)(c)
}

func (c TsRevrangeTotimestamp) AggregationAvg() TsRevrangeAggregationAggregationAvg {
	c.cs.s = append(c.cs.s, "AGGREGATION", "AVG")
	return (TsRevrangeAggregationAggregationAvg)(c)
}

func (c TsRevrangeTotimestamp) AggregationSum() TsRevrangeAggregationAggregationSum {
	c.cs.s = append(c.cs.s, "AGGREGATION", "SUM")
	return (TsRevrangeAggregationAggregationSum)(c)
}

func (c TsRevrangeTotimestamp) AggregationMin() TsRevrangeAggregationAggregationMin {
	c.cs.s = append(c.cs.s, "AGGREGATION", "MIN")
	return (TsRevrangeAggregationAggregationMin)(c)
}

func (c TsRevrangeTotimestamp) AggregationMax() TsRevrangeAggregationAggregationMax {
	c.cs.s = append(c.cs.s, "AGGREGATION", "MAX")
	return (TsRevrangeAggregationAggregationMax)(c)
}

func (c TsRevrangeTotimestamp) AggregationRange() TsRevrangeAggregationAggregationRange {
	c.cs.s = append(c.cs.s, "AGGREGATION", "RANGE")
	return (TsRevrangeAggregationAggregationRange)(c)
}

func (c TsRevrangeTotimestamp) AggregationCount() TsRevrangeAggregationAggregationCount {
	c.cs.s = append(c.cs.s, "AGGREGATION", "COUNT")
	return (TsRevrangeAggregationAggregationCount)(c)
}

func (c TsRevrangeTotimestamp) AggregationFirst() TsRevrangeAggregationAggregationFirst {
	c.cs.s = append(c.cs.s, "AGGREGATION", "FIRST")
	return (TsRevrangeAggregationAggregationFirst)(c)
}

func (c TsRevrangeTotimestamp) AggregationLast() TsRevrangeAggregationAggregationLast {
	c.cs.s = append(c.cs.s, "AGGREGATION", "LAST")
	return (TsRevrangeAggregationAggregationLast)(c)
}

func (c TsRevrangeTotimestamp) AggregationStdP() TsRevrangeAggregationAggregationStdP {
	c.cs.s = append(c.cs.s, "AGGREGATION", "STD.P")
	return (TsRevrangeAggregationAggregationStdP)(c)
}

func (c TsRevrangeTotimestamp) AggregationStdS() TsRevrangeAggregationAggregationStdS {
	c.cs.s = append(c.cs.s, "AGGREGATION", "STD.S")
	return (TsRevrangeAggregationAggregationStdS)(c)
}

func (c TsRevrangeTotimestamp) AggregationVarP() TsRevrangeAggregationAggregationVarP {
	c.cs.s = append(c.cs.s, "AGGREGATION", "VAR.P")
	return (TsRevrangeAggregationAggregationVarP)(c)
}

func (c TsRevrangeTotimestamp) AggregationVarS() TsRevrangeAggregationAggregationVarS {
	c.cs.s = append(c.cs.s, "AGGREGATION", "VAR.S")
	return (TsRevrangeAggregationAggregationVarS)(c)
}

func (c TsRevrangeTotimestamp) AggregationTwa() TsRevrangeAggregationAggregationTwa {
	c.cs.s = append(c.cs.s, "AGGREGATION", "TWA")
	return (TsRevrangeAggregationAggregationTwa)(c)
}

func (c TsRevrangeTotimestamp) Build() Completed {
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

type Unsubscribe Completed

func (b Builder) Unsubscribe() (c Unsubscribe) {
	c = Unsubscribe{cs: get(), ks: b.ks, cf: noRetTag}
	c.cs.s = append(c.cs.s, "UNSUBSCRIBE")
	return c
}

func (c Unsubscribe) Channel(channel ...string) UnsubscribeChannel {
	c.cs.s = append(c.cs.s, channel...)
	return (UnsubscribeChannel)(c)
}

func (c Unsubscribe) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type UnsubscribeChannel Completed

func (c UnsubscribeChannel) Channel(channel ...string) UnsubscribeChannel {
	c.cs.s = append(c.cs.s, channel...)
	return c
}

func (c UnsubscribeChannel) Build() Completed {
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

type Xack Completed

func (b Builder) Xack() (c Xack) {
	c = Xack{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "XACK")
	return c
}

func (c Xack) Key(key string) XackKey {
	if c.ks&NoSlot == NoSlot {
		c.ks = NoSlot | slot(key)
	} else {
		c.ks = check(c.ks, slot(key))
	}
	c.cs.s = append(c.cs.s, key)
	return (XackKey)(c)
}

type XackGroup Completed

func (c XackGroup) Id(id ...string) XackId {
	c.cs.s = append(c.cs.s, id...)
	return (XackId)(c)
}

type XackId Completed

func (c XackId) Id(id ...string) XackId {
	c.cs.s = append(c.cs.s, id...)
	return c
}

func (c XackId) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type XackKey Completed

func (c XackKey) Group(group string) XackGroup {
	c.cs.s = append(c.cs.s, group)
	return (XackGroup)(c)
}

type Xadd Completed

func (b Builder) Xadd() (c Xadd) {
	c = Xadd{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "XADD")
	return c
}

func (c Xadd) Key(key string) XaddKey {
	if c.ks&NoSlot == NoSlot {
		c.ks = NoSlot | slot(key)
	} else {
		c.ks = check(c.ks, slot(key))
	}
	c.cs.s = append(c.cs.s, key)
	return (XaddKey)(c)
}

type XaddFieldValue Completed

func (c XaddFieldValue) FieldValue(field string, value string) XaddFieldValue {
	c.cs.s = append(c.cs.s, field, value)
	return c
}

func (c XaddFieldValue) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type XaddId Completed

func (c XaddId) FieldValue() XaddFieldValue {
	return (XaddFieldValue)(c)
}

type XaddKey Completed

func (c XaddKey) Nomkstream() XaddNomkstream {
	c.cs.s = append(c.cs.s, "NOMKSTREAM")
	return (XaddNomkstream)(c)
}

func (c XaddKey) Maxlen() XaddTrimStrategyMaxlen {
	c.cs.s = append(c.cs.s, "MAXLEN")
	return (XaddTrimStrategyMaxlen)(c)
}

func (c XaddKey) Minid() XaddTrimStrategyMinid {
	c.cs.s = append(c.cs.s, "MINID")
	return (XaddTrimStrategyMinid)(c)
}

func (c XaddKey) Id(id string) XaddId {
	c.cs.s = append(c.cs.s, id)
	return (XaddId)(c)
}

type XaddNomkstream Completed

func (c XaddNomkstream) Maxlen() XaddTrimStrategyMaxlen {
	c.cs.s = append(c.cs.s, "MAXLEN")
	return (XaddTrimStrategyMaxlen)(c)
}

func (c XaddNomkstream) Minid() XaddTrimStrategyMinid {
	c.cs.s = append(c.cs.s, "MINID")
	return (XaddTrimStrategyMinid)(c)
}

func (c XaddNomkstream) Id(id string) XaddId {
	c.cs.s = append(c.cs.s, id)
	return (XaddId)(c)
}

type XaddTrimLimit Completed

func (c XaddTrimLimit) Id(id string) XaddId {
	c.cs.s = append(c.cs.s, id)
	return (XaddId)(c)
}

type XaddTrimOperatorAlmost Completed

func (c XaddTrimOperatorAlmost) Threshold(threshold string) XaddTrimThreshold {
	c.cs.s = append(c.cs.s, threshold)
	return (XaddTrimThreshold)(c)
}

type XaddTrimOperatorExact Completed

func (c XaddTrimOperatorExact) Threshold(threshold string) XaddTrimThreshold {
	c.cs.s = append(c.cs.s, threshold)
	return (XaddTrimThreshold)(c)
}

type XaddTrimStrategyMaxlen Completed

func (c XaddTrimStrategyMaxlen) Exact() XaddTrimOperatorExact {
	c.cs.s = append(c.cs.s, "=")
	return (XaddTrimOperatorExact)(c)
}

func (c XaddTrimStrategyMaxlen) Almost() XaddTrimOperatorAlmost {
	c.cs.s = append(c.cs.s, "~")
	return (XaddTrimOperatorAlmost)(c)
}

func (c XaddTrimStrategyMaxlen) Threshold(threshold string) XaddTrimThreshold {
	c.cs.s = append(c.cs.s, threshold)
	return (XaddTrimThreshold)(c)
}

type XaddTrimStrategyMinid Completed

func (c XaddTrimStrategyMinid) Exact() XaddTrimOperatorExact {
	c.cs.s = append(c.cs.s, "=")
	return (XaddTrimOperatorExact)(c)
}

func (c XaddTrimStrategyMinid) Almost() XaddTrimOperatorAlmost {
	c.cs.s = append(c.cs.s, "~")
	return (XaddTrimOperatorAlmost)(c)
}

func (c XaddTrimStrategyMinid) Threshold(threshold string) XaddTrimThreshold {
	c.cs.s = append(c.cs.s, threshold)
	return (XaddTrimThreshold)(c)
}

type XaddTrimThreshold Completed

func (c XaddTrimThreshold) Limit(count int64) XaddTrimLimit {
	c.cs.s = append(c.cs.s, "LIMIT", strconv.FormatInt(count, 10))
	return (XaddTrimLimit)(c)
}

func (c XaddTrimThreshold) Id(id string) XaddId {
	c.cs.s = append(c.cs.s, id)
	return (XaddId)(c)
}

type Xautoclaim Completed

func (b Builder) Xautoclaim() (c Xautoclaim) {
	c = Xautoclaim{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "XAUTOCLAIM")
	return c
}

func (c Xautoclaim) Key(key string) XautoclaimKey {
	if c.ks&NoSlot == NoSlot {
		c.ks = NoSlot | slot(key)
	} else {
		c.ks = check(c.ks, slot(key))
	}
	c.cs.s = append(c.cs.s, key)
	return (XautoclaimKey)(c)
}

type XautoclaimConsumer Completed

func (c XautoclaimConsumer) MinIdleTime(minIdleTime string) XautoclaimMinIdleTime {
	c.cs.s = append(c.cs.s, minIdleTime)
	return (XautoclaimMinIdleTime)(c)
}

type XautoclaimCount Completed

func (c XautoclaimCount) Justid() XautoclaimJustid {
	c.cs.s = append(c.cs.s, "JUSTID")
	return (XautoclaimJustid)(c)
}

func (c XautoclaimCount) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type XautoclaimGroup Completed

func (c XautoclaimGroup) Consumer(consumer string) XautoclaimConsumer {
	c.cs.s = append(c.cs.s, consumer)
	return (XautoclaimConsumer)(c)
}

type XautoclaimJustid Completed

func (c XautoclaimJustid) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type XautoclaimKey Completed

func (c XautoclaimKey) Group(group string) XautoclaimGroup {
	c.cs.s = append(c.cs.s, group)
	return (XautoclaimGroup)(c)
}

type XautoclaimMinIdleTime Completed

func (c XautoclaimMinIdleTime) Start(start string) XautoclaimStart {
	c.cs.s = append(c.cs.s, start)
	return (XautoclaimStart)(c)
}

type XautoclaimStart Completed

func (c XautoclaimStart) Count(count int64) XautoclaimCount {
	c.cs.s = append(c.cs.s, "COUNT", strconv.FormatInt(count, 10))
	return (XautoclaimCount)(c)
}

func (c XautoclaimStart) Justid() XautoclaimJustid {
	c.cs.s = append(c.cs.s, "JUSTID")
	return (XautoclaimJustid)(c)
}

func (c XautoclaimStart) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type Xclaim Completed

func (b Builder) Xclaim() (c Xclaim) {
	c = Xclaim{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "XCLAIM")
	return c
}

func (c Xclaim) Key(key string) XclaimKey {
	if c.ks&NoSlot == NoSlot {
		c.ks = NoSlot | slot(key)
	} else {
		c.ks = check(c.ks, slot(key))
	}
	c.cs.s = append(c.cs.s, key)
	return (XclaimKey)(c)
}

type XclaimConsumer Completed

func (c XclaimConsumer) MinIdleTime(minIdleTime string) XclaimMinIdleTime {
	c.cs.s = append(c.cs.s, minIdleTime)
	return (XclaimMinIdleTime)(c)
}

type XclaimForce Completed

func (c XclaimForce) Justid() XclaimJustid {
	c.cs.s = append(c.cs.s, "JUSTID")
	return (XclaimJustid)(c)
}

func (c XclaimForce) Lastid() XclaimLastid {
	c.cs.s = append(c.cs.s, "LASTID")
	return (XclaimLastid)(c)
}

func (c XclaimForce) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type XclaimGroup Completed

func (c XclaimGroup) Consumer(consumer string) XclaimConsumer {
	c.cs.s = append(c.cs.s, consumer)
	return (XclaimConsumer)(c)
}

type XclaimId Completed

func (c XclaimId) Id(id ...string) XclaimId {
	c.cs.s = append(c.cs.s, id...)
	return c
}

func (c XclaimId) Idle(ms int64) XclaimIdle {
	c.cs.s = append(c.cs.s, "IDLE", strconv.FormatInt(ms, 10))
	return (XclaimIdle)(c)
}

func (c XclaimId) Time(msUnixTime int64) XclaimTime {
	c.cs.s = append(c.cs.s, "TIME", strconv.FormatInt(msUnixTime, 10))
	return (XclaimTime)(c)
}

func (c XclaimId) Retrycount(count int64) XclaimRetrycount {
	c.cs.s = append(c.cs.s, "RETRYCOUNT", strconv.FormatInt(count, 10))
	return (XclaimRetrycount)(c)
}

func (c XclaimId) Force() XclaimForce {
	c.cs.s = append(c.cs.s, "FORCE")
	return (XclaimForce)(c)
}

func (c XclaimId) Justid() XclaimJustid {
	c.cs.s = append(c.cs.s, "JUSTID")
	return (XclaimJustid)(c)
}

func (c XclaimId) Lastid() XclaimLastid {
	c.cs.s = append(c.cs.s, "LASTID")
	return (XclaimLastid)(c)
}

func (c XclaimId) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type XclaimIdle Completed

func (c XclaimIdle) Time(msUnixTime int64) XclaimTime {
	c.cs.s = append(c.cs.s, "TIME", strconv.FormatInt(msUnixTime, 10))
	return (XclaimTime)(c)
}

func (c XclaimIdle) Retrycount(count int64) XclaimRetrycount {
	c.cs.s = append(c.cs.s, "RETRYCOUNT", strconv.FormatInt(count, 10))
	return (XclaimRetrycount)(c)
}

func (c XclaimIdle) Force() XclaimForce {
	c.cs.s = append(c.cs.s, "FORCE")
	return (XclaimForce)(c)
}

func (c XclaimIdle) Justid() XclaimJustid {
	c.cs.s = append(c.cs.s, "JUSTID")
	return (XclaimJustid)(c)
}

func (c XclaimIdle) Lastid() XclaimLastid {
	c.cs.s = append(c.cs.s, "LASTID")
	return (XclaimLastid)(c)
}

func (c XclaimIdle) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type XclaimJustid Completed

func (c XclaimJustid) Lastid() XclaimLastid {
	c.cs.s = append(c.cs.s, "LASTID")
	return (XclaimLastid)(c)
}

func (c XclaimJustid) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type XclaimKey Completed

func (c XclaimKey) Group(group string) XclaimGroup {
	c.cs.s = append(c.cs.s, group)
	return (XclaimGroup)(c)
}

type XclaimLastid Completed

func (c XclaimLastid) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type XclaimMinIdleTime Completed

func (c XclaimMinIdleTime) Id(id ...string) XclaimId {
	c.cs.s = append(c.cs.s, id...)
	return (XclaimId)(c)
}

type XclaimRetrycount Completed

func (c XclaimRetrycount) Force() XclaimForce {
	c.cs.s = append(c.cs.s, "FORCE")
	return (XclaimForce)(c)
}

func (c XclaimRetrycount) Justid() XclaimJustid {
	c.cs.s = append(c.cs.s, "JUSTID")
	return (XclaimJustid)(c)
}

func (c XclaimRetrycount) Lastid() XclaimLastid {
	c.cs.s = append(c.cs.s, "LASTID")
	return (XclaimLastid)(c)
}

func (c XclaimRetrycount) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type XclaimTime Completed

func (c XclaimTime) Retrycount(count int64) XclaimRetrycount {
	c.cs.s = append(c.cs.s, "RETRYCOUNT", strconv.FormatInt(count, 10))
	return (XclaimRetrycount)(c)
}

func (c XclaimTime) Force() XclaimForce {
	c.cs.s = append(c.cs.s, "FORCE")
	return (XclaimForce)(c)
}

func (c XclaimTime) Justid() XclaimJustid {
	c.cs.s = append(c.cs.s, "JUSTID")
	return (XclaimJustid)(c)
}

func (c XclaimTime) Lastid() XclaimLastid {
	c.cs.s = append(c.cs.s, "LASTID")
	return (XclaimLastid)(c)
}

func (c XclaimTime) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type Xdel Completed

func (b Builder) Xdel() (c Xdel) {
	c = Xdel{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "XDEL")
	return c
}

func (c Xdel) Key(key string) XdelKey {
	if c.ks&NoSlot == NoSlot {
		c.ks = NoSlot | slot(key)
	} else {
		c.ks = check(c.ks, slot(key))
	}
	c.cs.s = append(c.cs.s, key)
	return (XdelKey)(c)
}

type XdelId Completed

func (c XdelId) Id(id ...string) XdelId {
	c.cs.s = append(c.cs.s, id...)
	return c
}

func (c XdelId) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type XdelKey Completed

func (c XdelKey) Id(id ...string) XdelId {
	c.cs.s = append(c.cs.s, id...)
	return (XdelId)(c)
}

type XgroupCreate Completed

func (b Builder) XgroupCreate() (c XgroupCreate) {
	c = XgroupCreate{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "XGROUP", "CREATE")
	return c
}

func (c XgroupCreate) Key(key string) XgroupCreateKey {
	if c.ks&NoSlot == NoSlot {
		c.ks = NoSlot | slot(key)
	} else {
		c.ks = check(c.ks, slot(key))
	}
	c.cs.s = append(c.cs.s, key)
	return (XgroupCreateKey)(c)
}

type XgroupCreateEntriesread Completed

func (c XgroupCreateEntriesread) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type XgroupCreateGroup Completed

func (c XgroupCreateGroup) Id(id string) XgroupCreateId {
	c.cs.s = append(c.cs.s, id)
	return (XgroupCreateId)(c)
}

type XgroupCreateId Completed

func (c XgroupCreateId) Mkstream() XgroupCreateMkstream {
	c.cs.s = append(c.cs.s, "MKSTREAM")
	return (XgroupCreateMkstream)(c)
}

func (c XgroupCreateId) Entriesread(entriesRead int64) XgroupCreateEntriesread {
	c.cs.s = append(c.cs.s, "ENTRIESREAD", strconv.FormatInt(entriesRead, 10))
	return (XgroupCreateEntriesread)(c)
}

func (c XgroupCreateId) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type XgroupCreateKey Completed

func (c XgroupCreateKey) Group(group string) XgroupCreateGroup {
	c.cs.s = append(c.cs.s, group)
	return (XgroupCreateGroup)(c)
}

type XgroupCreateMkstream Completed

func (c XgroupCreateMkstream) Entriesread(entriesRead int64) XgroupCreateEntriesread {
	c.cs.s = append(c.cs.s, "ENTRIESREAD", strconv.FormatInt(entriesRead, 10))
	return (XgroupCreateEntriesread)(c)
}

func (c XgroupCreateMkstream) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type XgroupCreateconsumer Completed

func (b Builder) XgroupCreateconsumer() (c XgroupCreateconsumer) {
	c = XgroupCreateconsumer{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "XGROUP", "CREATECONSUMER")
	return c
}

func (c XgroupCreateconsumer) Key(key string) XgroupCreateconsumerKey {
	if c.ks&NoSlot == NoSlot {
		c.ks = NoSlot | slot(key)
	} else {
		c.ks = check(c.ks, slot(key))
	}
	c.cs.s = append(c.cs.s, key)
	return (XgroupCreateconsumerKey)(c)
}

type XgroupCreateconsumerConsumer Completed

func (c XgroupCreateconsumerConsumer) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type XgroupCreateconsumerGroup Completed

func (c XgroupCreateconsumerGroup) Consumer(consumer string) XgroupCreateconsumerConsumer {
	c.cs.s = append(c.cs.s, consumer)
	return (XgroupCreateconsumerConsumer)(c)
}

type XgroupCreateconsumerKey Completed

func (c XgroupCreateconsumerKey) Group(group string) XgroupCreateconsumerGroup {
	c.cs.s = append(c.cs.s, group)
	return (XgroupCreateconsumerGroup)(c)
}

type XgroupDelconsumer Completed

func (b Builder) XgroupDelconsumer() (c XgroupDelconsumer) {
	c = XgroupDelconsumer{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "XGROUP", "DELCONSUMER")
	return c
}

func (c XgroupDelconsumer) Key(key string) XgroupDelconsumerKey {
	if c.ks&NoSlot == NoSlot {
		c.ks = NoSlot | slot(key)
	} else {
		c.ks = check(c.ks, slot(key))
	}
	c.cs.s = append(c.cs.s, key)
	return (XgroupDelconsumerKey)(c)
}

type XgroupDelconsumerConsumername Completed

func (c XgroupDelconsumerConsumername) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type XgroupDelconsumerGroup Completed

func (c XgroupDelconsumerGroup) Consumername(consumername string) XgroupDelconsumerConsumername {
	c.cs.s = append(c.cs.s, consumername)
	return (XgroupDelconsumerConsumername)(c)
}

type XgroupDelconsumerKey Completed

func (c XgroupDelconsumerKey) Group(group string) XgroupDelconsumerGroup {
	c.cs.s = append(c.cs.s, group)
	return (XgroupDelconsumerGroup)(c)
}

type XgroupDestroy Completed

func (b Builder) XgroupDestroy() (c XgroupDestroy) {
	c = XgroupDestroy{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "XGROUP", "DESTROY")
	return c
}

func (c XgroupDestroy) Key(key string) XgroupDestroyKey {
	if c.ks&NoSlot == NoSlot {
		c.ks = NoSlot | slot(key)
	} else {
		c.ks = check(c.ks, slot(key))
	}
	c.cs.s = append(c.cs.s, key)
	return (XgroupDestroyKey)(c)
}

type XgroupDestroyGroup Completed

func (c XgroupDestroyGroup) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type XgroupDestroyKey Completed

func (c XgroupDestroyKey) Group(group string) XgroupDestroyGroup {
	c.cs.s = append(c.cs.s, group)
	return (XgroupDestroyGroup)(c)
}

type XgroupHelp Completed

func (b Builder) XgroupHelp() (c XgroupHelp) {
	c = XgroupHelp{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "XGROUP", "HELP")
	return c
}

func (c XgroupHelp) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type XgroupSetid Completed

func (b Builder) XgroupSetid() (c XgroupSetid) {
	c = XgroupSetid{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "XGROUP", "SETID")
	return c
}

func (c XgroupSetid) Key(key string) XgroupSetidKey {
	if c.ks&NoSlot == NoSlot {
		c.ks = NoSlot | slot(key)
	} else {
		c.ks = check(c.ks, slot(key))
	}
	c.cs.s = append(c.cs.s, key)
	return (XgroupSetidKey)(c)
}

type XgroupSetidEntriesread Completed

func (c XgroupSetidEntriesread) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type XgroupSetidGroup Completed

func (c XgroupSetidGroup) Id(id string) XgroupSetidId {
	c.cs.s = append(c.cs.s, id)
	return (XgroupSetidId)(c)
}

type XgroupSetidId Completed

func (c XgroupSetidId) Entriesread(entriesRead int64) XgroupSetidEntriesread {
	c.cs.s = append(c.cs.s, "ENTRIESREAD", strconv.FormatInt(entriesRead, 10))
	return (XgroupSetidEntriesread)(c)
}

func (c XgroupSetidId) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type XgroupSetidKey Completed

func (c XgroupSetidKey) Group(group string) XgroupSetidGroup {
	c.cs.s = append(c.cs.s, group)
	return (XgroupSetidGroup)(c)
}

type XinfoConsumers Completed

func (b Builder) XinfoConsumers() (c XinfoConsumers) {
	c = XinfoConsumers{cs: get(), ks: b.ks, cf: readonly}
	c.cs.s = append(c.cs.s, "XINFO", "CONSUMERS")
	return c
}

func (c XinfoConsumers) Key(key string) XinfoConsumersKey {
	if c.ks&NoSlot == NoSlot {
		c.ks = NoSlot | slot(key)
	} else {
		c.ks = check(c.ks, slot(key))
	}
	c.cs.s = append(c.cs.s, key)
	return (XinfoConsumersKey)(c)
}

type XinfoConsumersGroup Completed

func (c XinfoConsumersGroup) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type XinfoConsumersKey Completed

func (c XinfoConsumersKey) Group(group string) XinfoConsumersGroup {
	c.cs.s = append(c.cs.s, group)
	return (XinfoConsumersGroup)(c)
}

type XinfoGroups Completed

func (b Builder) XinfoGroups() (c XinfoGroups) {
	c = XinfoGroups{cs: get(), ks: b.ks, cf: readonly}
	c.cs.s = append(c.cs.s, "XINFO", "GROUPS")
	return c
}

func (c XinfoGroups) Key(key string) XinfoGroupsKey {
	if c.ks&NoSlot == NoSlot {
		c.ks = NoSlot | slot(key)
	} else {
		c.ks = check(c.ks, slot(key))
	}
	c.cs.s = append(c.cs.s, key)
	return (XinfoGroupsKey)(c)
}

type XinfoGroupsKey Completed

func (c XinfoGroupsKey) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type XinfoHelp Completed

func (b Builder) XinfoHelp() (c XinfoHelp) {
	c = XinfoHelp{cs: get(), ks: b.ks, cf: readonly}
	c.cs.s = append(c.cs.s, "XINFO", "HELP")
	return c
}

func (c XinfoHelp) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type XinfoStream Completed

func (b Builder) XinfoStream() (c XinfoStream) {
	c = XinfoStream{cs: get(), ks: b.ks, cf: readonly}
	c.cs.s = append(c.cs.s, "XINFO", "STREAM")
	return c
}

func (c XinfoStream) Key(key string) XinfoStreamKey {
	if c.ks&NoSlot == NoSlot {
		c.ks = NoSlot | slot(key)
	} else {
		c.ks = check(c.ks, slot(key))
	}
	c.cs.s = append(c.cs.s, key)
	return (XinfoStreamKey)(c)
}

type XinfoStreamFullCount Completed

func (c XinfoStreamFullCount) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type XinfoStreamFullFull Completed

func (c XinfoStreamFullFull) Count(count int64) XinfoStreamFullCount {
	c.cs.s = append(c.cs.s, "COUNT", strconv.FormatInt(count, 10))
	return (XinfoStreamFullCount)(c)
}

func (c XinfoStreamFullFull) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type XinfoStreamKey Completed

func (c XinfoStreamKey) Full() XinfoStreamFullFull {
	c.cs.s = append(c.cs.s, "FULL")
	return (XinfoStreamFullFull)(c)
}

func (c XinfoStreamKey) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type Xlen Completed

func (b Builder) Xlen() (c Xlen) {
	c = Xlen{cs: get(), ks: b.ks, cf: readonly}
	c.cs.s = append(c.cs.s, "XLEN")
	return c
}

func (c Xlen) Key(key string) XlenKey {
	if c.ks&NoSlot == NoSlot {
		c.ks = NoSlot | slot(key)
	} else {
		c.ks = check(c.ks, slot(key))
	}
	c.cs.s = append(c.cs.s, key)
	return (XlenKey)(c)
}

type XlenKey Completed

func (c XlenKey) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type Xpending Completed

func (b Builder) Xpending() (c Xpending) {
	c = Xpending{cs: get(), ks: b.ks, cf: readonly}
	c.cs.s = append(c.cs.s, "XPENDING")
	return c
}

func (c Xpending) Key(key string) XpendingKey {
	if c.ks&NoSlot == NoSlot {
		c.ks = NoSlot | slot(key)
	} else {
		c.ks = check(c.ks, slot(key))
	}
	c.cs.s = append(c.cs.s, key)
	return (XpendingKey)(c)
}

type XpendingFiltersConsumer Completed

func (c XpendingFiltersConsumer) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type XpendingFiltersCount Completed

func (c XpendingFiltersCount) Consumer(consumer string) XpendingFiltersConsumer {
	c.cs.s = append(c.cs.s, consumer)
	return (XpendingFiltersConsumer)(c)
}

func (c XpendingFiltersCount) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type XpendingFiltersEnd Completed

func (c XpendingFiltersEnd) Count(count int64) XpendingFiltersCount {
	c.cs.s = append(c.cs.s, strconv.FormatInt(count, 10))
	return (XpendingFiltersCount)(c)
}

type XpendingFiltersIdle Completed

func (c XpendingFiltersIdle) Start(start string) XpendingFiltersStart {
	c.cs.s = append(c.cs.s, start)
	return (XpendingFiltersStart)(c)
}

type XpendingFiltersStart Completed

func (c XpendingFiltersStart) End(end string) XpendingFiltersEnd {
	c.cs.s = append(c.cs.s, end)
	return (XpendingFiltersEnd)(c)
}

type XpendingGroup Completed

func (c XpendingGroup) Idle(minIdleTime int64) XpendingFiltersIdle {
	c.cs.s = append(c.cs.s, "IDLE", strconv.FormatInt(minIdleTime, 10))
	return (XpendingFiltersIdle)(c)
}

func (c XpendingGroup) Start(start string) XpendingFiltersStart {
	c.cs.s = append(c.cs.s, start)
	return (XpendingFiltersStart)(c)
}

func (c XpendingGroup) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type XpendingKey Completed

func (c XpendingKey) Group(group string) XpendingGroup {
	c.cs.s = append(c.cs.s, group)
	return (XpendingGroup)(c)
}

type Xrange Completed

func (b Builder) Xrange() (c Xrange) {
	c = Xrange{cs: get(), ks: b.ks, cf: readonly}
	c.cs.s = append(c.cs.s, "XRANGE")
	return c
}

func (c Xrange) Key(key string) XrangeKey {
	if c.ks&NoSlot == NoSlot {
		c.ks = NoSlot | slot(key)
	} else {
		c.ks = check(c.ks, slot(key))
	}
	c.cs.s = append(c.cs.s, key)
	return (XrangeKey)(c)
}

type XrangeCount Completed

func (c XrangeCount) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type XrangeEnd Completed

func (c XrangeEnd) Count(count int64) XrangeCount {
	c.cs.s = append(c.cs.s, "COUNT", strconv.FormatInt(count, 10))
	return (XrangeCount)(c)
}

func (c XrangeEnd) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type XrangeKey Completed

func (c XrangeKey) Start(start string) XrangeStart {
	c.cs.s = append(c.cs.s, start)
	return (XrangeStart)(c)
}

type XrangeStart Completed

func (c XrangeStart) End(end string) XrangeEnd {
	c.cs.s = append(c.cs.s, end)
	return (XrangeEnd)(c)
}

type Xread Completed

func (b Builder) Xread() (c Xread) {
	c = Xread{cs: get(), ks: b.ks, cf: readonly}
	c.cs.s = append(c.cs.s, "XREAD")
	return c
}

func (c Xread) Count(count int64) XreadCount {
	c.cs.s = append(c.cs.s, "COUNT", strconv.FormatInt(count, 10))
	return (XreadCount)(c)
}

func (c Xread) Block(milliseconds int64) XreadBlock {
	c.cf = blockTag
	c.cs.s = append(c.cs.s, "BLOCK", strconv.FormatInt(milliseconds, 10))
	return (XreadBlock)(c)
}

func (c Xread) Streams() XreadStreams {
	c.cs.s = append(c.cs.s, "STREAMS")
	return (XreadStreams)(c)
}

type XreadBlock Completed

func (c XreadBlock) Streams() XreadStreams {
	c.cs.s = append(c.cs.s, "STREAMS")
	return (XreadStreams)(c)
}

type XreadCount Completed

func (c XreadCount) Block(milliseconds int64) XreadBlock {
	c.cf = blockTag
	c.cs.s = append(c.cs.s, "BLOCK", strconv.FormatInt(milliseconds, 10))
	return (XreadBlock)(c)
}

func (c XreadCount) Streams() XreadStreams {
	c.cs.s = append(c.cs.s, "STREAMS")
	return (XreadStreams)(c)
}

type XreadId Completed

func (c XreadId) Id(id ...string) XreadId {
	c.cs.s = append(c.cs.s, id...)
	return c
}

func (c XreadId) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type XreadKey Completed

func (c XreadKey) Key(key ...string) XreadKey {
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

func (c XreadKey) Id(id ...string) XreadId {
	c.cs.s = append(c.cs.s, id...)
	return (XreadId)(c)
}

type XreadStreams Completed

func (c XreadStreams) Key(key ...string) XreadKey {
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
	return (XreadKey)(c)
}

type Xreadgroup Completed

func (b Builder) Xreadgroup() (c Xreadgroup) {
	c = Xreadgroup{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "XREADGROUP")
	return c
}

func (c Xreadgroup) Group(group string, consumer string) XreadgroupGroup {
	c.cs.s = append(c.cs.s, "GROUP", group, consumer)
	return (XreadgroupGroup)(c)
}

type XreadgroupBlock Completed

func (c XreadgroupBlock) Noack() XreadgroupNoack {
	c.cs.s = append(c.cs.s, "NOACK")
	return (XreadgroupNoack)(c)
}

func (c XreadgroupBlock) Streams() XreadgroupStreams {
	c.cs.s = append(c.cs.s, "STREAMS")
	return (XreadgroupStreams)(c)
}

type XreadgroupCount Completed

func (c XreadgroupCount) Block(milliseconds int64) XreadgroupBlock {
	c.cf = blockTag
	c.cs.s = append(c.cs.s, "BLOCK", strconv.FormatInt(milliseconds, 10))
	return (XreadgroupBlock)(c)
}

func (c XreadgroupCount) Noack() XreadgroupNoack {
	c.cs.s = append(c.cs.s, "NOACK")
	return (XreadgroupNoack)(c)
}

func (c XreadgroupCount) Streams() XreadgroupStreams {
	c.cs.s = append(c.cs.s, "STREAMS")
	return (XreadgroupStreams)(c)
}

type XreadgroupGroup Completed

func (c XreadgroupGroup) Count(count int64) XreadgroupCount {
	c.cs.s = append(c.cs.s, "COUNT", strconv.FormatInt(count, 10))
	return (XreadgroupCount)(c)
}

func (c XreadgroupGroup) Block(milliseconds int64) XreadgroupBlock {
	c.cf = blockTag
	c.cs.s = append(c.cs.s, "BLOCK", strconv.FormatInt(milliseconds, 10))
	return (XreadgroupBlock)(c)
}

func (c XreadgroupGroup) Noack() XreadgroupNoack {
	c.cs.s = append(c.cs.s, "NOACK")
	return (XreadgroupNoack)(c)
}

func (c XreadgroupGroup) Streams() XreadgroupStreams {
	c.cs.s = append(c.cs.s, "STREAMS")
	return (XreadgroupStreams)(c)
}

type XreadgroupId Completed

func (c XreadgroupId) Id(id ...string) XreadgroupId {
	c.cs.s = append(c.cs.s, id...)
	return c
}

func (c XreadgroupId) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type XreadgroupKey Completed

func (c XreadgroupKey) Key(key ...string) XreadgroupKey {
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

func (c XreadgroupKey) Id(id ...string) XreadgroupId {
	c.cs.s = append(c.cs.s, id...)
	return (XreadgroupId)(c)
}

type XreadgroupNoack Completed

func (c XreadgroupNoack) Streams() XreadgroupStreams {
	c.cs.s = append(c.cs.s, "STREAMS")
	return (XreadgroupStreams)(c)
}

type XreadgroupStreams Completed

func (c XreadgroupStreams) Key(key ...string) XreadgroupKey {
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
	return (XreadgroupKey)(c)
}

type Xrevrange Completed

func (b Builder) Xrevrange() (c Xrevrange) {
	c = Xrevrange{cs: get(), ks: b.ks, cf: readonly}
	c.cs.s = append(c.cs.s, "XREVRANGE")
	return c
}

func (c Xrevrange) Key(key string) XrevrangeKey {
	if c.ks&NoSlot == NoSlot {
		c.ks = NoSlot | slot(key)
	} else {
		c.ks = check(c.ks, slot(key))
	}
	c.cs.s = append(c.cs.s, key)
	return (XrevrangeKey)(c)
}

type XrevrangeCount Completed

func (c XrevrangeCount) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type XrevrangeEnd Completed

func (c XrevrangeEnd) Start(start string) XrevrangeStart {
	c.cs.s = append(c.cs.s, start)
	return (XrevrangeStart)(c)
}

type XrevrangeKey Completed

func (c XrevrangeKey) End(end string) XrevrangeEnd {
	c.cs.s = append(c.cs.s, end)
	return (XrevrangeEnd)(c)
}

type XrevrangeStart Completed

func (c XrevrangeStart) Count(count int64) XrevrangeCount {
	c.cs.s = append(c.cs.s, "COUNT", strconv.FormatInt(count, 10))
	return (XrevrangeCount)(c)
}

func (c XrevrangeStart) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type Xsetid Completed

func (b Builder) Xsetid() (c Xsetid) {
	c = Xsetid{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "XSETID")
	return c
}

func (c Xsetid) Key(key string) XsetidKey {
	if c.ks&NoSlot == NoSlot {
		c.ks = NoSlot | slot(key)
	} else {
		c.ks = check(c.ks, slot(key))
	}
	c.cs.s = append(c.cs.s, key)
	return (XsetidKey)(c)
}

type XsetidEntriesadded Completed

func (c XsetidEntriesadded) Maxdeletedid(maxDeletedEntryId string) XsetidMaxdeletedid {
	c.cs.s = append(c.cs.s, "MAXDELETEDID", maxDeletedEntryId)
	return (XsetidMaxdeletedid)(c)
}

func (c XsetidEntriesadded) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type XsetidKey Completed

func (c XsetidKey) LastId(lastId string) XsetidLastId {
	c.cs.s = append(c.cs.s, lastId)
	return (XsetidLastId)(c)
}

type XsetidLastId Completed

func (c XsetidLastId) Entriesadded(entriesAdded int64) XsetidEntriesadded {
	c.cs.s = append(c.cs.s, "ENTRIESADDED", strconv.FormatInt(entriesAdded, 10))
	return (XsetidEntriesadded)(c)
}

func (c XsetidLastId) Maxdeletedid(maxDeletedEntryId string) XsetidMaxdeletedid {
	c.cs.s = append(c.cs.s, "MAXDELETEDID", maxDeletedEntryId)
	return (XsetidMaxdeletedid)(c)
}

func (c XsetidLastId) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type XsetidMaxdeletedid Completed

func (c XsetidMaxdeletedid) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type Xtrim Completed

func (b Builder) Xtrim() (c Xtrim) {
	c = Xtrim{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, "XTRIM")
	return c
}

func (c Xtrim) Key(key string) XtrimKey {
	if c.ks&NoSlot == NoSlot {
		c.ks = NoSlot | slot(key)
	} else {
		c.ks = check(c.ks, slot(key))
	}
	c.cs.s = append(c.cs.s, key)
	return (XtrimKey)(c)
}

type XtrimKey Completed

func (c XtrimKey) Maxlen() XtrimTrimStrategyMaxlen {
	c.cs.s = append(c.cs.s, "MAXLEN")
	return (XtrimTrimStrategyMaxlen)(c)
}

func (c XtrimKey) Minid() XtrimTrimStrategyMinid {
	c.cs.s = append(c.cs.s, "MINID")
	return (XtrimTrimStrategyMinid)(c)
}

type XtrimTrimLimit Completed

func (c XtrimTrimLimit) Build() Completed {
	c.cs.Build()
	return Completed(c)
}

type XtrimTrimOperatorAlmost Completed

func (c XtrimTrimOperatorAlmost) Threshold(threshold string) XtrimTrimThreshold {
	c.cs.s = append(c.cs.s, threshold)
	return (XtrimTrimThreshold)(c)
}

type XtrimTrimOperatorExact Completed

func (c XtrimTrimOperatorExact) Threshold(threshold string) XtrimTrimThreshold {
	c.cs.s = append(c.cs.s, threshold)
	return (XtrimTrimThreshold)(c)
}

type XtrimTrimStrategyMaxlen Completed

func (c XtrimTrimStrategyMaxlen) Exact() XtrimTrimOperatorExact {
	c.cs.s = append(c.cs.s, "=")
	return (XtrimTrimOperatorExact)(c)
}

func (c XtrimTrimStrategyMaxlen) Almost() XtrimTrimOperatorAlmost {
	c.cs.s = append(c.cs.s, "~")
	return (XtrimTrimOperatorAlmost)(c)
}

func (c XtrimTrimStrategyMaxlen) Threshold(threshold string) XtrimTrimThreshold {
	c.cs.s = append(c.cs.s, threshold)
	return (XtrimTrimThreshold)(c)
}

type XtrimTrimStrategyMinid Completed

func (c XtrimTrimStrategyMinid) Exact() XtrimTrimOperatorExact {
	c.cs.s = append(c.cs.s, "=")
	return (XtrimTrimOperatorExact)(c)
}

func (c XtrimTrimStrategyMinid) Almost() XtrimTrimOperatorAlmost {
	c.cs.s = append(c.cs.s, "~")
	return (XtrimTrimOperatorAlmost)(c)
}

func (c XtrimTrimStrategyMinid) Threshold(threshold string) XtrimTrimThreshold {
	c.cs.s = append(c.cs.s, threshold)
	return (XtrimTrimThreshold)(c)
}

type XtrimTrimThreshold Completed

func (c XtrimTrimThreshold) Limit(count int64) XtrimTrimLimit {
	c.cs.s = append(c.cs.s, "LIMIT", strconv.FormatInt(count, 10))
	return (XtrimTrimLimit)(c)
}

func (c XtrimTrimThreshold) Build() Completed {
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

