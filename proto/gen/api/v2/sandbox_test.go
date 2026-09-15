package apiv2

import (
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/encoding/protojson"
)

func TestWriteSandboxFileDataUsesNumericBytesWritten(t *testing.T) {
	encoded, err := protojson.Marshal(&WriteSandboxFileData{
		Path:         "/tmp/message.txt",
		BytesWritten: 5,
	})
	require.NoError(t, err)
	require.JSONEq(t, `{"path":"/tmp/message.txt","bytesWritten":5}`, string(encoded))
}

func TestCreateSandboxRequestSecretReferences(t *testing.T) {
	request := &CreateSandboxRequest{}
	require.NoError(t, protojson.Unmarshal([]byte(`{"name":"secret-sandbox","environment":{"APP_ENV":"test"},"secrets":{"TOKEN":"d1a22481-b0a0-451b-927f-99f19ae964ba"}}`), request))
	require.Equal(t, "test", request.GetEnvironment()["APP_ENV"])
	require.Equal(t, "d1a22481-b0a0-451b-927f-99f19ae964ba", request.GetSecrets()["TOKEN"])
	encoded, err := protojson.Marshal(request)
	require.NoError(t, err)
	require.Contains(t, string(encoded), `"secrets"`)
}
