BTree map based on [google/btree](https://github.com/google/btree)

[![Build Status](https://github.com/RaduBerinde/btreemap/actions/workflows/ci.yaml/badge.svg)](https://github.com/RaduBerinde/btreemap/actions/workflows/ci.yaml)
[![Go Report Card](https://goreportcard.com/badge/github.com/RaduBerinde/btreemap)](https://goreportcard.com/report/github.com/RaduBerinde/btreemap)
[![GoDoc](https://godoc.org/github.com/RaduBerinde/btreemap?status.svg)](https://godoc.org/github.com/RaduBerinde/btreemap)

# btreemap

> An **ordered, in‑memory B‑Tree map for Go** – a generic key‑value store with fast range iteration and copy‑on‑write snapshots.
>
> **Forked from [`github.com/google/btree`](https://github.com/google/btree)** and modernized for Go1.22+ generics and [`iter`](https://pkg.go.dev/iter) sequences.

---

## Features

* **Ordered map** – keys are kept sorted; all operations are *O(log(n))*.
* **Generic** – works with any key/value types via a user‑supplied compare function.
* **Fast range iteration** – iterate forward or backward between arbitrary bounds without allocation.
* **Copy‑on‑write snapshots** – `Clone` produces a cheap, immutable view that can be read concurrently with the parent.
* **Custom degree / free‑list** – tune memory use vs. CPU by choosing the node degree and enabling node reuse.

> **Concurrency note**  
> *Reads* may run concurrently on the *same* `BTreeMap`. *Writes* (any method that mutates the map) must be serialized.  
> `Clone` creates an independent snapshot that may be mutated safely in parallel with the original.

## Quick start

```go
package main

import (
  "cmp"
  "fmt"
  "github.com/RaduBerinde/btreemap"
)

func main() {
  // A 2‑3‑4 tree mapping int → string.
  m := btreemap.New[int,string](2, cmp.Compare[int])

  // Insert / replace
  _, _, replaced := m.ReplaceOrInsert(42, "meaning")
  fmt.Println(replaced) // false – key was new

  // Lookup
  _, v, ok := m.Get(42)
  fmt.Println(ok, v)   // true meaning

  // Range iteration: 10 ≤ k < 100
  for k, v := range m.Ascend(btreemap.GE(10), btreemap.LT(100)) {
    fmt.Printf("%d → %s\n", k, v)
  }

  fmt.Println("len before:", m.Len())

  // Delete single key
  m.Delete(42)
  fmt.Println("len after:", m.Len())
}
```

---

## API overview

```go
// Construction
func New[K,V any](degree int, cmp btreemap.CmpFunc[K]) *BTreeMap[K,V]
func NewWithFreeList[K,V any](degree int, cmp CmpFunc[K], fl *FreeList[K,V]) *BTreeMap[K,V]

// Mutations
ReplaceOrInsert(key, value) (oldKey, oldValue, replaced bool)
Delete(key) (oldKey, oldValue, found bool)
DeleteMin() / DeleteMax()
Clear(addNodesToFreeList bool)

// Queries
Get(key) (key, value, found bool)
Has(key) bool
Min()/Max() (key, value, found bool)
Len() int

// Iteration (log(n) to first element, then amortized O(1))
Ascend(start LowerBound[K], stop UpperBound[K]) iter.Seq2[K,V]
Descend(start UpperBound[K], stop LowerBound[K]) iter.Seq2[K,V]

// Snapshots
Clone() *BTreeMap[K,V]
```

### Bounds helpers

```go
// Lower bound
btreemap.Min[T]()        // unbounded (−∞)
btreemap.GE(key)         // ≥ key (inclusive)
btreemap.GT(key)         // > key (exclusive)

// Upper bound
btreemap.Max[T]()        // unbounded (+∞)
btreemap.LE(key)         // ≤ key (inclusive)
btreemap.LT(key)         // < key (exclusive)
```

### Example: descending top‑N

```go
// Print the 5 largest entries.
count := 0
for k, v := range m.Descend(btreemap.Max[int](), btreemap.Min[int]()) {
  fmt.Println(k, v)
  count++
  if count == 5 {
    break
  }		
}
```

### Example: snapshot for concurrent readers

```go
snapshot := m.Clone() // cheap, O(1)
go func() {
  // Writer goroutine mutates the original map.
  for i := 0; i < 1_000; i++ {
    m.ReplaceOrInsert(i, "val")
  }
}()

// Reader goroutine works on an immutable view.
for k, v := range snapshot.Ascend(btreemap.Min[int](), btreemap.Max[int]()) {
  fmt.Println(k, v)
})
```

---

## Tuning

| Parameter | Effect |
|-----------|--------|
| **degree** | Maximum node size = `2*degree-1`. Small degrees use more pointers but less per‑node scan time (good for small maps). Higher degrees shrink tree height and improve cache locality for large maps. Typical values:`2…8`. |
| **FreeList** | Reuses nodes after deletes/`Clear`, reducing GC pressure in high‑churn workloads. Use `NewWithFreeList` if you need fine control over freelist size or sharing between many maps. |

---

## License

Apache 2.0 – see [LICENSE](LICENSE).  
Original work ©Google Inc. 2014‑2024