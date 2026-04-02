package anthropic

import (
	"context"
	"encoding/json"
	"net/http"
)

type MessagesResponseType string

const (
	MessagesResponseTypeMessage MessagesResponseType = "message"
	MessagesResponseTypeError   MessagesResponseType = "error"
)

type MessagesContentType string

const (
	MessagesContentTypeText             MessagesContentType = "text"
	MessagesContentTypeTextDelta        MessagesContentType = "text_delta"
	MessagesContentTypeImage            MessagesContentType = "image"
	MessagesContentTypeToolResult       MessagesContentType = "tool_result"
	MessagesContentTypeToolUse          MessagesContentType = "tool_use"
	MessagesContentTypeInputJsonDelta   MessagesContentType = "input_json_delta"
	MessagesContentTypeDocument         MessagesContentType = "document"
	MessagesContentTypeCitationsDelta   MessagesContentType = "citations_delta"
	MessagesContentTypeThinking         MessagesContentType = "thinking"
	MessagesContentTypeThinkingDelta    MessagesContentType = "thinking_delta"
	MessagesContentTypeSignatureDelta   MessagesContentType = "signature_delta"
	MessagesContentTypeRedactedThinking MessagesContentType = "redacted_thinking"
)

type CitationType string

const (
	CitationTypeCharLocation CitationType = "char_location"
	CitationTypePageNumber   CitationType = "page_number"
	CitationTypeBlockIndex   CitationType = "block_index"
)

type ThinkingType string

const (
	ThinkingTypeEnabled  ThinkingType = "enabled"
	ThinkingTypeDisabled ThinkingType = "disabled"
)

type MessagesStopReason string

const (
	MessagesStopReasonEndTurn      MessagesStopReason = "end_turn"
	MessagesStopReasonMaxTokens    MessagesStopReason = "max_tokens"
	MessagesStopReasonStopSequence MessagesStopReason = "stop_sequence"
	MessagesStopReasonToolUse      MessagesStopReason = "tool_use"
	MessagesStopRefusal            MessagesStopReason = "refusal"
)

type MessagesContentSourceType string

const (
	MessagesContentSourceTypeBase64  MessagesContentSourceType = "base64"
	MessagesContentSourceTypeText    MessagesContentSourceType = "text"
	MessagesContentSourceTypeContent MessagesContentSourceType = "content"
	MessagesContentSourceTypeUrl     MessagesContentSourceType = "url"
)

type OutputFormatType string

const (
	OutputFormatJsonSchema OutputFormatType = "json_schema"
)

type DocumentCitations struct {
	Enabled bool `json:"enabled"`
}

type MessagesRequest struct {
	Model            Model     `json:"model,omitempty"`
	AnthropicVersion string    `json:"anthropic_version,omitempty"`
	Messages         []Message `json:"messages"`
	MaxTokens        int       `json:"max_tokens,omitempty"`

	System        string              `json:"-"`
	MultiSystem   []MessageSystemPart `json:"-"`
	Metadata      map[string]any      `json:"metadata,omitempty"`
	StopSequences []string            `json:"stop_sequences,omitempty"`
	Stream        bool                `json:"stream,omitempty"`
	Temperature   *float32            `json:"temperature,omitempty"`
	TopP          *float32            `json:"top_p,omitempty"`
	TopK          *int                `json:"top_k,omitempty"`
	Tools         []ToolDefinition    `json:"tools,omitempty"`
	ToolChoice    *ToolChoice         `json:"tool_choice,omitempty"`
	Thinking      *Thinking           `json:"thinking,omitempty"`
	OutputFormat  *OutputFormat       `json:"output_format,omitempty"`
}

func (m MessagesRequest) MarshalJSON() ([]byte, error) {
	type Alias MessagesRequest
	aux := struct {
		System interface{} `json:"system,omitempty"`
		Alias
	}{
		Alias: (Alias)(m),
	}

	if len(m.MultiSystem) > 0 {
		aux.System = m.MultiSystem
	} else if len(m.System) > 0 {
		aux.System = m.System
	}

	return json.Marshal(aux)
}

var _ VertexAISupport = (*MessagesRequest)(nil)

func (m MessagesRequest) GetModel() Model {
	return m.Model
}

func (m *MessagesRequest) SetAnthropicVersion(version APIVersion) {
	m.AnthropicVersion = string(version)
	m.Model = ""
}

func (m *MessagesRequest) SetTemperature(t float32) {
	m.Temperature = &t
}

func (m *MessagesRequest) SetTopP(p float32) {
	m.TopP = &p
}

func (m *MessagesRequest) SetTopK(k int) {
	m.TopK = &k
}

func (m *MessagesRequest) IsStreaming() bool {
	return m.Stream
}

type MessageSystemPart struct {
	Type         string               `json:"type"`
	Text         string               `json:"text"`
	CacheControl *MessageCacheControl `json:"cache_control,omitempty"`
}

func NewMultiSystemMessages(texts ...string) []MessageSystemPart {
	var systemParts []MessageSystemPart
	for _, text := range texts {
		systemParts = append(systemParts, NewSystemMessagePart(text))
	}
	return systemParts
}

func NewSystemMessagePart(text string) MessageSystemPart {
	return MessageSystemPart{
		Type: "text",
		Text: text,
	}
}

type Message struct {
	Role    ChatRole         `json:"role"`
	Content []MessageContent `json:"content"`
}

func NewUserTextMessage(text string) Message {
	return Message{
		Role:    RoleUser,
		Content: []MessageContent{NewTextMessageContent(text)},
	}
}

func NewAssistantTextMessage(text string) Message {
	return Message{
		Role:    RoleAssistant,
		Content: []MessageContent{NewTextMessageContent(text)},
	}
}

func NewToolResultsMessage(toolUseID, content string, isError bool) Message {
	return Message{
		Role:    RoleUser,
		Content: []MessageContent{NewToolResultMessageContent(toolUseID, content, isError)},
	}
}

func (m Message) GetFirstContent() MessageContent {
	if len(m.Content) == 0 {
		return MessageContent{}
	}
	return m.Content[0]
}

type CacheControlType string

const (
	CacheControlTypeEphemeral CacheControlType = "ephemeral"
)

type MessageCacheControl struct {
	Type CacheControlType `json:"type"`
}

type Citation struct {
	Type          CitationType `json:"type"`
	CitedText     string       `json:"cited_text"`
	DocumentIndex int          `json:"document_index"`
	DocumentTitle string       `json:"document_title,omitempty"`

	// For char_location citations
	StartCharIndex *int `json:"start_char_index,omitempty"`
	EndCharIndex   *int `json:"end_char_index,omitempty"`

	// For page_number citations
	StartPage *int `json:"start_page,omitempty"`
	EndPage   *int `json:"end_page,omitempty"`

	// For block_index citations
	StartBlockIndex *int `json:"start_block_index,omitempty"`
	EndBlockIndex   *int `json:"end_block_index,omitempty"`
}

type MessageContent struct {
	Type MessagesContentType `json:"type"`

	Text *string `json:"text,omitempty"`

	Source *MessageContentSource `json:"source,omitempty"`

	*MessageContentToolResult

	*MessageContentToolUse

	PartialJson *string `json:"partial_json,omitempty"`

	CacheControl *MessageCacheControl `json:"cache_control,omitempty"`

	// Given the nature of the API and the MessageContent's struct multiple duties,
	// we have to override the standard json unmarshalling behavior from API responses to handle citations.
	// See UnmarshalJSON below where we give this the alias of citations during unmarshalling.
	Citations []Citation `json:"citations_list,omitempty"`

	// For citations_delta events in streaming
	Citation *Citation `json:"citation_delta,omitempty"`

	// For document content
	Title             string             `json:"title,omitempty"`
	Context           string             `json:"context,omitempty"`
	DocumentCitations *DocumentCitations `json:"citations,omitempty"` // Used in requests

	// Thinking-related fields
	*MessageContentThinking

	*MessageContentRedactedThinking
}

// UnmarshalJSON implements custom JSON unmarshaling for MessageContent
func (m *MessageContent) UnmarshalJSON(data []byte) error {
	type Alias MessageContent
	aux := &struct {
		Citations []Citation `json:"citations"`
		*Alias
	}{
		Alias: (*Alias)(m),
	}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}

	// Copy Citations from aux to m
	m.Citations = aux.Citations

	return nil
}

func NewTextMessageContent(text string) MessageContent {
	return MessageContent{
		Type:      MessagesContentTypeText,
		Text:      &text,
		Citations: make([]Citation, 0),
	}
}

func NewImageMessageContent(source MessageContentSource) MessageContent {
	return MessageContent{
		Type:   MessagesContentTypeImage,
		Source: &source,
	}
}

func NewImageUrlMessageContent(url string) MessageContent {
	return MessageContent{
		Type: MessagesContentTypeImage,
		Source: &MessageContentSource{
			Type: MessagesContentSourceTypeUrl,
			Url:  url,
		},
	}
}

func NewDocumentMessageContent(
	source MessageContentSource,
	title, context string,
	enableCitations bool,
) MessageContent {
	return MessageContent{
		Type:    MessagesContentTypeDocument,
		Source:  &source,
		Title:   title,
		Context: context,
		DocumentCitations: &DocumentCitations{
			Enabled: enableCitations,
		},
	}
}

func NewPDFDocumentMessageContent(
	base64EncodedPDFData, title, context string,
	enableCitations bool,
) MessageContent {
	return NewDocumentMessageContent(
		MessageContentSource{
			Type:      MessagesContentSourceTypeBase64,
			MediaType: "application/pdf",
			Data:      base64EncodedPDFData,
		},
		title,
		context,
		enableCitations,
	)
}

func NewTextDocumentMessageContent(
	text, title, context string,
	enableCitations bool,
) MessageContent {
	return NewDocumentMessageContent(
		MessageContentSource{
			Type:      MessagesContentSourceTypeText,
			MediaType: "text/plain",
			Data:      text,
		},
		title,
		context,
		enableCitations,
	)
}

func NewCustomContentDocumentMessageContent(
	content []MessageContent,
	title, context string,
	enableCitations bool,
) MessageContent {
	return NewDocumentMessageContent(
		MessageContentSource{
			Type:    MessagesContentSourceTypeContent,
			Content: content,
		},
		title,
		context,
		enableCitations,
	)
}

func NewToolResultMessageContent(toolUseID, content string, isError bool) MessageContent {
	return MessageContent{
		Type:                     MessagesContentTypeToolResult,
		MessageContentToolResult: NewMessageContentToolResult(toolUseID, content, isError),
	}
}

func NewToolUseMessageContent(toolUseID, name string, input json.RawMessage) MessageContent {
	return MessageContent{
		Type:                  MessagesContentTypeToolUse,
		MessageContentToolUse: NewMessageContentToolUse(toolUseID, name, input),
	}
}

func (m *MessageContent) SetCacheControl(ts ...CacheControlType) {
	t := CacheControlTypeEphemeral
	if len(ts) > 0 {
		t = ts[0]
	}
	m.CacheControl = &MessageCacheControl{
		Type: t,
	}
}

func (m *MessageContent) GetText() string {
	if m.Text != nil {
		return *m.Text
	}
	return ""
}

func (m *MessageContent) ConcatText(s string) {
	if m.Text == nil {
		m.Text = &s
	} else {
		*m.Text += s
	}
}

func (m *MessageContent) MergeContentDelta(mc MessageContent) {
	switch mc.Type {
	case MessagesContentTypeText:
		m.ConcatText(mc.GetText())
	case MessagesContentTypeTextDelta:
		m.ConcatText(mc.GetText())
	case MessagesContentTypeImage:
		m.Source = mc.Source
	case MessagesContentTypeToolResult:
		m.MessageContentToolResult = mc.MessageContentToolResult
	case MessagesContentTypeToolUse:
		m.MessageContentToolUse = &MessageContentToolUse{
			ID:   mc.MessageContentToolUse.ID,
			Name: mc.MessageContentToolUse.Name,
		}
	case MessagesContentTypeInputJsonDelta:
		if m.PartialJson == nil {
			m.PartialJson = mc.PartialJson
		} else {
			*m.PartialJson += *mc.PartialJson
		}
	case MessagesContentTypeCitationsDelta:
		if mc.Citation != nil {
			if m.Citations == nil {
				m.Citations = make([]Citation, 0)
			}
			m.Citations = append(m.Citations, *mc.Citation)
		}
	case MessagesContentTypeThinking,
		MessagesContentTypeThinkingDelta,
		MessagesContentTypeSignatureDelta:
		if m.MessageContentThinking == nil {
			m.MessageContentThinking = mc.MessageContentThinking
		} else {
			m.MessageContentThinking.Thinking += mc.MessageContentThinking.Thinking
			if mc.MessageContentThinking.Signature != "" {
				m.MessageContentThinking.Signature = mc.MessageContentThinking.Signature
			}
		}
	}
}

type MessageContentToolResult struct {
	ToolUseID *string          `json:"tool_use_id,omitempty"`
	Content   []MessageContent `json:"content,omitempty"`
	IsError   *bool            `json:"is_error,omitempty"`
}

func NewMessageContentToolResult(
	toolUseID, content string,
	isError bool,
) *MessageContentToolResult {
	return &MessageContentToolResult{
		ToolUseID: &toolUseID,
		Content: []MessageContent{
			{
				Type: MessagesContentTypeText,
				Text: &content,
			},
		},
		IsError: &isError,
	}
}

type MessageContentSource struct {
	Type      MessagesContentSourceType `json:"type"`
	MediaType string                    `json:"media_type,omitempty"`
	Data      any                       `json:"data,omitempty"`
	Content   []MessageContent          `json:"content,omitempty"`
	Url       string                    `json:"url,omitempty"`
}

func NewMessageContentSource(
	sourceType MessagesContentSourceType,
	mediaType string,
	data any,
) MessageContentSource {
	return MessageContentSource{
		Type:      sourceType,
		MediaType: mediaType,
		Data:      data,
	}
}

type MessageContentToolUse struct {
	ID    string          `json:"id"`
	Name  string          `json:"name"`
	Input json.RawMessage `json:"input"`
}

func NewMessageContentToolUse(
	toolUseId, name string,
	input json.RawMessage,
) *MessageContentToolUse {
	if input == nil {
		input = json.RawMessage(`{}`)
	}

	return &MessageContentToolUse{
		ID:    toolUseId,
		Name:  name,
		Input: input,
	}
}

func (c *MessageContentToolUse) UnmarshalInput(v any) error {
	return json.Unmarshal(c.Input, v)
}

type MessageContentThinking struct {
	Thinking  string `json:"thinking,omitempty"`
	Signature string `json:"signature,omitempty"`
}

type MessageContentRedactedThinking struct {
	Data string `json:"data,omitempty"`
}

type MessagesResponse struct {
	httpHeader

	ID           string               `json:"id"`
	Type         MessagesResponseType `json:"type"`
	Role         ChatRole             `json:"role"`
	Content      []MessageContent     `json:"content"`
	Model        Model                `json:"model"`
	StopReason   MessagesStopReason   `json:"stop_reason"`
	StopSequence string               `json:"stop_sequence"`
	Usage        MessagesUsage        `json:"usage"`
}

// GetFirstContentText get Content[0].Text avoid panic
func (m MessagesResponse) GetFirstContentText() string {
	if len(m.Content) == 0 {
		return ""
	}
	return m.Content[0].GetText()
}

type MessagesUsage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`

	// The number of tokens written to the cache when creating a new entry.
	CacheCreationInputTokens int `json:"cache_creation_input_tokens,omitempty"`
	// The number of tokens retrieved from the cache for associated request.
	CacheReadInputTokens int `json:"cache_read_input_tokens,omitempty"`
}

type ToolDefinition struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	// docs: https://platform.claude.com/docs/en/api/python/messages/create#tool.eager_input_streaming
	EagerInputStreaming *bool `json:"eager_input_streaming,omitempty"`
	// docs: https://platform.claude.com/docs/en/api/messages/create#tool.strict
	Strict *bool `json:"strict,omitempty"`
	// InputSchema is an object describing the tool.
	// You can pass json.RawMessage to describe the schema,
	// or you can pass in a struct which serializes to the proper JSON schema.
	// The jsonschema package is provided for convenience, but you should
	// consider another specialized library if you require more complex schemas.
	InputSchema any `json:"input_schema,omitempty"`

	CacheControl *MessageCacheControl `json:"cache_control,omitempty"`

	// Type is required for Anthropic defined tools.
	Type string `json:"type,omitempty"`
	// DisplayWidthPx is a required parameter of the Computer Use tool.
	DisplayWidthPx int `json:"display_width_px,omitempty"`
	// DisplayHeightPx is a required parameter of the Computer Use tool.
	DisplayHeightPx int `json:"display_height_px,omitempty"`
	// DisplayNumber is an optional parameter of the Computer Use tool.
	DisplayNumber *int `json:"display_number,omitempty"`
}

func NewComputerUseToolDefinition(
	name string,
	displayWidthPx int,
	displayHeightPx int,
	displayNumber *int,
) ToolDefinition {
	return ToolDefinition{
		Type:            "computer_20241022",
		Name:            name,
		DisplayWidthPx:  displayWidthPx,
		DisplayHeightPx: displayHeightPx,
		DisplayNumber:   displayNumber,
	}
}

func NewTextEditorToolDefinition(name string) ToolDefinition {
	return ToolDefinition{
		Type: "text_editor_20241022",
		Name: name,
	}
}

func NewBashToolDefinition(name string) ToolDefinition {
	return ToolDefinition{
		Type: "bash_20241022",
		Name: name,
	}
}

type ToolChoice struct {
	// oneof: auto(default) any tool
	Type string `json:"type"`
	Name string `json:"name,omitempty"`
}

type Thinking struct {
	Type ThinkingType `json:"type"`
	// Determines how many tokens Claude can use for its internal reasoning process. Larger budgets can enable more thorough analysis for complex problems, improving response quality.
	// Must be ≥1024 and less than max_tokens.
	BudgetTokens int `json:"budget_tokens"`
}

type OutputFormat struct {
	Type   OutputFormatType `json:"type"`
	Schema json.Marshaler   `json:"schema"`
}

func (c *Client) CreateMessages(
	ctx context.Context,
	request MessagesRequest,
) (response MessagesResponse, err error) {
	request.Stream = false

	var setters []requestSetter
	if len(c.config.BetaVersion) > 0 {
		setters = append(setters, withBetaVersion(c.config.BetaVersion...))
	}

	urlSuffix := "/messages"

	req, err := c.newRequest(ctx, http.MethodPost, urlSuffix, &request, setters...)
	if err != nil {
		return
	}

	err = c.sendRequest(req, &response)
	return
}
