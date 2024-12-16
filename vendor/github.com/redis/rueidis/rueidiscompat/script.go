// Copyright (c) 2013 The github.com/go-redis/redis Authors.
// All rights reserved.
//
// Redistribution and use in source and binary forms, with or without
// modification, are permitted provided that the following conditions are
// met:
//
// * Redistributions of source code must retain the above copyright
// notice, this list of conditions and the following disclaimer.
// * Redistributions in binary form must reproduce the above
// copyright notice, this list of conditions and the following disclaimer
// in the documentation and/or other materials provided with the
// distribution.
//
// THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS
// "AS IS" AND ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT
// LIMITED TO, THE IMPLIED WARRANTIES OF MERCHANTABILITY AND FITNESS FOR
// A PARTICULAR PURPOSE ARE DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT
// OWNER OR CONTRIBUTORS BE LIABLE FOR ANY DIRECT, INDIRECT, INCIDENTAL,
// SPECIAL, EXEMPLARY, OR CONSEQUENTIAL DAMAGES (INCLUDING, BUT NOT
// LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR SERVICES; LOSS OF USE,
// DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER CAUSED AND ON ANY
// THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY, OR TORT
// (INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE
// OF THIS SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.

package rueidiscompat

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"io"
	"strings"
)

type Scripter interface {
	Eval(ctx context.Context, script string, keys []string, args ...interface{}) *Cmd
	EvalSha(ctx context.Context, sha1 string, keys []string, args ...interface{}) *Cmd
	EvalRO(ctx context.Context, script string, keys []string, args ...interface{}) *Cmd
	EvalShaRO(ctx context.Context, sha1 string, keys []string, args ...interface{}) *Cmd
	ScriptExists(ctx context.Context, hashes ...string) *BoolSliceCmd
	ScriptLoad(ctx context.Context, script string) *StringCmd
}

var (
	_ Scripter = (*Compat)(nil)
)

type Script struct {
	src, hash string
}

func NewScript(src string) *Script {
	h := sha1.New()
	_, _ = io.WriteString(h, src)
	return &Script{
		src:  src,
		hash: hex.EncodeToString(h.Sum(nil)),
	}
}

func (s *Script) Hash() string {
	return s.hash
}

func (s *Script) Load(ctx context.Context, c Scripter) *StringCmd {
	return c.ScriptLoad(ctx, s.src)
}

func (s *Script) Exists(ctx context.Context, c Scripter) *BoolSliceCmd {
	return c.ScriptExists(ctx, s.hash)
}

func (s *Script) Eval(ctx context.Context, c Scripter, keys []string, args ...interface{}) *Cmd {
	return c.Eval(ctx, s.src, keys, args...)
}

func (s *Script) EvalRO(ctx context.Context, c Scripter, keys []string, args ...interface{}) *Cmd {
	return c.EvalRO(ctx, s.src, keys, args...)
}

func (s *Script) EvalSha(ctx context.Context, c Scripter, keys []string, args ...interface{}) *Cmd {
	return c.EvalSha(ctx, s.hash, keys, args...)
}

func (s *Script) EvalShaRO(ctx context.Context, c Scripter, keys []string, args ...interface{}) *Cmd {
	return c.EvalShaRO(ctx, s.hash, keys, args...)
}

// Run optimistically uses EVALSHA to run the script. If script does not exist
// it is retried using EVAL.
func (s *Script) Run(ctx context.Context, c Scripter, keys []string, args ...interface{}) *Cmd {
	r := s.EvalSha(ctx, c, keys, args...)
	if err := r.Err(); err != nil {
		msg := err.Error()
		msg = strings.TrimPrefix(msg, "ERR ")
		if strings.HasPrefix(msg, "NOSCRIPT") {
			return s.Eval(ctx, c, keys, args...)
		}
	}
	return r
}

// RunRO optimistically uses EVALSHA_RO to run the script. If script does not exist
// it is retried using EVAL_RO.
func (s *Script) RunRO(ctx context.Context, c Scripter, keys []string, args ...interface{}) *Cmd {
	r := s.EvalShaRO(ctx, c, keys, args...)
	if err := r.Err(); err != nil {
		msg := err.Error()
		msg = strings.TrimPrefix(msg, "ERR ")
		if strings.HasPrefix(msg, "NOSCRIPT") {
			return s.EvalRO(ctx, c, keys, args...)
		}
	}
	return r
}
