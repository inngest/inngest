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

package internal

import (
	"encoding/binary"
	"fmt"
	"math"

	"github.com/apache/iceberg-go"
	"github.com/twpayne/go-geom"
	"github.com/twpayne/go-geom/encoding/ewkb"
	"github.com/twpayne/go-geom/encoding/wkb"
	"github.com/twpayne/go-geom/encoding/wkbcommon"
)

// Geo bounding box dimension indices. X is longitude/easting, Y is
// latitude/northing; Z and M are optional. These index the per-dimension
// min/max/has arrays of geoBoundsAccumulator.
const (
	geoDimX = iota
	geoDimY
	geoDimZ
	geoDimM
	geoNumDims
)

var wkbEmptyPointOption = wkbcommon.WKBOptionEmptyPointHandling(wkbcommon.EmptyPointHandlingNaN)

// geoBoundsAccumulator computes a geospatial bounding box from a stream of WKB
// values. Iceberg stores geometry/geography column bounds using the single-value
// serialization for geospatial types (spec Appendix D): the concatenation of
// little-endian float64 coordinate values in X, Y[, Z][, M] order. The lower
// bound carries the per-dimension minimums and the upper bound the maximums.
// Null and NaN coordinate values are skipped, and a dimension that never sees a
// finite value is omitted from the resulting bounds. If the X or Y dimension is
// missing entirely, no bounding box is produced.
//
// Bounds are only emitted for geometry (planar edges). Geography bounds are
// omitted: see Bounds.
type geoBoundsAccumulator struct {
	min [geoNumDims]float64
	max [geoNumDims]float64
	has [geoNumDims]bool

	// geoms counts the leaf geometries that contributed coordinates; zGeoms and
	// mGeoms count how many of those carried a Z / M dimension. The has[] flags
	// are sticky across rows, so a column mixing e.g. XYZ and XYM geometries
	// would set both has[geoDimZ] and has[geoDimM] without any single row having
	// both. An optional dimension is only safe to emit when every geometry
	// carried it (count == geoms); otherwise the bound would imply rows that lack
	// the dimension have a value in range. See layout.
	geoms  int
	zGeoms int
	mGeoms int

	// isGeography marks a column whose edges are geodesics on a sphere, for
	// which raw vertex min/max is not a safe bounding box (see Bounds).
	isGeography bool
}

func newGeoBoundsAccumulator(isGeography bool) *geoBoundsAccumulator {
	return &geoBoundsAccumulator{isGeography: isGeography}
}

// AddWKB unmarshals a single WKB value and extends the bounding box with its
// coordinates. Both ISO WKB and EWKB values are accepted (see decodeWKB).
func (a *geoBoundsAccumulator) AddWKB(data []byte) error {
	g, err := decodeWKB(data)
	if err != nil {
		return err
	}

	a.extend(g)

	return nil
}

// EWKB dimension and SRID flags, carried in the high bits of the WKB type word.
const (
	ewkbFlagZ    uint32 = 0x80000000
	ewkbFlagM    uint32 = 0x40000000
	ewkbFlagSRID uint32 = 0x20000000
	ewkbFlags           = ewkbFlagZ | ewkbFlagM | ewkbFlagSRID
)

// WKB byte-order markers, the first byte of every WKB value.
const (
	wkbBigEndian    = 0
	wkbLittleEndian = 1
)

// decodeWKB decodes a geospatial value for statistics purposes, accepting both
// encodings found in the wild. Iceberg prescribes ISO WKB, which encodes the
// dimension in the type value itself (PointZ = 1001); some writers instead emit
// EWKB, which flags Z/M in the high bits of the type word and may embed an SRID
// after it. Both yield the same coordinates, so bounds do not depend on the
// encoding.
func decodeWKB(data []byte) (geom.T, error) {
	if isEWKB(data) {
		return ewkb.Unmarshal(data)
	}

	// ISO empty points use NaN coordinates; keep them as empty geometries so the
	// bounds accumulator can skip them instead of rejecting the value.
	return wkb.Unmarshal(data, wkbEmptyPointOption)
}

func normalizeWKB(data []byte) ([]byte, error) {
	if !isEWKB(data) {
		return data, nil
	}

	// EWKB decoding already preserves empty-point NaNs; the explicit option is
	// needed on the ISO marshal side so those coordinates remain an empty point.
	g, err := ewkb.Unmarshal(data)
	if err != nil {
		return nil, err
	}

	var order binary.ByteOrder = binary.LittleEndian
	if data[0] == wkbBigEndian {
		order = binary.BigEndian
	}

	return wkb.Marshal(g, order, wkbEmptyPointOption)
}

// isEWKB reports whether data's type word carries EWKB flags. The two decoders
// are not interchangeable: the ISO decoder rejects flagged type words, and the
// EWKB decoder rejects the ISO dimension offsets (it reads 1001 as an unknown
// type rather than PointZ), so the flags select which one applies. Values too
// short to hold a type word are left to the ISO decoder to reject.
//
// A 2D EWKB value without an SRID carries no flags, so it is reported as ISO and
// decoded by the ISO decoder. That is correct because the two encodings are
// byte-identical for plain 2D geometries; do not tighten the heuristic to claim
// such values as EWKB.
func isEWKB(data []byte) bool {
	if len(data) < 5 {
		return false
	}

	var typeWord uint32
	switch data[0] {
	case wkbBigEndian:
		typeWord = binary.BigEndian.Uint32(data[1:5])
	case wkbLittleEndian:
		typeWord = binary.LittleEndian.Uint32(data[1:5])
	default:
		return false
	}

	return typeWord&ewkbFlags != 0
}

// extend walks every coordinate of g, recursing into geometry collections,
// which cannot expose flat coordinates directly.
func (a *geoBoundsAccumulator) extend(g geom.T) {
	if gc, ok := g.(*geom.GeometryCollection); ok {
		for _, sub := range gc.Geoms() {
			a.extend(sub)
		}

		return
	}

	stride := g.Stride()
	flat := g.FlatCoords()
	if stride < 2 || len(flat) == 0 {
		return
	}

	layout := g.Layout()
	zIdx, mIdx := layout.ZIndex(), layout.MIndex()
	a.geoms++
	if zIdx >= 0 {
		a.zGeoms++
	}
	if mIdx >= 0 {
		a.mGeoms++
	}
	for base := 0; base+stride <= len(flat); base += stride {
		a.update(geoDimX, flat[base])
		a.update(geoDimY, flat[base+1])
		if zIdx >= 0 {
			a.update(geoDimZ, flat[base+zIdx])
		}
		if mIdx >= 0 {
			a.update(geoDimM, flat[base+mIdx])
		}
	}
}

func (a *geoBoundsAccumulator) update(dim int, v float64) {
	if math.IsNaN(v) {
		return
	}

	if !a.has[dim] {
		a.min[dim], a.max[dim] = v, v
		a.has[dim] = true

		return
	}

	if v < a.min[dim] {
		a.min[dim] = v
	}
	if v > a.max[dim] {
		a.max[dim] = v
	}
}

// layout selects the point layout for the bound points from the dimensions that
// saw finite values. The bool is false when the box has no X or Y dimension and
// therefore should not be emitted.
func (a *geoBoundsAccumulator) layout() (geom.Layout, bool) {
	if !a.has[geoDimX] || !a.has[geoDimY] {
		return geom.NoLayout, false
	}

	// An optional dimension is only emitted when every geometry carried it and it
	// saw a finite value. If Z or M is present in only some geometries (e.g. a
	// mix of XYZ and XYM rows, or XYZM and XYZ rows), emitting it would imply the
	// geometries that lack it have a value in range, driving wrong-answer pruning
	// once a reader wires up geo pruning; drop it to the largest safe layout.
	hasZ := a.has[geoDimZ] && a.zGeoms == a.geoms
	hasM := a.has[geoDimM] && a.mGeoms == a.geoms

	return geoPointLayout(hasZ, hasM), true
}

// geoPointLayout selects the bound point layout implied by which optional
// dimensions are present. X and Y are always assumed present; callers gate on
// that before calling.
func geoPointLayout(hasZ, hasM bool) geom.Layout {
	switch {
	case hasZ && hasM:
		return geom.XYZM
	case hasZ:
		return geom.XYZ
	case hasM:
		return geom.XYM
	default:
		return geom.XY
	}
}

// encodeGeoBound serializes a bound point using the Iceberg single-value
// serialization for geospatial types: little-endian float64 coordinates in
// X, Y[, Z][, M] order. Lengths are XY=16, XYZ=24, XYM=32, XYZM=32 bytes. For
// XYM the Z slot is written as NaN so a reader can tell XYM (NaN in slot 3)
// apart from XYZM (finite Z in slot 3).
func encodeGeoBound(vals [geoNumDims]float64, layout geom.Layout) []byte {
	var coords []float64
	switch layout {
	case geom.XYZ:
		coords = []float64{vals[geoDimX], vals[geoDimY], vals[geoDimZ]}
	case geom.XYM:
		coords = []float64{vals[geoDimX], vals[geoDimY], math.NaN(), vals[geoDimM]}
	case geom.XYZM:
		coords = []float64{vals[geoDimX], vals[geoDimY], vals[geoDimZ], vals[geoDimM]}
	default:
		coords = []float64{vals[geoDimX], vals[geoDimY]}
	}

	buf := make([]byte, len(coords)*8)
	for i, c := range coords {
		binary.LittleEndian.PutUint64(buf[i*8:], math.Float64bits(c))
	}

	return buf
}

// Bounds encodes the lower and upper bound points using the Iceberg geospatial
// single-value serialization (see encodeGeoBound). Both are nil when no bounding
// box could be produced (e.g. an all-null column).
//
// Geography bounds are always omitted. Geography edges are geodesics on a
// sphere, so a box built from raw vertex min/max is unsafe: a geodesic edge can
// reach a higher latitude than either endpoint, and the correct longitude
// interval may wrap across the antimeridian (xmin > xmax). Emitting naive vertex
// bounds would prune rows that actually fall inside a query region — a
// silent-wrong-results bug. Missing bounds only disable pruning, which is always
// safe, so geography is left unbounded until geodesic/antimeridian-aware
// computation is added.
func (a *geoBoundsAccumulator) Bounds() (lower, upper []byte) {
	layout, ok := a.layout()
	if !ok || a.isGeography {
		return nil, nil
	}

	return encodeGeoBound(a.min, layout), encodeGeoBound(a.max, layout)
}

// StatsAgg materializes the accumulated bounds into a StatsAgg carrying the
// precomputed single-point bounds, or nil when no box was produced. Unlike the
// generic statsAggregator, geo bounds are computed from raw WKB during the write
// (Parquet byte-array min/max over WKB are meaningless), so the aggregator just
// replays the precomputed bytes through the existing ToDataFile bounds plumbing.
func (a *geoBoundsAccumulator) StatsAgg() (StatsAgg, error) {
	lower, upper := a.Bounds()
	if lower == nil {
		return nil, nil
	}

	return &geoStatsAgg{lower: lower, upper: upper}, nil
}

// geoStatsAgg is a StatsAgg holding precomputed geo single-point bounds encoded
// with the Iceberg geospatial single-value serialization (see encodeGeoBound).
type geoStatsAgg struct {
	lower, upper []byte
}

func (g *geoStatsAgg) Min() iceberg.Literal {
	if g.lower == nil {
		return nil
	}

	return iceberg.BinaryLiteral(g.lower)
}

func (g *geoStatsAgg) Max() iceberg.Literal {
	if g.upper == nil {
		return nil
	}

	return iceberg.BinaryLiteral(g.upper)
}

// Update is a no-op. geoStatsAgg carries bounds precomputed from raw WKB during
// the write (see geoBoundsAccumulator), not min/max folded from Parquet column
// statistics. It satisfies the StatsAgg interface only so its precomputed bytes
// can flow through ToDataFile's ColAggs draining (via MinAsBytes/MaxAsBytes).
// Geo aggregators are injected into ColAggs after DataFileStatsFromMeta has
// finished folding Parquet stats, so nothing ever calls Update on one; any
// caller that does would be discarding precomputed bounds, so keep it a no-op.
func (g *geoStatsAgg) Update(interface{ HasMinMax() bool }) {}

func (g *geoStatsAgg) MinAsBytes() ([]byte, error) { return g.lower, nil }
func (g *geoStatsAgg) MaxAsBytes() ([]byte, error) { return g.upper, nil }

// decodeGeoBound is the inverse of encodeGeoBound: it parses an Iceberg
// geospatial single-value bound back into its per-dimension coordinates and the
// layout it was written with. A 32-byte bound is XYM when its Z slot is NaN and
// XYZM otherwise (see encodeGeoBound). ok is false when the length is not a
// valid bound length (16, 24, or 32 bytes).
func decodeGeoBound(data []byte) (vals [geoNumDims]float64, layout geom.Layout, ok bool) {
	var n int
	switch len(data) {
	case 16:
		n = 2
	case 24:
		n = 3
	case 32:
		n = 4
	default:
		return vals, geom.NoLayout, false
	}

	coords := make([]float64, n)
	for i := range coords {
		coords[i] = math.Float64frombits(binary.LittleEndian.Uint64(data[i*8:]))
	}

	vals[geoDimX], vals[geoDimY] = coords[0], coords[1]
	switch len(data) {
	case 16:
		layout = geom.XY
	case 24:
		vals[geoDimZ], layout = coords[2], geom.XYZ
	case 32:
		if math.IsNaN(coords[2]) {
			vals[geoDimM], layout = coords[3], geom.XYM
		} else {
			vals[geoDimZ], vals[geoDimM], layout = coords[2], coords[3], geom.XYZM
		}
	}

	return vals, layout, true
}

// GeoBoundsXY decodes a geometry column's lower and upper geo bounds (Iceberg
// single-value serialization; see encodeGeoBound) into their planar XY extents.
// ok is false when either bound is missing, malformed, or carries a NaN X/Y
// coordinate - all cases where the bound is unusable for pruning.
func GeoBoundsXY(lower, upper []byte) (minX, minY, maxX, maxY float64, ok bool) {
	lo, _, okLo := decodeGeoBound(lower)
	hi, _, okHi := decodeGeoBound(upper)
	if !okLo || !okHi {
		return 0, 0, 0, 0, false
	}
	if math.IsNaN(lo[geoDimX]) || math.IsNaN(lo[geoDimY]) ||
		math.IsNaN(hi[geoDimX]) || math.IsNaN(hi[geoDimY]) {
		return 0, 0, 0, 0, false
	}
	// Reject inverted bounds (lower > upper). iceberg-go's own accumulator never
	// emits them, but this is a trust-on-read decode path: a bound written by
	// another engine with lo > hi would make BBoxIntersectsXY always report
	// no-overlap and prune a file that should be kept. Treating it as unusable
	// leaves the file unprunable, which is always safe.
	if lo[geoDimX] > hi[geoDimX] || lo[geoDimY] > hi[geoDimY] {
		return 0, 0, 0, 0, false
	}

	return lo[geoDimX], lo[geoDimY], hi[geoDimX], hi[geoDimY], true
}

// BBoxIntersectsXY reports whether two planar (XY) bounding boxes intersect.
// Boxes touching only at an edge or corner count as intersecting (closed
// intervals), matching Iceberg's inclusive bbox pruning: a box that might
// contain a matching value must not be pruned. This is the geometry (planar)
// rule; geography columns emit no bounds (see Bounds), so the antimeridian
// wrap-around case never reaches pruning.
func BBoxIntersectsXY(aMinX, aMinY, aMaxX, aMaxY, bMinX, bMinY, bMaxX, bMaxY float64) bool {
	return aMinX <= bMaxX && aMaxX >= bMinX && aMinY <= bMaxY && aMaxY >= bMinY
}

// layoutHasZM reports which optional dimensions a bound layout carries.
func layoutHasZM(l geom.Layout) (hasZ, hasM bool) {
	switch l {
	case geom.XYZ:
		return true, false
	case geom.XYM:
		return false, true
	case geom.XYZM:
		return true, true
	default:
		return false, false
	}
}

// GeoBoundsAggregator combines the per-data-file geospatial bounds emitted by
// geoBoundsAccumulator (the single-value serialization written into a
// DataFile's lower/upper bounds; see Bounds) across multiple files into one
// bounding box. It is the manifest-level analogue of the primitive min/max
// aggregation the manifest writer already does per partition field, but geo
// bounds are coordinate tuples with no total order, so they must never be
// folded with a
// scalar byte comparison: each bound is decoded to its coordinates and merged
// dimension by dimension. The combined box is returned in the same
// serialization, so it round-trips through a manifest bound exactly like a
// per-file one.
//
// The aggregator is geometry-only: it folds each dimension with a scalar
// min/max, which is correct for geometry's planar bounds but wrong for
// geography. A geography box may cross the antimeridian, encoded as lower_x >
// upper_x (spec Appendix D); scalar min/max would silently unwrap it (merging a
// wrapped [170, -170] with [10, 20] yields [10, 20], dropping the wrapped
// range), producing a box that prunes rows it should keep. Because the bound
// bytes carry no type, callers must declare geography via NewGeoBoundsAggregator
// so Add can reject it (see Add). iceberg-go itself emits no per-file bounds for
// geography (see Bounds), so the common path — passing empty geography bounds
// through — is a harmless no-op; only non-empty geography bounds (e.g. from files
// written by another engine) are refused, until geodesic/antimeridian-aware
// aggregation is added.
type GeoBoundsAggregator struct {
	min [geoNumDims]float64
	max [geoNumDims]float64
	has [geoNumDims]bool

	// n counts the files added; zFiles and mFiles count how many carried a Z / M
	// dimension. An optional dimension is only emitted when every file carried it
	// (count == n) - the same omit-on-ambiguity rule geoBoundsAccumulator applies
	// across geometries within a single file. Emitting a Z/M that some files lack
	// would imply those files have a value in range and drive wrong-answer
	// pruning.
	n      int
	zFiles int
	mFiles int

	// isGeography marks a geography column, whose bounds must not be folded with
	// scalar min/max (see the type doc); Add refuses non-empty geography bounds.
	isGeography bool
}

// NewGeoBoundsAggregator returns an aggregator for a single geo column. Pass
// isGeography=true for geography columns so Add refuses their bounds rather than
// mis-merging antimeridian-crossing boxes (see GeoBoundsAggregator). The
// zero-value GeoBoundsAggregator is a valid geometry aggregator.
func NewGeoBoundsAggregator(isGeography bool) *GeoBoundsAggregator {
	return &GeoBoundsAggregator{isGeography: isGeography}
}

// Add merges one data file's lower and upper geo bounds into the aggregate. The
// two bounds must share a layout, which they always do: a file's lower and upper
// are written together by geoBoundsAccumulator. An empty pair (nil/empty lower
// and upper) contributes nothing, so files without geo bounds - including every
// geography file written by iceberg-go - can be passed through harmlessly. It
// errors when a bound is a non-empty but invalid length, when lower and upper
// disagree on layout, or when a non-empty bound is added to a geography
// aggregator (scalar folding would mis-merge antimeridian-crossing boxes; see
// GeoBoundsAggregator).
func (g *GeoBoundsAggregator) Add(lower, upper []byte) error {
	if len(lower) == 0 && len(upper) == 0 {
		return nil
	}

	if g.isGeography {
		return fmt.Errorf("%w: cannot aggregate geography geo bounds; "+
			"scalar min/max folding mis-merges antimeridian-crossing boxes",
			iceberg.ErrNotImplemented)
	}

	lo, loLayout, ok := decodeGeoBound(lower)
	if !ok {
		return fmt.Errorf("%w: geo lower bound must be 16, 24, or 32 bytes, got %d",
			iceberg.ErrInvalidBinSerialization, len(lower))
	}
	hi, hiLayout, ok := decodeGeoBound(upper)
	if !ok {
		return fmt.Errorf("%w: geo upper bound must be 16, 24, or 32 bytes, got %d",
			iceberg.ErrInvalidBinSerialization, len(upper))
	}
	if loLayout != hiLayout {
		return fmt.Errorf("%w: geo lower/upper bounds have mismatched layouts (%v vs %v)",
			iceberg.ErrInvalidBinSerialization, loLayout, hiLayout)
	}

	g.n++
	g.update(geoDimX, lo[geoDimX], hi[geoDimX])
	g.update(geoDimY, lo[geoDimY], hi[geoDimY])

	if hasZ, hasM := layoutHasZM(loLayout); hasZ || hasM {
		if hasZ {
			g.zFiles++
			g.update(geoDimZ, lo[geoDimZ], hi[geoDimZ])
		}
		if hasM {
			g.mFiles++
			g.update(geoDimM, lo[geoDimM], hi[geoDimM])
		}
	}

	return nil
}

// update folds one file's per-dimension lower/upper into the running min/max.
func (g *GeoBoundsAggregator) update(dim int, lo, hi float64) {
	if math.IsNaN(lo) || math.IsNaN(hi) {
		return
	}

	if !g.has[dim] {
		g.min[dim], g.max[dim] = lo, hi
		g.has[dim] = true

		return
	}

	if lo < g.min[dim] {
		g.min[dim] = lo
	}
	if hi > g.max[dim] {
		g.max[dim] = hi
	}
}

// Bounds returns the combined lower and upper bound points in the Iceberg
// geospatial single-value serialization, or nil when no file contributed a
// bounding box. An optional Z/M dimension is dropped unless every added file
// carried it (see the field docs).
func (g *GeoBoundsAggregator) Bounds() (lower, upper []byte) {
	if g.n == 0 || g.isGeography || !g.has[geoDimX] || !g.has[geoDimY] {
		return nil, nil
	}

	layout := geoPointLayout(g.has[geoDimZ] && g.zFiles == g.n, g.has[geoDimM] && g.mFiles == g.n)

	return encodeGeoBound(g.min, layout), encodeGeoBound(g.max, layout)
}
