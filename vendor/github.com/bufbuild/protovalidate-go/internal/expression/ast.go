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

package expression

import (
	"buf.build/gen/go/bufbuild/protovalidate/protocolbuffers/go/buf/validate"
	"github.com/bufbuild/protovalidate-go/internal/errors"
	"github.com/google/cel-go/cel"
	"github.com/google/cel-go/interpreter"
	"google.golang.org/protobuf/reflect/protoreflect"
)

// ASTSet represents a collection of compiledAST and their associated cel.Env.
type ASTSet struct {
	env  *cel.Env
	asts []compiledAST
}

// Merge combines a set with another, producing a new ASTSet.
func (set ASTSet) Merge(other ASTSet) ASTSet {
	out := ASTSet{
		env:  set.env,
		asts: make([]compiledAST, 0, len(set.asts)+len(other.asts)),
	}
	if out.env == nil {
		out.env = other.env
	}
	out.asts = append(out.asts, set.asts...)
	out.asts = append(out.asts, other.asts...)
	return out
}

// ReduceResiduals generates a ProgramSet, performing a partial evaluation of
// the ASTSet to optimize the expression. If the expression is optimized to
// either a true or empty string constant result, no compiledProgram is
// generated for it. The main usage of this is to elide tautological expressions
// from the final result.
func (set ASTSet) ReduceResiduals(opts ...cel.ProgramOption) (ProgramSet, error) {
	residuals := make([]compiledAST, 0, len(set.asts))
	options := append([]cel.ProgramOption{
		cel.EvalOptions(
			cel.OptTrackState,
			cel.OptExhaustiveEval,
			cel.OptOptimize,
			cel.OptPartialEval,
		),
	}, opts...)

	for _, ast := range set.asts {
		program, err := ast.toProgram(set.env, options...)
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
		residual, err := set.env.ResidualAst(ast.AST, details)
		if err != nil {
			residuals = append(residuals, ast)
		} else {
			residuals = append(residuals, compiledAST{
				AST:        residual,
				Source:     ast.Source,
				Path:       ast.Path,
				Value:      ast.Value,
				Descriptor: ast.Descriptor,
			})
		}
	}

	return ASTSet{
		env:  set.env,
		asts: residuals,
	}.ToProgramSet(opts...)
}

// ToProgramSet generates a ProgramSet from the specified ASTs.
func (set ASTSet) ToProgramSet(opts ...cel.ProgramOption) (out ProgramSet, err error) {
	if l := len(set.asts); l == 0 {
		return nil, nil
	}
	out = make(ProgramSet, len(set.asts))
	for i, ast := range set.asts {
		out[i], err = ast.toProgram(set.env, opts...)
		if err != nil {
			return nil, err
		}
	}
	return out, nil
}

// SetRuleValue sets the rule value for the programs in the ASTSet.
func (set *ASTSet) SetRuleValue(
	ruleValue protoreflect.Value,
	ruleDescriptor protoreflect.FieldDescriptor,
) {
	set.asts = append([]compiledAST{}, set.asts...)
	for i := range set.asts {
		set.asts[i].Value = ruleValue
		set.asts[i].Descriptor = ruleDescriptor
	}
}

type compiledAST struct {
	AST        *cel.Ast
	Source     *validate.Constraint
	Path       []*validate.FieldPathElement
	Value      protoreflect.Value
	Descriptor protoreflect.FieldDescriptor
}

func (ast compiledAST) toProgram(env *cel.Env, opts ...cel.ProgramOption) (out compiledProgram, err error) {
	prog, err := env.Program(ast.AST, opts...)
	if err != nil {
		return out, errors.NewCompilationErrorf(
			"failed to compile program %s: %w", ast.Source.GetId(), err)
	}
	return compiledProgram{
		Program:    prog,
		Source:     ast.Source,
		Path:       ast.Path,
		Value:      ast.Value,
		Descriptor: ast.Descriptor,
	}, nil
}
