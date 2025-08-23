// Copyright 2023-2025 Buf Technologies, Inc.
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
	"slices"

	"buf.build/gen/go/bufbuild/protovalidate/protocolbuffers/go/buf/validate"
	pvcel "buf.build/go/protovalidate/cel"
	"github.com/google/cel-go/cel"
	"github.com/google/cel-go/interpreter"
	"google.golang.org/protobuf/reflect/protoreflect"
)

// astSet represents a collection of compiledAST and their associated cel.Env.
type astSet []compiledAST

// Merge combines a set with another, producing a new ASTSet.
func (set astSet) Merge(other astSet) astSet {
	out := make([]compiledAST, 0, len(set)+len(other))
	out = append(out, set...)
	out = append(out, other...)
	return out
}

// ReduceResiduals generates a ProgramSet, performing a partial evaluation of
// the ASTSet to optimize the expression. If the expression is optimized to
// either a true or empty string constant result, no compiledProgram is
// generated for it. The main usage of this is to elide tautological expressions
// from the final result.
func (set astSet) ReduceResiduals(opts ...cel.ProgramOption) (programSet, error) {
	residuals := make(astSet, 0, len(set))
	options := append([]cel.ProgramOption{
		cel.EvalOptions(
			cel.OptTrackState,
			cel.OptExhaustiveEval,
			cel.OptOptimize,
			cel.OptPartialEval,
		),
	}, opts...)

	for _, ast := range set {
		options := slices.Clone(options)
		if ast.Value.IsValid() {
			options = append(options, cel.Globals(&variable{Name: "rule", Val: ast.Value.Interface()}))
		}
		program, err := ast.toProgram(ast.Env, options...)
		if err != nil {
			residuals = append(residuals, ast)
			continue
		}
		val, details, _ := program.Program.Eval(interpreter.EmptyActivation())
		if val != nil {
			switch value := val.Value().(type) {
			case bool:
				if value {
					continue
				}
			case string:
				if value == "" {
					continue
				}
			}
		}
		residual, err := ast.Env.ResidualAst(ast.AST, details)
		if err != nil {
			residuals = append(residuals, ast)
		} else {
			residuals = append(residuals, compiledAST{
				AST:        residual,
				Env:        ast.Env,
				Source:     ast.Source,
				Path:       ast.Path,
				Value:      ast.Value,
				Descriptor: ast.Descriptor,
			})
		}
	}

	return residuals.ToProgramSet(opts...)
}

// ToProgramSet generates a ProgramSet from the specified ASTs.
func (set astSet) ToProgramSet(opts ...cel.ProgramOption) (out programSet, err error) {
	if l := len(set); l == 0 {
		return nil, nil
	}
	out = make(programSet, len(set))
	for i, ast := range set {
		out[i], err = ast.toProgram(ast.Env, opts...)
		if err != nil {
			return nil, err
		}
	}
	return out, nil
}

// SetRuleValue sets the rule value for the programs in the ASTSet.
func (set astSet) WithRuleValue(
	ruleValue protoreflect.Value,
	ruleDescriptor protoreflect.FieldDescriptor,
) (out astSet, err error) {
	out = slices.Clone(set)
	for i := range set {
		out[i].Env, err = out[i].Env.Extend(
			cel.Constant(
				"rule",
				pvcel.ProtoFieldToType(ruleDescriptor, true, false),
				pvcel.ProtoFieldToValue(ruleDescriptor, ruleValue, false),
			),
		)
		if err != nil {
			return nil, err
		}
		out[i].Value = ruleValue
		out[i].Descriptor = ruleDescriptor
	}
	return out, nil
}

type compiledAST struct {
	AST        *cel.Ast
	Env        *cel.Env
	Source     *validate.Rule
	Path       []*validate.FieldPathElement
	Value      protoreflect.Value
	Descriptor protoreflect.FieldDescriptor
}

func (ast compiledAST) toProgram(env *cel.Env, opts ...cel.ProgramOption) (out compiledProgram, err error) {
	prog, err := env.Program(ast.AST, opts...)
	if err != nil {
		return out, &CompilationError{cause: fmt.Errorf("failed to compile program %s: %w", ast.Source.GetId(), err)}
	}
	return compiledProgram{
		Program:    prog,
		Source:     ast.Source,
		Path:       ast.Path,
		Value:      ast.Value,
		Descriptor: ast.Descriptor,
	}, nil
}
