package extractors

import (
	"bytes"
	"encoding/json"
	"math"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/inngest/inngest/pkg/enums"
	"github.com/inngest/inngest/pkg/tracing/metadata"
	"github.com/inngest/inngest/pkg/util"
	"github.com/inngest/inngest/pkg/util/aigateway"
)

//tygo:generate
const (
	KindInngestAI metadata.Kind = "inngest.ai"
)

//tygo:generate
type AIMetadata struct {
	InputTokens   int64  `json:"input_tokens"`
	OutputTokens  int64  `json:"output_tokens"`
	RequestModel  string `json:"request_model"`
	Provider      string `json:"provider"`
	OperationName string `json:"operation_name"`

	// Response identity. ResponseModel is the model that served the request (may
	// differ from the requested Model, e.g. a dated snapshot). FinishReasons is
	// stored raw per emitter — note OpenAI's native "tool_calls" is emitted as
	// the singular "tool_call" by some instrumentations.
	ResponseModel string   `json:"response_model,omitempty"`
	ResponseID    string   `json:"response_id,omitempty"`
	FinishReasons []string `json:"finish_reasons,omitempty"`

	LatencyMs     *int64   `json:"latency_ms,omitempty"`
	TotalTokens   *int64   `json:"total_tokens,omitempty"`
	EstimatedCost *float64 `json:"estimated_cost,omitempty"`
}

func (ms AIMetadata) Kind() metadata.Kind {
	return KindInngestAI
}

func (ms AIMetadata) Op() metadata.Opcode {
	return enums.MetadataOpcodeMerge
}

func (ms AIMetadata) Serialize() (metadata.Values, error) {
	var rawMetadata metadata.Values
	err := rawMetadata.FromStruct(ms)
	if err != nil {
		return nil, err
	}

	return rawMetadata, nil
}

func ExtractAIGatewayMetadata(req aigateway.Request, respStatus int, resp []byte, serverProcessingMs int64) ([]metadata.Structured, error) {
	parsedInput, err := aigateway.ParseInput(req)
	if err != nil {
		return nil, &metadata.WarningError{
			Key: "inngest.ai.request.parsing.failed",
			Err: err,
		}
	}

	u, err := url.Parse(parsedInput.URL)
	if err != nil {
		return nil, &metadata.WarningError{
			Key: "inngest.ai.request.parsing.failed",
			Err: err,
		}
	}

	parsedOutput, err := aigateway.ParseOutput(req.Format, resp)
	if err != nil {
		return nil, &metadata.WarningError{
			Key: "inngest.ai.response.parsing.failed",
			Err: err,
		}
	}

	inputTokens := int64(parsedOutput.TokensIn)
	outputTokens := int64(parsedOutput.TokensOut)
	totalTokens := inputTokens + outputTokens

	var latencyMs *int64
	if serverProcessingMs > 0 {
		latencyMs = &serverProcessingMs
	}

	// prefer the response model (the model that actually served the request)
	// for cost estimation, falling back to the requested model.
	costModel := parsedOutput.Model
	if costModel == "" {
		costModel = parsedInput.Model
	}

	aiMd := &AIMetadata{
		RequestModel:  parsedInput.Model,
		ResponseModel: parsedOutput.Model,
		Provider:      req.Format,
		OperationName: "",

		InputTokens:   inputTokens,
		OutputTokens:  outputTokens,
		TotalTokens:   &totalTokens,
		EstimatedCost: EstimateCost(costModel, inputTokens, outputTokens),
		LatencyMs:     latencyMs,
	}

	return []metadata.Structured{
		aiMd,
		&HTTPMetadata{
			Method:             http.MethodPost,
			Domain:             util.ToPtr(u.Host),
			Path:               util.ToPtr(u.Path),
			RequestContentType: util.ToPtr("application/json"),
			RequestSize:        util.ToPtr(int64(len(req.Body))),

			ResponseContentType: util.ToPtr("application/json"),
			ResponseSize:        util.ToPtr(int64(len(resp))),
			ResponseStatus:      util.ToPtr(int64(respStatus)),
		},
	}, nil
}

type vercelAIUsage struct {
	InputTokens  int64 `json:"inputTokens"`
	OutputTokens int64 `json:"outputTokens"`
	TotalTokens  int64 `json:"totalTokens"`
}

type vercelAIStepResponse struct {
	ModelID string            `json:"modelId"`
	Headers map[string]string `json:"headers"`
}

type vercelAIStepRequest struct {
	Body struct {
		Model string `json:"model"`
	} `json:"body"`
}

type vercelAIStep struct {
	Usage    *vercelAIUsage        `json:"usage"`
	Response *vercelAIStepResponse `json:"response"`
	Request  *vercelAIStepRequest  `json:"request"`
}

type vercelAIResponseData struct {
	TotalUsage *vercelAIUsage `json:"totalUsage"`
	Steps      []vercelAIStep `json:"steps"`
}

// vercelAIResponse represents the Vercel AI SDK response format from step.ai.wrap.
// The step output is wrapped in {"data": ...} by Inngest.
type vercelAIResponse struct {
	Data *vercelAIResponseData `json:"data"`
}

// ExtractAIOutputMetadata extracts ai metadata from step output
// which contains vercel ai sdk response format.
// stepDurationMs is the step execution duration in milliseconds, used as fallback for latency.
func ExtractAIOutputMetadata(output []byte, stepDurationMs int64) ([]metadata.Structured, error) {
	// skip unmarshal if output doesn't contain ai-specific fields
	if !bytes.Contains(output, []byte("totalUsage")) &&
		!bytes.Contains(output, []byte("inputTokens")) {
		return nil, nil
	}

	var resp vercelAIResponse
	if err := json.Unmarshal(output, &resp); err != nil {
		return nil, nil
	}

	// check if we have the expected vercel ai sdk structure
	if resp.Data == nil {
		return nil, nil
	}

	// extract the first step for model and latency lookups
	var firstStep *vercelAIStep
	if len(resp.Data.Steps) > 0 {
		firstStep = &resp.Data.Steps[0]
	}

	// try to get usage from totalUsage first, then from first step
	var inputTokens, outputTokens, totalTokens int64
	if resp.Data.TotalUsage != nil {
		inputTokens = resp.Data.TotalUsage.InputTokens
		outputTokens = resp.Data.TotalUsage.OutputTokens
		totalTokens = resp.Data.TotalUsage.TotalTokens
	} else if firstStep != nil && firstStep.Usage != nil {
		inputTokens = firstStep.Usage.InputTokens
		outputTokens = firstStep.Usage.OutputTokens
		totalTokens = firstStep.Usage.TotalTokens
	} else {
		return nil, nil
	}

	var requestModel string
	var responseModel string
	if firstStep != nil {
		if firstStep.Response != nil && firstStep.Response.ModelID != "" {
			responseModel = firstStep.Response.ModelID
		}

		if firstStep.Request != nil && firstStep.Request.Body.Model != "" {
			requestModel = firstStep.Request.Body.Model
		}
	}

	// extract latency from provider headers, fallback to step duration
	var latencyMs *int64
	if firstStep != nil && firstStep.Response != nil && firstStep.Response.Headers != nil {
		headers := firstStep.Response.Headers
		// try OpenAI header
		if ms, ok := headers["openai-processing-ms"]; ok {
			if parsed, err := strconv.ParseInt(ms, 10, 64); err == nil {
				latencyMs = &parsed
			}
		}
		// TODO: Add other provider headers (Anthropic, etc.) as needed
	}

	// fallback to step duration if no provider header
	if latencyMs == nil && stepDurationMs > 0 {
		latencyMs = &stepDurationMs
	}

	// prefer the response model (the model that actually served the request)
	// for cost estimation, falling back to the requested model.
	costModel := responseModel
	if costModel == "" {
		costModel = requestModel
	}

	aiMd := &AIMetadata{
		InputTokens:   inputTokens,
		OutputTokens:  outputTokens,
		TotalTokens:   &totalTokens,
		RequestModel:  requestModel,
		ResponseModel: responseModel,
		Provider:      "vercel-ai",
		LatencyMs:     latencyMs,
		EstimatedCost: EstimateCost(costModel, inputTokens, outputTokens),
	}

	return []metadata.Structured{aiMd}, nil
}

// ModelPricing contains input/output pricing per 1M tokens in USD
type ModelPricing struct {
	InputPer1M  float64
	OutputPer1M float64
}

// modelPricing is the exact match pricing table - prices in USD per 1M tokens
// Source: https://openai.com/pricing, https://anthropic.com/pricing, https://ai.google.dev/pricing
var modelPricing = map[string]ModelPricing{
	"gpt-5.2":     {1.75, 14.00},
	"gpt-5.1":     {1.25, 10.00},
	"gpt-5":       {1.25, 10.00},
	"gpt-5-mini":  {0.25, 2.00},
	"gpt-5-nano":  {0.05, 0.40},
	"gpt-5.2-pro": {21.00, 168.00},
	"gpt-5-pro":   {15.00, 120.00},

	"gpt-4.1":      {2.00, 8.00},
	"gpt-4.1-mini": {0.40, 1.60},
	"gpt-4.1-nano": {0.10, 0.40},

	"gpt-4o":      {2.50, 10.00},
	"gpt-4o-mini": {0.15, 0.60},
	"gpt-4-turbo": {10.00, 30.00},

	"o1":      {15.00, 60.00},
	"o1-pro":  {150.00, 600.00},
	"o1-mini": {1.10, 4.40},
	"o3":      {2.00, 8.00},
	"o3-pro":  {20.00, 80.00},
	"o3-mini": {1.10, 4.40},
	"o4-mini": {1.10, 4.40},

	"claude-opus-4-5":   {5.00, 25.00},
	"claude-opus-4-1":   {15.00, 75.00},
	"claude-opus-4":     {15.00, 75.00},
	"claude-sonnet-4-5": {3.00, 15.00},
	"claude-sonnet-4":   {3.00, 15.00},
	"claude-haiku-4-5":  {1.00, 5.00},

	"claude-haiku-3-5": {0.80, 4.00},

	"claude-3-haiku": {0.25, 1.25},

	"gemini-3-pro-preview":   {2.00, 12.00},
	"gemini-3-flash-preview": {0.50, 3.00},

	"gemini-2.5-pro":        {1.25, 10.00},
	"gemini-2.5-flash":      {0.30, 2.50},
	"gemini-2.5-flash-lite": {0.10, 0.40},

	"gemini-2.0-flash":      {0.10, 0.40},
	"gemini-2.0-flash-lite": {0.075, 0.30},

	"mistral-large-latest":  {4.00, 12.00},
	"mistral-medium-latest": {2.70, 8.10},
	"mistral-small-latest":  {1.00, 3.00},
	"open-mistral-7b":       {0.25, 0.25},
	"open-mixtral-8x7b":     {0.70, 0.70},
	"open-mixtral-8x22b":    {2.00, 6.00},

	"command-r-plus": {3.00, 15.00},
	"command-r":      {0.50, 1.50},
	"command":        {1.00, 2.00},
	"command-light":  {0.30, 0.60},
}

// EstimateCost calculates the estimated cost in USD for the given model and token counts
func EstimateCost(model string, inputTokens, outputTokens int64) *float64 {
	if model == "" {
		return nil
	}

	modelLower := strings.ToLower(model)

	// try exact match first
	pricing, ok := modelPricing[modelLower]
	if !ok {
		// try prefix match, find the longest matching prefix
		pricing, ok = findPricingByPrefix(modelLower)
		if !ok {
			return nil
		}
	}

	// Calculate cost: (tokens / 1M) * price_per_1M
	inputCost := (float64(inputTokens) / 1_000_000) * pricing.InputPer1M
	outputCost := (float64(outputTokens) / 1_000_000) * pricing.OutputPer1M
	totalCost := inputCost + outputCost

	// Round to 6 decimal places
	rounded := math.Round(totalCost*1_000_000) / 1_000_000

	return &rounded
}

// findPricingByPrefix finds the pricing for a model by matching the longest prefix.
func findPricingByPrefix(model string) (ModelPricing, bool) {
	var bestMatch string
	var bestPricing ModelPricing

	for key, pricing := range modelPricing {
		if strings.HasPrefix(model, key) {
			// Keep the longest matching prefix
			if len(key) > len(bestMatch) {
				bestMatch = key
				bestPricing = pricing
			}
		}
	}

	if bestMatch == "" {
		return ModelPricing{}, false
	}

	return bestPricing, true
}
