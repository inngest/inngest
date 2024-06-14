package driver

import (
	"encoding/json"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestStack(t *testing.T) {
	req := SDKRequest{Context: &SDKRequestContext{Stack: &FunctionStack{Stack: nil}}}

	marshaled, err := json.Marshal(req)
	require.NoError(t, err)

	assert.Equal(t, "{\"event\":null,\"events\":null,\"steps\":null,\"ctx\":{\"fn_id\":\"00000000-0000-0000-0000-000000000000\",\"run_id\":\"00000000000000000000000000\",\"env\":\"\",\"step_id\":\"\",\"attempt\":0,\"stack\":{\"stack\":[],\"current\":0},\"disable_immediate_execution\":false,\"use_api\":false},\"version\":0,\"use_api\":false}", string(marshaled))
}
