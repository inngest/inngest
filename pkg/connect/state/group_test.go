package state

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"testing"

	"github.com/google/uuid"
	"github.com/gowebpki/jcs"
	"github.com/inngest/inngest/pkg/connect/auth"
	"github.com/inngest/inngest/pkg/sdk"
	connectpb "github.com/inngest/inngest/proto/gen/connect/v1"
	"github.com/stretchr/testify/require"
)

const (
	fnAlpha = `{"id":"app-alpha","name":"Alpha","triggers":[{"event":"a/one"},{"event":"a/two"}],"steps":{"step":{"id":"step","name":"step","runtime":{"type":"ws"}}}}`
	fnBeta  = `{"name":"Beta","id":"app-beta","triggers":[{"cron":"* * * * *"}],"steps":{"step":{"id":"step","name":"step","runtime":{"type":"ws"}}}}`
	fnGamma = `{"id":"app-gamma","name":"Gamma","triggers":[{"event":"g/one"}],"steps":{"step":{"id":"step","name":"step","runtime":{"type":"ws"}}}}`
)

func fnList(fns ...string) []byte {
	out := "["
	for i, fn := range fns {
		if i > 0 {
			out += ","
		}
		out += fn
	}
	return []byte(out + "]")
}

func hashOf(t *testing.T, functions []byte) []byte {
	t.Helper()
	h, err := functionConfigHash(&connectpb.AppConfiguration{Functions: functions})
	require.NoError(t, err)
	return h
}

func TestFunctionConfigHashIgnoresFunctionOrder(t *testing.T) {
	want := hashOf(t, fnList(fnAlpha, fnBeta, fnGamma))

	for _, order := range [][]string{
		{fnAlpha, fnGamma, fnBeta},
		{fnBeta, fnAlpha, fnGamma},
		{fnBeta, fnGamma, fnAlpha},
		{fnGamma, fnAlpha, fnBeta},
		{fnGamma, fnBeta, fnAlpha},
	} {
		require.Equal(t, want, hashOf(t, fnList(order...)))
	}
}

func TestFunctionConfigHashUnchangedForSortedConfig(t *testing.T) {
	// Workers that already send functions sorted by ID, such as SDKs with the
	// ordering fix, must keep the hash produced before canonical ordering.
	sorted := fnList(fnAlpha, fnBeta, fnGamma)
	before, err := jcs.Transform(sorted)
	require.NoError(t, err)
	want := sha256.Sum256(before)

	require.Equal(t, want[:], hashOf(t, sorted))
}

func TestFunctionConfigHashKeepsNestedOrder(t *testing.T) {
	swapped := `{"id":"app-alpha","name":"Alpha","triggers":[{"event":"a/two"},{"event":"a/one"}],"steps":{"step":{"id":"step","name":"step","runtime":{"type":"ws"}}}}`

	require.NotEqual(t,
		hashOf(t, fnList(fnAlpha, fnBeta)),
		hashOf(t, fnList(swapped, fnBeta)),
	)
}

func TestFunctionConfigHashDetectsChanges(t *testing.T) {
	renamed := `{"id":"app-alpha","name":"Alpha v2","triggers":[{"event":"a/one"},{"event":"a/two"}],"steps":{"step":{"id":"step","name":"step","runtime":{"type":"ws"}}}}`

	require.NotEqual(t, hashOf(t, fnList(fnAlpha, fnBeta)), hashOf(t, fnList(renamed, fnBeta)))
	require.NotEqual(t, hashOf(t, fnList(fnAlpha, fnBeta)), hashOf(t, fnList(fnAlpha)))
}

func TestFunctionConfigHashOrdersDuplicateIDs(t *testing.T) {
	other := `{"id":"app-alpha","name":"Other"}`

	require.Equal(t,
		hashOf(t, fnList(fnAlpha, other, fnBeta)),
		hashOf(t, fnList(fnBeta, other, fnAlpha)),
	)
}

func TestCanonicalFunctionConfigEmptyList(t *testing.T) {
	got, err := canonicalFunctionConfig([]byte(` [ ] `))
	require.NoError(t, err)
	require.Equal(t, "[]", string(got))
}

func TestCanonicalFunctionConfigRejectsInvalidJSON(t *testing.T) {
	_, err := canonicalFunctionConfig([]byte(`[{"id":`))
	require.Error(t, err)
}

func TestWorkerGroupHashIgnoresFunctionOrder(t *testing.T) {
	ctx := context.Background()
	authResp := &auth.Response{AccountID: uuid.New(), EnvID: uuid.New()}

	caps, err := json.Marshal(sdk.Capabilities{Connect: sdk.ConnectV1})
	require.NoError(t, err)

	group := func(functions []byte) *WorkerGroup {
		appConfig := &connectpb.AppConfiguration{AppName: "app", Functions: functions}
		req := &connectpb.WorkerConnectRequestData{
			Apps:         []*connectpb.AppConfiguration{appConfig},
			SdkLanguage:  "go",
			SdkVersion:   "v0.0.0",
			Capabilities: caps,
			AuthData:     &connectpb.AuthData{SyncToken: "sync"},
		}
		wg, err := NewWorkerGroupFromConnRequest(ctx, req, authResp, appConfig)
		require.NoError(t, err)
		return wg
	}

	a := group(fnList(fnAlpha, fnBeta, fnGamma))
	b := group(fnList(fnGamma, fnAlpha, fnBeta))
	require.Equal(t, a.Hash, b.Hash)
	require.ElementsMatch(t, a.FunctionSlugs, b.FunctionSlugs)
}
