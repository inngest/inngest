// Copyright 2025 The Go MCP SDK Authors. All rights reserved.
// Use of this source code is governed by the license
// that can be found in the LICENSE file.

package mcp

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	internaljson "github.com/modelcontextprotocol/go-sdk/internal/json"
	"github.com/modelcontextprotocol/go-sdk/jsonrpc"
)

const (
	protocolVersionHeader        = "Mcp-Protocol-Version"
	sessionIDHeader              = "Mcp-Session-Id"
	lastEventIDHeader            = "Last-Event-ID"
	methodHeader                 = "Mcp-Method"
	nameHeader                   = "Mcp-Name"
	minVersionForStandardHeaders = protocolVersion20260630
)

func extractName(method string, params json.RawMessage) (string, bool) {
	switch method {
	case "tools/call":
		var p CallToolParams
		if err := internaljson.Unmarshal(params, &p); err == nil {
			return p.Name, true
		}
	case "prompts/get":
		var p GetPromptParams
		if err := internaljson.Unmarshal(params, &p); err == nil {
			return p.Name, true
		}
	case "resources/read":
		var p ReadResourceParams
		if err := internaljson.Unmarshal(params, &p); err == nil {
			return p.URI, true
		}
	}

	return "", false
}

// setStandardHeaders populates standard MCP headers.
// It requires the protocol version header to be set.
func setStandardHeaders(header http.Header, msg jsonrpc.Message) {
	if msg == nil {
		return
	}
	if header.Get(protocolVersionHeader) == "" || header.Get(protocolVersionHeader) < minVersionForStandardHeaders {
		return
	}

	switch msg := msg.(type) {
	case *jsonrpc.Request:
		header.Set(methodHeader, msg.Method)
		if name, ok := extractName(msg.Method, msg.Params); ok {
			header.Set(nameHeader, name)
		}
	}
}

func validateMcpHeaders(header http.Header, msg jsonrpc.Message) error {
	protocolVersion := header.Get(protocolVersionHeader)
	if protocolVersion == "" || protocolVersion < minVersionForStandardHeaders {
		return nil
	}

	switch msg := msg.(type) {
	case *jsonrpc.Request:
		methodInHeader := header.Get(methodHeader)
		if methodInHeader == "" {
			return errors.New("missing required Mcp-Method header")
		}
		if methodInHeader != msg.Method {
			return fmt.Errorf("header mismatch: Mcp-Method header value '%s' does not match body value '%s'", methodInHeader, msg.Method)
		}

		if msg.Method == "tools/call" || msg.Method == "resources/read" || msg.Method == "prompts/get" {
			nameInHeader := header.Get(nameHeader)
			if nameInHeader == "" {
				return fmt.Errorf("missing required Mcp-Name header for method %q", msg.Method)
			}
			nameInBody, ok := extractName(msg.Method, msg.Params)
			if !ok {
				return fmt.Errorf("failed to extract name from parameters for method %q", msg.Method)
			}
			if nameInHeader != nameInBody {
				return fmt.Errorf("header mismatch: Mcp-Name header value '%s' does not match body value '%s'", nameInHeader, nameInBody)
			}
		}
	}
	return nil
}
