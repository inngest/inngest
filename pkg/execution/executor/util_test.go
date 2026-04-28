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

// TestOpGroups_DeferOpsArePriority asserts that DeferAdd and DeferCancel are
// routed into the priority group so they drain before non-lazy ops in the same
// SDK response — fixing the [DeferAdd, RunComplete] race where Finalize would
// delete state before SaveDefer ran.
func TestOpGroups_DeferOpsArePriority(t *testing.T) {
	cases := []struct {
		name     string
		input    []*state.GeneratorOpcode
		expected OpcodeGroups
	}{
		{
			name: "lone DeferAdd",
			input: []*state.GeneratorOpcode{
				{Op: enums.OpcodeDeferAdd, ID: "1"},
			},
			expected: OpcodeGroups{
				PriorityGroup: OpcodeGroup{
					Opcodes: []*state.GeneratorOpcode{
						{Op: enums.OpcodeDeferAdd, ID: "1"},
					},
					ShouldStartHistoryGroup: false,
				},
				OtherGroup: OpcodeGroup{
					Opcodes:                 []*state.GeneratorOpcode{},
					ShouldStartHistoryGroup: false,
				},
			},
		},
		{
			name: "StepRun with DeferAdd piggyback",
			input: []*state.GeneratorOpcode{
				{Op: enums.OpcodeStepRun, ID: "1"},
				{Op: enums.OpcodeDeferAdd, ID: "2"},
			},
			expected: OpcodeGroups{
				PriorityGroup: OpcodeGroup{
					Opcodes: []*state.GeneratorOpcode{
						{Op: enums.OpcodeDeferAdd, ID: "2"},
					},
					ShouldStartHistoryGroup: true,
				},
				OtherGroup: OpcodeGroup{
					Opcodes: []*state.GeneratorOpcode{
						{Op: enums.OpcodeStepRun, ID: "1"},
					},
					ShouldStartHistoryGroup: true,
				},
			},
		},
		{
			name: "DeferAdd before RunComplete",
			input: []*state.GeneratorOpcode{
				{Op: enums.OpcodeDeferAdd, ID: "1"},
				{Op: enums.OpcodeRunComplete, ID: "2"},
			},
			expected: OpcodeGroups{
				PriorityGroup: OpcodeGroup{
					Opcodes: []*state.GeneratorOpcode{
						{Op: enums.OpcodeDeferAdd, ID: "1"},
					},
					ShouldStartHistoryGroup: true,
				},
				OtherGroup: OpcodeGroup{
					Opcodes: []*state.GeneratorOpcode{
						{Op: enums.OpcodeRunComplete, ID: "2"},
					},
					ShouldStartHistoryGroup: true,
				},
			},
		},
		{
			name: "WaitForEvent and DeferAdd both priority",
			input: []*state.GeneratorOpcode{
				{Op: enums.OpcodeWaitForEvent, ID: "1"},
				{Op: enums.OpcodeDeferAdd, ID: "2"},
			},
			expected: OpcodeGroups{
				PriorityGroup: OpcodeGroup{
					Opcodes: []*state.GeneratorOpcode{
						{Op: enums.OpcodeWaitForEvent, ID: "1"},
						{Op: enums.OpcodeDeferAdd, ID: "2"},
					},
					ShouldStartHistoryGroup: true,
				},
				OtherGroup: OpcodeGroup{
					Opcodes:                 []*state.GeneratorOpcode{},
					ShouldStartHistoryGroup: true,
				},
			},
		},
		{
			name: "lone DeferCancel",
			input: []*state.GeneratorOpcode{
				{Op: enums.OpcodeDeferCancel, ID: "1"},
			},
			expected: OpcodeGroups{
				PriorityGroup: OpcodeGroup{
					Opcodes: []*state.GeneratorOpcode{
						{Op: enums.OpcodeDeferCancel, ID: "1"},
					},
					ShouldStartHistoryGroup: false,
				},
				OtherGroup: OpcodeGroup{
					Opcodes:                 []*state.GeneratorOpcode{},
					ShouldStartHistoryGroup: false,
				},
			},
		},
		{
			name: "StepRun with DeferCancel piggyback",
			input: []*state.GeneratorOpcode{
				{Op: enums.OpcodeStepRun, ID: "1"},
				{Op: enums.OpcodeDeferCancel, ID: "2"},
			},
			expected: OpcodeGroups{
				PriorityGroup: OpcodeGroup{
					Opcodes: []*state.GeneratorOpcode{
						{Op: enums.OpcodeDeferCancel, ID: "2"},
					},
					ShouldStartHistoryGroup: true,
				},
				OtherGroup: OpcodeGroup{
					Opcodes: []*state.GeneratorOpcode{
						{Op: enums.OpcodeStepRun, ID: "1"},
					},
					ShouldStartHistoryGroup: true,
				},
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			actual := opGroups(tc.input)
			require.EqualValues(t, tc.expected, actual)
		})
	}
}
