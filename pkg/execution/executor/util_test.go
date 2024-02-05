package executor

import (
	"testing"

	"github.com/inngest/inngest/pkg/enums"
	"github.com/inngest/inngest/pkg/execution/state"
	"github.com/stretchr/testify/require"
)

func TestOpGroups(t *testing.T) {
	input := []*state.GeneratorOpcode{
		{
			Op: enums.OpcodeSleep,
			ID: "1",
		},
		{
			Op: enums.OpcodeWaitForEvent,
			ID: "2",
		},
		{
			Op: enums.OpcodeStepRun,
			ID: "3",
		},
		{
			Op: enums.OpcodeSleep,
			ID: "4",
		},
		{
			Op: enums.OpcodeWaitForEvent,
			ID: "5",
		},
	}

	expected := OpcodeGroups{
		PriorityGroup: OpcodeGroup{
			Opcodes: []*state.GeneratorOpcode{
				{
					Op: enums.OpcodeWaitForEvent,
					ID: "2",
				},
				{
					Op: enums.OpcodeWaitForEvent,
					ID: "5",
				},
			},
			ShouldStartHistoryGroup: true,
		},
		OtherGroup: OpcodeGroup{
			Opcodes: []*state.GeneratorOpcode{
				{
					Op: enums.OpcodeSleep,
					ID: "1",
				},
				{
					Op: enums.OpcodeStepRun,
					ID: "3",
				},
				{
					Op: enums.OpcodeSleep,
					ID: "4",
				},
			},
			ShouldStartHistoryGroup: true,
		},
	}
	actual := opGroups(input)

	require.EqualValues(t, expected, actual)
}

func TestOpGroupsNoInput(t *testing.T) {
	input := []*state.GeneratorOpcode{}

	expected := OpcodeGroups{
		PriorityGroup: OpcodeGroup{
			Opcodes:                 []*state.GeneratorOpcode{},
			ShouldStartHistoryGroup: false,
		},
		OtherGroup: OpcodeGroup{
			Opcodes:                 []*state.GeneratorOpcode{},
			ShouldStartHistoryGroup: false,
		},
	}
	actual := opGroups(input)

	require.EqualValues(t, expected, actual)
}

func TestOpGroupsSingleInput(t *testing.T) {
	input := []*state.GeneratorOpcode{
		{
			Op: enums.OpcodeSleep,
			ID: "1",
		},
	}

	expected := OpcodeGroups{
		PriorityGroup: OpcodeGroup{
			Opcodes:                 []*state.GeneratorOpcode{},
			ShouldStartHistoryGroup: false,
		},
		OtherGroup: OpcodeGroup{
			Opcodes: []*state.GeneratorOpcode{
				{
					Op: enums.OpcodeSleep,
					ID: "1",
				},
			},
			ShouldStartHistoryGroup: false,
		},
	}
	actual := opGroups(input)

	require.EqualValues(t, expected, actual)
}

func TestOpcodeGroupsAllWithMixedInput(t *testing.T) {
	input := []*state.GeneratorOpcode{
		{Op: enums.OpcodeWaitForEvent, ID: "1"},
		{Op: enums.OpcodeStepRun, ID: "2"},
	}
	groups := opGroups(input)

	expected := []OpcodeGroup{
		groups.PriorityGroup,
		groups.OtherGroup,
	}
	actual := groups.All()

	require.EqualValues(t, expected, actual)
}

func TestOpcodeGroupsAllWithEmptyInput(t *testing.T) {
	input := []*state.GeneratorOpcode{}
	groups := opGroups(input)

	expected := []OpcodeGroup{
		groups.PriorityGroup,
		groups.OtherGroup,
	}
	actual := groups.All()

	require.EqualValues(t, expected, actual)
}
