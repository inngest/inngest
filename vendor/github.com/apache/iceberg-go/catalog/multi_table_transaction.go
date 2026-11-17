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

package catalog

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/apache/iceberg-go/table"
)

// MultiTableTransaction collects changes across multiple tables and
// commits them atomically via [TransactionalCatalog.CommitTransaction].
//
// A MultiTableTransaction must not be used concurrently from multiple
// goroutines.
//
// Usage:
//
//	mtx, err := catalog.NewMultiTableTransaction(cat)
//	// ... err check ...
//
//	tx1 := tbl1.NewTransaction()
//	tx1.SetProperties(map[string]string{"key": "val"})
//	mtx.AddTransaction(tx1)
//
//	tx2 := tbl2.NewTransaction()
//	// ... build changes on tx2 ...
//	mtx.AddTransaction(tx2)
//
//	err = mtx.Commit(ctx)
type MultiTableTransaction struct {
	cat        TransactionalCatalog
	loader     Catalog
	txns       []*table.Transaction
	tableNames []string
	tableIDs   []table.Identifier
	committed  bool
}

// NewMultiTableTransaction creates a new multi-table transaction backed
// by the given catalog. Returns an error if the catalog does not
// implement [TransactionalCatalog].
func NewMultiTableTransaction(cat Catalog) (*MultiTableTransaction, error) {
	tc, ok := cat.(TransactionalCatalog)
	if !ok {
		return nil, errors.New("catalog does not support multi-table transactions")
	}

	return &MultiTableTransaction{cat: tc, loader: cat}, nil
}

// AddTransaction adds a table transaction to be committed atomically
// with all other transactions in this multi-table transaction.
// Returns an error if the transaction is nil, already committed, or
// targets a table that was already added.
//
// A transaction may still be empty here and gain updates before Commit,
// so emptiness is only decided when the payloads are built.
func (m *MultiTableTransaction) AddTransaction(tx *table.Transaction) error {
	if tx == nil {
		return errors.New("transaction must not be nil")
	}

	if m.committed {
		return errors.New("multi-table transaction has already been committed")
	}

	tc, err := tx.TableCommit()
	if err != nil {
		return err
	}

	for _, identifier := range m.tableIDs {
		if slices.Equal(identifier, tc.Identifier) {
			return fmt.Errorf("duplicate table in multi-table transaction: %s",
				strings.Join(tc.Identifier, "."))
		}
	}

	m.txns = append(m.txns, tx)
	m.tableNames = append(m.tableNames, strings.Join(tc.Identifier, "."))
	m.tableIDs = append(m.tableIDs, slices.Clone(tc.Identifier))

	return nil
}

// Commit extracts pending changes from all added transactions and
// commits them atomically. On success, all transactions are marked
// as committed. On failure, no transactions are marked committed.
//
// A retry must be rebuilt from freshly loaded tables:
// each payload asserts the branch head its transaction read.
//
// Transactions that stage no updates are left out of the request, since
// a catalog may reject a table change carrying none. They are still
// marked committed. A request with nothing left to send is not made.
//
// PostCommit hooks are not executed. Because the multi-table commit
// endpoint returns 204 No Content, callers must LoadTable individually
// to obtain updated metadata.
func (m *MultiTableTransaction) Commit(ctx context.Context) error {
	if m.committed {
		return errors.New("multi-table transaction has already been committed")
	}

	if len(m.txns) == 0 {
		return ErrEmptyCommitList
	}

	commits := make([]table.TableCommit, 0, len(m.txns))
	names := make([]string, 0, len(m.txns))
	for i, tx := range m.txns {
		tc, err := tx.TableCommit()
		if err != nil {
			return err
		}

		if len(tc.Updates) == 0 {
			continue
		}

		commits = append(commits, tc)
		names = append(names, m.tableNames[i])
	}

	if len(commits) > 0 {
		if err := m.cat.CommitTransaction(ctx, commits); err != nil {
			return fmt.Errorf("commit transaction for tables [%s]: %w",
				strings.Join(names, ", "), err)
		}
	}

	m.committed = true

	// Mark all transactions as committed to prevent reuse.
	for _, tx := range m.txns {
		tx.MarkCommitted()
	}

	return nil
}

// CommitAndReload commits the multi-table transaction atomically and
// then reloads all affected tables from the catalog. This is a
// convenience method that combines [MultiTableTransaction.Commit] with
// individual LoadTable calls, since the multi-table commit endpoint
// returns 204 No Content and does not include updated metadata.
//
// On commit failure, no tables are reloaded and the error is returned.
// On partial reload failure (commit succeeded but a LoadTable fails),
// the successfully loaded tables are still returned alongside the error.
func (m *MultiTableTransaction) CommitAndReload(ctx context.Context) ([]*table.Table, error) {
	if err := m.Commit(ctx); err != nil {
		return nil, err
	}

	tables := make([]*table.Table, 0, len(m.tableIDs))
	for _, ident := range m.tableIDs {
		tbl, err := m.loader.LoadTable(ctx, ident)
		if err != nil {
			return tables, fmt.Errorf("reload table %s after commit: %w",
				strings.Join(ident, "."), err)
		}

		tables = append(tables, tbl)
	}

	return tables, nil
}
