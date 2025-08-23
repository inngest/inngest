// Copyright 2023-2024 Buf Technologies, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package evaluator

import (
	"github.com/bufbuild/protovalidate-go/internal/errors"
	"google.golang.org/protobuf/reflect/protoreflect"
)

// message performs validation on a protoreflect.Message.
type message struct {
	// Err stores if there was a compilation error constructing this evaluator.
	// It is cached here so that it can be stored in the registry's lookup table.
	Err error

	// evaluators are the individual evaluators that are applied to a message.
	evaluators messageEvaluators
}

func (m *message) Evaluate(val protoreflect.Value, failFast bool) error {
	return m.EvaluateMessage(val.Message(), failFast)
}

func (m *message) EvaluateMessage(msg protoreflect.Message, failFast bool) error {
	if m.Err != nil {
		return m.Err
	}
	return m.evaluators.EvaluateMessage(msg, failFast)
}

func (m *message) Tautology() bool {
	// returning false for now to avoid recursive messages causing false positives
	// on tautology detection.
	//
	// TODO: use a more sophisticated method to detect recursions so we can
	//  continue to detect tautologies on message evaluators.
	return false
}

func (m *message) Append(eval MessageEvaluator) {
	if eval != nil && !eval.Tautology() {
		m.evaluators = append(m.evaluators, eval)
	}
}

// unknownMessage is a MessageEvaluator for an unknown descriptor. This is
// returned only if lazy-building of evaluators has been disabled and an unknown
// descriptor is encountered.
type unknownMessage struct {
	desc protoreflect.MessageDescriptor
}

func (u unknownMessage) Err() error {
	return errors.NewCompilationErrorf(
		"no evaluator available for %s",
		u.desc.FullName())
}

func (u unknownMessage) Tautology() bool { return false }

func (u unknownMessage) Evaluate(_ protoreflect.Value, _ bool) error {
	return u.Err()
}

func (u unknownMessage) EvaluateMessage(_ protoreflect.Message, _ bool) error {
	return u.Err()
}

// embeddedMessage is a wrapper for fields containing messages. It contains data that
// may differ per embeddedMessage message so that it is not cached.
type embeddedMessage struct {
	base

	message *message
}

func (m *embeddedMessage) Evaluate(val protoreflect.Value, failFast bool) error {
	err := m.message.EvaluateMessage(val.Message(), failFast)
	errors.UpdatePaths(err, m.base.FieldPathElement, nil)
	return err
}

func (m *embeddedMessage) Tautology() bool {
	return m.message.Tautology()
}

var (
	_ MessageEvaluator = (*message)(nil)
	_ MessageEvaluator = (*unknownMessage)(nil)
	_ evaluator        = (*embeddedMessage)(nil)
)
