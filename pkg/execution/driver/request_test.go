package driver

import (
	"encoding/json"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestStack(t *testing.T) {
	fnStack := FunctionStack{Stack: nil}

	marshaled, err := json.Marshal(fnStack)
	require.NoError(t, err)

	assert.Equal(t, "{\"stack\":[],\"current\":0}", string(marshaled))
}
