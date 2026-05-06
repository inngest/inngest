//go:generate go run github.com/dmarkham/enumer -trimprefix=StepType -type=StepType -json -text -transform title-lower

package enums

// StepType represents the user-facing type of a step in a function execution.
// These match the step tools defined in the SDK and are mostly plumbed for reference in Insights.
type StepType int

const (
	// StepTypeUnknown represents an unknown step type.
	// This should never be used in practice, but is included as a default value.
	StepTypeUnknown StepType = iota
	StepTypeRun
	StepTypeSendEvent
	StepTypeSendSignal
	StepTypeSleep
	StepTypeWaitForEvent
	StepTypeInvoke
	StepTypeAiInfer // Lowercase Ai to transform to `aiInfer` instead of `aIInfer`
	StepTypeAiWrap  // Lowercase Ai to transform to `aiWrap` instead of `aIWrap`
	StepTypeFetch
	StepTypeWaitForSignal
	StepTypeMetadata
	StepTypeGroupExperiment
	StepTypeRealtimePublish
)
