package enums

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestOpcodeLoopReturn(t *testing.T) {
	require.Equal(t, "LoopReturn", OpcodeLoopReturn.String())

	parsed, err := OpcodeString("LoopReturn")
	require.NoError(t, err)
	require.Equal(t, OpcodeLoopReturn, parsed)

	byt, err := json.Marshal(OpcodeLoopReturn)
	require.NoError(t, err)
	require.Equal(t, `"LoopReturn"`, string(byt))

	var op Opcode
	require.NoError(t, json.Unmarshal(byt, &op))
	require.Equal(t, OpcodeLoopReturn, op)

	require.False(t, OpcodeIsSync(OpcodeLoopReturn))
}
