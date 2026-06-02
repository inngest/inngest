# Region Tree

## Overview

Region Tree is a data structure that partitions a one-dimensional space into
contiguous segments (called *regions*), each associated with a value (called a
*property*). It is useful for representing piecewise-constant data along a line
(such as time intervals, indices, or coordinates) and supports efficient updates
and queries. Adjacent regions with the same property are automatically merged,
which helps keep the representation compact and easy to manage.

**Key Features:**

- **One-dimensional interval management:** Represents a set of non-overlapping
  intervals (regions) along a 1D axis, each with an associated property/value.

- **Automatic merging:** If two neighboring regions have equal properties, they
  merge into a single region (no duplicate adjacent segments with the same
  value).

- **Efficient updates:** Supports fast range updates (changing the property of
  all regions in a given interval) in O(log N + K) time, where N is the number
  of regions and K is the number of regions affected by the update.

- **Efficient queries:** Can enumerate (iterate over) all regions overlapping a
  query range, or all regions in the structure, skipping regions with the default
  value.

- **Generic design:** Works with any ordered boundary type (e.g. numbers,
  coordinates, etc.) and any property type. You provide a comparison function
  for boundaries and an equality function for properties.

- **Copy-on-write cloning:** Supports cheap cloning of the entire tree. A clone
  operation is O(1) (lazy copy), allowing you to fork versions of the region set
  without an expensive deep copy. Subsequent modifications on either copy will
  not affect the other (they diverge on first write).

## How It Works

Region Tree maintains a set of **boundary points** that divide the line into
contiguous regions. Each region covers the half-open interval from one boundary
up to (but not including) the next boundary, and has an associated property
value. By convention, the area up to the first boundary is considered to have
the "zero" property (the default value for the property type). The data
structure aims to store only boundaries where the property changes.

Internally, the region tree uses a B-tree to store the regions by their start
boundary, enabling logarithmic search and update. However, you do not need to
interact with the B-tree directly - the region tree API provides
high-level methods to update and query the structure.

## Usage Examples

Below are simple examples illustrating how to create a region tree, update
regions, and query them. These examples use integer boundaries and integer
properties for simplicity.

### Creating a Region Tree

To create a region tree, you call the `Make` function with a comparison
function for the boundary type and an equality function for the property type.
For basic types like integers, you can typically use a standard comparator and
equality check:

```go
import (
	"cmp"
    "fmt"
    "github.com/RaduBerinde/axisds/regiontree"
)

func main() {
    // Create a region tree with int boundaries and int properties.
    // Use a default integer comparison and equality check.
    rt := regiontree.Make[int, int](
        cmp.Compare[int],           // comparison for boundaries
        func(a, b int) bool {       // equality check for properties
            return a == b 
        },
    )

    fmt.Println(rt.IsEmpty())  // Expect "true" as no regions have been set yet.
}
```

In the above snippet:
- `cmp.Compare[int]` is a comparator that orders two `int` boundaries (this
  could be a simple function returning negative, zero, or positive for a<b,
  a==b, a>b).
- The property equality function `func(a, b int) bool { return a == b }` tells
  the region tree to treat two properties as equal if their values are exactly
  equal. This determines when regions can be merged. (For non-numeric properties,
  you might provide a different function. For example, if `P` is a struct, you may
  want to compare a specific field for equality.)

Initially, the tree is empty (no non-default regions), so `rt.IsEmpty()` returns
true.

### Setting (Inserting) a Region

You don't insert regions by specifying explicit intervals; instead you **update
a range** by providing a function that sets or modifies the property for that
range. For example, to mark the interval [10, 30) with a property value of `1`:

```go
// Set the property to 1 for the interval [10, 30).
rt.Update(10, 30, func(oldValue int) int {
    return 1  // ignore oldValue, simply set property to 1
})
```

After this update, the region [10, 30) has property `1`. Everything outside
[10, 30) remains at the default property (`0` in this case). Internally, the
tree now has boundaries at 10 and 30, defining one explicit region:
- [10, 30) = 1 (and implicitly, [30, ∞) = 0 as default, and [−∞, 10) = 0 as
  default, which are not stored since they are default).

If we update an adjacent interval with the same property, the regions will
merge. For example:

```go
// Set the property to 1 for [30, 40) as well.
rt.Update(30, 40, func(oldValue int) int {
    return 1
})
```

Now the interval [10, 40) will become one continuous region with property 1.
Initially, a boundary at 30 was present, but after the second update the
properties on both sides of that boundary are equal (both sides became 1), so
the region tree removes the unnecessary boundary at 30. The regions [10, 30) and
[30, 40) coalesce into a single region [10, 40) = 1.

### Overlapping Updates and Automatic Splitting

If you update a range that overlaps existing regions with a *different*
property, region tree will split and/or trim regions as needed. For example:

```go
// Set the property to 2 for [15, 25), overlapping the [10, 40) region.
rt.Update(15, 25, func(oldValue int) int {
    return 2
})
```

Before this update, we had [10, 40) = 1. The update [15, 25) = 2 intersects the
middle of [10, 40). The region tree will handle this by splitting the original
region into pieces and assigning new values accordingly:
- [10, 15) remains with property 1 (the portion before 15, unchanged).
- [15, 25) will have property 2 (the updated range).
- [25, 40) reverts to property 1 (the portion after 25, which was part of the
  original region and remains property 1).

The structure automatically creates boundaries at 15 and 25 during the update.
After this operation, the tree contains three regions:
- [10, 15) = 1
- [15, 25) = 2
- [25, 40) = 1

Notice that [10, 15) and [25, 40) both have property 1, but they are not
adjacent - there is a region with property 2 in between - so they remain
separate segments in the tree.

### Removing a Region (Setting Back to Default)

You can "remove" or clear a segment by updating it back to the default property
(zero value). For instance, continuing from the above state, if we want to clear
the interval [15, 25) (setting it back to 0):

```go
// Reset the property to 0 (default) for [15, 25), effectively removing that segment.
rt.Update(15, 25, func(oldValue int) int {
    return 0  // set back to default
})
```

After this operation, the [15, 25) segment with property 2 is removed. The
regions [10, 15) = 1 and [25, 40) = 1 remain, with [15, 25) now being default
(0).

Everything from 15 to 25 and outside 10 to 40 is just default (0) and not stored as a region.

### Enumerating (Querying) Regions

To retrieve the regions and their properties, you use the **Enumerate** methods.
The `Enumerate(start, end, emitFunc)` method iterates over all regions that
overlap the interval `[start, end)` and calls `emitFunc` for each with the
region's start, end, and property. There is also a convenience method
`EnumerateAll(emitFunc)` to iterate over *all* regions in the tree (with
non-default property).

Let's enumerate all regions in our example tree:

```go
rt.EnumerateAll(func(start, end int, prop int) bool {
    fmt.Printf("[%d, %d) = %d\n", start, end, prop)
    return true  // return false if you want to stop early
})
// Output:
// [10, 15) = 1
// [25, 40) = 1
```

As expected, it listed the two regions with property 1. Regions with the default
property (0) are omitted from enumeration output. If we want to query a specific
sub-range, say [0, 30), we can do:

```go
rt.Enumerate(0, 30, func(start, end int, prop int) bool {
    fmt.Printf("[%d, %d) = %d\n", start, end, prop)
    return true
})
// Output:
// [10, 15) = 1
// [25, 30) = 1
```

This prints only the portion of our regions that lies between 0 and 30. In this
case, [25, 40) was partially outside the query range (only [25, 30) falls in
[0,30)).

**Note on the callback:** The `emitFunc` should return a boolean. Return `true`
to continue enumeration, or `false` to stop early (useful if you only want the
first overlapping region, for example). The enumeration is done in ascending
order of region start boundaries.

### Cloning the Tree

If you need to work with a snapshot of the regions and modify it independently,
you can use `Clone()`:

```go
clone := rt.Clone()
// 'clone' now shares structure with 'rt' but can be modified independently.

// For example, modify the clone:
clone.Update(0, 100, func(old int) int { return 5 })

// The original 'rt' is unchanged; 'clone' now has its own set of regions.
```

Cloning is very fast (constant time) and implemented with copy-on-write. This
means initially the clone and original share the same underlying data, but as
soon as you perform an update on one of them, that update will not affect the
other. This is useful for branching scenarios or caching snapshots of the
interval state at a moment in time.