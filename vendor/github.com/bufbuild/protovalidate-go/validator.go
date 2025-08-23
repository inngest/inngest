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

package protovalidate

import (
	"fmt"
	"sync"

	"buf.build/gen/go/bufbuild/protovalidate/protocolbuffers/go/buf/validate"
	"github.com/bufbuild/protovalidate-go/celext"
	"github.com/bufbuild/protovalidate-go/internal/errors"
	"github.com/bufbuild/protovalidate-go/internal/evaluator"
	"github.com/bufbuild/protovalidate-go/resolver"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
)

var getGlobalValidator = sync.OnceValues(func() (*Validator, error) { return New() })

type (
	// ValidationError is returned if one or more constraints on a message are
	// violated. This error type is a composite of multiple Violation instances.
	//
	//    err = validator.Validate(msg)
	//    var valErr *ValidationError
	//    if ok := errors.As(err, &valErr); ok {
	//      violations := valErrs.Violations
	//      // ...
	//    }
	ValidationError = errors.ValidationError

	// A Violation provides information about one constraint violation.
	Violation = errors.Violation

	// A CompilationError is returned if a CEL expression cannot be compiled &
	// type-checked or if invalid standard constraints are applied to a field.
	CompilationError = errors.CompilationError

	// A RuntimeError is returned if a valid CEL expression evaluation is
	// terminated, typically due to an unknown or mismatched type.
	RuntimeError = errors.RuntimeError
)

// Validator performs validation on any proto.Message values. The Validator is
// safe for concurrent use.
type Validator struct {
	builder  *evaluator.Builder
	failFast bool
}

// New creates a Validator with the given options. An error may occur in setting
// up the CEL execution environment if the configuration is invalid. See the
// individual ValidatorOption for how they impact the fallibility of New.
func New(options ...ValidatorOption) (*Validator, error) {
	cfg := config{
		resolver:              resolver.DefaultResolver{},
		extensionTypeResolver: protoregistry.GlobalTypes,
	}
	for _, opt := range options {
		opt(&cfg)
	}

	env, err := celext.DefaultEnv(cfg.useUTC)
	if err != nil {
		return nil, fmt.Errorf(
			"failed to construct CEL environment: %w", err)
	}

	bldr := evaluator.NewBuilder(
		env,
		cfg.disableLazy,
		cfg.resolver,
		cfg.extensionTypeResolver,
		cfg.allowUnknownFields,
		cfg.desc...,
	)

	return &Validator{
		failFast: cfg.failFast,
		builder:  bldr,
	}, nil
}

// Validate checks that message satisfies its constraints. Constraints are
// defined within the Protobuf file as options from the buf.validate package.
// An error is returned if the constraints are violated (ValidationError), the
// evaluation logic for the message cannot be built (CompilationError), or
// there is a type error when attempting to evaluate a CEL expression
// associated with the message (RuntimeError).
func (v *Validator) Validate(msg proto.Message) error {
	if msg == nil {
		return nil
	}
	refl := msg.ProtoReflect()
	eval := v.builder.Load(refl.Descriptor())
	err := eval.EvaluateMessage(refl, v.failFast)
	errors.FinalizePaths(err)
	return err
}

// Validate uses a global instance of Validator constructed with no ValidatorOptions and
// calls its Validate function. For the vast majority of validation cases, using this global
// function is safe and acceptable. If you need to provide i.e. a custom
// ExtensionTypeResolver, you'll need to construct a Validator.
func Validate(msg proto.Message) error {
	globalValidator, err := getGlobalValidator()
	if err != nil {
		return err
	}
	return globalValidator.Validate(msg)
}

// FieldPathString returns a dotted path string for the provided
// validate.FieldPath.
func FieldPathString(path *validate.FieldPath) string {
	return errors.FieldPathString(path.GetElements())
}

type config struct {
	failFast              bool
	useUTC                bool
	disableLazy           bool
	desc                  []protoreflect.MessageDescriptor
	resolver              StandardConstraintResolver
	extensionTypeResolver protoregistry.ExtensionTypeResolver
	allowUnknownFields    bool
}

// A ValidatorOption modifies the default configuration of a Validator. See the
// individual options for their defaults and affects on the fallibility of
// configuring a Validator.
type ValidatorOption func(*config)

// WithUTC specifies whether timestamp operations should use UTC or the OS's
// local timezone for timestamp related values. By default, the local timezone
// is used.
func WithUTC(useUTC bool) ValidatorOption {
	return func(c *config) {
		c.useUTC = useUTC
	}
}

// WithFailFast specifies whether validation should fail on the first constraint
// violation encountered or if all violations should be accumulated. By default,
// all violations are accumulated.
func WithFailFast(failFast bool) ValidatorOption {
	return func(cfg *config) {
		cfg.failFast = failFast
	}
}

// WithMessages allows warming up the Validator with messages that are
// expected to be validated. Messages included transitively (i.e., fields with
// message values) are automatically handled.
func WithMessages(messages ...proto.Message) ValidatorOption {
	desc := make([]protoreflect.MessageDescriptor, len(messages))
	for i, msg := range messages {
		desc[i] = msg.ProtoReflect().Descriptor()
	}
	return WithDescriptors(desc...)
}

// WithDescriptors allows warming up the Validator with message
// descriptors that are expected to be validated. Messages included transitively
// (i.e., fields with message values) are automatically handled.
func WithDescriptors(descriptors ...protoreflect.MessageDescriptor) ValidatorOption {
	return func(cfg *config) {
		cfg.desc = append(cfg.desc, descriptors...)
	}
}

// WithDisableLazy prevents the Validator from lazily building validation logic
// for a message it has not encountered before. Disabling lazy logic
// additionally eliminates any internal locking as the validator becomes
// read-only.
//
// Note: All expected messages must be provided by WithMessages or
// WithDescriptors during initialization.
func WithDisableLazy(disable bool) ValidatorOption {
	return func(cfg *config) {
		cfg.disableLazy = disable
	}
}

// StandardConstraintResolver is responsible for resolving the standard
// constraints from the provided protoreflect.Descriptor. The default resolver
// can be intercepted and modified using WithStandardConstraintInterceptor.
type StandardConstraintResolver interface {
	ResolveMessageConstraints(desc protoreflect.MessageDescriptor) *validate.MessageConstraints
	ResolveOneofConstraints(desc protoreflect.OneofDescriptor) *validate.OneofConstraints
	ResolveFieldConstraints(desc protoreflect.FieldDescriptor) *validate.FieldConstraints
}

// StandardConstraintInterceptor can be provided to
// WithStandardConstraintInterceptor to allow modifying a
// StandardConstraintResolver.
type StandardConstraintInterceptor func(res StandardConstraintResolver) StandardConstraintResolver

// WithStandardConstraintInterceptor allows intercepting the
// StandardConstraintResolver used by the Validator to modify or replace it.
func WithStandardConstraintInterceptor(interceptor StandardConstraintInterceptor) ValidatorOption {
	return func(c *config) {
		c.resolver = interceptor(c.resolver)
	}
}

// WithExtensionTypeResolver specifies a resolver to use when reparsing unknown
// extension types. When dealing with dynamic file descriptor sets, passing this
// option will allow extensions to be resolved using a custom resolver.
//
// To ignore unknown extension fields, use the [WithAllowUnknownFields] option.
// Note that this may result in messages being treated as valid even though not
// all constraints are being applied.
func WithExtensionTypeResolver(extensionTypeResolver protoregistry.ExtensionTypeResolver) ValidatorOption {
	return func(c *config) {
		c.extensionTypeResolver = extensionTypeResolver
	}
}

// WithAllowUnknownFields specifies if the presence of unknown field constraints
// should cause compilation to fail with an error. When set to false, an unknown
// field will simply be ignored, which will cause constraints to silently not be
// applied. This condition may occur if a predefined constraint definition isn't
// present in the extension type resolver, or when passing dynamic messages with
// standard constraints defined in a newer version of protovalidate. The default
// value is false, to prevent silently-incorrect validation from occurring.
func WithAllowUnknownFields(allowUnknownFields bool) ValidatorOption {
	return func(c *config) {
		c.allowUnknownFields = allowUnknownFields
	}
}
