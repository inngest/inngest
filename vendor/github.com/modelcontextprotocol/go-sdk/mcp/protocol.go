// Copyright 2025 The Go MCP SDK Authors. All rights reserved.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.

package mcp

// Protocol types for version 2025-06-18.
// To see the schema changes from the previous version, run:
//
//   prefix=https://raw.githubusercontent.com/modelcontextprotocol/modelcontextprotocol/refs/heads/main/schema
//   sdiff -l <(curl $prefix/2025-03-26/schema.ts) <(curl $prefix/2025/06-18/schema.ts)

import (
	"encoding/json"
	"fmt"
	"maps"

	internaljson "github.com/modelcontextprotocol/go-sdk/internal/json"
)

// Optional annotations for the client. The client can use annotations to inform
// how objects are used or displayed.
type Annotations struct {
	// Describes who the intended customer of this object or data is.
	//
	// It can include multiple entries to indicate content useful for multiple
	// audiences (e.g., []Role{"user", "assistant"}).
	Audience []Role `json:"audience,omitempty"`
	// The moment the resource was last modified, as an ISO 8601 formatted string.
	//
	// Should be an ISO 8601 formatted string (e.g., "2025-01-12T15:00:58Z").
	//
	// Examples: last activity timestamp in an open file, timestamp when the
	// resource was attached, etc.
	LastModified string `json:"lastModified,omitempty"`
	// Describes how important this data is for operating the server.
	//
	// A value of 1 means "most important," and indicates that the data is
	// effectively required, while 0 means "least important," and indicates that the
	// data is entirely optional.
	Priority float64 `json:"priority,omitempty"`
}

// CallToolParams is used by clients to call a tool.
type CallToolParams struct {
	// Meta is reserved by the protocol to allow clients and servers to
	// attach additional metadata to their responses.
	Meta `json:"_meta,omitempty"`
	// Name is the name of the tool to call.
	Name string `json:"name"`
	// Arguments holds the tool arguments. It can hold any value that can be
	// marshaled to JSON.
	Arguments any `json:"arguments,omitempty"`
}

// CallToolParamsRaw is passed to tool handlers on the server. Its arguments
// are not yet unmarshaled (hence "raw"), so that the handlers can perform
// unmarshaling themselves.
type CallToolParamsRaw struct {
	// This property is reserved by the protocol to allow clients and servers to
	// attach additional metadata to their responses.
	Meta `json:"_meta,omitempty"`
	// Name is the name of the tool being called.
	Name string `json:"name"`
	// Arguments is the raw arguments received over the wire from the client. It
	// is the responsibility of the tool handler to unmarshal and validate the
	// Arguments (see [AddTool]).
	Arguments json.RawMessage `json:"arguments,omitempty"`
}

// A CallToolResult is the server's response to a tool call.
//
// The [ToolHandler] and [ToolHandlerFor] handler functions return this result,
// though [ToolHandlerFor] populates much of it automatically as documented at
// each field.
type CallToolResult struct {
	// This property is reserved by the protocol to allow clients and servers to
	// attach additional metadata to their responses.
	Meta `json:"_meta,omitempty"`

	// A list of content objects that represent the unstructured result of the tool
	// call.
	//
	// When using a [ToolHandlerFor] with structured output, if Content is unset
	// it will be populated with JSON text content corresponding to the
	// structured output value.
	Content []Content `json:"content"`

	// StructuredContent is an optional value that represents the structured
	// result of the tool call. It must marshal to a JSON object.
	//
	// When using a [ToolHandlerFor] with structured output, you should not
	// populate this field. It will be automatically populated with the typed Out
	// value.
	StructuredContent any `json:"structuredContent,omitempty"`

	// IsError reports whether the tool call ended in an error.
	//
	// If not set, this is assumed to be false (the call was successful).
	//
	// Any errors that originate from the tool should be reported inside the
	// Content field, with IsError set to true, not as an MCP protocol-level
	// error response. Otherwise, the LLM would not be able to see that an error
	// occurred and self-correct.
	//
	// However, any errors in finding the tool, an error indicating that the
	// server does not support tool calls, or any other exceptional conditions,
	// should be reported as an MCP error response.
	//
	// When using a [ToolHandlerFor], this field is automatically set when the
	// tool handler returns an error, and the error string is included as text in
	// the Content field.
	IsError bool `json:"isError,omitempty"`

	// The error passed to setError, if any.
	// It is not marshaled, and therefore it is only visible on the server.
	// Its only use is in server sending middleware, where it can be accessed
	// with getError.
	err error
}

// SetError sets the error for the tool result and populates the Content field
// with the error text. It also sets IsError to true.
func (r *CallToolResult) SetError(err error) {
	r.Content = []Content{&TextContent{Text: err.Error()}}
	r.IsError = true
	r.err = err
}

// GetError returns the error set with SetError, or nil if none.
// This function always returns nil on clients.
func (r *CallToolResult) GetError() error {
	return r.err
}

func (*CallToolResult) isResult() {}

// UnmarshalJSON handles the unmarshalling of content into the Content
// interface.
func (x *CallToolResult) UnmarshalJSON(data []byte) error {
	type res CallToolResult // avoid recursion
	var wire struct {
		res
		Content []*wireContent `json:"content"`
	}
	if err := internaljson.Unmarshal(data, &wire); err != nil {
		return err
	}
	var err error
	if wire.res.Content, err = contentsFromWire(wire.Content, nil); err != nil {
		return err
	}
	*x = CallToolResult(wire.res)
	return nil
}

func (x *CallToolParams) isParams()              {}
func (x *CallToolParams) GetProgressToken() any  { return getProgressToken(x) }
func (x *CallToolParams) SetProgressToken(t any) { setProgressToken(x, t) }

func (x *CallToolParamsRaw) isParams()              {}
func (x *CallToolParamsRaw) GetProgressToken() any  { return getProgressToken(x) }
func (x *CallToolParamsRaw) SetProgressToken(t any) { setProgressToken(x, t) }

type CancelledParams struct {
	// This property is reserved by the protocol to allow clients and servers to
	// attach additional metadata to their responses.
	Meta `json:"_meta,omitempty"`
	// An optional string describing the reason for the cancellation. This may be
	// logged or presented to the user.
	Reason string `json:"reason,omitempty"`
	// The ID of the request to cancel.
	//
	// This must correspond to the ID of a request previously issued in the same
	// direction.
	RequestID any `json:"requestId"`
}

func (x *CancelledParams) isParams()              {}
func (x *CancelledParams) GetProgressToken() any  { return getProgressToken(x) }
func (x *CancelledParams) SetProgressToken(t any) { setProgressToken(x, t) }

// RootCapabilities describes a client's support for roots.
type RootCapabilities struct {
	// ListChanged reports whether the client supports notifications for
	// changes to the roots list.
	ListChanged bool `json:"listChanged,omitempty"`
}

// Capabilities a client may support. Known capabilities are defined here, in
// this schema, but this is not a closed set: any client can define its own,
// additional capabilities.
type ClientCapabilities struct {
	// NOTE: any addition to ClientCapabilities must also be reflected in
	// [ClientCapabilities.clone].

	// Experimental reports non-standard capabilities that the client supports.
	// The caller should not modify the map after assigning it.
	Experimental map[string]any `json:"experimental,omitempty"`
	// Extensions reports extensions that the client supports.
	// Keys are extension identifiers in "{vendor-prefix}/{extension-name}" format.
	// Values are per-extension settings objects; use [ClientCapabilities.AddExtension]
	// to ensure nil settings are normalized to empty objects.
	// The caller should not modify the map or its values after assigning it.
	Extensions map[string]any `json:"extensions,omitempty"`
	// Roots describes the client's support for roots.
	//
	// Deprecated: use RootsV2. As described in #607, Roots should have been a
	// pointer to a RootCapabilities value. Roots will be continue to be
	// populated, but any new fields will only be added in the RootsV2 field.
	Roots struct {
		// ListChanged reports whether the client supports notifications for
		// changes to the roots list.
		ListChanged bool `json:"listChanged,omitempty"`
	} `json:"roots,omitempty"`
	// RootsV2 is present if the client supports roots. When capabilities are explicitly configured via [ClientOptions.Capabilities]
	RootsV2 *RootCapabilities `json:"-"`
	// Sampling is present if the client supports sampling from an LLM.
	Sampling *SamplingCapabilities `json:"sampling,omitempty"`
	// Elicitation is present if the client supports elicitation from the server.
	Elicitation *ElicitationCapabilities `json:"elicitation,omitempty"`
}

// AddExtension adds an extension with the given name and settings.
// If settings is nil, an empty map is used to ensure valid JSON serialization
// (the spec requires an object, not null).
// The settings map should not be modified after the call.
func (c *ClientCapabilities) AddExtension(name string, settings map[string]any) {
	if c.Extensions == nil {
		c.Extensions = make(map[string]any)
	}
	if settings == nil {
		settings = map[string]any{}
	}
	c.Extensions[name] = settings
}

// clone returns a copy of the ClientCapabilities.
// Values in the Extensions and Experimental maps are shallow-copied.
func (c *ClientCapabilities) clone() *ClientCapabilities {
	cp := *c
	cp.Experimental = maps.Clone(c.Experimental)
	cp.Extensions = maps.Clone(c.Extensions)
	cp.RootsV2 = shallowClone(c.RootsV2)
	if c.Sampling != nil {
		x := *c.Sampling
		x.Tools = shallowClone(c.Sampling.Tools)
		x.Context = shallowClone(c.Sampling.Context)
		cp.Sampling = &x
	}
	if c.Elicitation != nil {
		x := *c.Elicitation
		x.Form = shallowClone(c.Elicitation.Form)
		x.URL = shallowClone(c.Elicitation.URL)
		cp.Elicitation = &x
	}
	return &cp
}

// shallowClone returns a shallow clone of *p, or nil if p is nil.
func shallowClone[T any](p *T) *T {
	if p == nil {
		return nil
	}
	x := *p
	return &x
}

func (c *ClientCapabilities) toV2() *clientCapabilitiesV2 {
	return &clientCapabilitiesV2{
		ClientCapabilities: *c,
		Roots:              c.RootsV2,
	}
}

// clientCapabilitiesV2 is a version of ClientCapabilities that fixes the bug
// described in #607: Roots should have been a pointer to value type
// RootCapabilities.
type clientCapabilitiesV2 struct {
	ClientCapabilities
	Roots *RootCapabilities `json:"roots,omitempty"`
}

func (c *clientCapabilitiesV2) toV1() *ClientCapabilities {
	caps := c.ClientCapabilities
	caps.RootsV2 = c.Roots
	// Sync Roots from RootsV2 for backward compatibility (#607).
	if caps.RootsV2 != nil {
		caps.Roots = *caps.RootsV2
	}
	return &caps
}

type CompleteParamsArgument struct {
	// The name of the argument
	Name string `json:"name"`
	// The value of the argument to use for completion matching.
	Value string `json:"value"`
}

// CompleteContext represents additional, optional context for completions.
type CompleteContext struct {
	// Previously-resolved variables in a URI template or prompt.
	Arguments map[string]string `json:"arguments,omitempty"`
}

// CompleteReference represents a completion reference type (ref/prompt ref/resource).
// The Type field determines which other fields are relevant.
type CompleteReference struct {
	Type string `json:"type"`
	// Name is relevant when Type is "ref/prompt".
	Name string `json:"name,omitempty"`
	// URI is relevant when Type is "ref/resource".
	URI string `json:"uri,omitempty"`
}

func (r *CompleteReference) UnmarshalJSON(data []byte) error {
	type wireCompleteReference CompleteReference // for naive unmarshaling
	var r2 wireCompleteReference
	if err := internaljson.Unmarshal(data, &r2); err != nil {
		return err
	}
	switch r2.Type {
	case "ref/prompt", "ref/resource":
		if r2.Type == "ref/prompt" && r2.URI != "" {
			return fmt.Errorf("reference of type %q must not have a URI set", r2.Type)
		}
		if r2.Type == "ref/resource" && r2.Name != "" {
			return fmt.Errorf("reference of type %q must not have a Name set", r2.Type)
		}
	default:
		return fmt.Errorf("unrecognized content type %q", r2.Type)
	}
	*r = CompleteReference(r2)
	return nil
}

func (r *CompleteReference) MarshalJSON() ([]byte, error) {
	// Validation for marshalling: ensure consistency before converting to JSON.
	switch r.Type {
	case "ref/prompt":
		if r.URI != "" {
			return nil, fmt.Errorf("reference of type %q must not have a URI set for marshalling", r.Type)
		}
	case "ref/resource":
		if r.Name != "" {
			return nil, fmt.Errorf("reference of type %q must not have a Name set for marshalling", r.Type)
		}
	default:
		return nil, fmt.Errorf("unrecognized reference type %q for marshalling", r.Type)
	}

	type wireReference CompleteReference
	return json.Marshal(wireReference(*r))
}

type CompleteParams struct {
	// This property is reserved by the protocol to allow clients and servers to
	// attach additional metadata to their responses.
	Meta `json:"_meta,omitempty"`
	// The argument's information
	Argument CompleteParamsArgument `json:"argument"`
	Context  *CompleteContext       `json:"context,omitempty"`
	Ref      *CompleteReference     `json:"ref"`
}

func (*CompleteParams) isParams() {}

type CompletionResultDetails struct {
	HasMore bool     `json:"hasMore,omitempty"`
	Total   int      `json:"total,omitempty"`
	Values  []string `json:"values"`
}

// The server's response to a completion/complete request
type CompleteResult struct {
	// This property is reserved by the protocol to allow clients and servers to
	// attach additional metadata to their responses.
	Meta       `json:"_meta,omitempty"`
	Completion CompletionResultDetails `json:"completion"`
}

func (*CompleteResult) isResult() {}

type CreateMessageParams struct {
	// This property is reserved by the protocol to allow clients and servers to
	// attach additional metadata to their responses.
	Meta `json:"_meta,omitempty"`
	// A request to include context from one or more MCP servers (including the
	// caller), to be attached to the prompt. The client may ignore this request.
	//
	// The default is "none". Values "thisServer" and
	// "allServers" are soft-deprecated. Servers SHOULD only use these values if
	// the client declares ClientCapabilities.sampling.context. These values may
	// be removed in future spec releases.
	IncludeContext string `json:"includeContext,omitempty"`
	// The maximum number of tokens to sample, as requested by the server. The
	// client may choose to sample fewer tokens than requested.
	MaxTokens int64              `json:"maxTokens"`
	Messages  []*SamplingMessage `json:"messages"`
	// Optional metadata to pass through to the LLM provider. The format of this
	// metadata is provider-specific.
	Metadata any `json:"metadata,omitempty"`
	// The server's preferences for which model to select. The client may ignore
	// these preferences.
	ModelPreferences *ModelPreferences `json:"modelPreferences,omitempty"`
	StopSequences    []string          `json:"stopSequences,omitempty"`
	// An optional system prompt the server wants to use for sampling. The client
	// may modify or omit this prompt.
	SystemPrompt string  `json:"systemPrompt,omitempty"`
	Temperature  float64 `json:"temperature,omitempty"`
}

func (x *CreateMessageParams) isParams()              {}
func (x *CreateMessageParams) GetProgressToken() any  { return getProgressToken(x) }
func (x *CreateMessageParams) SetProgressToken(t any) { setProgressToken(x, t) }

// CreateMessageWithToolsParams is a sampling request that includes tools.
// It extends the basic [CreateMessageParams] fields with tools, tool choice,
// and messages that support array content (for parallel tool calls).
//
// Use with [ServerSession.CreateMessageWithTools].
type CreateMessageWithToolsParams struct {
	Meta           `json:"_meta,omitempty"`
	IncludeContext string `json:"includeContext,omitempty"`
	MaxTokens      int64  `json:"maxTokens"`
	// Messages supports array content for tool_use and tool_result blocks.
	Messages         []*SamplingMessageV2 `json:"messages"`
	Metadata         any                  `json:"metadata,omitempty"`
	ModelPreferences *ModelPreferences    `json:"modelPreferences,omitempty"`
	StopSequences    []string             `json:"stopSequences,omitempty"`
	SystemPrompt     string               `json:"systemPrompt,omitempty"`
	Temperature      float64              `json:"temperature,omitempty"`
	// Tools is the list of tools available for the model to use.
	Tools []*Tool `json:"tools,omitempty"`
	// ToolChoice controls how the model should use tools.
	ToolChoice *ToolChoice `json:"toolChoice,omitempty"`
}

func (x *CreateMessageWithToolsParams) isParams()              {}
func (x *CreateMessageWithToolsParams) GetProgressToken() any  { return getProgressToken(x) }
func (x *CreateMessageWithToolsParams) SetProgressToken(t any) { setProgressToken(x, t) }

// toBase converts to CreateMessageParams by taking the content block from each
// message. Tools and ToolChoice are dropped. Returns an error if any message
// has multiple content blocks, since SamplingMessage only supports one.
func (p *CreateMessageWithToolsParams) toBase() (*CreateMessageParams, error) {
	var msgs []*SamplingMessage
	for _, m := range p.Messages {
		if len(m.Content) > 1 {
			return nil, fmt.Errorf("message has %d content blocks; use CreateMessageWithToolsHandler to support multiple content", len(m.Content))
		}
		var content Content
		if len(m.Content) > 0 {
			content = m.Content[0]
		}
		msgs = append(msgs, &SamplingMessage{Content: content, Role: m.Role})
	}
	return &CreateMessageParams{
		Meta:             p.Meta,
		IncludeContext:   p.IncludeContext,
		MaxTokens:        p.MaxTokens,
		Messages:         msgs,
		Metadata:         p.Metadata,
		ModelPreferences: p.ModelPreferences,
		StopSequences:    p.StopSequences,
		SystemPrompt:     p.SystemPrompt,
		Temperature:      p.Temperature,
	}, nil
}

// SamplingMessageV2 describes a message issued to or received from an
// LLM API, supporting array content for parallel tool calls. The "V2" refers
// to the 2025-11-25 spec, which changed content from a single block to
// single-or-array. In v2 of the SDK, this will replace [SamplingMessage].
//
// When marshaling, a single-element Content slice is marshaled as a single
// object for compatibility with pre-2025-11-25 implementations. When
// unmarshaling, a single JSON content object is accepted and wrapped in a
// one-element slice.
type SamplingMessageV2 struct {
	Content []Content `json:"content"`
	Role    Role      `json:"role"`
}

var samplingWithToolsAllow = map[string]bool{
	"text": true, "image": true, "audio": true,
	"tool_use": true, "tool_result": true,
}

// MarshalJSON marshals the message. A single-element Content slice is marshaled
// as a single object for backward compatibility.
func (m *SamplingMessageV2) MarshalJSON() ([]byte, error) {
	if len(m.Content) == 1 {
		return json.Marshal(&SamplingMessage{Content: m.Content[0], Role: m.Role})
	}
	type msg SamplingMessageV2 // avoid recursion
	return json.Marshal((*msg)(m))
}

func (m *SamplingMessageV2) UnmarshalJSON(data []byte) error {
	type msg SamplingMessageV2 // avoid recursion
	var wire struct {
		msg
		Content json.RawMessage `json:"content"`
	}
	if err := internaljson.Unmarshal(data, &wire); err != nil {
		return err
	}
	var err error
	if wire.msg.Content, err = unmarshalContent(wire.Content, samplingWithToolsAllow); err != nil {
		return err
	}
	*m = SamplingMessageV2(wire.msg)
	return nil
}

// The client's response to a sampling/create_message request from the server.
// The client should inform the user before returning the sampled message, to
// allow them to inspect the response (human in the loop) and decide whether to
// allow the server to see it.
type CreateMessageResult struct {
	// This property is reserved by the protocol to allow clients and servers to
	// attach additional metadata to their responses.
	Meta    `json:"_meta,omitempty"`
	Content Content `json:"content"`
	// The name of the model that generated the message.
	Model string `json:"model"`
	Role  Role   `json:"role"`
	// The reason why sampling stopped, if known.
	//
	// Standard values:
	//  - "endTurn": natural end of the assistant's turn
	//  - "stopSequence": a stop sequence was encountered
	//  - "maxTokens": reached the maximum token limit
	//  - "toolUse": the model wants to use one or more tools
	StopReason string `json:"stopReason,omitempty"`
}

func (*CreateMessageResult) isResult() {}
func (r *CreateMessageResult) UnmarshalJSON(data []byte) error {
	type result CreateMessageResult // avoid recursion
	var wire struct {
		result
		Content *wireContent `json:"content"`
	}
	if err := internaljson.Unmarshal(data, &wire); err != nil {
		return err
	}
	var err error
	if wire.result.Content, err = contentFromWire(wire.Content, map[string]bool{"text": true, "image": true, "audio": true}); err != nil {
		return err
	}
	*r = CreateMessageResult(wire.result)
	return nil
}

// CreateMessageWithToolsResult is the client's response to a
// sampling/create_message request that included tools. Content is a slice to
// support parallel tool calls (multiple tool_use blocks in one response).
//
// Use [ServerSession.CreateMessageWithTools] to send a sampling request with
// tools and receive this result type.
//
// When unmarshaling, a single JSON content object is accepted and wrapped in a
// one-element slice, for compatibility with clients that return a single block.
type CreateMessageWithToolsResult struct {
	Meta    `json:"_meta,omitempty"`
	Content []Content `json:"content"`
	Model   string    `json:"model"`
	Role    Role      `json:"role"`
	// The reason why sampling stopped.
	//
	// Standard values: "endTurn", "stopSequence", "maxTokens", "toolUse".
	StopReason string `json:"stopReason,omitempty"`
}

// createMessageWithToolsResultAllow lists content types valid in assistant responses.
// tool_result is excluded: it only appears in user messages.
var createMessageWithToolsResultAllow = map[string]bool{
	"text": true, "image": true, "audio": true,
	"tool_use": true,
}

func (*CreateMessageWithToolsResult) isResult() {}

// MarshalJSON marshals the result. When Content has a single element, it is
// marshaled as a single object for compatibility with pre-2025-11-25
// implementations that expect a single content block.
func (r *CreateMessageWithToolsResult) MarshalJSON() ([]byte, error) {
	if len(r.Content) == 1 {
		return json.Marshal(&CreateMessageResult{
			Meta:       r.Meta,
			Content:    r.Content[0],
			Model:      r.Model,
			Role:       r.Role,
			StopReason: r.StopReason,
		})
	}
	type result CreateMessageWithToolsResult // avoid recursion
	return json.Marshal((*result)(r))
}

func (r *CreateMessageWithToolsResult) UnmarshalJSON(data []byte) error {
	type result CreateMessageWithToolsResult // avoid recursion
	var wire struct {
		result
		Content json.RawMessage `json:"content"`
	}
	if err := internaljson.Unmarshal(data, &wire); err != nil {
		return err
	}
	var err error
	if wire.result.Content, err = unmarshalContent(wire.Content, createMessageWithToolsResultAllow); err != nil {
		return err
	}
	*r = CreateMessageWithToolsResult(wire.result)
	return nil
}

// toWithTools converts a CreateMessageResult to CreateMessageWithToolsResult.
func (r *CreateMessageResult) toWithTools() *CreateMessageWithToolsResult {
	var content []Content
	if r.Content != nil {
		content = []Content{r.Content}
	}
	return &CreateMessageWithToolsResult{
		Meta:       r.Meta,
		Content:    content,
		Model:      r.Model,
		Role:       r.Role,
		StopReason: r.StopReason,
	}
}

type GetPromptParams struct {
	// This property is reserved by the protocol to allow clients and servers to
	// attach additional metadata to their responses.
	Meta `json:"_meta,omitempty"`
	// Arguments to use for templating the prompt.
	Arguments map[string]string `json:"arguments,omitempty"`
	// The name of the prompt or prompt template.
	Name string `json:"name"`
}

func (x *GetPromptParams) isParams()              {}
func (x *GetPromptParams) GetProgressToken() any  { return getProgressToken(x) }
func (x *GetPromptParams) SetProgressToken(t any) { setProgressToken(x, t) }

// The server's response to a prompts/get request from the client.
type GetPromptResult struct {
	// This property is reserved by the protocol to allow clients and servers to
	// attach additional metadata to their responses.
	Meta `json:"_meta,omitempty"`
	// An optional description for the prompt.
	Description string           `json:"description,omitempty"`
	Messages    []*PromptMessage `json:"messages"`
}

func (*GetPromptResult) isResult() {}

// InitializeParams is sent by the client to initialize the session.
type InitializeParams struct {
	// This property is reserved by the protocol to allow clients and servers to
	// attach additional metadata to their responses.
	Meta `json:"_meta,omitempty"`
	// Capabilities describes the client's capabilities.
	Capabilities *ClientCapabilities `json:"capabilities"`
	// ClientInfo provides information about the client.
	ClientInfo *Implementation `json:"clientInfo"`
	// ProtocolVersion is the latest version of the Model Context Protocol that
	// the client supports.
	ProtocolVersion string `json:"protocolVersion"`
}

func (p *InitializeParams) toV2() *initializeParamsV2 {
	return &initializeParamsV2{
		InitializeParams: *p,
		Capabilities:     p.Capabilities.toV2(),
	}
}

// initializeParamsV2 works around the mistake in #607: Capabilities.Roots
// should have been a pointer.
type initializeParamsV2 struct {
	InitializeParams
	Capabilities *clientCapabilitiesV2 `json:"capabilities"`
}

func (p *initializeParamsV2) toV1() *InitializeParams {
	p1 := p.InitializeParams
	if p.Capabilities != nil {
		p1.Capabilities = p.Capabilities.toV1()
	}
	return &p1
}

func (x *InitializeParams) isParams()              {}
func (x *InitializeParams) GetProgressToken() any  { return getProgressToken(x) }
func (x *InitializeParams) SetProgressToken(t any) { setProgressToken(x, t) }

// InitializeResult is sent by the server in response to an initialize request
// from the client.
type InitializeResult struct {
	// This property is reserved by the protocol to allow clients and servers to
	// attach additional metadata to their responses.
	Meta `json:"_meta,omitempty"`
	// Capabilities describes the server's capabilities.
	Capabilities *ServerCapabilities `json:"capabilities"`
	// Instructions describing how to use the server and its features.
	//
	// This can be used by clients to improve the LLM's understanding of available
	// tools, resources, etc. It can be thought of like a "hint" to the model. For
	// example, this information may be added to the system prompt.
	Instructions string `json:"instructions,omitempty"`
	// The version of the Model Context Protocol that the server wants to use. This
	// may not match the version that the client requested. If the client cannot
	// support this version, it must disconnect.
	ProtocolVersion string          `json:"protocolVersion"`
	ServerInfo      *Implementation `json:"serverInfo"`
}

func (*InitializeResult) isResult() {}

type InitializedParams struct {
	// Meta is reserved by the protocol to allow clients and servers to attach
	// additional metadata to their responses.
	Meta `json:"_meta,omitempty"`
}

func (x *InitializedParams) isParams()              {}
func (x *InitializedParams) GetProgressToken() any  { return getProgressToken(x) }
func (x *InitializedParams) SetProgressToken(t any) { setProgressToken(x, t) }

type ListPromptsParams struct {
	// This property is reserved by the protocol to allow clients and servers to
	// attach additional metadata to their responses.
	Meta `json:"_meta,omitempty"`
	// An opaque token representing the current pagination position. If provided,
	// the server should return results starting after this cursor.
	Cursor string `json:"cursor,omitempty"`
}

func (x *ListPromptsParams) isParams()              {}
func (x *ListPromptsParams) GetProgressToken() any  { return getProgressToken(x) }
func (x *ListPromptsParams) SetProgressToken(t any) { setProgressToken(x, t) }
func (x *ListPromptsParams) cursorPtr() *string     { return &x.Cursor }

// The server's response to a prompts/list request from the client.
type ListPromptsResult struct {
	// This property is reserved by the protocol to allow clients and servers to
	// attach additional metadata to their responses.
	Meta `json:"_meta,omitempty"`
	// An opaque token representing the pagination position after the last returned
	// result. If present, there may be more results available.
	NextCursor string    `json:"nextCursor,omitempty"`
	Prompts    []*Prompt `json:"prompts"`
}

func (x *ListPromptsResult) isResult()              {}
func (x *ListPromptsResult) nextCursorPtr() *string { return &x.NextCursor }

type ListResourceTemplatesParams struct {
	// This property is reserved by the protocol to allow clients and servers to
	// attach additional metadata to their responses.
	Meta `json:"_meta,omitempty"`
	// An opaque token representing the current pagination position. If provided,
	// the server should return results starting after this cursor.
	Cursor string `json:"cursor,omitempty"`
}

func (x *ListResourceTemplatesParams) isParams()              {}
func (x *ListResourceTemplatesParams) GetProgressToken() any  { return getProgressToken(x) }
func (x *ListResourceTemplatesParams) SetProgressToken(t any) { setProgressToken(x, t) }
func (x *ListResourceTemplatesParams) cursorPtr() *string     { return &x.Cursor }

// The server's response to a resources/templates/list request from the client.
type ListResourceTemplatesResult struct {
	// This property is reserved by the protocol to allow clients and servers to
	// attach additional metadata to their responses.
	Meta `json:"_meta,omitempty"`
	// An opaque token representing the pagination position after the last returned
	// result. If present, there may be more results available.
	NextCursor        string              `json:"nextCursor,omitempty"`
	ResourceTemplates []*ResourceTemplate `json:"resourceTemplates"`
}

func (x *ListResourceTemplatesResult) isResult()              {}
func (x *ListResourceTemplatesResult) nextCursorPtr() *string { return &x.NextCursor }

type ListResourcesParams struct {
	// This property is reserved by the protocol to allow clients and servers to
	// attach additional metadata to their responses.
	Meta `json:"_meta,omitempty"`
	// An opaque token representing the current pagination position. If provided,
	// the server should return results starting after this cursor.
	Cursor string `json:"cursor,omitempty"`
}

func (x *ListResourcesParams) isParams()              {}
func (x *ListResourcesParams) GetProgressToken() any  { return getProgressToken(x) }
func (x *ListResourcesParams) SetProgressToken(t any) { setProgressToken(x, t) }
func (x *ListResourcesParams) cursorPtr() *string     { return &x.Cursor }

// The server's response to a resources/list request from the client.
type ListResourcesResult struct {
	// This property is reserved by the protocol to allow clients and servers to
	// attach additional metadata to their responses.
	Meta `json:"_meta,omitempty"`
	// An opaque token representing the pagination position after the last returned
	// result. If present, there may be more results available.
	NextCursor string      `json:"nextCursor,omitempty"`
	Resources  []*Resource `json:"resources"`
}

func (x *ListResourcesResult) isResult()              {}
func (x *ListResourcesResult) nextCursorPtr() *string { return &x.NextCursor }

type ListRootsParams struct {
	// This property is reserved by the protocol to allow clients and servers to
	// attach additional metadata to their responses.
	Meta `json:"_meta,omitempty"`
}

func (x *ListRootsParams) isParams()              {}
func (x *ListRootsParams) GetProgressToken() any  { return getProgressToken(x) }
func (x *ListRootsParams) SetProgressToken(t any) { setProgressToken(x, t) }

// The client's response to a roots/list request from the server. This result
// contains an array of Root objects, each representing a root directory or file
// that the server can operate on.
type ListRootsResult struct {
	// This property is reserved by the protocol to allow clients and servers to
	// attach additional metadata to their responses.
	Meta  `json:"_meta,omitempty"`
	Roots []*Root `json:"roots"`
}

func (*ListRootsResult) isResult() {}

type ListToolsParams struct {
	// This property is reserved by the protocol to allow clients and servers to
	// attach additional metadata to their responses.
	Meta `json:"_meta,omitempty"`
	// An opaque token representing the current pagination position. If provided,
	// the server should return results starting after this cursor.
	Cursor string `json:"cursor,omitempty"`
}

func (x *ListToolsParams) isParams()              {}
func (x *ListToolsParams) GetProgressToken() any  { return getProgressToken(x) }
func (x *ListToolsParams) SetProgressToken(t any) { setProgressToken(x, t) }
func (x *ListToolsParams) cursorPtr() *string     { return &x.Cursor }

// The server's response to a tools/list request from the client.
type ListToolsResult struct {
	// This property is reserved by the protocol to allow clients and servers to
	// attach additional metadata to their responses.
	Meta `json:"_meta,omitempty"`
	// An opaque token representing the pagination position after the last returned
	// result. If present, there may be more results available.
	NextCursor string  `json:"nextCursor,omitempty"`
	Tools      []*Tool `json:"tools"`
}

func (x *ListToolsResult) isResult()              {}
func (x *ListToolsResult) nextCursorPtr() *string { return &x.NextCursor }

// The severity of a log message.
//
// These map to syslog message severities, as specified in RFC-5424:
// https://datatracker.ietf.org/doc/html/rfc5424#section-6.2.1
type LoggingLevel string

type LoggingMessageParams struct {
	// This property is reserved by the protocol to allow clients and servers to
	// attach additional metadata to their responses.
	Meta `json:"_meta,omitempty"`
	// The data to be logged, such as a string message or an object. Any JSON
	// serializable type is allowed here.
	Data any `json:"data"`
	// The severity of this log message.
	Level LoggingLevel `json:"level"`
	// An optional name of the logger issuing this message.
	Logger string `json:"logger,omitempty"`
}

func (x *LoggingMessageParams) isParams()              {}
func (x *LoggingMessageParams) GetProgressToken() any  { return getProgressToken(x) }
func (x *LoggingMessageParams) SetProgressToken(t any) { setProgressToken(x, t) }

// Hints to use for model selection.
//
// Keys not declared here are currently left unspecified by the spec and are up
// to the client to interpret.
type ModelHint struct {
	// A hint for a model name.
	//
	// The client should treat this as a substring of a model name; for example: -
	// `claude-3-5-sonnet` should match `claude-3-5-sonnet-20241022` - `sonnet`
	// should match `claude-3-5-sonnet-20241022`, `claude-3-sonnet-20240229`, etc. -
	// `claude` should match any Claude model
	//
	// The client may also map the string to a different provider's model name or a
	// different model family, as long as it fills a similar niche; for example: -
	// `gemini-1.5-flash` could match `claude-3-haiku-20240307`
	Name string `json:"name,omitempty"`
}

// The server's preferences for model selection, requested of the client during
// sampling.
//
// Because LLMs can vary along multiple dimensions, choosing the "best" model is
// rarely straightforward. Different models excel in different areas—some are
// faster but less capable, others are more capable but more expensive, and so
// on. This interface allows servers to express their priorities across multiple
// dimensions to help clients make an appropriate selection for their use case.
//
// These preferences are always advisory. The client may ignore them. It is also
// up to the client to decide how to interpret these preferences and how to
// balance them against other considerations.
type ModelPreferences struct {
	// How much to prioritize cost when selecting a model. A value of 0 means cost
	// is not important, while a value of 1 means cost is the most important factor.
	CostPriority float64 `json:"costPriority,omitempty"`
	// Optional hints to use for model selection.
	//
	// If multiple hints are specified, the client must evaluate them in order (such
	// that the first match is taken).
	//
	// The client should prioritize these hints over the numeric priorities, but may
	// still use the priorities to select from ambiguous matches.
	Hints []*ModelHint `json:"hints,omitempty"`
	// How much to prioritize intelligence and capabilities when selecting a model.
	// A value of 0 means intelligence is not important, while a value of 1 means
	// intelligence is the most important factor.
	IntelligencePriority float64 `json:"intelligencePriority,omitempty"`
	// How much to prioritize sampling speed (latency) when selecting a model. A
	// value of 0 means speed is not important, while a value of 1 means speed is
	// the most important factor.
	SpeedPriority float64 `json:"speedPriority,omitempty"`
}

type PingParams struct {
	// This property is reserved by the protocol to allow clients and servers to
	// attach additional metadata to their responses.
	Meta `json:"_meta,omitempty"`
}

func (x *PingParams) isParams()              {}
func (x *PingParams) GetProgressToken() any  { return getProgressToken(x) }
func (x *PingParams) SetProgressToken(t any) { setProgressToken(x, t) }

type ProgressNotificationParams struct {
	// This property is reserved by the protocol to allow clients and servers to
	// attach additional metadata to their responses.
	Meta `json:"_meta,omitempty"`
	// The progress token which was given in the initial request, used to associate
	// this notification with the request that is proceeding.
	ProgressToken any `json:"progressToken"`
	// An optional message describing the current progress.
	Message string `json:"message,omitempty"`
	// The progress thus far. This should increase every time progress is made, even
	// if the total is unknown.
	Progress float64 `json:"progress"`
	// Total number of items to process (or total progress required), if known.
	// Zero means unknown.
	Total float64 `json:"total,omitempty"`
}

func (*ProgressNotificationParams) isParams() {}

// IconTheme specifies the theme an icon is designed for.
type IconTheme string

const (
	// IconThemeLight indicates the icon is designed for a light background.
	IconThemeLight IconTheme = "light"
	// IconThemeDark indicates the icon is designed for a dark background.
	IconThemeDark IconTheme = "dark"
)

// Icon provides visual identifiers for their resources, tools, prompts, and implementations
// See [/specification/draft/basic/index#icons] for notes on icons
//
// TODO(iamsurajbobade): update specification url from draft.
type Icon struct {
	// Source is A URI pointing to the icon resource (required). This can be:
	// - An HTTP/HTTPS URL pointing to an image file
	// - A data URI with base64-encoded image data
	Source string `json:"src"`
	// Optional MIME type if the server's type is missing or generic
	MIMEType string `json:"mimeType,omitempty"`
	// Optional size specification (e.g., ["48x48"], ["any"] for scalable formats like SVG, or ["48x48", "96x96"] for multiple sizes)
	Sizes []string `json:"sizes,omitempty"`
	// Optional theme specifier. "light" indicates the icon is designed for a light
	// background, "dark" indicates the icon is designed for a dark background.
	Theme IconTheme `json:"theme,omitempty"`
}

// A prompt or prompt template that the server offers.
type Prompt struct {
	// See [specification/2025-06-18/basic/index#general-fields] for notes on _meta
	// usage.
	Meta `json:"_meta,omitempty"`
	// A list of arguments to use for templating the prompt.
	Arguments []*PromptArgument `json:"arguments,omitempty"`
	// An optional description of what this prompt provides
	Description string `json:"description,omitempty"`
	// Intended for programmatic or logical use, but used as a display name in past
	// specs or fallback (if title isn't present).
	Name string `json:"name"`
	// Intended for UI and end-user contexts — optimized to be human-readable and
	// easily understood, even by those unfamiliar with domain-specific terminology.
	Title string `json:"title,omitempty"`
	// Icons for the prompt, if any.
	Icons []Icon `json:"icons,omitempty"`
}

// Describes an argument that a prompt can accept.
type PromptArgument struct {
	// Intended for programmatic or logical use, but used as a display name in past
	// specs or fallback (if title isn't present).
	Name string `json:"name"`
	// Intended for UI and end-user contexts — optimized to be human-readable and
	// easily understood, even by those unfamiliar with domain-specific terminology.
	Title string `json:"title,omitempty"`
	// A human-readable description of the argument.
	Description string `json:"description,omitempty"`
	// Whether this argument must be provided.
	Required bool `json:"required,omitempty"`
}

type PromptListChangedParams struct {
	// This property is reserved by the protocol to allow clients and servers to
	// attach additional metadata to their responses.
	Meta `json:"_meta,omitempty"`
}

func (x *PromptListChangedParams) isParams()              {}
func (x *PromptListChangedParams) GetProgressToken() any  { return getProgressToken(x) }
func (x *PromptListChangedParams) SetProgressToken(t any) { setProgressToken(x, t) }

// Describes a message returned as part of a prompt.
//
// This is similar to SamplingMessage, but also supports the embedding of
// resources from the MCP server.
type PromptMessage struct {
	Content Content `json:"content"`
	Role    Role    `json:"role"`
}

// UnmarshalJSON handles the unmarshalling of content into the Content
// interface.
func (m *PromptMessage) UnmarshalJSON(data []byte) error {
	type msg PromptMessage // avoid recursion
	var wire struct {
		msg
		Content *wireContent `json:"content"`
	}
	if err := internaljson.Unmarshal(data, &wire); err != nil {
		return err
	}
	var err error
	if wire.msg.Content, err = contentFromWire(wire.Content, nil); err != nil {
		return err
	}
	*m = PromptMessage(wire.msg)
	return nil
}

type ReadResourceParams struct {
	// This property is reserved by the protocol to allow clients and servers to
	// attach additional metadata to their responses.
	Meta `json:"_meta,omitempty"`
	// The URI of the resource to read. The URI can use any protocol; it is up to
	// the server how to interpret it.
	URI string `json:"uri"`
}

func (x *ReadResourceParams) isParams()              {}
func (x *ReadResourceParams) GetProgressToken() any  { return getProgressToken(x) }
func (x *ReadResourceParams) SetProgressToken(t any) { setProgressToken(x, t) }

// The server's response to a resources/read request from the client.
type ReadResourceResult struct {
	// This property is reserved by the protocol to allow clients and servers to
	// attach additional metadata to their responses.
	Meta     `json:"_meta,omitempty"`
	Contents []*ResourceContents `json:"contents"`
}

func (*ReadResourceResult) isResult() {}

// A known resource that the server is capable of reading.
type Resource struct {
	// See [specification/2025-06-18/basic/index#general-fields] for notes on _meta
	// usage.
	Meta `json:"_meta,omitempty"`
	// Optional annotations for the client.
	Annotations *Annotations `json:"annotations,omitempty"`
	// A description of what this resource represents.
	//
	// This can be used by clients to improve the LLM's understanding of available
	// resources. It can be thought of like a "hint" to the model.
	Description string `json:"description,omitempty"`
	// The MIME type of this resource, if known.
	MIMEType string `json:"mimeType,omitempty"`
	// Intended for programmatic or logical use, but used as a display name in past
	// specs or fallback (if title isn't present).
	Name string `json:"name"`
	// The size of the raw resource content, in bytes (i.e., before base64 encoding
	// or any tokenization), if known.
	//
	// This can be used by Hosts to display file sizes and estimate context window
	// usage.
	Size int64 `json:"size,omitempty"`
	// Intended for UI and end-user contexts — optimized to be human-readable and
	// easily understood, even by those unfamiliar with domain-specific terminology.
	//
	// If not provided, the name should be used for display (except for Tool, where
	// Annotations.Title should be given precedence over using name, if
	// present).
	Title string `json:"title,omitempty"`
	// The URI of this resource.
	URI string `json:"uri"`
	// Icons for the resource, if any.
	Icons []Icon `json:"icons,omitempty"`
}

type ResourceListChangedParams struct {
	// This property is reserved by the protocol to allow clients and servers to
	// attach additional metadata to their responses.
	Meta `json:"_meta,omitempty"`
}

func (x *ResourceListChangedParams) isParams()              {}
func (x *ResourceListChangedParams) GetProgressToken() any  { return getProgressToken(x) }
func (x *ResourceListChangedParams) SetProgressToken(t any) { setProgressToken(x, t) }

// A template description for resources available on the server.
type ResourceTemplate struct {
	// See [specification/2025-06-18/basic/index#general-fields] for notes on _meta
	// usage.
	Meta `json:"_meta,omitempty"`
	// Optional annotations for the client.
	Annotations *Annotations `json:"annotations,omitempty"`
	// A description of what this template is for.
	//
	// This can be used by clients to improve the LLM's understanding of available
	// resources. It can be thought of like a "hint" to the model.
	Description string `json:"description,omitempty"`
	// The MIME type for all resources that match this template. This should only be
	// included if all resources matching this template have the same type.
	MIMEType string `json:"mimeType,omitempty"`
	// Intended for programmatic or logical use, but used as a display name in past
	// specs or fallback (if title isn't present).
	Name string `json:"name"`
	// Intended for UI and end-user contexts — optimized to be human-readable and
	// easily understood, even by those unfamiliar with domain-specific terminology.
	//
	// If not provided, the name should be used for display (except for Tool, where
	// Annotations.Title should be given precedence over using name, if
	// present).
	Title string `json:"title,omitempty"`
	// A URI template (according to RFC 6570) that can be used to construct resource
	// URIs.
	URITemplate string `json:"uriTemplate"`
	// Icons for the resource template, if any.
	Icons []Icon `json:"icons,omitempty"`
}

// The sender or recipient of messages and data in a conversation.
type Role string

// Represents a root directory or file that the server can operate on.
type Root struct {
	// See [specification/2025-06-18/basic/index#general-fields] for notes on _meta
	// usage.
	Meta `json:"_meta,omitempty"`
	// An optional name for the root. This can be used to provide a human-readable
	// identifier for the root, which may be useful for display purposes or for
	// referencing the root in other parts of the application.
	Name string `json:"name,omitempty"`
	// The URI identifying the root. This *must* start with file:// for now. This
	// restriction may be relaxed in future versions of the protocol to allow other
	// URI schemes.
	URI string `json:"uri"`
}

type RootsListChangedParams struct {
	// This property is reserved by the protocol to allow clients and servers to
	// attach additional metadata to their responses.
	Meta `json:"_meta,omitempty"`
}

func (x *RootsListChangedParams) isParams()              {}
func (x *RootsListChangedParams) GetProgressToken() any  { return getProgressToken(x) }
func (x *RootsListChangedParams) SetProgressToken(t any) { setProgressToken(x, t) }

// TODO: to be consistent with ServerCapabilities, move the capability types
// below directly above ClientCapabilities.

// SamplingCapabilities describes the client's support for sampling.
type SamplingCapabilities struct {
	// Context indicates the client supports includeContext values other than "none".
	Context *SamplingContextCapabilities `json:"context,omitempty"`
	// Tools indicates the client supports tools and toolChoice in sampling requests.
	Tools *SamplingToolsCapabilities `json:"tools,omitempty"`
}

// SamplingContextCapabilities indicates the client supports context inclusion.
type SamplingContextCapabilities struct{}

// SamplingToolsCapabilities indicates the client supports tool use in sampling.
type SamplingToolsCapabilities struct{}

// ToolChoice controls how the model uses tools during sampling.
type ToolChoice struct {
	// Mode controls tool invocation behavior:
	//  - "auto": Model decides whether to use tools (default)
	//  - "required": Model must use at least one tool
	//  - "none": Model must not use any tools
	Mode string `json:"mode,omitempty"`
}

// ElicitationCapabilities describes the capabilities for elicitation.
//
// If neither Form nor URL is set, the 'Form' capabilitiy is assumed.
type ElicitationCapabilities struct {
	Form *FormElicitationCapabilities `json:"form,omitempty"`
	URL  *URLElicitationCapabilities  `json:"url,omitempty"`
}

// FormElicitationCapabilities describes capabilities for form elicitation.
type FormElicitationCapabilities struct{}

// URLElicitationCapabilities describes capabilities for url elicitation.
type URLElicitationCapabilities struct{}

// Describes a message issued to or received from an LLM API.
//
// For assistant messages, Content may be text, image, audio, or tool_use.
// For user messages, Content may be text, image, audio, or tool_result.
type SamplingMessage struct {
	Content Content `json:"content"`
	Role    Role    `json:"role"`
}

// UnmarshalJSON handles the unmarshalling of content into the Content
// interface.
func (m *SamplingMessage) UnmarshalJSON(data []byte) error {
	type msg SamplingMessage // avoid recursion
	var wire struct {
		msg
		Content *wireContent `json:"content"`
	}
	if err := internaljson.Unmarshal(data, &wire); err != nil {
		return err
	}
	// Allow text, image, audio, tool_use, and tool_result in sampling messages
	var err error
	if wire.msg.Content, err = contentFromWire(wire.Content, map[string]bool{"text": true, "image": true, "audio": true, "tool_use": true, "tool_result": true}); err != nil {
		return err
	}
	*m = SamplingMessage(wire.msg)
	return nil
}

type SetLoggingLevelParams struct {
	// This property is reserved by the protocol to allow clients and servers to
	// attach additional metadata to their responses.
	Meta `json:"_meta,omitempty"`
	// The level of logging that the client wants to receive from the server. The
	// server should send all logs at this level and higher (i.e., more severe) to
	// the client as notifications/message.
	Level LoggingLevel `json:"level"`
}

func (x *SetLoggingLevelParams) isParams()              {}
func (x *SetLoggingLevelParams) GetProgressToken() any  { return getProgressToken(x) }
func (x *SetLoggingLevelParams) SetProgressToken(t any) { setProgressToken(x, t) }

// Definition for a tool the client can call.
type Tool struct {
	// See [specification/2025-06-18/basic/index#general-fields] for notes on _meta
	// usage.
	Meta `json:"_meta,omitempty"`
	// Optional additional tool information.
	//
	// Display name precedence order is: title, annotations.title, then name.
	Annotations *ToolAnnotations `json:"annotations,omitempty"`
	// A human-readable description of the tool.
	//
	// This can be used by clients to improve the LLM's understanding of available
	// tools. It can be thought of like a "hint" to the model.
	Description string `json:"description,omitempty"`
	// InputSchema holds a JSON Schema object defining the expected parameters
	// for the tool.
	//
	// From the server, this field may be set to any value that JSON-marshals to
	// valid JSON schema (including json.RawMessage). However, for tools added
	// using [AddTool], which automatically validates inputs and outputs, the
	// schema must be in a draft the SDK understands. Currently, the SDK uses
	// github.com/google/jsonschema-go for inference and validation, which only
	// supports the 2020-12 draft of JSON schema. To do your own validation, use
	// [Server.AddTool].
	//
	// From the client, this field will hold the default JSON marshaling of the
	// server's input schema (a map[string]any).
	InputSchema any `json:"inputSchema"`
	// Intended for programmatic or logical use, but used as a display name in past
	// specs or fallback (if title isn't present).
	Name string `json:"name"`
	// OutputSchema holds an optional JSON Schema object defining the structure
	// of the tool's output returned in the StructuredContent field of a
	// CallToolResult.
	//
	// From the server, this field may be set to any value that JSON-marshals to
	// valid JSON schema (including json.RawMessage). However, for tools added
	// using [AddTool], which automatically validates inputs and outputs, the
	// schema must be in a draft the SDK understands. Currently, the SDK uses
	// github.com/google/jsonschema-go for inference and validation, which only
	// supports the 2020-12 draft of JSON schema. To do your own validation, use
	// [Server.AddTool].
	//
	// From the client, this field will hold the default JSON marshaling of the
	// server's output schema (a map[string]any).
	OutputSchema any `json:"outputSchema,omitempty"`
	// Intended for UI and end-user contexts — optimized to be human-readable and
	// easily understood, even by those unfamiliar with domain-specific terminology.
	// If not provided, Annotations.Title should be used for display if present,
	// otherwise Name.
	Title string `json:"title,omitempty"`
	// Icons for the tool, if any.
	Icons []Icon `json:"icons,omitempty"`
}

// Additional properties describing a Tool to clients.
//
// NOTE: all properties in ToolAnnotations are hints. They are not
// guaranteed to provide a faithful description of tool behavior (including
// descriptive properties like title).
//
// Clients should never make tool use decisions based on ToolAnnotations
// received from untrusted servers.
type ToolAnnotations struct {
	// If true, the tool may perform destructive updates to its environment. If
	// false, the tool performs only additive updates.
	//
	// (This property is meaningful only when ReadOnlyHint == false.)
	//
	// Default: true
	DestructiveHint *bool `json:"destructiveHint,omitempty"`
	// If true, calling the tool repeatedly with the same arguments will have no
	// additional effect on the its environment.
	//
	// (This property is meaningful only when ReadOnlyHint == false.)
	//
	// Default: false
	IdempotentHint bool `json:"idempotentHint,omitempty"`
	// If true, this tool may interact with an "open world" of external entities. If
	// false, the tool's domain of interaction is closed. For example, the world of
	// a web search tool is open, whereas that of a memory tool is not.
	//
	// Default: true
	OpenWorldHint *bool `json:"openWorldHint,omitempty"`
	// If true, the tool does not modify its environment.
	//
	// Default: false
	ReadOnlyHint bool `json:"readOnlyHint,omitempty"`
	// A human-readable title for the tool.
	Title string `json:"title,omitempty"`
}

type ToolListChangedParams struct {
	// This property is reserved by the protocol to allow clients and servers to
	// attach additional metadata to their responses.
	Meta `json:"_meta,omitempty"`
}

func (x *ToolListChangedParams) isParams()              {}
func (x *ToolListChangedParams) GetProgressToken() any  { return getProgressToken(x) }
func (x *ToolListChangedParams) SetProgressToken(t any) { setProgressToken(x, t) }

// Sent from the client to request resources/updated notifications from the
// server whenever a particular resource changes.
type SubscribeParams struct {
	// This property is reserved by the protocol to allow clients and servers to
	// attach additional metadata to their responses.
	Meta `json:"_meta,omitempty"`
	// The URI of the resource to subscribe to.
	URI string `json:"uri"`
}

func (*SubscribeParams) isParams() {}

// Sent from the client to request cancellation of resources/updated
// notifications from the server. This should follow a previous
// resources/subscribe request.
type UnsubscribeParams struct {
	// This property is reserved by the protocol to allow clients and servers to
	// attach additional metadata to their responses.
	Meta `json:"_meta,omitempty"`
	// The URI of the resource to unsubscribe from.
	URI string `json:"uri"`
}

func (*UnsubscribeParams) isParams() {}

// A notification from the server to the client, informing it that a resource
// has changed and may need to be read again. This should only be sent if the
// client previously sent a resources/subscribe request.
type ResourceUpdatedNotificationParams struct {
	// This property is reserved by the protocol to allow clients and servers to
	// attach additional metadata to their responses.
	Meta `json:"_meta,omitempty"`
	// The URI of the resource that has been updated. This might be a sub-resource of the one that the client actually subscribed to.
	URI string `json:"uri"`
}

func (*ResourceUpdatedNotificationParams) isParams() {}

// TODO(jba): add CompleteRequest and related types.

// A request from the server to elicit additional information from the user via the client.
type ElicitParams struct {
	// This property is reserved by the protocol to allow clients and servers to
	// attach additional metadata to their responses.
	Meta `json:"_meta,omitempty"`
	// The mode of elicitation to use.
	//
	// If unset, will be inferred from the other fields.
	Mode string `json:"mode"`
	// The message to present to the user.
	Message string `json:"message"`
	// A JSON schema object defining the requested elicitation schema.
	//
	// From the server, this field may be set to any value that can JSON-marshal
	// to valid JSON schema (including json.RawMessage for raw schema values).
	// Internally, the SDK uses github.com/google/jsonschema-go for validation,
	// which only supports the 2020-12 draft of the JSON schema spec.
	//
	// From the client, this field will use the default JSON marshaling (a
	// map[string]any).
	//
	// Only top-level properties are allowed, without nesting.
	//
	// This is only used for "form" elicitation.
	RequestedSchema any `json:"requestedSchema,omitempty"`
	// The URL to present to the user.
	//
	// This is only used for "url" elicitation.
	URL string `json:"url,omitempty"`
	// The ID of the elicitation.
	//
	// This is only used for "url" elicitation.
	ElicitationID string `json:"elicitationId,omitempty"`
}

func (x *ElicitParams) isParams() {}

func (x *ElicitParams) GetProgressToken() any  { return getProgressToken(x) }
func (x *ElicitParams) SetProgressToken(t any) { setProgressToken(x, t) }

// The client's response to an elicitation/create request from the server.
type ElicitResult struct {
	// This property is reserved by the protocol to allow clients and servers to
	// attach additional metadata to their responses.
	Meta `json:"_meta,omitempty"`
	// The user action in response to the elicitation.
	// - "accept": User submitted the form/confirmed the action
	// - "decline": User explicitly declined the action
	// - "cancel": User dismissed without making an explicit choice
	Action string `json:"action"`
	// The submitted form data, only present when action is "accept".
	// Contains values matching the requested schema.
	Content map[string]any `json:"content,omitempty"`
}

func (*ElicitResult) isResult() {}

// ElicitationCompleteParams is sent from the server to the client, informing it that an out-of-band elicitation interaction has completed.
type ElicitationCompleteParams struct {
	// This property is reserved by the protocol to allow clients and servers to
	// attach additional metadata to their responses.
	Meta `json:"_meta,omitempty"`
	// The ID of the elicitation that has completed. This must correspond to the
	// elicitationId from the original elicitation/create request.
	ElicitationID string `json:"elicitationId"`
}

func (*ElicitationCompleteParams) isParams() {}

// An Implementation describes the name and version of an MCP implementation, with an optional
// title for UI representation.
type Implementation struct {
	// Intended for programmatic or logical use, but used as a display name in past
	// specs or fallback (if title isn't present).
	Name string `json:"name"`
	// Intended for UI and end-user contexts — optimized to be human-readable and
	// easily understood, even by those unfamiliar with domain-specific terminology.
	Title   string `json:"title,omitempty"`
	Version string `json:"version"`
	// WebsiteURL for the server, if any.
	WebsiteURL string `json:"websiteUrl,omitempty"`
	// Icons for the Server, if any.
	Icons []Icon `json:"icons,omitempty"`
}

// CompletionCapabilities describes the server's support for argument autocompletion.
type CompletionCapabilities struct{}

// LoggingCapabilities describes the server's support for sending log messages to the client.
type LoggingCapabilities struct{}

// PromptCapabilities describes the server's support for prompts.
type PromptCapabilities struct {
	// Whether this server supports notifications for changes to the prompt list.
	ListChanged bool `json:"listChanged,omitempty"`
}

// ResourceCapabilities describes the server's support for resources.
type ResourceCapabilities struct {
	// ListChanged reports whether the client supports notifications for
	// changes to the resource list.
	ListChanged bool `json:"listChanged,omitempty"`
	// Subscribe reports whether this server supports subscribing to resource
	// updates.
	Subscribe bool `json:"subscribe,omitempty"`
}

// ToolCapabilities describes the server's support for tools.
type ToolCapabilities struct {
	// ListChanged reports whether the client supports notifications for
	// changes to the tool list.
	ListChanged bool `json:"listChanged,omitempty"`
}

// ServerCapabilities describes capabilities that a server supports.
type ServerCapabilities struct {
	// NOTE: any addition to ServerCapabilities must also be reflected in
	// [ServerCapabilities.clone].

	// Experimental reports non-standard capabilities that the server supports.
	// The caller should not modify the map after assigning it.
	Experimental map[string]any `json:"experimental,omitempty"`
	// Extensions reports extensions that the server supports.
	// Keys are extension identifiers in "{vendor-prefix}/{extension-name}" format.
	// Values are per-extension settings objects; use [ServerCapabilities.AddExtension]
	// to ensure nil settings are normalized to empty objects.
	// The caller should not modify the map or its values after assigning it.
	Extensions map[string]any `json:"extensions,omitempty"`
	// Completions is present if the server supports argument autocompletion
	// suggestions.
	Completions *CompletionCapabilities `json:"completions,omitempty"`
	// Logging is present if the server supports log messages.
	Logging *LoggingCapabilities `json:"logging,omitempty"`
	// Prompts is present if the server supports prompts.
	Prompts *PromptCapabilities `json:"prompts,omitempty"`
	// Resources is present if the server supports resourcs.
	Resources *ResourceCapabilities `json:"resources,omitempty"`
	// Tools is present if the supports tools.
	Tools *ToolCapabilities `json:"tools,omitempty"`
}

// AddExtension adds an extension with the given name and settings.
// If settings is nil, an empty map is used to ensure valid JSON serialization
// (the spec requires an object, not null).
// The settings map should not be modified after the call.
func (c *ServerCapabilities) AddExtension(name string, settings map[string]any) {
	if c.Extensions == nil {
		c.Extensions = make(map[string]any)
	}
	if settings == nil {
		settings = map[string]any{}
	}
	c.Extensions[name] = settings
}

// clone returns a copy of the ServerCapabilities.
// Values in the Extensions and Experimental maps are shallow-copied.
func (c *ServerCapabilities) clone() *ServerCapabilities {
	cp := *c
	cp.Experimental = maps.Clone(c.Experimental)
	cp.Extensions = maps.Clone(c.Extensions)
	cp.Completions = shallowClone(c.Completions)
	cp.Logging = shallowClone(c.Logging)
	cp.Prompts = shallowClone(c.Prompts)
	cp.Resources = shallowClone(c.Resources)
	cp.Tools = shallowClone(c.Tools)
	return &cp
}

const (
	methodCallTool                  = "tools/call"
	notificationCancelled           = "notifications/cancelled"
	methodComplete                  = "completion/complete"
	methodCreateMessage             = "sampling/createMessage"
	methodElicit                    = "elicitation/create"
	notificationElicitationComplete = "notifications/elicitation/complete"
	methodGetPrompt                 = "prompts/get"
	methodInitialize                = "initialize"
	notificationInitialized         = "notifications/initialized"
	methodListPrompts               = "prompts/list"
	methodListResourceTemplates     = "resources/templates/list"
	methodListResources             = "resources/list"
	methodListRoots                 = "roots/list"
	methodListTools                 = "tools/list"
	notificationLoggingMessage      = "notifications/message"
	methodPing                      = "ping"
	notificationProgress            = "notifications/progress"
	notificationPromptListChanged   = "notifications/prompts/list_changed"
	methodReadResource              = "resources/read"
	notificationResourceListChanged = "notifications/resources/list_changed"
	notificationResourceUpdated     = "notifications/resources/updated"
	notificationRootsListChanged    = "notifications/roots/list_changed"
	methodSetLevel                  = "logging/setLevel"
	methodSubscribe                 = "resources/subscribe"
	notificationToolListChanged     = "notifications/tools/list_changed"
	methodUnsubscribe               = "resources/unsubscribe"
)
