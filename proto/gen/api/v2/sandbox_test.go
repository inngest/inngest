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

func TestSandboxSnapshotUsesStringStoredBytes(t *testing.T) {
	encoded, err := protojson.Marshal(&SandboxSnapshot{StoredBytes: 5})
	require.NoError(t, err)
	require.JSONEq(t, `{"storedBytes":"5"}`, string(encoded))
}

func TestCreateSandboxRequestSecretReferences(t *testing.T) {
	request := &CreateSandboxRequest{}
	require.NoError(t, protojson.Unmarshal([]byte(`{"name":"secret-sandbox","environment":{"APP_ENV":"test"},"secrets":["OPENAI_API_KEY","OTHER_TOKEN"]}`), request))
	require.Equal(t, "test", request.GetEnvironment()["APP_ENV"])
	require.Equal(t, []string{"OPENAI_API_KEY", "OTHER_TOKEN"}, request.GetSecrets())
	encoded, err := protojson.Marshal(request)
	require.NoError(t, err)
	roundTrip := &CreateSandboxRequest{}
	require.NoError(t, protojson.Unmarshal(encoded, roundTrip))
	require.Equal(t, request.GetSecrets(), roundTrip.GetSecrets())

	for _, selection := range []string{`{"OPENAI_API_KEY":"openai-production"}`, `"OPENAI_API_KEY"`, `[123]`} {
		t.Run(selection, func(t *testing.T) {
			require.Error(t, protojson.Unmarshal([]byte(`{"secrets":`+selection+`}`), &CreateSandboxRequest{}))
		})
	}
}
