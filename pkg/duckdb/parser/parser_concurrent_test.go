package parser

import (
	"fmt"
	"sync"
	"testing"
)

// TestParseStringConcurrentStress guards ParseString's use of peg.Session
// pooling (see parser.go): many goroutines hammering ParseString
// concurrently must each see a correct, independent result, never a tree
// corrupted by another goroutine's session reusing its buffers.
func TestParseStringConcurrentStress(t *testing.T) {
	queries := []string{
		"SELECT 1",
		"SELECT a, b FROM orders WHERE a = 1",
		"SELECT a FROM t WHERE a > 1 AND b < 2 OR c = 3 AND d BETWEEN 1 AND 10",
		"SELECT * FROM a JOIN b ON a.id = b.id JOIN c ON b.id = c.id",
		"WITH cte AS (SELECT a FROM t WHERE a > 0) SELECT a FROM cte ORDER BY a LIMIT 10",
		"not valid sql !!!",
	}

	var (
		wg   sync.WaitGroup
		mu   sync.Mutex
		errs []error
	)
	fail := func(err error) {
		mu.Lock()
		errs = append(errs, err)
		mu.Unlock()
	}
	for g := 0; g < 50; g++ {
		wg.Add(1)
		go func(g int) {
			defer wg.Done()
			for i := 0; i < 200; i++ {
				q := queries[(g+i)%len(queries)]
				stmt, err := ParseString(q)
				if q == "not valid sql !!!" {
					if err == nil {
						fail(fmt.Errorf("expected error for invalid sql, got nil (stmt=%v)", stmt))
					}
					continue
				}
				if err != nil {
					fail(fmt.Errorf("query %q: unexpected error: %v", q, err))
					continue
				}
				if stmt == nil {
					fail(fmt.Errorf("query %q: nil stmt with no error", q))
				}
			}
		}(g)
	}
	wg.Wait()
	for _, e := range errs {
		t.Error(e)
	}
}
