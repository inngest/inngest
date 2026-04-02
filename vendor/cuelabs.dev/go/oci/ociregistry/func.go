// Copyright 2023 CUE Labs AG
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package ociregistry

import (
	"context"
	"fmt"
	"io"
	"iter"
)

var _ Interface = (*Funcs)(nil)

// Funcs implements Interface by calling its member functions: there's one field
// for every corresponding method of [Interface].
//
// When a function is nil, the corresponding method will return
// an [ErrUnsupported] error. For nil functions that return an iterator,
// the corresponding method will return an iterator that returns no items and
// returns ErrUnsupported from its Err method.
//
// If Funcs is nil itself, all methods will behave as if the corresponding field was nil,
// so (*ociregistry.Funcs)(nil) is a useful placeholder to implement Interface.
//
// If you're writing your own implementation of Funcs, you'll need to embed a *Funcs
// value to get an implementation of the private method. This means that it will
// be possible to add members to Interface in the future without breaking compatibility.
type Funcs struct {
	NewError func(ctx context.Context, methodName, repo string) error

	GetBlob_               func(ctx context.Context, repo string, digest Digest) (BlobReader, error)
	GetBlobRange_          func(ctx context.Context, repo string, digest Digest, offset0, offset1 int64) (BlobReader, error)
	GetManifest_           func(ctx context.Context, repo string, digest Digest) (BlobReader, error)
	GetTag_                func(ctx context.Context, repo string, tagName string) (BlobReader, error)
	ResolveBlob_           func(ctx context.Context, repo string, digest Digest) (Descriptor, error)
	ResolveManifest_       func(ctx context.Context, repo string, digest Digest) (Descriptor, error)
	ResolveTag_            func(ctx context.Context, repo string, tagName string) (Descriptor, error)
	PushBlob_              func(ctx context.Context, repo string, desc Descriptor, r io.Reader) (Descriptor, error)
	PushBlobChunked_       func(ctx context.Context, repo string, chunkSize int) (BlobWriter, error)
	PushBlobChunkedResume_ func(ctx context.Context, repo, id string, offset int64, chunkSize int) (BlobWriter, error)
	MountBlob_             func(ctx context.Context, fromRepo, toRepo string, digest Digest) (Descriptor, error)
	PushManifest_          func(ctx context.Context, repo string, tag string, contents []byte, mediaType string) (Descriptor, error)
	DeleteBlob_            func(ctx context.Context, repo string, digest Digest) error
	DeleteManifest_        func(ctx context.Context, repo string, digest Digest) error
	DeleteTag_             func(ctx context.Context, repo string, name string) error
	Repositories_          func(ctx context.Context, startAfter string) iter.Seq2[string, error]
	Tags_                  func(ctx context.Context, repo string, startAfter string) iter.Seq2[string, error]
	Referrers_             func(ctx context.Context, repo string, digest Digest, artifactType string) iter.Seq2[Descriptor, error]
}

// This blesses Funcs as the canonical Interface implementation.
func (*Funcs) private() {}

func (f *Funcs) newError(ctx context.Context, methodName, repo string) error {
	if f != nil && f.NewError != nil {
		return f.NewError(ctx, methodName, repo)
	}
	return fmt.Errorf("%s: %w", methodName, ErrUnsupported)
}

func (f *Funcs) GetBlob(ctx context.Context, repo string, digest Digest) (BlobReader, error) {
	if f != nil && f.GetBlob_ != nil {
		return f.GetBlob_(ctx, repo, digest)
	}
	return nil, f.newError(ctx, "GetBlob", repo)
}

func (f *Funcs) GetBlobRange(ctx context.Context, repo string, digest Digest, offset0, offset1 int64) (BlobReader, error) {
	if f != nil && f.GetBlobRange_ != nil {
		return f.GetBlobRange_(ctx, repo, digest, offset0, offset1)
	}
	return nil, f.newError(ctx, "GetBlobRange", repo)
}

func (f *Funcs) GetManifest(ctx context.Context, repo string, digest Digest) (BlobReader, error) {
	if f != nil && f.GetManifest_ != nil {
		return f.GetManifest_(ctx, repo, digest)
	}
	return nil, f.newError(ctx, "GetManifest", repo)
}

func (f *Funcs) GetTag(ctx context.Context, repo string, tagName string) (BlobReader, error) {
	if f != nil && f.GetTag_ != nil {
		return f.GetTag_(ctx, repo, tagName)
	}
	return nil, f.newError(ctx, "GetTag", repo)
}

func (f *Funcs) ResolveBlob(ctx context.Context, repo string, digest Digest) (Descriptor, error) {
	if f != nil && f.ResolveBlob_ != nil {
		return f.ResolveBlob_(ctx, repo, digest)
	}
	return Descriptor{}, f.newError(ctx, "ResolveBlob", repo)
}

func (f *Funcs) ResolveManifest(ctx context.Context, repo string, digest Digest) (Descriptor, error) {
	if f != nil && f.ResolveManifest_ != nil {
		return f.ResolveManifest_(ctx, repo, digest)
	}
	return Descriptor{}, f.newError(ctx, "ResolveManifest", repo)
}

func (f *Funcs) ResolveTag(ctx context.Context, repo string, tagName string) (Descriptor, error) {
	if f != nil && f.ResolveTag_ != nil {
		return f.ResolveTag_(ctx, repo, tagName)
	}
	return Descriptor{}, f.newError(ctx, "ResolveTag", repo)
}

func (f *Funcs) PushBlob(ctx context.Context, repo string, desc Descriptor, r io.Reader) (Descriptor, error) {
	if f != nil && f.PushBlob_ != nil {
		return f.PushBlob_(ctx, repo, desc, r)
	}
	return Descriptor{}, f.newError(ctx, "PushBlob", repo)
}

func (f *Funcs) PushBlobChunked(ctx context.Context, repo string, chunkSize int) (BlobWriter, error) {
	if f != nil && f.PushBlobChunked_ != nil {
		return f.PushBlobChunked_(ctx, repo, chunkSize)
	}
	return nil, f.newError(ctx, "PushBlobChunked", repo)
}

func (f *Funcs) PushBlobChunkedResume(ctx context.Context, repo, id string, offset int64, chunkSize int) (BlobWriter, error) {
	if f != nil && f.PushBlobChunked_ != nil {
		return f.PushBlobChunkedResume_(ctx, repo, id, offset, chunkSize)
	}
	return nil, f.newError(ctx, "PushBlobChunked", repo)
}

func (f *Funcs) MountBlob(ctx context.Context, fromRepo, toRepo string, digest Digest) (Descriptor, error) {
	if f != nil && f.MountBlob_ != nil {
		return f.MountBlob_(ctx, fromRepo, toRepo, digest)
	}
	return Descriptor{}, f.newError(ctx, "MountBlob", toRepo)
}

func (f *Funcs) PushManifest(ctx context.Context, repo string, tag string, contents []byte, mediaType string) (Descriptor, error) {
	if f != nil && f.PushManifest_ != nil {
		return f.PushManifest_(ctx, repo, tag, contents, mediaType)
	}
	return Descriptor{}, f.newError(ctx, "PushManifest", repo)
}

func (f *Funcs) DeleteBlob(ctx context.Context, repo string, digest Digest) error {
	if f != nil && f.DeleteBlob_ != nil {
		return f.DeleteBlob_(ctx, repo, digest)
	}
	return f.newError(ctx, "DeleteBlob", repo)
}

func (f *Funcs) DeleteManifest(ctx context.Context, repo string, digest Digest) error {
	if f != nil && f.DeleteManifest_ != nil {
		return f.DeleteManifest_(ctx, repo, digest)
	}
	return f.newError(ctx, "DeleteManifest", repo)
}

func (f *Funcs) DeleteTag(ctx context.Context, repo string, name string) error {
	if f != nil && f.DeleteTag_ != nil {
		return f.DeleteTag_(ctx, repo, name)
	}
	return f.newError(ctx, "DeleteTag", repo)
}

func (f *Funcs) Repositories(ctx context.Context, startAfter string) iter.Seq2[string, error] {
	if f != nil && f.Repositories_ != nil {
		return f.Repositories_(ctx, startAfter)
	}
	return ErrorSeq[string](f.newError(ctx, "Repositories", ""))
}

func (f *Funcs) Tags(ctx context.Context, repo string, startAfter string) iter.Seq2[string, error] {
	if f != nil && f.Tags_ != nil {
		return f.Tags_(ctx, repo, startAfter)
	}
	return ErrorSeq[string](f.newError(ctx, "Tags", repo))
}

func (f *Funcs) Referrers(ctx context.Context, repo string, digest Digest, artifactType string) iter.Seq2[Descriptor, error] {
	if f != nil && f.Referrers_ != nil {
		return f.Referrers_(ctx, repo, digest, artifactType)
	}
	return ErrorSeq[Descriptor](f.newError(ctx, "Referrers", repo))
}
