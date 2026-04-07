// Package iface provides the interface registry — a mapping of inngest/
// interfaces to their monorepo/ implementations. This is used by the
// classify and ifacecheck tools to determine whether an interface change
// is breaking (monorepo implements it) or additive (monorepo only calls it).
package iface

import (
	"fmt"
	"os"

	"inngest.com/upstream-compat/internal/astdiff"
)

// Criticality indicates how severe a break would be.
type Criticality string

const (
	Critical Criticality = "critical"
	High     Criticality = "high"
	Medium   Criticality = "medium"
	Low      Criticality = "low"
)

// WatchedInterface describes an inngest/ interface that monorepo/ depends on.
type WatchedInterface struct {
	// Package is the full import path, e.g. "github.com/inngest/inngest/pkg/execution/queue"
	Package string
	// Name is the interface name, e.g. "Queue"
	Name string
	// Criticality of this interface
	Criticality Criticality
	// Implemented is true if monorepo/ has concrete types satisfying this interface.
	// If false, monorepo/ only uses the interface (calls methods on it).
	Implemented bool
	// Implementors lists monorepo/ files containing implementations.
	Implementors []string
}

// DefaultRegistry returns the known interface mappings between inngest/ and monorepo/.
func DefaultRegistry() []WatchedInterface {
	return []WatchedInterface{
		{
			Package:     "github.com/inngest/inngest/pkg/execution",
			Name:        "Executor",
			Criticality: Critical,
			Implemented: true,
			Implementors: []string{
				"pkg/execution/impls/internalredis/executor.go",
				"pkg/execution/impls/foundationdb/executor.go",
			},
		},
		{
			Package:     "github.com/inngest/inngest/pkg/execution/queue",
			Name:        "Queue",
			Criticality: Critical,
			Implemented: true,
			Implementors: []string{
				"pkg/execution/impls/internalredis/queue.go",
			},
		},
		{
			Package:     "github.com/inngest/inngest/pkg/execution/queue",
			Name:        "QueueShard",
			Criticality: Critical,
			Implemented: true,
			Implementors: []string{
				"pkg/fdbqueue/fdb.go",
				"pkg/db/shards/queue.go",
			},
		},
		{
			Package:     "github.com/inngest/inngest/pkg/execution/queue",
			Name:        "ShardOperations",
			Criticality: Critical,
			Implemented: true,
			Implementors: []string{
				"pkg/fdbqueue/fdb.go",
			},
		},
		{
			Package:     "github.com/inngest/inngest/pkg/execution/queue",
			Name:        "QueueManager",
			Criticality: Critical,
			Implemented: true,
			Implementors: []string{
				"pkg/execution/impls/internalredis/queue.go",
			},
		},
		{
			Package:     "github.com/inngest/inngest/pkg/execution/state/v2",
			Name:        "RunService",
			Criticality: Critical,
			Implemented: true,
			Implementors: []string{
				"pkg/state_proxy/client.go",
				"pkg/db/redisdb/manager.go",
				"pkg/db/memorydb/manager.go",
			},
		},
		{
			Package:     "github.com/inngest/inngest/pkg/service",
			Name:        "Service",
			Criticality: Medium,
			Implemented: true,
			Implementors: []string{
				"pkg/applogic/cloudcdc/cdc_service.go",
				"cmd/all-in-one/",
			},
		},
		{
			Package:     "github.com/inngest/inngest/pkg/execution/batch",
			Name:        "BatchManager",
			Criticality: Medium,
			Implemented: true,
			Implementors: []string{
				"pkg/execution/impls/internalredis/batch.go",
			},
		},
		{
			Package:     "github.com/inngest/inngest/pkg/execution/debounce",
			Name:        "Debouncer",
			Criticality: Medium,
			Implemented: true,
			Implementors: []string{
				"pkg/execution/impls/internalredis/",
			},
		},
		{
			Package:     "github.com/inngest/inngest/pkg/execution/pauses",
			Name:        "Manager",
			Criticality: Medium,
			Implemented: true,
			Implementors: []string{
				"pkg/execution/impls/internalredis/pause.go",
			},
		},
		{
			Package:     "github.com/inngest/inngest/pkg/execution/singleton",
			Name:        "SingletonStore",
			Criticality: Medium,
			Implemented: true,
			Implementors: []string{
				"pkg/execution/impls/internalredis/",
			},
		},
		{
			Package:     "github.com/inngest/inngest/pkg/execution/cron",
			Name:        "CronManager",
			Criticality: Medium,
			Implemented: true,
			Implementors: []string{
				"pkg/execution/crons/queue_cron.go",
			},
		},
	}
}

// Classification of a set of changes.
type Classification string

const (
	Safe     Classification = "SAFE"
	Additive Classification = "ADDITIVE"
	Breaking Classification = "BREAKING"
)

// ClassifiedChange wraps a Change with interface-aware classification.
type ClassifiedChange struct {
	astdiff.Change
	Classification Classification
	Reason         string
	// AffectedFiles in monorepo/ (only for breaking changes to implemented interfaces)
	AffectedFiles []string
}

// ClassifyChanges takes raw symbol changes and the interface registry,
// and returns classified changes with monorepo/ impact information.
func ClassifyChanges(changes []astdiff.Change, registry []WatchedInterface) []ClassifiedChange {
	// Build lookup: "pkgbase.InterfaceName" -> WatchedInterface
	ifaceMap := make(map[string]*WatchedInterface)
	for i := range registry {
		w := &registry[i]
		// Extract last path component of package
		pkgBase := lastPathComponent(w.Package)
		key := fmt.Sprintf("%s.%s", pkgBase, w.Name)
		ifaceMap[key] = w
	}

	var classified []ClassifiedChange

	for _, c := range changes {
		cc := ClassifiedChange{Change: c}

		switch c.Type {
		case astdiff.Removed:
			cc.Classification = Breaking
			cc.Reason = fmt.Sprintf("removed %s %s", c.Symbol.Kind, c.Symbol.Name)
			// Check if this is part of a watched interface
			if w := findWatchedInterface(c.Symbol, ifaceMap); w != nil {
				cc.AffectedFiles = w.Implementors
				cc.Reason += fmt.Sprintf(" [%s, implemented by monorepo]", w.Criticality)
			}

		case astdiff.Modified:
			cc.Classification = Breaking
			cc.Reason = fmt.Sprintf("changed signature of %s %s", c.Symbol.Kind, c.Symbol.Name)
			if w := findWatchedInterface(c.Symbol, ifaceMap); w != nil {
				cc.AffectedFiles = w.Implementors
				cc.Reason += fmt.Sprintf(" [%s, implemented by monorepo]", w.Criticality)
			}

		case astdiff.Added:
			if c.Symbol.Kind == astdiff.KindMethod {
				// Check if this method belongs to a watched implemented interface
				if w := findWatchedInterface(c.Symbol, ifaceMap); w != nil && w.Implemented {
					cc.Classification = Breaking
					cc.Reason = fmt.Sprintf("new method on implemented interface %s", c.Symbol.Name)
					cc.AffectedFiles = w.Implementors
					cc.Reason += fmt.Sprintf(" [%s]", w.Criticality)
				} else {
					cc.Classification = Additive
					cc.Reason = fmt.Sprintf("new method %s (interface not implemented by monorepo)", c.Symbol.Name)
				}
			} else {
				cc.Classification = Additive
				cc.Reason = fmt.Sprintf("new %s %s", c.Symbol.Kind, c.Symbol.Name)
			}
		}

		classified = append(classified, cc)
	}

	return classified
}

// OverallClassification returns the worst classification from a set.
func OverallClassification(changes []ClassifiedChange) Classification {
	result := Safe
	for _, c := range changes {
		switch c.Classification {
		case Breaking:
			return Breaking
		case Additive:
			result = Additive
		}
	}
	return result
}

func findWatchedInterface(sym astdiff.Symbol, ifaceMap map[string]*WatchedInterface) *WatchedInterface {
	// For methods like "Queue.Enqueue", check "pkgbase.Queue"
	parts := splitFirst(sym.Name, ".")
	if len(parts) == 2 {
		// Try to find by combining package info from file path
		for key, w := range ifaceMap {
			if keyMatchesSymbol(key, parts[0]) {
				return w
			}
		}
	}
	// Direct match for type-level changes
	for key, w := range ifaceMap {
		if keyMatchesSymbol(key, sym.Name) {
			return w
		}
	}
	return nil
}

func keyMatchesSymbol(key, name string) bool {
	// key is "pkg.Name", we want to match "Name"
	parts := splitFirst(key, ".")
	if len(parts) == 2 {
		return parts[1] == name
	}
	return key == name
}

func splitFirst(s, sep string) []string {
	for i := 0; i < len(s); i++ {
		if string(s[i]) == sep {
			return []string{s[:i], s[i+1:]}
		}
	}
	return []string{s}
}

func lastPathComponent(path string) string {
	for i := len(path) - 1; i >= 0; i-- {
		if path[i] == '/' {
			return path[i+1:]
		}
	}
	return path
}

// VerifyImplementors checks that listed implementor files exist in monorepoDir.
func VerifyImplementors(registry []WatchedInterface, monorepoDir string) []string {
	var warnings []string
	for _, w := range registry {
		for _, impl := range w.Implementors {
			path := monorepoDir + "/" + impl
			if _, err := os.Stat(path); os.IsNotExist(err) {
				warnings = append(warnings, fmt.Sprintf("warning: %s.%s implementor %s not found", w.Package, w.Name, impl))
			}
		}
	}
	return warnings
}
