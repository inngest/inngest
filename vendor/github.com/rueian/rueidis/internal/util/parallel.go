package util

import (
	"runtime"
	"sync"
)

func ParallelKeys[K comparable, V any](p map[K]V, fn func(k K)) {
	ch := make(chan K, len(p))
	for k := range p {
		ch <- k
	}
	closeThenParallel(ch, fn)
}

func ParallelVals[K comparable, V any](p map[K]V, fn func(k V)) {
	ch := make(chan V, len(p))
	for _, v := range p {
		ch <- v
	}
	closeThenParallel(ch, fn)
}

func closeThenParallel[V any](ch chan V, fn func(k V)) {
	close(ch)
	concurrency := len(ch)
	if cpus := runtime.NumCPU(); concurrency > cpus {
		concurrency = cpus
	}
	wg := sync.WaitGroup{}
	wg.Add(concurrency)
	for i := 1; i < concurrency; i++ {
		go func() {
			for v := range ch {
				fn(v)
			}
			wg.Done()
		}()
	}
	for v := range ch {
		fn(v)
	}
	wg.Done()
	wg.Wait()
}
