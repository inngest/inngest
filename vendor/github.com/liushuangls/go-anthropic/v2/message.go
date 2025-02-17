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
	MessagesContentTypeText           MessagesContentType = "text"
	MessagesContentTypeTextDelta      MessagesContentType = "text_delta"
	MessagesContentTypeImage          MessagesContentType = "image"
	MessagesContentTypeToolResult     MessagesContentType = "tool_result"
	MessagesContentTypeToolUse        MessagesContentType = "tool_use"
	MessagesContentTypeInputJsonDelta MessagesContentType = "input_json_delta"
	MessagesContentTypeDocument       MessagesContentType = "document"
)

type MessagesStopReason string

const (
	MessagesStopReasonEndTurn      MessagesStopReason = "end_turn"
	MessagesStopReasonMaxTokens    MessagesStopReason = "max_tokens"
	MessagesStopReasonStopSequence MessagesStopReason = "stop_sequence"
	MessagesStopReasonToolUse      MessagesStopReason = "tool_use"
)

type MessagesContentSourceType string

const (
	MessagesContentSourceTypeBase64 = "base64"
)

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
}

func (m MessagesRequest) MarshalJSON() ([]byte, error) {
	type Alias MessagesRequest
	aux := struct {
		System interface{} `json:"system,omitempty"`
		Alias
	}{
		Alias: (Alias)(m),
	}

	// 根据 MultiSystem 是否为空来设置 system 字段
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

type MessageContent struct {
	Type MessagesContentType `json:"type"`

	Text *string `json:"text,omitempty"`

	Source *MessageContentSource `json:"source,omitempty"`

	*MessageContentToolResult

	*MessageContentToolUse

	PartialJson *string `json:"partial_json,omitempty"`

	CacheControl *MessageCacheControl `json:"cache_control,omitempty"`
}

func NewTextMessageContent(text string) MessageContent {
	return MessageContent{
		Type: MessagesContentTypeText,
		Text: &text,
	}
}

func NewImageMessageContent(source MessageContentSource) MessageContent {
	return MessageContent{
		Type:   MessagesContentTypeImage,
		Source: &source,
	}
}

func NewDocumentMessageContent(source MessageContentSource) MessageContent {
	return MessageContent{
		Type:   MessagesContentTypeDocument,
		Source: &source,
	}
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
	MediaType string                    `json:"media_type"`
	Data      any                       `json:"data"`
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
	ID    string          `json:"id,omitempty"`
	Name  string          `json:"name,omitempty"`
	Input json.RawMessage `json:"input,omitempty"`
}

func NewMessageContentToolUse(
	toolUseId, name string,
	input json.RawMessage,
) *MessageContentToolUse {
	return &MessageContentToolUse{
		ID:    toolUseId,
		Name:  name,
		Input: input,
	}
}

func (c *MessageContentToolUse) UnmarshalInput(v any) error {
	return json.Unmarshal(c.Input, v)
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
	// InputSchema is an object describing the tool.
	// You can pass json.RawMessage to describe the schema,
	// or you can pass in a struct which serializes to the proper JSON schema.
	// The jsonschema package is provided for convenience, but you should
	// consider another specialized library if you require more complex schemas.
	InputSchema any `json:"input_schema"`

	CacheControl *MessageCacheControl `json:"cache_control,omitempty"`
}

type ToolChoice struct {
	// oneof: auto(default) any tool
	Type string `json:"type"`
	Name string `json:"name,omitempty"`
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
