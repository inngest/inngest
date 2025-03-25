package redis_state

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/rueidis"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeClient struct {
	rc        rueidis.Client
	doCounter int
}

func (f *fakeClient) B() rueidis.Builder {
	return f.rc.B()
}

func (f *fakeClient) Do(ctx context.Context, cmd rueidis.Completed) (resp rueidis.RedisResult) {
	f.doCounter++
	return f.rc.Do(ctx, cmd)
}

func (f *fakeClient) DoMulti(ctx context.Context, multi ...rueidis.Completed) (resp []rueidis.RedisResult) {
	//TODO implement me
	panic("implement me")
}

func (f *fakeClient) Receive(ctx context.Context, subscribe rueidis.Completed, fn func(msg rueidis.PubSubMessage)) error {
	//TODO implement me
	panic("implement me")
}

func (f *fakeClient) Close() {
	//TODO implement me
	panic("implement me")
}

func (f *fakeClient) DoCache(ctx context.Context, cmd rueidis.Cacheable, ttl time.Duration) (resp rueidis.RedisResult) {
	//TODO implement me
	panic("implement me")
}

func (f *fakeClient) DoMultiCache(ctx context.Context, multi ...rueidis.CacheableTTL) (resp []rueidis.RedisResult) {
	//TODO implement me
	panic("implement me")
}

func (f *fakeClient) DoStream(ctx context.Context, cmd rueidis.Completed) rueidis.RedisResultStream {
	//TODO implement me
	panic("implement me")
}

func (f *fakeClient) DoMultiStream(ctx context.Context, multi ...rueidis.Completed) rueidis.MultiRedisResultStream {
	//TODO implement me
	panic("implement me")
}

func (f *fakeClient) Dedicated(fn func(rueidis.DedicatedClient) error) (err error) {
	//TODO implement me
	panic("implement me")
}

func (f *fakeClient) Dedicate() (client rueidis.DedicatedClient, cancel func()) {
	//TODO implement me
	panic("implement me")
}

func (f *fakeClient) Nodes() map[string]rueidis.Client {
	//TODO implement me
	panic("implement me")
}

func newFakeClient(rc rueidis.Client) rueidis.Client {
	return &fakeClient{rc: rc}
}

func TestClusterSafeClient(t *testing.T) {
	ctx := context.Background()

	r := miniredis.RunT(t)

	rc, err := rueidis.NewClient(rueidis.ClientOption{
		InitAddress:  []string{r.Addr()},
		DisableCache: true,
	})
	require.NoError(t, err)
	defer rc.Close()

	r.SetError("CLUSTERDOWN")
	err = rc.Do(ctx, rc.B().Set().Key("test").Value("value").Build()).Error()
	rerr, ok := rueidis.IsRedisErr(err)
	require.Error(t, err)
	require.True(t, ok)
	require.True(t, rerr.IsClusterDown())

	r.SetError("")
	err = rc.Do(ctx, rc.B().Set().Key("test").Value("value").Build()).Error()
	require.NoError(t, err)

	fc := newFakeClient(rc)
	c := newRetryClusterDownClient(fc)

	r.SetError("CLUSTERDOWN")
	err = c.Do(ctx, func(client rueidis.Client) rueidis.Completed {
		return client.B().Set().Key("test").Value("value").Build()
	}).Error()
	rerr, ok = rueidis.IsRedisErr(err)
	require.Error(t, err)
	require.True(t, ok)
	require.True(t, rerr.IsClusterDown())
	assert.Equal(t, 6, fc.(*fakeClient).doCounter)

}
