package cluster

import (
	"math/rand"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/constraintapi"
	"github.com/inngest/inngest/pkg/constraintapi/memory"
	"github.com/inngest/inngest/pkg/enums"
	"github.com/stretchr/testify/require"
)

var allKinds = []memory.RecordKind{
	memory.RecSet,
	memory.RecSetDrop,
	memory.RecGrant,
	memory.RecTake,
	memory.RecExtend,
	memory.RecTAT,
	memory.RecCell,
	memory.RecDelta,
	memory.RecIdem,
	memory.RecSeqs,
}

func randUUID(rng *rand.Rand) (id uuid.UUID) {
	rng.Read(id[:])
	return id
}

func randSeqs(rng *rand.Rand, n int) []uint64 {
	seqs := make([]uint64, n)
	for i := range seqs {
		seqs[i] = rng.Uint64()
	}
	return seqs
}

func randSet(rng *rand.Rand) *memory.SetDef {
	return &memory.SetDef{
		ID:            rng.Uint64(),
		AccountID:     randUUID(rng),
		EnvID:         randUUID(rng),
		FunctionID:    randUUID(rng),
		AppID:         randUUID(rng),
		ConfigVersion: rng.Int(),
		Items: []constraintapi.ConstraintItem{
			{
				Kind: constraintapi.ConstraintKindConcurrency,
				Concurrency: &constraintapi.ConcurrencyConstraint{
					Mode:              enums.ConcurrencyModeRun,
					Scope:             enums.ConcurrencyScopeAccount,
					KeyExpressionHash: uuid.NewString(),
					EvaluatedKeyHash:  uuid.NewString(),
				},
			},
			{
				Kind: constraintapi.ConstraintKindThrottle,
				Throttle: &constraintapi.ThrottleConstraint{
					Scope:             enums.ThrottleScopeFn,
					KeyExpressionHash: uuid.NewString(),
				},
			},
			{
				Kind: constraintapi.ConstraintKindRateLimit,
				RateLimit: &constraintapi.RateLimitConstraint{
					Scope:            enums.RateLimitScopeEnv,
					EvaluatedKeyHash: uuid.NewString(),
				},
			},
			{
				Kind: constraintapi.ConstraintKindSemaphore,
				Semaphore: &constraintapi.SemaphoreConstraint{
					ID:      "fn:" + uuid.NewString(),
					Weight:  rng.Int63(),
					Release: constraintapi.SemaphoreReleaseManual,
				},
			},
		},
		Limits: []constraintapi.ConstraintLimits{
			{
				Limit: rng.Int(),
			},
			{
				Limit:  rng.Intn(1000),
				Burst:  rng.Intn(1000),
				Period: rng.Intn(100000),
			},
			{
				Limit:  -rng.Intn(1000),
				Period: rng.Int(),
			},
			{},
		},
	}
}

// randIdem gives half the grants and extends an idempotency record, and
// marks half of those as the export form that creates no lease.  a record
// with none has a zero key and a zero expiry.
func randIdem(rng *rand.Rand, r *memory.Record) {
	if rng.Intn(2) == 0 {
		return
	}
	r.KeyA = rng.Uint64()
	r.Value = 1 + rng.Int63()
	r.Force = rng.Intn(2) == 0
}

// randRecord fills every field the kind uses with a random value and leaves
// the other fields zero, which is what a decoded record looks like.
func randRecord(rng *rand.Rand, kind memory.RecordKind) memory.Record {
	r := memory.Record{Kind: kind}
	switch kind {
	case memory.RecSet:
		r.SetID = rng.Uint64()
		r.Set = randSet(rng)
	case memory.RecSetDrop:
		r.SetID = rng.Uint64()
	case memory.RecGrant:
		r.SetID = rng.Uint64()
		r.Seqs = randSeqs(rng, 1+rng.Intn(4))
		r.ExpiresAtMS = rng.Int63()
		r.Requested = rng.Int31()
		r.Granted = -rng.Int31()
		r.MaximumLifetime = time.Duration(rng.Int63())
		r.Source = constraintapi.LeaseSource{
			Service:           constraintapi.LeaseService(rng.Intn(5)),
			Location:          constraintapi.CallerLocation(rng.Intn(6)),
			RunProcessingMode: constraintapi.RunProcessingMode(rng.Intn(2)),
		}
		randIdem(rng, &r)
	case memory.RecTake:
		r.Seq = rng.Uint64()
		r.Force = rng.Intn(2) == 0
	case memory.RecExtend:
		r.Seq = rng.Uint64()
		r.NewSeq = rng.Uint64()
		r.ExpiresAtMS = -rng.Int63()
		randIdem(rng, &r)
	case memory.RecTAT:
		r.KeyA = rng.Uint64()
		r.KeyB = rng.Uint64()
		r.TAT = rng.NormFloat64() * 1e12
		r.ExpiresAtMS = rng.Int63()
		r.Version = rng.Uint64()
	case memory.RecCell, memory.RecDelta:
		r.KeyA = rng.Uint64()
		r.KeyB = rng.Uint64()
		r.Value = rng.Int63() - rng.Int63()
	case memory.RecIdem:
		r.KeyA = rng.Uint64()
		r.Value = rng.Int63() - rng.Int63()
		r.ExpiresAtMS = rng.Int63()
	case memory.RecSeqs:
		r.Seqs = randSeqs(rng, 16)
	}
	return r
}

func randRecords(rng *rand.Rand) []memory.Record {
	recs := make([]memory.Record, len(allKinds))
	for i, kind := range allKinds {
		recs[i] = randRecord(rng, kind)
	}
	return recs
}

func TestRecordsRoundTrip(t *testing.T) {
	rng := rand.New(rand.NewSource(1))
	for range 200 {
		recs := randRecords(rng)
		rng.Shuffle(len(recs), func(i, j int) { recs[i], recs[j] = recs[j], recs[i] })

		enc := AppendRecords([]byte("head"), recs)
		got, err := DecodeRecords(nil, enc[4:])
		require.NoError(t, err)
		require.Equal(t, recs, got)
	}

	// a set record with no definition and an empty list keep their form.
	bare := []memory.Record{
		{
			Kind:  memory.RecSet,
			SetID: 7,
		},
		{
			Kind: memory.RecSet,
			Set:  &memory.SetDef{},
		},
	}
	got, err := DecodeRecords(nil, AppendRecords(nil, bare))
	require.NoError(t, err)
	require.Equal(t, bare, got)

	require.Empty(t, AppendRecords(nil, nil))
	got, err = DecodeRecords(nil, nil)
	require.NoError(t, err)
	require.Nil(t, got)
}

// TestRecordsJoin checks that encodings joined end to end decode as one
// list, which is what the tail and the sender rely on, and that decoding
// appends to the buffer it is given.
func TestRecordsJoin(t *testing.T) {
	rng := rand.New(rand.NewSource(11))
	a, b, c := randRecords(rng), mixedBatch(rng, 37), randRecords(rng)
	joined := AppendRecords(nil, a)
	joined = append(joined, AppendRecords(nil, b)...)
	joined = append(joined, AppendRecords(nil, c)...)

	want := append(append(append([]memory.Record(nil), a...), b...), c...)
	got, err := DecodeRecords(nil, joined)
	require.NoError(t, err)
	require.Equal(t, want, got)

	// a buffer that is used again gives the same records, with nothing
	// left from the earlier use.
	buf := make([]memory.Record, 0, 4)
	buf, err = DecodeRecords(buf, AppendRecords(nil, c))
	require.NoError(t, err)
	buf, err = DecodeRecords(buf[:0], AppendRecords(nil, b))
	require.NoError(t, err)
	require.Equal(t, b, buf)

	head := []memory.Record{{Kind: memory.RecSetDrop, SetID: 1}}
	got, err = DecodeRecords(head, AppendRecords(nil, a))
	require.NoError(t, err)
	require.Equal(t, append(head[:1:1], a...), got)
}

func TestDecodeRecordsDoesNotAliasInput(t *testing.T) {
	recs := randRecords(rand.New(rand.NewSource(2)))
	enc := AppendRecords(nil, recs)
	got, err := DecodeRecords(nil, enc)
	require.NoError(t, err)
	for i := range enc {
		enc[i] = 0xff
	}
	require.Equal(t, recs, got)
}

func TestDecodeRecordsErrors(t *testing.T) {
	recs := randRecords(rand.New(rand.NewSource(3)))
	enc := AppendRecords(nil, recs)

	// there is no count in front.  a cut is an error unless it falls
	// between two records, and then it decodes as the records before it.
	ends := map[int]int{}
	for i := range recs {
		ends[len(AppendRecords(nil, recs[:i]))] = i
	}
	for n := range len(enc) {
		got, err := DecodeRecords(nil, enc[:n])
		if i, ok := ends[n]; ok {
			require.NoError(t, err, "cut at %d of %d", n, len(enc))
			require.Len(t, got, i)
			continue
		}
		require.Error(t, err, "cut at %d of %d", n, len(enc))
		require.Nil(t, got)
	}

	_, err := DecodeRecords(nil, []byte{0x3f, 0})
	require.ErrorContains(t, err, "unknown record kind")

	// a record of a kind this package does not know is written as its kind
	// byte alone.
	_, err = DecodeRecords(nil, AppendRecords(nil, []memory.Record{{Kind: 99}, {Kind: memory.RecSetDrop}}))
	require.Error(t, err)
}

func TestMessagesRoundTrip(t *testing.T) {
	rng := rand.New(rand.NewSource(4))
	recs := randRecords(rng)
	data := AppendRecords(nil, recs)
	msgs := []Message{
		Hello{
			Shard:   3,
			Node:    "node-a",
			Term:    rng.Uint64(),
			LastLSN: rng.Uint64(),
		},
		Hello{},
		Resume{Term: rng.Uint64()},
		StateBegin{Term: rng.Uint64()},
		StateRecords{
			Count: len(recs),
			Data:  data,
		},
		StateRecords{},
		StateEnd{LSN: rng.Uint64()},
		Batch{
			Term:     rng.Uint64(),
			FirstLSN: rng.Uint64(),
			Count:    len(recs),
			Data:     data,
		},
		Ack{LSN: rng.Uint64()},
		Fence{
			Shard: -1,
			Term:  rng.Uint64(),
		},
		FenceReply{
			OK:      true,
			LastLSN: rng.Uint64(),
		},
		TailRequest{
			Shard:   1 << 20,
			FromLSN: rng.Uint64(),
		},
		TailReply{
			OK:       true,
			FirstLSN: rng.Uint64(),
			Count:    len(recs),
			Data:     data,
		},
		TailReply{},
	}
	ids := map[byte]bool{}
	for _, m := range msgs {
		enc, err := EncodeMessage(nil, m)
		require.NoError(t, err)
		ids[enc[0]] = true
		got, err := DecodeMessage(enc)
		require.NoError(t, err)
		require.Equal(t, m, got)

		// the record bytes take the rest of the body, so a cut or an added
		// byte there is found when the records are decoded, not here.
		switch m.(type) {
		case StateRecords, Batch, TailReply:
			continue
		}
		for n := 1; n < len(enc); n++ {
			_, err := DecodeMessage(enc[:n])
			require.Error(t, err, "%T cut at %d of %d", m, n, len(enc))
		}
		_, err = DecodeMessage(append(enc, 0))
		require.ErrorIs(t, err, errTrailing)
	}
	require.Len(t, ids, 11)

	// the record bytes of a decoded message are a copy.
	enc, err := EncodeMessage(nil, Batch{
		Count: len(recs),
		Data:  data,
	})
	require.NoError(t, err)
	got, err := DecodeMessage(enc)
	require.NoError(t, err)
	for i := range enc {
		enc[i] = 0xff
	}
	require.Equal(t, data, got.(Batch).Data)

	// a count the bytes cannot contain is refused before anything is sized
	// by it.
	enc, err = EncodeMessage(nil, StateRecords{
		Count: 1 << 40,
		Data:  []byte{byte(memory.RecSetDrop), 1},
	})
	require.NoError(t, err)
	_, err = DecodeMessage(enc)
	require.ErrorIs(t, err, errShort)

	_, err = EncodeMessage(nil, &Ack{})
	require.Error(t, err, "a pointer is not a registered message")
}

func FuzzDecodeRecords(f *testing.F) {
	rng := rand.New(rand.NewSource(5))
	f.Add([]byte{})
	f.Add([]byte{0})
	f.Add([]byte{byte(memory.RecTake) | forceBit, 3, 9})
	f.Add([]byte{byte(memory.RecSeqs), 1, 0, 1, byte(memory.RecSet), 1, 0})
	f.Add([]byte{0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0x01})
	f.Add(AppendRecords(nil, randRecords(rng)))
	f.Add(AppendRecords(nil, mixedBatch(rng, 20)))
	f.Fuzz(func(t *testing.T, b []byte) {
		recs, err := DecodeRecords(nil, b)
		if err != nil {
			require.Nil(t, recs)
			return
		}
		// a record takes at least two bytes, so the input bounds what a
		// decode allocates.
		require.LessOrEqual(t, len(recs), len(b)/2)
		// what decodes also encodes, and the second decode reads all of it.
		again, err := DecodeRecords(nil, AppendRecords(nil, recs))
		require.NoError(t, err)
		require.Len(t, again, len(recs))
	})
}

// mixedBatch is n records in the mix one acquire and one release produce.
// each lease has a grant, a take and two TAT states.  the values have the
// size they have in production.
func mixedBatch(rng *rand.Rand, n int) []memory.Record {
	now := time.Now().UnixMilli()
	recs := make([]memory.Record, 0, n)
	for i := 0; len(recs) < n; i++ {
		seq := uint64(i%16)<<44 | uint64(1_000_000+i)
		tat := memory.Record{
			Kind:        memory.RecTAT,
			KeyA:        rng.Uint64(),
			KeyB:        rng.Uint64(),
			TAT:         float64(now) * 1e6,
			ExpiresAtMS: now + 60_000,
			Version:     uint64(now) << 20,
		}
		recs = append(recs, tat, memory.Record{
			Kind:            memory.RecGrant,
			SetID:           uint64(rng.Intn(5000)),
			Seqs:            []uint64{seq},
			ExpiresAtMS:     now + 30_000,
			Requested:       1,
			Granted:         1,
			MaximumLifetime: time.Hour,
			Source: constraintapi.LeaseSource{
				Service:  constraintapi.ServiceExecutor,
				Location: constraintapi.CallerLocationItemLease,
			},
			KeyA:  rng.Uint64(),
			Value: now + 5_000,
		}, tat, memory.Record{
			Kind: memory.RecTake,
			Seq:  seq,
		})
	}
	return recs[:n]
}

const benchBatch = 1000

func BenchmarkAppendRecords(b *testing.B) {
	recs := mixedBatch(rand.New(rand.NewSource(6)), benchBatch)
	buf := AppendRecords(nil, recs)
	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		buf = AppendRecords(buf[:0], recs)
	}
	b.ReportMetric(float64(b.Elapsed().Nanoseconds())/float64(b.N*benchBatch), "ns/record")
	b.ReportMetric(float64(len(buf))/benchBatch, "bytes/record")
}

// BenchmarkDecodeRecords decodes one batch into a new slice each time and
// into one buffer that is used again, which is what a follower does.
func BenchmarkDecodeRecords(b *testing.B) {
	buf := AppendRecords(nil, mixedBatch(rand.New(rand.NewSource(6)), benchBatch))
	for _, reuse := range []bool{false, true} {
		name := "fresh"
		if reuse {
			name = "reused"
		}
		b.Run(name, func(b *testing.B) {
			var recs []memory.Record
			b.ReportAllocs()
			b.ResetTimer()
			for range b.N {
				var err error
				if !reuse {
					recs = nil
				}
				if recs, err = DecodeRecords(recs[:0], buf); err != nil {
					b.Fatal(err)
				}
			}
			b.ReportMetric(float64(b.Elapsed().Nanoseconds())/float64(b.N*benchBatch), "ns/record")
			b.ReportMetric(float64(len(buf))/benchBatch, "bytes/record")
		})
	}
}

// TestDecodeRecordsAllocations checks that a decode into a buffer that is
// used again allocates the lease list of each grant and nothing else.
func TestDecodeRecordsAllocations(t *testing.T) {
	batch := mixedBatch(rand.New(rand.NewSource(6)), benchBatch)
	grants := 0
	for _, r := range batch {
		if r.Kind == memory.RecGrant {
			grants++
		}
	}
	require.NotZero(t, grants)
	buf := AppendRecords(nil, batch)
	recs, err := DecodeRecords(nil, buf)
	require.NoError(t, err)

	allocs := testing.AllocsPerRun(100, func() {
		if recs, err = DecodeRecords(recs[:0], buf); err != nil {
			t.Fatal(err)
		}
	})
	require.Equal(t, float64(grants), allocs)
}

// TestIdempotencyFieldsOnTheWire checks what a grant and an extend pay for
// their idempotency record, and that no other kind may claim one.
func TestIdempotencyFieldsOnTheWire(t *testing.T) {
	now := time.Now().UnixMilli()
	grant := memory.Record{
		Kind:        memory.RecGrant,
		SetID:       7,
		Seqs:        []uint64{3<<44 | 1_000_000},
		ExpiresAtMS: now + 30_000,
		Requested:   1,
		Granted:     1,
	}
	extend := memory.Record{
		Kind:        memory.RecExtend,
		Seq:         3<<44 | 1_000_000,
		NewSeq:      3<<44 | 1_000_001,
		ExpiresAtMS: now + 30_000,
	}
	for _, plain := range []memory.Record{grant, extend} {
		with := plain
		with.KeyA = 0xDEADBEEFCAFEF00D
		with.Value = now + 5_000

		short := AppendRecords(nil, []memory.Record{plain})
		long := AppendRecords(nil, []memory.Record{with})
		require.Equal(t, 8+3, len(long)-len(short), "the key hash and a three byte expiry distance")

		got, err := DecodeRecords(nil, long)
		require.NoError(t, err)
		require.Equal(t, []memory.Record{with}, got)

		got, err = DecodeRecords(nil, short)
		require.NoError(t, err)
		require.Equal(t, []memory.Record{plain}, got)
	}

	// a take with the idempotency bit set is refused
	bad := AppendRecords(nil, []memory.Record{{
		Kind: memory.RecTake,
		Seq:  5,
	}})
	bad[0] |= idemBit
	_, err := DecodeRecords(nil, bad)
	require.Error(t, err)
}
