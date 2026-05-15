// Copyright 2025 MinIO Inc.
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

package minlz

import (
	"fmt"
	"math"
	"math/bits"
	"sync"
)

// pools with hash tables for best encoding.
var encBestLPool sync.Pool
var encBestSPool sync.Pool

// encodeBlockBest encodes a non-empty src to a guaranteed-large-enough dst. It
// assumes that the varint-encoded length of the decompressed bytes has already
// been written.
//
// It also assumes that:
//
//	len(dst) >= MaxEncodedLen(len(src)) &&
//	minNonLiteralBlockSize <= len(src) && len(src) <= maxBlockSize
func encodeBlockBest(dst, src []byte, dict *dict) (d int) {
	// Initialize the hash tables.
	// TODO: dict
	const (
		// Long hash matches.
		lTableBits    = 20
		maxLTableSize = 1 << lTableBits

		// Short hash matches.
		sTableBits    = 18
		maxSTableSize = 1 << sTableBits

		inputMargin = 8 + 2

		debug = debugEncode
	)

	// sLimit is when to stop looking for offset/length copies. The inputMargin
	// lets us use a fast path for emitLiteral in the main loop, while we are
	// looking for copies.
	sLimit := len(src) - inputMargin
	if len(src) < minNonLiteralBlockSize {
		return 0
	}
	sLimitDict := len(src) - inputMargin
	if sLimitDict > maxDictSrcOffset-inputMargin {
		sLimitDict = maxDictSrcOffset - inputMargin
	}

	var lTable *[maxLTableSize]uint64
	if t := encBestLPool.Get(); t != nil {
		lTable = t.(*[maxLTableSize]uint64)
		*lTable = [maxLTableSize]uint64{}
	} else {
		lTable = new([maxLTableSize]uint64)
	}
	defer encBestLPool.Put(lTable)

	var sTable *[maxSTableSize]uint64
	if t := encBestSPool.Get(); t != nil {
		sTable = t.(*[maxSTableSize]uint64)
		*sTable = [maxSTableSize]uint64{}
	} else {
		sTable = new([maxSTableSize]uint64)
	}
	defer encBestSPool.Put(sTable)

	//var lTable [maxLTableSize]uint64
	//var sTable [maxSTableSize]uint64

	// Bail if we can't compress to at least this.
	dstLimit := len(src) - 5

	// nextEmit is where in src the next emitLiteral should start from.
	nextEmit := 0

	// The encoded form must start with a literal, as there are no previous
	// bytes to copy, so we start looking for hash matches at s == 1.
	s := 1
	repeat := 1
	if dict != nil {
		//dict.initBest()
		s = 0
		repeat = len(dict.dict) - dict.repeat
	}
	cv := load64(src, s)

	// We search for a repeat at -1, but don't output repeats when nextEmit == 0
	const lowbitMask = 0xffffffff
	getCur := func(x uint64) int {
		return int(x & lowbitMask)
	}
	getPrev := func(x uint64) int {
		return int(x >> 32)
	}
	const maxSkip = 64

	if debugEncode {
		fmt.Println("encodeBlockBest: Starting encode")
	}
	for {
		type match struct {
			offset    int
			s         int
			length    int
			score     int
			rep, dict bool
			nextrep   bool
		}
		var best match
		for {
			// Next src position to check
			nextS := (s-nextEmit)>>8 + 1
			if nextS > maxSkip {
				nextS = s + maxSkip
			} else {
				nextS += s
			}
			if nextS > sLimit {
				goto emitRemainder
			}
			if dict != nil && s >= maxDictSrcOffset {
				dict = nil
				if repeat > s {
					repeat = math.MinInt32
				}
			}
			hashL := hash8(cv, lTableBits)
			hashS := hash4(cv, sTableBits)
			candidateL := lTable[hashL]
			candidateS := sTable[hashS]

			score := func(m match) int {
				// Matches that are longer forward are penalized since we must emit it as a literal.
				ll := m.s - nextEmit
				// Bigger score is better.
				// -m.s indicates the base cost.
				score := m.length - emitLiteralSizeN(ll) - m.s
				offset := m.s - m.offset
				if m.rep {
					return score - emitRepeatSize(m.length)
				}

				if ll > 0 && offset > 1024 {
					// Check for fused discount
					if ll <= maxCopy2Lits && offset < 65536+63 && m.length <= copy2LitMaxLen {
						// 1-4 Literals can be embedded in copy2 without cost.
						score++
					} else if ll <= maxCopy3Lits {
						// 0-3 Literals can be embedded in copy3 without cost.
						score++
					}
				}
				return score - emitCopySize(offset, m.length)
			}

			matchAt := func(offset, s int, first uint32) match {
				if (best.length != 0 && best.s-best.offset == s-offset) || s-offset >= maxCopy3Offset || s <= offset {
					// Don't retest if we have the same offset.
					return match{offset: offset, s: s}
				}
				if debug && s == offset {
					panic(offset)
				}
				if load32(src, offset) != first {
					return match{offset: offset, s: s}
				}

				m := match{offset: offset, s: s, length: 4 + offset, rep: false}
				s += 4

				for s < len(src) {
					if len(src)-s < 8 {
						if src[s] == src[m.length] {
							m.length++
							s++
							continue
						}
						break
					}
					if diff := load64(src, s) ^ load64(src, m.length); diff != 0 {
						m.length += bits.TrailingZeros64(diff) >> 3
						break
					}
					s += 8
					m.length += 8
				}
				// Extend back...
				for m.s > nextEmit && m.offset > 0 {
					if src[m.offset-1] != src[m.s-1] {
						break
					}
					m.s--
					m.offset--
					m.length++
				}
				m.length -= offset

				m.score = score(m)
				if m.score <= -m.s {
					// Eliminate if no savings, we might find a better one.
					m.length = 0
				}
				if m.s+m.length < sLimit {
					const checkoff = 1
					a, b := m.s+m.length+checkoff, m.offset+m.length+checkoff
					m.nextrep = load32(src, a) == load32(src, b)
				}
				return m
			}
			matchAtRepeat := func(offset, s int, first uint32) match {
				if best.rep {
					// Don't retest if we already have a repeat
					return match{offset: offset, s: s}
				}
				// 2 gives close to no improvement,
				// since it may just give 'literal -> len 2 repeat -> literal' section.
				// which eats up the gains in overhead.
				// 3 gives pretty consistent improvement
				const checkbytes = 3
				mask := uint32((1 << (8 * checkbytes)) - 1)
				if load32(src, offset)&mask != first&mask {
					return match{offset: offset, s: s}
				}
				m := match{offset: offset, s: s, length: checkbytes + offset, rep: true}
				s += checkbytes
				for s < len(src) {
					if len(src)-s < 8 {
						if src[s] == src[m.length] {
							m.length++
							s++
							continue
						}
						break
					}
					if diff := load64(src, s) ^ load64(src, m.length); diff != 0 {
						m.length += bits.TrailingZeros64(diff) >> 3
						break
					}
					s += 8
					m.length += 8
				}
				// Extend back...
				for m.s > nextEmit && m.offset > 0 {
					if src[m.offset-1] != src[m.s-1] {
						break
					}
					m.s--
					m.offset--
					m.length++
				}
				m.length -= offset
				if m.s+m.length < sLimit {
					const checkoff = 1
					a, b := m.s+m.length+checkoff, m.offset+m.length+checkoff
					m.nextrep = load32(src, a) == load32(src, b)
				}
				m.score = score(m)
				if debug && m.length > 0 && m.length < 3 {
					fmt.Println("repeat", m.length, "offset", m.offset, "s", m.s, "score", m.score, "first", first, "mask", mask, "src", src[m.offset:m.offset+m.length], "src", src[m.s:m.s+m.length])
				}
				return m
			}
			matchDict := func(candidate, s int, first uint32, rep bool) match {
				if s >= maxDictSrcOffset {
					return match{offset: candidate, s: s}
				}
				// Calculate offset as if in continuous array with s
				offset := -len(dict.dict) + candidate
				if best.length != 0 && best.s-best.offset == s-offset && !rep {
					// Don't retest if we have the same offset.
					return match{offset: offset, s: s}
				}

				if load32(dict.dict, candidate) != first {
					return match{offset: offset, s: s}
				}
				m := match{offset: offset, s: s, length: 4 + candidate, rep: rep, dict: true}
				s += 4
				if !rep {
					for s < sLimitDict && m.length < len(dict.dict) {
						if len(src)-s < 8 || len(dict.dict)-m.length < 8 {
							if src[s] == dict.dict[m.length] {
								m.length++
								s++
								continue
							}
							break
						}
						if diff := load64(src, s) ^ load64(dict.dict, m.length); diff != 0 {
							m.length += bits.TrailingZeros64(diff) >> 3
							break
						}
						s += 8
						m.length += 8
					}
				} else {
					for s < len(src) && m.length < len(dict.dict) {
						if len(src)-s < 8 || len(dict.dict)-m.length < 8 {
							if src[s] == dict.dict[m.length] {
								m.length++
								s++
								continue
							}
							break
						}
						if diff := load64(src, s) ^ load64(dict.dict, m.length); diff != 0 {
							m.length += bits.TrailingZeros64(diff) >> 3
							break
						}
						s += 8
						m.length += 8
					}
				}
				m.length -= candidate
				m.score = score(m)
				if m.score <= -m.s {
					// Eliminate if no savings, we might find a better one.
					m.length = 0
				}
				return m
			}

			bestOf := func(a, b match) match {
				if b.length == 0 {
					return a
				}
				if a.length == 0 {
					return b
				}
				if a.score > b.score {
					return a
				}
				if b.score > a.score {
					return b
				}

				// Pick whichever starts the earliest,
				// we can probably find a match right away
				if a.s != b.s {
					if a.s < b.s {
						return a
					}
					return b
				}
				// If one is a good repeat candidate, pick it.
				if a.nextrep != b.nextrep {
					if a.nextrep {
						return a
					}
					return b
				}
				// Pick the smallest distance offset.
				if a.offset > b.offset {
					return a
				}
				return b
			}

			if s > 0 {
				best = bestOf(matchAt(getCur(candidateL), s, uint32(cv)), matchAt(getPrev(candidateL), s, uint32(cv)))
				best = bestOf(best, matchAt(getCur(candidateS), s, uint32(cv)))
				best = bestOf(best, matchAt(getPrev(candidateS), s, uint32(cv)))
			}
			if dict != nil {
				candidateL := dict.bestTableLong[hashL]
				candidateS := dict.bestTableShort[hashS]
				best = bestOf(best, matchDict(int(candidateL&0xffff), s, uint32(cv), false))
				best = bestOf(best, matchDict(int(candidateL>>16), s, uint32(cv), false))
				best = bestOf(best, matchDict(int(candidateS&0xffff), s, uint32(cv), false))
				best = bestOf(best, matchDict(int(candidateS>>16), s, uint32(cv), false))
			}
			{
				if dict == nil || repeat <= s {
					best = bestOf(best, matchAtRepeat(s-repeat, s, uint32(cv)))
					best = bestOf(best, matchAtRepeat(s-repeat+1, s+1, uint32(cv>>8)))
				} else if s-repeat < -4 && dict != nil {
					candidate := len(dict.dict) - (repeat - s)
					best = bestOf(best, matchDict(candidate, s, uint32(cv), true))
					candidate++
					best = bestOf(best, matchDict(candidate, s+1, uint32(cv>>8), true))
				}

				if best.length > 0 {
					hashS := hash4(cv>>8, sTableBits)
					// s+1
					nextShort := sTable[hashS]
					sFwd := s + 1
					cv := load64(src, sFwd)
					hashL := hash8(cv, lTableBits)
					nextLong := lTable[hashL]
					best = bestOf(best, matchAt(getCur(nextShort), sFwd, uint32(cv)))
					best = bestOf(best, matchAt(getPrev(nextShort), sFwd, uint32(cv)))
					best = bestOf(best, matchAt(getCur(nextLong), sFwd, uint32(cv)))
					best = bestOf(best, matchAt(getPrev(nextLong), sFwd, uint32(cv)))

					// dict at + 1
					if dict != nil {
						candidateL := dict.bestTableLong[hashL]
						candidateS := dict.bestTableShort[hashS]

						best = bestOf(best, matchDict(int(candidateL&0xffff), sFwd, uint32(cv), false))
						best = bestOf(best, matchDict(int(candidateS&0xffff), sFwd, uint32(cv), false))
					}

					// s+2
					if true {
						sFwd++
						cv = load64(src, sFwd)
						hashL := hash8(cv, lTableBits)
						nextLong = lTable[hashL]

						if dict == nil || repeat <= sFwd {
							// Repeat at + 2
							best = bestOf(best, matchAtRepeat(sFwd-repeat, sFwd, uint32(cv)))
						} else if repeat-sFwd > 4 && dict != nil {
							candidate := len(dict.dict) - (repeat - sFwd)
							best = bestOf(best, matchDict(candidate, sFwd, uint32(cv), true))
						}
						if true {
							hashS := hash4(cv, sTableBits)
							nextShort = sTable[hashS]
							best = bestOf(best, matchAt(getCur(nextShort), sFwd, uint32(cv)))
							best = bestOf(best, matchAt(getPrev(nextShort), sFwd, uint32(cv)))
						}
						best = bestOf(best, matchAt(getCur(nextLong), sFwd, uint32(cv)))
						best = bestOf(best, matchAt(getPrev(nextLong), sFwd, uint32(cv)))

						// dict at +2
						// Very small gain
						if dict != nil {
							candidateL := dict.bestTableLong[hashL]
							candidateS := dict.bestTableShort[hashS]

							best = bestOf(best, matchDict(int(candidateL&0xffff), sFwd, uint32(cv), false))
							best = bestOf(best, matchDict(int(candidateS&0xffff), sFwd, uint32(cv), false))
						}
					}

					// Search for a match at best match end, see if that is better.
					// Allow some bytes at the beginning to mismatch.
					// Sweet spot is around 1-2 bytes, but depends on input.
					// The skipped bytes are tested in Extend backwards,
					// and still picked up as part of the match if they do.
					const skipBeginning = 2
					const skipEnd = 1
					if sAt := best.s + best.length - skipEnd; sAt < sLimit {

						sBack := best.s + skipBeginning - skipEnd
						backL := best.length - skipBeginning
						// Load initial values
						cv = load64(src, sBack)

						// Grab candidates...
						next := lTable[hash8(load64(src, sAt), lTableBits)]

						if checkAt := getCur(next) - backL; checkAt > 0 {
							best = bestOf(best, matchAt(checkAt, sBack, uint32(cv)))
						}
						if checkAt := getPrev(next) - backL; checkAt > 0 {
							best = bestOf(best, matchAt(checkAt, sBack, uint32(cv)))
						}
						// Quite small gain, but generally a benefit on very compressible material.
						if true {
							next = sTable[hash4(load64(src, sAt), sTableBits)]
							if checkAt := getCur(next) - backL; checkAt > 0 {
								best = bestOf(best, matchAt(checkAt, sBack, uint32(cv)))
							}
							if checkAt := getPrev(next) - backL; checkAt > 0 {
								best = bestOf(best, matchAt(checkAt, sBack, uint32(cv)))
							}
						}
					}
				}
			}

			// Update table
			lTable[hashL] = uint64(s) | candidateL<<32
			sTable[hashS] = uint64(s) | candidateS<<32

			if best.length > 0 {
				break
			}

			cv = load64(src, nextS)
			s = nextS
		}
		startIdx := s + 1
		s = best.s

		if debug && best.offset >= s {
			panic(fmt.Errorf("t %d >= s %d", best.offset, s))
		}
		// Bail if we exceed the maximum size.
		if d+(s-nextEmit) > dstLimit {
			return 0
		}

		base := s
		offset := s - best.offset
		s += best.length
		// Bail if the match is equal or worse to the encoding.
		if !best.rep && best.length <= 4 {
			if offset > 65535 ||
				// Output will almost always be the same, and decoding will be slightly slower.
				// We might find a better match before end of these 4 bytes.
				(offset > maxCopy1Offset && offset <= maxCopy2Offset && base-nextEmit > maxCopy2Lits) {
				s = startIdx + 1
				if s >= sLimit {
					goto emitRemainder
				}
				cv = load64(src, s)
				continue
			}
		}
		if debug && nextEmit != base {
			fmt.Println("EMIT", base-nextEmit, "literals. base-after:", base)
		}

		if best.rep {
			if debug {
				fmt.Println("REPEAT, length", best.length, "offset:", offset, "s-after:", s, "dict:", best.dict, "best:", best)
			}
			d += emitLiteral(dst[d:], src[nextEmit:base])
			// same as `d := emitCopy(dst[d:], repeat, s-base)` but skips storing offset.
			d += emitRepeat(dst[d:], best.length)
		} else {
			lits := src[nextEmit:base]
			if debug {
				fmt.Println("COPY, length", best.length, "offset:", offset, "s-after:", s, "dict:", best.dict, "best:", best, "lits:", len(lits))
			}
			if len(lits) > 0 {
				if offset <= maxCopy2Offset {
					// 1-2 byte offsets
					if len(lits) > maxCopy2Lits || offset < 64 || (offset <= 1024 && best.length > copy2LitMaxLen) {
						d += emitLiteral(dst[d:], lits)
						if best.length > 18 && best.length <= 64 && offset >= 64 {
							// Size is equal.
							// Prefer Copy2, since it decodes faster
							d += encodeCopy2(dst[d:], offset, best.length)
						} else {
							d += emitCopy(dst[d:], offset, best.length)
						}
					} else {
						if best.length > 11 {
							// We are emitting remaining as a separate repeat.
							// We might as well do a search for a better match.
							d += emitCopyLits2(dst[d:], lits, offset, 11)
							s = best.s + 11
						} else {
							d += emitCopyLits2(dst[d:], lits, offset, best.length)
						}
					}
				} else {
					// 3 byte offset
					if len(lits) > maxCopy3Lits {
						d += emitLiteral(dst[d:], lits)
						d += emitCopy(dst[d:], offset, best.length)
					} else {
						d += emitCopyLits3(dst[d:], lits, offset, best.length)
					}
				}
			} else {
				if best.length > 18 && best.length <= 64 && offset >= 64 && offset <= maxCopy2Offset {
					// Size is equal.
					// Prefer Copy2, since it decodes faster
					d += encodeCopy2(dst[d:], offset, best.length)
				} else {
					d += emitCopy(dst[d:], offset, best.length)
				}
			}
		}
		repeat = offset

		nextEmit = s
		if s >= sLimit {
			goto emitRemainder
		}

		if d > dstLimit {
			// Do we have space for more, if not bail.
			return 0
		}
		// Fill tables...
		for i := startIdx; i < s; i++ {
			cv0 := load64(src, i)
			long0 := hash8(cv0, lTableBits)
			short0 := hash4(cv0, sTableBits)
			lTable[long0] = uint64(i) | lTable[long0]<<32
			sTable[short0] = uint64(i) | sTable[short0]<<32
		}
		cv = load64(src, s)
	}

emitRemainder:
	if nextEmit < len(src) {
		// Bail if we exceed the maximum size.
		litLen := len(src) - nextEmit
		if d+litLen+emitLiteralSizeN(litLen) > dstLimit {
			if debug && nextEmit != s {
				fmt.Println("emitting would exceed dstLimit. Not compressing")
			}
			return 0
		}
		if debug && nextEmit != s {
			fmt.Println("emitted ", len(src)-nextEmit, "literals")
		}
		d += emitLiteral(dst[d:], src[nextEmit:])
	}
	return d
}

// emitCopySize returns the size to encode the offset+length
//
// It assumes that:
//
//	1 <= offset && offset <= math.MaxUint32
//	4 <= length && length <= 1 << 24
func emitCopySize(offset, length int) int {
	if offset > 65536+63 {
		// 3 Byte offset + Variable length (base length 4).
		length -= 64 // Base is free. We can add 64 for free.
		if length <= 0 {
			return 4
		}
		return 4 + (bits.Len(uint(length))+7)/8
	}

	// Offset no more than 2 bytes.
	if offset <= 1024 {
		if length <= 18 {
			// Emit up to 18 bytes with short offset.
			return 2
		}
		if length < 18+256 {
			return 3
		}
		// Worst case we have to emit a repeat for the rest
		return 2 + emitRepeatSize(length-18)
	}
	// 2 byte offset + Variable length (base length 4).
	return emitCopy2Size(length)
}

// emitRepeatSize returns the number of bytes required to encode a repeat.
// Length must be at least 1 and < 1<<24
func emitRepeatSize(length int) int {
	if length <= 0 {
		return 0
	}

	if length <= 29 {
		return 1
	}
	length -= 29
	if length <= 256 {
		return 2
	}
	if length <= 65536 {
		return 3
	}
	return 4
}

// emitCopy2Size returns the number of bytes required to encode a copy2.
// Length must be less than 1<<24
func emitCopy2Size(length int) int {
	length -= 4

	if length <= 60 {
		// Length inside tag.
		return 3
	}
	length -= 60
	if length < 256 {
		// Length in 1 byte.
		return 4
	}
	if length < 65536 {
		// Length in 2 bytes.
		return 5
	}
	// Length in 3 bytes.
	return 6
}
