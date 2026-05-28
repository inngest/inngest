package defers

import (
	"testing"

	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/require"
)

func TestEventID(t *testing.T) {
	parent := ulid.MustParse("01HKQJZ5R7XR4MNTQGZ8Z3KPAB")
	require.Equal(t,
		"01HKQJZ5R7CYQKSKMYBEBKYTBM",
		EventID(parent, "fixed-hashed-id").String())
}

func TestSpanSeed(t *testing.T) {
	parent := ulid.MustParse("01HKQJZ5R7XR4MNTQGZ8Z3KPAB")
	require.Equal(t,
		[]byte(parent.String()+"fixed-hashed-id"+"s"),
		SpanSeed(parent, "fixed-hashed-id", SpanSchedule))
}
