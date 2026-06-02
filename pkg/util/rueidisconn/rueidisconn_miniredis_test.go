package rueidisconn

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/rueidis"
	"github.com/stretchr/testify/require"
)

func TestNewClient_Miniredis_NonLuaDoesNotEmitTTFB(t *testing.T) {
	events := make(chan TTFBEvent, 8)
	rc := newMiniredisClient(t, func(evt TTFBEvent) {
		events <- evt
	})

	err := rc.Do(context.Background(), rc.B().Set().Key("k").Value("v").Build()).Error()
	require.NoError(t, err)

	select {
	case evt := <-events:
		t.Fatalf("unexpected TTFB event for non-lua command: %+v", evt)
	case <-time.After(150 * time.Millisecond):
	}
}

func TestNewClient_Miniredis_EvalshaEmitsSha1(t *testing.T) {
	script := "return ARGV[1]"
	sum := sha1.Sum([]byte(script))
	expectedSHA1 := hex.EncodeToString(sum[:])

	events := make(chan TTFBEvent, 16)
	rc := newMiniredisClient(t, func(evt TTFBEvent) {
		events <- evt
	})

	lua := rueidis.NewLuaScript(script)
	out, err := lua.Exec(context.Background(), rc, nil, []string{"ok"}).ToString()
	require.NoError(t, err)
	require.Equal(t, "ok", out)

	// First execution may do EVALSHA -> NOSCRIPT -> EVAL. Execute again so at
	// least one successful EVALSHA is guaranteed.
	out, err = lua.Exec(context.Background(), rc, nil, []string{"ok2"}).ToString()
	require.NoError(t, err)
	require.Equal(t, "ok2", out)

	seenEvalSha := false
	deadline := time.After(2 * time.Second)
	for !seenEvalSha {
		select {
		case evt := <-events:
			if evt.Command == "EVALSHA" {
				require.Equal(t, expectedSHA1, evt.ScriptName)
				require.Greater(t, evt.TTFB, time.Duration(0))
				seenEvalSha = true
			}
		case <-deadline:
			t.Fatal("did not observe EVALSHA TTFB event")
		}
	}
}

func TestNewClient_Miniredis_EvalNoShaHasEmptyScriptName(t *testing.T) {
	events := make(chan TTFBEvent, 8)
	rc := newMiniredisClient(t, func(evt TTFBEvent) {
		events <- evt
	})

	lua := rueidis.NewLuaScriptNoSha("return ARGV[1]")
	out, err := lua.Exec(context.Background(), rc, nil, []string{"value"}).ToString()
	require.NoError(t, err)
	require.Equal(t, "value", out)

	select {
	case evt := <-events:
		require.Equal(t, "EVAL", evt.Command)
		require.Empty(t, evt.ScriptName)
		require.Greater(t, evt.TTFB, time.Duration(0))
	case <-time.After(2 * time.Second):
		t.Fatal("did not observe EVAL TTFB event")
	}
}

func newMiniredisClient(t *testing.T, handler TTFBHandler) rueidis.Client {
	t.Helper()

	r := miniredis.RunT(t)
	t.Cleanup(r.Close)

	rc, err := NewClient(rueidis.ClientOption{
		InitAddress:           []string{r.Addr()},
		DisableCache:          true,
		DisableAutoPipelining: true,
	}, handler)
	require.NoError(t, err)
	t.Cleanup(rc.Close)

	return rc
}
