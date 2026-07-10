package executor

// Perf baseline for executor.Schedule (docs/plans/006-executor-readability-refactor.md,
// "Perf baseline"). Reuses deferTestInfra so this measures the same
// miniredis+sqlite path as the characterization tests; it is a signal for
// relative regressions across refactors, not a pure-allocation benchmark.

import "testing"

func BenchmarkSchedule(b *testing.B) {
	infra := newDeferTestInfra(b)
	exec := infra.newExecutor(b, nil)

	b.ReportAllocs()
	for b.Loop() {
		infra.scheduleRun(b, exec)
	}
}
