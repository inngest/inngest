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

package table

import (
	"errors"
	"slices"
)

// TableCommit holds the identifier, requirements, and updates for a single
// table within a multi-table transaction. It is used with
// [catalog.TransactionalCatalog.CommitTransaction] to atomically commit
// changes across multiple tables.
type TableCommit struct {
	Identifier   Identifier
	Requirements []Requirement
	Updates      []Update
}

// TableCommit returns a TableCommit representing the pending changes in this
// transaction without actually committing them. This is intended for
// multi-table transactions where several TableCommit values are collected
// and submitted together via [catalog.TransactionalCatalog.CommitTransaction].
//
// Most callers should use [catalog.MultiTableTransaction] instead of calling
// this method directly — it handles extraction, commit, and lifecycle
// management automatically.
//
// The returned requirements are the ones [Transaction.Commit] would send,
// so the multi-table path is fenced by the target branch's base head.
// A transaction with no updates yields an empty payload and enforces none.
//
// TableCommit does not mark the transaction as committed — the caller is
// responsible for either calling Commit (single-table) or submitting the
// returned TableCommit via CommitTransaction (multi-table). After a
// successful multi-table commit the caller should call MarkCommitted to
// prevent accidental reuse.
//
// PostCommit hooks are NOT executed by this method. Because the multi-table
// commit endpoint returns 204 No Content (no metadata), callers must
// LoadTable after a successful CommitTransaction if they need updated state.
func (t *Transaction) TableCommit() (TableCommit, error) {
	if err := t.checkNotNil(); err != nil {
		return TableCommit{}, err
	}
	t.mx.Lock()
	defer t.mx.Unlock()

	meta, err := t.txnMeta()
	if err != nil {
		return TableCommit{}, err
	}

	if t.committed {
		return TableCommit{}, errors.New("transaction has already been committed")
	}

	if len(meta.updates) == 0 {
		return TableCommit{
			Identifier:   slices.Clone(t.tbl.identifier),
			Requirements: []Requirement{},
			Updates:      []Update{},
		}, nil
	}

	// Same derivation as Commit. Copying t.reqs alone leaves metadata-only transactions unfenced:
	// their updates stage no ref requirement. The helper returns a fresh
	// slice, so appending here cannot touch t.reqs.
	reqs := append(transactionRequirements(t.reqs, t.branch, t.tbl.metadata), AssertTableUUID(meta.uuid))

	updates := make([]Update, len(meta.updates))
	copy(updates, meta.updates)

	return TableCommit{
		Identifier:   slices.Clone(t.tbl.identifier),
		Requirements: reqs,
		Updates:      updates,
	}, nil
}

// MarkCommitted marks the transaction as committed, preventing further use.
// This should be called after a successful multi-table commit via
// [catalog.TransactionalCatalog.CommitTransaction].
func (t *Transaction) MarkCommitted() {
	t.mx.Lock()
	defer t.mx.Unlock()

	t.committed = true
}
