// Code generated DO NOT EDIT

package cmds

import "strconv"

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
