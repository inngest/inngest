package executor

import (
	"testing"

	"github.com/inngest/inngest/pkg/enums"
	"github.com/inngest/inngest/pkg/execution/state"
	"github.com/stretchr/testify/require"
)

func TestSortOps(t *testing.T) {
	input := []*state.GeneratorOpcode{
		{
			Op: enums.OpcodeStep,
		},
		{
			Op: enums.OpcodeStep,
		},
		{
			Op: enums.OpcodeSleep,
		},
		{
			Op: enums.OpcodeWaitForEvent,
		},
	}
	expected := []*state.GeneratorOpcode{
		{
			Op: enums.OpcodeWaitForEvent,
		},
		{
			Op: enums.OpcodeStep,
		},
		{
			Op: enums.OpcodeStep,
		},
		{
			Op: enums.OpcodeSleep,
		},
	}

	sortOps(input)
	require.EqualValues(t, expected, input)
}
