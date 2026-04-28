package anthropic

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"slices"
)

var (
	eventPrefix                   = []byte("event:")
	dataPrefix                    = []byte("data:")
	ErrTooManyEmptyStreamMessages = errors.New("stream has sent too many empty messages")
)

type (
	// MessagesEvent docs: https://docs.anthropic.com/claude/reference/messages-streaming
	MessagesEvent string
)

const (
	MessagesEventError             MessagesEvent = "error"
	MessagesEventMessageStart      MessagesEvent = "message_start"
	MessagesEventContentBlockStart MessagesEvent = "content_block_start"
	MessagesEventPing              MessagesEvent = "ping"
	MessagesEventContentBlockDelta MessagesEvent = "content_block_delta"
	MessagesEventContentBlockStop  MessagesEvent = "content_block_stop"
	MessagesEventMessageDelta      MessagesEvent = "message_delta"
	MessagesEventMessageStop       MessagesEvent = "message_stop"
)

type MessagesStreamRequest struct {
	MessagesRequest

	OnError             func(ErrorResponse)                                     `json:"-"`
	OnPing              func(MessagesEventPingData)                             `json:"-"`
	OnMessageStart      func(MessagesEventMessageStartData)                     `json:"-"`
	OnContentBlockStart func(MessagesEventContentBlockStartData)                `json:"-"`
	OnContentBlockDelta func(MessagesEventContentBlockDeltaData)                `json:"-"`
	OnContentBlockStop  func(MessagesEventContentBlockStopData, MessageContent) `json:"-"`
	OnMessageDelta      func(MessagesEventMessageDeltaData)                     `json:"-"`
	OnMessageStop       func(MessagesEventMessageStopData)                      `json:"-"`
}

type MessagesEventMessageStartData struct {
	Type    MessagesEvent    `json:"type"`
	Message MessagesResponse `json:"message"`
}

type MessagesEventContentBlockStartData struct {
	Type         MessagesEvent  `json:"type"`
	Index        int            `json:"index"`
	ContentBlock MessageContent `json:"content_block"`
}

type MessagesEventPingData struct {
	Type string `json:"type"`
}

type MessagesEventContentBlockDeltaData struct {
	Type  string         `json:"type"`
	Index int            `json:"index"`
	Delta MessageContent `json:"delta"`
}

// UnmarshalJSON implements custom JSON unmarshaling for MessagesEventContentBlockDeltaData.
// This is necessary because the MessageContent struct uses the same JSON field "citations"
// for different purposes:
// - In requests: maps to DocumentCitations (a flag to enable citations)
// - In responses: maps to Citations slice (actual citation data)
//
// For streaming, citation events arrive with type "citations_delta", but the citation data
// isn't properly captured by the default unmarshaler. This custom implementation ensures
// that citation data in streaming events is properly extracted and made available through
// the Citation field, allowing MergeContentDelta to add it to the Citations slice.
func (m *MessagesEventContentBlockDeltaData) UnmarshalJSON(data []byte) error {
	type Alias MessagesEventContentBlockDeltaData
	aux := &struct {
		*Alias
	}{
		Alias: (*Alias)(m),
	}

	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}

	if m.Delta.Type == MessagesContentTypeCitationsDelta {
		var rawData map[string]json.RawMessage
		if err := json.Unmarshal(data, &rawData); err != nil {
			return err
		}

		if deltaRaw, ok := rawData["delta"]; ok {
			var deltaMap map[string]json.RawMessage
			if err := json.Unmarshal(deltaRaw, &deltaMap); err != nil {
				return err
			}

			if citationsRaw, ok := deltaMap["citations"]; ok {
				var citations []Citation
				if err := json.Unmarshal(citationsRaw, &citations); err == nil &&
					len(citations) > 0 {
					m.Delta.Citation = &citations[0]
				}
			}

			if citationRaw, ok := deltaMap["citation"]; ok {
				var citation Citation
				if err := json.Unmarshal(citationRaw, &citation); err == nil {
					m.Delta.Citation = &citation
				}
			}
		}
	}

	return nil
}

type MessagesEventContentBlockStopData struct {
	Type  string `json:"type"`
	Index int    `json:"index"`
}

type MessagesEventMessageDeltaData struct {
	Type  string           `json:"type"`
	Delta MessagesResponse `json:"delta"`
	Usage MessagesUsage    `json:"usage"`
}

type MessagesEventMessageStopData struct {
	Type string `json:"type"`
}

func (c *Client) CreateMessagesStream(
	ctx context.Context,
	request MessagesStreamRequest,
) (response MessagesResponse, err error) {
	request.Stream = true

	var setters []requestSetter
	if len(c.config.BetaVersion) > 0 {
		setters = append(setters, withBetaVersion(c.config.BetaVersion...))
	}

	urlSuffix := "/messages"

	req, err := c.newStreamRequest(ctx, http.MethodPost, urlSuffix, &request, setters...)
	if err != nil {
		return
	}

	resp, err := c.config.HTTPClient.Do(req)
	if err != nil {
		return
	}
	defer resp.Body.Close()

	response.SetHeader(resp.Header)

	if err := c.handlerRequestError(resp); err != nil {
		return response, err
	}

	reader := bufio.NewReader(resp.Body)
	var (
		event             []byte
		emptyMessageCount uint
	)
	for {
		rawLine, readErr := reader.ReadBytes('\n')
		if readErr != nil {
			if errors.Is(readErr, io.EOF) {
				break
			}
			return response, readErr
		}

		noSpaceLine := bytes.TrimSpace(rawLine)
		if len(noSpaceLine) == 0 {
			continue
		}
		if bytes.HasPrefix(noSpaceLine, eventPrefix) {
			event = bytes.TrimSpace(bytes.TrimPrefix(noSpaceLine, eventPrefix))
			continue
		}
		if bytes.HasPrefix(noSpaceLine, dataPrefix) {
			var (
				data      = bytes.TrimPrefix(noSpaceLine, dataPrefix)
				eventType = MessagesEvent(event)
			)
			switch eventType {
			case MessagesEventError:
				var eventData ErrorResponse
				if err := json.Unmarshal(data, &eventData); err != nil {
					return response, err
				}
				if request.OnError != nil {
					request.OnError(eventData)
				}
				return response, eventData.Error
			case MessagesEventPing:
				var d MessagesEventPingData
				if err := json.Unmarshal(data, &d); err != nil {
					return response, err
				}
				if request.OnPing != nil {
					request.OnPing(d)
				}
				continue
			case MessagesEventMessageStart:
				var d MessagesEventMessageStartData
				if err := json.Unmarshal(data, &d); err != nil {
					return response, err
				}
				if request.OnMessageStart != nil {
					request.OnMessageStart(d)
				}
				response = d.Message
				response.SetHeader(resp.Header)
				continue
			case MessagesEventContentBlockStart:
				var d MessagesEventContentBlockStartData
				if err := json.Unmarshal(data, &d); err != nil {
					return response, err
				}
				if request.OnContentBlockStart != nil {
					request.OnContentBlockStart(d)
				}
				response.Content = slices.Insert(response.Content, d.Index, d.ContentBlock)
				continue
			case MessagesEventContentBlockDelta:
				var d MessagesEventContentBlockDeltaData
				if err := json.Unmarshal(data, &d); err != nil {
					return response, err
				}
				if request.OnContentBlockDelta != nil {
					request.OnContentBlockDelta(d)
				}
				if len(response.Content)-1 < d.Index {
					response.Content = slices.Insert(response.Content, d.Index, d.Delta)
				} else {
					response.Content[d.Index].MergeContentDelta(d.Delta)
				}
				continue
			case MessagesEventContentBlockStop:
				var d MessagesEventContentBlockStopData
				if err := json.Unmarshal(data, &d); err != nil {
					return response, err
				}
				var stopContent MessageContent
				if len(response.Content) > d.Index {
					stopContent = response.Content[d.Index]
					if stopContent.Type == MessagesContentTypeToolUse {
						if stopContent.PartialJson != nil {
							stopContent.Input = json.RawMessage(*stopContent.PartialJson)
						}
						stopContent.PartialJson = nil
						response.Content[d.Index] = stopContent
					}
				}
				if request.OnContentBlockStop != nil {
					request.OnContentBlockStop(d, stopContent)
				}
				continue
			case MessagesEventMessageDelta:
				var d MessagesEventMessageDeltaData
				if err := json.Unmarshal(data, &d); err != nil {
					return response, err
				}
				if request.OnMessageDelta != nil {
					request.OnMessageDelta(d)
				}
				response.StopReason = d.Delta.StopReason
				response.StopSequence = d.Delta.StopSequence
				response.Usage.OutputTokens = d.Usage.OutputTokens
				continue
			case MessagesEventMessageStop:
				var d MessagesEventMessageStopData
				if err := json.Unmarshal(data, &d); err != nil {
					return response, err
				}
				if request.OnMessageStop != nil {
					request.OnMessageStop(d)
				}
				continue
			}
		}
		emptyMessageCount++
		if emptyMessageCount > c.config.EmptyMessagesLimit {
			return response, ErrTooManyEmptyStreamMessages
		}
	}
	return
}
