package aigateway

// ParsedInferenceResponse represents the parsed output for a given inference request.
type ParsedInferenceResponse struct {
	ID         string `json:"id"`
	TokensIn   int32  `json:"tokens_in"`
	TokensOut  int32  `json:"tokens_out"`
	StopReason string `json:"stop_reason"`
	// TODO: Tool use selections, parsed.
}
