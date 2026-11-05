// Licensed to the Apache Software Foundation (ASF) under one
// or more contributor license agreements.  See the NOTICE file
// distributed with this work for additional information
// regarding copyright ownership.  The ASF licenses this file
// to you under the Apache License, Version 2.0 (the
// "License"); you may not use this file except in compliance
// with the License.  You may obtain a copy of the License at
//
//   http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package view

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"

	"github.com/apache/iceberg-go/internal"
	"github.com/apache/iceberg-go/table"
	"github.com/google/uuid"
)

const (
	reqAssertViewUUID = "assert-view-uuid"
)

// A Requirement is a validation rule that must be satisfied before attempting to
// make and commit changes to a table. Requirements are used to ensure that the
// view is in a valid state before making changes.
type Requirement interface {
	// Validate checks that the current view metadata satisfies the requirement.
	Validate(Metadata) error
	GetType() string
}

type Requirements []Requirement

func (r *Requirements) UnmarshalJSON(data []byte) error {
	var rawRequirements []json.RawMessage
	if err := json.Unmarshal(data, &rawRequirements); err != nil {
		return err
	}

	var requirements Requirements
	if len(rawRequirements) > 0 {
		requirements = make(Requirements, 0, len(rawRequirements))
	}
	for _, raw := range rawRequirements {
		req, err := parseRequirementBytes(raw, func(reqType string) error {
			return fmt.Errorf("unknown requirement type: %s", reqType)
		})
		if err != nil {
			return err
		}
		requirements = append(requirements, req)
	}

	*r = requirements

	return nil
}

// baseRequirement is a common struct that all requirements embed. It is used to
// identify the type of the requirement.
type baseRequirement struct {
	Type string `json:"type"`
}

type assertViewUUIDWire struct {
	UUID *uuid.UUID `json:"uuid"`
}

func requiredRequirementField(name string) error {
	return internal.MissingRequiredField(table.ErrInvalidRequirement, name)
}

func parseRequirementBytes(b []byte, unknown func(string) error) (Requirement, error) {
	var base internal.RequirementWire
	if err := json.Unmarshal(b, &base); err != nil {
		return nil, err
	}
	if base.Type == nil {
		return nil, requiredRequirementField("type")
	}

	switch *base.Type {
	case reqAssertViewUUID:
		var req assertViewUUIDWire
		if err := json.Unmarshal(b, &req); err != nil {
			return nil, err
		}
		if req.UUID == nil {
			return nil, requiredRequirementField("uuid")
		}

		return AssertViewUUID(*req.UUID), nil
	default:
		return nil, unknown(*base.Type)
	}
}

func (b baseRequirement) GetType() string {
	return b.Type
}

type assertViewUuid struct {
	baseRequirement
	UUID uuid.UUID `json:"uuid"`
}

// AssertViewUUID creates a requirement that the table UUID matches the given UUID.
func AssertViewUUID(uuid uuid.UUID) Requirement {
	return &assertViewUuid{
		baseRequirement: baseRequirement{Type: reqAssertViewUUID},
		UUID:            uuid,
	}
}

func (a *assertViewUuid) Validate(meta Metadata) error {
	if meta == nil {
		return errors.New("requirement failed: current view metadata does not exist")
	}

	if meta.ViewUUID() != a.UUID {
		return fmt.Errorf("UUID mismatch: %s != %s", meta.ViewUUID(), a.UUID)
	}

	return nil
}

// ParseRequirement parses json data provided by the reader into a Requirement
func ParseRequirement(r io.Reader) (Requirement, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}

	return ParseRequirementBytes(data)
}

// ParseRequirementString parses json string into a Requirement
func ParseRequirementString(s string) (Requirement, error) {
	return ParseRequirementBytes([]byte(s))
}

// ParseRequirementBytes parses json bytes into a Requirement
func ParseRequirementBytes(b []byte) (Requirement, error) {
	return parseRequirementBytes(b, func(string) error { return table.ErrInvalidRequirement })
}
