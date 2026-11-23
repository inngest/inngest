package icebergcatalog

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sts"
	ststypes "github.com/aws/aws-sdk-go-v2/service/sts/types"
	"github.com/stretchr/testify/require"
)

type fakeSTS struct {
	in    *sts.AssumeRoleInput
	calls int
}

func (f *fakeSTS) AssumeRole(_ context.Context, in *sts.AssumeRoleInput, _ ...func(*sts.Options)) (*sts.AssumeRoleOutput, error) {
	f.in = in
	f.calls++
	return &sts.AssumeRoleOutput{Credentials: &ststypes.Credentials{
		AccessKeyId: aws.String("AK"), SecretAccessKey: aws.String("SK"), SessionToken: aws.String("TOK"),
		Expiration: aws.Time(time.UnixMilli(1_800_000_000_000)),
	}}, nil
}

func TestSTSVendor(t *testing.T) {
	f := &fakeSTS{}
	v := &STSVendor{STS: f, RoleARN: "arn:x", Duration: 15 * time.Minute, Region: "us-east-1", Endpoint: "http://seaweedfs:8333", PathStyle: true, ReadOnly: true}
	props, err := v.VendCredentials(t.Context(), []string{"inngest", "runs v2"}, "s3://bucket/wh/inngest.db/runs")
	require.NoError(t, err)
	require.Equal(t, map[string]string{
		"s3.access-key-id": "AK", "s3.secret-access-key": "SK", "s3.session-token": "TOK",
		"s3.session-token-expires-at-ms": "1800000000000",
		"s3.region":                      "us-east-1", "s3.endpoint": "http://seaweedfs:8333", "s3.path-style-access": "true",
	}, map[string]string(props))

	require.Equal(t, "iceberg-inngest.runs_v2", *f.in.RoleSessionName)
	require.EqualValues(t, 900, *f.in.DurationSeconds)
	require.JSONEq(t, `{"Version":"2012-10-17","Statement":[
		{"Effect":"Allow","Action":["s3:GetObject"],"Resource":["arn:aws:s3:::bucket/wh/inngest.db/runs/*"]},
		{"Effect":"Allow","Action":["s3:ListBucket"],"Resource":["arn:aws:s3:::bucket"],
		 "Condition":{"StringLike":{"s3:prefix":["wh/inngest.db/runs","wh/inngest.db/runs/*"]}}}]}`, *f.in.Policy)

	// Cached until halfway to expiry (1_800_000_000_000ms).
	now := time.UnixMilli(1_700_000_000_000)
	v.Now = func() time.Time { return now }
	v.cache = nil
	f.calls = 0
	loc := "s3://bucket/wh/inngest.db/runs"
	for range 3 {
		_, err := v.VendCredentials(t.Context(), []string{"inngest", "runs"}, loc)
		require.NoError(t, err)
	}
	require.Equal(t, 1, f.calls)
	_, err = v.VendCredentials(t.Context(), []string{"inngest", "spans"}, "s3://bucket/wh/inngest.db/spans")
	require.NoError(t, err)
	require.Equal(t, 2, f.calls, "per location")
	now = time.UnixMilli(1_750_000_000_001)
	_, err = v.VendCredentials(t.Context(), []string{"inngest", "runs"}, loc)
	require.NoError(t, err)
	require.Equal(t, 3, f.calls, "refreshed past half life")

	for _, loc := range []string{
		"/local/dir",
		"s3://bucket",
		"s3://bucket/",
		"gs://b/p",
		"s3://user:pass@bucket/wh/t",
		"s3://bucket/wh/t*",
		"s3://bucket/wh/t?",
		"s3://bucket/wh/${aws:username}",
		"s3://bucket/wh/t%2a",
	} {
		_, err := v.VendCredentials(t.Context(), []string{"t"}, loc)
		require.Error(t, err, loc)
	}
}

type blockingSTS struct {
	calls   atomic.Int64
	started chan struct{}
	release chan struct{}
}

func (b *blockingSTS) AssumeRole(_ context.Context, _ *sts.AssumeRoleInput, _ ...func(*sts.Options)) (*sts.AssumeRoleOutput, error) {
	if b.calls.Add(1) == 1 {
		close(b.started)
	}
	<-b.release
	return &sts.AssumeRoleOutput{Credentials: &ststypes.Credentials{
		AccessKeyId: aws.String("AK"), SecretAccessKey: aws.String("SK"), SessionToken: aws.String("TOK"),
		Expiration: aws.Time(time.UnixMilli(1_800_000_000_000)),
	}}, nil
}

func TestSTSVendorHardening(t *testing.T) {
	t.Run("singleflight coalesces concurrent calls", func(t *testing.T) {
		b := &blockingSTS{started: make(chan struct{}), release: make(chan struct{})}
		now := time.UnixMilli(1_700_000_000_000)
		v := &STSVendor{STS: b, RoleARN: "arn:x", Now: func() time.Time { return now }}

		const n = 8
		var wg sync.WaitGroup
		wg.Add(n)
		for range n {
			go func() {
				defer wg.Done()
				props, err := v.VendCredentials(t.Context(), []string{"app", "t"}, "s3://bucket/wh/app.db/t")
				require.NoError(t, err)
				require.Equal(t, "AK", props["s3.access-key-id"])
			}()
		}
		<-b.started
		time.Sleep(10 * time.Millisecond)
		close(b.release)
		wg.Wait()
		require.Equal(t, int64(1), b.calls.Load())
	})

	t.Run("returns cloned properties map", func(t *testing.T) {
		f := &fakeSTS{}
		now := time.UnixMilli(1_700_000_000_000)
		v := &STSVendor{STS: f, RoleARN: "arn:x", Now: func() time.Time { return now }}
		p1, err := v.VendCredentials(t.Context(), []string{"app", "t"}, "s3://bucket/wh/app.db/t")
		require.NoError(t, err)
		p1["s3.access-key-id"] = "MUTATED"

		p2, err := v.VendCredentials(t.Context(), []string{"app", "t"}, "s3://bucket/wh/app.db/t")
		require.NoError(t, err)
		require.Equal(t, "AK", p2["s3.access-key-id"])
	})

	t.Run("evicts expired entries on write", func(t *testing.T) {
		f := &fakeSTS{}
		now := time.UnixMilli(1_700_000_000_000)
		v := &STSVendor{STS: f, RoleARN: "arn:x", Now: func() time.Time { return now }}
		_, err := v.VendCredentials(t.Context(), []string{"app", "a"}, "s3://bucket/wh/app.db/a")
		require.NoError(t, err)
		require.Len(t, v.cache, 1)

		// Advance past half-life (1_750_000_000_000ms) and vend for another table.
		now = time.UnixMilli(1_760_000_000_000)
		_, err = v.VendCredentials(t.Context(), []string{"app", "b"}, "s3://bucket/wh/app.db/b")
		require.NoError(t, err)
		require.Len(t, v.cache, 1)
		require.Contains(t, v.cache, "s3://bucket/wh/app.db/b")
		require.NotContains(t, v.cache, "s3://bucket/wh/app.db/a")
	})

	t.Run("rejects invalid durations", func(t *testing.T) {
		for _, d := range []time.Duration{-time.Second, 500 * time.Millisecond, 999 * time.Millisecond} {
			f := &fakeSTS{}
			v := &STSVendor{STS: f, RoleARN: "arn:x", Duration: d}
			_, err := v.VendCredentials(t.Context(), []string{"app", "t"}, "s3://bucket/wh/app.db/t")
			require.Error(t, err, "duration %v", d)
			require.Equal(t, 0, f.calls)
		}
	})
}
